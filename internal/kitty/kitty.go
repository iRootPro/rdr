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
