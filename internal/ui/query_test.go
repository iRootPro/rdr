package ui

import (
	"testing"
	"time"

	"github.com/iRootPro/rdr/internal/db"
)

func mkItem(title, feed, desc string, pub time.Time, read, starred bool) db.SearchItem {
	it := db.SearchItem{
		Title:       title,
		FeedName:    feed,
		Description: desc,
		PublishedAt: pub,
	}
	if read {
		t := pub.Add(time.Hour)
		it.ReadAt = &t
	}
	if starred {
		t := pub.Add(time.Hour)
		it.StarredAt = &t
	}
	return it
}

func TestParseQuery_StatusAtoms(t *testing.T) {
	atoms, err := ParseQuery("unread starred")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(atoms) != 2 {
		t.Fatalf("want 2 atoms, got %d", len(atoms))
	}
	if atoms[0].Kind != atomStatusRead || atoms[0].StatusValue != false {
		t.Fatalf("first atom wrong: %+v", atoms[0])
	}
	if atoms[1].Kind != atomStatusStar || atoms[1].StatusValue != true {
		t.Fatalf("second atom wrong: %+v", atoms[1])
	}
}

func TestParseQuery_Negation(t *testing.T) {
	atoms, err := ParseQuery("~read ~title:sponsor")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(atoms) != 2 {
		t.Fatalf("want 2 atoms, got %d", len(atoms))
	}
	for _, a := range atoms {
		if !a.Negate {
			t.Fatalf("atom not negated: %+v", a)
		}
	}
}

func TestParseQuery_FieldAtoms(t *testing.T) {
	atoms, err := ParseQuery("title:rust feed:habr description:memory")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(atoms) != 3 {
		t.Fatalf("want 3 atoms, got %d", len(atoms))
	}
	for i, want := range []string{"title", "feed", "description"} {
		if atoms[i].Kind != atomField || atoms[i].Field != want {
			t.Fatalf("atom %d: got %+v, want field=%s", i, atoms[i], want)
		}
	}
}

func TestParseQuery_FreeWord(t *testing.T) {
	atoms, err := ParseQuery("kernel")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(atoms) != 1 || atoms[0].Kind != atomFreeWord || atoms[0].Value != "kernel" {
		t.Fatalf("wrong atom: %+v", atoms)
	}
}

func TestParseQuery_UnknownQualifierBecomesFreeWord(t *testing.T) {
	atoms, err := ParseQuery("author:torvalds")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Unsupported qualifier → treated as free word with full token
	if len(atoms) != 1 || atoms[0].Kind != atomFreeWord {
		t.Fatalf("want free-word fallback, got %+v", atoms)
	}
}

func TestParseQuery_DurationNewer(t *testing.T) {
	atoms, err := ParseQuery("newer:1w")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(atoms) != 1 || atoms[0].Kind != atomTimeNewer {
		t.Fatalf("wrong atom: %+v", atoms)
	}
	// Since should be ~7 days in the past
	delta := time.Since(atoms[0].Since)
	if delta < 6*24*time.Hour || delta > 8*24*time.Hour {
		t.Fatalf("unexpected newer:1w since delta: %v", delta)
	}
}

func TestParseQuery_DurationOlder(t *testing.T) {
	atoms, err := ParseQuery("older:2d")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(atoms) != 1 || atoms[0].Kind != atomTimeOlder {
		t.Fatalf("wrong atom: %+v", atoms)
	}
}

func TestParseQuery_BadDurationErrors(t *testing.T) {
	_, err := ParseQuery("newer:abc")
	if err == nil {
		t.Fatalf("expected error for bad duration")
	}
}

func TestParseQuery_TodayYesterday(t *testing.T) {
	for _, kw := range []string{"today", "yesterday"} {
		atoms, err := ParseQuery(kw)
		if err != nil {
			t.Fatalf("%s: parse err %v", kw, err)
		}
		if len(atoms) != 1 || atoms[0].Kind != atomTimeBetween {
			t.Fatalf("%s: wrong atom: %+v", kw, atoms)
		}
		if atoms[0].Until.Sub(atoms[0].Since) != 24*time.Hour {
			t.Fatalf("%s: window != 24h: %v", kw, atoms[0])
		}
	}
}

