package term

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/media"
)

// effects is a Sink's output file and its process runner in one value: what the sink writes and what
// it runs land in one log, in the order they happened. That log is the entry's order — no assertion
// about a screenshot or about a terminal can see it, and this is what can.
type effects struct {
	log []string
	// writes counts every Write call, whatever its size, because "one write" is the requirement and
	// a frame of one row and a frame of twenty-four must both be one.
	writes int
	// written is the bytes of each write, separately: TestEnterPanicIsOneWrite counts the calls and
	// the order tests read the one string.
	written []string

	runErr   error
	writeErr error
	// output is what the runner hands back, for the read-back command.
	output []byte
}

func (e *effects) Write(p []byte) (int, error) {
	e.writes++
	e.written = append(e.written, string(p))
	e.log = append(e.log, "write")
	if e.writeErr != nil {
		return 0, e.writeErr
	}
	return len(p), nil
}

func (e *effects) Run(name string, args ...string) ([]byte, error) {
	e.log = append(e.log, strings.Join(append([]string{name}, args...), " "))
	if e.runErr != nil {
		return nil, e.runErr
	}
	return e.output, nil
}

// env is an environment built from a map: an absent key is unset, which is the case the pane table
// has to cover separately from an empty value.
func env(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

// sink builds a Sink over one recorder, with the environment given.
func sink(e *effects, values map[string]string) Sink {
	return Sink{Out: e, Env: env(values), Run: e.Run}
}

// panicFrame is a frame shaped the way the real one is: rows padded to the pane's width, joined with
// newlines and with no trailing one. The rows carry no escape byte, because the real frame does not.
var panicFrame = strings.Join([]string{
	"ok      corp/ledger                     0.655s",
	"ok      corp/ledger/internal/reconcile  0.121s",
	"ok      corp/retry                      0.002s",
	strings.Repeat(" ", 44),
}, "\n")

// entryOrder reports whether an entry places every frame row's text — in order, each one after an
// absolute column-1 move for its own row — before the ED 3 sequence, and the ED 3 sequence before
// the title. It is a function rather than a block of assertions inside one test so that the swapped
// arm below can be held to the same rule the real entry is: an arm that only says "the real one is
// fine" tests nothing.
func entryOrder(entry string, rows []string, title string) bool {
	lastRow := -1
	for i, row := range rows {
		at := strings.Index(entry, ansi.CursorPosition(1, i+1)+row)
		if at < 0 || at <= lastRow {
			return false
		}
		lastRow = at
	}
	erase := strings.Index(entry, "\x1b[3J")
	if erase < 0 || erase <= lastRow {
		return false
	}
	osc := strings.Index(entry, ansi.SetWindowTitle(title))
	return osc > erase
}

// TestEntryOrder is the requirement itself: the entry's own bytes are the frame's cells, then ED 3,
// then the title, and each row sits at its own absolute column 1.
func TestEntryOrder(t *testing.T) {
	const title = "make"
	rows := strings.Split(panicFrame, "\n")
	entry := Entry(panicFrame, title)

	if !entryOrder(entry, rows, title) {
		t.Fatalf("the entry does not place every row, then ED 3, then the title:\n%q", entry)
	}
	if got := strings.Count(entry, "\x1b[3J"); got != 1 {
		t.Errorf("the entry carries ED 3 %d times, want once", got)
	}
	if got := strings.Count(entry, ansi.SetWindowTitle(title)); got != 1 {
		t.Errorf("the entry carries the title %d times, want once", got)
	}
	if strings.Contains(entry, "\n") {
		t.Errorf("the entry carries a newline: a row's own wrap is left pending and a trailing newline would scroll the frame")
	}
	for i := range rows {
		// Every row is written whole: its own text follows the absolute move to its own column 1.
		if !strings.Contains(entry, ansi.CursorPosition(1, i+1)+rows[i]) {
			t.Fatalf("row %d is not written at its own column 1:\n%q", i, entry)
		}
	}

	// The arm: the assertion above is the order, so an entry with the ED 3 sequence first — the
	// swap a careless implementation makes when it clears before painting — must not satisfy it.
	swapped := "\x1b[3J" + ansi.SetWindowTitle(title) + entry
	if entryOrder(swapped, rows, title) {
		t.Errorf("the order rule accepted an entry with ED 3 before the frame: it tests nothing")
	}
}

// TestEnterPanicIsOneWrite: whatever the frame's size, the write side of the entry is exactly one
// Write call. Two writes would mean the clear could land between the frame's rows.
func TestEnterPanicIsOneWrite(t *testing.T) {
	for _, rows := range []int{1, 3, 24} {
		frame := strings.Join(repeat("ok      corp/retry                      0.002s", rows), "\n")
		e := &effects{}
		sink(e, nil).EnterPanic(frame, "make")
		if e.writes != 1 {
			t.Errorf("a %d-row frame was written in %d writes, want one", rows, e.writes)
		}
		// The entry is Entry plus the image delete, which is the one thing this sink adds: a bare
		// terminal gets the bare sequence, and the frame's own bytes follow it unchanged.
		want := media.KittyDeleteAll() + Entry(frame, "make")
		if got := e.written[0]; got != want {
			t.Errorf("the write is not the entry's own bytes:\n got %q\nwant %q", got, want)
		}
	}
}

// TestEnterPanicOrderAcrossEffects is the sequence the UI-SPEC orders: the write, then the
// multiplexer's clear, then the window rename — all inside the one sink call the shell commands.
func TestEnterPanicOrderAcrossEffects(t *testing.T) {
	e := &effects{}
	sink(e, map[string]string{"TMUX_PANE": "%3"}).EnterPanic(panicFrame, "make")

	want := []string{
		"write",
		"tmux clear-history -t %3",
		"tmux rename-window -t %3 make",
	}
	if len(e.log) != len(want) {
		t.Fatalf("the entry ran %q, want %q", e.log, want)
	}
	for i := range want {
		if e.log[i] != want[i] {
			t.Fatalf("the entry ran %q, want %q", e.log, want)
		}
	}
}

// TestEnterPanicOrderHoldsHoweverTheRunnerAnswers: the clear is dispatched before the rename whatever
// the clear reports, because the order is the requirement and not a consequence of success.
func TestEnterPanicOrderHoldsHoweverTheRunnerAnswers(t *testing.T) {
	e := &effects{runErr: errors.New("no server running")}
	sink(e, map[string]string{"TMUX_PANE": "%3"}).EnterPanic(panicFrame, "make")

	clear := indexOf(e.log, "tmux clear-history -t %3")
	rename := indexOf(e.log, "tmux rename-window -t %3 make")
	if clear < 0 || rename < 0 || clear > rename {
		t.Errorf("a failing clear ran %q, want the clear before the rename", e.log)
	}
}

// TestPaneTable: only $TMUX_PANE decides whether there is multiplexer work, and an unset variable,
// an empty value and a pane id are the three cases. With no pane the runner is never called and the
// entry is still written — the bare-terminal leg, where ED 3 is the whole of the clear.
func TestPaneTable(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  map[string]string
		want []string
	}{
		{"unset", nil, []string{"write"}},
		{"empty", map[string]string{"TMUX_PANE": ""}, []string{"write"}},
		{"tmux set, pane empty", map[string]string{"TMUX": "/tmp/tmux-1000/default,123,0"}, []string{"write"}},
		{"pane", map[string]string{"TMUX_PANE": "%7"}, []string{
			"write", "tmux clear-history -t %7", "tmux rename-window -t %7 make",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Pane(env(tc.env)); got != tc.env["TMUX_PANE"] {
				t.Errorf("Pane read %q, want %q", got, tc.env["TMUX_PANE"])
			}
			e := &effects{}
			sink(e, tc.env).EnterPanic(panicFrame, "make")
			if e.writes != 1 {
				t.Errorf("the entry was written %d times with %s, want once", e.writes, tc.name)
			}
			if len(e.log) != len(tc.want) {
				t.Fatalf("with %s the entry ran %q, want %q", tc.name, e.log, tc.want)
			}
			for i := range tc.want {
				if e.log[i] != tc.want[i] {
					t.Fatalf("with %s the entry ran %q, want %q", tc.name, e.log, tc.want)
				}
			}
		})
	}
	if got := Pane(nil); got != "" {
		t.Errorf("Pane(nil) read %q, want empty", got)
	}
}

