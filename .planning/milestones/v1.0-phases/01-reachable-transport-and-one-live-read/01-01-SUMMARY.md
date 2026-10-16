---
phase: 01-reachable-transport-and-one-live-read
plan: 01
subsystem: auth
tags: [go, xdg, session-cookie, redaction, slog, proxy, cli-dispatch]

# Dependency graph
requires: []
provides:
  - "go.mod: module github.com/LZD121/x-terminal, go 1.27, zero third-party requires"
  - "internal/cli: one command table (doctor, whoami, --help), strict exit codes, and stubs that cannot pass as finished verbs"
  - "internal/config: XDG paths, the config.toml line reader, and the whole environment read — proxy precedence including ALL_PROXY, NO_PROXY passthrough, X_TERMINAL_SESSION"
  - "internal/session: Secret (redacting) / Session / Load with symlink and mode refusals / Valid / Summary / ErrNoSession"
affects: [01-02, 01-03, phase-2]

# Actuals — pairs with the plan's estimate to calibrate future estimates.
# chars/4 over the 56,387 bytes actually written (8 files, tests included).
actuals:
  tokens: 14097
  tasks: 3
  commits: 0

# Tech tracking
tech-stack:
  added: []          # stdlib only, as the plan requires
  patterns:
    - "a redacting credential type whose formatting methods are the contract"
    - "an injected environment seam per package, so precedence tests are hermetic"
    - "warnings as data returned to the caller, never written to a stream by a library"

key-files:
  created:
    - go.mod
    - cmd/x-terminal/main.go
    - internal/cli/cli.go
    - internal/config/config.go
    - internal/config/config_test.go
    - internal/session/session.go
    - internal/session/session_test.go
    - internal/session/leak_test.go
  modified: []

key-decisions:
  - "session.CTO is a Secret, not a string: the plan's struct literal says string but its prose says to give ct0 the same treatment, and ct0 alone forges x-csrf-token"
  - "Source is composed per value (path / environment / both), so it never names the file for a value an override replaced"
  - "Config carries the warnings (Load's signature is fixed at two results) and DataDir/ConfigDir, which plans 02 and 03 need"
  - "No warning and no refusal ever echoes a value; the session loader will not echo a 'key' that is not a short identifier, because that file is where a cookie can be pasted on the wrong side of the ="
  - "Secret deliberately does not implement fmt.Formatter: %v must keep going through String(), which is what the plan's revert check exercises"
  - "a relative XDG_* value is ignored in favour of $HOME, the rule os.UserConfigDir follows"

patterns-established:
  - "Exit codes: 0 only for a body's own success, 1 for a failed run, 2 for a bad invocation and for a verb with no body yet"
  - "Usage is rendered from the command table, so the surface and its copy cannot drift"
  - "Refusals name the exact fix (chmod 600 <path>) and the thing refused, and print no credential"
  - "Anti-vacuity arms: a scan that found nothing, formatted nothing, or holds a needle it cannot find fails the test instead of passing it"

requirements-completed: [AUTH-01, AUTH-02, AUTH-04, SEC-01]

