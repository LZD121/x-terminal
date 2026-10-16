// Package release is the GitHub release wire the `update` verb reads, and
// nothing else: one GET of this repository's latest Release, and one GET of a
// URL that Release names.
//
// It is deliberately not part of internal/xweb. xweb is the x.com socket: its
// limiter is the account's request budget, its session field is the credential,
// and its NoRedirect rule is a statement about x.com — a redirect there means
// the request was refused. A GitHub Release is none of those, and a request that
// carries no x.com credential has no business in the package that owns the
// credential. What is borrowed is the one *http.Transport every request in this
// program goes through, xweb.NewTransport(cfg), so that the operator's resolved
// proxy and NO_PROXY rules apply here exactly as they do to a read.
//
// The redirect policy is net/http's own default, and that is deliberate: an
// asset URL answers 302 to a signed object URL (measured against the v1.1.0
// Release on 2026-10-16), NoRedirect would refuse to follow it, and net/http
// drops the Authorization header on a cross-host hop, which is what the signed
// URL requires.
//
// Nothing here retries, backs off or waits. RUNBOOK.md's first section is the
// reason: a repair is one request, and a failed repair is a second diagnosis.
package release

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb"
)

const (
	// apiBase is GitHub's REST origin, and repository is the one repository this
	// program upgrades itself from: the release workflow publishes there on a
	// `v*` tag push (.github/workflows/release.yml). The two are constants rather
	// than configuration because the upgrade path is a fact about the build, not
	// a setting — a verb whose upstream could be pointed anywhere is a verb that
	// downloads binaries from anywhere.
	apiBase    = "https://api.github.com"
	repository = "LZD121/x-terminal"

	// metaLimit bounds the release metadata this package will read. A Release
	// object is a few kilobytes; a body an order of magnitude over that is not one
	// to spend memory on, so it is a named error rather than a truncation. The
	// asset payload's bound is the caller's, because an asset is as large as the
	// release made it.
	metaLimit = 1 << 20

	// requestTimeout bounds one whole request, body included. A wire with no
	// timeout is one that can hang a verb that has nothing else to do.
	requestTimeout = 30 * time.Second
)

// Release is the minimum of a GitHub release object this verb uses: the tag it
// was published under and the assets it carries. A struct that spelled the whole
// release object out would be a second copy of a schema to keep in step, and
// every field nobody reads is a field that can drift.
type Release struct {
	Tag    string  `json:"tag_name"`
	Assets []Asset `json:"assets"`
}

// Asset is one file attached to a Release. Name and URL come from the Release
// itself and are never derived from the tag: a derived name is the guess that a
// release with no asset for this platform would answer with a 404 on a download
// URL instead of a state the verb can print.
type Asset struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	URL  string `json:"browser_download_url"`
}

// HTTPError is one non-2xx answer. It is this package's own type rather than a
// reuse of xweb.APIError: that type is the x.com classifier's vocabulary — its
// outcomes, its reset instant, its remedy sentences — and a GitHub answer is
// none of those. A caller inspects this one to say "404" instead of "failed".
type HTTPError struct {
	Status int
	URL    string
	Body   string
}

func (e *HTTPError) Error() string {
	detail := e.Body
	if detail == "" {
		detail = "the body was empty"
	}
	return fmt.Sprintf("release: %s answered %d: %s", e.URL, e.Status, detail)
}

// Client is the release wire: one http.Client over the transport every request
// in this program goes through, the API origin, and the token an operator may
// have exported.
//
// The fields are unexported so that a test inside this package points them at an
// httptest server without the production type growing a way to be lied to; the
// verifiable API is New, Latest and Fetch.
type Client struct {
	http *http.Client
	// token is a session.Secret and never a string: that type redacts every
	// formatting path (String, GoString, Format, MarshalJSON, MarshalText,
	// LogValue), which is the same mechanism that keeps the cookie out of a bug
	// report and the reason this package can claim the token cannot be printed
	// rather than promising that nobody prints it. It has exactly one reader,
	// Reveal, and one caller, headers.
	token session.Secret
	// apiBase is GitHub's REST origin and downloadBase the origin this tool will
	// download from. They are one value in production and two in a test.
	apiBase      string
	downloadBase string
	repo         string
}

// Format covers every verb fmt consults a type for, and it is here because fmt
// prints an unexported field's value without consulting that field's own String
// method: measured on this toolchain, 2026-10-16, a `%v` of a Client holding a
// session.Secret printed the token, while a Session — whose credential fields are
// exported — rendered the placeholder. So the client answers for itself, and no
// struct dump, JSON encode or log line of it can be the reason the token ends up
// in a bug report. The fields stay unexported and the type stays safe to print.
func (c Client) Format(f fmt.State, _ rune) {
	fmt.Fprint(f, c.describe())
}

