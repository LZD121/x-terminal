---
phase: 02-following-timeline-in-the-pane
plan: 02
subsystem: ui
tags: [width, wcwidth, grapheme-clusters, tmux, charmbracelet-x-ansi, uniseg, property-tests, REND-02]

# Dependency graph
requires:
  - phase: 01
    provides: the module skeleton (go.mod, `go 1.27`, module path) this plan hangs its first dependency on, and the measured note that module fetches need the local proxy
provides:
  - "internal/text — Width, Wrap, Pad, Truncate: the program's only string measurement, dispatching on $TMUX"
  - "the A4 agreement property suite over the adversarial corpus, in both environments, with anti-vacuity arms"
  - "the module's first third-party dependency: github.com/charmbracelet/x/ansi v0.11.9 (+ 4 indirect deps) in go.mod/go.sum"
affects: [02-03-render, 02-04-tui-keys, 02-05-golden-frames, phase-03-grammars]

# Actuals — chars/4 over the files actually changed (6 new files in internal/text + go.mod/go.sum)
actuals:
  tokens: 7050
  tasks: 2
  commits: 0

# Tech tracking
tech-stack:
  added:
    - "github.com/charmbracelet/x/ansi v0.11.9 (direct)"
    - "indirect: clipperhouse/displaywidth v0.11.0, clipperhouse/uax29/v2 v2.7.0, mattn/go-runewidth v0.0.27, lucasb-eyer/go-colorful v1.4.1"
  patterns:
    - "one width model: an environment-dispatched measure (method) plus the cluster iterator of the same module for every cut and break"
    - "agreement (A4) written as a corpus x widths x environments table, with an arm that fails when the implementation is made naive"
    - "Pad fills to exactly w: the field's column is a fact about the frame, not about the value in it"

key-files:
  created:
    - internal/text/width.go
    - internal/text/wrap.go
    - internal/text/pad.go
    - internal/text/truncate.go
    - internal/text/text_test.go
    - internal/text/agreement_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "Wrap does not write a cluster wider than the whole measure (no line may exceed the measure), instead of emitting an over-wide line"
  - "Pad cuts and then fills with spaces to exactly w; the plan's literal 'return Truncate(s, w)' leaves the column short whenever a wide cluster straddles the limit"
  - "Truncate is Method.Truncate guarded by a re-measure and a cluster walk-back, because the pinned v0.11.9 early-returns on grapheme width whatever method it was asked for"
  - "both environments are exercised inside the test binary with t.Setenv, so the suite's result does not depend on the shell it is run from"

patterns-established:
  - "Width is the only measurement; the cluster iterator comes from the same module as the measure, so wrap points and widths cannot disagree"
  - "adversarial corpus entries are named, counted and self-checked against the widths their names claim"
  - "anti-vacuity: prove by hand that a naive Width/Wrap/Pad/Truncate makes the suite fail, then revert"

requirements-completed: [REND-02]

# Coverage metadata
coverage:
  - id: D1
    description: "internal/text with Width, Wrap, Pad, Truncate: one measurement dispatching on $TMUX (wcwidth under tmux, grapheme clusters bare), a greedy space wrap that breaks over-long tokens at cluster boundaries, an exact-width field fill, and a cluster-boundary cut that appends nothing"
    requirement: REND-02
    verification:
      - kind: unit
        ref: "internal/text/text_test.go#TestWidthDispatchesOnTMUX"
        status: pass
      - kind: unit
        ref: "internal/text/text_test.go#TestWidth"
        status: pass
      - kind: unit
        ref: "internal/text/text_test.go#TestWrap"
        status: pass
      - kind: unit
        ref: "internal/text/text_test.go#TestPad"
        status: pass
      - kind: unit
        ref: "internal/text/text_test.go#TestTruncate"
        status: pass
      - kind: unit
        ref: "internal/text/text_test.go#TestTruncateCutsAtAClusterBoundary"
        status: pass
    human_judgment: false
  - id: D2
    description: "The A4 agreement properties hold for every corpus entry at every width in {1,2,3,5,8,57,77,96} in both environments: Pad reports exactly w, Truncate stays within w and lands on a cluster boundary, every wrapped line stays within w and ends in no space, and nothing is lost or duplicated where the measure can hold every cluster"
    requirement: REND-02
    verification:
      - kind: unit
        ref: "internal/text/agreement_test.go#TestA4Agreement"
        status: pass
      - kind: unit
        ref: "internal/text/agreement_test.go#TestCorpusIsWhatItClaims"
        status: pass
      - kind: unit
        ref: "internal/text/agreement_test.go#TestEnvironmentMovesTheMeasure"
        status: pass
      - kind: unit
        ref: "internal/text/agreement_test.go#TestA4ArmsAreNotVacuous"
        status: pass
      - kind: integration
        ref: "TMUX=x go test ./internal/text/... -count=1 && env -u TMUX go test ./internal/text/... -count=1 && (inside a tmux pane) go test ./internal/text/... -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "The module's first third-party dependency resolved: go.mod requires github.com/charmbracelet/x/ansi v0.11.9 with no other UI library, through the local proxy"
    verification:
      - kind: other
        ref: "HTTPS_PROXY=http://127.0.0.1:7897 HTTP_PROXY=http://127.0.0.1:7897 go mod tidy (exit 0, 1.38s) then cat go.mod"
        status: pass
    human_judgment: false

