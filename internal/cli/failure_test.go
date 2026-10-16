package cli

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/netcheck"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// resetInstant is the instant the rate-limited rows are asserted with, so a state's Reset can be read
// back.
var resetInstant = time.Date(2026, 10, 14, 14, 32, 0, 0, time.UTC)

// TestEveryOutcomeMapsToItsKind is the mapper's table: every outcome the classifier can return has a
// kind here, and every one of them carries the classifier's own sentence as its Detail — the pane's
// body line, and the reason no failure can render as a blank line.
func TestEveryOutcomeMapsToItsKind(t *testing.T) {
	for _, row := range []struct {
		outcome xweb.Outcome
		kind    domain.FailureKind
	}{
		{xweb.OutcomeAuthRejected, domain.FailureSessionExpired},
		{xweb.OutcomeSignerOrAuth, domain.FailureSignerOrAuth},
		{xweb.OutcomeUnknownOperation, domain.FailureOperationStale},
		{xweb.OutcomeEndpointRetired, domain.FailureEndpointRetired},
		{xweb.OutcomeRateLimited, domain.FailureRateLimited},
		{xweb.OutcomeSoftBan, domain.FailureSoftBan},
		{xweb.OutcomeLoadShed, domain.FailureLoadShed},
		{xweb.OutcomeBadRequest, domain.FailureBadRequest},
		{xweb.OutcomeEmptyBody, domain.FailureEmptyBody},
		{xweb.OutcomeChallenge, domain.FailureChallenge},
		{xweb.OutcomeUnauthorized, domain.FailureUnauthorized},
		// Not a failure upstream would raise, and not a state with no row: an answer this build has
		// no rule for is named as unclassified rather than dropped.
		{xweb.OutcomeUnclassified, domain.FailureUnclassified},
		{xweb.OutcomeOK, domain.FailureUnclassified},
	} {
		t.Run(string(row.outcome), func(t *testing.T) {
			failure := failureFor(&xweb.APIError{Operation: "HomeLatestTimeline", Outcome: row.outcome, Status: 429}, nil)
			if failure.Kind != row.kind {
				t.Errorf("the outcome %q mapped to %v, want %v", row.outcome, failure.Kind, row.kind)
			}
			if got, want := failure.Detail, row.outcome.Remedy(); got != want {
				t.Errorf("the outcome %q carries %q as its Detail, want the classifier's own sentence %q", row.outcome, got, want)
			}
		})
	}
}

// TestTheRateLimitedRowCarriesTheReset: the state line's `until 14:32` comes from the answer's own
// header, and a rate limit with no header is a different state — the zero reset the line reads as
// "reset unknown" — rather than a reset that has already passed.
func TestTheRateLimitedRowCarriesTheReset(t *testing.T) {
	withReset := failureFor(&xweb.APIError{Outcome: xweb.OutcomeRateLimited, Reset: resetInstant}, nil)
	if withReset.Kind != domain.FailureRateLimited {
		t.Fatalf("a 429 mapped to %v", withReset.Kind)
	}
	if !withReset.Reset.Equal(resetInstant) {
		t.Errorf("the rate limit carries the reset %v, want %v", withReset.Reset, resetInstant)
	}
	if withReset.Retry {
		t.Errorf("a rate limit allows a retry before its reset")
	}

	without := failureFor(&xweb.APIError{Outcome: xweb.OutcomeRateLimited}, nil)
	if !without.Reset.IsZero() {
		t.Errorf("a rate limit with no reset header carries %v", without.Reset)
	}
	if without.Retry {
		t.Errorf("a rate limit with no end in sight allows a retry")
	}
}

// TestTheSoftBanStopsTheSession is the one state whose rule is "stop for the session": the mapper
// says so in the failure itself, which is what screen.List and the shell read.
func TestTheSoftBanStopsTheSession(t *testing.T) {
	failure := failureFor(&xweb.APIError{Outcome: xweb.OutcomeSoftBan, Status: 200}, nil)
	if failure.Kind != domain.FailureSoftBan {
		t.Fatalf("a soft ban mapped to %v", failure.Kind)
	}
	if !failure.Stopped {
		t.Errorf("the soft ban does not stop the session")
	}
	if failure.Retry {
		t.Errorf("the soft ban allows a retry")
	}
	if failure.Detail != xweb.OutcomeSoftBan.Remedy() {
		t.Errorf("the soft ban carries %q as its Detail", failure.Detail)
	}
}

// TestTheBudgetAndNoSessionRows: the two states that are not the classifier's are still named, and
// their Detail is a sentence rather than a blank line. Both sentinels arrive wrapped, as the client
// and the session loader wrap them.
func TestTheBudgetAndNoSessionRows(t *testing.T) {
	spent := failureFor(fmt.Errorf("xweb: %w: 200 requests issued, cap 200", xweb.ErrRequestCap), nil)
	if spent.Kind != domain.FailureBudgetSpent {
		t.Errorf("the spent budget mapped to %v", spent.Kind)
	}
	if spent.Detail == "" {
		t.Errorf("the spent budget has nothing to show")
	}
	if spent.Retry {
		t.Errorf("the spent budget allows a retry: only a restart resets it")
	}

	none := failureFor(fmt.Errorf("%w: put auth_token=VALUE and ct0=VALUE in /tmp/session", session.ErrNoSession), nil)
	if none.Kind != domain.FailureNoSession {
		t.Errorf("a missing session mapped to %v", none.Kind)
	}
	if none.Detail == "" {
		t.Errorf("a missing session has nothing to show")
	}
	if !none.Retry {
		t.Errorf("a missing session is not retryable: a re-copied cookie takes effect without a restart")
	}
	// The wrapped message names the whole fix and a path; the pane prints one sentence, so the mapper
	// uses the classifier's own rather than half of a longer one.
	if none.Detail != xweb.OutcomeAuthRejected.Remedy() {
		t.Errorf("a missing session carries %q as its Detail, want the classifier's sentence for a session x.com will not accept", none.Detail)
	}
}

