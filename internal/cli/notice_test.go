package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheStatementFitsTheFiveLinesItPromises: a statement nobody reads protects nobody, and the
// plan caps it at five lines. This is the test that keeps a later edit from turning it into an
// essay.
func TestTheStatementFitsTheFiveLinesItPromises(t *testing.T) {
	lines := strings.Split(strings.TrimRight(noticeText, "\n"), "\n")
	if len(lines) > 5 {
		t.Errorf("the account-risk statement is %d lines, want 5 or fewer:\n%s", len(lines), noticeText)
	}
	for _, want := range []string{"identity", "throttle", "read-only", "distribute"} {
		if !strings.Contains(noticeText, want) {
			t.Errorf("the statement does not mention %q, so it leaves out one of the four things it exists to say:\n%s", want, noticeText)
		}
	}
}

// TestShowNoticeOncePrintsItOnceAndRecordsIt is the once-only contract: the statement goes to
// stderr (stdout belongs to the verb), the marker records that it was shown, and the second run
// says nothing at all.
func TestShowNoticeOncePrintsItOnceAndRecordsIt(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "notice-shown")

	print := func() (string, string) {
		_, stdout, stderr := run(t, func(ctx *Context, _ []string) int {
			ShowNoticeOnce(ctx, marker)
			return ExitOK
		})
		return stdout, stderr
	}

	stdout, stderr := print()
	if stdout != "" {
		t.Errorf("the first-run statement went to stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "session cookie") {
		t.Errorf("the first run did not print the statement: %q", stderr)
	}

	info, err := os.Stat(marker)
	if err != nil {
		t.Fatalf("the marker was not written: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("the marker is %o, want 0600", mode)
	}
	if info.Size() == 0 {
		t.Error("the marker is empty, so it records nothing about when the statement was shown")
	}

	stdout, stderr = print()
	if stdout != "" || stderr != "" {
		t.Errorf("the second run printed the statement again: stdout %q, stderr %q", stdout, stderr)
	}
}

// TestShowNoticeForcesTheStatement is what `doctor --show-notice` does: it prints whether or not
// the marker exists, and it goes to stdout because it was asked for.
func TestShowNoticeForcesTheStatement(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "notice-shown")
	if err := os.WriteFile(marker, []byte("shown\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, stdout, stderr := run(t, func(ctx *Context, _ []string) int {
		ShowNoticeOnce(ctx, marker)
		ShowNotice(ctx)
		return ExitOK
	})
	if stderr != "" {
		t.Errorf("a marked run said something on stderr: %q", stderr)
	}
	if !strings.Contains(stdout, "session cookie") {
		t.Errorf("the forced statement did not print: %q", stdout)
	}
}

// TestAnUnwritableMarkerStillPrintsTheStatement: the statement is the point. Losing it because a
// directory is not writable would be the wrong way round, and a user who was not warned is the
// failure this file exists to avoid.
func TestAnUnwritableMarkerStillPrintsTheStatement(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, stderr := run(t, func(ctx *Context, _ []string) int {
		ShowNoticeOnce(ctx, filepath.Join(blocker, "notice-shown"))
		return ExitOK
	})
	if !strings.Contains(stderr, "session cookie") {
		t.Errorf("an unwritable marker swallowed the statement: %q", stderr)
	}
	if !strings.Contains(stderr, "could not be written") {
		t.Errorf("the unwritable marker was not reported: %q", stderr)
	}
}
