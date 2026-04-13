package feed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iRootPro/rdr/internal/db"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "rdr.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func serveFixture(t *testing.T, name string) *httptest.Server {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write(body)
	}))
}

func TestFetchOne_AtomHappyPath(t *testing.T) {
	d := openTestDB(t)
	srv := serveFixture(t, "atom_feed.xml")
	defer srv.Close()

	feed, err := d.UpsertFeed("Example", srv.URL)
	if err != nil {
		t.Fatalf("UpsertFeed: %v", err)
	}

	f := New(d)
	result, err := f.FetchOne(context.Background(), feed)
	if err != nil {
		t.Fatalf("FetchOne: %v", err)
	}
	if result.Added != 3 {
		t.Fatalf("Added: got %d, want 3", result.Added)
	}
	if result.Updated != 0 {
		t.Fatalf("Updated: got %d, want 0", result.Updated)
	}
	if result.Err != nil {
		t.Fatalf("Result.Err: %v", result.Err)
	}

	articles, err := d.ListArticles(feed.ID, 10)
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	if len(articles) != 3 {
		t.Fatalf("articles: got %d, want 3", len(articles))
	}
	// Articles are ordered by published_at DESC, so [0] is the newest.
	if articles[0].Title != "First post" {
		t.Fatalf("articles[0].Title: got %q, want %q", articles[0].Title, "First post")
	}
	if articles[0].URL != "http://example.org/2026/04/13/first" {
		t.Fatalf("articles[0].URL: got %q", articles[0].URL)
	}
}

func TestFetchOne_IsIdempotent(t *testing.T) {
	d := openTestDB(t)
	srv := serveFixture(t, "atom_feed.xml")
	defer srv.Close()
	feed, err := d.UpsertFeed("Example", srv.URL)
	if err != nil {
		t.Fatalf("UpsertFeed: %v", err)
	}

	f := New(d)
	if _, err := f.FetchOne(context.Background(), feed); err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	result, err := f.FetchOne(context.Background(), feed)
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if result.Added != 0 {
		t.Fatalf("Added on rerun: got %d, want 0", result.Added)
	}
	if result.Updated != 3 {
		t.Fatalf("Updated on rerun: got %d, want 3", result.Updated)
	}
}

func TestFetchOne_MalformedXMLReturnsError(t *testing.T) {
	d := openTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<not xml at all"))
	}))
	defer srv.Close()
	feed, err := d.UpsertFeed("Bad", srv.URL)
	if err != nil {
		t.Fatalf("UpsertFeed: %v", err)
	}

	f := New(d)
	if _, err := f.FetchOne(context.Background(), feed); err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestFetchOne_HTTP500ReturnsError(t *testing.T) {
	d := openTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	feed, err := d.UpsertFeed("Boom", srv.URL)
	if err != nil {
		t.Fatalf("UpsertFeed: %v", err)
	}

	f := New(d)
	if _, err := f.FetchOne(context.Background(), feed); err == nil {
		t.Fatal("expected http error, got nil")
	}
}

func TestFetchOne_EmptyTitleUsesFallback(t *testing.T) {
	d := openTestDB(t)
	srv := serveFixture(t, "notitle_feed.xml")
	defer srv.Close()
	feed, err := d.UpsertFeed("NoTitle", srv.URL)
	if err != nil {
		t.Fatalf("UpsertFeed: %v", err)
	}

	f := New(d)
	if _, err := f.FetchOne(context.Background(), feed); err != nil {
		t.Fatalf("FetchOne: %v", err)
	}
	articles, err := d.ListArticles(feed.ID, 10)
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	if len(articles) != 1 {
		t.Fatalf("articles: got %d, want 1", len(articles))
	}
	if articles[0].Title != "(без заголовка)" {
		t.Fatalf("title fallback: got %q", articles[0].Title)
	}
}

