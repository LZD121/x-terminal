// Package config resolves everything the client reads from outside itself: the
// XDG paths, the small config file under them, and the environment variables
// that override both.
//
// Every environment read in this repository happens here. A caller that needs a
// path, a proxy or a threshold asks this package, which is what lets the
// resolution rules be tested as one table instead of being re-derived — and
// quietly re-decided — at each call site.
//
// Secrets are deliberately not this package's business: the session cookie
// lives in its own 0600 file (internal/session), never in config.toml, whose
// keys and values are safe to print in diagnostics.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

const (
	// appDir is the directory name under each XDG base directory.
	appDir = "x-terminal"
	// ConfigFileName is the file this package reads inside ConfigDir.
	ConfigFileName = "config.toml"
	// sessionFileName is the default session file inside ConfigDir. The loader
	// that reads it is internal/session.
	sessionFileName = "session"
	// sessionsDirName is the directory the named accounts live in, beside the legacy file.
	sessionsDirName = "sessions"

	// DefaultRequestFloor is the minimum spacing between two upstream requests.
	// The upstream tolerates roughly one request per 1.5s per account; a second
	// request inside the floor waits rather than failing.
	DefaultRequestFloor = 1500 * time.Millisecond
	// DefaultSessionRequestCap is how many requests one session may spend before
	// the choke point refuses.
	DefaultSessionRequestCap = 200
	// DefaultMediaOpener is the external program media links are handed to.
	DefaultMediaOpener = "xdg-open"
	// DefaultTranslateLanguage is the language the translation endpoint is asked for when neither
	// the config file nor the locale names one. It is a setting and not a credential, which is why
	// it is a config key the doctor can print (this file's own rule about what is safe to print).
	DefaultTranslateLanguage = "zh-CN"
	// DefaultDisplayState and DefaultDisplayCover are what the two display keys
	// are when the file does not name them: the plain reader, and the first of
	// the four cover styles.
	DefaultDisplayState = DisplayStateNormal
	DefaultDisplayCover = DisplayCoverServiceLog

	// The display state's vocabulary: which state the pane starts in.
	DisplayStateNormal    = "normal"
	DisplayStateImmersive = "immersive"
	DisplayStatePanic     = "panic"

	// The cover style's vocabulary: which style the immersive state wears. The
	// plain reader is not a style — it is the absence of one.
	DisplayCoverServiceLog = "servicelog"
	DisplayCoverGitLog     = "gitlog"
	DisplayCoverCI         = "ci"
	DisplayCoverInternal   = "internal"
)

// DisplayStates is the values display.state accepts, in the order the state
// machine walks them. DisplayCovers is the same for display.cover, in the order
// the cycle key steps through the styles.
//
// The vocabulary is declared here, once each, because it is a fact about this
// file's schema and this package is the file's reader: a value the reader does
// not accept is a value a user cannot write. The state machine and the render
// package have vocabularies of their own — the shell's states and the cover
// styles' ids — and one test joins the three, so the file's words and the
// machine's words cannot drift apart silently.
func DisplayStates() []string {
	return []string{DisplayStateNormal, DisplayStateImmersive, DisplayStatePanic}
}

func DisplayCovers() []string {
	return []string{DisplayCoverServiceLog, DisplayCoverGitLog, DisplayCoverCI, DisplayCoverInternal}
}