// TestSetWindowIsTheRenameAlone: the name follows the title in every state, on the pane and nowhere
// else, and a bare terminal runs nothing.
func TestSetWindowIsTheRenameAlone(t *testing.T) {
	e := &effects{}
	sink(e, map[string]string{"TMUX_PANE": "%3"}).SetWindow("svc-log")
	if len(e.log) != 1 || e.log[0] != "tmux rename-window -t %3 svc-log" {
		t.Errorf("SetWindow ran %q, want the rename alone", e.log)
	}

	bare := &effects{}
	sink(bare, nil).SetWindow("svc-log")
	if len(bare.log) != 0 {
		t.Errorf("SetWindow ran %q without a pane, want nothing", bare.log)
	}
}

// TestFailuresAreSwallowed: a runner that errors and a writer that errors both leave the sink with
// nothing to report and nothing printed. The frame was written and ED 3 was written before either
// failure could happen, and an error on screen would be a worse leak than the missing clear.
func TestFailuresAreSwallowed(t *testing.T) {
	e := &effects{runErr: errors.New("tmux: command not found")}
	sink(e, map[string]string{"TMUX_PANE": "%3"}).EnterPanic(panicFrame, "make")
	if e.log[0] != "write" {
		t.Errorf("a failing runner ran %q, want the write first", e.log)
	}

	broken := &effects{writeErr: errors.New("no space left on device")}
	sink(broken, map[string]string{"TMUX_PANE": "%3"}).EnterPanic(panicFrame, "make")
	if len(broken.log) != 3 {
		t.Errorf("a failing write ran %q, want the multiplexer work to have been attempted anyway", broken.log)
	}

	// The sink built with no output file at all: nothing to write to, and no panic either.
	Sink{}.EnterPanic(panicFrame, "make")
	if got := (Sink{}).WindowName(); got != "" {
		t.Errorf("a sink with no runner read the window name %q, want empty", got)
	}
	if got := sink(&effects{}, map[string]string{"TMUX_PANE": "%3"}).WindowName(); got != "" {
		t.Errorf("a failing display-message read %q, want empty", got)
	}
}

