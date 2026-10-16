---
phase: 02-following-timeline-in-the-pane
plan: 04
subsystem: ui
tags: [tui, keymap, bubbletea, back-stack, anchor, status-line, failure-mapper, composition-root, boundary-test, FEED-01, FEED-04, REND-03, REND-05, REND-06, OPS-01, OPS-02]

# Dependency graph
requires:
  - phase: 02-01
    provides: internal/domain (Source port, Post/User/PostPage, Failure/FailureKind) and internal/screen (List.Update + events + EffectPage/EffectRefresh) this shell drives
  - phase: 02-02
    provides: "internal/text — the only string measurement, which the shell never calls"
  - phase: 02-03
    provides: internal/render (Blocks/StateBlock, Frame, StatusFields/StatusLine, Help/HelpRow, StateWord/Remedy, the palette) — the shell composes frames out of it and adds no bytes
provides:
  - "internal/keys — the phase's 13-row keymap table, the keystroke matcher and the one key-list join rule"
  - "internal/tui — Deps/Model/frame, the frame stack with an anchor, the effect-to-command translation, the help overlay, and the two AST scans (the import boundary and A6)"
  - "internal/cli/failure.go — failureFor: every error class the transport can raise mapped to a domain.Failure"
  - "internal/cli/feed.go — the composition root (`feed`): config, registry, session, one transport, one limiter, one client, the adapter, the shell"
  - "internal/cli/keys.go — the third keymap surface, and the `feed`/`keys` rows of the CLI table"
affects: [02-05-golden-frames, phase-03-grammars, phase-04-detail, phase-05-search, phase-06-saved, phase-07-media]

# Actuals — chars/4 over the files actually changed
actuals:
  tokens: 41000
  tasks: 3
  commits: 0

# Tech tracking
tech-stack:
  added:
    - "charm.land/bubbletea/v2 v2.1.0 (pulled through the user's Clash proxy, as the plan recorded; it brings 14 indirect modules and upgrades github.com/mattn/go-runewidth 0.0.27 -> 0.0.30 by MVS)"
  patterns:
    - "the shell is pure: Update/View take values and return values, every read is a tea.Cmd closure, and an AST scan (not a comment) keeps it unable to reach I/O"
    - "one table, three surfaces: internal/keys.Rows() drives input handling, the `?` overlay (through render.HelpRow) and the `keys` subcommand"
    - "the frame stack stores frames by value and mutates only the top one; a result is routed to its frame by feed kind and a result with no frame is dropped"
    - "the anchor is the selected block's *head* row's distance from the pane's first row; the scroll offset is derived (max(0, above-anchor)) and never stored, so a prepend moves the list under a still pane"
    - "the failure is per-process: it lives on the model, the status line reads it whatever frame is on top, and the shell's own gate (RequestAllowed) refuses keys a stopped state forbids"
    - "the budget line reads the one limiter the composition built, and Deps.Budget is that limiter's own Budget method"

key-files:
  created:
    - internal/keys/keys.go
    - internal/keys/keys_test.go
    - internal/tui/model.go
    - internal/tui/update.go
    - internal/tui/view.go
    - internal/tui/tui_test.go
    - internal/tui/boundary_test.go
    - internal/cli/keys.go
    - internal/cli/keys_test.go
    - internal/cli/failure.go
    - internal/cli/failure_test.go
    - internal/cli/feed.go
    - internal/cli/feed_test.go
  modified:
    - internal/cli/cli.go
    - go.mod
    - go.sum

