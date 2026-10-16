package opsgen

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// The fixture has the shape of the real cookie — auth_token 40 bytes, ct0 160 — and is not a
// credential. It is the needle the request-inspection assertions search for.
var (
	fixtureToken = "0123456789abcdef0123456789abcdef01234567"
	fixtureCTO   = strings.Repeat("b1c2d3e4", 20)
)

func fixtureSession() session.Session {
	return session.Session{AuthToken: session.Secret(fixtureToken), CTO: session.Secret(fixtureCTO), Source: "/tmp/session"}
}

// testClient points a real client at one httptest.Server: whatever host a request names, the
// dial goes to the server's listener. That is what lets a test exercise the refresh path —
// the shell on x.com, the bundle on abs.twimg.com — without either host resolving, and it is
// why the host each request carried can be asserted afterwards.
func testClient(t *testing.T, srv *httptest.Server) *xweb.Client {
	t.Helper()
	transport := srv.Client().Transport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, srv.Listener.Addr().String())
	}
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return &xweb.Client{
		HTTP:    &http.Client{Transport: transport, CheckRedirect: xweb.NoRedirect},
		Limiter: xweb.NewLimiter(0, 100),
	}
}

// bundleSlice is the recorded piece of the live bundle, with its provenance comment stripped
// by the parser's own rules (it is a comment, not a record).
func bundleSlice(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "bundle-slice.js"))
	if err != nil {
		t.Fatalf("reading the recorded bundle slice: %v", err)
	}
	return data
}

// TestParseReadsTheRealBundleShape is the anti-vacuity test of this file: the fixture is a
// verbatim slice of the bundle recorded on 2026-10-14, so a parser that only understood a
// bundle written by its own author would fail here.
func TestParseReadsTheRealBundleShape(t *testing.T) {
	records, err := Parse(bundleSlice(t))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(records) != 7 {
		t.Fatalf("the recorded slice yielded %d records, want the 7 it contains: %v", len(records), names(records))
	}

	byName := map[string]Record{}
	for _, record := range records {
		byName[record.Name] = record
	}
	disclosure, ok := byName["AddContentDisclosure"]
	switch {
	case !ok:
		t.Fatalf("AddContentDisclosure is missing: %v", names(records))
	case disclosure.QueryID != "D1nwFlsu_qHsX92YzoRaaA":
		t.Errorf("AddContentDisclosure has id %q, want the one the bundle ships", disclosure.QueryID)
	case disclosure.OperationType != "mutation":
		t.Errorf("AddContentDisclosure is a %q, want the mutation the bundle declares", disclosure.OperationType)
	case len(disclosure.FeatureSwitches) != 0 || len(disclosure.FieldToggles) != 0:
		t.Errorf("AddContentDisclosure declares %v / %v, want the empty lists it ships", disclosure.FeatureSwitches, disclosure.FieldToggles)
	}

	blocked, ok := byName["BlockedAccountsAll"]
	if !ok {
		t.Fatalf("BlockedAccountsAll is missing: %v", names(records))
	}
	switch {
	case blocked.QueryID != "9gQUtZmRLsKAsGCedwML0g":
		t.Errorf("BlockedAccountsAll has id %q, want the one the bundle ships", blocked.QueryID)
	case blocked.OperationType != "query":
		t.Errorf("BlockedAccountsAll is a %q, want a query", blocked.OperationType)
	case len(blocked.FeatureSwitches) < 5:
		t.Errorf("BlockedAccountsAll came back with %d feature switches, want the list the bundle ships: %v", len(blocked.FeatureSwitches), blocked.FeatureSwitches)
	case len(blocked.FieldToggles) == 0:
		t.Error("BlockedAccountsAll came back with no field toggles, but the bundle declares them with its id")
	}
	if blocked.FeatureSwitches[0] != "rweb_video_screen_enabled" {
		t.Errorf("the feature switches start with %q, want the first one the bundle ships", blocked.FeatureSwitches[0])
	}
}

