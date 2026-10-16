package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LZD121/x-terminal/internal/audit"
)

// auditRun is the seam tests replace: production loads and runs the checks
// against the repository root. The shape matches feedSource — a package-level
// var so a test drives findings without rewriting the tree.
var auditRun = audit.Run

// repoRoot resolves the repository root the way doctor --refresh-ids resolves
// the registry path: the program's own working directory, with a named error
// when the tree is not there.
func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("audit: working directory: %w", err)
	}
	if _, err := os.Stat(filepath.Join(wd, "go.mod")); err != nil {
		return "", fmt.Errorf("audit: %s is not the repository root (no go.mod): run from the x-terminal checkout", wd)
	}
	if _, err := os.Stat(filepath.Join(wd, "internal", "audit")); err != nil {
		return "", fmt.Errorf("audit: %s has go.mod but no internal/audit: not this repository", wd)
	}
	return wd, nil
}

// Audit is the `audit` verb: run every check against the working tree, print
// one line per check with the number of files it inspected and every finding as
// file and line, and exit non-zero when anything is found. A run that inspected
// no files is itself a finding.
//
// The report is bounded: no post text, no token and no path outside the
// repository. stdout carries only the report; every diagnostic goes to stderr.
func Audit(ctx *Context, args []string) int {
	var asJSON bool
	for _, arg := range args {
		switch arg {
		case "--json":
			asJSON = true
		default:
			fmt.Fprintf(ctx.Stderr, "audit: unknown argument %s\n", arg)
			usage(ctx.Stderr)
			return ExitUsage
		}
	}

	root, err := repoRoot()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "%v\n", err)
		return ExitError
	}
	reports, err := auditRun(root)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "audit: %v\n", err)
		return ExitError
	}

	if asJSON {
		return auditJSON(ctx, reports)
	}
	text := audit.FormatReports(reports)
	// Bound: never an absolute path outside the repository.
	if abs := filepath.Clean(root); abs != "." && abs != "" {
		text = strings.ReplaceAll(text, abs+string(filepath.Separator), "")
		text = strings.ReplaceAll(text, abs, "")
	}
	fmt.Fprint(ctx.Stdout, text)
	if audit.FindingCount(reports) > 0 {
		return ExitError
	}
	return ExitOK
}

func auditJSON(ctx *Context, reports []audit.Report) int {
	type finding struct {
		Path   string `json:"path,omitempty"`
		Line   int    `json:"line,omitempty"`
		Detail string `json:"detail"`
	}
	type check struct {
		Name     string    `json:"name"`
		Promise  string    `json:"promise"`
		Files    int       `json:"files"`
		Findings []finding `json:"findings"`
	}
	out := struct {
		Checks   []check `json:"checks"`
		Findings int     `json:"findings"`
		Files    int     `json:"files"`
	}{}
	for _, r := range reports {
		// The promise carries whole: it is code-authored — a fixed sentence, the
		// declared destination names and one repository-relative path per write
		// call site — and the text report prints it whole too, so the JSON twin
		// states the same set. Only a finding's detail is clipped, because that
		// text can come from anywhere.
		c := check{Name: r.Check, Promise: r.Promise, Files: r.Files, Findings: []finding{}}
		out.Files += r.Files
		for _, f := range r.Findings {
			out.Findings++
			c.Findings = append(c.Findings, finding{Path: f.Path, Line: f.Line, Detail: clipAudit(f.Detail)})
		}
		out.Checks = append(out.Checks, c)
	}
	data, err := json.Marshal(out)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "audit: %v\n", err)
		return ExitError
	}
	fmt.Fprintln(ctx.Stdout, string(data))
	if out.Findings > 0 {
		return ExitError
	}
	return ExitOK
}

func clipAudit(s string) string {
	const limit = 200
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= limit {
		return s
	}
	return s[:limit-1] + "~"
}
