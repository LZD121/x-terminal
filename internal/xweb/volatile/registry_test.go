package volatile

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// place where an id is allowed to appear in Go, so it has to be the place that notices when
// the data file changes under it.
const (
	viewerID           = "TpQxQtWMTMnIAMPlMGMqNw"
	userByScreenNameID = "AMIBMjtxEEATh4z8V9GtRg"
	userTweetsID       = "7_KBXBpG0zBPRN6UCIV0-w"
)

func TestTheEmbeddedSnapshotCarriesEveryMinedOperation(t *testing.T) {
	reg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// The mined bundle held 104 operations. A floor rather than an equality, so that adding
	// an operation to the snapshot is not a test failure — but pruning one is.
	if got := reg.Len(); got < 100 {
		t.Fatalf("the embedded registry holds %d operations, want at least the 100 the mined bundle had", got)
	}

	for name, want := range map[string]string{
		"Viewer":           viewerID,
		"UserByScreenName": userByScreenNameID,
		"UserTweets":       userTweetsID,
	} {
		op, err := reg.Lookup(name)
		if err != nil {
			t.Fatalf("Lookup(%q): %v", name, err)
		}
		if op.QueryID != want {
			t.Errorf("the snapshot has %s = %q, want the measured %q", name, op.QueryID, want)
		}
		if op.Source != SourceSnapshot {
			t.Errorf("%s came from %q, want the snapshot", name, op.Source)
		}
	}

	// Viewer is one of the operations that ships feature switches and field toggles with its
	// id; a registry that lost them would send a request the upstream answers with (336).
	viewer, err := reg.Lookup("Viewer")
	if err != nil {
		t.Fatal(err)
	}
	if len(viewer.FeatureSwitches) == 0 || len(viewer.FieldToggles) == 0 {
		t.Errorf("Viewer carries %d switches and %d toggles, want both non-empty: the bundle ships them with the id",
			len(viewer.FeatureSwitches), len(viewer.FieldToggles))
	}
	if viewer.OperationType != "query" {
		t.Errorf("Viewer is a %q, want a query", viewer.OperationType)
	}

	prov := reg.Provenance()
	switch {
	case prov.MinedAt == "":
		t.Error("the registry records no mining date")
	case prov.SourceBundle == "":
		t.Error("the registry names no source bundle")
	case prov.Method == "":
		t.Error("the registry records no method")
	}
}

// TestTheSnapshotKeepsEveryMinedOperation is the anti-pruning arm: the reshape from the
// research snapshot into the registry schema is allowed to change the file's shape and
// nothing else.
//
// The research snapshot is committed to this repository, so a checkout where it cannot be read
// is not a checkout with less to check — it is a broken checkout, or a run from outside the
// module — and this guard would be checking nothing. It fails rather than skips: a guard that
// disables itself the moment its input goes missing is a guard nobody would notice had stopped
// working.
func TestTheSnapshotKeepsEveryMinedOperation(t *testing.T) {
	mined, err := os.ReadFile(filepath.Join(repoRoot(t), ".planning", "research", "x-ops-snapshot.json"))
	if err != nil {
		t.Fatalf("the research snapshot at .planning/research/x-ops-snapshot.json could not be read: %v — it is committed to this repository and this test compares the registry against it, so its absence is a failure here rather than a reason to check nothing", err)
	}
	var file struct {
		Operations map[string]Operation `json:"operations"`
	}
	if err := json.Unmarshal(mined, &file); err != nil {
		t.Fatalf("parsing the research snapshot: %v", err)
	}
	if len(file.Operations) == 0 {
		t.Fatal("the research snapshot parsed as empty")
	}

	reg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for name, want := range file.Operations {
		got, err := reg.Lookup(name)
		if err != nil {
			t.Errorf("the registry lost %s: %v", name, err)
			continue
		}
		if !sameRecord(got, want) {
			t.Errorf("%s came out as %+v, want what the mined bundle said: %+v", name, got, want)
		}
	}
}

