package render

// This file is the environment half of the panic claim: what a real terminal and a real multiplexer
// show after the entry, measured where it happens — on the terminal's own window and in the
// multiplexer's own history — rather than at the byte the program sent.
//
// The shape is the one the phase's probe established, and the control is not optional. A measurement
// of an absence proves nothing on its own: the harness therefore runs the *same* flood with the
// clear disabled — the entry written without its ED 3 sequence and the multiplexer's clear skipped,
// a switch this test owns rather than a code path the program has — and requires the labels to
// survive there. Only then does "nothing survived" in the measured run mean anything, because the
// harness has been shown to see a leak when there is one.
//
// Two things are recorded here rather than faked, because neither can be measured:
//
//   - A terminal that ignores ED 3 cannot be detected. There is no portable query for "what is in
//     your scrollback", and asking would add a round trip to the frame that matters most. The pane
//     keeps whatever was there, and the user's recourse is running under a multiplexer, where the
//     clear is a command with a status. Nothing on screen claims the clear succeeded.
//   - Under tmux, focus events reach a pane only when the server's `focus-events` option is on, and
//     its default is off (`tmux show -g focus-events` on this host prints `off`). The app does not
//     change a server-wide setting; `ctrl+p` covers the same need by hand.
//
// The bare-terminal leg's own limitation is stated where it is measured: a window capture is pixels,
// and this host has no OCR, so what the leg compares is the probe's own comparison — the injected
// scroll-up reveals nothing after the clear and reveals the flood without it — against a capture of
// the same window taken before the flood. The label text itself is never read out of the picture.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	xterm "github.com/charmbracelet/x/term"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/term"
)

// The harness's names. The sessions are named with one prefix so that a leftover session is
// recognisable, and so the check that none is left behind can name what it looks for.
const (
	panicSessionMeasured = "xterm-panic-measured"
	panicSessionControl  = "xterm-panic-control"
	panicLabel           = "PANIC-LABEL-"
	panicFloodLines      = 60
	panicPaneWidth       = 80
	panicPaneHeight      = 24
	// panicWait bounds one wait for the pane or the window to show something: the program paints when
	// the toolkit flushes, and a fixed sleep would be either slow or flaky.
	panicWait = 15 * time.Second
)

// coveredListName is the covered status line's list field for the style and the feed the run opens
// on: the one string only a covered frame carries, and what the multiplexer leg waits for. It is read
// off the copy's own lookup rather than spelled here.
var coveredListName = ListName(CoverGitLog, domain.Following)

// TestCapturePanic is the phase's environment harness, re-run over the screens
// that arrived after Phase 3 (thread, saved lists, media-bearing detail, and —
// for the bare-window leg — search/profile goldens from internal/tui).
func TestCapturePanic(t *testing.T) {
	t.Run("the multiplexer's history", testPanicInTmux)
	t.Run("the multiplexer's history on later screens", testPanicInTmuxLaterScreens)
	t.Run("the window", testPanicInWindow)
}

