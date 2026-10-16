---
phase: 08-operations-repair-path-and-audit
plan: 03
type: summary
status: complete
requirements: [OPS-06]
subsystem: ops
tags: [runbook, ops-06, dsgt-08, corpus, panic]
requires:
  - phase: 08-operations-repair-path-and-audit
    provides: doctor RunbookAnchors, --probe, mirror line
  - phase: 03-three-state-disguise-on-the-working-renderer
    provides: cover grammars, panic harnesses
provides:
  - RUNBOOK.md four-part repair sections + request discipline
  - internal/cli/runbook_test.go state→section anti-rot suite
  - covered golden cross-product (render + tui) and DSGT-08 corpus arms
  - panic re-verification on post-Phase-3 screens
affects: [repair-path, cover-guarantee]
actuals:
  tokens: 0
  tasks: 3
  commits: 0
tech-stack:
  added: []
  patterns: [state-to-section-table, covered-cross-product, request-discipline]
key-files:
  created:
    - RUNBOOK.md
    - internal/cli/runbook_test.go
    - internal/render/golden_cover_corpus_test.go
  modified:
    - internal/render/golden_test.go
    - internal/render/corpus_test.go
    - internal/render/capture_panic_test.go
    - internal/render/capture_tmux_test.go
    - internal/render/capture_pty_test.go
    - internal/render/saved.go
    - internal/render/copy.go
    - internal/tui/golden_test.go
    - internal/tui/corpus_test.go
    - internal/tui/view.go
    - internal/keys/keys.go
key-decisions:
  - "Coverage table is hand-written over failing xweb/netcheck outcomes; accidental-distribution is the only named orphan exception; likes unavailable maps to #the-likes-endpoint"
  - "Covered 80-col saved frames with a long state word drop synced (same status drop as capability frames); TestSyncedFieldCorpusRule exempts them via the cover state word"
  - "Help covered frames use h=36 so three media HelpCovered rows stay on screen for key-column comparison"
patterns-established:
  - "Doctor pointer ↔ RUNBOOK heading is a double-sided contract (RunbookAnchors + headingExists)"
  - "Missing covered frame = missing golden file = TestGoldenFrames failure"
requirements-completed: [OPS-06]
coverage:
  - id: D1
    description: "RUNBOOK.md fourteen cases + Request discipline; four parts; verbatim symptoms; built Verify commands"
    requirement: OPS-06
    verification:
      - kind: unit
        ref: "internal/cli/runbook_test.go#TestRunbook"
        status: pass
      - kind: manual_procedural
        ref: "remove soft-ban coverage row / rename ## Soft ban / truncate Soft ban to one part → FAIL; restore → PASS"
        status: pass
    human_judgment: false
  - id: D2
    description: "Covered corpus cross-product + no-identity / no-http / help keys / gated / file-set arms"
    requirement: OPS-06
    verification:
      - kind: unit
        ref: "internal/render/... TestGoldenFrames|TestCorpusAudit; internal/tui/... TestGoldenFrames|TestCorpusAudit|TestCoverBreakingActionsGated"
        status: pass
      - kind: manual_procedural
        ref: "plant handle lzd into cover_gitlog_80.golden → V2 FAIL; restore → PASS"
        status: pass
    human_judgment: false
  - id: D3
    description: "Panic re-verified on later screens (thread/bookmarks/likes/media_detail) via tmux; Ghostty window skips with named command"
    requirement: OPS-06
    verification:
      - kind: unit
        ref: "internal/render/capture_panic_test.go#TestCapturePanic"
        status: pass
    human_judgment: false
verified: |
  gofmt -l internal cmd                         empty
  CGO_ENABLED=0 go build ./...                  ok
  CGO_ENABLED=0 go test ./... -count=1          all packages ok (22 packages)
  go test ./internal/cli -run TestRunbook -v    prints Verify command per section
  go test ./internal/render -run TestCapturePanic -v
    tmux shipped: before=62 labelled → after=0
    later screens: thread/bookmarks/likes/media_detail before=62 → after=0 each
    window (Ghostty): SKIP (TERM_PROGRAM=""; command named in skip)
unverified: |
  Bare-Ghostty panic leg needs a Ghostty window:
    go test ./internal/render -run TestCapturePanic -count=1 -v
duration: 120min
completed: 2026-10-15
---
# 08-03 SUMMARY — runbook + covered re-audit (OPS-06)

**`RUNBOOK.md` is the repair path's written half; every failing classifier state has a section; every screen since Phase 3 is stored under every cover at both widths.**

## Accomplishments

- `RUNBOOK.md` — Request discipline + 14 four-part sections (anchors match `RunbookAnchors`).
- `internal/cli/runbook_test.go` — total coverage table, doctor-pointer arms, exercisability (prints command found), four-part order, verbatim symptoms, orphan exception for accidental-distribution, likes→`#the-likes-endpoint`.
- Covered cross-product — `coveredCrossProduct` grows thread/bookmarks/likes/media_detail/help × 4 styles × 80/120; tui grows search/people/profile covers to both widths; media_gated_* at 120 completes the width product.
- Corpus arms — no real identity / product nouns / URLs over whole covered set; help key column byte-identical; gated twelve combos still refuse; file-set for gated + covered help.
- Panic — tmux leg re-run with later covered screens on screen; before/after label counts prove history clear; Ghostty window skip names the command.

## Deliberate failures

- Soft-ban coverage row removed → `TestRunbook/coverage` FAIL → restored.
- `## Soft ban` renamed → FAIL → restored.
- Soft-ban section truncated to Symptom only → `TestRunbook/four` FAIL → restored.
- Planted `lzd` into `cover_gitlog_80.golden` → `TestCorpusAudit/V2` FAIL → restored.

## Decisions

- Covered 80-col saved frames that carry a long failure word drop `synced` (status drop order); the synced corpus rule accepts the cover state word there instead of demanding `cached`.
- Help covered goldens use pane height 36 so media HelpCovered rows are not clipped.

## Files

- `RUNBOOK.md`, `internal/cli/runbook_test.go`
- `internal/render/{golden_test,golden_cover_corpus_test,corpus_test,capture_panic_test,capture_tmux_test,capture_pty_test,saved,copy}.go`
- `internal/tui/{golden_test,corpus_test,view}.go`, `internal/keys/keys.go`
- `internal/render/testdata/TestGoldenFrames/*`, `internal/tui/testdata/TestGoldenFrames/*`
