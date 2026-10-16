package audit

import (
	"strings"
	"testing"
)

func treeOf(files ...File) Tree { return Tree(files) }

func TestReadOnlyPassesOnReads(t *testing.T) {
	src := `package p
const op = "HomeTimeline"
func f() { _ = "Viewer" }
`
	r := checkReadonly(treeOf(File{Path: "internal/xsource/x.go", Src: []byte(src)}))
	if len(r.Findings) != 0 {
		t.Fatalf("reads failed: %+v", r.Findings)
	}
	if r.Files != 1 {
		t.Fatalf("Files=%d", r.Files)
	}
}

func TestReadOnlyFailsUnclassified(t *testing.T) {
	// Name in registry set via a fake registry file, not in opClass.
	reg := `{"operations":{"BrandNewOp":{"queryId":"x","operationType":"query"}}}`
	src := `package p
const op = "BrandNewOp"
`
	r := checkReadonly(treeOf(
		File{Path: "internal/xweb/volatile/registry.json", Src: []byte(reg)},
		File{Path: "internal/xsource/x.go", Src: []byte(src)},
	))
	if !hasDetail(r, "unclassified") {
		t.Fatalf("want unclassified finding, got %+v", r.Findings)
	}
	if !hasPathLine(r, "internal/xsource/x.go") {
		t.Fatalf("finding missing file/line: %+v", r.Findings)
	}
}

func TestReadOnlyFailsWriteName(t *testing.T) {
	src := `package p
const op = "CreateTweet"
`
	r := checkReadonly(treeOf(File{Path: "internal/xsource/x.go", Src: []byte(src)}))
	if !hasDetail(r, "write operation") {
		t.Fatalf("want write finding, got %+v", r.Findings)
	}
}

func TestReadOnlyFailsEmptyCollectedSet(t *testing.T) {
	src := `package p
const x = "not-an-operation"
`
	r := checkReadonly(treeOf(File{Path: "internal/xsource/x.go", Src: []byte(src)}))
	if !hasDetail(r, "anti-vacuity") && !hasDetail(r, "collected 0") {
		t.Fatalf("want anti-vacuity, got %+v", r.Findings)
	}
}

func TestReadOnlyFailsEmptyTree(t *testing.T) {
	r := checkReadonly(nil)
	if len(r.Findings) == 0 {
		t.Fatal("empty tree passed")
	}
}

// TestReadOnlyFailsOnAnUndecodableRegistry: the registry is data this check
// reads, and a read that fails used to hand it an empty name set in silence, so
// a name only the registry knows stopped being collected without a word.
func TestReadOnlyFailsOnAnUndecodableRegistry(t *testing.T) {
	r := checkReadonly(treeOf(
		File{Path: registryJSONFile, Src: []byte(`{"operations": {`)},
		File{Path: "internal/xsource/x.go", Src: []byte("package p\n\nconst op = \"HomeTimeline\"\n")},
	))
	if !hasDetail(r, "does not decode") {
		t.Fatalf("an undecodable registry passed in silence: %+v", r.Findings)
	}
	var named bool
	for _, f := range r.Findings {
		if f.Path == registryJSONFile {
			named = true
		}
	}
	if !named {
		t.Fatalf("the finding does not name the registry: %+v", r.Findings)
	}
}

// TestOpNamesFailsOnAnEntryTheRegistryDoesNotHold is the gap this check was written for: a table
// entry whose operation the snapshot no longer holds. The finding names the classification the table
// promised, the entry itself, and the file the table lives in — which is where the entry has to
// come out.
func TestOpNamesFailsOnAnEntryTheRegistryDoesNotHold(t *testing.T) {
	reg := `{"operations":{"Bookmarks":{"queryId":"x"},"Likes":{"queryId":"y"}}}`
	r := checkOpNames(treeOf(File{Path: registryJSONFile, Src: []byte(reg)}))
	if len(r.Findings) == 0 {
		t.Fatal("a table whose names the snapshot does not hold passed")
	}
	var named bool
	for _, f := range r.Findings {
		if f.Path == "internal/audit/readonly.go" && strings.Contains(f.Detail, `"Viewer"`) && strings.Contains(f.Detail, "read") {
			named = true
		}
	}
	if !named {
		t.Fatalf("no finding names the table's own file and an entry the registry lacks: %+v", r.Findings)
	}
}

