// Release identity: the one place a build can say what it is.
//
// Two variables a release build sets with -ldflags, and their defaults for every other build. The
// zero value of both is deliberately honest rather than empty: a binary built from a working tree
// says so, because "unknown" that looks like a version is how a bug report loses an afternoon.
//
// The names are short because they appear on one line of `--version` output beside a Go version and
// an operating system, and the values never reach a screen the user did not ask for.
package cli

import (
	"fmt"
	"runtime"
)

var (
	// version is the release this binary is, set at build time to the tag that produced it
	// (`-X ...version=v1.0.0`). Empty in a build from a working tree.
	version string
	// commit is the revision it was built from, set at build time (`-X ...commit=$(git rev-parse
	// --short HEAD)`). Empty in a build from a working tree.
	commit string
)

// Version is the string a release build was stamped with, or "" when none was.
func Version() string { return version }

// Commit is the revision a release build was stamped with, or "" when none was.
func Commit() string { return commit }

// versionLine is the one line `--version` prints: the release and revision when the build carries
// them, the Go toolchain's own version, and the platform — every field a bug report needs and no
// field a reader has to interpret. It names no account, no session path and no configuration, so it
// is safe to paste.
func versionLine() string {
	what := "x-terminal (working tree build)"
	if version != "" {
		what = "x-terminal " + version
		if commit != "" {
			what += " (" + commit + ")"
		}
	}
	return fmt.Sprintf("%s %s %s/%s", what, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
