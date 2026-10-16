package xweb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

// WebBearer is the authorization this client presents on API requests: the web client token
// x.com's own bundle carries.
//
// It is not a credential and it is not the account. It is baked into the bundle this project
// mined on 2026-10-14 (verbatim, `%3D` and all: that is the URL-encoded `=`), every maintained
// client ships the same string, and the thing that says *which* account is asking is the
// session cookie. A request with this header and no cookie is answered 403 with an empty body,
// which is why the client refuses to send one.
const WebBearer = "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"

const (
	// maxResponseBody bounds one API response. A page of 20 posts measured 130,245 bytes; this
	// is a hundred times that. It exists so that a redirect, a video stream or a proxy's idea of
	// a good time cannot be read into memory, not to be a limit anybody meets.
	maxResponseBody = 16 << 20
	// errorDetailLimit bounds the upstream's own words when they are carried into an error. An
	// error travels into a terminal and into whatever a user pastes into a bug report.
	errorDetailLimit = 200
)

// Signer is the seam the transaction-id signer (x-client-transaction-id) will land behind: a
// request builder asks it to add whatever headers the operation it is about to send requires.
//
// No implementation of it signs anything yet, and that is a measurement rather than a gap:
// measured on this host on 2026-10-14, `UserByScreenName` and `UserTweets` both answered 200 —
// 2,227 and 130,245 bytes — with no transaction id at all, so the research's "the signer is
// required" hypothesis does not hold for them. It stays a hypothesis for other operations;
// when one is measured to need it, this interface is where the per-operation requirement goes,
// and the change is one implementation and one line in the request builder.
type Signer interface {
	// Header adds the signer's headers to req. An error aborts the request.
	Header(ctx context.Context, req *http.Request) error
}

// NoopSigner adds nothing. It is what the client carries until an operation is measured to
// need more.
type NoopSigner struct{}

// Header implements Signer by doing nothing, successfully.
func (NoopSigner) Header(context.Context, *http.Request) error { return nil }

// SignerName returns the name of the header the request builder asks the signer
// to add. While NoopSigner is installed it returns "none": measured on this host
// on 2026-10-14, UserByScreenName and UserTweets both answered 200 with no
// transaction id at all (2,227 and 130,245 bytes), so no operation this build
// reads requires one yet. The method is what lets the doctor say so out loud
// instead of a reader having to trust a comment.
func (c *Client) SignerName() string {
	if c == nil || c.signer == nil {
		return "none"
	}
	if _, ok := c.signer.(NoopSigner); ok {
		return "none"
	}
	return "x-client-transaction-id"
}

// APIError is an answer the upstream gave that was not data. It carries the classified outcome
// rather than a string, so that a caller can name the state in its own words and act on it, and
// it carries the upstream's own reset instant when the answer carried one, so that a rate limit
// on a screen can say when the window reopens without parsing a header a second time.
type APIError struct {
	// Operation is the operation name the request was for.
	Operation string
	// Outcome is what the answer meant, from the classifier's measured table.
	Outcome Outcome
	// Status is the HTTP status the answer arrived under, when there was one.
	Status int
	// Detail is the upstream's own message, when the answer was an errors[] body it meant the
	// caller to read. It is bounded and never a body this package did not understand.
	Detail string
	// Reset is the instant the upstream's rate-limit window reopens, from x-rate-limit-reset, and
	// the zero time when the header was absent or unreadable. The zero value is a state of its
	// own — a rate limit with no end in sight — and not a reset that has already passed.
	Reset time.Time
}

// Error names the state, in the classifier's own words, with the upstream's reset instant when
// there is one. The instant is printed as an instant — never as a countdown — because a countdown
// is stale the moment it is written down and this program has exactly one clock.
func (e *APIError) Error() string {
	message := fmt.Sprintf("xweb: %s: %s", e.Operation, e.Outcome.Error())
	if e.Status != 0 {
		message += fmt.Sprintf(" (HTTP %d)", e.Status)
	}
	if e.Detail != "" {
		message += ": " + e.Detail
	}
	if !e.Reset.IsZero() {
		message += " reset " + e.Reset.UTC().Format(time.RFC3339)
	}
	return message + " — " + e.Outcome.Remedy()
}

