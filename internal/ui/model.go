package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/config"
	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/feed"
)

// entryKind tags a row in the unified feed list: smart folder or plain feed.
type entryKind int

const (
	entryFolder entryKind = iota
	entryFeed
)

// feedEntry is a single row in the Feeds pane. It may be either a smart
// folder (virtual, query-backed) or a regular feed pulled from the DB.
type feedEntry struct {
	Kind entryKind

	// Feed-only:
	FeedID      int64
	UnreadCount int
	HasError    bool

	// Folder-only:
	FolderIdx int // index into m.smartFolders

	// Shared:
	Name string
}

type focus int

const (
	focusFeeds focus = iota
	focusArticles
	focusReader
	focusSettings
	focusSearch
	focusCommand
	focusLinks
)

type settingsMode int

const (
	smList settingsMode = iota
	smAddName
	smAddURL
	smRename
)

type articleFilter = db.ArticleFilter

const (
	filterAll     = db.FilterAll
	filterUnread  = db.FilterUnread
	filterStarred = db.FilterStarred
)

func filterLabel(f articleFilter) string {
	switch f {
	case filterUnread:
		return "unread"
	case filterStarred:
		return "starred"
	default:
		return "all"
	}
}

type Model struct {
	db      *db.DB
	fetcher *feed.Fetcher
	keys    keyMap

	feeds        []db.Feed
	smartFolders []config.SmartFolder
	articles     []db.Article
	selEntry     int
	selArt       int
	focus        focus

	// allArticles is a cached cross-feed snapshot used to compute smart
	// folder match counts without hitting the DB on every keystroke.
	allArticles  []db.Article
	folderCounts []int

	width  int
	height int

	spin     spinner.Model
	fetching bool

	reader    viewport.Model
	readerArt *db.Article

	help     help.Model
	showHelp bool

	feedErrors map[int64]error

	settingsMode  settingsMode
	settingsSel   int
	settingsInput textinput.Model
	pendingName   string

	searchInput   textinput.Model
	searchAll     []db.SearchItem
	searchMatches []int
	searchSel     int
	searchScroll  int
	searchPrev    focus
	searchErr     error

	filter            articleFilter
	zenMode           bool
	showImages        bool
	afterSyncCommands []string
	refreshInterval   time.Duration
	home              string

	// Link picker overlay (reader → press L to open).
	links    []articleLink
	linksSel int

	commandInput   textinput.Model
	commandPrev    focus
	commandSuggIdx int

	// Command history (vim-style :<prev command>). historyPos: -1 = not
	// browsing, else index into commandHistory (0 = most recent). When the
	// user starts browsing we stash the partial input in historyStash so
	// Ctrl+N can restore it.
	commandHistory []string
	historyPos     int
	historyStash   string

	sortField   string // "date" or "title"
	sortReverse bool

	status string
	err    error
}

func New(database *db.DB, fetcher *feed.Fetcher, smartFolders []config.SmartFolder, afterSyncCommands []string, refreshIntervalMinutes int, home string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorAccent)

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(colorAccent)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(colorMuted)
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(colorAccent)
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(colorMuted)

	ti := textinput.New()
	ti.CharLimit = 256
	ti.Prompt = "› "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent)

	si := textinput.New()
	si.CharLimit = 128
	si.Prompt = "› "
	si.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent)

	ci := textinput.New()
	ci.CharLimit = 128
	ci.Prompt = ":"
	ci.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent)

	return Model{
		db:       database,
		fetcher:  fetcher,
		keys:     defaultKeys(),
		status:   "fetching…",
		spin:     s,
		fetching: true,
		reader:        viewport.New(0, 0),
		help:          h,
		feedErrors:        map[int64]error{},
		smartFolders:      smartFolders,
		afterSyncCommands: afterSyncCommands,
		refreshInterval:   time.Duration(refreshIntervalMinutes) * time.Minute,
		home:              home,
		settingsInput:     ti,
		searchInput:       si,
		commandInput:      ci,
		commandHistory:    readHistoryFile(home),
		historyPos:        -1,
		sortField:         "date",
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		loadFeedsCmd(m.db),
		loadAllArticlesCmd(m.db),
		fetchAllCmd(m.fetcher),
		m.spin.Tick,
	}
	if tick := scheduleRefreshCmd(m.refreshInterval); tick != nil {
		cmds = append(cmds, tick)
	}
	return tea.Batch(cmds...)
}

