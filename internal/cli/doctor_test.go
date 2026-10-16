package cli

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/netcheck"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/store"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// The two flags this package used to stub out are wired, and both of them leave the network alone:
// a pin writes one line into the user's own pins file, and a refresh that cannot read a session
// says so before it spends a request. What these tests protect is exactly that — either flag
// quietly falling through into the connectivity check would spend requests on the account and
// still report nothing about operation ids.
func TestDoctorPinsAnOperationWithoutTouchingTheNetwork(t *testing.T) {
	hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))

	code, stdout, stderr := run(t, Doctor, "--pin", "HomeTimeline", "1u0Wlkw6Ru1NwBUD-pDiww")
	if code != ExitOK {
		t.Fatalf("doctor --pin exited %d (%s), want %d", code, stderr, ExitOK)
	}
	if !strings.Contains(stdout, "HomeTimeline") || !strings.Contains(stdout, "1u0Wlkw6Ru1NwBUD-pDiww") {
		t.Errorf("the pin was written without saying so: %q", stdout)
	}

	path := filepath.Join(os.Getenv("XDG_DATA_HOME"), "x-terminal", "pins.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no pins file at %s: %v", path, err)
	}
	var pins struct {
		Operations map[string]struct {
			QueryID string `json:"queryId"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(data, &pins); err != nil {
		t.Fatalf("the pins file does not parse: %v", err)
	}
	if pins.Operations["HomeTimeline"].QueryID != "1u0Wlkw6Ru1NwBUD-pDiww" {
		t.Errorf("the pins file holds %v, want HomeTimeline pinned to the id that was given", pins.Operations)
	}
}

func TestDoctorRefusesAPinItCannotWrite(t *testing.T) {
	hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))

	for _, args := range [][]string{
		{"--pin", "Home Timeline", "1u0Wlkw6Ru1NwBUD-pDiww"},
		{"--pin", "HomeTimeline", "not an id"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			code, stdout, stderr := run(t, Doctor, args...)
			if code != ExitError {
				t.Fatalf("doctor %v exited %d, want %d", args, code, ExitError)
			}
			if stdout != "" {
				t.Errorf("a refused pin wrote to stdout: %q", stdout)
			}
			if !strings.Contains(stderr, "pin") {
				t.Errorf("the refusal does not say what was refused: %q", stderr)
			}
			if _, err := os.Stat(filepath.Join(os.Getenv("XDG_DATA_HOME"), "x-terminal", "pins.json")); err == nil {
				t.Error("a refused pin wrote a pins file anyway")
			}
		})
	}
}

func TestDoctorRefreshesNothingWithoutASession(t *testing.T) {
	hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))

	code, stdout, stderr := run(t, Doctor, "--refresh-ids")
	if code != ExitError {
		t.Fatalf("doctor --refresh-ids without a session exited %d, want %d", code, ExitError)
	}
	if stdout != "" {
		t.Errorf("a refresh that could not run wrote to stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "session") {
		t.Errorf("the refusal does not say a session is what is missing: %q", stderr)
	}
}

func TestDoctorRefusesToPrintTheNoticeIntoAJSONReport(t *testing.T) {
	hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))

	code, stdout, stderr := run(t, Doctor, "--show-notice", "--json")
	if code != ExitUsage {
		t.Fatalf("doctor --show-notice --json exited %d, want %d", code, ExitUsage)
	}
	if stdout != "" {
		t.Errorf("the rejected combination wrote to stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "usage: x-terminal") {
		t.Errorf("the rejection does not carry the usage: %q", stderr)
	}
}

// TestDoctorShowNoticeRunsNoCheck is the account's request budget's arm. Asking for the statement
// used to run the whole connectivity check beside it, which on this project costs live requests
// against the account, so the transport is the thing to watch: the configured proxy is the first
// hop every request of the client a check would build takes, and here it is a listener that records
// connections instead of serving them. A run that quietly performed the check either dials it —
// and the counter below says so — or leaves its report on stdout; there is nowhere to hide.
func TestDoctorShowNoticeRunsNoCheck(t *testing.T) {
	hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))

	proxy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening for the transport the notice path must not use: %v", err)
	}
	t.Cleanup(func() { proxy.Close() })
	var dialed atomic.Int64
	go func() {
		for {
			conn, err := proxy.Accept()
			if err != nil {
				return
			}
			dialed.Add(1)
			conn.Close()
		}
	}()
	t.Setenv("X_TERMINAL_PROXY", "http://"+proxy.Addr().String())

	code, stdout, stderr := run(t, Doctor, "--show-notice")
	if used := dialed.Load(); used != 0 {
		t.Errorf("doctor --show-notice dialed the configured proxy %d time(s): the statement is not a check, and the check spends requests on the account", used)
	}
	if code != ExitOK {
		t.Fatalf("doctor --show-notice exited %d (%s), want %d", code, stderr, ExitOK)
	}
	if stdout != noticeText {
		t.Errorf("doctor --show-notice printed %q, want the statement and nothing else", stdout)
	}
	if stderr != "" {
		t.Errorf("doctor --show-notice wrote to stderr as well: %q", stderr)
	}
}

// TestDoctorRefusesTheNoticeBesideAFlagThatWouldSpendRequests pins the refusal that goes with the
// verb above: the statement is the whole run, so the flags that would run the rest of the verb
// beside it are a bad invocation rather than something silently dropped.
func TestDoctorRefusesTheNoticeBesideAFlagThatWouldSpendRequests(t *testing.T) {
	hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))

	for _, args := range [][]string{
		{"--show-notice", "--refresh-ids"},
		{"--show-notice", "--pin", "HomeTimeline", "1u0Wlkw6Ru1NwBUD-pDiww"},
		{"--show-notice", "--probe", "Viewer"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			code, stdout, stderr := run(t, Doctor, args...)
			if code != ExitUsage {
				t.Fatalf("doctor %v exited %d, want %d", args, code, ExitUsage)
			}
			if stdout != "" {
				t.Errorf("the rejected combination wrote to stdout: %q", stdout)
			}
			if !strings.Contains(stderr, "usage: x-terminal") {
				t.Errorf("the rejection does not carry the usage: %q", stderr)
			}
		})
	}
}

// A flag the verb cannot make sense of, and a flag whose arguments are missing, are bad
// invocations: they exit with the usage rather than silently doing something else.
func TestDoctorRejectsWhatItCannotParse(t *testing.T) {
	for _, args := range [][]string{
		{"--pin", "HomeTimeline"},
		{"--pin"},
		{"--nonsense"},
		{"extra"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			code, stdout, stderr := run(t, Doctor, args...)
			if code != ExitUsage {
				t.Fatalf("doctor %v exited %d, want %d", args, code, ExitUsage)
			}
			if !strings.Contains(stderr, "usage: x-terminal") {
				t.Errorf("the invocation was rejected without the usage: %q", stderr)
			}
			if stdout != "" {
				t.Errorf("a rejected invocation wrote to stdout: %q", stdout)
			}
		})
	}
}

func withStubDoctorCheck(t *testing.T, results []netcheck.Result) {
	t.Helper()
	previous := doctorRunCheck
	doctorRunCheck = func(context.Context, config.Config, session.Session) ([]netcheck.Result, func() (int, int, time.Time), string, *xweb.Client) {
		return results, func() (int, int, time.Time) { return 1, 0, time.Unix(1, 0).UTC() }, "", &xweb.Client{}
	}
	t.Cleanup(func() { doctorRunCheck = previous })
}

func writePresentMirror(t *testing.T, dataHome string) string {
	t.Helper()
	path := filepath.Join(dataHome, "x-terminal", "saved.db")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	ctx := context.Background()
	if err := s.SetOwner(ctx, "aergon"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)
	items := make([]domain.SavedItem, 0, 15)
	for i := range 12 {
		id := fmt.Sprintf("b%d", i)
		items = append(items, domain.SavedItem{
			Kind: domain.Bookmark, SortIndex: id, FirstSeen: now, LastSeen: now, Present: true,
			Post: domain.Post{ID: domain.PostID(id), Text: "b", Author: domain.User{ScreenName: "a", ID: "1"}, CreatedAt: now},
		})
	}
	for i := range 3 {
		id := fmt.Sprintf("l%d", i)
		items = append(items, domain.SavedItem{
			Kind: domain.Like, SortIndex: id, FirstSeen: now, LastSeen: now, Present: true,
			Post: domain.Post{ID: domain.PostID(id), Text: "l", Author: domain.User{ScreenName: "b", ID: "2"}, CreatedAt: now},
		})
	}
	if _, err := s.Upsert(ctx, items); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordRun(ctx, domain.SyncResult{
		Kind: domain.Bookmark, Pages: 1, Added: 12, Stop: domain.StopNoUnknownIDs,
		Started: now, Finished: now,
	}); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDoctorMirrorLineShapes(t *testing.T) {
	longDetail := strings.Repeat("x", jsonLineLimit+40)
	skipped := []netcheck.Result{{
		Outcome: netcheck.OutcomeSessionSkipped,
		Detail:  "no session configured",
	}}

	t.Run("absent", func(t *testing.T) {
		hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))
		withStubDoctorCheck(t, skipped)
		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}
		want := fmt.Sprintf("mirror     no mirror yet (%s)\n", cfg.SavedDBPath())
		code, stdout, _ := run(t, Doctor)
		if code != ExitOK {
			t.Fatalf("exit %d", code)
		}
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q missing %q", stdout, want)
		}
	})

	t.Run("present", func(t *testing.T) {
		data := t.TempDir()
		t.Setenv("XDG_DATA_HOME", data)
		hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))
		t.Setenv("XDG_DATA_HOME", data)
		path := writePresentMirror(t, data)
		withStubDoctorCheck(t, skipped)
		want := fmt.Sprintf("mirror     %s  schema 1  owner @aergon  bookmarks 12 present 0 absent  likes 3 present 0 absent  last sync 2026-10-14 09:12 (no unknown ids)\n", path)
		code, stdout, _ := run(t, Doctor)
		if code != ExitOK {
			t.Fatalf("exit %d", code)
		}
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q missing %q", stdout, want)
		}
	})

	t.Run("newer", func(t *testing.T) {
		data := t.TempDir()
		t.Setenv("XDG_DATA_HOME", data)
		hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))
		t.Setenv("XDG_DATA_HOME", data)
		path := writePresentMirror(t, data)
		db, err := sql.Open("sqlite", "file:"+path+"?mode=rw")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE meta SET value = '2' WHERE key = 'schema_version'`); err != nil {
			t.Fatal(err)
		}
		_ = db.Close()
		withStubDoctorCheck(t, skipped)
		want := fmt.Sprintf("mirror     %s  schema 2 (newer than this build knows - see RUNBOOK.md#the-mirror)\n", path)
		code, stdout, _ := run(t, Doctor)
		if code != ExitError {
			t.Fatalf("exit %d", code)
		}
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q missing %q", stdout, want)
		}
	})

	t.Run("unreadable", func(t *testing.T) {
		data := t.TempDir()
		t.Setenv("XDG_DATA_HOME", data)
		hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))
		t.Setenv("XDG_DATA_HOME", data)
		path := filepath.Join(data, "x-terminal", "saved.db")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("not a sqlite file"), 0o600); err != nil {
			t.Fatal(err)
		}
		withStubDoctorCheck(t, []netcheck.Result{{
			Outcome: netcheck.OutcomeSessionSkipped,
			Detail:  longDetail,
		}})
		code, stdout, _ := run(t, Doctor)
		if code != ExitError {
			t.Fatalf("exit %d", code)
		}
		if !strings.Contains(stdout, "mirror     "+path+"  unreadable:") {
			t.Fatalf("stdout %q", stdout)
		}
		if !strings.Contains(stdout, "see RUNBOOK.md#the-mirror") {
			t.Fatalf("missing runbook pointer: %q", stdout)
		}
	})
}

