# rdr Phase 2 — Step 2: Kitty Graphics (inline images)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Render inline images in the full-article reader using the
Kitty Graphics Protocol + Unicode placeholders. Fallback to `[📷 alt]`
text on non-Kitty terminals. Images cached to disk so repeat opens are
instant.

**Architecture:**
- New `internal/kitty/` package — pure protocol generators + terminal
  detection. No dependency on UI or DB. Unit tests.
- New `internal/feed/images.go` — parallel image downloader with
  filesystem cache under `<home>/cache/images/<sha256>.<ext>`.
- UI integration in `internal/ui/reader.go`:
  - Detect Kitty at startup, store flag on Model.
  - When rendering a cached full article, split the markdown on image
    refs: render text chunks with glamour, replace each image ref with
    a Kitty Unicode placeholder block.
  - Before emitting the placeholder block, transmit the image bytes
    once per session (Kitty dedupes by image ID).
  - Placeholders scroll naturally with the viewport because they're
    regular text cells.

**Key technical facts:**
- Kitty Unicode Placeholder char: `U+10EEEE`. Each cell is followed by
  up to 3 combining diacritics encoding row/col/high-ID-bits.
- Diacritic set: 297 specific Unicode code points. Hardcoded table.
- Transmit escape: `\x1b_Ga=t,f=100,i=<id>,q=2,m=<more>;<base64>\x1b\\`
  chunked to 4096 bytes per chunk (Kitty spec).
- Virtual placement: `\x1b_Ga=p,U=1,i=<id>,c=<cols>,r=<rows>,q=2\x1b\\`
- Image ID: first 4 bytes of SHA256(imageBytes), interpreted as uint32.

**Tech Stack:** stdlib `image`, `image/png`, `image/jpeg`, `image/gif`
for dimensions; `encoding/base64`; `crypto/sha256`; no new deps.

**Working directory:** `/Users/sasha/Code/github.com/iRootPro/rdr`

**Scope:** only Kitty. WezTerm + Ghostty support Kitty protocol so they
work too. iTerm2 has its own protocol — out of scope. Sixel — out of
scope.

---

## Task 1: `internal/kitty/` package — detection + protocol (TDD)

**Files:**
- Create: `internal/kitty/kitty.go`
- Create: `internal/kitty/diacritics.go` (big static table)
- Create: `internal/kitty/kitty_test.go`

**Step 1.1: Create the diacritic table**

The Kitty protocol uses 297 specific Unicode code points for
row/col/id diacritics. Create `internal/kitty/diacritics.go` with a
`var diacritics = []rune{...}` slice containing all 297 entries from
Kitty's rowcolumn-diacritics.txt. The file is just a static slice, no
logic.

Full list (paste verbatim):