// scheduleRefreshCmd arms a one-shot timer that fires refreshTickMsg
// after interval. Returns nil when auto-refresh is disabled so callers
// can cheaply branch on the result.
func scheduleRefreshCmd(interval time.Duration) tea.Cmd {
	if interval <= 0 {
		return nil
	}
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.reader.Width = m.width - 4
		m.reader.Height = m.height - 2
		if m.readerArt != nil {
			feedName := readerFeedName(m.feeds, m.readerArt.FeedID)
			m.reader.SetContent(buildReaderContent(*m.readerArt, feedName, m.reader.Width-4, m.showImages))
		}
		return m, nil

	case tea.KeyMsg:
		if m.focus == focusSettings {
			return m.updateSettings(msg)
		}
		if m.focus == focusSearch {
			return m.updateSearch(msg)
		}
		if m.focus == focusCommand {
			return m.updateCommand(msg)
		}
		if m.focus == focusLinks {
			return m.updateLinks(msg)
		}
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Settings):
			m.focus = focusSettings
			m.settingsMode = smList
			m.settingsSel = 0
			return m, nil
		case key.Matches(msg, m.keys.Search):
			m.searchPrev = m.focus
			m.focus = focusSearch
			m.searchInput.SetValue("")
			m.searchInput.Focus()
			m.searchSel = 0
			return m, tea.Batch(loadSearchCmd(m.db), textinput.Blink)
		case key.Matches(msg, m.keys.Zen):
			m.zenMode = !m.zenMode
			return m, nil
		case key.Matches(msg, m.keys.FilterAll):
			if m.filter != filterAll {
				m.filter = filterAll
				m.selArt = 0
				if len(m.feeds) > 0 {
					return m, m.loadCurrentCmd()
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.FilterUnread):
			if m.filter != filterUnread {
				m.filter = filterUnread
				m.selArt = 0
				if len(m.feeds) > 0 {
					return m, m.loadCurrentCmd()
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.FilterStarred):
			if m.filter != filterStarred {
				m.filter = filterStarred
				m.selArt = 0
				if len(m.feeds) > 0 {
					return m, m.loadCurrentCmd()
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.Star):
			return m.toggleStarOnCurrent()
		case key.Matches(msg, m.keys.NextUnread):
			return m.jumpToNextUnread()
		case key.Matches(msg, m.keys.Command):
			m.commandPrev = m.focus
			m.focus = focusCommand
			m.commandInput.SetValue("")
			m.commandInput.Focus()
			m.commandSuggIdx = 0
			return m, textinput.Blink
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			m.help.ShowAll = m.showHelp
			return m, nil
		case key.Matches(msg, m.keys.FullArticle):
			if m.focus == focusReader && m.readerArt != nil && m.readerArt.URL != "" && !m.fetching {
				m.fetching = true
				m.status = "loading full…"
				return m, tea.Batch(
					fetchFullCmd(m.fetcher, m.db, m.readerArt.ID, m.readerArt.URL),
					m.spin.Tick,
				)
			}
			return m, nil
		case key.Matches(msg, m.keys.OpenURL):
			var url string
			switch m.focus {
			case focusArticles:
				if len(m.articles) > 0 {
					url = m.articles[m.selArt].URL
				}
			case focusReader:
				if m.readerArt != nil {
					url = m.readerArt.URL
				}
			}
			if url != "" {
				if err := openInBrowser(url); err != nil {
					m.err = err
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.Tab):
			if m.focus == focusFeeds {
				m.focus = focusArticles
			} else {
				m.focus = focusFeeds
			}
			return m, nil
		case key.Matches(msg, m.keys.LinkPicker):
			if m.focus == focusReader {
				return m.openLinkPickerOnCurrent()
			}
			return m, nil
		case key.Matches(msg, m.keys.ToggleRead):
			return m.toggleReadOnCurrent()
		case key.Matches(msg, m.keys.MarkAllRead):
			return m.markAllReadOnCurrentEntry()
		case key.Matches(msg, m.keys.NextArticle):
			if m.focus == focusReader {
				return m.readerJump(+1)
			}
			return m, nil
		case key.Matches(msg, m.keys.PrevArticle):
			if m.focus == focusReader {
				return m.readerJump(-1)
			}
			return m, nil
		case msg.String() == " ":
			// Space as page-down in reader (classic less/more convention).
			if m.focus == focusReader {
				pgDown := tea.KeyMsg{Type: tea.KeyPgDown}
				var cmd tea.Cmd
				m.reader, cmd = m.reader.Update(pgDown)
				return m, cmd
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.focus == focusReader {
				var cmd tea.Cmd
				m.reader, cmd = m.reader.Update(msg)
				return m, cmd
			}
			return m.moveDown()
		case key.Matches(msg, m.keys.Up):
			if m.focus == focusReader {
				var cmd tea.Cmd
				m.reader, cmd = m.reader.Update(msg)
				return m, cmd
			}
			return m.moveUp()
		case key.Matches(msg, m.keys.Right), key.Matches(msg, m.keys.Enter):
			switch m.focus {
			case focusFeeds:
				if len(m.articles) > 0 {
					m.focus = focusArticles
				}
			case focusArticles:
				if len(m.articles) > 0 {
					return m.openReader()
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.Left), key.Matches(msg, m.keys.Back):
			switch m.focus {
			case focusArticles:
				m.focus = focusFeeds
				return m, nil
			case focusReader:
				m.focus = focusArticles
				m.readerArt = nil
				cmds := []tea.Cmd{loadFeedsCmd(m.db)}
				if len(m.feeds) > 0 {
					cmds = append(cmds, m.loadCurrentCmd())
				}
				return m, tea.Batch(cmds...)
			}
			return m, nil
		case key.Matches(msg, m.keys.Top):
			if m.focus == focusReader {
				m.reader.GotoTop()
				return m, nil
			}
			return m.moveTo(0)
		case key.Matches(msg, m.keys.Bottom):
			if m.focus == focusReader {
				m.reader.GotoBottom()
				return m, nil
			}
			return m.moveToEnd()
		case key.Matches(msg, m.keys.PageDown):
			if m.focus == focusReader {
				var cmd tea.Cmd
				m.reader, cmd = m.reader.Update(msg)
				return m, cmd
			}
			return m.moveByPage(+1)
		case key.Matches(msg, m.keys.PageUp):
			if m.focus == focusReader {
				var cmd tea.Cmd
				m.reader, cmd = m.reader.Update(msg)
				return m, cmd
			}
			return m.moveByPage(-1)
		case key.Matches(msg, m.keys.RefreshAll), key.Matches(msg, m.keys.RefreshOne):
			if m.fetching {
				return m, nil
			}
			m.fetching = true
			m.status = "fetching…"
			return m, tea.Batch(fetchAllCmd(m.fetcher), m.spin.Tick)
		}

	case spinner.TickMsg:
		if !m.fetching {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case fetchDoneMsg:
		m.fetching = false
		m.feedErrors = map[int64]error{}
		var failed int
		for _, r := range msg.results {
			if r.Err != nil {
				m.feedErrors[r.Feed.ID] = r.Err
				failed++
			}
		}
		if failed > 0 {
			m.status = fmt.Sprintf("fetched · %d error(s)", failed)
		} else {
			m.status = "fetched"
		}
		cmds := []tea.Cmd{loadFeedsCmd(m.db), loadAllArticlesCmd(m.db)}
		if len(m.feeds) > 0 {
			cmds = append(cmds, m.loadCurrentCmd())
		}
		// Fire any configured after_sync_commands. Each runs through the
		// same dispatchCommand path as user-typed :commands, so behavior
		// stays consistent. Errors from parse/dispatch surface via m.err.
		for _, line := range m.afterSyncCommands {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			nm, cmd := dispatchCommand(m, trimmed)
			m = nm.(Model)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case feedsLoadedMsg:
		m.feeds = msg.feeds
		m.err = nil
		if !m.fetching {
			m.status = "ready"
		}
		total := len(m.entries())
		if total > 0 {
			if m.selEntry >= total {
				m.selEntry = 0
			}
			return m, m.loadCurrentCmd()
		}
		return m, nil

	case articlesLoadedMsg:
		m.err = nil
		e, ok := m.currentEntry()
		if ok && e.Kind == entryFeed && e.FeedID == msg.feedID {
			m.articles = msg.articles
			applySort(m.articles, m.sortField, m.sortReverse)
			if m.selArt >= len(m.articles) {
				m.selArt = 0
			}
		}
		return m, nil

	case folderArticlesLoadedMsg:
		m.err = nil
		e, ok := m.currentEntry()
		if ok && e.Kind == entryFolder && e.FolderIdx == msg.folderIdx {
			m.articles = msg.articles
			applySort(m.articles, m.sortField, m.sortReverse)
			if m.selArt >= len(m.articles) {
				m.selArt = 0
			}
		}
		return m, nil

	case allArticlesLoadedMsg:
		m.err = nil
		m.allArticles = msg.articles
		m.refreshFolderCounts()
		return m, nil

	case refreshTickMsg:
		// Re-arm the timer unconditionally; either way we want the next
		// tick. Skip the fetch itself if one is already running.
		cmds := []tea.Cmd{}
		if tick := scheduleRefreshCmd(m.refreshInterval); tick != nil {
			cmds = append(cmds, tick)
		}
		if !m.fetching {
			m.fetching = true
			m.status = "auto-fetching…"
			cmds = append(cmds, fetchAllCmd(m.fetcher), m.spin.Tick)
		}
		return m, tea.Batch(cmds...)

	case batchAppliedMsg:
		m.err = nil
		m.status = fmt.Sprintf("%s · %d articles", msg.action, msg.count)
		// Refresh feed list (unread counts), current selection's article
		// list, and the cross-feed cache for folder counts.
		cmds := []tea.Cmd{loadFeedsCmd(m.db), loadAllArticlesCmd(m.db)}
		if c := m.loadCurrentCmd(); c != nil {
			cmds = append(cmds, c)
		}
		return m, tea.Batch(cmds...)

	case articleMarkedMsg:
		m.err = nil
		now := time.Now().UTC()
		// Sync both the currently loaded article list and the cross-feed
		// cache so folder counts / UI update in place without reloads.
		updateRead := func(a *db.Article) {
			if msg.unread {
				a.ReadAt = nil
				return
			}
			if a.ReadAt == nil {
				a.ReadAt = &now
			}
		}
		for i := range m.articles {
			if m.articles[i].ID == msg.articleID {
				updateRead(&m.articles[i])
				break
			}
		}
		for i := range m.allArticles {
			if m.allArticles[i].ID == msg.articleID {
				updateRead(&m.allArticles[i])
				break
			}
		}
		if m.readerArt != nil && m.readerArt.ID == msg.articleID {
			updateRead(m.readerArt)
		}
		m.refreshFolderCounts()
		if msg.unread {
			m.status = "marked unread"
		} else {
			m.status = "marked read"
		}
		// Refresh the feed list so unread counters update. The current
		// article list stays as-is (we already patched it) unless the
		// user is on filter=unread, where the row should disappear.
		cmds := []tea.Cmd{loadFeedsCmd(m.db)}
		if m.filter == filterUnread && !msg.unread && len(m.feeds) > 0 {
			cmds = append(cmds, m.loadCurrentCmd())
		}
		return m, tea.Batch(cmds...)

	case feedMarkedReadMsg:
		m.err = nil
		m.status = fmt.Sprintf("marked %d read", msg.count)
		// Reload feeds (counters), cache (folder counts), and current list.
		cmds := []tea.Cmd{loadFeedsCmd(m.db), loadAllArticlesCmd(m.db)}
		if c := m.loadCurrentCmd(); c != nil {
			cmds = append(cmds, c)
		}
		return m, tea.Batch(cmds...)

	case fullArticleLoadedMsg:
		m.fetching = false
		m.status = "full article"
		if m.readerArt != nil && m.readerArt.ID == msg.articleID {
			m.readerArt.CachedBody = msg.markdown
			now := time.Now().UTC()
			m.readerArt.CachedAt = &now
			feedName := readerFeedName(m.feeds, m.readerArt.FeedID)
			m.reader.SetContent(buildReaderContent(*m.readerArt, feedName, m.reader.Width-4, m.showImages))
			m.reader.GotoTop()
		}
		return m, nil

	case searchLoadedMsg:
		m.searchAll = msg.items
		m.searchSel = 0
		m.searchScroll = 0
		recomputeMatches(&m)
		clampSearchScroll(&m)
		return m, nil

	case starToggledMsg:
		m.err = nil
		// Update local article state so the UI reflects the change without reload.
		for i := range m.articles {
			if m.articles[i].ID == msg.articleID {
				if msg.starred {
					t := time.Now().UTC()
					m.articles[i].StarredAt = &t
				} else {
					m.articles[i].StarredAt = nil
				}
				break
			}
		}
		if m.readerArt != nil && m.readerArt.ID == msg.articleID {
			if msg.starred {
				t := time.Now().UTC()
				m.readerArt.StarredAt = &t
			} else {
				m.readerArt.StarredAt = nil
			}
		}
		// Update cached article for folder-count freshness.
		for i := range m.allArticles {
			if m.allArticles[i].ID == msg.articleID {
				if msg.starred {
					t := time.Now().UTC()
					m.allArticles[i].StarredAt = &t
				} else {
					m.allArticles[i].StarredAt = nil
				}
				break
			}
		}
		m.refreshFolderCounts()
		if msg.starred {
			m.status = "starred"
		} else {
			m.status = "unstarred"
		}
		// If viewing the starred filter and we just unstarred, reload so the
		// row falls out of the list.
		if m.filter == filterStarred && !msg.starred && len(m.feeds) > 0 {
			return m, m.loadCurrentCmd()
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}
	return m, nil
}

func (m Model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.settingsMode != smList {
		switch {
		case key.Matches(msg, m.keys.Back):
			m.settingsMode = smList
			m.settingsInput.Blur()
			m.settingsInput.SetValue("")
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			return m.settingsSubmit()
		}
		var cmd tea.Cmd
		m.settingsInput, cmd = m.settingsInput.Update(msg)
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Settings), key.Matches(msg, m.keys.Back):
		m.focus = focusFeeds
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.settingsSel < len(m.feeds)-1 {
			m.settingsSel++
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.settingsSel > 0 {
			m.settingsSel--
		}
		return m, nil
	case key.Matches(msg, m.keys.Top):
		m.settingsSel = 0
		return m, nil
	case key.Matches(msg, m.keys.Bottom):
		if len(m.feeds) > 0 {
			m.settingsSel = len(m.feeds) - 1
		}
		return m, nil
	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
		return m, nil
	case msg.String() == "a":
		m.settingsMode = smAddName
		m.settingsInput.SetValue("")
		m.settingsInput.Focus()
		return m, textinput.Blink
	case msg.String() == "d":
		if len(m.feeds) == 0 {
			return m, nil
		}
		id := m.feeds[m.settingsSel].ID
		if err := m.db.DeleteFeed(id); err != nil {
			m.err = err
			return m, nil
		}
		if m.settingsSel > 0 {
			m.settingsSel--
		}
		return m, loadFeedsCmd(m.db)
	case msg.String() == "e":
		if len(m.feeds) == 0 {
			return m, nil
		}
		m.settingsMode = smRename
		m.settingsInput.SetValue(m.feeds[m.settingsSel].Name)
		m.settingsInput.CursorEnd()
		m.settingsInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) settingsSubmit() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.settingsInput.Value())
	if value == "" {
		return m, nil
	}
	switch m.settingsMode {
	case smAddName:
		m.pendingName = value
		m.settingsMode = smAddURL
		m.settingsInput.SetValue("")
		return m, textinput.Blink
	case smAddURL:
		if _, err := m.db.UpsertFeed(m.pendingName, value); err != nil {
			m.err = err
			return m, nil
		}
		m.pendingName = ""
		m.settingsMode = smList
		m.settingsInput.Blur()
		m.settingsInput.SetValue("")
		return m, loadFeedsCmd(m.db)
	case smRename:
		if len(m.feeds) == 0 {
			return m, nil
		}
		id := m.feeds[m.settingsSel].ID
		if err := m.db.RenameFeed(id, value); err != nil {
			m.err = err
			return m, nil
		}
		m.settingsMode = smList
		m.settingsInput.Blur()
		m.settingsInput.SetValue("")
		return m, loadFeedsCmd(m.db)
	}
	return m, nil
}

func (m Model) openReader() (tea.Model, tea.Cmd) {
	a := m.articles[m.selArt]
	m.readerArt = &a
	m.focus = focusReader
	m.reader.Width = m.width - 4
	m.reader.Height = m.height - 2
	feedName := readerFeedName(m.feeds, a.FeedID)
	m.reader.SetContent(buildReaderContent(a, feedName, m.reader.Width-4, m.showImages))
	m.reader.GotoTop()
	if a.ReadAt == nil {
		return m, markReadCmd(m.db, a.ID)
	}
	return m, nil
}

// readerJump advances selArt by dir within the current article list and
// refreshes reader content without leaving focusReader. No-op if there is
// no neighbour in the requested direction. Marks the new article read.
func (m Model) readerJump(dir int) (tea.Model, tea.Cmd) {
	target := m.selArt + dir
	if target < 0 || target >= len(m.articles) {
		m.status = "end of list"
		return m, nil
	}
	m.selArt = target
	a := m.articles[target]
	m.readerArt = &a
	feedName := readerFeedName(m.feeds, a.FeedID)
	m.reader.SetContent(buildReaderContent(a, feedName, m.reader.Width-4, m.showImages))
	m.reader.GotoTop()
	if a.ReadAt == nil {
		return m, markReadCmd(m.db, a.ID)
	}
	return m, nil
}

func (m Model) moveDown() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusFeeds:
		if m.selEntry < len(m.entries())-1 {
			m.selEntry++
			return m, m.loadCurrentCmd()
		}
	case focusArticles:
		if m.selArt < len(m.articles)-1 {
			m.selArt++
		}
	}
	return m, nil
}

func (m Model) moveUp() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusFeeds:
		if m.selEntry > 0 {
			m.selEntry--
			return m, m.loadCurrentCmd()
		}
	case focusArticles:
		if m.selArt > 0 {
			m.selArt--
		}
	}
	return m, nil
}

func (m Model) moveTo(idx int) (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusFeeds:
		if idx < 0 || idx >= len(m.entries()) {
			return m, nil
		}
		m.selEntry = idx
		return m, m.loadCurrentCmd()
	case focusArticles:
		if idx < 0 || idx >= len(m.articles) {
			return m, nil
		}
		m.selArt = idx
	}
	return m, nil
}

