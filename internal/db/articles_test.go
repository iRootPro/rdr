package db

import (
	"fmt"
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
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
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
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
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
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
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

	// fetchCutoff in the future so ALL rows look "stale" to trim —
	// this simulates the case where the articles are no longer in the
	// current RSS response. readCutoff also in the future so the
	// just-marked-read rows count as "old enough" to trim (otherwise
	// the grace window would protect them).
	fetchCutoff := time.Now().UTC().Add(time.Hour)
	readCutoff := time.Now().UTC().Add(time.Hour)
	if err := d.TrimArticles(f.ID, 3, fetchCutoff, readCutoff); err != nil {
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

// TestTrimArticles_SkipsFreshlyFetched guards the regression where trim
// would delete read articles that the RSS feed still lists, causing
// them to re-appear as unread on the next fetch.
func TestTrimArticles_SkipsFreshlyFetched(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
	base := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	// Upsert 5 articles, all "freshly fetched".
	for i := 0; i < 5; i++ {
		a := newArticle(f.ID, urlN(i), fmt.Sprintf("art%d", i), base.Add(time.Duration(i)*time.Minute))
		_, _ = d.UpsertArticle(a)
	}
	// Mark everything read.
	list, _ := d.ListArticles(f.ID, 10)
	for _, a := range list {
		_ = d.MarkRead(a.ID)
	}

	// fetchCutoff BEFORE the fetch start — every article is "fresh"
	// (its last_fetched_at >= fetchCutoff), so trim must not touch
	// them even when the limit is 0 (would otherwise delete all of
	// them). readCutoff is irrelevant here because the fetchCutoff
	// guard kicks in first.
	fetchCutoff := base.Add(-time.Hour)
	readCutoff := time.Now().UTC().Add(time.Hour)
	if err := d.TrimArticles(f.ID, 1, fetchCutoff, readCutoff); err != nil {
		t.Fatalf("TrimArticles: %v", err)
	}

	list, _ = d.ListArticles(f.ID, 10)
	if len(list) != 5 {
		t.Fatalf("freshly fetched rows must survive trim, got %d remaining", len(list))
	}
}

// TestTrimArticles_UserFlowSurvivesRefresh reproduces the exact user
// report: mark read in Inbox → exit/return → refresh (R) → marked
// article must NOT come back as unread. Simulates two fetch cycles
// around a MarkRead in between.
func TestTrimArticles_UserFlowSurvivesRefresh(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
	base := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	// Fetch #1: insert 5 articles that simulate the RSS response.
	for i := 0; i < 5; i++ {
		a := newArticle(f.ID, urlN(i), fmt.Sprintf("art%d", i), base.Add(time.Duration(i)*time.Minute))
		_, _ = d.UpsertArticle(a)
	}
	// User marks one article read.
	list, _ := d.ListArticles(f.ID, 10)
	var markedID int64
	for _, a := range list {
		if a.Title == "art2" {
			_ = d.MarkRead(a.ID)
			markedID = a.ID
			break
		}
	}
	if markedID == 0 {
		t.Fatal("could not find art2 to mark read")
	}

	// Fetch #2: same RSS response. fetchStart is captured before the
	// upsert loop, matching how fetcher.FetchOne does it.
	fetchStart := time.Now().UTC()
	for i := 0; i < 5; i++ {
		a := newArticle(f.ID, urlN(i), fmt.Sprintf("art%d", i), base.Add(time.Duration(i)*time.Minute))
		_, _ = d.UpsertArticle(a)
	}
	// Trim with a very aggressive limit (would normally delete read
	// articles): the protection must keep the just-upserted rows.
	// readCutoff in the future so the read-grace doesn't mask the
	// regression — we want fetchCutoff alone to do the protecting.
	readCutoff := time.Now().UTC().Add(time.Hour)
	if err := d.TrimArticles(f.ID, 1, fetchStart, readCutoff); err != nil {
		t.Fatalf("TrimArticles: %v", err)
	}

	// Marked article must still exist and still be read.
	list, _ = d.ListArticles(f.ID, 10)
	found := false
	for _, a := range list {
		if a.ID == markedID {
			found = true
			if a.ReadAt == nil {
				t.Fatal("marked article lost its read state after refresh")
			}
			break
		}
	}
	if !found {
		t.Fatal("marked article disappeared after refresh — this is the bug")
	}
}

// TestTrimArticles_PreservesStarred guards the data-loss bug where a
// starred article that the user later marks read (or that gets read via
// any path) could be wiped by trim. Star is an explicit save signal.
func TestTrimArticles_PreservesStarred(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
	base := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	// 5 read articles, oldest first. Star the oldest one — it would
	// normally be the first victim of trim.
	for i := 0; i < 5; i++ {
		a := newArticle(f.ID, urlN(i), fmt.Sprintf("art%d", i), base.Add(time.Duration(i)*time.Minute))
		_, _ = d.UpsertArticle(a)
	}
	list, _ := d.ListArticles(f.ID, 10)
	var starredID int64
	for _, a := range list {
		_ = d.MarkRead(a.ID)
		if a.Title == "art0" {
			starredID = a.ID
			if _, err := d.ToggleStar(a.ID); err != nil {
				t.Fatalf("ToggleStar: %v", err)
			}
		}
	}

	// Aggressive trim: cap=1, both cutoffs in the future. Without
	// the star protection, art0 is the oldest read row and gets
	// deleted first.
	future := time.Now().UTC().Add(time.Hour)
	if err := d.TrimArticles(f.ID, 1, future, future); err != nil {
		t.Fatalf("TrimArticles: %v", err)
	}

	list, _ = d.ListArticles(f.ID, 10)
	for _, a := range list {
		if a.ID == starredID {
			return
		}
	}
	t.Fatal("starred article was deleted by trim — star must protect")
}

// TestTrimArticles_PreservesBookmarked is the read-later equivalent of
// the star test. Bookmark is also an explicit save signal.
func TestTrimArticles_PreservesBookmarked(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
	base := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		a := newArticle(f.ID, urlN(i), fmt.Sprintf("art%d", i), base.Add(time.Duration(i)*time.Minute))
		_, _ = d.UpsertArticle(a)
	}
	list, _ := d.ListArticles(f.ID, 10)
	var bookmarkedID int64
	for _, a := range list {
		_ = d.MarkRead(a.ID)
		if a.Title == "art0" {
			bookmarkedID = a.ID
			if _, err := d.ToggleBookmark(a.ID); err != nil {
				t.Fatalf("ToggleBookmark: %v", err)
			}
		}
	}

	future := time.Now().UTC().Add(time.Hour)
	if err := d.TrimArticles(f.ID, 1, future, future); err != nil {
		t.Fatalf("TrimArticles: %v", err)
	}

	list, _ = d.ListArticles(f.ID, 10)
	for _, a := range list {
		if a.ID == bookmarkedID {
			return
		}
	}
	t.Fatal("bookmarked article was deleted by trim — bookmark must protect")
}

// TestTrimArticles_PreservesRecentlyRead asserts the grace window:
// articles read after readCutoff survive trim even if the feed is over
// cap. Catches the user-reported issue: accidentally pressing `x`
// shouldn't permanently destroy an article on the next sync.
func TestTrimArticles_PreservesRecentlyRead(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
	base := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		a := newArticle(f.ID, urlN(i), fmt.Sprintf("art%d", i), base.Add(time.Duration(i)*time.Minute))
		_, _ = d.UpsertArticle(a)
	}
	list, _ := d.ListArticles(f.ID, 10)
	for _, a := range list {
		_ = d.MarkRead(a.ID) // read_at ≈ time.Now()
	}

	// readCutoff in the past → all read_at values are >= readCutoff
	// → all rows protected. fetchCutoff in the future so that guard
	// is NOT what's saving them; we want to verify the grace alone.
	fetchCutoff := time.Now().UTC().Add(time.Hour)
	readCutoff := time.Now().UTC().Add(-time.Hour)
	if err := d.TrimArticles(f.ID, 1, fetchCutoff, readCutoff); err != nil {
		t.Fatalf("TrimArticles: %v", err)
	}

	list, _ = d.ListArticles(f.ID, 10)
	if len(list) != 5 {
		t.Fatalf("recently-read articles should survive grace window: got %d remaining, want 5", len(list))
	}
}