key-decisions:
  - "the module fetch/tidy line that worked: `HTTPS_PROXY=http://127.0.0.1:7897 HTTP_PROXY=http://127.0.0.1:7897 go get charm.land/bubbletea/v2@v2.1.0` then `HTTPS_PROXY=http://127.0.0.1:7897 HTTP_PROXY=http://127.0.0.1:7897 go mod tidy` (tidy is what makes bubbletea a direct requirement; go.mod ends up with exactly bubbletea v2.1.0 and x/ansi as direct)"
  - "pageMsg carries Refresh and failureMsg carries Kind (the plan's shapes did not): screen.List keeps its refresh latch unexported, so an answer must say whether it belongs to screen.Refreshed or screen.PageFetched, and the plan's own routing rule ('route a result to its frame by domain.HomeKind') cannot be applied to a failure that does not name a kind"
  - "internal/netcheck is NOT on the forbidden-import list, alone among the internal packages: Deps.Failure must name []netcheck.Result, which the plan itself specifies, so the import cannot also be forbidden. The scan keeps the intent instead: netcheck may be imported by model.go alone and may never be called (TestTheShellOnlyNamesNetcheckAsAType, with its own planted-call arm). Reported rather than silently resolved."
  - "toggle-feed pushes a fresh frame AND issues that frame's page one. The plan says 'fetching nothing'; a frame that is never read renders the empty-state copy ('this is an empty read that succeeded - press r to refresh') about a read that never happened, which is the lie the phase's goal forbids, and FEED-01's 'can reach the algorithmic feed' would not hold. Pressing t again pops it and fetches nothing, so the restore property is untouched."
  - "a shifted letter is two spellings in Bubble Tea v2: the decoder lower-cases the ASCII byte and marks it shifted, so Keystroke() reports `shift+g` while String() reports `G`. The shell looks the table up with Keystroke() and then String(), and a test pins the measurement (TestTheKeymapMatchesWhatTheDecoderReports)."
  - "the shell's gate (requestAllowed) mirrors screen.List's blocked/3 rules at the process level and is checked before page, refresh and the toggle's own fetch: the frame a toggle pushes has never seen the failure, and OPS-04 says no request is issued after a soft ban."
  - "WithFailure is the shell's one API the plan did not name: feed.go needs to open on a state (no session file) without issuing a request, and it applies that state to the live frames so their gates agree with the process's."
  - "keys.KeyJoin/Row.KeysJoined: the overlay's key column and the subcommand's first field write a multi-key row the same way, so the join is not hand-written on either surface."
  - "the A6 scan collects string-typed declarations (struct fields, vars, consts, function results — not parameters) and flags len() of them, plus unicode/utf8 imports and `%`-verbs carrying `*`; internal/text is exempt from the len() rule by the plan's own list and is separately asserted not to import fmt."
  - "Deps.Budget's third return (the limiter's own spacing window) is deliberately not printed: the state line's reset is the upstream's x-rate-limit-reset carried on the failure."

patterns-established:
  - "the anchor measures head rows: a block that follows another opens with the blank separator row, so the offset is 'every row before it plus its own separator' — a first version counted to the separator and put the selected head one row below the pane (found by the 80-column stack test)"
  - "both boundary scans carry real arms: the scan must have parsed >=3 files and >=8 imports and must find charm.land/bubbletea/v2 (the positive control), a planted net/http is reported, an empty package fails, _test.go files are skipped with a planted pair, and the A6 scanners are exercised on planted sources both ways (a violation and a non-violation)"

requirements-completed: [FEED-01, FEED-04, REND-03, REND-05, REND-06, OPS-01, OPS-02]

