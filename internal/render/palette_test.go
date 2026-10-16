package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// env sets the four environment variables the rung is chosen from, so no case can inherit this
// host's shell: its non-interactive shells export TERM=dumb and NO_COLOR=1.
func env(t *testing.T, term, colorterm, noColor string) {
	t.Helper()
	t.Setenv("TERM", term)
	t.Setenv("COLORTERM", colorterm)
	t.Setenv("NO_COLOR", noColor)
}

// TestRungFromEnvIsTheLadder: the four rungs, chosen from the environment and from nothing else.
func TestRungFromEnvIsTheLadder(t *testing.T) {
	tests := []struct {
		name      string
		term      string
		colorterm string
		noColor   string
		want      Rung
	}{
		{"a truecolor terminal", "xterm-256color", "truecolor", "", RungTrueColor},
		{"a 24-bit terminal that spells it differently", "xterm-256color", "24bit", "", RungTrueColor},
		{"a 256-colour terminal", "xterm-256color", "", "", Rung256},
		{"a 16-colour terminal", "xterm", "", "", Rung16},
		{"a terminal with no name at all", "", "", "", RungMono},
		{"a dumb terminal", "dumb", "", "", RungMono},
		{"NO_COLOR on a 256-colour terminal", "xterm-256color", "", "1", RungMono},
		{"NO_COLOR on a 16-colour terminal", "xterm", "", "1", RungMono},
		{"NO_COLOR on a truecolor terminal: the opt-out wins", "xterm-256color", "truecolor", "1", RungMono},
		{"NO_COLOR and a dumb terminal together", "dumb", "", "1", RungMono},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env(t, tc.term, tc.colorterm, tc.noColor)
			if got := RungFromEnv(); got != tc.want {
				t.Fatalf("RungFromEnv() = %d, want %d", got, tc.want)
			}
		})
	}

	// And the order is the contract, not just the outcome: a terminal that advertises truecolor in
	// COLORTERM while its TERM says 256color is the richer one.
	env(t, "xterm-256color", "truecolor", "")
	if got := RungFromEnv(); got != RungTrueColor {
		t.Fatalf("COLORTERM was read after TERM: got rung %d, want truecolor", got)
	}
	// The empty-but-set NO_COLOR is not an opt-out (https://no-color.org/ reads it as "set and
	// non-empty"), and an empty one with a truecolor COLORTERM still selects truecolor.
	env(t, "xterm-256color", "truecolor", "")
	if got := RungFromEnv(); got != RungTrueColor {
		t.Fatalf("an empty NO_COLOR suppressed colour: got rung %d", got)
	}
}

// TestSGRIsTheUISpecsTable pins every escape: the three colours at each rung, the attribute for
// the selection, and nothing at all for the dominant role — the terminal's own foreground is not
// something a program can paint.
func TestSGRIsTheUISpecsTable(t *testing.T) {
	env(t, "xterm-256color", "truecolor", "")
	truecolor := For(RungTrueColor)
	env(t, "xterm-256color", "", "")
	term256 := For(Rung256)
	env(t, "xterm", "", "")
	term16 := For(Rung16)
	env(t, "xterm", "", "")
	mono := For(RungMono)

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"dominant, truecolor", truecolor.SGR(RoleDominant), ""},
		{"muted, truecolor", truecolor.SGR(RoleMuted), "\x1b[38;2;138;138;138m"},
		{"state, truecolor", truecolor.SGR(RoleState), "\x1b[38;2;215;175;0m"},
		{"state stop, truecolor", truecolor.SGR(RoleStateStop), "\x1b[38;2;215;95;95m"},
		{"selection, truecolor", truecolor.SGR(RoleSelection), "\x1b[7m"},
		{"muted, 256", term256.SGR(RoleMuted), "\x1b[38;5;245m"},
		{"state, 256", term256.SGR(RoleState), "\x1b[38;5;178m"},
		{"state stop, 256", term256.SGR(RoleStateStop), "\x1b[38;5;167m"},
		{"muted, 16", term16.SGR(RoleMuted), "\x1b[90m"},
		{"state, 16", term16.SGR(RoleState), "\x1b[33m"},
		{"state stop, 16", term16.SGR(RoleStateStop), "\x1b[31m"},
		{"muted, mono", mono.SGR(RoleMuted), ""},
		{"state, mono", mono.SGR(RoleState), ""},
		{"state stop, mono", mono.SGR(RoleStateStop), ""},
		{"selection, mono", mono.SGR(RoleSelection), "\x1b[7m"},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s: SGR = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// TestNoMeaningIsCarriedByColourAlone is A1 at the palette's own level: for every role and every
