package xweb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/LZD121/x-terminal/internal/session"
)

// translateTestID and translateTestLang are the id and language the measured request was made with:
// a public post id recorded in .planning/research/UPSTREAM-PROBE.md (2026-10-16) and the default
// destination language. Neither is a fixture body — this file asserts the request, not an answer.
const (
	translateTestID   = "2110709021544153425"
	translateTestLang = "zh-CN"
)

// measuredTranslatePath is the request path this build must build, written out in full rather than
// assembled from the constants the builder uses: a test that re-derived it from translateURL's own
// pieces would agree with the builder by construction and could not see a suffix creep in.
//
// The absence of `.json` is the whole point of pinning it by equality here: with the suffix the
// upstream answers `404 {"code":34,"message":"Sorry, that page does not exist"}` (measured
// 2026-10-16), which reads exactly like an endpoint that does not exist.
const measuredTranslatePath = "/1.1/strato/column/None/tweetId=2110709021544153425," +
	"destinationLanguage=Some(zh-CN),translationSource=Some(X),feature=None,timeout=None,onlyCached=None" +
	"/translation/service/translateTweet"

// TestTranslateBuildsTheMeasuredRequest is this file's anti-vacuity test, in the shape
// TestGraphQLBuildsTheMeasuredRequest established: it asserts the request the client actually sent —
// its path by equality, its empty query string, its method and its headers — rather than that some
// bytes came back.
func TestTranslateBuildsTheMeasuredRequest(t *testing.T) {
	var (
		mu       sync.Mutex
		gotPath  string
		gotRaw   string
		gotQuery url.Values
		gotHead  http.Header
		gotMeth  string
		calls    int
	)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		gotPath, gotRaw, gotQuery, gotHead = r.URL.EscapedPath(), r.URL.String(), r.URL.Query(), r.Header.Clone()
		gotMeth = r.Method
		mu.Unlock()
		w.Header().Set("content-type", "application/json")
		// The measured shape of a Success answer for this post: the fields this build reads, plus the
		// ones it does not, so the test asserts the request against a body of the real response's
		// width rather than a two-field stub.
		_, _ = w.Write([]byte(`{"id":2110709021544153425,"id_str":"2110709021544153425",` +
			`"translation":"translated body","translationState":"Success",` +
			`"sourceLanguage":"ja","localizedSourceLanguage":"ja",` +
			`"translationSource":"Google","destinationLanguage":"zh-CN","entities":[]}`))
	}))
	defer srv.Close()

	client := testClient(t, srv, fixtureSession())
	body, err := client.Translate(context.Background(), translateTestID, translateTestLang)
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if !strings.Contains(string(body), `"translationState":"Success"`) {
		t.Errorf("the body came back as %q, want what the server answered", body)
	}

	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("the client sent %d requests, want exactly 1: one call is one request", calls)
	}
	if gotPath != measuredTranslatePath {
		t.Errorf("the request path is\n  %q\nwant\n  %q", gotPath, measuredTranslatePath)
	}
	// The raw target carries no `?` at all: this endpoint's columns live in the path, and a query
	// string would be a second, unmeasured request shape.
	if i := strings.IndexByte(gotRaw, '?'); i >= 0 {
		t.Errorf("the request target is %q, want no query string on this endpoint", gotRaw)
	}
	if len(gotQuery) != 0 {
		t.Errorf("the request carries the query parameters %v, want none", gotQuery)
	}
	if gotMeth != http.MethodGet {
		t.Errorf("the request method is %q, want GET", gotMeth)
	}

	// The header set is apiHeaders', pinned here as well as in TestGraphQLBuildsTheMeasuredRequest so
	// that a second copy of the eight Set calls can never satisfy one call site and not the other.
	for header, want := range map[string]string{
		"Authorization":             "Bearer " + WebBearer,
		"Cookie":                    "auth_token=" + fixtureToken + "; ct0=" + fixtureCTO,
		"X-Csrf-Token":              fixtureCTO,
		"X-Twitter-Auth-Type":       "OAuth2Session",
		"X-Twitter-Active-User":     "yes",
		"X-Twitter-Client-Language": "en",
		"User-Agent":                UserAgent,
		"Content-Type":              "application/json",
	} {
		if got := gotHead.Get(header); got != want {
			t.Errorf("the %s header is %q, want %q", header, got, want)
		}
	}
	// And nothing that was not measured: this endpoint was measured answering 200 with no
	// transaction id, so a request that quietly carried one would be a different request.
	if got := gotHead.Get("x-client-transaction-id"); got != "" {
		t.Errorf("the request carried x-client-transaction-id %q, but the endpoint was measured needing none", got)
	}
}

