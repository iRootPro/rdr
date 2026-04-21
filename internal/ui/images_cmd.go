package ui

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iRootPro/rdr/internal/feed"
	"github.com/iRootPro/rdr/internal/kitty"
	"github.com/iRootPro/rdr/internal/rlog"
)

// cellAspect is the height-to-width ratio of a single terminal cell in
// pixels. Kitty scales inline images to fit the c×r cell box without
// preserving aspect, so this value is what keeps visual proportions
// correct. Populated from the terminal's own `CSI 16 t` reply at
// startup via SetCellPixelSize; falls back to 2.1 (a decent average
// across common monospace fonts) when the terminal didn't answer. Also
// honours RDR_CELL_ASPECT for manual override when the query reports
// wrong values (e.g. some emulators return logical vs physical pixels).
var cellAspect = func() float64 {
	if v := os.Getenv("RDR_CELL_ASPECT"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0.5 && f < 5 {
			return f
		}
	}
	return 2.1
}()

// SetCellPixelSize records the measured cell size (pixels) so cellSize
// can compute image geometry from the terminal's actual font metrics.
// No-op when either dimension is non-positive so callers can pass
// through a failed query without guarding.
func SetCellPixelSize(widthPx, heightPx int) {
	if widthPx <= 0 || heightPx <= 0 {
		return
	}
	// RDR_CELL_ASPECT still wins if explicitly set, so we only update
	// from the query when there's no user override.
	if os.Getenv("RDR_CELL_ASPECT") != "" {
		return
	}
	cellAspect = float64(heightPx) / float64(widthPx)
}

// maxImgCols caps the horizontal cell size of any single inline image.
// Very wide images would otherwise eat the whole reader column; 60 cells
// is a readable portrait size while still leaving room for the indent.
const maxImgCols = 60

// maxImgRows caps vertical cell size so tall images don't push the rest
// of the article off-screen.
const maxImgRows = 30

// maxImagesPerArticle bounds how many inline images one article can
// surface. Articles with long photo galleries would otherwise blow up
// memory (Kitty holds every PNG), inflate render time, and exhaust the
// 256-marker index space in the placeholder scheme. Extra images fall
// through to their original markdown link text.
const maxImagesPerArticle = 20

// prepareImagesCmd downloads every image URL appearing in `md`, converts
// each to PNG, uploads it to the terminal via kitty.TransmitOnly, and
// returns an imagesReadyMsg pairing each URL with a computed Placement.
// Runs the whole pipeline off the bubbletea main goroutine. Writes to
// os.Stdout from inside the cmd goroutine: Kitty's graphics protocol
// reassembles multi-chunk transmits by image ID, so occasional byte
// interleaving with the bubbletea renderer is tolerated.
//
// `contentW` is the max content width in cells; individual image cols
// are capped at both maxImgCols and contentW.
func prepareImagesCmd(articleID int64, md string, home string, contentW int) tea.Cmd {
	return func() tea.Msg {
		urls := feed.ExtractImageURLs(md)
		if len(urls) == 0 {
			return imagesReadyMsg{articleID: articleID}
		}
		if len(urls) > maxImagesPerArticle {
			urls = urls[:maxImagesPerArticle]
		}

		cacheDir := filepath.Join(home, "cache", "images")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		paths, err := feed.DownloadImages(ctx, urls, cacheDir)
		if err != nil {
			rlog.Logf("images", " download error: %v", err)
			// Fall through — partial results are still useful.
		}

		out := imagesReadyMsg{articleID: articleID}
		for _, u := range urls {
			path, ok := paths[u]
			if !ok {
				continue
			}
			raw, rerr := os.ReadFile(path)
			if rerr != nil {
				rlog.Logf("images", " read %s: %v", path, rerr)
				continue
			}
			pngBytes, size, cerr := feed.ToPNG(raw)
			if cerr != nil {
				rlog.Logf("images", " decode %s: %v", u, cerr)
				continue
			}
			cols, rows := cellSize(size.X, size.Y, contentW)
			if cols <= 0 || rows <= 0 {
				continue
			}
			id := feed.ImageID(u)
			if _, werr := os.Stdout.WriteString(kitty.TransmitOnly(id, pngBytes)); werr != nil {
				rlog.Logf("images", " transmit %s: %v", u, werr)
				continue
			}
			out.urls = append(out.urls, u)
			out.placements = append(out.placements, kitty.Placement{ID: id, Cols: cols, Rows: rows})
		}
		return out
	}
}

// cellSize translates a pixel-sized image into a cell-sized Kitty
// placement. Assumes a ~1:2.1 cell aspect ratio (width:height in pixels)
// — a slightly better fit than plain 2:1 across common monospace fonts
// (FiraCode, JetBrains Mono, Iosevka at default sizes). Kitty stretches
// the image to fill `c × r` with no letterboxing, so picking the right
// ratio is what keeps the aspect visually correct. For pixel-perfect
// fidelity we'd query the terminal for cell dimensions, but that's a
// bigger plumbing change; this gets us within a couple of percent.
func cellSize(imgW, imgH, contentW int) (cols, rows int) {
	if imgW <= 0 || imgH <= 0 {
		return 0, 0
	}
	cols = maxImgCols
	if contentW > 0 && contentW-2 < cols {
		cols = contentW - 2
	}
	if cols < 10 {
		cols = 10
	}
	// rows = cols * (imgH/imgW) / cellAspect, rounded to nearest so
	// either direction of integer truncation error is equally likely,
	// keeping aspect closest to the source image. Ceiling would bias
	// toward narrower/taller displays; flooring would bias the other
	// way. Nearest minimises the average distortion.
	exact := float64(cols) * float64(imgH) / (float64(imgW) * cellAspect)
	rows = int(exact + 0.5)
	if rows < 3 {
		rows = 3
	}
	if rows > maxImgRows {
		rows = maxImgRows
	}
	return cols, rows
}

// maybePrepareImagesCmd returns a tea.Cmd that kicks off image download
// + transmit for the current reader article, or nil when there's
// nothing to do (no article, no body, non-Kitty terminal, or the
// showImages flag is off). Call this after any state change that brings
// a new article body into view (full-article load, article switch in
// reader, AI translation result).
func (m Model) maybePrepareImagesCmd() tea.Cmd {
	if !m.showImages || !kitty.IsSupported() {
		return nil
	}
	if m.readerArt == nil {
		return nil
	}
	body := m.readerArt.CachedBody
	if body == "" {
		return nil
	}
	return prepareImagesCmd(m.readerArt.ID, body, m.home, m.reader.Width)
}

// deletePlacements emits Kitty "delete placement by image id" APCs for
// the given placements, so the images don't linger as ghosts on
// subsequent screens (e.g. after exiting reader back into the article
// list). Writes directly to os.Stdout — safe because placement deletion
// is a single short APC per image, atomic against the renderer.
func deletePlacements(placements []kitty.Placement) {
	for _, p := range placements {
		_, _ = os.Stdout.WriteString(kitty.DeletePlacement(p.ID))
	}
}