// externalOperations are the entries registry.json carries from a public capture of the web
// client's own operation table rather than from the mined bundle — the bundle holds no record of
// them at all — and the three of them are this project's whole path to its home screens and its
// bookmarks. They are pinned here, in the one place a test may name an id, so that neither a
// refresh nor a well-meaning reshape of the file can drop them or lose what the file says about
// where they came from without this failing.
var externalOperations = map[string]string{
	"HomeTimeline":       "og4a4SdSF3WiQkkwaPCdPg",
	"HomeLatestTimeline": "OQPHTgwczzp9RMAPt6BH9A",
	"Bookmarks":          "-dgKZ58Dr9YSJYrcgEb5KA",
}

func TestTheExternallySourcedOperationsAreInTheSnapshot(t *testing.T) {
	reg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for name, id := range externalOperations {
		op, err := reg.Lookup(name)
		if err != nil {
			t.Errorf("the snapshot lost %s, and the bundle cannot supply it: %v", name, err)
			continue
		}
		if op.QueryID != id {
			t.Errorf("%s is %q, want the measured %q", name, op.QueryID, id)
		}
		if op.Source != SourceExternal {
			t.Errorf("%s came from %q, want %q: an id the bundle never carried has to keep saying so", name, op.Source, SourceExternal)
		}
		if op.OperationType != "query" {
			t.Errorf("%s is a %q, want a query", name, op.OperationType)
		}
		if len(op.FeatureSwitches) != 0 || len(op.FieldToggles) != 0 {
			t.Errorf("%s carries %d feature switches and %d field toggles; HomeLatestTimeline was measured answering 200 with 155 posts when no features parameter was sent at all, so the bag is optional for these operations and nothing here may invent one", name, len(op.FeatureSwitches), len(op.FieldToggles))
		}
	}
}

func TestAnUnknownOperationIsTypedAndSaysHowToFixIt(t *testing.T) {
	reg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	_, err = reg.Lookup("NoSuchOperation")
	if !errors.Is(err, ErrUnknownOperation) {
		t.Fatalf("Lookup of an unknown name returned %v, want an error that wraps ErrUnknownOperation", err)
	}
	for _, want := range []string{"NoSuchOperation", "--refresh-ids", "--pin"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q, so it does not say what to do: %v", want, err)
		}
	}
}

func TestAPinWinsOverTheSnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, PinsFileName)

	// One pin for an operation the bundle never revealed, and one for an operation it did:
	// the first is built from the pin alone, the second keeps the fields the pin omits.
	if err := Pin(path, "HomeTimeline", "1u0Wlkw6Ru1NwBUD-pDiww"); err != nil {
		t.Fatalf("Pin(HomeTimeline): %v", err)
	}
	if err := Pin(path, "UserTweets", "7_KBXBpG0zBPRN6UCIV0-w"); err != nil {
		t.Fatalf("Pin(UserTweets): %v", err)
	}
	if err := Pin(path, "UserTweets", "AAAAAAAAAAAAAAAAAAAAAA"); err != nil {
		t.Fatalf("Pin(UserTweets) again: %v", err)
	}

	reg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if warnings := reg.Warnings(); len(warnings) != 0 {
		t.Fatalf("a pins file this package wrote produced warnings: %v", warnings)
	}

	home, err := reg.Lookup("HomeTimeline")
	if err != nil {
		t.Fatalf("the pin for HomeTimeline is not in the registry: %v", err)
	}
	switch {
	case home.QueryID != "1u0Wlkw6Ru1NwBUD-pDiww":
		t.Errorf("HomeTimeline resolved to %q, want the pinned id", home.QueryID)
	case home.Source != SourcePin:
		t.Errorf("HomeTimeline came from %q, want it marked as a pin", home.Source)
	case home.OperationType != "query":
		t.Errorf("a pinned operation with no type came out as %q, want a query", home.OperationType)
	}

	tweets, err := reg.Lookup("UserTweets")
	if err != nil {
		t.Fatal(err)
	}
	if tweets.QueryID != "AAAAAAAAAAAAAAAAAAAAAA" {
		t.Errorf("the second pin for UserTweets did not take: %q", tweets.QueryID)
	}
	if len(tweets.FeatureSwitches) == 0 {
		t.Error("pinning an id dropped the snapshot's feature switches; a pin that names only an id overrides only the id")
	}
	if tweets.Source != SourcePin {
		t.Errorf("UserTweets came from %q, want it marked as a pin", tweets.Source)
	}

	// The pins are the second writer's business: re-pinning one operation must not lose the
	// other, and the file must parse as what it is.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file pinFile
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("the pins file this package wrote does not parse: %v", err)
	}
	if len(file.Operations) != 2 {
		t.Errorf("the pins file holds %d operations, want 2", len(file.Operations))
	}
}