coverage:
  - id: D1
    description: "A buildable Go module (github.com/LZD121/x-terminal, go 1.27) with no third-party requirements, dispatching doctor/whoami/--help and rejecting unknown verbs"
    requirement: ""
    verification:
      - kind: manual_procedural
        ref: "go build ./... && go vet ./... && go test ./...; go run ./cmd/x-terminal (usage naming doctor and whoami, exit 2); go run ./cmd/x-terminal doctor (\"not implemented yet\", exit 2); go run ./cmd/x-terminal bogus (\"unknown command: bogus\" + usage, exit 2); go run ./cmd/x-terminal --help (usage on stdout, exit 0); cat go.mod"
        status: pass
    human_judgment: false
  - id: D2
    description: "Config resolution: XDG paths with the $HOME fallback, the config.toml reader, and proxy precedence X_TERMINAL_PROXY > file > HTTPS_PROXY > HTTP_PROXY > ALL_PROXY > direct, with NO_PROXY passed through"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/config/config_test.go#TestProxyPrecedence (incl. \"ALL_PROXY is honoured when HTTPS_PROXY and HTTP_PROXY are unset\"), #TestDefaultsResolveFromHomeWhenXDGIsUnset, #TestXDGBaseDirectoriesWinOverHome, #TestNoProxyIsPassedThroughUnchanged, #TestMalformedLinesAndValuesKeepTheDefaults, #TestUnknownKeyIsAWarningNotAnError, #TestSessionPathPrecedence, #TestRedactedDropsProxyCredentials"
        status: pass
    human_judgment: false
  - id: D3
    description: "A session loader that refuses a symlink and a mode looser than 0600 with the exact chmod command, accepts 0600, honours X_TERMINAL_AUTH_TOKEN/X_TERMINAL_CT0 over the file, and returns typed ErrNoSession when there is nothing to read"
    requirement: AUTH-01
    verification:
      - kind: unit
        ref: "internal/session/session_test.go#TestLoadAcceptsMode0600, #TestLoadRefusesLooseMode, #TestLoadRefusesSymlink, #TestEnvironmentOverridesWin, #TestBothOverridesWithoutAFileAreASession, #TestMissingFileWithoutOverridesIsErrNoSession, #TestHalfASessionIsIncomplete, #TestUnknownKeyWarnsAndTheKnownKeysStillLoad"
        status: pass
    human_judgment: false
  - id: D4
    description: "The credential cannot be printed: %v, %+v, %#v, JSON, text marshalling, slog and wrapped/joined errors all yield <redacted>, proven by captured output with four anti-vacuity arms"
    requirement: SEC-01
    verification:
      - kind: unit
        ref: "internal/session/leak_test.go#TestSecretCannotLeak"
        status: pass
      - kind: manual_procedural
        ref: "temporarily changed String() to return string(s), ran go test ./internal/session/... -run TestSecretCannotLeak -count=1 (FAILED at both credentials, offsets 11 and 116), reverted"
        status: pass
    human_judgment: false
  - id: D5
    description: "The real session file at ~/.config/x-terminal/session loads and reports lengths and presence facts only (40-byte auth_token, 160-byte ct0), and the dump verbs redact the real bytes"
    requirement: SEC-01
    verification:
      - kind: manual_procedural
        ref: "temporary in-module cmd/xterm-probe (since deleted): session: present (auth_token 40 bytes, ct0 160 bytes, source /home/user/.config/x-terminal/session); %+v and %#v printed <redacted>; output screened for any 30+ character token-shaped run before being read"
        status: pass
    human_judgment: false

# Metrics
duration: ~12min
completed: 2026-10-14
status: complete
---

# Phase 1: Reachable Transport and One Live Read Summary

**The repository is now a buildable Go module whose credential path is closed by construction: a `0600`-only, symlink-refusing session loader wrapped in a `Secret` type that renders as `<redacted>` through every formatting path, plus CLI dispatch and a fully tested config resolver that reads `ALL_PROXY` because the standard library does not.**

## Performance

- **Duration:** ~12 min (start time approximate; first recorded activity 13:15 local, verification finished 13:27:03 +08:00)
- **Started:** 2026-10-14T13:15+08:00
- **Completed:** 2026-10-14T13:27:03+08:00
- **Tasks:** 3
- **Files modified:** 8 (all created; nothing existed in the tree but `.planning/`)

## Accomplishments

- **A module with one decision table, not a framework.** `internal/cli` holds the whole surface — `doctor`, `whoami`, `--help` — and renders usage from that table, so a verb cannot exist without appearing in the help. Unknown verbs get `unknown command: X` plus the usage on stderr and exit 2; a verb whose body has not landed exits 2 with `not implemented yet` rather than printing nothing and passing as finished.
- **Config precedence proved, not assumed.** `internal/config` is the only package in the repository that reads the environment. Its test table covers `X_TERMINAL_PROXY` > file > `HTTPS_PROXY` > `HTTP_PROXY` > `ALL_PROXY` > direct, including the row the standard library would fail (`ALL_PROXY` alone, honoured) and the lower-case aliases. `NO_PROXY` is passed through as given; `Redacted()` drops proxy userinfo and refuses to echo an unparseable setting.
- **The credential cannot be printed.** `session.Secret`'s `String`, `GoString`, `MarshalJSON`, `MarshalText` and `LogValue` all yield `<redacted>`; `Reveal()` is the single named escape hatch and has no production caller. The loader refuses a symlink, refuses a mode looser than `0600` with the exact `chmod 600 <path>` command, takes `auth_token`/`ct0` from either the file or the environment, and returns typed `ErrNoSession` (whose message says where to put the cookie and that it dies with the browser) when there is nothing to read.
- **The redaction contract is proven by output, not by inspection.** `leak_test.go` concatenates `%v`, `%+v`, `%#v`, `%s`, `%q`, `json.Marshal`, `MarshalText`, a `slog` line, a `slog` error and an `errors.Join` carrying a credential, then searches it for both fixture cookies. Four arms keep it honest: the haystack is non-empty, it does say `<redacted>`, `Reveal()` really does hold the fixture, and `strings.Contains` really can find that fixture. Verified to fail when `String()` is made to return the raw value (both credentials were reported, with the offending sinks quoted) and reverted.

