package ui

import "testing"

func TestExtractLinks_MarkdownLinks(t *testing.T) {
	md := `Read [the spec](https://example.com/spec) and [the blog](https://blog.example.com).`
	got := extractLinks(md)
	if len(got) != 2 {
		t.Fatalf("want 2 links, got %d: %+v", len(got), got)
	}
	if got[0].Text != "the spec" || got[0].URL != "https://example.com/spec" {
		t.Fatalf("link 0: %+v", got[0])
	}
	if got[1].Text != "the blog" || got[1].URL != "https://blog.example.com" {
		t.Fatalf("link 1: %+v", got[1])
	}
}

func TestExtractLinks_SkipsImages(t *testing.T) {
	md := `Image first: ![alt text](https://cdn.example.com/pic.jpg)
Then a real link: [docs](https://example.com/docs)`
	got := extractLinks(md)
	if len(got) != 1 {
		t.Fatalf("want 1 link (image excluded), got %d: %+v", len(got), got)
	}
	if got[0].URL != "https://example.com/docs" {
		t.Fatalf("wrong link: %+v", got[0])
	}
}

func TestExtractLinks_BareURLs(t *testing.T) {
	md := `Check https://example.com/article for details.
Also see https://foo.example.com/path?q=1.`
	got := extractLinks(md)
	if len(got) != 2 {
		t.Fatalf("want 2 bare URLs, got %d: %+v", len(got), got)
	}
	// Trailing '.' should be trimmed.
	for _, l := range got {
		if l.URL == "" || l.URL[len(l.URL)-1] == '.' {
			t.Fatalf("trailing punctuation not trimmed: %q", l.URL)
		}
	}
}

func TestExtractLinks_DedupesByURL(t *testing.T) {
	md := `Same link: [first](https://example.com) and [second](https://example.com)`
	got := extractLinks(md)
	if len(got) != 1 {
		t.Fatalf("want 1 deduped link, got %d: %+v", len(got), got)
	}
	if got[0].Text != "first" {
		t.Fatalf("dedup kept wrong label: %+v", got[0])
	}
}

func TestExtractLinks_SkipsBareImageURLs(t *testing.T) {
	md := `Standalone URL: https://cdn.example.com/image.png
Article: https://example.com/post`
	got := extractLinks(md)
	if len(got) != 1 {
		t.Fatalf("want 1 non-image link, got %d: %+v", len(got), got)
	}
	if got[0].URL != "https://example.com/post" {
		t.Fatalf("wrong link: %+v", got[0])
	}
}

func TestExtractLinks_EmptyInputReturnsNil(t *testing.T) {
	if got := extractLinks(""); got != nil {
		t.Fatalf("want nil, got %+v", got)
	}
}

func TestExtractLinks_MarkdownLinksPreferredOverBare(t *testing.T) {
	// A bare URL that also appears inside a markdown link should not
	// produce a duplicate entry.
	md := `See [the docs](https://example.com/docs). Full URL: https://example.com/docs`
	got := extractLinks(md)
	if len(got) != 1 {
		t.Fatalf("want 1 link, got %d: %+v", len(got), got)
	}
	if got[0].Text != "the docs" {
		t.Fatalf("markdown form should win: %+v", got[0])
	}
}
