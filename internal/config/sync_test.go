package config

import (
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

func TestSync_EmptyConfigIsNoOp(t *testing.T) {
	d := openTestDB(t)
	if err := Sync(d, &Config{}); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	feeds, err := d.ListFeeds()
	if err != nil {
		t.Fatalf("ListFeeds: %v", err)
	}
	if len(feeds) != 0 {
		t.Fatalf("expected no feeds, got %d", len(feeds))
	}
}

func TestSync_InsertsFeedsInYAMLOrder(t *testing.T) {
	d := openTestDB(t)
	cfg := &Config{Feeds: []FeedEntry{
		{Name: "HN", URL: "https://hnrss.org/frontpage"},
		{Name: "Go", URL: "https://go.dev/blog/feed.atom"},
		{Name: "Lobsters", URL: "https://lobste.rs/rss"},
	}}
	if err := Sync(d, cfg); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	feeds, _ := d.ListFeeds()
	if len(feeds) != 3 {
		t.Fatalf("want 3 feeds, got %d", len(feeds))
	}
	wantNames := []string{"HN", "Go", "Lobsters"}
	for i, f := range feeds {
		if f.Name != wantNames[i] {
			t.Fatalf("feed[%d] name: got %q, want %q", i, f.Name, wantNames[i])
		}
		if f.Position != i {
			t.Fatalf("feed[%d] position: got %d, want %d", i, f.Position, i)
		}
	}
}

func TestSync_IsIdempotent(t *testing.T) {
	d := openTestDB(t)
	cfg := &Config{Feeds: []FeedEntry{
		{Name: "HN", URL: "https://hnrss.org/frontpage"},
		{Name: "Go", URL: "https://go.dev/blog/feed.atom"},
	}}
	if err := Sync(d, cfg); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if err := Sync(d, cfg); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	feeds, _ := d.ListFeeds()
	if len(feeds) != 2 {
		t.Fatalf("want 2 feeds after double sync, got %d", len(feeds))
	}
}

func TestSync_DoesNotDeleteFeedsRemovedFromYAML(t *testing.T) {
	d := openTestDB(t)
	full := &Config{Feeds: []FeedEntry{
		{Name: "HN", URL: "https://hnrss.org/frontpage"},
		{Name: "Go", URL: "https://go.dev/blog/feed.atom"},
	}}
	if err := Sync(d, full); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	shrunk := &Config{Feeds: []FeedEntry{
		{Name: "HN", URL: "https://hnrss.org/frontpage"},
	}}
	if err := Sync(d, shrunk); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	feeds, _ := d.ListFeeds()
	if len(feeds) != 2 {
		t.Fatalf("expected 2 feeds (no deletion), got %d", len(feeds))
	}
}

func TestSync_RenamesExistingFeedAndPreservesPosition(t *testing.T) {
	d := openTestDB(t)
	first := &Config{Feeds: []FeedEntry{
		{Name: "Hacker News", URL: "https://hnrss.org/frontpage"},
		{Name: "Go Blog", URL: "https://go.dev/blog/feed.atom"},
	}}
	if err := Sync(d, first); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	feedsBefore, _ := d.ListFeeds()
	hnPos := feedsBefore[0].Position
	hnID := feedsBefore[0].ID

	renamed := &Config{Feeds: []FeedEntry{
		{Name: "HN", URL: "https://hnrss.org/frontpage"},
		{Name: "Go Blog", URL: "https://go.dev/blog/feed.atom"},
	}}
	if err := Sync(d, renamed); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	feedsAfter, _ := d.ListFeeds()
	var hn db.Feed
	for _, f := range feedsAfter {
		if f.URL == "https://hnrss.org/frontpage" {
			hn = f
		}
	}
	if hn.Name != "HN" {
		t.Fatalf("rename failed: %q", hn.Name)
	}
	if hn.Position != hnPos {
		t.Fatalf("position changed: %d → %d", hnPos, hn.Position)
	}
	if hn.ID != hnID {
		t.Fatalf("id changed: %d → %d", hnID, hn.ID)
	}
}
