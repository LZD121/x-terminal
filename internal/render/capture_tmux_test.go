package render

// This file is the in-tmux half of the environment specification: the frames the corpus stores for
// a multiplexer are written into a real tmux pane of the frame's own size, read back with
// `capture-pane`, and required to agree with the stored bytes.
//
// What the capture *is for* is the text: the pane is the environment the width model's other arm
// exists for, so the comparison is made on the characters the pane holds and on how many rows it
// took to hold them — the wrap points. Byte-equality of escape sequences is not achievable and not
// what this checks: tmux re-synthesises SGR runs when it captures a pane, and the frame the pane is
// given is the plain rung's (TERM=dumb, no colour at all), so the escape half of the specification
// stays where it belongs — the rung matrix's four frames and A1 over the corpus.
//
// One disagreement is *measured* here rather than smoothed away, and it is worth its own paragraph:
// tmux 3.7 counts a ZWJ family as one two-cell cluster — the grapheme's width — where the model's
// multiplexer arm counts the six cells of its code points (ansi.WcWidth). The pane therefore finds
// the row holding the family four cells narrower than the frame pads it, and the frame stops four
// cells short of the pane's right edge instead of wrapping. The error is in the model's direction
// that cannot wrap a row (it over-counts, so the frame is narrower than it thinks), the *text* and
// the row count agree exactly, and the harness reports the padding difference instead of hiding it:
// see the note in 02-05-SUMMARY.md.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/text"
)

// tmuxSocket is this run's own tmux server socket, and tmuxConf is the configuration
// it starts with: no status line, so the pane gets the whole window, and nothing
// else — the user's own tmux configuration never reaches into a frame's size. The
// server lives on its own socket so that the user's sessions, panes and key bindings
// are untouched, and so that a failing assertion cannot leave a session where the
// user would find it.
//
// The name carries the process that opened it — the pid and the nanosecond the
// package's tests started — so two runs on one machine (two checkouts, two CI shards,
// a parallel verification pass) get two servers. A fixed name was a machine-wide
// collision: whichever run finished second killed the first one's server with its
// `kill-server`, and the first then failed with "no server running" in the middle of
// a capture that had done nothing wrong.
var tmuxSocket = fmt.Sprintf("xterm-golden-%d-%d", os.Getpid(), time.Now().UnixNano())

// tmuxConf is the one setting the harness's server starts with: no status line, so
// every row of the window belongs to the frame.
const tmuxConf = "set-option -g status off\n"

// tmuxTimeout bounds one tmux command and one wait for the pane's own signal. A capture that hangs
// is a failure with a message, never a test that runs until the suite is killed.
const tmuxTimeout = 10 * time.Second

