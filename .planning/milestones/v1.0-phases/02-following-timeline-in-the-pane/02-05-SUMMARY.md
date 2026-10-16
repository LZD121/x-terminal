---
phase: 02-following-timeline-in-the-pane
plan: 05
subsystem: ui
tags: [golden-frames, view-specification, corpus-audit, tmux-capture, pty-harness, dsr, width-model, REND-08, A1, A2, A3, A7]

# Dependency graph
requires:
  - phase: 02-01
    provides: internal/domain (Post/User/MediaRef/Failure/FailureKind), internal/screen (List + its reducers) and internal/fake (the recorded fixture and the hand-authored adversarial page set) the corpus's states are made of
  - phase: 02-02
    provides: "internal/text — Width/Wrap/Pad/Truncate, the only measurement the frames, the audit and the captures are made with"
  - phase: 02-03
    provides: internal/render (Blocks/StateBlock, Frame, StatusFields, Help, the palette, Plain and its measure) which composes every stored byte
  - phase: 02-04
    provides: internal/tui (Model.Render) and internal/keys.Rows(), the two things the corpus's states were verified against by hand
provides:
  - "internal/render/testdata/TestGoldenFrames/*.golden — 22 model frames at 60/80/120/200 columns and H=24 (plus 8 `_tmux` twins): the view specification as stored bytes"
  - "internal/render/golden_test.go — TestGoldenFrames: the corpus, its pinned environments, the injected clock, the twin rule and the -update path (charmbracelet/x/exp/golden)"
  - "internal/render/corpus_test.go — TestCorpusAudit: the adversarial-content audit, the absences, A1/A2/A3/A7 over the stored bytes, and the anti-vacuity arms"
  - "internal/render/capture_tmux_test.go — TestCaptureTmux: the stored multiplexer frames captured through real tmux panes at 80×24 and 120×24, with a comparator self-test"
  - "internal/render/capture_pty_test.go — TestCaptureInGhostty (the bare window leg, a run-it-in-a-window check) + TestPtyHarnessPlumbing (the leg's own machinery on a PTY pair)"
affects: [phase-03-grammars, phase-04-detail, phase-05-search, phase-06-saved, phase-07-media]

# Actuals — chars/4 over the files actually changed
actuals:
  tokens: 38000
  tasks: 3
  commits: 0

# Tech tracking
tech-stack:
  added:
    - "github.com/charmbracelet/x/exp/golden v0.0.0-20250806222409-83e3a29d542f (the version the plan pinned; it brings github.com/aymanbagabas/go-udiff 0.2.0 -> 0.3.1)"
    - "github.com/charmbracelet/x/term v0.2.2 and golang.org/x/sys v0.47.0 are now direct requires: both were already in the module graph as bubbletea's indirects, and the PTY harness imports them"
  patterns:
    - "a stored frame is a fact: every audit assertion reads the bytes on disk and none of them re-renders the state it checks"
    - "the environment is data (a small `environment` value per frame: TMUX/TERM/COLORTERM/NO_COLOR) and is pinned for every render, because this host's shells export TERM=dumb and NO_COLOR=1 and $TMUX switches the measure"
    - "the clock is injected (2026-10-14T09:12:00Z with TZ=UTC), so a stored stamp is a property of the state and not of the day"
    - "the corpus's states are the lower layers' own values — a screen.List absorbed through the reducers, domain.Failure values, keys.Rows() mapped into render.HelpRow — composed by the three functions the shell's Render composes"
    - "a frame gets a `_tmux` twin exactly when its bytes depend on $TMUX: the frame whose text holds a cluster the two measures disagree on (the ZWJ family)"
    - "the capture legs compare text, row count and padding, never escape bytes: tmux re-synthesises SGR runs when it captures a pane, and the state frames are the plain rung's anyway"