# Metrics
duration: 10min
completed: 2026-10-14
status: complete
---

# Phase 2: Following timeline in the pane — Plan 02 Summary

**`internal/text` lands the program's one width model — Width/Wrap/Pad/Truncate dispatching on `$TMUX`, with an A4 agreement suite over a 19-entry adversarial corpus in both environments and four proven-failing anti-vacuity arms.**

## Performance

- **Duration:** ~10 min of execution (first artifact 06:54:46Z, last 07:01:18Z; plan reading before that)
- **Started:** 2026-10-14T06:52Z (approx.)
- **Completed:** 2026-10-14T07:01Z
- **Tasks:** 2
- **Files modified:** 8 (6 created in `internal/text`, `go.mod`, `go.sum`)

## Accomplishments

- **The four functions, one measure.** `Width` (`internal/text/width.go:47`) is `method().StringWidth(s)`; `method` (`width.go:35`) is the only branch on `$TMUX` in the program: set → `ansi.WcWidth`, unset → `ansi.GraphemeWidth`. `Wrap` (`wrap.go:34`), `Pad` (`pad.go:15`) and `Truncate` (`truncate.go:20`) all walk clusters with `ansi.FirstGraphemeCluster(s[i:], method())`.
- **The cluster iterator and the measure come from the pinned module, so they cannot disagree.** Measured, asserted as literals: `Width("日本語のテキスト") == 16` in both environments; the ZWJ family `"👨👩👧"` is **2 cells with `$TMUX` unset and 6 with it set, in one run**; `"ab日" == 4`; a single emoji `== 2`.
- **Wrap is greedy on spaces and breaks an over-long token at a cluster boundary**, never inserting a hyphen; `Wrap(url, w)` returns every line within `w` for the 220-cell URL at every tested width, while the naive whole-token wrap is over every one of them. `Wrap(s, 0) == Wrap(s, 1)`; an empty input returns no lines.
- **Pad reports exactly the width asked for**: `Width(Pad(s, w)) == w` for every corpus entry at every tested width, including the entries whose wide clusters straddle the limit (where the cut lands a cell short and the fill closes the gap) and the ones where nothing fits (all spaces).
- **Truncate appends nothing and lands on a cluster boundary**: checked against the byte offsets of the input under the active method, verified by walking `s` with the iterator.
- **A4 runs the whole table twice in one binary** (`t.Setenv` and re-run), 19 corpus entries × 8 widths × 2 environments = 304 property checks per property, plus the environment arm proving the dispatch is live (`Wrap(family, 4)` = 1 line bare, 0 lines under tmux; `Wrap(family, 8)` under tmux = one line of 6 cells).
- **The dependency resolves through the proxy**: `HTTPS_PROXY=http://127.0.0.1:7897 HTTP_PROXY=http://127.0.0.1:7897 go mod tidy` → exit 0, wall 1.38 s; `go.mod` gained `github.com/charmbracelet/x/ansi v0.11.9` plus its four transitive requires as indirect.

## Task Commits

No commits are recorded here: the orchestrator commits (executors do not run `git`). Both tasks were verified in the working tree, in order, before this summary was written.

