package audit

import (
	"strings"
)

// hiddenPromise is the line this check prints. It is the design's "no flag, config key or env var
// selects vivid mode", stated as a thing a machine can hold: the state's name appears in exactly the
// two places that must know it — the type's own vocabulary and the string a test failure prints — and
// nowhere a start-up path reads.
const hiddenPromise = "the second display state is reachable only from the key sequence: no config name, flag or env var selects it"

// checkHiddenMode is the vivid presentation's gate. It holds two facts about the tree:
//
//  1. "vivid" is not a case of StateOf, which is the config file's vocabulary. A name the file could
//     hold would make a config key a way in, and the design says the sequence is the only door.
//  2. The composition root never assigns the state directly: it reads Display.State, so the only
//     assignment this build may make is the transition's own.
//
// It reads source as data and names nothing else, the way every check in this package does.
func checkHiddenMode(t Tree) Report {
	const name = "hidden"
	r := Report{Check: name, Promise: hiddenPromise, Files: len(t)}
	scanned := 0
	for _, f := range t {
		if !strings.HasSuffix(f.Path, ".go") || strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		scanned++
		src := string(f.Src)
		switch {
		case strings.HasSuffix(f.Path, "internal/tui/model.go"):
			// StateOf's body may not name the state. Its own switch is the one place the check
			// reads, and a case for the vivid name there is the finding.
			if body, ok := funcBody(src, "StateOf"); ok {
				if strings.Contains(body, `"vivid"`) || strings.Contains(body, "StateVivid") {
					r.Findings = append(r.Findings, Finding{
						Check:  name,
						Path:   f.Path,
						Detail: "StateOf names the second display state: the config file would become a way in",
						Line:   lineOf(src, "StateVivid"),
					})
				}
			}
		case strings.HasSuffix(f.Path, "internal/cli/feed.go"):
			// The composition root builds the start-up display. It may pass the state it read from
			// the file and nothing else.
			if strings.Contains(src, "tui.StateVivid") {
				r.Findings = append(r.Findings, Finding{
					Check:  name,
					Path:   f.Path,
					Detail: "the composition root names the second display state: a start-up path reaches it",
					Line:   lineOf(src, "tui.StateVivid"),
				})
			}
		}
	}
	if scanned == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "inspected no Go source: a scan of nothing proves nothing"))
	}
	r.Files = scanned
	return r
}

// funcBody is the source of one function's body, found by its name and closed at the first line that
// is a lone closing brace in column one. It is a text scan and it says so: this package parses files
// as data and a full AST walk here would be a second parser to keep honest.
func funcBody(src, name string) (string, bool) {
	marker := "func " + name + "("
	i := strings.Index(src, marker)
	if i < 0 {
		return "", false
	}
	rest := src[i:]
	if end := strings.Index(rest, "\n}\n"); end >= 0 {
		return rest[:end], true
	}
	return rest, true
}

// lineOf is the 1-based line a substring first appears on, or 0.
func lineOf(src, needle string) int {
	i := strings.Index(src, needle)
	if i < 0 {
		return 0
	}
	return strings.Count(src[:i], "\n") + 1
}
