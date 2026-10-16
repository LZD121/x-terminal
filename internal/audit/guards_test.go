package audit

import (
	"fmt"
	"strings"
	"testing"
)

func TestTimersFailOnTicker(t *testing.T) {
	src := `package p
import "time"
func f() { _ = time.NewTicker(time.Second) }
`
	r := checkTimers(treeOf(
		File{Path: "internal/cli/x.go", Src: []byte(src)},
		File{Path: limiterFloorFile, Src: []byte(`package xweb
import "time"
func sleepContext() { _ = time.NewTimer(0) }
`)},
	))
	if !hasDetail(r, "NewTicker") {
		t.Fatalf("want NewTicker finding, got %+v", r.Findings)
	}
}

func TestTimersFailWhenExemptionMissing(t *testing.T) {
	src := `package p
func f() {}
`
	r := checkTimers(treeOf(File{Path: "internal/cli/x.go", Src: []byte(src)}))
	if !hasDetail(r, "exemption") {
		t.Fatalf("want missing-exemption finding, got %+v", r.Findings)
	}
}

func TestTimersPassWithOnlyExemption(t *testing.T) {
	r := checkTimers(treeOf(File{Path: limiterFloorFile, Src: []byte(`package xweb
import "time"
func sleepContext() { t := time.NewTimer(0); _ = t }
`)}))
	if len(r.Findings) != 0 {
		t.Fatalf("exemption alone failed: %+v", r.Findings)
	}
	if !strings.Contains(r.Promise, limiterFloorFile) {
		t.Fatalf("exemption not printed in report: %q", r.Promise)
	}
}

// TestPersistencePassesDeclaredSet: the committed shape of every destination,
// and the promise that names them with the call site that reached each.
func TestPersistencePassesDeclaredSet(t *testing.T) {
	r := checkPersistence(persistenceBaseTree())
	if len(r.Findings) != 0 {
		t.Fatalf("the declared set failed: %+v", r.Findings)
	}
	for _, d := range writeDestinations {
		if !strings.Contains(r.Promise, d.Name+"@") {
			t.Errorf("destination %q is not printed with the call site that reached it: %q", d.Name, r.Promise)
		}
	}
	if want := fmt.Sprintf("%d named destinations", len(writeDestinations)); !strings.Contains(r.Promise, want) {
		t.Errorf("promise does not state %q: %q", want, r.Promise)
	}
}

// TestPersistenceFailsFifth: a write outside the declared set is a finding that
// carries its own file and line — in every file, including the three the scan
// used to skip.
func TestPersistenceFailsFifth(t *testing.T) {
	const planted = `package p
import "os"
func f() { _ = os.WriteFile("/tmp/extra.log", nil, 0o600) }
`
	for _, path := range []string{"internal/cli/extra.go", "cmd/x-opsgen/main.go", "cmd/spike/main.go"} {
		t.Run(path, func(t *testing.T) {
			base := append(persistenceBaseTree(), File{Path: path, Src: []byte(planted)})
			r := checkPersistence(base)
			if !hasDetail(r, "no declared destination") {
				t.Fatalf("want an undeclared-destination finding, got %+v", r.Findings)
			}
			var named bool
			for _, f := range r.Findings {
				if f.Path == path && f.Line == 3 {
					named = true
				}
			}
			if !named {
				t.Fatalf("the finding for %s does not carry file:line: %+v", path, r.Findings)
			}
		})
	}
}

// TestPersistenceFailsWriteBesideTheRecorder: the spike's recording destination
// is the recorder function, not the file: a second write in the same file is a
// new destination and must be reported.
func TestPersistenceFailsWriteBesideTheRecorder(t *testing.T) {
	base := persistenceBaseTree()
	for i := range base {
		if base[i].Path == spikeRecorderFile {
			base[i].Src = append([]byte(base[i].Src), []byte("\nfunc helper(p string, b []byte) { _ = os.WriteFile(p, b, 0o600) }\n")...)
		}
	}
	r := checkPersistence(base)
	if !hasDetail(r, "no declared destination") {
		t.Fatalf("a write beside the recorder was not reported: %+v", r.Findings)
	}
	for _, f := range r.Findings {
		if f.Path != spikeRecorderFile {
			t.Errorf("finding on the wrong file: %+v", f)
		}
	}
}