## Files Created/Modified

- `internal/text/width.go` (2613 B) — package doc, `method()` (`$TMUX` dispatch, the only one in the program), `Width`.
- `internal/text/wrap.go` (3615 B) — `Wrap`: greedy on spaces, cluster-boundary break for over-long tokens, no hyphen, no trailing space, empty input → no lines.
- `internal/text/pad.go` (881 B) — `Pad`: cut with `Truncate` when wider than the column, then fill with spaces to exactly `w`; `w < 1` → `""`.
- `internal/text/truncate.go` (1545 B) — `Truncate`: `Method.Truncate` under the active method, re-measured, with a cluster walk-back when the module's grapheme-hardcoded early return hands back something too wide; `w < 1` → `""`.
- `internal/text/text_test.go` (5733 B) — literal-value unit tests for the four functions, plus the one-run two-environment dispatch test.
- `internal/text/agreement_test.go` (12601 B) — the A4 property suite, the corpus self-check, the environment arm, the anti-vacuity arms.
- `go.mod`, `go.sum` — the direct `ansi v0.11.9` require and its four indirect requires with checksums; module path and `go 1.27` untouched; no other UI library.

### ansi symbols used (pinned v0.11.9, verified in the module cache)

| Symbol | Where | Why |
|---|---|---|
| `ansi.Method.StringWidth` | method.go:41 → width.go:78 `stringWidth(m, s)` | the plan's `method().StringWidth(s)`; it is the same function the package-level `ansi.StringWidth` (width.go:66, grapheme) and `ansi.StringWidthWc` (width.go:75, wcwidth) call, both of which the UI-SPEC names |
| `ansi.WcWidth`, `ansi.GraphemeWidth` | method.go:33-34 | the two semantics `method()` selects between |
| `ansi.FirstGraphemeCluster` | parser_decode.go:435 | the only cluster iterator; used by `Wrap`, `Truncate`'s walk-back and the tests' boundary check |
| `ansi.Method.Truncate` | method.go:49 → truncate.go:64 | the cut in `Truncate`, guarded (see Deviations) |
| `ansi.Truncate` | truncate.go:54 | deliberately **not** used: it hardcodes `GraphemeWidth` |

## Decisions Made

- **`$TMUX` is read at call time, never cached.** `Width`/`Wrap`/`Truncate` call `method()`; the tests rely on this (`t.Setenv` between measurements), and it is what makes the environment arm meaningful.
- **A cluster wider than the whole measure is not written by `Wrap`.** The alternative (emit it on a line of its own) puts a line wider than the pane on screen, which is the one thing the measure exists to prevent. Reachable only at absurd measures: the program's measure is `min(W-3-2·depth, 96)` with `W ≥ 60`, so ≥ 57 cells in every frame.
- **`Pad` fills to exactly `w` after cutting**, so `Width(Pad(s, w)) == w` holds even when a wide cluster straddling the limit leaves the cut short. This is what keeps the 16-cell identity column the same width for every post in plan 02-03.
- **`Truncate` keeps the module's escape-sequence awareness** (the plan chose `Method.Truncate` over a hand-rolled cut) and adds a re-measure guard, rather than replacing the cut outright.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Correctness] `Truncate` cannot be the unguarded `method().Truncate(s, w, "")` the plan specifies**

- **Found during:** Task 1, probing the pinned module before writing the cut.
- **Issue:** the plan says implement `Truncate` as `method().Truncate(s, w, "")`. In v0.11.9 that function's early return is `if sw := StringWidth(s); sw <= length { return s }` (truncate.go:66) — `ansi.StringWidth`, i.e. grapheme width, whatever method it was asked for. Under tmux a cluster whose grapheme width fits `w` while its wcwidth width does not therefore comes back whole and **wider than `w`**. Measured: `Method(WcWidth).Truncate(family, 5, "")` returns the whole family, 6 cells wide. That violates the plan's own A4 property (`Width(Truncate(s, w)) <= w`) in the tmux arm, for the corpus's ZWJ family at w ∈ {3, 5}.
- **Fix:** `Truncate` calls `m.Truncate(s, w, "")` and returns it only when `Width(cut) <= w`; otherwise it walks the prefix back cluster by cluster with the same iterator `Wrap` uses. Both paths keep the escape-sequence awareness and the "appends nothing" contract.
- **Files modified:** `internal/text/truncate.go`.
- **Verification:** replacing the body with the grapheme-hardcoded `ansi.Truncate(s, w, "")` makes `TestA4Agreement/tmux_pane/ZWJ_family` fail with `Width(Truncate(s, 2)) = 6, wider than the measure`; reverted and re-run green (below).
- **Committed in:** not committed (orchestrator commits).

