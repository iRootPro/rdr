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

func TestTransmit_SingleChunk_EmitsCombinedDirective(t *testing.T) {
	data := []byte("hello")
	out := Transmit(42, data, 10, 5)
	if !strings.HasPrefix(out, "\x1b_G") {
		t.Fatalf("missing prefix: %q", out)
	}
	if !strings.HasSuffix(out, "\x1b\\") {
		t.Fatalf("missing suffix: %q", out)
	}
	for _, want := range []string{"a=T", "q=2", "f=100", "U=1", "i=42", "c=10", "r=5"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
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
	out := Transmit(1, data, 40, 15)
	envelopes := strings.Count(out, "\x1b\\")
	if envelopes < 4 {
		t.Fatalf("expected ≥4 chunks, got %d envelopes", envelopes)
	}
	if !strings.Contains(out, "m=0") {
		t.Fatalf("no terminating chunk with m=0: %q", out[:200])
	}
}

func TestPlaceholderBlock_EmitsFGColorAndDiacritics(t *testing.T) {
	// 0xAABBCC: low 24 bits encode to RGB 170, 187, 204.
	block := PlaceholderBlock(0xAABBCC, 3, 2)

	if !strings.Contains(block, "\x1b[38:2:170:187:204m") {
		t.Fatalf("missing fg color prefix: %q", block)
	}
	if !strings.HasSuffix(block, "\x1b[0m") {
		t.Fatalf("missing SGR reset: %q", block)
	}

	// Strip color codes and inspect the cell grid.
	stripped := strings.TrimPrefix(block, "\x1b[38:2:170:187:204m")
	stripped = strings.TrimSuffix(stripped, "\x1b[0m")
	lines := strings.Split(strings.TrimRight(stripped, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 rows, got %d: %q", len(lines), stripped)
	}
	for _, line := range lines {
		runes := []rune(line)
		// Each cell is placeholder + 2 combining marks = 3 runes.
		if len(runes) != 3*3 {
			t.Fatalf("row has %d runes, want %d: %q", len(runes), 3*3, line)
		}
		for i := 0; i < len(runes); i += 3 {
			if runes[i] != 0x10EEEE {
				t.Fatalf("cell %d: got %U, want U+10EEEE", i/3, runes[i])
			}
		}
	}
}