key-files:
  created:
    - internal/render/golden_test.go
    - internal/render/corpus_test.go
    - internal/render/capture_tmux_test.go
    - internal/render/capture_pty_test.go
    - internal/render/testdata/TestGoldenFrames/ (30 files: 22 frames + 8 twins)
  modified:
    - go.mod
    - go.sum
    - internal/fake/fake.go (the ZWJ family's joiners: reported below)

# Coverage metadata
coverage:
  - requirement: REND-08
    status: complete-with-one-deferred-human-step
    evidence: "the stored corpus, the audit, the tmux capture (run: 30 frames, panes agree) and the bare-terminal harness; the bare window leg itself could not be run from this host's sandbox and is recorded as a human check"
---

# Phase 2, Plan 05 Summary

## Accomplishments

- The view specification exists as bytes: **22 model frames** at `H = 24` under
  `internal/render/testdata/TestGoldenFrames/`, every one of them a state the shipped code composes,
  pinned to an injected clock and a pinned environment, byte-compared by `TestGoldenFrames` and read
  back — never recomputed — by `TestCorpusAudit`.
- **8 `_tmux` twins** are stored beside the frames whose bytes depend on `$TMUX`.
- The audit asserts each of the checklist's 20 required cases in a *named* frame, the three required
  absences, the four width properties (A1, A2, A3, A7) over the stored bytes, and four arms that stop
  the corpus shrinking into vacuity.
- The same frames were **captured through real tmux panes** at 80×24 and 120×24: the text, the row
  count and every row's padding agree with the stored frame, except on the one row holding the ZWJ
  family, where the pane's own width table and the model's multiplexer measure part company (measured
  and reported below, not smoothed away).
- The bare-terminal harness is written and its machinery is tested on a real PTY pair; the window
  leg itself is a **deferred human check** (see "What the human saw" below).

## The stored corpus (22 frames + 8 twins)

Every frame is 24 rows of its own width, the last row the status line. The frames that show posts
share one list: `fake.Adversarial()`'s whole three-page set absorbed through `screen.List`'s own
reducers, with the recorded fixture's engagement numbers on the posts (so "no count is printed" is a
check and not a tautology) and the rate-limited state on the status line — the failure the UI-SPEC's
own 80-column frame carries, which is the one FEED-05 keeps the posts visible under.

| Frame | W | State it encodes |
|-------|---|------------------|
| `timeline_60` | 60 | the list from its top, CJK post selected, at the pane's floor: the measure is 57, so the CJK body and the two exact-fill lines wrap, and the status line is cut with `~` |
| `timeline_80` | 80 | the same state at the primary width: dirty handles, CJK, emoji, family, the over-measure URL broken at 77 cells, the 96/95/77/76-cell lines, both quote depths below the fold, the selected block at rows 4–5 |
| `timeline_120` | 120 | the same state at 120: the body wraps at the 96-cell cap and the surplus columns stay blank — the frame that shows the cap is the cap |
| `timeline_200` | 200 | the same state at 200: identical content to 120, 100-odd blank columns behind it |
| `empty_80`, `empty_120` | 80, 120 | the empty read that succeeded: the heading and `CopyEmptyBody`, the position omitted |
| `loading_80` | 80 | the first page is out and nothing has answered: `loading following...` and nothing else |
| `help_80`, `help_120` | 80, 120 | the whole 13-row keymap as `render.Help` (keys.Rows() mapped into render.HelpRow) over the timeline's own status line |
| `expired_80`, `expired_120` | 80, 120 | cold failure, session expired: the state word as the head line and the classifier's sentence as the body; status = the UI-SPEC's worked line 2, byte for byte |
| `offline_80`, `offline_120` | 80, 120 | cold failure, no route: netcheck's own sentence (`GET <base>/ failed: …`); worked line 4 |
| `schemachange_80`, `schemachange_120` | 80, 120 | cold failure, operation id stale; worked line 5 (and the one the UI-SPEC corrected) |
| `stalled_80` | 80 | the walk that ended because the cursor came back: pages 1–2 loaded, the pane at the reader's place — the post carrying the depth-1 quote, with the depth-2 quote below it — and `no older posts (cursor stalled)` in the notice |
| `narrow_79` | 59 | the pane below its floor: the pane's own line, and the status line still rendering, cut with `~` (see the naming note below) |
| `color_truecolor`, `color_256`, `color_16`, `color_mono` | 80 | the timeline state at each rung of the ladder: the rung changes the escapes and nothing else (A1, and A3 over the mono rung) |
| `timeline_last_80` | 80 | the same list with `G` applied: the last post selected, the anchor at its ceiling, the selected head on the pane's last visible row — where the continuation pair, the repost, the limited post, the tombstone and both attachment lines are |

