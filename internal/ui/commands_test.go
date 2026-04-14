package ui

import (
	"testing"
	"time"

	"github.com/iRootPro/rdr/internal/config"
	"github.com/iRootPro/rdr/internal/db"
)

func TestApplySort_DateDescByDefault(t *testing.T) {
	base := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	articles := []db.Article{
		{Title: "old", PublishedAt: base},
		{Title: "new", PublishedAt: base.Add(2 * time.Hour)},
		{Title: "mid", PublishedAt: base.Add(1 * time.Hour)},
	}
	applySort(articles, "date", false)
	want := []string{"new", "mid", "old"}
	for i, w := range want {
		if articles[i].Title != w {
			t.Fatalf("pos %d: got %q want %q", i, articles[i].Title, w)
		}
	}
}

func TestApplySort_DateReverse(t *testing.T) {
	base := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	articles := []db.Article{
		{Title: "new", PublishedAt: base.Add(2 * time.Hour)},
		{Title: "old", PublishedAt: base},
	}
	applySort(articles, "date", true)
	if articles[0].Title != "old" {
		t.Fatalf("want old first, got %q", articles[0].Title)
	}
}

func TestApplySort_TitleAsc(t *testing.T) {
	articles := []db.Article{
		{Title: "banana"},
		{Title: "apple"},
		{Title: "cherry"},
	}
	applySort(articles, "title", false)
	want := []string{"apple", "banana", "cherry"}
	for i, w := range want {
		if articles[i].Title != w {
			t.Fatalf("pos %d: got %q want %q", i, articles[i].Title, w)
		}
	}
}

func TestApplySort_TitleReverse(t *testing.T) {
	articles := []db.Article{
		{Title: "apple"},
		{Title: "banana"},
	}
	applySort(articles, "title", true)
	if articles[0].Title != "banana" {
		t.Fatalf("want banana first, got %q", articles[0].Title)
	}
}

func TestDispatchCommand_QuitReturnsQuitCmd(t *testing.T) {
	m := Model{}
	_, cmd := dispatchCommand(m, "q")
	if cmd == nil {
		t.Fatal("expected tea.Quit, got nil")
	}
}

func TestDispatchCommand_SortDate(t *testing.T) {
	m := Model{sortField: "title"}
	m2, _ := dispatchCommand(m, "sort date")
	mm := m2.(Model)
	if mm.sortField != "date" {
		t.Fatalf("sortField=%q, want date", mm.sortField)
	}
}

func TestDispatchCommand_SortRejectsBadField(t *testing.T) {
	m := Model{sortField: "date"}
	m2, _ := dispatchCommand(m, "sort bogus")
	mm := m2.(Model)
	if mm.err == nil {
		t.Fatal("expected error for bad sort field")
	}
	if mm.sortField != "date" {
		t.Fatalf("sortField mutated to %q", mm.sortField)
	}
}

func TestDispatchCommand_SortReverseToggles(t *testing.T) {
	m := Model{sortField: "date", sortReverse: false}
	m2, _ := dispatchCommand(m, "sortreverse")
	mm := m2.(Model)
	if !mm.sortReverse {
		t.Fatal("sortReverse should be true after toggle")
	}
	m3, _ := dispatchCommand(mm, "sortreverse")
	mmm := m3.(Model)
	if mmm.sortReverse {
		t.Fatal("sortReverse should be false after second toggle")
	}
}

func TestDispatchCommand_FilterUnreadSetsFilter(t *testing.T) {
	m := Model{filter: filterAll}
	m2, _ := dispatchCommand(m, "filter unread")
	mm := m2.(Model)
	if mm.filter != filterUnread {
		t.Fatalf("filter=%v, want filterUnread", mm.filter)
	}
}

func TestDispatchCommand_ZenToggles(t *testing.T) {
	m := Model{zenMode: false}
	m2, _ := dispatchCommand(m, "zen")
	if !m2.(Model).zenMode {
		t.Fatal("zen should be true")
	}
}

