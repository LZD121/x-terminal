---
phase: 03-three-state-disguise-on-the-working-renderer
plan: 02
type: summary
status: complete
requirements: [DSGT-01, DSGT-02, DSGT-03, DSGT-04, DSGT-05, DSGT-11]
verified: |
  go build ./... && go vet ./...            clean
  gofmt -l internal cmd                     empty
  go test ./... -count=1                    613 passed in 19 packages
  grep -rln "os/exec" internal/ --include=*.go | grep -v _test.go   → internal/term/tmux.go only
  go test ./internal/tui/... -run 'TestTheShellCannotReachIO|TestOsExecIsOnlyInInternalTerm|TestNoSecondMeasurement' -count=1   ok
  tmux -L xtermprobe -f /dev/null show -g focus-events   → focus-events off
---

# 03-02 SUMMARY — the disguise wired into the running program

## 1. `internal/term` — the module's only process-spawning package

**API** (`internal/term/term.go`, `internal/term/tmux.go`):

| Value | Shape | Notes |
|-------|-------|-------|
| `Entry(frame, title string) string` | pure builder | the entry's one byte string |
| `type Runner func(name string, args ...string) ([]byte, error)` | seam | the tmux subprocess |
| `func Exec(name string, args ...string) ([]byte, error)` | real runner | `exec.Command(...).Output()`, named in `tmux.go` only |
| `type Sink struct { Out io.Writer; Env func(string) string; Run Runner }` | the sink | zero value is inert per seam |
| `func New(out io.Writer) Sink` | composition helper | `os.Stdout`-shaped output, `os.Getenv`, `Exec` |
| `func (Sink) EnterPanic(frame, title string)` | one write, then tmux | every error swallowed, nothing returned |
| `func (Sink) SetWindow(name string)` | rename alone | no-op without a pane |
| `func (Sink) WindowName() string` | read once at start-up | trimmed; empty on failure or without a pane |
| `func Pane(env func(string) string) string` | `$TMUX_PANE` | `$TMUX` is never read |
| `ClearHistoryArgs(pane)`, `RenameWindowArgs(pane, name)`, `WindowNameArgs()` | `[]string` | the three argument vectors, in one place each |

**The entry's byte layout**, in order, in one `Write`:

```
ESC[<row>;1H <row-1 text>   × every frame row (row 1 is ESC[H, the library's own home spelling)
ESC[3J                      ED 3, ansi.EraseEntireDisplay
ESC]2;<title>BEL            ansi.SetWindowTitle
```

No newline anywhere: a row padded to the pane's width leaves the wrap pending, and a trailing newline
would scroll the frame's first row off the pane. `ESC[3J` is the byte string Probe D measured; the
plan's note that the library has no constant for it is wrong — `ansi.EraseEntireDisplay` is exactly
`\x1b[3J` and is used rather than a hand-spelled literal.

**The tmux commands**, dispatched in this order inside `EnterPanic`, after the write, only when
`$TMUX_PANE` is non-empty:

```
tmux clear-history -t <pane>
tmux rename-window -t <pane> <title>
tmux display-message -p '#W'      (WindowName, at start-up, and only with a pane)
```

## 2. `internal/keys` — the one table

Seven rows added, all group `view`, plain wording rendered verbatim from the UI-SPEC, covered
wording = the UI-SPEC's covered table (covered text is empty for every other row):

| Key(s) | Action | Help | HelpCovered |
|--------|--------|------|-------------|
| `m` | `cover` | `wear the feed as work-software output, or take it off` | `switch display mode` |
| `c` | `next-cover` | `next cover style` | `next display style` |
| `1`–`4` | `cover-1`…`cover-4` | `cover style 1`…`cover style 4` | `display style 1`…`display style 4` |
| `ctrl+p` | `panic` | `replace the screen now, and again to come back` | `switch to the idle screen` |

