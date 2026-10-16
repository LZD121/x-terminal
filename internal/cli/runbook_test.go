package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/netcheck"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// stateToAnchor is the coverage table: every failing classifier / netcheck state
// this build can produce maps onto one RUNBOOK.md section. The list is written
// here rather than derived from RUNBOOK.md so a truncated document cannot agree
// with itself. Accidental distribution is named as the one section no state
// must reach — the exception is explicit.

var stateToAnchor = []struct {
	state  string
	anchor string
}{
	{string(netcheck.OutcomeConfigFailed), "no-path-to-x.com"},
	{string(netcheck.OutcomeNoRoute), "no-path-to-x.com"},
	{string(netcheck.OutcomeProxyRefused), "no-path-to-x.com"},
	{string(netcheck.OutcomeUpstreamBlocked), "no-path-to-x.com"},
	{string(netcheck.OutcomeChallenge), "no-path-to-x.com"},
	{string(netcheck.OutcomeSessionRejected), "session-expiry"},
	{string(netcheck.OutcomeAPIRejected), "unclassified"},

	{string(xweb.OutcomeAuthRejected), "session-expiry"},
	{string(xweb.OutcomeSignerOrAuth), "signer-break"},
	{string(xweb.OutcomeUnknownOperation), "operation-id-rotation"},
	{string(xweb.OutcomeEndpointRetired), "endpoint-retired"},
	{string(xweb.OutcomeRateLimited), "429"},
	{string(xweb.OutcomeSoftBan), "soft-ban"},
	{string(xweb.OutcomeLoadShed), "load-shed"},
	{string(xweb.OutcomeBadRequest), "request-shape-drift"},
	{string(xweb.OutcomeEmptyBody), "empty-response-body"},
	{string(xweb.OutcomeChallenge), "no-path-to-x.com"},
	{string(xweb.OutcomeUnauthorized), "unclassified"},
	{string(xweb.OutcomeUnclassified), "unclassified"},

	{"mirror newer", "the-mirror"},
	{"mirror unreadable", "the-mirror"},
	{"likes unavailable", "the-likes-endpoint"},

	// The update verb's own five states, mapped to the section that repairs them. They
	// are read from the declaration the verb prints from (updateStates), not typed here,
	// so a state renamed in update.go without this section being renamed fails the build
	// — and a section whose states have no rows trips the "no orphan sections" arm.
	{stateNotReachable, "the-update-path"},
	{stateNoAsset, "the-update-path"},
	{stateNotVerified, "the-update-path"},
	{stateNotWritable, "the-update-path"},
	{stateNotARelease, "the-update-path"},
}

// orphanExempt is the one section a classifier state does not have to reach.
const orphanExempt = "accidental-distribution"

var builtCommands = []string{
	"x-terminal doctor",
	"doctor --probe",
	"doctor --pin",
	"doctor --refresh-ids",
	"x-terminal audit",
	"x-terminal sync",
	"go test",
	"x-terminal update",
}

var ownedFiles = []string{
	"config.toml",
	"session",
	"pins.json",
	"saved.db",
}

