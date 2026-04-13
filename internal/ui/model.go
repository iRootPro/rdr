package ui

import (
	"context"
	"fmt"
	"path/filepath"
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
	"github.com/iRootPro/rdr/internal/kitty"
)

type focus int

const (
	focusFeeds focus = iota
	focusArticles
	focusReader
	focusSettings
)

type settingsMode int

const (
	smList settingsMode = iota
	smAddName
	smAddURL
	smRename
)

type Model struct {
	db      *db.DB
	fetcher *feed.Fetcher
	keys    keyMap

	feeds    []db.Feed
	articles []db.Article
	selFeed  int
	selArt   int
	focus    focus

	width  int
	height int

	spin     spinner.Model
	fetching bool

	reader    viewport.Model
	readerArt *db.Article

	help     help.Model
	showHelp bool

	feedErrors map[int64]error

	kittyOn    bool
	imageCache string

	settingsMode  settingsMode
	settingsSel   int
	settingsInput textinput.Model
	pendingName   string

	status string
	err    error
}

func New(database *db.DB, fetcher *feed.Fetcher, home string) Model {
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

	return Model{
		db:       database,
		fetcher:  fetcher,
		keys:     defaultKeys(),
		status:   "fetching…",
		spin:     s,
		fetching: true,
		reader:        viewport.New(0, 0),
		help:          h,
		feedErrors:    map[int64]error{},
		settingsInput: ti,
		kittyOn:       kitty.IsSupported(),
		imageCache:    filepath.Join(home, "cache", "images"),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadFeedsCmd(m.db),
		fetchAllCmd(m.fetcher),
		m.spin.Tick,
	)
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
			m.reader.SetContent(buildReaderContent(*m.readerArt, feedName, m.reader.Width-4, m.kittyOn, m.imageCache))
		}
		return m, nil

	case tea.KeyMsg:
		if m.focus == focusSettings {
			return m.updateSettings(msg)
		}
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Settings):
			m.focus = focusSettings
			m.settingsMode = smList
			m.settingsSel = 0
			return m, nil
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			m.help.ShowAll = m.showHelp
			return m, nil
		case key.Matches(msg, m.keys.FullArticle):
			if m.focus == focusReader && m.readerArt != nil && m.readerArt.URL != "" && !m.fetching {
				m.fetching = true
				m.status = "loading full…"
				return m, tea.Batch(
					fetchFullCmd(m.fetcher, m.db, m.readerArt.ID, m.readerArt.URL, m.imageCache),
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
					cmds = append(cmds, loadArticlesCmd(m.db, m.feeds[m.selFeed].ID))
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
		cmds := []tea.Cmd{loadFeedsCmd(m.db)}
		if len(m.feeds) > 0 {
			cmds = append(cmds, loadArticlesCmd(m.db, m.feeds[m.selFeed].ID))
		}
		return m, tea.Batch(cmds...)

	case feedsLoadedMsg:
		m.feeds = msg.feeds
		m.err = nil
		if !m.fetching {
			m.status = "ready"
		}
		if len(m.feeds) > 0 {
			if m.selFeed >= len(m.feeds) {
				m.selFeed = 0
			}
			return m, loadArticlesCmd(m.db, m.feeds[m.selFeed].ID)
		}
		return m, nil

	case articlesLoadedMsg:
		m.err = nil
		if len(m.feeds) > 0 && m.feeds[m.selFeed].ID == msg.feedID {
			m.articles = msg.articles
			if m.selArt >= len(m.articles) {
				m.selArt = 0
			}
		}
		return m, nil

	case articleMarkedMsg:
		m.err = nil
		return m, nil

	case fullArticleLoadedMsg:
		m.fetching = false
		m.status = "full article"
		if m.readerArt != nil && m.readerArt.ID == msg.articleID {
			m.readerArt.CachedBody = msg.markdown
			now := time.Now().UTC()
			m.readerArt.CachedAt = &now
			feedName := readerFeedName(m.feeds, m.readerArt.FeedID)
			m.reader.SetContent(buildReaderContent(*m.readerArt, feedName, m.reader.Width-4, m.kittyOn, m.imageCache))
			m.reader.GotoTop()
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
	m.reader.SetContent(buildReaderContent(a, feedName, m.reader.Width-4, m.kittyOn, m.imageCache))
	m.reader.GotoTop()
	if a.ReadAt == nil {
		return m, markReadCmd(m.db, a.ID)
	}
	return m, nil
}

func (m Model) moveDown() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusFeeds:
		if m.selFeed < len(m.feeds)-1 {
			m.selFeed++
			return m, loadArticlesCmd(m.db, m.feeds[m.selFeed].ID)
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
		if m.selFeed > 0 {
			m.selFeed--
			return m, loadArticlesCmd(m.db, m.feeds[m.selFeed].ID)
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
		if idx < 0 || idx >= len(m.feeds) {
			return m, nil
		}
		m.selFeed = idx
		return m, loadArticlesCmd(m.db, m.feeds[m.selFeed].ID)
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
		return m.moveTo(len(m.feeds) - 1)
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
		return m.moveTo(clamp(m.selFeed+dir*step, 0, len(m.feeds)-1))
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

	if m.focus == focusReader && m.readerArt != nil {
		statusText := "rdr · reader"
		if m.err != nil {
			statusText += "  " + errStyle.Render("! "+m.err.Error())
		}
		status := statusBar.Width(m.width).Render(statusText)
		body := paneActive.Width(m.width - 2).Height(m.height - 2 - helpH).Render(m.reader.View())
		return lipgloss.JoinVertical(lipgloss.Top, body, status, helpView)
	}

	leftW := m.width/3 - 2
	if leftW < 10 {
		leftW = 10
	}
	rightW := m.width - leftW - 4
	if rightW < 10 {
		rightW = 10
	}
	paneH := m.height - 2 - helpH

	left := renderFeedList(m.feeds, m.feedErrors, m.selFeed, m.focus == focusFeeds, leftW, paneH)
	right := renderArticleList(m.articles, m.selArt, m.focus == focusArticles, rightW, paneH)

	row := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	statusText := "rdr · " + m.status
	if m.fetching {
		statusText = "rdr · " + m.spin.View() + " " + m.status
	}
	if m.err != nil {
		statusText += "  " + errStyle.Render("! "+m.err.Error())
	}
	status := statusBar.Width(m.width).Render(statusText)

	return lipgloss.JoinVertical(lipgloss.Top, row, status, helpView)
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

func loadArticlesCmd(d *db.DB, feedID int64) tea.Cmd {
	return func() tea.Msg {
		articles, err := d.ListArticles(feedID, 100)
		if err != nil {
			return errMsg{err}
		}
		return articlesLoadedMsg{feedID: feedID, articles: articles}
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

func fetchFullCmd(f *feed.Fetcher, d *db.DB, articleID int64, articleURL, imageCache string) tea.Cmd {
	return func() tea.Msg {
		md, err := f.FetchFull(context.Background(), articleURL)
		if err != nil {
			return errMsg{err}
		}
		if err := d.CacheArticle(articleID, md); err != nil {
			return errMsg{err}
		}
		if imageCache != "" {
			if urls := imageURLs(md); len(urls) > 0 {
				_, _ = feed.DownloadImages(context.Background(), urls, imageCache)
			}
		}
		return fullArticleLoadedMsg{articleID: articleID, markdown: md}
	}
}