// TestCaptureTmux captures the stored multiplexer frames through real panes at their own sizes.
func TestCaptureTmux(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not on PATH (%v): the in-tmux capture needs it, and a silent skip would make a missing tmux look like an agreeing one", err)
	}
	// The cleanup is registered before anything is started, so a failure in any step below leaves no
	// server, no session and no pane behind.
	t.Cleanup(func() { tmuxKill(t) })

	conf := filepath.Join(t.TempDir(), "tmux.conf")
	if err := os.WriteFile(conf, []byte(tmuxConf), 0o600); err != nil {
		t.Fatalf("writing the harness's tmux configuration: %v", err)
	}

	for _, size := range []struct {
		name string
		w    int
	}{{"timeline_80", 80}, {"timeline_120", 120}} {
		t.Run(size.name, func(t *testing.T) {
			frame := storedTwin(t, fmt.Sprintf("%s%s", size.name, twinSuffix), size.w)
			t.Setenv("TMUX", envTmux.tmux) // the frame's own measure, which is the pane's semantics

			session := "xterm-golden-" + strconv.Itoa(size.w)
			t.Cleanup(func() { tmuxKill(t) })

			// The frame goes into the pane as a file the pane's command reads, without a trailing
			// newline: a trailing newline on the last row would scroll the pane and change what the
			// capture returns.
			path := filepath.Join(t.TempDir(), size.name+".frame")
			if err := os.WriteFile(path, []byte(frame.bytes), 0o600); err != nil {
				t.Fatalf("writing the frame for the pane: %v", err)
			}
			// The clear-screen and cursor-home come first because the pane's shell prints a line of
			// its own before the command: one line of it would cost the frame its first row.
			// The wait-for is the pane telling this test that the frame is fully painted — no sleep,
			// and nothing captured half-written.
			painted := session + "-painted"
			command := fmt.Sprintf("printf '\\033[2J\\033[H'; cat %s; tmux wait-for -S %s; exec sleep 30", path, painted)

			tmuxRun(t, "-f", conf, "new-session", "-d", "-s", session, "-x", strconv.Itoa(size.w), "-y", "24", command)
			tmuxRun(t, "wait-for", painted)
			captured := tmuxRun(t, "capture-pane", "-p", "-N", "-t", session)

			if problem := compareCapture(frame, captured); problem != "" {
				t.Errorf("the pane disagrees with the stored frame %s: %s", frame.name, problem)
			}
			assertPaneShape(t, frame, captured)
			assertSelectionIsLegible(t, frame, captured)

			// The comparator's own arm: a copy of the frame with one cell changed, and a copy with
			// one row removed, must each be reported as a disagreement. Without this, a comparator
			// that agreed with everything would pass this test and prove nothing.
			if problem := compareCapture(perturbedCell(frame), captured); problem == "" {
				t.Errorf("the comparator accepted the frame %s with one cell changed: it agrees with everything", frame.name)
			}
			if problem := compareCapture(perturbedRow(frame), captured); problem == "" {
				t.Errorf("the comparator accepted the frame %s with one row removed: it cannot see the pane's row count", frame.name)
			}

			// The measured disagreement is recorded, not swallowed: the row holding the ZWJ family
			// is padded by the pane's own width table, which is not the measure the program is built
			// on — so the two paddings are reported here, and the comparator above requires every
			// other row's padding to agree exactly.
			pane := strings.Split(strings.TrimSuffix(captured, "\n"), "\n")
			for i, row := range strings.Split(frame.bytes, "\n") {
				if !strings.Contains(row, "\u200d") {
					continue
				}
				t.Logf("the family row %d is %d cells of text; the pane pads it with %d spaces and the frame with %d: the pane's own table counts the cluster differently",
					i+1, text.Width(strings.TrimRight(ansi.Strip(row), " ")), trailingCells(ansi.Strip(pane[i])), trailingCells(ansi.Strip(row)))
			}
		})
	}

	// The sessions are gone: the arm the plan's verification greps for, made a check rather than an
	// expectation. (The server is on this test's own socket, so `tmux ls` on the user's default
	// socket cannot see it either way.)
	tmuxKill(t)
	if out, err := tmuxMaybe("ls"); err == nil {
		t.Errorf("the harness left a tmux server behind on its socket:\n%s", out)
	}
	// The other half of the cleanup, and the one the sweep counted: the socket *file* used to stay
	// behind (+1 per run, with no server on it), and a stale socket file is what a later client reads
	// as "server exited unexpectedly".
	for _, path := range tmuxSocketFiles(tmuxSocket) {
		if _, err := os.Lstat(path); err == nil {
			t.Errorf("the harness left its socket file behind at %s: a later run's client would find a socket no server answers on", path)
		}
	}
}

// TestTmuxSocketIsPrivateToThisRun is the harness's own specification: the socket is
// this run's — named for the process that opened it — and the cleanup kills that
// socket and no other. A second server on a socket this harness never names stands in
// for a concurrent run: it must still be answering after this run has killed its own.
// The fixed name this replaces is what the two runs used to share, and the name
// assertion below is what fails on that harness. The stand-in arm is the other half
// of the property, and it holds on any harness: it fails the moment a cleanup that
// means to reach only this run's socket reaches past it.
func TestTmuxSocketIsPrivateToThisRun(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not on PATH (%v): a socket's privacy is checked against a real server", err)
	}
	if tmuxSocket == "xterm-golden" || !strings.Contains(tmuxSocket, strconv.Itoa(os.Getpid())) {
		t.Fatalf("the harness socket %q does not name this run (pid %d): two runs would share one server and kill each other's",
			tmuxSocket, os.Getpid())
	}

	// The stand-in run's server: a socket of its own, derived from this run's so that
	// two runs' stand-in servers cannot collide either.
	other := tmuxSocket + "-stand-in"
	runOther := func(args ...string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), tmuxTimeout)
		defer cancel()
		out, err := exec.CommandContext(ctx, "tmux", append([]string{"-L", other}, args...)...).CombinedOutput()
		return string(out), err
	}
	t.Cleanup(func() {
		if _, err := runOther("kill-server"); err != nil {
			t.Logf("the stand-in server was already gone: %v", err)
		}
		removeTmuxSocket(t, other)
	})
	if out, err := runOther("new-session", "-d", "-s", "stand-in", "-x", "80", "-y", "24", "sh"); err != nil {
		t.Skipf("cannot start the stand-in server on its own socket (%v): %s", err, out)
	}

	// This run's server, started and killed the way the capture starts and kills it.
	t.Cleanup(func() { tmuxKill(t) })
	tmuxRun(t, "new-session", "-d", "-s", "socket-privacy", "-x", "80", "-y", "24", "sh")
	tmuxKill(t)

	if out, err := tmuxMaybe("ls"); err == nil {
		t.Errorf("this run's server survived its own kill-server:\n%s", out)
	}
	if out, err := runOther("ls"); err != nil {
		t.Fatalf("the stand-in server was killed by this run's kill-server: %v\n%s", err, out)
	}
}

