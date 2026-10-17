package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

// fakeEnv installs vars as this package's only environment. A key that is absent
// reads as "", which the package treats as unset, so a precedence test is
// hermetic: whether the developer's shell exports HTTPS_PROXY or a session
// override cannot change what the test sees, and "unset" is expressible instead
// of merely implied.
func fakeEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	previous := env
	env = func(key string) string { return vars[key] }
	t.Cleanup(func() { env = previous })
}

// configHome writes body as config.toml inside a fresh XDG config home and
// returns that home. An empty body writes no file at all, which is the state
// every default-resolution test is about.
func configHome(t *testing.T, body string) string {
	t.Helper()
	home := t.TempDir()
	if body == "" {
		return home
	}
	dir := filepath.Join(home, appDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("creating the config directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(body), 0o600); err != nil {
		t.Fatalf("writing %s: %v", ConfigFileName, err)
	}
	return home
}

func TestDefaultsResolveFromHomeWhenXDGIsUnset(t *testing.T) {
	home := t.TempDir()
	fakeEnv(t, map[string]string{"HOME": home})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with no XDG variables and no config file: %v", err)
	}

	wantConfig := filepath.Join(home, ".config", appDir)
	if cfg.ConfigDir != wantConfig {
		t.Errorf("ConfigDir = %q, want %q", cfg.ConfigDir, wantConfig)
	}
	if want := filepath.Join(home, ".local", "share", appDir); cfg.DataDir != want {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, want)
	}
	if want := filepath.Join(wantConfig, sessionFileName); cfg.SessionPath != want {
		t.Errorf("SessionPath = %q, want %q", cfg.SessionPath, want)
	}
	if cfg.Proxy != "" {
		t.Errorf("Proxy = %q, want the direct dial the defaults describe", cfg.Proxy)
	}
	if cfg.NoProxy != "" {
		t.Errorf("NoProxy = %q, want empty when nothing sets it", cfg.NoProxy)
	}
	if cfg.MediaOpener != DefaultMediaOpener {
		t.Errorf("MediaOpener = %q, want %q", cfg.MediaOpener, DefaultMediaOpener)
	}
	if cfg.RequestFloor != DefaultRequestFloor {
		t.Errorf("RequestFloor = %s, want %s", cfg.RequestFloor, DefaultRequestFloor)
	}
	if cfg.SessionRequestCap != DefaultSessionRequestCap {
		t.Errorf("SessionRequestCap = %d, want %d", cfg.SessionRequestCap, DefaultSessionRequestCap)
	}
	if len(cfg.Warnings) != 0 {
		t.Errorf("a default configuration warned: %q", cfg.Warnings)
	}
}

func TestXDGBaseDirectoriesWinOverHome(t *testing.T) {
	home, xdgConfig, xdgData := t.TempDir(), t.TempDir(), t.TempDir()
	fakeEnv(t, map[string]string{
		"HOME":            home,
		"XDG_CONFIG_HOME": xdgConfig,
		"XDG_DATA_HOME":   xdgData,
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with both XDG variables set: %v", err)
	}
	if want := filepath.Join(xdgConfig, appDir); cfg.ConfigDir != want {
		t.Errorf("ConfigDir = %q, want %q", cfg.ConfigDir, want)
	}
	if want := filepath.Join(xdgData, appDir); cfg.DataDir != want {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, want)
	}
	if want := filepath.Join(xdgConfig, appDir, sessionFileName); cfg.SessionPath != want {
		t.Errorf("SessionPath = %q, want %q", cfg.SessionPath, want)
	}
	// The fallback is not what answered: a test whose XDG variables are ignored
	// would still agree with the paths above on a machine with a $HOME.
	if strings.HasPrefix(cfg.ConfigDir, home) {
		t.Errorf("$HOME decided the config directory while XDG_CONFIG_HOME was set: %q", cfg.ConfigDir)
	}
}

func TestRelativeXDGIgnored(t *testing.T) {
	home := t.TempDir()
	fakeEnv(t, map[string]string{"HOME": home, "XDG_CONFIG_HOME": "relative/config"})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with a relative XDG_CONFIG_HOME: %v", err)
	}
	if want := filepath.Join(home, ".config", appDir); cfg.ConfigDir != want {
		t.Errorf("ConfigDir = %q, want %q — a relative value was joined onto the working directory", cfg.ConfigDir, want)
	}
}

