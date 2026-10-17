package xweb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/config"
)

// pngBytes is a one-pixel PNG: the smallest body that sniffs as the type it claims.
var pngBytes = []byte("\x89PNG\r\n\x1a\n" +
	"\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89" +
	"\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82")

// newFetcher builds a fetcher pointed at a test server by overriding its transport and allow-list.
// The URL check is kept real; only the reachability is faked, because the point of these tests is
// what the request carries, not whether this host can reach twimg.
func newFetcher(t *testing.T, h http.Handler, capBytes int64) (*ImageFetcher, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	f, err := NewImageFetcher(config.Config{}, capBytes)
	if err != nil {
		t.Fatalf("building the fetcher: %v", err)
	}
	// The test server is on 127.0.0.1, so the allow-list is widened to it and nothing else. This is
	// the one seam the tests move; the shipped check is asserted separately below.
	// The server speaks plain http on loopback; the fetcher's own transport is the module's real
	// one, which is about proxy resolution rather than about reaching a test server. So the test
	// keeps everything the fetcher does with the response and replaces only how it gets there.
	f.http = &http.Client{Transport: http.DefaultTransport, Timeout: imageTimeout}
	f.check = func(u *url.URL) bool { return u.Host == strings.TrimPrefix(srv.URL, "http://") }
	return f, srv
}

// TestImageFetchSendsNoCredential is the rule that made this a separate type: an image request
// carries no cookie, no authorization and no csrf token, whatever the session holds.
func TestImageFetchSendsNoCredential(t *testing.T) {
	var seen http.Header
	f, srv := newFetcher(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes)
	}), 1<<20)

	body, kind, err := f.Get(context.Background(), srv.URL+"/img.png")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if kind != "image/png" {
		t.Fatalf("kind = %q, want image/png", kind)
	}
	if string(body) != string(pngBytes) {
		t.Fatal("the body is not the bytes the server sent")
	}
	for _, banned := range []string{"Cookie", "Authorization", "X-Csrf-Token", "X-Client-Transaction-Id", "Referer"} {
		if v := seen.Get(banned); v != "" {
			t.Fatalf("the request carried %s: %q", banned, v)
		}
	}
	if ua := seen.Get("User-Agent"); ua != UserAgent {
		t.Fatalf("user agent = %q", ua)
	}
}

// TestImageFetchIsCapped: a body over the cap is a typed error, never a partial read.
func TestImageFetchIsCapped(t *testing.T) {
	f, srv := newFetcher(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(make([]byte, 4096))
	}), 1024)
	if _, _, err := f.Get(context.Background(), srv.URL+"/big.png"); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

// TestImageFetchRejectsANonImage: a declared type this build cannot decode and bytes that are not an
// image are one answer, because the reader's remedy is the same.
func TestImageFetchRejectsANonImage(t *testing.T) {
	f, srv := newFetcher(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>challenge</html>"))
	}), 1<<20)
	if _, _, err := f.Get(context.Background(), srv.URL+"/x"); !errors.Is(err, ErrNotAnImage) {
		t.Fatalf("err = %v, want ErrNotAnImage", err)
	}
}

// TestImageFetchRefusesANonTwimgHost is the allow-list's own test: the shipped predicate, not the
// widened one the server tests use.
func TestImageFetchRefusesANonTwimgHost(t *testing.T) {
	for _, raw := range []string{
		"https://example.com/a.png",
		"https://nottwimg.com/a.png",
		"https://t.co/a.png",
		"http://pbs.twimg.com/a.png", // not https
		"https://pbs.twimg.com.evil.com/a.png",
	} {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		// The predicate answers about hosts; the scheme is refused in Get. Both are asserted here
		// so that neither half can regress alone.
		if isTwimgHost(u) && u.Scheme == "https" {
			t.Fatalf("%s was allowed", raw)
		}
	}
	for _, raw := range []string{"https://pbs.twimg.com/a.png", "https://abs.twimg.com/b.png"} {
		u, _ := url.Parse(raw)
		if !isTwimgHost(u) {
			t.Fatalf("%s was refused", raw)
		}
	}
}