// describe is the printable form: what the client is pointed at, never the
// credential it carries. session.Secret's own formatter renders the placeholder in
// the token's place.
func (c Client) describe() string {
	return fmt.Sprintf("release.Client{api:%s download:%s repo:%s token:%s}",
		c.apiBase, c.downloadBase, c.repo, c.token)
}

// New builds the release client: the transport for cfg's proxy (so the operator's
// proxy and NO_PROXY rules apply here as they do to a read), a named timeout, and
// the repository's two origins. It returns an error only when the transport cannot
// be built.
func New(cfg config.Config, token session.Secret) (*Client, error) {
	transport, err := xweb.NewTransport(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{
		http: &http.Client{
			Transport: transport,
			Timeout:   requestTimeout,
			// No CheckRedirect: net/http's default follows the 302 an asset URL
			// answers, and drops Authorization on the cross-host hop, which is
			// what the signed object URL requires.
		},
		token:        token,
		apiBase:      apiBase,
		downloadBase: "https://github.com",
		repo:         repository,
	}, nil
}

// Latest resolves the repository's latest published Release. It issues exactly one
// GET, and its answer is the only thing that decides which asset this machine
// takes: the asset list comes from the Release itself, never from a name derived
// from the tag.
func (c *Client) Latest(ctx context.Context) (Release, error) {
	url := c.apiBase + "/repos/" + c.repo + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	c.headers(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("release: GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := readBounded(resp, metaLimit, url)
	if err != nil {
		return Release{}, err
	}
	if err := statusError(resp, url, body); err != nil {
		return Release{}, err
	}

	var rel Release
	if err := json.Unmarshal(body, &rel); err != nil {
		// Reported as a decode failure and never as "no assets": a body this
		// package cannot read is a fact about the answer, and reading it as an
		// empty Release would turn a schema change into "there is nothing newer".
		return Release{}, fmt.Errorf("release: %s answered %d with a body that is not a release object: %w", url, resp.StatusCode, err)
	}
	return rel, nil
}

// Fetch reads one URL from the release origin, bounded by limit. It is what an
// asset and its published digest are read with: it sets the same headers through
// the same one place, sends through the same client, checks the status, and
// decodes nothing — a payload is bytes, and a digest is bytes this package's own
// parser reads.
//
// One request per URL, for ever: RUNBOOK.md's first section says a repair is one
// request and a failed repair is a second diagnosis, and that discipline is what
// makes this verb safe to run while the upstream is flaky.
func (c *Client) Fetch(ctx context.Context, url string, limit int64) ([]byte, error) {
	// The URL arrives from the Release's own asset list, which is the whole point
	// of reading the list rather than deriving names — but it is still a string a
	// remote document supplied, and this is the one place that decides what this
	// process will download. An asset URL that leaves the release origin is
	// refused rather than fetched: the digest that verifies the payload comes from
	// the same origin, so fetching the binary from anywhere else would be
	// verifying nothing.
	if !strings.HasPrefix(url, c.downloadBase+"/") {
		return nil, fmt.Errorf("release: %s is not under %s: a Release asset this tool downloads comes from the release's own origin", url, c.downloadBase)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.headers(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("release: GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := readBounded(resp, limit, url)
	if err != nil {
		return nil, err
	}
	if err := statusError(resp, url, body); err != nil {
		return nil, err
	}
	return body, nil
}

// headers is the one place this package's request headers are set.
//
// User-Agent is a requirement rather than a courtesy: api.github.com answers 403
// to a request without one (documented platform behaviour, and the reason this is
// not optional). Authorization is set only when the operator exported a token:
// the repository is public, so the anonymous path is the primary one, a private
// fork or a future flip still works with a token set, and nothing here verifies a
// permission the request did not need.
func (c *Client) headers(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", UserAgent())
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token.Reveal())
	}
}

// buildIdentity is what the User-Agent header announces to GitHub. It is a
// package-level variable because the version stamp belongs to internal/cli, which
// imports this package and therefore cannot be imported here; the update verb
// hands its own Version() over with SetIdentity before its first request. The
// default is the truth about a build with no stamp rather than an empty field.
var buildIdentity = "working tree"

// SetIdentity names the build in every following request's User-Agent. It is the
// update verb's one call into this package's request shape: the header's value is
// a fact about the build, and the empty string is ignored rather than announced,
// so an unstamped build goes on saying so.
func SetIdentity(v string) {
	if v != "" {
		buildIdentity = v
	}
}

// UserAgent is the identity every request here presents: "x-terminal/<tag>" for a
// release build, "x-terminal/working tree" for one built from a checkout.
func UserAgent() string {
	return "x-terminal/" + buildIdentity
}

// readBounded drains a response body and refuses one over the bound, with the +1
// guard internal/xweb already uses for exactly this reason: reading to the bound
// cannot tell "exactly this many bytes" from "more, cut", so the extra byte is
// what makes an over-long body a named error instead of a silent truncation.
func readBounded(resp *http.Response, limit int64, url string) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("release: reading %s: %w", url, err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("release: %s answered more than the %d bytes this tool will read", url, limit)
	}
	return body, nil
}

// statusError turns a non-2xx answer into this package's own *HTTPError, carrying
// a bounded slice of the body so that the caller can print what GitHub said
// rather than only that it said no.
func statusError(resp *http.Response, url string, body []byte) error {
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return nil
	}
	return &HTTPError{Status: resp.StatusCode, URL: url, Body: clip(string(body))}
}

// clipDetail bounds one diagnostic sentence. A body quoted into an error travels
// into a terminal and into whatever a user pastes from it, so it is one line and
// a sentence long, and a cut says that it cut.
func clip(s string) string {
	const limit = 200
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= limit {
		return s
	}
	for cut := limit; cut > 0; cut-- {
		if isRuneStart(s[cut]) {
			return s[:cut] + "..."
		}
	}
	return s[:limit] + "..."
}

// isRuneStart reports whether b begins a UTF-8 rune, so that a cut lands on a
// boundary rather than in the middle of a character.
func isRuneStart(b byte) bool { return b&0xc0 != 0x80 }

// tagPattern is the only tag shape this package will order:
// ^v<major>.<minor>.<patch>$. A looser pattern would have to decide what
// v1.1.0-rc1 means relative to v1.1.0, and a decision it cannot justify is worse
// than a refusal it can name.
var tagPattern = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

// Version is a release tag read as three numbers.
type Version struct {
	Major int
	Minor int
	Patch int
}

// ParseTag reads a release tag, or refuses the value by name.
func ParseTag(tag string) (Version, error) {
	fields := tagPattern.FindStringSubmatch(tag)
	if fields == nil {
		return Version{}, fmt.Errorf("release: %q is not a release tag: expected v<major>.<minor>.<patch>", tag)
	}
	var numbers [3]int
	for i, field := range fields[1:] {
		n, err := strconv.Atoi(field)
		if err != nil {
			return Version{}, fmt.Errorf("release: %q has a field that is not a number: %w", tag, err)
		}
		numbers[i] = n
	}
	return Version{Major: numbers[0], Minor: numbers[1], Patch: numbers[2]}, nil
}

// CompareTags orders two release tags: -1 when a is older than b, 0 when they are
// the same release, +1 when a is newer.
//
// Numeric field-wise comparison, not a string compare, and the test says why:
// v1.10.0 is newer than v1.9.0, and a string compare says the opposite.
func CompareTags(a, b string) (int, error) {
	left, err := ParseTag(a)
	if err != nil {
		return 0, err
	}
	right, err := ParseTag(b)
	if err != nil {
		return 0, err
	}
	switch {
	case left.Major != right.Major:
		return compareInt(left.Major, right.Major), nil
	case left.Minor != right.Minor:
		return compareInt(left.Minor, right.Minor), nil
	case left.Patch != right.Patch:
		return compareInt(left.Patch, right.Patch), nil
	}
	return 0, nil
}

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// AssetName is the release workflow's own asset name for a tag and platform:
// `x-terminal_<tag>_<goos>_<goarch>`. The form is not invented here — it is
// .github/workflows/release.yml's build step
// (`name="x-terminal_${{ steps.v.outputs.tag }}_${GOOS}_${GOARCH}"`), measured
// against the seven assets of the v1.1.0 Release on 2026-10-16.
func AssetName(tag, goos, goarch string) string {
	return fmt.Sprintf("x-terminal_%s_%s_%s", tag, goos, goarch)
}

// AssetFor picks the asset this platform's release carries. It is a pure function
// over the Release's own list — an exact match on the workflow's name, and nothing
// else — and its refusal carries both the name it wanted and the names the release
// does carry, sorted, so the state a verb prints is a fact about the release rather
// than a guess about it.
func AssetFor(assets []Asset, tag, goos, goarch string) (Asset, error) {
	return AssetNamed(assets, AssetName(tag, goos, goarch))
}

// AssetNamed finds the asset with exactly this name, refused the same way
// AssetFor refuses. The digest's own asset is looked up through this and not by
// appending a suffix to a URL: the release's list says which name the digest
// file carries, and a name the release does not carry is a finding rather than a
// 404 on a guessed path.
func AssetNamed(assets []Asset, name string) (Asset, error) {
	for _, asset := range assets {
		if asset.Name == name {
			return asset, nil
		}
	}
	names := make([]string, 0, len(assets))
	for _, asset := range assets {
		names = append(names, asset.Name)
	}
	slices.Sort(names)
	carries := strings.Join(names, ", ")
	if carries == "" {
		// The anti-vacuity arm: a list nobody read cannot answer a picker, and
		// "it carries " followed by nothing would read as though it had.
		carries = "no assets at all"
	}
	return Asset{}, fmt.Errorf("release: no asset named %s in this release; it carries %s", name, carries)
}
