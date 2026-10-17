// Package session reads the x.com session cookie from its own 0600 file and
// makes it impossible to print by accident.
//
// The two values — auth_token and ct0 — are the whole authentication this tool
// has: either one alone is enough to act as the account. They are therefore not
// strings. They are Secrets, whose every formatting path (String, GoString, Format,
// MarshalJSON, MarshalText and slog's LogValuer) yields "<redacted>", and whose
// raw bytes leave through the explicitly named Reveal alone. A struct dump, a
// JSON encode, a log line or an error that happens to carry a Session therefore
// cannot be the reason the cookie ends up in a bug report;
// internal/session/leak_test.go is the gate that keeps it that way.
//
// The file is refused, never repaired: a symlink at the path and a mode looser
// than 0600 are both errors naming what to do about them, because either one
// means the credential is readable by something other than its owner.
package session

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Redacted is the literal every formatter of a Secret produces. It is exported
// so that a caller can assert on it without spelling the string twice.
const Redacted = "<redacted>"

// environment is the Source value for a value that came from an override rather
// than from the file.
const environment = "environment"

// Secret is a session-cookie value. Every implicit path out of it returns
// Redacted; the bytes themselves leave through Reveal, which is named so that a
// reader of a diff sees it.
//
// Its underlying type is string — that is what makes len and == work without a
// conversion and what lets the request builder in a later phase hand the value
// to a header — so a deliberate `string(secret)` conversion would also reach the
// bytes. Nothing outside Reveal does that, and every formatting path a caller
// reaches for by habit is covered here.
type Secret string

// String covers %v and %s, and therefore covers the field of a struct dumped
// with %v or %+v.
func (s Secret) String() string { return Redacted }

// GoString covers %#v, the verb that is reached for when something goes wrong.
func (s Secret) GoString() string { return Redacted }

// Format covers every verb fmt consults a type for. fmt reaches for a type's own
// method when it has one and otherwise falls back to a placeholder that renders the
// value itself: without this method `fmt.Sprintf("%d", secret)` returns
// `%!d(string=<the cookie>)`, which is how a format verb nobody thought about
// becomes the leak. Every verb fmt asks about here — v, +v, #v, s, q, d, x, X, c, U,
// t, b, o, f, e, with width and precision — answers Redacted, and no verb reaches
// the bytes.
//
// v, s and q are delegated to String, flags and all. That is deliberate: it keeps
// String the single place a reader — or a test — can break to see the redaction
// contract fail, which is the property plan 01's revert check rests on.
//
// The one verb fmt does not consult anybody about is %p: it is handled before any
// method is looked for ("we always do them first", fmt/print.go), so `%p` on a
// Secret renders fmt's own placeholder, `%!p(session.Secret=<the value>)`. Measured
// on this toolchain, 2026-10-14. Nothing this package implements can intercept it;
// only a pointer-shaped underlying type would print an address instead, and the
// string underlying type is what makes len, == and the string() conversion in the
// request builder work. internal/session/leak_test.go pins this single residual,
// counts it, and fails if a credential reaches anywhere else.
func (s Secret) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v', 's', 'q':
		fmt.Fprintf(f, formatVerb(f, verb), s.String())
	default:
		fmt.Fprint(f, Redacted)
	}
}

// formatVerb rebuilds the verb a caller asked for — "%", fmt's own flags, the width
// and the precision, the verb — so that a delegated value keeps the caller's
// formatting instead of losing it.
func formatVerb(f fmt.State, verb rune) string {
	var b strings.Builder
	b.WriteByte('%')
	for _, flag := range []byte{'-', '+', '#', ' ', '0'} {
		if f.Flag(int(flag)) {
			b.WriteByte(flag)
		}
	}
	if width, ok := f.Width(); ok {
		b.WriteString(strconv.Itoa(width))
	}
	if precision, ok := f.Precision(); ok {
		b.WriteByte('.')
		b.WriteString(strconv.Itoa(precision))
	}
	b.WriteRune(verb)
	return b.String()
}

// MarshalJSON keeps the cookie out of an accidental encode of the struct that
// holds it.
func (s Secret) MarshalJSON() ([]byte, error) { return []byte(strconv.Quote(Redacted)), nil }

// MarshalText covers the other half of encoding: encoding/xml, a URL query
// builder, and any future header writer that takes an encoding.TextMarshaler.
func (s Secret) MarshalText() ([]byte, error) { return []byte(Redacted), nil }