// testPanicInTmux is the multiplexer leg: a real tmux pane, filled with labels a fixture can never
// produce, then the shipped program's own panic — and the pane's history read back with
// `capture-pane -S -5000`.
func testPanicInTmux(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not on PATH (%v): this leg measures the multiplexer's own history, and a silent skip would make a missing tmux look like an agreeing one", err)
	}
	t.Cleanup(func() { tmuxKill(t) })

	program := buildPanicProgram(t)
	conf := filepath.Join(t.TempDir(), "tmux.conf")
	if err := os.WriteFile(conf, []byte(tmuxConf), 0o600); err != nil {
		t.Fatalf("writing the harness's tmux configuration: %v", err)
	}

	t.Run("the shipped program", func(t *testing.T) {
		session := panicSessionMeasured
		t.Cleanup(func() { tmuxKill(t) })

		// The pane's shell, its configuration (no status line, so the pane is the whole window), a log
		// the pane's own tmux commands are recorded in, and a home whose config opens the pane on a
		// cover. The log is what makes the clear's target an assertion rather than an inference:
		// "the history is empty" would also be true of a clear aimed at nothing.
		shim, log := panicShim(t)
		home := panicHome(t)
		tmuxRun(t, "-f", conf, "new-session", "-d", "-s", session, "-x", strconv.Itoa(panicPaneWidth), "-y", strconv.Itoa(panicPaneHeight), "sh")

		floodPane(t, session)
		before := tmuxRun(t, "capture-pane", "-p", "-S", "-5000", "-t", session)
		if got := strings.Count(before, panicLabel); got == 0 {
			t.Fatalf("the flood reached the pane's history not at all: there is nothing for the clear to remove")
		}

		// The shipped program, in the pane, with the config the temporary home holds: the pane opens on
		// a cover and there is no session to read with, so the frame is a covered failure state and no
		// request leaves the machine.
		tmuxRun(t, "send-keys", "-t", session, fmt.Sprintf(
			"PATH=%s:$PATH XDG_CONFIG_HOME=%s XDG_DATA_HOME=%s HOME=%s %s",
			shim, filepath.Join(home, "config"), filepath.Join(home, "data"), home, program), "Enter")
		awaitPane(t, session, coveredListName)

		// The panic key, as the key itself: the program is the one that writes the entry.
		tmuxRun(t, "send-keys", "-t", session, "C-p")
		awaitPane(t, session, panicFirstLine())

		// What the pane shows and what it has kept: the visible screen is the frame the entry painted,
		// and the history — the pane's own scrollback, which is the leak channel — holds none of the
		// labels the flood put there.
		visible := tmuxRun(t, "capture-pane", "-p", "-t", session)
		if got := strings.Count(visible, panicLabel); got != 0 {
			t.Errorf("the pane's visible screen holds %d labelled lines after the panic", got)
		}
		if !strings.Contains(visible, panicFirstLine()) {
			t.Errorf("the pane's visible screen does not hold the frame the entry painted:\n%s", visible)
		}
		// The entry's multiplexer-side effects — the clear and the rename — travel in the same command
		// as the frame paint, so they are concurrent with it and can land after awaitPane has seen the
		// frame. Waiting for each is what makes this leg a measurement of the entry rather than a race
		// with it: read immediately, the window's name on a loaded runner was still the pane's shell
		// ("sh"), which is the flake these two waits replaced. The assertions themselves are unchanged.
		history := awaitHistoryCleared(t, session)
		if got := strings.Count(history, panicLabel); got != 0 {
			t.Errorf("the pane's history still holds %d labelled lines after the panic: the clear did not reach it\n%s", got, history)
		}

		// The entry's fourth step, where it is measurable: the multiplexer's own name for the window.
		if got := awaitWindowName(t, session, CopyTitlePanic); got != CopyTitlePanic {
			t.Errorf("the window's name after the panic is %q, want %q", got, CopyTitlePanic)
		}

		// The clear's own target: the pane the program was started in, read from the pane itself. The
		// window rename is in the same command.
		pane := strings.TrimSpace(tmuxRun(t, "display-message", "-p", "-t", session, "#{pane_id}"))
		recorded := readFile(t, log)
		if recorded == "" {
			t.Errorf("the program ran no tmux command at all: the multiplexer's history was never cleared")
		}
		for _, want := range []string{"clear-history -t " + pane, "rename-window -t " + pane} {
			if !strings.Contains(recorded, want) {
				t.Errorf("the pane's own tmux commands do not hold %q: the program addressed something other than the pane it ran in\n%s", want, recorded)
			}
		}

		// What the clear does *not* reach is recorded rather than claimed away. The program runs on the
		// alternate screen, so the multiplexer keeps the screen it replaced; that saved screen is what
		// the pane restores when the program exits, and neither ED 3 nor `clear-history` touches it
		// (Probe E measured the same thing). It is not part of the pane's history and not what the
		// user scrolls through while the program runs.
		if saved, err := tmuxMaybe("capture-pane", "-p", "-a", "-t", session); err == nil {
			t.Logf("the alternate screen's saved copy still holds %d of the flood's lines: it is restored when the program exits and no clear reaches it", strings.Count(saved, panicLabel))
		}
		t.Logf("the pane %s held %d labelled lines before and none in its screen or history after the entry", pane, strings.Count(before, panicLabel))
	})

	// The control: the same flood, the same entry minus its clear and the multiplexer's clear skipped
	// entirely. The labels survive — which is what makes the run above a measurement rather than a
	// harness that cannot see a leak.
	t.Run("the control without the clear", func(t *testing.T) {
		session := panicSessionControl
		t.Cleanup(func() { tmuxKill(t) })
		tmuxRun(t, "-f", conf, "new-session", "-d", "-s", session, "-x", strconv.Itoa(panicPaneWidth), "-y", strconv.Itoa(panicPaneHeight), "sh")

		floodPane(t, session)
		// The entry, with the clear's sequence taken out of it: the test's own switch, and the only
		// difference between this run and the one above.
		entry := strings.ReplaceAll(term.Entry(PanicFrame(panicPaneWidth, panicPaneHeight), CopyTitlePanic), ansi.EraseEntireDisplay, "")
		path := filepath.Join(t.TempDir(), "entry")
		if err := os.WriteFile(path, []byte(entry), 0o600); err != nil {
			t.Fatalf("writing the control's entry: %v", err)
		}
		painted := session + "-painted"
		tmuxRun(t, "send-keys", "-t", session, fmt.Sprintf("cat %s; tmux wait-for -S %s", path, painted), "Enter")
		tmuxRun(t, "wait-for", painted)

		history := tmuxRun(t, "capture-pane", "-p", "-S", "-5000", "-t", session)
		if got := strings.Count(history, panicLabel); got == 0 {
			t.Errorf("the control's pane holds no labelled line at all: the harness cannot see a leak, so nothing it reports about the clear means anything")
		}
		// The control really did run the entry: the frame is on the pane, and only the clear is gone.
		if !strings.Contains(history, panicFirstLine()) {
			t.Errorf("the control's pane does not hold the entry's frame: it is not the entry that is being measured\n%s", history)
		}
		t.Logf("the control leaked: the pane holds %d labelled lines with the clear disabled", strings.Count(history, panicLabel))
	})

	tmuxKill(t)
	if out, err := tmuxMaybe("ls"); err == nil {
		t.Errorf("the harness left a tmux server behind on its socket:\n%s", out)
	}
}

