---
phase: 04-post-detail-threads-and-quotes
plan: 03
status: complete
requirements: [THRD-03]
created: "2026-10-14"
---

# 04-03 — The conversation on screen: markers, frames, `enter`

## What landed

| File | What it holds |
|------|---------------|
| `internal/render/copy.go` | eight conversation markers + six status/notice words, each with covered twins; `NoticeLine` arms for replies/nothing-to-open; `ThreadListName` |
| `internal/render/grammar.go` | `MarkerSet` +8 fields; `Grammar.ReplyIndent`; `measure(w, depth, replyLevel)` reproducing Phase 2 at replyLevel 0 |
| `internal/render/cover.go` | all four covers carry `ReplyIndent` and the eight markers; `CoverOfGrammar` |
| `internal/render/status.go` | `NoticeLoadingReplies` / `NewReplies` / `NoNewReplies` / `NothingToOpen`; `StatusFields.List` is the list's own word (`string`) |
| `internal/render/block.go` | builder `indent`/`reply`/`surface`/`lead`; `post(..., focal, parent)`; focal + reply-to markers; surface headless tombstones; withheld literal only on the conversation surface |
| `internal/render/thread.go` | `Conversation`, `ThreadBlocks` — ancestors, focal, continuations, replies, conditional statements |
| `internal/render/thread_test.go`, `golden_thread_test.go`, `corpus_test.go` | row pins, stored frames at 60/80/120/200 + partial/visibility/empty/chaincut/narrow/color + adversarial/selection/status variants; adversarial audit with anti-vacuity |
| `internal/fake/thread.go` (+ parity + fixtures) | fixture-backed `Thread` source (`ThreadRecorded`/`Shapes`/`Tombstone`), `Calls`/`FailNext`/`StallWith`, decoder parity |
| `internal/keys/keys.go` | `Screens []string`; `enter`→`open`; `t`→`switch-list` screens `home`; `Lookup(k, screen)` |
| `internal/tui/{model,update,view}.go` | `frameKind` home/thread; `Deps.Threads`; `open()`; thread answer routing by frame id; `ThreadBlocks` pane + thread status/help |
| `internal/cli/feed.go` | `threadSource` seam beside `feedSource`; both handed to `tui.Deps` |
| `internal/screen/thread.go` | refresh/`Anchor` support the shell's `r` and restore (small append for Task 3) |

## Verification (exact commands)

- `go build ./...` → OK
- `go test ./... -count=1` → **778 passed, 19 packages**; `gofmt -l internal cmd` empty
- Task greps: `ReplyIndent` ≥1 / `ReplyIndent:` 1; `ThreadBlocks` 1; `Screens []string` 1; `ActionOpen` 1; `switch-list` 1; `open()` 1; `threadSource` ≥1; `(*Thread).Thread` 1; three constructors 3; 14 required `thread_*.golden` present
- Anti-vacuity: renaming a required case's frame to `thread_MISSING_FRAME` → `TestCorpusAudit/the_required_cases` failed; reverted → green
- Smoke (`TestSmokeEnterAgainstFake` against `fake.Adversarial` + `fake.ThreadShapes`, 80×24):
  - after `enter`: focal held copy + `loading thread...`, `thread.Calls` delta 0, status `thread`
  - after page one: shapes conversation (continued, depth-2 quote, limited, tombstone, withheld), `focal=true`, loading gone, `thread 4/11+`
- Enter suite: `TestEnterOpensAConversationWithLoadingThenPage`, quote/tombstone/nothing-to-open, `q` restore, `t` noop + overlay, pagination single-flight, stall, failed first page — all green

## Tests watched failing (broken deliberately, then reverted)

1. Corpus required-case frame renamed to a missing name → audit failed on the absent frame (anti-vacuity arm).
2. (Sibling) shell/render work also exercised break-then-fix for stall / open / measure where the plan required it; final tree is green with those edits reverted or corrected.

## Decisions the plan missed (and the choices made)

- **List withheld wording stays Phase 2's `post unavailable`**: only a conversation `surface` builder uses `Markers.Withheld`, so Phase 2's stored list frames do not move.
- **`StatusFields.List` is a string** (as the plan asked): callers pass `ListName` / `ThreadListName`; covered status tests set the covered name in `coveredWorked`.
- **Tombstone focal**: a headless record still draws no head row, with one exception — a headless *focal* draws a head row carrying only the `focal` marker above the state marker, because A1-T requires a stripped thread frame to name the focal post even when its text is unavailable (the first version drew no head row at all, so the state could not carry the marker). `thread_tombstone_focal_80.golden` carries that row and the audit's required case requires the marker; the marker is still read as a free head-row word, because English "focal" in bodies must not false-positive. *(Fixed after 04-VERIFICATION.md.)*
- **`ranking` removed from the forbidden substring list**: the scan reads the whole frame, the writers' body rows included, and the reply in `thread_partial_80/120` reads "that is the reply ranking, not the walk" — content, not chrome, so a frame-wide audit cannot forbid the word. *(This decision was first recorded here as a collision with the status line's `rate-limited`; that is wrong — `rate-limited` never matched `ranking`. The true cause is the post's own text.)* `ranked` / `chronological` / `load more` remain, and `internal/render/corpus_test.go` carries the note beside the list.
- **Empty-read body reuse**: `CopyEmptyBody` under `no replies` is allowed on `paneThread`; the empty-state heading alone remains exclusive to `paneEmpty`.
- **Shell**: process-wide outstanding count; answers routed by opaque frame id; `ThreadRefreshRequested` for `r`; selection rebound after page-one focal replace; home `List.Anchor` mirrored on the frame for restore.
- **Fake stall**: unknown-cursor / stall behaviour aligned so a repeated cursor ends the walk without inventing pages.

## Unverified here

- Live tmux twin capture of a thread frame against a real terminal (plan verification checklist item); stored frames + corpus A1/A7 cover the same bytes the engine produces.
- Interactive binary against a live session (smoke used the fake source only, as required).