func (m Model) moveToEnd() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusFeeds:
		return m.moveTo(len(m.entries()) - 1)
	case focusArticles:
		return m.moveTo(len(m.articles) - 1)
	}
	return m, nil
}

func (m Model) moveByPage(dir int) (tea.Model, tea.Cmd) {
	step := m.height - 4
	if step < 1 {
		step = 1
	}
	switch m.focus {
	case focusFeeds:
		return m.moveTo(clamp(m.selEntry+dir*step, 0, len(m.entries())-1))
	case focusArticles:
		return m.moveTo(clamp(m.selArt+dir*step, 0, len(m.articles)-1))
	}
	return m, nil
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "rdr — " + m.status
	}
	if m.width < 40 || m.height < 10 {
		return "rdr: terminal too small"
	}

	helpView := m.helpView()
	helpH := lipgloss.Height(helpView)

	if m.focus == focusSettings {
		body := renderSettings(
			m.feeds,
			m.settingsSel,
			m.settingsMode,
			m.settingsInput.View(),
			m.width,
			m.height-1-helpH,
		)
		statusText := "rdr · settings"
		if m.err != nil {
			statusText += "  " + errStyle.Render("! "+m.err.Error())
		}
		status := statusBar.Width(m.width).Render(statusText)
		return lipgloss.JoinVertical(lipgloss.Top, body, status, helpView)
	}

	if m.focus == focusSearch {
		body := renderSearch(m, m.width, m.height-1-helpH)
		statusText := "rdr · search"
		if m.err != nil {
			statusText += "  " + errStyle.Render("! "+m.err.Error())
		}
		status := statusBar.Width(m.width).Render(statusText)
		return lipgloss.JoinVertical(lipgloss.Top, body, status, helpView)
	}

	if m.focus == focusLinks {
		body := renderLinkPicker(m, m.width, m.height-1-helpH)
		statusText := "rdr · links"
		if m.err != nil {
			statusText += "  " + errStyle.Render("! "+m.err.Error())
		}
		status := statusBar.Width(m.width).Render(statusText)
		return lipgloss.JoinVertical(lipgloss.Top, body, status, helpView)
	}

	if m.focus == focusReader && m.readerArt != nil {
		statusText := "rdr · reader"
		if m.err != nil {
			statusText += "  " + errStyle.Render("! "+m.err.Error())
		}
		status := statusBar.Width(m.width).Render(statusText)
		body := paneActive.Width(m.width - 2).Height(m.height - 2 - helpH).Render(m.reader.View())
		frame := lipgloss.JoinVertical(lipgloss.Top, body, status, helpView)
		return frame
	}

	// In command mode, reserve vertical space for the suggestions popup so it
	// doesn't overlap with the main panes.
	popup := ""
	popupH := 0
	if m.focus == focusCommand {
		popup = renderCommandPopup(m, m.width)
		if popup != "" {
			popupH = lipgloss.Height(popup)
		}
	}

	paneH := m.height - 2 - helpH - popupH

	var row string
	if m.zenMode {
		// Only the focused pane is drawn at full width.
		fullW := m.width - 2
		if fullW < 10 {
			fullW = 10
		}
		entries := m.entries()
		if m.focus == focusFeeds {
			row = renderFeedList(entries, m.selEntry, true, fullW, paneH)
		} else {
			row = renderArticleList(m.articles, m.selArt, true, fullW, paneH)
		}
	} else {
		leftW := m.width/3 - 2
		if leftW < 10 {
			leftW = 10
		}
		rightW := m.width - leftW - 4
		if rightW < 10 {
			rightW = 10
		}
		entries := m.entries()
		left := renderFeedList(entries, m.selEntry, m.focus == focusFeeds, leftW, paneH)
		right := renderArticleList(m.articles, m.selArt, m.focus == focusArticles, rightW, paneH)
		row = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	if m.focus == focusCommand {
		cmdLine := statusBar.Width(m.width).Render(m.commandInput.View())
		return lipgloss.JoinVertical(lipgloss.Top, row, popup, cmdLine, helpView)
	}

	var status string
	{
		statusText := "rdr · " + m.status
		if m.fetching {
			statusText = "rdr · " + m.spin.View() + " " + m.status
		}
		statusText += "  " + searchCount.Render("["+filterLabel(m.filter)+"]")
		if m.sortField != "date" || m.sortReverse {
			dir := "↓"
			if m.sortReverse {
				dir = "↑"
			}
			statusText += " " + searchCount.Render("[sort:"+m.sortField+dir+"]")
		}
		if m.zenMode {
			statusText += " " + searchCount.Render("[zen]")
		}
		if m.err != nil {
			statusText += "  " + errStyle.Render("! "+m.err.Error())
		}
		status = statusBar.Width(m.width).Render(statusText)
	}

	return lipgloss.JoinVertical(lipgloss.Top, row, status, helpView)
}