func TestParseRefusesABundleWithNoRecords(t *testing.T) {
	_, err := Parse([]byte(`(globalThis.webpackChunk_twitter_responsive_web||=[]).push([[38792],{203642(e){e.exports={}}}]);`))
	if err == nil {
		t.Fatal("a bundle with no operation records parsed as if it had some")
	}
	if !strings.Contains(err.Error(), "no {queryId,operationName,operationType,metadata} records") {
		t.Errorf("the error does not say what the parser was looking for: %v", err)
	}
}

func TestParseRefusesABundleThatNamesOneOperationTwice(t *testing.T) {
	bundle := []byte(`{queryId:"D1nwFlsu_qHsX92YzoRaaA",operationName:"Viewer",operationType:"query"}` +
		`{queryId:"r7VUmxbfqNkx7uwjgONSNw",operationName:"Viewer",operationType:"query"}`)
	_, err := Parse(bundle)
	if err == nil {
		t.Fatal("two different ids for one operation name were merged as if one of them were current")
	}
	if !strings.Contains(err.Error(), "Viewer") {
		t.Errorf("the error does not name the operation: %v", err)
	}
}

func TestBundleURLReadsBothShellShapes(t *testing.T) {
	for _, tc := range []struct{ name, shell, want string }{
		{
			name:  "the absolute URL the build links today",
			shell: `<script src="https://abs.twimg.com/responsive-web/client-web/main.a9b169a91ce3b55ba.js"></script>`,
			want:  "https://abs.twimg.com/responsive-web/client-web/main.a9b169a91ce3b55ba.js",
		},
		{
			name:  "the bare path an older build reconstructs from its hash map",
			shell: `{"main":"responsive-web/client-web/main.a9b169a91ce3b55ba.js"}`,
			want:  "https://abs.twimg.com/responsive-web/client-web/main.a9b169a91ce3b55ba.js",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BundleURL([]byte(tc.shell))
			if err != nil {
				t.Fatalf("BundleURL: %v", err)
			}
			if got != tc.want {
				t.Errorf("BundleURL = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBundleURLRefusesALoggedOutShell(t *testing.T) {
	// The logged-out entry chunk, measured on 2026-10-14: it references its own bundle and no
	// responsive-web main bundle at all, because the operation map only ships to an
	// authenticated client.
	shell := `<script src="https://abs.twimg.com/x-web/x-web/entry-client-logged-out-BJDlp_4g.js"></script>`
	_, err := BundleURL([]byte(shell))
	if err == nil {
		t.Fatal("a shell with no main bundle produced a URL, which would then be fetched and mined")
	}
	if !strings.Contains(err.Error(), "responsive-web/client-web/main.<hash>a.js") {
		t.Errorf("the error does not name what it looked for: %v", err)
	}
}

func TestFetchReadsTheShellAndTheBundle(t *testing.T) {
	bundle := bundleSlice(t)

	var (
		mu       sync.Mutex
		paths    []string
		hosts    []string
		cookies  []string
		auths    []string
		requests int
	)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		paths = append(paths, r.URL.Path)
		hosts = append(hosts, r.Host)
		cookies = append(cookies, r.Header.Get("Cookie"))
		auths = append(auths, r.Header.Get("Authorization"))
		mu.Unlock()

		if r.URL.Path == "/home" {
			// The authenticated shell, as x.com serves it: a page that links the main bundle.
			w.Header().Set("content-type", "text/html; charset=utf-8")
			w.Write([]byte(`<!DOCTYPE html><html><head><script src="https://abs.twimg.com/responsive-web/client-web/main.a9b169a91ce3b55ba.js"></script></head><body>shell</body></html>`))
			return
		}
		w.Header().Set("content-type", "application/javascript")
		w.Write(bundle)
	}))
	defer srv.Close()

	client := testClient(t, srv)
	refresh, err := Fetch(context.Background(), client, fixtureSession(), ShellURL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if got := len(refresh.Records); got != 7 {
		t.Errorf("the refresh read %d records, want the 7 in the bundle", got)
	}
	if refresh.BundleURL != "https://abs.twimg.com/responsive-web/client-web/main.a9b169a91ce3b55ba.js" {
		t.Errorf("the refresh mined %q", refresh.BundleURL)
	}
	sum := sha256.Sum256(bundle)
	if want := hex.EncodeToString(sum[:]); refresh.SHA256 != want {
		t.Errorf("the refresh recorded sha256 %s, want %s", refresh.SHA256, want)
	}
	if refresh.Bytes != len(bundle) {
		t.Errorf("the refresh recorded %d bytes, want %d", refresh.Bytes, len(bundle))
	}
	if refresh.At.IsZero() {
		t.Error("the refresh recorded no time, so the provenance it writes would be undated")
	}

	// The two requests went where they were supposed to: the shell to x.com (by Host, which is
	// what the TLS session was for), the bundle to the CDN. The cookie travels with the first
	// and not the second — a .x.com cookie is not sent to abs.twimg.com by a browser either.
	mu.Lock()
	defer mu.Unlock()
	if requests != 2 {
		t.Fatalf("the refresh made %d requests, want 2 (the shell and the bundle)", requests)
	}
	if hosts[1] != "abs.twimg.com" {
		t.Errorf("the bundle request went to %q, want abs.twimg.com", hosts[1])
	}
	if !strings.Contains(cookies[0], "auth_token="+fixtureToken) || !strings.Contains(cookies[0], "ct0="+fixtureCTO) {
		t.Error("the shell request carried no session cookie, which is the only credential that request class accepts")
	}
	if cookies[1] != "" {
		t.Errorf("the bundle request carried a cookie (%q), and the CDN is not x.com", cookies[1])
	}
	for i, auth := range auths {
		if auth != "" {
			t.Errorf("request %d carried an authorization header (%q); a navigation request with a bearer is answered 401 with a zero-byte body", i, auth)
		}
	}
	if _, _, _ = client.Limiter.Budget(); false {
		t.Fatal("unreachable")
	}
	if used, _, _ := client.Limiter.Budget(); used != 2 {
		t.Errorf("the limiter counted %d requests, want the 2 the refresh spent", used)
	}
}

func TestFetchRefusesWithoutASession(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("a refresh with no session still asked %s for %s", r.Host, r.URL.Path)
	}))
	defer srv.Close()

	client := testClient(t, srv)
	_, err := Fetch(context.Background(), client, session.Session{}, ShellURL)
	if err == nil {
		t.Fatal("a refresh with no session was attempted")
	}
	if !strings.Contains(err.Error(), "session") {
		t.Errorf("the error does not say a session is what is missing: %v", err)
	}
	if used, _, _ := client.Limiter.Budget(); used != 0 {
		t.Errorf("the refused refresh spent %d requests", used)
	}
}

