// Package volatile holds the one artifact this client cannot work out for itself: the
// GraphQL operation ids x.com's own web client uses, and the feature switches and field
// toggles each of those operations declares.
//
// They are data, not code. registry.json is embedded at build time and is the only file in
// the repository that spells a query id (XAPI-02), because the ids rotate with every web
// build and a literal in Go is a stale id waiting to be shipped. An id that is wrong fails
// the way this upstream always fails: a 404 with an empty body, or a 200 with a body shaped
// like nothing the caller expected, and no message anywhere saying which id was the problem.
//
// The snapshot is authoritative (XAPI-03). It was mined from the authenticated shell's own
// bundle on 2026-10-14 (see .planning/research/UPSTREAM-PROBE.md and
// .planning/research/x-ops-snapshot.json, whose 104 operations it carries unchanged), and
// every entry keeps the feature switches and field toggles that shipped beside it — the same
// artifact answers "which id?" and "which flags?", which is exactly the pair that drifts
// together. A refresh (cmd/x-opsgen) may only add or update; it can never delete an entry,
// and the file records where it came from so a later reader can judge it.
//
// Three entries are not from that bundle, because it holds no record of them at all:
// HomeTimeline, HomeLatestTimeline and Bookmarks carry `"source": "external"` — an id from a
// public capture of the web client's own operation table, named in the file's note — with no
// feature switches and no field toggles, because HomeLatestTimeline was measured answering
// 200 with 155 posts when no `features` parameter was sent at all. No bundle refresh can see
// them, which is exactly why the file has to say where they came from.
//
// A user pin (pins.go) wins over all of that: `x-terminal doctor --pin <operation> <queryId>`
// exists because the bundle cannot reveal operations it loads lazily, so there has to be a way
// to say "use this id" without editing the snapshot — which the externally sourced entries
// now do for the operations this project reads by default, and which a pin still overrides.
//
// Nothing here parses a response or builds a request: this package is the table, and
// internal/xweb/graphql.go is the only request builder that reads it.
package volatile

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"time"
)

// SchemaVersion is the schema this package reads and writes. It is a field rather than an
// implicit version so that a file from a future layout can be refused by name instead of
// being half-read: nothing here guesses at a schema it does not know.
const SchemaVersion = 1

//go:embed registry.json
var registryJSON []byte

// Source says where one operation's entry came from. The file's own provenance covers the
// bundle every mined entry was read out of, so only the exceptions need a source of their own:
// the entries a user pin supplied or overrode, and the ones that came from a public capture of
// the web client's operation table rather than from that bundle. They have different
// lifetimes — a bundle entry changes when x.com ships, a pin changes when the user says so,
// and the third has no refresh path at all — so a reader has to be able to tell them apart.
//
// Source is written to the file (as "source"), because a refresh rewrites it: a provenance the
// file does not carry would be gone after the first merge.
type Source string

const (
	// SourceSnapshot is an entry mined from the web bundle, and it is the zero value: an entry
	// with no source of its own is covered by the file's provenance, which is what all but a
	// handful of the entries are. The field is therefore omitted rather than repeated 104 times.
	SourceSnapshot Source = ""
	// SourcePin is an entry a user pin supplied or overrode. It is never a value in
	// registry.json — a pin is a pin because the pins file says so — and parse refuses a
	// registry file that claims one.
	SourcePin Source = "pin"
	// SourceExternal is an entry taken from a public capture of the web client's own operation
	// table, when the mined bundle held no record of the operation at all. Such an entry has no
	// refresh path: a merge of the bundle cannot see it, and the file's note is the only record
	// of where it came from.
	SourceExternal Source = "external"
)

// Operation is one GraphQL operation as the bundle declares it.
type Operation struct {
	// QueryID is the persisted-query hash x.com routes the operation by.
	QueryID string `json:"queryId"`
	// OperationType is "query" or "mutation" — which of the two the web client declares for this
	// operation. It is not the wire method: measured on 2026-10-15, SearchTimeline is a `query`
	// the upstream answers only over POST, so the two are not the same thing and this field is
	// only used to refuse anything that is not a query before a request is built. Which method a
	// request goes out with is decided in internal/xweb/graphql.go, because the bundle carries no
	// method and a refresh would drop one kept here.
	OperationType string `json:"operationType"`
	// FeatureSwitches are the feature flags this operation must be asked with. They are
	// data because they drift exactly when the ids do, and the upstream answers a missing
	// one with (336) The following features cannot be null: <names>.
	FeatureSwitches []string `json:"featureSwitches"`
	// FieldToggles are the same thing on the request side: which optional fields the
	// response should carry.
	FieldToggles []string `json:"fieldToggles"`
	// Source is where this entry came from: the snapshot the file's provenance names, a pin, or
	// an external capture. An entry with no source of its own is a snapshot entry and writes no
	// field at all; the two exceptions are written, because a refresh rewrites this file and a
	// provenance the file does not carry would not survive it.
	Source Source `json:"source,omitempty"`
}