// TestWindowNameIsTrimmed: the command prints the name with a newline, and the name is what is left.
func TestWindowNameIsTrimmed(t *testing.T) {
	e := &effects{output: []byte("editor \n")}
	if got := sink(e, map[string]string{"TMUX_PANE": "%3"}).WindowName(); got != "editor" {
		t.Errorf("WindowName read %q, want %q", got, "editor")
	}
	if len(e.log) != 1 || e.log[0] != "tmux display-message -p #W" {
		t.Errorf("WindowName ran %q, want the read-back command alone", e.log)
	}

	// Without a pane the name is not read at all: there is no window this process renames, so the
	// value could never be used and the subprocess is not worth starting.
	bare := &effects{output: []byte("editor\n")}
	if got := sink(bare, nil).WindowName(); got != "" {
		t.Errorf("WindowName read %q without a pane, want empty", got)
	}
	if len(bare.log) != 0 {
		t.Errorf("WindowName ran %q without a pane, want nothing", bare.log)
	}
}

// TestCommandArguments: the three commands' argument vectors, each in one place, with an explicit
// pane. They are what the table-driven tests above assert and what the UI-SPEC names.
func TestCommandArguments(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  []string
		want []string
	}{
		{"clear-history", ClearHistoryArgs("%2"), []string{"clear-history", "-t", "%2"}},
		{"rename-window", RenameWindowArgs("%2", "git"), []string{"rename-window", "-t", "%2", "git"}},
		{"display-message", WindowNameArgs(), []string{"display-message", "-p", "#W"}},
	} {
		if strings.Join(tc.got, " ") != strings.Join(tc.want, " ") {
			t.Errorf("%s is %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// TestEraseScrollbackIsED3 pins the sequence itself: it is the byte string Probe D measured, and a
// constant that quietly became something else would make every other assertion here pass about the
// wrong bytes.
func TestEraseScrollbackIsED3(t *testing.T) {
	if eraseScrollback != "\x1b[3J" {
		t.Errorf("the clear is %q, want ESC [ 3 J", eraseScrollback)
	}
}

// repeat is n copies of s.
func repeat(s string, n int) []string {
	out := make([]string, 0, n)
	for range n {
		out = append(out, s)
	}
	return out
}

// indexOf is the first index of want in the log, or -1.
func indexOf(log []string, want string) int {
	for i, entry := range log {
		if entry == want {
			return i
		}
	}
	return -1
}

// TestPanicDeletesImagesBeforeTheScreen is the vivid presentation's extra promise: the terminal's own
// image store is emptied first, so the cover paints over cells that no longer have a picture in them.
func TestPanicDeletesImagesBeforeTheScreen(t *testing.T) {
	e := &effects{}
	sink(e, nil).EnterPanic(panicFrame, "make")
	got := e.written[0]
	del := strings.Index(got, media.KittyDeleteAll())
	first := strings.Index(got, ansi.CursorPosition(1, 1))
	ed3 := strings.Index(got, ansi.EraseEntireDisplay)
	if del < 0 {
		t.Fatal("the entry does not delete the image store")
	}
	if del > first || del > ed3 {
		t.Fatalf("the delete is not first: delete %d, first row %d, ED 3 %d", del, first, ed3)
	}
}

// TestPanicWrapsTheDeleteUnderTmux: under a multiplexer the same sequence is DCS-wrapped with its
// inner ESC doubled, or tmux eats it and the image survives the panic.
func TestPanicWrapsTheDeleteUnderTmux(t *testing.T) {
	e := &effects{}
	sink(e, map[string]string{"TMUX_PANE": "%3"}).EnterPanic(panicFrame, "make")
	got := e.written[0]
	if !strings.Contains(got, "\x1bPtmux;\x1b\x1b_Ga=d,d=A,q=2\x1b\x1b\\\x1b\\") {
		t.Fatalf("the delete was not wrapped and doubled: %q", got)
	}
	if strings.Count(got, "\x1bPtmux;") != 1 {
		t.Fatalf("%d DCS frames, want one", strings.Count(got, "\x1bPtmux;"))
	}
}

// TestClientTermNameAsksTheMultiplexer: inside tmux the process's TERM names the multiplexer, so the
// outer terminal's name is one command away — and a sink with no pane asks nothing.
func TestClientTermNameAsksTheMultiplexer(t *testing.T) {
	e := &effects{}
	e.output = []byte("xterm-ghostty\n")
	s := sink(e, map[string]string{"TMUX_PANE": "%3"})
	if got := s.ClientTermName(); got != "xterm-ghostty" {
		t.Fatalf("ClientTermName = %q", got)
	}
	// The program is tmux and the rest is the command's own argv: a runner called with the args and
	// no program would try to execute `display-message`, which is a different bug with the same
	// symptom (an empty answer and no picture).
	want := "tmux " + strings.Join(ClientTermNameArgs(), " ")
	found := false
	for _, entry := range e.log {
		if entry == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("the multiplexer was not asked with %q: %v", want, e.log)
	}
	// No pane: no subprocess, no name.
	e2 := &effects{}
	if got := sink(e2, nil).ClientTermName(); got != "" {
		t.Fatalf("a sink with no pane answered %q", got)
	}
	if len(e2.log) != 0 {
		t.Fatalf("a sink with no pane ran %v", e2.log)
	}
}