// rung, stripping the painted run leaves the text it was given, and the mono and truecolor strips
// of the same run are byte-identical — so a colour adds nothing a reader has to see.
func TestNoMeaningIsCarriedByColourAlone(t *testing.T) {
	const sample = "42 requests, 3 rate-limited"
	roles := []Role{RoleDominant, RoleMuted, RoleState, RoleStateStop, RoleSelection}
	rungs := []Rung{RungTrueColor, Rung256, Rung16, RungMono}

	stripped := map[Rung]map[Role]string{}
	for _, rung := range rungs {
		env(t, "xterm-256color", "", "")
		pal := For(rung)
		stripped[rung] = map[Role]string{}
		for _, role := range roles {
			painted := pal.SGR(role) + sample + pal.Reset()
			if got := ansi.Strip(painted); got != sample {
				t.Errorf("rung %d role %d: stripping %q leaves %q, want the text itself", rung, role, painted, got)
			}
			stripped[rung][role] = ansi.Strip(painted)
		}
	}
	for _, role := range roles {
		if got, want := stripped[RungMono][role], stripped[RungTrueColor][role]; got != want {
			t.Errorf("role %d strips to %q in mono and %q in truecolor", role, got, want)
		}
		if got, want := ansi.Strip(paletteText(t, RungTrueColor, role)), sample; got != want {
			t.Errorf("role %d: the painted run's strip is %q, want %q", role, got, want)
		}
	}
}

// paletteText paints one run with the palette of a rung, in a colour-capable terminal.
func paletteText(t *testing.T, rung Rung, role Role) string {
	t.Helper()
	env(t, "xterm-256color", "", "")
	pal := For(rung)
	return pal.SGR(role) + "42 requests, 3 rate-limited" + pal.Reset()
}

// TestTheMonoRungEmitsNoColour: the mono rung paints nothing except the selection's reverse video,
// which is an attribute and not a colour, and a dumb terminal paints not even that — the `>` in
// column 1 is then the whole of the selection.
func TestTheMonoRungEmitsNoColour(t *testing.T) {
	const sample = "a stamp"
	t.Run("a mono terminal", func(t *testing.T) {
		env(t, "xterm", "", "1")
		pal := For(RungMono)
		for _, role := range []Role{RoleDominant, RoleMuted, RoleState, RoleStateStop} {
			if got := pal.paint(role, sample); got != sample {
				t.Errorf("role %d emitted %q in mono", role, got)
			}
		}
		if got, want := pal.paint(RoleSelection, sample), "\x1b[7m"+sample+"\x1b[27m"; got != want {
			t.Errorf("the selection in mono is %q, want reverse video", got)
		}
		if got, want := pal.Reset(), "\x1b[0m"; got != want {
			t.Errorf("Reset() = %q in mono, want %q", got, want)
		}
		if !strings.Contains(pal.paint(RoleSelection, sample), "\x1b[7m") {
			t.Error("the mono rung lost the reverse video the selection keeps")
		}
	})
	t.Run("a dumb terminal", func(t *testing.T) {
		env(t, "dumb", "", "1")
		pal := For(RungMono)
		for _, role := range []Role{RoleDominant, RoleMuted, RoleState, RoleStateStop, RoleSelection} {
			if got := pal.paint(role, sample); got != sample {
				t.Errorf("role %d emitted %q on a dumb terminal", role, got)
			}
			if got := pal.SGR(role); got != "" {
				t.Errorf("role %d has the escape %q on a dumb terminal", role, got)
			}
		}
		if got := pal.Reset(); got != "" {
			t.Errorf("Reset() = %q on a dumb terminal, want nothing", got)
		}
	})
	t.Run("a terminal with no NAME", func(t *testing.T) {
		// TERM unset is RungMono too, and there is no terminal to ask for reverse video: the rung
		// is the mono one and only TERM=dumb silences the selection, which is what the UI-SPEC's
		// table says.
		env(t, "", "", "")
		pal := For(RungMono)
		if got := pal.SGR(RoleSelection); got != "\x1b[7m" {
			t.Errorf("the selection on a TERM-less mono rung is %q", got)
		}
	})
}

// TestPaintNestsWithoutEatingTheBar: a muted run inside the selected block's reverse video ends
// with a specific off code rather than a full reset, so the bar keeps running to the end of the
// row. Without that, the stamp would end the selection halfway across the pane.
func TestPaintNestsWithoutEatingTheBar(t *testing.T) {
	env(t, "xterm-256color", "truecolor", "")
	pal := For(RungTrueColor)
	row := pal.paint(RoleSelection, " lzd              "+pal.paint(RoleMuted, "Oct 14 09:12")+"      ")
	if !strings.HasPrefix(row, "\x1b[7m") || !strings.HasSuffix(row, "\x1b[27m") {
		t.Fatalf("the bar is not the outermost run: %q", row)
	}
	if strings.Contains(row, "\x1b[0m") {
		t.Fatalf("a nested run closed with a full reset, which ends the bar: %q", row)
	}
	if got, want := ansi.Strip(row), " lzd              Oct 14 09:12      "; got != want {
		t.Fatalf("the nested run strips to %q, want %q", got, want)
	}
	if pal.paint(RoleMuted, "") != "" {
		t.Fatal("an empty run emitted escapes around nothing")
	}
}
