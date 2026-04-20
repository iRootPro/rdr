package feed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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

// ToPNG ensures the given bytes are PNG-encoded. Kitty Graphics
// Protocol with f=100 requires PNG; we convert JPEG/GIF/etc. via the
// stdlib image package. Already-PNG input is returned unchanged.
func ToPNG(data []byte) ([]byte, image.Point, error) {
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		cfg, _, cerr := image.DecodeConfig(bytes.NewReader(data))
		if cerr != nil {
			return data, image.Point{}, nil
		}
		return data, image.Point{X: cfg.Width, Y: cfg.Height}, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, image.Point{}, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, image.Point{}, err
	}
	sz := img.Bounds().Size()
	return buf.Bytes(), sz, nil
}

// imgURLPattern matches markdown image references `![alt](url)`. The
// URL is captured in the first submatch. We intentionally accept spaces
// in the alt text and any non-paren chars in the URL.
var imgURLPattern = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)

// ExtractImageURLs scans the markdown for `![alt](url)` references and
// returns URLs in the order they appear, deduplicated. URLs without a
// scheme are skipped (local/relative refs aren't reachable here).
func ExtractImageURLs(md string) []string {
	matches := imgURLPattern.FindAllStringSubmatch(md, -1)
	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		u := strings.TrimSpace(m[1])
		if u == "" {
			continue
		}
		if !(strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")) {
			continue
		}
		if _, dup := seen[u]; dup {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}

// ImageID returns a deterministic 24-bit image ID derived from the URL.
// We fold SHA256 down to 24 bits so the low bytes fit in a standard
// truecolor FG triple (used by older placeholder approach). IDs are
// not strictly unique but collisions across one article are vanishingly
// rare and tolerable.
func ImageID(url string) uint32 {
	sum := sha256.Sum256([]byte(url))
	return uint32(sum[0])<<16 | uint32(sum[1])<<8 | uint32(sum[2])
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
