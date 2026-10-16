package render

import (
	"os"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/netcheck"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// This test file is the one place the render package reaches for the classifiers, and it does it
// on purpose: it is the pin that stops the words on the screen from drifting away from the words
// the transport layer produces. It imports internal/xweb and internal/netcheck, which the
// package's own boundary test forbids for non-test files only. No non-test file in this package
// imports either of them.

// allKinds is every failure kind the domain declares, the zero value first. It is written out
// here rather than derived, because a kind added to the domain without a row in copy.go must fail
// this test, and a list that read itself out of the domain could not say that.
func allKinds() []domain.FailureKind {
	return []domain.FailureKind{
		domain.FailureNone,
		domain.FailureSessionExpired,
		domain.FailureSignerOrAuth,
		domain.FailureOperationStale,
		domain.FailureEndpointRetired,
		domain.FailureRateLimited,
		domain.FailureSoftBan,
		domain.FailureLoadShed,
		domain.FailureBadRequest,
		domain.FailureEmptyBody,
		domain.FailureChallenge,
		domain.FailureUnauthorized,
		domain.FailureUnclassified,
		domain.FailureBudgetSpent,
		domain.FailureNoRoute,
		domain.FailureProxyRefused,
		domain.FailureUpstreamBlocked,
		domain.FailureTLSChallenge,
		domain.FailureTransportUnusable,
		domain.FailureNoSession,
	}

}

// wantStateRows is the number of named kinds: every kind but the zero value, which is not a row of
// either table. It is a literal so that a table that lost a row cannot pass by iterating fewer
// kinds.
const wantStateRows = 19

// source is where a kind's state word comes from: the classifier constant it is the spelling of,
// and whether the row is one of the four documented renames.
type source struct {
	name    string
	word    string
	renamed bool
}

// sources maps every named kind to the classifier constant its state word is pinned to.
func sources() map[domain.FailureKind]source {
	return map[domain.FailureKind]source{
		domain.FailureSessionExpired: {name: "xweb.OutcomeAuthRejected", word: string(xweb.OutcomeAuthRejected), renamed: true},
		domain.FailureSignerOrAuth:   {name: "xweb.OutcomeSignerOrAuth", word: string(xweb.OutcomeSignerOrAuth)},
		domain.FailureOperationStale: {name: "xweb.OutcomeUnknownOperation", word: string(xweb.OutcomeUnknownOperation), renamed: true},
		domain.FailureEndpointRetired: {name: "xweb.OutcomeEndpointRetired",
			word: string(xweb.OutcomeEndpointRetired)},
		domain.FailureRateLimited: {name: "xweb.OutcomeRateLimited", word: string(xweb.OutcomeRateLimited)},
		domain.FailureSoftBan:     {name: "xweb.OutcomeSoftBan", word: string(xweb.OutcomeSoftBan)},
		domain.FailureLoadShed:    {name: "xweb.OutcomeLoadShed", word: string(xweb.OutcomeLoadShed)},
		domain.FailureBadRequest:  {name: "xweb.OutcomeBadRequest", word: string(xweb.OutcomeBadRequest)},
		domain.FailureEmptyBody:   {name: "xweb.OutcomeEmptyBody", word: string(xweb.OutcomeEmptyBody)},
		domain.FailureChallenge:   {name: "xweb.OutcomeChallenge", word: string(xweb.OutcomeChallenge)},
		domain.FailureUnauthorized: {name: "xweb.OutcomeUnauthorized",
			word: string(xweb.OutcomeUnauthorized)},
		domain.FailureUnclassified: {name: "xweb.OutcomeUnclassified", word: string(xweb.OutcomeUnclassified)},
		domain.FailureNoRoute:      {name: "netcheck.OutcomeNoRoute", word: string(netcheck.OutcomeNoRoute)},
		domain.FailureProxyRefused: {name: "netcheck.OutcomeProxyRefused", word: string(netcheck.OutcomeProxyRefused)},
		domain.FailureUpstreamBlocked: {name: "netcheck.OutcomeUpstreamBlocked", word: string(netcheck.OutcomeUpstreamBlocked),
			renamed: true},
		domain.FailureTLSChallenge: {name: "netcheck.OutcomeChallenge", word: string(netcheck.OutcomeChallenge)},
		domain.FailureTransportUnusable: {name: "netcheck.OutcomeConfigFailed",
			word: string(netcheck.OutcomeConfigFailed)},
		domain.FailureNoSession: {name: "netcheck.OutcomeSessionSkipped", word: string(netcheck.OutcomeSessionSkipped),
			renamed: true},
	}
}

// TestEveryKindHasItsTwoRows: the state column and the remedy column have exactly one row per
// named failure kind, no row for the zero value, and no row for a kind the domain does not have.
func TestEveryKindHasItsTwoRows(t *testing.T) {
	if got := len(stateWords); got != wantStateRows {
		t.Fatalf("the state table has %d rows, want %d: a kind without a row is a state with no words", got, wantStateRows)
	}
	if got := len(remedies); got != wantStateRows {
		t.Fatalf("the remedy table has %d rows, want %d", got, wantStateRows)
	}
	if len(allKinds()) != wantStateRows+1 {
		t.Fatalf("the kinds under test number %d, want %d named kinds and the zero value", len(allKinds()), wantStateRows)
	}
	seen := map[domain.FailureKind]bool{}
	for _, kind := range allKinds() {
		state, stateRow := stateWords[kind]
		remedy, remedyRow := remedies[kind]
		if kind == domain.FailureNone {
			if stateRow || remedyRow {
				t.Fatalf("the zero kind has a row: %q %q", state, remedy)
			}
			continue
		}
		if !stateRow {
			t.Errorf("the kind %d has no state row", kind)
		}
		if !remedyRow {
			t.Errorf("the kind %d has no remedy row", kind)
		}
		if state == "" {
			t.Errorf("the kind %d has an empty state word", kind)
		}
		if remedy == "" {
			t.Errorf("the kind %d has an empty remedy", kind)
		}
		if seen[kind] {
			t.Errorf("the kind %d is listed twice under test", kind)
		}
		seen[kind] = true
	}
	for _, kind := range []domain.FailureKind{
		domain.FailureSessionExpired, domain.FailureRateLimited, domain.FailureNoRoute, domain.FailureOperationStale,
	} {
		if !seen[kind] {
			t.Fatalf("the four named states are not all under test: kind %d is missing", kind)
		}
	}
}

// TestTheStateWordsAreTheClassifiers is the pin: every row's word is the classifier's own, except
// the four renames, each of which is asserted to differ from the constant it was renamed away
// from — so a rename that was never made, or a rename made by half, fails here.
func TestTheStateWordsAreTheClassifiers(t *testing.T) {
	src := sources()
	// Every named kind but one is the spelling of a classifier outcome: the budget-spent state is
	// this process's own request cap (xweb.ErrRequestCap), which has no outcome, and
	// TestTheBudgetSpentRowNamesItsSource is what pins that row.
	if len(src)+1 != wantStateRows {
		t.Fatalf("the pin covers %d kinds, want the %d outcome-sourced rows", len(src), wantStateRows-1)
	}
	if _, ok := src[domain.FailureBudgetSpent]; ok {
		t.Fatal("the budget-spent row is listed as outcome-sourced, and xweb has no such outcome")
	}
	renames := 0
	for kind, from := range src {
		got := stateWords[kind]
		if got == "" {
			t.Errorf("%s: the copy table has no row for this kind", from.name)
			continue
		}
		if from.renamed {
			renames++
			if got == from.word {
				t.Errorf("%s: the row reads %q, which is the constant itself: the rename is missing", from.name, got)
			}
			continue
		}
		if got != from.word {
			t.Errorf("%s: the state row is %q, want the classifier's own %q", from.name, got, from.word)
		}
	}
	if renames != 4 {
		t.Fatalf("%d renamed rows, want the four the phase documents", renames)
	}

	// The classifier's own Error() is what the table must agree with, not just the constant.
	for kind, want := range map[domain.FailureKind]string{
		domain.FailureSessionExpired:  string(xweb.OutcomeAuthRejected),
		domain.FailureOperationStale:  string(xweb.OutcomeUnknownOperation),
		domain.FailureUpstreamBlocked: string(netcheck.OutcomeUpstreamBlocked),
		domain.FailureNoSession:       string(netcheck.OutcomeSessionSkipped),
	} {
		if stateWords[kind] == want {
			t.Errorf("the renamed row for %q still reads the classifier's word", want)
		}
	}
	if got := xweb.Outcome("").Error(); got != string(xweb.OutcomeUnclassified) {
		t.Errorf("the classifier's zero outcome reports %q, which is not the row this table pins", got)
	}
	if got := stateWords[domain.FailureUnclassified]; got != xweb.Outcome("").Error() {
		t.Errorf("the unclassified row is %q, want the classifier's zero value's own words %q", got, xweb.Outcome("").Error())
	}
}

// TestTheRemediesAreDistinct: the remedy column is a second, terser vocabulary, and the rows that
// carry the phase's named states must not collapse into one word — a remedy that says "failed" for
// four different states is the thing this column exists to avoid. The rate limit's two forms are
// asserted distinct as well, because a wait with no end is a different instruction.
func TestTheRemediesAreDistinct(t *testing.T) {
	named := []domain.FailureKind{
		domain.FailureSessionExpired,
		domain.FailureRateLimited,
		domain.FailureNoRoute,
		domain.FailureOperationStale,
	}
	seen := map[string]domain.FailureKind{}
	for _, kind := range named {
		remedy := Remedy(kind)
		if remedy == "" {
			t.Fatalf("the named state %q has no remedy", StateWord(kind))
		}
		if other, dup := seen[remedy]; dup {
			t.Fatalf("the states %q and %q share the remedy %q", StateWord(other), StateWord(kind), remedy)
		}
		seen[remedy] = kind
	}
	if len(seen) != len(named) {
		t.Fatalf("%d distinct remedies for %d named states", len(seen), len(named))
	}
	if CopyRemedyRateLimited == CopyRemedyRateLimitedUnknown {
		t.Fatal("the two rate-limited remedies are the same string")
	}
	if Remedy(domain.FailureRateLimited) != CopyRemedyRateLimited {
		t.Fatalf("the rate-limited remedy row is %q, want the reset-known form %q",
			Remedy(domain.FailureRateLimited), CopyRemedyRateLimited)
	}
	for _, kind := range allKinds() {
		if kind == domain.FailureNone {
			continue
		}
		if strings.TrimSpace(Remedy(kind)) != Remedy(kind) {
			t.Errorf("the remedy for %q has stray space around it", StateWord(kind))
		}
	}
}

// TestTheBudgetSpentRowNamesItsSource: the one state that is not a classifier outcome — this
// process's own request cap — is keyed to xweb.ErrRequestCap in copy.go, so a reader can tell
// where its words came from.
func TestTheBudgetSpentRowNamesItsSource(t *testing.T) {
	body, err := os.ReadFile("copy.go")
	if err != nil {
		t.Fatalf("reading the copy table: %v", err)
	}
	text := string(body)
	at := strings.Index(text, "FailureBudgetSpent")
	if at < 0 {
		t.Fatal("copy.go has no row for the budget-spent kind")
	}
	from := max(0, at-200)
	to := min(len(text), at+200)
	if !strings.Contains(text[from:to], "xweb.ErrRequestCap") {
		t.Fatalf("the budget-spent row does not name xweb.ErrRequestCap in its comment:\n%s", text[from:to])
	}
	if strings.Contains(text[from:to], "OutcomeBudget") {
		t.Fatal("the budget-spent row claims a classifier outcome that does not exist")
	}
}

// TestTheListWordsAreTheModels: the status line's two list words are the model's own names, so the
// line and the frame cannot spell the same feed two ways.
func TestTheListWordsAreTheModels(t *testing.T) {
	if CopyListFollowing != domain.Following.String() {
		t.Errorf("the list field says %q, the model says %q", CopyListFollowing, domain.Following.String())
	}
	if CopyListForYou != domain.ForYou.String() {
		t.Errorf("the list field says %q, the model says %q", CopyListForYou, domain.ForYou.String())
	}
	if ListName("", domain.Following) == ListName("", domain.ForYou) {
		t.Fatal("the two feeds render the same list word")
	}
	if CopyEmptyHeadingFollowing == CopyEmptyHeadingForYou {
		t.Fatal("the two feeds render the same empty heading")
	}
	if !strings.Contains(CopyEmptyHeadingFollowing, CopyListFollowing) ||
		!strings.Contains(CopyEmptyHeadingForYou, CopyListForYou) {
		t.Fatal("an empty heading does not name its own feed")
	}
	if CopyLoadingFollowing == CopyLoadingForYou {
		t.Fatal("the two feeds render the same loading line")
	}
}

// TestTheFourNamedStatesAreTheFourOPSRows: the four states OPS-01 promises — an expired session, a
// rate limit with its reset, an unreachable host and a stale operation — are the four rows whose
// words the phase's own table spells out, and neither the word nor the remedy of one is the word of
// another.
func TestTheFourNamedStatesAreTheFourOPSRows(t *testing.T) {
	want := map[domain.FailureKind]struct{ state, remedy string }{
		domain.FailureSessionExpired: {state: "session expired", remedy: "copy the cookie again"},
		domain.FailureRateLimited:    {state: "rate limited", remedy: "no retry"},
		domain.FailureNoRoute:        {state: "no route to x.com", remedy: "check the proxy"},
		domain.FailureOperationStale: {state: "operation id stale", remedy: "doctor --refresh-ids"},
	}
	words := map[string]bool{}
	for kind, row := range want {
		if got := StateWord(kind); got != row.state {
			t.Errorf("the state word for %d is %q, want %q", kind, got, row.state)
		}
		if got := Remedy(kind); got != row.remedy {
			t.Errorf("the remedy for %d is %q, want %q", kind, got, row.remedy)
		}
		if words[row.state] {
			t.Errorf("the state word %q is used by two named states", row.state)
		}
		words[row.state] = true
	}
	if len(want) != 4 {
		t.Fatalf("the OPS-01 table under test has %d rows, want 4", len(want))
	}
	// The verified detail sentences stay the classifier's, and none of them is the state word.
	for kind, outcome := range map[domain.FailureKind]xweb.Outcome{
		domain.FailureSessionExpired: xweb.OutcomeAuthRejected,
		domain.FailureNoRoute:        "", // netcheck, asserted below
		domain.FailureOperationStale: xweb.OutcomeUnknownOperation,
	} {
		if outcome == "" {
			continue
		}
		if detail := outcome.Remedy(); detail == StateWord(kind) {
			t.Errorf("the pane's body line for %q is the state word itself", StateWord(kind))
		}
	}
	if got := netcheck.OutcomeNoRoute; string(got) != StateWord(domain.FailureNoRoute) {
		t.Errorf("the reachability check's own words for the no-route state are %q, want the copy table's %q",
			got, StateWord(domain.FailureNoRoute))
	}
	if netcheck.OutcomeNoRoute.Failed() == false {
		t.Error("the reachability check no longer reports the no-route outcome as a failure")
	}
	if netcheck.OutcomeSessionSkipped.Failed() {
		t.Error("a skipped session is reported as a failure, which is the confusion the state exists to avoid")
	}
}