// toggleReadOnCurrent flips the read state of the article under the cursor
// (articles list) or the one being read (reader focus). No-op otherwise.
func (m Model) toggleReadOnCurrent() (tea.Model, tea.Cmd) {
	var (
		id       int64
		makeRead bool
	)
	switch {
	case m.focus == focusReader && m.readerArt != nil:
		id = m.readerArt.ID
		makeRead = m.readerArt.ReadAt == nil
	case m.focus == focusArticles && len(m.articles) > 0:
		a := m.articles[m.selArt]
		id = a.ID
		makeRead = a.ReadAt == nil
	default:
		return m, nil
	}
	return m, toggleReadCmd(m.db, id, makeRead)
}

func toggleReadCmd(d *db.DB, articleID int64, makeRead bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if makeRead {
			err = d.MarkRead(articleID)
		} else {
			err = d.MarkUnread(articleID)
		}
		if err != nil {
			return errMsg{err}
		}
		return articleMarkedMsg{articleID: articleID, unread: !makeRead}
	}
}

// markAllReadOnCurrentEntry marks every article in the current feed or
// folder read. For folders it dispatches a batch query command reusing
// the existing batchApplyCmd plumbing.
func (m Model) markAllReadOnCurrentEntry() (tea.Model, tea.Cmd) {
	e, ok := m.currentEntry()
	if !ok {
		return m, nil
	}
	switch e.Kind {
	case entryFeed:
		return m, markFeedReadCmd(m.db, e.FeedID)
	case entryFolder:
		q := m.smartFolders[e.FolderIdx].Query
		return m, batchApplyCmd(m.db, q, "read")
	}
	return m, nil
}