// TestTrimArticles_ZeroReadCutoffKeepsAllReadArticles asserts that
// passing zero time.Time as readCutoff disables the age-based trim —
// this is the "retention=0 / unlimited" path. Articles still get
// protected by starred/bookmarked and by the fetch-cutoff guard; only
// the age check is bypassed.
func TestTrimArticles_ZeroReadCutoffKeepsAllReadArticles(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
	base := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		a := newArticle(f.ID, urlN(i), fmt.Sprintf("art%d", i), base.Add(time.Duration(i)*time.Minute))
		_, _ = d.UpsertArticle(a)
	}
	list, _ := d.ListArticles(f.ID, 10)
	for _, a := range list {
		_ = d.MarkRead(a.ID)
	}
	// Backdate read_at well into the past so a normal readCutoff would
	// have removed them — only the zero-cutoff bypass should save us.
	// (Using a far-future fetchCutoff so the fetch guard isn't what's
	// protecting the rows.)
	fetchCutoff := time.Now().UTC().Add(time.Hour)
	if err := d.TrimArticles(f.ID, 1, fetchCutoff, time.Time{}); err != nil {
		t.Fatalf("TrimArticles: %v", err)
	}
	list, _ = d.ListArticles(f.ID, 10)
	if len(list) != 5 {
		t.Fatalf("zero readCutoff must skip age filter: got %d remaining, want 5", len(list))
	}
}

