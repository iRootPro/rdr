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
