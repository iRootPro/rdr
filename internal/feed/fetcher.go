package feed

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	readability "github.com/go-shiori/go-readability"
	"github.com/mmcdole/gofeed"
	"golang.org/x/sync/errgroup"

	"github.com/iRootPro/rdr/internal/db"
)

const (
	userAgent            = "rdr/0.1 (+https://github.com/iRootPro/rdr)"
	maxConcurrentFetches = 8
	// readGracePeriod is the window during which freshly read articles
	// survive trim, even if the feed is over its per-feed cap. Protects
	// against accidental `x` keystrokes destroying rows the user might
	// still want to revisit.
	readGracePeriod = 7 * 24 * time.Hour
)

type FetchResult struct {
	Feed    db.Feed
	Added   int
	Updated int
	Err     error
}

type Fetcher struct {
	db     *db.DB
	client *http.Client
}

func New(d *db.DB) *Fetcher {
	return &Fetcher{
		db:     d,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *Fetcher) FetchOne(ctx context.Context, feed db.Feed) (FetchResult, error) {
	body, err := f.get(ctx, feed.URL)
	if err != nil {
		return FetchResult{}, err
	}
	defer body.Close()

	// gofeed.Parser lazy-inits shared fields on Parse, so it is not safe for
	// concurrent use. Constructing a fresh parser per call sidesteps the race.
	parsed, err := gofeed.NewParser().Parse(body)
	if err != nil {
		return FetchResult{}, fmt.Errorf("parse feed: %w", err)
	}

	// Snapshot the fetch start time before any upserts so TrimArticles
	// can protect anything touched by this cycle. Articles upserted
	// below will have last_fetched_at >= fetchStart.
	fetchStart := time.Now().UTC()

	result := FetchResult{Feed: feed}
	for _, item := range parsed.Items {
		article := mapItem(feed.ID, item)
		inserted, err := f.db.UpsertArticle(article)
		if err != nil {
			return FetchResult{}, fmt.Errorf("upsert: %w", err)
		}
		if inserted {
			result.Added++
		} else {
			result.Updated++
		}
	}
	// Trim failure is fatal for now — there's no logger wired up, and silent
	// swallow is worse than propagating. Revisit once Fetcher has warnings.
	// Articles read within readGracePeriod are protected from trim so a
	// stray `x` keystroke doesn't permanently destroy a row before the
	// user notices.
	readCutoff := fetchStart.Add(-readGracePeriod)
	if err := f.db.TrimArticles(feed.ID, f.maxArticlesPerFeed(), fetchStart, readCutoff); err != nil {
		return FetchResult{}, fmt.Errorf("trim: %w", err)
	}
	return result, nil
}

func (f *Fetcher) FetchFull(ctx context.Context, articleURL string) (string, error) {
	_, md, err := f.FetchFullWithTitle(ctx, articleURL)
	return md, err
}

// FetchFullWithTitle fetches a URL, extracts the main article via
// go-readability, and returns the page title alongside the Markdown
// body. Title is whatever readability resolved (often the <title> tag
// or first <h1>); callers should keep their own placeholder when this
// returns an empty string.
func (f *Fetcher) FetchFullWithTitle(ctx context.Context, articleURL string) (string, string, error) {
	body, err := f.get(ctx, articleURL)
	if err != nil {
		return "", "", err
	}
	defer body.Close()

	raw, err := io.ReadAll(body)
	if err != nil {
		return "", "", fmt.Errorf("read body: %w", err)
	}

	parsed, err := url.Parse(articleURL)
	if err != nil {
		return "", "", fmt.Errorf("parse url: %w", err)
	}
	article, err := readability.FromReader(bytes.NewReader(raw), parsed)
	if err != nil {
		return "", "", fmt.Errorf("readability: %w", err)
	}

	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
		),
	)
	md, err := conv.ConvertString(article.Content)
	if err != nil {
		return "", "", fmt.Errorf("html to markdown: %w", err)
	}
	return article.Title, md, nil
}

func (f *Fetcher) maxArticlesPerFeed() int {
	const fallback = 50
	v, err := f.db.GetSetting("max_articles_per_feed")
	if err != nil || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func (f *Fetcher) FetchAll(ctx context.Context) ([]FetchResult, error) {
	feeds, err := f.db.ListFeeds()
	if err != nil {
		return nil, fmt.Errorf("list feeds: %w", err)
	}
	// ListFeeds already excludes system feeds (e.g. Library), so we don't
	// need an extra filter here. Kept as a defensive comment in case the
	// invariant changes.
	results := make([]FetchResult, len(feeds))
	g, gctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, maxConcurrentFetches)

	for i, feed := range feeds {
		g.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-gctx.Done():
				return gctx.Err()
			}
			defer func() { <-sem }()

			r, err := f.FetchOne(gctx, feed)
			if err != nil {
				results[i] = FetchResult{Feed: feed, Err: err}
				return nil
			}
			results[i] = r
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

func (f *Fetcher) get(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	if resp.StatusCode >= 400 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func mapItem(feedID int64, item *gofeed.Item) db.Article {
	a := db.Article{
		FeedID:      feedID,
		Title:       item.Title,
		URL:         item.Link,
		Description: item.Description,
		Content:     item.Content,
	}
	if a.Content == "" {
		a.Content = item.Description
	}
	if a.Title == "" {
		a.Title = "(без заголовка)"
	}
	if item.PublishedParsed != nil {
		a.PublishedAt = *item.PublishedParsed
	} else if item.UpdatedParsed != nil {
		a.PublishedAt = *item.UpdatedParsed
	} else {
		a.PublishedAt = time.Now().UTC()
	}
	return a
}