```go
package kitty

// diacritics is the ordered set of Unicode combining characters used by
// Kitty's Unicode-placeholder protocol to encode (row, col, id-high-bits)
// per placeholder cell. Index into this slice equals the encoded value.
// Source: https://sw.kovidgoyal.net/kitty/_downloads/f0a0de9ec8d9ff4456206db8e0814937/rowcolumn-diacritics.txt
var diacritics = []rune{
	0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F,
	0x0346, 0x034A, 0x034B, 0x034C, 0x0350, 0x0351, 0x0352, 0x0357,
	0x035B, 0x0363, 0x0364, 0x0365, 0x0366, 0x0367, 0x0368, 0x0369,
	0x036A, 0x036B, 0x036C, 0x036D, 0x036E, 0x036F, 0x0483, 0x0484,
	0x0485, 0x0486, 0x0487, 0x0592, 0x0593, 0x0594, 0x0595, 0x0597,
	0x0598, 0x0599, 0x059C, 0x059D, 0x059E, 0x059F, 0x05A0, 0x05A1,
	0x05A8, 0x05A9, 0x05AB, 0x05AC, 0x05AF, 0x05C4, 0x0610, 0x0611,
	0x0612, 0x0613, 0x0614, 0x0615, 0x0616, 0x0617, 0x0657, 0x0658,
	0x0659, 0x065A, 0x065B, 0x065D, 0x065E, 0x06D6, 0x06D7, 0x06D8,
	0x06D9, 0x06DA, 0x06DB, 0x06DC, 0x06DF, 0x06E0, 0x06E1, 0x06E2,
	0x06E4, 0x06E7, 0x06E8, 0x06EB, 0x06EC, 0x0730, 0x0732, 0x0733,
	0x0735, 0x0736, 0x073A, 0x073D, 0x073F, 0x0740, 0x0741, 0x0743,
	0x0745, 0x0747, 0x0749, 0x074A, 0x07EB, 0x07EC, 0x07ED, 0x07EE,
	0x07EF, 0x07F0, 0x07F1, 0x07F3, 0x0816, 0x0817, 0x0818, 0x0819,
	0x081B, 0x081C, 0x081D, 0x081E, 0x081F, 0x0820, 0x0821, 0x0822,
	0x0823, 0x0825, 0x0826, 0x0827, 0x0829, 0x082A, 0x082B, 0x082C,
	0x082D, 0x0951, 0x0953, 0x0954, 0x0F82, 0x0F83, 0x0F86, 0x0F87,
	0x135D, 0x135E, 0x135F, 0x17DD, 0x193A, 0x1A17, 0x1A75, 0x1A76,
	0x1A77, 0x1A78, 0x1A79, 0x1A7A, 0x1A7B, 0x1A7C, 0x1B6B, 0x1B6D,
	0x1B6E, 0x1B6F, 0x1B70, 0x1B71, 0x1B72, 0x1B73, 0x1CD0, 0x1CD1,
	0x1CD2, 0x1CDA, 0x1CDB, 0x1CE0, 0x1DC0, 0x1DC1, 0x1DC3, 0x1DC4,
	0x1DC5, 0x1DC6, 0x1DC7, 0x1DC8, 0x1DC9, 0x1DCB, 0x1DCC, 0x1DD1,
	0x1DD2, 0x1DD3, 0x1DD4, 0x1DD5, 0x1DD6, 0x1DD7, 0x1DD8, 0x1DD9,
	0x1DDA, 0x1DDB, 0x1DDC, 0x1DDD, 0x1DDE, 0x1DDF, 0x1DE0, 0x1DE1,
	0x1DE2, 0x1DE3, 0x1DE4, 0x1DE5, 0x1DE6, 0x1DFE, 0x20D0, 0x20D1,
	0x20D4, 0x20D5, 0x20D6, 0x20D7, 0x20DB, 0x20DC, 0x20E1, 0x20E7,
	0x20E9, 0x20F0, 0x2CEF, 0x2CF0, 0x2CF1, 0x2DE0, 0x2DE1, 0x2DE2,
	0x2DE3, 0x2DE4, 0x2DE5, 0x2DE6, 0x2DE7, 0x2DE8, 0x2DE9, 0x2DEA,
	0x2DEB, 0x2DEC, 0x2DED, 0x2DEE, 0x2DEF, 0x2DF0, 0x2DF1, 0x2DF2,
	0x2DF3, 0x2DF4, 0x2DF5, 0x2DF6, 0x2DF7, 0x2DF8, 0x2DF9, 0x2DFA,
	0x2DFB, 0x2DFC, 0x2DFD, 0x2DFE, 0x2DFF, 0xA66F, 0xA67C, 0xA67D,
	0xA6F0, 0xA6F1, 0xA8E0, 0xA8E1, 0xA8E2, 0xA8E3, 0xA8E4, 0xA8E5,
	0xA8E6, 0xA8E7, 0xA8E8, 0xA8E9, 0xA8EA, 0xA8EB, 0xA8EC, 0xA8ED,
	0xA8EE, 0xA8EF, 0xA8F0, 0xA8F1, 0xAAB0, 0xAAB2, 0xAAB3, 0xAAB7,
	0xAAB8, 0xAABE, 0xAABF, 0xAAC1, 0xFE20, 0xFE21, 0xFE22, 0xFE23,
	0xFE24, 0xFE25, 0xFE26, 0x10A0F, 0x10A38, 0x1D185, 0x1D186, 0x1D187,
	0x1D188, 0x1D189, 0x1D1AA, 0x1D1AB, 0x1D1AC, 0x1D1AD, 0x1D242, 0x1D243,
	0x1D244,
}
```