func TestRunbook(t *testing.T) {
	body := readRunbook(t)
	headings := runbookHeadings(body)

	t.Run("coverage table is total and non-empty", func(t *testing.T) {
		if len(stateToAnchor) == 0 {
			t.Fatal("coverage table is empty: every assertion below would pass by vacuity")
		}
		seenState := map[string]bool{}
		for _, row := range stateToAnchor {
			seenState[row.state] = true
			if !headingExists(headings, row.anchor) {
				t.Errorf("state %q maps to #%s, which is not a ## heading in RUNBOOK.md", row.state, row.anchor)
			}
		}
		for _, o := range failingXwebOutcomes() {
			if !seenState[string(o)] {
				t.Errorf("failing xweb.Outcome %q has no coverage row", o)
			}
		}
		for _, o := range failingNetcheckOutcomes() {
			if !seenState[string(o)] {
				t.Errorf("failing netcheck.Outcome %q has no coverage row", o)
			}
		}
		distinct := map[string]bool{}
		for _, row := range stateToAnchor {
			distinct[row.anchor] = true
		}
		if len(headings) < len(distinct) {
			t.Errorf("RUNBOOK.md has %d ## sections, coverage table needs at least %d distinct anchors", len(headings), len(distinct))
		}
	})

	t.Run("doctor pointer anchors exist", func(t *testing.T) {
		for state, ptr := range RunbookAnchors() {
			if state == "mirror missing" {
				if ptr != "" {
					t.Errorf("mirror missing must have empty pointer, got %q", ptr)
				}
				continue
			}
			if ptr == "" {
				t.Errorf("failing state %q has empty runbook pointer", state)
				continue
			}
			anchor := strings.TrimPrefix(ptr, "RUNBOOK.md#")
			if !headingExists(headings, anchor) {
				t.Errorf("doctor pointer %q for %q has no heading", ptr, state)
			}
		}
	})

	t.Run("mirror shapes", func(t *testing.T) {
		if RunbookAnchor("mirror missing") != "" {
			t.Fatal("mirror missing is not a failure and must have no section pointer")
		}
		if got := RunbookAnchor("mirror newer"); got != "RUNBOOK.md#the-mirror" {
			t.Fatalf("mirror newer → %q", got)
		}
		if got := RunbookAnchor("mirror unreadable"); got != "RUNBOOK.md#the-mirror" {
			t.Fatalf("mirror unreadable → %q", got)
		}
		if !headingExists(headings, "the-mirror") {
			t.Fatal("## The mirror heading missing")
		}
	})

	t.Run("exercisability", func(t *testing.T) {
		for _, h := range headings {
			if h.slug == "request-discipline" {
				continue
			}
			sec := sectionBody(body, h.slug)
			verify := partText(sec, "Verify")
			repair := partText(sec, "Repair")
			cmd := findBuiltCommand(verify)
			if cmd == "" {
				t.Errorf("section %q Verify names no built command: %q", h.title, verify)
			} else {
				t.Logf("section %q Verify exercises %q", h.title, cmd)
			}
			if findBuiltCommand(repair) == "" && findOwnedFile(repair) == "" {
				t.Errorf("section %q Repair names neither a built command nor an owned file: %q", h.title, repair)
			}
		}

	})

	t.Run("four parts and verbatim symptoms", func(t *testing.T) {
		words := classifierWords()
		for _, h := range headings {
			if h.slug == "request-discipline" {
				continue
			}
			sec := sectionBody(body, h.slug)
			for _, name := range []string{"Symptom", "What doctor shows", "Repair", "Verify"} {
				text := partText(sec, name)
				if strings.TrimSpace(text) == "" {
					t.Errorf("section %q missing non-empty **%s.**", h.title, name)
				}
			}
			if !partsInOrder(sec) {
				t.Errorf("section %q does not carry the four parts in order", h.title)
			}
			symptom := partText(sec, "Symptom")
			if !containsAny(symptom, words) {
				t.Errorf("section %q Symptom holds no verbatim classifier/state word: %q", h.title, symptom)
			}
		}
	})

	t.Run("request discipline", func(t *testing.T) {
		fifth := len(body) / 5
		if fifth < 1 {
			fifth = len(body)
		}
		head := body[:fifth]
		if !strings.Contains(head, "Request discipline") {
			t.Fatal("## Request discipline missing from the first fifth of RUNBOOK.md")
		}
		rule := "A repair is one request; there is no retry loop; a failed repair is a second diagnosis"
		// The file states the rule in prose; accept the plan's three clauses.
		needles := []string{"one request", "no retry loop", "second diagnosis"}
		for _, n := range needles {
			if !strings.Contains(strings.ToLower(head), n) {
				t.Errorf("request-discipline paragraph missing %q (near top)", n)
			}
		}
		for _, slug := range []string{"429", "soft-ban"} {
			sec := strings.ToLower(sectionBody(body, slug))
			for _, n := range needles {
				if !strings.Contains(sec, n) {
					t.Errorf("section #%s missing discipline clause %q", slug, n)
				}
			}
		}
		_ = rule
	})

	t.Run("no orphan sections", func(t *testing.T) {
		mapped := map[string]bool{}
		for _, row := range stateToAnchor {
			mapped[row.anchor] = true
		}
		mapped[orphanExempt] = true // explicit exception
		for _, h := range headings {
			if h.slug == "request-discipline" {
				continue
			}
			if !mapped[h.slug] {
				t.Errorf("orphan section %q (#%s): no state maps to it (accidental-distribution is the only named exception)", h.title, h.slug)
			}
		}
	})
}

