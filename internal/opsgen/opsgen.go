// Package opsgen is the refresh path for the operation registry: it reads the operation
// records out of x.com's own web bundle and merges them into the registry file that
// internal/xweb/volatile embeds.
//
// It exists because the ids rotate with every web build, and because the only place that
// knows both the ids and the feature switches an operation needs is that bundle — the same
// artifact answers "which id?" and "which flags?", which is the pair that drifts together.
//
// It is a development tool, not part of the read path: `cmd/x-opsgen` drives it, and
// `x-terminal doctor --refresh-ids` is the same logic behind a flag. Nothing here runs on a
// normal invocation, and nothing here can delete an entry — the merge rules are in
// internal/xweb/volatile, next to the data they protect.
//
// Measured on 2026-10-14 (see .planning/research/UPSTREAM-PROBE.md):
//
//   - The operation map is only in the shell an *authenticated* client gets. The entry chunk
//     the logged-out shell loads — 30,364 bytes — contains no queryId at all, so a refresh
//     needs the session, and it is a navigation request: the same GET with
//     `authorization: Bearer` added is answered 401 with a zero-byte body.
//   - The authenticated shell references https://abs.twimg.com/responsive-web/client-web/
//     main.<hash>a.js (1,157,073 bytes on 2026-10-14), which held 104 records shaped
//     {queryId:"…",operationName:"…",operationType:"…",metadata:{featureSwitches:[…],fieldToggles:[…]}}.
//     The logged-out shell references a different file at a different path, which is why
//     BundleURL refuses to guess and names what it looked for.
package opsgen

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"time"

	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

const (
	// ShellURL is the page a refresh starts from: the authenticated shell, the only document
	// that publishes the main bundle's URL.
	ShellURL = "https://x.com/home"
	// RegistryPath is the registry file a refresh rewrites, relative to the repository root. The
	// snapshot is embedded in the binary at build time, so a refresh edits the source file and is
	// only in effect after a rebuild: the ids are reviewed in a diff rather than changing under a
	// running client.
	RegistryPath = "internal/xweb/volatile/registry.json"
	// BundleOrigin is the host the bundle is served from. It is not x.com, which is why the
	// session cookie is not sent with the bundle download: a browser holding a .x.com cookie
	// does not send it to abs.twimg.com either.
	BundleOrigin = "https://abs.twimg.com"
	// shellLimit and bundleLimit bound the two downloads. The shell was 308,786 bytes and the
	// bundle 1,157,073 bytes when measured; the limits are large enough not to refuse a
	// bigger build and small enough that a redirect into a video stream cannot be read into
	// memory.
	shellLimit  = 8 << 20
	bundleLimit = 32 << 20
	// requestTimeout bounds one download. Neither the bundle nor the shell is streamed to the
	// user, so a download that has not finished in this long is not going to.
	requestTimeout = 60 * time.Second
)

// Record is one operation as the bundle declares it.
type Record struct {
	Name string
	volatile.Operation
}

// Refresh is the result of one live refresh: what the bundle declared, and where it came from.
type Refresh struct {
	Records   []Record
	BundleURL string
	SHA256    string
	Bytes     int
	At        time.Time
}

// Operations is the records as the map a merge takes.
func Operations(records []Record) map[string]volatile.Operation {
	mined := make(map[string]volatile.Operation, len(records))
	for _, record := range records {
		mined[record.Name] = record.Operation
	}
	return mined
}

// Operations is this refresh's records as the map a merge takes.
func (r Refresh) Operations() map[string]volatile.Operation { return Operations(r.Records) }

var (
	// recordPattern matches one operation record as a unit — the id with its own name, in the
	// order the bundle writes them. Matching the id alone would misalign ids with operations,
	// which is the failure mode the discovery recipe warns about (research/STACK.md §1.2).
	recordPattern = regexp.MustCompile(`\{queryId:"([A-Za-z0-9_-]{10,96})",operationName:"([A-Za-z][A-Za-z0-9_]*)",operationType:"(query|mutation)"(?:,metadata:\{([^}]*)\})?`)
	// switchesPattern and togglesPattern read the two lists out of a record's metadata. Either
	// may be absent: 42 of the 104 mined records declare no feature switches at all.
	switchesPattern = regexp.MustCompile(`featureSwitches:\[([^\]]*)\]`)
	togglesPattern  = regexp.MustCompile(`fieldToggles:\[([^\]]*)\]`)
	// stringPattern reads the elements out of one of those lists.
	stringPattern = regexp.MustCompile(`"([^"]*)"`)
	// bundleAbsPattern and bundleRelPattern find the main bundle the shell references. The
	// build has changed shape before (Vite links the file directly; the older webpack build
	// embeds a hash map to reconstruct it), so both the absolute URL and the bare path are
	// accepted — but only for the responsive-web main bundle, because refusing to guess is
	// what keeps a refresh from merging records out of the wrong file.
	bundleAbsPattern = regexp.MustCompile(`https://abs\.twimg\.com/(responsive-web/client-web/main\.[A-Za-z0-9_-]+a\.js)`)
	bundleRelPattern = regexp.MustCompile(`(responsive-web/client-web/main\.[A-Za-z0-9_-]+a\.js)`)
)

