package render

// This file is the bare-terminal half of the environment specification: the stored 80-column frame
// is written into a real terminal emulator and the cursor's position is read back from it, which is
// the terminal's own word that it consumed exactly the frame's rows at the frame's width.
//
// Why the terminal the test is already running on, and not a pair of its own: the answer to a
// device-status request comes from the emulator that parses the bytes, and a freshly opened
// /dev/ptmx pair has nothing at its far end but the test itself — no emulator, no answer. The pair
// this check needs is the one the test's own standard input and output are on: the emulator holds
// the master, the test writes the frame into the slave, and the report arrives back on it.
//
// The window's size is the emulator's, not the program's: an emulator keeps drawing its own grid
// whatever a program asks the kernel for, so the harness *requires* a window 80 columns wide and at
// least one row taller than the frame, and skips with the numbers when it is not. A wider window
// would let a row that wrapped hide inside the row it wrapped into, and a window exactly as tall as
// the frame would scroll instead of moving the cursor down — which is the movement the report is
// read for.

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/unix"

	"github.com/LZD121/x-terminal/internal/text"
)

// ghosttyFrame is the frame this check writes: the stored bare-terminal frame at the primary
// example's width. The multiplexer twin is task 2's frame, not this one.
const ghosttyFrame = "timeline_80"

// clearScreenAndHome blanks the screen and puts the cursor at its first cell: what has to happen
// before the frame is written, or the frame's first row would land on whatever was on screen.
const clearScreenAndHome = "\x1b[2J\x1b[H"

// cursorReport matches a terminal's answer to a device-status request: `ESC [ row ; col R`.
var cursorReport = regexp.MustCompile(`\x1b\[(\d+);(\d+)R`)

// TestCaptureInGhostty runs only in a bare Ghostty window and skips with a printed reason anywhere
// else: a run on another machine (or inside a multiplexer) must not pass by doing nothing.
//
// Its first subtest is the harness's own arm, and that one runs everywhere: the frame is checked
// before it is written into somebody's terminal, because a frame that did not fit would wrap and the
// cursor report would then be read as a width the frame does not have. The terminal leg skips
// wherever there is no window to write into, which is every machine but the one the frames were
// rendered for.
func TestCaptureInGhostty(t *testing.T) {
	t.Run("the frame is checked before it is written", func(t *testing.T) {
		frame := storedBare(t, ghosttyFrame)
		if problem := frameFits(frame, frame.w); problem != "" {
			t.Fatalf("the stored frame %s is not writable into a %d-column terminal: %s", frame.name, frame.w, problem)
		}
		wide := oneCellTooWide(frame)
		if problem := frameFits(wide, frame.w); problem == "" {
			t.Error("the frame check accepted a row one cell too wide: a frame that would wrap would be written and the cursor report would pass for the wrong reason")
		}
	})

	program := os.Getenv("TERM_PROGRAM")
	if tmux := os.Getenv("TMUX"); tmux != "" || program != "ghostty" {
		t.Skipf("this check runs in a bare Ghostty window and nowhere else: TMUX=%q (want it unset) and TERM_PROGRAM=%q (want %q). In a Ghostty window 80 columns wide, in the repository root, run: go test ./internal/render -run TestCaptureInGhostty -count=1 -v",
			tmux, program, "ghostty")
	}
	// The frame is the bare terminal's, so its measure is the bare terminal's.
	t.Setenv("TMUX", "")
	t.Setenv("TZ", "UTC")

	frame := storedBare(t, ghosttyFrame)
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		t.Skip("this check needs the terminal it is running in: standard input and output are not a terminal, so no emulator is parsing what the harness writes")
	}
	cols, rows, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		t.Fatalf("asking the terminal for its size: %v", err)
	}
	if cols != frame.w || rows < corpusH+1 {
		t.Skipf("the window is %d columns by %d rows: this check needs exactly %d columns (the frame's own width, so a row that wrapped has nowhere to hide) and at least %d rows (one more than the frame, so a wrapped row moves the cursor down instead of scrolling the frame away)",
			cols, rows, frame.w, corpusH+1)
	}

	restore := rawInput(t)
	defer restore()

	// A blank screen and the cursor home first: the frame must start at the terminal's first row, or
	// the row count it is read for would be measured from whatever happened to be on screen.
	if _, err := fmt.Fprint(os.Stdout, clearScreenAndHome); err != nil {
		t.Fatalf("clearing the screen: %v", err)
	}
	// The frame verbatim, with no trailing newline (it ends with the status line's last cell), then
	// the device-status request: the cursor is where the frame left it, and the terminal is about to
	// say where that is.
	if _, err := fmt.Fprint(os.Stdout, frame.bytes, "\x1b[6n"); err != nil {
		t.Fatalf("writing the frame: %v", err)
	}

	row, col := awaitCursorReport(t, os.Stdin, 5*time.Second)
	// The frame is corpusH rows of exactly frame.w cells, and its last row has no newline after it:
	// a terminal that consumed it exactly is on row corpusH, one column past the last cell it drew.
	if row != corpusH || col != frame.w+1 {
		t.Errorf("the terminal reports the cursor at row %d, column %d; want row %d (the frame's last row, so no row wrapped) and column %d (one past its last cell, so the frame filled the width it claims)",
			row, col, corpusH, frame.w+1)
	}
	t.Logf("the bare terminal consumed the %d stored rows of %s at %d cells each: cursor at row %d, column %d",
		corpusH, frame.name, frame.w, row, col)
}