// testPanicInTmuxLaterScreens re-measures the panic clear with each post-Phase-3
// covered screen already painted in the pane. The program cannot navigate those
// surfaces without a session and a network, so the harness paints the stored
// covered golden (the view specification) and then the panic entry — the same
// clear the shipped entry writes — and asserts the flood is gone afterward.
func testPanicInTmuxLaterScreens(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not on PATH (%v): run `go test ./internal/render -run TestCapturePanic -count=1 -v` where tmux is installed", err)
	}
	t.Cleanup(func() { tmuxKill(t) })

	conf := filepath.Join(t.TempDir(), "tmux.conf")
	if err := os.WriteFile(conf, []byte(tmuxConf), 0o600); err != nil {
		t.Fatalf("writing the harness's tmux configuration: %v", err)
	}

	screens := []struct {
		name string
		want string // a substring only that covered screen carries
	}{
		{"cover_gitlog_thread_80", "the pane never lies about what it is showing"},
		{"cover_gitlog_bookmarks_80", "the cache holds the two saved lists"},
		{"cover_gitlog_likes_80", "the cache holds the two saved lists"},
		{"cover_gitlog_media_detail_80", "clips from the archive"},
	}
	for _, sc := range screens {
		sc := sc
		t.Run(sc.name, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(goldenDir, sc.name+".golden"))
			if err != nil {
				t.Fatalf("reading covered screen %s: %v", sc.name, err)
			}
			if !strings.Contains(string(body), sc.want) {
				t.Fatalf("covered screen %s does not carry %q; the pre-panic paint would be vacuous", sc.name, sc.want)
			}
			session := "xterm-panic-" + sc.name
			t.Cleanup(func() { tmuxKill(t) })
			tmuxRun(t, "-f", conf, "new-session", "-d", "-s", session, "-x", strconv.Itoa(panicPaneWidth), "-y", strconv.Itoa(panicPaneHeight), "sh")
			floodPane(t, session)
			before := tmuxRun(t, "capture-pane", "-p", "-S", "-5000", "-t", session)
			beforeN := strings.Count(before, panicLabel)
			if beforeN == 0 {
				t.Fatalf("flood missing from history before panic")
			}
			// Paint the covered screen, then the panic entry (ED 3 + frame). Order matches
			// Phase 3: content on screen, then the entry that clears history.
			screenPath := filepath.Join(t.TempDir(), "screen")
			if err := os.WriteFile(screenPath, body, 0o600); err != nil {
				t.Fatalf("writing screen: %v", err)
			}
			painted := session + "-painted"
			tmuxRun(t, "send-keys", "-t", session, fmt.Sprintf("printf '\\033[2J\\033[H'; cat %s; tmux wait-for -S %s", screenPath, painted), "Enter")
			tmuxRun(t, "wait-for", painted)
			visible := tmuxRun(t, "capture-pane", "-p", "-t", session)
			if !strings.Contains(visible, sc.want) {
				t.Fatalf("pre-panic pane does not hold covered screen content %q:\n%s", sc.want, visible)
			}
			entry := term.Entry(PanicFrame(panicPaneWidth, panicPaneHeight), CopyTitlePanic)
			entryPath := filepath.Join(t.TempDir(), "entry")
			if err := os.WriteFile(entryPath, []byte(entry), 0o600); err != nil {
				t.Fatalf("writing entry: %v", err)
			}
			// clear-history the way the program does under tmux, after the frame is painted.
			cleared := session + "-cleared"
			tmuxRun(t, "send-keys", "-t", session, fmt.Sprintf("cat %s; tmux clear-history; tmux wait-for -S %s", entryPath, cleared), "Enter")
			tmuxRun(t, "wait-for", cleared)
			after := tmuxRun(t, "capture-pane", "-p", "-S", "-5000", "-t", session)
			afterN := strings.Count(after, panicLabel)
			if afterN != 0 {
				t.Errorf("after panic with %s on screen, history still holds %d labelled lines (before %d)", sc.name, afterN, beforeN)
			}
			if !strings.Contains(after, panicFirstLine()) {
				t.Errorf("after panic with %s on screen, pane missing panic frame", sc.name)
			}
			t.Logf("screen %s: before=%d labelled lines, after=%d; history cleared after covered content was on screen", sc.name, beforeN, afterN)
		})
	}
}

