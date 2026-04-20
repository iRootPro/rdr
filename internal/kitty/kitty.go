package kitty

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// IsSupported reports whether the current terminal speaks the Kitty
// Graphics Protocol. Covers Kitty, Ghostty, and WezTerm.
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

// InsideTmux reports whether we're running under a tmux session. tmux
// strips unknown terminal escape sequences unless `set -g allow-passthrough on`
// is configured, which blocks Kitty Graphics Protocol APCs. Callers can
// use this to surface a hint when images mysteriously fail to render.
func InsideTmux() bool {
	return os.Getenv("TMUX") != "" || os.Getenv("TERM_PROGRAM") == "tmux"
}

const chunkSize = 4096

// Transmit returns the escape sequence(s) that upload the given PNG
// bytes to the terminal under the given ID and create a virtual
// placement of (cols × rows) terminal cells. Large payloads are split
// into 4096-byte base64 chunks. Matches the canonical form used by
// Kitty's own icat kitten with `--unicode-placeholder`.
//
// Callers must pair this with PlaceholderBlock to draw the image.
func Transmit(id uint32, data []byte, cols, rows int) string {
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
			fmt.Fprintf(&b,
				"\x1b_Ga=T,q=2,f=100,U=1,i=%d,c=%d,r=%d,m=%d;%s\x1b\\",
				id, cols, rows, more, encoded[i:end],
			)
		} else {
			fmt.Fprintf(&b, "\x1b_Gm=%d;%s\x1b\\", more, encoded[i:end])
		}
	}
	return b.String()
}

// TransmitOnly uploads the PNG under the given ID without creating any
// placement. Pair with CreateVirtualPlacement + placeholder text (or with
// placeholders alone if a placement was already made). Using a=t (lowercase)
// avoids the a=T+U=1 one-shot form, which has proved fragile inside TUI
// renderers — splitting the upload from the placement lets us emit the
// payload out-of-band (before bubbletea takes over stdout) and keep only
// the cheap placement bytes flowing through View().
func TransmitOnly(id uint32, data []byte) string {
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
			fmt.Fprintf(&b,
				"\x1b_Ga=t,q=2,f=100,i=%d,m=%d;%s\x1b\\",
				id, more, encoded[i:end],
			)
		} else {
			fmt.Fprintf(&b, "\x1b_Gm=%d;%s\x1b\\", more, encoded[i:end])
		}
	}
	return b.String()
}

// CreateVirtualPlacement registers a virtual placement of (cols × rows)
// for an already-transmitted image. The placement is virtual: it occupies
// no cells on its own; Unicode placeholders emitted later reference it by
// image ID (encoded in the FG color) to render the image.
func CreateVirtualPlacement(id uint32, cols, rows int) string {
	return fmt.Sprintf("\x1b_Ga=p,q=2,U=1,i=%d,c=%d,r=%d;\x1b\\", id, cols, rows)
}

// DeletePlacement removes all placements for the given image ID (virtual
// and real). Call on reader exit to avoid "ghost" images on subsequent
// screens. Using d=I restricts deletion to this image's placements only.
func DeletePlacement(id uint32) string {
	return fmt.Sprintf("\x1b_Ga=d,d=I,i=%d;\x1b\\", id)
}

// ─── Marker-based placeholder approach ──────────────────────────────
//
// Background: lipgloss reflow recognises SGR (CSI colour) escapes and
// preserves them across Width/Padding/Border operations, but does NOT
// recognise the APC `\x1b_G...\x1b\\` bytes used by the Kitty Graphics
// Protocol. Embedding an InlinePlacement APC directly inside content
// that later flows through `lipgloss.NewStyle().Width(N).Render(...)`
// results in the APC being mangled (bytes split by injected padding).
//
// Solution: render a "placeholder block" of cols × rows visible-but-
// inert Private Use Area runes into the content. lipgloss measures
// each rune as width 1, so reflow/padding behave correctly. AFTER all
// lipgloss passes are done — i.e. at the very end of the model's
// View() — we scan the rendered frame for our marker runes and swap
// each block's first-row marker for an actual `a=p` APC. By then the
// frame is just bytes flowing to stdout; no layout engine remains to
// mangle the escape.