// GraphQL issues one GraphQL operation and returns its response body.
//
// Everything about the request that could go stale comes from the registry: the query id, the
// feature switches and the field toggles. An operation the registry does not have is a typed
// error from Lookup — there is no stale-id fallback, because a request sent with one answers 404
// with an empty body, which is indistinguishable from every other empty answer this upstream
// gives.
//
// The wire method is this file's own and not the registry's, because the bundle the registry is
// mined from carries no method: a refresh (`cmd/x-opsgen` → `Registry.Merge`) replaces a whole
// entry with what the bundle carries, so a method stored there would be dropped, silently, the
// next time anyone runs `x-terminal doctor --refresh-ids`. The default is GET and the table
// above names the operations measured to need POST. The rest of the request is the measured one
// — `variables`, and `features`/`fieldToggles` when the operation declares flags — and the
// header set the measurements settled on. It goes through the limiter, and its answer goes
// through the classifier; a non-success outcome becomes an *APIError and the body is discarded
// rather than returned as if it were data.
func (c *Client) GraphQL(ctx context.Context, opName string, variables map[string]any) ([]byte, error) {
	op, err := c.operation(opName)
	if err != nil {
		return nil, err
	}
	if op.OperationType != "query" {
		return nil, fmt.Errorf("xweb: %s is a %s, and this client issues queries only", opName, op.OperationType)
	}
	if !c.session.Valid() {
		return nil, fmt.Errorf("xweb: %s cannot be issued without a session: %w", opName, session.ErrNoSession)
	}

	req, err := c.graphQLRequest(ctx, opName, op, variables)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("xweb: %s: %w", opName, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody+1))
	if err != nil {
		return nil, fmt.Errorf("xweb: %s: reading the answer: %w", opName, err)
	}
	if len(body) > maxResponseBody {
		return nil, fmt.Errorf("xweb: %s answered more than the %d bytes this client will hold", opName, maxResponseBody)
	}

	if outcome := Classify(resp.StatusCode, resp.Header, body); !outcome.Success() {
		// The reset instant travels with the error: the state line prints it, and nothing above
		// this package reads the header a second time.
		reset, _ := rateLimitReset(resp.Header)
		return nil, &APIError{
			Operation: opName,
			Outcome:   outcome,
			Status:    resp.StatusCode,
			Detail:    apiMessage(body),
			Reset:     reset,
		}
	}
	return body, nil
}

// operation resolves one operation through the registry the client carries, falling back to the
// embedded snapshot when it was built without one (which is what a hand-built client in a test
// does).
func (c *Client) operation(opName string) (volatile.Operation, error) {
	if c.registry == nil {
		return volatile.Lookup(opName)
	}
	return c.registry.Lookup(opName)
}

// postOperations names the operations measured to be answerable only over POST, one row per
// measurement. Every row carries, in its own comment, the measurement that put it there: a row
// without one cannot be judged later, and a wire method is a claim about the upstream's contract
// exactly the way a variable is.
//
// The table lives here rather than in the registry for a mechanical reason. The bundle
// cmd/x-opsgen mines carries the operation's id, its name, its declared type and its flags, and
// no method; Registry.Merge replaces a whole entry with what the bundle carries, and Write
// marshals the Operation struct — so a method stored in the registry would be dropped,
// silently, the next time anyone ran `x-terminal doctor --refresh-ids`, reverting the operation
// below to the GET that answers 404 with a zero-byte body. It is not a pin either: a pin is an
// operator's own override, and the shipped build has to be able to read without one.
var postOperations = map[string]bool{
	// SearchTimeline answers GET with HTTP 404 and a zero-byte body, and POST with HTTP 200 and
	// real results — measured on this host on 2026-10-15 through the operator's own session
	// cookie and the proxy at http://127.0.0.1:7897, with the same query id, the same variables,
	// the same feature switches, the same field toggles and the same headers on both requests.
	// The wire method is the only difference between the two answers.
	"SearchTimeline": true,
}

