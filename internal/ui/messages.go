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

type bookmarkToggledMsg struct {
	articleID  int64
	bookmarked bool
}

type aiResultMsg struct {
	kind    string // "translate" | "summarize"
	content string
}

type aiErrorMsg struct {
	err error
}

// librarySavedMsg is delivered after a URL has been inserted into the
// Library feed via the AddURL modal. The handler refreshes lists and
// kicks off the background full-content fetch.
type librarySavedMsg struct {
	articleID int64
	url       string
}

// libraryFetchedMsg is delivered when the background readability fetch
// for a saved URL completes. err is non-nil on failure; on success
// title is whatever readability resolved (may still be empty).
type libraryFetchedMsg struct {
	articleID int64
	title     string
	err       error
}

// libraryDeletedMsg is delivered after a saved URL has been removed
// from the DB. Triggers a refresh of the cross-feed cache and feeds
// list so unread counts stay in sync.
type libraryDeletedMsg struct {
	articleID int64
}