func TestFetchNamesAnExpiredSession(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Measured 2026-10-14: a dead cookie answers /home with a 307 into the login flow and
		// a zero-byte body.
		w.Header().Set("location", "/i/jf/onboarding/web?redirect_after_login=%2Fhome&mode=login")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer srv.Close()

	client := testClient(t, srv)
	_, err := Fetch(context.Background(), client, fixtureSession(), ShellURL)
	if err == nil {
		t.Fatal("an expired session produced a refresh")
	}
	if !strings.Contains(err.Error(), "auth rejected") {
		t.Errorf("the error does not name the outcome the classifier gave: %v", err)
	}
	if used, _, _ := client.Limiter.Budget(); used != 1 {
		t.Errorf("the refresh spent %d requests on a session the upstream rejected, want 1", used)
	}
}

func TestGetRefusesABodyOverTheLimit(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("x", 100)))
	}))
	defer srv.Close()

	client := testClient(t, srv)
	if _, err := get(context.Background(), client, session.Session{}, srv.URL, 16); err == nil {
		t.Fatal("a body over the limit was read in full")
	} else if !strings.Contains(err.Error(), "16 bytes") {
		t.Errorf("the error does not say what the limit was: %v", err)
	}
}

func names(records []Record) []string {
	out := make([]string, 0, len(records))
	for _, record := range records {
		out = append(out, record.Name)
	}
	return out
}