func TestAPinsFileThatCannotBeReadIsAWarningNotSilence(t *testing.T) {
	path := filepath.Join(t.TempDir(), PinsFileName)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	reg, err := Load(path)
	if err != nil {
		t.Fatalf("a broken pins file failed the whole load: %v", err)
	}
	warnings := reg.Warnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0], path) {
		t.Fatalf("the registry reported %v, want one warning naming %s", warnings, path)
	}
	if _, err := reg.Lookup("Viewer"); err != nil {
		t.Errorf("a broken pins file cost the snapshot: %v", err)
	}
}

func TestAMergeAddsAndUpdatesButNeverDeletes(t *testing.T) {
	reg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	before := reg.Len()

	// A bundle that revealed nothing — a lazy chunk that did not load, a build mid-rollout —
	// is not evidence that anything was removed.
	if res := reg.Merge(nil); res.Changed() {
		t.Errorf("merging nothing reported %q", res)
	}
	if reg.Len() != before {
		t.Fatalf("merging nothing changed the table from %d to %d entries", before, reg.Len())
	}

	// One new operation, one whose record changed, and one that is byte-identical to the
	// snapshot: the last one is what makes the counter meaningful rather than always "updated".
	//
	// The new one is a name no snapshot carries. HomeTimeline would have done while the bundle
	// simply did not reveal it; now the file carries it as an externally sourced entry, so its
	// merge is an update rather than an addition, and this test is about the counters.
	same, err := reg.Lookup("UserByScreenName")
	if err != nil {
		t.Fatal(err)
	}
	res := reg.Merge(map[string]Operation{
		"NeverMinedOperation": {QueryID: "1u0Wlkw6Ru1NwBUD-pDiww", OperationType: "query", FeatureSwitches: []string{"a"}},
		"UserTweets":          {QueryID: "ZZZZZZZZZZZZZZZZZZZZZZ", OperationType: "query", FeatureSwitches: []string{"b"}},
		"UserByScreenName":    same,
	})
	switch {
	case len(res.Added) != 1 || res.Added[0] != "NeverMinedOperation":
		t.Errorf("added %v, want just NeverMinedOperation", res.Added)
	case len(res.Updated) != 1 || res.Updated[0] != "UserTweets":
		t.Errorf("updated %v, want just UserTweets", res.Updated)
	case len(res.Unchanged) != 1 || res.Unchanged[0] != "UserByScreenName":
		t.Errorf("unchanged %v, want just UserByScreenName", res.Unchanged)
	}
	if reg.Len() != before+1 {
		t.Errorf("the table went from %d to %d entries; a merge adds and updates, and never drops the 100 operations the bundle did not mention", before, reg.Len())
	}
	if op, err := reg.Lookup("UserTweets"); err != nil || op.QueryID != "ZZZZZZZZZZZZZZZZZZZZZZ" {
		t.Errorf("the updated id did not take: %+v, %v", op, err)
	}
	if op, err := reg.Lookup("NeverMinedOperation"); err != nil || op.QueryID != "1u0Wlkw6Ru1NwBUD-pDiww" {
		t.Errorf("the added operation is not resolvable: %+v, %v", op, err)
	}
	if op, err := reg.Lookup("Viewer"); err != nil || op.QueryID != viewerID {
		t.Errorf("an operation the merge never mentioned is gone: %+v, %v", op, err)
	}

	// The whole entry is replaced, not folded into: switches drift with the id.
	if op, _ := reg.Lookup("UserTweets"); len(op.FeatureSwitches) != 1 || op.FeatureSwitches[0] != "b" {
		t.Errorf("the updated entry kept %v, want the switches the new record carried", op.FeatureSwitches)
	}
}

func TestAMergeLeavesAPinAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), PinsFileName)
	if err := Pin(path, "UserTweets", "AAAAAAAAAAAAAAAAAAAAAA"); err != nil {
		t.Fatal(err)
	}
	reg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	res := reg.Merge(map[string]Operation{"UserTweets": {QueryID: "ZZZZZZZZZZZZZZZZZZZZZZ", OperationType: "query"}})
	if len(res.Pinned) != 1 || res.Pinned[0] != "UserTweets" {
		t.Fatalf("the merge reported %q, want UserTweets kept because it is pinned", res)
	}
	op, err := reg.Lookup("UserTweets")
	if err != nil {
		t.Fatal(err)
	}
	if op.QueryID != "AAAAAAAAAAAAAAAAAAAAAA" {
		t.Errorf("a refresh overwrote the user's pin with %q", op.QueryID)
	}
}

// TestAnExternalEntrySurvivesARefresh is the reason those three entries carry a source at all.
// A refresh merges what the bundle declares and writes the registry file back; the bundle
// declares nothing about them, so the merge has to leave them exactly as they are, and the
// write has to carry the record that they did not come from the bundle. A provenance the file
// does not carry is a provenance the first refresh would silently lose, along with the only
// ids this project has for its home screens.
func TestAnExternalEntrySurvivesARefresh(t *testing.T) {
	reg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	before := reg.Len()

	// The whole shape a merge is allowed to have: one operation the table had never held, and
	// one whose record changed. Neither of them mentions the three.
	res := reg.Merge(map[string]Operation{
		"NeverMinedOperation": {QueryID: "AAAAAAAAAAAAAAAAAAAAAA", OperationType: "query"},
		"UserTweets":          {QueryID: "BBBBBBBBBBBBBBBBBBBBBB", OperationType: "query"},
	})
	if len(res.Added) != 1 || len(res.Updated) != 1 {
		t.Fatalf("the merge reported %q, want one addition and one update", res)
	}
	if reg.Len() != before+1 {
		t.Fatalf("the table went from %d to %d entries, want %d", before, reg.Len(), before+1)
	}
	mentioned := append(append([]string{}, res.Added...), res.Updated...)
	mentioned = append(mentioned, res.Pinned...)
	for _, name := range mentioned {
		if _, external := externalOperations[name]; external {
			t.Errorf("the merge reported %s, and the bundle it read has no record of that operation", name)
		}
	}

	// Lookup still resolves them, with the same id and the same provenance: a merge adds and
	// updates and never deletes, and nothing here named them.
	for name, id := range externalOperations {
		op, err := reg.Lookup(name)
		if err != nil {
			t.Errorf("the merge dropped %s: %v", name, err)
			continue
		}
		if op.QueryID != id || op.Source != SourceExternal {
			t.Errorf("after the merge %s is %+v, want %q marked %q", name, op, id, SourceExternal)
		}
	}

	// And the same again through the file, because a refresh writes the registry it merged
	// into: ReadFile -> Merge -> Write is the round trip the refresh tool performs.
	path := filepath.Join(t.TempDir(), "registry.json")
	if err := reg.Write(path); err != nil {
		t.Fatalf("Write: %v", err)
	}
	back, err := ReadFile(path)
	if err != nil {
		t.Fatalf("the registry this package wrote does not read back: %v", err)
	}
	for name, id := range externalOperations {
		op, err := back.Lookup(name)
		if err != nil {
			t.Errorf("the rewritten registry lost %s: %v", name, err)
			continue
		}
		if op.QueryID != id || op.Source != SourceExternal {
			t.Errorf("the rewritten registry holds %s as %+v, want %q marked %q", name, op, id, SourceExternal)
		}
	}
}

func TestTheDiffSummaryIsTheLineTheToolPrints(t *testing.T) {
	line := MergeResult{
		Added:     []string{"a", "b", "c"},
		Updated:   []string{"d"},
		Unchanged: make([]string, 100),
	}.String()
	if want := "+3 added, ~1 updated, 100 unchanged"; line != want {
		t.Errorf("the diff summary is %q, want %q", line, want)
	}
	withPin := MergeResult{Pinned: []string{"HomeTimeline"}}.String()
	if !strings.Contains(withPin, "pinned") {
		t.Errorf("a merge that kept a pin does not say so: %q", withPin)
	}
}