func markFeedReadCmd(d *db.DB, feedID int64) tea.Cmd {
	return func() tea.Msg {
		n, err := d.MarkFeedRead(feedID)
		if err != nil {
			return errMsg{err}
		}
		return feedMarkedReadMsg{feedID: feedID, count: n}
	}
}

// toggleStarOnCurrent toggles the starred flag for the article under the
// cursor (in articles list) or the one being read (in reader focus).
func (m Model) toggleStarOnCurrent() (tea.Model, tea.Cmd) {
	var id int64
	switch {
	case m.focus == focusReader && m.readerArt != nil:
		id = m.readerArt.ID
	case m.focus == focusArticles && len(m.articles) > 0:
		id = m.articles[m.selArt].ID
	default:
		return m, nil
	}
	return m, toggleStarCmd(m.db, id)
}

func toggleStarCmd(d *db.DB, articleID int64) tea.Cmd {
	return func() tea.Msg {
		starred, err := d.ToggleStar(articleID)
		if err != nil {
			return errMsg{err}
		}
		return starToggledMsg{articleID: articleID, starred: starred}
	}
}

// jumpToNextUnread advances the selection to the next unread article. If the
// current article list has no unread items past the cursor, it walks the feed
// list forward looking for a feed with unread items and loads it.
func (m Model) jumpToNextUnread() (tea.Model, tea.Cmd) {
	// Within current list first.
	for i := m.selArt + 1; i < len(m.articles); i++ {
		if m.articles[i].ReadAt == nil {
			m.selArt = i
			m.focus = focusArticles
			m.status = "next unread"
			return m, nil
		}
	}
	// Otherwise hop to next feed with unread. Walk through entries() but
	// skip folder rows — "next unread" is a per-feed concept.
	entries := m.entries()
	if len(entries) == 0 {
		m.status = "no unread"
		return m, nil
	}
	for off := 1; off <= len(entries); off++ {
		i := (m.selEntry + off) % len(entries)
		e := entries[i]
		if e.Kind != entryFeed {
			continue
		}
		if e.UnreadCount > 0 {
			m.selEntry = i
			m.selArt = 0
			m.focus = focusArticles
			m.status = "next feed: " + e.Name
			return m, loadArticlesCmd(m.db, e.FeedID, m.filter)
		}
	}
	m.status = "no unread"
	return m, nil
}

