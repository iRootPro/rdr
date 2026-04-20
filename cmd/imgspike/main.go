// imgspike is a diagnostic harness for inline image rendering inside a
// bubbletea TUI. Earlier attempts to show Kitty Graphics images in the
// rdr Reader failed after six iterations (reverted in commit db0034a)
// with the byte dump looking correct but no image appearing on screen.
// Rather than re-attempt inside the full reader, this tool isolates the
// problem: we build the minimal possible bubbletea program that tries
// to render one Kitty image, then grow it one layer at a time until it
// breaks, so we can pinpoint which layer (alt-screen / viewport /
// lipgloss) is the actual blocker.
//
// Run:
//
//	go run ./cmd/imgspike -img path/to/picture.jpg -mode 1a
//	go run ./cmd/imgspike -img path/to/picture.jpg -mode 1b
//	go run ./cmd/imgspike -img path/to/picture.jpg -mode 1c
//
// Exit with q or ctrl+c. If the image is visible above "---bottom---"
// the mode works. Set RDR_DUMP_VIEW=<file> to also capture every View()
// output to disk for byte-level inspection.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iRootPro/rdr/internal/kitty"
)

var (
	mode       = flag.String("mode", "1a", "1a | 1b | 1c | 1d | 1e | 1f")
	imgPath    = flag.String("img", "", "path to image file (png/jpeg/gif)")
	cols       = flag.Int("cols", 40, "image width in terminal cells")
	rows       = flag.Int("rows", 15, "image height in terminal cells")
	width  = flag.Int("width", 80, "outer width for 1c lipgloss frame")
	height = flag.Int("height", 24, "outer height for 1b/1c viewport")
	noAlt  = flag.Bool("no-alt", false, "disable bubbletea alt-screen (isolates alt-screen vs image-placement issues)")
	after  = flag.Bool("after", false, "emit transmit+placement AFTER bubbletea starts (via Init cmd), not before")
	verbose = flag.Bool("v", false, "print escape sequences to stderr for byte-level inspection")
)

// imageID used throughout. Pick a value with all three low bytes non-zero
// so the placeholder's FG color encodes a visible colour too (helpful when
// Kitty is NOT interpreting the placeholder — you'll see a coloured block
// instead of nothing, which is a good failure signal).
const imageID = uint32(0xAABBCC)