// Config is the resolved configuration: paths that exist, a proxy that may be
// empty (empty means "dial directly", which is a supported configuration and
// not a broken one), and the two request-discipline numbers.
type Config struct {
	// ConfigDir is where config.toml and the session file live.
	ConfigDir string
	// DataDir is where this tool keeps its own state (pins, the first-run
	// marker). Nothing in Phase 1 writes to it yet.
	DataDir string
	// SessionPath is the file internal/session reads: the single file `login` wrote before there
	// could be more than one account, and the account named session.DefaultName.
	SessionPath string
	// SessionsDir is where the named accounts live: one 0600 file per name. It sits beside the
	// legacy single file rather than replacing it, so a session that already exists keeps working
	// with no migration.
	SessionsDir string
	// Account is the name of the account to start on, from the file's `account` key, or empty for
	// "whichever one is there" (session.Store.Active decides, and reports what it chose).
	Account string
	// Proxy is the proxy setting with an http, https, socks5 or socks5h scheme,
	// or "" for a direct dial. It may carry credentials and is therefore never
	// printed raw: print Redacted.
	Proxy string
	// NoProxy is the NO_PROXY value, passed to the transport as it was given.
	// This package does not implement the matching; the transport's own matcher
	// is the one that decides.
	NoProxy string
	// MediaOpener is the program that opens a link.
	MediaOpener string
	// RequestFloor is the minimum spacing between two requests.
	RequestFloor time.Duration
	// SessionRequestCap is the request budget for one session.
	SessionRequestCap int
	// DisplayState is the state the pane opens in: one of DisplayStates(), or
	// DefaultDisplayState when the file does not say. A start-up state of panic
	// is accepted — the user may choose it deliberately — and the sequence runs
	// after the first frame.
	DisplayState string
	// DisplayCover is the cover style the immersive state wears: one of
	// DisplayCovers(), or DefaultDisplayCover. It is remembered in the plain
	// state too, where it is not worn.
	DisplayCover string
	// TranslateLanguage is the language a translation is rendered in: the file's
	// `translate.language` key when it is there, then the locale (LC_ALL, then LANG), then
	// DefaultTranslateLanguage. The file is the user's deliberate choice and outranks the
	// environment; the locale is the environment's own statement about the language this host
	// is being used in, and it outranks the built-in default.
	//
	// It is a setting, not a credential, so the doctor prints it and Redacted leaves it alone.
	TranslateLanguage string
	// Warnings are problems in config.toml that were not fatal: a key nobody
	// knows, a value that would not parse. A warning never holds the value it
	// complains about, only the key and the line.
	Warnings []string
}

// env is this package's whole view of the process environment. The tests
// replace it, which is what lets a precedence test be hermetic on a machine that
// already exports a proxy; an empty value means "unset", so `HTTPS_PROXY= ` is
// read the way a user means it — as turning the proxy off for one command.
var env = os.Getenv

// Load resolves the configuration. A missing config.toml is not an error — the
// defaults are the configuration — but a file that exists and cannot be read
// is, because in that case the run would silently be a different configuration
// than the one the user wrote.
func Load() (Config, error) {
	configHome, err := xdgDir("XDG_CONFIG_HOME", ".config")
	if err != nil {
		return Config{}, err
	}
	dataHome, err := xdgDir("XDG_DATA_HOME", filepath.Join(".local", "share"))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		ConfigDir:         filepath.Join(configHome, appDir),
		DataDir:           filepath.Join(dataHome, appDir),
		SessionPath:       filepath.Join(configHome, appDir, sessionFileName),
		SessionsDir:       filepath.Join(configHome, appDir, sessionsDirName),
		MediaOpener:       DefaultMediaOpener,
		RequestFloor:      DefaultRequestFloor,
		SessionRequestCap: DefaultSessionRequestCap,
		DisplayState:      DefaultDisplayState,
		DisplayCover:      DefaultDisplayCover,
		// Seeded from the locale and refined by applySettings below: the file is the user's
		// deliberate choice and outranks the environment, and the locale is the environment's own
		// statement about the language this host is being used in.
		TranslateLanguage: localeLanguage(),
	}

	switch data, err := os.ReadFile(filepath.Join(cfg.ConfigDir, ConfigFileName)); {
	case err == nil:
		settings, warnings := parseSettings(data)
		cfg.Warnings = append(cfg.Warnings, warnings...)
		applySettings(&cfg, settings, &cfg.Warnings)
	case errors.Is(err, fs.ErrNotExist):
		// No file: the defaults above are the whole configuration.
	default:
		return Config{}, fmt.Errorf("config: %w", err)
	}

	applyEnv(&cfg)
	return cfg, nil
}

