package feed

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestToPNG_PassesThroughPNG(t *testing.T) {
	// Valid 1×1 PNG header + data.
	original := []byte{
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
	out, size, err := ToPNG(original)
	if err != nil {
		t.Fatalf("ToPNG on PNG: %v", err)
	}
	if !bytes.Equal(out, original) {
		t.Fatal("PNG should pass through unchanged")
	}
	if size.X != 1 || size.Y != 1 {
		t.Fatalf("size: got %v, want 1×1", size)
	}
}

func TestToPNG_ConvertsJPEG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, color.RGBA{200, 100, 50, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	out, size, err := ToPNG(buf.Bytes())
	if err != nil {
		t.Fatalf("ToPNG on JPEG: %v", err)
	}
	if len(out) < 8 || !bytes.Equal(out[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) {
		t.Fatal("output missing PNG signature")
	}
	if size.X != 4 || size.Y != 3 {
		t.Fatalf("size: got %v, want 4×3", size)
	}
	// Verify the output PNG decodes cleanly.
	if _, err := png.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("output PNG re-decode: %v", err)
	}
}

func TestExtractImageURLs_DedupesAndSkipsNonHTTP(t *testing.T) {
	md := `Some text with ![first](https://example.com/a.png) and
more text ![second](https://example.com/b.jpg "title here").
Then ![dup](https://example.com/a.png) repeated.
And ![local](file:///etc/hosts) that we should skip.
And ![relative](/assets/logo.png) — also skipped.`
	urls := ExtractImageURLs(md)
	want := []string{"https://example.com/a.png", "https://example.com/b.jpg"}
	if len(urls) != len(want) {
		t.Fatalf("got %d URLs, want %d: %v", len(urls), len(want), urls)
	}
	for i, u := range want {
		if urls[i] != u {
			t.Fatalf("URL[%d]: got %q, want %q", i, urls[i], u)
		}
	}
}

func TestImageID_DeterministicFromURL(t *testing.T) {
	a1 := ImageID("https://example.com/x.png")
	a2 := ImageID("https://example.com/x.png")
	b := ImageID("https://example.com/y.png")
	if a1 != a2 {
		t.Fatal("same URL should produce same ID")
	}
	if a1 == b {
		t.Fatalf("different URLs should produce different IDs (got same: %d)", a1)
	}
	if a1>>24 != 0 {
		t.Fatalf("ID should fit in 24 bits, got 0x%x", a1)
	}
}

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
