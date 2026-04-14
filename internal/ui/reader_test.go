package ui

import (
	"strings"
	"testing"
)

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
