// Package term is the module's only package that runs a program, and the only place the panic
// entry's bytes are built. It is a sink: the shell hands it values and reads nothing back.
//
// Why the entry carries the frame. The ordering table of the phase's UI-SPEC requires the frame's
// cells, then ED 3, then the title to be one write. Two facts about the pinned toolkit
// (charm.land/bubbletea/v2 v2.1.0) settle how that is possible, and both were read out of the
// module rather than assumed: the renderer parses the view's content into cells
// (cursed_renderer.go, `uv.NewStyledString(view.Content)` in the flush path), so an escape sequence
// appended to the content is consumed by the parser and never reaches the terminal; and the
// renderer's own write happens when its flush runs, while a command's result is delivered by the
// event loop — so no write of ours can be *guaranteed* to land after the frame's paint.
//
// The entry is therefore one self-contained byte string, written once: each frame row at its own
// absolute column 1, then ED 3, then the title. The panic state's View also returns the frame, so a
// repaint or a resize after the entry still shows it; that second paint writes byte-identical cells
// and is the price of a provable order.
//
// Two limitations are recorded here rather than faked. A terminal that ignores ED 3 keeps whatever
// was in its scrollback (there is no portable query, and a query would add a round trip to the frame
// that matters most); under a multiplexer ED 3 does nothing to the multiplexer's own history, which
// is why the clear is also a command. And this program does not enable tmux's server-wide
// `focus-events` option and does not read it: a user who wants focus-driven panic turns it on
// themselves, and `ctrl+p` covers the same need by hand.
package term

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Runner runs one program and returns its output. It is the seam the tests replace with a
// recorder, and it is deliberately the narrowest shape that covers the three tmux commands: a
// name, its arguments, and stdout.
type Runner func(name string, args ...string) ([]byte, error)

// Exec is the real Runner: one program, its arguments, and its stdout, with no shell in between,
// no timeout and no retry. A command that fails is a command that failed — the callers of this
// package swallow that rather than working around it.
func Exec(name string, args ...string) ([]byte, error) {
	return execCommand(name, args...)
}

// Sink is the terminal-side effect boundary: the output file the entry is written to, the
// environment the pane is read from, and the runner the tmux commands go through. The zero Sink is
// inert — no output file, no environment, no runner — which is what a test that cares about one
// seam gets for the other two.
type Sink struct {
	// Out is the program's own output file. The entry is one Write to it.
	Out io.Writer
	// Env reads the process environment. Only $TMUX_PANE is read, and only to decide whether there
	// is a multiplexer to work with.
	Env func(string) string
	// Run runs the tmux commands. Every failure it reports is swallowed.
	Run Runner
}

// New is the sink the composition root builds: the program's own output, the process environment,
// and the real process runner.
func New(out io.Writer) Sink {
	return Sink{Out: out, Env: os.Getenv, Run: Exec}
}

// eraseScrollback is ED 3, the sequence that deletes the lines a bare terminal has saved. It is the
// sequence Probe D measured; the library's own constant is used rather than a hand-spelled escape
// so the bytes live in one place.
const eraseScrollback = ansi.EraseEntireDisplay

// Entry builds the panic entry: one byte string whose own order is the frame's cells, then ED 3,
// then the title.
//
// Every row of the frame is written at its own absolute column 1 through `ansi.CursorPosition`, and
// there is no newline anywhere in the entry. Per-row positioning is deliberate on both counts: a
// row padded to the pane's full width leaves the terminal's wrap pending, and a trailing newline
// after the last row would scroll the frame's first row off the pane — which is the one thing a
// frame that is supposed to look like a finished command's output may not do. The frame arrives
// already padded to the pane's width by its own builder (internal/render.PanicFrame), so nothing
// here measures anything.
func Entry(frame, title string) string {
	var b strings.Builder
	for i, row := range strings.Split(frame, "\n") {
		b.WriteString(ansi.CursorPosition(1, i+1))
		b.WriteString(row)
	}
	b.WriteString(eraseScrollback)
	b.WriteString(ansi.SetWindowTitle(title))
	return b.String()
}

// EnterPanic is the panic entry: one write of Entry, then the multiplexer's own clear and the
// window name, in that order.
//
// Every error is swallowed. The frame is painted and ED 3 was written by the time a subprocess
// could fail, and printing an error would be a worse leak than the missing clear — it would put
// this program's own words on a screen that exists to say nothing about it. Nothing is returned:
// the shell is not told what happened, because there is nothing it could do about it and nothing on
// screen may claim otherwise.
func (s Sink) EnterPanic(frame, title string) {
	_, _ = s.out().Write([]byte(Entry(frame, title)))

	pane := Pane(s.env())
	if pane == "" {
		// There is no multiplexer history to clear and no window to rename. No subprocess is
		// started, nothing is reported, and this is the expected case in a bare terminal.
		return
	}
	s.call(ClearHistoryArgs(pane))
	s.call(RenameWindowArgs(pane, title))
}

// SetWindow renames the multiplexer's window to the name the state is titled with. It is the rename
// alone and it is a no-op without a pane: the window name follows the title in every state.
func (s Sink) SetWindow(name string) {
	pane := Pane(s.env())
	if pane == "" {
		return
	}
	s.call(RenameWindowArgs(pane, name))
}

// WindowName is the name to restore at exit, read once at start-up. It is trimmed of the newline
// the command prints, and it is empty on any failure — which the caller treats as "nothing to
// restore".
//
// Without a pane the name is not read at all: SetWindow is a no-op there, so the value could never
// be used, and the same branch that decides against the clear decides against this subprocess.
func (s Sink) WindowName() string {
	pane := Pane(s.env())
	if pane == "" || s.Run == nil {
		return ""
	}
	out, err := s.Run(tmuxProgram, WindowNameArgs()...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// out is the sink's output file. A sink built without one discards what it is asked to write
// rather than dereferencing nothing: the entry is still built and the multiplexer work still
// happens, which keeps every seam independent.
func (s Sink) out() io.Writer {
	if s.Out == nil {
		return io.Discard
	}
	return s.Out
}

// env is the sink's view of the environment.
func (s Sink) env() func(string) string {
	if s.Env == nil {
		return func(string) string { return "" }
	}
	return s.Env
}

// call runs one tmux command and throws its answer away.
func (s Sink) call(args []string) {
	if s.Run == nil {
		return
	}
	_, _ = s.Run(tmuxProgram, args...)
}