Six Phase 2 rows gained a covered wording: `next`/`prev` → `next record`/`previous record`,
`first`/`last` → `first loaded record`/`last loaded record`, `refresh` →
`poll for new records, keep the place`, `toggle-feed` → `switch streams`.
`Row.HelpText(covered bool) string` is the one accessor; the `keys` subcommand keeps printing
`row.Help` (plain) and its output is unchanged for every Phase 2 row (`cli/keys_test.go` still passes
byte for byte). **The plan says "the four new rows"; there are seven.**

## 3. `internal/config` — the two persisted keys

`display.state` (`normal`|`immersive`|`panic`, default `normal`) → `Config.DisplayState`;
`display.cover` (`servicelog`|`gitlog`|`ci`|`internal`, default `servicelog`) → `Config.DisplayCover`.
The vocabularies are `config.DisplayStates()` / `config.DisplayCovers()`, declared once each, joined to
the shell's and the render package's by `tui.TestDisplayVocabularyAgrees`. An unknown value (including
wrong case) warns with `config.toml:<line>: display.state is not one of normal, immersive, panic;
keeping normal` — file, line and key, never the value.

`SetDisplay(path, state, cover string) error` guarantees: replaces **every** line that sets either key
(the reader takes the last one, so a stale line would disagree), appends whichever key is absent,
leaves every other line byte for byte alone, writes through `config.toml.tmp*` in the same directory
and renames over the original (no temporary file survives), creates the file 0600 and preserves an
existing file's mode. Two additions the plan did not name: an empty path is an error, and the parent
directory is created (`os.MkdirAll`, 0700) — without it the "creates the file when it does not exist"
branch cannot work on a fresh install, which is the state a first run is in.

## 4. `internal/tui` — the three states

`State` (`StateNormal`/`StateImmersive`/`StatePanic`) with `String()` and `StateOf`; `Display{State,
Cover}`; the two-method `Term` port (`EnterPanic`, `SetWindow`); `Deps` gained `Term`, `Display`,
`Persist func(state, cover string)`; `Model` gained `state`, `style`, `from`, `entryPending`.

| # | From | Event | To | Effects as implemented (one command each) |
|---|------|-------|----|-------------------------------------------|
| T1 | normal | `m` | immersive, remembered style | `SetWindow(style title)`; `Persist("immersive", style)` |
| T2 | immersive | `m` | normal | `SetWindow(x-terminal)`; `Persist("normal", style)` |
| T3 | normal | `c` | immersive, next style | as T1 with `display.cover` = the next style |
| T4 | immersive | `c` | immersive, next style in place | as T3; rows, selection and anchor untouched |
| T5 | normal/immersive | `1`–`4` | immersive, that style | as T3, the digit numbering `render.CoverStyles()` |
| T6 | normal/immersive | `ctrl+p` | panic | one command: `Term.EnterPanic(PanicFrame(w,h), "make")`; **no persist, no `SetWindow`** |
| T7 | panic | `ctrl+p` | the state entered from | `SetWindow` that state's title; **no persist** |
| T8 | normal/immersive | `tea.BlurMsg` | panic | identical to T6 (same function) |
| T9 | any | `tea.FocusMsg` | unchanged | nothing, in every state |
| T10 | panic | any key but `ctrl+p`/`ctrl+c` | panic | swallowed before the help check: no overlay, no movement, no request |
| T11 | panic | `ctrl+c` | quit | `tea.Quit`; no write |
| T12 | any | `q`/`esc` at the bottom frame | quit | Phase 2's path; no write |
| T13 | any | quit | — | the composition root renames the window back to the recorded name |

Start-up: `New` reads `Deps.Display`; a start-up `panic` sets `entryPending`, the first
`WindowSizeMsg` with a usable size clears the flag and returns the entry command (the entry runs after
the first frame, never before a size is known). `View` sets `WindowTitle = title()` and
`ReportFocus = true`; `Render` returns `render.PanicFrame(w,h)` in panic and the composed frame
otherwise; `covered()` is the single place the state decides whether a cover is worn, feeding the
grammar, the empty/loading copy, the help wording and `StatusFields.Cover`.

## 5. The boundary test's arms

`internal/tui/boundary_test.go`:

