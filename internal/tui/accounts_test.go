package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/render"
)

// accountsShell is a shell with two accounts and a switch that answers with a second shell's worth
// of dependencies.
func accountsShell(t *testing.T) (*shell, *[]string) {
	t.Helper()
	s := newShell(t, 100, 30)
	switched := new([]string)
	s.m.deps.Accounts = []Account{{Name: "default"}, {Name: "work"}}
	s.m.deps.Active = "default"
	s.m.deps.SwitchAccount = func(name string) (Deps, error) {
		*switched = append(*switched, name)
		next := s.m.deps
		next.Active = name
		return next, nil
	}
	return s, switched
}

// TestTheSwitchKeyAsksForTheNextAccountAndStartsOver: `A` rebuilds the shell as the next account and
// issues that shell's opening request; the old frame stack is gone.
func TestTheSwitchKeyAsksForTheNextAccountAndStartsOver(t *testing.T) {
	s, switched := accountsShell(t)
	before := s.src.Calls()
	s.press("j")
	s.run(s.key("A"))
	if len(*switched) != 1 || (*switched)[0] != "work" {
		t.Fatalf("asked for %v", *switched)
	}
	if s.m.deps.Active != "work" {
		t.Fatalf("active = %q", s.m.deps.Active)
	}
	if len(s.m.frames) != 1 || s.m.top().Kind != frameHome {
		t.Fatalf("the stack was carried over: %d frames", len(s.m.frames))
	}
	if s.src.Calls() <= before {
		t.Fatal("the new shell issued no opening request")
	}
	if !strings.Contains(s.status(), "work") {
		t.Fatalf("the status line does not name the account: %q", s.status())
	}
}

// TestTheSwitchCyclesAndWraps: the second `A` comes back to the first.
func TestTheSwitchCyclesAndWraps(t *testing.T) {
	s, switched := accountsShell(t)
	s.press("A")
	s.press("A")
	if len(*switched) != 2 || (*switched)[0] != "work" || (*switched)[1] != "default" {
		t.Fatalf("asked for %v", *switched)
	}
}

// TestASingleAccountSaysSo: one account is not a switch, and the key spends nothing.
func TestASingleAccountSaysSo(t *testing.T) {
	s := newShell(t, 100, 30)
	before := s.src.Calls()
	s.press("A")
	if got := s.src.Calls(); got != before {
		t.Fatal("a single-account shell issued a request on A")
	}
	if !strings.Contains(s.status(), "no other account") {
		t.Fatalf("the status line says %q", s.status())
	}
}

// TestAFailedSwitchKeepsTheReaderWhereTheyWere: the store's refusal is a notice, and the pane still
// reads as the previous account.
func TestAFailedSwitchKeepsTheReaderWhereTheyWere(t *testing.T) {
	s, _ := accountsShell(t)
	s.m.deps.SwitchAccount = func(string) (Deps, error) { return Deps{}, errors.New("loose mode") }
	s.press("A")
	if s.m.deps.Active != "default" {
		t.Fatalf("active changed to %q after a failure", s.m.deps.Active)
	}
	if !strings.Contains(s.status(), "could not be read") {
		t.Fatalf("the status line says %q", s.status())
	}
}

// TestTheAccountNameIsNotOnACoveredLine: the name is the operator's own word, and a covered line
// carries no identity at all.
func TestTheAccountNameIsNotOnACoveredLine(t *testing.T) {
	s, _ := accountsShell(t)
	if got := s.m.accountField(); got != "default" {
		t.Fatalf("plain account field = %q", got)
	}
	s.m.state = StateImmersive
	if got := s.m.accountField(); got != "" {
		t.Fatalf("covered account field = %q, want empty", got)
	}
}

// TestTheSwitchKeepsTheDisplayState: a cover survives the switch, because it is about the terminal
// and not about the account.
func TestTheSwitchKeepsTheDisplayState(t *testing.T) {
	s, _ := accountsShell(t)
	s.m.state = StateImmersive
	s.m.style = render.CoverServiceLog
	s.press("A")
	if s.m.state != StateImmersive || s.m.style != render.CoverServiceLog {
		t.Fatalf("after the switch: %v / %v", s.m.state, s.m.style)
	}
}