func TestNoHomeAndNoXDGIsAnError(t *testing.T) {
	fakeEnv(t, nil)

	if _, err := Load(); err == nil {
		t.Fatal("Load resolved a config directory with neither XDG_CONFIG_HOME nor HOME set")
	} else if !strings.Contains(err.Error(), "XDG_CONFIG_HOME") {
		t.Errorf("the error does not say what to set: %v", err)
	}
}

func TestProxyPrecedence(t *testing.T) {
	tests := []struct {
		name string
		file string
		env  map[string]string
		want string
	}{
		{
			name: "X_TERMINAL_PROXY beats the file and the standard variables",
			file: `proxy = "http://from-file:1"`,
			env:  map[string]string{"X_TERMINAL_PROXY": "http://from-app:2", "HTTPS_PROXY": "http://from-https:3"},
			want: "http://from-app:2",
		},
		{
			name: "the file beats the standard variables",
			file: `proxy = "http://from-file:1"`,
			env: map[string]string{
				"HTTPS_PROXY": "http://from-https:3",
				"HTTP_PROXY":  "http://from-http:4",
				"ALL_PROXY":   "http://from-all:5",
			},
			want: "http://from-file:1",
		},
		{
			name: "HTTPS_PROXY beats HTTP_PROXY and ALL_PROXY",
			env: map[string]string{
				"HTTPS_PROXY": "http://from-https:3",
				"HTTP_PROXY":  "http://from-http:4",
				"ALL_PROXY":   "http://from-all:5",
			},
			want: "http://from-https:3",
		},
		{
			name: "HTTP_PROXY beats ALL_PROXY",
			env:  map[string]string{"HTTP_PROXY": "http://from-http:4", "ALL_PROXY": "socks5h://from-all:5"},
			want: "http://from-http:4",
		},
		{
			// net/http honours only HTTP_PROXY, HTTPS_PROXY and NO_PROXY, so
			// this is the row that would fail if ALL_PROXY were left to it.
			name: "ALL_PROXY is honoured when HTTPS_PROXY and HTTP_PROXY are unset",
			env:  map[string]string{"ALL_PROXY": "socks5h://from-all:5"},
			want: "socks5h://from-all:5",
		},
		{
			name: "the lower-case name is read too",
			env:  map[string]string{"all_proxy": "http://from-lower:6"},
			want: "http://from-lower:6",
		},
		{
			name: "an empty variable counts as unset",
			env:  map[string]string{"HTTPS_PROXY": "", "ALL_PROXY": "http://from-all:5"},
			want: "http://from-all:5",
		},
		{
			name: "nothing configured means a direct dial",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := make(map[string]string, len(tt.env)+2)
			for key, value := range tt.env {
				vars[key] = value
			}
			vars["HOME"] = t.TempDir()
			vars["XDG_CONFIG_HOME"] = configHome(t, tt.file)
			fakeEnv(t, vars)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Proxy != tt.want {
				t.Errorf("Proxy = %q, want %q", cfg.Proxy, tt.want)
			}
		})
	}
}

func TestNoProxyIsPassedThroughUnchanged(t *testing.T) {
	// A pattern list with every shape NO_PROXY accepts: the point of the test is
	// that this package does not touch it on the way to the transport.
	const listed = "example.com,127.0.0.1,.internal,10.0.0.0/8"

	t.Run("from the environment", func(t *testing.T) {
		fakeEnv(t, map[string]string{
			"HOME":            t.TempDir(),
			"XDG_CONFIG_HOME": configHome(t, `no_proxy = "http://the-file-was-here:1"`),
			"NO_PROXY":        listed,
		})

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.NoProxy != listed {
			t.Errorf("NoProxy = %q, want the environment's own value %q", cfg.NoProxy, listed)
		}
	})

	t.Run("from the file when the environment has none", func(t *testing.T) {
		fakeEnv(t, map[string]string{
			"HOME":            t.TempDir(),
			"XDG_CONFIG_HOME": configHome(t, `no_proxy = "`+listed+`"`),
		})

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.NoProxy != listed {
			t.Errorf("NoProxy = %q, want the file's own value %q", cfg.NoProxy, listed)
		}
	})

	t.Run("the lower-case name", func(t *testing.T) {
		fakeEnv(t, map[string]string{
			"HOME":            t.TempDir(),
			"XDG_CONFIG_HOME": configHome(t, ""),
			"no_proxy":        listed,
		})

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.NoProxy != listed {
			t.Errorf("NoProxy = %q, want %q", cfg.NoProxy, listed)
		}
	})
}

