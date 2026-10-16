package xsource

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

// provenancePath is the record every fixture in testdata/ is read through. It is one row per
// fixture file, written beside the fixtures rather than in a planning document, because a
// fixture without a row is a body nobody can judge: it says which operation answered, which
// post it was about, what the answer cost, and what was redacted out of it before it landed
// here (SEC-01).
const provenancePath = "testdata/provenance.json"

// The fixture files, by the name inside testdata/ — which is the name a provenance row carries and
// the name a test reads a body by.
//
// recordedDetailPath and recordedTimelinePath are the two bodies a live answer carried: the
// recording is the evidence and what the tests read, there is no second copy of either, and each
// row's own digest is what pins it. The three under them are the bodies written for the shapes no
// recording carries — the provenance rows are where they say so.
const (
	recordedDetailPath   = "fixture-tweetdetail.json"
	recordedTimelinePath = "fixture-usertweets.json"
	shapesPath           = "fixture-tweetdetail-shapes.json"
	emptyPath            = "fixture-tweetdetail-empty.json"
	tombstonePath        = "fixture-tweetdetail-tombstone.json"
)

// provenanceRow is one fixture's record. The field order here is the file's own order: it reads
// as a record, and a reader of the JSON should not have to hunt for the digest.
type provenanceRow struct {
	File      string `json:"file"`
	Operation string `json:"operation"`
	QueryID   string `json:"queryId"`
	FocalID   string `json:"focalId"`

	CapturedAt string         `json:"capturedAt"`
	Bytes      int            `json:"bytes"`
	SHA256     string         `json:"sha256"`
	Source     string         `json:"source"`
	SourceFile string         `json:"sourceFile"`
	Redaction  string         `json:"redaction"`
	Notes      []string       `json:"notes"`
	Variables  map[string]any `json:"variables,omitempty"`
}

// provenanceFile is the on-disk document: one list under one key, so that a later phase adding
// a fixture adds a row rather than a schema.
type provenanceFile struct {
	Files []provenanceRow `json:"files"`
}

// provenance reads the record. A record that does not parse fails every test in this file: the
// alternative is a suite that passes because it read nothing.
func provenance(t *testing.T) provenanceFile {
	t.Helper()
	data, err := os.ReadFile(provenancePath)
	if err != nil {
		t.Fatalf("reading %s: %v", provenancePath, err)
	}
	var file provenanceFile
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("%s is not readable JSON: %v", provenancePath, err)
	}
	if len(file.Files) == 0 {
		t.Fatalf("%s records no fixture", provenancePath)
	}
	return file
}

// fixtureNames lists every fixture in testdata/, by the name inside the directory, sorted so that
// a failure names the same file on every run. The record itself is not a fixture.
func fixtureNames(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob("testdata/*.json")
	if err != nil {
		t.Fatalf("listing testdata/: %v", err)
	}
	names := make([]string, 0, len(paths))
	for _, path := range paths {
		name := filepath.Base(path)
		if name == filepath.Base(provenancePath) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatal("testdata/ holds no fixture, so every test in this file would pass for the wrong reason")
	}
	return names
}

// readFixture reads one fixture by the name inside testdata/.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading testdata/%s: %v", name, err)
	}
	return data
}

