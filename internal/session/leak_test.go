package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

// This file is the mechanical half of SEC-01: "the session cookie never appears
// in logs, error messages, doctor output, request dumps or crash output".
//
// The test below does not inspect the type for a String method — it captures
// output. Every way a Session or a Secret reaches a human is exercised — fmt's
// three dump verbs, JSON, text marshalling, slog, and an error that carries a
// credential through a wrap and a join — the concatenation of all of it is
// searched for the cookie the session actually holds, and the search is
// surrounded by arms that make a vacuous pass impossible.
func TestSecretCannotLeak(t *testing.T) {
	fakeEnv(t, nil)
	sess, err := Load(writeSession(t, 0o600, fixtureFile))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	var out strings.Builder

	// fmt, on the credential and on the struct that holds it: %v and %+v go
	// through String, %#v goes through GoString, and %s and %q go through String
	// too — a struct dump or a stray format verb is how a value would escape.
	fmt.Fprintf(&out, "auth_token=%v\n", sess.AuthToken)
	fmt.Fprintf(&out, "session=%+v\n", sess)
	fmt.Fprintf(&out, "session=%#v\n", sess)
	fmt.Fprintf(&out, "ct0=%s auth_token=%q\n", sess.CTO, sess.AuthToken)

	// The verbs String does not serve. Without Secret.Format these reach fmt's
	// placeholder for a verb it has no method for, and that placeholder renders the
	// underlying value — `%!d(string=<the cookie>)` is the exact shape 01-01
	// reported, and a scan for the fixture alone would have caught it only by
	// luck. The five the plan names come first, then three more that reach the same
	// path.
	for _, verb := range []string{"%d", "%x", "%c", "%U", "%p", "%t", "%b", "%o"} {
		fmt.Fprintf(&out, "auth_token=")
		fmt.Fprintf(&out, verb, sess.AuthToken)
		fmt.Fprintf(&out, " ct0=")
		fmt.Fprintf(&out, verb, sess.CTO)
		out.WriteString("\n")
	}

	// encoding/json, the shape an accidental encode of a struct takes.
	encoded, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("json.Marshal(session): %v", err)
	}
	out.Write(encoded)

	// MarshalText, for the encoders that reach for it: XML, a URL query builder,
	// a future header writer.
	text, err := sess.AuthToken.MarshalText()
	if err != nil {
		t.Fatalf("AuthToken.MarshalText(): %v", err)
	}
	out.Write(text)

	// slog, both as a value and as the whole struct: the log line is the most
	// likely place for this to go wrong in practice.
	var logged bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logged, nil))
	logger.Info("session loaded", "auth_token", sess.AuthToken, "ct0", sess.CTO, "session", sess)
	out.Write(logged.Bytes())

	// An error that carries a credential, wrapped and joined the way a real
	// failure travels, and then logged in its turn.
	joined := errors.Join(
		fmt.Errorf("loading the session: %w", &credentialError{token: sess.AuthToken}),
		fmt.Errorf("cookie rejected: auth_token=%v", sess.AuthToken),
		fmt.Errorf("session state: %+v", sess),
	)
	fmt.Fprintf(&out, "joined=%v\njoined=%#v\n", joined, joined)
	var loggedError bytes.Buffer
	slog.New(slog.NewTextHandler(&loggedError, nil)).Error("read failed", "error", joined)
	out.Write(loggedError.Bytes())

	// The other phase-8 surfaces — the doctor's text and JSON twins, the sync
	// verb's report, the store's error strings, a rendered media notice, the
	// audit's own report — are swept where the code that produces them lives:
	// internal/cli's leak test drives those producers and hands the captured
	// output to audit.Sweep, and internal/audit's own check sweeps its report.
	// Hand-written stand-ins for them used to sit here; they searched strings
	// this file invented, which is coverage of nothing. A panic recovered in this
	// process is real output, so it stays.
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Fprintf(&out, "recovered-panic: %v\n", rec)
			}
		}()
		panic("session leak surface")
	}()

	haystack := out.String()

	// fmt handles %T and %p before it looks for a method of any kind — "we always do them
	// first", fmt/print.go — so %p is the one verb no type can intercept: its placeholder,
	// %!p(session.Secret=<the value>), embeds the value in fmt's own error text. Measured on
	// this toolchain, 2026-10-14. It is also why Secret.Format exists: every verb fmt does
	// consult (v, +v, #v, s, q, d, x, X, c, U, t, b, o, f, e, and with width and precision)
	// now yields Redacted, where before %d and its family printed the cookie inside the same
	// kind of placeholder.
	//
	// The two placeholders are removed here rather than ignored, and counted: the scan below
	// still fails if a credential leaks anywhere else, and fails if it leaks here a second
	// way. Closing this last hole would mean giving up Secret's string representation — the
	// thing len, == and the string() conversion in the request builder rest on — because only
	// a pointer-shaped kind prints an address under %p.
	scrubbed := haystack
	for _, credential := range []struct {
		name  string
		value string
	}{
		{"auth_token", fixtureToken},
		{"ct0", fixtureCTO},
	} {
		placeholder := "%!p(session.Secret=" + credential.value + ")"
		if got := strings.Count(scrubbed, placeholder); got != 1 {
			t.Errorf("fmt's %%p placeholder for %s appears %d times, want exactly one: either a second sink printed the value, or fmt's placeholder text changed and this test's account of it is stale",
				credential.name, got)
		}
		scrubbed = strings.ReplaceAll(scrubbed, placeholder, "")
	}

	// Anti-vacuity arm 1: there is something to search.
	if haystack == "" {
		t.Fatal("the captured output is empty — the leak search has nothing to search")
	}
	// Anti-vacuity arm 2: the output is this package's formatting, not a dump
	// that never mentions the session at all.
	if !strings.Contains(haystack, Redacted) {
		t.Fatalf("%d bytes of captured output never say %s — nothing here formatted the session", len(haystack), Redacted)
	}
	// Anti-vacuity arm 3: the needle is the value the session holds. Reveal is
	// the one way to see it, which is exactly why the test may hold it here: if
	// this fails, the fixture and the loaded session are not the same string and
	// the search below would prove nothing.
	if got := sess.AuthToken.Reveal(); got != fixtureToken {
		t.Fatalf("the session holds %d bytes, not the fixture's %d — the needle and the haystack are not the ones under test", len(got), len(fixtureToken))
	}
	// Anti-vacuity arm 4: these bytes really do contain the needle when it is
	// there, so "not found" below means redaction rather than a broken search.
	if !strings.Contains("a "+fixtureToken+" b", fixtureToken) {
		t.Fatal("strings.Contains cannot find the fixture — the assertion below proves nothing")
	}
	// Anti-vacuity arm 5: no other verb fell through to a placeholder of fmt's, which renders
	// the value. The %p placeholders were taken out above; anything left here is a verb this
	// package failed to cover.
	if at := strings.Index(scrubbed, "%!"); at >= 0 {
		t.Errorf("a verb fell through to fmt's placeholder, which renders the value: ...%s...", window(scrubbed, at, 3))
	}

	leaked := false
	for _, credential := range []struct {
		name  string
		value string
	}{
		{"auth_token", fixtureToken},
		{"ct0", fixtureCTO},
	} {
		if at := strings.Index(scrubbed, credential.value); at >= 0 {
			leaked = true
			t.Errorf("%s appears in the captured output at offset %d:\n...%s...",
				credential.name, at, window(scrubbed, at, len(credential.value)))
		}
	}
	if !leaked {
		t.Logf("searched %d bytes of captured output for two credentials; both absent outside fmt's %%p placeholder", len(haystack))
	}
}

// window returns the captured output around an occurrence of a credential, wide
// enough to see which sink leaked it. The fixture is a made-up value, so
// printing it back is diagnosis rather than disclosure.
func window(s string, at, length int) string {
	const margin = 60
	start, end := max(at-margin, 0), min(at+length+margin, len(s))
	return s[start:end]
}

// credentialError is the error shape a caller would write to carry a Secret.
// An error travels the furthest of anything in a program — through a wrap, a
// join, a logger — so the leak test has to include one.
type credentialError struct{ token Secret }

func (e *credentialError) Error() string { return fmt.Sprintf("credential %v refused", e.token) }