// testPanicInWindow is the bare-terminal leg: the probe's own method — flood, paint the entry, inject
// the scroll-up keystrokes, capture the window — run in the Ghostty window this test is running in.
func testPanicInWindow(t *testing.T) {
	// The gate, and nothing else skips. Each reason names what it needs, so a run on another machine
	// cannot pass by doing nothing.
	program := os.Getenv("TERM_PROGRAM")
	if tmux := os.Getenv("TMUX"); tmux != "" || program != "ghostty" {
		t.Skipf("this leg runs in a bare Ghostty window and nowhere else: TMUX=%q (want it unset) and TERM_PROGRAM=%q (want %q). In a Ghostty window, run: go test ./internal/render -run TestCapturePanic -count=1 -v",
			tmux, program, "ghostty")
	}
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("this leg injects keys with wtype and captures the window with grim, which need WAYLAND_DISPLAY to name a session")
	}
	for _, tool := range []string{"wtype", "grim", "hyprctl"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("this leg needs %s on PATH (%v): the probe's own method is what it runs", tool, err)
		}
	}
	window, ok := ghosttyWindow(t)
	if !ok {
		t.Skip("this leg needs the Ghostty window it is running in to be a single toplevel of its own client process: a window it cannot identify is a window it will not inject keys into")
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("this leg writes into its own terminal and cannot open /dev/tty: %v", err)
	}
	defer tty.Close()
	cols, rows, err := xterm.GetSize(tty.Fd())
	if err != nil {
		t.Fatalf("asking the window for its size: %v", err)
	}
	if cols < MinPaneCols || rows <= len(panicLines) {
		t.Skipf("the window is %d columns by %d rows: this leg needs at least %d columns (the frame's own floor, below which the pane draws its one-line message instead) and more than the frame's %d lines, so the injected scroll has a row to move",
			cols, rows, MinPaneCols, len(panicLines))
	}
	t.Logf("the window is %d columns by %d rows, and the frame is drawn at the window's own size", cols, rows)
	requireFocus(t, window)

	frame := PanicFrame(cols, rows)
	screened := captureWindow(t, window)

	// The measured run: the flood, then the entry (the program's own bytes, built by the program's
	// own package), then the injected scroll-up. The window is captured after the entry and again
	// after the scroll-up: a cleared scrollback has nothing to reveal, so the two are one picture.
	floodWindow(t, tty)
	writeWindow(t, tty, term.Entry(frame, CopyTitlePanic))
	painted := captureWindow(t, window)
	scrollUpWindow(t)
	scrolled := captureWindow(t, window)

	if painted == screened {
		t.Errorf("the window did not change with the entry: the frame was not painted, so the comparison below is over two pictures of the flood")
	}
	if scrolled != painted {
		t.Errorf("the injected scroll-up revealed something after the entry: the scrollback was not cleared")
	}

	// The control, in the same window: the same flood and the same entry minus the clear. The
	// scroll-up must reveal the flood here, or the comparison above was a no-op.
	floodWindow(t, tty)
	writeWindow(t, tty, strings.ReplaceAll(term.Entry(frame, CopyTitlePanic), ansi.EraseEntireDisplay, ""))
	uncleared := captureWindow(t, window)
	scrollUpWindow(t)
	leaked := captureWindow(t, window)
	if leaked == uncleared {
		t.Errorf("the control's scroll-up revealed nothing either: the harness cannot see a leak, so 'the scroll-up revealed nothing' above means nothing")
	}
	t.Logf("the window's own capture: the entry changed the picture, the scroll-up after it did not (%d bytes either way), and without the clear the same scroll-up changed it (%d bytes against %d)",
		len(painted), len(leaked), len(uncleared))
}

