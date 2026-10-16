package xweb

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

// The fixture has the shape of the real cookie — auth_token 40 bytes, ct0 160 — and is not a
// credential. It is the needle the header assertions below search for.
var (
	fixtureToken = "0123456789abcdef0123456789abcdef01234567"
	fixtureCTO   = strings.Repeat("b1c2d3e4", 20)
)

// viewerID is the operation id the snapshot carries for Viewer, measured against the live site
// on 2026-10-14. It is written out here on purpose: this is the one test that pins the wire
// format, and a test that read the id out of the registry could not tell a stale registry from a
// correct one. (The volatile package's guard test exempts test files for exactly this reason.)
const viewerID = "TpQxQtWMTMnIAMPlMGMqNw"

func fixtureSession() session.Session {
	return session.Session{AuthToken: session.Secret(fixtureToken), CTO: session.Secret(fixtureCTO), Source: "/tmp/session"}
}

// testClient points a real client at one httptest.Server: whatever host a request names, the dial
// goes to the server's listener, so a request built for https://x.com/… arrives here and the
// request the client actually built can be inspected whole.
func testClient(t *testing.T, srv *httptest.Server, sess session.Session) *Client {
	t.Helper()
	transport := srv.Client().Transport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, srv.Listener.Addr().String())
	}
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	registry, err := volatile.Load("")
	if err != nil {
		t.Fatalf("loading the snapshot: %v", err)
	}
	return &Client{
		HTTP:     &http.Client{Transport: transport, CheckRedirect: NoRedirect},
		Limiter:  NewLimiter(0, 10),
		session:  sess,
		registry: registry,
		signer:   NoopSigner{},
	}
}

