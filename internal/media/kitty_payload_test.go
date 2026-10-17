package media

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTransmitIsChunkedAt4096 is the protocol's one hard limit: an APC payload longer than a chunk is
// split, every piece but the last carrying `m=1`, and the base64 is not broken by the split.
func TestTransmitIsChunkedAt4096(t *testing.T) {
	payload := bytes.Repeat([]byte{0xAB}, 3000) // base64 is 4000 bytes: one chunk
	one := KittyTransmit(7, payload)
	if strings.Count(one, "\x1b_G") != 1 {
		t.Fatalf("a small image must be one escape, got %d", strings.Count(one, "\x1b_G"))
	}
	if !strings.Contains(one, "a=t,f=100,i=7,q=2,m=0;") {
		t.Fatalf("the first escape's control data is wrong: %q", one[:60])
	}

	payload = bytes.Repeat([]byte{0xAB}, 30000) // base64 is 40000 bytes: ten chunks
	many := KittyTransmit(7, payload)
	parts := strings.Split(many, "\x1b_G")
	if len(parts) != 11 {
		t.Fatalf("%d escapes, want 10", len(parts)-1)
	}
	if !strings.Contains(parts[1], "a=t,f=100,i=7,q=2,m=1;") {
		t.Fatalf("the first escape must announce an upload in progress: %q", parts[1][:40])
	}
	for i, part := range parts[1:] {
		body := strings.SplitN(part, ";", 2)
		if len(body) != 2 {
			t.Fatalf("escape %d has no payload separator", i+1)
		}
		chunk := strings.TrimSuffix(body[1], "\x1b\\")
		if len(chunk) > kittyChunk {
			t.Fatalf("escape %d carries %d bytes of payload", i+1, len(chunk))
		}
	}
	// The reassembled base64 must decode to the original bytes: a split that dropped or doubled a
	// character would still pass every count above.
	var got []byte
	for _, part := range parts[1:] {
		got = append(got, []byte(strings.TrimSuffix(strings.SplitN(part, ";", 2)[1], "\x1b\\"))...)
	}
	decoded, err := base64.StdEncoding.DecodeString(string(got))
	if err != nil {
		t.Fatalf("the reassembled payload does not decode: %v", err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatal("the reassembled payload is not the bytes that went in")
	}
}

// TestPlaceUsesPlaceholders: the placement is virtual, sized in cells, and addressed by nothing but the
// id — the cells are the pane's to print.
func TestPlaceUsesPlaceholders(t *testing.T) {
	place := KittyPlace(9, 12, 4)
	want := "\x1b_Ga=p,U=1,i=9,c=12,r=4,q=2\x1b\\"
	if place != want {
		t.Fatalf("place = %q, want %q", place, want)
	}
	// and an oversized request is clamped to the diacritic table rather than lying about its size
	if got := KittyPlace(9, 40, 40); got != "\x1b_Ga=p,U=1,i=9,c=16,r=16,q=2\x1b\\" {
		t.Fatalf("clamped place = %q", got)
	}
}

// TestCellsCarryTheIDInTheColour is the protocol's addressing rule: every placeholder cell is
// U+10EEEE with its row and column diacritics, painted in a foreground whose blue channel is the id.
func TestCellsCarryTheIDInTheColour(t *testing.T) {
	cells := KittyCells(5, 3, 2)
	if len(cells) != 2 {
		t.Fatalf("%d rows, want 2", len(cells))
	}
	for r, row := range cells {
		if !strings.HasPrefix(row, "\x1b[38;2;0;0;5m") {
			t.Fatalf("row %d does not carry the id: %q", r, row)
		}
		if !strings.HasSuffix(row, "\x1b[39m") {
			t.Fatalf("row %d does not restore the foreground", r)
		}
		body := strings.TrimSuffix(strings.TrimPrefix(row, "\x1b[38;2;0;0;5m"), "\x1b[39m")
		runes := []rune(body)
		if len(runes) != 3*3 {
			t.Fatalf("row %d has %d runes, want 9 (three cells of three runes)", r, len(runes))
		}
		for c := 0; c < 3; c++ {
			if runes[c*3] != utf8.RuneError && runes[c*3] != 0x10EEEE {
				t.Fatalf("row %d cell %d is %U, want U+10EEEE", r, c, runes[c*3])
			}
			if runes[c*3+1] != rowColDiacritics[r] || runes[c*3+2] != rowColDiacritics[c] {
				t.Fatalf("row %d cell %d carries the wrong diacritics", r, c)
			}
		}
	}
}

// TestTmuxWrappingDoublesEveryEscape: tmux forwards a DCS payload verbatim and every ESC inside has to
// be doubled, or the forwarding stops at the first ESC it sees.
func TestTmuxWrappingDoublesEveryEscape(t *testing.T) {
	seq := "\x1b_Ga=t,f=100,i=1,q=2;AAAA\x1b\\"
	wrapped := TmuxWrap(seq)
	if !strings.HasPrefix(wrapped, "\x1bPtmux;") || !strings.HasSuffix(wrapped, "\x1b\\") {
		t.Fatalf("not DCS-framed: %q", wrapped)
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(wrapped, "\x1bPtmux;"), "\x1b\\")
	if strings.Count(inner, "\x1b") != 2*strings.Count(seq, "\x1b") {
		t.Fatalf("an ESC inside was not doubled: %q", inner)
	}
	if TmuxWrap("") != "" {
		t.Fatal("wrapping nothing must produce nothing")
	}
}

// TestDeleteAllRemovesEveryImage: a delete-all carries no placement and no payload.
func TestDeleteAllRemovesEveryImage(t *testing.T) {
	if got, want := KittyDeleteAll(), "\x1b_Ga=d,d=A,q=2\x1b\\"; got != want {
		t.Fatalf("delete = %q, want %q", got, want)
	}
}

// TestKittySupportedReadsTheRightSignals: Ghostty does not export KITTY_WINDOW_ID, so TERM and
// TERM_PROGRAM are what detection rests on — and a terminal that speaks neither is not a graphics
// terminal however modern it is.
func TestKittySupportedReadsTheRightSignals(t *testing.T) {
	cases := []struct {
		env  map[string]string
		want bool
	}{
		{map[string]string{"TERM": "xterm-ghostty"}, true},
		{map[string]string{"TERM_PROGRAM": "ghostty"}, true},
		{map[string]string{"TERM": "xterm-kitty"}, true},
		{map[string]string{"TERM": "tmux-256color", "TERM_PROGRAM": "ghostty"}, true},
		{map[string]string{"TERM": "tmux-256color"}, false},
		{map[string]string{"TERM": "dumb"}, false},
		{map[string]string{}, false},
	}
	for _, c := range cases {
		got := KittySupported(func(k string) string { return c.env[k] })
		if got != c.want {
			t.Fatalf("%v: got %v, want %v", c.env, got, c.want)
		}
	}
}

// TestKittySupportedNameIsTheDetectionOverOneName: under a multiplexer the process's own TERM names
// tmux, so the outer terminal's name arrives from somewhere else — tmux's client term name — and the
// detection has to answer for that string alone.
func TestKittySupportedNameIsTheDetectionOverOneName(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"xterm-ghostty", true},
		{"ghostty", true},
		{"xterm-kitty", true},
		{"tmux-256color", false},
		{"xterm-256color", false},
		{"", false},
	} {
		if got := KittySupportedName(tc.name); got != tc.want {
			t.Fatalf("KittySupportedName(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
	// The env form still answers for a bare terminal.
	if !KittySupported(func(k string) string {
		if k == "TERM" {
			return "xterm-ghostty"
		}
		return ""
	}) {
		t.Fatal("the env form stopped reading TERM")
	}
}