func TestSessionPathPrecedence(t *testing.T) {
	const fromFile = "/tmp/from-the-file/session"

	t.Run("the file beats the default", func(t *testing.T) {
		home := t.TempDir()
		fakeEnv(t, map[string]string{
			"HOME":            home,
			"XDG_CONFIG_HOME": configHome(t, `session_path = "`+fromFile+`"`),
		})

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.SessionPath != fromFile {
			t.Errorf("SessionPath = %q, want the file's value %q", cfg.SessionPath, fromFile)
		}
	})

	t.Run("X_TERMINAL_SESSION beats the file", func(t *testing.T) {
		const fromEnv = "/tmp/from-the-environment/session"
		fakeEnv(t, map[string]string{
			"HOME":               t.TempDir(),
			"XDG_CONFIG_HOME":    configHome(t, `session_path = "`+fromFile+`"`),
			"X_TERMINAL_SESSION": fromEnv,
		})

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.SessionPath != fromEnv {
			t.Errorf("SessionPath = %q, want %q", cfg.SessionPath, fromEnv)
		}
	})
}

func TestUnknownKeyIsAWarningNotAnError(t *testing.T) {
	fakeEnv(t, map[string]string{
		"HOME":            t.TempDir(),
		"XDG_CONFIG_HOME": configHome(t, "session_path = \"/tmp/known/session\"\nnot_a_key = \"x\"\n"),
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with an unknown key: %v", err)
	}
	if len(cfg.Warnings) != 1 {
		t.Fatalf("Warnings = %q, want exactly the one unknown key", cfg.Warnings)
	}
	if !strings.Contains(cfg.Warnings[0], "not_a_key") || !strings.Contains(cfg.Warnings[0], ":2:") {
		t.Errorf("the warning names neither the key nor its line: %q", cfg.Warnings[0])
	}
	// The file was read at all: without this, a Load that returned before
	// parsing would still show one warning.
	if cfg.SessionPath != "/tmp/known/session" {
		t.Errorf("SessionPath = %q, want the file's own value — the file was not read", cfg.SessionPath)
	}
}

func TestMalformedLinesAndValuesKeepTheDefaults(t *testing.T) {
	body := strings.Join([]string{
		"this line is not a setting",
		`request_floor = "soon"`,
		`session_request_cap = "many"`,
		`proxy = "http://proxy.example:8080" # a trailing comment`,
		`auth_token = "this key belongs in the session file"`,
	}, "\n") + "\n"
	fakeEnv(t, map[string]string{"HOME": t.TempDir(), "XDG_CONFIG_HOME": configHome(t, body)})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with a malformed config file: %v", err)
	}
	if len(cfg.Warnings) != 4 {
		t.Fatalf("Warnings = %q, want one for each of the four bad lines", cfg.Warnings)
	}
	if cfg.RequestFloor != DefaultRequestFloor {
		t.Errorf("RequestFloor = %s, want the default %s", cfg.RequestFloor, DefaultRequestFloor)
	}
	if cfg.SessionRequestCap != DefaultSessionRequestCap {
		t.Errorf("SessionRequestCap = %d, want the default %d", cfg.SessionRequestCap, DefaultSessionRequestCap)
	}
	// The line that was well formed still applied: the warnings are not the
	// whole of what this Load did.
	if cfg.Proxy != "http://proxy.example:8080" {
		t.Errorf("Proxy = %q, want the file's quoted value without the trailing comment", cfg.Proxy)
	}
	// A warning goes to a terminal and to whatever captures it, and this file is
	// allowed to hold a proxy password. No warning may echo a value.
	for _, warning := range cfg.Warnings {
		for _, value := range []string{"soon", "many", "belongs in the session file"} {
			if strings.Contains(warning, value) {
				t.Errorf("warning %q echoes the value it complains about", warning)
			}
		}
	}
}