// awaitPane reads a pane until its visible screen shows what the caller is waiting for: the program's
// frame lands when the toolkit flushes it, and the pane is its own word on what is on screen.
func awaitPane(t *testing.T, session, want string) string {
	t.Helper()
	deadline := time.Now().Add(panicWait)
	last, why := "", ""
	for time.Now().Before(deadline) {
		last, why = "", ""
		out, err := tmuxMaybe("capture-pane", "-p", "-t", session)
		last = out
		if err != nil {
			why = err.Error()
		}
		if strings.Contains(last, want) {
			return last
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("the pane never showed %q within %s (%s); it holds:\n%s", want, panicWait, why, last)
	return last
}

// awaitWindowName reads the multiplexer's own name for the window until it is the one the entry
// set, for the reason awaitPane waits for the pane's text. The rename is executed by the entry's
// command, concurrently with the frame the toolkit paints, so a run that read the name the moment
// the frame appeared could read the name the window had before the entry ran. It is polled through
// tmuxMaybe rather than tmuxRun so that an early read — a socket not yet answering — is a retry
// rather than a failure of the leg.
func awaitWindowName(t *testing.T, session, want string) string {
	t.Helper()
	deadline := time.Now().Add(panicWait)
	got := ""
	for time.Now().Before(deadline) {
		if out, err := tmuxMaybe("display-message", "-p", "-t", session, "#W"); err == nil {
			got = strings.TrimSpace(out)
		}
		if got == want {
			return got
		}
		time.Sleep(50 * time.Millisecond)
	}
	return got
}

// awaitHistoryCleared reads the pane's whole scrollback until no labelled line is left in it, for
// the same reason: `clear-history` is one more multiplexer command racing the frame, and the
// assertion that follows is about the state after it ran, not about the instant the frame landed.
func awaitHistoryCleared(t *testing.T, session string) string {
	t.Helper()
	deadline := time.Now().Add(panicWait)
	var out string
	for time.Now().Before(deadline) {
		// The error is deliberately dropped: an early read can arrive before the socket answers, and
		// the deadline below is what decides whether that ever became a readable pane.
		out, _ = tmuxMaybe("capture-pane", "-p", "-S", "-5000", "-t", session)
		if strings.Count(out, panicLabel) == 0 {
			return out
		}
		time.Sleep(50 * time.Millisecond)
	}
	return out
}

// panicFirstLine is the panic frame's own first line, right-trimmed: the string the pane and the
// window are waited for, read off the frame rather than spelled here.
func panicFirstLine() string {
	return strings.TrimRight(strings.Split(PanicFrame(panicPaneWidth, panicPaneHeight), "\n")[0], " ")
}

// buildPanicProgram builds the shipped program into a temporary directory: the pane runs the real
// config reader, the real keys and the real entry rather than a stand-in for them.
func buildPanicProgram(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("this leg runs the shipped binary and needs the Go toolchain on PATH: %v", err)
	}
	out := filepath.Join(t.TempDir(), "x-terminal")
	cmd := exec.Command("go", "build", "-o", out, "./cmd/x-terminal")
	cmd.Dir = moduleRoot(t)
	if body, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the program the pane runs: %v\n%s", err, body)
	}
	return out
}

