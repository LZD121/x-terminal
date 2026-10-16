// Package xweb is the network path to x.com and the vocabulary for what the upstream
// answers back: one http.Transport built from the resolved proxy, one request choke
// point (Limiter), and the measured status table (Classify).
//
// Every rule in here was measured on 2026-10-14 and recorded in
// .planning/research/UPSTREAM-PROBE.md, because this upstream's failure mode is
// silence rather than documentation. Three of those measurements shape the code:
//
//   - The header set has to match the request class. A navigation request that carries
//     `authorization: Bearer` is answered 401 with a zero-byte body; the same request
//     with cookies only is answered 200. An API request is the other way around: it
//     needs the bearer, the cookie pair and `x-csrf-token`, and does not need the
//     transaction-id signer for the operations measured so far.
//   - Failure is empty-bodied and the status is the whole signal: 403 with no body is a
//     request refused before routing, 404 with no body is an unknown operation, 404
//     with a JSON body is a retired endpoint, 429 is a rate limit.
//   - An expired session is a 307 to the login flow with a zero-byte body, not a 401.
//
// Nothing in this package formats output or decides what to render: a caller gets
// bytes or a typed outcome whose Remedy says what to do about it.
package xweb

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

const (
	// BaseURL is the origin the browser-facing endpoints share: navigation
	// (https://x.com/home) and the GraphQL API (/i/api/graphql/...) both live here, and
	// it is the host every measured 200 was served from.
	BaseURL = "https://x.com"
	// APIBaseURL is the REST origin. Only its unauthenticated endpoints answered in the
	// probe (the guest token); its authenticated v1.1 endpoints are retired, answering
	// 404 with a JSON body.
	APIBaseURL = "https://api.x.com"
	// UserAgent is the browser identity every request presents. The measured request
	// classes were recorded with a browser user agent, and an edge that sorts clients by
	// how they announce themselves is not the place to look like a script.
	UserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36"
	// requestTimeout bounds one whole request, body included. The dial and handshake
	// timeouts do not cover a server that accepts a connection and then says nothing, and
	// a self-check that hangs is one of the ways this tool could stop telling the truth.
	requestTimeout = 30 * time.Second
)

// Client is the one way a request leaves this program: through the transport the
// configuration resolved, and through the limiter that counts it.
type Client struct {
	// HTTP is the client that carries the requests. It is exported because a test — or
	// the connectivity self-check pointed at a fake upstream — has to be able to put a
	// transport under it; the request itself still goes through Do.
	HTTP *http.Client
	// Limiter is the choke point every request passes through. A client built here
	// always has one, and Do refuses to send anything without it.
	Limiter *Limiter
	// proxy is the resolved first hop, or nil for a direct dial. Unexported on purpose: a
	// proxy URL may carry a password, and ProxyLine is the printable form.
	proxy *url.URL
	// session is the cookie an API request is authenticated with. Unexported: it is not a
	// setting, it is a credential, and the only way in is WithSession.
	session session.Session
	// registry is the operation table GraphQL resolves names through, pins included.
	// Unexported for the same reason as the proxy: a caller must not be able to swap it for
	// one that names a stale id.
	registry *volatile.Registry
	// signer is the transaction-id seam. Nil means NoopSigner.
	signer Signer
}

// WithSession returns a copy of the client that authenticates its API requests with sess. The
// copy shares the transport and the limiter — a second client would be a second request budget
// — and it is a copy rather than a mutation because the client is built before the session is
// read (the connectivity self-check runs before it too).
func (c *Client) WithSession(sess session.Session) *Client {
	copied := *c
	copied.session = sess
	return &copied
}

// WithSigner returns a copy of the client that signs its requests with signer.
func (c *Client) WithSigner(signer Signer) *Client {
	copied := *c
	copied.signer = signer
	return &copied
}

// NewClient builds the client the binary uses: the transport for cfg's proxy, a limiter
// holding cfg's request discipline, and the operation registry with the user's pins merged
// over the snapshot.
//
// The registry is loaded here rather than at each call site so that every client in the
// program resolves operation names through the same table — a client that could be built
// without one would be a client that could name a stale id. A pins file that cannot be read is
// carried as a warning on the registry (see volatile.Load) rather than failing this
// constructor: the transport is fine, and the doctor prints what is wrong with the table.
func NewClient(cfg config.Config) (*Client, error) {
	transport, err := NewTransport(cfg)
	if err != nil {
		return nil, err
	}
	proxy, err := proxySetting(cfg.Proxy)
	if err != nil {
		return nil, err
	}
	registry, err := volatile.Load(cfg.PinsPath())
	if err != nil {
		return nil, err
	}
	return &Client{
		HTTP: &http.Client{
			Transport:     transport,
			Timeout:       requestTimeout,
			CheckRedirect: NoRedirect,
		},
		Limiter:  NewLimiter(cfg.RequestFloor, cfg.SessionRequestCap),
		proxy:    proxy,
		registry: registry,
		signer:   NoopSigner{},
	}, nil
}

// Do issues req through the limiter. It is the only method here that sends anything: a
// caller that reached around it would be spending requests the budget line cannot see.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	switch {
	case c == nil || c.HTTP == nil:
		return nil, errors.New("xweb: the client has no http.Client - build it with NewClient")
	case c.Limiter == nil:
		return nil, errors.New("xweb: the client has no limiter - every request has to pass through one, and NewClient puts one in")
	}
	return c.Limiter.Do(ctx, func() (*http.Response, error) { return c.HTTP.Do(req) })
}

// ProxyLine names the first hop every request of this client takes, for a diagnostic
// line: "direct", or the proxy's scheme and host without its userinfo.
func (c *Client) ProxyLine() string {
	if c == nil || c.proxy == nil {
		return "direct"
	}
	if c.proxy.Scheme == "" {
		return c.proxy.Host
	}
	return c.proxy.Scheme + "://" + c.proxy.Host
}

// NoRedirect is the redirect policy every client that speaks to this upstream carries: a
// redirect is a state to name, not a page to chase. With a session that is no longer valid,
// x.com answers /home with a 307 into the login flow and a zero-byte body (measured
// 2026-10-14), so a client that followed the redirect would see the 200 of the login page
// and report a perfectly good session.
//
// It is exported rather than inlined into NewClient because a client built by hand — a test
// pointed at a fake upstream, say — has to be able to carry the same rule.
func NoRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
