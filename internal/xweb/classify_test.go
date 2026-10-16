package xweb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The fixtures are the shapes the upstream was measured sending, copied from
// .planning/research/UPSTREAM-PROBE.md and from the re-measurements made while this plan
// was executed. They are what the table is keyed on, so the test cases below use the
// bytes rather than a paraphrase of them.
const (
	// GET /i/api/1.1/account/settings.json with a live session, 2026-10-14: the v1.1 REST
	// API is retired, and it says so in a JSON body.
	retiredBody = `{"errors":[{"message":"Sorry, that page does not exist","code":34}]}`
	// The same answer, reported at the sibling v1.1 endpoint (/account/verify_credentials).
	retiredBodyShort = `{"errors":[{"message":"Sorry, that page does not exist"}]}`
	// errors[] code 88 with the rate-limit budget spent is a rate limit; with budget left
	// it is the account being held back.
	rateLimitBody = `{"errors":[{"code":88,"message":"Rate limit exceeded"}]}`
	softBanBody   = `{"errors":[{"code":88,"message":"Your account is temporarily limited"}]}`
	// The capacity error the load-shed row exists for.
	loadShedBody = `{"errors":[{"message":"Over capacity","code":130}]}`
	// What a Cloudflare interstitial looks like when it lands instead of an API answer.
	challengeBody = `<!DOCTYPE html><html><head><title>Just a moment...</title></head><body><div id="cf-chl-widget"></div></body></html>`
	// A dead cookie: /home answered 307 with a zero-byte body, this location, and
	// set-cookie expiring auth_token and ct0 (measured 2026-10-14).
	deadSessionLocation = "/i/jf/onboarding/web?redirect_after_login=%2Fhome&mode=login"
	// The rejected-request-shape body the upstream sends when a required variable is missing,
	// measured 2026-10-15: 422, 172 bytes, and the name of the variable is in the error's PATH
	// rather than in its message (which says only "must be defined"). Its code arrives as a
	// string where every body above carries a number.
	validationBody = `{"errors":[{"code":"GRAPHQL_VALIDATION_FAILED","extensions":{"code":"GRAPHQL_VALIDATION_FAILED"},"message":"must be defined","path":["variable","includePromotedContent"]}]}`
)

