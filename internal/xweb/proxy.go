package xweb

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/LZD121/x-terminal/internal/config"
)

const (
	// DialTimeout bounds one TCP connect. A connect that has not answered inside ten
	// seconds is the network refusing, which is a state to name rather than to wait on.
	DialTimeout = 10 * time.Second
	// TLSHandshakeTimeout bounds the handshake that follows the connect.
	TLSHandshakeTimeout = 10 * time.Second
	// idleConnTimeout is how long a spare connection is kept. The zero value means
	// "forever", which is not what a program that talks to one host wants.
	idleConnTimeout = 90 * time.Second
)

// TransportBudget is the pair of client-side budgets a transport is built with: how long a
// connect may take, and how long the handshake after it may.
//
// It exists because this program's two request classes are answered by networks with very
// different latencies, and one budget cannot be right for both. The constants above were set
// against x.com through the operator's proxy, where a handshake measured well under a second
// (doctor's own "upstream reachable 681ms" line, 2026-10-15) and where a connect that has not
// answered in ten seconds really is the network refusing — which is why they stay the default.
// GitHub through the same proxy is a different animal, and internal/release says so with the
// numbers it measured.
type TransportBudget struct {
	// Dial bounds one TCP connect.
	Dial time.Duration
	// Handshake bounds the TLS handshake that follows the connect.
	Handshake time.Duration
}

// DefaultTransportBudget is the budget every request in this program gets unless its caller
// deliberately asks for another: the package constants above, which is what every staged
// connectivity check, every read in the pane and the update verb's own prefetch share.
func DefaultTransportBudget() TransportBudget {
	return TransportBudget{Dial: DialTimeout, Handshake: TLSHandshakeTimeout}
}

// NewTransport builds the one http.Transport every request in this program goes
// through. It is built from the resolved configuration rather than from the
// environment, because on a host where x.com is only reachable through a local proxy
// there is nothing to inherit: the proxy has to be named, or the client silently
// reports an empty timeline.
//
// The proxy's scheme decides the protocol. net/http's transport understands "http",
// "https", "socks5" and "socks5h" proxy URLs (net/http/transport.go), including
// resolving the name at the SOCKS5 end for the h form, so the setting is parsed,
// validated and handed over rather than turned into a dialer of this package's own.
func NewTransport(cfg config.Config) (*http.Transport, error) {
	return NewTransportWithBudget(cfg, DefaultTransportBudget())
}