// Provenance is what the file records about where its ids came from. A registry that cannot
// say when it was mined and from which bundle cannot be judged: an id is either current or
// it is a 404 with no body, and the difference is entirely in this record.
type Provenance struct {
	MinedAt            string `json:"mined_at"`
	SourceBundle       string `json:"source_bundle"`
	SourceBundleSHA256 string `json:"source_bundle_sha256,omitempty"`
	SourceBundleBytes  int    `json:"source_bundle_bytes,omitempty"`
	SourceContext      string `json:"source_context,omitempty"`
	Method             string `json:"method"`
	Note               string `json:"note,omitempty"`
	// RefreshedAt and RefreshedFrom are set by the refresh tool when it merges a newer
	// bundle in; MinedAt stays the date of the original snapshot.
	RefreshedAt   string `json:"refreshed_at,omitempty"`
	RefreshedFrom string `json:"refreshed_from,omitempty"`
}

// ErrUnknownOperation is the typed "this operation is not in the table" state. There is
// deliberately no fallback id behind it: a request sent with a stale id answers 404 with an
// empty body, which is indistinguishable from every other empty answer this upstream gives,
// and the one thing this project promises is that no state of the upstream is ever rendered
// as "no posts".
var ErrUnknownOperation = errors.New("xweb/volatile: unknown operation")

// Registry is the operation table: the snapshot, with the user's pins over it. It is not
// safe for concurrent use, and it does not need to be — it is read once per run.
type Registry struct {
	provenance Provenance
	ops        map[string]Operation
	warnings   []string
}

// registryFile is the on-disk shape. The operations map is keyed by operation name because
// that is what a caller has: the query id is the thing that goes stale, and it is the value.
type registryFile struct {
	Schema     int                  `json:"schema"`
	Provenance Provenance           `json:"provenance"`
	Operations map[string]Operation `json:"operations"`
}

// snapshotOnce parses the embedded file once. The parse cannot fail in a tested build — the
// data is compiled in and registry_test.go reads it — so the error path exists only to keep
// a broken build from panicking in the middle of a user's session.
var snapshotOnce = sync.OnceValues(func() (*Registry, error) { return parse(registryJSON, "the embedded registry") })

// Lookup resolves an operation through the embedded snapshot alone — no pins, no file on
// disk. It is the entry point for code that has no configuration to hand (a package-level
// default), so that a caller cannot accidentally resolve ids through a table it did not
// choose.
func Lookup(name string) (Operation, error) {
	reg, err := snapshotOnce()
	if err != nil {
		return Operation{}, err
	}
	return reg.Lookup(name)
}

// Load returns the snapshot with the pins at pinsPath merged over it, with pin precedence.
// An empty path means "the snapshot alone".
//
// A pins file that is missing is not a state worth reporting: not everyone has pinned
// anything. A pins file that is there but unreadable is reported as a warning on the
// registry rather than as an error, because a run that silently ignored the user's own
// override would be exactly the silent-wrongness this package exists to prevent — and a
// warning printed by `doctor` is how the user finds out. Writing pins is stricter: Pin
// refuses to touch a file it could not read.
func Load(pinsPath string) (*Registry, error) {
	reg, err := parse(registryJSON, "the embedded registry")
	if err != nil {
		return nil, err
	}
	if pinsPath == "" {
		return reg, nil
	}
	pins, err := readPins(pinsPath)
	if err != nil {
		reg.warnings = append(reg.warnings, err.Error())
		return reg, nil
	}
	reg.applyPins(pins)
	return reg, nil
}

// ReadFile reads a registry.json-shaped file from disk. It is what the refresh tool
// (cmd/x-opsgen) merges into: the snapshot is embedded in the binary at build time, so a
// refresh has to rewrite the source file and be followed by a build.
func ReadFile(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("xweb/volatile: reading %s: %w", path, err)
	}
	return parse(data, path)
}

