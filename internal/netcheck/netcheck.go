// Package netcheck answers one question honestly: is there a path from this host to
// x.com right now, and if there is not, which link of it is broken?
//
// It exists because this upstream does not say so itself. An unreachable host, a proxy
// that is not running, a proxy that cannot reach upstream, an edge interstitial and an
// expired session all arrive as nothing useful — a timeout, an empty body, a status with
// no explanation — and the failure this project is built against is any of them being
// rendered as an empty timeline. So the check runs the links in order, stops at the first
// that breaks, and names it: no route to x.com, proxy refused, proxy reached but upstream
// blocked, TLS/Cloudflare challenge, or authenticated read OK.
//
// Every request it makes goes through the same xweb.Client the API layer will use, so the
// request budget it reports is the budget it spent.
package netcheck

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xsource"
	"github.com/LZD121/x-terminal/internal/xweb"
)

const (
	// dialTimeout bounds one TCP connect of the route and proxy stages. It matches the
	// transport's own timeout, so a black-holed hop costs the same wherever it is met.
	dialTimeout = 10 * time.Second
	// peek is how much of a response body any stage reads. The check needs the shape of an
	// answer, not its content: a 300KB authenticated shell is discarded after its first few
	// kilobytes, and an errors[] body is small by construction.
	peek = 4 << 10
	// detailLimit bounds one Detail line. A Detail travels into a terminal and into whatever
	// a user pastes into a bug report, and a body quoted into one must not be able to carry
	// something that belongs nowhere near it.
	detailLimit = 200
)

// Outcome names where the path broke, or that it did not. The five outcomes the project
// promises are here verbatim: no route to x.com, proxy refused, proxy reached but upstream
// blocked, TLS/Cloudflare challenge, authenticated read OK.
type Outcome string

const (
	// OutcomeConfigFailed: the path could not even be built — an unusable proxy setting.
	// Nothing was dialed.
	OutcomeConfigFailed Outcome = "transport unusable"
	// OutcomeRouteOK: x.com resolves, and (when this client dials directly) the address it
	// resolves to accepted a connection.
	OutcomeRouteOK Outcome = "route to x.com OK"
	// OutcomeNoRoute: x.com did not resolve, or no connection to it could be opened. No
	// proxy is configured in that second case, because a proxy's own hop is stage 2.
	OutcomeNoRoute Outcome = "no route to x.com"
	// OutcomeProxyOK: the configured proxy accepted a TCP connection.
	OutcomeProxyOK Outcome = "proxy reached"
	// OutcomeProxyRefused: nothing is listening on the configured proxy's host:port, or it
	// did not answer. This is the outcome that must never be reported as an expired session.
	OutcomeProxyRefused Outcome = "proxy refused"
	// OutcomeUpstreamOK: a plain GET of https://x.com/ came back with a body.
	OutcomeUpstreamOK Outcome = "upstream reachable"
	// OutcomeUpstreamBlocked: the proxy answered but the request through it did not complete,
	// or came back as something other than a page.
	OutcomeUpstreamBlocked Outcome = "proxy reached but upstream blocked"
	// OutcomeChallenge: an edge interstitial answered instead of x.com.
	OutcomeChallenge Outcome = "TLS/Cloudflare challenge"
	// OutcomeAuthenticatedOK: one authenticated navigation read came back 200 with a body.
	OutcomeAuthenticatedOK Outcome = "authenticated read OK"
	// OutcomeAPIReadOK: the registry-driven API read — the call the client actually reads x.com
	// with — came back with data, and the answer named the account the session belongs to. This
	// is the outcome the phase exists to make true: the navigation read above proves the cookie
	// is accepted, and this proves the operation id, the feature switches and the API header set
	// all work from this host.
	OutcomeAPIReadOK Outcome = "API read OK"
	// OutcomeAPIRejected: the session was accepted for navigation, and the API read did not
	// deliver data. It is its own outcome rather than "session rejected", because the two are
	// fixed differently: the Detail carries what the classifier made of the answer (an unknown
	// operation, a signer, a rate limit), and those are not all the session's fault.
	OutcomeAPIRejected Outcome = "API read failed"
	// OutcomeSessionRejected: the authenticated read was attempted and the upstream refused
	// it, or answered with something xweb.Classify had a better name for (which the Detail
	// carries).
	OutcomeSessionRejected Outcome = "session rejected"
	// OutcomeSessionSkipped: there is no session to authenticate with, so the reads that need
	// one were not attempted. This is a state, not a failure: the transport path can be
	// perfectly fine while the cookie is missing, and calling that a session error is exactly
	// the confusion this command exists to remove.
	OutcomeSessionSkipped Outcome = "session not configured"
)

