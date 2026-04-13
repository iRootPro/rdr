package feed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestDownloadImages_CachesAndDedupes(t *testing.T) {
	pngBytes := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
		0x54, 0x08, 0x99, 0x63, 0xf8, 0x0f, 0x00, 0x00,
		0x01, 0x01, 0x00, 0x01, 0x5a, 0x6d, 0x22, 0x85,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44,
		0xae, 0x42, 0x60, 0x82,
	}
	var calls int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&calls, 1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()

	cacheDir := filepath.Join(t.TempDir(), "images")
	urls := []string{srv.URL + "/a.png", srv.URL + "/b.png"}

	paths, err := DownloadImages(context.Background(), urls, cacheDir)
	if err != nil {
		t.Fatalf("DownloadImages: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("want 2 paths, got %d: %v", len(paths), paths)
	}
	for u, p := range paths {
		if p == "" {
			t.Fatalf("empty path for %s", u)
		}
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("file missing at %s: %v", p, err)
		}
	}

	before := atomic.LoadInt64(&calls)
	if _, err := DownloadImages(context.Background(), urls, cacheDir); err != nil {
		t.Fatalf("DownloadImages (second): %v", err)
	}
	if atomic.LoadInt64(&calls) != before {
		t.Fatalf("expected cache hit, calls went %d -> %d", before, atomic.LoadInt64(&calls))
	}
}