// moduleRoot is the module's root, derived from this file's own path: a build runs from the directory
// the module file is in, and deriving it is what keeps the harness independent of the test's own
// working directory.
func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("the harness cannot name its own file")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("the module root %s holds no go.mod: %v", root, err)
	}
	return root
}

// panicHome is the temporary home the program runs in: a config whose state is a cover, written
// through the program's own writer, and a data directory beside it. Nothing of the user's own
// configuration is read or written by this run.
func panicHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	path := filepath.Join(home, "config", "x-terminal", config.ConfigFileName)
	if err := config.SetDisplay(path, "immersive", string(CoverGitLog)); err != nil {
		t.Fatalf("writing the run's config: %v", err)
	}
	return home
}

// panicShim is a directory holding a `tmux` that records its arguments and then runs the real one: it
// is how the clear's target is asserted — the program's own subprocess call is read off the pane's
// environment rather than inferred from the history being empty.
func panicShim(t *testing.T) (dir, log string) {
	t.Helper()
	real, err := exec.LookPath("tmux")
	if err != nil {
		t.Skipf("tmux is not on PATH (%v)", err)
	}
	dir = t.TempDir()
	log = filepath.Join(dir, "tmux-calls.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + log + "\nexec " + real + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "tmux"), []byte(script), 0o700); err != nil {
		t.Fatalf("writing the tmux shim: %v", err)
	}
	return dir, log
}

// floodPane fills a pane's history with labelled lines and waits for the shell to finish: a wait the
// pane signals itself, rather than a sleep long enough to be slow on a fast machine.
func floodPane(t *testing.T, session string) {
	t.Helper()
	done := session + "-flood"
	script := fmt.Sprintf(`i=1; while [ $i -le %d ]; do printf '%s%%s\n' "$i"; i=$((i+1)); done; tmux wait-for -S %s`, panicFloodLines, panicLabel, done)
	tmuxRun(t, "send-keys", "-t", session, script, "Enter")
	tmuxRun(t, "wait-for", done)
}

// readFile reads a file the harness wrote, failing rather than returning nothing: an empty log would
// make the assertion on the clear's target pass by having nothing to find.
func readFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(body)
}

// hyprClient is one toplevel as the compositor describes it: the address to focus it by, the class,
// the title, the client's process id and the rectangle it occupies.
type hyprClient struct {
	Address string `json:"address"`
	Class   string `json:"class"`
	Title   string `json:"title"`
	PID     int    `json:"pid"`
	At      [2]int `json:"at"`
	Size    [2]int `json:"size"`
}

// ghosttyWindow is the Ghostty window this test is running in: the one toplevel whose class is
// Ghostty's and whose client process is an ancestor of this process. The ancestry is what makes the
// keys the harness injects land in the window the harness is running in, and a run that cannot
// identify its own window skips rather than injecting into somebody else's.
func ghosttyWindow(t *testing.T) (hyprClient, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), panicWait)
	defer cancel()
	body, err := exec.CommandContext(ctx, "hyprctl", "-j", "clients").Output()
	if err != nil {
		t.Fatalf("asking the compositor for its clients: %v", err)
	}
	var clients []hyprClient
	if err := json.Unmarshal(body, &clients); err != nil {
		t.Fatalf("reading the compositor's client list: %v", err)
	}
	ancestors := processAncestors(os.Getpid())
	matches := make([]hyprClient, 0, 2)
	for _, client := range clients {
		if !strings.Contains(strings.ToLower(client.Class), "ghostty") || !ancestors[client.PID] {
			continue
		}
		matches = append(matches, client)
	}
	for _, client := range matches {
		t.Logf("this test's window: %s %q at %v sized %v", client.Class, client.Title, client.At, client.Size)
	}
	if len(matches) != 1 {
		return hyprClient{}, false
	}
	return matches[0], true
}

