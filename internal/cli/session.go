package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/session"
)

// Logout is the `logout` verb: delete the local session copy, and nothing else.
//
// This is the one command that removes a file, so it removes exactly one: the session file
// the configuration resolves, after saying which path that is and what the file holds. It
// never touches the directory, never follows a symlink (a symlink means the path is not the
// file that was checked), and never prints the cookie — the browser's own session, which is
// where the credential came from, is left alone.
func Logout(ctx *Context, args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(ctx.Stderr, "logout takes no arguments\n")
		usage(ctx.Stderr)
		return ExitUsage
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "logout: %v\n", err)
		return ExitError
	}
	path := cfg.SessionPath

	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// Nothing to delete is not a failure: the file the user asked to be gone is gone.
		fmt.Fprintf(ctx.Stdout, "no session file at %s\n", path)
		return ExitOK
	case err != nil:
		fmt.Fprintf(ctx.Stderr, "logout: %v\n", err)
		return ExitError
	case info.Mode()&fs.ModeSymlink != 0:
		fmt.Fprintf(ctx.Stderr, "refusing to delete %s: it is a symlink, so the file this command would remove is not the file it checked; remove the link itself if that is what you mean\n", path)
		return ExitError
	case !info.Mode().IsRegular():
		fmt.Fprintf(ctx.Stderr, "refusing to delete %s: it is not a regular file\n", path)
		return ExitError
	}

	// What is about to be deleted, said before it is deleted: the path, and the state of the
	// session it holds — lengths, presence and origin, never a value. A file that will not load
	// (a mode looser than 0600, say) is still worth deleting, so its refusal is reported as the
	// state rather than acted on.
	fmt.Fprintf(ctx.Stdout, "about to delete %s\n", path)
	if sess, err := session.Load(path); err == nil {
		fmt.Fprintf(ctx.Stdout, "%s\n", sess.Summary())
	} else {
		fmt.Fprintf(ctx.Stdout, "the session file is there but does not load: %v\n", err)
	}

	if err := os.Remove(path); err != nil {
		fmt.Fprintf(ctx.Stderr, "logout: %v\n", err)
		return ExitError
	}
	fmt.Fprintf(ctx.Stdout, "deleted %s — the browser's own session is untouched; the next run needs the cookie again\n", path)
	return ExitOK
}