func TestQuotedValuesAndComments(t *testing.T) {
	body := strings.Join([]string{
		"# the whole line is a comment",
		`session_path = "/tmp/x-terminal/session" # the comment is not part of the value`,
		`media_opener = xdg-open # a bare value ends at a " #"`,
		"",
		`request_floor = "750ms"`,
	}, "\n") + "\n"
	fakeEnv(t, map[string]string{"HOME": t.TempDir(), "XDG_CONFIG_HOME": configHome(t, body)})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with comments and quotes: %v", err)
	}
	if cfg.SessionPath != "/tmp/x-terminal/session" {
		t.Errorf("SessionPath = %q, want the quoted value without the comment", cfg.SessionPath)
	}
	if cfg.MediaOpener != "xdg-open" {
		t.Errorf("MediaOpener = %q, want the bare value without the comment", cfg.MediaOpener)
	}
	if cfg.RequestFloor != 750*time.Millisecond {
		t.Errorf("RequestFloor = %s, want 750ms", cfg.RequestFloor)
	}
	if len(cfg.Warnings) != 0 {
		t.Errorf("a well-formed file warned: %q", cfg.Warnings)
	}
}

func TestUnreadableConfigFileIsAnError(t *testing.T) {
	home := configHome(t, "")
	if err := os.MkdirAll(filepath.Join(home, appDir, ConfigFileName), 0o700); err != nil {
		t.Fatalf("making %s a directory: %v", ConfigFileName, err)
	}
	fakeEnv(t, map[string]string{"HOME": t.TempDir(), "XDG_CONFIG_HOME": home})

	if _, err := Load(); err == nil {
		t.Fatal("Load succeeded with a config.toml it could not read — the run would not be the configuration the user wrote")
	}
}

func TestRedactedDropsProxyCredentials(t *testing.T) {
	tests := []struct {
		name  string
		proxy string
		want  string
	}{
		{"userinfo is dropped", "http://alice:hunter2@proxy.example:8080", "http://proxy.example:8080"},
		{"a socks5 proxy is kept whole", "socks5h://127.0.0.1:7897", "socks5h://127.0.0.1:7897"},
		{"nothing configured stays nothing", "", ""},
		{"a setting that does not parse is not echoed", "://hunter2@nonsense", "<set>"},
		{"a setting with no host is not echoed", "/just/a/path/hunter2", "<set>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Proxy: tt.proxy}
			redacted := cfg.Redacted()

			if redacted.Proxy != tt.want {
				t.Errorf("Redacted().Proxy = %q, want %q", redacted.Proxy, tt.want)
			}
			if tt.proxy != "" && strings.Contains(redacted.Proxy, "hunter2") {
				t.Errorf("the redacted proxy still holds the credential: %q", redacted.Proxy)
			}
			// Redacted is a copy: the value the transport has to dial through
			// must survive it.
			if cfg.Proxy != tt.proxy {
				t.Errorf("Redacted modified its receiver: %q", cfg.Proxy)
			}
		})
	}
}

// displayConfig writes body as config.toml in a fresh home and returns the home and the file's
// path. A body of "" writes no file, which is the state the created-file test is about.
func displayConfig(t *testing.T, body string) (home, path string) {
	t.Helper()
	home = configHome(t, body)
	return home, filepath.Join(home, appDir, ConfigFileName)
}

// displayEnv points the package at a fixture: XDG_CONFIG_HOME is the home whose x-terminal
// directory holds the fixture's config.toml, and HOME is a directory of its own so that the data
// directory resolves without touching the developer's.
func displayEnv(t *testing.T, home string) {
	t.Helper()
	fakeEnv(t, map[string]string{"XDG_CONFIG_HOME": home, "HOME": t.TempDir()})
}

// TestDisplayKeysDefaultToThePlainReaderAndTheFirstStyle pins the two defaults by their spelling
// rather than by the constants that name them: the constants are what the code reads and the
// literals are what the file holds, and a default that quietly became a different word would be a
// start-up state the UI-SPEC does not describe.
func TestDisplayKeysDefaultToThePlainReaderAndTheFirstStyle(t *testing.T) {
	displayEnv(t, configHome(t, ""))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with no config file: %v", err)
	}
	if cfg.DisplayState != "normal" {
		t.Errorf("DisplayState = %q with nothing set, want %q", cfg.DisplayState, "normal")
	}
	if cfg.DisplayCover != "servicelog" {
		t.Errorf("DisplayCover = %q with nothing set, want %q", cfg.DisplayCover, "servicelog")
	}
	if len(cfg.Warnings) != 0 {
		t.Errorf("the default configuration warned: %q", cfg.Warnings)
	}
}