// TestGraphQLBuildsTheMeasuredRequest is this file's anti-vacuity test: it asserts the request
// the client actually sent — its path, its three query parameters and its headers — rather than
// that some bytes came back.
func TestGraphQLBuildsTheMeasuredRequest(t *testing.T) {
	var (
		mu       sync.Mutex
		gotURL   *url.URL
		gotPath  string
		gotQuery url.Values
		gotHead  http.Header
		calls    int
	)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		gotURL, gotPath, gotQuery, gotHead = r.URL, r.URL.EscapedPath(), r.URL.Query(), r.Header.Clone()
		mu.Unlock()
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{"data":{"viewer":{"user_results":{"result":{"rest_id":"783214"}}}}}`)
	}))
	defer srv.Close()

	client := testClient(t, srv, fixtureSession())
	body, err := client.GraphQL(context.Background(), "Viewer", map[string]any{"withPayments": false, "userId": "783214"})
	if err != nil {
		t.Fatalf("GraphQL: %v", err)
	}
	if !strings.Contains(string(body), `"rest_id":"783214"`) {
		t.Errorf("the body came back as %q, want what the server answered", body)
	}

	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("the client sent %d requests, want 1", calls)
	}

	// The path is the operation id from the registry and the operation's own name.
	if want := "/i/api/graphql/" + viewerID + "/Viewer"; gotPath != want {
		t.Errorf("the request path is %q, want %q", gotPath, want)
	}
	if !strings.Contains(gotURL.String(), "/i/api/graphql/"+viewerID+"/Viewer") {
		t.Errorf("the request URL is %q, want it to carry the operation path", gotURL)
	}

	// variables is URL-encoded JSON; features is the object the registry's list renders as, with
	// the operation's own switches as keys.
	var variables map[string]any
	if err := jsonUnmarshal(gotQuery.Get("variables"), &variables); err != nil {
		t.Fatalf("variables came through as %q, which is not JSON: %v", gotQuery.Get("variables"), err)
	}
	if variables["userId"] != "783214" || variables["withPayments"] != false {
		t.Errorf("variables decoded to %v, want the map the caller passed", variables)
	}

	viewer, err := volatile.Lookup("Viewer")
	if err != nil {
		t.Fatal(err)
	}
	var features map[string]bool
	if err := jsonUnmarshal(gotQuery.Get("features"), &features); err != nil {
		t.Fatalf("features came through as %q, which is not the object the wire format wants: %v", gotQuery.Get("features"), err)
	}
	if len(features) == 0 {
		t.Fatal("features came through empty, but Viewer declares switches")
	}
	for _, name := range viewer.FeatureSwitches {
		if !features[name] {
			t.Errorf("features does not carry %q, which the registry says the operation needs", name)
		}
	}
	var toggles map[string]bool
	if err := jsonUnmarshal(gotQuery.Get("fieldToggles"), &toggles); err != nil {
		t.Fatalf("fieldToggles came through as %q, which is not the object the wire format wants: %v", gotQuery.Get("fieldToggles"), err)
	}
	for _, name := range viewer.FieldToggles {
		if !toggles[name] {
			t.Errorf("fieldToggles does not carry %q, which the registry says the operation needs", name)
		}
	}

	// The header set, exactly as measured on 2026-10-14.
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
	// And nothing that was not measured: the transaction-id signer is not implemented, and a
	// request that quietly carried a stale one would be indistinguishable from this one.
	if got := gotHead.Get("x-client-transaction-id"); got != "" {
		t.Errorf("the request carried x-client-transaction-id %q, but no signer is implemented", got)
	}
}

// TestSearchTimelineGoesOutAsAPostWithItsParametersInTheBody pins the wire form the second
// measured request shape needs: SearchTimeline answers GET with a 404 and a zero-byte body, so
// it goes out as a POST whose parameters travel as one JSON body and whose URL carries no query
// string. The expected values are read from the registry at run time, the way the Viewer case
// above reads them, so this asserts the form rather than a copy of the registry.
func TestSearchTimelineGoesOutAsAPostWithItsParametersInTheBody(t *testing.T) {
	var (
		mu      sync.Mutex
		gotMeth string
		gotURL  *url.URL
		gotBody []byte
		gotHead http.Header
		calls   int
	)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		calls++
		gotMeth, gotURL, gotBody, gotHead = r.Method, r.URL, body, r.Header.Clone()
		mu.Unlock()
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{"data":{"search_by_raw_query":{}}}`)
	}))
	defer srv.Close()

	client := testClient(t, srv, fixtureSession())
	if _, err := client.GraphQL(context.Background(), "SearchTimeline", map[string]any{
		"rawQuery": "sea otter",
		"count":    20,
		"product":  "Latest",
		"cursor":   "",
	}); err != nil {
		t.Fatalf("GraphQL: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("the client sent %d requests, want 1", calls)
	}
	if gotMeth != http.MethodPost {
		t.Fatalf("SearchTimeline went out as %s, want POST: GET is answered 404 with a zero-byte body (measured 2026-10-15)", gotMeth)
	}
	if gotURL.RawQuery != "" {
		t.Errorf("the POST URL carries the query string %q, want none: the three parameters are the body's", gotURL.RawQuery)
	}

	op, err := volatile.Lookup("SearchTimeline")
	if err != nil {
		t.Fatal(err)
	}
	if want := "/i/api/graphql/" + op.QueryID + "/SearchTimeline"; gotURL.EscapedPath() != want {
		t.Errorf("the request path is %q, want %q", gotURL.EscapedPath(), want)
	}

	var body struct {
		Variables    map[string]any  `json:"variables"`
		Features     map[string]bool `json:"features"`
		FieldToggles map[string]bool `json:"fieldToggles"`
	}
	if err := jsonUnmarshal(string(gotBody), &body); err != nil {
		t.Fatalf("the body came through as %q, which is not the object the POST form wants: %v", gotBody, err)
	}
	if body.Variables["rawQuery"] != "sea otter" || body.Variables["count"] != float64(20) || body.Variables["product"] != "Latest" {
		t.Errorf("the body's variables decoded to %v, want the map the caller passed", body.Variables)
	}
	if len(body.Features) == 0 {
		t.Fatal("the body carries no features, but SearchTimeline declares switches")
	}
	for _, name := range op.FeatureSwitches {
		if !body.Features[name] {
			t.Errorf("the body's features do not carry %q, which the registry says the operation needs", name)
		}
	}
	for _, name := range op.FieldToggles {
		if !body.FieldToggles[name] {
			t.Errorf("the body's fieldToggles do not carry %q, which the registry says the operation needs", name)
		}
	}
	// The header set is the same one the GET form sends: the method is the only difference
	// between the two shapes (measured 2026-10-15).
	for header, want := range map[string]string{
		"Authorization": "Bearer " + WebBearer,
		"Cookie":        "auth_token=" + fixtureToken + "; ct0=" + fixtureCTO,
		"Content-Type":  "application/json",
		"User-Agent":    UserAgent,
	} {
		if got := gotHead.Get(header); got != want {
			t.Errorf("the %s header is %q, want %q", header, got, want)
		}
	}
}

