package xsource

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

// captureGate is the environment variable that turns this file's one test from a skip into a
// live request. It is a gate rather than a subcommand because a recorded body has to come out of
// the program's own read path: the limiter, the request floor, the process budget and the
// measured classifier all apply to a test binary's request exactly as they apply to a read the
// program makes, and a body captured by anything else would be evidence of a different request.
const captureGate = "X_TERMINAL_CAPTURE"

// captureDir is the only directory the harness writes to, and captureFile is the body it writes.
// The target is a constant rather than an argument so that recording a body is something an
// operator does on purpose, with the name in front of them, and never a side effect of a test run.
const (
	captureDir  = "testdata"
	captureFile = "fixture-tweetdetail.json"
)

// captureDate is the format a provenance row's capturedAt is written in.
const captureDate = "2006-01-02"

// The two placeholders a captured body's viewer identity is rewritten to. They are fixed strings
// rather than something derived from the identity: a placeholder made of the real value's shape
// would be a placeholder a reader could mistake for one.
const (
	redactedAccountID = "0"
	redactedHandle    = "redacted"
)

// The two variables that name the identity a captured body is cleaned of, for a session whose
// file does not carry it. internal/session reads the two cookies and deliberately nothing else
// (SEC-01), so the account id comes from the session file's own twid line when it has one, and
// anything else has to be said out loud.
const (
	viewerIDVar     = "X_TERMINAL_VIEWER_ID"
	viewerHandleVar = "X_TERMINAL_VIEWER_HANDLE"
)

// counted is the harness's own count of what it issued. One request is the whole capture, and a
// count is what makes "one" an assertion rather than an intention: a second request would be a
// second body nobody recorded.
type counted struct {
	client GraphQLer
	calls  int
}

func (c *counted) GraphQL(ctx context.Context, opName string, variables map[string]any) ([]byte, error) {
	c.calls++
	return c.client.GraphQL(ctx, opName, variables)
}

// TestCaptureTweetDetail records one TweetDetail answer into testdata/ and rewrites that file's
// provenance row. It is skipped unless X_TERMINAL_CAPTURE=1, so a normal run never opens a socket.
//
// It fails rather than writes when it cannot tell whose session it read with: a body carrying the
// viewer's own identity, next to a row claiming it was cleaned, is worse than no body at all.
func TestCaptureTweetDetail(t *testing.T) {
	if os.Getenv(captureGate) != "1" {
		t.Skipf("%s is not 1: this test issues one live TweetDetail request and rewrites %s/%s, so it runs only when an operator asks for it", captureGate, captureDir, captureFile)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("the configuration this host reads with: %v", err)
	}
	for _, warning := range cfg.Warnings {
		t.Logf("config warning: %s", warning)
	}
	// The operation table is read here the way internal/cli reads it, so that a pins file this
	// run cannot parse is visible rather than silently absent.
	if _, err := volatile.Load(cfg.PinsPath()); err != nil {
		t.Fatalf("the operation table: %v", err)
	}
	sess, err := session.Load(cfg.SessionPath)
	if err != nil {
		t.Fatalf("the session this capture reads with: %v", err)
	}
	for _, warning := range sess.Warnings {
		t.Logf("session warning: %s", warning)
	}
	client, err := xweb.NewClient(cfg)
	if err != nil {
		t.Fatalf("the transport: %v", err)
	}

	// The record is read before anything is written, and so is the row: a body that landed
	// without a row would be a body nobody could judge, which is exactly what this phase's
	// fixtures exist to prevent.
	record := provenance(t)
	row := rowFor(t, record, captureFile)
	if row.FocalID == "" {
		t.Fatalf("%s records no focalId for %s: there is no post to ask for", provenancePath, captureFile)
	}

	counter := &counted{client: client.WithSession(sess)}
	raw, err := counter.GraphQL(context.Background(), opTweetDetail, map[string]any{
		"focalTweetId": row.FocalID,
		"cursor":       "",
	})
	if err != nil {
		// The error carries the classifier's own sentence. The cookie is not in it: xweb never
		// puts a credential in an error.
		t.Fatalf("the one request this capture makes: %v", err)
	}
	if counter.calls != 1 {
		t.Fatalf("the capture issued %d requests, want exactly 1: this request is the whole capture", counter.calls)
	}
	if len(raw) == 0 {
		t.Fatal("the answer is empty: an empty body is not a body to check in")
	}

	accountID, handle := viewerIdentity(t, cfg.SessionPath)
	if accountID == "" && handle == "" {
		t.Fatalf("this harness cannot tell whose session it read with, and it will not write a body it cannot clean: internal/session deliberately carries the two cookies and no identity, so either put a twid=<account id> line in the session file (which is where x.com's own cookie jar keeps it) or set %s, and %s for the handle", viewerIDVar, viewerHandleVar)
	}
	body, redaction := redactBody(raw, accountID, handle)

	// The write stays inside testdata/: the file name is a constant, and the join is checked
	// rather than trusted.
	path := filepath.Join(captureDir, captureFile)
	if filepath.Dir(path) != captureDir || filepath.Base(path) != captureFile {
		t.Fatalf("%s is not inside %s/: this harness writes there and nowhere else", path, captureDir)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}

	for i := range record.Files {
		if record.Files[i].File != captureFile {
			continue
		}
		record.Files[i].Bytes = len(body)
		record.Files[i].SHA256 = digestOf(body)
		record.Files[i].CapturedAt = now().UTC().Format(captureDate)
		record.Files[i].Redaction = redaction
		record.Files[i].Source = "recorded"
		// A freshly captured body is not the evidence file's copy any more, so the row stops
		// claiming it is one. Copying the new body into .planning/research/evidence/ and pointing
		// sourceFile at it again is the operator's step, and the source test fails until it
		// happens — which is the honest state rather than a row that overstates its provenance.
		record.Files[i].SourceFile = ""
	}
	writeProvenance(t, record)
	t.Logf("recorded %d bytes into %s and rewrote its provenance row: %s", len(body), path, redaction)
	if row.SourceFile != "" {
		t.Logf("the row no longer names %s as this body's source: copy the new body there and point sourceFile at it, or the source test will keep failing", row.SourceFile)
	}
}