func TestDoctorLeavesMirrorBytesAndMtimeUnchanged(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))
	t.Setenv("XDG_DATA_HOME", data)
	path := writePresentMirror(t, data)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mtime := info.ModTime()
	withStubDoctorCheck(t, []netcheck.Result{{
		Outcome: netcheck.OutcomeSessionSkipped,
		Detail:  "no session configured",
	}})
	if code, _, _ := run(t, Doctor); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("doctor changed the mirror bytes")
	}
	info, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(mtime) {
		t.Fatalf("mtime moved from %s to %s", mtime, info.ModTime())
	}
}

func TestDoctorProbeOneRequestAndNoBody(t *testing.T) {
	const marker = "PROBE-BODY-MARKER-MUST-NOT-LEAK"
	var calls atomic.Int64
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("content-type", "application/json")
		w.Header().Set("x-rate-limit-remaining", "7")
		w.Header().Set("x-rate-limit-reset", fmt.Sprintf("%d", time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC).Unix()))
		fmt.Fprint(w, `{"data":{"viewer":{"secret":"`+marker+`"}}}`)
	}))
	defer srv.Close()

	hermeticConfig(t, writeSession(t, t.TempDir(), "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n"))
	previous := newAPIClient
	newAPIClient = func(_ config.Config, sess session.Session) (*xweb.Client, error) {
		transport := srv.Client().Transport.(*http.Transport).Clone()
		transport.Proxy = nil
		transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, srv.Listener.Addr().String())
		}
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client, err := xweb.NewClient(config.Config{RequestFloor: 0, SessionRequestCap: 10})
		if err != nil {
			return nil, err
		}
		client.HTTP = &http.Client{Transport: transport, CheckRedirect: xweb.NoRedirect}
		return client.WithSession(sess), nil
	}
	t.Cleanup(func() { newAPIClient = previous })

	code, stdout, stderr := run(t, Doctor, "--probe", "Viewer")
	if code != ExitOK {
		t.Fatalf("exit %d stderr %q", code, stderr)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("probe issued %d requests", got)
	}
	if !strings.Contains(stdout, "probe      Viewer  ok  remaining 7  reset ") {
		t.Fatalf("stdout %q", stdout)
	}
	if strings.Contains(stdout, marker) || strings.Contains(stderr, marker) {
		t.Fatal("probe output carried the body marker")
	}
}

