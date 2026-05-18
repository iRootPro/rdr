package feed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
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

const youtubeBrowserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

func shouldFetchYouTubeHTMLFallback(raw string, err error) bool {
	var statusErr *httpStatusError
	if !errors.As(err, &statusErr) {
		return false
	}
	if statusErr.code != http.StatusNotFound && statusErr.code < 500 {
		return false
	}
	return youtubeRSSChannelID(raw) != ""
}

func youtubeRSSChannelID(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	h := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	if h != "youtube.com" && h != "m.youtube.com" {
		return ""
	}
	if u.Path != "/feeds/videos.xml" {
		return ""
	}
	return u.Query().Get("channel_id")
}

func youtubeChannelVideosURL(channelID string) string {
	return "https://www.youtube.com/channel/" + url.PathEscape(channelID) + "/videos"
}

func youtubeWatchURL(videoID string) string {
	return "https://www.youtube.com/watch?v=" + url.QueryEscape(videoID)
}

func (f *Fetcher) fetchYouTubeHTMLFeed(ctx context.Context, raw string) (*gofeed.Feed, error) {
	channelID := youtubeRSSChannelID(raw)
	if channelID == "" {
		return nil, fmt.Errorf("youtube channel id not found")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, youtubeChannelVideosURL(channelID), nil)
	if err != nil {
		return nil, fmt.Errorf("build youtube fallback request: %w", err)
	}
	req.Header.Set("User-Agent", youtubeBrowserUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch youtube channel page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("youtube channel page http %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("read youtube channel page: %w", err)
	}
	parsed, err := parseYouTubeVideosHTML(string(body))
	if err != nil {
		return nil, err
	}
	if parsed.Title == "" {
		parsed.Title = channelID
	}
	if parsed.Link == "" {
		parsed.Link = "https://www.youtube.com/channel/" + url.PathEscape(channelID)
	}
	return parsed, nil
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

func parseYouTubeVideosHTML(s string) (*gofeed.Feed, error) {
	initialData := extractYouTubeInitialData(s)
	if initialData == "" {
		return nil, fmt.Errorf("youtube initial data not found")
	}

	var root any
	if err := json.NewDecoder(strings.NewReader(initialData)).Decode(&root); err != nil {
		return nil, fmt.Errorf("parse youtube initial data: %w", err)
	}

	name, _, _ := parseYouTubeHTML(s)
	feed := &gofeed.Feed{Title: name}
	seen := make(map[string]struct{})
	now := time.Now().UTC()
	roots := selectedYouTubeTabContents(root, nil, 0)
	if len(roots) == 0 {
		return nil, fmt.Errorf("youtube videos tab not found")
	}
	for _, root := range roots {
		collectYouTubeVideoItems(root, feed, seen, 0, now)
	}
	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("youtube videos not found")
	}
	return feed, nil
}

func extractYouTubeInitialData(s string) string {
	for _, marker := range []string{"var ytInitialData =", "window[\"ytInitialData\"] =", "ytInitialData ="} {
		i := strings.Index(s, marker)
		if i < 0 {
			continue
		}
		start := i + len(marker)
		for start < len(s) && (s[start] == ' ' || s[start] == '\n' || s[start] == '\t' || s[start] == '\r') {
			start++
		}
		if start >= len(s) || s[start] != '{' {
			continue
		}
		end := jsonObjectEnd(s, start)
		if end > start {
			return s[start:end]
		}
	}
	return ""
}

func jsonObjectEnd(s string, start int) int {
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch c {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

func selectedYouTubeTabContents(v any, out []any, depth int) []any {
	if depth > 80 {
		return out
	}
	switch x := v.(type) {
	case map[string]any:
		if tab, ok := x["tabRenderer"].(map[string]any); ok {
			if selected, _ := tab["selected"].(bool); selected {
				if content := valueAt(tab, "content"); content != nil {
					out = append(out, content)
				}
			}
			return out
		}
		for _, child := range x {
			out = selectedYouTubeTabContents(child, out, depth+1)
		}
	case []any:
		for _, child := range x {
			out = selectedYouTubeTabContents(child, out, depth+1)
		}
	}
	return out
}

func collectYouTubeVideoItems(v any, feed *gofeed.Feed, seen map[string]struct{}, depth int, now time.Time) {
	if depth > 80 {
		return
	}
	switch x := v.(type) {
	case map[string]any:
		if child, ok := x["lockupViewModel"].(map[string]any); ok {
			item, itemOK := youtubeItemFromLockup(child, now)
			addYouTubeItem(feed, seen, item, itemOK)
		}
		if child, ok := x["videoRenderer"].(map[string]any); ok {
			item, itemOK := youtubeItemFromVideoRenderer(child, now)
			addYouTubeItem(feed, seen, item, itemOK)
		}
		if child, ok := x["gridVideoRenderer"].(map[string]any); ok {
			item, itemOK := youtubeItemFromVideoRenderer(child, now)
			addYouTubeItem(feed, seen, item, itemOK)
		}
		for _, child := range x {
			collectYouTubeVideoItems(child, feed, seen, depth+1, now)
		}
	case []any:
		for _, child := range x {
			collectYouTubeVideoItems(child, feed, seen, depth+1, now)
		}
	}
}

func addYouTubeItem(feed *gofeed.Feed, seen map[string]struct{}, item *gofeed.Item, ok bool) {
	if !ok {
		return
	}
	key := item.GUID
	if key == "" {
		key = item.Link
	}
	if key == "" {
		return
	}
	if _, exists := seen[key]; exists {
		return
	}
	seen[key] = struct{}{}
	feed.Items = append(feed.Items, item)
}

func youtubeItemFromLockup(m map[string]any, now time.Time) (*gofeed.Item, bool) {
	contentType := stringAt(m, "contentType")
	if contentType != "" && contentType != "LOCKUP_CONTENT_TYPE_VIDEO" {
		return nil, false
	}

	videoID := stringAt(m, "contentId")
	if videoID == "" {
		videoID = stringAt(m, "rendererContext", "commandContext", "onTap", "innertubeCommand", "watchEndpoint", "videoId")
	}
	if videoID == "" {
		videoID = youtubeVideoIDFromWatchURL(stringAt(m, "rendererContext", "commandContext", "onTap", "innertubeCommand", "commandMetadata", "webCommandMetadata", "url"))
	}

	title := youtubeText(valueAt(m, "metadata", "lockupMetadataViewModel", "title"))
	if title == "" {
		title = youtubeText(valueAt(m, "title"))
	}
	if videoID == "" || title == "" {
		return nil, false
	}

	published := youtubeLockupPublishedText(m)
	return &gofeed.Item{
		Title:           title,
		Link:            youtubeWatchURL(videoID),
		GUID:            videoID,
		Published:       published,
		PublishedParsed: parseYouTubeRelativeTime(published, now),
	}, true
}

func youtubeItemFromVideoRenderer(m map[string]any, now time.Time) (*gofeed.Item, bool) {
	videoID := stringAt(m, "videoId")
	title := youtubeText(valueAt(m, "title"))
	if videoID == "" || title == "" {
		return nil, false
	}

	published := youtubeText(valueAt(m, "publishedTimeText"))
	return &gofeed.Item{
		Title:           title,
		Link:            youtubeWatchURL(videoID),
		GUID:            videoID,
		Description:     youtubeText(valueAt(m, "descriptionSnippet")),
		Published:       published,
		PublishedParsed: parseYouTubeRelativeTime(published, now),
	}, true
}

func youtubeLockupPublishedText(m map[string]any) string {
	rows, _ := valueAt(m, "metadata", "lockupMetadataViewModel", "metadata", "contentMetadataViewModel", "metadataRows").([]any)
	for _, row := range rows {
		parts, _ := valueAt(row, "metadataParts").([]any)
		for _, part := range parts {
			text := youtubeText(valueAt(part, "text"))
			if _, ok := youtubeRelativeDuration(text); ok {
				return text
			}
		}
	}
	return ""
}

func parseYouTubeRelativeTime(s string, now time.Time) *time.Time {
	if d, ok := youtubeRelativeDuration(s); ok {
		t := now.Add(-d)
		return &t
	}
	return nil
}

func youtubeRelativeDuration(s string) (time.Duration, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "streamed ")
	fields := strings.Fields(s)
	for i := 0; i+1 < len(fields); i++ {
		n, err := strconv.Atoi(strings.Trim(fields[i], ",."))
		if err != nil {
			continue
		}
		unit := strings.Trim(fields[i+1], ",.")
		var d time.Duration
		switch {
		case strings.HasPrefix(unit, "second"):
			d = time.Duration(n) * time.Second
		case strings.HasPrefix(unit, "minute"):
			d = time.Duration(n) * time.Minute
		case strings.HasPrefix(unit, "hour"):
			d = time.Duration(n) * time.Hour
		case strings.HasPrefix(unit, "day"):
			d = time.Duration(n) * 24 * time.Hour
		case strings.HasPrefix(unit, "week"):
			d = time.Duration(n) * 7 * 24 * time.Hour
		case strings.HasPrefix(unit, "month"):
			d = time.Duration(n) * 30 * 24 * time.Hour
		case strings.HasPrefix(unit, "year"):
			d = time.Duration(n) * 365 * 24 * time.Hour
		default:
			continue
		}
		return d, true
	}
	return 0, false
}

func youtubeVideoIDFromWatchURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Path != "/watch" {
		return ""
	}
	return u.Query().Get("v")
}

func valueAt(v any, path ...string) any {
	for _, key := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[key]
	}
	return v
}

func stringAt(m map[string]any, path ...string) string {
	s, _ := valueAt(m, path...).(string)
	return s
}

func youtubeText(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(html.UnescapeString(x))
	case map[string]any:
		if s, _ := x["content"].(string); s != "" {
			return strings.TrimSpace(html.UnescapeString(s))
		}
		if s, _ := x["simpleText"].(string); s != "" {
			return strings.TrimSpace(html.UnescapeString(s))
		}
		runs, _ := x["runs"].([]any)
		if len(runs) == 0 {
			return ""
		}
		var b strings.Builder
		for _, run := range runs {
			if text := stringAtMap(run, "text"); text != "" {
				b.WriteString(text)
			}
		}
		return strings.TrimSpace(html.UnescapeString(b.String()))
	default:
		return ""
	}
}

func stringAtMap(v any, key string) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	s, _ := m[key].(string)
	return s
}