// wireMethod is the one place that decides the method a request is built with: POST for the
// operations the table above names, GET for every operation it does not. It does not consult the
// variables, the configuration, the session or anything else — the method is a property of the
// operation, and it was measured that way.
func wireMethod(opName string) string {
	if postOperations[opName] {
		return http.MethodPost
	}
	return http.MethodGet
}

// graphQLParams is the parameter set one operation's request carries, encoded once so that the
// two wire forms cannot disagree about which parameters a given operation sends: the GET form
// puts these in the query string, the POST form puts them in a JSON body.
//
// Features and fieldToggles are present only when the operation declares flags, which is what
// the web client's own request builder does (measured in the bundle:
// `_&&(u.features=JSON.stringify(_))`, where `_` is undefined for an empty list). That decision
// is made here and nowhere else, so it is identical in both forms.
type graphQLParams struct {
	Variables    string // the variables object, as JSON
	Features     string // {"name":true,…}, or "" when the operation declares no switches
	FieldToggles string // {"name":true,…}, or "" when the operation declares no toggles
}

func buildGraphQLParams(opName string, op volatile.Operation, variables map[string]any) (graphQLParams, error) {
	if variables == nil {
		variables = map[string]any{}
	}
	encoded, err := json.Marshal(variables)
	if err != nil {
		return graphQLParams{}, fmt.Errorf("xweb: %s: encoding the variables: %w", opName, err)
	}
	// The flag values are objects of {"name":true}, not the arrays the registry stores.
	params := graphQLParams{Variables: string(encoded)}
	if len(op.FeatureSwitches) > 0 {
		features, err := volatile.FeaturesJSON(op)
		if err != nil {
			return graphQLParams{}, fmt.Errorf("xweb: %s: %w", opName, err)
		}
		params.Features = features
	}
	if len(op.FieldToggles) > 0 {
		toggles, err := volatile.TogglesJSON(op)
		if err != nil {
			return graphQLParams{}, fmt.Errorf("xweb: %s: %w", opName, err)
		}
		params.FieldToggles = toggles
	}
	return params, nil
}

// query renders the parameters as the URL query the GET form carries. url.Values does the
// encoding: a query parameter built by string concatenation is how a variable containing an `&`
// becomes a different request than the one that was meant.
func (p graphQLParams) query() url.Values {
	q := url.Values{}
	q.Set("variables", p.Variables)
	if p.Features != "" {
		q.Set("features", p.Features)
	}
	if p.FieldToggles != "" {
		q.Set("fieldToggles", p.FieldToggles)
	}
	return q
}

// body renders the parameters as the JSON body the POST form carries. It is a struct rather than
// a map so that an absent flag parameter is absent from the body rather than a null: the two
// forms have to send the same parameters, and only the absence rule above decides which.
func (p graphQLParams) body() ([]byte, error) {
	return json.Marshal(struct {
		Variables    json.RawMessage `json:"variables"`
		Features     json.RawMessage `json:"features,omitempty"`
		FieldToggles json.RawMessage `json:"fieldToggles,omitempty"`
	}{
		Variables:    json.RawMessage(p.Variables),
		Features:     json.RawMessage(p.Features),
		FieldToggles: json.RawMessage(p.FieldToggles),
	})
}

