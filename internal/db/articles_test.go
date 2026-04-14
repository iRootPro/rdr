package db

import (
	"testing"
	"time"
)

func newArticle(feedID int64, url, title string, published time.Time) Article {
	return Article{
		FeedID:      feedID,
		Title:       title,
		URL:         url,
		Description: "desc",
		Content:     "<p>body</p>",
		PublishedAt: published,
	}
}

func TestUpsertArticle_InsertThenUpdatePreservesReadAt(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss")
	now := time.Now().UTC().Truncate(time.Second)

	a := newArticle(f.ID, "https://a.example/1", "v1", now)
	if _, err := d.UpsertArticle(a); err != nil {
		t.Fatalf("insert: %v", err)
	}

	list, err := d.ListArticles(f.ID, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Title != "v1" {
		t.Fatalf("unexpected list: %+v", list)
	}

	if err := d.MarkRead(list[0].ID); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}

	a.Title = "v2"
	if _, err := d.UpsertArticle(a); err != nil {
		t.Fatalf("update: %v", err)
	}

	list, _ = d.ListArticles(f.ID, 10)
	if len(list) != 1 {
		t.Fatalf("expected 1 row after update, got %d", len(list))
	}
	if list[0].Title != "v2" {
		t.Fatalf("title not updated: %q", list[0].Title)
	}
	if list[0].ReadAt == nil {
		t.Fatalf("read_at was cleared by upsert, must be preserved")
	}
}

func TestListArticles_OrdersByPublishedDesc(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss")
	base := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	_, _ = d.UpsertArticle(newArticle(f.ID, "https://a.example/1", "older", base))
	_, _ = d.UpsertArticle(newArticle(f.ID, "https://a.example/2", "newer", base.Add(time.Hour)))
	_, _ = d.UpsertArticle(newArticle(f.ID, "https://a.example/3", "middle", base.Add(30*time.Minute)))

	list, err := d.ListArticles(f.ID, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	gotTitles := []string{list[0].Title, list[1].Title, list[2].Title}
	want := []string{"newer", "middle", "older"}
	for i := range want {
		if gotTitles[i] != want[i] {
			t.Fatalf("order: got %v, want %v", gotTitles, want)
		}
	}
}

func TestTrimArticles_RemovesOldestReadBeyondLimit(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss")
	base := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		a := newArticle(f.ID, urlN(i), "read", base.Add(time.Duration(i)*time.Minute))
		_, _ = d.UpsertArticle(a)
	}
	for i := 3; i < 5; i++ {
		a := newArticle(f.ID, urlN(i), "unread", base.Add(time.Duration(i)*time.Minute))
		_, _ = d.UpsertArticle(a)
	}
	list, _ := d.ListArticles(f.ID, 10)
	for _, a := range list {
		if a.Title == "read" {
			_ = d.MarkRead(a.ID)
		}
	}

	if err := d.TrimArticles(f.ID, 3); err != nil {
		t.Fatalf("TrimArticles: %v", err)
	}

	list, _ = d.ListArticles(f.ID, 10)
	if len(list) != 3 {
		t.Fatalf("want 3 remaining, got %d", len(list))
	}
	var unread int
	for _, a := range list {
		if a.ReadAt == nil {
			unread++
		}
	}
	if unread != 2 {
		t.Fatalf("want 2 unread remaining, got %d", unread)
	}
}

func urlN(i int) string {
	return "https://a.example/" + string(rune('0'+i))
}

func TestUpsertArticle_ReturnsInsertedFlag(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss")
	a := newArticle(f.ID, "https://a.example/1", "v1", time.Now().UTC())

	inserted, err := d.UpsertArticle(a)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if !inserted {
		t.Fatalf("first upsert: expected inserted=true")
	}

	inserted, err = d.UpsertArticle(a)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if inserted {
		t.Fatalf("second upsert: expected inserted=false")
	}
}

func TestListArticlesFiltered_UnreadOnly(t *testing.T) {
	d := openTestDB(t)
	f, err := d.UpsertFeed("A", "https://a.example/rss")
	if err != nil {
		t.Fatalf("feed: %v", err)
	}
	base := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		a := newArticle(f.ID, urlN(i), "t"+string(rune('0'+i)), base.Add(time.Duration(i)*time.Minute))
		if _, err := d.UpsertArticle(a); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}
	list, err := d.ListArticles(f.ID, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// Mark two as read.
	if err := d.MarkRead(list[0].ID); err != nil {
		t.Fatalf("mark 0: %v", err)
	}
	if err := d.MarkRead(list[2].ID); err != nil {
		t.Fatalf("mark 2: %v", err)
	}

	all, err := d.ListArticlesFiltered(f.ID, FilterAll, 10)
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("all=%d, want 4", len(all))
	}

	unread, err := d.ListArticlesFiltered(f.ID, FilterUnread, 10)
	if err != nil {
		t.Fatalf("unread: %v", err)
	}
	if len(unread) != 2 {
		t.Fatalf("unread=%d, want 2", len(unread))
	}
	for _, a := range unread {
		if a.ReadAt != nil {
			t.Fatalf("unread list contains read article: %+v", a)
		}
	}
}

