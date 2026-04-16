package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/feed"
	"github.com/iRootPro/rdr/internal/i18n"
)

// entryKind tags a row in the unified feed list: smart folder, category
// header, or a plain feed grouped under its category.
type entryKind int

const (
	entryFolder entryKind = iota
	entryCategory
	entryFeed
)

// feedEntry is a single row in the Feeds pane. May be a smart folder
// (query-backed virtual), a category header, or a feed under a category.
type feedEntry struct {
	Kind entryKind

	// Feed-only:
	FeedID      int64
	FeedURL     string
	UnreadCount int
	HasError    bool

	// Folder-only:
	FolderIdx int // index into m.smartFolders

	// Category-only:
	CategoryFeedIDs []int64
	Collapsed       bool

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
	focusHelp
	focusCatalog
)

type settingsMode int

const (
	smList settingsMode = iota
	smAddName
	smAddURL
	smRename
	smCategory
	smCategoryPicker
	smImport
	smExport
	smSmartFolderAddName
	smSmartFolderAddQuery
	smSmartFolderEditName
	smSmartFolderEditQuery
	smFolderRename
	smAfterSyncAdd
	smAfterSyncEdit
)

type settingsSection int

const (
	secFeeds settingsSection = iota
	secGeneral
	secFolders
	secSmartFolders
	secAfterSync
)

// availableLangs is the fixed list of languages the General settings pane
// lets the user cycle through. Order matters — this is also the cursor
// index space for the Language selector.
var availableLangs = []i18n.Lang{i18n.EN, i18n.RU}

func langDisplayName(l i18n.Lang) string {
	switch l {
	case i18n.RU:
		return "Русский"
	default:
		return "English"
	}
}

type articleFilter = db.ArticleFilter

const (
	filterAll     = db.FilterAll
	filterUnread  = db.FilterUnread
	filterStarred = db.FilterStarred
)

// batchActionLabel maps the internal batch action id ("read", "unread",
// "star", "unstar") to a short localized label used in the toast.
func batchActionLabel(action string, tr *i18n.Strings) string {
	switch action {
	case "read":
		return tr.Toasts.MarkedRead
	case "unread":
		return tr.Toasts.MarkedUnread
	case "star":
		return tr.Toasts.Starred
	case "unstar":
		return tr.Toasts.Unstarred
	}
	return action
}

func filterLabel(f articleFilter, tr *i18n.Strings) string {
	switch f {
	case filterUnread:
		return tr.Filters.Unread
	case filterStarred:
		return tr.Filters.Starred
	default:
		return tr.Filters.All
	}
}

