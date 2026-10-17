package vivid

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/lucasb-eyer/go-colorful"

	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/text"
)

// Theme is the presentation's colour, as values rather than as escapes scattered through the
// renderer: one rung in, every colour decision out.
//
// It is a value the caller passes, not a global, because the terminal's capability is a fact about
// the terminal and tests must be able to state it. The zero Theme paints nothing — the mono rung —
// which is the safe default for a value nobody set.
type Theme struct {
	rung render.Rung
}

// NewTheme is the theme for one colour rung.
func NewTheme(rung render.Rung) Theme { return Theme{rung: rung} }

// Mono reports whether the theme paints nothing at all.
func (t Theme) Mono() bool { return t.rung == render.RungMono }

// The presentation's own palette. The greys are the plain reader's, so the two surfaces agree about
// what "muted" means; the accents are the presentation's, because a cover has no accent and the
// plain reader has exactly one.
const (
	// ColourRune is the colour of the frame's rules and borders: a hairline, not a box.
	ColourRune = "#484f58"
	// ColourMuted is metadata, and the dimming of a row already read.
	ColourMuted = "#8b949e"
	// ColourNew is the news mark, and the bar beside a post the last refresh brought.
	ColourNew = "#7ee787"
	// ColourAccent is the presentation's one bright value: the current destination, the reading
	// pane's header, the selection.
	ColourAccent = "#58a6ff"
	// ColourWarn is the rate-limited field, and nothing else.
	ColourWarn = "#d29922"
	// ColourAlert is a state that stops the session.
	ColourAlert = "#f85149"
	// ColourOnAccent is the text drawn on top of an accent-filled cell.
	ColourOnAccent = "#0d1117"
	// ColourSelection is the selected row's background: a block of colour rather than reverse
	// video, because reverse video inverts the reader's own foreground and reads as a glitch.
	ColourSelection = "#1f2a3a"
)

// authorHues are the accents a post's author is drawn in. Eight, evenly spaced, and deliberately
// desaturated: a feed of saturated names is a feed nobody can read. The hue is chosen by a hash of
// the handle, so the same person is the same colour in every frame and every run without anything
// being stored.
var authorHues = [8]string{
	"#79c0ff", "#7ee787", "#ffa657", "#d2a8ff",
	"#56d4dd", "#ff7b72", "#a5d6ff", "#e3b341",
}

// AuthorColour is the accent for one handle: stable across runs because it is a function of the
// handle and nothing else.
func AuthorColour(handle string) string {
	if handle == "" {
		return ColourMuted
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(handle)))
	return authorHues[h.Sum32()%uint32(len(authorHues))]
}

// fg is the escape that sets a foreground, or "" when this rung cannot paint one.
func (t Theme) fg(hex string) string {
	switch t.rung {
	case render.RungTrueColor:
		if r, g, b, ok := parseHex(hex); ok {
			return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
		}
	case render.Rung256:
		if r, g, b, ok := parseHex(hex); ok {
			return fmt.Sprintf("\x1b[38;5;%dm", to256(r, g, b))
		}
	case render.Rung16:
		return sixteenFG(hex)
	}
	return ""
}

// bg is the escape that sets a background, or "" when this rung cannot paint one.
func (t Theme) bg(hex string) string {
	switch t.rung {
	case render.RungTrueColor:
		if r, g, b, ok := parseHex(hex); ok {
			return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
		}
	case render.Rung256:
		if r, g, b, ok := parseHex(hex); ok {
			return fmt.Sprintf("\x1b[48;5;%dm", to256(r, g, b))
		}
	case render.Rung16:
		if code, ok := sixteenBG(hex); ok {
			return code
		}
	}
	return ""
}

// Paint wraps one string in a foreground and a background. Either may be empty for "leave it
// alone", and a mono theme wraps nothing — the string is returned as it came, byte for byte.
func (t Theme) Paint(s, fg, bg string) string {
	if s == "" || t.Mono() {
		return s
	}
	open := t.fg(fg) + t.bg(bg)
	if open == "" {
		return s
	}
	return open + s + "\x1b[0m"
}

