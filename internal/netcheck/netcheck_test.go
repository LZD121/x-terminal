package netcheck

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

// The fixture has the shape of the real cookie — auth_token 40 bytes, ct0 160 — and is not
// a credential. It is the needle the request-inspection assertions below search for, and it
// is never printed.
var (
	fixtureToken = "0123456789abcdef0123456789abcdef01234567"
	fixtureCTO   = strings.Repeat("b1c2d3e4", 20)
)

// deadSessionLocation is what x.com answered /home with when the cookie was dead
// (measured 2026-10-14): a redirect into the login flow, with a zero-byte body.
const deadSessionLocation = "/i/jf/onboarding/web?redirect_after_login=%2Fhome&mode=login"

func fixtureSession() session.Session {
	return session.Session{AuthToken: session.Secret(fixtureToken), CTO: session.Secret(fixtureCTO), Source: "/tmp/session"}
}

// testTarget points the whole check at one httptest.Server: the resolver answers with the
// server's address, the dialer connects to its listener, and the client speaks to it
// whatever host the request names. Nothing here leaves the machine.
//
// address is what the check treats as the configured proxy: "" means a direct dial, and
// anything else is dialed as a bare TCP connect before the request stages run.
func testTarget(t *testing.T, srv *httptest.Server, address string) *Target {
	t.Helper()

	transport := srv.Client().Transport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, srv.Listener.Addr().String())
	}
	// The test server's certificate is issued for 127.0.0.1 and example.com while the requests
	// below are built for x.com: what is under test is the states the check names, not TLS
	// trust. The transport around it is a real one, TLS handshake and all.
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return &Target{
		client: &xweb.Client{
			// The redirect policy is part of the client the binary builds, not a detail of this
			// test: the expired-session answer is a 307, and a client that followed it would report
			// the login page's 200 as an accepted session. TestCheckNamesARejectedSession is the
			// arm that fails if this is dropped.
			HTTP:    &http.Client{Transport: transport, CheckRedirect: xweb.NoRedirect},
			Limiter: xweb.NewLimiter(0, 100),
		},
		address: address,
		resolve: func(context.Context, string) ([]string, error) { return []string{"127.0.0.1"}, nil },
		dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, addr)
		},
	}
}

// page writes a page: the shape a reachable x.com answers with.
func page(w http.ResponseWriter, body string) {
	w.Header().Set("content-type", "text/html; charset=utf-8")
	fmt.Fprint(w, "<!DOCTYPE html><html><head><title>x</title></head><body>"+body+"</body></html>")
}

// viewerAnswer is what the Viewer operation answers with: the account the session belongs to, in
// the current schema — core, avatar, no legacy section at all.
const viewerAnswer = `{"data":{"viewer":{"user_results":{"result":{"__typename":"User","rest_id":"783214","core":{"screen_name":"X","name":"X"},"avatar":{"image_url":"https://pbs.twimg.com/a.jpg"},"has_graduated_access":true}}}}}`

// viewerPath is the path every GraphQL request is built for, whatever operation it names: the id
// is in the path, and it is not written out here — the registry is where ids live.
const viewerPath = "/i/api/graphql/"

// pageHandler serves "/" and "/home" as pages and the API path as a Viewer answer: a path that is
// reachable, a session that is accepted, and an API read that carries an account.
func pageHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/" || r.URL.Path == "/home":
			page(w, strings.Repeat("shell ", 100))
		case strings.HasPrefix(r.URL.Path, viewerPath):
			w.Header().Set("content-type", "application/json")
			fmt.Fprint(w, viewerAnswer)
		default:
			http.NotFound(w, r)
		}
	})
}

// failures lists the outcomes that are failures, so a test can say which of the results it
// expects to have gone wrong.
func failures(results []Result) []Outcome {
	var out []Outcome
	for _, r := range results {
		if r.Outcome.Failed() {
			out = append(out, r.Outcome)
		}
	}
	return out
}

func outcomes(results []Result) []Outcome {
	out := make([]Outcome, 0, len(results))
	for _, r := range results {
		out = append(out, r.Outcome)
	}
	return out
}