**2. [Rule 1 — Correctness] `Pad` must fill to `w` after the cut, not merely return `Truncate(s, w)`**

- **Found during:** Task 1, working the contract `Width(Pad(s, w)) == w` against a wide cluster at the limit.
- **Issue:** the plan's sentence "if `Width(s) > w` return `Truncate(s, w)`" read literally leaves the field short of its column whenever `Truncate` cuts inside a wide cluster: `Truncate("ab日", 3) == "ab"` is 2 cells, so `Width(Pad("ab日", 3)) == 2 ≠ 3`. The UI-SPEC's own row is "`Width(Pad(s, w)) == w` exactly", and plan 02-03 fixes the identity column with it.
- **Fix:** cut with `Truncate` when the value is wider than the column, then append `w - Width(cut)` spaces. `Pad("abcdef", 3) == Truncate("abcdef", 3) == "abc"` still holds, so the plan's named unit test stands as written.
- **Files modified:** `internal/text/pad.go`.
- **Verification:** with `Pad` reverted to the literal reading, `TestPad` and `TestA4Agreement` fail with `Width(Pad(s, 1)) = 0, want exactly 1` and `Width(Pad("ab日", 3)) = 2, want exactly 3`; reverted and re-run green.
- **Committed in:** not committed (orchestrator commits).

**3. [Rule 1 — Correctness] The Task 2 environment arm's "6 cells of the first line" is asserted where a first line exists**

- **Found during:** Task 2, writing the arm the plan describes.
- **Issue:** the plan asks that `Wrap(family, 4)` "returns a different number of lines under the two environments, and with the environment set to tmux the family occupies 6 cells of the first line". Six cells cannot fit in a four-cell measure, so under tmux `Wrap(family, 4)` returns **no line at all** (verified: 1 line bare, 0 lines under tmux — the difference the same bullet asks for), and there is no first line to measure.
- **Fix:** `TestEnvironmentMovesTheMeasure` asserts the differing line counts literally (1 vs 0) and asserts the 6-cell claim where a line exists: `Wrap(family, 8)` under tmux = one line of 6 cells (`Width` of it = 6), one line of 2 cells bare.
- **Files modified:** `internal/text/agreement_test.go`.
- **Verification:** the arm passes with the shipped code and its `TMUX` half fails when `Width`/`Truncate`/`Wrap` are made naive (arms 1-4 below).
- **Committed in:** not committed (orchestrator commits).

**4. [Rule 1 — Correctness] The `Wrap` doc comment's claim about the module's wrap functions was not true of `ansi.Wrap`**

- **Found during:** Task 1, checking the comment the plan dictated against the pinned source.
- **Issue:** the plan's comment text says `ansi.Wordwrap`, `ansi.Wrap` and `Method.Wordwrap` all "leave a token wider than the limit unbroken". Measured on v0.11.9: `Wordwrap` does leave it whole (the 220-cell URL is one 220-cell line at w=8), but `Wrap` **does** break it (28 lines, widest 8) and can itself hand back a line wider than the limit (`Wrap(family, 1, "-")` = `"\n👨👩👧"`, a leading empty line and a 2-cell line at a 1-cell measure), and it decides for itself where a broken token's remainder joins the next word (`Wrap("aaa bbbbbbbbbbbb ccc", 8)` = `"aaa\nbbbbbbbb\nbbbb ccc"`). Writing the plan's sentence verbatim would have put a false claim in the package.
- **Fix:** the comment now states the verified behaviour of each module function and keeps the decision (hand-written walk, only the cluster iterator shared) with the true reasons: the contract here is `[]string` lines, none wider than the measure and none ending in a space.
- **Files modified:** `internal/text/wrap.go` (comment only).
- **Verification:** each claim in the comment was produced by running the pinned module (see the arm log below).
- **Committed in:** not committed (orchestrator commits).

