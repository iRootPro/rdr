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
	data := make([]byte, 10_000)
	for i := range data {
		data[i] = byte(i)
	}
	out := Transmit(1, data)
	envelopes := strings.Count(out, "\x1b\\")
	if envelopes < 4 {
		t.Fatalf("expected ≥4 chunks, got %d envelopes", envelopes)
	}
	if !strings.Contains(out, "m=0") {
		t.Fatalf("no terminating chunk with m=0: %q", out[:200])
	}
}

func TestPlaceholderBlock_DimensionsAndEncoding(t *testing.T) {
	block := PlaceholderBlock(1, 3, 2)
	lines := strings.Split(strings.TrimRight(block, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 rows, got %d: %q", len(lines), block)
	}
	for _, line := range lines {
		runes := []rune(line)
		if len(runes) != 3*4 {
			t.Fatalf("row has %d runes, want %d: %q", len(runes), 3*4, line)
		}
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
	for _, want := range []string{"a=p", "U=1", "i=7", "q=2"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}