// TestViewerStillGoesOutAsAGetWithItsParametersInTheQuery is the contrast arm: an operation the
// POST table does not name — Viewer, which declares flags — still builds a GET whose three
// parameters are URL-encoded query parameters and whose body is empty. A table that leaked into
// the default would fail here.
func TestViewerStillGoesOutAsAGetWithItsParametersInTheQuery(t *testing.T) {
	var (
		mu       sync.Mutex
		gotMeth  string
		gotQuery url.Values
		gotBody  []byte
	)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		gotMeth, gotQuery, gotBody = r.Method, r.URL.Query(), body
		mu.Unlock()
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	client := testClient(t, srv, fixtureSession())
	if _, err := client.GraphQL(context.Background(), "Viewer", map[string]any{"userId": "783214"}); err != nil {
		t.Fatalf("GraphQL: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotMeth != http.MethodGet {
		t.Fatalf("Viewer went out as %s, want GET", gotMeth)
	}
	if len(gotBody) != 0 {
		t.Errorf("the GET request carried a body %q, want none", gotBody)
	}
	if gotQuery.Get("variables") == "" {
		t.Error("the Viewer request carried no variables parameter")
	}
	if gotQuery.Get("features") == "" {
		t.Error("the Viewer request carried no features parameter, but the operation declares switches")
	}
	if gotQuery.Get("fieldToggles") == "" {
		t.Error("the Viewer request carried no fieldToggles parameter, but the operation declares toggles")
	}
}

func TestGraphQLRefusesAnEmptyBodiedRefusalAsASignerOrAuthFailure(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Measured 2026-10-14: a request refused before the operation is routed answers 403 with
		// a zero-byte body.
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := testClient(t, srv, fixtureSession())
	body, err := client.GraphQL(context.Background(), "Viewer", nil)
	if err == nil {
		t.Fatalf("a 403 with an empty body returned %d bytes and no error", len(body))
	}
	if body != nil {
		t.Errorf("a refused request returned %d bytes of body", len(body))
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("the failure is %T (%v), want an *APIError carrying the classified outcome", err, err)
	}
	if apiErr.Outcome != OutcomeSignerOrAuth {
		t.Errorf("the outcome is %q, want %q", apiErr.Outcome, OutcomeSignerOrAuth)
	}
	if apiErr.Status != http.StatusForbidden {
		t.Errorf("the error carries status %d, want 403", apiErr.Status)
	}
	if apiErr.Operation != "Viewer" {
		t.Errorf("the error names %q, want the operation", apiErr.Operation)
	}
	for _, want := range []string{"Viewer", OutcomeSignerOrAuth.Remedy()} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not say %q: %v", want, err)
		}
	}
}

func TestGraphQLCarriesTheUpstreamsOwnMessage(t *testing.T) {
	// A 404 with a JSON body is a retired endpoint, and its message is the only thing that says
	// which one: measured on /1.1/account/verify_credentials.json.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"errors":[{"message":"Sorry, that page does not exist","code":34}]}`)
	}))
	defer srv.Close()

	client := testClient(t, srv, fixtureSession())
	_, err := client.GraphQL(context.Background(), "Viewer", nil)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("the failure is %T (%v), want an *APIError", err, err)
	}
	if apiErr.Outcome != OutcomeEndpointRetired {
		t.Errorf("the outcome is %q, want %q: a 404 with a JSON body is not an unknown operation", apiErr.Outcome, OutcomeEndpointRetired)
	}
	if !strings.Contains(apiErr.Detail, "does not exist") {
		t.Errorf("the error carries %q, want the upstream's own message", apiErr.Detail)
	}
}

