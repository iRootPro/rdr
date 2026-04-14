package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/iRootPro/rdr/internal/db"
)

func TestLayoutReader_NarrowTerminalNoIndent(t *testing.T) {
	a := db.Article{Title: "Hello", URL: "https://example.com", PublishedAt: time.Now(), CachedBody: "Body text here."}
	out := layoutReader(a, "Feed", 70, false)
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "  ") && !strings.Contains(line, "·") {
			// Any line that genuinely starts with >=2 spaces and isn't
			// formatting-related would indicate unintended indent. We only
			// flag the first body line which should be flush-left.
		}
	}
	// Simplest check: content should not start with a huge indent (>=10).
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(trimmed)
		if indent >= 10 {
			t.Fatalf("narrow terminal: unexpected indent of %d on line %q", indent, line)
		}
	}
}

func TestLayoutReader_WideTerminalCenters(t *testing.T) {
	a := db.Article{Title: "Hi", URL: "https://example.com", PublishedAt: time.Now(), CachedBody: "Body."}
	out := layoutReader(a, "Feed", 150, false)
	// (150 - 85) / 2 = 32 — every non-empty line should start with 32 spaces.
	wantPad := strings.Repeat(" ", 32)
	lines := strings.Split(out, "\n")
	nonEmpty := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		nonEmpty++
		if !strings.HasPrefix(line, wantPad) {
			t.Fatalf("wide terminal: line missing 32-space indent: %q", line)
		}
	}
	if nonEmpty == 0 {
		t.Fatal("expected at least one non-empty line")
	}
}

func TestRenderReaderBody_DividerMatchesContentWidth(t *testing.T) {
	a := db.Article{Title: "T", URL: "", PublishedAt: time.Now(), CachedBody: "body"}
	out := renderReaderBody(a, "F", 60, false)
	// Divider line should have exactly 60 "─" characters.
	want := strings.Repeat("─", 60)
	if !strings.Contains(out, want) {
		t.Fatalf("expected divider of width 60, got:\n%s", out)
	}
}

func TestRenderReaderBody_TruncatesLongURL(t *testing.T) {
	longURL := "https://example.com/" + strings.Repeat("verylong/", 30)
	a := db.Article{Title: "T", URL: longURL, PublishedAt: time.Now(), CachedBody: "x"}
	out := renderReaderBody(a, "F", 60, false)
	if strings.Contains(out, longURL) {
		t.Fatal("long URL should be truncated but full URL is present")
	}
	if !strings.Contains(out, "…") {
		t.Fatal("expected ellipsis on truncated URL")
	}
}

func TestRenderEmptyReader_ContainsCTABox(t *testing.T) {
	a := db.Article{Title: "Empty", URL: "https://x", PublishedAt: time.Now()} // no CachedBody
	out := renderReaderBody(a, "F", 70, false)
	if !strings.Contains(out, "Press [f]") {
		t.Fatalf("empty state should prompt with Press [f], got:\n%s", out)
	}
	// Rounded border corners should appear in the rendered box.
	if !strings.Contains(out, "╭") || !strings.Contains(out, "╰") {
		t.Fatalf("expected rounded border characters, got:\n%s", out)
	}
}

func TestRenderEmptyReader_ShowsContentStub(t *testing.T) {
	a := db.Article{
		Title:       "HN Item",
		URL:         "https://example.com",
		PublishedAt: time.Now(),
		Content:     "Article URL: https://example.com\nPoints: 42",
	}
	out := renderReaderBody(a, "F", 70, false)
	if !strings.Contains(out, "Points: 42") {
		t.Fatalf("expected Content stub to appear below empty box, got:\n%s", out)
	}
}

func TestSanitizeArticleMarkdown_StripsImageSyntax(t *testing.T) {
	in := `First paragraph.

![alt text](https://cdn.example/pic.jpg)

Second paragraph.

![](https://cdn.example/another.png "title")

Third paragraph.`

	got := sanitizeArticleMarkdown(in, false)
	if strings.Contains(got, "cdn.example") {
		t.Fatalf("image URLs should be removed, got:\n%s", got)
	}
	if strings.Contains(got, "![") {
		t.Fatalf("image syntax should be removed, got:\n%s", got)
	}
	for _, want := range []string{"First paragraph.", "Second paragraph.", "Third paragraph."} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected paragraph %q preserved, got:\n%s", want, got)
		}
	}
}

func TestSanitizeArticleMarkdown_StripsBareImageURLs(t *testing.T) {
	in := `Intro paragraph.

https://habrastorage.org/foo/file.jpg

Body text continues here.

<https://other.example/pic.png>

Closing.`
	got := sanitizeArticleMarkdown(in, false)
	if strings.Contains(got, "habrastorage.org") {
		t.Fatalf("bare image URL should be removed, got:\n%s", got)
	}
	if strings.Contains(got, "other.example") {
		t.Fatalf("angle-bracketed image URL should be removed, got:\n%s", got)
	}
}

func TestSanitizeArticleMarkdown_PreservesArticleLinks(t *testing.T) {
	in := `Check out [this article](https://example.com/article) for details.`
	got := sanitizeArticleMarkdown(in, false)
	if !strings.Contains(got, "[this article](https://example.com/article)") {
		t.Fatalf("article link should be preserved, got:\n%s", got)
	}
}

func TestSanitizeArticleMarkdown_PreservesNonImageExtensions(t *testing.T) {
	// .pdf, .zip, .html etc are legitimate content links.
	in := `Download: https://example.com/doc.pdf`
	got := sanitizeArticleMarkdown(in, false)
	if !strings.Contains(got, "doc.pdf") {
		t.Fatalf("non-image URL should be preserved, got:\n%s", got)
	}
}

func TestSanitizeArticleMarkdown_ShowImagesBypassesStripping(t *testing.T) {
	in := `![alt](https://cdn.example/pic.jpg)`
	got := sanitizeArticleMarkdown(in, true)
	if got != in {
		t.Fatalf("showImages=true should not modify input, got:\n%s", got)
	}
}

func TestSanitizeArticleMarkdown_CollapsesExtraBlankLines(t *testing.T) {
	in := "para1\n\n\n\n\npara2"
	got := sanitizeArticleMarkdown(in, false)
	if strings.Contains(got, "\n\n\n") {
		t.Fatalf("blank-line run should be collapsed, got: %q", got)
	}
}