// LogValue is slog's LogValuer, so `slog.Info("loaded", "auth_token", s.AuthToken)`
// logs the placeholder rather than the cookie.
func (s Secret) LogValue() slog.Value { return slog.StringValue(Redacted) }

// Reveal returns the cookie's bytes. It is the only escape hatch, and the
// package that builds an authenticated request is its first production caller.
func (s Secret) Reveal() string { return string(s) }

// Session is what the loader found: the two values, where they came from, and
// anything about the file its owner should fix.
//
// CTO is the CSRF token. It is not as sensitive as auth_token — it travels in
// x-csrf-token on every API request by design — but it is enough to forge that
// header, so it gets the same treatment and is never printed either.
type Session struct {
	AuthToken Secret
	CTO       Secret
	// Source names where the values came from: the file's path, "environment",
	// or both, for the doctor line. It never holds a value.
	Source string
	// Warnings are presence facts about the file that are never fatal — an
	// unknown key, a missing half of the pair. They hold no cookie bytes.
	Warnings []string
}

// Valid reports whether the pair the upstream requires is present. A session
// with one half is not a session: x.com answers 401 to an auth_token without a
// matching ct0, and that failure reads exactly like an expired cookie.
func (s Session) Valid() bool { return s.AuthToken != "" && s.CTO != "" }

// Summary describes the session by length and by origin and never by value, so
// that it can be printed in a bug report. It is what `doctor` shows.
func (s Session) Summary() string {
	state, source := "present", s.Source
	if !s.Valid() {
		state = "incomplete"
	}
	if source == "" {
		source = "unknown"
	}
	return fmt.Sprintf("session: %s (%s, %s, source %s)",
		state, described("auth_token", s.AuthToken), described("ct0", s.CTO), source)
}

// ErrNoSession is the typed "there is nothing to read" state: no file and no
// override. It is not a condition to paper over — an unauthenticated read is
// indistinguishable from an expired one — so it is a sentinel a caller can test
// with errors.Is and print.
var ErrNoSession = errors.New("session: no session cookie found")

// env is this package's view of the environment: the two documented override
// variables, read in Load and nowhere else. The tests replace it, so no test
// depends on whether the developer's shell exports a cookie.
var env = os.Getenv

// Load reads a session from path, with the two environment overrides applied.
//
// The refusals come first and in this order: a symlink is refused before its
// mode is looked at, because a symlink means the path is not the file that was
// checked; a mode looser than 0600 is refused with the exact chmod command,
// because a 0644 session file is a session anyone on the machine can use. A
// missing or empty file is not an error as long as the environment supplies the
// pair: the overrides exist to keep a session out of a file altogether.
// FileSession reads the session file and nothing else: no environment override is applied. It is
// what `login` asks about an existing file — with the overrides merged in, a pair supplied through
// X_TERMINAL_* would look like a session already on disk and refuse the very write it was meant to
// provide.
func FileSession(path string) (Session, error) {
	token, cto, warnings, err := readFile(path)
	if err != nil {
		return Session{}, err
	}
	if token == "" && cto == "" {
		return Session{}, ErrNoSession
	}
	source := ""
	if token != "" || cto != "" {
		source = path
	}
	return Session{AuthToken: token, CTO: cto, Source: source, Warnings: warnings}, nil
}

// readFile is the file half of Load: the checks that refuse a symlink or a loose mode, the read, and
// the parse. A path with no file is not an error here — Load's overrides can still supply the pair —
// and the returned warnings carry no cookie bytes.
func readFile(path string) (token, cto Secret, warnings []string, err error) {
	info, statErr := os.Lstat(path)
	readable := statErr == nil

	switch {
	case readable && info.Mode()&fs.ModeSymlink != 0:
		return "", "", nil, fmt.Errorf("session file is a symlink: %s", path)
	case readable && !info.Mode().IsRegular():
		return "", "", nil, fmt.Errorf("session file is not a regular file: %s", path)
	case readable:
		if perm := info.Mode().Perm(); perm&0o077 != 0 {
			return "", "", nil, fmt.Errorf("session file %s has mode %04o: users other than its owner can read the cookie; fix it with: chmod 600 %s",
				path, perm, path)
		}
	case errors.Is(statErr, fs.ErrNotExist):
		// No file: the overrides can still supply the whole pair.
	default:
		return "", "", nil, fmt.Errorf("session: %w", statErr)
	}

	if !readable {
		return "", "", nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", nil, fmt.Errorf("session: %w", err)
	}
	token, cto, warnings = parseFile(data)
	return token, cto, warnings, nil
}