// storedTwin reads the stored multiplexer frame for a name, failing if it was not stored: the twins
// are what the pane is compared against, and a missing one is a missing half of the specification.
func storedTwin(t *testing.T, name string, w int) storedFrame {
	t.Helper()
	base := strings.TrimSuffix(name, twinSuffix)
	for _, f := range corpus(t) {
		if f.name != base {
			continue
		}
		env := f.env
		env.tmux = envTmux.tmux
		return readFrame(t, name, w, f.h, env, f.state)
	}
	t.Fatalf("the corpus names no frame %q: the pane has nothing to be compared with", base)
	return storedFrame{}
}

// compareCapture compares the pane's text with the stored frame's, row for row, and returns the
// first disagreement or the empty string.
//
// The rules, in order: the pane holds exactly as many rows as the frame; every row's text is the
// frame's row's text (the padding and the escapes are not text); and the padding itself agrees on
// every row that does not hold a ZWJ cluster — a family is where this tmux's width table and the
// model's multiplexer measure part company, and a padding that disagrees anywhere else is a
// disagreement about the width model.
func compareCapture(frame storedFrame, captured string) string {
	want := strings.Split(frame.bytes, "\n")
	got := strings.Split(strings.TrimSuffix(captured, "\n"), "\n")
	if len(got) != len(want) {
		return fmt.Sprintf("the pane painted %d rows for a frame of %d: a row wrapped, or the frame did not fill the pane", len(got), len(want))
	}
	for i := range want {
		w, g := ansi.Strip(want[i]), ansi.Strip(got[i])
		if strings.TrimRight(w, " ") != strings.TrimRight(g, " ") {
			return fmt.Sprintf("row %d is %q in the pane and %q in the frame", i+1, strings.TrimRight(g, " "), strings.TrimRight(w, " "))
		}
		if strings.Contains(w, "\u200d") {
			// The pane's own width table is what pads this row: measured below, not asserted away.
			continue
		}
		if trailing := trailingCells(w); trailing != trailingCells(g) {
			return fmt.Sprintf("row %d is %q in the pane and %q in the frame: the padding disagrees by %d cells outside a ZWJ cluster",
				i+1, strings.TrimRight(g, " "), strings.TrimRight(w, " "), trailing-trailingCells(g))
		}
	}
	return ""
}

// trailingCells is how many trailing space cells a row carries: the row's own padding, which is
// what a disagreement about a cluster's width moves.
func trailingCells(row string) int {
	return text.Width(row) - text.Width(strings.TrimRight(row, " "))
}

// assertPaneShape is the frame's own geometry, checked against the pane: the stored frame is exactly
// the pane's size in cells, and the pane holds exactly the frame's rows — which is what catches a
// frame that did not fill the pane and a row that wrapped.
func assertPaneShape(t *testing.T, frame storedFrame, captured string) {
	t.Helper()
	rows := strings.Split(strings.TrimSuffix(captured, "\n"), "\n")
	if len(rows) != corpusH {
		t.Errorf("the pane holds %d rows, want the %d the frame is: the frame is the pane's whole height", len(rows), corpusH)
	}
	for i, row := range strings.Split(frame.bytes, "\n") {
		if got := text.Width(ansi.Strip(row)); got != frame.w {
			t.Errorf("the frame's row %d is %d cells of text, want the pane's %d: a frame that does not fill its pane is not the pane's frame", i+1, got, frame.w)
		}
	}
}

// assertSelectionIsLegible is the mark's half of the specification: the selected block's rows carry
// the selection in the captured text, on a terminal that was asked for no attributes at all.
func assertSelectionIsLegible(t *testing.T, frame storedFrame, captured string) {
	t.Helper()
	rows := strings.Split(strings.TrimSuffix(captured, "\n"), "\n")
	marked := 0
	for _, row := range rows {
		if strings.HasPrefix(row, MarkSelection) {
			marked++
		}
	}
	if marked == 0 {
		t.Errorf("no row of the pane carries the selection mark: the selection is carried by nothing in %s", frame.name)
	}
	// The status line is the pane's last row, always: the frame's own last row.
	if last, want := rows[len(rows)-1], strings.Split(frame.bytes, "\n")[corpusH-1]; strings.TrimRight(ansi.Strip(last), " ") != strings.TrimRight(ansi.Strip(want), " ") {
		t.Errorf("the pane's last row is %q, want the status line %q", last, want)
	}
}

