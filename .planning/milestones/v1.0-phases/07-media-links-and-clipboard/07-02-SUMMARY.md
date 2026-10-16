---
phase: 07-media-links-and-clipboard
plan: 02
type: summary
status: complete
requirements: [MEDIA-03]
subsystem: ui
tags: [bubbletea, osc8, goldens, media-fallback]
requires:
  - phase: 07-media-links-and-clipboard
    provides: media.Renderer seam and openMedia dispatch (07-01)
provides:
  - numbered attachment line with in-place URL fallback
  - Fallbacks state + normal-mode-only pass
  - media notice vocabulary and goldens
affects: [07-03, REND-07]
actuals:
  tokens: 0
  tasks: 3
  commits: 0
tech-stack:
  added: []
  patterns: [url-replaces-noun, normal-mode-only-fallbacks, osc8-after-layout]
key-files:
  created:
    - internal/render/media.go
    - internal/render/media_test.go
  modified:
    - internal/render/block.go
    - internal/render/copy.go
    - internal/render/status.go
    - internal/render/golden_test.go
    - internal/render/corpus_test.go
    - internal/tui/media.go
    - internal/tui/model.go
    - internal/fake/media.go
key-decisions:
  - "Notice for a multi-success set is NoticeOpenedMedia / opened {n} media; mixed sets report successes and keep only failed indexes."
  - "First failed attachment decides no-opener vs opened… when no successes remain; with any success the success notice wins."
patterns-established:
  - "URL fallback is a row in the block, never truncated; OSC 8 after layout; zero Fallbacks under a cover."
requirements-completed: [MEDIA-03]
coverage:
  - id: D1
    description: Numbered attachment line and in-place URL fallback with wrap/OSC 8
    requirement: MEDIA-03
    verification:
      - kind: unit
        ref: internal/render/media_test.go#TestAttachmentFallbackSplitsAndWraps
        status: pass
    human_judgment: false
  - id: D2
    description: Status-line notice vocabulary and shell Fallbacks lifecycle
    requirement: MEDIA-03
    verification:
      - kind: unit
        ref: internal/tui/tui_test.go#TestMediaFallbackHiddenUnderCoverRestores
        status: pass
    human_judgment: false
  - id: D3
    description: Golden frames and corpus no-URL-without-failed-action audit
    requirement: MEDIA-03
    verification:
      - kind: unit
        ref: internal/render/corpus_test.go#TestCorpusNoURLWithoutFailedAction
        status: pass
    human_judgment: false
duration: 0min
completed: 2026-10-15
verified: |
  gofmt -l internal cmd                         empty
  CGO_ENABLED=0 go build ./...                  ok
  CGO_ENABLED=0 go test ./... -count=1          1064 passed / 8 skipped
  go test ./internal/render -run 'TestGoldenFrames|TestCorpusAudit' -count=1  ok
  planted clean-frame URL in TestCorpusNoURLWithoutFailedAction → FAIL
    ("frame planted_media_url prints media URL … without a failed action");
    revert → PASS
unverified: |
  REND-07 permalink `url ` value from plan 03 (row shape landed; Permalink bool wired);
  live OSC 8 click in a real terminal.
---
# 07-02 SUMMARY — MEDIA-03 numbered line and URL fallback

**In-place URL fallbacks replace failed attachment nouns, wrap without truncation, and stay out of covered frames while notices name the outcome.**

## Accomplishments
- `internal/render/media.go` — `MediaFallback`/`Fallbacks`, per-entry markers, `targetURL`, URL wrap at `measure-4`/`url-col`, OSC 8 after layout.
- `Blocks(..., falls Fallbacks)` + `attachmentRows`/`urlRows`; cover markers stay numbered lists with zero falls.
- Shell `frame.Fall` + `mediaAnswered` mapping; normal-mode-only pass in `blocksIn`; fallbacks cleared only by a later success on that post.
- Notices: `opened {noun}` / `opened {n} media` / `no opener - URL shown` / `no media in this post` (+ poster); totality arm.
- Goldens: `media_*` + `color_*_fallback`; regenerated attachment-bearing Phase 2/6 frames only.

## Golden hash audit
**Changed (17, all previously held an attachment line):**
`timeline_last_80`, `bookmarks_{80,120}{,_tmux}`, `bookmarks_syncing_80{,_tmux}`, `bookmarks_absent_rows_80{,_tmux}`, `color_{truecolor,256,16,mono}_saved{,_tmux}`.
**Added (10):** `media_opened_80`, `media_fallback_{80,120}`, `media_no_opener_all_80`, `media_none_80`, `media_wrap_exact_80`, `color_{truecolor,256,16,mono}_fallback`.
No frame without an attachment line moved.

## Decisions Made
- Multi-success → `NoticeOpenedMedia` with count; mixed → success notice + failed indexes only.
- Cover templates rewritten to the same per-entry `[{n}] {noun}` shape (literal words of old prefixes dropped with Phase 2's `attachment:`).

## Deviations from Plan
None material. ThreadBlocks also took `Fallbacks` so conversation frames share the seam. Planted-audit proof used an in-test synthetic frame (table plant would have required a non-fallback URL renderer path that does not exist).

## Next Phase Readiness
Plan 03 can reuse the `url ` row kind for the `o` permalink fallback and the DSGT-10 gate.