const (
	captureBookmarksFile = "fixture-bookmarks.json"
	captureLikesFile     = "fixture-likes.json"
	captureHomeFile      = "fixture-home.json"
)

// TestCaptureBookmarks records one Bookmarks answer into testdata/ and rewrites that file's
// provenance row. It is skipped unless X_TERMINAL_CAPTURE=1, so a normal run never opens a socket.
func TestCaptureBookmarks(t *testing.T) {
	if os.Getenv(captureGate) != "1" {
		t.Skipf("%s is not 1: this test issues one live Bookmarks request and rewrites %s/%s, so it runs only when an operator asks for it", captureGate, captureDir, captureBookmarksFile)
	}
	captureOne(t, opBookmarks, captureBookmarksFile, map[string]any{
		"count":  pageCount,
		"cursor": "",
	}, true)
}

// TestCaptureLikes records one Likes answer into testdata/ and rewrites that file's provenance
// row. It is skipped unless X_TERMINAL_CAPTURE=1. The measured outcome — a list, a named refusal,
// or no operation id — is written into the provenance notes; the port's contract is the same
// either way.
func TestCaptureLikes(t *testing.T) {
	if os.Getenv(captureGate) != "1" {
		t.Skipf("%s is not 1: this test issues one live Likes request and rewrites %s/%s, so it runs only when an operator asks for it", captureGate, captureDir, captureLikesFile)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("the configuration this host reads with: %v", err)
	}
	accountID, _ := viewerIdentity(t, cfg.SessionPath)
	if accountID == "" {
		t.Fatalf("Likes needs the viewer's account id: set %s or put a twid line in the session file", viewerIDVar)
	}
	captureOne(t, opLikes, captureLikesFile, map[string]any{
		"userId":                 accountID,
		"count":                  pageCount,
		"cursor":                 "",
		"includePromotedContent": false,
	}, true)
}