// TestOpNamesPassesOnTheRealTableAndSnapshot reads the two real files through this check alone, so
// the arm fails the moment either drifts; and it asserts the registry was read, because a pass over
// a name set nobody read would be the vacuous one.
func TestOpNamesPassesOnTheRealTableAndSnapshot(t *testing.T) {
	tree, err := Load(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	r := checkOpNames(tree)
	if len(r.Findings) != 0 {
		t.Fatalf("the classification table and the registry disagree: %+v", r.Findings)
	}
	if r.Files == 0 {
		t.Fatal("the check read no registry file: the pass would be vacuous")
	}
}

// TestOpNamesFailsEmptyTree: a tree with no registry file in it is a finding, not a pass.
func TestOpNamesFailsEmptyTree(t *testing.T) {
	if r := checkOpNames(nil); len(r.Findings) == 0 {
		t.Fatal("an empty tree passed")
	}
}

// TestOpNamesFailsOnAnUndecodableRegistry: the snapshot is data, so a read that failed is said
// rather than turned into "every name is missing".
func TestOpNamesFailsOnAnUndecodableRegistry(t *testing.T) {
	r := checkOpNames(treeOf(File{Path: registryJSONFile, Src: []byte(`{"operations": {`)}))
	if !hasDetail(r, "does not decode") {
		t.Fatalf("an undecodable registry passed in silence: %+v", r.Findings)
	}
}

// TestOpNamesFailsOnAnEmptyTable is the anti-vacuity arm: an empty table satisfies the subset
// trivially, so it is a finding of its own.
func TestOpNamesFailsOnAnEmptyTable(t *testing.T) {
	saved := opClass
	opClass = map[string]string{}
	t.Cleanup(func() { opClass = saved })
	r := checkOpNames(treeOf(File{Path: registryJSONFile, Src: []byte(`{"operations":{"Bookmarks":{"queryId":"x"}}}`)}))
	if !hasDetail(r, "empty") {
		t.Fatalf("an empty classification table passed: %+v", r.Findings)
	}
}

// TestTransportFailsNonGET is the POST arm as it now stands: a POST outside the one declared
// exemption file fails, both as the method it was built with and as the constant it spells.
func TestTransportFailsNonGET(t *testing.T) {
	src := `package p
import "net/http"
func f() { _, _ = http.NewRequest(http.MethodPost, "http://x", nil) }
`
	r := checkTransport(treeOf(File{Path: "internal/xweb/x.go", Src: []byte(src)}))
	if !hasDetail(r, "outside the declared exemption") {
		t.Fatalf("want a POST-outside-the-exemption finding, got %+v", r.Findings)
	}
	if !hasDetail(r, "MethodPost") {
		t.Fatalf("want the MethodPost constant named, got %+v", r.Findings)
	}
}

// TestTransportPassesAPostInsideTheExemption is the other half: the same POST in the one file
// that owns the per-operation method decision passes, and the exemption is printed in the
// promise rather than tolerated in silence.
func TestTransportPassesAPostInsideTheExemption(t *testing.T) {
	src := `package p
import "net/http"
func f() { _, _ = http.NewRequest(http.MethodPost, "http://x", nil) }
`
	r := checkTransport(treeOf(File{Path: methodExemptionFile, Src: []byte(src)}))
	if len(r.Findings) != 0 {
		t.Fatalf("a POST inside the exemption failed: %+v", r.Findings)
	}
	if !strings.Contains(r.Promise, methodExemptionFile) {
		t.Fatalf("the promise does not name the exemption it accepted: %q", r.Promise)
	}
}

// TestTransportFailsDeleteAnywhere: DELETE is not a method this tree may spell at all, in the
// exemption file or anywhere else.
func TestTransportFailsDeleteAnywhere(t *testing.T) {
	src := `package p
import "net/http"
func f() { _, _ = http.NewRequestWithContext(nil, http.MethodDelete, "http://x", nil) }
`
	for _, path := range []string{methodExemptionFile, "internal/netcheck/x.go"} {
		r := checkTransport(treeOf(File{Path: path, Src: []byte(src)}))
		if !hasDetail(r, "neither GET nor POST") && !hasDetail(r, "MethodDelete") {
			t.Fatalf("%s: want a DELETE finding, got %+v", path, r.Findings)
		}
	}
}

// TestTransportFailsPutLiteral: a method literal is read too, not just the http.Method*
// constants.
func TestTransportFailsPutLiteral(t *testing.T) {
	src := `package p
import "net/http"
func f() { _, _ = http.NewRequest("PUT", "http://x", nil) }
`
	r := checkTransport(treeOf(File{Path: methodExemptionFile, Src: []byte(src)}))
	if !hasDetail(r, "neither GET nor POST") {
		t.Fatalf("want a PUT finding, got %+v", r.Findings)
	}
}

// TestTransportFailsOnAMissingExemption is the anti-vacuity arm of the POST rule: a tree that
// does not hold the exemption file cannot have exercised the exemption, so that is a finding
// rather than a pass.
func TestTransportFailsOnAMissingExemption(t *testing.T) {
	src := `package p
import "net/http"
func f() { _, _ = http.NewRequestWithContext(nil, http.MethodGet, "http://x", nil) }
`
	r := checkTransport(treeOf(File{Path: "internal/xweb/x.go", Src: []byte(src)}))
	if !hasDetail(r, "exemption") || !hasDetail(r, "not in the tree") {
		t.Fatalf("want a missing-exemption finding, got %+v", r.Findings)
	}
}

func TestTransportFailsBuilderOutside(t *testing.T) {
	src := `package p
import "net/http"
func f() { _, _ = http.NewRequestWithContext(nil, http.MethodGet, "http://x", nil) }
`
	r := checkTransport(treeOf(File{Path: "internal/cli/x.go", Src: []byte(src)}))
	if !hasDetail(r, "outside") {
		t.Fatalf("want outside-builder finding, got %+v", r.Findings)
	}
}

func TestTransportPassesRealShape(t *testing.T) {
	src := `package p
import "net/http"
func f() { _, _ = http.NewRequestWithContext(nil, http.MethodGet, "http://x", nil) }
`
	r := checkTransport(treeOf(File{Path: "internal/xweb/graphql.go", Src: []byte(src)}))
	if len(r.Findings) != 0 {
		t.Fatalf("GET inside xweb failed: %+v", r.Findings)
	}
}

func hasDetail(r Report, sub string) bool { return findingsSay(r.Findings, sub) }

// findingsSay is the same test over a bare finding slice, for Sweep, which
// returns findings rather than a report.
func findingsSay(findings []Finding, sub string) bool {
	for _, f := range findings {
		if strings.Contains(f.Detail, sub) {
			return true
		}
	}
	return false
}

func hasPathLine(r Report, path string) bool {
	for _, f := range r.Findings {
		if f.Path == path && f.Line > 0 {
			return true
		}
	}
	return false
}