**Step 1.2: Write the failing tests**

Create `internal/kitty/kitty_test.go`:

```go
package kitty

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestIsSupported_DetectsEnvVars(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"kitty TERM", map[string]string{"TERM": "xterm-kitty"}, true},
		{"KITTY_WINDOW_ID set", map[string]string{"KITTY_WINDOW_ID": "1"}, true},
		{"Ghostty", map[string]string{"TERM_PROGRAM": "ghostty"}, true},
		{"WezTerm", map[string]string{"TERM_PROGRAM": "WezTerm"}, true},
		{"plain xterm", map[string]string{"TERM": "xterm-256color"}, false},
		{"empty", map[string]string{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear the relevant vars first so each sub-test is isolated.
			t.Setenv("TERM", "")
			t.Setenv("KITTY_WINDOW_ID", "")
			t.Setenv("TERM_PROGRAM", "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if got := IsSupported(); got != tc.want {
				t.Fatalf("IsSupported: got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTransmit_SingleChunk_EmitsEscapeSequence(t *testing.T) {
	data := []byte("hello")
	out := Transmit(42, data)
	if !strings.HasPrefix(out, "\x1b_G") {
		t.Fatalf("missing prefix: %q", out)
	}
	if !strings.HasSuffix(out, "\x1b\\") {
		t.Fatalf("missing suffix: %q", out)
	}
	if !strings.Contains(out, "i=42") {
		t.Fatalf("missing id: %q", out)
	}
	if !strings.Contains(out, "f=100") {
		t.Fatalf("missing format: %q", out)
	}
	expectedB64 := base64.StdEncoding.EncodeToString(data)
	if !strings.Contains(out, expectedB64) {
		t.Fatalf("missing base64 payload: %q", out)
	}
}

func TestTransmit_LargeData_ChunksAt4096(t *testing.T) {
	// 10_000 bytes of base64 → ~13_336 characters → 4 chunks.
	data := make([]byte, 10_000)
	for i := range data {
		data[i] = byte(i)
	}
	out := Transmit(1, data)
	// Count escape sequence envelopes.
	envelopes := strings.Count(out, "\x1b\\")
	if envelopes < 4 {
		t.Fatalf("expected ≥4 chunks, got %d envelopes", envelopes)
	}
	// First chunk carries m=1, middle chunks m=1, last chunk m=0.
	if !strings.Contains(out, "m=0") {
		t.Fatalf("no terminating chunk with m=0: %q", out[:200])
	}
}

func TestPlaceholderBlock_DimensionsAndEncoding(t *testing.T) {
	block := PlaceholderBlock(1, 3, 2) // 3 cols × 2 rows
	lines := strings.Split(strings.TrimRight(block, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 rows, got %d: %q", len(lines), block)
	}
	for _, line := range lines {
		runes := []rune(line)
		// Each cell is placeholder + 3 combining marks = 4 runes.
		if len(runes) != 3*4 {
			t.Fatalf("row has %d runes, want %d: %q", len(runes), 3*4, line)
		}
		// Every 4th rune from index 0 must be the placeholder.
		for i := 0; i < len(runes); i += 4 {
			if runes[i] != 0x10EEEE {
				t.Fatalf("cell %d: got %U, want U+10EEEE", i/4, runes[i])
			}
		}
	}
}

func TestPlacement_EmitsVirtualPlacementEscape(t *testing.T) {
	out := Placement(7, 10, 5)
	if !strings.HasPrefix(out, "\x1b_G") || !strings.HasSuffix(out, "\x1b\\") {
		t.Fatalf("envelope: %q", out)
	}
	for _, want := range []string{"a=p", "U=1", "i=7", "c=10", "r=5", "q=2"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}
```