- `TestTheShellCannotReachIO` — forbidden list unchanged; arms: ≥3 files parsed, ≥8 imports collected,
  `charm.land/bubbletea/v2` present, the scanner exercised on a planted source, and **a planted
  `os/exec` in a real package file** — `view.go`'s own source copied into a temp directory with one
  import added — reported by the same `forbiddenIn` scanner (`plantImport`).
- `TestOsExecIsOnlyInInternalTerm` (new) — walks every package directory from the module root (path
  derived from the test file, `go.mod` asserted), parses each with the existing `parsePackage`, and
  asserts every `os/exec` importer is under `internal/term/`. Arms: ≥10 package directories, files ≥
  directories and ≥1 import somewhere, `internal/term/tmux.go` **must** be among the importers, and a
  synthetic source carrying `os/exec` is reported.
- `TestNoSecondMeasurement`, `TestTestFilesAreNotScanned` untouched and still green.

## 6. Commands the work was accepted on

```
go build ./... && go vet ./...                 clean
gofmt -l internal cmd                          empty
go test ./... -count=1                         613 passed in 19 packages
go test ./internal/term/... -count=1 -v        ok (11 tests)
go test ./internal/tui/... -count=1 -v         ok (50 tests)
go test ./internal/keys/... ./internal/config/... -count=1    ok
grep -rln "os/exec" internal/ --include=*.go | grep -v _test.go   → internal/term/tmux.go
tmux -L xtermprobe -f /dev/null show -g focus-events              → focus-events off
```

Break-once checks, each run and reverted:

| Planted | Test | Result |
|---------|------|--------|
| `_ "os/exec"` in `internal/render/cover.go` | `TestOsExecIsOnlyInInternalTerm` | FAIL: "internal/render/cover.go imports os/exec, and internal/term is the module's only package that may"; reverted, passes |
| `_ "os/exec"` in `internal/tui/view.go` | `TestTheShellCannotReachIO` | FAIL: "internal/tui/view.go imports os/exec, which the shell may not reach"; reverted, passes |
| title OSC written before ED 3 in `Entry` | `TestEntryOrder` | FAIL: "the entry does not place every row, then ED 3, then the title"; reverted, passes |

## 7. Reported, not implemented

- **Two golden frames were regenerated**: `internal/render/testdata/TestGoldenFrames/help_{80,120}.golden`.
  Adding rows to the one keymap table necessarily changes the plain `?` overlay, and Phase 3's own
  keymap table adds them. The change is exactly the seven appended lines; every other stored frame is
  byte-identical (verified by checksumming the directory before and after `-update`). The phase's
  "plain frames stay byte-identical" rule therefore holds for the frames that render content, and
  cannot hold for the overlay.
- **`grep -rn "os/exec" internal/ --include=*.go -l` also lists three pre-existing test files**
  (`internal/render/capture_tmux_test.go`, `internal/render/alias_test.go`, plus this plan's
  `internal/tui/boundary_test.go`, which carries the string in planted sources). Non-test files:
  `internal/term/tmux.go` alone. The criterion's own "(tests excluded)" parenthetical is what holds.
- **`WindowName()` starts no subprocess without `$TMUX_PANE`.** The plan describes the read
  unconditionally; without a pane `SetWindow` can never use the value, so the same branch that skips
  the clear skips the read. This also keeps the unit tests from spawning a real tmux.
- **`SetDisplay` creates the config directory** (see §3) — needed for the plan's created-file branch.
- **`tmux show -g focus-events` on the default socket has no server on this host**; the value was read
  on a throwaway server instead (`tmux -L xtermprobe -f /dev/null`), which is the installed tmux's
  own default: `off`. The UI-SPEC's recorded limitation (focus events do not reach a pane until the
  user turns the server option on) is therefore a measurement here, and the app does not enable it.
- The plan's Task 2 says "the four new rows"; the UI-SPEC's table is seven (see §2).
- `TestStateTransitions` covers T1–T9 as rows; T10–T13 are their own tests
  (`TestPanicSwallowsKeys`, `TestTheQuitKeysWriteNothing`) because their assertion is about keys that
  do *nothing* rather than about a state that changes.