// digestOf is a file's sha256, in the hex the record carries.
func digestOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// rowFor returns the record's row for one fixture name, and fails when there is not exactly one:
// two rows for one file is as unreadable as none.
func rowFor(t *testing.T, file provenanceFile, name string) provenanceRow {
	t.Helper()
	var found []provenanceRow
	for _, row := range file.Files {
		if row.File == name {
			found = append(found, row)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s carries %d rows for %s, want exactly one", provenancePath, len(found), name)
	}
	return found[0]
}

// TestEveryFixtureIsProvenanced is the guard that makes the record complete rather than
// decorative: a fixture nobody recorded, a digest that no longer matches the file, or a row
// naming a file that is not there all fail here.
func TestEveryFixtureIsProvenanced(t *testing.T) {
	file := provenance(t)
	names := fixtureNames(t)
	onDisk := make(map[string]bool, len(names))

	for _, name := range names {
		data := readFixture(t, name)
		onDisk[name] = true
		row := rowFor(t, file, name)

		if row.Bytes != len(data) {
			t.Errorf("%s is %d bytes on disk and the record says %d: the record is about a different body", name, len(data), row.Bytes)
		}
		if want := digestOf(data); row.SHA256 != want {
			t.Errorf("%s hashes to %s and the record says %s: a fixture edited after it was recorded must not pass unnoticed", name, want, row.SHA256)
		}
		if row.Source != "recorded" && row.Source != "synthetic" {
			t.Errorf("%s is recorded as source %q, want recorded or synthetic: a body has to say whether a live answer carried it", name, row.Source)
		}
		if row.Operation == "" || row.CapturedAt == "" {
			t.Errorf("%s carries operation %q and capturedAt %q, want both", name, row.Operation, row.CapturedAt)
		}
		if row.Source == "synthetic" && len(row.Notes) == 0 {
			t.Errorf("%s is synthetic and its row explains nothing: a synthetic body has to say which shapes it carries and why", name)
		}
	}

	// Which bodies are recordings and which are hand-written is pinned by name: a body that
	// quietly changed sides would be a body this phase's evidence no longer covers, and a reader
	// deciding whether to trust a shape has to be able to see which side of that line it is on.
	for _, want := range []struct {
		name   string
		source string
	}{
		{recordedDetailPath, "recorded"},
		{recordedTimelinePath, "recorded"},
		{shapesPath, "synthetic"},
		{emptyPath, "synthetic"},
		{tombstonePath, "synthetic"},
	} {
		if row := rowFor(t, file, want.name); row.Source != want.source {
			t.Errorf("%s is recorded as source %q, want %q", want.name, row.Source, want.source)
		}
	}

	for _, row := range file.Files {
		if row.File == "" {
			t.Errorf("%s carries a row with no file name: a row that names nothing is a row nobody can check", provenancePath)
			continue
		}
		if !onDisk[row.File] {
			t.Errorf("%s records %s, which is not in testdata/", provenancePath, row.File)
		}
	}
}

// TestTheRecordedFixtureNamesItsSource asserts where this phase's ground truth came from: the
// body is a recording, and the operation it names is the operation the registry actually pins — a
// row naming an id the registry does not have would describe a request no build of this program can
// make. (It used to also compare the checked-in copy against a second, repository-root copy under
// `.planning/research/evidence/`; that copy was byte-identical to this one and is gone, so the
// recording and its own row are the two things left to disagree.)
func TestTheRecordedFixtureNamesItsSource(t *testing.T) {
	row := rowFor(t, provenance(t), recordedDetailPath)

	// The row must not claim a second copy: with the duplicate removed, a sourceFile naming one
	// would be a row pointing at a file the repository does not have.
	if row.SourceFile != "" {
		t.Errorf("%s records sourceFile %q, want none: the recording is this file", recordedDetailPath, row.SourceFile)
	}
	if row.Source != "recorded" {
		t.Errorf("%s is recorded as source %q, want recorded", recordedDetailPath, row.Source)
	}
	if row.Operation != "TweetDetail" {
		t.Errorf("%s records the operation %q, want TweetDetail", recordedDetailPath, row.Operation)
	}

	// The operation the row names is the one the registry pins. It is asserted here as well as over
	// every row (TestEveryProvenanceRowResolvesThroughTheRegistry) because this test is the one a
	// reviewer runs on its own: a row that disagreed with the table would describe a request no
	// build of this program can make, and the answer to that request is an empty body.
	op, err := volatile.Lookup(row.Operation)
	if err != nil {
		t.Fatalf("the registry has no %s: %v", row.Operation, err)
	}
	if op.QueryID != row.QueryID {
		t.Errorf("%s records %s as %q and the registry pins %q", recordedDetailPath, row.Operation, row.QueryID, op.QueryID)
	}

	// With the duplicate gone the row's digest and byte count are checked against the recording
	// itself; TestEveryFixtureIsProvenanced makes the same comparison for every fixture, and this
	// one keeps it beside the operation lookup a reviewer runs on its own.
	body := readFixture(t, recordedDetailPath)
	if want := digestOf(body); row.SHA256 != want {
		t.Errorf("%s records sha256 %s and the file hashes to %s", recordedDetailPath, row.SHA256, want)
	}
	if len(body) != row.Bytes {
		t.Errorf("%s is %d bytes and the record says %d", recordedDetailPath, len(body), row.Bytes)
	}
}

// TestEveryProvenanceRowResolvesThroughTheRegistry asserts that no row claims an id the registry
// does not pin. A query id is only valid until x.com ships a new bundle (XAPI-02): the registry is
// the one place it lives, and a record that disagreed with it would be a claim about a request
// that answers with an empty body and no message.
func TestEveryProvenanceRowResolvesThroughTheRegistry(t *testing.T) {
	for _, row := range provenance(t).Files {
		if row.Operation == "" || row.QueryID == "" {
			t.Errorf("%s names no operation or no id: a row that records neither is not a record", row.File)
			continue
		}
		op, err := volatile.Lookup(row.Operation)
		if err != nil {
			t.Errorf("%s names the operation %s, which the registry does not have: %v", row.File, row.Operation, err)
			continue
		}
		if op.QueryID != row.QueryID {
			t.Errorf("%s records %s as %q and the registry pins %q: the recorded id is not the id this build would send", row.File, row.Operation, row.QueryID, op.QueryID)
		}
	}
}

// rawDetail and the types under it are the recorded answer read with encoding/json instead of
// with ParseThread. They exist so the census and the decoder's tests can count what the body
// carries rather than what the decoder kept: a decoder that silently dropped half of the entries
// would otherwise certify its own arithmetic.
//
// Only the fields these tests read are declared. A struct that spelled the response out would be
// a second copy of the schema to keep in step, which is the thing this package exists to avoid.
type rawDetail struct {
	Data struct {
		Threaded struct {
			Instructions []rawInstruction `json:"instructions"`
		} `json:"threaded_conversation_with_injections_v2"`
	} `json:"data"`
}

type rawInstruction struct {
	Type    string     `json:"type"`
	Entries []rawEntry `json:"entries"`
}

type rawEntry struct {
	EntryID string     `json:"entryId"`
	Content rawContent `json:"content"`
}

type rawContent struct {
	Typename    string          `json:"__typename"`
	CursorType  string          `json:"cursorType"`
	Value       string          `json:"value"`
	ItemContent *rawItemContent `json:"itemContent"`
	Items       []rawModuleItem `json:"items"`
}

type rawModuleItem struct {
	EntryID string `json:"entryId"`
	Item    struct {
		ItemContent rawItemContent `json:"itemContent"`
	} `json:"item"`
}

type rawItemContent struct {
	Typename         string `json:"__typename"`
	TweetDisplayType string `json:"tweetDisplayType"`
	CursorType       string `json:"cursorType"`
	Value            string `json:"value"`
	TweetResults     struct {
		Result json.RawMessage `json:"result"`
	} `json:"tweet_results"`
}

// rawTweet is one result read far enough to see its identity, its conversation and the post it
// answers. The result is read through unwrap on the way to a page; this is only for tests that
// compare a decoded value against the value the body carried.
type rawTweet struct {
	Typename string `json:"__typename"`
	RestID   string `json:"rest_id"`
	Legacy   *struct {
		ReplyCount           int      `json:"reply_count"`
		ConversationIDStr    string   `json:"conversation_id_str"`
		InReplyToStatusIDStr string   `json:"in_reply_to_status_id_str"`
		WithheldInCountries  []string `json:"withheld_in_countries"`
	} `json:"legacy"`
	// Tweet is the inner tweet of a TweetWithVisibilityResults wrapper.
	Tweet *json.RawMessage `json:"tweet"`
	// QuotedStatusResult is the post this one quotes, nested the way the result carries it.
	QuotedStatusResult *struct {
		Result json.RawMessage `json:"result"`
	} `json:"quoted_status_result"`
}

// rawTweetOf reads one result node, unwrapping the visibility wrapper so that a test can see the
// post's own fields either way.
func rawTweetOf(t *testing.T, raw json.RawMessage) rawTweet {
	t.Helper()
	var node rawTweet
	if err := json.Unmarshal(raw, &node); err != nil {
		t.Fatalf("a result is not a tweet shape this test can read: %v", err)
	}
	if node.Typename == "TweetWithVisibilityResults" && node.Tweet != nil {
		var inner rawTweet
		if err := json.Unmarshal(*node.Tweet, &inner); err != nil {
			t.Fatalf("a visibility wrapper's tweet is not readable: %v", err)
		}
		inner.Typename = node.Typename
		return inner
	}
	return node
}

// detailEntries reads a TweetDetail body far enough to hand back the entries of its
// TimelineAddEntries instruction, without going through ParseThread: the tests that assert what
// the decoder kept have to compare it against what the body carried, and a test that asked the
// decoder would be asking it to mark its own homework.
func detailEntries(t *testing.T, data []byte) []rawEntry {
	t.Helper()
	var body rawDetail
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("the body is not readable JSON: %v", err)
	}
	if len(body.Data.Threaded.Instructions) == 0 {
		t.Fatal("the body carries no instructions: it is not a TweetDetail answer")
	}
	var entries []rawEntry
	for _, instruction := range body.Data.Threaded.Instructions {
		if instruction.Type == "TimelineAddEntries" {
			entries = append(entries, instruction.Entries...)
		}
	}
	return entries
}