// NewTransportWithBudget is NewTransport with the budgets chosen by the caller, and it is the
// same transport otherwise — the same resolved proxy, the same idle policy, the same
// ForceAttemptHTTP2. A caller that passes another budget is stating a fact about the network it
// talks to, not a preference: x.com and a GitHub Release are reached through the same proxy and
// answer handshakes seconds apart, so the one number cannot serve both.
//
// The zero value of either field is refused rather than silently meaning "no limit": a budget
// nobody set is not a bigger budget, and a transport built with no handshake bound is one that
// can hang, which is the state this package exists to make nameable.
func NewTransportWithBudget(cfg config.Config, budget TransportBudget) (*http.Transport, error) {
	if budget.Dial <= 0 || budget.Handshake <= 0 {
		return nil, fmt.Errorf("xweb: a transport budget needs a positive dial and handshake bound, got dial=%s handshake=%s", budget.Dial, budget.Handshake)
	}
	proxy, err := proxyFunc(cfg)
	if err != nil {
		return nil, err
	}
	return &http.Transport{
		Proxy:               proxy,
		DialContext:         (&net.Dialer{Timeout: budget.Dial, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout: budget.Handshake,
		IdleConnTimeout:     idleConnTimeout,
		// ForceAttemptHTTP2 keeps the measured request shape: a custom Proxy or
		// DialContext otherwise turns HTTP/2 off, and every 200 in the probe was
		// served over HTTP/2 by an edge that sees the protocol version.
		ForceAttemptHTTP2: true,
	}, nil
}

// ProxyAddress returns the host:port of a proxy setting, or "" for a direct dial. It
// is what the connectivity self-check dials to tell "the proxy is not running" from
// "the proxy is running and the upstream is not answering".
func ProxyAddress(setting string) (string, error) {
	proxy, err := proxySetting(setting)
	if err != nil || proxy == nil {
		return "", err
	}
	return proxy.Host, nil
}

// proxySetting parses and validates a proxy setting. No error it returns ever quotes
// the setting: a proxy URL may carry a password (http://user:secret@host:port), and a
// refusal is one of the places a credential would otherwise end up in a terminal and
// in whatever captures it. config.Config.Redacted is the form to print.
func proxySetting(setting string) (*url.URL, error) {
	if setting == "" {
		return nil, nil
	}
	proxy, err := url.Parse(setting)
	if err != nil {
		return nil, errors.New("xweb: the proxy setting is not a URL: expected scheme://host:port with the http, https, socks5 or socks5h scheme")
	}
	switch proxy.Scheme {
	case "http", "https", "socks5", "socks5h":
	default:
		return nil, errors.New("xweb: the proxy setting's scheme is not supported: expected scheme://host:port with the http, https, socks5 or socks5h scheme")
	}
	if proxy.Host == "" {
		return nil, errors.New("xweb: the proxy setting names no host:port: expected scheme://host:port")
	}
	return proxy, nil
}

// proxyFunc is the transport's Proxy hook: the configured proxy for every request,
// except the hosts NO_PROXY exempts.
func proxyFunc(cfg config.Config) (func(*http.Request) (*url.URL, error), error) {
	proxy, err := proxySetting(cfg.Proxy)
	if err != nil {
		return nil, err
	}
	if proxy == nil {
		// Nothing is configured, so the environment's own resolution is the fallback:
		// a caller that built a transport without config.Load still gets the proxy its
		// shell exports. config.Load resolves those variables itself and hands the
		// result in as cfg.Proxy, which is why this path is the fallback and not the rule.
		return http.ProxyFromEnvironment, nil
	}
	return func(req *http.Request) (*url.URL, error) {
		if req == nil || req.URL == nil {
			return proxy, nil
		}
		host, port := hostPort(req.URL)
		if noProxyMatch(cfg.NoProxy, host, port) {
			return nil, nil
		}
		return proxy, nil
	}, nil
}

// noProxyMatch reports whether the request's host is on the NO_PROXY list, so that a
// configured proxy is bypassed for it. The list comes from the environment as it was
// written, which is why the matching happens here: net/http's own NO_PROXY handling
// only runs on the ProxyFromEnvironment path, and a proxy named in config.toml would
// otherwise be used for hosts the user explicitly exempted.
//
// The entries follow the standard library's rules: "*" matches everything, a name
// matches the host itself and any name under it, an entry may carry a port, and an
// address may be written as a literal or as a CIDR.
func noProxyMatch(list, host, port string) bool {
	if list == "" || host == "" {
		return false
	}
	for _, entry := range strings.Split(list, ",") {
		entry = strings.TrimSpace(entry)
		switch {
		case entry == "":
			continue
		case entry == "*":
			return true
		case strings.Contains(entry, "/"):
			prefix, err := netip.ParsePrefix(entry)
			if err != nil {
				continue
			}
			if addr, err := netip.ParseAddr(host); err == nil && prefix.Contains(addr) {
				return true
			}
		default:
			entryHost, entryPort := splitHostPort(entry)
			if entryPort != "" && entryPort != port {
				continue
			}
			if domainMatch(entryHost, host) {
				return true
			}
		}
	}
	return false
}

// domainMatch reports whether a NO_PROXY entry names the host: exactly, or as the
// name it sits under (the rule the standard library's proxy resolution documents, so
// "x.com" matches x.com and anything.x.com, while "example.com" does not match
// notexample.com).
func domainMatch(entry, host string) bool {
	entry = strings.TrimPrefix(strings.TrimPrefix(entry, "*."), ".")
	entry = strings.TrimSuffix(entry, ".")
	if entry == "" {
		return false
	}
	if strings.EqualFold(entry, host) {
		return true
	}
	return strings.HasSuffix(strings.ToLower(host), "."+strings.ToLower(entry))
}

// splitHostPort splits a NO_PROXY entry into a host and an optional port. An entry
// that is not host:port — a bare name, or a bare IPv6 literal — is returned whole:
// a hand-written list must not have a name cut in half by a colon.
func splitHostPort(entry string) (host, port string) {
	if host, port, err := net.SplitHostPort(entry); err == nil {
		return host, port
	}
	return entry, ""
}

// hostPort is a URL's host and port, with the scheme's default port when the URL
// carries none: NO_PROXY entries may name a port, and a request URL almost never does.
func hostPort(u *url.URL) (host, port string) {
	host = u.Hostname()
	port = u.Port()
	if port != "" {
		return host, port
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		return host, "80"
	case "https":
		return host, "443"
	}
	return host, ""
}