// PinsPath is the file the user's own operation ids live in: the pins file under DataDir, or ""
// when the data directory is not configured (a zero Config in a test, say), which every reader
// takes to mean "no pins".
//
// The path is built here rather than at each call site so that the refresh tool, the client and
// the doctor cannot disagree about where a pin is written and where it is read; the file's *name*
// is the registry package's (volatile.PinsFileName), beside the file's own shape, so that the two
// cannot disagree about it either.
func (c Config) PinsPath() string {
	if c.DataDir == "" {
		return ""
	}
	return filepath.Join(c.DataDir, volatile.PinsFileName)
}

// NoticePath is the marker file that records that the account-risk statement has been shown, so
// that it is shown once. It lives beside the pins for the same reason: this is the tool's own
// state, not the user's configuration.
func (c Config) NoticePath() string {
	if c.DataDir == "" {
		return ""
	}
	return filepath.Join(c.DataDir, "notice-shown")
}

// SavedDBPath is the mirror file — saved.db under DataDir — or "" when the data
// directory is not configured. Built here for the same reason PinsPath and
// NoticePath are: the sync verb, the pane and (in Phase 8) doctor must not be
// able to disagree about where the file is.
func (c Config) SavedDBPath() string {
	if c.DataDir == "" {
		return ""
	}
	return filepath.Join(c.DataDir, "saved.db")
}

// Redacted returns a copy that is safe to print. The proxy setting is the one
// field that can carry a credential — http://user:password@host:port is a legal
// proxy URL, and a password in a bug report is a password — so it is reduced to
// scheme://host. A setting that does not parse is not echoed at all: a string
// that is not a URL may hold a credential anywhere inside it.
//
// The session path is not redacted. It names a file rather than its contents,
// and it is precisely what a user needs in order to fix a session that will not
// load (see session.Summary).
func (c Config) Redacted() Config {
	if c.Proxy == "" {
		return c
	}
	u, err := url.Parse(c.Proxy)
	if err != nil || u.Host == "" {
		c.Proxy = "<set>"
		return c
	}
	if u.Scheme == "" {
		c.Proxy = u.Host
		return c
	}
	c.Proxy = u.Scheme + "://" + u.Host
	return c
}

// applyEnv folds the environment into cfg.
//
// The proxy order is the plan's: X_TERMINAL_PROXY, then the file's `proxy` key
// (already in cfg), then the standard variables HTTPS_PROXY, HTTP_PROXY,
// ALL_PROXY — each upper-case name before its lower-case twin, which is the
// rule the standard library's own proxy resolution follows. ALL_PROXY is read
// here and nowhere else: net/http honours only HTTP_PROXY, HTTPS_PROXY and
// NO_PROXY, so a proxy a user exported for everything else would otherwise be
// the one setting this tool silently ignores.
func applyEnv(cfg *Config) {
	if v := env("X_TERMINAL_PROXY"); v != "" {
		cfg.Proxy = v
	}
	if cfg.Proxy == "" {
		cfg.Proxy = firstEnv("HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy")
	}
	if v := firstEnv("NO_PROXY", "no_proxy"); v != "" {
		cfg.NoProxy = v
	}
	if v := env("X_TERMINAL_SESSION"); v != "" {
		cfg.SessionPath = v
	}
}

// firstEnv returns the first of names that is set to a non-empty value.
func firstEnv(names ...string) string {
	for _, name := range names {
		if v := env(name); v != "" {
			return v
		}
	}
	return ""
}

