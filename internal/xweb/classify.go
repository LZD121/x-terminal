package xweb

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Outcome is what one answer from the upstream meant. It is a typed string rather
// than a number because every one of these states has to reach a human as a sentence,
// and the sentence is the contract: Error names the state, Remedy says what to do
// about it.
//
// There is no outcome meaning "no posts". An empty body is a state of its own
// (OutcomeEmptyBody) precisely so that a truncated or challenged answer can never be
// rendered as an empty timeline.
type Outcome string

const (
	// OutcomeUnclassified is the zero value, and the honest answer for a status this
	// table has no row for. It is not a success, and its remedy is not a retry.
	OutcomeUnclassified Outcome = "unclassified"
	// OutcomeOK is the one success: a 2xx that carried a body.
	OutcomeOK Outcome = "ok"
	// OutcomeAuthRejected: a 401, or the redirect the upstream sends a session it no
	// longer accepts — x.com answered /home with a 307 to the login flow, a zero-byte
	// body and set-cookie expiring auth_token and ct0 when the cookie was dead
	// (measured 2026-10-14).
	OutcomeAuthRejected Outcome = "auth rejected"
	// OutcomeSignerOrAuth: a 403 with a zero-byte body. Measured twice, on requests the
	// upstream refused before it routed the operation; the status alone cannot say
	// whether the auth headers were unsupported or a transaction-id was required.
	OutcomeSignerOrAuth Outcome = "signer or auth rejected"
	// OutcomeUnknownOperation: a 404 with a zero-byte body — the operation id, or the path
	// it was asked for, is not one this upstream knows.
	OutcomeUnknownOperation Outcome = "unknown operation"
	// OutcomeEndpointRetired: a 404 that carried a JSON body. Measured on both v1.1 REST
	// endpoints tried: {"errors":[{"message":"Sorry, that page does not exist","code":34}]}.
	OutcomeEndpointRetired Outcome = "endpoint retired"
	// OutcomeRateLimited: a 429, or an errors[] body with code 88 and the rate-limit
	// budget spent.
	OutcomeRateLimited Outcome = "rate limited"
	// OutcomeSoftBan: errors[].code 88 while x-rate-limit-remaining is still above zero.
	// The account is being held back for reasons the client cannot see, which is why the
	// remedy is not "wait for the reset" but "stop for the session".
	OutcomeSoftBan Outcome = "soft ban"
	// OutcomeLoadShed: a 5xx, or a body carrying a capacity error.
	OutcomeLoadShed Outcome = "load shed"
	// OutcomeBadRequest: 400 or 422 — the request's own shape was rejected, so the
	// variables or features are wrong rather than the session.
	OutcomeBadRequest Outcome = "bad request"
	// OutcomeEmptyBody: a 2xx that carried no bytes at all. Never a success.
	OutcomeEmptyBody Outcome = "empty response body"
	// OutcomeChallenge: a 403 or 503 that is an edge interstitial rather than an API
	// answer. Its body is not API data and must not be parsed as any.
	OutcomeChallenge Outcome = "edge challenge"
	// OutcomeUnauthorized: any other answer carrying errors[] — the body the upstream
	// meant the caller to read, and the only one.
	OutcomeUnauthorized Outcome = "unauthorized"
)

// Error names the state. The zero value reports itself as unclassified rather than as
// an empty string, so a caller that prints it cannot print nothing at all.
func (o Outcome) Error() string {
	if o == "" {
		return string(OutcomeUnclassified)
	}
	return string(o)
}

// Remedy is what to do about the state, in the words the doctor and any later screen
// print. It is deliberately part of the type: an outcome whose remedy a caller has to
// invent is an outcome whose meaning the caller is guessing at.
func (o Outcome) Remedy() string {
	switch o {
	case OutcomeOK:
		return "the call succeeded - parse the body"
	case OutcomeAuthRejected:
		return "session missing or expired - re-copy the cookie"
	case OutcomeSignerOrAuth:
		return "auth headers unsupported or the transaction-id is required for this operation"
	case OutcomeUnknownOperation:
		return "the operation id is stale or unknown - refresh the registry or pin it"
	case OutcomeEndpointRetired:
		return "this endpoint is retired - do not retry"
	case OutcomeRateLimited:
		return "wait until the reset time from x-rate-limit-reset"
	case OutcomeSoftBan:
		return "stop for the session"
	case OutcomeLoadShed:
		return "bounded backoff, distinct from rate limiting"
	case OutcomeBadRequest:
		return "the request shape was rejected - fix the variables or features it names; do not retry"
	case OutcomeEmptyBody:
		return "the upstream answered with no body at all; that is not an empty timeline - re-check the route and do not render it as data"
	case OutcomeChallenge:
		return "the body is not API data; a Cloudflare or edge interstitial answered instead of the API"
	case OutcomeUnauthorized:
		return "print the API's own message from the errors[] body"
	default:
		return "this shape has not been measured - record it before acting on it"
	}
}

