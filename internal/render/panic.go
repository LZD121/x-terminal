package render

import (
	"strings"

	"github.com/LZD121/x-terminal/internal/text"
)

// CopyTitlePlain and CopyTitlePanic are two of the three names of the thing on screen: the plain
// reader's own name and the panic frame's. They are declared here, with the four styles' names in
// cover.go, because the title is a function of the state in every state and nothing else spells it.
const (
	// CopyTitlePlain is the plain reader's terminal title and tmux window name.
	CopyTitlePlain = "x-terminal"
	// CopyTitlePanic is the panic frame's terminal title and tmux window name: the name of a command
	// that a build really runs, because the title names what is on screen and the frame is output.
	CopyTitlePanic = "make"
	// CopyTitleVivid is the vivid presentation's title. It is a command name too, and it is a
	// different one because the frame is a different thing: the plain reader's name would be a lie
	// about a pane that no longer looks like a log, and a product name would be the one word the
	// plain reader exists to avoid. It is not hidden — a window title is not a way in — and it says
	// nothing about x.com.
	CopyTitleVivid = "viewer"
)

// The panic frame's three columns, in cells: the `ok`, the package and the duration of a `go test`
// line, with the package field 32 cells wide. All three are multiples of 4 and none of them is
// computed from the content's length. panic_test.go reads these three values and holds the frame's
// own lines to them.
const (
	panicStatusCol   = 0
	panicPackageCol  = 8
	panicDurationCol = 40
	// panicPackageCells is the package field's width: the cells between the package column and the
	// duration column, which is where a package name longer than the field would reach.
	panicPackageCells = panicDurationCol - panicPackageCol
)

// panicLines is the panic frame's whole content: the `ok` lines of a `go test ./...` run of a
// monorepo that does not exist, in the order the run printed them, at the frame's three columns and
// spaced with spaces only — the program emits no tab byte in any state.
//
// It is one screen of the most boring output a terminal can show, and that is the point: `go test`
// output is what a terminal shows after a command has scrolled a screenful of results, nothing about
// it invites typing or waiting, an observer's question about what is being read has a one-word
// answer, and it carries no identity, no prompt, no hostname, no clock and no path from the user's
// machine. The module is `corp/...`, a name that cannot collide with a real handle or display name.
var panicLines = [...]string{
	"ok      corp/billing/internal/ledger    0.412s",
	"ok      corp/billing/internal/rates     0.031s",
	"ok      corp/billing/internal/invoice   0.208s",
	"ok      corp/billing/internal/pricing   0.019s",
	"ok      corp/billing/internal/pdf       0.512s",
	"ok      corp/billing/internal/tax       0.077s",
	"ok      corp/billing/internal/dunning   0.003s",
	"ok      corp/billing/internal/import    0.033s",
	"ok      corp/ledger                     0.655s",
	"ok      corp/ledger/internal/reconcile  0.121s",
	"ok      corp/ledger/internal/reports    0.088s",
	"ok      corp/api/internal/auth          0.044s",
	"ok      corp/api/internal/handlers      0.210s",
	"ok      corp/api/internal/ratelimit     0.009s",
	"ok      corp/jobs/internal/runner       0.121s",
	"ok      corp/jobs/internal/schedule     0.015s",
	"ok      corp/notify/internal/mail       0.062s",
	"ok      corp/notify/internal/webhook    0.028s",
	"ok      corp/config                     0.007s",
	"ok      corp/logging                    0.005s",
	"ok      corp/retry                      0.002s",
}

// PanicFrame is the panic state's whole screen: one constant frame that asks the model for nothing
// but the pane's size, and is the same frame whether it was entered from the plain reader or from a
// cover. It issues no request, reads no file, starts no timer, animates nothing and renders no
// character of the feed; it carries no selection column, no status line and not one escape byte.
//
// The frame owns rows 1..H: every row is exactly W cells wide, space-padded, and the row count is
// exactly H at every size. At a width above the longest line the lines are printed as they are and
// never stretched — terminal output does not widen because the window did, and a frame that filled
// every cell would read as a UI rather than as output. At a width below the longest line the lines
// wrap the way a terminal wraps them, and content taller than the pane shows its last H rows with no
// truncation mark, because the terminal scrolled and nothing was cut. Below the pane's floor even
// that cannot be laid out, and the frame is the plain reader's own floor line.
func PanicFrame(w, h int) string {
	if h < 1 {
		return ""
	}
	rows := make([]string, 0, h)
	if w < MinPaneCols {
		rows = append(rows, cutField(" "+CopyPaneTooNarrow, w, MarkTruncation))
	} else {
		for _, line := range panicLines {
			if text.Width(line) <= w {
				rows = append(rows, line)
				continue
			}
			rows = append(rows, text.Wrap(line, w)...)
		}
		if len(rows) > h {
			rows = rows[len(rows)-h:]
		}
	}
	for len(rows) < h {
		// The blank rows below the run are what a paused command's output looks like. Filling them
		// would mean inventing a prompt — a claim that the command finished, which the frame has no
		// reason to make.
		rows = append(rows, "")
	}
	for i := range rows {
		rows[i] = text.Pad(rows[i], w)
	}
	return strings.Join(rows, "\n")
}