// TestCheckReportsAuthenticatedReadOK is the anti-vacuity test for this file: a run that
// should succeed has to come back with the one success outcome, spend the requests it
// reports, carry the session cookie on the authenticated read and no auth header anywhere —
// and it has to have asked a real server for those two paths, or nothing above means
// anything.
func TestCheckReportsAuthenticatedReadOK(t *testing.T) {
	var (
		mu      sync.Mutex
		paths   []string
		cookies = map[string]string{}
		auths   = map[string]string{}
	)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		cookies[r.URL.Path] = r.Header.Get("Cookie")
		auths[r.URL.Path] = r.Header.Get("Authorization")
		mu.Unlock()

		if r.Host != "x.com" {
			t.Errorf("the request was built for host %q, want x.com", r.Host)
		}
		pageHandler().ServeHTTP(w, r)
	}))
	defer srv.Close()

	target := testTarget(t, srv, srv.Listener.Addr().String())
	results := CheckWith(context.Background(), target, fixtureSession())

	want := []Outcome{OutcomeRouteOK, OutcomeProxyOK, OutcomeUpstreamOK, OutcomeAuthenticatedOK, OutcomeAPIReadOK}
	if got := outcomes(results); !equal(got, want) {
		t.Fatalf("outcomes = %v, want %v (details: %s)", got, want, details(results))
	}
	if failed := failures(results); len(failed) != 0 {
		t.Errorf("a successful run reports failures: %v", failed)
	}

	// The anti-vacuity arm: the fake upstream served exactly the three requests the stages make,
	// so the assertions about headers and cookies below are about requests that happened.
	apiPath := viewerPath + viewerID(t) + "/Viewer"
	if got := strings.Join(paths, " "); got != "/ /home "+apiPath {
		t.Fatalf("the fake upstream was asked for %q, want exactly %q", got, "/ /home "+apiPath)
	}

	last := results[len(results)-1]
	if last.Outcome != OutcomeAPIReadOK {
		t.Fatalf("the verdict is %q, want %q", last.Outcome, OutcomeAPIReadOK)
	}
	if last.Elapsed <= 0 {
		t.Errorf("the API read reports Elapsed = %s: it measured nothing", last.Elapsed)
	}
	if !strings.Contains(last.Detail, "@X") {
		t.Errorf("the verdict does not name the account the answer carries: %q", last.Detail)
	}

	// The request classes: the reachability stage carries no credentials at all, the authenticated
	// navigation read carries the cookie and no bearer (a bearer on a navigation request is
	// answered 401 with a zero-byte body), and the API read carries both.
	if cookies["/"] != "" || auths["/"] != "" {
		t.Errorf("the reachability request carried credentials: cookie %q, authorization %q", cookies["/"], auths["/"])
	}
	for _, needle := range []string{"auth_token=" + fixtureToken, "ct0=" + fixtureCTO} {
		name := strings.SplitN(needle, "=", 2)[0]
		if !strings.Contains(cookies["/home"], needle) {
			t.Errorf("the authenticated navigation read did not carry the %s cookie", name)
		}
		if !strings.Contains(cookies[apiPath], needle) {
			t.Errorf("the API read did not carry the %s cookie", name)
		}
	}
	if got := auths["/home"]; got != "" {
		t.Errorf("the navigation read carried an authorization header (%q)", got)
	}
	if got := auths[apiPath]; !strings.HasPrefix(got, "Bearer ") {
		t.Errorf("the API read carried no authorization header (%q), which the upstream answers with 403 and an empty body", got)
	}
	if used, limited, reset := target.Client().Limiter.Budget(); used != 3 || limited != 0 || reset.IsZero() {
		t.Errorf("Budget() = (%d used, %d limited, reset %s), want (3, 0, a time)", used, limited, reset)
	}
}

// viewerID reads the Viewer operation's id out of the registry: this test is about the request
// classes, and an id literal here would be one more place the id could go stale.
func viewerID(t *testing.T) string {
	t.Helper()
	op, err := volatile.Lookup("Viewer")
	if err != nil {
		t.Fatalf("the registry has no Viewer operation: %v", err)
	}
	return op.QueryID
}

