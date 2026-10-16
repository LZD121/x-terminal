package audit

import (
	"strings"
	"testing"
)

func TestLeakSweepControlAndEmpty(t *testing.T) {
	r := Sweep("leak", nil)
	if !findingsSay(r, "empty") {
		t.Fatalf("empty surfaces should fail: %+v", r)
	}
}

func TestLeakSweepFindsCookieInSurface(t *testing.T) {
	r := Sweep("leak", []Surface{{
		Name: "bad",
		Body: "token=" + FixtureAuthToken,
	}})
	if !findingsSay(r, "auth_token") {
		t.Fatalf("want cookie finding, got %+v", r)
	}
}

func TestLeakSweepCleanSurfaces(t *testing.T) {
	r := Sweep("leak", []Surface{
		{Name: "doctor", Body: "session: present (auth_token 40 bytes)"},
		{Name: "sync", Body: "bookmarks pages 1 items 1"},
		{Name: "audit", Body: FormatReports(nil)},
		{Name: "media", Body: "opened image\nno media in this post"},
		{Name: "store", Body: "store: open saved.db: refused"},
		{Name: "panic", Body: "panic: audit leak surface"},
	})
	if len(r) != 0 {
		t.Fatalf("clean surfaces failed: %+v", r)
	}
}

// TestLeakFailsWhenASourceSurfaceIsAbsent: the promise names the media-notice
// constants and the store sources as surfaces read from the tree. A tree that
// carries neither is a tree this check did not search them over, and saying
// nothing would be the widest kind of silent pass.
func TestLeakFailsWhenASourceSurfaceIsAbsent(t *testing.T) {
	r := checkLeak(treeOf(File{Path: "internal/cli/x.go", Src: []byte("package cli\n")}))
	for _, want := range []string{"media-notice-constants", "store-source"} {
		if !hasDetail(r, want) {
			t.Fatalf("the %s surface is absent and the check said nothing: %+v", want, r.Findings)
		}
	}
}

// TestLeakCheckOnRealTree pins the check's own surfaces: which are this run's
// output, which are read from the tree, and that the promise says where the
// rest are swept. The stand-ins that used to live here — hand-written
// doctor/sync-shaped strings — searched text this test invented, which is
// coverage of nothing; the surfaces the CLI's code produces are driven in
// internal/cli/leak_test.go.
func TestLeakCheckOnRealTree(t *testing.T) {
	tree, err := Load(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	r := checkLeak(tree)
	if len(r.Findings) != 0 {
		t.Fatalf("leak findings: %+v", r.Findings)
	}
	if r.Files == 0 {
		t.Fatal("leak inspected 0 files")
	}

	surfaces := defaultLeakSurfaces(tree)
	names := map[string]bool{}
	for _, s := range surfaces {
		if s.Body == "" {
			t.Errorf("surface %q is empty: the sweep passes over it whatever it holds", s.Name)
		}
		names[s.Name] = true
	}
	for _, want := range []string{
		"audit-report(output of this run)",
		"recovered-panic(output of this run)",
		"media-notice-constants(source internal/render/copy.go)",
	} {
		if !names[want] {
			t.Errorf("surface %q missing from the sweep: %v", want, names)
		}
	}
	var storeSurfaces int
	for name := range names {
		if strings.HasPrefix(name, "store-source(source internal/store/") {
			storeSurfaces++
		}
	}
	if storeSurfaces == 0 {
		t.Errorf("no store source surface: the store half of the sweep is gone")
	}
	// The source-derived surfaces say so in their names and in the promise, so
	// the printed claim is not wider than what was searched.
	for _, s := range surfaces {
		if strings.Contains(s.Name, "(source ") && !strings.Contains(r.Promise, "read from the tree") {
			t.Fatalf("promise does not admit the source-derived surfaces: %q", r.Promise)
		}
	}
	if !strings.Contains(r.Promise, "internal/cli's leak test") {
		t.Fatalf("promise does not say where the CLI-owned output surfaces are swept: %q", r.Promise)
	}
	if !strings.Contains("a "+FixtureAuthToken+" b", FixtureAuthToken) {
		t.Fatal("control search broken")
	}
}

// TestLeakSweepFailsOnPlantedSourceSurface: the sweep the audit runs must fail
// on the very surfaces it builds when one of them carries the cookie.
func TestLeakSweepFailsOnPlantedSourceSurface(t *testing.T) {
	tree, err := Load(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	surfaces := defaultLeakSurfaces(tree)
	planted := make([]Surface, len(surfaces))
	copy(planted, surfaces)
	for i := range planted {
		if strings.HasPrefix(planted[i].Name, "media-notice-constants(") {
			planted[i].Body += "\n" + FixtureAuthToken
		}
	}
	if got := Sweep("leak", planted); !findingsSay(got, "auth_token") {
		t.Fatalf("a cookie planted into the media-notice surface was not caught: %+v", got)
	}
}