func main() {
	flag.Parse()
	if !kitty.IsSupported() {
		fmt.Fprintln(os.Stderr, "warning: Kitty Graphics not detected via env; image likely will not render")
	}
	var pngData []byte
	var err error
	if *imgPath == "" {
		// Synthesise a recognisable test image — coloured horizontal
		// bands. If you see a striped block, the protocol works; if you
		// see a flat block of a single colour, the placeholder FG colour
		// is bleeding through (image not actually rendered).
		pngData, err = syntheticPNG(320, 200)
		if err != nil {
			die(fmt.Sprintf("synth PNG: %v", err))
		}
	} else {
		data, rerr := os.ReadFile(*imgPath)
		if rerr != nil {
			die(fmt.Sprintf("read image: %v", rerr))
		}
		pngData, err = toPNG(data)
		if err != nil {
			die(fmt.Sprintf("decode/encode PNG: %v", err))
		}
	}

	transmit := kitty.TransmitOnly(imageID, pngData)
	placement := kitty.CreateVirtualPlacement(imageID, *cols, *rows)
	if *verbose {
		fmt.Fprintf(os.Stderr, "transmit: %d bytes, placement: %d bytes\n", len(transmit), len(placement))
	}

	// -after=false (default): write transmit+placement to stdout BEFORE
	// bubbletea starts, so the large APC payload never passes through
	// the renderer. -after=true: defer emission to an Init() cmd — used
	// to test whether post-alt-screen placement is what Kitty needs.
	if !*after {
		if _, err := os.Stdout.WriteString(transmit); err != nil {
			die(fmt.Sprintf("transmit: %v", err))
		}
		if _, err := os.Stdout.WriteString(placement); err != nil {
			die(fmt.Sprintf("placement: %v", err))
		}
	}

	var m tea.Model
	switch *mode {
	case "1a":
		m = minimalModel{id: imageID, cols: *cols, rows: *rows, upload: upload{transmit: transmit, placement: placement, after: *after}}
	case "1b":
		m = newViewportModel(imageID, *cols, *rows, *width, *height, false, upload{transmit: transmit, placement: placement, after: *after})
	case "1c":
		m = newViewportModel(imageID, *cols, *rows, *width, *height, true, upload{transmit: transmit, placement: placement, after: *after})
	case "1d":
		// Real inline placement test: transmit once now, emit a=p in View().
		// We must NOT have pre-emitted a virtual placement above, because
		// 1d uses real placements only. Override that: re-transmit (we already
		// did it above before this switch — that's fine) but skip the virtual.
		// Simpler: just rely on the fact that pre-emit virtual placement is
		// harmless; mode 1d just won't reference it.
		m = newInlineModel(imageID, *cols, *rows, *width, *height)
	case "1e":
		// 1d + viewport. Tests whether real-placement inline survives
		// scrolling through a viewport. Expected: previous frame's
		// placement may "ghost" since cursor position changes per frame.
		m = newInlineViewportModel(imageID, *cols, *rows, *width, *height, false)
	case "1f":
		// 1d + viewport + lipgloss border. Full reader stack.
		m = newInlineViewportModel(imageID, *cols, *rows, *width, *height, true)
	default:
		die("unknown -mode: " + *mode)
	}

	opts := []tea.ProgramOption{}
	if !*noAlt {
		opts = append(opts, tea.WithAltScreen())
	}
	p := tea.NewProgram(m, opts...)
	finalModel, err := p.Run()
	// Always clean up placement on exit so the image doesn't "ghost" on
	// whatever screen scrolls up next.
	_, _ = os.Stdout.WriteString(kitty.DeletePlacement(imageID))
	if err != nil {
		die(fmt.Sprintf("tea: %v", err))
	}
	_ = finalModel
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

// syntheticPNG builds a striped test image so the spike does not depend
// on having a sample file on disk. Bands give a visible signal: a real
// render shows colour bars; a failed render shows a flat colour block
// (the placeholder's FG colour bleeding through).
func syntheticPNG(w, h int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	bands := []color.RGBA{
		{0xE6, 0x19, 0x4B, 0xFF}, // red
		{0xF5, 0x82, 0x31, 0xFF}, // orange
		{0xFF, 0xE1, 0x19, 0xFF}, // yellow
		{0x3C, 0xB4, 0x4B, 0xFF}, // green
		{0x42, 0xD4, 0xF4, 0xFF}, // cyan
		{0x43, 0x63, 0xD8, 0xFF}, // blue
		{0x91, 0x1E, 0xB4, 0xFF}, // purple
	}
	bandH := h / len(bands)
	for y := 0; y < h; y++ {
		idx := y / bandH
		if idx >= len(bands) {
			idx = len(bands) - 1
		}
		c := bands[idx]
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// toPNG ensures input bytes are PNG (Kitty f=100 requires PNG). JPEG/GIF
// get decoded and re-encoded.
func toPNG(data []byte) ([]byte, error) {
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return data, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// dumpWriter, if RDR_DUMP_VIEW is set, receives every View() output.
var dumpWriter io.Writer = func() io.Writer {
	path := os.Getenv("RDR_DUMP_VIEW")
	if path == "" {
		return io.Discard
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "RDR_DUMP_VIEW:", err)
		return io.Discard
	}
	return f
}()

// upload holds the APC bytes that upload the image and create its
// virtual placement. If after==true, they are emitted via Init() instead
// of pre-bubbletea so we can test post-alt-screen timing.
type upload struct {
	transmit  string
	placement string
	after     bool
}

// emitCmd returns a tea.Cmd that writes the transmit+placement to stdout.
// It runs in a goroutine spawned by bubbletea's cmd loop, so this races
// against the renderer; we accept the risk here because the purpose is
// diagnostic.
func (u upload) emitCmd() tea.Cmd {
	if !u.after {
		return nil
	}
	return func() tea.Msg {
		_, _ = os.Stdout.WriteString(u.transmit)
		_, _ = os.Stdout.WriteString(u.placement)
		return nil
	}
}

// ─── 1a: minimal ─────────────────────────────────────────────────────

type minimalModel struct {
	id         uint32
	cols, rows int
	upload     upload
}

func (m minimalModel) Init() tea.Cmd { return m.upload.emitCmd() }

func (m minimalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m minimalModel) View() string {
	out := "--- top ---\n" +
		kitty.PlaceholderBlock(m.id, m.cols, m.rows) +
		"--- bottom ---\nmode 1a (minimal). Press q to quit.\n"
	_, _ = dumpWriter.Write([]byte(out))
	return out
}

// ─── 1b: +viewport, 1c: +viewport+lipgloss border ───────────────────

type viewportModel struct {
	vp         viewport.Model
	id         uint32
	cols, rows int
	width      int
	height     int
	framed     bool // 1c
	upload     upload
}

func newViewportModel(id uint32, icols, irows, w, h int, framed bool, up upload) viewportModel {
	// Content: text, image placeholder block, more text. Long enough to scroll.
	var sb bytes.Buffer
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&sb, "line %02d — some filler text before the image so the viewport can scroll\n", i)
	}
	sb.WriteString("\n=== IMAGE BELOW (cols=")
	fmt.Fprintf(&sb, "%d rows=%d) ===\n", icols, irows)
	sb.WriteString(kitty.PlaceholderBlock(id, icols, irows))
	sb.WriteString("=== IMAGE ABOVE ===\n\n")
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&sb, "line %02d — filler text after the image so we can scroll past it\n", i)
	}

	innerW, innerH := w, h
	if framed {
		// Account for lipgloss rounded border (1 cell each side) + padding (1).
		innerW = w - 4
		innerH = h - 4
	}
	vp := viewport.New(innerW, innerH)
	vp.SetContent(sb.String())

	return viewportModel{vp: vp, id: id, cols: icols, rows: irows, width: w, height: h, framed: framed, upload: up}
}