// TestTheRecordedFixtureIsTheOneWeThinkItIs is the census, taken from the raw JSON: this is the
// fixture the phase's decoder is written against, and a body that is not this one fails here
// before any decoder test runs. The numbers are the measured ones — 43 entries (41
// conversationthread modules, the focal tweet entry and the bottom cursor), 41 module items, 5
// of them promotions, 42 tweet items in all — and a decoder change that moved them would have
// changed the recording rather than the reading of it.
func TestTheRecordedFixtureIsTheOneWeThinkItIs(t *testing.T) {
	var body rawDetail
	if err := json.Unmarshal(readFixture(t, recordedDetailPath), &body); err != nil {
		t.Fatalf("the recorded fixture is not readable JSON: %v", err)
	}
	if got := len(body.Data.Threaded.Instructions); got != 3 {
		t.Fatalf("the answer carries %d instructions, want the measured 3", got)
	}

	entries := detailEntries(t, readFixture(t, recordedDetailPath))
	kinds := map[string]int{}
	modules, moduleItems, promoted, tweetItems := 0, 0, 0, 0
	var focalReplies int
	for _, entry := range entries {
		switch {
		case strings.HasPrefix(entry.EntryID, "conversationthread-"):
			kinds["conversationthread-"]++
			modules++
		case strings.HasPrefix(entry.EntryID, "cursor-"):
			kinds["cursor-"]++
		case strings.HasPrefix(entry.EntryID, "tweet-"):
			kinds["tweet-"]++
		default:
			kinds["other"]++
		}
		if entry.Content.ItemContent != nil && entry.Content.ItemContent.Typename == "TimelineTweet" {
			tweetItems++
			if node := rawTweetOf(t, entry.Content.ItemContent.TweetResults.Result); node.Legacy != nil {
				focalReplies = node.Legacy.ReplyCount
			}
		}
		for _, item := range entry.Content.Items {
			if item.Item.ItemContent.Typename != "TimelineTweet" {
				continue
			}
			moduleItems++
			tweetItems++
			if strings.Contains(item.EntryID, "-promoted-tweet-") {
				promoted++
			}
		}
	}

	switch {
	case len(entries) != 43:
		t.Errorf("the answer carries %d entries, want the measured 43", len(entries))
	case modules != 41:
		t.Errorf("the answer carries %d conversationthread modules, want the measured 41", modules)
	case kinds["tweet-"] != 1:
		t.Errorf("the answer carries %d tweet entries, want the measured 1 (the focal post)", kinds["tweet-"])
	case kinds["cursor-"] != 1:
		t.Errorf("the answer carries %d cursor entries, want the measured 1 (the bottom cursor)", kinds["cursor-"])
	case kinds["other"] != 0:
		t.Errorf("the answer carries %d entries of no kind this census knows", kinds["other"])
	case moduleItems != 41:
		t.Errorf("the modules carry %d items, want the measured 41", moduleItems)
	case promoted != 5:
		t.Errorf("the answer carries %d promoted items, want the measured 5", promoted)
	case tweetItems != 42:
		t.Errorf("the answer carries %d tweet items, want the measured 42", tweetItems)
	case focalReplies <= 0:
		t.Errorf("the focal post reports reply_count = %d, want above zero: the third partial-thread condition reads it as evidence", focalReplies)
	}
}