// translateLanguage is the shape a language tag is allowed to have before it becomes the setting the
// translation endpoint is asked for: two or three letters, then zero or more `-`-separated
// alphanumeric subtags. `zh-CN`, `de`, `ja` and `en` — the four the endpoint was measured with on
// 2026-10-16 — all match; a value holding a parenthesis, a comma, a slash, a space or a percent sign
// does not.
//
// It is deliberately a second, independent guard rather than a call into internal/xweb's: that one
// exists to refuse a value before a URL is assembled out of it, and this one exists to warn about a
// line of a config file. Merging them would make a warning about the user's file depend on the
// transport's escaping rule, which is not the same question — and the day the endpoint takes a
// parameter this file does not know about, the two answers part company.
var translateLanguage = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]+)*$`)

// localeLanguage is the language this host says it is being used in: LC_ALL, then LANG, then
// DefaultTranslateLanguage. It is the middle of the setting's three sources — the file is the user's
// deliberate choice and outranks the environment, and the locale outranks the built-in default,
// because a host told to speak Japanese should translate into Japanese without being configured.
//
// It reads the environment through this package's own seam, so the resolution is testable without a
// subprocess and the developer's own LANG cannot leak into a test.
func localeLanguage() string {
	for _, name := range []string{"LC_ALL", "LANG"} {
		if tag := languageTag(env(name)); tag != "" {
			return tag
		}
	}
	return DefaultTranslateLanguage
}

// languageTag reduces one locale value to a language tag, or to "" when it names no language at all.
//
// The codeset and the modifier are dropped at the first `.` or `@` (`zh_CN.UTF-8` → `zh_CN`,
// `de_DE@euro` → `de_DE`), the subtag separator is spelled the way a tag spells it (`zh_CN` →
// `zh-CN`) and the subtags after the first are upper-cased (`zh_cn` → `zh-CN`, `zh_Hans_CN` →
// `zh-HANS-CN`). `C` and `POSIX` name no language — they mean "no locale", which is what an
// unconfigured host exports — and a value that is not a tag this build will ask the endpoint for is
// refused here too, so the setting is either something the endpoint was measured with or the default.
func languageTag(locale string) string {
	if i := strings.IndexAny(locale, ".@"); i >= 0 {
		locale = locale[:i]
	}
	switch locale {
	case "", "C", "POSIX":
		return ""
	}
	parts := strings.Split(strings.ReplaceAll(locale, "-", "_"), "_")
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i])
	}
	tag := strings.Join(parts, "-")
	if !translateLanguage.MatchString(tag) {
		return ""
	}
	return tag
}

// xdgDir resolves one XDG base directory: the variable when it names an
// absolute path, else the fallback under $HOME.
//
// A relative value is ignored rather than joined onto the working directory.
// That is the rule os.UserConfigDir follows, and it is the one that keeps a
// stray XDG_CONFIG_HOME=./conf from putting the session file wherever the shell
// happened to be standing.
func xdgDir(variable, fallback string) (string, error) {
	if v := env(variable); filepath.IsAbs(v) {
		return v, nil
	}
	home := env("HOME")
	if !filepath.IsAbs(home) {
		return "", fmt.Errorf("config: cannot resolve the XDG base directories: set %s or HOME", variable)
	}
	return filepath.Join(home, fallback), nil
}

// setting is one parsed line of config.toml: which key set what, and on which
// line, so that a warning can point at it.
type setting struct {
	line  int
	key   string
	value string
}

// parseSettings reads the KEY = "value" lines of a config.toml-shaped file. It
// is deliberately not a TOML parser: this file holds a path, a proxy, a program
// name, a duration and a count, and a dependency that reads them would be a
// larger surface than the configuration it reads. Escapes are not processed, so
// a value is taken literally — no key here wants a newline in it.
//
// Keys are matched case-insensitively; the last line for a key wins.
func parseSettings(data []byte) (settings []setting, warnings []string) {
	for i, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			warnings = append(warnings, formatWarning(i+1, "not a KEY = \"VALUE\" line"))
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			warnings = append(warnings, formatWarning(i+1, "no key before the ="))
			continue
		}
		value, ok = valueOf(value)
		if !ok {
			// The key is the user's own identifier and the file holds no
			// credential by design (the cookie has its own 0600 file), so the
			// key is named here: that is what makes the warning fixable.
			warnings = append(warnings, formatWarning(i+1, fmt.Sprintf("key %q has a malformed value; use \"quoted\" or bare", key)))
			continue
		}
		settings = append(settings, setting{line: i + 1, key: strings.ToLower(key), value: value})
	}
	return settings, warnings
}

// valueOf reads the right-hand side of a setting: either a quoted string, whose
// closing quote ends the value, or a bare run of characters up to a ` #`
// comment. A quoted value keeps whatever whitespace and `#` characters it holds.
func valueOf(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, `"`) {
		rest := raw[1:]
		end := strings.IndexByte(rest, '"')
		if end < 0 {
			return "", false
		}
		if trailing := strings.TrimSpace(rest[end+1:]); trailing != "" && !strings.HasPrefix(trailing, "#") {
			return "", false
		}
		return rest[:end], true
	}
	if i := strings.Index(raw, " #"); i >= 0 {
		raw = raw[:i]
	}
	return strings.TrimSpace(raw), true
}