// Success reports whether the answer delivered data. Exactly one outcome does, and it
// requires a non-empty body.
func (o Outcome) Success() bool { return o == OutcomeOK }

// Classify turns one answer into the state it represents, from the status, the body's
// shape and two headers. The rows are the measured ones; where an Outcome constant
// documents a measurement, that measurement is the test fixture for it.
//
// Two rules the table exists to keep:
//
//   - An empty body is never a success. A 2xx with no bytes is OutcomeEmptyBody, and
//     nothing here returns an outcome whose remedy is "no posts".
//   - A 429 is OutcomeRateLimited, whose remedy is to wait for the reset header. A
//     retry is never the answer to it, so no row may put one in front of the 429 check.
func Classify(status int, headers http.Header, body []byte) Outcome {
	errs, apiJSON := parseErrors(body)
	empty := len(body) == 0
	remaining, remainingKnown := rateLimitRemaining(headers)

	switch {
	case status == http.StatusTooManyRequests:
		// First among the statuses on purpose: this is the one state whose remedy must not be
		// a retry, and the one an interstitial body must not be able to displace.
		return OutcomeRateLimited

	case looksLikeInterstitial(status, headers, body):
		return OutcomeChallenge

	case status == http.StatusNotFound:
		switch {
		case empty:
			return OutcomeUnknownOperation
		case apiJSON:
			return OutcomeEndpointRetired
		default:
			return OutcomeChallenge
		}

	case status == http.StatusUnauthorized:
		return OutcomeAuthRejected

	case status == http.StatusForbidden:
		switch {
		case empty:
			return OutcomeSignerOrAuth
		case len(errs) > 0:
			return classifyErrors(status, errs, remaining, remainingKnown)
		default:
			return OutcomeChallenge
		}

	case status >= 200 && status < 300:
		switch {
		case empty:
			return OutcomeEmptyBody
		case len(errs) > 0:
			return classifyErrors(status, errs, remaining, remainingKnown)
		default:
			return OutcomeOK
		}

	case status == http.StatusBadRequest || status == http.StatusUnprocessableEntity:
		return OutcomeBadRequest

	case status >= 500:
		return OutcomeLoadShed

	case status >= 300 && status < 400:
		if expiredSessionRedirect(headers) {
			return OutcomeAuthRejected
		}
		return OutcomeUnclassified

	default:
		return OutcomeUnclassified
	}
}

// classifyErrors decides among the states an errors[] body can carry. The status it
// arrived under is part of the evidence: the same body means "the service is shedding
// load" at 5xx and "your account is held back" at 200.
func classifyErrors(status int, errs []apiError, remaining int, remainingKnown bool) Outcome {
	if status >= 500 {
		return OutcomeLoadShed
	}
	for _, e := range errs {
		if e.Code.Number == 88 {
			// Code 88 is reached both by "you have used your requests" and by "this account is
			// being held back". The remaining budget is what tells them apart, and it is only
			// read as zero when it has actually been read: an absent header cannot be evidence
			// that an account is out of budget.
			if remainingKnown && remaining > 0 {
				return OutcomeSoftBan
			}
			return OutcomeRateLimited
		}
		if loadShedCode(e.Code.Number) || loadShedMessage(e.Message) {
			return OutcomeLoadShed
		}
	}
	return OutcomeUnauthorized
}

// apiError is one entry of an errors[] body.
//
// Path is the position in the request an error is about, when the upstream sends one. It is
// where a rejected request shape names the variable it wanted: measured on 2026-10-15, a missing
// variable answers 422 with {"errors":[{"code":"GRAPHQL_VALIDATION_FAILED","message":"must be
// defined","path":["variable","includePromotedContent"]}]} — the message alone says nothing an
// operator could act on, which is why the path travels with it.
type apiError struct {
	Code    apiCode
	Message string
	Path    []string `json:"path"`
}

// apiCode is an errors[].code, which the upstream sends as a JSON number in some answers and as
// a JSON string in others: {"code":88} and {"code":"GRAPHQL_VALIDATION_FAILED"} are both measured
// shapes (2026-10-14 and 2026-10-15). Decoding straight into an int made the string form fail the
// decode of the whole errors[] array, and that failure is silent wrongness rather than a reported
// parse error: the array read as absent, a 200 carrying one classified as a success, and the
// classifier's remedy sentence — which promises to name what the upstream named — had nothing to
// name. Both readings are kept: the number where the upstream sent one, the name where it sent
// that.
type apiCode struct {
	Number int
	Name   string
}