// Failed reports whether this outcome means the check failed. Only a skipped stage is
// neither: the path was proven and the read was never attempted, which the line says. The
// zero value is a failure, so an outcome a caller forgot to set cannot pass as success.
func (o Outcome) Failed() bool {
	switch o {
	case OutcomeRouteOK, OutcomeProxyOK, OutcomeUpstreamOK, OutcomeAuthenticatedOK, OutcomeAPIReadOK, OutcomeSessionSkipped:
		return false
	default:
		return true
	}
}

// Result is one stage's line: what came of it, in one bounded sentence, and how long it
// took. Elapsed is zero for a stage that did not run.
type Result struct {
	Outcome Outcome
	Detail  string
	Elapsed time.Duration
}

// Target is the path Check measures: a resolver, a dialer for the links below HTTP, and
// the one client every request goes through. The fields are unexported and set by
// NewTarget, so the only way to point a check somewhere other than the real x.com is from
// inside this package — which is how the tests drive all of its outcomes from an
// httptest.Server without a socket leaving the machine.
type Target struct {
	client  *xweb.Client
	address string // the configured proxy's host:port, or "" for a direct dial
	resolve func(ctx context.Context, host string) ([]string, error)
	dial    func(ctx context.Context, network, address string) (net.Conn, error)
}

// NewTarget builds the real path from the resolved configuration: the transport for cfg's
// proxy, the system resolver and the system dialer.
func NewTarget(cfg config.Config) (*Target, error) {
	client, err := xweb.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return NewTargetWithClient(cfg, client)
}

// NewTargetWithClient is NewTarget around a client the caller already has: a program that checks
// the path while it is working shares the one client (and so the one limiter, and the one budget
// line) with the reads it is diagnosing, rather than building a second transport whose requests no
// budget reports. The client is used exactly as it is: no session is attached here, and the stages
// that need one are handed the session by the caller.
func NewTargetWithClient(cfg config.Config, client *xweb.Client) (*Target, error) {
	if client == nil {
		return nil, fmt.Errorf("netcheck: a target needs a client to speak through")
	}
	address, err := xweb.ProxyAddress(cfg.Proxy)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: dialTimeout}
	return &Target{
		client:  client,
		address: address,
		resolve: net.DefaultResolver.LookupHost,
		dial:    dialer.DialContext,
	}, nil
}

// Client is the client the stages that speak HTTP use — the same client, transport and
// limiter the API layer will use. A caller that prints the request budget reads it from
// here, so the number cannot describe a different limiter than the one that spent it.
func (t *Target) Client() *xweb.Client { return t.client }

// Check runs the stages against a path built from cfg. It is the whole self-check; a
// caller that needs the request budget as well builds the Target itself (NewTarget) and
// calls CheckWith, which is what `doctor` does.
func Check(ctx context.Context, cfg config.Config, sess session.Session) []Result {
	target, err := NewTarget(cfg)
	if err != nil {
		return []Result{{Outcome: OutcomeConfigFailed, Detail: clip(err.Error())}}
	}
	return CheckWith(ctx, target, sess)
}