type runbookHeading struct {
	title string
	slug  string
}

func readRunbook(t *testing.T) string {
	t.Helper()
	path := filepath.Join(moduleRoot(t), "RUNBOOK.md")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading RUNBOOK.md: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("RUNBOOK.md is empty")
	}
	return string(body)
}

func runbookHeadings(body string) []runbookHeading {
	var out []runbookHeading
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		title := strings.TrimSpace(strings.TrimPrefix(line, "## "))
		out = append(out, runbookHeading{title: title, slug: slugify(title)})
	}
	return out
}

func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	// Keep dots (no-path-to-x.com) and digits (429).
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '.':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func headingExists(headings []runbookHeading, slug string) bool {
	for _, h := range headings {
		if h.slug == slug {
			return true
		}
	}
	return false
}

func sectionBody(body, slug string) string {
	lines := strings.Split(body, "\n")
	var start = -1
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") && slugify(strings.TrimSpace(strings.TrimPrefix(line, "## "))) == slug {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return ""
	}
	var end = len(lines)
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

var partRE = regexp.MustCompile(`(?m)^\*\*(Symptom|What doctor shows|Repair|Verify)\.\*\*\s*(.*)$`)

func partText(sec, name string) string {
	matches := partRE.FindAllStringSubmatch(sec, -1)
	for _, m := range matches {
		if m[1] == name {
			return m[2]
		}
	}
	return ""
}

func partsInOrder(sec string) bool {
	order := []string{"Symptom", "What doctor shows", "Repair", "Verify"}
	matches := partRE.FindAllStringSubmatch(sec, -1)
	if len(matches) < 4 {
		return false
	}
	at := 0
	for _, m := range matches {
		if at < len(order) && m[1] == order[at] {
			at++
		}
	}
	return at == len(order)
}

func findBuiltCommand(s string) string {
	for _, c := range builtCommands {
		if strings.Contains(s, c) {
			return c
		}
	}
	// Accept bare doctor / audit / sync when prefixed by x-terminal earlier in the same line.
	lower := s
	for _, bare := range []string{"doctor", "audit", "sync --full", "sync"} {
		if strings.Contains(lower, "x-terminal "+bare) || strings.Contains(lower, bare) && strings.Contains(lower, "x-terminal") {
			return "x-terminal " + bare
		}
	}
	return ""
}

func findOwnedFile(s string) string {
	for _, f := range ownedFiles {
		if strings.Contains(s, f) {
			return f
		}
	}
	return ""
}

func classifierWords() []string {
	var out []string
	for _, o := range failingXwebOutcomes() {
		out = append(out, string(o))
	}
	for _, o := range failingNetcheckOutcomes() {
		out = append(out, string(o))
	}
	// Domain / presence words the short sections and accidental-distribution quote.
	out = append(out,
		"wrong account",
		"saved.db",
		"no session",
		"auth_token",
		"code 88",
		"this build",
	)
	// The update verb's vocabulary, read out of the declaration the verb prints from
	// rather than typed a second time here: the section that repairs it has to quote one
	// of these words verbatim, and renaming a state in update.go is what makes the text
	// and the code drift apart.
	out = append(out, updateStates...)
	return out
}

func containsAny(hay string, needles []string) bool {
	lower := strings.ToLower(hay)
	for _, n := range needles {
		if n != "" && strings.Contains(lower, strings.ToLower(n)) {
			return true
		}
	}
	return false
}