func TestToggleStar_TogglesAndPersistsTimestamp(t *testing.T) {
	d := openTestDB(t)
	f, err := d.UpsertFeed("A", "https://a.example/rss")
	if err != nil {
		t.Fatalf("feed: %v", err)
	}
	a := newArticle(f.ID, "https://a.example/1", "v1", time.Now().UTC())
	if _, err := d.UpsertArticle(a); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	list, err := d.ListArticles(f.ID, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	id := list[0].ID
	if list[0].StarredAt != nil {
		t.Fatalf("pre-toggle StarredAt should be nil")
	}

	starred, err := d.ToggleStar(id)
	if err != nil {
		t.Fatalf("toggle 1: %v", err)
	}
	if !starred {
		t.Fatalf("first toggle should return starred=true")
	}
	list, _ = d.ListArticles(f.ID, 10)
	if list[0].StarredAt == nil {
		t.Fatalf("StarredAt nil after first toggle")
	}

	starred, err = d.ToggleStar(id)
	if err != nil {
		t.Fatalf("toggle 2: %v", err)
	}
	if starred {
		t.Fatalf("second toggle should return starred=false")
	}
	list, _ = d.ListArticles(f.ID, 10)
	if list[0].StarredAt != nil {
		t.Fatalf("StarredAt should be nil after second toggle")
	}
}

func TestListArticlesFiltered_StarredOnly(t *testing.T) {
	d := openTestDB(t)
	f, err := d.UpsertFeed("A", "https://a.example/rss")
	if err != nil {
		t.Fatalf("feed: %v", err)
	}
	base := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		a := newArticle(f.ID, urlN(i), "t", base.Add(time.Duration(i)*time.Minute))
		if _, err := d.UpsertArticle(a); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}
	list, _ := d.ListArticles(f.ID, 10)
	// Star two articles.
	if _, err := d.ToggleStar(list[0].ID); err != nil {
		t.Fatalf("star 0: %v", err)
	}
	if _, err := d.ToggleStar(list[2].ID); err != nil {
		t.Fatalf("star 2: %v", err)
	}

	starred, err := d.ListArticlesFiltered(f.ID, FilterStarred, 10)
	if err != nil {
		t.Fatalf("filter starred: %v", err)
	}
	if len(starred) != 2 {
		t.Fatalf("want 2 starred, got %d", len(starred))
	}
	for _, a := range starred {
		if a.StarredAt == nil {
			t.Fatalf("starred result has nil StarredAt: %+v", a)
		}
	}
}

func TestBulkMarkRead_MarksOnlyUnread(t *testing.T) {
	d := openTestDB(t)
	f, err := d.UpsertFeed("A", "https://a.example/rss")
	if err != nil {
		t.Fatalf("feed: %v", err)
	}
	base := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		a := newArticle(f.ID, urlN(i), "t", base.Add(time.Duration(i)*time.Minute))
		if _, err := d.UpsertArticle(a); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}
	list, _ := d.ListArticles(f.ID, 10)
	ids := []int64{list[0].ID, list[1].ID, list[2].ID}

	if err := d.BulkMarkRead(ids); err != nil {
		t.Fatalf("BulkMarkRead: %v", err)
	}
	after, _ := d.ListArticles(f.ID, 10)
	for _, a := range after {
		if a.ReadAt == nil {
			t.Fatalf("article %d not marked read", a.ID)
		}
	}
}

func TestBulkMarkUnread_ClearsReadAt(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss")
	a := newArticle(f.ID, "https://a.example/1", "v", time.Now().UTC())
	if _, err := d.UpsertArticle(a); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	list, _ := d.ListArticles(f.ID, 10)
	id := list[0].ID
	if err := d.MarkRead(id); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if err := d.BulkMarkUnread([]int64{id}); err != nil {
		t.Fatalf("BulkMarkUnread: %v", err)
	}
	list, _ = d.ListArticles(f.ID, 10)
	if list[0].ReadAt != nil {
		t.Fatalf("BulkMarkUnread did not clear read_at")
	}
}