**Step 1.3: Run — expect failure**

```bash
go test ./internal/kitty/... -v
```

Expected: compile errors — `IsSupported`, `Transmit`,
`PlaceholderBlock`, `Placement` undefined.

**Step 1.4: Implement `kitty.go`**

Create `internal/kitty/kitty.go`:

```go
package kitty

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// IsSupported reports whether the current terminal supports the Kitty
// Graphics Protocol. Checks the three common hosts: Kitty itself,
// Ghostty, and WezTerm.
func IsSupported() bool {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	if os.Getenv("TERM") == "xterm-kitty" {
		return true
	}
	switch os.Getenv("TERM_PROGRAM") {
	case "ghostty", "WezTerm":
		return true
	}
	return false
}

const chunkSize = 4096 // Kitty spec max per escape envelope

// Transmit returns the escape sequence(s) that upload the given image
// bytes to the terminal under the given ID. Large payloads are split
// into 4096-byte base64 chunks; each chunk is wrapped in its own
// Kitty escape envelope with m=1 for all-but-last and m=0 for last.
func Transmit(id uint32, data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	var b strings.Builder
	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		more := 1
		if end == len(encoded) {
			more = 0
		}
		if i == 0 {
			fmt.Fprintf(&b, "\x1b_Ga=t,f=100,i=%d,q=2,m=%d;%s\x1b\\", id, more, encoded[i:end])
		} else {
			fmt.Fprintf(&b, "\x1b_Gm=%d;%s\x1b\\", more, encoded[i:end])
		}
	}
	return b.String()
}

// Placement returns the virtual-placement escape sequence for an
// already-transmitted image ID, declaring its size in cells.
func Placement(id uint32, cols, rows int) string {
	return fmt.Sprintf("\x1b_Ga=p,U=1,i=%d,c=%d,r=%d,q=2\x1b\\", id, cols, rows)
}

// PlaceholderBlock returns a text block of Unicode placeholders that
// Kitty replaces with the image. Every cell carries three diacritic
// marks: row, column, and image-ID high byte.
func PlaceholderBlock(id uint32, cols, rows int) string {
	if cols <= 0 || rows <= 0 {
		return ""
	}
	if cols > len(diacritics) {
		cols = len(diacritics)
	}
	if rows > len(diacritics) {
		rows = len(diacritics)
	}
	idHigh := int(id >> 24)
	if idHigh >= len(diacritics) {
		idHigh = 0
	}
	idHighRune := diacritics[idHigh]

	var b strings.Builder
	for r := 0; r < rows; r++ {
		rowRune := diacritics[r]
		for c := 0; c < cols; c++ {
			colRune := diacritics[c]
			b.WriteRune(0x10EEEE)
			b.WriteRune(rowRune)
			b.WriteRune(colRune)
			b.WriteRune(idHighRune)
		}
		b.WriteRune('\n')
	}
	return b.String()
}
```

**Step 1.5: Run — expect pass**

```bash
go test ./internal/kitty/... -v
```

Expected: all 4 tests PASS.