# Coverage metadata
coverage:
  - id: D1
    description: "The keymap: one 13-row table whose rows drive input handling, the `?` overlay and the `keys` subcommand; every keystroke round-trips through the matcher; the CLI's `--help` lists the verb"
    requirement: REND-05
    verification:
      - kind: unit
        ref: "internal/keys/keys_test.go"
        status: pass
      - kind: unit
        ref: "internal/cli/keys_test.go"
        status: pass
      - kind: unit
        ref: "internal/tui/tui_test.go#TestTheHelpOverlayIsTheOneTable"
        status: pass
      - kind: integration
        ref: "go run ./cmd/x-terminal keys | wc -l == 13"
        status: pass
    human_judgment: false
  - id: D2
    description: "The back stack: t pushes the other feed's frame and t pops it, with the frame returned to byte-identical, no request issued by the return, the same post id selected and its block back on the same pane row"
    requirement: REND-06
    verification:
      - kind: unit
        ref: "internal/tui/tui_test.go#TestTheBackStackRestoresTheFrame"
        status: pass
      - kind: integration
        ref: "internal/cli/feed_test.go#TestTheToggleSwitchesFeedsAndReturns"
        status: pass
      - kind: e2e
        ref: "tmux smoke run: t -> `for you  1/9+`, t -> the following pane back byte-identical"
        status: pass
    human_judgment: false
  - id: D3
    description: "Pagination is single-flight and never moves what the reader is reading; refresh is debounced and announced; a failed page keeps every post and states the failure in the status line"
    requirement: FEED-04
    verification:
      - kind: unit
        ref: "internal/tui/tui_test.go#TestThePaginationTriggerIsSingleFlight"
        status: pass
      - kind: unit
        ref: "internal/tui/tui_test.go#TestRefreshIsDebouncedAndAnnounced"
        status: pass
      - kind: unit
        ref: "internal/tui/tui_test.go#TestAFailedPageKeepsThePosts"
        status: pass
      - kind: integration
        ref: "internal/cli/feed_test.go#TestASoftBanStopsTheSessionAndKeepsThePosts"
        status: pass
    human_judgment: false
  - id: D4
    description: "The frame is a pure function of (state, W, H): a resized model renders the bytes of a model built at that size, and a resize issues no request"
    requirement: REND-03
    verification:
      - kind: unit
        ref: "internal/tui/tui_test.go#TestResizeRerendersWithoutFetching"
        status: pass
    human_judgment: false
  - id: D5
    description: "The shell cannot reach I/O: an AST import-boundary scan over internal/tui forbids net/http, os/exec, database/sql, internal/xsource, internal/store, internal/xweb, internal/session, internal/cli and internal/fake, with anti-vacuity arms; A6 keeps internal/text the only measure"
    requirement: FEED-01
    verification:
      - kind: unit
        ref: "internal/tui/boundary_test.go#TestTheShellCannotReachIO"
        status: pass
      - kind: unit
        ref: "internal/tui/boundary_test.go#TestNoSecondMeasurement"
        status: pass
      - kind: unit
        ref: "internal/tui/boundary_test.go#TestTestFilesAreNotScanned"
        status: pass
      - kind: automated
        ref: "grep -rn 'net/http|os/exec|database/sql|internal/xsource|internal/store' internal/tui/ returns only boundary_test.go"
        status: pass
    human_judgment: false
  - id: D6
    description: "Four named failure states render with their own state word, remedy and pane body; the budget line is in every frame including every failure frame"
    requirement: OPS-01
    verification:
      - kind: unit
        ref: "internal/tui/tui_test.go#TestTheFourNamedStatesRenderTheirOwnWords"
        status: pass
      - kind: unit
        ref: "internal/tui/tui_test.go#TestTheBudgetIsInEveryFrame"
        status: pass
      - kind: unit
        ref: "internal/cli/failure_test.go"
        status: pass
    human_judgment: false
  - id: D7
    description: "A soft ban stops the session: the state is on the line in the classifier's words, the posts stay, and no key issues a request again for the life of the process"
    requirement: OPS-02
    verification:
      - kind: unit
        ref: "internal/tui/tui_test.go#TestASoftBanStopsTheSession"
        status: pass
      - kind: integration
        ref: "internal/cli/feed_test.go#TestASoftBanStopsTheSessionAndKeepsThePosts"
        status: pass
    human_judgment: false

duration: 90min
completed: 2026-10-14
status: complete
---

# Phase 2, Plan 04 Summary

**The shell landed: one keymap table behind three surfaces, a frame stack that restores the frame you left byte for byte, an
anchor measured in head rows, a status line that reads the live limiter, and the composition root that puts the binary on
screen — with an AST boundary that keeps the shell unable to touch anything but the port it was handed.**

## Accomplishments

- `internal/keys` holds the phase's 13-row table (order, help text and groups exactly the UI-SPEC's), the matcher `ByKeystroke`,
  and `Row.KeysJoined` — the one join the overlay and the subcommand both write. Keys, overlay and `keys` subcommand read it.
- `internal/tui` is the shell: `Deps{Source, Budget, Failure, Now}`, `Model` (frame stack, help flag, size, the sticky failure,
  the refresh notice, the per-feed request count and read flags), `frame{List, Anchor}`, `New`/`WithFailure`/`Init`,
  `Update`/`update`/`View`/`Render`, the effect-to-command translation and the anchor geometry.
- `internal/cli/failure.go` maps every error class to a `domain.Failure`: the classifier's outcomes, `xweb.ErrRequestCap`,
  `session.ErrNoSession`, and the reachability check's own report; `Detail` is never empty.
- `internal/cli/feed.go` is the composition root: config warnings and registry warnings to stderr, the session read before
  anything is requested (a missing one is a state, not a crash), one transport, one limiter, one client, `xsource.New`,
  `tui.New`, and `tea.NewProgram`. `keys` and `feed` are rows of the CLI table.
- `internal/tui/boundary_test.go` carries both scans with their arms, and `internal/tui/tui_test.go` drives the shell through
  `internal/fake` with a frozen clock.

