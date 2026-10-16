package audit

import (
	"fmt"
	"regexp"
	"strings"
)

// FixtureAuthToken is the same shape and value internal/session's leak test uses.
// It is a made-up value, so printing it back is diagnosis rather than disclosure.
const FixtureAuthToken = "0123456789abcdef0123456789abcdef01234567"

// FixtureCTO matches the session fixture's ct0 length (160 bytes of "b1c2d3e4").
const FixtureCTO = "b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4b1c2d3e4"

// Surface is one named output the fixture cookie must not appear in. A surface
// whose body is text read out of the tree rather than produced by a run says so
// in its name: this package may not import internal/store, internal/xweb or
// internal/cli (TestPackageImportsNeitherStoreNorXweb), so its own sweep can
// produce this run's report and a panic in-process and nothing else. The
// surfaces the CLI's own code produces — the doctor's report, the sync verb's
// output, a store error, a rendered media notice — are swept by internal/cli's
// leak test, which drives those producers and hands the captured text to Sweep.
type Surface struct {
	Name string
	Body string
}

// leakPromise is the line this check prints. It names what this run produces
// and what it only reads, and where the rest is swept, so the printed claim is
// the claim that was tested and not a wider one.
const leakPromise = "the fixture cookie appears in none of the named surfaces: audit-report and recovered-panic are this run's own output; media-notice constants and store sources are read from the tree; the CLI's own outputs are swept by internal/cli's leak test"

func checkLeak(t Tree) Report {
	const name = "leak"
	r := Report{Check: name, Promise: leakPromise, Files: len(t)}
	if len(t) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "inspected no files: a scan of nothing proves nothing"))
		return r
	}
	surfaces := defaultLeakSurfaces(t)
	r.Findings = append(r.Findings, Sweep(name, surfaces)...)
	// The promise names the media-notice constants and the store sources as
	// surfaces read from the tree, not produced by this run. A tree that no
	// longer carries one of them would be swept as one surface lighter — the
	// promise unchanged — so their absence is a finding rather than one less
	// search.
	var media, store int
	for _, s := range surfaces {
		switch {
		case strings.HasPrefix(s.Name, "media-notice-constants("):
			media++
		case strings.HasPrefix(s.Name, "store-source("):
			store++
		}
	}
	if media == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "no media-notice-constants surface: the constants the renderer prints are read out of internal/render/copy.go, and this tree does not carry it"))
	}
	if store == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "no store-source surface: the text the store's errors are built from is read out of internal/store, and this tree does not carry it"))
	}
	return r
}

// Sweep searches the named surfaces for the fixture credentials: the
// empty-surface arm, the control arm that proves the search finds what it is
// looking for, and the needle search itself. It is exported because the
// surfaces that matter are outputs produced by other packages' code — the sync
// verb's report, a store error, a rendered media notice, the doctor's report —
// and internal/cli, which can drive those producers, hands the captured text
// here rather than to a second and weaker search of its own.
func Sweep(check string, surfaces []Surface) []Finding {
	if len(surfaces) == 0 {
		return []Finding{emptyFinding(check, "surface list is empty: the cookie search has nothing to search")}
	}
	var findings []Finding
	var bodies []Surface
	for _, s := range surfaces {
		if s.Body == "" {
			findings = append(findings, emptyFinding(check, fmt.Sprintf("surface %q is empty: a search over an empty string proves nothing about it", s.Name)))
			continue
		}
		bodies = append(bodies, s)
	}

	// Control arm: the search must find the cookie in a string that contains it.
	control := "control " + FixtureAuthToken + " end"
	if !strings.Contains(control, FixtureAuthToken) {
		findings = append(findings, emptyFinding(check, "strings.Contains cannot find the fixture: the assertion below proves nothing"))
		return findings
	}

	// Each surface is searched on its own, so a finding names the surface that
	// carried the credential rather than only an offset into a concatenation.
	for _, s := range bodies {
		for _, credential := range []struct{ name, value string }{
			{"auth_token", FixtureAuthToken},
			{"ct0", FixtureCTO},
		} {
			if at := strings.Index(s.Body, credential.value); at >= 0 {
				findings = append(findings, emptyFinding(check, fmt.Sprintf("%s appears in surface %q at offset %d", credential.name, s.Name, at)))
			}
		}
	}
	return findings
}

