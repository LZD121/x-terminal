---
phase: 03-three-state-disguise-on-the-working-renderer
plan: 03
type: summary
status: complete
requirements: [DSGT-08]
verified: |
  go build ./... && go vet ./...                 clean
  gofmt -l internal cmd                          empty
  go test ./... -count=1                         671 passed in 19 packages
  go test ./internal/render -run 'TestGoldenFrames' -update && sha256sum ... && go test ... -run 'Golden|Corpus'   ok, 31 Phase-2 frames byte-identical
  go test ./internal/render/... -run TestCapturePanic -v      both legs ran, the control leaked
  tmux show -g focus-events (throwaway server)   off
---

# 03-03 SUMMARY — the corpus, the round trip, and the environment

## 1. The stored frames (Task 1)

`internal/render/testdata/TestGoldenFrames/`: **18 new files** — the ten the plan names, plus the eight
`_tmux` twins the corpus's own existing twin rule decides (every covered frame's pane carries the ZWJ
family, whose width the two measures disagree on, so both byte streams are stored and `corpus_test.go`
asserts the twin decision from the bytes). The plan's "ten new files" is the frame list; the twins are
the rule the harness already had, and suppressing them would have broken `storedFrames`' own
equivalence.

| frame | state it encodes |
|---|---|
| `cover_{servicelog,gitlog,ci,internal}_{80,120}` (+ `_tmux`) | the loaded adversarial list (three pages absorbed, selection = the CJK post — the same model `timeline_*` uses), the session-expired failure, budget 200/12, rendered through the style's grammar |
| `panic_{80,120}` | the panic frame at each width: `PanicFrame(W, 24)`, composed from the pane's size alone |

Two readings of the plan were reconciled and are worth naming:

- **The covered frames carry the failure the drop rule needs.** The plan asks the covered frames to
  show "the drop rule doing its declared work: at 80 columns the `list` field is absent from the
  covered forms of exactly the states the UI-SPEC names". Of the three states the UI-SPEC names, only
  a loaded timeline with `session expired` reaches that at 80 columns: the covered line is 87 cells
  (state + remedy + `branch=main` + position + budget), so the position drops and then the list
  (68 cells). With the rate-limited failure the line is 83, the position drops and the list *stays* at
  77 — the declared consequence would not be in the corpus. So the covered frames carry the loaded
  list *and* the session-expired failure: the model is the timeline's, the failure is a state field.
- **`each covered frame carries its style's own list name`** therefore holds at 120 in all four styles
  and at 80 in the three whose names fit (`stream=in` 79, `queue=ci` 78, `view=mine` 79); the gitlog
  80 frame drops it. The audit asserts exactly that, byte for byte, and the arm is the plain
  `expired_80` frame: the same failure and budget with `following` in the same 79 cells, and one cell
  over the pane the moment `branch=main` takes its place.

## 2. The audit (`corpus_test.go`, all over the stored bytes, never a re-render)

| case | what it asserts |
|---|---|
| V2 identity | for all 16 covered + 2 panic frames: none of the fixture's handles, display names, the upstream's host, the product's name or its nouns (`following`, `for you`, `post(s)`, `timeline(s)`, `feed(s)`, word-bounded and case-insensitive after `ansi.Strip`); **and** every covered frame carries the identity its own style prints, for every account on its pane |
| V5 panic | each panic frame carries zero `ESC` bytes and no word (or alias projection) of the feed's own vocabulary |
| the copies | each covered frame carries its style's own head literals and pane header, carries no plain list name, and carries its first post's words verbatim; the panic frames carry no status line (last row blank, none of the status vocabulary); the 120 frame is its own five fields and the 80 frame is exactly the fields the declared drop order leaves |
| the widths | every row of every covered and panic frame is exactly `W` cells through `internal/text`; A7 now measures each row against its own grammar's budget at its own quote depth (identical to Phase 2's `1 + 2 + min(W-3, 96)` for the plain grammar); the panic frame's lines are the same bytes at 80 and 120 |
| the arms | ≥10 frames scanned, the plain frame prints the identities the scan looks for, ≥8 covered frames carry a selected block and ≥8 aliases, the panic frame draws ≥15 lines, the corpus holds at least a frame per table entry |

Three refinements the work needed, each reported rather than hidden:

1. **`assertNoCounts` is now word-bounded.** A cover's own reference tokens are digit runs
   (`seq=4614498`), so the Phase 2 substring scan reported `449`. The assertion is "a count stands on
   its own", which is what a renderer that printed counts would print.
2. **A7 is grammar-aware.** The plain frame's `3 + measure` is not the budget of a git log's 4-cell
   message indent (101 cells at 120 columns), so the rule is now the composer's own arithmetic.
