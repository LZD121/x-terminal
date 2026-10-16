package domain

import "time"

// FailureKind is the closed taxonomy of states a read can end in, one constant per row of the
// UI-SPEC's two state tables plus the zero value. It is closed on purpose: every state the
// classifier or the reachability check can report has a constant here, so that no failure can
// reach a screen as an untyped error it has no line for.
//
// The *words* for these states are not here. internal/render owns the state column and the
// terse remedy column, and the pane's longer body line is the classifier's own sentence
// (`xweb.Outcome.Remedy`) or the failing reachability check's `Detail`, which the caller copies
// into Failure.Detail. This package names the state; it never spells it.
type FailureKind uint8

const (
	// FailureNone is "no failure": the zero value, so that a Failure nobody set cannot be read
	// as one of the named states. It is not a row of either table.
	FailureNone FailureKind = iota

	// FailureSessionExpired: the session cookie is missing or expired (state words: session
	// expired). Retryable — a re-copied cookie takes effect without a restart.
	FailureSessionExpired
	// FailureSignerOrAuth: the request was rejected for a reason that is not the cookie —
	// authorization header or the transaction id the operation needs.
	FailureSignerOrAuth
	// FailureOperationStale: the registry's query id was refused (404 with an empty body).
	FailureOperationStale
	// FailureEndpointRetired: the operation itself is gone upstream.
	FailureEndpointRetired
	// FailureRateLimited: 429, or a 200 whose remaining budget is zero. Reset is the instant
	// the upstream said the window reopens.
	FailureRateLimited
	// FailureSoftBan: the account is held back (code 88 while the budget is not spent). It is
	// the one state whose rule is "stop for the session", and Stopped says so.
	FailureSoftBan
	// FailureLoadShed: the service is at capacity (130/131). Retryable after a backoff.
	FailureLoadShed
	// FailureBadRequest: the request shape was rejected (400/422).
	FailureBadRequest
	// FailureEmptyBody: a success status with no bytes at all. Never an empty timeline.
	FailureEmptyBody
	// FailureChallenge: an edge interstitial answered instead of the API.
	FailureChallenge
	// FailureUnauthorized: a 200 carrying an errors[] body.
	FailureUnauthorized
	// FailureUnclassified: an answer this build has no rule for. It is a state of its own
	// rather than a fallback so that a new shape is visible instead of being blamed on a
	// known one.
	FailureUnclassified
	// FailureBudgetSpent: this process spent its own request cap (xweb.ErrRequestCap). Only a
	// restart resets it.
	FailureBudgetSpent
	// FailureNoRoute: x.com did not resolve, or nothing accepted the connection.
	FailureNoRoute
	// FailureProxyRefused: nothing is listening on the configured proxy.
	FailureProxyRefused
	// FailureUpstreamBlocked: the proxy answered and the request through it did not complete.
	FailureUpstreamBlocked
	// FailureTLSChallenge: an edge interstitial answered the reachability check.
	FailureTLSChallenge
	// FailureTransportUnusable: the transport could not be built at all (an unusable proxy
	// setting). Nothing was dialed.
	FailureTransportUnusable
	// FailureNoSession: there is no session to authenticate with, so no read was attempted.
	FailureNoSession
)

// Failure is one state the pane shows, as a value.
//
// A Failure is sticky: it stays until a fetch succeeds, and no timer clears it. The two fields
// that change what a caller may do are Retry and Stopped, and they are explicit rather than
// derived here, because deriving them means re-implementing the state table in every caller.
type Failure struct {
	// Kind is which state this is. The zero value is FailureNone.
	Kind FailureKind
	// Detail is the sentence the pane prints when it has nothing else to show. The caller
	// copies it from the classifier's own remedy (`xweb.Outcome.Remedy()`) or from the failing
	// reachability check's `Detail`; this package never builds it.
	Detail string
	// Reset is the upstream's reset instant (x-rate-limit-reset) for FailureRateLimited, and
	// the zero time when the header was absent — which is a different, non-retryable state
	// ("no retry; reset unknown"), not a reset in the past.
	Reset time.Time
	// Retry says whether an action may re-request. It is false for the states whose remedy is
	// "do not retry": rate limited before its reset, soft ban, budget spent, bad request and
	// endpoint retired. A zero Failure has Retry false, so an unset failure cannot silently
	// enable a request.
	Retry bool
	// Stopped is true only for the soft ban, whose rule is "stop for the session": after it,
	// no page request and no refresh is issued again for the life of the process.
	Stopped bool
}
