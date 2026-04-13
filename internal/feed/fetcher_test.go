package feed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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