## Task Commits

**None made by this executor.** The orchestrator commits this plan's files; no git command was run here and no hashes are claimed.

## Files Created/Modified

- `go.mod` — module `github.com/LZD121/x-terminal`, `go 1.27`, no `require` block at all.
- `cmd/x-terminal/main.go` — the process boundary: `os.Exit(cli.Main(os.Args[1:]))` and nothing else.
- `internal/cli/cli.go` — exit codes, `Context`, `Handler`, the command table, `Main`, `usage` rendered from the table, and the `stub` body.
- `internal/config/config.go` — `Config`, `Load`, `Redacted`, the `env` seam, proxy precedence, `xdgDir`, and the ~40-line `config.toml` reader with value-free warnings.
- `internal/config/config_test.go` — 12 tests (30 cases) over defaults, XDG precedence, malformed files, unknown keys, `NO_PROXY`, `X_TERMINAL_SESSION` and redaction.
- `internal/session/session.go` — `Redacted`, `Secret` with its five formatting methods and `Reveal`, `Session`, `Valid`, `Summary`, `ErrNoSession`, `Load`, the file parser and its key guard.
- `internal/session/session_test.go` — 11 tests over the acceptance list plus the incomplete-session, empty-file and pasted-value cases.
- `internal/session/leak_test.go` — the SEC-01 gate and its four anti-vacuity arms.

## Decisions Made

1. **`Session.CTO` is a `Secret`, not a `string`.** The plan's struct literal writes `CTO string`; its own prose then says to give `ct0` the same treatment. `ct0` alone forges `x-csrf-token`, so the prose won: `Session` has no printable field. (Consequence for plan 03: `whoami` reads `sess.CTO != ""` for its "ct0 present" fact, and the request builder calls `Reveal()`.)
2. **`Session.Source` is provenance per value.** "path", "environment", or "path, environment" when the file supplied one half and an override the other. Accumulating origins per attempt would have named the file for a value the override had already replaced — my own first implementation did exactly that, and `TestEnvironmentOverridesWin` caught it.
3. **Warnings ride in the structs.** The plan fixes `Load() (Config, error)` and `session.Load(path) (Session, error)`, so the unknown-key/malformed-line reports are `Config.Warnings` and `Session.Warnings` — data for the caller, never a library writing to the user's terminal.
4. **No warning echoes a value, and the session loader guards the key too.** A config key is the user's identifier and is named; a session-file "key" is only echoed when it is a short identifier, because that file is where a cookie can be pasted on the wrong side of the `=`.
5. **`Secret` deliberately does not implement `fmt.Formatter`.** With a `Format` method, `%v` would stop reaching `String()` and the plan's mandated revert check ("change `String()` to return the raw value and watch the test fail") would no longer fail — the specification's method set is what makes that check load-bearing. The residual boundary: an unnamed verb (`%d`) reaches fmt's `badVerb` path and would print `%!d(string=<value>)`. No code does it, and the leak test's haystack covers the verbs a caller actually reaches for. Worth closing in a later phase if a `Format` method can keep `String()` load-bearing.
6. **A relative `XDG_CONFIG_HOME`/`XDG_DATA_HOME` is ignored** in favour of `$HOME`, matching `os.UserConfigDir`; an *existing but unreadable* `config.toml` is a hard error (a run that silently used different settings would be the lie this project exists to avoid), while a missing one is the defaults.
7. **`--help` is the one path that gets what it asked for:** usage on stdout, exit 0. No args, an unknown verb and a verb with no body all print to stderr and exit 2. `ExitUsage` and `ExitUnimplemented` share the value 2 deliberately — both mean "nothing ran".
8. **`Config` carries `ConfigDir` and `DataDir`** beyond the plan's "at least" list, because plan 01-03 needs `<datadir>` for `pins.json` and `notice-shown`.

