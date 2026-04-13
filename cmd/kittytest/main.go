// kittytest emits a single image via the Kitty Graphics Protocol using
// the internal/kitty package and prints a placeholder block. Used to
// verify the protocol code in isolation, without bubbletea/lipgloss.
package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"

	"github.com/iRootPro/rdr/internal/kitty"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kittytest <image-path> [cols] [rows]")
		os.Exit(1)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}

	// Convert to PNG if not already — Kitty f=100 requires PNG.
	if !(len(data) >= 8 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G') {
		img, _, derr := image.Decode(bytes.NewReader(data))
		if derr != nil {
			fmt.Fprintln(os.Stderr, "decode:", derr)
			os.Exit(1)
		}
		var buf bytes.Buffer
		if eerr := png.Encode(&buf, img); eerr != nil {
			fmt.Fprintln(os.Stderr, "encode:", eerr)
			os.Exit(1)
		}
		data = buf.Bytes()
		fmt.Fprintln(os.Stderr, "converted to PNG,", len(data), "bytes")
	}

	cols, rows := 40, 15
	if len(os.Args) >= 4 {
		_, _ = fmt.Sscanf(os.Args[2], "%d", &cols)
		_, _ = fmt.Sscanf(os.Args[3], "%d", &rows)
	}

	id := uint32(42)
	fmt.Print(kitty.Transmit(id, data))
	fmt.Print(kitty.Placement(id, cols, rows))
	fmt.Println()
	fmt.Print(kitty.PlaceholderBlock(id, cols, rows))
	fmt.Println()
	fmt.Println("^^ if you see an image above, protocol code is correct")
}