// TestDisplayKeysResolveFromTheFile is both vocabularies, value by value: every state pairs with
// every style, and the pair the file names is the pair the resolved configuration carries.
func TestDisplayKeysResolveFromTheFile(t *testing.T) {
	for _, state := range DisplayStates() {
		for _, cover := range DisplayCovers() {
			home, _ := displayConfig(t, fmt.Sprintf("display.state = %q\ndisplay.cover = %q\n", state, cover))
			displayEnv(t, home)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load with display.state = %q: %v", state, err)
			}
			if cfg.DisplayState != state {
				t.Errorf("DisplayState = %q, want %q", cfg.DisplayState, state)
			}
			if cfg.DisplayCover != cover {
				t.Errorf("DisplayCover = %q, want %q", cfg.DisplayCover, cover)
			}
			if len(cfg.Warnings) != 0 {
				t.Errorf("display.state = %q warned: %q", state, cfg.Warnings)
			}
		}
	}
	// Anti-vacuity: the loop has something to walk.
	if len(DisplayStates()) != 3 || len(DisplayCovers()) != 4 {
		t.Fatalf("the vocabularies hold %d states and %d styles, want 3 and 4", len(DisplayStates()), len(DisplayCovers()))
	}
}

// TestAnUnknownDisplayValueKeepsTheDefaultAndWarns is the file's established rule at the two new
// keys: the run starts, the default stands, the warning names the file and the line, and the
// warning never holds the value — a warning goes to a terminal and to whatever captures it.
func TestAnUnknownDisplayValueKeepsTheDefaultAndWarns(t *testing.T) {
	for _, tc := range []struct {
		name  string
		body  string
		state string
		cover string
	}{
		{"an unknown state", "display.state = \"Panic\"\n", DefaultDisplayState, DefaultDisplayCover},
		{"an unknown style", "display.cover = \"servicelog-2\"\n", DefaultDisplayState, DefaultDisplayCover},
		{"wrong case is unknown", "display.state = \"Immersive\"\ndisplay.cover = \"GitLog\"\n", DefaultDisplayState, DefaultDisplayCover},
		{"the plain reader is not a style", "display.cover = \"plain\"\n", DefaultDisplayState, DefaultDisplayCover},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, _ := displayConfig(t, tc.body)
			displayEnv(t, home)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load refused a file with an unknown display value: %v", err)
			}
			if cfg.DisplayState != tc.state || cfg.DisplayCover != tc.cover {
				t.Errorf("the values resolved to (%q, %q), want (%q, %q)", cfg.DisplayState, cfg.DisplayCover, tc.state, tc.cover)
			}
			if len(cfg.Warnings) == 0 {
				t.Fatalf("an unknown display value warned nothing")
			}
			for _, warning := range cfg.Warnings {
				if !strings.HasPrefix(warning, ConfigFileName+":") {
					t.Errorf("the warning %q does not name the file and the line", warning)
				}
				if !strings.Contains(warning, "is not one of") {
					t.Errorf("the warning %q does not say which values are accepted", warning)
				}
				// The value it complains about is never echoed. Every value this table wrote is
				// checked, so a warning that held one is caught whichever key it was about.
				for _, value := range []string{"Panic", "Immersive", "GitLog", "servicelog-2", "plain"} {
					if strings.Contains(warning, value) {
						t.Errorf("the warning %q holds the value %q", warning, value)
					}
				}
			}
		})
	}
}

// TestSetDisplayRoundTrips is the writer's contract against the reader: what SetDisplay writes, Load
// reads back.
func TestSetDisplayRoundTrips(t *testing.T) {
	home, path := displayConfig(t, "proxy = \"http://proxy.example:8080\"\n")
	displayEnv(t, home)

	if err := SetDisplay(path, DisplayStateImmersive, DisplayCoverGitLog); err != nil {
		t.Fatalf("SetDisplay: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load after SetDisplay: %v", err)
	}
	if cfg.DisplayState != DisplayStateImmersive || cfg.DisplayCover != DisplayCoverGitLog {
		t.Errorf("the round trip resolved (%q, %q), want (%q, %q)", cfg.DisplayState, cfg.DisplayCover, DisplayStateImmersive, DisplayCoverGitLog)
	}
	if cfg.Proxy != "http://proxy.example:8080" {
		t.Errorf("the round trip lost the file's other key: proxy = %q", cfg.Proxy)
	}
	if len(cfg.Warnings) != 0 {
		t.Errorf("the writer's own output warned when it was read back: %q", cfg.Warnings)
	}
}

