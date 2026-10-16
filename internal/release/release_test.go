package release

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// newFake is the fake GitHub API every test in this file drives: an httptest server on loopback,
// so no test opens a socket that leaves the machine and a plain `go test ./...` makes no outbound
// request. The client's two origins are pointed at it through this package's unexported fields,
// which is the reason those fields are unexported: the production type grows no way to be lied to.
func newFake(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// newTestClient is a Client whose origins are the fake's. Only this package can build one, so only
// this package's tests can send the real client somewhere other than GitHub.
func newTestClient(srv *httptest.Server, token session.Secret) *Client {
	return &Client{
		http:         &http.Client{Timeout: requestTimeout},
		token:        token,
		apiBase:      srv.URL,
		downloadBase: srv.URL,
		repo:         repository,
	}
}

// The invariant that made the first live upgrade a sixteen-attempt affair, pinned so it cannot come
// back: the client's own timeout is what cuts a request off, so it has to leave room for the
// handshake the transport is allowed — otherwise the handshake budget is decoration, and the
// request dies at the client's bound first. Both budgets also have to be larger than xweb's, which
// were measured against x.com through this same proxy and are the reason this package asks for
// another pair at all.
func TestTheReleaseWireOutlivesItsOwnHandshakeBudget(t *testing.T) {
	client, err := New(config.Config{}, session.Secret(""))
	if err != nil {
		t.Fatalf("New with no proxy at all: %v", err)
	}
	if client.http == nil || client.http.Transport == nil {
		t.Fatal("the client has no transport: nothing about its budgets can be read")
	}
	transport, ok := client.http.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("the transport is %T, want *http.Transport", client.http.Transport)
	}
	if transport.TLSHandshakeTimeout != githubHandshakeTimeout {
		t.Errorf("the transport's handshake budget is %s, want this package's %s: it is carrying xweb's default for the wrong host",
			transport.TLSHandshakeTimeout, githubHandshakeTimeout)
	}
	if client.http.Timeout <= githubHandshakeTimeout {
		t.Errorf("the client's timeout is %s and the handshake budget is %s: the client cuts the request off before the handshake may finish, so the budget decides nothing",
			client.http.Timeout, githubHandshakeTimeout)
	}
	if githubHandshakeTimeout <= xweb.TLSHandshakeTimeout || githubDialTimeout <= xweb.DialTimeout {
		t.Errorf("this package's budgets (%s dial, %s handshake) are not larger than xweb's (%s, %s): the second pair exists for a slower network than the first, and this test is what says so",
			githubDialTimeout, githubHandshakeTimeout, xweb.DialTimeout, xweb.TLSHandshakeTimeout)
	}
}

