package kitty

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/term"
)

// CellSize reports the pixel dimensions of a single terminal cell by
// sending the xterm "report cell size" CSI (`CSI 16 t`) and parsing the
// `CSI 6 ; H ; W t` response from stdin. Must be called BEFORE bubbletea
// starts — once bubbletea captures stdin we can't reliably read the
// response without racing the renderer.
//
// Runs with a short deadline so non-conforming terminals don't hang the
// startup path; returns zeros on any error or timeout, signalling the
// caller to fall back to heuristic defaults.
func CellSize() (width, height int, ok bool) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return 0, 0, false
	}
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return 0, 0, false
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	if _, err := fmt.Fprint(os.Stdout, "\x1b[16t"); err != nil {
		return 0, 0, false
	}

	// Read with a 200ms soft deadline. Terminals that don't implement
	// the query just stay silent; we shouldn't block startup waiting
	// for them.
	done := make(chan struct{})
	var buf []byte
	go func() {
		defer close(done)
		tmp := make([]byte, 64)
		n, rerr := os.Stdin.Read(tmp)
		if rerr != nil {
			return
		}
		buf = tmp[:n]
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		return 0, 0, false
	}

	return parseCellSize(string(buf))
}

// Matches both 7-bit (ESC [) and 8-bit (C1 CSI = 0x9b) forms of the
// response. Some emulators emit the 8-bit form even in UTF-8 mode.
var csiCellSizeRE = regexp.MustCompile(`(?:\x1b\[|\x9b)6;(\d+);(\d+)t`)

func parseCellSize(s string) (width, height int, ok bool) {
	m := csiCellSizeRE.FindStringSubmatch(s)
	if len(m) != 3 {
		return 0, 0, false
	}
	h, err := strconv.Atoi(m[1])
	if err != nil || h <= 0 {
		return 0, 0, false
	}
	w, err := strconv.Atoi(m[2])
	if err != nil || w <= 0 {
		return 0, 0, false
	}
	return w, h, true
}