3. **The pin table** `internal/render/testdata/cover-alias-pins.txt` is hand-written (the program never
   writes it): the UI-SPEC's two worked examples, four projections each; `TestCoverAliasPinTable`
   reproduces them, and the ids and handles are asserted to be the ones
   `TestCoverAliasWorkedExamples` names, so the two artifacts cannot drift.

**The identity scan was broken once and reverted**: `Author: Linus Zhang` in `cover_gitlog_80.golden`
was replaced by the fixture's own display name `name of lzd` (the same 11 cells, so the width check
stays quiet). `TestCorpusAudit/V2` reported `the stored frame cover_gitlog_80 prints "name of lzd", an
identity of the content it was built from`, Phase 2's `assertNoDisplayNames` reported it too, and
`TestGoldenFrames` reported the byte difference. Reverted; all three pass.

## 3. The round trip, the write order, the surface, the titles, the disk (`internal/tui/cover_e2e_test.go`)

- **`TestPanicRoundTripIsExact`** — from each of the five non-panic start states, the shell is walked to
  a non-trivial place (selection on the 4th post, a half page, a second frame from the feed toggle, the
  session-expired failure on the line, a page left in flight) and the *seven* captured facts are
  compared before and after `ctrl+p`, `ctrl+p`: the frame byte for byte (1 738-byte frames at 100×30),
  the fake's call count, the stack depth (2), the selected post id, the anchor, the in-flight latch
  (`outstanding()` = 1) and the failure state. The arms require every fact to be non-trivial first
  (depth ≥ 2, selection ≠ the first post, latch > 0, failure ≠ none). A second leg delivers a page
  answer while panicked: the frame does not change, the list behind the overlay gains the two posts,
  no request is spent, and leaving the overlay shows the absorbed answer.
- **`TestPanicEntryOrderAndBranches`** — a recording writer is the sink's output and a recording runner
  is its process runner: exactly one write, carrying every frame row, then `ESC[3J`, then
  `ESC]2;make`; the runner reads the writer's own state when it is called (so the subprocess is
  *after* the write), and the three pane branches are asserted on the argument vectors —
  `clear-history -t %7` and `rename-window -t %7 make` for a pane, nothing at all for unset or empty.
  A start-up panic writes nothing before the first size, then the same order, then the same two
  commands, and a later resize re-runs nothing.
- **`TestTheWholeSurfaceUnderACover`** — nine screens (the timeline, the empty read, the loading frame,
  the help overlay, the four failure states, the stalled cursor), each reached in a plain run and a
  covered run built from the same wiring: the two models are asserted identical (feed, selection,
  loaded count, walk end, stalled flag, post count, every post's text/author/instant, anchor, failure,
  notice, stack depth, overlay), the frames differ, and the same key sequence
  (`j ctrl+d r t ? ? esc`) costs the same number of requests. Taking the cover off leaves a frame
  byte-identical to the plain run's, `t`,`t` and `ctrl+p`,`ctrl+p` each return the frame they left, and
  (where the state is not modal) the same model's own covered/plain pair returns byte for byte.
- **`TestTheTitleIsAFunctionOfTheState`** — the UI-SPEC's title column: the view's `WindowTitle` and the
  transition's `SetWindow` call for the plain reader and the four styles, the entry's title for panic,
  the plain title restored on leaving, and the arm that no covered title carries a product name or noun.
- **`TestNoAliasTableOnDisk`** — a temporary home through `XDG_CONFIG_HOME`/`XDG_DATA_HOME`, the run
  driven through the closure `internal/cli/feed.go` passes: `c`, `3`, `4`, `m`, `1`, `c`, a panic round
  trip, and a restart from the same config. The tree diff (names, sizes, bytes) is **the config file
  alone**, and inside it **exactly two lines**, both `display.*`. Every file in the tree is then
  searched for the fixture's handles, display names and *all four* alias projections in all four
  styles; a planted file proves the search can see what it is looking for.
- **`state_test.go`** — `TestThePersistedStateIsTheDeliberateOne`: `m` touches `display.state` alone,
  `c` and the digits write both keys (the file's own lines are the evidence; digit 1 numbers the style
  the file already holds, so its diff shows one key), and a **blur → panic → focus cycle leaves the
  config file byte-identical**.
- **`alias_test.go`** — `TestCoverAliasesAcrossProcesses` re-executes the test binary with a marker in
  the environment; the child (no shared memory, no map order) prints each account's alias in each
  style, and the parent compares it with its own call *and* with the pin table.

## 4. The environment harness (`internal/render/capture_panic_test.go`) — both legs ran here

