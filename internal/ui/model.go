package ui

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
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

	status string
	err    error
}

func New(database *db.DB, fetcher *feed.Fetcher) Model {
	return Model{
		db:      database,
		fetcher: fetcher,
		keys:    defaultKeys(),
		status:  "loading…",
	}
}

func (m Model) Init() tea.Cmd {
	return loadFeedsCmd(m.db)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

	case feedsLoadedMsg:
		m.feeds = msg.feeds
		m.status = "ready"
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return errStyle.Render("error: "+m.err.Error()) + "\n"
	}
	if m.width == 0 || m.height == 0 {
		return "rdr — " + m.status
	}

	leftW := m.width/3 - 2
	if leftW < 10 {
		leftW = 10
	}
	paneH := m.height - 2

	left := renderFeedList(m.feeds, m.selFeed, m.focus == focusFeeds, leftW, paneH)

	status := statusBar.Width(m.width).Render("rdr · " + m.status)

	return lipgloss.JoinVertical(lipgloss.Top, left, status)
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