---

**Total deviations:** 4 auto-fixed (all Rule 1 — correctness; three of them are places where the plan's literal text and the plan's own acceptance properties contradicted each other, resolved in favour of the properties; one is a comment that would have asserted something false about the pinned module).
**Impact on plan:** no scope creep, no API change: the four exported signatures are exactly the UI-SPEC's, and every property and anti-vacuity arm the plan lists is present and passing.

## Evidence

Commands run (all from `<repo>`; the shared context restricts this executor to its own package, so no `./...` command was run):

| Command | Result |
|---|---|
| `HTTPS_PROXY=http://127.0.0.1:7897 HTTP_PROXY=http://127.0.0.1:7897 go mod tidy` | exit 0, 1.38 s wall; `go.mod` = ansi v0.11.9 + 4 indirect |
| `go build ./internal/text/...` | ok |
| Task 1 verify: `go test ./internal/text/... -count=1 && go vet ./internal/text/... && ! grep -rn 'fmt\.' internal/text/ && ! grep -rn 'unicode/utf8' internal/text/` | `TASK1 VERIFY OK` — 67 test cases passed; vet clean; both greps empty |
| Task 2 verify: `go test ./internal/text/... -count=1 -run 'Agreement\|A4' -v` | PASS (44 subtests: 19 cases × 2 environments + 3 top-level + 3 more) |
| Task 2 verify: `TMUX=x go test ./internal/text/... -count=1` | ok |
| `env -u TMUX go test ./internal/text/... -count=1` | ok |
| `TERM=dumb NO_COLOR=1 go test ./internal/text/... -count=1` | ok |
| inside a real tmux pane (`tmux new-session -d … 'go test ./internal/text/... -count=1'`) | exit code 0 inside the pane; session killed afterwards, no server left |
| `grep -rn 'utf8\|fmt\.' internal/text/*.go` | no matches |
| `gofmt -l internal/text/` | no output |
| `go doc ./internal/text` | exactly `Pad`, `Truncate`, `Width`, `Wrap` exported |

Anti-vacuity, by hand, then reverted (`diff -r` against a baseline copy confirms each revert):

1. `Width` → `return len(s)`: `TestWidth` fails (`Width("日本語のテキスト") = 24, want 16`), `TestWidthDispatchesOnTMUX` fails, `TestCorpusIsWhatItClaims` fails, `TestA4ArmsAreNotVacuous` fails in **both** environment arms (`len(…) = 24 and Width = 24: this case cannot tell a byte count from a measure`), and `TestA4Agreement` fails and then panics out of `Pad` (`strings: negative Repeat count`) because a byte count breaks the invariant `Width(Truncate(s, w)) <= w` that `Pad` fills from. Reverted.
2. `Wrap` stops breaking an over-long token (the two break checks removed from the cluster loop): `TestA4Agreement` fails on the corpus in both environments (`a line of Wrap(s, 1) = "😀" is 2 cells wide, wider than the measure`), `TestA4ArmsAreNotVacuous` fails with `a line of Wrap(url, 1) = "https://…" is 220 cells wide: the token was not broken` for every width in `{1,2,3,5,8,57,77,96}`, and `TestWrap`/`TestEnvironmentMovesTheMeasure` fail. Reverted.
3. `Pad` returns the bare truncation instead of filling to `w`: `TestPad` fails (`Pad("ab日", 3) = "ab", want "ab "`; `Width(Pad("ab日", 3)) = 2, want exactly 3`) and `TestA4Agreement` fails by name and width (`Width(Pad(s, 1)) = 0, want exactly 1` for the emoji, CJK, ZWJ and boundary entries). Reverted.
4. `Truncate` → grapheme-hardcoded `ansi.Truncate(s, w, "")`: `TestA4Agreement/tmux_pane/ZWJ_family` fails with `Width(Truncate(s, 2)) = 6, wider than the measure (s = "👨👩👧", Truncate = "👨👩👧")`, which is exactly the pinned module's defect. Reverted.

Upstream behaviour probed (scratch module in `/tmp`, module cache, pinned version):

