package term

import (
	"os/exec"
)

// tmuxProgram is the multiplexer's program name. It is a constant because it is named in one place
// — every call goes through it — and because the tests assert the arguments of the commands this
// package builds.
const tmuxProgram = "tmux"

// This file is the multiplexer half of the package: the decisions, separable from the doing,
// because they are what the table-driven tests drive. Each command's arguments are built in one
// place and passed to the runner as an argument vector — never through a shell — so a pane id or a
// window name can never become a word of a command line.

// Pane is the multiplexer pane this process is running in: `$TMUX_PANE`, or empty.
//
// Empty or unset means "there is no tmux history to clear and no window to rename": no subprocess is
// started, nothing is reported, and no error is surfaced. The app never guesses a pane, so `$TMUX`
// set with `$TMUX_PANE` empty is the same case — it is not read at all.
func Pane(env func(string) string) string {
	if env == nil {
		return ""
	}
	return env("TMUX_PANE")
}

// ClearHistoryArgs is `tmux clear-history -t <pane>`: the command that removes the pane's saved
// lines. Under a multiplexer it is the only mechanism that does, because ED 3 changes nothing in
// tmux's own history (Probe E).
//
// The target is always an explicit pane. The alternative — letting tmux pick the current one — reads
// a server-wide notion of "current" that this process does not own.
func ClearHistoryArgs(pane string) []string {
	return []string{"clear-history", "-t", pane}
}

// RenameWindowArgs is `tmux rename-window -t <pane> <name>`: the window name is set to the state's
// title by name rather than by index, on the same explicit pane.
func RenameWindowArgs(pane, name string) []string {
	return []string{"rename-window", "-t", pane, name}
}

// WindowNameArgs is `tmux display-message -p '#W'`: the window's current name, printed and nothing
// else. It is what the exit-time restore puts back.
func WindowNameArgs() []string {
	return []string{"display-message", "-p", "#W"}
}

// execCommand is one of two places in the module that name os/exec (with
// internal/media); the module-wide boundary test (internal/tui's
// TestProcessSpawnsAreOnlyInTermAndMedia) is what keeps it that way. Output()
// rather than a combined stream: the three commands print their answer on stdout
// and their failures on stderr, and a failure is thrown away by the callers.
func execCommand(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}