func TestPersistenceFailsWhenDestinationMissing(t *testing.T) {
	r := checkPersistence(treeOf(File{Path: "internal/cli/x.go", Src: []byte("package p\n")}))
	if !hasDetail(r, "reached by no write call site") {
		t.Fatalf("want missing-destination finding, got %+v", r.Findings)
	}
	for _, d := range writeDestinations {
		if !hasDetail(r, d.Name) {
			t.Errorf("the missing-destination findings never name %q: %+v", d.Name, r.Findings)
		}
	}
}

// TestPersistenceFailsZeroWriteTree: the anti-vacuity arm is reached by a tree that
// genuinely holds no write-capable call, and the arm's own message is what is
// asserted — not a destination finding the arm used to hide behind.
func TestPersistenceFailsZeroWriteTree(t *testing.T) {
	r := checkPersistence(treeOf(File{Path: "internal/cli/x.go", Src: []byte("package p\nfunc f() {}\n")}))
	if !hasDetail(r, "found 0 write-capable call sites") {
		t.Fatalf("a tree with no write-capable call produced no anti-vacuity finding: %+v", r.Findings)
	}
	// The same tree also leaves every declared destination unreached: both
	// statements are true of it, and its own emptiness is reported as itself
	// rather than only through the destinations behind it.
	if !hasDetail(r, "reached by no write call site") {
		t.Errorf("the destinations the zero-write tree left unreached are not reported: %+v", r.Findings)
	}
}

// persistenceDestinationNames is the declared write set, written out by hand in the
// order the promise prints it: the running tool's five, then the checkout's two.
// The promise line and the JSON twin are read as this set, so a destination
// renamed, folded into another or added changes this list or fails the build — and
// "session" is deliberately not in it: the credential file is read and never
// written, and a promise that named it would claim a write that does not exist
// while leaving the config file's write unnamed.
var persistenceDestinationNames = []string{"config", "pins", "notice", "saved", "update", "registry", "spike-recording"}

// TestPersistenceDestinationSetIsPinned: the printed promise names the set the check
// enforces, by name — not by a count both sides read out of the same table.
func TestPersistenceDestinationSetIsPinned(t *testing.T) {
	var got []string
	for _, d := range writeDestinations {
		got = append(got, d.Name)
	}
	if strings.Join(got, ",") != strings.Join(persistenceDestinationNames, ",") {
		t.Fatalf("declared destinations are %v, want %v", got, persistenceDestinationNames)
	}
	want := fmt.Sprintf("every write resolves to one of the %d named destinations: config, pins, notice, saved, update (the running tool); registry, spike-recording (a development tool)", len(persistenceDestinationNames))
	if promise := persistencePromise(); promise != want {
		t.Fatalf("the printed promise does not name exactly the enforced set:\n got: %s\nwant: %s", promise, want)
	}
}

// TestPersistenceFailsAWriteInANewConfigFile: the config destination is anchored to
// the file that writes the config file, not to the directory: a second writer beside
// it is a new destination carrying its own file:line, whatever it names. This is the
// write the check used to accept silently, because any write under internal/config/
// that mentioned the session file — or called writeFile/os.CreateTemp — resolved to
// a declared destination.
func TestPersistenceFailsAWriteInANewConfigFile(t *testing.T) {
	const planted = `package config
import "os"
func saveSession(path string) error { return os.WriteFile(path, nil, 0o600) }
const sessionFileName = "session"
`
	base := append(persistenceBaseTree(), File{Path: "internal/config/write.go", Src: []byte(planted)})
	r := checkPersistence(base)
	if !hasDetail(r, "no declared destination") {
		t.Fatalf("a write beside the config writer resolved to a declared destination: %+v", r.Findings)
	}
	var named bool
	for _, f := range r.Findings {
		if f.Path == "internal/config/write.go" && f.Line == 3 {
			named = true
		}
	}
	if !named {
		t.Fatalf("the finding for the new config writer does not carry its file:line: %+v", r.Findings)
	}
}

