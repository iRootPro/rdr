package feed

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// ResolvedFeed is a user-facing URL resolved to an RSS/Atom feed URL.
type ResolvedFeed struct {
	Name string
	URL  string
}

// IsYouTubeURL reports whether raw points at a YouTube host we know how to
// resolve to the built-in channel RSS feed.
func IsYouTubeURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return false
	}
	h := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	return h == "youtube.com" || h == "m.youtube.com" || h == "youtu.be"
}

// ResolveYouTube resolves a YouTube channel/handle/video URL to YouTube's
// RSS endpoint: https://www.youtube.com/feeds/videos.xml?channel_id=...
func (f *Fetcher) ResolveYouTube(ctx context.Context, raw string) (ResolvedFeed, bool, error) {
	if !IsYouTubeURL(raw) {
		return ResolvedFeed{}, false, nil
	}
	return resolveYouTube(ctx, f.client, raw)
}

func resolveYouTube(ctx context.Context, client *http.Client, raw string) (ResolvedFeed, bool, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ResolvedFeed{}, true, fmt.Errorf("parse youtube url: %w", err)
	}
	if id := channelIDFromURL(u); id != "" {
		return ResolvedFeed{URL: youtubeRSSURL(id)}, true, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return ResolvedFeed{}, true, fmt.Errorf("build youtube request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return ResolvedFeed{}, true, fmt.Errorf("fetch youtube page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return ResolvedFeed{}, true, fmt.Errorf("youtube http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return ResolvedFeed{}, true, fmt.Errorf("read youtube page: %w", err)
	}
	name, rss, id := parseYouTubeHTML(string(body))
	if rss == "" && id != "" {
		rss = youtubeRSSURL(id)
	}
	if rss == "" {
		return ResolvedFeed{}, true, fmt.Errorf("youtube channel id not found")
	}
	return ResolvedFeed{Name: name, URL: rss}, true, nil
}

func channelIDFromURL(u *url.URL) string {
	if strings.Contains(u.Path, "/feeds/videos.xml") {
		return u.Query().Get("channel_id")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "channel" && strings.HasPrefix(parts[1], "UC") {
		return parts[1]
	}
	return ""
}

func youtubeRSSURL(channelID string) string {
	return "https://www.youtube.com/feeds/videos.xml?channel_id=" + url.QueryEscape(channelID)
}

var (
	ytRSSRe          = regexp.MustCompile(`(?i)<link[^>]+type=["']application/rss\+xml["'][^>]+href=["']([^"']+)["']`)
	ytChannelIDRes   = []*regexp.Regexp{regexp.MustCompile(`"channelId"\s*:\s*"(UC[^"]+)"`), regexp.MustCompile(`"externalId"\s*:\s*"(UC[^"]+)"`), regexp.MustCompile(`channel_id=([^"'&]+)`)}
	ytOGTitleRe      = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:title["'][^>]+content=["']([^"']+)["']`)
	ytTitleRe        = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	ytAttrTitleInRSS = regexp.MustCompile(`(?i)<link[^>]+type=["']application/rss\+xml["'][^>]+title=["']([^"']+)["']`)
)

func parseYouTubeHTML(s string) (name, rss, channelID string) {
	if m := ytRSSRe.FindStringSubmatch(s); len(m) == 2 {
		rss = html.UnescapeString(m[1])
	}
	for _, re := range ytChannelIDRes {
		if m := re.FindStringSubmatch(s); len(m) == 2 {
			channelID = html.UnescapeString(m[1])
			break
		}
	}
	for _, re := range []*regexp.Regexp{ytAttrTitleInRSS, ytOGTitleRe, ytTitleRe} {
		if m := re.FindStringSubmatch(s); len(m) == 2 {
			name = cleanYouTubeTitle(m[1])
			if name != "" {
				break
			}
		}
	}
	return name, rss, channelID
}

func cleanYouTubeTitle(s string) string {
	s = html.UnescapeString(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, " - YouTube")
	s = strings.TrimSuffix(s, "- YouTube")
	return strings.TrimSpace(s)
}
