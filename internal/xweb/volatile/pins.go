package volatile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"sort"
)

// PinsFileName is the file, under the data directory, that holds the user's own operation ids. It
// is not the registry: the registry is a snapshot of what x.com ships, and this is what the user
// decided on top of it.
//
// The name lives here, beside the file's own shape, and the *path* is config.PinsPath's, which
// builds it from this constant: a name that only one of the two files knew would be a file the
// reader and the writer could disagree about.
const PinsFileName = "pins.json"

// pinFile is the on-disk shape of pins.json:
//
//	{"operations": {"HomeTimeline": {"queryId": "…"}}}
//
// An entry may also carry operationType, featureSwitches and fieldToggles; a field the pin
// does not name keeps whatever the snapshot says (see applyPins).
type pinFile struct {
	Operations map[string]Operation `json:"operations"`
}

// operationName matches the names x.com uses: letters, digits and underscores, beginning with
// a letter. It is checked so that a typo is refused at the moment of pinning rather than
// silently pinning an operation nobody will ever ask for.
var operationName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// queryIDShape matches the persisted-query hashes the bundle ships: 22 characters of
// base64url in every one of the 104 mined records. The width is checked loosely (10 to 96)
// so that a future build is not refused by an old rule, and the alphabet strictly, because a
// query id pasted with a URL fragment or a quote attached would otherwise become a 404 with
// no explanation.
var queryIDShape = regexp.MustCompile(`^[A-Za-z0-9_-]{10,96}$`)

// Pin writes one operation -> queryId override into the pins file at path, atomically, and
// leaves every other pin in the file alone.
//
// It refuses a path it cannot read rather than replacing it: pins are the user's own
// overrides and the one thing this function must never do is lose one it did not understand.
// It also refuses an operation name or a query id that does not look like one, because a pin
// is written by hand and a pin that does not work fails the same silent way a stale id does.
func Pin(path, operation, queryID string) error {
	if path == "" {
		return errors.New("xweb/volatile: no pins file to write; the data directory is not configured")
	}
	if !operationName.MatchString(operation) {
		return fmt.Errorf("xweb/volatile: %q is not an operation name, so there is nothing to pin; x.com's own names are letters, digits and underscores", operation)
	}
	if !queryIDShape.MatchString(queryID) {
		return fmt.Errorf("xweb/volatile: %q is not a query id, so pinning it would send a request the upstream cannot route; the ids the bundle ships are base64url (letters, digits, - and _)", queryID)
	}

	pins, err := readPins(path)
	if err != nil {
		return err
	}
	if pins.Operations == nil {
		pins.Operations = make(map[string]Operation)
	}
	// The pin keeps any fields it already had: pinning the same operation twice with a new
	// id should not throw away the feature switches the user set with the first pin.
	entry := pins.Operations[operation]
	entry.QueryID = queryID
	pins.Operations[operation] = entry

	data, err := json.MarshalIndent(pins, "", "  ")
	if err != nil {
		return fmt.Errorf("xweb/volatile: encoding %s: %w", path, err)
	}
	// 0600: the pins file is not a credential, but it lives beside the session file and is
	// just as much the user's own business.
	return writeAtomically(path, append(data, '\n'), 0o600)
}

// readPins reads the pins file. A missing file is an empty set of pins, not an error — the
// file only exists once something has been pinned.
func readPins(path string) (pinFile, error) {
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return pinFile{}, nil
	case err != nil:
		return pinFile{}, fmt.Errorf("xweb/volatile: reading the pins at %s: %w", path, err)
	}
	var pins pinFile
	if err := json.Unmarshal(data, &pins); err != nil {
		return pinFile{}, fmt.Errorf("xweb/volatile: the pins at %s do not parse, so they are being ignored: %w", path, err)
	}
	return pins, nil
}

// applyPins overlays the user's pins on the table and marks those entries as pinned.
//
// A pin normally names only an id — `doctor --pin HomeTimeline <id>` is the whole gesture —
// so the rule is per field: a field the pin sets replaces the snapshot's, and a field it
// omits keeps whatever was there. For an operation the bundle never revealed (the timeline
// operations are loaded lazily and are absent from main.js) there is nothing to keep, and the
// entry is built from the pin alone; an operation with no type is a query, which is what the
// web client sends a GET for.
//
// A pin with no queryId is reported as a warning and skipped: it is a broken file, and
// quietly pinning the operation to the empty string would be worse than saying so.
func (r *Registry) applyPins(pins pinFile) {
	for _, name := range sortedNames(pins.Operations) {
		pin := pins.Operations[name]
		if pin.QueryID == "" {
			r.warnings = append(r.warnings, fmt.Sprintf("xweb/volatile: the pin for %s names no queryId and was ignored", name))
			continue
		}
		op := r.ops[name]
		if pin.OperationType != "" {
			op.OperationType = pin.OperationType
		}
		if len(pin.FeatureSwitches) > 0 {
			op.FeatureSwitches = pin.FeatureSwitches
		}
		if len(pin.FieldToggles) > 0 {
			op.FieldToggles = pin.FieldToggles
		}
		if op.OperationType == "" {
			op.OperationType = "query"
		}
		op.QueryID = pin.QueryID
		op.Source = SourcePin
		r.ops[name] = op
	}
}

// sortedNames is the keys of an operations map, sorted, so that the warnings a bad pins file
// produces come out in the same order twice.
func sortedNames(ops map[string]Operation) []string {
	names := make([]string, 0, len(ops))
	for name := range ops {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