// entries returns the unified feed list: smart folders first, then regular
// feeds. Indices are stable within a single Update/View pair. Folder match
// counts come from m.folderCounts (populated by refreshFolderCounts).
func (m Model) entries() []feedEntry {
	out := make([]feedEntry, 0, len(m.smartFolders)+len(m.feeds))
	for i, f := range m.smartFolders {
		count := 0
		if i < len(m.folderCounts) {
			count = m.folderCounts[i]
		}
		out = append(out, feedEntry{
			Kind:        entryFolder,
			Name:        f.Name,
			FolderIdx:   i,
			UnreadCount: count,
		})
	}
	for _, f := range m.feeds {
		_, hasErr := m.feedErrors[f.ID]
		out = append(out, feedEntry{
			Kind:        entryFeed,
			Name:        f.Name,
			FeedID:      f.ID,
			UnreadCount: f.UnreadCount,
			HasError:    hasErr,
		})
	}
	return out
}

// refreshFolderCounts re-evaluates each smart folder's query against the
// cached allArticles set and stores match counts. Cheap even for hundreds
// of folders × thousands of articles because filtering is a single pass.
func (m *Model) refreshFolderCounts() {
	if len(m.smartFolders) == 0 {
		m.folderCounts = nil
		return
	}
	counts := make([]int, len(m.smartFolders))
	// Pre-parse atoms once per folder to avoid re-parsing during the inner
	// article loop.
	parsed := make([][]queryAtom, len(m.smartFolders))
	for i, f := range m.smartFolders {
		atoms, err := ParseQuery(f.Query)
		if err != nil {
			parsed[i] = nil
			continue
		}
		parsed[i] = atoms
	}
	for _, a := range m.allArticles {
		it := db.SearchItem{
			Title:       a.Title,
			FeedName:    a.FeedName,
			Description: a.Description,
			PublishedAt: a.PublishedAt,
			ReadAt:      a.ReadAt,
			StarredAt:   a.StarredAt,
		}
		for i, atoms := range parsed {
			if atoms == nil {
				continue
			}
			if EvalQuery(atoms, it) {
				counts[i]++
			}
		}
	}
	m.folderCounts = counts
}

