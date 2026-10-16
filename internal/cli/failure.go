package cli

import (
	"errors"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/netcheck"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// apiKinds is the classifier's outcome table read as kinds: every outcome xweb.Classify can return
// has a row here, including the two it renames for the screen (OutcomeAuthRejected is the session
// expired, OutcomeUnknownOperation is a stale operation id). The words for a state are not here:
// internal/render owns both columns and reads them off the kind, and each row's Detail is the
// classifier's own remedy sentence, so the pane prints Phase 1's words rather than this file's.
var apiKinds = map[xweb.Outcome]domain.FailureKind{
	xweb.OutcomeAuthRejected:     domain.FailureSessionExpired,
	xweb.OutcomeSignerOrAuth:     domain.FailureSignerOrAuth,
	xweb.OutcomeUnknownOperation: domain.FailureOperationStale,
	xweb.OutcomeEndpointRetired:  domain.FailureEndpointRetired,
	xweb.OutcomeRateLimited:      domain.FailureRateLimited,
	xweb.OutcomeSoftBan:          domain.FailureSoftBan,
	xweb.OutcomeLoadShed:         domain.FailureLoadShed,
	xweb.OutcomeBadRequest:       domain.FailureBadRequest,
	xweb.OutcomeEmptyBody:        domain.FailureEmptyBody,
	xweb.OutcomeChallenge:        domain.FailureChallenge,
	xweb.OutcomeUnauthorized:     domain.FailureUnauthorized,
}

// pathKinds is the reachability check's outcome table read as kinds: the five ways a path can be
// broken, the one it reports when the session was refused, and the one it reports when the API read
// itself failed. Only the outcomes netcheck.Outcome.Failed reports are here — the ones that say the
// path is fine never reach this table — and an outcome with no row is named as unclassified rather
// than dropped.
var pathKinds = map[netcheck.Outcome]domain.FailureKind{
	netcheck.OutcomeConfigFailed:    domain.FailureTransportUnusable,
	netcheck.OutcomeNoRoute:         domain.FailureNoRoute,
	netcheck.OutcomeProxyRefused:    domain.FailureProxyRefused,
	netcheck.OutcomeUpstreamBlocked: domain.FailureUpstreamBlocked,
	netcheck.OutcomeChallenge:       domain.FailureTLSChallenge,
	netcheck.OutcomeSessionRejected: domain.FailureSessionExpired,
	netcheck.OutcomeAPIRejected:     domain.FailureUnclassified,
}

// noRetry is the set of states whose remedy is "do not retry": asking again before the state has
// changed spends a request to be refused. These are the five rows the UI-SPEC's state table marks,
// and screen.List reads the Failure.Retry it produces when a key asks whether it may request — the
// rate-limited row's end is the reset the upstream named, which screen.List's own rule grants.
var noRetry = map[domain.FailureKind]bool{
	domain.FailureRateLimited:     true,
	domain.FailureSoftBan:         true,
	domain.FailureBudgetSpent:     true,
	domain.FailureBadRequest:      true,
	domain.FailureEndpointRetired: true,
}

// failureFor maps one error into the state the screen shows: the one place an error becomes a kind
// with words to print.
//
// The sources are named by their own types rather than by parsing a message: a classified answer is
// an *xweb.APIError, a spent budget is xweb.ErrRequestCap, a missing session is
// session.ErrNoSession, and everything else is read through the reachability check's results, whose
// first failure names where the path broke and carries its own sentence. Detail is never empty: a
// state with nothing better to say still says the classifier's own sentence for a shape it has no
// rule for, so a failure can never reach the pane as a blank line.
func failureFor(err error, check []netcheck.Result) domain.Failure {
	var api *xweb.APIError
	switch {
	case errors.As(err, &api):
		return apiFailure(api)
	case errors.Is(err, xweb.ErrRequestCap):
		// The budget this process imposed on itself. Its own sentence is the only one that exists
		// for it: no classifier row describes a cap this program set.
		return domain.Failure{
			Kind:   domain.FailureBudgetSpent,
			Detail: xweb.ErrRequestCap.Error(),
		}
	case errors.Is(err, session.ErrNoSession):
		// No session at all, which is a different state from one the upstream refused — the state
		// word says which — and the same instruction: the classifier's sentence for a session x.com
		// will not accept is what the pane prints under it.
		return domain.Failure{
			Kind:   domain.FailureNoSession,
			Detail: xweb.OutcomeAuthRejected.Remedy(),
			Retry:  true,
		}
	}

	if failure, ok := pathFailure(check); ok {
		return failure
	}

	// Nothing named the state. The error's own sentence is what the pane prints, and the kind is
	// the classifier's row for a shape this build has no rule for.
	detail := xweb.OutcomeUnclassified.Remedy()
	if err != nil {
		detail = err.Error()
	}
	return domain.Failure{Kind: domain.FailureUnclassified, Detail: detail, Retry: true}
}

// apiFailure reads one classified answer as a state. The detail is the classifier's own remedy
// sentence, the reset is the instant the upstream sent with the answer, and the two rows that
// change what a caller may do — the rate limit and the soft ban — carry it in the failure rather
// than leaving every caller to re-derive it.
func apiFailure(api *xweb.APIError) domain.Failure {
	kind, known := apiKinds[api.Outcome]
	if !known {
		kind = domain.FailureUnclassified
	}
	failure := domain.Failure{
		Kind:   kind,
		Detail: api.Outcome.Remedy(),
		Retry:  !noRetry[kind],
	}
	if kind == domain.FailureRateLimited {
		// Zero when the header was absent, which is its own state: no reset to wait for.
		failure.Reset = api.Reset
	}
	if kind == domain.FailureSoftBan {
		// The one state whose rule is "stop for the session".
		failure.Stopped = true
	}
	return failure
}

// pathFailure names the path state the first failed reachability result reports, with the Detail
// copied from that result's own sentence. A result whose outcome the table has no row for is still
// named — as unclassified, with the check's own sentence — because a failing path that rendered as
// nothing is the state this mapper exists to make impossible.
func pathFailure(check []netcheck.Result) (domain.Failure, bool) {
	for _, result := range check {
		if !result.Outcome.Failed() {
			continue
		}
		kind, known := pathKinds[result.Outcome]
		if !known {
			kind = domain.FailureUnclassified
		}
		detail := result.Detail
		if detail == "" {
			detail = xweb.OutcomeUnclassified.Remedy()
		}
		return domain.Failure{Kind: kind, Detail: detail, Retry: !noRetry[kind]}, true
	}
	return domain.Failure{}, false
}