const (
	// placeholderFillRune fills cells of the placeholder grid. It lives
	// in Supplementary PUA plane 16 (U+100000+), which Nerd Fonts and
	// Material Icons don't cover — so when an image scrolls off-viewport
	// and its cells are briefly exposed, the user sees a neutral .notdef
	// glyph instead of a random icon. BMP PUA (U+E000+) was rejected
	// because Nerd Fonts map most of it to Powerline / Codicons /
	// Fontawesome glyphs, and U+F0000+ is claimed by Material Icons.
	placeholderFillRune = '\U00100000'
	// placeholderMarkerBase + index (0-based) is the rune at the first
	// cell of the first row of each block. Same plane-16 region,
	// offset so markers and fill don't collide when scanning.
	placeholderMarkerBase = '\U00100100'
)

// Placement describes one image ready to be rendered at a placeholder
// slot: its Kitty image ID (pre-uploaded via TransmitOnly) and the size
// it should occupy in terminal cells.
type Placement struct {
	ID   uint32
	Cols int
	Rows int
}

// PlaceholderFill renders a block of cols × rows cells, terminated by
// a trailing newline. The first cell of the first row is a marker rune
// encoding `index` (0-based); the rest are fill runes. Call this for
// each image in article order, passing index 0, 1, 2, ...
//
// The same index must be used when building the []Placement slice
// passed to ReplacePlaceholders so the N-th block is paired with the
// N-th placement.
func PlaceholderFill(index, cols, rows int) string {
	if cols <= 0 || rows <= 0 || index < 0 || index > 255 {
		return ""
	}
	var b strings.Builder
	b.WriteRune(placeholderMarkerBase + rune(index))
	for i := 1; i < cols; i++ {
		b.WriteRune(placeholderFillRune)
	}
	b.WriteByte('\n')
	for r := 1; r < rows; r++ {
		for c := 0; c < cols; c++ {
			b.WriteRune(placeholderFillRune)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// ReplacePlaceholders scans the rendered frame for first-row placeholder
// markers (runes in [U+E800, U+E8FF]) and substitutes each one with an
// inline delete-then-place APC for the corresponding Placement. Each
// line independently carries both `a=d,d=i` (delete prior placements of
// this image) and `a=p` (create new placement at the cursor). The
// subsequent rows of the block are left as fill runes — the image drawn
// by Kitty covers those cells visually.
//
// Leading bytes before the marker on the same line (ANSI SGR colours
// and whitespace introduced by lipgloss padding) are preserved so the
// cursor arrives at the correct column when Kitty parses the APC.
//
// Callers are responsible for a parallel cleanup path: when a marker
// scrolls off-screen its line is no longer emitted by viewport, so the
// delete APC never reaches Kitty through this function. The rdr reader
// writes explicit `DeletePlacement` APCs to os.Stdout from its Update()
// handler on scroll events to clear those orphans — that path bypasses
// bubbletea's frame-diff renderer, which otherwise skips unchanged
// lines and would swallow any prefix-level cleanup we try to emit here.
func ReplacePlaceholders(rendered string, placements []Placement) string {
	if len(placements) == 0 {
		return rendered
	}
	lines := strings.Split(rendered, "\n")
	visible := make([]bool, len(placements))
	for i, line := range lines {
		idx, markerStart, markerEnd, ok := findPlaceholderMarker(line)
		if ok && idx < len(placements) {
			p := placements[idx]
			// Count how many rows of this block are actually present in
			// the rendered output (marker row + consecutive fill-only
			// rows below it). Kitty draws `r` cells downward regardless
			// of what UI lives beneath, so scaling to the pane would
			// distort the image and a full-height placement would spill
			// past the reader pane into the status bar.
			visibleRows := 1
			for j := i + 1; j < len(lines) && visibleRows < p.Rows; j++ {
				if !strings.ContainsRune(lines[j], placeholderFillRune) {
					break
				}
				visibleRows++
			}
			visible[idx] = true
			// Scale cols proportionally when partially clipped so the
			// image preserves aspect ratio instead of distorting. Image
			// shrinks left-aligned; the right portion of each row stays
			// as fill and gets blanked by blankFillRunes. Browser-style
			// crop (Option C) would preserve size but needs cell pixel
			// dimensions — deferred as a separate improvement.
			cols := p.Cols
			if visibleRows < p.Rows && p.Rows > 0 {
				cols = p.Cols * visibleRows / p.Rows
				if cols < 1 {
					cols = 1
				}
			}
			apc := fmt.Sprintf("\x1b_Ga=d,d=i,i=%d,q=2;\x1b\\\x1b_Ga=p,q=2,i=%d,c=%d,r=%d,C=1;\x1b\\", p.ID, p.ID, cols, visibleRows)
			lines[i] = line[:markerStart] + apc + blankFillRunes(line[markerEnd:])
			continue
		}
		// No marker, but the line might be an interior row of a block
		// (all fill runes). Blank those too so scrolled-off blocks
		// don't expose the fill grid.
		lines[i] = blankFillRunes(line)
	}
	// For placements whose markers are NOT visible this frame (marker
	// scrolled off-screen), append a delete APC to the last line. That
	// line's byte content changes whenever the visibility set changes,
	// which forces bubbletea to re-emit it — letting the delete reach
	// Kitty and unstick ghost placements.
	var suffix strings.Builder
	for i, p := range placements {
		if visible[i] {
			continue
		}
		fmt.Fprintf(&suffix, "\x1b_Ga=d,d=i,i=%d,q=2;\x1b\\", p.ID)
	}
	if suffix.Len() > 0 && len(lines) > 0 {
		lines[len(lines)-1] += suffix.String()
	}
	return strings.Join(lines, "\n")
}

// findPlaceholderMarker returns the image index (0-based), byte offsets
// of the marker rune, and whether a marker was found on this line.
// Ignores runes that are not our marker so leading ANSI sequences or
// padding spaces pass through unchanged.
func findPlaceholderMarker(line string) (index, startByte, endByte int, ok bool) {
	for bi, r := range line {
		if r >= placeholderMarkerBase && r < placeholderMarkerBase+256 {
			return int(r - placeholderMarkerBase), bi, bi + len(string(r)), true
		}
	}
	return 0, 0, 0, false
}

// blankFillRunes replaces every fill rune (and any stray marker rune)
// in the line with a single ASCII space. Used at post-process time to
// keep the placeholder grid invisible when the image isn't drawn on
// top of it — otherwise fonts render the plane-16 PUA code points as
// .notdef boxes, flashing a grid while the user scrolls past an image.
func blankFillRunes(line string) string {
	if !strings.ContainsRune(line, placeholderFillRune) && !hasMarkerRune(line) {
		return line
	}
	var b strings.Builder
	b.Grow(len(line))
	for _, r := range line {
		if r == placeholderFillRune || (r >= placeholderMarkerBase && r < placeholderMarkerBase+256) {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func hasMarkerRune(line string) bool {
	for _, r := range line {
		if r >= placeholderMarkerBase && r < placeholderMarkerBase+256 {
			return true
		}
	}
	return false
}

// InlinePlacement returns an APC that creates a real placement of an
// already-transmitted image at the cursor's CURRENT position, sized to
// (cols × rows). C=1 prevents Kitty from moving the cursor; we then
// emit `rows` newlines so the next text starts below the image. This is
// the alternative to virtual placement+Unicode placeholders: the image
// is bound to the current screen buffer, so it survives bubbletea's
// alt-screen (where virtual placements made in main-screen disappear).
//
// Pair with TransmitOnly (image upload) done once per article. Each
// View() emits InlinePlacement(id, c, r) at the spot where each image
// should appear.
func InlinePlacement(id uint32, cols, rows int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\x1b_Ga=p,q=2,i=%d,c=%d,r=%d,C=1;\x1b\\", id, cols, rows)
	for i := 0; i < rows; i++ {
		b.WriteString("\r\n")
	}
	return b.String()
}

// PlaceholderBlock returns a text block of Unicode placeholders with
// the foreground color encoding the low 24 bits of the image ID. The
// block is wrapped with SGR reset so surrounding text is not affected.
// Each cell carries two diacritics (row, col); callers with image IDs
// whose high byte is non-zero also need to emit a third diacritic —
// not supported here because rdr uses small IDs derived from hashes.
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

	r := byte((id >> 16) & 0xff)
	g := byte((id >> 8) & 0xff)
	bv := byte(id & 0xff)
	// Use semicolon-separated SGR so lipgloss/reflow recognize this as
	// a truecolor FG and don't strip or mangle it. The colon-separator
	// form is ISO-valid but not universally parsed.
	colorPrefix := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, bv)

	var b strings.Builder
	for row := 0; row < rows; row++ {
		// Set our FG color at the start of EVERY row. Pane borders
		// (lipgloss) emit their own SGR reset between lines, so we
		// must re-apply the image-id color on each row or subsequent
		// placeholder cells lose their image association.
		b.WriteString(colorPrefix)
		rowRune := diacritics[row]
		for col := 0; col < cols; col++ {
			colRune := diacritics[col]
			b.WriteRune(0x10EEEE)
			b.WriteRune(rowRune)
			b.WriteRune(colRune)
		}
		b.WriteString("\x1b[0m\n")
	}
	return b.String()
}