func TestClassifyTheMeasuredTable(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		headers http.Header
		body    string
		want    Outcome
	}{
		{
			name:   "403 with a zero-byte body (bearer and no transaction-id, measured twice)",
			status: http.StatusForbidden,
			want:   OutcomeSignerOrAuth,
		},
		{
			name:   "404 with a zero-byte body (an operation id the upstream does not know)",
			status: http.StatusNotFound,
			want:   OutcomeUnknownOperation,
		},
		{
			name:   "404 with the retired v1.1 error body",
			status: http.StatusNotFound,
			body:   retiredBody,
			want:   OutcomeEndpointRetired,
		},
		{
			name:   "404 with the shortened version of the same body",
			status: http.StatusNotFound,
			body:   retiredBodyShort,
			want:   OutcomeEndpointRetired,
		},
		{
			name:    "429 with the reset header",
			status:  http.StatusTooManyRequests,
			headers: http.Header{"X-Rate-Limit-Reset": {"1791956000"}, "X-Rate-Limit-Remaining": {"0"}},
			want:    OutcomeRateLimited,
		},
		{
			name:    "200 carrying errors[].code 88 while the budget is not spent",
			status:  http.StatusOK,
			headers: http.Header{"X-Rate-Limit-Remaining": {"15"}},
			body:    softBanBody,
			want:    OutcomeSoftBan,
		},
		{
			name:    "200 carrying errors[].code 88 with the budget spent",
			status:  http.StatusOK,
			headers: http.Header{"X-Rate-Limit-Remaining": {"0"}},
			body:    rateLimitBody,
			want:    OutcomeRateLimited,
		},
		{
			name:   "200 carrying errors[].code 88 with no remaining header at all",
			status: http.StatusOK,
			body:   rateLimitBody,
			want:   OutcomeRateLimited,
		},
		{
			name:   "401 with a zero-byte body (a bearer on a navigation request, measured)",
			status: http.StatusUnauthorized,
			want:   OutcomeAuthRejected,
		},
		{
			name:   "401 with an API error body",
			status: http.StatusUnauthorized,
			body:   `{"errors":[{"code":32,"message":"Could not authenticate you"}]}`,
			want:   OutcomeAuthRejected,
		},
		{
			name:   "403 with a Cloudflare interstitial",
			status: http.StatusForbidden,
			body:   challengeBody,
			want:   OutcomeChallenge,
		},
		{
			name:    "503 the edge mitigated",
			status:  http.StatusServiceUnavailable,
			headers: http.Header{"Cf-Mitigated": {"challenge"}},
			body:    challengeBody,
			want:    OutcomeChallenge,
		},
		{
			name:   "404 with an HTML body (an edge page, not an API answer)",
			status: http.StatusNotFound,
			body:   "<html><body>nothing here</body></html>",
			want:   OutcomeChallenge,
		},
		{
			name:   "422 with an errors[] body",
			status: http.StatusUnprocessableEntity,
			body:   `{"errors":[{"message":"The following features cannot be null: responsive_web_graphql_timeline_navigation_enabled"}]}`,
			want:   OutcomeBadRequest,
		},
		{
			// The measured shape as it arrives: the code is a string and the variable the
			// upstream is missing is in the path. The row above is the other 422 spelling.
			name:   "422 as measured on 2026-10-15, code as a string and the variable in the path",
			status: http.StatusUnprocessableEntity,
			body:   validationBody,
			want:   OutcomeBadRequest,
		},
		{
			name:   "500 with a zero-byte body",
			status: http.StatusInternalServerError,
			want:   OutcomeLoadShed,
		},
		{
			name:   "200 carrying an over-capacity error",
			status: http.StatusOK,
			body:   loadShedBody,
			want:   OutcomeLoadShed,
		},
		{
			name:   "200 with a real body",
			status: http.StatusOK,
			body:   `{"data":{"user":{"result":{}}}}`,
			want:   OutcomeOK,
		},
		{
			name:   "200 with a zero-byte body (never a success)",
			status: http.StatusOK,
			want:   OutcomeEmptyBody,
		},
		{
			name:    "307 to the login flow with the session cookies cleared",
			status:  http.StatusTemporaryRedirect,
			headers: http.Header{"Location": {deadSessionLocation}, "Set-Cookie": {"auth_token=; Max-Age=-1791955999; Path=/; Domain=.x.com", "ct0=; Max-Age=-1791955999; Path=/; Domain=.x.com"}},
			want:    OutcomeAuthRejected,
		},
		{
			name:    "307 somewhere else",
			status:  http.StatusTemporaryRedirect,
			headers: http.Header{"Location": {"/home?lang=en"}},
			want:    OutcomeUnclassified,
		},
		{
			name:   "403 with an API errors[] body",
			status: http.StatusForbidden,
			body:   `{"errors":[{"code":353,"message":"This request requires a matching csrf cookie."}]}`,
			want:   OutcomeUnauthorized,
		},
		{
			name:   "a status the table has no row for",
			status: http.StatusTeapot,
			body:   `something else entirely`,
			want:   OutcomeUnclassified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.status, tt.headers, []byte(tt.body))
			if got != tt.want {
				t.Fatalf("Classify(%d, %v, %d bytes of body) = %q, want %q", tt.status, tt.headers, len(tt.body), got, tt.want)
			}
			// Every outcome has to be able to reach a human as a sentence: a state whose name
			// or whose remedy is empty is a state the UI would render as nothing.
			if got.Error() == "" || got.Remedy() == "" {
				t.Errorf("%q says nothing about itself, or nothing about what to do", got)
			}
		})
	}
}