## Task Commits

No commits were made (the assignment forbade git commands); the work is in the working tree.

## The landed shapes

- `tui.Deps{Source domain.Source; Budget func() (used, limited int, reset time.Time); Failure func(error, []netcheck.Result) domain.Failure; Now func() time.Time}`
- `tui.Model{deps Deps; frames []frame; help bool; w, h int; failure domain.Failure; notice render.NoticeKind; noticeCount int; pending [2]int; read [2]bool}`
- `tui.frame{List screen.List; Anchor int}` (anchor = the selected block's head row's distance from the pane's first row)
- Messages: `pageMsg{Kind, Page, Cursor, Refresh}` and `failureMsg{Kind, Err}`
- `failureFor(err error, check []netcheck.Result) domain.Failure`

## The keymap rows (in table order)

| keys | action | help | group |
|------|--------|------|-------|
| `j` `down` | next | next post | navigate |
| `k` `up` | prev | previous post | navigate |
| `ctrl+d` | half-down | half a pane down | navigate |
| `ctrl+u` | half-up | half a pane up | navigate |
| `pgdown` | page-down | one pane down | navigate |
| `pgup` | page-up | one pane up | navigate |
| `g` `home` | first | first loaded post | navigate |
| `G` `end` | last | last loaded post | navigate |
| `r` | refresh | fetch newer posts, keep the place | feed |
| `t` | toggle-feed | switch between following and for you | feed |
| `?` | help | show or hide this list | view |
| `q` `esc` | back | back, or quit at the start | app |
| `ctrl+c` | quit | quit now | app |

## The routing rule for in-flight results

A result (answer or failure) is routed **by `domain.HomeKind`**: `Model.frameIndex(kind)` finds the frame that asked, and a
result whose kind has no frame anymore (the user popped it while a page was out) is **dropped** — it cannot corrupt a list that
is not on the stack. The answer also decides its own event: `Refresh` on the message picks `screen.Refreshed` (prepend) over
`screen.PageFetched` (append), because `screen.List` keeps its refresh latch to itself. The process's bookkeeping
(`pending[kind]`, `read[kind]`) is released/updated for every answer, frame or not.

## The anchor's clamp rule

`scroll = max(0, rowsAbove(selected head) - Anchor)`; a selection move keeps the old `scroll` and stores
`Anchor = clamp(rowsAbove(new head) - scroll, 0, paneRows-1)`. `rowsAbove` is the head row's offset, which is every row of every
block before the selected one **plus the selected block's own separator row** (the first version counted to the separator and put
the selected head one row below the pane — the 80-column stack test caught it). A resize changes nothing but
`Anchor = clamp(Anchor, 0, paneRows-1)`, and the pane is rebuilt from the same state at the new width. A prepended refresh
leaves the anchor alone, so the scroll absorbs the new rows and the pane's bytes do not move.

## The mapper's kind table

`*xweb.APIError` by outcome: auth rejected → session expired, signer or auth → signer/auth, unknown operation → operation id
stale, endpoint retired → endpoint retired, rate limited → rate limited (+ `Reset` from the header), soft ban → soft ban
(`Stopped`, not retryable), load shed, bad request, empty body, edge challenge, unauthorized; any other outcome (including
`ok`/unclassified) → unclassified. `xweb.ErrRequestCap` → budget spent; `session.ErrNoSession` → no session file;
anything else → the first failed `netcheck.Result` (config failed → transport unusable, no route, proxy refused, upstream
blocked, challenge → TLS challenge, session rejected → session expired, API rejected → unclassified) with its own `Detail`;
nothing better → unclassified with the error's own sentence. `Retry` is false for rate limited, soft ban, budget spent, bad
request and endpoint retired; `Stopped` only for the soft ban.

## Files Created/Modified