type Model struct {
	db      *db.DB
	fetcher *feed.Fetcher
	keys    keyMap

	tr   *i18n.Strings
	lang i18n.Lang

	feeds        []db.Feed
	smartFolders []db.SmartFolder
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
	helpPrev focus

	feedErrors map[int64]error

	settingsMode              settingsMode
	settingsSection           settingsSection
	settingsSel               int
	settingsGeneralSel        int
	settingsFolderSel         int
	settingsSmartFolderSel    int
	settingsAfterSyncSel      int
	catalogSel                int
	settingsCategoryPickerSel int
	settingsInput             textinput.Model
	pendingName               string

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
	showPreview       bool
	themeName         string

	// Visual selection mode (vim-style) for the articles pane. When
	// active, j/k auto-extend a range anchored at visualAnchor, and
	// actions (x/m/y/Y) apply to every article in the range instead of
	// just the one under the cursor.
	visualMode   bool
	visualAnchor int
	afterSyncCommands []string
	refreshInterval   time.Duration
	home              string

	// Link picker overlay (reader → press L to open).
	links    []articleLink
	linksSel int

	// Category collapse state keyed by raw category name ("" for Other).
	collapsedCats map[string]bool

	// Toast notifications: a short-lived status chip for batch ops.
	toast     string
	toastID   int
	syncTotal int

	// Vim-style count prefix. Digits accumulate here until the user
	// presses a movement key, which consumes it as a repeat count.
	countPrefix int

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

func New(database *db.DB, fetcher *feed.Fetcher, afterSyncCommands []string, refreshIntervalMinutes int, home string, lang i18n.Lang, showImages bool, sortField string, sortReverse bool, showPreview bool, themeName string) Model {
	smartFolders, _ := database.ListSmartFolders()
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle()

	h := help.New()
	applyHelpModelStyles(&h)

	ti := textinput.New()
	ti.CharLimit = 256
	ti.Prompt = "› "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG)

	si := textinput.New()
	si.CharLimit = 128
	si.Prompt = "› "
	si.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG)

	ci := textinput.New()
	ci.CharLimit = 128
	ci.Prompt = ":"
	ci.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG)

	tr := i18n.For(lang)

	return Model{
		db:       database,
		fetcher:  fetcher,
		tr:       tr,
		lang:     lang,
		keys:     defaultKeys(tr),
		status:   tr.Status.Fetching,
		spin:     s,
		fetching: true,
		reader:        viewport.New(0, 0),
		help:          h,
		feedErrors:        map[int64]error{},
		collapsedCats:     readCollapsedCats(home),
		smartFolders:      smartFolders,
		afterSyncCommands: afterSyncCommands,
		refreshInterval:   time.Duration(refreshIntervalMinutes) * time.Minute,
		home:              home,
		settingsInput:     ti,
		searchInput:       si,
		commandInput:      ci,
		commandHistory:    readHistoryFile(home),
		historyPos:        -1,
		showImages:        showImages,
		sortField:         sortField,
		sortReverse:       sortReverse,
		showPreview:       showPreview,
		themeName:         themeName,
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
			feedURL := readerFeedURL(m.feeds, m.readerArt.FeedID)
			m.reader.SetContent(buildReaderContent(*m.readerArt, feedName, feedURL, m.reader.Width, m.showImages, m.tr))
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
		if m.focus == focusHelp {
			return m.updateHelp(msg)
		}
		if m.focus == focusCatalog {
			return m.updateCatalog(msg)
		}
		// Digit prefixes accumulate vim-style counts for the next movement
		// key. Only consumes bare digits (no modifier). 0 alone is ignored
		// when the buffer is empty (avoids clobbering future "go to top"
		// style bindings).
		if d := digitKey(msg); d >= 0 {
			if d == 0 && m.countPrefix == 0 {
				// Bare 0 with no pending count → no-op for now.
				return m, nil
			}
			m.countPrefix = m.countPrefix*10 + d
			if m.countPrefix > 999 {
				m.countPrefix = 999
			}
			return m, nil
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
		case key.Matches(msg, m.keys.Star):
			return m.toggleStarOnCurrent()
		case key.Matches(msg, m.keys.FilterAll):
			return m.switchFilter(filterAll)
		case key.Matches(msg, m.keys.FilterUnread):
			return m.switchFilter(filterUnread)
		case key.Matches(msg, m.keys.FilterStarred):
			return m.switchFilter(filterStarred)
		case key.Matches(msg, m.keys.NextUnread):
			return m.jumpToNextUnread()
		case keyIs(msg, "p"):
			if m.focus == focusFeeds || m.focus == focusArticles {
				m.showPreview = !m.showPreview
				if m.db != nil {
					_ = m.db.SetShowPreview(m.showPreview)
				}
				label := m.tr.Status.PreviewOff
				if m.showPreview {
					label = m.tr.Status.PreviewOn
				}
				return m, m.showToast(label)
			}
			return m, nil
		case keyIs(msg, "v"):
			if m.focus == focusArticles && len(m.articles) > 0 && !m.visualMode {
				m.visualMode = true
				m.visualAnchor = m.selArt
				return m, nil
			}
			return m, nil
		case key.Matches(msg, m.keys.Command):
			m.commandPrev = m.focus
			m.focus = focusCommand
			m.commandInput.SetValue("")
			m.commandInput.Focus()
			m.commandSuggIdx = 0
			return m, textinput.Blink
		case key.Matches(msg, m.keys.Help):
			m.helpPrev = m.focus
			m.focus = focusHelp
			return m, nil
		case key.Matches(msg, m.keys.FullArticle):
			if m.focus == focusReader && m.readerArt != nil && m.readerArt.URL != "" && !m.fetching {
				tick := m.startBusy(m.tr.Status.LoadingFullArticle)
				return m, tea.Batch(
					fetchFullCmd(m.fetcher, m.db, m.readerArt.ID, m.readerArt.URL),
					tick,
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
		case key.Matches(msg, m.keys.YankURL):
			return m.yankCurrent(false)
		case key.Matches(msg, m.keys.YankMarkdown):
			return m.yankCurrent(true)
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
		case key.Matches(msg, m.keys.ToggleFold):
			// Space toggles category fold in the feeds pane, acts as
			// page-down inside the reader, and is a no-op elsewhere.
			switch m.focus {
			case focusFeeds:
				return m.toggleCategoryFold()
			case focusReader:
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
			// Visual mode swallows esc to cancel selection rather than
			// falling through to the pane-back handler.
			if m.visualMode && m.focus == focusArticles {
				m.visualMode = false
				m.visualAnchor = 0
				return m, nil
			}
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
			m.syncTotal = len(m.feeds)
			label := m.tr.Status.Syncing
			if m.syncTotal > 0 {
				label = fmt.Sprintf(m.tr.Status.SyncingNFmt, m.syncTotal)
			}
			tick := m.startBusy(label)
			return m, tea.Batch(fetchAllCmd(m.fetcher), tick)
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
		var (
			failed int
			added  int
		)
		for _, r := range msg.results {
			if r.Err != nil {
				m.feedErrors[r.Feed.ID] = r.Err
				failed++
			}
			added += r.Added
		}
		var toastMsg string
		switch {
		case failed > 0 && added > 0:
			toastMsg = fmt.Sprintf(m.tr.Toasts.SyncedOkErrFmt, len(msg.results), added, failed)
		case failed > 0:
			toastMsg = fmt.Sprintf(m.tr.Toasts.SyncedErr, len(msg.results), failed)
		case added > 0:
			toastMsg = fmt.Sprintf(m.tr.Toasts.SyncedOk, len(msg.results), added)
		default:
			toastMsg = fmt.Sprintf(m.tr.Toasts.SyncedOkNothing, len(msg.results))
		}
		m.status = m.tr.Status.Ready
		m.syncTotal = 0
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
		cmds = append(cmds, m.showToast(toastMsg))
		return m, tea.Batch(cmds...)

	case feedsLoadedMsg:
		m.feeds = msg.feeds
		m.err = nil
		if !m.fetching {
			m.status = m.tr.Status.Ready
		}
		total := len(m.entries())
		if total > 0 {
			if m.selEntry >= total {
				m.selEntry = 0
			}
			return m, m.loadCurrentCmd()
		}
		// Onboarding: open catalog on first launch (no feeds).
		if len(m.feeds) == 0 && m.focus == focusFeeds {
			m.focus = focusCatalog
			m.catalogSel = 0
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

	case categoryArticlesLoadedMsg:
		m.err = nil
		e, ok := m.currentEntry()
		if ok && e.Kind == entryCategory && e.Name == msg.name {
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

	case smartFoldersLoadedMsg:
		m.err = nil
		m.smartFolders = msg.folders
		m.refreshFolderCounts()
		// Clamp cursor if the list shrank (e.g. after delete).
		if m.settingsFolderSel >= len(m.smartFolders) && m.settingsFolderSel > 0 {
			m.settingsFolderSel = len(m.smartFolders) - 1
		}
		if m.settingsFolderSel < 0 {
			m.settingsFolderSel = 0
		}
		return m, nil

	case refreshTickMsg:
		// Re-arm the timer unconditionally; either way we want the next
		// tick. Skip the fetch itself if one is already running.
		cmds := []tea.Cmd{}
		if tick := scheduleRefreshCmd(m.refreshInterval); tick != nil {
			cmds = append(cmds, tick)
		}
		if !m.fetching {
			spinTick := m.startBusy(m.tr.Status.AutoFetching)
			cmds = append(cmds, fetchAllCmd(m.fetcher), spinTick)
		}
		return m, tea.Batch(cmds...)

	case batchAppliedMsg:
		m.err = nil
		m.fetching = false
		cmds := []tea.Cmd{loadFeedsCmd(m.db), loadAllArticlesCmd(m.db)}
		if c := m.loadCurrentCmd(); c != nil {
			cmds = append(cmds, c)
		}
		// Only surface a toast when something actually changed. Zero-match
		// batches fire constantly from after_sync_commands and would
		// otherwise clobber the "synced N feeds" toast with a misleading
		// "read · 0 articles".
		if msg.count > 0 {
			cmds = append(cmds, m.showToast(fmt.Sprintf(m.tr.Toasts.BatchFmt, batchActionLabel(msg.action, m.tr), msg.count)))
		}
		return m, tea.Batch(cmds...)

	case toastExpiredMsg:
		if msg.id == m.toastID {
			m.toast = ""
		}
		return m, nil

	case copiedMsg:
		m.err = nil
		m.fetching = false
		return m, m.showToast(fmt.Sprintf(m.tr.Toasts.CopiedFmt, msg.count, msg.format))

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
		label := m.tr.Toasts.MarkedRead
		if msg.unread {
			label = m.tr.Toasts.MarkedUnread
		}
		// Refresh the feed list so unread counters update. The current
		// article list stays as-is (we already patched it) unless the
		// user is on filter=unread, where the row should disappear.
		cmds := []tea.Cmd{loadFeedsCmd(m.db), m.showToast(label)}
		if m.filter == filterUnread && !msg.unread && len(m.feeds) > 0 {
			cmds = append(cmds, m.loadCurrentCmd())
		}
		return m, tea.Batch(cmds...)

	case feedMarkedReadMsg:
		m.err = nil
		cmds := []tea.Cmd{loadFeedsCmd(m.db), loadAllArticlesCmd(m.db)}
		if c := m.loadCurrentCmd(); c != nil {
			cmds = append(cmds, c)
		}
		cmds = append(cmds, m.showToast(fmt.Sprintf(m.tr.Toasts.MarkedReadFmt, msg.count)))
		return m, tea.Batch(cmds...)

	case fullArticleLoadedMsg:
		m.fetching = false
		m.status = "full article"
		if m.readerArt != nil && m.readerArt.ID == msg.articleID {
			m.readerArt.CachedBody = msg.markdown
			now := time.Now().UTC()
			m.readerArt.CachedAt = &now
			feedName := readerFeedName(m.feeds, m.readerArt.FeedID)
			feedURL := readerFeedURL(m.feeds, m.readerArt.FeedID)
			m.reader.SetContent(buildReaderContent(*m.readerArt, feedName, feedURL, m.reader.Width, m.showImages, m.tr))
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
		label := m.tr.Toasts.Starred
		if !msg.starred {
			label = m.tr.Toasts.Unstarred
		}
		cmds := []tea.Cmd{m.showToast(label)}
		// If viewing the starred filter and we just unstarred, reload so the
		// row falls out of the list.
		if m.filter == filterStarred && !msg.starred && len(m.feeds) > 0 {
			cmds = append(cmds, m.loadCurrentCmd())
		}
		return m, tea.Batch(cmds...)

	case errMsg:
		m.err = msg.err
		// Clear the busy flag on any async failure so the spinner
		// doesn't keep spinning after a failed batch/copy/scrape.
		m.fetching = false
		return m, nil
	}
	return m, nil
}

func (m Model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.settingsMode == smCategoryPicker {
		return m.updateSettingsCategoryPicker(msg)
	}
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

	// Global keys that work on every section.
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Settings), key.Matches(msg, m.keys.Back):
		m.focus = focusFeeds
		return m, nil
	case key.Matches(msg, m.keys.Tab):
		m.settingsSection = (m.settingsSection + 1) % 5
		return m, nil
	case key.Matches(msg, m.keys.Help):
		m.helpPrev = m.focus
		m.focus = focusHelp
		return m, nil
	}

	switch m.settingsSection {
	case secGeneral:
		return m.updateSettingsGeneral(msg)
	case secFolders:
		return m.updateSettingsFolders(msg)
	case secSmartFolders:
		return m.updateSettingsSmartFolders(msg)
	case secAfterSync:
		return m.updateSettingsAfterSync(msg)
	default:
		return m.updateSettingsFeeds(msg)
	}
}

func (m Model) updateSettingsFeeds(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
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
	case keyIs(msg, "a"):
		m.settingsMode = smAddName
		m.settingsInput.SetValue("")
		m.settingsInput.Focus()
		return m, textinput.Blink
	case keyIs(msg, "d"):
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
	case keyIs(msg, "e"):
		if len(m.feeds) == 0 {
			return m, nil
		}
		m.settingsMode = smRename
		m.settingsInput.SetValue(m.feeds[m.settingsSel].Name)
		m.settingsInput.CursorEnd()
		m.settingsInput.Focus()
		return m, textinput.Blink
	case keyIs(msg, "c"):
		if len(m.feeds) == 0 {
			return m, nil
		}
		m.settingsMode = smCategoryPicker
		m.settingsCategoryPickerSel = initialCategoryPickerSel(&m)
		return m, nil
	case keyIs(msg, "i"):
		m.settingsMode = smImport
		m.settingsInput.SetValue("")
		m.settingsInput.Focus()
		return m, textinput.Blink
	case keyIs(msg, "E"):
		m.settingsMode = smExport
		m.settingsInput.SetValue("")
		m.settingsInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

// updateSettingsFolders handles keystrokes in the Folders top-level
// section. Regular folders are a derived view over feed.Category; this
// handler operates on uniqueCategories(m.feeds) and uses its own cursor.
func (m Model) updateSettingsFolders(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cats := uniqueCategories(m.feeds)
	switch {
	case key.Matches(msg, m.keys.Down):
		if m.settingsFolderSel < len(cats)-1 {
			m.settingsFolderSel++
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.settingsFolderSel > 0 {
			m.settingsFolderSel--
		}
		return m, nil
	case key.Matches(msg, m.keys.Top):
		m.settingsFolderSel = 0
		return m, nil
	case key.Matches(msg, m.keys.Bottom):
		if len(cats) > 0 {
			m.settingsFolderSel = len(cats) - 1
		}
		return m, nil
	case keyIs(msg, "e"):
		if len(cats) == 0 {
			return m, nil
		}
		m.settingsMode = smFolderRename
		m.settingsInput.SetValue(cats[m.settingsFolderSel])
		m.settingsInput.CursorEnd()
		m.settingsInput.Focus()
		return m, textinput.Blink
	case keyIs(msg, "d"):
		if len(cats) == 0 {
			return m, nil
		}
		if err := m.db.DeleteCategory(cats[m.settingsFolderSel]); err != nil {
			m.err = err
			return m, nil
		}
		m.settingsFolderSel = 0
		return m, loadFeedsCmd(m.db)
	}
	return m, nil
}

// updateSettingsCategoryPicker handles keystrokes while the folder
// picker (opened by `c` on a feed) is active. Navigation is j/k; enter
// applies the selected row, or transitions into the smCategory text
// input when the user picks the "+ New folder…" pseudo-row.
func (m Model) updateSettingsCategoryPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := buildCategoryPickerRows(&m)
	switch {
	case key.Matches(msg, m.keys.Back):
		m.settingsMode = smList
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.settingsCategoryPickerSel < len(rows)-1 {
			m.settingsCategoryPickerSel++
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.settingsCategoryPickerSel > 0 {
			m.settingsCategoryPickerSel--
		}
		return m, nil
	case key.Matches(msg, m.keys.Top):
		m.settingsCategoryPickerSel = 0
		return m, nil
	case key.Matches(msg, m.keys.Bottom):
		m.settingsCategoryPickerSel = len(rows) - 1
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		if m.settingsCategoryPickerSel < 0 || m.settingsCategoryPickerSel >= len(rows) {
			return m, nil
		}
		row := rows[m.settingsCategoryPickerSel]
		if row.IsNew {
			// Route into the existing smCategory text-input path so
			// settingsSubmit applies the new value to the feed exactly
			// the same way as the old direct-prompt flow did.
			m.settingsMode = smCategory
			m.settingsInput.SetValue("")
			m.settingsInput.Focus()
			return m, textinput.Blink
		}
		if len(m.feeds) == 0 || m.settingsSel >= len(m.feeds) {
			return m, nil
		}
		id := m.feeds[m.settingsSel].ID
		if err := m.db.SetFeedCategory(id, row.Value); err != nil {
			m.err = err
			return m, nil
		}
		m.settingsMode = smList
		return m, loadFeedsCmd(m.db)
	}
	return m, nil
}

// initialCategoryPickerSel places the picker cursor on the row
// corresponding to the selected feed's current folder. Falls back to
// the first row ("no folder") if no match is found.
func initialCategoryPickerSel(m *Model) int {
	if len(m.feeds) == 0 || m.settingsSel >= len(m.feeds) {
		return 0
	}
	current := m.feeds[m.settingsSel].Category
	rows := buildCategoryPickerRows(m)
	for i, r := range rows {
		if !r.IsNew && r.Value == current {
			return i
		}
	}
	return 0
}

// uniqueCategories returns a sorted list of non-empty categories present
// in the feed list. "Other" (empty string) is excluded because it is the
// fallback bucket and cannot be meaningfully renamed or deleted.
func uniqueCategories(feeds []db.Feed) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range feeds {
		if f.Category == "" || seen[f.Category] {
			continue
		}
		seen[f.Category] = true
		out = append(out, f.Category)
	}
	sort.Strings(out)
	return out
}

// categoryCounts returns a map of category → number of feeds.
// Only non-empty categories are counted.
func categoryCounts(feeds []db.Feed) map[string]int {
	out := map[string]int{}
	for _, f := range feeds {
		if f.Category == "" {
			continue
		}
		out[f.Category]++
	}
	return out
}

func (m Model) updateSettingsGeneral(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := buildGeneralRows(&m)
	switch {
	case key.Matches(msg, m.keys.Down):
		if m.settingsGeneralSel < len(rows)-1 {
			m.settingsGeneralSel++
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.settingsGeneralSel > 0 {
			m.settingsGeneralSel--
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter), msg.String() == " ":
		return m.cycleGeneralRow()
	}
	return m, nil
}

// cycleGeneralRow applies the "next value" action to the currently
// highlighted row in the General settings list. Each row knows its own
// cycling logic; persistence to the settings DB happens inline so that
// the next startup picks up the chosen value.
func (m Model) cycleGeneralRow() (tea.Model, tea.Cmd) {
	switch m.settingsGeneralSel {
	case 0: // Language
		cur := 0
		for i, l := range availableLangs {
			if l == m.lang {
				cur = i
				break
			}
		}
		target := availableLangs[(cur+1)%len(availableLangs)]
		if target == m.lang {
			return m, nil
		}
		m.lang = target
		m.tr = i18n.For(target)
		m.keys = defaultKeys(m.tr)
		_ = m.db.SetLanguage(string(target))
		return m, m.showToast(m.tr.Toasts.LanguageChanged)

	case 1: // Images
		m.showImages = !m.showImages
		_ = m.db.SetShowImages(m.showImages)
		if m.focus == focusReader && m.readerArt != nil {
			feedName := readerFeedName(m.feeds, m.readerArt.FeedID)
			feedURL := readerFeedURL(m.feeds, m.readerArt.FeedID)
			m.reader.SetContent(buildReaderContent(*m.readerArt, feedName, feedURL, m.reader.Width, m.showImages, m.tr))
		}
		label := m.tr.Status.ImagesOff
		if m.showImages {
			label = m.tr.Status.ImagesOn
		}
		return m, m.showToast(label)

	case 2: // Sort
		switch {
		case m.sortField == "date" && !m.sortReverse:
			m.sortReverse = true
		case m.sortField == "date" && m.sortReverse:
			m.sortField = "title"
			m.sortReverse = false
		case m.sortField == "title" && !m.sortReverse:
			m.sortReverse = true
		default:
			m.sortField = "date"
			m.sortReverse = false
		}
		applySort(m.articles, m.sortField, m.sortReverse)
		_ = m.db.SetSortField(m.sortField)
		_ = m.db.SetSortReverse(m.sortReverse)
		return m, m.showToast(sortDisplayName(m.sortField, m.sortReverse, m.tr))

	case 3: // Preview
		m.showPreview = !m.showPreview
		_ = m.db.SetShowPreview(m.showPreview)
		label := m.tr.Status.PreviewOff
		if m.showPreview {
			label = m.tr.Status.PreviewOn
		}
		return m, m.showToast(label)

	case 4: // Theme
		cur := 0
		for i, t := range availableThemes {
			if t.Name == m.themeName {
				cur = i
				break
			}
		}
		next := availableThemes[(cur+1)%len(availableThemes)]
		m.themeName = next.Name
		applyTheme(next)
		applyHelpModelStyles(&m.help)
		m.settingsInput.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG)
		m.searchInput.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG)
		m.commandInput.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG)
		if m.db != nil {
			_ = m.db.SetTheme(next.Name)
		}
		return m, m.showToast(fmt.Sprintf(m.tr.Toasts.ThemeChangedFmt, next.Name))
	case 5: // Refresh interval
		options := []int{0, 5, 15, 30, 60}
		cur := 0
		mins := int(m.refreshInterval / time.Minute)
		for i, v := range options {
			if v == mins {
				cur = i
				break
			}
		}
		next := options[(cur+1)%len(options)]
		m.refreshInterval = time.Duration(next) * time.Minute
		if m.db != nil {
			_ = m.db.SetRefreshInterval(next)
		}
		label := m.tr.Settings.RefreshOff
		if next > 0 {
			label = fmt.Sprintf(m.tr.Settings.RefreshFmt, next)
		}
		return m, m.showToast(label)
	}
	return m, nil
}

// applyHelpModelStyles copies the current palette into the bubbles
// help.Model so its short/full help bars match the active theme. The
// bubbles model caches Styles, so we rewrite them whenever the theme
// swaps at runtime.
func applyHelpModelStyles(h *help.Model) {
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG)
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG)
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG)
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG)
}