func TestCheckNamesNoRoute(t *testing.T) {
	t.Run("x.com does not resolve", func(t *testing.T) {
		srv := httptest.NewTLSServer(pageHandler())
		defer srv.Close()

		target := testTarget(t, srv, srv.Listener.Addr().String())
		target.resolve = func(context.Context, string) ([]string, error) {
			return nil, &net.DNSError{Err: "no such host", Name: "x.com", IsNotFound: true}
		}
		results := CheckWith(context.Background(), target, fixtureSession())

		if got := outcomes(results); !equal(got, []Outcome{OutcomeNoRoute}) {
			t.Fatalf("outcomes = %v, want just %q (details: %s)", got, OutcomeNoRoute, details(results))
		}
		if !strings.Contains(results[0].Detail, "did not resolve") {
			t.Errorf("the detail does not say the name did not resolve: %q", results[0].Detail)
		}
	})

	t.Run("a direct dial whose address accepts nothing", func(t *testing.T) {
		srv := httptest.NewTLSServer(pageHandler())
		defer srv.Close()

		// No proxy: the first hop is x.com itself, and on this host that hop is what is blocked
		// when the proxy is not running — which must be reported as no route rather than as a
		// session that expired.
		target := testTarget(t, srv, "")
		target.dial = func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("dial tcp 162.159.140.229:443: i/o timeout")
		}
		results := CheckWith(context.Background(), target, fixtureSession())

		if got := outcomes(results); !equal(got, []Outcome{OutcomeNoRoute}) {
			t.Fatalf("outcomes = %v, want just %q (details: %s)", got, OutcomeNoRoute, details(results))
		}
		if !strings.Contains(results[0].Detail, "did not accept a TCP connection") {
			t.Errorf("the detail does not name the connection that failed: %q", results[0].Detail)
		}
	})
}

func TestCheckNamesProxyRefused(t *testing.T) {
	srv := httptest.NewTLSServer(pageHandler())
	defer srv.Close()

	target := testTarget(t, srv, "127.0.0.1:9")
	target.dial = func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("dial tcp 127.0.0.1:9: connect: connection refused")
	}
	results := CheckWith(context.Background(), target, fixtureSession())

	// Two results and no more: a dead proxy is not reported as a session that expired, and the
	// check stops where the path broke rather than spending requests on a session it cannot
	// reach.
	if got := outcomes(results); !equal(got, []Outcome{OutcomeRouteOK, OutcomeProxyRefused}) {
		t.Fatalf("outcomes = %v, want %v (details: %s)", got, []Outcome{OutcomeRouteOK, OutcomeProxyRefused}, details(results))
	}
	if !strings.Contains(results[1].Detail, "127.0.0.1:9") {
		t.Errorf("the detail does not name the proxy: %q", results[1].Detail)
	}
	if used, _, _ := target.Client().Limiter.Budget(); used != 0 {
		t.Errorf("the check spent %d upstream requests before finding the proxy dead, want 0", used)
	}
}

func TestCheckNamesUpstreamBlocked(t *testing.T) {
	// What a proxy answers when it is up and cannot reach what it was asked for.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, "Bad Gateway")
	}))
	defer srv.Close()

	target := testTarget(t, srv, srv.Listener.Addr().String())
	results := CheckWith(context.Background(), target, fixtureSession())

	want := []Outcome{OutcomeRouteOK, OutcomeProxyOK, OutcomeUpstreamBlocked}
	if got := outcomes(results); !equal(got, want) {
		t.Fatalf("outcomes = %v, want %v (details: %s)", got, want, details(results))
	}
	if !strings.Contains(results[2].Detail, "load shed") {
		t.Errorf("the detail does not carry the classifier's name for a 502: %q", results[2].Detail)
	}
}

func TestCheckNamesTheCloudflareChallenge(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("cf-mitigated", "challenge")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "<!DOCTYPE html><html><head><title>Just a moment...</title></head></html>")
	}))
	defer srv.Close()

	target := testTarget(t, srv, srv.Listener.Addr().String())
	results := CheckWith(context.Background(), target, fixtureSession())

	want := []Outcome{OutcomeRouteOK, OutcomeProxyOK, OutcomeChallenge}
	if got := outcomes(results); !equal(got, want) {
		t.Fatalf("outcomes = %v, want %v (details: %s)", got, want, details(results))
	}
}

func TestCheckNamesARejectedSession(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/home" {
			w.Header().Set("location", deadSessionLocation)
			w.Header().Set("set-cookie", "auth_token=; Max-Age=-1791955999; Path=/")
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
		page(w, strings.Repeat("logged out shell ", 100))
	}))
	defer srv.Close()

	target := testTarget(t, srv, srv.Listener.Addr().String())
	results := CheckWith(context.Background(), target, fixtureSession())

	want := []Outcome{OutcomeRouteOK, OutcomeProxyOK, OutcomeUpstreamOK, OutcomeSessionRejected}
	if got := outcomes(results); !equal(got, want) {
		t.Fatalf("outcomes = %v, want %v (details: %s)", got, want, details(results))
	}
	if detail := results[3].Detail; !strings.Contains(detail, "auth rejected") {
		t.Errorf("the detail does not carry the classifier's name for an expired cookie: %q", detail)
	}
}

