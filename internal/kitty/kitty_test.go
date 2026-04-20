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

func TestTransmitOnly_UsesLowercaseAction(t *testing.T) {
	out := TransmitOnly(7, []byte("hi"))
	if !strings.Contains(out, "a=t,") {
		t.Fatalf("want lowercase a=t (transmit-only), got: %q", out)
	}
	if strings.Contains(out, "a=T") {
		t.Fatalf("must not include a=T (transmit+display): %q", out)
	}
	if !strings.Contains(out, "i=7") {
		t.Fatalf("want i=7, got: %q", out)
	}
}

func TestCreateVirtualPlacement_IncludesU1(t *testing.T) {
	out := CreateVirtualPlacement(5, 10, 3)
	for _, want := range []string{"a=p", "U=1", "i=5", "c=10", "r=3"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestDeletePlacement_FreesImageByID(t *testing.T) {
	out := DeletePlacement(9)
	// Uppercase d=I frees both placements and image data — used on
	// reader exit so Kitty RAM doesn't bloat from articles we've
	// closed. Inline per-frame deletion (ReplacePlaceholders) uses
	// lowercase d=i to keep image data hot for the next placement.
	if !strings.Contains(out, "d=I") {
		t.Fatalf("want d=I (free image on exit), got: %q", out)
	}
	if !strings.Contains(out, "a=d") {
		t.Fatalf("want a=d (delete action), got: %q", out)
	}
	if !strings.Contains(out, "i=9") {
		t.Fatalf("want i=9, got: %q", out)
	}
}

func TestPlaceholderFill_MarkerAtFirstCell(t *testing.T) {
	block := PlaceholderFill(3, 5, 2)
	lines := strings.Split(strings.TrimRight(block, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 rows, got %d: %q", len(lines), block)
	}
	// First row: first rune is marker for index 3.
	firstRowRunes := []rune(lines[0])
	if len(firstRowRunes) != 5 {
		t.Fatalf("first row runes: got %d, want 5", len(firstRowRunes))
	}
	want := placeholderMarkerBase + 3
	if firstRowRunes[0] != want {
		t.Fatalf("marker rune: got %U, want %U", firstRowRunes[0], want)
	}
	for i := 1; i < 5; i++ {
		if firstRowRunes[i] != placeholderFillRune {
			t.Fatalf("first row cell %d: got %U, want fill", i, firstRowRunes[i])
		}
	}
	// Subsequent rows: all fill runes.
	for i := 1; i < len(lines); i++ {
		for j, r := range []rune(lines[i]) {
			if r != placeholderFillRune {
				t.Fatalf("row %d cell %d: got %U, want fill", i, j, r)
			}
		}
	}
}

func TestReplacePlaceholders_InlinesDeleteAndPlace(t *testing.T) {
	block := PlaceholderFill(0, 4, 3)
	input := "before\n" + block + "after"
	placements := []Placement{{ID: 0x112233, Cols: 4, Rows: 3}}
	out := ReplacePlaceholders(input, placements)
	if !strings.Contains(out, "a=d,d=i,i=1122867") {
		t.Fatalf("want per-image delete APC with ID 0x112233, got: %q", out)
	}
	if !strings.Contains(out, "a=p,q=2,i=1122867,c=4,r=3,C=1") {
		t.Fatalf("want place APC when fully visible, got: %q", out)
	}
	// Fill runes must be gone from the output (replaced by spaces).
	if strings.ContainsRune(out, placeholderFillRune) {
		t.Fatalf("fill runes leaked past post-processing: %q", out)
	}
}

func TestReplacePlaceholders_ScaledWhenPartiallyVisible(t *testing.T) {
	// Marker row + only 1 fill row in output (image clipped at pane edge).
	block := string(placeholderMarkerBase) + strings.Repeat(string(placeholderFillRune), 9) + "\n" +
		strings.Repeat(string(placeholderFillRune), 10) + "\n" +
		"normal text\n"
	placements := []Placement{{ID: 1, Cols: 10, Rows: 4}}
	out := ReplacePlaceholders(block, placements)
	// visibleRows = 2 (marker + 1 fill), so cols should scale to Cols*2/4 = 5.
	if !strings.Contains(out, "c=5,r=2") {
		t.Fatalf("want scaled c=5,r=2, got: %q", out)
	}
}

func TestReplacePlaceholders_EmitsDeleteForOffscreenBlocks(t *testing.T) {
	// No marker in rendered — placeholder fully scrolled off-screen.
	placements := []Placement{{ID: 99, Cols: 10, Rows: 5}}
	out := ReplacePlaceholders("just text\nno markers here\n", placements)
	if !strings.Contains(out, "a=d,d=i,i=99") {
		t.Fatalf("want suffix delete for offscreen image ID 99, got: %q", out)
	}
}

func TestPlaceholderBlock_EmitsFGColorPerRow(t *testing.T) {
	// 0xAABBCC: low 24 bits encode to RGB 170, 187, 204.
	block := PlaceholderBlock(0xAABBCC, 3, 2)

	// Each row must independently set our FG color because lipgloss
	// borders emit a reset between lines.
	colorMarker := "\x1b[38;2;170;187;204m"
	if count := strings.Count(block, colorMarker); count != 2 {
		t.Fatalf("want 2 FG color prefixes (one per row), got %d: %q", count, block)
	}
	if count := strings.Count(block, "\x1b[0m"); count != 2 {
		t.Fatalf("want 2 SGR resets (one per row), got %d", count)
	}

	// Validate placeholder cell structure in each row.
	lines := strings.Split(strings.TrimRight(block, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 rows, got %d", len(lines))
	}
	for _, line := range lines {
		stripped := strings.TrimPrefix(line, colorMarker)
		stripped = strings.TrimSuffix(stripped, "\x1b[0m")
		runes := []rune(stripped)
		// Each cell is placeholder + 2 combining marks = 3 runes.
		if len(runes) != 3*3 {
			t.Fatalf("row has %d runes, want 9: %q", len(runes), stripped)
		}
		for i := 0; i < len(runes); i += 3 {
			if runes[i] != 0x10EEEE {
				t.Fatalf("cell %d: got %U, want U+10EEEE", i/3, runes[i])
			}
		}
	}
}