// CheckWith runs the stages in order and returns one Result per stage that ran, stopping at
// the first that failed. The last Result is the verdict; the ones before it say how far the
// path got, which is what tells a refused proxy from a blocked upstream.
func CheckWith(ctx context.Context, target *Target, sess session.Session) []Result {
	if target == nil || target.client == nil {
		return []Result{{Outcome: OutcomeConfigFailed, Detail: "the check has no client to speak through"}}
	}
	results := make([]Result, 0, 4)

	// Stage 1 — route: the name, and the first hop of the path this client would really take.
	start := time.Now()
	route := target.route(ctx)
	results = append(results, Result{Outcome: route.outcome, Detail: clip(route.detail), Elapsed: time.Since(start)})
	if route.outcome.Failed() {
		return results
	}

	// Stage 2 — proxy, only when one is configured. A bare TCP connect, no TLS and no
	// request: this is the stage that separates "nothing is listening" from "the proxy is
	// running and the upstream is not answering".
	if target.address != "" {
		start = time.Now()
		conn, err := target.dial(ctx, "tcp", target.address)
		if err != nil {
			results = append(results, Result{
				Outcome: OutcomeProxyRefused,
				Detail:  clip(fmt.Sprintf("dialing the proxy at %s failed: %v", target.address, err)),
				Elapsed: time.Since(start),
			})
			return results
		}
		conn.Close()
		results = append(results, Result{
			Outcome: OutcomeProxyOK,
			Detail:  clip(fmt.Sprintf("the proxy at %s accepted a TCP connection", target.address)),
			Elapsed: time.Since(start),
		})
	}

	// Stage 3 — upstream: one navigation request, carrying no credentials at all, because
	// what is being tested here is the path rather than the session.
	start = time.Now()
	resp, err := target.fetch(ctx, http.MethodGet, xweb.BaseURL+"/", session.Session{})
	if err != nil {
		outcome := OutcomeUpstreamBlocked
		if target.address == "" {
			// Nothing is proxying this connection, and the request did not complete after the
			// route stage accepted the address: the network is what dropped it.
			outcome = OutcomeNoRoute
		}
		results = append(results, Result{outcome, clip(fmt.Sprintf("GET %s/ failed: %v", xweb.BaseURL, err)), time.Since(start)})
		return results
	}
	classified := xweb.Classify(resp.status, resp.header, resp.prefix)
	switch {
	case resp.ok() && len(resp.prefix) > 0:
		results = append(results, Result{
			Outcome: OutcomeUpstreamOK,
			Detail:  clip(fmt.Sprintf("GET %s/ answered %d with %s", xweb.BaseURL, resp.status, resp.size())),
			Elapsed: time.Since(start),
		})
	case classified == xweb.OutcomeChallenge:
		results = append(results, Result{
			Outcome: OutcomeChallenge,
			Detail:  clip(fmt.Sprintf("GET %s/ answered %d with an edge interstitial rather than a page: %s", xweb.BaseURL, resp.status, classified.Remedy())),
			Elapsed: time.Since(start),
		})
		return results
	default:
		// Every other shape is the upstream being unreachable through a proxy that is up: a
		// 5xx from the proxy itself, an empty body, a status no page uses.
		results = append(results, Result{
			Outcome: OutcomeUpstreamBlocked,
			Detail:  clip(fmt.Sprintf("GET %s/ answered %d (%s: %s)", xweb.BaseURL, resp.status, classified.Error(), classified.Remedy())),
			Elapsed: time.Since(start),
		})
		return results
	}

	// Stage 4 — session. Skipped, not failed, when there is nothing to authenticate with.
	if !sess.Valid() {
		results = append(results, Result{
			Outcome: OutcomeSessionSkipped,
			Detail:  clip(fmt.Sprintf("no session cookie is configured (%s); the authenticated read was not attempted, and nothing in this line is a failure of the network path", sess.Summary())),
		})
		return results
	}

	start = time.Now()
	resp, err = target.fetch(ctx, http.MethodGet, xweb.BaseURL+"/home", sess)
	if err != nil {
		results = append(results, Result{
			Outcome: OutcomeSessionRejected,
			Detail:  clip(fmt.Sprintf("the authenticated read of %s/home failed: %v", xweb.BaseURL, err)),
			Elapsed: time.Since(start),
		})
		return results
	}
	classified = xweb.Classify(resp.status, resp.header, resp.prefix)
	if !classified.Success() {
		results = append(results, Result{
			Outcome: OutcomeSessionRejected,
			Detail:  clip(fmt.Sprintf("%s: %s (a %d from %s/home)", classified.Error(), classified.Remedy(), resp.status, xweb.BaseURL)),
			Elapsed: time.Since(start),
		})
		return results
	}
	results = append(results, Result{
		Outcome: OutcomeAuthenticatedOK,
		Detail: clip(fmt.Sprintf("the session cookie was accepted: %s/home answered %d with %s",
			xweb.BaseURL, resp.status, resp.size())),
		Elapsed: time.Since(start),
	})

	// Stage 5 — the API read: the registry-driven call this client actually reads x.com with.
	// The navigation read above proves the cookie is accepted; this proves the operation id, the
	// feature switches with it and the API header set all work from this host, and it is the only
	// line of the report that names the account. It costs one request, and it is a navigation
	// read no longer: the API request class is a different one, with the bearer and the CSRF
	// header and no page around it.
	//
	// A failure here is its own outcome rather than "session rejected": the answer could name an
	// unknown operation, a missing signer, a rate limit or a challenge, and the classifier's
	// sentence for it travels in the Detail.
	start = time.Now()
	raw, err := target.client.WithSession(sess).GraphQL(ctx, viewerOperation, nil)
	if err != nil {
		results = append(results, Result{
			Outcome: OutcomeAPIRejected,
			Detail:  clip(fmt.Sprintf("the registry-driven %s read failed: %v", viewerOperation, err)),
			Elapsed: time.Since(start),
		})
		return results
	}
	viewer, err := xsource.ParseUser(raw)
	if err != nil {
		results = append(results, Result{
			Outcome: OutcomeAPIRejected,
			Detail:  clip(fmt.Sprintf("the %s answer could not be read: %v", viewerOperation, err)),
			Elapsed: time.Since(start),
		})
		return results
	}
	results = append(results, Result{
		Outcome: OutcomeAPIReadOK,
		Detail: clip(fmt.Sprintf("the %s operation answered 200 for @%s (%s): the API path is the one this client will read with",
			viewerOperation, viewer.ScreenName, viewer.Name)),
		Elapsed: time.Since(start),
	})
	return results
}