// TestClassifyKeepsTheTwoEmptyBodiedRefusalsApart is the assertion the whole classifier
// exists for: a 403 with no body and a 404 with no body are the two shapes this upstream
// uses for "something was wrong", and rendering either as the other — or as an empty
// timeline — is the failure mode the project is built against.
func TestClassifyKeepsTheTwoEmptyBodiedRefusalsApart(t *testing.T) {
	forbidden := Classify(http.StatusForbidden, nil, nil)
	notFound := Classify(http.StatusNotFound, nil, nil)

	if forbidden == notFound {
		t.Fatalf("a 403 with no body and a 404 with no body both classify as %q", forbidden)
	}
	if forbidden != OutcomeSignerOrAuth || notFound != OutcomeUnknownOperation {
		t.Fatalf("Classify gave (%q, %q), want (%q, %q)", forbidden, notFound, OutcomeSignerOrAuth, OutcomeUnknownOperation)
	}
	if !strings.Contains(forbidden.Remedy(), "transaction-id") {
		t.Errorf("the 403 remedy does not name the signer: %q", forbidden.Remedy())
	}
	if !strings.Contains(notFound.Remedy(), "operation id") {
		t.Errorf("the 404 remedy does not name the operation id: %q", notFound.Remedy())
	}
}

// TestClassifyEmptyBodyIsNeverASuccess sweeps every status there is. An empty body is
// what this upstream sends when it refused a request, and there is no status under which
// it may be read as data — including the 2xx range, where it would otherwise look like an
// empty timeline.
func TestClassifyEmptyBodyIsNeverASuccess(t *testing.T) {
	for status := 100; status <= 599; status++ {
		if got := Classify(status, nil, nil); got.Success() {
			t.Errorf("a zero-byte body at status %d succeeded as %q", status, got)
		}
	}
	// Anti-vacuity: Success is reachable at all, so the sweep above is not passing because
	// nothing can ever be a success.
	if got := Classify(http.StatusOK, nil, []byte(`{"data":{}}`)); !got.Success() {
		t.Fatalf("a 200 with a body classified as %q: Success() is unreachable and the sweep proves nothing", got)
	}
}

// TestClassifyNeverAdvisesARetryFor429 covers the second rule the table keeps. A 429 is
// the upstream asking for less traffic; a caller that retried it would be spending the
// budget the limiter counts and risking a longer block.
func TestClassifyNeverAdvisesARetryFor429(t *testing.T) {
	variants := []struct {
		name    string
		headers http.Header
		body    string
	}{
		{name: "bare", body: ""},
		{name: "with the reset header", headers: http.Header{"X-Rate-Limit-Reset": {"1791956000"}}, body: ""},
		{name: "with an errors[] body out of budget", headers: http.Header{"X-Rate-Limit-Remaining": {"0"}}, body: rateLimitBody},
		{name: "with an errors[] body that still has budget", headers: http.Header{"X-Rate-Limit-Remaining": {"42"}}, body: rateLimitBody},
		{name: "with an interstitial body", body: challengeBody},
	}
	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			got := Classify(http.StatusTooManyRequests, variant.headers, []byte(variant.body))
			if got != OutcomeRateLimited {
				t.Fatalf("a 429 classified as %q, want %q", got, OutcomeRateLimited)
			}
			if remedy := strings.ToLower(got.Remedy()); strings.Contains(remedy, "retry") {
				t.Errorf("the 429 remedy says to retry: %q", got.Remedy())
			}
			if !strings.Contains(got.Remedy(), "x-rate-limit-reset") {
				t.Errorf("the 429 remedy does not point at the reset header: %q", got.Remedy())
			}
		})
	}
}

// TestEveryOutcomeExplainsItself walks the whole enumeration. A new outcome that nobody
// wrote a sentence for is an outcome a screen would print as a blank.
func TestEveryOutcomeExplainsItself(t *testing.T) {
	outcomes := []Outcome{
		OutcomeUnclassified, OutcomeOK, OutcomeAuthRejected, OutcomeSignerOrAuth,
		OutcomeUnknownOperation, OutcomeEndpointRetired, OutcomeRateLimited, OutcomeSoftBan,
		OutcomeLoadShed, OutcomeBadRequest, OutcomeEmptyBody, OutcomeChallenge, OutcomeUnauthorized,
	}
	if len(outcomes) < 13 {
		t.Fatalf("only %d outcomes are listed: the enumeration grew and this walk grew stale", len(outcomes))
	}
	for _, outcome := range outcomes {
		if outcome.Error() == "" || outcome.Remedy() == "" {
			t.Errorf("%q has no name or no remedy", outcome)
		}
		if outcome != OutcomeOK && outcome.Success() {
			t.Errorf("%q reports success, and only %q may", outcome, OutcomeOK)
		}
	}
	// The zero value is the one a forgotten assignment produces: it must not be a success,
	// and it must not be an empty string either.
	var zero Outcome
	if zero.Success() {
		t.Error("the zero value reports success")
	}
	if zero.Error() == "" {
		t.Error("the zero value prints as an empty string")
	}
}

