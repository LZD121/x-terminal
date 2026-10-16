package xweb

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/LZD121/x-terminal/internal/session"
)

// The translation service's path, assembled from the measured segments. Every piece that is not the
// post id or the language is the upstream's own vocabulary and was copied from the request the web
// client's bundle builds — `strato/column/None` is its own container expression,
// `translationSource=Some(X)` names its own translator, and `feature`/`timeout`/`onlyCached` are
// parameters it sends as `None`. None of it was chosen here, and none of it may be re-spelled: a
// character changed anywhere in the prefix or the suffix is a different request.
//
// The one measured thing that is *absent* is just as load-bearing: there is no `.json` suffix. The
// bundle's own call site is `apiClient.get(path, {}, headers, "")`, and `get(e,t,r,i=".json",n)`
// builds `/1.1/${path}${i}`, so the empty fourth argument is what removes the suffix. Measured on
// this host on 2026-10-16 through this client (see below): the unsuffixed path answers 200, and the
// same path with `.json` appended answers `404 {"code":34,"message":"Sorry, that page does not
// exist"}` — a false negative indistinguishable from "this endpoint does not exist", which is the
// single most expensive thing to get wrong here.
const (
	// translatePrefix is the path up to and including the column expression's opening, in the
	// upstream's own order.
	translatePrefix = "/1.1/strato/column/None/tweetId="
	// translateMiddle is everything between the post id and the language: the destination-language
	// column and the translator selection.
	translateMiddle = ",destinationLanguage=Some("
	// translateSuffix closes the column expression and names the service. Nothing between the
	// language and this is interpolated.
	translateSuffix = "),translationSource=Some(X),feature=None,timeout=None,onlyCached=None" +
		"/translation/service/translateTweet"
)

// translateID is the shape a post id is allowed to have before it is interpolated into the path:
// the upstream's ids are decimal and never longer than 20 digits.
var translateID = regexp.MustCompile(`^[0-9]{1,20}$`)

// translateLang is the shape a language tag is allowed to have. It is deliberately conservative —
// two or three letters, then zero or more `-`-separated alphanumeric subtags — because the value is
// a setting interpolated into a path by hand, where there is no url.Values to encode it. `zh-CN`,
// `de`, `ja` and `en` are the values every measurement used and all four match; a value holding a
// parenthesis, a comma, a slash, a space or a percent sign does not.
//
// A parenthesis or a comma would not merely be an odd language: it would close the column
// expression or start a second column, which is a different request that was never measured. That is
// the hazard graphql.go's query() comment describes for a query parameter, and here the value has no
// encoder to give it a second chance.
var translateLang = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]+)*$`)

// Translate issues one translation request and returns its response body. It asks the upstream's
// own translation service for the post `id` rendered in `lang`, over one authenticated REST GET.
//
// Measured on this host on 2026-10-16 through this client (`Client.Do`, so through the limiter),
// with the operator's own session cookie and the proxy at http://127.0.0.1:7897, one request at a
// time with four seconds between them:
//
//	GET https://api.x.com/1.1/strato/column/None/tweetId=<ID>,destinationLanguage=Some(<LANG>),translationSource=Some(X),feature=None,timeout=None,onlyCached=None/translation/service/translateTweet
//
//	request             | answer
//	--------------------+------------------------------------------------------------------------
//	zh-CN, id 211…425   | 200, translationState "Success", the post in Chinese
//	de, the same id     | 200, the same post in German
//	ja, the same id     | 200, the post in Japanese
//	the same path+`.json`| 404 {"code":34,"message":"Sorry, that page does not exist"}
//	id=1 (no such post) | 200 with translation:"" and translationState:"Missing"
//
// Two consequences shape the callers, and neither is this function's to enforce: a `200` does not
// mean a translation exists (id=1 answered 200 with nothing in it), and the response carries the
// answerer's `id_str` alongside the text. internal/xsource's ParseTranslation is where both are
// checked; this function returns the bytes of a successful read and nothing about their content.
//
// The header set is apiHeaders' — the same eight every authenticated API request of this client
// carries — and no transaction id was needed: the bundle's own signer guards the GraphQL path it is
// computed for and this endpoint answered without one, so no signer call is made here.
//
// A client without a valid session sends nothing: an unauthenticated request to this endpoint is a
// spent request that measures nothing, and the typed session error names what to fix.
//
// One call is one request and there is no retry: a refusal is a second diagnosis, not a second
// attempt (RUNBOOK.md's first section). A non-success outcome is classified by the same Classify the
// GraphQL path uses and returned as the same *APIError, so a refusal from this endpoint reaches the
// screen as the classifier's own word.
func (c *Client) Translate(ctx context.Context, id, lang string) ([]byte, error) {
	if !c.session.Valid() {
		return nil, fmt.Errorf("xweb: translate cannot be issued without a session: %w", session.ErrNoSession)
	}
	endpoint, err := translateURL(id, lang)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("xweb: translate: %w", err)
	}
	c.apiHeaders(req)

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("xweb: translate: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody+1))
	if err != nil {
		return nil, fmt.Errorf("xweb: translate: reading the answer: %w", err)
	}
	if len(body) > maxResponseBody {
		return nil, fmt.Errorf("xweb: translate answered more than the %d bytes this client will hold", maxResponseBody)
	}

	if outcome := Classify(resp.StatusCode, resp.Header, body); !outcome.Success() {
		reset, _ := rateLimitReset(resp.Header)
		return nil, &APIError{
			Operation: "translate",
			Outcome:   outcome,
			Status:    resp.StatusCode,
			Detail:    apiMessage(body),
			Reset:     reset,
		}
	}
	return body, nil
}

// translateURL assembles the measured path for one post and one language, or refuses both values
// before a request is built.
//
// Both are validated first and neither is escaped. url.PathEscape is deliberately *not* applied to
// the column expression: it escapes the parentheses and the result is a different request
// (`Some%28zh-CN%29`), which was never measured. Validation is what makes hand interpolation safe
// here — a refused value returns an error and builds no request at all.
func translateURL(id, lang string) (string, error) {
	if !translateID.MatchString(id) {
		return "", fmt.Errorf("xweb: translate: %q is not a post id: the upstream's ids are 1 to 20 decimal digits", id)
	}
	if !translateLang.MatchString(lang) {
		return "", fmt.Errorf("xweb: translate: %q is not a language tag this build will put in a URL: two or three letters and `-`-separated subtags", lang)
	}
	return APIBaseURL + translatePrefix + id + translateMiddle + lang + translateSuffix, nil
}