// perturbedCell is the frame with the first cell of its first row changed: the smallest difference
// the comparator has to see. Every frame's first row begins with the selection column, so the cell
// is a space in one rung and a mark in another.
func perturbedCell(frame storedFrame) storedFrame {
	rows := strings.Split(frame.bytes, "\n")
	rows[0] = "~" + rows[0][1:]
	frame.bytes = strings.Join(rows, "\n")
	return frame
}

// perturbedRow is the frame with its second row removed: the pane still holds the row, so a
// comparison that did not count rows would miss it.
func perturbedRow(frame storedFrame) storedFrame {
	rows := strings.Split(frame.bytes, "\n")
	frame.bytes = strings.Join(append(rows[:1], rows[2:]...), "\n")
	return frame
}

// tmuxRun runs one tmux command on the harness's socket and returns its output, failing the test
// with the command line and the error when it does not exit cleanly.
func tmuxRun(t *testing.T, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), tmuxTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "tmux", append([]string{"-L", tmuxSocket}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tmux -L %s %s: %v\n%s", tmuxSocket, strings.Join(args, " "), err, out)
	}
	return string(out)
}

// tmuxMaybe runs one tmux command without failing, for the cleanup paths and the final check.
func tmuxMaybe(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), tmuxTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "tmux", append([]string{"-L", tmuxSocket}, args...)...).CombinedOutput()
	return string(out), err
}

// tmuxSocketFiles is where this run's socket lives, as a list of candidates: tmux puts `-L <name>`
// at `$TMUX_TMPDIR/tmux-<uid>/<name>` (with TMUX_TMPDIR's own default, which is the system's
// temporary directory). The candidates are tried in order and the first existing one is the socket.
func tmuxSocketFiles(name string) []string {
	dirs := []string{os.Getenv("TMUX_TMPDIR"), os.TempDir(), "/tmp"}
	out := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		out = append(out, filepath.Join(dir, fmt.Sprintf("tmux-%d", os.Getuid()), name))
	}
	return out
}

// removeTmuxSocket takes this run's socket file — and only this run's — out of the filesystem once no
// server is answering on it. The server removes its own socket when it exits normally, but a server
// taken down with `kill-server` from a fresh client leaves the file behind, and a *stale* socket file
// is what makes the next client report "server exited unexpectedly" instead of starting a server:
// that was the recycle this harness flaked on under load, and the leftover file is the leak the
// sweep counted (+1 per run).
func removeTmuxSocket(t *testing.T, name string) {
	t.Helper()
	for _, path := range tmuxSocketFiles(name) {
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSocket == 0 {
			// Not this harness's socket: a regular file of the same name is somebody else's and is
			// not ours to delete. Say so rather than passing silently.
			t.Logf("not removing %s: it is not a socket", path)
			continue
		}
		if err := os.Remove(path); err != nil {
			t.Errorf("removing the harness's socket %s: %v", path, err)
			continue
		}
	}
}

// tmuxKill kills the harness's server and takes its socket with it. It is idempotent: a server that
// is already gone is the state the caller wanted.
//
// The wait is the recycle's fix: `kill-server` returns as soon as the server has been told to exit,
// and a client that connects in the window between that and the server's own exit is answered with
// "server exited unexpectedly" — which is what a second leg of the same process saw under load. The
// loop waits for the socket to stop answering before the file goes, so the next command either finds
// a live server or no socket at all, and never a stale one.
func tmuxKill(t *testing.T) {
	t.Helper()
	if _, err := tmuxMaybe("kill-server"); err != nil {
		// No server on this socket is what every caller of this wants: it is not a failure. The
		// socket file may still be there, and is removed below like any other.
		t.Logf("kill-server on %s: %v", tmuxSocket, err)
	}
	deadline := time.Now().Add(tmuxTimeout)
	for {
		if _, err := tmuxMaybe("ls"); err != nil {
			// Not answering: either no server or no socket. Both are the state this wants.
			break
		}
		if time.Now().After(deadline) {
			t.Errorf("the harness's server on %s was still answering %s after kill-server: the recycle would race with it", tmuxSocket, tmuxTimeout)
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	removeTmuxSocket(t, tmuxSocket)
}