// TestRateLimitResetReadsTheInstant covers the header reader on its own: the value is Unix
// seconds, an absent header is not a reset at the epoch, and a header that is not a number is
// not a reset either. The boolean is what tells those apart, and the rate-limited state's
// remedy depends on it.
func TestRateLimitResetReadsTheInstant(t *testing.T) {
	want := time.Date(2026, 10, 14, 14, 32, 0, 0, time.UTC)
	tests := []struct {
		name    string
		headers http.Header
		want    time.Time
		ok      bool
	}{
		{name: "the header the upstream sends", headers: http.Header{"X-Rate-Limit-Reset": {"1791988320"}}, want: want, ok: true},
		{name: "the same value with space around it", headers: http.Header{"X-Rate-Limit-Reset": {" 1791988320 "}}, want: want, ok: true},
		{name: "no header at all", headers: http.Header{"X-Rate-Limit-Remaining": {"0"}}},
		{name: "an empty header", headers: http.Header{"X-Rate-Limit-Reset": {""}}},
		{name: "a header that is not a number", headers: http.Header{"X-Rate-Limit-Reset": {"soon"}}},
		{name: "a header that is a date", headers: http.Header{"X-Rate-Limit-Reset": {"2026-10-14T14:32:00Z"}}},
		{name: "no headers at all"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := rateLimitReset(tc.headers)
			if ok != tc.ok {
				t.Fatalf("rateLimitReset reported known = %v, want %v", ok, tc.ok)
			}
			if !got.Equal(tc.want) {
				t.Fatalf("rateLimitReset = %v, want %v", got, tc.want)
			}
		})
	}
	if reset, _ := rateLimitReset(nil); !reset.IsZero() {
		t.Fatalf("a nil header map produced the reset %v", reset)
	}
	// The instant is the upstream's own, and its Unix seconds are what it was sent as.
	if got := want.Unix(); got != 1791988320 {
		t.Fatalf("the fixture instant is %d Unix seconds, want the header's own value", got)
	}
}