// storedBare reads one stored frame and asserts it is the bare terminal's: a twin would be the
// multiplexer's frame, whose measure is not the one this check is about.
func storedBare(t *testing.T, name string) storedFrame {
	t.Helper()
	for _, f := range corpus(t) {
		if f.name != name {
			continue
		}
		if f.env.tmux != "" {
			t.Fatalf("the frame %s is stored for a multiplexer: this check is the bare terminal's", name)
		}
		return readFrame(t, name, f.w, f.h, f.env, f.state)
	}
	t.Fatalf("the corpus names no frame %q: this check has nothing to write", name)
	return storedFrame{}
}

// frameFits is the pre-check the frame passes before it is written into a terminal: the frame's own
// row count, and no row wider than the pane in the measure the frame was rendered with. (Every row
// is padded to exactly the pane's width, so "no row wider than the pane" is the frame's own
// definition of filling it, and the cursor report is the terminal's.)
func frameFits(f storedFrame, w int) string {
	rows := strings.Split(f.bytes, "\n")
	if len(rows) != corpusH {
		return fmt.Sprintf("it holds %d rows, want the %d the frame is", len(rows), corpusH)
	}
	for i, row := range rows {
		if got := text.Width(ansi.Strip(row)); got > w {
			return fmt.Sprintf("row %d is %d cells of text, over the pane's %d", i+1, got, w)
		}
	}
	return ""
}

// oneCellTooWide is the frame with one cell added to its first row: the row a terminal would wrap,
// and the difference the pre-check has to report rather than let the terminal swallow.
func oneCellTooWide(f storedFrame) storedFrame {
	rows := strings.Split(f.bytes, "\n")
	rows[0] += "x"
	f.bytes = strings.Join(rows, "\n")
	return f
}

// rawInput puts the terminal's *input* side into the mode the report can be read in — no echo, no
// canonical buffering, a read that gives up after a tenth of a second — and leaves its output side
// alone. That last part is the point: the frame goes out verbatim, and what returns the cursor to
// column 1 after each of its newlines is the terminal's own output processing. (x/term's MakeRaw
// clears the output flags along with the input ones, which would draw the frame as a staircase; the
// termios bits are therefore cleared here, one by one.)
func rawInput(t *testing.T) func() {
	t.Helper()
	return rawInputFd(t, int(os.Stdin.Fd()))
}

// rawInputFd is rawInput for one file descriptor: the window leg's is the terminal's stdin, and the
// plumbing check's is the slave of a pair it opened itself.
func rawInputFd(t *testing.T, fd int) func() {
	t.Helper()
	before, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		t.Fatalf("reading the terminal's modes: %v", err)
	}
	after := *before
	after.Lflag &^= unix.ECHO | unix.ICANON | unix.ISIG | unix.IEXTEN
	after.Cc[unix.VMIN] = 0
	after.Cc[unix.VTIME] = 1 // a tenth of a second, in the terminal's own unit
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &after); err != nil {
		t.Fatalf("putting the terminal's input into a readable mode: %v", err)
	}
	restored := false
	restore := func() {
		if restored {
			return
		}
		restored = true
		if err := unix.IoctlSetTermios(fd, unix.TCSETS, before); err != nil {
			t.Errorf("restoring the terminal's modes: %v", err)
		}
	}
	t.Cleanup(restore)
	return restore
}

