package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// viewerAnswer is what the Viewer operation answers with: the account the session belongs to.
const viewerAnswer = `{"data":{"viewer":{"user_results":{"result":{"__typename":"User","rest_id":"783214","core":{"screen_name":"X","name":"X"},"avatar":{"image_url":"https://pbs.twimg.com/a.jpg"},"has_graduated_access":true}}}}}`

// pointAtTestServer replaces the verb's client seam with one that dials srv whatever host the
// request names, so a real verb runs its whole path — config, session, request, parse, print —
// against a fake upstream. The client is built by hand here, which is also the proof that a
// hand-built *xweb.Client outside that package can carry a session.
func pointAtTestServer(t *testing.T, srv *httptest.Server) {
	t.Helper()
	previous := newAPIClient
	newAPIClient = func(_ config.Config, sess session.Session) (*xweb.Client, error) {
		transport := srv.Client().Transport.(*http.Transport).Clone()
		transport.Proxy = nil
		transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, srv.Listener.Addr().String())
		}
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		return (&xweb.Client{
			HTTP:    &http.Client{Transport: transport, CheckRedirect: xweb.NoRedirect},
			Limiter: xweb.NewLimiter(0, 10),
		}).WithSession(sess), nil
	}
	t.Cleanup(func() { newAPIClient = previous })
}

func TestWhoamiNamesTheAccount(t *testing.T) {
	var paths []string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, viewerAnswer)
	}))
	defer srv.Close()

	dir := t.TempDir()
	hermeticConfig(t, writeSession(t, dir, "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n"))
	pointAtTestServer(t, srv)

	code, stdout, stderr := run(t, Whoami)
	if code != ExitOK {
		t.Fatalf("whoami exited %d (%s)", code, stderr)
	}
	if !strings.Contains(stdout, "@X — X") {
		t.Errorf("whoami printed %q, want the handle and the name", stdout)
	}
	if !strings.Contains(stdout, "session: present") || !strings.Contains(stdout, "ct0 160 bytes") {
		t.Errorf("whoami printed %q, want the session's own state beside the handle", stdout)
	}
	if len(paths) != 1 || !strings.HasPrefix(paths[0], "/i/api/graphql/") {
		t.Errorf("whoami asked for %v, want exactly one API operation", paths)
	}
	// The first successful run prints the account-risk statement, on stderr, once.
	if !strings.Contains(stderr, "session cookie") {
		t.Errorf("the first run printed no risk statement: %q", stderr)
	}
	assertNoCredential(t, stdout, stderr)

	// The second run does not print it again, and says the same thing about the account.
	code, stdout, stderr = run(t, Whoami)
	if code != ExitOK {
		t.Fatalf("the second whoami exited %d (%s)", code, stderr)
	}
	if strings.Contains(stderr, "session cookie") {
		t.Errorf("the risk statement was printed a second time: %q", stderr)
	}
	if !strings.Contains(stdout, "@X — X") {
		t.Errorf("the second whoami printed %q", stdout)
	}
}

func TestWhoamiReportsARejectedSession(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Measured 2026-10-14: refused before the operation was routed, with no body at all.
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	dir := t.TempDir()
	hermeticConfig(t, writeSession(t, dir, "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n"))
	pointAtTestServer(t, srv)

	code, stdout, stderr := run(t, Whoami)
	if code != ExitError {
		t.Fatalf("whoami exited %d, want %d", code, ExitError)
	}
	if stdout != "" {
		t.Errorf("a failed read still printed to stdout: %q", stdout)
	}
	for _, want := range []string{"Viewer", "signer or auth rejected", "403"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("the refusal does not name %q: %q", want, stderr)
		}
	}
	assertNoCredential(t, stdout, stderr)
}

func TestWhoamiWithoutASessionSaysSo(t *testing.T) {
	hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))
	pointAtTestServer(t, httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("whoami asked %s for %s with no session to ask with", r.Host, r.URL.Path)
	})))

	code, stdout, stderr := run(t, Whoami)
	if code != ExitError {
		t.Fatalf("whoami exited %d, want %d", code, ExitError)
	}
	if stdout != "" {
		t.Errorf("whoami printed to stdout with no session: %q", stdout)
	}
	if !strings.Contains(stderr, "no session") {
		t.Errorf("the failure does not name the missing session: %q", stderr)
	}
}

func TestWhoamiRefusesAnAnswerWithoutAHandle(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A body shaped like the retired schema: no core, no rest_id, nothing this client reads.
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{"data":{"viewer":{"user_results":{"result":{"__typename":"User","legacy":{"screen_name":"oldname","followers_count":1234}}}}}}`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	hermeticConfig(t, writeSession(t, dir, "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n"))
	pointAtTestServer(t, srv)

	code, stdout, stderr := run(t, Whoami)
	if code != ExitError {
		t.Fatalf("whoami exited %d, want %d", code, ExitError)
	}
	if stdout != "" {
		t.Errorf("an unreadable answer still printed a handle: %q", stdout)
	}
	if !strings.Contains(stderr, "data.viewer.user_results.result") {
		t.Errorf("the failure does not name the path that was missing: %q", stderr)
	}
}

func TestWhoamiTakesNoArguments(t *testing.T) {
	code, stdout, stderr := run(t, Whoami, "extra")
	if code != ExitUsage {
		t.Fatalf("whoami extra exited %d, want %d", code, ExitUsage)
	}
	if stdout != "" {
		t.Errorf("a rejected invocation wrote to stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "usage: x-terminal") {
		t.Errorf("the rejection does not carry the usage: %q", stderr)
	}
}

// assertNoCredential is the leak assertion every verb's tests share: the fixture cookie's first
// eight characters appear nowhere in what the verb printed.
func assertNoCredential(t *testing.T, output ...string) {
	t.Helper()
	needle := fixtureToken[:8]
	for _, text := range output {
		if strings.Contains(text, needle) {
			t.Errorf("the output carries the session cookie: %q", text)
		}
	}
}

// TestTheFixtureHasTheRealCookieShape keeps the fixture honest: the assertions above are about a
// session whose shape is the real one — a 40-byte auth_token and a 160-byte ct0 — so that a length
// shown in a bug report and a length asserted in a test mean the same thing.
func TestTheFixtureHasTheRealCookieShape(t *testing.T) {
	if len(fixtureToken) != 40 || len(fixtureCTO) != 160 {
		t.Errorf("the fixture is %d/%d bytes, want the 40/160 the real cookie is", len(fixtureToken), len(fixtureCTO))
	}
	if strings.ContainsAny(fixtureToken, "GHIJKLMNOPQRSTUVWXYZ") {
		t.Error("the fixture token looks like base64 of something real")
	}
}
