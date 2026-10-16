// Package cli is argument handling, dispatch and exit codes: the whole surface is
// one table, and nothing else in the binary decides what a verb is.
//
// Business logic does not live here. The bodies live next to this file, one per
// verb — `doctor` and the registry flags in internal/cli/doctor.go, `logout` in
// internal/cli/session.go, `whoami` in internal/cli/whoami.go, the first-run
// statement in internal/cli/notice.go, `keys` in internal/cli/keys.go, the
// timeline in internal/cli/feed.go, `sync` in internal/cli/sync.go, `audit` in
// internal/cli/audit.go and `update` in internal/cli/update.go — and the table
// below is the whole surface. Every
// entry has a body: a verb with none would have to be visible as a stub, and
// there is no longer a stub to be.
//
// Nothing here reads the environment, opens a file or makes a request, and no
// flag parsing happens in this package: a verb's arguments are whatever the
// verb's own body makes of them.
package cli

import (
	"fmt"
	"io"
	"os"
)

// Exit codes. 0 is reserved for a body's own success, so no path through Main
// returns it without a body having produced it.
const (
	ExitOK    = 0
	ExitError = 1
	// ExitUsage is a bad invocation. It is not 0: nothing ran, and a script that treated a
	// mistaken invocation as success would act on output that was never produced.
	ExitUsage = 2
)

// Context is what a verb's body gets: the two streams. Stdout belongs to the
// verb's own output; every diagnostic and every usage line goes to Stderr, so a
// piped or TERM=dumb terminal cannot find escape salad spliced into the data it
// is reading.
type Context struct {
	Stdout io.Writer
	Stderr io.Writer
}

// Handler is a verb's body. It returns the process exit code rather than an
// error because a verb owns its own diagnostics and its own exit code.
type Handler func(*Context, []string) int

// Command is one entry of the CLI surface.
type Command struct {
	Name  string
	Usage string // the verb's own line in the usage listing
	Run   Handler
}

// commands is the whole CLI surface: exactly these verbs, and no more. The
// usage listing is rendered from this table, so the surface and its copy cannot
// drift apart — a verb added here appears in the help, and a verb absent here
// cannot be dispatched.
//
// It is a function rather than a package-level slice because a verb's body prints
// the usage after a mistaken invocation, and the usage is rendered from this
// table: as an initializer that is a cycle (commands -> Doctor -> usage ->
// commands), and a table that cannot name its own verbs is not a table.
func commands() []Command {
	return []Command{
		{Name: "doctor", Usage: "doctor   check the network path, the session and the request budget", Run: Doctor},
		{Name: "logout", Usage: "logout   delete the local session copy (the browser session is untouched)", Run: Logout},
		{Name: "whoami", Usage: "whoami   name the account the loaded session belongs to", Run: Whoami},
		{Name: "feed", Usage: "feed     read the Following timeline in the pane", Run: Feed},
		{Name: "keys", Usage: "keys     list the keymap: keystrokes, action and help, tab-separated", Run: Keys},
		{Name: "sync", Usage: "sync     fill the saved mirror from x.com (--full re-lists, --kind picks one list)", Run: Sync},
		{Name: "audit", Usage: "audit    check the tree against the promises this build makes", Run: Audit},
		{Name: "update", Usage: "update   back this binary from the latest GitHub Release (--check asks without installing)", Run: Update},
	}
}

// Main dispatches args, already stripped of the program name, and returns the
// process exit code.
func Main(args []string) int {
	ctx := &Context{Stdout: os.Stdout, Stderr: os.Stderr}

	if len(args) == 0 {
		// No arguments is not a mistake: it is the program's own surface, and the surface is the
		// Following timeline in the pane. It is the one verb that opens a screen rather than printing
		// something, which is what running the binary bare is for; every other shape of invocation —
		// `--help`, a named verb, an unknown word — keeps the usage path it had.
		return Feed(ctx, nil)
	}

	if args[0] == "--version" {
		// The second path that asked for the text it gets, and the same rule as --help: it is not a
		// diagnostic, so it goes to stdout and succeeds. A release binary that cannot name itself is
		// one nobody can report a bug against.
		fmt.Fprintln(ctx.Stdout, versionLine())
		return ExitOK
	}

	if args[0] == "--help" {
		// The one path that asked for the text it gets. Help that was requested
		// is not a diagnostic, so it goes to stdout and it succeeds, where a
		// usage printed after a mistake goes to stderr and fails.
		usage(ctx.Stdout)
		return ExitOK
	}

	for _, cmd := range commands() {
		if cmd.Name == args[0] {
			return cmd.Run(ctx, args[1:])
		}
	}

	fmt.Fprintf(ctx.Stderr, "unknown command: %s\n", args[0])
	usage(ctx.Stderr)
	return ExitUsage
}

// usage prints the header, one line per table entry and the hint, to whichever
// stream the caller chose.
func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: x-terminal [command] [args...]")
	fmt.Fprintln(w)
	for _, cmd := range commands() {
		fmt.Fprintln(w, "  "+cmd.Usage)
	}
	// The two flags are a second group with their own column: `--version` is wider than the table's
	// name field, and a row that cannot line up with its own neighbours is the defect this listing
	// exists to avoid. The test asserts each group's column separately.
	fmt.Fprintln(w, "  --help     print this message")
	fmt.Fprintln(w, "  --version  print what this build is")
	fmt.Fprintln(w, "With no command, x-terminal opens the feed.")
}