// TestClassifyCarriesTheResetOnARateLimit is the plumbing the state line needs: a 429 through the
// client arrives as an *APIError whose Reset is the header's instant, a 429 without the header
// arrives with a zero Reset (which is a different state to report, not a reset in the past), and
// the error's text carries the instant rather than a countdown.
func TestClassifyCarriesTheResetOnARateLimit(t *testing.T) {
	reset := time.Date(2026, 10, 14, 14, 32, 0, 0, time.UTC)

	t.Run("with the reset header", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("x-rate-limit-reset", strconv.FormatInt(reset.Unix(), 10))
			w.Header().Set("x-rate-limit-remaining", "0")
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer srv.Close()

		client := testClient(t, srv, fixtureSession())
		body, err := client.GraphQL(context.Background(), "Viewer", nil)
		if err == nil {
			t.Fatalf("a 429 returned %d bytes and no error", len(body))
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("the failure is %T (%v), want an *APIError carrying the classified outcome", err, err)
		}
		if apiErr.Outcome != OutcomeRateLimited {
			t.Fatalf("the outcome is %q, want %q", apiErr.Outcome, OutcomeRateLimited)
		}
		if !apiErr.Reset.Equal(reset) {
			t.Fatalf("the error carries the reset %v, want the header's own %v", apiErr.Reset, reset)
		}
		if !strings.Contains(err.Error(), "reset 2026-10-14T14:32:00Z") {
			t.Fatalf("the error does not print the instant: %v", err)
		}
		for _, countdown := range []string{"in ", "seconds", "from now"} {
			if strings.Contains(err.Error(), countdown) {
				t.Fatalf("the error prints the reset as a countdown (%q): %v", countdown, err)
			}
		}
	})

	t.Run("without the reset header", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer srv.Close()

		client := testClient(t, srv, fixtureSession())
		_, err := client.GraphQL(context.Background(), "Viewer", nil)
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("the failure is %T (%v), want an *APIError", err, err)
		}
		if !apiErr.Reset.IsZero() {
			t.Fatalf("a 429 with no reset header carries the instant %v", apiErr.Reset)
		}
		if strings.Contains(err.Error(), reset.UTC().Format(time.RFC3339)) {
			t.Fatalf("the error prints a reset instant that was never sent: %v", err)
		}
		if !strings.HasSuffix(err.Error(), OutcomeRateLimited.Remedy()) {
			t.Fatalf("the error does not end with the classifier's own remedy: %v", err)
		}
		if apiErr.Outcome != OutcomeRateLimited {
			t.Fatalf("the outcome is %q, want %q", apiErr.Outcome, OutcomeRateLimited)
		}
	})

	// The soft-ban row is unchanged by the new field: a code 88 error with budget left is still
	// the account being held back, whatever the reset header says.
	t.Run("the soft ban is not the rate limit", func(t *testing.T) {
		headers := http.Header{
			"X-Rate-Limit-Remaining": {"42"},
			"X-Rate-Limit-Reset":     {strconv.FormatInt(reset.Unix(), 10)},
		}
		if got := Classify(http.StatusOK, headers, []byte(softBanBody)); got != OutcomeSoftBan {
			t.Fatalf("a soft ban with a reset header classified as %q, want %q", got, OutcomeSoftBan)
		}
		if got := Classify(http.StatusOK, headers, []byte(rateLimitBody)); got != OutcomeSoftBan {
			t.Fatalf("code 88 with budget left classified as %q, want %q", got, OutcomeSoftBan)
		}
		if got := Classify(http.StatusTooManyRequests, headers, nil); got != OutcomeRateLimited {
			t.Fatalf("a 429 classified as %q, want %q", got, OutcomeRateLimited)
		}
	})
}

// TestParseErrorsReadsTheMeasuredValidationBody is the arm the 2026-10-15 measurement forced: the
// upstream's rejected-request-shape body carries its code as a STRING and names the variable it
// wants in the error's path. A reader that decoded code straight into an int dropped the whole
// errors[] array on this body — silently, since the decode failure is indistinguishable from an
// absent array — which classified a body that says what is wrong as though it said nothing.
func TestParseErrorsReadsTheMeasuredValidationBody(t *testing.T) {
	errs, apiJSON := parseErrors([]byte(validationBody))
	if !apiJSON {
		t.Fatal("the measured 422 body was not read as a JSON API answer")
	}
	if len(errs) != 1 {
		t.Fatalf("the body decoded to %d errors, want the one it carries", len(errs))
	}
	if errs[0].Code.Name != "GRAPHQL_VALIDATION_FAILED" {
		t.Errorf("the code's name is %q, want GRAPHQL_VALIDATION_FAILED: it arrives as a string on this body", errs[0].Code.Name)
	}
	if len(errs[0].Path) != 2 || errs[0].Path[0] != "variable" || errs[0].Path[1] != "includePromotedContent" {
		t.Fatalf("the path is %v, want [\"variable\",\"includePromotedContent\"]: that is where the upstream names the variable it wants", errs[0].Path)
	}

	// The numeric spelling from the older bodies keeps decoding as a number, so the classifier's
	// code-88 rows cannot have been disturbed by this reading.
	numberErrs, _ := parseErrors([]byte(rateLimitBody))
	if len(numberErrs) != 1 || numberErrs[0].Code.Number != 88 {
		t.Fatalf("{\"code\":88} decoded to %+v, want the number 88", numberErrs)
	}

	// And the two readings are one field: an apiCode carries whichever the upstream sent.
	if apiMessage([]byte(validationBody)) != "must be defined at variable.includePromotedContent" {
		t.Errorf("apiMessage = %q, want the message with the path it names", apiMessage([]byte(validationBody)))
	}
}