- `Method(WcWidth).Truncate(family, 5, "")` returns the whole family, 6 cells — the guard's reason (deviation 1).
- `Wordwrap(url, 8, "-")` = 1 line of 220 cells; `Wrap(url, 8, "-")` = 28 lines, widest 8; `Wrap(family, 1, "-")` = `"\n👨👩👧"` (deviation 4).
- `FirstGraphemeCluster` returns one space per cluster (a run of spaces is cuttable byte by byte), so the module's ASCII fast path is boundary-compatible with the iterator.

## Corpus and arms (as recorded by the plan's output requirement)

- **Corpus, 19 named entries** (`wantCorpusLen = 19`, asserted, names unique, every entry exercised): a 15-cell handle `longhandle_name`, a 3-cell handle `lzd`, the CJK run `日本語のテキスト` (16 cells), a single emoji `😀` (2), the ZWJ family `👨👩👧` (2 bare / 6 tmux), a URL past 200 cells (`https://example.com/` + 200 `x` = 220 cells, no spaces), a 10 000-cell prose body (13 000 cells), the empty string, a run of spaces, a line of exactly 57 cells, a line of 56 cells, and one entry per width for a wide cluster sitting on that boundary (`"a"×(w-1) + "日" + "bb"` for w ∈ {1,2,3,5,8,57,77,96}).
- **Widths:** `{1, 2, 3, 5, 8, 57, 77, 96}`. **Environments:** `TMUX` unset and `TMUX` set, both inside one test binary.
- **Properties per (entry, width, environment):** `Width(Pad(s, w)) == w`; `Width(Truncate(s, w)) <= w`; `Truncate(s, w)` is a prefix of `s` and its byte length is a cluster start of `s`; every `Wrap(s, w)` line has `Width <= w`; no `Wrap` line ends with a space; and for w ≥ 6 (the corpus's widest cluster) the wrapped lines carry exactly the non-space characters of `s` in order.
- **Environment arm:** `Width(family)` = 2 bare / 6 tmux and they differ; `Wrap(family, 4)` = 1 line bare / 0 lines tmux; `Wrap(family, 8)` under tmux = one line of 6 cells.
- **Corpus arm:** `TestCorpusIsWhatItClaims` holds each named entry to the width and shape its name claims (`"😀" == 2`, the 57-cell line `== 57`, the URL `> 200` with no space, the body `≥ 10 000`, each boundary entry's prefix `== w-1` cells).

## Issues Encountered

- The plan's module-fetch note is accurate and saved time: the cached module still needed the proxy for the checksum/transitive resolution, and `go mod tidy` (not `go get`) is what drops a require that no package imports yet — so `go.mod` was edited first, then the package written, then tidy run: one pass, 1.38 s.
- `t.Setenv` makes `Width` live mid-test: measuring a slice captured in the bare arm after switching `$TMUX` measures it as tmux would. The first version of `TestEnvironmentMovesTheMeasure` did exactly that and the arm caught it; each arm now records its own numbers while its own environment is set.

## User Setup Required

None — no external service configuration. The dependency is fetched from the local module cache through the existing proxy:

```
HTTPS_PROXY=http://127.0.0.1:7897 HTTP_PROXY=http://127.0.0.1:7897 go mod tidy
```

## Next Phase Readiness

- Plans 02-03 (`internal/render`) and 02-04 (`internal/tui`) can measure, wrap, pad and cut through `internal/text` only: `Width(s)`, `Wrap(s, w) []string`, `Pad(s, w) string`, `Truncate(s, w) string`, with the environment handled inside the package.
- The renderer measures **plain** content and colours runs afterwards: `Width` follows the module and counts an ANSI sequence as zero cells, while `Wrap`/`Truncate` walk clusters and expect text, not escape sequences. Any later plan that wants to cut coloured text has to measure first and colour after, or widen this package's contract.
- Not verified here, by the shared context's rule (a peer edits other packages): `go build ./...`, `go vet ./...`, `go test ./... -count=1`. The dependency is additive — Phase 1 packages do not import `ansi` — and the orchestrator's full-suite run is what settles that Phase 1's 210 tests still pass with it.
- Plan 02-05's capture half is where "the running terminal agreed" gets its own evidence; this plan proves the model against the pinned semantics, not against a live Ghostty or tmux window.

---
*Phase: 02-following-timeline-in-the-pane*
*Completed: 2026-10-14*
