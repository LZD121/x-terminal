// Package audit is the repo-wide proof that the promises this build makes still
// hold: every file it reads as source parses, read-only, every operation name
// the classification table knows resolvable, no mutation expressible, no timers,
// one persistence file, no distributable form of this program and no hosted mode, one account,
// the cookie sweep, and the import boundaries.
//
// The package reads files as data and nothing else — no internal/store, no
// internal/xweb, no shell, no process — so that a broken build can still be
// audited. Its report never carries a post body, a token or a path outside the
// repository, because the report is the thing a user pastes when something is
// wrong.
//
// Each check is a pure function over a file list. A scan that inspected no files
// is itself a finding: "no findings" over an empty file list is the one output
// that would make the whole audit worthless — and so is a scan that quietly
// dropped a file it could not parse, which is why the source check reports every
// Go file that does not parse and every Go scan counts only the files it parsed.
package audit

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// File is one repository file the checks may read. Path is relative to the
// repository root and uses `/` separators.
type File struct {
	Path string
	Src  []byte
}

// Tree is the files a check was handed. Load walks the repository once; each
// check may filter the list further.
type Tree []File

// Finding is one failure of one check. Line is 0 when the finding is about the
// file as a whole.
type Finding struct {
	Check  string
	Path   string
	Detail string
	Line   int
}

// Report is what one check returns: the promise it holds, how many files it
// inspected, and every finding.
type Report struct {
	Check    string
	Promise  string
	Files    int
	Findings []Finding
}

// Check is one promise expressed as a pure function over a Tree.
type Check struct {
	Name    string
	Promise string
	Run     func(Tree) Report
}

// manifestGlobs are the non-Go files a check may need: the registry JSON (data,
// exempt from the write-name rule), packaging and release artefacts, and CI
// workflows.
var manifestGlobs = []string{
	registryJSONFile,
	"go.mod",
	"go.sum",
	"Dockerfile",
	"Dockerfile.*",
	"PKGBUILD",
	".goreleaser",
	".goreleaser.*",
	"*.goreleaser.yml",
	"*.goreleaser.yaml",
	"install.sh",
	"install",
	"release.sh",
	"scripts/install*",
	"scripts/release*",
	".github/workflows/*",
}

// Load walks root for every non-test .go file of the module plus the manifests
// a check may need. Dot-directories other than .github are skipped; testdata is
// skipped because it is fixture material, not the module.
func Load(root string) (Tree, error) {
	if root == "" {
		return nil, fmt.Errorf("audit: no repository root")
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("audit: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("audit: %s is not a directory", root)
	}

	seen := map[string]bool{}
	var tree Tree
	add := func(rel string) error {
		rel = filepath.ToSlash(rel)
		if seen[rel] {
			return nil
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("audit: reading %s: %w", rel, err)
		}
		seen[rel] = true
		tree = append(tree, File{Path: rel, Src: data})
		return nil
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			base := filepath.Base(path)
			switch {
			case rel == ".":
				return nil
			case base == "testdata":
				return fs.SkipDir
			case base == ".git" || base == ".planning" || base == ".agents":
				return fs.SkipDir
			case strings.HasPrefix(base, ".") && base != ".github":
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") {
			return nil
		}
		if strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		return add(rel)
	})
	if err != nil {
		return nil, fmt.Errorf("audit: walking %s: %w", root, err)
	}

	for _, pattern := range manifestGlobs {
		matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
		if err != nil {
			return nil, fmt.Errorf("audit: glob %s: %w", pattern, err)
		}
		for _, match := range matches {
			rel, err := filepath.Rel(root, match)
			if err != nil {
				return nil, err
			}
			if err := add(rel); err != nil {
				return nil, err
			}
		}
	}

	return tree, nil
}

// GoFiles returns the non-test Go sources in t.
func (t Tree) GoFiles() Tree {
	out := make(Tree, 0, len(t))
	for _, f := range t {
		if strings.HasSuffix(f.Path, ".go") {
			out = append(out, f)
		}
	}
	return out
}

// Checks is every check this package runs, in fixed report order.
func Checks() []Check {
	return []Check{
		{Name: "source", Promise: sourcePromise, Run: checkSource},
		{Name: "readonly", Promise: "no write operation is nameable in non-test source", Run: checkReadonly},
		{Name: "opnames", Promise: "every operation name the classification table knows is one the registry resolves", Run: checkOpNames},
		{Name: "transport", Promise: "no mutation is expressible over the wire", Run: checkTransport},
		{Name: "timers", Promise: "no timer or sleep outside the limiter's floor", Run: checkTimers},
		{Name: "persistence", Promise: persistencePromise(), Run: checkPersistence},
		{Name: "distribution", Promise: "no distributable form of this program (image, package, installer) and no hosted mode", Run: checkDistribution},
		{Name: "account", Promise: "exactly one session path and one credential pair", Run: checkAccount},
		{Name: "boundaries", Promise: "import boundaries and volatile vocabulary still hold", Run: checkBoundaries},
		{Name: "leak", Promise: "the fixture cookie appears in none of the named surfaces", Run: checkLeak},
	}
}

// Run loads the tree at root once and runs every check. A check that panics is
// caught and becomes a finding — an audit that dies with a panic has audited
// nothing.
func Run(root string) ([]Report, error) {
	tree, err := Load(root)
	if err != nil {
		return nil, err
	}
	reports := make([]Report, 0, len(Checks()))
	for _, c := range Checks() {
		reports = append(reports, runCheck(c, tree))
	}
	return reports, nil
}

func runCheck(c Check, tree Tree) (report Report) {
	defer func() {
		if r := recover(); r != nil {
			report = Report{
				Check:   c.Name,
				Promise: c.Promise,
				Files:   0,
				Findings: []Finding{{
					Check:  c.Name,
					Detail: fmt.Sprintf("check panicked: %v", r),
				}},
			}
		}
	}()
	if c.Run == nil {
		return Report{
			Check:   c.Name,
			Promise: c.Promise,
			Findings: []Finding{{
				Check:  c.Name,
				Detail: "check has no Run function",
			}},
		}
	}
	report = c.Run(tree)
	if report.Check == "" {
		report.Check = c.Name
	}
	if report.Promise == "" {
		report.Promise = c.Promise
	}
	return report
}

func emptyFinding(check, detail string) Finding {
	return Finding{Check: check, Detail: detail}
}

func fileFinding(check, path, detail string, line int) Finding {
	return Finding{Check: check, Path: path, Detail: detail, Line: line}
}
