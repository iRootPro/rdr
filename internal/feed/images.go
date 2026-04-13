package feed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

const imageFetchConcurrency = 4

// DownloadImages fetches all URLs into cacheDir, returning a url→path
// map. Files are named after the SHA256 of the URL so the same URL
// resolves to the same path across runs. Already-cached URLs are
// skipped; network errors on individual URLs are logged-and-ignored
// so one bad image doesn't break the batch.
func DownloadImages(ctx context.Context, urls []string, cacheDir string) (map[string]string, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir cache: %w", err)
	}

	paths := make(map[string]string, len(urls))
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, imageFetchConcurrency)

	client := &http.Client{Timeout: 15 * time.Second}

	for _, u := range urls {
		g.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-gctx.Done():
				return gctx.Err()
			}
			defer func() { <-sem }()

			dest := filepath.Join(cacheDir, ImageFileName(u))
			if _, err := os.Stat(dest); err == nil {
				mu.Lock()
				paths[u] = dest
				mu.Unlock()
				return nil
			}

			req, err := http.NewRequestWithContext(gctx, http.MethodGet, u, nil)
			if err != nil {
				return nil
			}
			req.Header.Set("User-Agent", userAgent)
			resp, err := client.Do(req)
			if err != nil {
				return nil
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				return nil
			}
			tmp, err := os.CreateTemp(cacheDir, ".dl-*")
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(tmp, resp.Body)
			closeErr := tmp.Close()
			if copyErr != nil || closeErr != nil {
				_ = os.Remove(tmp.Name())
				return nil
			}
			if err := os.Rename(tmp.Name(), dest); err != nil {
				_ = os.Remove(tmp.Name())
				return err
			}
			mu.Lock()
			paths[u] = dest
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return paths, err
	}
	return paths, nil
}

// ImageFileName returns a deterministic filename for the given image URL.
// The filename is the first 16 bytes of the URL's SHA256 in hex, plus
// the URL's file extension (if it's a sane-looking ASCII extension).
func ImageFileName(u string) string {
	sum := sha256.Sum256([]byte(u))
	ext := ".img"
	if idx := strings.LastIndex(u, "."); idx != -1 && idx > len(u)-6 {
		ext = u[idx:]
		if strings.ContainsAny(ext, "?#/&") {
			ext = ".img"
		}
	}
	return hex.EncodeToString(sum[:16]) + ext
}