// TestSetDisplayTouchesExactlyTwoLines is the property the writer exists for: everything else in the
// file — the comments, the blank lines, the unknown keys and the keys of other phases — is the
// user's writing and comes back byte for byte.
func TestSetDisplayTouchesExactlyTwoLines(t *testing.T) {
	body := strings.Join([]string{
		"# the reader's own file",
		"",
		"display.state = \"normal\"   # where the pane opens",
		"proxy = \"http://proxy.example:8080\"",
		"# a comment in the middle",
		"display.cover=servicelog",
		"session_request_cap = 200",
		"",
	}, "\n")
	_, path := displayConfig(t, body)

	if err := SetDisplay(path, DisplayStatePanic, DisplayCoverCI); err != nil {
		t.Fatalf("SetDisplay: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the file back: %v", err)
	}

	want := strings.Join([]string{
		"# the reader's own file",
		"",
		"display.state = \"panic\"",
		"proxy = \"http://proxy.example:8080\"",
		"# a comment in the middle",
		"display.cover = \"ci\"",
		"session_request_cap = 200",
		"",
	}, "\n")
	if string(got) != want {
		t.Errorf("the file is:\n%q\nwant:\n%q", got, want)
	}
	if lines, before := strings.Split(string(got), "\n"), strings.Split(body, "\n"); len(lines) != len(before) {
		t.Errorf("the line count moved from %d to %d", len(before), len(lines))
	}
}

// TestSetDisplayAppendsWhatTheFileDoesNotHold: a file naming neither key — or only one of them —
// gains the missing lines and keeps the ones it had.
func TestSetDisplayAppendsWhatTheFileDoesNotHold(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"neither", "# nothing but a comment\n"},
		{"only the state", "display.state = \"normal\"\n"},
		{"only the cover", "display.cover = \"ci\"\n"},
		{"no trailing newline", "proxy = \"http://proxy.example:8080\""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, path := displayConfig(t, tc.body)
			displayEnv(t, home)

			if err := SetDisplay(path, DisplayStateImmersive, DisplayCoverInternal); err != nil {
				t.Fatalf("SetDisplay: %v", err)
			}
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load after SetDisplay: %v", err)
			}
			if cfg.DisplayState != DisplayStateImmersive || cfg.DisplayCover != DisplayCoverInternal {
				t.Errorf("the appended file resolved (%q, %q), want (%q, %q)", cfg.DisplayState, cfg.DisplayCover, DisplayStateImmersive, DisplayCoverInternal)
			}
			if len(cfg.Warnings) != 0 {
				t.Errorf("the appended file warned: %q", cfg.Warnings)
			}
		})
	}
}

// TestSetDisplayCreatesTheFile: a missing file is not an error. The two lines are written into a new
// one, and a file this tool creates is 0600 because it holds the user's settings.
func TestSetDisplayCreatesTheFile(t *testing.T) {
	home, path := displayConfig(t, "")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("the fixture left a file at %s", path)
	}
	displayEnv(t, home)

	if err := SetDisplay(path, DisplayStateImmersive, DisplayCoverServiceLog); err != nil {
		t.Fatalf("SetDisplay on a missing file: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("the file was not created: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("the created file is %o, want 0600", perm)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load after the file was created: %v", err)
	}
	if cfg.DisplayState != DisplayStateImmersive || cfg.DisplayCover != DisplayCoverServiceLog {
		t.Errorf("the created file resolved (%q, %q)", cfg.DisplayState, cfg.DisplayCover)
	}
}

// TestSetDisplayLeavesNoTemporaryFile: the write goes through a temporary file in the same
// directory and a rename. The renaming is what makes the file never half-written; a temporary file
// left behind would be a second, stale configuration sitting next to the real one.
func TestSetDisplayLeavesNoTemporaryFile(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"an existing file", "display.state = \"normal\"\ndisplay.cover = \"ci\"\n"},
		{"a created file", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, path := displayConfig(t, tc.body)
			if err := SetDisplay(path, DisplayStateNormal, DisplayCoverServiceLog); err != nil {
				t.Fatalf("SetDisplay: %v", err)
			}
			entries, err := os.ReadDir(filepath.Dir(path))
			if err != nil {
				t.Fatalf("reading the config directory: %v", err)
			}
			names := make([]string, 0, len(entries))
			for _, entry := range entries {
				names = append(names, entry.Name())
			}
			if len(names) != 1 || names[0] != ConfigFileName {
				t.Errorf("the directory holds %q, want %q alone", names, ConfigFileName)
			}
		})
	}
}