// releaseBody is a Release object shaped the way api.github.com shapes one, cut down to the fields
// this package decodes.
func releaseBody(tag string, assets ...Asset) string {
	var b strings.Builder
	fmt.Fprintf(&b, `{"tag_name":%q,"assets":[`, tag)
	for i, asset := range assets {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"name":%q,"size":%d,"browser_download_url":%q}`, asset.Name, asset.Size, asset.URL)
	}
	b.WriteString("]}")
	return b.String()
}

// v110Assets is the v1.1.0 Release's asset list as it was published on 2026-10-16 and recorded in
// the plan: the name form is the release workflow's own, and the digest is a per-asset file beside
// the artifact it verifies.
func v110Assets(base string) []Asset {
	names := []string{
		"SHA256SUMS",
		"x-terminal_v1.1.0_linux_amd64", "x-terminal_v1.1.0_linux_amd64.sha256",
		"x-terminal_v1.1.0_linux_arm64", "x-terminal_v1.1.0_linux_arm64.sha256",
		"x-terminal_v1.1.0_darwin_arm64", "x-terminal_v1.1.0_darwin_arm64.sha256",
	}
	assets := make([]Asset, 0, len(names))
	for _, name := range names {
		assets = append(assets, Asset{Name: name, Size: 1, URL: base + "/releases/download/v1.1.0/" + name})
	}
	return assets
}

// TestLatestBuildsTheDocumentedRequest is the wire: one GET of the documented path, the Accept and
// User-Agent headers this API requires, and an Authorization header only when a token was exported.
// The anonymous arm is asserted on the header's absence rather than on an empty value, because
// "Authorization: " and no Authorization at all are different requests.
func TestLatestBuildsTheDocumentedRequest(t *testing.T) {
	for _, tc := range []struct {
		name     string
		token    session.Secret
		wantAuth string // "" means the header must be absent
	}{
		{name: "anonymous", token: "", wantAuth: ""},
		{name: "token", token: "ghp_fixture_token", wantAuth: "Bearer ghp_fixture_token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var (
				method string
				path   string
				header http.Header
			)
			srv := newFake(t, func(w http.ResponseWriter, r *http.Request) {
				method, path, header = r.Method, r.URL.Path, r.Header.Clone()
				w.Header().Set("content-type", "application/json")
				fmt.Fprint(w, releaseBody("v1.1.0", v110Assets("https://github.com/LZD121/x-terminal")...))
			})

			rel, err := newTestClient(srv, tc.token).Latest(context.Background())
			if err != nil {
				t.Fatalf("Latest: %v", err)
			}
			if method != http.MethodGet {
				t.Errorf("Latest built a %s request, want GET", method)
			}
			if want := "/repos/" + repository + "/releases/latest"; path != want {
				t.Errorf("Latest asked for %q, want %q", path, want)
			}
			if got := header.Get("Accept"); got != "application/vnd.github+json" {
				t.Errorf("Accept is %q, want application/vnd.github+json", got)
			}
			if got := header.Get("User-Agent"); got != UserAgent() {
				t.Errorf("User-Agent is %q, want the build's own identity %q", got, UserAgent())
			}
			if got := header.Get("Authorization"); got != tc.wantAuth {
				t.Errorf("Authorization is %q, want %q", got, tc.wantAuth)
			}
			if rel.Tag != "v1.1.0" {
				t.Errorf("Latest decoded tag %q, want v1.1.0", rel.Tag)
			}
			if len(rel.Assets) != 7 {
				t.Fatalf("Latest decoded %d assets, want the seven the release carries", len(rel.Assets))
			}
			if rel.Assets[0].Name != "SHA256SUMS" || rel.Assets[0].Size != 1 || !strings.HasSuffix(rel.Assets[0].URL, "/SHA256SUMS") {
				t.Errorf("Latest decoded the first asset as %+v", rel.Assets[0])
			}
		})
	}
}

// TestIdentityIsTheBuildsOwnOrHonest: the User-Agent says which build is asking, and an unstamped
// build says that rather than saying nothing. SetIdentity is handed the empty string by a working
// tree build's Version(), and that must not turn the header into "x-terminal/".
func TestIdentityIsTheBuildsOwnOrHonest(t *testing.T) {
	saved := buildIdentity
	t.Cleanup(func() { buildIdentity = saved })

	buildIdentity = "working tree"
	if got := UserAgent(); got != "x-terminal/working tree" {
		t.Errorf("an unstamped build announces %q", got)
	}
	SetIdentity("v1.2.0")
	if got := UserAgent(); got != "x-terminal/v1.2.0" {
		t.Errorf("a stamped build announces %q", got)
	}
	SetIdentity("")
	if got := UserAgent(); got != "x-terminal/v1.2.0" {
		t.Errorf("the empty identity replaced a real one: %q", got)
	}
}

// TestNewBuildsTheTransportFromTheConfiguration: New is the only production constructor, and the
// one error it returns is an unusable configuration — a proxy URL the transport cannot parse or a
// scheme it does not speak.
func TestNewBuildsTheTransportFromTheConfiguration(t *testing.T) {
	client, err := New(config.Config{Proxy: "http://127.0.0.1:7897"}, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.http == nil || client.http.Transport == nil {
		t.Fatal("New built a client with no transport: nothing would route through the configured proxy")
	}
	if client.apiBase != apiBase || client.downloadBase != "https://github.com" || client.repo != repository {
		t.Fatalf("New built %+v", client)
	}
	if client.token != "" {
		t.Fatal("New carried a token that was never given")
	}
	// The redirect policy is net/http's own: an asset URL answers 302 to a signed object URL, and
	// a client that refused to follow it could never download an asset.
	if client.http.CheckRedirect != nil {
		t.Fatal("New installed a redirect policy: an asset URL answers 302 and must be followed")
	}

	if _, err := New(config.Config{Proxy: "ftp://proxy.example:1080"}, ""); err == nil {
		t.Fatal("New accepted a proxy scheme the transport does not speak")
	}
}

// TestLatestRefusesABodyThatIsNotJSON: a decode failure is reported as a decode failure and never
// as "no assets" — an empty Release would read as "there is nothing newer".
func TestLatestRefusesABodyThatIsNotJSON(t *testing.T) {
	srv := newFake(t, func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "<html>not a release object</html>")
	})
	_, err := newTestClient(srv, "").Latest(context.Background())
	if err == nil {
		t.Fatal("a non-JSON body decoded")
	}
	if !strings.Contains(err.Error(), "not a release object") {
		t.Errorf("the decode failure is reported as %v", err)
	}
}

// TestLatestReportsTheStatusesGitHubSends: 404 (a repository the caller cannot see, or a release
// that does not exist) and 500 (GitHub's own fault) are both *HTTPError, carrying the status, the
// URL and the API's own message, so a caller can say "404" instead of "failed".
func TestLatestReportsTheStatusesGitHubSends(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv := newFake(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				fmt.Fprint(w, `{"message":"Not Found"}`)
			})
			_, err := newTestClient(srv, "").Latest(context.Background())
			var httpErr *HTTPError
			if !errors.As(err, &httpErr) {
				t.Fatalf("a %d is %v, want an *HTTPError", status, err)
			}
			if httpErr.Status != status {
				t.Errorf("HTTPError.Status is %d, want %d", httpErr.Status, status)
			}
			if !strings.Contains(httpErr.URL, "/releases/latest") {
				t.Errorf("HTTPError.URL is %q, want the URL that answered", httpErr.URL)
			}
			if !strings.Contains(httpErr.Body, "Not Found") {
				t.Errorf("HTTPError.Body is %q, want the API's own message", httpErr.Body)
			}
		})
	}
}

// TestLatestRefusesAnOverLongBody: the bound is a refusal, not a truncation. A body cut at the
// bound would decode into a Release object that is missing its assets, which is the one wrong
// answer that looks like a right one.
func TestLatestRefusesAnOverLongBody(t *testing.T) {
	srv := newFake(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write(bytes.Repeat([]byte("x"), metaLimit+1))
	})
	_, err := newTestClient(srv, "").Latest(context.Background())
	if err == nil {
		t.Fatal("an over-long body was read instead of refused")
	}
	if !strings.Contains(err.Error(), "more than") {
		t.Errorf("the bound's refusal says %v", err)
	}
}

// TestCompareTagsOrdersNumerically is the arm that says why this is not a string compare: v1.10.0
// is newer than v1.9.0, and a string compare says the opposite. The result is negative when the
// first tag is the older one — the convention cmp.Compare uses, which is what lets the verb read
// "the release is newer" as `CompareTags(running, rel.Tag) < 0`.
func TestCompareTagsOrdersNumerically(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"v1.2.0", "v1.1.0", 1},
		{"v1.10.0", "v1.9.0", 1},
		{"v2.0.0", "v1.9.9", 1},
		{"v1.1.0", "v1.1.0", 0},
		{"v1.1.0", "v1.2.0", -1},
		{"v1.1.1", "v1.1.0", 1},
		{"v1.9.9", "v1.10.0", -1},
	} {
		got, err := CompareTags(tc.a, tc.b)
		if err != nil {
			t.Fatalf("CompareTags(%q, %q): %v", tc.a, tc.b, err)
		}
		if got != tc.want {
			t.Errorf("CompareTags(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

// TestParseTagRefusesWhatIsNotATag: a looser pattern would have to decide what a release candidate
// or a fourth field means relative to a release, and a decision this package cannot justify is
// worse than a refusal that names the value.
func TestParseTagRefusesWhatIsNotATag(t *testing.T) {
	for _, tag := range []string{"1.1.0", "v1.1", "v1.1.0-rc1", "v1.1.0.1", "", "v1.1.0 "} {
		if _, err := ParseTag(tag); err == nil {
			t.Errorf("ParseTag(%q) accepted a value that is not a release tag", tag)
		} else if !strings.Contains(err.Error(), fmt.Sprintf("%q", tag)) {
			t.Errorf("ParseTag(%q)'s refusal does not name the value: %v", tag, err)
		}
	}
	for _, tag := range []string{"v1.0.0", "v1.1.0", "v10.20.30"} {
		if _, err := ParseTag(tag); err != nil {
			t.Errorf("ParseTag(%q): %v", tag, err)
		}
	}
	if got, err := ParseTag("v1.10.20"); err != nil {
		t.Errorf("ParseTag: %v", err)
	} else if (Version{Major: 1, Minor: 10, Patch: 20}) != got {
		t.Errorf("ParseTag read v1.10.20 as %+v", got)
	}
	// Both sides of a comparison are read: a comparison against a tag nobody can parse is refused
	// with the value in the message rather than ordered by a guess.
	if _, err := CompareTags("v1.1.0", "nonsense"); err == nil || !strings.Contains(err.Error(), "nonsense") {
		t.Errorf("CompareTags against an unreadable tag says %v", err)
	}
}

// TestAssetForPicksTheReleasesOwnNames: the three published platforms are picked by the exact
// names the Release carries, and a platform the release does not build for is refused with both
// the name that was wanted and the names that exist — the difference between a state the verb can
// print and a 404 on a download URL it guessed.
func TestAssetForPicksTheReleasesOwnNames(t *testing.T) {
	assets := v110Assets("https://github.com/LZD121/x-terminal")
	for _, tc := range []struct{ goos, goarch string }{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "arm64"},
	} {
		asset, err := AssetFor(assets, "v1.1.0", tc.goos, tc.goarch)
		if err != nil {
			t.Fatalf("AssetFor(%s/%s): %v", tc.goos, tc.goarch, err)
		}
		if want := AssetName("v1.1.0", tc.goos, tc.goarch); asset.Name != want {
			t.Errorf("AssetFor(%s/%s) picked %q, want %q", tc.goos, tc.goarch, asset.Name, want)
		}
		if asset.Name == "SHA256SUMS" || strings.HasSuffix(asset.Name, ".sha256") {
			t.Errorf("AssetFor(%s/%s) picked a digest file: %q", tc.goos, tc.goarch, asset.Name)
		}
		if !strings.Contains(asset.URL, "/releases/download/v1.1.0/") {
			t.Errorf("AssetFor(%s/%s) picked %q, whose URL is %q", tc.goos, tc.goarch, asset.Name, asset.URL)
		}
	}

	for _, tc := range []struct{ goos, goarch string }{
		{"darwin", "amd64"},
		{"windows", "amd64"},
	} {
		_, err := AssetFor(assets, "v1.1.0", tc.goos, tc.goarch)
		if err == nil {
			t.Fatalf("AssetFor(%s/%s) picked an asset the release does not carry", tc.goos, tc.goarch)
		}
		if want := AssetName("v1.1.0", tc.goos, tc.goarch); !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
		if !strings.Contains(err.Error(), "x-terminal_v1.1.0_linux_amd64") {
			t.Errorf("the refusal does not name the assets the release carries: %v", err)
		}
	}

	// The anti-vacuity arm: a list nobody read cannot answer a picker, so an empty list is a
	// refusal naming itself rather than a pass or a blank.
	_, err := AssetFor(nil, "v1.1.0", "linux", "amd64")
	if err == nil {
		t.Fatal("an empty asset list answered a picker")
	}
	if !strings.Contains(err.Error(), "no assets at all") {
		t.Errorf("the empty-list refusal says %v", err)
	}
}

// TestAssetNamedRefusesANameTheReleaseDoesNotCarry: the digest's own asset is looked up by the name
// the release carries, so a missing one is a state and not a 404 on an assembled URL.
func TestAssetNamedRefusesANameTheReleaseDoesNotCarry(t *testing.T) {
	assets := v110Assets("https://github.com/LZD121/x-terminal")
	want := "x-terminal_v1.1.0_darwin_amd64.sha256"
	if _, err := AssetNamed(assets, want); err == nil {
		t.Fatalf("the v1.1.0 release does not carry %s", want)
	} else if !strings.Contains(err.Error(), want) {
		t.Errorf("the refusal does not name the asset that was wanted: %v", err)
	}
	found, err := AssetNamed(assets, "x-terminal_v1.1.0_linux_amd64.sha256")
	if err != nil {
		t.Fatalf("AssetNamed: %v", err)
	}
	if !strings.HasSuffix(found.URL, "/x-terminal_v1.1.0_linux_amd64.sha256") {
		t.Errorf("AssetNamed returned %+v", found)
	}
}

// TestTheTokenNeverReachesAString is the structural half of the leak promise: the token is a
// session.Secret, so every formatting path a caller reaches for by habit renders the placeholder
// rather than the credential. It is what makes "no line of this verb carries the token" a property
// of the type instead of a promise about care.
func TestTheTokenNeverReachesAString(t *testing.T) {
	const secret = "ghp_a_token_that_must_never_be_printed"
	client := &Client{
		http:         &http.Client{},
		token:        session.Secret(secret),
		apiBase:      apiBase,
		downloadBase: "https://github.com",
		repo:         repository,
	}
	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%d"} {
		got := fmt.Sprintf(verb, client)
		if strings.Contains(got, secret) {
			t.Errorf("%s printed the token: %q", verb, got)
		}
		if !strings.Contains(got, session.Redacted) {
			t.Errorf("%s did not render %s: %q", verb, session.Redacted, got)
		}
	}
}

// TestFetchReadsOneURLWithTheSameWire: an asset and its digest are read through the
// same one place the release metadata is, with the same headers, one request per URL,
// and no decode — a payload is bytes.
func TestFetchReadsOneURLWithTheSameWire(t *testing.T) {
	const path = "/LZD121/x-terminal/releases/download/v1.1.0/x-terminal_v1.1.0_linux_amd64"
	var (
		method string
		header http.Header
	)
	srv := newFake(t, func(w http.ResponseWriter, r *http.Request) {
		method, header = r.Method, r.Header.Clone()
		_, _ = w.Write([]byte("the asset's bytes"))
	})
	client := newTestClient(srv, "")

	body, err := client.Fetch(context.Background(), srv.URL+path, 1<<20)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(body) != "the asset's bytes" {
		t.Errorf("Fetch returned %q", body)
	}
	if method != http.MethodGet {
		t.Errorf("Fetch built a %s request, want GET", method)
	}
	if got := header.Get("Accept"); got != "application/vnd.github+json" {
		t.Errorf("Fetch sent Accept %q", got)
	}
	if got := header.Get("User-Agent"); got != UserAgent() {
		t.Errorf("Fetch sent User-Agent %q, want %q", got, UserAgent())
	}
}

// TestFetchRefusesAURLOutsideTheReleaseOrigin: the URL arrives in the Release's own
// asset list, but it is still a string a remote document supplied, and this is the one
// place that decides what this process will download. The origin is the only thing that
// makes the digest beside it worth anything.
func TestFetchRefusesAURLOutsideTheReleaseOrigin(t *testing.T) {
	client := &Client{
		http:         &http.Client{},
		downloadBase: "https://github.com",
		repo:         repository,
	}
	if _, err := client.Fetch(context.Background(), "https://example.com/x-terminal", 1<<20); err == nil {
		t.Fatal("Fetch would have downloaded from a host the release does not name")
	} else if !strings.Contains(err.Error(), "https://github.com") {
		t.Errorf("the refusal does not name the origin it required: %v", err)
	}
}

// TestFetchReportsTheStatusesAndTheBound: a non-2xx is an *HTTPError carrying the URL,
// and a body over the bound is a refusal rather than a truncated asset — a partial
// payload that passed the digest is not a thing, but a payload silently cut before the
// digest is a misleading failure to diagnose.
func TestFetchReportsTheStatusesAndTheBound(t *testing.T) {
	t.Run("404", func(t *testing.T) {
		srv := newFake(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Not Found")
		})
		_, err := newTestClient(srv, "").Fetch(context.Background(), srv.URL+"/missing", 1<<10)
		var httpErr *HTTPError
		if !errors.As(err, &httpErr) || httpErr.Status != http.StatusNotFound {
			t.Fatalf("Fetch reported %v, want an *HTTPError with status 404", err)
		}
		if !strings.Contains(httpErr.URL, "/missing") {
			t.Errorf("HTTPError.URL is %q", httpErr.URL)
		}
	})

	t.Run("over the bound", func(t *testing.T) {
		srv := newFake(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(bytes.Repeat([]byte("x"), 1025))
		})
		_, err := newTestClient(srv, "").Fetch(context.Background(), srv.URL+"/asset", 1024)
		if err == nil {
			t.Fatal("an over-long asset body was read instead of refused")
		}
		if !strings.Contains(err.Error(), "more than the 1024 bytes") {
			t.Errorf("the bound's refusal says %v", err)
		}
	})
}
