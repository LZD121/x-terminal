package session

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Save writes a session's two values to path: the file `login` produces and `Load` reads.
//
// The write is the only place this program writes a credential, and it is deliberately shaped like
// the one other atomic write in the tree (the updater's binary replacement): a directory at 0700, a
// temporary file beside the target at 0600, fsync, rename, and an explicit chmod after it. A
// credential that lands in a world-readable file for the microseconds between create and chmod is
// the failure the mode argument exists to prevent, and a rename is what makes a half-written pair
// impossible to load.
//
// The file is refused rather than repaired, exactly as Load refuses it: a symlink at the path could
// point the credential anywhere, and a path that is not a regular file is not this program's file.
func Save(path string, authToken, cto Secret) error {
	if authToken == "" || cto == "" {
		// Half a pair is not a session: Load warns about a missing half, and writing one would make
		// that warning look like the user's own doing.
		return errors.New("session: both auth_token and ct0 are required")
	}
	if err := ensureWritable(path); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("session: creating %s: %w", dir, err)
	}
	// MkdirAll leaves an existing directory's mode alone, so a directory that was created by
	// something else with a looser mode is tightened here: the file inside it is the credential.
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("session: setting %s to 0700: %w", dir, err)
	}

	body := "# x-terminal session, written by `x-terminal login`. mode 0600.\n" +
		"auth_token=" + string(authToken) + "\n" +
		"ct0=" + string(cto) + "\n"

	tmp, err := os.CreateTemp(dir, ".session-*")
	if err != nil {
		return fmt.Errorf("session: creating a temporary file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	// Every failure path removes the temporary file: a stray file holding the cookie is worse than
	// the failure that left it.
	fail := func(err error) error {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		return fail(fmt.Errorf("session: setting the temporary file to 0600: %w", err))
	}
	if _, err := tmp.WriteString(body); err != nil {
		return fail(fmt.Errorf("session: writing the temporary file: %w", err))
	}
	if err := tmp.Sync(); err != nil {
		return fail(fmt.Errorf("session: flushing the temporary file: %w", err))
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("session: closing the temporary file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("session: replacing %s: %w", path, err)
	}
	// The rename kept the temporary file's 0600; this makes it a fact about the target rather than
	// an inference, and it is the line the mode test reads.
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("session: setting %s to 0600: %w", path, err)
	}
	return nil
}

// ensureWritable refuses the two shapes Load refuses, before anything is created: a symlink could
// move the credential to a path the program never named, and anything that is not a regular file is
// not a session file.
func ensureWritable(path string) error {
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("session: %w", err)
	case info.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("session file is a symlink: %s (remove it and run login again)", path)
	case !info.Mode().IsRegular():
		return fmt.Errorf("session file is not a regular file: %s", path)
	}
	return nil
}

// CleanInput trims a pasted value the way a reader would: whitespace, a surrounding pair of quotes,
// and the `auth_token=` prefix a user who copied the whole cookie line brings with them. It does not
// validate the shape — the upstream is the only authority on that — and it never prints the value.
func CleanInput(key, value string) Secret {
	v := strings.TrimSpace(value)
	// A paste may be the whole line — `auth_token = abc # from devtools` — so the key is matched
	// over the text before the first `=`, with the spaces parseFile tolerates.
	if k, rest, ok := strings.Cut(v, "="); ok && strings.EqualFold(strings.TrimSpace(k), key) {
		v = strings.TrimSpace(rest)
	}
	return Secret(unquoted(v))
}