// accountName is the one validation this file does of an account name, and it mirrors
// session.ValidName: config cannot import the session package (the session package is the thing
// that reads what this name points at, and the direction of that dependency is the wrong way), so
// the rule is stated twice with a test on each side that they agree.
func accountName(name string) bool {
	if name == "" || len(name) > 64 || name[0] == '.' {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

// applySettings folds the file's settings into cfg. Nothing here fails: a
// config file with a typo in it must still start the tool, and a value that is
// refused keeps the default it would have had, with a warning naming the line.
//
// A duration or a count that is zero or negative is refused rather than
// accepted, because those two values are the request discipline: a floor of 0
// would mean "no floor", and a cap of 0 would mean "no requests allowed", and
// neither is what someone writing `request_floor = "0s"` was trying to say.
func applySettings(cfg *Config, settings []setting, warnings *[]string) {
	for _, s := range settings {
		switch s.key {
		case "account":
			if !accountName(s.value) {
				*warnings = append(*warnings, formatWarning(s.line, "account "+s.value+" is not a name this build uses (letters, digits, - and _, not starting with a dot); keeping "+cfg.Account))
				continue
			}
			cfg.Account = s.value
		case "session_path":
			if s.value == "" {
				*warnings = append(*warnings, formatWarning(s.line, "session_path is empty; keeping "+cfg.SessionPath))
				continue
			}
			cfg.SessionPath = s.value
		case "proxy":
			cfg.Proxy = s.value
		case "no_proxy":
			cfg.NoProxy = s.value
		case "media_opener":
			if s.value == "" {
				*warnings = append(*warnings, formatWarning(s.line, "media_opener is empty; keeping "+cfg.MediaOpener))
				continue
			}
			cfg.MediaOpener = s.value
		case "request_floor":
			d, err := time.ParseDuration(s.value)
			if err != nil || d <= 0 {
				*warnings = append(*warnings, formatWarning(s.line, fmt.Sprintf("request_floor is not a positive duration like 1500ms; keeping %s", cfg.RequestFloor)))
				continue
			}
			cfg.RequestFloor = d
		case "session_request_cap":
			n, err := strconv.Atoi(s.value)
			if err != nil || n <= 0 {
				*warnings = append(*warnings, formatWarning(s.line, fmt.Sprintf("session_request_cap is not a positive number; keeping %d", cfg.SessionRequestCap)))
				continue
			}
			cfg.SessionRequestCap = n
		case "display.state":
			if !slices.Contains(DisplayStates(), s.value) {
				*warnings = append(*warnings, formatWarning(s.line, fmt.Sprintf("display.state is not one of %s; keeping %s", strings.Join(DisplayStates(), ", "), cfg.DisplayState)))
				continue
			}
			cfg.DisplayState = s.value
		case "display.cover":
			if !slices.Contains(DisplayCovers(), s.value) {
				*warnings = append(*warnings, formatWarning(s.line, fmt.Sprintf("display.cover is not one of %s; keeping %s", strings.Join(DisplayCovers(), ", "), cfg.DisplayCover)))
				continue
			}
			cfg.DisplayCover = s.value
		case "translate.language":
			// A value outside the tag pattern is refused rather than carried: the setting becomes a
			// query parameter of a request this build builds by hand, and a file's typo must be
			// visible at the line that holds it rather than at the endpoint. The value is never
			// echoed — the warning names what is accepted and what stands instead.
			if !translateLanguage.MatchString(s.value) {
				*warnings = append(*warnings, formatWarning(s.line, fmt.Sprintf("translate.language is not a language tag like zh-CN or de; keeping %s", cfg.TranslateLanguage)))
				continue
			}
			cfg.TranslateLanguage = s.value
		default:
			*warnings = append(*warnings, formatWarning(s.line, fmt.Sprintf("unknown key %q", s.key)))
		}
	}
}

// SetDisplay writes the two display keys into the config file at path: the
// existing lines are replaced in place and whichever of the two is absent is
// appended. The rest of the file is left byte for byte alone — the comments, the
// blank lines and the keys this build does not know are the user's writing, not
// this program's — and nothing else about the configuration is written at all.
//
// The values are the state's name and the style's id, and neither names an
// account: a write of this file can never become the alias table the phase
// forbids, because this package never sees an alias.
//
// A missing file is not an error: both lines are written into a new one, created
// 0600, because a file this tool creates holds the user's own settings. An
// existing file keeps its own permissions.
func SetDisplay(path, state, cover string) error {
	if path == "" {
		return errors.New("config: SetDisplay: no path to write")
	}

	body, err := os.ReadFile(path)
	created := errors.Is(err, fs.ErrNotExist)
	if err != nil && !created {
		return fmt.Errorf("config: %w", err)
	}

	lines, trailing := splitLines(string(body))
	written := map[string]bool{}
	for i, line := range lines {
		key, ok := settingKey(line)
		if !ok {
			continue
		}
		switch key {
		case "display.state":
			// Every line that sets the key is replaced, not only the first: the
			// reader takes the last one, and a replacement that left a stale
			// line behind would disagree with itself.
			lines[i], written[key] = formatSetting(key, state), true
		case "display.cover":
			lines[i], written[key] = formatSetting(key, cover), true
		}
	}
	for _, setting := range []struct{ key, value string }{
		{"display.state", state},
		{"display.cover", cover},
	} {
		if !written[setting.key] {
			lines = append(lines, formatSetting(setting.key, setting.value))
		}
	}

	content := strings.Join(lines, "\n")
	if len(lines) > 0 && (trailing || created) {
		content += "\n"
	}
	return writeFile(path, content)
}

// splitLines splits a file into its lines without their terminators, and reports
// whether the file ended with one. An empty file is no lines at all rather than
// one empty line, so that appending two settings to it does not begin with a
// blank row.
func splitLines(body string) (lines []string, trailing bool) {
	if body == "" {
		return nil, false
	}
	trailing = strings.HasSuffix(body, "\n")
	return strings.Split(strings.TrimSuffix(body, "\n"), "\n"), trailing
}

// settingKey is the key a line of the file sets, lower-cased, or false for a
// line that sets none. It reads the file exactly the way parseSettings does —
// comments and blank lines set nothing, a line with no `=` sets nothing, and the
// last line for a key wins — so a line the reader accepts is a line the writer
// replaces.
func settingKey(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	key, _, ok := strings.Cut(trimmed, "=")
	if !ok {
		return "", false
	}
	if key = strings.TrimSpace(key); key == "" {
		return "", false
	}
	return strings.ToLower(key), true
}

// formatSetting is one key's canonical line: the shape the reader documents and
// the shape every warning names.
func formatSetting(key, value string) string {
	return fmt.Sprintf("%s = %q", key, value)
}

// writeFile writes content to path through a temporary file in the same
// directory, renamed over the original. The rename is atomic within a
// filesystem, so a reader of config.toml sees either the old file or the new one:
// a half-written configuration is the one outcome this writer may not have. The
// temporary file lives beside its target because a rename across filesystems is
// not a rename.
func writeFile(path, content string) error {
	mode := fs.FileMode(0o600)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		// The file is this tool's own, in its own directory: a fresh install has neither, and a
		// choice the user made with a key still has to be the one the next run starts in.
		return fmt.Errorf("config: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ConfigFileName+".tmp")
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	name := tmp.Name()
	defer os.Remove(name)

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("config: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return fmt.Errorf("config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return nil
}

// formatWarning builds one warning line. No warning prints a setting's value:
// this file is allowed to hold a proxy password, and a warning goes to a
// terminal and to whatever captures it.
func formatWarning(line int, message string) string {
	return fmt.Sprintf("%s:%d: %s", ConfigFileName, line, message)
}
