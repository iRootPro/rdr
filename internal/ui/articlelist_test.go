package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/iRootPro/rdr/internal/db"
)

func TestRenderArticleList_SelectedUnreadRowHasStrongMarker(t *testing.T) {
	now := time.Now()
	articles := []db.Article{
		{Title: "Selected story", FeedName: "Habr", PublishedAt: now.Add(-14 * time.Minute)},
		{Title: "Already read", FeedName: "Ixbt", PublishedAt: now.Add(-16 * time.Minute), ReadAt: &now},
	}

	out := renderArticleList(articles, 0, true, 100, 8, testTR, false, -1, -1)
	line := renderedLineContaining(out, "Selected story")
	if !strings.Contains(line, "▌ ● Selected story") {
		t.Fatalf("selected unread row should use active bar plus unread marker, got line %q\nfull output:\n%s", line, stripANSI(out))
	}
}

func TestRenderArticleList_MetadataIsCompactAndDoesNotAddInnerDivider(t *testing.T) {
	now := time.Now()
	articles := []db.Article{
		{Title: "Selected story", FeedName: "Habr", PublishedAt: now.Add(-14 * time.Minute)},
	}

	out := renderArticleList(articles, 0, true, 100, 8, testTR, false, -1, -1)
	line := renderedLineContaining(out, "Selected story")
	if !strings.Contains(line, "Habr ·") {
		t.Fatalf("metadata should combine source and time in one quiet gutter, got line %q\nfull output:\n%s", line, stripANSI(out))
	}
	if got := strings.Count(line, "│"); got != 2 {
		t.Fatalf("article row should only contain pane borders, got %d vertical bars in %q", got, line)
	}
}

func TestRenderArticleList_UnselectedUnreadRowsDoNotRepeatDotMarker(t *testing.T) {
	now := time.Now()
	articles := []db.Article{
		{Title: "Selected read story", FeedName: "Habr", PublishedAt: now.Add(-14 * time.Minute), ReadAt: &now},
		{Title: "Unread but not selected", FeedName: "Ixbt", PublishedAt: now.Add(-16 * time.Minute)},
	}

	out := renderArticleList(articles, 0, true, 100, 8, testTR, false, -1, -1)
	line := renderedLineContaining(out, "Unread but not selected")
	if strings.Contains(line, "●") {
		t.Fatalf("unselected unread rows should rely on text contrast instead of repeated dot markers, got line %q\nfull output:\n%s", line, stripANSI(out))
	}
}
func renderedLineContaining(rendered, needle string) string {
	for _, line := range strings.Split(stripANSI(rendered), "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}
