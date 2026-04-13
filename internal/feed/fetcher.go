package feed

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
	"golang.org/x/sync/errgroup"

	"github.com/iRootPro/rdr/internal/db"
)

const (
	userAgent            = "rdr/0.1 (+https://github.com/iRootPro/rdr)"
	maxConcurrentFetches = 8
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
		client: &http.Client{Timeout: 15 * time.Second},
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
	return result, nil
}

func (f *Fetcher) FetchAll(ctx context.Context) ([]FetchResult, error) {
	feeds, err := f.db.ListFeeds()
	if err != nil {
		return nil, fmt.Errorf("list feeds: %w", err)
	}
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