**The stored `_tmux` twins** (TMUX set, same rung, same clock): `timeline_60_tmux`,
`timeline_80_tmux`, `timeline_120_tmux`, `timeline_200_tmux`, `color_truecolor_tmux`,
`color_256_tmux`, `color_16_tmux`, `color_mono_tmux`. The rule is the bytes: a frame gets a twin
exactly when its text holds a cluster the two measures disagree on (the ZWJ family), and the audit
asserts the same equivalence independently from the stored files. Frames whose only wide clusters
measure the same in both environments (CJK, the single emoji) render identically either way and have
no twin — `timeline_last_80`'s window ends past the family, so it has none.

## The audit (`TestCorpusAudit`)

**The required cases**, each asserted in the frame named here (a failure names the frame to look at):
15-cell handle, 3-cell handle, a CJK run, a single emoji, a ZWJ family (asserted as the disagreement
itself: one grapheme cluster of 2 cells, 6 by `ansi.WcWidth`), a URL longer than the measure (both
ends of it on screen in different rows), a line that exactly fills the measure, a line one cell
shorter, a depth-1 quote, a depth-2 quote, a quote below a longer body (all four quote cases in
`stalled_80`), a continuation pair with no blank row between them, a repost marker, a limited
marker, a tombstone block (marker with nothing printed above it), an attachment line with a singular
noun, one with a plural (and the image/video/gif order), a selected block whose first row is not the
pane's first row (`timeline_80`), a selection whose block is the pane's last visible row
(`timeline_last_80`), and a status line with a failure state, a reset time and a three-digit budget
(`timeline_80`).

**The absences**: no stored frame prints an engagement count the corpus's posts carry (the recorded
fixture's own numbers), none prints a media URL or its basename, none prints an account's display
name, and the empty-state copy appears exactly in the frames whose state is the successful zero-post
read — never in a frame carrying a failure.