// TestCaptureHome records one HomeLatestTimeline answer into testdata/ and rewrites that file's
// provenance row. It is skipped unless X_TERMINAL_CAPTURE=1, so a normal run never opens a socket.
//
// It exists because the Following feed — the program's default surface — had no body to be read
// against: fixture-usertweets.json and the bookmarks/likes recordings are all user timelines or
// saved lists, and *none* of them carries the home timelines' envelope
// (data.home.home_timeline_urt.instructions). timelineFrom therefore gained a fourth candidate
// path that no body had ever been read against, and measured 2026-10-15 the live upstream answered
// bare x-terminal with its instructions at exactly that path while the reader refused it. A
// fixture is what makes the arm falsifiable: without one, a drift in this envelope fails only in
// a pane.
//
// The variables are the adapter's own (the four internal/xsource/home.go sends), so the recorded
// body is the answer to the request this build actually makes. HomeTimeline (For-you) was measured
// answering under the same path, so this one recording covers both arms.
//
// Post bodies are redacted the way the two saved-list recordings are (captureOne's redactPosts):
// a home timeline is read through one account's own Following line, so the checked-in body must not
// carry the natural-language text of posts that account happens to follow — some of which are not
// public at all. The entries, ids, authors, dates, cursors and attachment refs are the recording's
// subject and are left exact.
func TestCaptureHome(t *testing.T) {
	if os.Getenv(captureGate) != "1" {
		t.Skipf("%s is not 1: this test issues one live HomeLatestTimeline request and rewrites %s/%s, so it runs only when an operator asks for it", captureGate, captureDir, captureHomeFile)
	}
	captureOne(t, opHomeLatest, captureHomeFile, map[string]any{
		"count":                  pageCount,
		"cursor":                 "",
		"seenTweetIds":           []string{},
		"includePromotedContent": false,
	}, true)
}

// captureOne is the shared gated capture: one GraphQL call, identity and post-body redaction,
// provenance rewrite. It is named for what it does rather than for the lists that first used it —
// the home timelines are read through it too, and they are not saved lists.
//
// When the call fails, the fixture records the classified outcome's shape (no viewer identity) and
// the notes name the measured refusal.
func captureOne(t *testing.T, opName, file string, vars map[string]any, redactPosts bool) {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("the configuration this host reads with: %v", err)
	}
	for _, warning := range cfg.Warnings {
		t.Logf("config warning: %s", warning)
	}
	if _, err := volatile.Load(cfg.PinsPath()); err != nil {
		t.Fatalf("the operation table: %v", err)
	}
	sess, err := session.Load(cfg.SessionPath)
	if err != nil {
		t.Fatalf("the session this capture reads with: %v", err)
	}
	for _, warning := range sess.Warnings {
		t.Logf("session warning: %s", warning)
	}
	client, err := xweb.NewClient(cfg)
	if err != nil {
		t.Fatalf("the transport: %v", err)
	}

	record := provenance(t)
	_ = rowFor(t, record, file) // the row must already exist so a body without a row cannot land

	counter := &counted{client: client.WithSession(sess)}
	raw, callErr := counter.GraphQL(context.Background(), opName, vars)
	if counter.calls != 1 {
		t.Fatalf("the capture issued %d requests, want exactly 1: this request is the whole capture", counter.calls)
	}

	accountID, handle := viewerIdentity(t, cfg.SessionPath)
	if accountID == "" && handle == "" {
		t.Fatalf("this harness cannot tell whose session it read with, and it will not write a body it cannot clean: set %s / %s or a twid/screen_name line in the session file", viewerIDVar, viewerHandleVar)
	}

	var body []byte
	var redaction string
	var notes []string
	var source string
	if callErr != nil {
		outcome, detail, status := classifyCaptureError(callErr)
		body = refusalFixture(opName, outcome, detail, status)
		redaction = "measured refusal: the answer carried no viewer identity into the fixture; the cookie never enters a fixture (SEC-01)"
		source = "synthetic"
		notes = []string{
			"measured outcome: the server refused the " + opName + " request as " + outcome,
			"status: " + strconv.Itoa(status),
			ifNonEmpty("detail: ", detail),
			"variables: " + formatVars(vars),
			"the fixture records the refusal shape so Saved returns the classifier's own named error; nothing above the port has to know which answer it was",
		}
	} else {
		if len(raw) == 0 {
			t.Fatal("the answer is empty: an empty body is not a body to check in")
		}
		body, redaction = redactBody(raw, accountID, handle)
		if redactPosts {
			var n int
			body, n = redactPostTexts(body)
			redaction += "; " + strconv.Itoa(n) + " post bodies replaced with synthetic text of the same lengths"
		}
		source = "recorded"
		entryKinds, bottom := summariseTimeline(body)
		notes = []string{
			"measured outcome: the server answered " + opName + " with a list",
			"entry kinds: " + entryKinds,
			"bottom cursor present: " + strconv.FormatBool(bottom != ""),
			"variables: " + formatVars(vars),
		}
	}

	path := filepath.Join(captureDir, file)
	if filepath.Dir(path) != captureDir || filepath.Base(path) != file {
		t.Fatalf("%s is not inside %s/: this harness writes there and nowhere else", path, captureDir)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}

	op, err := volatile.Lookup(opName)
	queryID := ""
	if err == nil {
		queryID = op.QueryID
	}
	// Provenance variables must not carry the viewer's account id: rewrite userId to the
	// placeholder before the row is written (T-06-05).
	provVars := map[string]any{}
	for k, v := range vars {
		provVars[k] = v
	}
	if _, ok := provVars["userId"]; ok {
		provVars["userId"] = redactedAccountID
	}
	for i := range record.Files {
		if record.Files[i].File != file {
			continue
		}
		record.Files[i].Operation = opName
		record.Files[i].QueryID = queryID
		record.Files[i].Bytes = len(body)
		record.Files[i].SHA256 = digestOf(body)
		record.Files[i].CapturedAt = now().UTC().Format(captureDate)
		record.Files[i].Redaction = redaction
		record.Files[i].Source = source
		record.Files[i].SourceFile = ""
		record.Files[i].Notes = notes
		record.Files[i].Variables = provVars
	}
	writeProvenance(t, record)
	t.Logf("recorded %d bytes into %s (%s): %s", len(body), path, source, redaction)
}