// Parse reads every operation record out of a bundle.
//
// A bundle that yields no records is an error rather than an empty result: the merge must
// never be able to run on a file whose shape changed, because "nothing changed" and "I could
// not read this" would then be the same sentence.
func Parse(bundle []byte) ([]Record, error) {
	matches := recordPattern.FindAllSubmatch(bundle, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("opsgen: no {queryId,operationName,operationType,metadata} records in the bundle; if x.com changed the shape of its bundle, this pattern needs updating before a refresh is worth anything")
	}

	byName := make(map[string]Record, len(matches))
	for _, m := range matches {
		record := Record{
			Name: string(m[2]),
			Operation: volatile.Operation{
				QueryID:         string(m[1]),
				OperationType:   string(m[3]),
				FeatureSwitches: list(string(m[4]), switchesPattern),
				FieldToggles:    list(string(m[4]), togglesPattern),
			},
		}
		if previous, ok := byName[record.Name]; ok && !sameRecord(previous, record) {
			return nil, fmt.Errorf("opsgen: the bundle names %s twice, as %q and %q, and there is no way to tell which one is current", record.Name, previous.QueryID, record.QueryID)
		}
		byName[record.Name] = record
	}

	records := make([]Record, 0, len(byName))
	for _, record := range byName {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Name < records[j].Name })
	return records, nil
}

// list reads one ["a","b"] list out of a record's metadata. An absent list and an empty list
// are the same answer — the operation declares no such flags — and both come back as nil.
func list(metadata string, pattern *regexp.Regexp) []string {
	found := pattern.FindStringSubmatch(metadata)
	if found == nil {
		return nil
	}
	var names []string
	for _, m := range stringPattern.FindAllStringSubmatch(found[1], -1) {
		names = append(names, m[1])
	}
	return names
}

func sameRecord(a, b Record) bool {
	if a.QueryID != b.QueryID || a.OperationType != b.OperationType || len(a.FeatureSwitches) != len(b.FeatureSwitches) || len(a.FieldToggles) != len(b.FieldToggles) {
		return false
	}
	for i := range a.FeatureSwitches {
		if a.FeatureSwitches[i] != b.FeatureSwitches[i] {
			return false
		}
	}
	for i := range a.FieldToggles {
		if a.FieldToggles[i] != b.FieldToggles[i] {
			return false
		}
	}
	return true
}

// BundleURL finds the main web bundle the shell references.
//
// It is deliberately narrow. The shell also references a dozen lazily loaded chunks, and the
// operation map is in the main one; picking the wrong file would produce a merge that says
// "0 added, 0 updated" for a bundle that simply had none, which is why the "no reference"
// case is an error naming the pattern that failed.
func BundleURL(shell []byte) (string, error) {
	if m := bundleAbsPattern.FindSubmatch(shell); m != nil {
		return BundleOrigin + "/" + string(m[1]), nil
	}
	if m := bundleRelPattern.FindSubmatch(shell); m != nil {
		return BundleOrigin + "/" + string(m[1]), nil
	}
	return "", fmt.Errorf("opsgen: the shell references no responsive-web/client-web/main.<hash>a.js bundle, so there is nothing to mine; a logged-out shell references a different file, and the session is what decides which one this is")
}

// Fetch runs one live refresh: the authenticated shell, the bundle it references, and the
// records that bundle declares.
//
// Every request goes through the client it is handed, which is the one that carries the
// configured proxy and counts the requests — the shell request is a navigation request and
// carries the cookie and no authorization header, which is the measured request class for it.
func Fetch(ctx context.Context, client *xweb.Client, sess session.Session, shellURL string) (Refresh, error) {
	if !sess.Valid() {
		return Refresh{}, fmt.Errorf("opsgen: refreshing needs the session cookie: %w", session.ErrNoSession)
	}

	shell, err := get(ctx, client, sess, shellURL, shellLimit)
	if err != nil {
		return Refresh{}, fmt.Errorf("opsgen: reading the shell: %w", err)
	}
	bundleURL, err := BundleURL(shell)
	if err != nil {
		return Refresh{}, err
	}

	bundle, err := get(ctx, client, session.Session{}, bundleURL, bundleLimit)
	if err != nil {
		return Refresh{}, fmt.Errorf("opsgen: downloading %s: %w", bundleURL, err)
	}
	records, err := Parse(bundle)
	if err != nil {
		return Refresh{}, fmt.Errorf("opsgen: %s: %w", bundleURL, err)
	}

	sum := sha256.Sum256(bundle)
	return Refresh{
		Records:   records,
		BundleURL: bundleURL,
		SHA256:    hex.EncodeToString(sum[:]),
		Bytes:     len(bundle),
		At:        time.Now().UTC(),
	}, nil
}

// get performs one bounded GET for the refresh path.
//
// The session is attached as the cookie and nothing else when one is given: no authorization
// header, because a navigation request that carries one is answered 401 with a zero-byte
// body (measured 2026-10-14). A nil session means a request that needs no credentials — the
// bundle on the CDN — and gets none.
func get(ctx context.Context, client *xweb.Client, sess session.Session, rawURL string, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	if _, err := url.Parse(rawURL); err != nil {
		return nil, fmt.Errorf("opsgen: %q is not a URL: %w", rawURL, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("user-agent", xweb.UserAgent)
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	if sess.Valid() {
		req.Header.Set("cookie", "auth_token="+sess.AuthToken.Reveal()+"; ct0="+sess.CTO.Reveal())
	}

	resp, err := client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rawURL, err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%s answered more than the %d bytes this tool will read", rawURL, limit)
	}

	// The shell is classified with the same table every other answer goes through, so that an
	// expired session is named as one instead of being read as a shell that simply had no
	// bundle reference in it.
	if outcome := xweb.Classify(resp.StatusCode, resp.Header, body); !outcome.Success() {
		return nil, fmt.Errorf("%s answered %d: %s — %s", rawURL, resp.StatusCode, outcome.Error(), outcome.Remedy())
	}
	return body, nil
}