func TestGraphQLRefusesWhatItCannotIssue(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("a request that should not have been issued reached the server: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	t.Run("an operation the registry does not have", func(t *testing.T) {
		client := testClient(t, srv, fixtureSession())
		_, err := client.GraphQL(context.Background(), "NoSuchOperation", nil)
		if !errors.Is(err, volatile.ErrUnknownOperation) {
			t.Fatalf("the failure is %v, want one that wraps ErrUnknownOperation", err)
		}
	})

	t.Run("no session", func(t *testing.T) {
		client := testClient(t, srv, session.Session{})
		_, err := client.GraphQL(context.Background(), "Viewer", nil)
		if err == nil || !strings.Contains(err.Error(), "session") {
			t.Fatalf("the failure is %v, want one that says a session is what is missing", err)
		}
	})

	t.Run("a mutation", func(t *testing.T) {
		// This client issues queries only. A mutation is someone else's problem, and with the
		// wire method now a property of the operation there is no method it could be sent with.
		client := testClient(t, srv, fixtureSession())
		mutations := 0
		for _, name := range client.registry.Names() {
			op, _ := client.registry.Lookup(name)
			if op.OperationType == "mutation" {
				mutations++
				if _, err := client.GraphQL(context.Background(), name, nil); err == nil {
					t.Fatalf("the mutation %s was issued", name)
				}
			}
		}
		if mutations == 0 {
			t.Fatal("the registry holds no mutation, so this arm proves nothing")
		}
	})
}

func TestGraphQLRunsTheSignerSeam(t *testing.T) {
	var got string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("x-client-transaction-id")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	client := testClient(t, srv, fixtureSession()).WithSigner(stubSigner{header: "x-client-transaction-id", value: "abc"})
	if _, err := client.GraphQL(context.Background(), "Viewer", nil); err != nil {
		t.Fatalf("GraphQL: %v", err)
	}
	if got != "abc" {
		t.Errorf("the request carried x-client-transaction-id %q, want what the signer set", got)
	}

	t.Run("a signer that fails aborts the request", func(t *testing.T) {
		got = ""
		client := testClient(t, srv, fixtureSession()).WithSigner(stubSigner{err: errors.New("no key material")})
		_, err := client.GraphQL(context.Background(), "Viewer", nil)
		if err == nil || !strings.Contains(err.Error(), "no key material") {
			t.Fatalf("the failure is %v, want the signer's own error", err)
		}
		if got != "" {
			t.Errorf("a request the signer refused still reached the upstream (%q)", got)
		}
	})
}

// stubSigner is a Signer that sets one header, or fails.
type stubSigner struct {
	header, value string
	err           error
}

func (s stubSigner) Header(_ context.Context, req *http.Request) error {
	if s.err != nil {
		return s.err
	}
	req.Header.Set(s.header, s.value)
	return nil
}

// TestGraphQLFallsBackToTheEmbeddedSnapshot covers the client that was built without a registry
// of its own: it resolves through the embedded snapshot rather than refusing, which is what the
// zero-value-registry path in operation() is for.
func TestGraphQLFallsBackToTheEmbeddedSnapshot(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/i/api/graphql/" + viewerID + "/Viewer"; r.URL.Path != want {
			t.Errorf("the request path is %q, want %q", r.URL.Path, want)
		}
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	client := testClient(t, srv, fixtureSession())
	client.registry = nil
	if _, err := client.GraphQL(context.Background(), "Viewer", nil); err != nil {
		t.Fatalf("a client with no registry of its own could not resolve Viewer: %v", err)
	}
}

// jsonUnmarshal is encoding/json's Unmarshal under a name that keeps this file's import list
// short; the assertions above are about the request, not about the decoder.
func jsonUnmarshal(data string, into any) error {
	return json.Unmarshal([]byte(data), into)
}