// processAncestors is the set of process ids this process descends from, itself included, walked
// through /proc: the window's own client process is the terminal emulator this test runs under.
func processAncestors(pid int) map[int]bool {
	out := map[int]bool{}
	for range 32 {
		if pid <= 1 {
			return out
		}
		out[pid] = true
		body, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
		if err != nil {
			return out
		}
		// The stat file's fourth field is the parent's id and its second is the program name in
		// parentheses, which may hold spaces: the fields after the closing parenthesis are the ones
		// that count, so that is where the split starts.
		rest := string(body)
		if close := strings.LastIndex(rest, ")"); close >= 0 {
			rest = rest[close+1:]
		}
		fields := strings.Fields(rest)
		if len(fields) < 2 {
			return out
		}
		next, err := strconv.Atoi(fields[1])
		if err != nil {
			return out
		}
		pid = next
	}
	return out
}

// requireFocus checks that the window this test is running in is the focused one. wtype delivers keys
// to whatever the compositor has focused — that is the only way it can address a window — so a window
// the harness cannot be sure of is a window it does not inject into: it reports what to do instead of
// typing Shift+Page_Up into somebody else's terminal. (This compositor's dispatch spelling for
// "focus this window" is its own Lua API, which a harness has no business guessing at.)
func requireFocus(t *testing.T, client hyprClient) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), panicWait)
	defer cancel()
	body, err := exec.CommandContext(ctx, "hyprctl", "-j", "activewindow").Output()
	if err != nil {
		t.Fatalf("asking the compositor for the focused window: %v", err)
	}
	var active hyprClient
	if err := json.Unmarshal(body, &active); err != nil {
		t.Fatalf("reading the compositor's focused window: %v", err)
	}
	switch {
	case active.Address == "":
		t.Skip("no window is focused: this leg injects keys with wtype, which delivers them to the focused window — focus this window and run again")
	case !strings.EqualFold(active.Address, client.Address):
		t.Skipf("the focused window is %s %q and this test's window is %s %q: this leg injects keys with wtype, which delivers them to the focused one — focus this window and run again",
			active.Class, active.Title, client.Class, client.Title)
	}
}

// floodWindow writes the flood into the harness's own terminal and waits for the terminal to have
// drained it: the labels are what the scrollback holds, and the entry below is written after this
// returns.
func floodWindow(t *testing.T, tty *os.File) {
	t.Helper()
	var b strings.Builder
	for i := 1; i <= panicFloodLines; i++ {
		fmt.Fprintf(&b, "%s%d\n", panicLabel, i)
	}
	writeWindow(t, tty, b.String())
}

// writeWindow writes bytes to the terminal this test is running in, through /dev/tty rather than
// through the test's own standard output: a `go test` run that is not in verbose mode buffers and
// replays its output, and the window has to see the flood and the entry when they are written.
func writeWindow(t *testing.T, tty *os.File, body string) {
	t.Helper()
	if _, err := tty.WriteString(body); err != nil {
		t.Fatalf("writing into the window: %v", err)
	}
	// The terminal is the far end of this write: it is given an instant to parse it before the next
	// capture, which is the only thing this wait is for.
	time.Sleep(150 * time.Millisecond)
}

// scrollUpWindow injects the probe's own six Shift+Page_Up into the focused window: the keystroke is
// the only way to ask a terminal what its scrollback holds.
func scrollUpWindow(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), panicWait)
	defer cancel()
	for range 6 {
		if out, err := exec.CommandContext(ctx, "wtype", "-M", "shift", "-k", "Prior", "-m", "shift").CombinedOutput(); err != nil {
			t.Fatalf("injecting Shift+Page_Up: %v\n%s", err, out)
		}
	}
	time.Sleep(200 * time.Millisecond)
}

// captureWindow captures the window this test is running in, and returns the file's bytes: what the
// window shows is pixels, and pixels are what the comparison is made of.
func captureWindow(t *testing.T, client hyprClient) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "window.png")
	region := fmt.Sprintf("%d,%d %dx%d", client.At[0], client.At[1], client.Size[0], client.Size[1])
	ctx, cancel := context.WithTimeout(context.Background(), panicWait)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "grim", "-g", region, path).CombinedOutput(); err != nil {
		t.Fatalf("capturing the window with grim -g %q: %v\n%s", region, err, out)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the window's capture: %v", err)
	}
	if len(body) == 0 {
		t.Fatalf("the window's capture is empty: a comparison of two empty pictures agrees with everything")
	}
	return string(body)
}