// updateSettingsFolders handles keystrokes while the Folders section of
// the Settings screen is active. Mirrors updateSettingsFeeds: j/k to
// move cursor, a/d/e for CRUD. Delete applies immediately; add and edit
// route through settingsSubmit with a two-step prompt (name → query).
func (m Model) updateSettingsSmartFolders(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Down):
		if m.settingsSmartFolderSel < len(m.smartFolders)-1 {
			m.settingsSmartFolderSel++
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.settingsSmartFolderSel > 0 {
			m.settingsSmartFolderSel--
		}
		return m, nil
	case key.Matches(msg, m.keys.Top):
		m.settingsSmartFolderSel = 0
		return m, nil
	case key.Matches(msg, m.keys.Bottom):
		if len(m.smartFolders) > 0 {
			m.settingsSmartFolderSel = len(m.smartFolders) - 1
		}
		return m, nil
	case keyIs(msg, "a"):
		m.settingsMode = smSmartFolderAddName
		m.settingsInput.SetValue("")
		m.settingsInput.Focus()
		return m, textinput.Blink
	case keyIs(msg, "d"):
		if len(m.smartFolders) == 0 {
			return m, nil
		}
		id := m.smartFolders[m.settingsSmartFolderSel].ID
		if err := m.db.DeleteSmartFolder(id); err != nil {
			m.err = err
			return m, nil
		}
		return m, loadSmartFoldersCmd(m.db)
	case keyIs(msg, "e"):
		if len(m.smartFolders) == 0 {
			return m, nil
		}
		m.settingsMode = smSmartFolderEditName
		m.settingsInput.SetValue(m.smartFolders[m.settingsSmartFolderSel].Name)
		m.settingsInput.CursorEnd()
		m.settingsInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) updateSettingsAfterSync(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Down):
		if m.settingsAfterSyncSel < len(m.afterSyncCommands)-1 {
			m.settingsAfterSyncSel++
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.settingsAfterSyncSel > 0 {
			m.settingsAfterSyncSel--
		}
		return m, nil
	case key.Matches(msg, m.keys.Top):
		m.settingsAfterSyncSel = 0
		return m, nil
	case key.Matches(msg, m.keys.Bottom):
		if len(m.afterSyncCommands) > 0 {
			m.settingsAfterSyncSel = len(m.afterSyncCommands) - 1
		}
		return m, nil
	case keyIs(msg, "a"):
		m.settingsMode = smAfterSyncAdd
		m.settingsInput.SetValue("")
		m.settingsInput.Focus()
		return m, textinput.Blink
	case keyIs(msg, "d"):
		if len(m.afterSyncCommands) == 0 {
			return m, nil
		}
		i := m.settingsAfterSyncSel
		m.afterSyncCommands = append(m.afterSyncCommands[:i], m.afterSyncCommands[i+1:]...)
		_ = m.db.SetAfterSyncCommands(m.afterSyncCommands)
		if m.settingsAfterSyncSel >= len(m.afterSyncCommands) && m.settingsAfterSyncSel > 0 {
			m.settingsAfterSyncSel--
		}
		return m, nil
	case keyIs(msg, "e"):
		if len(m.afterSyncCommands) == 0 {
			return m, nil
		}
		m.settingsMode = smAfterSyncEdit
		m.settingsInput.SetValue(m.afterSyncCommands[m.settingsAfterSyncSel])
		m.settingsInput.CursorEnd()
		m.settingsInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) settingsSubmit() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.settingsInput.Value())
	switch m.settingsMode {
	case smAddName:
		if value == "" {
			return m, nil
		}
		m.pendingName = value
		m.settingsMode = smAddURL
		m.settingsInput.SetValue("")
		return m, textinput.Blink
	case smAddURL:
		if value == "" {
			return m, nil
		}
		if _, err := m.db.UpsertFeed(m.pendingName, value, ""); err != nil {
			m.err = err
			return m, nil
		}
		m.pendingName = ""
		m.settingsMode = smList
		m.settingsInput.Blur()
		m.settingsInput.SetValue("")
		return m, loadFeedsCmd(m.db)
	case smRename:
		if value == "" || len(m.feeds) == 0 {
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
	case smCategory:
		if len(m.feeds) == 0 {
			return m, nil
		}
		id := m.feeds[m.settingsSel].ID
		if err := m.db.SetFeedCategory(id, value); err != nil {
			m.err = err
			return m, nil
		}
		m.settingsMode = smList
		m.settingsInput.Blur()
		m.settingsInput.SetValue("")
		return m, loadFeedsCmd(m.db)
	case smImport:
		if value == "" {
			return m, nil
		}
		path := expandPath(value)
		n, err := importOPMLFile(m.db, path)
		m.settingsMode = smList
		m.settingsInput.Blur()
		m.settingsInput.SetValue("")
		if err != nil {
			m.err = err
			return m, nil
		}
		m.fetching = true
		m.status = m.tr.Status.Fetching
		return m, tea.Batch(
			loadFeedsCmd(m.db),
			fetchAllCmd(m.fetcher),
			m.showToast(fmt.Sprintf(m.tr.Status.ImportedFmt, n)),
			m.spin.Tick,
		)
	case smExport:
		if value == "" {
			return m, nil
		}
		path := expandPath(value)
		n, err := exportOPMLFile(m.db, path)
		m.settingsMode = smList
		m.settingsInput.Blur()
		m.settingsInput.SetValue("")
		if err != nil {
			m.err = err
			return m, nil
		}
		return m, m.showToast(fmt.Sprintf(m.tr.Status.ExportedFmt, n, path))

	case smSmartFolderAddName:
		if value == "" {
			return m, nil
		}
		m.pendingName = value
		m.settingsMode = smSmartFolderAddQuery
		m.settingsInput.SetValue("")
		return m, textinput.Blink

	case smSmartFolderAddQuery:
		if value == "" {
			return m, nil
		}
		if _, err := ParseQuery(value); err != nil {
			m.err = err
			return m, nil
		}
		if _, err := m.db.InsertSmartFolder(m.pendingName, value); err != nil {
			m.err = err
			return m, nil
		}
		m.pendingName = ""
		m.settingsMode = smList
		m.settingsInput.Blur()
		m.settingsInput.SetValue("")
		return m, loadSmartFoldersCmd(m.db)

	case smSmartFolderEditName:
		if value == "" {
			return m, nil
		}
		m.pendingName = value
		m.settingsMode = smSmartFolderEditQuery
		m.settingsInput.SetValue(m.smartFolders[m.settingsFolderSel].Query)
		m.settingsInput.CursorEnd()
		return m, textinput.Blink

	case smSmartFolderEditQuery:
		if value == "" {
			return m, nil
		}
		if _, err := ParseQuery(value); err != nil {
			m.err = err
			return m, nil
		}
		if m.settingsFolderSel >= len(m.smartFolders) {
			return m, nil
		}
		id := m.smartFolders[m.settingsFolderSel].ID
		if err := m.db.UpdateSmartFolder(id, m.pendingName, value); err != nil {
			m.err = err
			return m, nil
		}
		m.pendingName = ""
		m.settingsMode = smList
		m.settingsInput.Blur()
		m.settingsInput.SetValue("")
		return m, loadSmartFoldersCmd(m.db)

	case smFolderRename:
		cats := uniqueCategories(m.feeds)
		if m.settingsFolderSel >= len(cats) {
			return m, nil
		}
		old := cats[m.settingsFolderSel]
		if err := m.db.RenameCategory(old, value); err != nil {
			m.err = err
			return m, nil
		}
		m.settingsMode = smList
		m.settingsInput.Blur()
		m.settingsInput.SetValue("")
		m.settingsFolderSel = 0
		return m, loadFeedsCmd(m.db)

	case smAfterSyncAdd:
		if value == "" {
			return m, nil
		}
		m.afterSyncCommands = append(m.afterSyncCommands, value)
		_ = m.db.SetAfterSyncCommands(m.afterSyncCommands)
		m.settingsMode = smList
		m.settingsInput.Blur()
		m.settingsInput.SetValue("")
		return m, nil

	case smAfterSyncEdit:
		if value == "" {
			return m, nil
		}
		if m.settingsAfterSyncSel < len(m.afterSyncCommands) {
			m.afterSyncCommands[m.settingsAfterSyncSel] = value
			_ = m.db.SetAfterSyncCommands(m.afterSyncCommands)
		}
		m.settingsMode = smList
		m.settingsInput.Blur()
		m.settingsInput.SetValue("")
		return m, nil
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
	feedURL := readerFeedURL(m.feeds, a.FeedID)
	m.reader.SetContent(buildReaderContent(a, feedName, feedURL, m.reader.Width, m.showImages, m.tr))
	m.reader.GotoTop()
	if a.ReadAt == nil {
		return m, markReadCmd(m.db, a.ID)
	}
	return m, nil
}

// readerJump advances selArt by dir within the current article list and
// refreshes reader content without leaving focusReader. Consumes any
// pending count prefix so `3J` jumps forward by 3.
func (m Model) readerJump(dir int) (tea.Model, tea.Cmd) {
	count := m.consumeCount()
	step := dir * count
	target := m.selArt + step
	if target < 0 || target >= len(m.articles) {
		// Clamp to end of list rather than refusing entirely.
		if target < 0 {
			target = 0
		}
		if target >= len(m.articles) {
			target = len(m.articles) - 1
		}
		if target == m.selArt {
			m.status = m.tr.Status.EndOfList
			return m, nil
		}
	}
	m.selArt = target
	a := m.articles[target]
	m.readerArt = &a
	feedName := readerFeedName(m.feeds, a.FeedID)
	feedURL := readerFeedURL(m.feeds, a.FeedID)
	m.reader.SetContent(buildReaderContent(a, feedName, feedURL, m.reader.Width, m.showImages, m.tr))
	m.reader.GotoTop()
	if a.ReadAt == nil {
		return m, markReadCmd(m.db, a.ID)
	}
	return m, nil
}

func (m Model) moveDown() (tea.Model, tea.Cmd) {
	count := m.consumeCount()
	switch m.focus {
	case focusFeeds:
		total := len(m.entries())
		target := m.selEntry + count
		if target >= total {
			target = total - 1
		}
		if target > m.selEntry {
			m.selEntry = target
			return m, m.loadCurrentCmd()
		}
	case focusArticles:
		target := m.selArt + count
		if target >= len(m.articles) {
			target = len(m.articles) - 1
		}
		if target > m.selArt {
			m.selArt = target
		}
	}
	return m, nil
}

func (m Model) moveUp() (tea.Model, tea.Cmd) {
	count := m.consumeCount()
	switch m.focus {
	case focusFeeds:
		target := m.selEntry - count
		if target < 0 {
			target = 0
		}
		if target < m.selEntry {
			m.selEntry = target
			return m, m.loadCurrentCmd()
		}
	case focusArticles:
		target := m.selArt - count
		if target < 0 {
			target = 0
		}
		if target < m.selArt {
			m.selArt = target
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
		return fmt.Sprintf(m.tr.Status.TinyPrefix, m.status)
	}
	if m.width < 40 || m.height < 10 {
		return m.tr.Status.TooSmall
	}

	helpView := m.helpView()
	helpH := lipgloss.Height(helpView)

	if m.focus == focusSettings {
		body := renderSettings(
			&m,
			m.settingsInput.View(),
			m.width,
			m.height-1-helpH,
		)
		status := renderPowerline([]segment{appSegment(), {Text: m.tr.Status.SettingsCrumb, FG: colorText, BG: colorAltBG}}, m.width)
		return lipgloss.JoinVertical(lipgloss.Top, body, status, helpView)
	}

	if m.focus == focusSearch {
		body := renderSearch(m, m.width, m.height-1-helpH)
		status := renderPowerline([]segment{appSegment(), {Text: m.tr.Status.SearchCrumb, FG: colorText, BG: colorAltBG}}, m.width)
		return lipgloss.JoinVertical(lipgloss.Top, body, status, helpView)
	}

	if m.focus == focusLinks {
		body := renderLinkPicker(m, m.width, m.height-1-helpH)
		status := renderPowerline([]segment{appSegment(), {Text: m.tr.Status.LinksCrumb, FG: colorText, BG: colorAltBG}}, m.width)
		return lipgloss.JoinVertical(lipgloss.Top, body, status, helpView)
	}

	if m.focus == focusHelp {
		body := renderHelpScreen(m, m.width, m.height-1-helpH)
		label := fmt.Sprintf(m.tr.Status.HelpCrumbFmt, focusLabel(m.helpPrev, m.tr))
		status := renderPowerline([]segment{appSegment(), {Text: label, FG: colorText, BG: colorAltBG}}, m.width)
		return lipgloss.JoinVertical(lipgloss.Top, body, status, helpView)
	}

	if m.focus == focusCatalog {
		catH := m.height - 4 - helpH
		body := renderCatalog(m, m.width, catH)
		status := renderPowerline([]segment{appSegment(), {Text: m.tr.Catalog.Crumb, FG: colorText, BG: colorAltBG}}, m.width)
		return lipgloss.JoinVertical(lipgloss.Top, body, status, helpView)
	}

	if m.focus == focusReader && m.readerArt != nil {
		// Breadcrumb: "FeedName / Truncated title" replaces the flat
		// "rdr · reader" so you always know what you're looking at.
		feedName := readerFeedName(m.feeds, m.readerArt.FeedID)
		titleBudget := m.width - lipgloss.Width(feedName) - 20
		if titleBudget < 10 {
			titleBudget = 10
		}
		var status string
		if m.toast != "" {
			toastSeg := segment{Text: m.toast, FG: colorAccent, BG: colorAltBG, Bold: true}
			status = renderPowerline([]segment{appSegment(), toastSeg}, m.width)
		} else if m.fetching {
			status = renderPowerline([]segment{
				appSegment(),
				{Text: m.spin.View() + " " + m.status, FG: colorText, BG: colorAltBG},
			}, m.width)
		} else {
			status = renderPowerline(readerSegments(feedName, m.readerArt.Title, titleBudget), m.width)
		}
		if m.err != nil {
			status = paintLineBG(status+"  "+errStyle.Render("! "+m.err.Error()), m.width)
		}
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

	paneH := m.height - 4 - helpH - popupH

	var row string
	if m.zenMode {
		// Only the focused pane is drawn at full width.
		fullW := m.width - 2
		if fullW < 10 {
			fullW = 10
		}
		entries := m.entries()
		visualLo, visualHi := -1, -1
		articlePreview := m.showPreview
		if m.visualMode {
			visualLo, visualHi = m.visualRange()
			// Hide the preview popup while multi-selecting — it would
			// cover range rows and looks noisy.
			articlePreview = false
		}
		if m.focus == focusFeeds {
			row = renderFeedList(entries, m.selEntry, true, fullW, paneH, m.tr)
		} else {
			row = renderArticleList(m.articles, m.selArt, true, fullW, paneH, m.tr, articlePreview, visualLo, visualHi)
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
		visualLo, visualHi := -1, -1
		articlePreview := m.showPreview
		if m.visualMode {
			visualLo, visualHi = m.visualRange()
			articlePreview = false
		}
		left := renderFeedList(entries, m.selEntry, m.focus == focusFeeds, leftW, paneH, m.tr)
		right := renderArticleList(m.articles, m.selArt, m.focus == focusArticles, rightW, paneH, m.tr, articlePreview, visualLo, visualHi)
		row = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	if m.focus == focusCommand {
		cmdLine := renderPowerline([]segment{
			appSegment(),
			{Text: m.commandInput.View(), FG: colorText, BG: colorAltBG},
		}, m.width)
		return lipgloss.JoinVertical(lipgloss.Top, row, popup, cmdLine, helpView)
	}

	var status string
	if m.toast != "" {
		toastSeg := segment{Text: m.toast, FG: colorAccent, BG: colorAltBG, Bold: true}
		status = renderPowerline([]segment{appSegment(), toastSeg}, m.width)
	} else {
		statusLabel := m.status
		if m.fetching {
			statusLabel = m.spin.View() + " " + m.status
		}
		segs := statusSegments(statusLabel, filterLabel(m.filter, m.tr),
			m.sortField, m.sortReverse, m.zenMode)
		if m.countPrefix > 0 {
			segs = append(segs, segment{
				Text: fmt.Sprintf("%d…", m.countPrefix),
				FG:   colorText,
				BG:   colorBorder,
			})
		}
		status = renderPowerline(segs, m.width)
	}
	if m.err != nil {
		status = paintLineBG(status+"  "+errStyle.Render("! "+m.err.Error()), m.width)
	}

	return lipgloss.JoinVertical(lipgloss.Top, row, status, helpView)
}

// digitKey returns 0-9 when the keystroke is a bare digit, else -1.
// Used by count-prefix parsing. Modifier keys disqualify.
func digitKey(msg tea.KeyMsg) int {
	if msg.Type != tea.KeyRunes || len(msg.Runes) != 1 {
		return -1
	}
	r := msg.Runes[0]
	if r < '0' || r > '9' {
		return -1
	}
	return int(r - '0')
}

// consumeCount returns the pending count prefix (default 1) and clears it.
func (m *Model) consumeCount() int {
	c := m.countPrefix
	m.countPrefix = 0
	if c < 1 {
		c = 1
	}
	return c
}

// toggleCategoryFold collapses or expands the category under the cursor.
// If the cursor is on a feed row, its parent category is folded instead,
// so power users can collapse from inside a group without scrolling up.
func (m Model) toggleCategoryFold() (tea.Model, tea.Cmd) {
	es := m.entries()
	if m.selEntry < 0 || m.selEntry >= len(es) {
		return m, nil
	}
	targetKey := ""
	switch es[m.selEntry].Kind {
	case entryCategory:
		// Look up the raw key for this header via the feeds it owns.
		targetKey = m.categoryKeyFromEntry(es[m.selEntry])
	case entryFeed:
		// Find the parent category by scanning backwards.
		for i := m.selEntry - 1; i >= 0; i-- {
			if es[i].Kind == entryCategory {
				targetKey = m.categoryKeyFromEntry(es[i])
				// Move selection to the header before collapsing so the
				// user doesn't fall out of view.
				m.selEntry = i
				break
			}
		}
	default:
		return m, nil
	}
	if m.collapsedCats == nil {
		m.collapsedCats = map[string]bool{}
	}
	m.collapsedCats[targetKey] = !m.collapsedCats[targetKey]
	writeCollapsedCats(m.home, m.collapsedCats)
	return m, nil
}

// categoryKeyFromEntry returns the raw category string (possibly empty
// for "Other") given a rendered entryCategory. We look up a feed id to
// recover the original value since the display name may be "Other".
func (m Model) categoryKeyFromEntry(e feedEntry) string {
	if e.Kind != entryCategory || len(e.CategoryFeedIDs) == 0 {
		// Best-effort fallback: the header name is the key except when
		// it's the localized "Other" label which maps to the empty
		// category bucket.
		if e.Name == m.tr.Feeds.OtherCategory {
			return ""
		}
		return e.Name
	}
	id := e.CategoryFeedIDs[0]
	for _, f := range m.feeds {
		if f.ID == id {
			return f.Category
		}
	}
	return ""
}

// visualRange returns the (lo, hi) pair of article indices currently
// covered by the visual selection. Always normalized so lo <= hi.
func (m Model) visualRange() (lo, hi int) {
	lo, hi = m.visualAnchor, m.selArt
	if lo > hi {
		lo, hi = hi, lo
	}
	if lo < 0 {
		lo = 0
	}
	if hi >= len(m.articles) {
		hi = len(m.articles) - 1
	}
	return
}

// visualIDs returns the article IDs inside the current visual range.
func (m Model) visualIDs() []int64 {
	lo, hi := m.visualRange()
	if lo > hi {
		return nil
	}
	out := make([]int64, 0, hi-lo+1)
	for i := lo; i <= hi; i++ {
		out = append(out, m.articles[i].ID)
	}
	return out
}

// applyReadOnVisual marks every article in the current visual range as
// read. Exits visual mode on completion.
func (m Model) applyReadOnVisual() (tea.Model, tea.Cmd) {
	ids := m.visualIDs()
	if len(ids) == 0 {
		m.visualMode = false
		return m, nil
	}
	m.visualMode = false
	m.visualAnchor = 0
	return m, tea.Batch(
		bulkMarkReadCmd(m.db, ids),
		m.showToast(fmt.Sprintf(m.tr.Toasts.MarkedReadFmt, len(ids))),
	)
}

// applyStarOnVisual toggles starred on every article in the current
// visual range. Direction is decided by the anchor's current state:
// if the anchor is NOT starred, star all; otherwise unstar all.
func (m Model) applyStarOnVisual() (tea.Model, tea.Cmd) {
	ids := m.visualIDs()
	if len(ids) == 0 || m.visualAnchor >= len(m.articles) {
		m.visualMode = false
		return m, nil
	}
	star := m.articles[m.visualAnchor].StarredAt == nil
	m.visualMode = false
	m.visualAnchor = 0
	label := m.tr.Toasts.Starred
	if !star {
		label = m.tr.Toasts.Unstarred
	}
	return m, tea.Batch(
		bulkSetStarredCmd(m.db, ids, star),
		m.showToast(label),
	)
}

// yankVisual copies URLs (or markdown-formatted links) for every
// article in the current visual range to the clipboard via OSC 52.
func (m Model) yankVisual(markdown bool) (tea.Model, tea.Cmd) {
	lo, hi := m.visualRange()
	if lo > hi {
		m.visualMode = false
		return m, nil
	}
	var lines []string
	for i := lo; i <= hi; i++ {
		a := m.articles[i]
		if a.URL == "" {
			continue
		}
		if markdown {
			title := a.Title
			if title == "" {
				title = a.URL
			}
			lines = append(lines, fmt.Sprintf("[%s](%s)", title, a.URL))
		} else {
			lines = append(lines, a.URL)
		}
	}
	m.visualMode = false
	m.visualAnchor = 0
	if len(lines) == 0 {
		m.err = fmt.Errorf("%s", m.tr.Errors.NoURLToCopy)
		return m, nil
	}
	osc52Copy(strings.Join(lines, "\n"))
	label := m.tr.Toasts.URLCopied
	if markdown {
		label = m.tr.Toasts.MarkdownCopied
	}
	return m, m.showToast(label)
}

// bulkMarkReadCmd runs BulkMarkRead in a goroutine and emits a message
// that triggers a list refresh, mirroring the single-article path.
func bulkMarkReadCmd(d *db.DB, ids []int64) tea.Cmd {
	return func() tea.Msg {
		if err := d.BulkMarkRead(ids); err != nil {
			return errMsg{err}
		}
		return batchAppliedMsg{action: "read", count: len(ids)}
	}
}

// bulkSetStarredCmd runs BulkSetStarred in a goroutine. action is
// "star" or "unstar" so the shared batchAppliedMsg handler surfaces a
// meaningful toast label.
func bulkSetStarredCmd(d *db.DB, ids []int64, starred bool) tea.Cmd {
	return func() tea.Msg {
		if err := d.BulkSetStarred(ids, starred); err != nil {
			return errMsg{err}
		}
		action := "star"
		if !starred {
			action = "unstar"
		}
		return batchAppliedMsg{action: action, count: len(ids)}
	}
}

// yankCurrent copies the selected article's URL to the system clipboard
// via OSC 52. If markdown is true, it copies a `[title](url)` link
// instead. Works from the articles list or the reader.
func (m Model) yankCurrent(markdown bool) (tea.Model, tea.Cmd) {
	if m.visualMode && m.focus == focusArticles {
		return m.yankVisual(markdown)
	}
	var title, url string
	switch {
	case m.focus == focusReader && m.readerArt != nil:
		title = m.readerArt.Title
		url = m.readerArt.URL
	case m.focus == focusArticles && len(m.articles) > 0:
		a := m.articles[m.selArt]
		title = a.Title
		url = a.URL
	default:
		return m, nil
	}
	if url == "" {
		m.err = fmt.Errorf("%s", m.tr.Errors.NoURLToCopy)
		return m, nil
	}
	text := url
	label := m.tr.Toasts.URLCopied
	if markdown {
		text = fmt.Sprintf("[%s](%s)", title, url)
		label = m.tr.Toasts.MarkdownCopied
	}
	osc52Copy(text)
	return m, m.showToast(label)
}

// toggleReadOnCurrent flips the read state of the article under the cursor
// (articles list) or the one being read (reader focus). No-op otherwise.
func (m Model) toggleReadOnCurrent() (tea.Model, tea.Cmd) {
	if m.visualMode && m.focus == focusArticles {
		return m.applyReadOnVisual()
	}
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

// startBusy sets the async-in-progress flag, stashes a human label in
// m.status and returns the spinner tick to kick off rotation. Every
// code path that runs a long tea.Cmd should funnel through this so
// the status bar consistently shows feedback.
func (m *Model) startBusy(label string) tea.Cmd {
	m.fetching = true
	m.status = label
	return m.spin.Tick
}

// showToast overrides the status bar with a short-lived message. The
// returned Cmd schedules a tea.Tick after 2s to clear it (if no newer
// toast has replaced it). Mutates m in place via pointer receiver.
func (m *Model) showToast(text string) tea.Cmd {
	m.toastID++
	m.toast = text
	id := m.toastID
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return toastExpiredMsg{id: id}
	})
}

// switchFilter changes the article filter. If the user is already in the
// target mode it's a no-op (no toast, no reload), otherwise the current
// selection's article list is refetched and a toast confirms the switch.
func (m Model) switchFilter(target articleFilter) (tea.Model, tea.Cmd) {
	if m.filter == target {
		return m, nil
	}
	m.filter = target
	m.selArt = 0
	label := fmt.Sprintf(m.tr.Toasts.ShowingFmt, filterLabel(target, m.tr))
	cmds := []tea.Cmd{m.showToast(label)}
	if len(m.feeds) > 0 {
		if c := m.loadCurrentCmd(); c != nil {
			cmds = append(cmds, c)
		}
	}
	return m, tea.Batch(cmds...)
}

// toggleStarOnCurrent toggles the starred flag for the article under the
// cursor (in articles list) or the one being read (in reader focus).
func (m Model) toggleStarOnCurrent() (tea.Model, tea.Cmd) {
	if m.visualMode && m.focus == focusArticles {
		return m.applyStarOnVisual()
	}
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
			m.status = m.tr.Status.NextUnread
			return m, nil
		}
	}
	// Otherwise hop to next feed with unread. Walk through entries() but
	// skip folder rows — "next unread" is a per-feed concept.
	entries := m.entries()
	if len(entries) == 0 {
		m.status = m.tr.Status.NoUnread
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
			m.status = fmt.Sprintf(m.tr.Status.NextFeedFmt, e.Name)
			return m, loadArticlesCmd(m.db, e.FeedID, m.filter)
		}
	}
	m.status = m.tr.Status.NoUnread
	return m, nil
}

// entries returns the unified feed list: smart folders first, then
// category headers followed by their feeds. Collapsed categories hide
// their feeds but keep the header visible. "Other" (empty category
// string) is pinned to the bottom.
func (m Model) entries() []feedEntry {
	out := make([]feedEntry, 0, len(m.smartFolders)+len(m.feeds)+8)
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

	// Bucket feeds by category preserving first-seen order.
	type group struct {
		feeds []db.Feed
	}
	var order []string
	groups := make(map[string]*group)
	for _, f := range m.feeds {
		if _, ok := groups[f.Category]; !ok {
			groups[f.Category] = &group{}
			order = append(order, f.Category)
		}
		groups[f.Category].feeds = append(groups[f.Category].feeds, f)
	}
	// Pin the empty category to the bottom as "Other".
	for i, k := range order {
		if k == "" {
			order = append(order[:i], order[i+1:]...)
			order = append(order, "")
			break
		}
	}

	for _, key := range order {
		g := groups[key]
		displayName := key
		if displayName == "" {
			displayName = m.tr.Feeds.OtherCategory
		}
		ids := make([]int64, 0, len(g.feeds))
		unreadTotal := 0
		for _, f := range g.feeds {
			ids = append(ids, f.ID)
			unreadTotal += f.UnreadCount
		}
		collapsed := m.collapsedCats[key]
		out = append(out, feedEntry{
			Kind:            entryCategory,
			Name:            displayName,
			CategoryFeedIDs: ids,
			Collapsed:       collapsed,
			UnreadCount:     unreadTotal,
		})
		if collapsed {
			continue
		}
		for _, f := range g.feeds {
			_, hasErr := m.feedErrors[f.ID]
			out = append(out, feedEntry{
				Kind:        entryFeed,
				Name:        f.Name,
				FeedID:      f.ID,
				FeedURL:     f.URL,
				UnreadCount: f.UnreadCount,
				HasError:    hasErr,
			})
		}
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
// currently selected — folder, category, or plain feed.
func (m Model) loadCurrentCmd() tea.Cmd {
	e, ok := m.currentEntry()
	if !ok {
		return nil
	}
	switch e.Kind {
	case entryFolder:
		return loadFolderArticlesCmd(m.db, e.FolderIdx, m.smartFolders[e.FolderIdx].Query)
	case entryCategory:
		return loadCategoryArticlesCmd(m.db, e.Name, e.CategoryFeedIDs)
	}
	return loadArticlesCmd(m.db, e.FeedID, m.filter)
}

func (m Model) helpView() string {
	raw := m.help.ShortHelpView(shortHelpFor(m.focus, m.keys))
	return paintLineBG("  "+raw, m.width)
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

// loadCategoryArticlesCmd loads articles belonging to the feeds of a
// category. Uses ListAllArticles (cross-feed cache SQL) and filters in
// memory to a feed-id set — no new DB method required.
func loadCategoryArticlesCmd(d *db.DB, name string, feedIDs []int64) tea.Cmd {
	return func() tea.Msg {
		all, err := d.ListAllArticles(2000)
		if err != nil {
			return errMsg{err}
		}
		idSet := make(map[int64]bool, len(feedIDs))
		for _, id := range feedIDs {
			idSet[id] = true
		}
		out := make([]db.Article, 0, len(all))
		for _, a := range all {
			if idSet[a.FeedID] {
				out = append(out, a)
			}
		}
		return categoryArticlesLoadedMsg{name: name, articles: out}
	}
}

// loadSmartFoldersCmd fetches the smart folder list from the DB. Used
// after CRUD operations in the Folders settings section to refresh the
// in-memory slice and the counter cache.
func loadSmartFoldersCmd(d *db.DB) tea.Cmd {
	return func() tea.Msg {
		folders, err := d.ListSmartFolders()
		if err != nil {
			return errMsg{err}
		}
		return smartFoldersLoadedMsg{folders: folders}
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
