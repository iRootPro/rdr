package ui

import (
	"strings"
	"testing"
)

func TestRenderFeedList_ActiveSelectionUsesStrongBar(t *testing.T) {
	entries := []feedEntry{
		{Kind: entryLibrary, Name: "Library", UnreadCount: 5},
		{Kind: entryCategory, Name: "Tech News"},
		{Kind: entryFeed, Name: "Hacker News", FeedURL: "https://news.ycombinator.com/rss", UnreadCount: 15},
	}

	out := renderFeedList(entries, 2, true, 44, 8, testTR, feedListInfo{})
	line := renderedLineContaining(out, "Hacker News")
	if !strings.Contains(line, "▌") {
		t.Fatalf("active feed selection should use the same strong bar as article selection, got line %q\nfull output:\n%s", line, stripANSI(out))
	}
	if strings.Contains(line, "›") {
		t.Fatalf("active feed selection should not use the weaker chevron marker, got line %q", line)
	}
}

func TestRenderFeedList_SectionSeparatorsAreInset(t *testing.T) {
	entries := []feedEntry{
		{Kind: entryLibrary, Name: "Library", UnreadCount: 5},
		{Kind: entryCategory, Name: "Tech News"},
	}

	out := renderFeedList(entries, 0, false, 44, 6, testTR, feedListInfo{})
	for _, line := range strings.Split(stripANSI(out), "\n") {
		if strings.HasPrefix(line, "╭") || strings.HasPrefix(line, "╰") || !strings.Contains(line, "──") {
			continue
		}
		if !strings.Contains(line, "│   ─") {
			t.Fatalf("section separator should be inset so it does not read as another pane border, got %q", line)
		}
		return
	}
	t.Fatalf("expected section separator in output:\n%s", stripANSI(out))
}