// currentEntry returns the unified entry at selEntry, or a zero entry if
// selection is out of range (e.g. during the initial render).
func (m Model) currentEntry() (feedEntry, bool) {
	es := m.entries()
	if m.selEntry < 0 || m.selEntry >= len(es) {
		return feedEntry{}, false
	}
	return es[m.selEntry], true
}

// loadCurrentCmd returns the load command appropriate for whatever is
// currently selected — folder or feed. Returns nil if nothing is loadable.
func (m Model) loadCurrentCmd() tea.Cmd {
	e, ok := m.currentEntry()
	if !ok {
		return nil
	}
	if e.Kind == entryFolder {
		return loadFolderArticlesCmd(m.db, e.FolderIdx, m.smartFolders[e.FolderIdx].Query)
	}
	return loadArticlesCmd(m.db, e.FeedID, m.filter)
}

func (m Model) helpView() string {
	if m.showHelp {
		return m.help.View(m.keys)
	}
	return m.help.ShortHelpView(m.keys.ShortHelp())
}

func loadFeedsCmd(d *db.DB) tea.Cmd {
	return func() tea.Msg {
		feeds, err := d.ListFeeds()
		if err != nil {
			return errMsg{err}
		}
		return feedsLoadedMsg{feeds: feeds}
	}
}