// TestPersistenceFailsAWriteBesideTheInstaller: the update verb's destination is
// anchored to the file that writes it (internal/release/install.go), not to the
// package it lives in: a second file beside it is a new destination carrying its own
// file:line. That pair — this and TestPersistenceFailsAWriteInANewConfigFile above —
// is what makes the anchor the file rather than the directory.
func TestPersistenceFailsAWriteBesideTheInstaller(t *testing.T) {
	const planted = `package release
import "os"
func other(path string, b []byte) error { return os.WriteFile(path, b, 0o755) }
`
	base := append(persistenceBaseTree(), File{Path: "internal/release/other.go", Src: []byte(planted)})
	r := checkPersistence(base)
	if !hasDetail(r, "no declared destination") {
		t.Fatalf("a write beside the installer resolved to a declared destination: %+v", r.Findings)
	}
	var named bool
	for _, f := range r.Findings {
		if f.Path == "internal/release/other.go" && f.Line == 3 {
			named = true
		}
	}
	if !named {
		t.Fatalf("the finding for the write beside the installer does not carry its file:line: %+v", r.Findings)
	}
}

// persistenceBaseTree is the committed shape of every declared destination: one
// write call site per destination, reached the way the real tree reaches it.
func persistenceBaseTree() Tree {
	return treeOf(
		File{Path: "internal/config/config.go", Src: []byte("package config\nimport (\n\"os\"\n\"path/filepath\"\n)\nconst sessionFileName = \"session\"\nfunc (c Config) PinsPath() string { return \"pins.json\" }\nfunc (c Config) NoticePath() string { return \"notice-shown\" }\nfunc (c Config) SavedDBPath() string { return \"saved.db\" }\nfunc writeFile(path, content string) error { return nil }\nfunc save(path, content string) error { return writeFile(path, content) }\nfunc saveTemp(path string) error {\n_, err := os.CreateTemp(filepath.Dir(path), \"config.toml.tmp\")\nreturn err\n}\n")},
		File{Path: "internal/cli/notice.go", Src: []byte("package cli\nimport \"os\"\nfunc mark(path string) { _ = os.WriteFile(path, []byte(\"notice-shown\"), 0o600) }\n")},
		File{Path: "internal/xweb/volatile/pins.go", Src: []byte("package volatile\nfunc writeAtomically() {}\nfunc Write(path string) error { return writeAtomically(path, nil, 0o600) }\nconst x = \"pins.json\"\n")},
		File{Path: "internal/store/schema.go", Src: []byte("package store\nimport \"database/sql\"\nfunc Open() { _, _ = sql.Open(\"sqlite\", \"saved.db\") }\n")},
		File{Path: registryWriterFile, Src: []byte("package volatile\nimport (\n\"os\"\n\"path/filepath\"\n)\nfunc (r *Registry) Write(path string) error { return writeAtomically(path, nil, 0o644) }\nfunc writeAtomically(path string, data []byte, mode os.FileMode) error { _, err := os.CreateTemp(filepath.Dir(path), \"tmp\"); return err }\n")},
		File{Path: spikeRecorderFile, Src: []byte("package main\nimport (\n\"os\"\n\"path/filepath\"\n)\nfunc recorder(dir string) func(string, []byte) {\nreturn func(n string, b []byte) { _ = os.WriteFile(filepath.Join(dir, n), b, 0o600) }\n}\n")},
		File{Path: updaterWriterFile, Src: []byte("package release\nimport (\n\"os\"\n\"path/filepath\"\n)\nfunc Replace(target string, payload []byte) error {\ntmp, err := os.CreateTemp(filepath.Dir(target), \".x-terminal-update-*\")\nif err != nil {\nreturn err\n}\nreturn os.Rename(tmp, target)\n}\n")},
	)
}

func TestDistributionFailsContainerName(t *testing.T) {
	r := checkDistribution(treeOf(File{Path: "Dockerfile", Src: []byte("FROM scratch\n")}))
	if !hasDetail(r, "distributable artifact") {
		t.Fatalf("want Dockerfile finding, got %+v", r.Findings)
	}
}

