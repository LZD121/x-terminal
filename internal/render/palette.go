package render

import (
	"os"
	"strings"
)

// Rung is how much colour the terminal can be asked to draw: the degradation ladder, richest
// first.
type Rung uint8

const (
	// RungTrueColor is a terminal advertising 24-bit colour: the hex values below.
	RungTrueColor Rung = iota
	// Rung256 is a terminal advertising 256 colours: the same three colours as palette indices.
	Rung256
	// Rung16 is any other terminal with a TERM at all: the ANSI colours.
	Rung16
	// RungMono is no colour at all: an opt-out (NO_COLOR) or a TERM that cannot draw one (dumb,
	// or empty, which is no terminal).
	RungMono
)

// The escapes. The muted, state and state-stop greys are the UI-SPEC's own values in all three
// colour rungs: #8a8a8a is 245, #d7af00 is 178 and #d75f5f is 167, which is why the 256 rung
// emits the index rather than an approximation of the hex.
const (
	sgrReset             = "\x1b[0m"
	sgrReverse           = "\x1b[7m"
	sgrReverseOff        = "\x1b[27m"
	sgrDefaultForeground = "\x1b[39m"

	sgrMutedTrueColor = "\x1b[38;2;138;138;138m" // #8a8a8a
	sgrMuted256       = "\x1b[38;5;245m"
	sgrMuted16        = "\x1b[90m"

	sgrStateTrueColor = "\x1b[38;2;215;175;0m" // #d7af00
	sgrState256       = "\x1b[38;5;178m"
	sgrState16        = "\x1b[33m"

	sgrStateStopTrueColor = "\x1b[38;2;215;95;95m" // #d75f5f
	sgrStateStop256       = "\x1b[38;5;167m"
	sgrStateStop16        = "\x1b[31m"

	truecolorMarker = "truecolor"
	truecolorShort  = "24bit"
	term256Marker   = "256color"
	termDumb        = "dumb"
)

// RungFromEnv chooses the rung from the environment and from nothing else: never from a config
// setting, never re-evaluated mid-session, never from a terminal query (a query would add a round
// trip to the first frame).
//
// The order of the checks is the contract, and it is this order because the opt-outs are opt-outs
// and must win. NO_COLOR and TERM=dumb come first: they are a user saying "no colour", and a
// terminal's own advertisement of what it can draw must not override the person looking at it.
// COLORTERM comes before TERM because a terminal that says truecolor in COLORTERM and 256color in
// TERM means the richer one — that pair is what several emulators and tmux send — and reading TERM
// first would draw the 256 rung on a truecolor terminal.
func RungFromEnv() Rung {
	if os.Getenv("NO_COLOR") != "" {
		return RungMono
	}
	term := os.Getenv("TERM")
	if term == "" || term == termDumb {
		return RungMono
	}
	if colorterm := os.Getenv("COLORTERM"); strings.Contains(colorterm, truecolorMarker) || strings.Contains(colorterm, truecolorShort) {
		return RungTrueColor
	}
	if strings.Contains(term, term256Marker) {
		return Rung256
	}
	return Rung16
}

// Role is what a run of text is for. Every role is one colour (or one attribute) and every run
// has exactly one role; nothing carries meaning by colour alone, so a role is always a second
// statement of something the characters already say.
type Role uint8

const (
	// RoleDominant is the terminal's own foreground: post text and handles. It paints nothing.
	RoleDominant Role = iota
	// RoleMuted is metadata: stamps, markers, the status line's other fields, the attachment
	// line, the help overlay's key column.
	RoleMuted
	// RoleState is the one accent, used only for the status line's state word while a recoverable
	// failure is active.
	RoleState
	// RoleStateStop is the other accent value, used only for the status line's state word when
	// the session is over (expired, soft-banned, out of budget): those two cases call for
	// different human actions, and the words say so as well.
	RoleStateStop
	// RoleSelection is reverse video on the selected block's rows. The `>` in column 1 is what
	// carries the selection; the colour repeats it, and a mono terminal keeps the mark.
	RoleSelection
)

// Palette paints runs at one rung. Its zero value is the mono rung of a terminal that cannot draw
// SGR at all, which is the safe default for a value nobody set.
type Palette struct {
	rung Rung
	// dumb is whether the terminal can be asked for SGR at all. TERM=dumb is the one case where
	// even reverse video is withheld: the selection mark alone then marks the selected block.
	dumb bool
}

// For returns the palette for a rung. The terminal's dumbness is read here rather than passed in,
// because it is the terminal's state and not the caller's decision.
func For(rung Rung) Palette {
	return Palette{rung: rung, dumb: os.Getenv("TERM") == termDumb}
}

// Rung is the rung this palette paints at.
func (p Palette) Rung() Rung { return p.rung }

// SGR returns the escape prefix for a role: the sequence that turns the role on, or the empty
// string when this palette cannot paint it. RoleDominant is the terminal's own foreground and
// returns nothing at every rung — there is nothing to turn on.
func (p Palette) SGR(role Role) string {
	if role == RoleDominant {
		return ""
	}
	if role == RoleSelection {
		if p.dumb {
			return ""
		}
		return sgrReverse
	}
	if p.rung == RungMono {
		return ""
	}
	switch role {
	case RoleMuted:
		switch p.rung {
		case RungTrueColor:
			return sgrMutedTrueColor
		case Rung256:
			return sgrMuted256
		default:
			return sgrMuted16
		}
	case RoleState:
		switch p.rung {
		case RungTrueColor:
			return sgrStateTrueColor
		case Rung256:
			return sgrState256
		default:
			return sgrState16
		}
	case RoleStateStop:
		switch p.rung {
		case RungTrueColor:
			return sgrStateStopTrueColor
		case Rung256:
			return sgrStateStop256
		default:
			return sgrStateStop16
		}
	}
	return ""
}

// Reset is the escape that ends everything this palette can paint, and the empty string when the
// palette paints nothing at all (the mono rung of a dumb terminal): a reset with nothing to reset
// is a stray sequence in a frame a user may be diffing.
func (p Palette) Reset() string {
	if p.rung == RungMono && p.dumb {
		return ""
	}
	return sgrReset
}

// paint wraps s in the role's escape, and returns s unchanged when this palette has none for it.
// An empty run paints nothing rather than a pair of escapes around nothing.
func (p Palette) paint(role Role, s string) string {
	if s == "" {
		return ""
	}
	on := p.SGR(role)
	if on == "" {
		return s
	}
	return on + s + p.off(role)
}

// off is the sequence that undoes SGR(role) — deliberately not a full reset. A muted stamp inside
// the selected block's reverse video must end without turning the reverse video off, so the off
// code is the specific one: the foreground back to the terminal's own, the reverse attribute
// cleared.
func (p Palette) off(role Role) string {
	switch role {
	case RoleDominant:
		return ""
	case RoleSelection:
		return sgrReverseOff
	default:
		return sgrDefaultForeground
	}
}