func loadArticlesCmd(d *db.DB, feedID int64, filter articleFilter) tea.Cmd {
	return func() tea.Msg {
		articles, err := d.ListArticlesFiltered(feedID, filter, 100)
		if err != nil {
			return errMsg{err}
		}
		return articlesLoadedMsg{feedID: feedID, articles: articles}
	}
}

// loadAllArticlesCmd fetches the cross-feed cache used by smart folder
// match counts. Separate from per-selection loaders so recomputing counts
// doesn't trash the currently displayed article list.
func loadAllArticlesCmd(d *db.DB) tea.Cmd {
	return func() tea.Msg {
		all, err := d.ListAllArticles(2000)
		if err != nil {
			return errMsg{err}
		}
		return allArticlesLoadedMsg{articles: all}
	}
}

// loadFolderArticlesCmd evaluates a smart folder's query against the full
// article set. Parsing + filtering happens in the command goroutine so the
// main loop stays responsive on large DBs.
func loadFolderArticlesCmd(d *db.DB, folderIdx int, query string) tea.Cmd {
	return func() tea.Msg {
		atoms, err := ParseQuery(query)
		if err != nil {
			return errMsg{fmt.Errorf("folder query: %w", err)}
		}
		all, err := d.ListAllArticles(2000)
		if err != nil {
			return errMsg{err}
		}
		out := make([]db.Article, 0, len(all))
		for _, a := range all {
			// Reuse the search-item evaluator by mapping required fields.
			it := db.SearchItem{
				Title:       a.Title,
				FeedName:    a.FeedName,
				Description: a.Description,
				PublishedAt: a.PublishedAt,
				ReadAt:      a.ReadAt,
				StarredAt:   a.StarredAt,
			}
			if EvalQuery(atoms, it) {
				out = append(out, a)
			}
		}
		return folderArticlesLoadedMsg{folderIdx: folderIdx, articles: out}
	}
}

func fetchAllCmd(f *feed.Fetcher) tea.Cmd {
	return func() tea.Msg {
		results, err := f.FetchAll(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return fetchDoneMsg{results: results}
	}
}

func markReadCmd(d *db.DB, articleID int64) tea.Cmd {
	return func() tea.Msg {
		if err := d.MarkRead(articleID); err != nil {
			return errMsg{err}
		}
		return articleMarkedMsg{articleID: articleID}
	}
}

func fetchFullCmd(f *feed.Fetcher, d *db.DB, articleID int64, articleURL string) tea.Cmd {
	return func() tea.Msg {
		md, err := f.FetchFull(context.Background(), articleURL)
		if err != nil {
			return errMsg{err}
		}
		if err := d.CacheArticle(articleID, md); err != nil {
			return errMsg{err}
		}
		return fullArticleLoadedMsg{articleID: articleID, markdown: md}
	}
}
