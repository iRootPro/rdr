package ui

import (
	"strings"
	"testing"
)

func TestHelpViewUsesCompactDotSeparatedActions(t *testing.T) {
	m := Model{
		tr:    testTR,
		keys:  defaultKeys(testTR),
		focus: focusFeeds,
		width: 80,
	}

	line := stripANSI(m.helpView())
	if !strings.Contains(line, " · ") {
		t.Fatalf("help row should use compact middle-dot separators, got %q", line)
	}
	if strings.Contains(line, " • ") {
		t.Fatalf("help row should not use the heavier default bullet separator, got %q", line)
	}
}

func TestReaderScrollLabel_AllWhenContentFits(t *testing.T) {
	if got := readerScrollLabel(12, 20, 0); got != "ALL" {
		t.Fatalf("content shorter than viewport: got %q, want ALL", got)
	}
	if got := readerScrollLabel(20, 20, 0); got != "ALL" {
		t.Fatalf("content equal to viewport: got %q, want ALL", got)
	}
}

func TestReaderScrollLabel_ScrollingStates(t *testing.T) {
	if got := readerScrollLabel(100, 20, 0); got != "TOP" {
		t.Fatalf("at top: got %q, want TOP", got)
	}
	if got := readerScrollLabel(100, 20, 40); got != "50%" {
		t.Fatalf("middle: got %q, want 50%%", got)
	}
	if got := readerScrollLabel(100, 20, 80); got != "BOT" {
		t.Fatalf("at bottom: got %q, want BOT", got)
	}
}