// parse reads one registry.json. origin names the file in every error message, because a
// parse failure here is always "which file is wrong", never "what did I do wrong".
func parse(data []byte, origin string) (*Registry, error) {
	var file registryFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("xweb/volatile: %s does not parse: %w", origin, err)
	}
	if file.Schema != SchemaVersion {
		return nil, fmt.Errorf("xweb/volatile: %s has schema %d, and this build reads schema %d", origin, file.Schema, SchemaVersion)
	}
	if len(file.Operations) == 0 {
		return nil, fmt.Errorf("xweb/volatile: %s holds no operations", origin)
	}
	for name, op := range file.Operations {
		if op.QueryID == "" {
			return nil, fmt.Errorf("xweb/volatile: %s: operation %q carries no queryId", origin, name)
		}
		switch op.Source {
		case SourceSnapshot:
			// The overwhelming majority: the entry says nothing about where it came from,
			// because the file's provenance already covers it.
		case SourceExternal:
			// Carried as it is written: an entry no bundle refresh can see.
		case SourcePin:
			return nil, fmt.Errorf("xweb/volatile: %s: operation %q says it came from a pin, and a pin lives in the pins file rather than in this registry", origin, name)
		default:
			return nil, fmt.Errorf("xweb/volatile: %s: operation %q carries the unknown source %q; this build knows no source at all (a mined entry) and %q (an externally sourced one)", origin, name, op.Source, SourceExternal)
		}
		file.Operations[name] = op
	}
	return &Registry{provenance: file.Provenance, ops: file.Operations}, nil
}

// Lookup resolves one operation name. An unknown name is a typed error that says which name
// was unknown and how to fix it — refresh the snapshot, or pin the operation — rather than a
// fallback to any id at all.
func (r *Registry) Lookup(name string) (Operation, error) {
	if r == nil {
		return Operation{}, fmt.Errorf("%w: there is no registry to look %q up in", ErrUnknownOperation, name)
	}
	op, ok := r.ops[name]
	if !ok {
		return Operation{}, fmt.Errorf("%w: %q is not in the snapshot and no pin names it; run `x-terminal doctor --refresh-ids`, or pin it with `x-terminal doctor --pin %s <queryId>`", ErrUnknownOperation, name, name)
	}
	return op, nil
}

// Provenance returns where this registry's ids came from.
func (r *Registry) Provenance() Provenance {
	if r == nil {
		return Provenance{}
	}
	return r.provenance
}

// Warnings are the things wrong with the table that were not fatal: today, a pins file that
// could not be read. A caller that prints them (the doctor does) turns a silently ignored
// override into a sentence.
func (r *Registry) Warnings() []string {
	if r == nil {
		return nil
	}
	return slices.Clone(r.warnings)
}

// Len is how many operations the table holds.
func (r *Registry) Len() int {
	if r == nil {
		return 0
	}
	return len(r.ops)
}