func (c *apiCode) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	if number, err := strconv.Atoi(strings.Trim(trimmed, `"`)); err == nil {
		c.Number = number
		return nil
	}
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	c.Name = name
	return nil
}

// parseErrors reads the errors[] array the upstream puts in a body it means the caller
// to read. The boolean reports whether the body was a JSON object at all, which is how
// the classifier tells "this endpoint answered with a body it means me to parse" from
// "something between me and the endpoint answered": an HTML interstitial must not be
// read as an API error, and a 404 carrying JSON means something different from a 404
// carrying nothing.
func parseErrors(body []byte) (errs []apiError, apiJSON bool) {
	if len(body) == 0 {
		return nil, false
	}
	var payload struct {
		Errors []apiError `json:"errors"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false
	}
	return payload.Errors, true
}

// rateLimitRemaining reads x-rate-limit-remaining. Whether the header was there at all
// is part of the answer: the soft-ban row is defined by a code 88 error that arrives
// while the budget is not spent, so an absent header cannot be read as zero.
func rateLimitRemaining(headers http.Header) (int, bool) {
	if headers == nil {
		return 0, false
	}
	value := strings.TrimSpace(headers.Get("x-rate-limit-remaining"))
	if value == "" {
		return 0, false
	}
	remaining, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return remaining, true
}

// rateLimitReset reads x-rate-limit-reset, the instant the upstream's window reopens, in Unix
// seconds. Whether the header was there at all is part of the answer: a rate limit with no reset
// is a different state to report ("no retry; reset unknown") from one with a time, and an absent
// header cannot be read as a reset at the epoch. A value that is not a number is not a reset
// either — this is the upstream's own clock written down, and guessing at a malformed one would
// put a fabricated time on the state line.
func rateLimitReset(headers http.Header) (time.Time, bool) {
	if headers == nil {
		return time.Time{}, false
	}
	value := strings.TrimSpace(headers.Get("x-rate-limit-reset"))
	if value == "" {
		return time.Time{}, false
	}
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(seconds, 0).UTC(), true
}

// looksLikeInterstitial reports whether the answer came from the edge rather than from
// the API. The statuses are the two a challenge was measured (or reported) under; a 200
// with a challenge body has not been seen, and inventing a row for it here would be
// inventing the one thing this table exists to avoid.
func looksLikeInterstitial(status int, headers http.Header, body []byte) bool {
	if status != http.StatusForbidden && status != http.StatusServiceUnavailable {
		return false
	}
	if headers != nil && headers.Get("cf-mitigated") != "" {
		return true
	}
	lower := strings.ToLower(string(body))
	for _, marker := range []string{"just a moment", "cf-chl-", "cf-mitigated", "attention required", "cloudflare"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// expiredSessionRedirect reports whether a redirect is the upstream saying the session
// is gone rather than moving the caller somewhere. Measured with a dead cookie on
// 2026-10-14: /home answered 307 with a zero-byte body, location
// /i/jf/onboarding/web?redirect_after_login=%2Fhome&mode=login, and set-cookie expiring
// auth_token and ct0 — an expiry notice the client has to read, because the status is
// a 307 rather than a 401.
func expiredSessionRedirect(headers http.Header) bool {
	if headers == nil {
		return false
	}
	for _, value := range headers.Values("Set-Cookie") {
		lower := strings.ToLower(value)
		if strings.Contains(lower, "auth_token=;") || strings.Contains(lower, "ct0=;") {
			return true
		}
	}
	for _, value := range headers.Values("Location") {
		lower := strings.ToLower(value)
		for _, marker := range []string{"mode=login", "redirect_after_login", "flow/login", "/login"} {
			if strings.Contains(lower, marker) {
				return true
			}
		}
	}
	return false
}

// loadShedCode reports whether an errors[].code is one the upstream uses for capacity
// rather than for the request. 130 ("Over capacity") and 131 (internal error) are the
// codes it has used; the messages that come with them are checked as well, because the
// code alone has not been observed here.
func loadShedCode(code int) bool { return code == 130 || code == 131 }

// loadShedMessage reports whether an errors[].message says the service is at capacity.
func loadShedMessage(message string) bool {
	lower := strings.ToLower(message)
	for _, marker := range []string{"over capacity", "internal error", "service unavailable", "load shed"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
