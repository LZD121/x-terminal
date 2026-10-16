package cli

import (
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/keys"
)

// TestKeysPrintsTheWholeTable is the third surface's contract: one line per table row, in table
// order, three tab-separated fields, and no field written by hand in this package. A row added to
// the table with no second thought is a row that shows up here.
func TestKeysPrintsTheWholeTable(t *testing.T) {
	code, stdout, stderr := run(t, Keys)
	if code != ExitOK {
		t.Fatalf("keys exited %d (%s)", code, stderr)
	}

	// Anti-vacuity: the output is a table, not an empty stream with nothing to compare against.
	if stdout == "" {
		t.Fatal("keys printed nothing at all")
	}
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if len(lines) != len(keys.Rows()) {
		t.Fatalf("keys printed %d lines for a table of %d rows", len(lines), len(keys.Rows()))
	}

	for i, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			t.Fatalf("line %d is %d tab-separated fields, not three: %q", i+1, len(fields), line)
		}
		row := keys.Rows()[i]
		if fields[0] != row.KeysJoined() {
			t.Errorf("line %d names the keystrokes %q, the table says %q", i+1, fields[0], row.KeysJoined())
		}
		if fields[1] != string(row.Action) {
			t.Errorf("line %d names the action %q, the table says %q", i+1, fields[1], row.Action)
		}
		if fields[2] != row.Help {
			t.Errorf("line %d says %q, the table says %q", i+1, fields[2], row.Help)
		}
		// Every keystroke of the row is in the first field: a row whose keys were dropped would
		// still print a line, and the line would be a lie about what the key does.
		for _, key := range row.Keys {
			if !strings.Contains(fields[0], key) {
				t.Errorf("line %d does not name the keystroke %q", i+1, key)
			}
		}
	}
}

// TestKeysRefusesArguments keeps the verb's surface to itself: this phase has no configuration of
// its behaviour, and a verb that silently ignored an argument would be a verb whose output a
// caller cannot trust.
func TestKeysRefusesArguments(t *testing.T) {
	code, stdout, stderr := run(t, Keys, "--json")
	if code != ExitUsage {
		t.Fatalf("keys --json exited %d, want %d", code, ExitUsage)
	}
	if stdout != "" {
		t.Errorf("keys --json printed %q on stdout", stdout)
	}
	if !strings.Contains(stderr, "unexpected argument") {
		t.Errorf("keys --json said %q", stderr)
	}
}
