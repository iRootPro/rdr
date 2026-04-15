package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func keysContain(bindings []key.Binding, wantKey string) bool {
	for _, b := range bindings {
		for _, k := range b.Keys() {
			if k == wantKey {
				return true
			}
		}
	}
	return false
}

func TestShortHelpFor_FeedsHasTabAndSearch(t *testing.T) {
	k := defaultKeys()
	got := shortHelpFor(focusFeeds, k)
	if !keysContain(got, "tab") {
		t.Fatal("focusFeeds should include tab (switch pane)")
	}
	if !keysContain(got, "/") {
		t.Fatal("focusFeeds should include / (search)")
	}
	if keysContain(got, "f") {
		t.Fatal("focusFeeds should NOT include f (reader-only)")
	}
	if keysContain(got, "J") {
		t.Fatal("focusFeeds should NOT include J (reader-only)")
	}
}

func TestShortHelpFor_ReaderHasJumpAndFull(t *testing.T) {
	k := defaultKeys()
	got := shortHelpFor(focusReader, k)
	if !keysContain(got, "J") || !keysContain(got, "K") {
		t.Fatal("focusReader should include J/K (next/prev article)")
	}
	if !keysContain(got, "f") {
		t.Fatal("focusReader should include f (load full)")
	}
	if !keysContain(got, "L") {
		t.Fatal("focusReader should include L (link picker)")
	}
	if keysContain(got, "tab") {
		t.Fatal("focusReader should NOT include tab (no pane to switch to)")
	}
}

func TestShortHelpFor_CommandContextual(t *testing.T) {
	k := defaultKeys()
	got := shortHelpFor(focusCommand, k)
	if !keysContain(got, "enter") || !keysContain(got, "esc") {
		t.Fatal("focusCommand should include enter and esc")
	}
	// Command mode swallows ':' as literal input — don't advertise it.
	if keysContain(got, ":") {
		t.Fatal("focusCommand should NOT re-advertise ':' (already active)")
	}
}

func TestShortHelpFor_HelpOnlyHasClose(t *testing.T) {
	k := defaultKeys()
	got := shortHelpFor(focusHelp, k)
	if !keysContain(got, "esc") {
		t.Fatal("focusHelp should include esc to close")
	}
	if !keysContain(got, "?") {
		t.Fatal("focusHelp should include ? to toggle")
	}
}

func TestFullHelpFor_AllFocusesNonEmpty(t *testing.T) {
	all := []focus{
		focusFeeds, focusArticles, focusReader, focusSettings,
		focusSearch, focusCommand, focusLinks, focusHelp,
	}
	for _, f := range all {
		sections := fullHelpFor(f)
		if len(sections) == 0 {
			t.Fatalf("fullHelpFor(%v) returned no sections", focusLabel(f))
		}
		total := 0
		for _, sec := range sections {
			total += len(sec.Entries)
		}
		if total == 0 {
			t.Fatalf("fullHelpFor(%v) returned no entries", focusLabel(f))
		}
	}
}

func TestFullHelpFor_ReaderHasArticleOps(t *testing.T) {
	sections := fullHelpFor(focusReader)
	var articleSec *helpSection
	for i := range sections {
		if sections[i].Title == "Article ops" {
			articleSec = &sections[i]
			break
		}
	}
	if articleSec == nil {
		t.Fatal("focusReader full help should have 'Article ops' section")
	}
	wantKeys := []string{"f", "L", "y / Y", "x", "m"}
	for _, k := range wantKeys {
		found := false
		for _, e := range articleSec.Entries {
			if strings.Contains(e.Keys, k) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("focusReader Article ops missing %q", k)
		}
	}
}

func TestFullHelpFor_SearchHasQuerySyntax(t *testing.T) {
	sections := fullHelpFor(focusSearch)
	var hasSyntax bool
	for _, sec := range sections {
		if sec.Title == "Query syntax" {
			hasSyntax = true
			break
		}
	}
	if !hasSyntax {
		t.Fatal("focusSearch full help should include 'Query syntax' section")
	}
}

func TestFocusLabel_AllFocuses(t *testing.T) {
	cases := map[focus]string{
		focusFeeds:    "Feeds",
		focusArticles: "Articles",
		focusReader:   "Reader",
		focusSettings: "Settings",
		focusSearch:   "Search",
		focusCommand:  "Command",
		focusLinks:    "Links",
		focusHelp:     "Help",
	}
	for f, want := range cases {
		if got := focusLabel(f); got != want {
			t.Fatalf("focusLabel(%v) = %q, want %q", f, got, want)
		}
	}
}

func TestPadRight_ExactPadding(t *testing.T) {
	got := padRight("j / k", 10)
	if got != "j / k     " {
		t.Fatalf("padRight: got %q, want %q", got, "j / k     ")
	}
}

func TestPadRight_TruncatesLong(t *testing.T) {
	got := padRight("abcdefghij", 5)
	if got != "abcd…" {
		t.Fatalf("padRight: got %q, want %q", got, "abcd…")
	}
}