// TestThePathRows: with nothing classified, the reachability check's own report names the state, and
// the Detail is copied from the failing result's own sentence — including the outcomes the mapper has
// no row for, which are named as unclassified rather than dropped.
func TestThePathRows(t *testing.T) {
	for _, row := range []struct {
		outcome netcheck.Outcome
		kind    domain.FailureKind
	}{
		{netcheck.OutcomeConfigFailed, domain.FailureTransportUnusable},
		{netcheck.OutcomeNoRoute, domain.FailureNoRoute},
		{netcheck.OutcomeProxyRefused, domain.FailureProxyRefused},
		{netcheck.OutcomeUpstreamBlocked, domain.FailureUpstreamBlocked},
		{netcheck.OutcomeChallenge, domain.FailureTLSChallenge},
		{netcheck.OutcomeSessionRejected, domain.FailureSessionExpired},
		{netcheck.OutcomeAPIRejected, domain.FailureUnclassified},
	} {
		t.Run(string(row.outcome), func(t *testing.T) {
			// The first result succeeded and the second failed: the failed one names the state.
			check := []netcheck.Result{
				{Outcome: netcheck.OutcomeRouteOK, Detail: "x.com resolves"},
				{Outcome: row.outcome, Detail: "the check's own sentence about " + string(row.outcome)},
			}
			failure := failureFor(errors.New("the request did not complete"), check)
			if failure.Kind != row.kind {
				t.Errorf("the check outcome %q mapped to %v, want %v", row.outcome, failure.Kind, row.kind)
			}
			if want := check[1].Detail; failure.Detail != want {
				t.Errorf("the state carries %q, want the check's own sentence %q", failure.Detail, want)
			}
			if !failure.Retry {
				t.Errorf("the path row %q is not retryable: a fixed proxy is worth trying again", row.outcome)
			}
		})
	}

	// The arm: a check whose failures are all behind it does not name a path state.
	if failure := failureFor(errors.New("unrelated"), []netcheck.Result{{Outcome: netcheck.OutcomeRouteOK}}); failure.Kind != domain.FailureUnclassified {
		t.Errorf("a check with no failed result mapped to %v", failure.Kind)
	}
}

// TestAFailureAlwaysHasWords is the arm the whole mapper rests on: every kind it can produce carries
// a Detail, because the pane prints that sentence and a state with nothing to show is the one thing
// the phase's honesty rules forbid.
func TestAFailureAlwaysHasWords(t *testing.T) {
	inputs := []mapperInput{
		{"no error at all", nil, nil},
		{"an error with no state", errors.New("something went wrong"), nil},
		{"a check with no detail", errors.New("silent"), []netcheck.Result{{Outcome: netcheck.OutcomeNoRoute}}},
		{"a spent budget", xweb.ErrRequestCap, nil},
		{"no session", session.ErrNoSession, nil},
	}
	for _, outcome := range []xweb.Outcome{
		xweb.OutcomeAuthRejected, xweb.OutcomeSignerOrAuth, xweb.OutcomeUnknownOperation,
		xweb.OutcomeEndpointRetired, xweb.OutcomeRateLimited, xweb.OutcomeSoftBan,
		xweb.OutcomeLoadShed, xweb.OutcomeBadRequest, xweb.OutcomeEmptyBody,
		xweb.OutcomeChallenge, xweb.OutcomeUnauthorized, xweb.OutcomeUnclassified,
	} {
		inputs = append(inputs, mapperInput{"the answer " + string(outcome), &xweb.APIError{Outcome: outcome}, nil})
	}
	for _, result := range []netcheck.Outcome{
		netcheck.OutcomeConfigFailed, netcheck.OutcomeNoRoute, netcheck.OutcomeProxyRefused,
		netcheck.OutcomeUpstreamBlocked, netcheck.OutcomeChallenge, netcheck.OutcomeSessionRejected,
		netcheck.OutcomeAPIRejected,
	} {
		inputs = append(inputs, mapperInput{"the check " + string(result), errors.New("the request did not complete"), []netcheck.Result{{Outcome: result}}})
	}

	for _, input := range inputs {
		t.Run(input.name, func(t *testing.T) {
			failure := failureFor(input.err, input.check)
			if failure.Kind == domain.FailureNone {
				t.Errorf("%s mapped to FailureNone, which is not a state", input.name)
			}
			if failure.Detail == "" {
				t.Errorf("%s has nothing to show: the pane would render a blank line", input.name)
			}
		})
	}
}

// mapperInput is one error and one check report, as the mapper is handed them: the inputs the words
// arm walks.
type mapperInput struct {
	name  string
	err   error
	check []netcheck.Result
}