## Deviations from Plan

1. **`CTO Secret` instead of `CTO string`** — see decision 1. Within the plan's own instruction ("give it the same treatment for safety").
2. **Extra fields** (`Config.ConfigDir`, `Config.DataDir`, `Config.Warnings`, `Session.Warnings`) — the plan's struct literals are minimums ("holding at least"), and the fixed `Load` signatures leave warnings nowhere else to go.
3. **A temporary `cmd/xterm-probe/main.go` was created to run the plan's `<verification>` item 4** (the "`session-probe ~/.config/x-terminal/session`-style manual check"). The command table is `doctor`/`whoami`/`--help` by Task 1's own specification, so no `session-probe` subcommand exists; Go's `internal/` visibility rule also forbids an out-of-module probe (first attempt failed with `use of internal package … not allowed`). The probe was run, its output screened for token-shaped runs before being read, and the directory deleted; `go build ./...`, `go vet ./...`, `go test ./...` were re-run cleanly afterwards.
4. **No `go.mod` `require` block is present at all** rather than an empty one — `go 1.27` accepts a module file with no requirements, and there is nothing to require.

## Issues Encountered

- **`Source` claimed the file after an override replaced the value** (found by `TestEnvironmentOverridesWin`, the first test run): per-attempt provenance was replaced with per-value provenance. Fixed in `internal/session/session.go` before the verification block.
- **An out-of-module probe cannot import `internal/…`.** Resolved by the temporary in-module command described above.
- **`fmt`'s nested-`Stringer`/`GoStringer` behaviour was checked empirically before writing the type** (`%v`, `%+v`, `%#v`, `%q`, JSON, `slog`, pointers and `errors.Join`), so the redaction design rests on measured behaviour rather than on a reading of the fmt documentation.

## User Setup Required

None. The session file already exists at `~/.config/x-terminal/session` with mode 0600 and was loaded successfully; no keyring, no daemon and no service configuration is involved. The proxy is unset in a non-interactive shell, which the resolver reports as a direct dial — the supported default, not a broken state.

## Next Phase Readiness

- **Plan 01-02 can start immediately.** `config.Config` exposes `Proxy`, `NoProxy`, `RequestFloor`, `SessionRequestCap`, `MediaOpener`, `SessionPath`, `DataDir`; `session.Session` exposes `AuthToken`/`CTO` (as `Secret`), `Valid()` and `Summary()` for the doctor line; `internal/cli` dispatches, and its table is where the real `doctor` body replaces `stub` (one line per verb — `internal/cli/doctor.go` will also need that table entry changed, since the plan fixes `Run: stub` in `cli.go`).
- **Verified evidence, all green:** `go build ./...` exit 0; `go vet ./...` exit 0; `go test ./... -count=1` 43 tests in 4 packages, exit 0; `gofmt -l` empty; `cat go.mod` shows no third-party requires.
- **`grep -rn "Reveal()" internal/ cmd/`** matches exactly two lines: the method declaration in `internal/session/session.go:68` and the leak test's anti-vacuity arm at `internal/session/leak_test.go:88`. No production call site exists yet, as the plan's acceptance criterion requires.
- **AUTH-04 is only half-satisfied by this plan.** The "never writes the cookie back to any other file" half is structural — no file in `internal/` or `cmd/` opens anything for writing (`grep` for `os.WriteFile|os.Create|os.OpenFile|os.MkdirAll` outside tests is empty) — but the plan's tasks contain no delete command, so `x-terminal` cannot yet remove the local session copy. The requirement ID is carried here because the plan's frontmatter lists it; the delete verb has to land elsewhere in Phase 1.
- **Known boundary for a later phase:** `%d` (and any other verb `String` does not serve) on a `Secret` reaches fmt's `badVerb` path. See decision 5 for why closing it was left out of this plan.

---
*Phase: 01-reachable-transport-and-one-live-read*
*Completed: 2026-10-14*