// TestNoFixtureCarriesViewerIdentity is the fixture half of SEC-01. Two arms, for two different
// failures: the always-on arm asserts that no fixture holds a session-shaped key or a credential
// value, and the session arm — which needs a session on disk — asserts that the viewer's own
// account id and handle appear nowhere in testdata/, using the same identity the capture harness
// rewrites out of a fresh body.
func TestNoFixtureCarriesViewerIdentity(t *testing.T) {
	names := fixtureNames(t)

	for _, name := range names {
		data := readFixture(t, name)
		for _, key := range []string{`"auth_token"`, `"ct0"`, "auth_token=", "ct0="} {
			if bytes.Contains(data, []byte(key)) {
				t.Errorf("testdata/%s carries a session-shaped key (%s): no cookie, token or ct0 value may appear in a fixture (SEC-01)", name, key)
			}
		}
	}
	for _, row := range provenance(t).Files {
		if strings.TrimSpace(row.Redaction) == "" {
			t.Errorf("%s carries no redaction sentence: every fixture has to say what was rewritten out of it", row.File)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("skipping the scan against the viewer's own identity: no configuration to find a session through (%v); the arms above ran regardless", err)
	}
	sess, err := session.Load(cfg.SessionPath)
	if err != nil {
		t.Skipf("skipping the scan against the viewer's own identity: no usable session at %s (%v); the arms above ran regardless", cfg.SessionPath, err)
	}

	// The session's own two values never reach a fixture, whatever the identity scan below can
	// and cannot name.
	for _, secret := range []session.Secret{sess.AuthToken, sess.CTO} {
		value := secret.Reveal()
		if value == "" {
			continue
		}
		for _, name := range names {
			if bytes.Contains(readFixture(t, name), []byte(value)) {
				t.Errorf("testdata/%s carries the session's own credential value: the cookie never enters a fixture (SEC-01)", name)
			}
		}
	}

	accountID, handle := viewerIdentity(t, cfg.SessionPath)
	if accountID == "" && handle == "" {
		t.Skipf("skipping the scan for the viewer's own account id and handle: the session at %s names neither, and neither X_TERMINAL_VIEWER_ID nor X_TERMINAL_VIEWER_HANDLE is set; the arms above ran regardless", cfg.SessionPath)
	}
	for _, name := range names {
		data := readFixture(t, name)
		if accountID != "" && bytes.Contains(data, []byte(accountID)) {
			t.Errorf("testdata/%s carries the viewer's own account id: the capture harness rewrites it to a placeholder before writing (SEC-01)", name)
		}
		if handle == "" {
			continue
		}
		// The handle is looked for as a JSON value rather than as a bare run of characters: a
		// short handle is a substring of a hundred other words, and a scan that failed on those
		// is a scan a reader learns to ignore.
		for _, needle := range []string{`"screen_name":"` + handle + `"`, `"screen_name": "` + handle + `"`, `"handle":"` + handle + `"`, `"handle": "` + handle + `"`} {
			if bytes.Contains(data, []byte(needle)) {
				t.Errorf("testdata/%s carries the viewer's own handle: the capture harness rewrites it to a placeholder before writing (SEC-01)", name)
			}
		}
	}
}