// TestDoctorProbeUnknownOpIsUsage: the unknown name is one no table holds — TweetDetail used to
// be it, and it is a known operation now — and the printed list must carry every row of the
// probe's own table, because that list is what the operator reads to find the name they meant.
func TestDoctorProbeUnknownOpIsUsage(t *testing.T) {
	hermeticConfig(t, writeSession(t, t.TempDir(), "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n"))
	code, stdout, stderr := run(t, Doctor, "--probe", "NoSuchOperation")
	if code != ExitUsage {
		t.Fatalf("exit %d", code)
	}
	if stdout != "" {
		t.Fatalf("stdout %q", stdout)
	}
	known := probeOperations()
	if len(known) == 0 {
		t.Fatal("the probe table is empty, so this arm would pass by vacuity")
	}
	for _, name := range known {
		if !strings.Contains(stderr, name) {
			t.Errorf("stderr missing %s: %q", name, stderr)
		}
	}
}

// TestDoctorProbePrintsTheUpstreamsSentence is the arm this task exists for: a rejected request
// shape prints the upstream's own sentence naming what it wanted, in the text report and as
// `detail` in the JSON one, while the failure keeps the exit code it always had.
func TestDoctorProbePrintsTheUpstreamsSentence(t *testing.T) {
	const upstream = `Variable '$includePromotedContent' of required type 'Boolean!' was not provided`
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"errors":[{"message":"`+upstream+`","extensions":{"path":["variable","includePromotedContent"]}}]}`)
	}))
	defer srv.Close()
	hermeticConfig(t, writeSession(t, t.TempDir(), "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n"))
	previous := newAPIClient
	newAPIClient = func(_ config.Config, sess session.Session) (*xweb.Client, error) {
		transport := srv.Client().Transport.(*http.Transport).Clone()
		transport.Proxy = nil
		transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, srv.Listener.Addr().String())
		}
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client, err := xweb.NewClient(config.Config{RequestFloor: 0, SessionRequestCap: 10})
		if err != nil {
			return nil, err
		}
		client.HTTP = &http.Client{Transport: transport, CheckRedirect: xweb.NoRedirect}
		return client.WithSession(sess), nil
	}
	t.Cleanup(func() { newAPIClient = previous })

	code, stdout, _ := run(t, Doctor, "--probe", "HomeLatestTimeline")
	if code != ExitError {
		t.Fatalf("a rejected probe exited %d, want the failure code it always had", code)
	}
	if !strings.Contains(stdout, upstream) {
		t.Errorf("the text report does not carry the upstream's sentence: %q", stdout)
	}
	if !strings.Contains(stdout, string(xweb.OutcomeBadRequest)) {
		t.Errorf("the text report does not carry the classifier's word: %q", stdout)
	}

	code, stdout, _ = run(t, Doctor, "--json", "--probe", "HomeLatestTimeline")
	if code != ExitError {
		t.Fatalf("a rejected JSON probe exited %d, want the failure code it always had", code)
	}
	var report struct {
		Probe struct {
			Operation string `json:"operation"`
			Outcome   string `json:"outcome"`
			Detail    string `json:"detail"`
		} `json:"probe"`
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("the JSON probe report is not JSON: %v (%q)", err, stdout)
	}
	if report.Probe.Detail != upstream {
		t.Errorf("the JSON probe's detail is %q, want the upstream's own sentence", report.Probe.Detail)
	}
	if report.OK {
		t.Error("a rejected probe reported ok=true")
	}
}

func TestDoctorProbeWithoutSession(t *testing.T) {
	hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))
	code, stdout, stderr := run(t, Doctor, "--probe", "Viewer")
	if code != ExitError {
		t.Fatalf("exit %d", code)
	}
	if stdout != "" {
		t.Fatalf("stdout %q", stdout)
	}
	if !strings.Contains(stderr, "session") {
		t.Fatalf("stderr %q", stderr)
	}
}

func TestRunbookAnchorTableIsTotal(t *testing.T) {
	table := RunbookAnchors()
	if len(table) == 0 {
		t.Fatal("runbook table is empty")
	}
	for _, o := range failingNetcheckOutcomes() {
		if _, ok := table[string(o)]; !ok {
			t.Errorf("missing netcheck row %q", o)
		}
		if ptr := RunbookAnchor(string(o)); ptr == "" {
			t.Errorf("failing netcheck %q has empty pointer", o)
		}
	}
	for _, o := range failingXwebOutcomes() {
		if _, ok := table[string(o)]; !ok {
			t.Errorf("missing xweb row %q", o)
		}
		if ptr := RunbookAnchor(string(o)); ptr == "" {
			t.Errorf("failing xweb %q has empty pointer", o)
		}
	}
	for _, key := range []string{"mirror newer", "mirror unreadable", "mirror missing"} {
		if _, ok := table[key]; !ok {
			t.Errorf("missing mirror row %q", key)
		}
	}
	if RunbookAnchor("mirror missing") != "" {
		t.Fatal("missing mirror must not be a failure pointer")
	}
	if RunbookAnchor("mirror newer") != "RUNBOOK.md#the-mirror" {
		t.Fatal("mirror newer pointer drifted")
	}
}

func TestDoctorMediaLineHasBothDispatches(t *testing.T) {
	hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))
	t.Setenv("WAYLAND_DISPLAY", "wayland-secret-42")
	withStubDoctorCheck(t, []netcheck.Result{{
		Outcome: netcheck.OutcomeSessionSkipped,
		Detail:  "no session configured",
	}})
	code, stdout, _ := run(t, Doctor)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, "media xdg-open (images, links), mpv (video, gif)") {
		t.Fatalf("config line missing media facts: %q", stdout)
	}
	if strings.Contains(stdout, "wayland-secret-42") || strings.Contains(strings.ToLower(stdout), "clipboard") {
		t.Fatalf("media line leaked env/clipboard: %q", stdout)
	}
	if !strings.Contains(stdout, "signer     none (no transaction-id header is sent; the read below is the measurement)") {
		t.Fatalf("signer line missing: %q", stdout)
	}
}

// TestDoctorNamesTheLanguageItResolved is the translation fact: the config line carries the language
// the pane would ask for, read off the same resolved setting the pane reads. It is a presence fact and
// not a probe — naming whether api.x.com answers this endpoint would spend one of the account's
// requests, which is the one thing this verb must not do outside the check it already reports.
//
// The arm reads the language back out of config.Load() rather than spelling it, so it holds on any
// host: the run under test resolved the same value, from the same file and the same locale.
func TestDoctorNamesTheLanguageItResolved(t *testing.T) {
	hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))
	withStubDoctorCheck(t, []netcheck.Result{{
		Outcome: netcheck.OutcomeSessionSkipped,
		Detail:  "no session configured",
	}})

	code, stdout, _ := run(t, Doctor)
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.TranslateLanguage == "" {
		t.Fatalf("the run resolved no translation language at all")
	}
	if !strings.Contains(stdout, "translate "+cfg.TranslateLanguage) {
		t.Fatalf("the config line does not name the resolved language %q: %q", cfg.TranslateLanguage, stdout)
	}
	// The fact stays on the config line: no check line carries it, which is what keeps it a fact
	// about the configuration rather than a claim about the upstream.
	for _, line := range strings.Split(stdout, "\n") {
		for _, label := range []string{"probe", "route", "proxy", "upstream", "signer", "mirror"} {
			if strings.HasPrefix(line, label) && strings.Contains(line, cfg.TranslateLanguage) {
				t.Errorf("the %s line carries the language %q: it is a config fact, not a check", label, cfg.TranslateLanguage)
			}
		}
	}
}

func TestDoctorPresenceFactsTextAndJSON(t *testing.T) {
	longDetail := strings.Repeat("d", jsonLineLimit+80)
	cases := []struct {
		name    string
		results []netcheck.Result
		mirror  string // "", "present", "newer", "unreadable"
		wantOK  bool
	}{
		{"skipped session", []netcheck.Result{{Outcome: netcheck.OutcomeSessionSkipped, Detail: "no session"}}, "", true},
		{"no route", []netcheck.Result{{Outcome: netcheck.OutcomeNoRoute, Detail: longDetail}}, "", false},
		{"missing mirror", []netcheck.Result{{Outcome: netcheck.OutcomeSessionSkipped, Detail: "no session"}}, "", true},
		{"unreadable mirror", []netcheck.Result{{Outcome: netcheck.OutcomeSessionSkipped, Detail: "no session"}}, "unreadable", false},
		{"newer mirror", []netcheck.Result{{Outcome: netcheck.OutcomeSessionSkipped, Detail: "no session"}}, "newer", false},
	}
	credRe := regexp.MustCompile(`[A-Za-z0-9+/=_-]{40,}`)
	for _, tc := range cases {
		for _, asJSON := range []bool{false, true} {
			name := tc.name
			if asJSON {
				name += "/json"
			} else {
				name += "/text"
			}
			t.Run(name, func(t *testing.T) {
				data := t.TempDir()
				t.Setenv("XDG_DATA_HOME", data)
				path := writeSession(t, t.TempDir(), "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n")
				hermeticConfig(t, path)
				t.Setenv("XDG_DATA_HOME", data)
				switch tc.mirror {
				case "present":
					_ = writePresentMirror(t, data)
				case "newer":
					mp := writePresentMirror(t, data)
					db, err := sql.Open("sqlite", "file:"+mp+"?mode=rw")
					if err != nil {
						t.Fatal(err)
					}
					if _, err := db.Exec(`UPDATE meta SET value = '2' WHERE key = 'schema_version'`); err != nil {
						t.Fatal(err)
					}
					_ = db.Close()
				case "unreadable":
					mp := filepath.Join(data, "x-terminal", "saved.db")
					if err := os.MkdirAll(filepath.Dir(mp), 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(mp, []byte("not a sqlite file"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				withStubDoctorCheck(t, tc.results)
				args := []string{}
				if asJSON {
					args = append(args, "--json")
				}
				code, stdout, stderr := run(t, Doctor, args...)
				wantCode := ExitOK
				if !tc.wantOK {
					wantCode = ExitError
				}
				if code != wantCode {
					t.Fatalf("exit %d want %d stderr=%q", code, wantCode, stderr)
				}
				control := "control " + fixtureToken + " " + fixtureCTO
				if !strings.Contains(control, fixtureToken) || !strings.Contains(control, fixtureCTO) {
					t.Fatal("control search cannot find the fixture")
				}
				haystack := stdout + stderr
				for _, secret := range []string{
					fixtureToken, fixtureCTO,
					base64.StdEncoding.EncodeToString([]byte(fixtureToken)),
					base64.StdEncoding.EncodeToString([]byte(fixtureCTO)),
				} {
					if strings.Contains(haystack, secret) {
						t.Fatalf("report leaked credential material %q", secret[:8])
					}
				}
				if asJSON {
					var report doctorReport
					if err := json.Unmarshal([]byte(stdout), &report); err != nil {
						t.Fatalf("json: %v\n%s", err, stdout)
					}
					if report.OK != tc.wantOK {
						t.Fatalf("ok=%v want %v", report.OK, tc.wantOK)
					}
					if report.Config.Media == "" || !strings.Contains(report.Config.Media, "mpv") {
						t.Fatalf("media %q", report.Config.Media)
					}
					for _, check := range report.Checks {
						if len(check.Detail) > jsonLineLimit+3 {
							t.Fatalf("detail not clipped: %d", len(check.Detail))
						}
						if netcheck.Outcome(check.Outcome).Failed() && check.Runbook == "" {
							t.Fatalf("failing check missing runbook: %+v", check)
						}
					}
					if tc.mirror == "unreadable" || tc.mirror == "newer" {
						if report.Mirror.Runbook != "RUNBOOK.md#the-mirror" {
							t.Fatalf("mirror runbook %q", report.Mirror.Runbook)
						}
					}
				} else {
					for _, line := range strings.Split(strings.TrimRight(stdout, "\n"), "\n") {
						if len(line) > 400 {
							t.Fatalf("text line too long (%d): %q", len(line), line)
						}
					}
					for _, label := range doctorLabels {
						if label == "probe" || label == "route" || label == "proxy" || label == "upstream" {
							continue
						}
						needle := label
						if label == "requests" {
							needle = "requests:"
						}
						if !strings.Contains(stdout, needle) {
							t.Fatalf("missing label %q in %q", needle, stdout)
						}
					}
					for _, line := range strings.Split(stdout, "\n") {
						if strings.HasPrefix(line, "mirror") || strings.HasPrefix(line, "signer") ||
							strings.Contains(line, "media ") || strings.HasPrefix(line, "requests:") {
							if m := credRe.FindString(line); m != "" && len(m) >= len(fixtureToken) {
								if !strings.ContainsAny(m, "/:-.") && len(m) >= len(fixtureToken) {
									if m == fixtureToken || m == fixtureCTO || len(m) > len(fixtureToken)+20 {
										t.Fatalf("credential-shaped run in %q", line)
									}
								}
							}
						}
					}
					if tc.results[0].Outcome.Failed() && !strings.Contains(stdout, "RUNBOOK.md#") {
						t.Fatalf("failing check missing runbook: %q", stdout)
					}
				}
			})
		}
	}
}

func TestDoctorFailingCheckCarriesRunbookPointer(t *testing.T) {
	hermeticConfig(t, filepath.Join(t.TempDir(), "session-that-is-not-there"))
	withStubDoctorCheck(t, []netcheck.Result{{
		Outcome: netcheck.OutcomeAPIRejected,
		Detail:  "the Viewer answer: " + string(xweb.OutcomeRateLimited),
	}})
	code, stdout, _ := run(t, Doctor)
	if code != ExitError {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, "see RUNBOOK.md#429") {
		t.Fatalf("stdout %q", stdout)
	}
}