**The width properties**: A1 — `ansi.Strip` of `color_truecolor`, of `color_mono` and of the plain
`timeline_80` are byte-identical, with the arm that the truecolor frame actually carries SGR and a
selected block; A2 — every frame carrying a failure prints that state's literal word (and, for the
rate-limited state, the reset `resetSuffix` renders), and the state word is preceded by the state
role's own escape; A3 — `expired_80`'s status line is exactly its fields joined by two cells (composed
from the package's own `stateField`/`remedyField`/`listField`/`budgetField`), and in `color_mono`
every top-level head row's stamp starts at cell `1 + HandleCols + 1` with its markers exactly two
cells behind it; A7 — no pane row of any frame exceeds `3 + min(W-3, 96)` cells of text (the status
line is the one row outside that rule: it is cut at `W` and never wraps), with the arm that
`timeline_200` really does fill the 96-cell cap.

**The arms**: every frame the table names is stored and every stored file is named (a deleted frame
fails, and so does a stale one — the arm caught a leftover `stalled_80_tmux` while this plan was
being built); the corpus holds at least as many frames as the table generates; each case is asserted
in a named frame; the strip comparison runs over a frame with SGR and a selected block; and the
`narrow_79` frame's pane holds the pane's own line while its status line still renders.

## The tmux capture (`TestCaptureTmux`)

For each of `timeline_80_tmux` and `timeline_120_tmux`, on a private socket so the user's tmux
server, sessions and configuration cannot reach in (and so a failing assertion leaves nothing):

```
tmux -L xterm-golden -f <tmp>/tmux.conf new-session -d -s xterm-golden-80 -x 80 -y 24 \
     "printf '\033[2J\033[H'; cat <tmp>/timeline_80.frame; tmux wait-for -S xterm-golden-80-painted; exec sleep 30"
tmux -L xterm-golden wait-for xterm-golden-80-painted
tmux -L xterm-golden capture-pane -p -N -t xterm-golden-80
tmux -L xterm-golden kill-server            # in a t.Cleanup, on every path
```

The configuration (`set-option -g status off`) is the harness's own, so the pane is the whole window;
the frame is written as a file `cat` reads, with no trailing newline, so nothing scrolls; the
clear-screen is there because the pane's shell prints a line of its own before the command; and the
`wait-for` is the pane's own word that the frame is fully painted — no sleeps and no half-written
captures. What is asserted: the capture has exactly 24 rows; the pane's text equals the stored
frame's text row for row; every other row's padding agrees cell for cell; the selected block's rows
carry the `>` mark in the captured text; the status line is the capture's last row; and — the
comparator's own arm — a copy of the frame with one cell changed and a copy with one row removed must
each be reported as a disagreement.

**The one measured disagreement** (recorded by the harness, reported here): tmux 3.7 counts a ZWJ
family as *one two-cell cluster* — the grapheme's width — where the model's multiplexer arm counts
the six cells of its code points (`ansi.WcWidth`). The pane therefore pads the family row four cells
more than the frame does (75 vs 71 spaces at 80 columns, 115 vs 111 at 120), i.e. the frame stops
four cells short of the pane's right edge instead of wrapping. The error is in the direction that
cannot wrap a row, and the text, the row count and the wrap points agree exactly; but it means the
UI-SPEC's premise that `$TMUX` set means "what a multiplexer's grid models" is **not true of tmux
3.7 for a ZWJ cluster** — see Decisions & Deviations.

## The bare-terminal harness (`TestCaptureInGhostty`, deferred window leg)

`TestCaptureInGhostty` runs only when `$TMUX` is unset and `$TERM_PROGRAM` is `ghostty`; everywhere
else it skips with a printed reason naming both variables (so a run on another machine cannot pass by
doing nothing). Its first subtest — the frame check — runs everywhere: the stored `timeline_80` frame
must hold exactly 24 rows and no row wider than its pane, and a copy with one cell added to a row
must be reported rather than written.

The terminal leg: the harness writes `clear-screen + cursor-home`, then the stored bare
`timeline_80.golden` frame *verbatim* (no trailing newline), then the device-status request
(`ESC [ 6n`), and reads the cursor report back. It needs the terminal to be **exactly 80 columns**
(the frame's width: a wider window would let a wrapped row hide inside the row it wrapped into) and
**at least 25 rows** (one more than the frame: a shorter window would scroll instead of moving the
cursor down) — otherwise it skips with the numbers. Its assertions: the reported row is exactly
`H = 24`, and the reported column is `W + 1 = 81` (one column past the frame's last cell), which is
the terminal's own word that it consumed the frame's rows at the frame's width with no extra wrap.
The harness uses the terminal it is running on rather than opening a fresh `/dev/ptmx` pair, and says
why in the file: a freshly opened pair has no emulator at its far end, so nothing would answer the
query. It clears only the terminal's *input* flags (not `x/term`'s `MakeRaw`, which clears the output
flags too and would draw the frame as a staircase — the terminal's own `ONLCR` is what returns the
cursor to column 1 after each of the frame's newlines).

`TestPtyHarnessPlumbing` exercises that machinery on a PTY pair the test opens itself
(`/dev/ptmx` → `/dev/pts/N`): the window size it asks for (80×25), the frame written verbatim
through it (each newline turned into CRLF by the line discipline the harness leaves alone), and the
read and parse of a cursor report. Its answer is canned — the peer at the far end is the test, not a
terminal — so it is evidence about the harness, not about any terminal, and it is named for what it
is.

## Files Created/Modified

Created: `internal/render/golden_test.go`, `internal/render/corpus_test.go`,
`internal/render/capture_tmux_test.go`, `internal/render/capture_pty_test.go`, and
`internal/render/testdata/TestGoldenFrames/` (30 files: 22 frames + 8 twins).
Modified: `go.mod`, `go.sum`, and `internal/fake/fake.go` (see below).

## Decisions & Deviations

1. **`internal/fake/fake.go` was modified** (the only file outside the plan's list). The fixture's
   "ZWJ family" post had lost its zero-width joiners: it held four separate emoji, which measure the
   same in both environments — so no stored frame would have contained a ZWJ family, no `_tmux` twin
   would have differed from its frame, and task 2's capture would have had no twin to capture. The
   file's own comment claims "a CJK run, a single emoji and a ZWJ family, none of which is one cell
   per rune", and `internal/text`'s and `internal/render`'s own fixtures use
   `"\U0001F468\u200D\U0001F469\u200D\U0001F467"`, so the literal was repaired to exactly that
   (written as escapes, so no editor can strip it again). The audit's ZWJ case — one grapheme cluster
   of 2 cells and 6 by wcwidth — is the check that would catch it recurring.
2. **`narrow_79` is rendered at 59 columns** (`render.MinPaneCols - 1`), not at 79. The plan's name
   list says "the width the name carries" but the plan's own audit arm requires that frame's pane to
   hold the narrow-pane line, and the UI-SPEC's rule puts that line below 60 columns; 02-03-SUMMARY
   says the same ("should render the `narrow_79` frame at `render.MinPaneCols - 1` columns if it
   expects the narrow-pane line"), and phase 4's spec describes the same frame name as "below 60
   columns, Phase 2's narrow line and the status line". The digits look transposed (`79` for `59`);
   the frame is stored at 59 and the corpus test says so in its comment.
3. **The plan's Task 3 says "no row of it measures more than `W - 1` cells of text at `W = 80`"**, and
   that cannot hold: a body row that fills the 77-cell measure is `3 + 77 = 80` cells of text, and the
   frame's rows are *padded to exactly W* by definition. The check implemented is the one the
   human-verify step describes ("no row runs past the 80th column"): every row of the frame is at
   most `W` cells of text.
4. **The corpus composes its panes with the shell's own geometry, restated.** `internal/render`'s
   test package is an internal test package, so it cannot import `internal/tui` (which imports
   `internal/render`); `golden_test.go` therefore states each frame's window as a row count and
   slices the blocks with `window`/`blockOffset`/`headOffset` — the same rule as `internal/tui`'s
   `scroll`/`scrolled`/`heads`, documented as such. **This was verified by hand rather than assumed**:
   a scratch test in `internal/tui` drove the shipped shell to each state and compared
   `Model.Render` with the stored bytes — `timeline_60/80/120/200`, `timeline_last_80`, `help_80/120`,
   `loading_80`, `empty_80/120`, `expired_80/120`, `offline_80/120`, `schemachange_80/120`,
   `narrow_79`, the four `color_*` rungs and the stalled notice all matched byte for byte (the walk
   `j G G g j`, and `G` again for `timeline_last_80`). The scratch file was removed afterwards; the
   fleet's own `internal/tui` tests are unaffected. `stalled_80`'s window (the reader's place at the
   pane's top) was *not* driven by keys — its status line was, through the shell's own stalled walk.
5. **The tmux capture's socket is private** (`-L xterm-golden`, killed in a `t.Cleanup`), so the
   plan's verification grep — `! tmux ls | grep xterm-golden` — cannot see it on the user's default
   socket. The same intent is checked *in* the test instead: after the cleanup it asserts that no
   server answers on its own socket, and `tmux ls` on the default socket stays untouched.
6. **A real finding about the environment, not a deviation** (details in the capture section): tmux
   3.7's grid counts a ZWJ family as one 2-cell cluster, while the width model's `$TMUX` arm counts 6
   cells. The UI-SPEC's claim that `ansi.StringWidthWc` "is what a multiplexer's grid models" is
   therefore false on this host for that cluster. Nothing was changed in `internal/text` — that is a
   phase-1/2 design decision outside this plan's scope — but the capture harness measures the
   difference, records it with the numbers, and requires it to be confined to the family row. It also
   means the model errs on the side that cannot wrap a row (the frame stops short of the pane's edge
   rather than overflowing it), so the pane never lies; the frames' text and row counts still agree.
7. **Rendering the frames under `TERM=dumb` + `NO_COLOR=1`** (no SGR at all) is a deliberate choice:
   the state frames are the layout and the words, are reviewable as plain text, and the colour half of
   the specification is the four `color_*` rungs plus A1. The corpus pins all four variables for every
   render, as the plan requires.
8. **The `.golden` files hold the frame's bytes verbatim** (as `charmbracelet/x/exp/golden` writes
   them, and as the plan's pinned version does): the state frames are plain text, and the four
   `color_*` files carry their SGR bytes. The helper's failure output renders both sides escaped
   (`\x1b…`), which is how a diff of a coloured frame is read.

## Verification (all run, all green)

| Command | Result |
|---------|--------|
| `go build ./... && go vet ./...` | clean |
| `gofmt -l internal cmd` | empty |
| `go test ./... -count=1` | **503 tests, 18 packages, ok** (459 before this plan) |
| `go test ./internal/render/... -count=1 -run 'TestGoldenFrames' -update` then `-run 'Golden\|Corpus'` + `sha256sum -c` | the corpus is byte-stable: 30 files unchanged by a non-`-update` run |
| `go test ./internal/render/... -count=1 -run TestCaptureTmux -v` | both sizes pass; the pane's text, row count and padding agree; `! tmux ls \| grep xterm-golden` and no server on the harness's socket |
| `go test ./internal/render/ -run TestPtyHarnessPlumbing -count=1 -v` | passes (a real PTY pair, the 80×25 window, the frame verbatim, the report read) |
| `go test ./internal/render/ -run TestCaptureInGhostty -count=1 -v` | the frame check passes; the window leg **skips with a printed reason** on this host (see below) |

Hand verifications the plan's acceptance criteria ask for, each run once and reverted:
- a stored frame edited by one cell (`timeline_80.golden`, last cell of row 1 → `.`): `TestGoldenFrames`
  and `TestCorpusAudit` (A1's cross-frame arm) both fail; reverting restores green.
- a stored frame deleted (`timeline_120_tmux.golden`): both tests fail on the missing file; restoring
  restores green.
- a stored `_tmux` frame with one row removed (`timeline_80_tmux.golden`, row 2 deleted): the capture
  test fails ("the pane painted 24 rows for a frame of 23"); restoring restores green. (A *content*
  change to a stored frame is `TestGoldenFrames`' business, which re-renders the state and does not
  compare the file with itself — verified with the one-cell edit above.)
- the capture test run from **inside a tmux pane** (a second session on a private socket, running the
  compiled test binary) and from a plain shell: both green, and the harness's own socket is left with
  no server either way (`tmux ls` on the default socket reports no server at all).
- the tmux-absent path: the compiled test binary run with `PATH` empty of tmux prints
  "tmux is not on PATH … a silent skip would make a missing tmux look like an agreeing one" and skips.
- `go mod tidy` through `HTTPS_PROXY=http://127.0.0.1:7897`: no new module enters the graph;
  `charmbracelet/x/exp/golden` is added and `charmbracelet/x/term`/`golang.org/x/sys` become direct
  (both were already bubbletea's indirects, and the PTY harness imports them).

## What the human saw — the deferred step

**Nothing yet: this is the one leg that needs a person at a Ghostty window.** The harness is written,
skips elsewhere with a reason, its frame check and its plumbing are tested here, but the window leg
itself could not be run from this session: the sandbox this agent runs in has no Wayland socket and
no session bus (`find / -type s -name 'wayland*'` is empty; the running Ghostty's own root,
`/proc/448486/root/run/user/1000/`, holds neither `wayland-1` nor `bus`; the host-desktop channel
reports `backend: "unavailable"`), so no window can be opened and `wtype`/`grim` have nothing to
attach to. It was **not** faked.

To finish it, in a Ghostty window 80 columns wide (at least 25 rows), in the repository root:

```
go test ./internal/render -run TestCaptureInGhostty -count=1 -v
```

and look at what the harness prints into the window: it is the stored 80-column timeline frame.
Confirm the handle column lines up, the selected block is findable from its `>` margin alone, the
status line is the last row and did not wrap, and no row runs past the 80th column (`NO_COLOR=1`
makes this easier to judge; the frame to compare against is `internal/render/testdata/TestGoldenFrames/timeline_80.golden`).
The test must report `cursor at row 24, column 81` and must not skip. If the window's own width
differs from 80, the test says so and skips rather than passing.

## Next Phase Readiness

- Phase 3's grammars add frames the same way: a new state is one table entry in `corpus(t)`, a new
  rung or environment is one `environment` value, and the audit's file-set and twin arms follow the
  table automatically.
- `-update` is the only way a stored frame changes; every such change is a view-specification change
  and shows up as a diff of the `.golden` files.
- The capture harnesses are reusable as written: task 2's pane capture takes any stored `_tmux`
  frame, and the PTY harness takes any bare frame at its own width (the width constant is
  `ghosttyFrame`'s frame).