func Load(path string) (Session, error) {
	token, cto, warnings, err := readFile(path)
	if err != nil {
		return Session{}, err
	}
	readable := true
	if _, statErr := os.Lstat(path); errors.Is(statErr, fs.ErrNotExist) {
		readable = false
	}
	if !readable {
		token, cto = "", ""
	}
	var tokenSource, ctoSource string
	if token != "" {
		tokenSource = path
	}
	if cto != "" {
		ctoSource = path
	}

	// Provenance is per value rather than per attempt: an override replaces the
	// file's value, and a Source that named the file after that happened would
	// tell the user their file is being used when it is not.
	if value := env("X_TERMINAL_AUTH_TOKEN"); value != "" {
		token, tokenSource = Secret(value), environment
	}
	if value := env("X_TERMINAL_CT0"); value != "" {
		cto, ctoSource = Secret(value), environment
	}

	if token == "" && cto == "" {
		note := ""
		if readable {
			// Saying "no session" about a file that is right there and wrong
			// would be the kind of half-truth this project exists to avoid.
			note = "; the file was read and holds no auth_token or ct0 line"
		}
		return Session{}, fmt.Errorf("%w: put auth_token=VALUE and ct0=VALUE in %s (mode 0600), or set X_TERMINAL_AUTH_TOKEN and X_TERMINAL_CT0; both are session cookies that disappear when the browser closes%s",
			ErrNoSession, path, note)
	}
	if token == "" {
		warnings = append(warnings, "session: auth_token is missing - add it to the session file or set X_TERMINAL_AUTH_TOKEN")
	}
	if cto == "" {
		warnings = append(warnings, "session: ct0 is missing - add it to the session file or set X_TERMINAL_CT0")
	}

	// The two origins, in value order and without repeats: "path", "environment",
	// or "path, environment" when the file supplied one half and the override the
	// other.
	sources := make([]string, 0, 2)
	for _, source := range []string{tokenSource, ctoSource} {
		if source != "" {
			sources = appendUnique(sources, source)
		}
	}

	return Session{
		AuthToken: token,
		CTO:       cto,
		Source:    strings.Join(sources, ", "),
		Warnings:  warnings,
	}, nil
}

// parseFile reads KEY=VALUE lines: blank lines and # comments are skipped, the
// two keys are matched case-insensitively, and a line that is neither is a
// warning. Nothing here is fatal — a session file is written by hand, and a
// stray line must not be the difference between the tool running and not. The
// last line for a key wins.
func parseFile(data []byte) (token, cto Secret, warnings []string) {
	for i, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			warnings = append(warnings, fmt.Sprintf("session file line %d: not a KEY=VALUE line", i+1))
			continue
		}
		key = strings.TrimSpace(key)
		value = unquoted(strings.TrimSpace(value))

		switch strings.ToLower(key) {
		case "auth_token":
			token = Secret(value)
		case "ct0":
			cto = Secret(value)
		default:
			warnings = append(warnings, fmt.Sprintf("session file line %d: unknown key %s", i+1, keyName(key)))
		}
	}
	return token, cto, warnings
}

// unquoted strips the quotes and a ` #...` tail from a value, so that a line
// copied out of a shell script — auth_token="abc" # from zen — still loads. A
// `#` with no space in front of it stays part of the value: silently truncating
// a credential would be a worse failure than a value that does not work.
func unquoted(value string) string {
	if i := strings.Index(value, " #"); i >= 0 {
		value = value[:i]
	}
	value = strings.TrimSpace(value)
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return value[1 : len(value)-1]
	}
	return value
}

// keyName renders an unknown key for a warning. A real key is a short
// identifier — `twid` is the one people paste by habit — and anything else is
// not echoed: this is the credential file, and a value pasted on the wrong side
// of the `=` would otherwise reach the terminal and whatever captures it, which
// is exactly what SEC-01 promises it does not.
func keyName(key string) string {
	if key == "" || len(key) > 32 {
		return "<not a key name>"
	}
	for i, r := range key {
		switch {
		case r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
		case r >= '0' && r <= '9' && i > 0:
		default:
			return "<not a key name>"
		}
	}
	return strconv.Quote(key)
}

// described names one credential by its length and never by its value.
func described(name string, s Secret) string {
	if s == "" {
		return name + " absent"
	}
	return fmt.Sprintf("%s %d bytes", name, len(s))
}

// appendUnique keeps Source stable and free of repeats when both values came
// from the same place.
func appendUnique(list []string, value string) []string {
	for _, existing := range list {
		if existing == value {
			return list
		}
	}
	return append(list, value)
}
