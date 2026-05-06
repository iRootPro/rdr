package feed

import (
	"net/url"
	"testing"
)

func TestIsYouTubeURL(t *testing.T) {
	cases := map[string]bool{
		"https://www.youtube.com/@veritasium": true,
		"https://youtu.be/abc":                true,
		"https://example.com/@veritasium":     false,
	}
	for raw, want := range cases {
		if got := IsYouTubeURL(raw); got != want {
			t.Fatalf("IsYouTubeURL(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestParseYouTubeHTML_RSSAndTitle(t *testing.T) {
	name, rss, id := parseYouTubeHTML(`<html><head>
		<link rel="alternate" type="application/rss+xml" title="Veritasium" href="https://www.youtube.com/feeds/videos.xml?channel_id=UC123">
		<meta property="og:title" content="Fallback">
	</head></html>`)
	if name != "Veritasium" {
		t.Fatalf("name = %q", name)
	}
	if rss != "https://www.youtube.com/feeds/videos.xml?channel_id=UC123" {
		t.Fatalf("rss = %q", rss)
	}
	if id != "UC123" {
		t.Fatalf("id = %q", id)
	}
}

func TestChannelIDFromURL(t *testing.T) {
	u := mustParseURL(t, "https://www.youtube.com/channel/UCabcdef")
	if got := channelIDFromURL(u); got != "UCabcdef" {
		t.Fatalf("id = %q", got)
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