// copyNoticeConst matches the notice copy table's string constants: the values
// the renderer prints, read as data so this package needs no import of the
// renderer. A rename that stops the match leaves the surface empty, and an
// empty surface is a finding rather than a silent pass.
var copyNoticeConst = regexp.MustCompile(`(?m)^\s*Copy(?:Cover)?[A-Za-z]*Notice[A-Za-z]*\s*=\s*"([^"]*)"`)

// defaultLeakSurfaces is what this package can produce and read without
// importing the packages that own the doctor, the sync verb and the store.
func defaultLeakSurfaces(t Tree) []Surface {
	var surfaces []Surface

	// The audit's own report over this tree (without the leak check, to avoid
	// recursion). This is output: the checks ran and this is what they said.
	var reports []Report
	for _, c := range Checks() {
		if c.Name == "leak" {
			continue
		}
		reports = append(reports, runCheck(c, t))
	}
	surfaces = append(surfaces, Surface{Name: "audit-report(output of this run)", Body: FormatReports(reports)})

	for _, f := range t {
		// Media notices: the copy table's constants, which are the strings the
		// renderer prints. The rendered lines are swept by internal/cli's leak
		// test, which calls render.NoticeLine.
		if f.Path == "internal/render/copy.go" {
			var notices []string
			for _, m := range copyNoticeConst.FindAllStringSubmatch(string(f.Src), -1) {
				notices = append(notices, m[1])
			}
			surfaces = append(surfaces, Surface{
				Name: fmt.Sprintf("media-notice-constants(source %s)", f.Path),
				Body: strings.Join(notices, "\n"),
			})
		}
		// The store's source: the text its error values are built from, which is
		// all a read-only package can see. A live error string is swept by
		// internal/cli's leak test, which reads a store that fails.
		if strings.HasPrefix(f.Path, "internal/store/") && strings.HasSuffix(f.Path, ".go") {
			surfaces = append(surfaces, Surface{Name: fmt.Sprintf("store-source(source %s)", f.Path), Body: string(f.Src)})
		}
	}

	// A recovered panic's output, produced in-process.
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				surfaces = append(surfaces, Surface{Name: "recovered-panic(output of this run)", Body: fmt.Sprintf("panic: %v", rec)})
			}
		}()
		panic("audit leak surface")
	}()

	return surfaces
}

// FormatReports is the verb's text report: one line per check, findings as
// path:line: detail, and a closing summary. Paths stay repository-relative.
func FormatReports(reports []Report) string {
	var b strings.Builder
	var findings, files int
	for _, r := range reports {
		fmt.Fprintf(&b, "%s\t%d files\t%s\n", r.Check, r.Files, r.Promise)
		files += r.Files
		for _, f := range r.Findings {
			findings++
			detail := clipSentence(f.Detail)
			switch {
			case f.Path != "" && f.Line > 0:
				fmt.Fprintf(&b, "  %s:%d: %s\n", f.Path, f.Line, detail)
			case f.Path != "":
				fmt.Fprintf(&b, "  %s: %s\n", f.Path, detail)
			default:
				fmt.Fprintf(&b, "  %s\n", detail)
			}
		}
	}
	fmt.Fprintf(&b, "%d checks, %d findings, %d files inspected\n", len(reports), findings, files)
	return b.String()
}

func clipSentence(s string) string {
	const limit = 200
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= limit {
		return s
	}
	return s[:limit-1] + "~"
}

// FindingCount returns the total findings across reports.
func FindingCount(reports []Report) int {
	n := 0
	for _, r := range reports {
		n += len(r.Findings)
	}
	return n
}