// awaitCursorReport reads the terminal's answer to the device-status request: the first cursor report
// in whatever arrives on the terminal's input, which is the emulator's own word for where the cursor
// ended up.
func awaitCursorReport(t *testing.T, in *os.File, timeout time.Duration) (row, col int) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var (
		seen []byte
		last error
	)
	buf := make([]byte, 256)
	for time.Now().Before(deadline) {
		n, err := in.Read(buf)
		if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
			last = err
		}
		if n <= 0 {
			continue
		}
		seen = append(seen, buf[:n]...)
		match := cursorReport.FindSubmatch(seen)
		if match == nil {
			continue
		}
		row, err := strconv.Atoi(string(match[1]))
		if err != nil {
			t.Fatalf("the terminal's cursor report %q has no row in it: %v", match[0], err)
		}
		col, err := strconv.Atoi(string(match[2]))
		if err != nil {
			t.Fatalf("the terminal's cursor report %q has no column in it: %v", match[0], err)
		}
		return row, col
	}
	t.Fatalf("the terminal did not answer the cursor query within %s (read %q, last error %v): an emulator that does not answer cannot confirm what the frame occupied", timeout, seen, last)
	return 0, 0
}

// TestPtyHarnessPlumbing exercises the window leg's own machinery on a PTY pair this test opens: the
// pair, its window size, the frame written through it, and the read and parse of a cursor report.
//
// The report is canned — the peer at the far end of this pair is the test, not a terminal — so this
// is not evidence about any terminal, and it does not stand in for the window leg: it is the part of
// that leg this host can check with no display, and it is named for what it is. What it does check
// is that the harness writes the stored frame verbatim (each row's newline turned into the
// terminal's own carriage return and line feed by the line discipline the harness deliberately
// leaves alone), that the window size it asks for is the frame's width and one row taller, and that
// it reads an answer off the pair rather than waiting for one that never comes.
func TestPtyHarnessPlumbing(t *testing.T) {
	frame := storedBare(t, ghosttyFrame)
	t.Setenv("TMUX", "")
	t.Setenv("TZ", "UTC")

	master, slave := ptyPair(t)
	// The window the leg asks for: the frame's width, one row taller than the frame, so a row that
	// wrapped would move the cursor down rather than scrolling the frame away.
	if err := unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: uint16(corpusH + 1), Col: uint16(frame.w)}); err != nil {
		t.Fatalf("setting the pair's window size: %v", err)
	}
	size, err := unix.IoctlGetWinsize(int(slave.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		t.Fatalf("reading the pair's window size back: %v", err)
	}
	if int(size.Col) != frame.w || int(size.Row) != corpusH+1 {
		t.Errorf("the pair's window is %dx%d, want %dx%d", size.Col, size.Row, frame.w, corpusH+1)
	}

	// The peer: it reads what the harness writes and answers the way a terminal would. A terminal's
	// own answer is the window leg's business; what is checked here is the harness's half of it.
	written := make(chan string, 1)
	go func() {
		var seen strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := master.Read(buf)
			if n > 0 {
				seen.Write(buf[:n])
				if strings.Contains(seen.String(), "\x1b[6n") {
					fmt.Fprintf(master, "\x1b[%d;%dR", corpusH, frame.w+1)
					written <- seen.String()
					return
				}
			}
			if err != nil {
				written <- seen.String()
				return
			}
		}
	}()

	restore := rawInputFd(t, int(slave.Fd()))
	defer restore()
	if _, err := fmt.Fprint(slave, clearScreenAndHome, frame.bytes, "\x1b[6n"); err != nil {
		t.Fatalf("writing the frame through the pair: %v", err)
	}
	row, col := awaitCursorReport(t, slave, 5*time.Second)
	if row != corpusH || col != frame.w+1 {
		t.Errorf("the harness read the report as row %d, column %d; want row %d and column %d", row, col, corpusH, frame.w+1)
	}
	want := clearScreenAndHome + strings.ReplaceAll(frame.bytes, "\n", "\r\n") + "\x1b[6n"
	if got := <-written; got != want {
		t.Errorf("the pair received %q,\nwant the stored frame verbatim behind the clear and the request:\n%q", got, want)
	}
}

// ptyPair opens a pseudo-terminal pair: the master, which holds the terminal's input and output, and
// the slave, which is the terminal a program would see. It is the pair the window leg's description
// is written in terms of, and it skips rather than fails where this host has none.
func ptyPair(t *testing.T) (master, slave *os.File) {
	t.Helper()
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("this check needs a pseudo-terminal and /dev/ptmx cannot be opened: %v", err)
	}
	if err := unix.IoctlSetPointerInt(int(m.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		m.Close()
		t.Skipf("this check needs a pseudo-terminal and the pair cannot be unlocked: %v", err)
	}
	number, err := unix.IoctlGetInt(int(m.Fd()), unix.TIOCGPTN)
	if err != nil {
		m.Close()
		t.Skipf("this check needs a pseudo-terminal and the pair reports no number: %v", err)
	}
	s, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		m.Close()
		t.Skipf("this check needs a pseudo-terminal and /dev/pts/%d cannot be opened: %v", number, err)
	}
	t.Cleanup(func() {
		s.Close()
		m.Close()
	})
	return m, s
}
