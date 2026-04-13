package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/feed"
)

type focus int

const (
	focusFeeds focus = iota
	focusArticles
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

	status string
	err    error
}

func New(database *db.DB, fetcher *feed.Fetcher) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorAccent)
	return Model{
		db:       database,
		fetcher:  fetcher,
		keys:     defaultKeys(),
		status:   "fetching…",
		spin:     s,
		fetching: true,
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
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Tab):
			if m.focus == focusFeeds {
				m.focus = focusArticles
			} else {
				m.focus = focusFeeds
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			return m.moveDown()
		case key.Matches(msg, m.keys.Up):
			return m.moveUp()
		case key.Matches(msg, m.keys.Right), key.Matches(msg, m.keys.Enter):
			if m.focus == focusFeeds && len(m.articles) > 0 {
				m.focus = focusArticles
			}
			return m, nil
		case key.Matches(msg, m.keys.Left), key.Matches(msg, m.keys.Back):
			if m.focus == focusArticles {
				m.focus = focusFeeds
			}
			return m, nil
		case key.Matches(msg, m.keys.Top):
			return m.moveTo(0)
		case key.Matches(msg, m.keys.Bottom):
			return m.moveToEnd()
		case key.Matches(msg, m.keys.PageDown):
			return m.moveByPage(+1)
		case key.Matches(msg, m.keys.PageUp):
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
		var failed int
		for _, r := range msg.results {
			if r.Err != nil {
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
			m.selArt = 0
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
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

	leftW := m.width/3 - 2
	if leftW < 10 {
		leftW = 10
	}
	rightW := m.width - leftW - 4
	if rightW < 10 {
		rightW = 10
	}
	paneH := m.height - 2

	left := renderFeedList(m.feeds, m.selFeed, m.focus == focusFeeds, leftW, paneH)
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

	return lipgloss.JoinVertical(lipgloss.Top, row, status)
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