func TestFeaturesJSONIsTheWireObject(t *testing.T) {
	for _, tc := range []struct {
		name  string
		op    Operation
		want  string
		fails bool
	}{
		{name: "two switches", op: Operation{FeatureSwitches: []string{"a", "b"}}, want: `{"a":true,"b":true}`},
		{name: "none", op: Operation{FeatureSwitches: nil}, want: `{}`},
		{name: "an empty name is refused", op: Operation{FeatureSwitches: []string{"a", ""}}, fails: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := FeaturesJSON(tc.op)
			if tc.fails {
				if err == nil {
					t.Fatalf("FeaturesJSON returned %q and no error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("FeaturesJSON: %v", err)
			}
			if got != tc.want {
				t.Errorf("FeaturesJSON = %q, want %q", got, tc.want)
			}
		})
	}

	got, err := TogglesJSON(Operation{FieldToggles: []string{"withPayments", "withDmBlocks"}})
	if err != nil {
		t.Fatalf("TogglesJSON: %v", err)
	}
	if want := `{"withDmBlocks":true,"withPayments":true}`; got != want {
		t.Errorf("TogglesJSON = %q, want %q", got, want)
	}
}

func TestPinWritesAtomicallyAndKeepsItsNeighbours(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, PinsFileName)

	if err := Pin(path, "HomeTimeline", "1u0Wlkw6Ru1NwBUD-pDiww"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("the pins file is %o, want 0600: it sits beside the session file", mode)
	}

	// No scratch files: the write goes through a temporary file and a rename, and a rename
	// that left its temporary behind would put a file nobody wrote into the repository.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("the pins directory holds %d files, want just the pins file", len(entries))
	}
}

func TestPinRefusesWhatItCannotPin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, PinsFileName)
	if err := Pin(path, "HomeTimeline", "1u0Wlkw6Ru1NwBUD-pDiww"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ operation, queryID string }{
		{"Home Timeline", "1u0Wlkw6Ru1NwBUD-pDiww"},
		{"", "1u0Wlkw6Ru1NwBUD-pDiww"},
		{"HomeTimeline", ""},
		{"HomeTimeline", "too-short"},
		{"HomeTimeline", "1u0Wlkw6Ru1NwBUD-pDiww!"},
		{"HomeTimeline", "1u0Wlkw6Ru1NwBUD-pDi/ww"},
	} {
		if err := Pin(path, tc.operation, tc.queryID); err == nil {
			t.Errorf("Pin(%q, %q) was accepted", tc.operation, tc.queryID)
		}
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Errorf("a refused pin changed the file:\n%s\n%s", before, after)
	}
}

func TestPinRefusesToReplacePinsItCannotRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), PinsFileName)
	broken := []byte("{\"operations\": {\"HomeTimeline\": {\"queryId\":")
	if err := os.WriteFile(path, broken, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Pin(path, "HomeTimeline", "1u0Wlkw6Ru1NwBUD-pDiww"); err == nil {
		t.Fatal("Pin replaced a pins file it could not parse, which is how the user's own overrides get lost")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(broken) {
		t.Errorf("the refused pin left the file as %q, want it untouched", after)
	}
}

func TestWriteRoundTripsTheTable(t *testing.T) {
	reg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "registry.json")
	if err := reg.Write(path); err != nil {
		t.Fatal(err)
	}

	back, err := ReadFile(path)
	if err != nil {
		t.Fatalf("the file this package wrote does not read back: %v", err)
	}
	if back.Len() != reg.Len() {
		t.Errorf("the file holds %d operations, want the %d that were written", back.Len(), reg.Len())
	}
	op, err := back.Lookup("Viewer")
	if err != nil || op.QueryID != viewerID {
		t.Errorf("Viewer came back as %+v, %v", op, err)
	}
	if back.Provenance().SourceBundle != reg.Provenance().SourceBundle {
		t.Error("the round trip lost the provenance")
	}
}

