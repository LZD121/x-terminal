---
phase: 07-media-links-and-clipboard
plan: 03
type: summary
status: complete
requirements: [REND-07, DSGT-10]
subsystem: ui
tags: [bubbletea, clipboard, permalink, gate, goldens]
requires:
  - phase: 07-media-links-and-clipboard
    provides: media seam OpenURL, Clipboard chain, Gated markers, y/o keys (07-01/02 + siblings)
provides:
  - dispatcher cover-breaking gate
  - y/o actions with notices + permalink fallback
  - recording Clipboard proof and covered gated goldens
affects: []
actuals:
  tokens: 0
  tasks: 3
  commits: 0
tech-stack:
  added: []
  patterns: [one-dispatcher-gate, permalink-fallback-row, recording-clipboard]
key-files:
  created:
    - internal/tui/links.go
  modified:
    - internal/tui/update.go
    - internal/tui/view.go
    - internal/tui/model.go
    - internal/tui/media.go
    - internal/tui/tui_test.go
    - internal/tui/state_test.go
    - internal/fake/media.go
    - internal/media/spawn.go
    - internal/cli/feed.go
    - internal/render/golden_test.go
    - internal/render/golden_thread_test.go
    - internal/render/corpus_test.go
    - internal/render/testdata/TestGoldenFrames
key-decisions:
  - "Deps.Clip is a tiny Copy interface so *fake.Clipboard can record; production wires media.Clipboard value."
  - "Covered gated goldens reuse absorbed() adversarial feed so corpus alias/product arms stay armed."
patterns-established:
  - "Gate after help short-circuit / before switch; action bodies never test state."
requirements-completed: [REND-07, DSGT-10]
coverage:
  - id: D1
    description: Dispatcher gate refuses media/copy-link/open-post under cover with one-field status diff
    requirement: DSGT-10
    verification:
      - kind: unit
        ref: internal/tui/tui_test.go#TestCoverBreakingActionsGated
        status: pass
    human_judgment: false
  - id: D2
    description: y copies permalink / o opens permalink; NoClipboard and NoOpener print url row
    requirement: REND-07
    verification:
      - kind: unit
        ref: internal/tui/tui_test.go#TestCopyLink
        status: pass
    human_judgment: false
  - id: D3
    description: Covered gated goldens and corpus audit for refusal frames
    requirement: DSGT-10
    verification:
      - kind: unit
        ref: internal/render/corpus_test.go#TestCorpusGatedMediaFrames
        status: pass
    human_judgment: false
duration: 0min
completed: 2026-10-15
verified: |
  gofmt -l internal cmd                         empty
  CGO_ENABLED=0 go build ./...                  ok
  CGO_ENABLED=0 go test ./... -count=1          18 packages ok / 0 fail
  gate disabled (commented): TestCoverBreakingActionsGated/servicelog/i
    FAIL tui_test.go:1820: renderer called under cover: open=4 url=0
    (gate restored; test passes)
unverified: |
  live OSC 52 / wl-copy / browser open on a real terminal;
  panic CapturePanic flake previously seen was not reproduced in final suite.
---
# 07-03 SUMMARY — REND-07 permalink actions and DSGT-10 gate

**`y`/`o` copy or open the permalink through injected Clip/Media seams; one dispatcher gate refuses all three cover-breaking actions under immersive with a grammar-local notice and zero side effects.**

## Accomplishments
- `internal/tui/update.go` — `coverBreaking` + gate after help short-circuit; arms for `ActionCopyLink` / `ActionOpenPost`.
- `internal/tui/links.go` — `copyLink` / `openPost` cmds; NoClipboard / opener failure set `Fall[id]={Permalink,Link}`.
- `internal/tui/view.go` — `StatusFields.Gated` from `grammar().Markers.Gated`.
- `internal/fake/media.go` — recording `Clipboard` (`Count`/`Last`/`Answer`).
- `internal/media/spawn.go` — `RunWithStdin` for wl-copy; `internal/cli/feed.go` wires `Clip` with `ctx.Stdout`.
- Tests: 4 covers × 3 keys gated table, panic ignore, covered help, `TestCopyLink` / `TestOpenPost`.
- Goldens: `permalink_fallback_80`, `media_copied_{,_wl}_80`, `media_gated_{servicelog,gitlog,ci,internal}_80` (+ tmux twins).

## Gate-disable failure text
```
=== RUN   TestCoverBreakingActionsGated/servicelog/i
    tui_test.go:1820: renderer called under cover: open=4 url=0
--- FAIL: TestCoverBreakingActionsGated (0.00s)
    --- FAIL: TestCoverBreakingActionsGated/servicelog/i (0.00s)
```

## Golden hash audit
**Added (11):** `permalink_fallback_80`, `media_copied_80`, `media_copied_wl_80`, `media_gated_{servicelog,gitlog,ci,internal}_80`, and four `*_tmux` twins for the gated covers.
**Changed:** none of the pre-existing 108 frames.

## Decisions Made
- Clip seam is an interface on `Deps` for recorders; feed still builds a `media.Clipboard` value.
- Gated frames share the adversarial absorbed list with Phase 3 cover frames.

## Deviations from Plan
- Help overlay heights raised in a few existing tests (120×40 / 100×40) so three new media rows fit the pane.
- `TestCorpusGatedMediaFrames` asserts permalink/media URLs absent rather than forbidding every body-text `http` token from the adversarial feed.

## Next Phase Readiness
REND-07 and DSGT-10 closed for the shell; live terminal smoke of OSC 52 / wl-copy remains optional.