// TestSetDisplayRefusesAnEmptyPath: the writer never guesses where the configuration lives.
func TestSetDisplayRefusesAnEmptyPath(t *testing.T) {
	if err := SetDisplay("", DisplayStateNormal, DefaultDisplayCover); err == nil {
		t.Fatal("SetDisplay wrote without a path")
	}
}

func TestSavedDBPathBesidePinsAndNotice(t *testing.T) {
	cfg := Config{DataDir: "/tmp/x-data"}
	if got, want := cfg.SavedDBPath(), "/tmp/x-data/saved.db"; got != want {
		t.Fatalf("SavedDBPath = %q, want %q", got, want)
	}
	if cfg.PinsPath() == "" || cfg.NoticePath() == "" {
		t.Fatal("PinsPath/NoticePath empty with a DataDir set")
	}
	if (Config{}).SavedDBPath() != "" {
		t.Fatal("empty DataDir must yield an empty SavedDBPath")
	}
}

// TestPinsPathIsTheRegistersName: the pins file's *name* is the registry package's — the file's own
// shape and its reader and writer live beside that constant — and its *location* is this package's,
// so the path is built from the name and the two cannot drift. The name itself is pinned here as
// well, because the operator's existing pins file is already at it: renaming the constant is a
// migration rather than an edit, and it should fail an arm before it fails a user.
func TestPinsPathIsTheRegistersName(t *testing.T) {
	if volatile.PinsFileName != "pins.json" {
		t.Errorf("the pins file's name is %q, want pins.json: an operator's pins file is at that name, so a rename is a migration", volatile.PinsFileName)
	}
	cfg := Config{DataDir: "/tmp/x-data"}
	if got, want := cfg.PinsPath(), "/tmp/x-data/pins.json"; got != want {
		t.Errorf("PinsPath = %q, want %q", got, want)
	}
	if (Config{}).PinsPath() != "" {
		t.Error("a zero Config must yield an empty PinsPath: no data directory, no pins file")
	}
}

// localeEnv points the package at a fixture whose locale is the test's own: XDG_CONFIG_HOME is the
// home holding the fixture's config.toml, HOME is a directory of its own so the data directory
// resolves without touching the developer's, and LC_ALL/LANG are whatever the arm passes. A name the
// arm does not set reads as unset, so the shell this test runs under cannot change what it sees.
func localeEnv(t *testing.T, home string, locale map[string]string) {
	t.Helper()
	vars := map[string]string{"XDG_CONFIG_HOME": home, "HOME": t.TempDir()}
	for name, value := range locale {
		vars[name] = value
	}
	fakeEnv(t, vars)
}

// TestTheTranslateLanguageResolvesFromTheFileThenTheLocaleThenTheDefault pins the setting's three
// sources and their order, one arm at a time: the file is the user's deliberate choice, the locale is
// the environment's own statement about this host, and the built-in default is what is left when
// neither speaks.
func TestTheTranslateLanguageResolvesFromTheFileThenTheLocaleThenTheDefault(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		locale map[string]string
		want   string
	}{
		{name: "nothing set is the default", want: DefaultTranslateLanguage},
		{name: "the locale wins over the default", locale: map[string]string{"LANG": "zh_CN.UTF-8"}, want: "zh-CN"},
		{name: "LC_ALL wins over LANG", locale: map[string]string{"LC_ALL": "ja_JP.UTF-8", "LANG": "de_DE.UTF-8"}, want: "ja-JP"},
		{name: "an LC_ALL naming no language falls through to LANG", locale: map[string]string{"LC_ALL": "C.UTF-8", "LANG": "de_DE.UTF-8"}, want: "de-DE"},
		{name: "a host with no locale at all keeps the default", locale: map[string]string{"LC_ALL": "POSIX", "LANG": "C"}, want: DefaultTranslateLanguage},
		{name: "the file wins over the locale", body: "translate.language = \"ja\"\n", locale: map[string]string{"LANG": "de_DE.UTF-8"}, want: "ja"},
		{name: "the file wins over nothing being set", body: "translate.language = \"en\"\n", want: "en"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			localeEnv(t, configHome(t, tc.body), tc.locale)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.TranslateLanguage != tc.want {
				t.Errorf("TranslateLanguage = %q, want %q", cfg.TranslateLanguage, tc.want)
			}
			if len(cfg.Warnings) != 0 {
				t.Errorf("a resolvable language warned: %q", cfg.Warnings)
			}
		})
	}
}