func TestBulkSetStarred_BothDirections(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss")
	for i := 0; i < 2; i++ {
		a := newArticle(f.ID, urlN(i), "t", time.Now().UTC())
		if _, err := d.UpsertArticle(a); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}
	list, _ := d.ListArticles(f.ID, 10)
	ids := []int64{list[0].ID, list[1].ID}

	if err := d.BulkSetStarred(ids, true); err != nil {
		t.Fatalf("BulkSetStarred true: %v", err)
	}
	list, _ = d.ListArticles(f.ID, 10)
	for _, a := range list {
		if a.StarredAt == nil {
			t.Fatalf("article %d not starred", a.ID)
		}
	}

	if err := d.BulkSetStarred(ids, false); err != nil {
		t.Fatalf("BulkSetStarred false: %v", err)
	}
	list, _ = d.ListArticles(f.ID, 10)
	for _, a := range list {
		if a.StarredAt != nil {
			t.Fatalf("article %d still starred", a.ID)
		}
	}
}

func TestBulkMarkRead_EmptyIsNoop(t *testing.T) {
	d := openTestDB(t)
	if err := d.BulkMarkRead(nil); err != nil {
		t.Fatalf("BulkMarkRead nil: %v", err)
	}
	if err := d.BulkMarkRead([]int64{}); err != nil {
		t.Fatalf("BulkMarkRead empty: %v", err)
	}
}

func TestSearchArticles_ReturnsCrossfeedOrdered(t *testing.T) {
	d := openTestDB(t)
	fa, err := d.UpsertFeed("Habr", "https://habr.example/rss")
	if err != nil {
		t.Fatalf("feed A: %v", err)
	}
	fb, err := d.UpsertFeed("Hacker News", "https://news.example/rss")
	if err != nil {
		t.Fatalf("feed B: %v", err)
	}
	base := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	mk := func(feedID int64, idx int, title string, published time.Time) {
		a := newArticle(feedID, "https://ex.example/"+string(rune('a'+idx)), title, published)
		if _, err := d.UpsertArticle(a); err != nil {
			t.Fatalf("upsert %d: %v", idx, err)
		}
	}
	mk(fa.ID, 0, "habr1", base.Add(3*time.Hour))
	mk(fa.ID, 1, "habr2", base.Add(1*time.Hour))
	mk(fa.ID, 2, "habr3", base.Add(5*time.Hour))
	mk(fb.ID, 3, "hn1", base.Add(4*time.Hour))
	mk(fb.ID, 4, "hn2", base.Add(2*time.Hour))
	mk(fb.ID, 5, "hn3", base.Add(0*time.Hour))

	items, err := d.SearchArticles(100)
	if err != nil {
		t.Fatalf("SearchArticles: %v", err)
	}
	if len(items) != 6 {
		t.Fatalf("want 6 items, got %d", len(items))
	}
	wantTitles := []string{"habr3", "hn1", "habr1", "hn2", "habr2", "hn3"}
	for i, w := range wantTitles {
		if items[i].Title != w {
			t.Fatalf("pos %d: got %q, want %q (full order: %v)", i, items[i].Title, w, titlesOf(items))
		}
	}
	for _, it := range items {
		switch it.FeedID {
		case fa.ID:
			if it.FeedName != "Habr" {
				t.Fatalf("feed name: got %q, want Habr", it.FeedName)
			}
		case fb.ID:
			if it.FeedName != "Hacker News" {
				t.Fatalf("feed name: got %q, want Hacker News", it.FeedName)
			}
		default:
			t.Fatalf("unknown feed id: %d", it.FeedID)
		}
	}
}

func titlesOf(items []SearchItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Title
	}
	return out
}

func TestCacheArticle_SetsCachedBodyAndCachedAt(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss")
	a := newArticle(f.ID, "https://a.example/1", "v1", time.Now().UTC())
	if _, err := d.UpsertArticle(a); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	list, _ := d.ListArticles(f.ID, 10)
	if len(list) != 1 {
		t.Fatalf("want 1 article, got %d", len(list))
	}
	id := list[0].ID
	if list[0].CachedBody != "" || list[0].CachedAt != nil {
		t.Fatalf("pre-cache state wrong: %+v", list[0])
	}

	if err := d.CacheArticle(id, "# Hello\n\nBody"); err != nil {
		t.Fatalf("CacheArticle: %v", err)
	}
	list, _ = d.ListArticles(f.ID, 10)
	if list[0].CachedBody != "# Hello\n\nBody" {
		t.Fatalf("cached_body: %q", list[0].CachedBody)
	}
	if list[0].CachedAt == nil {
		t.Fatalf("cached_at is nil after CacheArticle")
	}
}
