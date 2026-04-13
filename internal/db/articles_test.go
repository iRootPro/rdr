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
	if err := d.UpsertArticle(a); err != nil {
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
	if err := d.UpsertArticle(a); err != nil {
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

	_ = d.UpsertArticle(newArticle(f.ID, "https://a.example/1", "older", base))
	_ = d.UpsertArticle(newArticle(f.ID, "https://a.example/2", "newer", base.Add(time.Hour)))
	_ = d.UpsertArticle(newArticle(f.ID, "https://a.example/3", "middle", base.Add(30*time.Minute)))

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
		_ = d.UpsertArticle(a)
	}
	for i := 3; i < 5; i++ {
		a := newArticle(f.ID, urlN(i), "unread", base.Add(time.Duration(i)*time.Minute))
		_ = d.UpsertArticle(a)
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