// TestTheLocaleReaderTurnsOneLocaleIntoATag is the reader's own table, without a subprocess: the
// codeset and the modifier drop, the subtags are spelled and cased the way a tag spells them, and a
// value that names no language — the C locale an unconfigured host exports, and anything that is not a
// tag this build will ask the endpoint for — is refused rather than passed through.
func TestTheLocaleReaderTurnsOneLocaleIntoATag(t *testing.T) {
	for _, tc := range []struct {
		locale string
		want   string
	}{
		{"zh_CN.UTF-8", "zh-CN"},
		{"de_DE@euro", "de-DE"},
		{"en_US.UTF-8", "en-US"},
		{"ja", "ja"},
		{"zh-Hans-CN", "zh-HANS-CN"},
		{"zh_Hans_CN", "zh-HANS-CN"},
		{"", ""},
		{"C", ""},
		{"POSIX", ""},
		{"C.UTF-8", ""},
		{"en US", ""},
		{"zh(CN)", ""},
		{"some,thing", ""},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			if got := languageTag(tc.locale); got != tc.want {
				t.Errorf("languageTag(%q) = %q, want %q", tc.locale, got, tc.want)
			}
		})
	}
}

// TestAMalformedTranslateLanguageKeepsWhatItWouldOtherwiseHaveHad: the file's established rule at the
// new key — the run starts, the value the file would otherwise have had stands (here the locale's, not
// the built-in default), and the warning names the file and the line and never echoes the value.
func TestAMalformedTranslateLanguageKeepsWhatItWouldOtherwiseHaveHad(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"an empty value", "translate.language = \"\"\n"},
		{"a locale spelling rather than a tag", "translate.language = \"zh_CN\"\n"},
		{"a value holding the column expression's own punctuation", "translate.language = \"Some(zh-CN)\"\n"},
		{"a value holding a separator", "translate.language = \"zh-CN,en\"\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			localeEnv(t, configHome(t, tc.body), map[string]string{"LANG": "ja_JP.UTF-8"})

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load refused a file with a malformed language: %v", err)
			}
			if cfg.TranslateLanguage != "ja-JP" {
				t.Errorf("TranslateLanguage = %q, want the locale's own ja-JP: a refused value keeps the value it would otherwise have had", cfg.TranslateLanguage)
			}
			if len(cfg.Warnings) == 0 {
				t.Fatalf("a malformed language warned nothing")
			}
			for _, warning := range cfg.Warnings {
				if !strings.HasPrefix(warning, ConfigFileName+":") {
					t.Errorf("the warning %q does not name the file and the line", warning)
				}
				for _, value := range []string{"Some(zh-CN)", "zh_CN", "zh-CN,en"} {
					if strings.Contains(warning, value) {
						t.Errorf("the warning %q holds the value %q", warning, value)
					}
				}
			}
		})
	}
}

// TestTheTranslateLanguageIsASettingNotACredential: Redacted is the form every report prints, and the
// language has to survive it — the doctor prints it, so a redaction that dropped it would make the
// presence fact a lie. It is the counter-arm to the proxy test above.
func TestTheTranslateLanguageIsASettingNotACredential(t *testing.T) {
	cfg := Config{Proxy: "http://user:password@proxy.example:8080", TranslateLanguage: "de"}
	if got := cfg.Redacted(); got.Proxy != "http://proxy.example:8080" {
		t.Errorf("Redacted().Proxy = %q, want the credentials dropped", got.Proxy)
	}
	if cfg.Redacted().TranslateLanguage != "de" {
		t.Errorf("Redacted() dropped the language: the doctor prints it, and it is a setting")
	}
}

// TestTheAccountKeyIsAName: the file's `account` key selects the start-up account, and a name that
// could be a path is refused with the previous value kept.
func TestTheAccountKeyIsAName(t *testing.T) {
	home := t.TempDir()
	write := func(body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(home, appDir), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, appDir, ConfigFileName), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("account = \"work-laptop_2\"\n")
	t.Setenv("XDG_CONFIG_HOME", home)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Account != "work-laptop_2" {
		t.Fatalf("account = %q", cfg.Account)
	}
	if cfg.SessionsDir != filepath.Join(home, appDir, "sessions") {
		t.Fatalf("sessions dir = %q", cfg.SessionsDir)
	}

	write("account = \"../escape\"\n")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Account != "" {
		t.Fatalf("a path-shaped name was accepted: %q", cfg.Account)
	}
	if len(cfg.Warnings) == 0 {
		t.Fatal("the refusal said nothing")
	}
}