// TestDistributionAllowsAReleaseWorkflow: the promise was narrowed on purpose. A private
// repository's CI/release job is the build this project already was, not a distributable form of
// it, so the check must not report one — and it must still report an image, a package recipe or an
// installer, which is what the two arms below prove. Both halves matter: without the first the
// check lies about the repository, and without the second it would pass for the wrong reason.
func TestDistributionAllowsAReleaseWorkflow(t *testing.T) {
	tree := treeOf(
		File{Path: ".github/workflows/release.yml", Src: []byte("name: release\n")},
		File{Path: ".github/workflows/ci.yml", Src: []byte("name: ci\n")},
	)
	if r := checkDistribution(tree); len(r.Findings) != 0 {
		t.Fatalf("a release workflow is allowed under the narrowed promise, got %+v", r.Findings)
	}
}

func TestDistributionFailsPublishingAWorkflow(t *testing.T) {
	for _, path := range []string{
		".github/workflows/publish-image.yml",
		".github/workflows/build-container.yml",
		".github/workflows/package-pkg.yml",
		"scripts/release.sh",
		"PKGBUILD",
		".goreleaser.yml",
	} {
		r := checkDistribution(treeOf(File{Path: path, Src: []byte("x\n")}))
		if len(r.Findings) == 0 {
			t.Errorf("%s is a distributable form of this program and produced no finding", path)
		}
	}
}

func TestDistributionFailsListener(t *testing.T) {
	src := `package p
import "net"
func f() { _, _ = net.Listen("tcp", ":8080") }
`
	r := checkDistribution(treeOf(File{Path: "internal/cli/x.go", Src: []byte(src)}))
	if !hasDetail(r, "Listen") && !hasDetail(r, "bind address") {
		t.Fatalf("want listener finding, got %+v", r.Findings)
	}
}

func TestAccountFailsSecondSessionPath(t *testing.T) {
	cfg := `package config
var _ = Config{SessionPath: "a"}
var _ = Config{SessionPath: "b"}
`
	sess := `package session
func Load(path string) {}
`
	extra := `package other
func LoadSession(path string) {}
var _ = struct{ SessionPath string }{SessionPath: "c"}
`
	// Plant a third SessionPath key outside config to trip "second session-path".
	r := checkAccount(treeOf(
		File{Path: "internal/config/config.go", Src: []byte(cfg)},
		File{Path: "internal/session/session.go", Src: []byte(sess + "\nconst a = \"auth_token\"\nconst c = \"ct0\"\n")},
		File{Path: "internal/config/extra.go", Src: []byte(`package config
var _ = Config{SessionPath: "second"}
`)},
		File{Path: "internal/other/x.go", Src: []byte(extra)},
	))
	_ = r
	// The account check counts SessionPath key-values in config.go only plus Load.
	// Plant a second Load-shaped resolution by duplicating SessionPath defaults.
	r = checkAccount(treeOf(
		File{Path: "internal/config/config.go", Src: []byte(`package config
var a = Config{SessionPath: "one"}
var b = Config{SessionPath: "two"}
var c = Config{SessionPath: "three"}
`)},
		File{Path: "internal/session/session.go", Src: []byte(`package session
func Load(path string) {}
const x = "auth_token"
const y = "ct0"
`)},
	))
	if !hasDetail(r, "second session-path") {
		t.Fatalf("want second session-path, got %+v", r.Findings)
	}
}

func TestAccountFailsZeroSessionPath(t *testing.T) {
	r := checkAccount(treeOf(File{Path: "internal/cli/x.go", Src: []byte("package p\n")}))
	if !hasDetail(r, "0 session-path") {
		t.Fatalf("want zero session-path finding, got %+v", r.Findings)
	}
}

func TestBoundariesFailOnTUIImport(t *testing.T) {
	src := "package tui\nimport \"net/http\"\n"
	r := checkBoundaries(treeOf(
		File{Path: "internal/tui/view.go", Src: []byte(src)},
		File{Path: "internal/term/term.go", Src: []byte("package term\nimport \"os/exec\"\n")},
		File{Path: "internal/media/spawn.go", Src: []byte("package media\nimport \"os/exec\"\n")},
	))
	if !hasDetail(r, "net/http") {
		t.Fatalf("want tui import finding, got %+v", r.Findings)
	}
}