// TestWriteRefusesATableWithTheUsersPinsInIt closes the one way this file and parse could
// disagree now that the file carries a source: a pin is written to the pins file, and a registry
// file that claims one is refused when it is read back, so it must not be written in the first
// place either.
func TestWriteRefusesATableWithTheUsersPinsInIt(t *testing.T) {
	pins := filepath.Join(t.TempDir(), PinsFileName)
	if err := Pin(pins, "HomeTimeline", "1u0Wlkw6Ru1NwBUD-pDiww"); err != nil {
		t.Fatal(err)
	}
	reg, err := Load(pins)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "registry.json")
	err = reg.Write(path)
	if err == nil {
		t.Fatal("Write accepted a table with the user's pin in it, and the file it wrote is one parse refuses to read")
	}
	if !strings.Contains(err.Error(), "HomeTimeline") {
		t.Errorf("the refusal does not name the pin it refused: %v", err)
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Error("the refused write left a file behind")
	}
}

// TestTheOperationIDsLiveOnlyInThisPackagesData is XAPI-02's proof, and it is the reason
// registry.go can claim that no id literal exists anywhere else. Two arms, because they fail
// for different reasons: the first catches an id written next to the word queryId (the shape
// a hard-coded call site takes), and the second catches any of the 104 known ids anywhere in
// a non-test Go file, whatever the wording around it.
//
// Test files are exempt on purpose. A test that pins the wire format has to be able to name
// the id it expects — internal/xweb/graphql_test.go is the one that asserts the request path
// for Viewer — and pinning the expected value is the opposite of falling back to a stale one.
func TestTheOperationIDsLiveOnlyInThisPackagesData(t *testing.T) {
	root := repoRoot(t)
	ids, err := Load("")
	if err != nil {
		t.Fatal(err)
	}

	// The plan's rule, made executable: a 20+ character base64url-looking run of characters on
	// a line that mentions queryId, outside this package's data and tests.
	looksLikeAnID := regexp.MustCompile(`[A-Za-z0-9_-]{20,}`)
	mentionsQueryID := regexp.MustCompile(`(?i)query[_]?id`)

	var nearQueryID, hardCoded []string
	for _, path := range goFilesOutsideThisPackage(t, root) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		rel, _ := filepath.Rel(root, path)

		for i, line := range strings.Split(string(data), "\n") {
			if mentionsQueryID.MatchString(line) {
				if run := looksLikeAnID.FindString(line); run != "" {
					nearQueryID = append(nearQueryID, rel+":"+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
				}
			}
		}
		for _, name := range ids.Names() {
			op, _ := ids.Lookup(name)
			if bytes.Contains(data, []byte(op.QueryID)) {
				hardCoded = append(hardCoded, rel+": carries the id of "+name+" ("+op.QueryID+")")
			}
		}
	}

	const remedy = "Operation ids are data (XAPI-02): they live in internal/xweb/volatile/registry.json and nowhere else. " +
		"Resolve one with volatile.Lookup(name) or its client, or pin an operation with `x-terminal doctor --pin <operation> <queryId>`; " +
		"a refresh rewrites the data file: `x-opsgen --live --merge`."
	if len(nearQueryID) > 0 {
		t.Errorf("%d line(s) outside internal/xweb/volatile spell an id next to queryId:\n%s\n%s", len(nearQueryID), strings.Join(nearQueryID, "\n"), remedy)
	}
	if len(hardCoded) > 0 {
		t.Errorf("%d file(s) hold an operation id from the registry:\n%s\n%s", len(hardCoded), strings.Join(hardCoded, "\n"), remedy)
	}
}

// goFilesOutsideThisPackage lists every non-test .go file under internal/ and cmd/ that is
// not part of this package. The data file's own package is where ids are allowed to be; the
// tests are the one place they may be pinned as expectations.
func goFilesOutsideThisPackage(t *testing.T, root string) []string {
	t.Helper()
	here := filepath.Join(root, "internal", "xweb", "volatile")

	var paths []string
	for _, dir := range []string{filepath.Join(root, "internal"), filepath.Join(root, "cmd")} {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return err
			case d.IsDir():
				if path == here {
					return fs.SkipDir
				}
				return nil
			case !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go"):
				return nil
			}
			paths = append(paths, path)
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", dir, err)
		}
	}
	if len(paths) == 0 {
		t.Fatal("the guard found no files to check, so it cannot fail and proves nothing")
	}
	return paths
}

// repoRoot walks up from the test's working directory to the module root. The guard has to
// look at the whole module, and the test's directory is inside it.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", dir)
		}
		dir = parent
	}
}
