package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultName is the account the pre-store single file is called: `login` wrote it before there was
// more than one, and an operator who never runs `login --as` keeps using it. It is a name like any
// other, so nothing in the shell has to special-case it.
const DefaultName = "default"

// Store is a directory of session files, one per account name, plus the single file this program
// wrote before there was more than one.
//
// The split is deliberate: the legacy path stays where it was (so an existing session keeps working
// with no migration), and every named account lives in its own 0600 file in Dir. A name is the only
// thing that crosses between the two — nothing above this package ever sees a path per account.
type Store struct {
	// Dir is the sessions directory: one regular file per named account.
	Dir string
	// Legacy is the single file the first version of `login` wrote, and the file the account named
	// DefaultName resolves to. It is read and written exactly like any other.
	Legacy string
}

// Path is the file one account's pair lives in, or the empty string for a name this store refuses.
func (s Store) Path(name string) string {
	if name == DefaultName {
		return s.Legacy
	}
	if !ValidName(name) {
		return ""
	}
	return filepath.Join(s.Dir, name)
}

// ValidName reports whether a name is one this store will use as a file name. The rules are what
// keeps a name from becoming a path: no separator, no traversal, no leading dot (a name that looks
// like a hidden temporary file), and a bounded length.
func ValidName(name string) bool {
	if name == "" || len(name) > 64 || strings.HasPrefix(name, ".") {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

// Names lists every account the store holds, DefaultName first, then the directory's files sorted.
// A directory that does not exist yet is not an error: it is a store with no named accounts.
func (s Store) Names() ([]string, error) {
	names := make([]string, 0, 4)
	if _, err := os.Lstat(s.Legacy); err == nil {
		names = append(names, DefaultName)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("session: %w", err)
	}
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, os.ErrNotExist) {
		return names, nil
	}
	if err != nil {
		return nil, fmt.Errorf("session: %w", err)
	}
	named := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !ValidName(e.Name()) {
			continue
		}
		named = append(named, e.Name())
	}
	sort.Strings(named)
	return append(names, named...), nil
}

// Load reads one account's pair, refusing anything Load refuses.
func (s Store) Load(name string) (Session, error) {
	path := s.Path(name)
	if path == "" {
		return Session{}, fmt.Errorf("session: %q is not a name this store uses", name)
	}
	return Load(path)
}

// Save writes one account's pair through Save, which is where the file's mode and its atomic
// replacement live.
func (s Store) Save(name string, authToken, cto Secret) error {
	path := s.Path(name)
	if path == "" {
		return fmt.Errorf("session: %q is not a name this store uses", name)
	}
	return Save(path, authToken, cto)
}

// Delete removes one account's file. The browser session is untouched, exactly as Logout documents.
func (s Store) Delete(name string) error {
	path := s.Path(name)
	if path == "" {
		return fmt.Errorf("session: %q is not a name this store uses", name)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("session: %w", err)
	}
	return nil
}

// Active resolves the account to start on: the requested name when the store holds it, then
// DefaultName, then the first name there is. An empty answer means the store is empty, which the
// caller reports as "no session" rather than guessing.
func (s Store) Active(preferred string) (string, bool, error) {
	names, err := s.Names()
	if err != nil {
		return "", false, err
	}
	if len(names) == 0 {
		return "", false, nil
	}
	for _, n := range names {
		if n == preferred {
			return n, true, nil
		}
	}
	for _, n := range names {
		if n == DefaultName {
			return n, false, nil
		}
	}
	return names[0], false, nil
}