func TestFetchAll_ContinuesAfterPerFeedError(t *testing.T) {
	d := openTestDB(t)
	good := serveFixture(t, "atom_feed.xml")
	defer good.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<broken"))
	}))
	defer bad.Close()

	goodFeed, err := d.UpsertFeed("Good", good.URL)
	if err != nil {
		t.Fatalf("UpsertFeed good: %v", err)
	}
	badFeed, err := d.UpsertFeed("Bad", bad.URL)
	if err != nil {
		t.Fatalf("UpsertFeed bad: %v", err)
	}

	f := New(d)
	results, err := f.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results: got %d, want 2", len(results))
	}

	byID := map[int64]FetchResult{
		results[0].Feed.ID: results[0],
		results[1].Feed.ID: results[1],
	}
	if g := byID[goodFeed.ID]; g.Err != nil || g.Added != 3 {
		t.Fatalf("good feed: Err=%v Added=%d", g.Err, g.Added)
	}
	if b := byID[badFeed.ID]; b.Err == nil {
		t.Fatalf("bad feed: expected error, got Added=%d", b.Added)
	}

	// The good feed's articles must be in the database despite the bad feed failing.
	articles, err := d.ListArticles(goodFeed.ID, 10)
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	if len(articles) != 3 {
		t.Fatalf("good feed articles: got %d, want 3", len(articles))
	}
}

func TestFetchAll_ContextCancelSurfaces(t *testing.T) {
	d := openTestDB(t)
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
	}))
	defer srv.Close()
	defer close(block)

	if _, err := d.UpsertFeed("Slow", srv.URL); err != nil {
		t.Fatalf("UpsertFeed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	f := New(d)
	results, err := f.FetchAll(ctx)
	// Cancel must surface either as outer err (semaphore-wait branch)
	// or as a per-feed err inside results (FetchOne's ctx-aware HTTP path).
	if err != nil {
		return
	}
	if len(results) == 0 || results[0].Err == nil {
		t.Fatalf("expected cancellation to surface, got clean results %+v", results)
	}
}

func TestFetchOne_TrimsReadArticlesToSettingCap(t *testing.T) {
	d := openTestDB(t)
	// cap=4: after fetch we have 3 unread + 3 seeded read = 6 total,
	// TrimArticles deletes (6-4)=2 oldest read rows, leaving 3 unread + 1 read.
	if err := d.SetSetting("max_articles_per_feed", "4"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}

	srv := serveFixture(t, "atom_feed.xml")
	defer srv.Close()
	feed, err := d.UpsertFeed("Example", srv.URL)
	if err != nil {
		t.Fatalf("UpsertFeed: %v", err)
	}

	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		a := db.Article{
			FeedID:      feed.ID,
			Title:       "old",
			URL:         "https://old.example/" + string(rune('a'+i)),
			PublishedAt: base.Add(time.Duration(i) * time.Hour),
		}
		if _, err := d.UpsertArticle(a); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	all, err := d.ListArticles(feed.ID, 100)
	if err != nil {
		t.Fatalf("ListArticles seed: %v", err)
	}
	for _, a := range all {
		if err := d.MarkRead(a.ID); err != nil {
			t.Fatalf("MarkRead: %v", err)
		}
	}

	f := New(d)
	if _, err := f.FetchOne(context.Background(), feed); err != nil {
		t.Fatalf("FetchOne: %v", err)
	}

	all, err = d.ListArticles(feed.ID, 100)
	if err != nil {
		t.Fatalf("ListArticles after fetch: %v", err)
	}
	var unread, read int
	for _, a := range all {
		if a.ReadAt == nil {
			unread++
		} else {
			read++
		}
	}
	if unread != 3 {
		t.Fatalf("unread: got %d, want 3", unread)
	}
	if read != 1 {
		t.Fatalf("read: got %d, want 1 (oldest 2 of 3 reads trimmed)", read)
	}
}

func TestFetchFull_ExtractsAndConvertsToMarkdown(t *testing.T) {
	d := openTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		body, _ := os.ReadFile(filepath.Join("testdata", "article.html"))
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	f := New(d)
	md, err := f.FetchFull(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchFull: %v", err)
	}
	if md == "" {
		t.Fatal("empty markdown")
	}
	for _, want := range []string{"first paragraph", "second paragraph", "List item one"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}
	if strings.Contains(md, "Site header noise") || strings.Contains(md, "Footer noise") {
		t.Fatalf("readability did not strip chrome:\n%s", md)
	}
}