// Fill paints every cell of a string that has already been fitted to a width, background included.
// It differs from Paint in one way that matters for a full-width bar: the background runs to the
// row's last cell, so the bar is a bar and not a word with a coloured highlight.
func (t Theme) Fill(s string, w int, fg, bg string) string {
	if w < 1 {
		return ""
	}
	fitted := text.Pad(s, w)
	if t.Mono() {
		return fitted
	}
	open := t.fg(fg) + t.bg(bg)
	if open == "" {
		return fitted
	}
	return open + fitted + "\x1b[0m"
}

// Gradient paints a string cell by cell from one hue to another. It is used on the surface's
// headings, where it costs nothing to read and makes the two surfaces unmistakably different.
//
// A mono theme returns the string unchanged; a rung that cannot do 24-bit colour uses the second
// hue for the whole run rather than approximating, because a stepped gradient at 16 colours looks
// like a rendering bug.
func (t Theme) Gradient(s, from, to string) string {
	if s == "" || t.Mono() {
		return s
	}
	if t.rung != render.RungTrueColor {
		return t.Paint(s, to, "")
	}
	a, err1 := colorful.Hex(from)
	b, err2 := colorful.Hex(to)
	if err1 != nil || err2 != nil {
		return s
	}
	cells := []rune(s)
	if len(cells) == 0 {
		return s
	}
	var out strings.Builder
	for i, r := range cells {
		f := 0.0
		if len(cells) > 1 {
			f = float64(i) / float64(len(cells)-1)
		}
		c := a.BlendLuv(b, f)
		out.WriteString(t.fg(c.Hex()))
		out.WriteRune(r)
	}
	out.WriteString("\x1b[0m")
	return out.String()
}

// parseHex reads #rrggbb.
func parseHex(hex string) (r, g, b int, ok bool) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0, false
	}
	if _, err := fmt.Sscanf(hex[1:], "%02x%02x%02x", &r, &g, &b); err != nil {
		return 0, 0, 0, false
	}
	return r, g, b, true
}

// to256 is the nearest xterm-256 cube index for one colour: the 6x6x6 cube at 16, plus the greys.
// It is the standard 6-level quantisation and is close enough that a border and its fill still look
// related.
func to256(r, g, b int) int {
	q := func(v int) int { return (v*5 + 127) / 255 }
	return 16 + 36*q(r) + 6*q(g) + q(b)
}

// sixteenFG maps the palette onto the sixteen ANSI colours: the nearest of the eight bright tones,
// by hue rather than by distance, so a green stays green.
func sixteenFG(hex string) string {
	c, err := colorful.Hex(hex)
	if err != nil {
		return ""
	}
	h, _, _ := c.Hsl()
	return fmt.Sprintf("\x1b[%dm", 30+ansiFromHue(h))
}

// sixteenBG is the same map for backgrounds.
func sixteenBG(hex string) (string, bool) {
	c, err := colorful.Hex(hex)
	if err != nil {
		return "", false
	}
	h, _, _ := c.Hsl()
	return fmt.Sprintf("\x1b[%dm", 40+ansiFromHue(h)), true
}

// ansiFromHue buckets a hue in degrees onto the six chromatic ANSI colours plus black and white.
func ansiFromHue(hue float64) int {
	switch {
	case hue < 20 || hue >= 330:
		return 1 // red
	case hue < 45:
		return 3 // yellow
	case hue < 70:
		return 3
	case hue < 160:
		return 2 // green
	case hue < 200:
		return 6 // cyan
	case hue < 260:
		return 4 // blue
	case hue < 330:
		return 5 // magenta
	}
	return 7
}

// Gauge is the request budget as a bar: filled cells for spent requests, empty ones for the rest of
// the floor. It is the same fact the plain reader prints as a number, drawn so a glance is enough.
//
// The scale is the limiter's own floor, because that is the only number a reader can act on; a bar
// against the account's unpublished ceiling would be a bar that never moves.
func Gauge(used, floor, cells int) string {
	if cells < 1 {
		return ""
	}
	if floor < 1 {
		floor = 1
	}
	if used < 0 {
		used = 0
	}
	filled := used * cells / floor
	if filled > cells {
		filled = cells
	}
	return strings.Repeat("▰", filled) + strings.Repeat("▱", cells-filled)
}
