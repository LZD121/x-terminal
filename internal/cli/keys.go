package cli

import (
	"fmt"

	"github.com/LZD121/x-terminal/internal/keys"
)

// Keys is the `keys` verb: the one keymap table, printed for the surfaces outside the program —
// the shipped shell completion scripts read it, and a user can read the whole keymap without
// starting the pane.
//
// One line per binding, tab-separated, in table order: the keystrokes, the action, the help text.
// Every one of those three fields is the table's own — the keystrokes through the table's join
// rule, the action and the help verbatim — so nothing in this output is written here and a row
// added to the table appears in it with no second edit.
func Keys(ctx *Context, args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(ctx.Stderr, "keys: unexpected argument %s\n", args[0])
		usage(ctx.Stderr)
		return ExitUsage
	}

	for _, row := range keys.Rows() {
		fmt.Fprintf(ctx.Stdout, "%s\t%s\t%s\n", row.KeysJoined(), row.Action, row.Help)
	}
	return ExitOK
}