func ifNonEmpty(prefix, s string) string {
	if s == "" {
		return prefix + "(empty)"
	}
	return prefix + s
}

func formatVars(vars map[string]any) string {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	// stable order so a rewrite does not churn the notes
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		if k == "userId" {
			parts = append(parts, "userId=<viewer>")
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%v", k, vars[k]))
	}
	return strings.Join(parts, ", ")
}

func classifyCaptureError(err error) (outcome, detail string, status int) {
	var api *xweb.APIError
	if errors.As(err, &api) {
		return string(api.Outcome), api.Detail, api.Status
	}
	if errors.Is(err, volatile.ErrUnknownOperation) {
		return "unknown operation (registry)", err.Error(), 0
	}
	return "unclassified", err.Error(), 0
}

func refusalFixture(opName, outcome, detail string, status int) []byte {
	body := map[string]any{
		"data": nil,
		"errors": []map[string]any{{
			"message":    "measured " + opName + " refusal: " + outcome,
			"extensions": map[string]any{"code": status, "detail": detail},
		}},
	}
	raw, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return []byte(`{"data":null,"errors":[{"message":"measured refusal"}]}`)
	}
	return append(raw, '\n')
}

// redactPostTexts replaces every legacy.full_text and note_tweet text with synthetic ASCII of
// the same length, so a checked-in fixture cannot carry a real saved post's body. Returns the
// rewritten body and how many strings were replaced.
func redactPostTexts(body []byte) ([]byte, int) {
	replaced := 0
	out := body
	for _, key := range []string{`"full_text"`, `"text"`} {
		out, replaced = replaceJSONStringValues(out, key, replaced)
	}
	return out, replaced
}

func replaceJSONStringValues(body []byte, key string, already int) ([]byte, int) {
	needle := []byte(key + `:`)
	needleSpaced := []byte(key + `: `)
	replaced := already
	search := body
	var out bytes.Buffer
	for {
		i := bytes.Index(search, needleSpaced)
		alt := bytes.Index(search, needle)
		useSpaced := true
		if i < 0 || (alt >= 0 && alt < i) {
			i = alt
			useSpaced = false
		}
		if i < 0 {
			out.Write(search)
			return out.Bytes(), replaced
		}
		out.Write(search[:i])
		if useSpaced {
			out.Write(needleSpaced)
			search = search[i+len(needleSpaced):]
		} else {
			out.Write(needle)
			search = search[i+len(needle):]
		}
		search = bytes.TrimLeft(search, " \t\n\r")
		if len(search) == 0 || search[0] != '"' {
			continue
		}
		end, ok := jsonStringEnd(search)
		if !ok {
			out.Write(search)
			return out.Bytes(), replaced
		}
		value := search[1 : end-1]
		// Only rewrite plausible post bodies (short keys like "text":"en" for lang stay).
		if len(value) >= 8 || bytes.Contains(value, []byte(" ")) {
			out.WriteByte('"')
			out.Write(bytes.Repeat([]byte("x"), len(value)))
			out.WriteByte('"')
			replaced++
		} else {
			out.Write(search[:end])
		}
		search = search[end:]
	}
}

func jsonStringEnd(s []byte) (int, bool) {
	if len(s) == 0 || s[0] != '"' {
		return 0, false
	}
	escaped := false
	for i := 1; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if c == '"' {
			return i + 1, true
		}
	}
	return 0, false
}

func summariseTimeline(body []byte) (kinds string, bottom string) {
	page, err := ParseTimeline(body)
	if err != nil {
		return "unparseable: " + err.Error(), ""
	}
	return fmt.Sprintf("TimelineTweet posts=%d requested=%d", len(page.Posts), page.Requested), page.BottomCursor
}

