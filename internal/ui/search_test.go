package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/iRootPro/rdr/internal/db"
)

func mkModelWith(items []db.SearchItem, query string) Model {
	ti := textinput.New()
	ti.SetValue(query)
	m := Model{
		searchInput: ti,
		searchAll:   items,
	}
	recomputeMatches(&m)
	return m
}

func matchedTitles(m Model) []string {
	out := make([]string, 0, len(m.searchMatches))
	for _, idx := range m.searchMatches {
		out = append(out, m.searchAll[idx].Title)
	}
	return out
}

func TestRecomputeMatches_EmptyQueryReturnsAll(t *testing.T) {
	items := []db.SearchItem{
		{Title: "How Linux handles memory", FeedName: "Habr"},
		{Title: "Rust async explained", FeedName: "Hacker News"},
		{Title: "Go generics deep dive", FeedName: "Lobsters"},
	}
	m := mkModelWith(items, "")
	if got := len(m.searchMatches); got != 3 {
		t.Fatalf("empty query: want 3 matches, got %d", got)
	}
	got := matchedTitles(m)
	for i := range items {
		if got[i] != items[i].Title {
			t.Fatalf("order preserved: want %q at pos %d, got %q", items[i].Title, i, got[i])
		}
	}
}

func TestRecomputeMatches_FiltersByTitleSubsequence(t *testing.T) {
	items := []db.SearchItem{
		{Title: "How Linux handles memory", FeedName: "Habr"},
		{Title: "Rust async explained", FeedName: "Hacker News"},
		{Title: "Linux kernel library", FeedName: "Hacker News"},
		{Title: "Go generics deep dive", FeedName: "Lobsters"},
	}
	m := mkModelWith(items, "linux")
	titles := matchedTitles(m)
	if len(titles) != 2 {
		t.Fatalf("want 2 linux matches, got %d: %v", len(titles), titles)
	}
	for _, tt := range titles {
		if !containsCI(tt, "linux") {
			t.Fatalf("match without 'linux': %q", tt)
		}
	}
}

func TestRecomputeMatches_FiltersByFeedName(t *testing.T) {
	items := []db.SearchItem{
		{Title: "How Linux handles memory", FeedName: "Habr"},
		{Title: "Rust async explained", FeedName: "Hacker News"},
		{Title: "Article about cats", FeedName: "Habr"},
	}
	m := mkModelWith(items, "habr")
	titles := matchedTitles(m)
	if len(titles) != 2 {
		t.Fatalf("want 2 habr matches, got %d: %v", len(titles), titles)
	}
	for _, idx := range m.searchMatches {
		if items[idx].FeedName != "Habr" {
			t.Fatalf("non-Habr match: %+v", items[idx])
		}
	}
}

func TestRecomputeMatches_NoMatchReturnsEmpty(t *testing.T) {
	items := []db.SearchItem{
		{Title: "aaa", FeedName: "x"},
		{Title: "bbb", FeedName: "y"},
	}
	m := mkModelWith(items, "zzz")
	if len(m.searchMatches) != 0 {
		t.Fatalf("want 0 matches, got %d", len(m.searchMatches))
	}
}

func TestRecomputeMatches_ClampsSelection(t *testing.T) {
	items := []db.SearchItem{
		{Title: "foo"},
		{Title: "bar"},
	}
	ti := textinput.New()
	ti.SetValue("")
	m := Model{searchInput: ti, searchAll: items, searchSel: 5}
	recomputeMatches(&m)
	if m.searchSel != 0 {
		t.Fatalf("want selection reset to 0, got %d", m.searchSel)
	}
}

func containsCI(hay, needle string) bool {
	hl := []rune(hay)
	nl := []rune(needle)
	for i := 0; i+len(nl) <= len(hl); i++ {
		ok := true
		for j := range nl {
			a := hl[i+j]
			b := nl[j]
			if a >= 'A' && a <= 'Z' {
				a += 32
			}
			if b >= 'A' && b <= 'Z' {
				b += 32
			}
			if a != b {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}
