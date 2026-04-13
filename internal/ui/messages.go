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
}

type errMsg struct {
	err error
}