**Step 1.6: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/kitty/
git commit -m "feat(rdr): add kitty package with protocol primitives"
```

---

## Task 2: Image downloader + filesystem cache

**Files:**
- Create: `internal/feed/images.go`
- Create: `internal/feed/images_test.go`

**Step 2.1: Write the failing test**

Create `rdr/internal/feed/images_test.go`:

```go
package feed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadImages_CachesAndDedupes(t *testing.T) {
	// PNG magic bytes so image.DecodeConfig works later.
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
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
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

	// Second call with the same URLs must skip network — hit count unchanged.
	before := calls
	if _, err := DownloadImages(context.Background(), urls, cacheDir); err != nil {
		t.Fatalf("DownloadImages (second): %v", err)
	}
	if calls != before {
		t.Fatalf("expected cache hit, calls went %d -> %d", before, calls)
	}
}
```

**Step 2.2: Run — expect failure**

```bash
go test ./internal/feed/... -run TestDownloadImages -v
```

Expected: `undefined: DownloadImages`.

**Step 2.3: Implement `images.go`**

Create `rdr/internal/feed/images.go`:

```go
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
	"time"

	"golang.org/x/sync/errgroup"
)

const imageFetchConcurrency = 4

// DownloadImages fetches all URLs into cacheDir, returning a url→path
// map. Files are named after the SHA256 of the URL so the same URL
// resolves to the same path across runs. Already-cached URLs are not
// re-downloaded.
func DownloadImages(ctx context.Context, urls []string, cacheDir string) (map[string]string, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir cache: %w", err)
	}

	paths := make(map[string]string, len(urls))
	var mu syncMutex
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

			dest := filepath.Join(cacheDir, fileName(u))
			if _, err := os.Stat(dest); err == nil {
				mu.Lock()
				paths[u] = dest
				mu.Unlock()
				return nil
			}

			req, err := http.NewRequestWithContext(gctx, http.MethodGet, u, nil)
			if err != nil {
				return nil // skip bad URLs silently
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
			if copyErr != nil {
				_ = os.Remove(tmp.Name())
				return nil
			}
			if closeErr != nil {
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

func fileName(u string) string {
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

// syncMutex avoids an extra import; sync.Mutex works too.
type syncMutex struct{ m sync.Mutex }

func (s *syncMutex) Lock()   { s.m.Lock() }
func (s *syncMutex) Unlock() { s.m.Unlock() }
```

Add `"sync"` to the imports (the helper type wraps `sync.Mutex`).

Actually, simpler: use `sync.Mutex` directly. Replace the `syncMutex`
struct with just `var mu sync.Mutex`.

**Step 2.4: Run — expect pass**

```bash
go test ./internal/feed/... -run TestDownloadImages -v
```

Expected: PASS. Second-call cache hit verified.

**Step 2.5: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/feed/images.go rdr/internal/feed/images_test.go
git commit -m "feat(rdr): add parallel image downloader with fs cache"
```

---

## Task 3: UI integration — render images inline in reader

**Files:**
- Modify: `internal/ui/reader.go`
- Modify: `internal/ui/model.go`

**Step 3.1: Extend Model with image support**

Add field to `Model`:

```go
kittyOn    bool
imageCache string // absolute path to <home>/cache/images
```

Initialize in `New` — but `New` doesn't know `home`. Thread it through
`New(database, fetcher, home)` and update `main.go` to pass the home
directory.

Set `kittyOn = kitty.IsSupported()` (import `github.com/iRootPro/rdr/internal/kitty`).

**Step 3.2: Rebuild reader content with image handling**

In `reader.go`, add a new function that takes the cached markdown and
returns a rendered ANSI string with image placeholder blocks inserted:

```go
import (
	// ... existing ...
	"crypto/sha256"
	"encoding/binary"
	"os"
	"regexp"

	"github.com/iRootPro/rdr/internal/kitty"
)

var reImageRef = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

type imageSpec struct {
	alt  string
	url  string
	path string // local cached path
}

// extractImages returns all image refs in the markdown in document order,
// together with the markdown split into text chunks between them.
func extractImages(md string) (chunks []string, images []imageSpec) {
	last := 0
	for _, match := range reImageRef.FindAllStringSubmatchIndex(md, -1) {
		chunks = append(chunks, md[last:match[0]])
		images = append(images, imageSpec{
			alt: md[match[2]:match[3]],
			url: md[match[4]:match[5]],
		})
		last = match[1]
	}
	chunks = append(chunks, md[last:])
	return chunks, images
}

// imageID returns a stable uint32 derived from the image URL.
func imageID(url string) uint32 {
	sum := sha256.Sum256([]byte(url))
	return binary.BigEndian.Uint32(sum[:4])
}
```

Update `buildReaderContent` — when `cachedBody != ""` AND `kittyOn`:

```go
if a.CachedBody != "" {
	if kittyOn && imageCache != "" {
		rendered := renderWithKittyImages(a.CachedBody, width, imageCache)
		b.WriteString(rendered)
	} else if rendered, err := renderMarkdown(a.CachedBody, width); err == nil {
		b.WriteString(rendered)
	} else {
		b.WriteString(readerBody.Render(wrap(stripHTML(a.CachedBody), width)))
	}
}
```

`buildReaderContent` now needs two extra args (`kittyOn bool`,
`imageCache string`). Update the call site in `model.go` (openReader,
WindowSizeMsg handler, fullArticleLoadedMsg handler).

Implement `renderWithKittyImages`:

```go
func renderWithKittyImages(md string, width int, imageCache string) string {
	chunks, images := extractImages(md)

	// Read image bytes + dimensions up-front so every image gets a
	// transmit block plus a placeholder block inline.
	var out strings.Builder
	for i, chunk := range chunks {
		if chunk != "" {
			rendered, err := renderMarkdown(chunk, width)
			if err != nil {
				out.WriteString(chunk)
			} else {
				out.WriteString(rendered)
			}
		}
		if i < len(images) {
			spec := images[i]
			path := filepath.Join(imageCache, feed.ImageFileName(spec.url))
			data, err := os.ReadFile(path)
			if err != nil {
				// Fallback: text tag.
				out.WriteString("\n")
				out.WriteString(readerHint.Render("[📷 " + spec.alt + "]"))
				out.WriteString("\n")
				continue
			}
			id := imageID(spec.url)
			cols, rows := imageCells(data, width)
			out.WriteString(kitty.Transmit(id, data))
			out.WriteString(kitty.Placement(id, cols, rows))
			out.WriteString("\n")
			out.WriteString(kitty.PlaceholderBlock(id, cols, rows))
			out.WriteString("\n")
		}
	}
	return out.String()
}

func imageCells(data []byte, termWidth int) (cols, rows int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	maxCols := termWidth - 4
	if maxCols < 10 {
		maxCols = 10
	}
	if err != nil {
		return maxCols, 10
	}
	cols = maxCols
	// Cells are ~2x taller than wide, so rows = cols * h/w / 2.
	rows = cols * cfg.Height / cfg.Width / 2
	if rows < 1 {
		rows = 1
	}
	if rows > 30 {
		rows = 30
	}
	return cols, rows
}
```

Imports to add: `bytes`, `image`, `image/jpeg`, `image/png`, `image/gif`
(the last three for side-effect registration), `path/filepath`, and
the `feed` package if I use `feed.ImageFileName`.

Actually — `fileName` in `images.go` is lowercase. Export it as
`ImageFileName`. Update `images.go`:

```go
func ImageFileName(u string) string { ... }
// Change the in-package call site in DownloadImages accordingly.
```

**Step 3.3: Kick off image downloads when full article loads**

In `fetchFullCmd`, after `CacheArticle` succeeds, parse image refs and
call `DownloadImages`. Pass the cache dir through.

Easier shape: add a new command `fetchImagesCmd(urls, cacheDir)` that
returns a new `imagesReadyMsg{}`. Fire it as part of the Batch after
`fullArticleLoadedMsg` arrives (second pass).

To keep Task 3 shippable, take the simplest path:

1. In `fetchFullCmd`, after CacheArticle, synchronously run
   `feed.DownloadImages(ctx, urls, cacheDir)` — one more network
   roundtrip before returning `fullArticleLoadedMsg`. Pass
   `cacheDir` into the command closure.
2. User sees spinner longer, then the final render has images.

```go
func fetchFullCmd(f *feed.Fetcher, d *db.DB, articleID int64, url, cacheDir string) tea.Cmd {
	return func() tea.Msg {
		md, err := f.FetchFull(context.Background(), url)
		if err != nil {
			return errMsg{err}
		}
		if err := d.CacheArticle(articleID, md); err != nil {
			return errMsg{err}
		}
		// Parallel image download — best-effort, errors ignored.
		_, _, urls := parseImageRefs(md) // or use reImageRef directly
		if len(urls) > 0 {
			_, _ = feed.DownloadImages(context.Background(), urls, cacheDir)
		}
		return fullArticleLoadedMsg{articleID: articleID, markdown: md}
	}
}
```

For `parseImageRefs`, reuse the `reImageRef` regex or duplicate a tiny
helper in model.go. The helper in reader.go is already enough — call
it from there.

**Step 3.4: Thread `home` through Model**

In `main.go`:

```go
program := tea.NewProgram(ui.New(database, fetcher, home), tea.WithAltScreen())
```

In `ui.New`:

```go
func New(database *db.DB, fetcher *feed.Fetcher, home string) Model {
	// ... existing ...
	return Model{
		// ... existing ...
		kittyOn:    kitty.IsSupported(),
		imageCache: filepath.Join(home, "cache", "images"),
	}
}
```

Add `"path/filepath"` to `model.go` imports.

**Step 3.5: Build**

```bash
go build ./...
```

Fix compile errors as they appear. Expected hot spots:
- `renderWithKittyImages` not seeing `feed.ImageFileName` → export it.
- `image.DecodeConfig` needs the codec imports with blank `_`.
- `fetchFullCmd` needs the new `cacheDir` arg — update the call site
  in `Update`.

**Step 3.6: Smoke test**

```bash
rm -rf dev && mkdir dev && cp config.yaml dev/config.yaml
RDR_HOME=./dev go run .
```

In a Kitty / Ghostty / WezTerm terminal:
1. Let the fetch complete
2. Pick an article that has embedded images (e.g. a Lobsters link with
   a blog post, or Go Blog posts often have them)
3. Press `enter` to open the reader
4. Press `f` to fetch the full article — spinner runs longer as images
   download
5. Image should appear inline in the reader body
6. `j`/`k` scrolls — image scrolls with the text (Unicode placeholders
   are text cells)
7. `esc` back, reopen — image is instant (cache hit)

If the terminal doesn't support Kitty (plain `xterm-256color`):
- No placeholder block, just `[📷 alt]` text fallback

**Step 3.7: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/ui/reader.go rdr/internal/ui/model.go \
           rdr/main.go rdr/internal/feed/images.go
git commit -m "feat(rdr): render inline kitty images in the full-article reader"
```

---

## Known limitations to document after Step 3

1. **Scroll flicker** — depending on Kitty version, placeholders can
   flicker on fast scroll. No fix in Phase 2.
2. **Large images** — `imageCells` caps rows at 30. Taller images get
   cropped.
3. **Aspect ratio** — cell ratio is estimated as 1:2. On terminals with
   different fonts it may look stretched.
4. **Non-supported image formats** — JPEG 2000, AVIF, etc. Falls back
   to `[📷 alt]`.
5. **First render latency** — `f` now also downloads images
   synchronously. For image-heavy articles this doubles the wait. Can
   be async in a follow-up with a second `imagesReadyMsg` pass.
6. **Cache growth** — no eviction policy. Lives in `<home>/cache/images`.
   Use the filesystem.

Phase 2 Step 3 (future): async image download + cache eviction + inline
image zoom (`i` to open in external viewer).