func TestDispatchCommand_UnknownSetsErr(t *testing.T) {
	m := Model{}
	m2, _ := dispatchCommand(m, "nosuch thing")
	if m2.(Model).err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestPushHistory_DedupesAdjacent(t *testing.T) {
	m := Model{}
	m.pushHistory("sync")
	m.pushHistory("sync")
	m.pushHistory("sort date")
	m.pushHistory("sort date")
	if got := len(m.commandHistory); got != 2 {
		t.Fatalf("want 2 entries after dedup, got %d: %v", got, m.commandHistory)
	}
	if m.commandHistory[0] != "sort date" {
		t.Fatalf("want most-recent first, got %v", m.commandHistory)
	}
}

func TestPushHistory_CapsAt50(t *testing.T) {
	m := Model{}
	for i := 0; i < 60; i++ {
		m.pushHistory("cmd" + string(rune('a'+i%26)) + string(rune('0'+i%10)))
	}
	if got := len(m.commandHistory); got != 50 {
		t.Fatalf("want 50, got %d", got)
	}
}

func TestRefreshFolderCounts_MatchesQuery(t *testing.T) {
	now := time.Now().UTC()
	readT := now.Add(-time.Hour)
	m := Model{
		smartFolders: []config.SmartFolder{
			{Name: "Inbox", Query: "unread"},
			{Name: "Habr", Query: "feed:habr"},
		},
		allArticles: []db.Article{
			{Title: "a", FeedName: "Habr", PublishedAt: now},
			{Title: "b", FeedName: "Habr", PublishedAt: now, ReadAt: &readT},
			{Title: "c", FeedName: "HN", PublishedAt: now},
		},
	}
	m.refreshFolderCounts()
	// Inbox = unread articles = 2 (a, c)
	// Habr = any feed:habr = 2 (a, b)
	if m.folderCounts[0] != 2 {
		t.Fatalf("Inbox count = %d, want 2", m.folderCounts[0])
	}
	if m.folderCounts[1] != 2 {
		t.Fatalf("Habr count = %d, want 2", m.folderCounts[1])
	}
}

func TestCommandSuggestions_EmptyReturnsAll(t *testing.T) {
	got := commandSuggestionsFor("")
	if len(got) != len(commandCompletions) {
		t.Fatalf("empty input: want %d, got %d", len(commandCompletions), len(got))
	}
}

func TestCommandSuggestions_PrefixFilters(t *testing.T) {
	got := commandSuggestionsFor("so")
	// Expected: sort date, sort title, sortreverse
	if len(got) != 3 {
		t.Fatalf("prefix 'so': want 3, got %d: %+v", len(got), got)
	}
	for _, s := range got {
		if s.Complete[:2] != "so" {
			t.Fatalf("result %q does not start with 'so'", s.Complete)
		}
	}
}

func TestCommandSuggestions_PrefixWithSpace(t *testing.T) {
	got := commandSuggestionsFor("sort ")
	if len(got) != 2 {
		t.Fatalf("prefix 'sort ': want 2 (date, title), got %d: %+v", len(got), got)
	}
}

func TestCommandSuggestions_NoMatch(t *testing.T) {
	got := commandSuggestionsFor("zzz")
	if len(got) != 0 {
		t.Fatalf("want 0, got %d", len(got))
	}
}

func TestCommandSuggestions_FullMatchReturnsSelf(t *testing.T) {
	got := commandSuggestionsFor("quit")
	if len(got) != 1 || got[0].Complete != "quit" {
		t.Fatalf("want exact 'quit', got %+v", got)
	}
}

func TestCommandSuggestions_LeadingSpaceIgnored(t *testing.T) {
	got := commandSuggestionsFor("   sync")
	if len(got) != 1 || got[0].Complete != "sync" {
		t.Fatalf("want 1 sync match, got %+v", got)
	}
}

func TestDispatchCommand_EmptyNoop(t *testing.T) {
	m := Model{}
	m2, cmd := dispatchCommand(m, "")
	if cmd != nil {
		t.Fatal("empty command should not emit tea.Cmd")
	}
	if m2.(Model).err != nil {
		t.Fatal("empty command should not set err")
	}
}
