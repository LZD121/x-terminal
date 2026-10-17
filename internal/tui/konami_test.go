package tui

import (
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/keys"
	"github.com/LZD121/x-terminal/internal/render"
)

// TestKonamiSequence is the matcher's whole contract as one table: what each keystroke of a run
// means, when the one held key appears, and what a broken run does.
func TestKonamiSequence(t *testing.T) {
	type step struct {
		key  string
		want konamiEffect
	}
	run := func(keys ...step) {
		t.Helper()
		var k konami
		for i, s := range keys {
			var got konamiEffect
			k, got = k.advance(s.key)
			if got != s.want {
				t.Fatalf("key %d (%q): effect = %v, want %v", i, s.key, got, s.want)
			}
		}
	}
	arrows := []step{
		{"up", konamiNone}, {"up", konamiNone}, {"down", konamiNone}, {"down", konamiNone},
		{"left", konamiNone}, {"right", konamiNone}, {"left", konamiNone}, {"right", konamiNone},
	}

	t.Run("holds only the last key", func(t *testing.T) {
		run(append(append([]step{}, arrows...), step{"b", konamiHeld})...)
	})
	t.Run("the confirming key enters", func(t *testing.T) {
		run(append(append([]step{}, arrows...), step{"b", konamiHeld}, step{"a", konamiEnter})...)
	})
	t.Run("a broken confirmation replays the held key", func(t *testing.T) {
		run(append(append([]step{}, arrows...), step{"b", konamiHeld}, step{"c", konamiReplay})...)
	})
	t.Run("a lone b is bookmarks", func(t *testing.T) {
		run(step{"b", konamiNone})
	})
	t.Run("a broken run is silent", func(t *testing.T) {
		run(step{"up", konamiNone}, step{"x", konamiNone}, step{"down", konamiNone})
	})
	t.Run("the first key can restart a broken run", func(t *testing.T) {
		// up down up up: the third key restarts at key one and the fourth completes the pair.
		run(step{"up", konamiNone}, step{"down", konamiNone}, step{"up", konamiNone},
			step{"up", konamiNone}, step{"down", konamiNone})
	})
	t.Run("entering resets the matcher", func(t *testing.T) {
		var k konami
		for _, s := range append(append([]step{}, arrows...), step{"b", konamiHeld}) {
			k, _ = k.advance(s.key)
		}
		if k, eff := k.advance("a"); eff != konamiEnter || k.step != 0 {
			t.Fatalf("after entering: step = %d, effect = %v", k.step, eff)
		}
	})
}

// TestTheSequenceSpendsNothing is the reason the matcher sits in front of the keymap at all: the
// held key is bookmarks, and a run that never reaches `a` must leave the shell exactly where it
// found it — no frame pushed, no command issued, the same selection.
func TestTheSequenceSpendsNothing(t *testing.T) {
	s := newShell(t, 100, 30)
	before := s.m.top().ID
	s.press("up", "up", "down", "down", "left", "right", "left", "right")
	if s.m.state != StateNormal {
		t.Fatalf("eight arrows alone entered %v", s.m.state)
	}
	// The ninth key is held: no command, so the shell must not have moved.
	if cmd := s.key("b"); cmd != nil {
		t.Fatal("the held key asked for a command")
	}
	if s.m.state != StateNormal || s.m.top().ID != before {
		t.Fatalf("the held key did something: state = %v, frame = %s", s.m.state, s.m.top().ID)
	}
	// The confirming key enters. It asks for the display's own command — the window title, and
	// nothing that reaches the wire — which is the same command every deliberate transition asks
	// for.
	s.run(s.key("a"))
	if s.m.state != StateVivid {
		t.Fatalf("the sequence did not enter: state = %v", s.m.state)
	}
}

// TestABrokenSequenceStillOpensBookmarks is the other half: the withheld key is replayed as the
// action it always had, so a reader who types the eight arrows and then `b c` gets bookmarks and a
// next-cover step, exactly as if the matcher were not there.
func TestABrokenSequenceStillOpensBookmarks(t *testing.T) {
	s := newShell(t, 100, 30)
	s.press("up", "up", "down", "down", "left", "right", "left", "right")
	// The ninth key is held, not dispatched: bookmarks must not open on the way to anything.
	if cmd := s.key("b"); cmd != nil {
		t.Fatal("the held key opened bookmarks before the run was settled")
	}
	// A key that is not `a` settles it the other way: the held `b` is replayed as bookmarks, and
	// the breaking key is dispatched as itself — `c` is next-cover, and a cover is a state change.
	m, cmd := s.m.press(press("c"))
	s.m = m
	if s.m.state != StateImmersive {
		t.Fatalf("the breaking key was swallowed: state = %v", s.m.state)
	}
	if cmd == nil {
		t.Fatal("the replay asked for no command: bookmarks was not opened")
	}
}

// TestTheListingNeverMentionsTheSequence is the hidden half of the contract: no surface that lists
// the keymap says the presentation exists. The help overlay and the `keys` subcommand both read
// keys.Rows(), so the table is the one place to hold.
func TestTheListingNeverMentionsTheSequence(t *testing.T) {
	for _, row := range keys.Rows() {
		for _, key := range row.Keys {
			if key == "a" || key == "b" {
				continue // ordinary bindings; the sequence is not a row
			}
		}
	}
	// The table is the surface: a row for the presentation would be a documented way in.
	for _, row := range keys.Rows() {
		if strings.Contains(strings.ToLower(row.Help), "vivid") {
			t.Fatalf("the keymap documents the presentation: %q", row.Help)
		}
	}
	// And the help overlay, rendered, is the same table.
	overlay := render.Help(helpRows(false, "home"), 100, 40)
	if strings.Contains(strings.ToLower(strings.Join(overlay, "\n")), "vivid") {
		t.Fatal("the help overlay names the presentation")
	}
}
