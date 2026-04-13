// kittytest emits a single image via the Kitty Graphics Protocol using
// the internal/kitty package and prints a placeholder block. Used to
// verify the protocol code in isolation, without bubbletea/lipgloss.
package main

import (
	"bytes"
	"encoding/base64"
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

	// Use an ID with all three low bytes non-zero so the FG color is
	// visually bright if Kitty doesn't replace the placeholders.
	id := uint32(0xAABBCC)

	mode := "direct"
	if len(os.Args) >= 5 {
		mode = os.Args[4]
	}

	fmt.Println()
	fmt.Println("mode:", mode, "(override via 4th arg: direct|placeholder)")
	fmt.Println()

	switch mode {
	case "direct":
		// Simplest possible: a=T displays at cursor, no U=1, no placeholders.
		fmt.Print(directDisplay(id, data, cols, rows))
		fmt.Println()
		fmt.Println("^^ direct: if you see image above, Kitty protocol works")
	case "placeholder":
		// Canonical form per kitty icat: a=T,U=1,c=,r= transmit AND
		// a FG color that encodes the low 24 bits of the image id.
		fmt.Print(combinedTransmitPlacement(id, data, cols, rows))
		fmt.Println()
		r := byte((id >> 16) & 0xff)
		g := byte((id >> 8) & 0xff)
		bv := byte(id & 0xff)
		fmt.Printf("\x1b[38:2:%d:%d:%dm", r, g, bv)
		// Emit placeholder cells with row/col diacritics (2 marks each).
		for rr := 0; rr < rows; rr++ {
			rowRune := diacriticFor(rr)
			for c := 0; c < cols; c++ {
				colRune := diacriticFor(c)
				fmt.Printf("\U0010EEEE%c%c", rowRune, colRune)
			}
			fmt.Println()
		}
		fmt.Print("\x1b[0m")
		fmt.Println("^^ placeholder (a=T,U=1 + FG color ID): canonical form")
	case "plain":
		// Simplest possible: transmit with U=1, then a grid of bare U+10EEEE.
		// Kitty auto-fills with the most recently transmitted image.
		fmt.Print(rawKittyTransmit(id, data))
		fmt.Println()
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				fmt.Print("\U0010EEEE")
			}
			fmt.Println()
		}
		fmt.Println("^^ plain: bare U+10EEEE grid, no diacritics")
	case "combined":
		// Transmit with U=1 (virtual) + placeholders with row/col diacritics only.
		fmt.Print(rawKittyTransmit(id, data))
		fmt.Println()
		for r := 0; r < rows; r++ {
			rowRune := diacriticFor(r)
			for c := 0; c < cols; c++ {
				colRune := diacriticFor(c)
				fmt.Printf("\U0010EEEE%c%c", rowRune, colRune)
			}
			fmt.Println()
		}
		fmt.Println("^^ combined: U=1 transmit + row/col diacritics, no id byte")
	}
	_ = kitty.Transmit
}

// combinedTransmitPlacement writes the single a=T escape that transmits
// the image AND creates a virtual placement in one shot, matching how
// Kitty's own icat kitten does it with --unicode-placeholder.
func combinedTransmitPlacement(id uint32, data []byte, cols, rows int) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	var b []byte
	const chunk = 4096
	for i := 0; i < len(encoded); i += chunk {
		end := i + chunk
		if end > len(encoded) {
			end = len(encoded)
		}
		more := 1
		if end == len(encoded) {
			more = 0
		}
		if i == 0 {
			b = append(b, []byte(fmt.Sprintf("\x1b_Ga=T,q=2,f=100,U=1,i=%d,c=%d,r=%d,m=%d;", id, cols, rows, more))...)
		} else {
			b = append(b, []byte(fmt.Sprintf("\x1b_Gm=%d;", more))...)
		}
		b = append(b, encoded[i:end]...)
		b = append(b, '\x1b', '\\')
	}
	return string(b)
}

// transmitOnly transmits the image without creating any placement (a=t
// lowercase). A separate a=p,U=1 call later creates the virtual placement.
func transmitOnly(id uint32, data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	var b []byte
	const chunk = 4096
	for i := 0; i < len(encoded); i += chunk {
		end := i + chunk
		if end > len(encoded) {
			end = len(encoded)
		}
		more := 1
		if end == len(encoded) {
			more = 0
		}
		if i == 0 {
			b = append(b, []byte(fmt.Sprintf("\x1b_Ga=t,f=100,i=%d,m=%d;", id, more))...)
		} else {
			b = append(b, []byte(fmt.Sprintf("\x1b_Gm=%d;", more))...)
		}
		b = append(b, encoded[i:end]...)
		b = append(b, '\x1b', '\\')
	}
	return string(b)
}

// directDisplay uses a=T to transmit AND display the image at the cursor,
// no virtual placement, no Unicode placeholders.
func directDisplay(id uint32, data []byte, cols, rows int) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	var b []byte
	const chunk = 4096
	for i := 0; i < len(encoded); i += chunk {
		end := i + chunk
		if end > len(encoded) {
			end = len(encoded)
		}
		more := 1
		if end == len(encoded) {
			more = 0
		}
		if i == 0 {
			b = append(b, []byte(fmt.Sprintf("\x1b_Ga=T,f=100,i=%d,c=%d,r=%d,m=%d;", id, cols, rows, more))...)
		} else {
			b = append(b, []byte(fmt.Sprintf("\x1b_Gm=%d;", more))...)
		}
		b = append(b, encoded[i:end]...)
		b = append(b, '\x1b', '\\')
	}
	return string(b)
}

func rawKittyTransmit(id uint32, data []byte) string {
	encoded := base64StdEncode(data)
	var b []byte
	const chunk = 4096
	for i := 0; i < len(encoded); i += chunk {
		end := i + chunk
		if end > len(encoded) {
			end = len(encoded)
		}
		more := 1
		if end == len(encoded) {
			more = 0
		}
		if i == 0 {
			b = append(b, []byte(fmt.Sprintf("\x1b_Ga=T,f=100,U=1,i=%d,m=%d;", id, more))...)
		} else {
			b = append(b, []byte(fmt.Sprintf("\x1b_Gm=%d;", more))...)
		}
		b = append(b, encoded[i:end]...)
		b = append(b, '\x1b', '\\')
	}
	return string(b)
}

func base64StdEncode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Small subset of Kitty diacritics so kittytest is self-contained.
var diacriticsList = []rune{
	0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F,
	0x0346, 0x034A, 0x034B, 0x034C, 0x0350, 0x0351, 0x0352, 0x0357,
	0x035B, 0x0363, 0x0364, 0x0365, 0x0366, 0x0367, 0x0368, 0x0369,
	0x036A, 0x036B, 0x036C, 0x036D, 0x036E, 0x036F, 0x0483, 0x0484,
	0x0485, 0x0486, 0x0487, 0x0592, 0x0593, 0x0594, 0x0595, 0x0597,
	0x0598, 0x0599, 0x059C, 0x059D, 0x059E, 0x059F, 0x05A0, 0x05A1,
}

func diacriticFor(v int) rune {
	if v < 0 || v >= len(diacriticsList) {
		return diacriticsList[0]
	}
	return diacriticsList[v]
}
