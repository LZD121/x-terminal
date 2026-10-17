package media

import (
	"encoding/base64"
	"strconv"
	"strings"
)

// The Kitty graphics protocol, as four pure functions. Everything that decides bytes is here and
// everything that writes them is kitty.go, so the protocol can be held to its own spec by tests with
// no terminal in the room.
//
// Two facts from .planning/research/TERMINAL-MEDIA-PROBE.md shape the payloads:
//
//   - A bare Ghostty renders an APC graphics sequence; tmux consumes a bare one.
//   - A DCS-wrapped one renders under tmux, but a plain *put* leaves tmux's grid empty, so later text
//     is drawn across the image and scroll smears it. The Unicode-placeholder form fixes that by
//     making the image occupy real cells: the terminal paints it into cells tmux knows about.
//
// So the shipped form is: transmit once, place with placeholders, and let the reading pane print the
// placeholder cells itself.

// kittyChunk is the largest base64 payload one APC string may carry. The protocol's own limit is 4096
// bytes of payload per escape, and the probe's single-chunk payload was well under it.
const kittyChunk = 4096

// placeholder is U+10EEEE, the codepoint the kitty protocol reserves for a cell an image occupies.
const placeholder = "\U0010EEEE"

// rowColDiacritics are the combining marks that carry a placeholder's row and column. The protocol
// defines 255 of each; sixteen suffice for a reading pane's image block, and a full table would be a
// second thing to get wrong. The values are kitty's own table, first sixteen entries.
var rowColDiacritics = []rune{
	0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F,
	0x0346, 0x034A, 0x034B, 0x034C, 0x0350, 0x0351, 0x0352, 0x0357,
}

// maxPlaceholderDim is how many cells one image may occupy on either axis: the table's own length.
const maxPlaceholderDim = 16

// KittyTransmit is the sequence that uploads PNG bytes under an image id: one APC string, or several
// when the payload is longer than one escape may carry, each continuation marked `m=1`.
//
// `a=t` transmits without displaying, `f=100` says the payload is a PNG, and `q=2` suppresses the
// terminal's replies — this program reads no answer from the graphics protocol, and a reply would be
// bytes arriving in the middle of a frame it is painting.
func KittyTransmit(id int, png []byte) string {
	payload := base64.StdEncoding.EncodeToString(png)
	var b strings.Builder
	for first := true; first || payload != ""; first = false {
		chunk := payload
		if len(chunk) > kittyChunk {
			chunk, payload = chunk[:kittyChunk], chunk[kittyChunk:]
		} else {
			payload = ""
		}
		more := 0
		if payload != "" {
			more = 1
		}
		b.WriteString("\x1b_G")
		switch {
		case first:
			b.WriteString("a=t,f=100,i=" + strconv.Itoa(id) + ",q=2,m=" + strconv.Itoa(more))
		default:
			b.WriteString("q=2,m=" + strconv.Itoa(more))
		}
		b.WriteString(";")
		b.WriteString(chunk)
		b.WriteString("\x1b\\")
	}
	return b.String()
}

// KittyPlace is the sequence that turns the transmitted image into a virtual placement of cols by rows
// cells: `a=p,U=1` is a placement addressed by placeholders, `c=` and `r=` are its size in cells.
//
// The image is not drawn by this sequence. The caller prints the placeholder cells in the reading
// pane, and the terminal paints the image into them — which is what makes the image occupy cells tmux
// knows about.
func KittyPlace(id, cols, rows int) string {
	if cols > maxPlaceholderDim {
		cols = maxPlaceholderDim
	}
	if rows > maxPlaceholderDim {
		rows = maxPlaceholderDim
	}
	return "\x1b_Ga=p,U=1,i=" + strconv.Itoa(id) + ",c=" + strconv.Itoa(cols) +
		",r=" + strconv.Itoa(rows) + ",q=2\x1b\\"
}

// KittyCells is the block of placeholder cells for an image id of cols by rows: one row per line, each
// cell the placeholder codepoint plus its row and column diacritics, and the whole row painted in the
// foreground colour that carries the image id.
//
// The colour is how the protocol says which image a cell belongs to: `38;2;r;g;b` with the id in the
// blue channel, which is the encoding the reference implementation uses. The id must therefore be in
// 0..255; the pane's allocator keeps it there.
func KittyCells(id, cols, rows int) []string {
	if cols < 1 || rows < 1 {
		return nil
	}
	if cols > maxPlaceholderDim {
		cols = maxPlaceholderDim
	}
	if rows > maxPlaceholderDim {
		rows = maxPlaceholderDim
	}
	colour := "\x1b[38;2;0;0;" + strconv.Itoa(id%256) + "m"
	const reset = "\x1b[39m"
	out := make([]string, 0, rows)
	for r := range rows {
		var b strings.Builder
		b.WriteString(colour)
		for c := range cols {
			b.WriteString(placeholder)
			b.WriteRune(rowColDiacritics[r])
			b.WriteRune(rowColDiacritics[c])
		}
		b.WriteString(reset)
		out = append(out, b.String())
	}
	return out
}

// KittyDeleteAll is the sequence that removes every image the terminal is holding. It is what panic
// sends before it clears the screen: without it the image lives in the terminal's own store, and a
// full repaint of the cover would draw around a picture of a post.
func KittyDeleteAll() string {
	return "\x1b_Ga=d,d=A,q=2\x1b\\"
}

// TmuxWrap wraps one terminal sequence for the multiplexer: DCS-framed, every ESC inside doubled.
// tmux forwards a DCS-wrapped escape verbatim and consumes a bare one.
//
// It is a no-op for the empty string so a caller can wrap unconditionally.
func TmuxWrap(seq string) string {
	if seq == "" {
		return ""
	}
	return "\x1bPtmux;" + strings.ReplaceAll(seq, "\x1b", "\x1b\x1b") + "\x1b\\"
}

// KittySupported reports whether this terminal speaks the graphics protocol. It reads TERM and
// TERM_PROGRAM and never KITTY_WINDOW_ID as its only signal, which Ghostty does not export — the
// probe measured that.
func KittySupported(env func(string) string) bool {
	if env == nil {
		return false
	}
	if env("KITTY_WINDOW_ID") != "" {
		// Another kitty-protocol terminal may export it; Ghostty does not, so this is a bonus
		// signal rather than the one the detection rests on.
		return true
	}
	if KittySupportedName(env("TERM")) {
		return true
	}
	return KittySupportedName(env("TERM_PROGRAM"))
}

// KittySupportedName is the detection over one terminal name: the value of TERM, of TERM_PROGRAM, or
// — under a multiplexer — the name tmux reports for the terminal attached to its client.
//
// That third source is what the tmux case needs: inside tmux TERM is `tmux-256color` and the outer
// terminal's name is nowhere in this process's environment, so a shell that only read TERM would
// conclude no graphics on a machine whose terminal has them. Measured on this host on 2026-10-17:
// `tmux display-message -p '#{client_termname}'` answered `xterm-ghostty` while TERM was
// `tmux-256color`.
func KittySupportedName(name string) bool {
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	return strings.Contains(lower, "ghostty") || strings.Contains(lower, "kitty")
}