func urlN(i int) string {
	return "https://a.example/" + string(rune('0'+i))
}

func TestUpsertArticle_ReturnsInsertedFlag(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
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
	f, err := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
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
	f, err := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
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
	f, err := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
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

func TestMarkFeedRead_MarksOnlyThatFeedsUnread(t *testing.T) {
	d := openTestDB(t)
	fa, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
	fb, _ := d.UpsertFeed("B", "https://b.example/rss", "", "", "")
	base := time.Now().UTC()

	for i := 0; i < 3; i++ {
		if _, err := d.UpsertArticle(newArticle(fa.ID, "https://a.example/"+string(rune('a'+i)), "a", base.Add(time.Duration(i)*time.Minute))); err != nil {
			t.Fatalf("upsert A: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := d.UpsertArticle(newArticle(fb.ID, "https://b.example/"+string(rune('a'+i)), "b", base.Add(time.Duration(i)*time.Minute))); err != nil {
			t.Fatalf("upsert B: %v", err)
		}
	}

	n, err := d.MarkFeedRead(fa.ID)
	if err != nil {
		t.Fatalf("MarkFeedRead: %v", err)
	}
	if n != 3 {
		t.Fatalf("want 3 marked, got %d", n)
	}

	listA, _ := d.ListArticles(fa.ID, 10)
	for _, a := range listA {
		if a.ReadAt == nil {
			t.Fatalf("feed A article %d still unread", a.ID)
		}
	}
	listB, _ := d.ListArticles(fb.ID, 10)
	for _, b := range listB {
		if b.ReadAt != nil {
			t.Fatalf("feed B article %d unexpectedly read", b.ID)
		}
	}
}

func TestMarkFeedRead_IdempotentOnAlreadyRead(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
	if _, err := d.UpsertArticle(newArticle(f.ID, "https://a.example/1", "v", time.Now().UTC())); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if _, err := d.MarkFeedRead(f.ID); err != nil {
		t.Fatalf("first MarkFeedRead: %v", err)
	}
	n, err := d.MarkFeedRead(f.ID)
	if err != nil {
		t.Fatalf("second MarkFeedRead: %v", err)
	}
	if n != 0 {
		t.Fatalf("want 0 on idempotent second call, got %d", n)
	}
}

func TestMarkUnread_ClearsReadAt(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
	if _, err := d.UpsertArticle(newArticle(f.ID, "https://a.example/1", "v", time.Now().UTC())); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	list, _ := d.ListArticles(f.ID, 10)
	id := list[0].ID
	if err := d.MarkRead(id); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if err := d.MarkUnread(id); err != nil {
		t.Fatalf("MarkUnread: %v", err)
	}
	list, _ = d.ListArticles(f.ID, 10)
	if list[0].ReadAt != nil {
		t.Fatalf("read_at still set after MarkUnread")
	}
}

func TestBulkMarkRead_MarksOnlyUnread(t *testing.T) {
	d := openTestDB(t)
	f, err := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
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
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
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
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
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
	fa, err := d.UpsertFeed("Habr", "https://habr.example/rss", "", "", "")
	if err != nil {
		t.Fatalf("feed A: %v", err)
	}
	fb, err := d.UpsertFeed("Hacker News", "https://news.example/rss", "", "", "")
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
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "", "", "")
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
