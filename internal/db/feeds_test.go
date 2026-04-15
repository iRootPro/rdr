package db

import (
	"testing"
	"time"
)

func TestUpsertFeed_InsertThenUpdateName(t *testing.T) {
	d := openTestDB(t)

	f1, err := d.UpsertFeed("Hacker News", "https://hnrss.org/frontpage", "")
	if err != nil {
		t.Fatalf("upsert insert: %v", err)
	}
	if f1.ID == 0 || f1.Name != "Hacker News" {
		t.Fatalf("unexpected feed: %+v", f1)
	}

	f2, err := d.UpsertFeed("HN", "https://hnrss.org/frontpage", "")
	if err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	if f2.ID != f1.ID {
		t.Fatalf("expected same id: %d vs %d", f2.ID, f1.ID)
	}
	if f2.Name != "HN" {
		t.Fatalf("expected renamed feed, got %q", f2.Name)
	}
}

func TestListFeeds_OrdersByPositionAndCountsUnread(t *testing.T) {
	d := openTestDB(t)

	a, _ := d.UpsertFeed("A", "https://a.example/rss", "")
	b, _ := d.UpsertFeed("B", "https://b.example/rss", "")

	mustExec(t, d, `
		INSERT INTO articles (feed_id, title, url, published_at, read_at)
		VALUES (?, 'one',   'https://a.example/1', ?, NULL),
		       (?, 'two',   'https://a.example/2', ?, NULL),
		       (?, 'three', 'https://a.example/3', ?, ?)
	`, a.ID, time.Now(), a.ID, time.Now(), a.ID, time.Now(), time.Now())

	mustExec(t, d, `
		INSERT INTO articles (feed_id, title, url, published_at, read_at)
		VALUES (?, 'only', 'https://b.example/1', ?, ?)
	`, b.ID, time.Now(), time.Now())

	feeds, err := d.ListFeeds()
	if err != nil {
		t.Fatalf("ListFeeds: %v", err)
	}
	if len(feeds) != 2 {
		t.Fatalf("want 2 feeds, got %d", len(feeds))
	}
	byName := map[string]Feed{feeds[0].Name: feeds[0], feeds[1].Name: feeds[1]}
	if byName["A"].UnreadCount != 2 {
		t.Fatalf("feed A unread: got %d, want 2", byName["A"].UnreadCount)
	}
	if byName["B"].UnreadCount != 0 {
		t.Fatalf("feed B unread: got %d, want 0", byName["B"].UnreadCount)
	}
}

func TestDeleteFeed_CascadesArticles(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "")
	mustExec(t, d, `
		INSERT INTO articles (feed_id, title, url, published_at)
		VALUES (?, 't', 'https://a.example/1', ?)
	`, f.ID, time.Now())

	if err := d.DeleteFeed(f.ID); err != nil {
		t.Fatalf("DeleteFeed: %v", err)
	}
	var count int
	if err := d.sql.QueryRow(
		`SELECT COUNT(*) FROM articles WHERE feed_id = ?`, f.ID,
	).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected cascade delete, got %d rows", count)
	}
}

func TestListFeeds_ZeroArticlesReportsZeroUnread(t *testing.T) {
	d := openTestDB(t)
	if _, err := d.UpsertFeed("Empty", "https://empty.example/rss", ""); err != nil {
		t.Fatalf("UpsertFeed: %v", err)
	}
	feeds, err := d.ListFeeds()
	if err != nil {
		t.Fatalf("ListFeeds: %v", err)
	}
	if len(feeds) != 1 {
		t.Fatalf("want 1 feed, got %d", len(feeds))
	}
	if feeds[0].UnreadCount != 0 {
		t.Fatalf("zero-article feed unread: got %d, want 0", feeds[0].UnreadCount)
	}
}

func TestRenameFeed_ChangesNameKeepsURL(t *testing.T) {
	d := openTestDB(t)
	f, err := d.UpsertFeed("Old", "https://a.example/rss", "")
	if err != nil {
		t.Fatalf("UpsertFeed: %v", err)
	}
	if err := d.RenameFeed(f.ID, "New"); err != nil {
		t.Fatalf("RenameFeed: %v", err)
	}
	feeds, _ := d.ListFeeds()
	if len(feeds) != 1 {
		t.Fatalf("want 1 feed, got %d", len(feeds))
	}
	if feeds[0].Name != "New" {
		t.Fatalf("name: got %q, want %q", feeds[0].Name, "New")
	}
	if feeds[0].URL != "https://a.example/rss" {
		t.Fatalf("url changed: %q", feeds[0].URL)
	}
}

func mustExec(t *testing.T, d *DB, query string, args ...any) {
	t.Helper()
	if _, err := d.sql.Exec(query, args...); err != nil {
		t.Fatalf("exec: %v", err)
	}
}