// Names lists every operation name, sorted, so that two runs and two tools agree on the
// order of the same table.
func (r *Registry) Names() []string {
	if r == nil {
		return nil
	}
	names := make([]string, 0, len(r.ops))
	for name := range r.ops {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// MergeResult is what a merge did, in the counters the refresh tool prints. The four lists
// are the operation names in each state, not just counts: a refresh that updated an entry
// the user cared about should be able to say which one.
type MergeResult struct {
	Added     []string
	Updated   []string
	Unchanged []string
	Pinned    []string
}

// String renders the diff summary: `+3 added, ~1 updated, 100 unchanged`.
func (m MergeResult) String() string {
	line := fmt.Sprintf("+%d added, ~%d updated, %d unchanged", len(m.Added), len(m.Updated), len(m.Unchanged))
	if len(m.Pinned) > 0 {
		line += fmt.Sprintf(", %d pinned (kept)", len(m.Pinned))
	}
	return line
}

// Changed reports whether anything in the file would differ after this merge.
func (m MergeResult) Changed() bool { return len(m.Added) > 0 || len(m.Updated) > 0 }

// Merge folds freshly mined operations into the table. It adds an entry that was not there
// and replaces one whose record differs; it never removes anything, because the only thing
// a merge is allowed to know is what the bundle it just read contains, and a bundle that
// lost an operation (a lazy chunk, a build that has not finished rolling out) is not
// evidence that the operation is gone (XAPI-03).
//
// Three rules that are not obvious:
//
//   - An entry that came from a pin is left exactly as the user wrote it. A refresh knows
//     what x.com ships today; the user knows why they overrode it.
//   - A fresh record replaces the whole entry rather than being folded into it, because the
//     feature switches drift with the id: keeping yesterday's switches beside today's id is
//     the (336) failure mode with extra steps.
//   - An externally sourced entry needs no rule of its own. The bundle a refresh reads holds no
//     record of it — that is why it is external — so the merge cannot name it, and the add-and-
//     update rule leaves it exactly as the file wrote it. A bundle that one day does name it is
//     the thing that ships the ids: its record replaces the external one, provenance included,
//     because the entry is a mined entry by then, and keeping the old claim beside it would be a
//     lie about where the id came from.
func (r *Registry) Merge(mined map[string]Operation) MergeResult {
	var result MergeResult
	names := make([]string, 0, len(mined))
	for name := range mined {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		op := mined[name]
		op.Source = SourceSnapshot
		existing, ok := r.ops[name]
		switch {
		case !ok:
			r.ops[name] = op
			result.Added = append(result.Added, name)
		case existing.Source == SourcePin:
			result.Pinned = append(result.Pinned, name)
		case sameRecord(existing, op):
			result.Unchanged = append(result.Unchanged, name)
		default:
			r.ops[name] = op
			result.Updated = append(result.Updated, name)
		}
	}
	return result
}

// RecordRefresh notes that this table was refreshed from source at at. MinedAt keeps naming
// the original snapshot: the two dates answer different questions ("when were these ids
// mined" and "when was this file last touched"), and overwriting the first with the second
// would erase the only record of how old the ids actually are.
func (r *Registry) RecordRefresh(at time.Time, source string) {
	if r == nil {
		return
	}
	r.provenance.RefreshedAt = at.UTC().Format(time.RFC3339)
	r.provenance.RefreshedFrom = source
}

// sameRecord reports whether two entries would be the same record. Source is deliberately
// not compared: it is where an entry came from, not what it says.
func sameRecord(a, b Operation) bool {
	return a.QueryID == b.QueryID &&
		a.OperationType == b.OperationType &&
		slices.Equal(a.FeatureSwitches, b.FeatureSwitches) &&
		slices.Equal(a.FieldToggles, b.FieldToggles)
}

// Write rewrites path with this table, atomically: a temporary file in the same directory,
// then a rename, so that an interrupted refresh cannot leave a half-written registry behind.
// The file it writes is the file ReadFile reads, which is what makes the refresh tool's
// round trip (read, merge, write) meaningful — and it is why a table with the user's pins in it
// is refused here: a pin is not the snapshot's to write, and a file claiming one is a file parse
// would not read back. Every caller in this repository writes what ReadFile returned, and the
// pins file is where a pin is written.
func (r *Registry) Write(path string) error {
	for name, op := range r.ops {
		if op.Source == SourcePin {
			return fmt.Errorf("xweb/volatile: %s is a pin, and a pin lives in the pins file: writing this table would put the user's own id for %s into the snapshot and claim it came from there", path, name)
		}
	}
	file := registryFile{
		Schema:     SchemaVersion,
		Provenance: r.provenance,
		Operations: r.ops,
	}
	// A registry is documentation as much as data: it is read by people deciding whether an
	// id is current, so it is written with one operation per line and its provenance first.
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("xweb/volatile: encoding the registry: %w", err)
	}
	return writeAtomically(path, append(data, '\n'), 0o644)
}

// writeAtomically writes data to path through a temporary file in the same directory and a
// rename. The temporary file is created 0600 and chmodded to mode before the rename, so the
// file at path never exists with another mode.
func writeAtomically(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("xweb/volatile: preparing %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("xweb/volatile: writing %s: %w", path, err)
	}
	tmpName := tmp.Name()
	// From here on every failure has to remove the temporary file: a refresh that left its
	// scratch files in the repository would be a diff nobody wrote.
	defer func() {
		tmp.Close()
		os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("xweb/volatile: writing %s: %w", path, err)
	}
	if err := tmp.Chmod(mode); err != nil {
		return fmt.Errorf("xweb/volatile: setting the mode of %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("xweb/volatile: writing %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("xweb/volatile: installing %s: %w", path, err)
	}
	return nil
}

// FeaturesJSON renders an operation's feature switches as the object the wire format wants:
// {"switch":true,…}, with the keys sorted, because encoding/json sorts map keys.
//
// The upstream takes this as an object rather than as the array the registry stores. An empty
// list renders as {} — the caller decides whether to send that or to omit the parameter
// altogether (the web client omits it; see internal/xweb/graphql.go).
func FeaturesJSON(op Operation) (string, error) {
	return flagsJSON("feature switch", op.FeatureSwitches)
}

// TogglesJSON is FeaturesJSON for field toggles: {"toggle":true,…}.
func TogglesJSON(op Operation) (string, error) {
	return flagsJSON("field toggle", op.FieldToggles)
}

// flagsJSON turns a list of flag names into the {"name":true} object the wire format wants.
// An empty name is refused rather than sent: the upstream answers a flag it cannot resolve
// with (336) and a list of names, and a list that names nothing is not something a caller
// could act on.
func flagsJSON(kind string, names []string) (string, error) {
	flags := make(map[string]bool, len(names))
	for _, name := range names {
		if name == "" {
			return "", fmt.Errorf("xweb/volatile: the operation declares an empty %s name", kind)
		}
		flags[name] = true
	}
	encoded, err := json.Marshal(flags)
	if err != nil {
		return "", fmt.Errorf("xweb/volatile: encoding the %ss: %w", kind, err)
	}
	return string(encoded), nil
}
