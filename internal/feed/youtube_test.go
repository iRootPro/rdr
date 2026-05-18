package feed

import (
	"net/url"
	"testing"
	"time"
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

func TestParseYouTubeVideosHTML_UsesSelectedVideosTabOnly(t *testing.T) {
	feed, err := parseYouTubeVideosHTML(`<!doctype html><html><head><title>Example - YouTube</title></head><body><script>var ytInitialData = {
		"contents": {
			"twoColumnBrowseResultsRenderer": {
				"tabs": [
					{
						"tabRenderer": {
							"title": "Home",
							"content": {
								"lockupViewModel": {
									"contentId": "other123",
									"contentType": "LOCKUP_CONTENT_TYPE_VIDEO",
									"metadata": {
										"lockupMetadataViewModel": {
											"title": {"content": "Unrelated video"}
										}
									}
								}
							}
						}
					},
					{
						"tabRenderer": {
							"title": "Videos",
							"selected": true,
							"content": {
								"lockupViewModel": {
									"contentId": "selected123",
									"contentType": "LOCKUP_CONTENT_TYPE_VIDEO",
									"metadata": {
										"lockupMetadataViewModel": {
											"title": {"content": "Selected video"},
											"metadata": {
												"contentMetadataViewModel": {
													"metadataRows": [
														{"metadataParts": [
															{"text": {"content": "12 views"}},
															{"text": {"content": "2 weeks ago"}}
														]}
													]
												}
											}
										}
									}
								}
							}
						}
					}
				]
			}
		}
	};</script></body></html>`)
	if err != nil {
		t.Fatalf("parseYouTubeVideosHTML: %v", err)
	}
	if len(feed.Items) != 1 {
		t.Fatalf("items: got %d, want 1", len(feed.Items))
	}
	if feed.Items[0].GUID != "selected123" || feed.Items[0].Title != "Selected video" {
		t.Fatalf("item: %+v", feed.Items[0])
	}
	if feed.Items[0].PublishedParsed == nil {
		t.Fatal("PublishedParsed should be parsed from the lockup metadata")
	}

}

func TestParseYouTubeRelativeTime(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	got := parseYouTubeRelativeTime("2 weeks ago", now)
	if got == nil {
		t.Fatal("got nil")
	}
	want := now.Add(-14 * 24 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("got %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestShouldFetchYouTubeHTMLFallback(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		code int
		want bool
	}{
		{"youtube 404", "https://www.youtube.com/feeds/videos.xml?channel_id=UCabc", 404, true},
		{"youtube 503", "https://www.youtube.com/feeds/videos.xml?channel_id=UCabc", 503, true},
		{"youtube other 4xx", "https://www.youtube.com/feeds/videos.xml?channel_id=UCabc", 403, false},
		{"non youtube", "https://example.com/feeds/videos.xml?channel_id=UCabc", 404, false},
	}
	for _, tc := range cases {
		if got := shouldFetchYouTubeHTMLFallback(tc.raw, &httpStatusError{code: tc.code}); got != tc.want {
			t.Fatalf("%s: got %v, want %v", tc.name, got, tc.want)
		}
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