// writeProvenance rewrites the record with its own formatting: two-space indent, one field per
// line, a trailing newline. A rewrite by the harness and an edit by hand produce the same file.
func writeProvenance(t *testing.T, record provenanceFile) {
	t.Helper()
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatalf("writing the record: %v", err)
	}
	if err := os.WriteFile(provenancePath, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("writing %s: %v", provenancePath, err)
	}
}

// redactBody rewrites the viewer's own identity to the two placeholders and reports, in one
// sentence for the record's row, what it rewrote.
//
// The account id is rewritten wherever its digits appear: an id is a long run of digits and
// nothing else in a response looks like one. The handle is rewritten only where it is a whole
// JSON string value — `"lzd121"` — because a handle can be one character long and a bare
// replacement would rewrite that character everywhere in the body: a body destroyed rather than a
// body redacted. A handle inside a post's text is left alone, because that is public content and
// this rewrite's subject is the session's own identity.
func redactBody(body []byte, accountID, handle string) ([]byte, string) {
	ids, handles := 0, 0
	if accountID != "" {
		ids = bytes.Count(body, []byte(accountID))
		body = bytes.ReplaceAll(body, []byte(accountID), []byte(redactedAccountID))
	}
	if handle != "" {
		quoted := `"` + handle + `"`
		handles = bytes.Count(body, []byte(quoted))
		body = bytes.ReplaceAll(body, []byte(quoted), []byte(`"`+redactedHandle+`"`))
	}
	return body, redactionSentence(ids, handles)
}

// redactionSentence is the row's redaction string: what was rewritten, and the standing fact that
// no fixture may carry a cookie.
func redactionSentence(ids, handles int) string {
	const cookie = "; the cookie never enters a fixture (SEC-01)"
	switch {
	case ids == 0 && handles == 0:
		return "nothing rewritten: the answer carried neither the viewer's own account id nor their handle" + cookie
	case handles == 0:
		return "the viewer's account id was rewritten to " + redactedAccountID + " (" + strconv.Itoa(ids) + " occurrences); their handle was absent from the answer" + cookie
	case ids == 0:
		return "the viewer's handle was rewritten to " + redactedHandle + " (" + strconv.Itoa(handles) + " occurrences); their account id was absent from the answer" + cookie
	default:
		return "the viewer's account id was rewritten to " + redactedAccountID + " (" + strconv.Itoa(ids) + " occurrences) and their handle to " + redactedHandle + " (" + strconv.Itoa(handles) + " occurrences)" + cookie
	}
}

// viewerIdentity reads who the loaded session belongs to: the account id and the handle that must
// never reach a checked-in fixture.
//
// It reads the session file's own lines — the twid line x.com's cookie jar writes, and any
// id/screen_name line an operator added — and the two environment variables win over them. It
// never returns, logs or reasons about the cookie: the two credential keys are skipped by name,
// and a value that cannot be read as an identity is dropped rather than guessed at.
func viewerIdentity(t *testing.T, sessionPath string) (accountID, handle string) {
	t.Helper()
	accountID, handle = os.Getenv(viewerIDVar), os.Getenv(viewerHandleVar)
	if sessionPath == "" {
		return accountID, handle
	}
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		// A session supplied entirely through the environment has no file to read an identity out
		// of, and that is not a failure: the variables above are what name one then.
		return accountID, handle
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = identityValue(strings.TrimSpace(value))
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "twid":
			if accountID == "" {
				accountID = accountIDOfTwid(value)
			}
		case "id", "account_id", "user_id":
			if accountID == "" {
				accountID = value
			}
		case "screen_name", "handle", "username":
			if handle == "" {
				handle = value
			}
		}
	}
	return accountID, handle
}

// identityValue strips the quotes and a ` #...` tail from a session file value, the way
// internal/session does: a line copied out of a shell script still names what it names.
func identityValue(value string) string {
	if i := strings.Index(value, " #"); i >= 0 {
		value = value[:i]
	}
	value = strings.TrimSpace(value)
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return value[1 : len(value)-1]
	}
	return value
}

// accountIDOfTwid reads the account id out of the twid cookie's value, which is the URL-encoded
// string `u=<id>` in every session this project has seen. A value that is not that shape yields
// nothing: an id guessed at is a body cleaned of the wrong thing.
func accountIDOfTwid(value string) string {
	if i := strings.Index(strings.ToLower(value), "%3d"); i >= 0 {
		return digitsOnly(value[i+len("%3d"):])
	}
	if i := strings.Index(value, "="); i >= 0 {
		return digitsOnly(value[i+1:])
	}
	return digitsOnly(value)
}

