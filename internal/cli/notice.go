package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// noticeText is the account-risk statement: what this tool does with the user's identity, and
// what that costs them. Five lines and no more — a warning nobody reads protects nobody.
//
// It is not a licence: there is no click-through and nothing is recorded but the fact that it was
// shown once. What it is for is the first run, which is the moment a person decides whether to
// point this at their own account.
const noticeText = `x-terminal reads x.com with your own session cookie, so every request carries your real identity
and is indistinguishable from you using the website. The thresholds are undocumented — the numbers
in this tool are conservative, and a retry loop is how a fifteen-minute throttle becomes a
twenty-four-hour block, which is why nothing here retries and nothing runs two requests at once.
It is read-only by construction. Do not distribute it, and do not share the session file.
`

// ShowNoticeOnce prints the account-risk statement on the first successful run of a verb, records
// that it was shown, and does nothing on every run after that.
//
// It goes to stderr because stdout belongs to the verb's own output: a first-run statement
// spliced into `whoami`'s output would land in whatever pipeline asked for the handle. A marker
// that cannot be written is reported and the statement is printed anyway — the statement is the
// point, and losing it because a directory is not writable would be the wrong way round.
func ShowNoticeOnce(ctx *Context, markerPath string) {
	if markerPath == "" || noticeShown(markerPath) {
		return
	}
	fmt.Fprint(ctx.Stderr, noticeText)
	if err := markNoticeShown(markerPath); err != nil {
		fmt.Fprintf(ctx.Stderr, "(the first-run marker at %s could not be written: %v — this statement will be shown again)\n", markerPath, err)
	}
}

// ShowNotice prints the statement whether or not it has been shown before. It is what
// `doctor --show-notice` does, and it goes to stdout because it was asked for.
func ShowNotice(ctx *Context) {
	fmt.Fprint(ctx.Stdout, noticeText)
}

// noticeShown reports whether the marker file at path exists. A path that cannot be looked at is
// reported as "not shown": the cost of printing the statement twice is a paragraph, and the cost
// of never printing it is a user who was not warned.
func noticeShown(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// markNoticeShown writes the marker. The content is the time it was written, which is the only
// thing that makes the file legible a year later; nothing reads it back except this package's
// tests.
func markNoticeShown(path string) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	shown := time.Now().UTC().Format(time.RFC3339) + "\n"
	return os.WriteFile(path, []byte(shown), 0o600)
}
