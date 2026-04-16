package ui

import (
	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/feed"
)

type feedsLoadedMsg struct {
	feeds []db.Feed
}

type articlesLoadedMsg struct {
	feedID   int64
	articles []db.Article
}

type fetchDoneMsg struct {
	results []feed.FetchResult
}

type articleMarkedMsg struct {
	articleID int64
	unread    bool // true when this was a mark-unread op
}

type feedMarkedReadMsg struct {
	feedID int64
	count  int
}

type fullArticleLoadedMsg struct {
	articleID int64
	markdown  string
}

type searchLoadedMsg struct {
	items []db.SearchItem
}

type starToggledMsg struct {
	articleID int64
	starred   bool
}

type folderArticlesLoadedMsg struct {
	folderIdx int
	articles  []db.Article
}

type categoryArticlesLoadedMsg struct {
	name     string
	articles []db.Article
}

type allArticlesLoadedMsg struct {
	articles []db.Article
}

type smartFoldersLoadedMsg struct {
	folders []db.SmartFolder
}

type batchAppliedMsg struct {
	action string // "read" | "unread" | "star" | "unstar"
	count  int
}

// refreshTickMsg fires from the background tea.Tick when it is time to
// auto-sync. The handler re-arms the timer after kicking off a fetch.
type refreshTickMsg struct{}

// toastExpiredMsg is delivered 2s after a toast is shown. The handler
// only clears the toast if the id still matches the currently visible
// one — otherwise a newer toast has replaced it and we leave it alone.
type toastExpiredMsg struct {
	id int
}

type copiedMsg struct {
	count  int
	format string
}

type errMsg struct {
	err error
}

type aiResultMsg struct {
	kind    string // "translate" | "summarize"
	content string
}

type aiErrorMsg struct {
	err error
}