- `internal/keys/keys.go`, `internal/keys/keys_test.go` — the table, the matcher, the join; 7 tests
- `internal/tui/model.go`, `update.go`, `view.go` — the shell (pure `update`/`Render`), the geometry, the keymap dispatch
- `internal/tui/tui_test.go` — 16 tests: FEED-01/02/04/05 behaviours, REND-03/05/06, OPS-01/02/04, the decoder seam
- `internal/tui/boundary_test.go` — the import boundary, the netcheck-as-a-type rule, A6, the skip rule, with all arms
- `internal/cli/keys.go`, `keys_test.go` — the subcommand and its table-order/tab-separated test
- `internal/cli/failure.go`, `failure_test.go` — the mapper and 7 tests over every outcome, sentinel, path row and the words arm
- `internal/cli/feed.go`, `feed_test.go` — the composition root, the two test seams, and the verb-level assertions
- `internal/cli/cli.go` — the `feed` and `keys` rows (usage shows `feed     read the Following timeline in the pane`)
- `go.mod`, `go.sum` — `charm.land/bubbletea/v2 v2.1.0` (proxy-prefixed `go get` then `go mod tidy`)

## Decisions & Deviations

1. **`internal/netcheck` is not in the forbidden-import list.** The plan forbids it *and* requires `Deps.Failure` to name
   `[]netcheck.Result`. The scan keeps the intent: only `model.go` may import it, and no call into it is allowed anywhere
   (with a planted-call arm). The five phase-requirement paths plus xweb/session/cli/fake stay forbidden outright.
2. **`t` fetches its frame's page one** (the plan says "fetching nothing"): a never-read frame would render the empty-state copy
   about a read that never happened, and FEED-01's "reach the algorithmic feed" would not hold. `t` again pops and fetches
   nothing, so the restore property is exact.
3. **`pageMsg` carries `Refresh`, `failureMsg` carries `Kind`** — needed for the routing rule and for choosing between
   `Refreshed` and `PageFetched`.
4. **`WithFailure`** is the one shell API the plan did not name; `feed.go` needs it for the no-session state.
5. **A shifted letter is matched twice** (`Keystroke()` then `String()`): v2's decoder reports `shift+g` for `G`, a measurement
   pinned by a test rather than assumed.
6. **Keys and the shell's gate**: `keys.KeyJoin`/`Row.KeysJoined` were added so no surface hand-writes the join; the shell's
   `requestAllowed` mirrors `screen.List`'s three blocking rules at the process level (the plan's "in-flight latch" plus
   OPS-04), and is checked before page, refresh and the toggle's push.

## Verification (all run, all green)

```
go build ./...                                              # 0
go vet ./...                                                # 0
go test ./... -count=1                                       # 459 tests, 18 packages, ok
gofmt -l internal cmd                                        # empty
go run ./cmd/x-terminal keys | wc -l                         # 13
go run ./cmd/x-terminal --help | grep -qx '  feed.*timeline' # matches
TERM=dumb NO_COLOR=1 go test ./internal/tui/... -count=1     # ok
TMUX=… inside a tmux pane: go test ./internal/tui/... -count=1  # ok (the marked-row helper strips SGR)
TERM=screen-256color / COLORTERM=truecolor: go test ./internal/tui/... # ok
grep -rn 'net/http|os/exec|database/sql|internal/xsource|internal/store' internal/tui/  # only boundary_test.go
```

### Smoke run (tmux, 80×24, fake source)

Driven through a **temporary harness** (an untracked test file that swapped the verb's two test seams and was deleted after the
run; the shipped binary has no source-swapping flag, per the plan). What was captured, in one session:

```
frame 1  (opening)      following  1/9+  0 requests, 0 rate-limited      # the Following list, page one
frame 2  (j j j)        the marks moved down the pane                     # scrolling
frame 3  (G)            following  9/14+  0 requests, 0 rate-limited      # page two arrived, position moved
frame 4  (r)            no new posts  following  9/14+  0 requests...     # the refresh announced
frame 5  (t)            no new posts  for you  1/9+  0 requests...        # the other feed, its own page one
frame 6  (t)            no new posts  following  9/14+  ...               # back, pane byte-identical
frame 7  (q)            harness log: "feed: exit 0, stderr \"\""          # clean exit
```

The shipped binary's own smoke run (`/tmp/x-terminal feed` in a tmux pane with no session file) renders
`no session file  write the session file  following  0 requests, 0 rate-limited`, answers `?` with the 13-row table, and closes
the overlay on `q` rather than quitting (a second `q` then exits 0).

## Next Phase Readiness

Phase 4 can push real frames through `frames []frame` and replace the kind-keyed lookup with opaque frame ids in one place
(`frameIndex`/`answered`/`failed`); the golden-frame plan (02-05) can call `tui.Model.Render` — the frame is exactly
`render.Frame`'s bytes, which is what makes a stored frame the specification of the shipped screen.