// TestCheckSkipsTheSessionStageWithoutOne covers the rule that a missing cookie is a state
// rather than a failure: the path can be perfect while there is nothing to authenticate
// with, and the line has to say which of the two it is.
func TestCheckSkipsTheSessionStageWithoutOne(t *testing.T) {
	srv := httptest.NewTLSServer(pageHandler())
	defer srv.Close()

	target := testTarget(t, srv, srv.Listener.Addr().String())
	results := CheckWith(context.Background(), target, session.Session{})

	want := []Outcome{OutcomeRouteOK, OutcomeProxyOK, OutcomeUpstreamOK, OutcomeSessionSkipped}
	if got := outcomes(results); !equal(got, want) {
		t.Fatalf("outcomes = %v, want %v (details: %s)", got, want, details(results))
	}
	if failed := failures(results); len(failed) != 0 {
		t.Errorf("a skipped session is reported as a failure: %v", failed)
	}
	if detail := results[3].Detail; !strings.Contains(detail, "not attempted") {
		t.Errorf("the line does not say the read was skipped: %q", detail)
	}
	if results[3].Elapsed != 0 {
		t.Errorf("the skipped stage took %s: nothing was attempted", results[3].Elapsed)
	}
	if used, _, _ := target.Client().Limiter.Budget(); used != 1 {
		t.Errorf("the check spent %d requests with no session, want 1 (the reachability probe alone)", used)
	}
}

// TestNewTargetWithClientSpendsFromTheClientsOwnLimiter is the reason that constructor exists: a
// program that checks the path while it is working shares the client its reads go through, so the
// request the check spends is a request the budget line that reads that client can see. A target with
// a client of its own would be a request no budget reports — the exact drift this project exists to
// keep out of the status line.
func TestNewTargetWithClientSpendsFromTheClientsOwnLimiter(t *testing.T) {
	srv := httptest.NewTLSServer(pageHandler())
	defer srv.Close()

	// The client a feed would already have: a real transport and a real limiter, no session.
	transport := srv.Client().Transport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, srv.Listener.Addr().String())
	}
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &xweb.Client{
		HTTP:    &http.Client{Transport: transport, CheckRedirect: xweb.NoRedirect},
		Limiter: xweb.NewLimiter(0, 10),
	}

	target, err := NewTargetWithClient(config.Config{}, client)
	if err != nil {
		t.Fatalf("building a target over a client: %v", err)
	}
	if target.Client() != client {
		t.Fatalf("the target speaks through another client: the request it spends is not the one the budget line counts")
	}
	// The path is pointed at the test server without touching the client: what is under test is whose
	// limiter the request passes through, not how a name resolves. The listener stands in for the
	// configured proxy, exactly as testTarget does it, because a configured first hop is what keeps the
	// route stage from dialing x.com itself.
	target.resolve = func(context.Context, string) ([]string, error) { return []string{"127.0.0.1"}, nil }
	target.address = srv.Listener.Addr().String()

	results := CheckWith(context.Background(), target, session.Session{})
	if got, want := outcomes(results), []Outcome{OutcomeRouteOK, OutcomeProxyOK, OutcomeUpstreamOK, OutcomeSessionSkipped}; !equal(got, want) {
		t.Fatalf("outcomes = %v, want %v (details: %s)", got, want, details(results))
	}
	if used, _, _ := client.Limiter.Budget(); used != 1 {
		t.Errorf("the check spent %d requests on the client's own limiter, want its one navigation read", used)
	}
}

// TestCheckReportsAnUnusableProxySetting covers the one failure that happens before any
// socket does: a proxy setting the transport cannot use. It must be named as such rather than
// reported as a route that did not resolve, and it must not echo the setting.
func TestCheckReportsAnUnusableProxySetting(t *testing.T) {
	results := Check(context.Background(), config.Config{Proxy: "ftp://user:secret@proxy.example:21"}, session.Session{})

	if got := outcomes(results); !equal(got, []Outcome{OutcomeConfigFailed}) {
		t.Fatalf("outcomes = %v, want just %q (details: %s)", got, OutcomeConfigFailed, details(results))
	}
	if !strings.Contains(results[0].Detail, "scheme") {
		t.Errorf("the detail does not name what was wrong with the setting: %q", results[0].Detail)
	}
	if strings.Contains(results[0].Detail, "secret") {
		t.Errorf("the detail quotes the proxy credential: %q", results[0].Detail)
	}
	if !results[0].Outcome.Failed() {
		t.Error("an unusable transport is not reported as a failure")
	}
}