func TestEvalQuery_EmptyMatchesAll(t *testing.T) {
	it := mkItem("anything", "any", "", time.Now(), false, false)
	if !EvalQuery(nil, it) {
		t.Fatalf("empty query should match")
	}
}

func TestEvalQuery_FreeWordMatchesTitleOrFeed(t *testing.T) {
	now := time.Now()
	atoms, _ := ParseQuery("habr")
	cases := []struct {
		it   db.SearchItem
		want bool
	}{
		{mkItem("Linux kernel", "Habr", "", now, false, false), true}, // feed matches
		{mkItem("habr explained", "Other", "", now, false, false), true},
		{mkItem("nothing here", "Other", "", now, false, false), false},
	}
	for i, c := range cases {
		got := EvalQuery(atoms, c.it)
		if got != c.want {
			t.Fatalf("case %d: got %v want %v for %+v", i, got, c.want, c.it)
		}
	}
}

func TestEvalQuery_UnreadFiltersReadArticles(t *testing.T) {
	now := time.Now()
	atoms, _ := ParseQuery("unread")
	unreadItem := mkItem("x", "f", "", now, false, false)
	readItem := mkItem("x", "f", "", now, true, false)
	if !EvalQuery(atoms, unreadItem) {
		t.Fatalf("unread should match unread item")
	}
	if EvalQuery(atoms, readItem) {
		t.Fatalf("unread should NOT match read item")
	}
}

func TestEvalQuery_StarredFiltersCorrectly(t *testing.T) {
	now := time.Now()
	atoms, _ := ParseQuery("starred")
	if !EvalQuery(atoms, mkItem("x", "f", "", now, false, true)) {
		t.Fatalf("starred should match starred item")
	}
	if EvalQuery(atoms, mkItem("x", "f", "", now, false, false)) {
		t.Fatalf("starred should NOT match unstarred item")
	}
}

func TestEvalQuery_NegationOnTitle(t *testing.T) {
	now := time.Now()
	atoms, _ := ParseQuery("~title:sponsor")
	normal := mkItem("How Linux works", "Feed", "", now, false, false)
	spammy := mkItem("Sponsored content", "Feed", "", now, false, false)
	if !EvalQuery(atoms, normal) {
		t.Fatalf("negation should keep non-matching items")
	}
	if EvalQuery(atoms, spammy) {
		t.Fatalf("negation should exclude matching items")
	}
}

func TestEvalQuery_ConjunctionOfMultipleAtoms(t *testing.T) {
	now := time.Now()
	atoms, err := ParseQuery("unread feed:habr title:rust")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	match := mkItem("Why Rust is great", "Habr", "", now, false, false)
	wrongFeed := mkItem("Why Rust is great", "HN", "", now, false, false)
	wrongTitle := mkItem("Why Go is great", "Habr", "", now, false, false)
	readItem := mkItem("Why Rust is great", "Habr", "", now, true, false)

	if !EvalQuery(atoms, match) {
		t.Fatalf("should match all conditions")
	}
	if EvalQuery(atoms, wrongFeed) {
		t.Fatalf("should not match different feed")
	}
	if EvalQuery(atoms, wrongTitle) {
		t.Fatalf("should not match different title word")
	}
	if EvalQuery(atoms, readItem) {
		t.Fatalf("should not match read item")
	}
}

func TestEvalQuery_TimeNewer(t *testing.T) {
	atoms, _ := ParseQuery("newer:1d")
	fresh := mkItem("x", "f", "", time.Now().Add(-2*time.Hour), false, false)
	old := mkItem("x", "f", "", time.Now().Add(-48*time.Hour), false, false)
	if !EvalQuery(atoms, fresh) {
		t.Fatalf("newer:1d should match 2h-old item")
	}
	if EvalQuery(atoms, old) {
		t.Fatalf("newer:1d should not match 48h-old item")
	}
}