```
go test ./internal/render/... -count=1 -run TestCapturePanic -v     exit 0
  the_multiplexer's_history/the_shipped_program        PASS   the pane %0 held 62 labelled lines before
                                                             and none in its screen or history after
                                                             the entry; window name `make`
  the_multiplexer's_history/the_control_without_the_clear PASS   the control leaked: 41 labelled lines
  the_window (bare Ghostty, 126x58)                    PASS   728011 bytes either way; the entry changed
                                                             the picture, the scroll-up after it did not,
                                                             and without the clear the same scroll-up
                                                             changed it (729221 vs 728726)
! tmux ls 2>/dev/null | grep -q 'xterm-panic'          → no session on the default socket
tmux show -g focus-events (throwaway server)           → off
```

The measured run is the **shipped binary**: built by the test from the module root, run in a real tmux
pane at 80×24 with a temporary `XDG_CONFIG_HOME` (state `immersive`, cover `gitlog`) and a temporary
`HOME`, no session file — so the frame is a covered failure state and no request leaves the machine. The
flood (`PANIC-LABEL-<n>`, 60 lines) is signalled back with `tmux wait-for`, the panic key is delivered by
`tmux send-keys C-p`, and the pane's own `tmux` is a shim that logs its argv: the clear is asserted to be
`clear-history -t %0` for **the pane the program was actually started in** (`#{pane_id}` read from the
pane), not inferred from the history being empty. The control is the same flood with the entry written
without `ansi.EraseEntireDisplay` and the clear skipped — a switch the test owns.

**What the clear does not reach, recorded not claimed:** the program runs on the alternate screen, so
tmux keeps the screen it replaced, and neither ED 3 nor `clear-history` touches it (`capture-pane -a`
still shows 13 of the flood's lines). That saved screen is restored when the program exits; it is not
part of the pane's history and not what the user scrolls through while the program runs. Also recorded in
the harness's own comment: a terminal that ignores ED 3 cannot be detected (no portable query, and a
query would add a round trip to the frame that matters most), and under tmux focus events reach a pane
only when the server's `focus-events` is on — measured `off` on this host, and the app does not change a
server-global setting.

**What the human saw** (I ran it, unattended, and watched the log; the plan's Task 3 is a human-verify
checkpoint): the bare leg found *its own* window by process ancestry (`com.mitchellh.ghostty`, the only
toplevel whose client process is an ancestor of the test), required the compositor's focused window to be
that same window (wtype delivers keys only to the focused surface, and this Hyprland's `dispatch` is a
Lua API a harness must not guess at — so an unfocused window skips with that reason instead of typing
into somebody else's terminal), flooded its own terminal through `/dev/tty`, painted the entry, injected
six `Shift+Page_Up` with wtype, captured the window region with `grim`, and compared the pictures: after
the clear the scroll-up revealed nothing, without it the scroll-up revealed the flood.

**A window capture is pixels and this host has no OCR**, so the bare leg's assertion is the probe's own
comparison (with its control), not a text scan of the picture; the harness says so where it measures it.
The label text itself is asserted in the multiplexer leg, where the pane is text.

## 5. Commands the work was accepted on

```
go build ./... && go vet ./...                       clean
gofmt -l internal cmd                                empty
go test ./... -count=1                               671 passed in 19 packages
go test ./internal/render -run TestGoldenFrames -count=1 -update   then without -update: byte-equal
sha256sum -c <the 31 Phase-2 frames, hashed before this plan>      31 OK
go test ./internal/render/... -count=1 -run TestCapturePanic -v    exit 0, both legs, the control leaked
go test ./internal/tui/... ./internal/render/... -count=1 -run 'TestPanicRoundTrip|TestPanicEntry|TestTheWholeSurface|TestTheTitle|TestNoAliasTable|TestState|TestCoverAlias'   ok
```

Break-once checks, each run and reverted (this plan's own; 03-01/03-02 recorded theirs):

| Planted | Test | Result |
|---|---|---|
| the fixture's display name in a stored covered frame | `TestCorpusAudit/V2`, `assertNoDisplayNames`, `TestGoldenFrames` | each reported it; reverted, green |
| the entry written without its clear (the harness's control) | `TestCapturePanic` control | leaks 41 labelled lines; the measured run leaks none |

## 6. Reported, not implemented

- **18 stored files, not ten** (see §1): the corpus's own twin rule decided the eight `_tmux` twins.
- **The covered frames' failure is `session expired`, not the timeline's rate-limited state** (see §1);
  the corpus comment and the audit say why, and the drop rule's declared consequence is only reachable
  with one of the three states the UI-SPEC names.
- **`assertNoCounts` and A7 were widened** (§2) — both changes are needed for the covered frames to be
  auditable at all, and both are documented at the assertion.
- **The alternate screen's saved copy is not cleared** (§4) — a fact about the multiplexer's alternate
  screen and a candidate for a later decision, not something this plan's files can change.
- **The bare leg injects only into a window it has proved is its own and focused**; it skips (naming the
  focused window) rather than using this compositor's Lua dispatch to steal focus.