// digitsOnly returns the value when it is nothing but digits, and an empty string otherwise.
func digitsOnly(value string) string {
	for _, r := range value {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return value
}

// TestRedactionRewritesTheViewersIdentity is the harness's own mechanism, tested without a socket:
// the account id is rewritten wherever it appears, the handle only where it is a whole JSON string
// value, an identity that is absent is reported as absent, and a one-character handle does not eat
// the body around it.
func TestRedactionRewritesTheViewersIdentity(t *testing.T) {
	const id = "9876543210987654321"
	raw := []byte(`{"result":{"rest_id":"9876543210987654321","legacy":{"full_text":"the viewer says 9876543210987654321 is a long number"}},"core":{"user_results":{"result":{"core":{"screen_name":"lzd121"}}}}}`)

	body, sentence := redactBody(raw, id, "lzd121")

	if bytes.Contains(body, []byte(id)) {
		t.Errorf("the viewer's account id survives in %s", body)
	}
	if !bytes.Contains(body, []byte(`"rest_id":"`+redactedAccountID+`"`)) {
		t.Errorf("the account id was not rewritten in place: %s", body)
	}
	// The id appears inside a post's text as well, and a redaction that rewrote only JSON values
	// would leave the viewer's identity inside the corpus it captured.
	if !bytes.Contains(body, []byte("the viewer says "+redactedAccountID)) {
		t.Errorf("the id inside the public text was not rewritten: %s", body)
	}
	if !bytes.Contains(body, []byte(`"screen_name":"`+redactedHandle+`"`)) {
		t.Errorf("the handle was not rewritten in place: %s", body)
	}
	if !strings.Contains(sentence, "cookie never enters a fixture") {
		t.Errorf("the row's redaction sentence is %q, want the sentence that says no cookie enters a fixture", sentence)
	}

	// The arms: a body carrying neither identity says so rather than claiming a rewrite, and a
	// one-character handle — which is a substring of everything — is left to the JSON value rule.
	if _, absent := redactBody([]byte(`{"a":"b"}`), "", "lzd121"); !strings.Contains(absent, "nothing rewritten") {
		t.Errorf("a body carrying neither identity reports %q, want a sentence naming absence", absent)
	}
	short, _ := redactBody([]byte(`{"text":"x marks the spot"}`), "", "x")
	if string(short) != `{"text":"x marks the spot"}` {
		t.Errorf("a one-character handle rewrote the body into %s, want the body unchanged: a bare handle is not a JSON value", short)
	}
}

// TestViewerIdentityReadsTheSessionFilesOwnIdentity is the other half of the harness, also without
// a socket: the account id comes out of a twid line (URL-encoded, the way the cookie carries it),
// a screen_name line names the handle, and the credential keys contribute nothing.
func TestViewerIdentityReadsTheSessionFilesOwnIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session")
	fake := "auth_token=not-a-real-token\nct0=not-a-real-csrf\n# a comment\ntwid=u%3D9876543210987654321\nscreen_name=lzd121\n"
	if err := os.WriteFile(path, []byte(fake), 0o600); err != nil {
		t.Fatal(err)
	}

	accountID, handle := viewerIdentity(t, path)
	if accountID != "9876543210987654321" {
		t.Errorf("the account id came out as %q, want the one the twid line names", accountID)
	}
	if handle != "lzd121" {
		t.Errorf("the handle came out as %q, want the one the screen_name line names", handle)
	}

	// A file that names no identity yields none, rather than the credential values.
	if err := os.WriteFile(path, []byte("auth_token=not-a-real-token\nct0=not-a-real-csrf\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gotID, gotHandle := viewerIdentity(t, path)
	if gotID != "" || gotHandle != "" {
		t.Errorf("a session file with no identity yielded (%q, %q), want nothing at all", gotID, gotHandle)
	}

	// A twid that is not an id is not an id: it yields nothing rather than a guess.
	if got := accountIDOfTwid("u%3Dnot-a-number"); got != "" {
		t.Errorf("accountIDOfTwid read %q out of a value that is not an id", got)
	}
	if got := accountIDOfTwid("u%3D1234567890"); got != "1234567890" {
		t.Errorf("accountIDOfTwid read %q out of u%%3D1234567890", got)
	}
}