// graphQLRequest builds the measured API request.
//
// The method is the operation's own (wireMethod), and it decides the form the parameters travel
// in: POST carries them as one JSON body with no query string, because that is the shape the web
// client's own request builder sends for a POST operation; GET carries them as URL-encoded query
// parameters.
//
// The header set is apiHeaders', which is the one function that sets it: this builder owns the
// per-operation method and the signer seam, and translate.go's own builder owns the translation
// path. Both are the same measured API request class — measured 2026-10-14 for the GraphQL
// operations and 2026-10-16 for the translation endpoint — and a second copy of the eight
// Header.Set calls is how the two would drift.
//
// The signer call stays here and not in apiHeaders on purpose: the signer is a per-operation seam
// (no operation measured so far needs a transaction id), and the translation endpoint was measured
// needing none either, so this is where a per-operation requirement goes when one is measured.
func (c *Client) graphQLRequest(ctx context.Context, opName string, op volatile.Operation, variables map[string]any) (*http.Request, error) {
	params, err := buildGraphQLParams(opName, op, variables)
	if err != nil {
		return nil, err
	}

	endpoint := BaseURL + "/i/api/graphql/" + url.PathEscape(op.QueryID) + "/" + url.PathEscape(opName)
	method := wireMethod(opName)
	var body io.Reader
	if method == http.MethodPost {
		encoded, err := params.body()
		if err != nil {
			return nil, fmt.Errorf("xweb: %s: encoding the request body: %w", opName, err)
		}
		body = bytes.NewReader(encoded)
	} else {
		endpoint += "?" + params.query().Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("xweb: %s: %w", opName, err)
	}
	c.apiHeaders(req)

	signer := c.signer
	if signer == nil {
		signer = NoopSigner{}
	}
	if err := signer.Header(ctx, req); err != nil {
		return nil, fmt.Errorf("xweb: %s: the signer refused to sign the request: %w", opName, err)
	}
	return req, nil
}

// apiHeaders sets the eight headers every authenticated API request of this client carries: the
// web bearer, the cookie pair, x-csrf-token mirroring ct0, x-twitter-auth-type,
// x-twitter-active-user, x-twitter-client-language, a browser user agent and content-type.
//
// It is one function because it is one request class, not because the two builders happen to agree
// today: the set was measured answering 200 on 2026-10-14 for the GraphQL operations
// (UserByScreenName, UserTweets and the rest of the read path) and on 2026-10-16 for the
// translation endpoint (translate.go), with no transaction id on either. It is not the navigation
// set: the same bearer on a navigation request is answered 401 with a zero-byte body, which is why
// the request class and the header set have to match — and why this lives beside the client that
// decides the class rather than at each call site.
//
// The literals, their values and their order are the measured ones and this function is their one
// home; TestGraphQLBuildsTheMeasuredRequest pins them.
func (c *Client) apiHeaders(req *http.Request) {
	req.Header.Set("authorization", "Bearer "+WebBearer)
	req.Header.Set("cookie", "auth_token="+c.session.AuthToken.Reveal()+"; ct0="+c.session.CTO.Reveal())
	req.Header.Set("x-csrf-token", c.session.CTO.Reveal())
	req.Header.Set("x-twitter-auth-type", "OAuth2Session")
	req.Header.Set("x-twitter-active-user", "yes")
	req.Header.Set("x-twitter-client-language", "en")
	req.Header.Set("user-agent", UserAgent)
	req.Header.Set("content-type", "application/json")
}

// apiMessage reads the upstream's own words out of an answer, bounded. A body that is JSON is
// quoted (an errors[] message when there is one, the body itself otherwise, which is how a
// rejected request shape names the variable it did not like); a body that is not JSON — an HTML
// interstitial, a proxy's page — contributes nothing, because quoting one of those into a terminal
// is a worse diagnostic than no message at all.
//
// The path an error carries is part of the sentence when there is one, because that is where the
// upstream names what it is complaining about: a missing variable answers 422 with the message
// "must be defined" and path ["variable","includePromotedContent"] (measured 2026-10-15), and a
// reader that printed the message alone would print a sentence naming nothing.
func apiMessage(body []byte) string {
	if errs, apiJSON := parseErrors(body); apiJSON {
		if len(errs) > 0 && (strings.TrimSpace(errs[0].Message) != "" || len(errs[0].Path) > 0) {
			message := strings.TrimSpace(errs[0].Message)
			if len(errs[0].Path) > 0 {
				if message != "" {
					message += " "
				}
				message += "at " + strings.Join(errs[0].Path, ".")
			}
			return bounded(message)
		}
		if trimmed := strings.TrimSpace(string(body)); json.Valid([]byte(trimmed)) {
			return bounded(trimmed)
		}
	}
	return ""
}

// bounded cuts a diagnostic to errorDetailLimit runes on a rune boundary, and says that it cut
// something.
func bounded(message string) string {
	if utf8.RuneCountInString(message) <= errorDetailLimit {
		return message
	}
	return string([]rune(message)[:errorDetailLimit]) + "…"
}