// viewerOperation is the operation that says whose session this is. It is a name, not an id: the
// id comes from the registry, and there is no id literal anywhere but there.
const viewerOperation = "Viewer"

// routeStage is stage 1's outcome and the sentence that goes with it.
type routeStage struct {
	outcome Outcome
	detail  string
}

// route resolves the upstream's name, and — when this client dials directly — connects to
// the address it resolved to.
//
// With a proxy configured, the connect is left to the next stage: the first hop of the path
// this client takes is then the proxy, and a direct connect to x.com would be testing a path
// the program never uses. On this host that direct path is blocked while the proxy works, so
// testing it would report a failure that affects nothing — the kind of true-but-useless
// answer this command exists to avoid.
func (t *Target) route(ctx context.Context) routeStage {
	addrs, err := t.resolve(ctx, "x.com")
	if err != nil {
		return routeStage{OutcomeNoRoute, fmt.Sprintf("x.com did not resolve: %v", err)}
	}
	if len(addrs) == 0 {
		return routeStage{OutcomeNoRoute, "x.com resolved to no addresses at all"}
	}
	detail := "x.com resolves to " + strings.Join(addrs, ", ")

	if t.address != "" {
		return routeStage{OutcomeRouteOK, detail + "; the first hop is the proxy, which the next stage connects to"}
	}
	addr := net.JoinHostPort(addrs[0], "443")
	conn, err := t.dial(ctx, "tcp", addr)
	if err != nil {
		return routeStage{OutcomeNoRoute, fmt.Sprintf("%s; %s did not accept a TCP connection: %v", detail, addr, err)}
	}
	conn.Close()
	return routeStage{OutcomeRouteOK, detail + "; " + addr + " accepted a TCP connection"}
}

// response is what a stage needs from one answer: its status, its headers, the length the
// body declared, and a bounded prefix of that body. The prefix is deliberate — the check has
// no business holding a timeline in memory to find out whether one arrived.
type response struct {
	status int
	header http.Header
	prefix []byte
	// length is the declared Content-Length, or -1 when the body declared none: a gzip'd or
	// chunked answer has no size until it has been read, and this package does not pretend
	// otherwise.
	length int64
}

// ok reports whether the status is one a page is served under.
func (r response) ok() bool { return r.status >= 200 && r.status < 400 }

// size describes how much of a response was seen, without claiming more than was: the
// declared length when the server sent one, and otherwise what was read. A gzip'd or chunked
// body has no declared length, and reporting the peek as if it were the whole body would be
// the kind of small lie this package exists to keep out.
func (r response) size() string {
	if r.length >= 0 {
		return fmt.Sprintf("%d bytes", r.length)
	}
	return fmt.Sprintf("at least %d bytes (the body declared no length)", len(r.prefix))
}

// fetch issues one request through the shared client, which is the only place in this
// package that speaks HTTP: every request therefore passes through the limiter, and the
// budget a caller reads afterwards is this check's own spending.
//
// The header set is the measured navigation class. With a session, this carries the cookies
// and nothing else: the same request with `authorization: Bearer` was answered 401 with a
// zero-byte body (measured 2026-10-14), while an API request is the opposite way around. A
// caller that adds auth headers here would be testing a request class x.com rejects.
func (t *Target) fetch(ctx context.Context, method, url string, sess session.Session) (response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return response{}, err
	}
	req.Header.Set("user-agent", xweb.UserAgent)
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("referer", xweb.BaseURL+"/")
	if sess.Valid() {
		req.Header.Set("cookie", "auth_token="+sess.AuthToken.Reveal()+"; ct0="+sess.CTO.Reveal())
	}

	resp, err := t.client.Do(ctx, req)
	if err != nil {
		return response{}, err
	}
	defer resp.Body.Close()
	prefix, err := io.ReadAll(io.LimitReader(resp.Body, peek))
	if err != nil {
		return response{}, err
	}
	// resp.ContentLength is left as it is. Clamping it to what was read would turn "the body
	// declared no length" into a number that looks like the body's size, which is exactly the
	// kind of confident wrong answer this check exists to keep out: both live stages come back
	// gzip'd, so both arrive with no declared length at all.
	return response{status: resp.StatusCode, header: resp.Header, prefix: prefix, length: resp.ContentLength}, nil
}

// clip bounds one diagnostic sentence to detailLimit bytes, on a rune boundary, and says
// that it cut something.
func clip(s string) string {
	if len(s) <= detailLimit {
		return s
	}
	for cut := detailLimit; cut > 0; cut-- {
		if utf8.RuneStart(s[cut]) {
			return s[:cut] + "..."
		}
	}
	return s[:detailLimit]
}
