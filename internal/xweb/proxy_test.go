package xweb

import (
	"net/http"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/config"
)

// These tests cover the transport's proxy resolution. They live here rather than in
// limiter_test.go because the plan's file list did not name a proxy_test.go and the code
// they cover — the scheme handling and the NO_PROXY matching — is in proxy.go; what
// matters is that it is covered at all, since a proxy setting that is accepted but
// misapplied reports as an empty timeline rather than as an error.

func TestNewTransportAcceptsBothMeasuredProxySchemes(t *testing.T) {
	// Measured on this host, 2026-10-14: Clash Verge Rev listens on 127.0.0.1:7897 as a
	// mixed port, and both forms return 200 for https://x.com/.
	for _, setting := range []string{
		"http://127.0.0.1:7897",
		"https://127.0.0.1:7897",
		"socks5://127.0.0.1:7897",
		"socks5h://127.0.0.1:7897",
	} {
		t.Run(setting, func(t *testing.T) {
			transport, err := NewTransport(config.Config{Proxy: setting})
			if err != nil {
				t.Fatalf("NewTransport(%q): %v", setting, err)
			}
			req, err := http.NewRequest(http.MethodGet, BaseURL+"/", nil)
			if err != nil {
				t.Fatalf("building the request: %v", err)
			}
			got, err := transport.Proxy(req)
			if err != nil {
				t.Fatalf("the transport's proxy hook failed: %v", err)
			}
			if got == nil || got.String() != setting {
				t.Fatalf("the transport would use %v for https://x.com/, want %s", got, setting)
			}
			if !transport.ForceAttemptHTTP2 {
				t.Error("ForceAttemptHTTP2 is off: the measured request shape is HTTP/2")
			}
		})
	}
}

func TestNewTransportRefusesAProxyItCannotUseWithoutEchoingIt(t *testing.T) {
	for _, setting := range []string{
		"ftp://user:secret@proxy.example:21", // a scheme net/http does not speak
		"127.0.0.1:7897",                     // no scheme: parsed as an opaque URL
		"http://",                            // no host
		"http://user:secret@",                // a userinfo and no host
	} {
		t.Run(setting, func(t *testing.T) {
			transport, err := NewTransport(config.Config{Proxy: setting})
			if err == nil {
				t.Fatalf("NewTransport accepted %q and built %v", setting, transport)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Errorf("the refusal quotes the proxy credential: %v", err)
			}
			if transport != nil {
				t.Error("a refused setting still produced a transport")
			}
		})
	}
}

func TestNewTransportCarriesTheTimeouts(t *testing.T) {
	transport, err := NewTransport(config.Config{})
	if err != nil {
		t.Fatalf("NewTransport with no proxy at all: %v", err)
	}
	if transport.TLSHandshakeTimeout != TLSHandshakeTimeout {
		t.Errorf("TLSHandshakeTimeout = %s, want %s", transport.TLSHandshakeTimeout, TLSHandshakeTimeout)
	}
	if transport.DialContext == nil {
		t.Error("DialContext is nil: the dial timeout cannot be applied to anything")
	}
	if transport.IdleConnTimeout <= 0 {
		t.Errorf("IdleConnTimeout = %s: a zero here keeps spare connections forever", transport.IdleConnTimeout)
	}
}

func TestNoProxyExemptsTheHostsItNames(t *testing.T) {
	tests := []struct {
		name    string
		noProxy string
		url     string
		exempt  bool // true: the list names it, so the request dials directly
	}{
		{name: "a wildcard exempts everything", noProxy: "*", url: BaseURL + "/", exempt: true},
		{name: "the host itself", noProxy: "x.com", url: BaseURL + "/", exempt: true},
		{name: "a name under it", noProxy: "x.com", url: APIBaseURL, exempt: true},
		{name: "a leading dot is the same rule", noProxy: ".x.com", url: BaseURL + "/home", exempt: true},
		{name: "another host is not exempt", noProxy: "example.com", url: BaseURL + "/", exempt: false},
		{name: "a name that merely ends the same is not exempt", noProxy: "example.com", url: "https://notexample.com/", exempt: false},
		{name: "the port has to match when it is given", noProxy: "x.com:443", url: BaseURL + "/", exempt: true},
		{name: "a different port is not exempt", noProxy: "x.com:8080", url: BaseURL + "/", exempt: false},
		{name: "an address literal", noProxy: "162.159.140.229", url: "https://162.159.140.229/", exempt: true},
		{name: "a CIDR block", noProxy: "162.159.140.0/24", url: "https://162.159.140.229/", exempt: true},
		{name: "entries are a list", noProxy: "example.com, x.com", url: BaseURL + "/", exempt: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := NewTransport(config.Config{Proxy: "http://127.0.0.1:7897", NoProxy: tt.noProxy})
			if err != nil {
				t.Fatalf("NewTransport: %v", err)
			}
			req, err := http.NewRequest(http.MethodGet, tt.url, nil)
			if err != nil {
				t.Fatalf("building the request: %v", err)
			}
			got, err := transport.Proxy(req)
			if err != nil {
				t.Fatalf("the transport's proxy hook failed: %v", err)
			}
			if tt.exempt && got != nil {
				t.Fatalf("NO_PROXY=%q did not exempt %s: it would go through %v", tt.noProxy, tt.url, got)
			}
			if !tt.exempt && got == nil {
				t.Fatalf("NO_PROXY=%q exempted %s, but the list does not name it", tt.noProxy, tt.url)
			}
		})
	}
}