// TestTranslateRefusesValuesItWillNotPutInAURL is the validation arm: a language holding a
// parenthesis or a comma, and an id that is not numeric, each produce an error and **zero** requests.
// The server below exists to be un-reached — the assertion is its call count — and the parenthesis
// case is the one that matters most: unescaped, it would close the column expression and build a
// different request.
func TestTranslateRefusesValuesItWillNotPutInAURL(t *testing.T) {
	var (
		mu    sync.Mutex
		calls int
	)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id_str":"2110709021544153425","translation":"x","translationState":"Success"}`))
	}))
	defer srv.Close()

	client := testClient(t, srv, fixtureSession())
	for _, tc := range []struct {
		name string
		id   string
		lang string
	}{
		{"a language holding a parenthesis", translateTestID, "zh-CN)"},
		{"a language holding a comma", translateTestID, "zh,CN"},
		{"a language holding a slash", translateTestID, "zh/CN"},
		{"a language holding a space", translateTestID, "zh CN"},
		{"a language holding a percent sign", translateTestID, "zh%2DCN"},
		{"a language that is empty", translateTestID, ""},
		{"a language of one letter", translateTestID, "z"},
		{"an id that is not numeric", "not-a-number", translateTestLang},
		{"an id holding path punctuation", "2110709021544153425)", translateTestLang},
		{"an id that is empty", "", translateTestLang},
		{"an id longer than 20 digits", strings.Repeat("9", 21), translateTestLang},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := client.Translate(context.Background(), tc.id, tc.lang)
			if err == nil {
				t.Fatalf("Translate(%q, %q) answered %q with no error, want a refusal", tc.id, tc.lang, body)
			}
		})
	}

	mu.Lock()
	defer mu.Unlock()
	if calls != 0 {
		t.Fatalf("the client sent %d requests for values it should have refused, want 0", calls)
	}
}

// TestTranslateClassifiesLikeEveryOtherRead pins that a refusal from this endpoint reaches the
// caller as the classifier's own outcome and status, through the one Classify the whole read path
// uses: an empty-bodied 404 is the unknown-operation state and a 429 is the rate-limited one, with
// the upstream's own reset instant carried along.
func TestTranslateClassifiesLikeEveryOtherRead(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		headers map[string]string
		body    string
		want    Outcome
	}{
		{
			name:   "an empty-bodied 404 is the unknown-operation state",
			status: http.StatusNotFound,
			want:   OutcomeUnknownOperation,
		},
		{
			name:   "a 429 is the rate-limited state, with its reset",
			status: http.StatusTooManyRequests,
			headers: map[string]string{
				"x-rate-limit-reset":     "1761000000",
				"x-rate-limit-remaining": "0",
			},
			want: OutcomeRateLimited,
		},
		{
			name:   "the suffixed form's 404 body is the retired-endpoint state",
			status: http.StatusNotFound,
			body:   `{"code":34,"message":"Sorry, that page does not exist"}`,
			want:   OutcomeEndpointRetired,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tc.headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			client := testClient(t, srv, fixtureSession())
			body, err := client.Translate(context.Background(), translateTestID, translateTestLang)
			if err == nil {
				t.Fatalf("Translate answered %q for a %d, want a classified error", body, tc.status)
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("the error is %T (%v), want *xweb.APIError", err, err)
			}
			if apiErr.Outcome != tc.want {
				t.Errorf("the outcome is %q, want %q", apiErr.Outcome, tc.want)
			}
			if apiErr.Status != tc.status {
				t.Errorf("the status is %d, want %d", apiErr.Status, tc.status)
			}
			if tc.want == OutcomeRateLimited {
				if apiErr.Reset.IsZero() {
					t.Error("the rate-limited error carries no reset instant, want the upstream's own")
				}
				if apiErr.Error() == "" {
					t.Error("the rate-limited error renders no sentence")
				}
			}
		})
	}
}

// TestTranslateWithoutASessionSendsNothing: an unauthenticated request to this endpoint is a spent
// request that measures nothing, so the client refuses before building one — and the error names the
// session, because that is the thing to fix.
func TestTranslateWithoutASessionSendsNothing(t *testing.T) {
	var (
		mu    sync.Mutex
		calls int
	)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
	}))
	defer srv.Close()

	client := testClient(t, srv, session.Session{})
	body, err := client.Translate(context.Background(), translateTestID, translateTestLang)
	if err == nil {
		t.Fatalf("Translate answered %q with no session, want a refusal", body)
	}
	if !errors.Is(err, session.ErrNoSession) {
		t.Errorf("the error is %v, want it to wrap session.ErrNoSession", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 0 {
		t.Fatalf("the client sent %d requests without a session, want 0", calls)
	}
}

// TestTranslateRefusesBeforeTheTransportSeesAnInvalidValue is the same validation arm at the URL
// builder itself, where the two patterns live: this is the unit under the request test above, and it
// pins the values the measurement used as the ones that pass.
func TestTranslateRefusesBeforeTheTransportSeesAnInvalidValue(t *testing.T) {
	for _, tc := range []struct {
		id, lang string
		ok       bool
	}{
		{translateTestID, "zh-CN", true},
		{translateTestID, "de", true},
		{translateTestID, "ja", true},
		{translateTestID, "en", true},
		{translateTestID, "zh-Hans-CN", true},
		{"1", "en", true},
		{translateTestID, "zh-CN)", false},
		{translateTestID, "Some(zh-CN),x", false},
		{translateTestID, "11", false},
		{"1e5", "en", false},
	} {
		got, err := translateURL(tc.id, tc.lang)
		if tc.ok && err != nil {
			t.Errorf("translateURL(%q, %q): %v, want a path", tc.id, tc.lang, err)
			continue
		}
		if !tc.ok {
			if err == nil {
				t.Errorf("translateURL(%q, %q) = %q, want a refusal", tc.id, tc.lang, got)
			}
			continue
		}
		if !strings.HasPrefix(got, APIBaseURL+translatePrefix) || !strings.HasSuffix(got, translateSuffix) {
			t.Errorf("translateURL(%q, %q) = %q, want it inside the measured prefix and suffix", tc.id, tc.lang, got)
		}
		if strings.Contains(got, ".json") {
			t.Errorf("translateURL(%q, %q) = %q, which carries the `.json` suffix the measurement rules out", tc.id, tc.lang, got)
		}
	}
}