func (m viewportModel) Init() tea.Cmd { return m.upload.emitCmd() }

func (m viewportModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m viewportModel) View() string {
	inner := m.vp.View()
	var out string
	if m.framed {
		border := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 1).
			Width(m.width - 2).
			Height(m.height - 2)
		out = border.Render(inner) + "\nmode 1c (viewport + lipgloss border). j/k to scroll, q to quit.\n"
	} else {
		out = inner + "\nmode 1b (viewport). j/k to scroll, q to quit.\n"
	}
	_, _ = dumpWriter.Write([]byte(out))
	return out
}

// ─── 1d: real inline placement (a=p without U=1) ─────────────────────
//
// Image is uploaded once to Kitty's RAM (a=t) before bubbletea starts.
// In View() we emit kitty.InlinePlacement(id, c, r) at the spot where
// the image should appear. This produces a real placement bound to the
// CURRENT screen buffer (i.e. alt-screen), so the alt-screen problem
// observed in 1a/1b/1c does not apply.
//
// We do NOT use the viewport here: the question is whether bubbletea's
// renderer + lipgloss preserve the embedded APC byte-for-byte. If 1d
// works without viewport, the next experiment is 1d+viewport.

type inlineModel struct {
	id         uint32
	cols, rows int
	width      int
	height     int
}

func newInlineModel(id uint32, icols, irows, w, h int) inlineModel {
	return inlineModel{id: id, cols: icols, rows: irows, width: w, height: h}
}

func (m inlineModel) Init() tea.Cmd { return nil }

func (m inlineModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m inlineModel) View() string {
	out := "--- top: text before image ---\n" +
		kitty.InlinePlacement(m.id, m.cols, m.rows) +
		"--- bottom: text after image ---\nmode 1d (real inline placement). Press q to quit.\n"
	_, _ = dumpWriter.Write([]byte(out))
	return out
}

// ─── 1e / 1f: inline placement + viewport (+ optional lipgloss frame)
//
// The same content as 1b/1c but with the image rendered via real inline
// placement (kitty.InlinePlacement) instead of a placeholder block. This
// is the closest match to the real rdr Reader and tells us whether real
// placements survive scrolling through the viewport.

type inlineViewportModel struct {
	vp         viewport.Model
	id         uint32
	cols, rows int
	width      int
	height     int
	framed     bool
}

func newInlineViewportModel(id uint32, icols, irows, w, h int, framed bool) inlineViewportModel {
	var sb bytes.Buffer
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&sb, "line %02d — filler text BEFORE the image so the viewport can scroll\n", i)
	}
	sb.WriteString("\n=== IMAGE BELOW ===\n")
	sb.WriteString(kitty.InlinePlacement(id, icols, irows))
	sb.WriteString("=== IMAGE ABOVE ===\n\n")
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&sb, "line %02d — filler text AFTER the image so we can scroll past it\n", i)
	}

	innerW, innerH := w, h
	if framed {
		innerW = w - 4
		innerH = h - 4
	}
	vp := viewport.New(innerW, innerH)
	vp.SetContent(sb.String())

	return inlineViewportModel{vp: vp, id: id, cols: icols, rows: irows, width: w, height: h, framed: framed}
}

func (m inlineViewportModel) Init() tea.Cmd { return nil }

func (m inlineViewportModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m inlineViewportModel) View() string {
	inner := m.vp.View()
	var out string
	if m.framed {
		border := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 1).
			Width(m.width - 2).
			Height(m.height - 2)
		out = border.Render(inner) + "\nmode 1f (inline placement + viewport + lipgloss). j/k to scroll, q to quit.\n"
	} else {
		out = inner + "\nmode 1e (inline placement + viewport). j/k to scroll, q to quit.\n"
	}
	_, _ = dumpWriter.Write([]byte(out))
	return out
}