// TestClipBoundsEveryLine: no stage may hand a caller an unbounded string, because a detail
// line ends up in a terminal and in whatever a user pastes into a bug report.
func TestClipBoundsEveryLine(t *testing.T) {
	long := strings.Repeat("e", detailLimit*3)
	got := clip(long)
	if len(got) > detailLimit+len("...") {
		t.Fatalf("clip returned %d bytes, want at most %d", len(got), detailLimit+len("..."))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("clip cut a long line without saying so")
	}
	if short := clip("a line that fits"); short != "a line that fits" {
		t.Errorf("clip changed a short line: %q", short)
	}
	// A multi-byte rune must not be cut in half: the result has to stay valid text.
	multi := strings.Repeat("é", detailLimit)
	if clipped := clip(multi); !strings.HasSuffix(strings.TrimSuffix(clipped, "..."), "é") {
		t.Errorf("clip split a multi-byte rune: %q", clipped[len(clipped)-8:])
	}
}

// TestCheckSpendsExactlyThreeRequestsOnASuccessfulRun is this file's guard against a stage that
// quietly makes an extra request — or quietly stops making one — and against the check ever
// reaching a host other than the one the test handed it.
func TestCheckSpendsExactlyThreeRequestsOnASuccessfulRun(t *testing.T) {
	var (
		mu    sync.Mutex
		hosts []string
	)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hosts = append(hosts, r.Host)
		mu.Unlock()
		pageHandler().ServeHTTP(w, r)
	}))
	defer srv.Close()

	target := testTarget(t, srv, srv.Listener.Addr().String())
	results := CheckWith(context.Background(), target, fixtureSession())

	if failed := failures(results); len(failed) != 0 {
		t.Fatalf("the run failed: %v (details: %s)", failed, details(results))
	}
	if len(hosts) != 3 {
		t.Fatalf("the fake upstream saw %d requests, want 3 (the reachability probe and the two authenticated reads): %v", len(hosts), hosts)
	}
	for _, host := range hosts {
		if host != "x.com" {
			t.Errorf("a request went to %q; every request the check makes is built for x.com and routed by the transport", host)
		}
	}
}

func equal(got, want []Outcome) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func details(results []Result) string {
	var out strings.Builder
	for _, r := range results {
		fmt.Fprintf(&out, "\n  %s: %s (%s)", r.Outcome, r.Detail, r.Elapsed.Round(time.Microsecond))
	}
	return out.String()
}

// TestSizeDoesNotPassThePeekOffAsTheBody: a gzip'd or chunked body arrives with no declared
// length, and the Detail must not then present the four kilobytes this package read as if
// they were the whole response. The live upstream answers exactly that way — gzip is the
// transport's default, so both live stages came back without a length — and an httptest
// response cannot be made to arrive that way on demand, so both shapes are pinned here.
func TestSizeDoesNotPassThePeekOffAsTheBody(t *testing.T) {
	unknown := response{status: http.StatusOK, prefix: make([]byte, peek), length: -1}
	if got := unknown.size(); !strings.Contains(got, "declared no length") {
		t.Errorf("a body of unknown length is described as %q", got)
	} else if !strings.Contains(got, fmt.Sprintf("at least %d bytes", peek)) {
		t.Errorf("the peek is not named as the least that was seen: %q", got)
	}

	known := response{status: http.StatusOK, prefix: make([]byte, peek), length: 308706}
	if got := known.size(); got != "308706 bytes" {
		t.Errorf("a declared length is described as %q, want the declared number", got)
	}
}

// TestCheckReportsABodyWithoutADeclaredLengthWithoutInventingOne drives the same shape
// end to end: a flushed response arrives chunked, the client learns no length, and the Detail
// has to say so. This is the arm that catches a fetch that helpfully replaces the missing
// length with what it read — the live run showed exactly that, printing the 4096-byte peek as
// if it were the body.
func TestCheckReportsABodyWithoutADeclaredLengthWithoutInventingOne(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, viewerPath) {
			w.Header().Set("content-type", "application/json")
			fmt.Fprint(w, viewerAnswer)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		page(w, strings.Repeat("shell ", 100))
	}))
	defer srv.Close()

	target := testTarget(t, srv, srv.Listener.Addr().String())
	results := CheckWith(context.Background(), target, fixtureSession())

	if failed := failures(results); len(failed) != 0 {
		t.Fatalf("the run failed: %v (details: %s)", failed, details(results))
	}
	// The navigation read is the stage whose Detail describes a body's size; the API read's line
	// names the account instead.
	var navigation string
	for _, r := range results {
		if r.Outcome == OutcomeAuthenticatedOK {
			navigation = r.Detail
		}
	}
	if !strings.Contains(navigation, "declared no length") {
		t.Errorf("a chunked body is reported as if its size were known: %q", navigation)
	}
}
