---
phase: 06-bookmarks-likes-and-the-mirror-only-store
plan: 02
type: summary
status: complete
requirements: [SAVE-01, XAPI-04]
verified: |
  gofmt -l internal cmd                         empty
  go build ./...                                ok
  go vet ./internal/domain/ ./internal/xsource/ ./internal/fake/  ok
  go test ./internal/domain/... ./internal/xsource/... ./internal/fake/... -count=1  ok
  go test ./... -count=1                        ok (render TestCapturePanic flaked once on tmux server; re-run PASS)
  unshare -rn (lo up) go test ./internal/xsource/... ./internal/fake/... ./internal/domain/...  ok
  grep -q opBookmarks/opLikes saved.go          ok
  grep -c X_TERMINAL_CAPTURE capture_test.go    4
  go test ./internal/xsource/ -run TestCaptureBookmarks -v  SKIP naming X_TERMINAL_CAPTURE
  go test ./internal/fake/ -run TestFakeSavedPagesMatchTheFixture  FAIL after hand-edited id; PASS after revert
  live capture (X_TERMINAL_CAPTURE=1): Bookmarks list 146490 B / 20 posts; Likes list 107736 B / 19 posts
unverified: |
  Composition-root wiring of WithViewerID from Viewer into the sync path — plan 03.
  Screen/pane consumption of domain.Source.Saved — later plans; pane reads the mirror on open.
decision: |
  Likes variables are the measured request: userId + count=20 + cursor + includePromotedContent=false.
  Adapter.WithViewerID carries userId so Saved stays one GraphQL call; Viewer remains plan 03.
  ParseTimeline accepts data.bookmark_timeline_v2 (Bookmarks) in addition to user timelines; empty
  tweet_results.result entries are skipped+counted so Likes pages with one empty slot still parse.
---
# 06-02 SUMMARY — bookmarks/likes reads, fixtures, fake, pin path (SAVE-01, XAPI-04)

## Likes measurement

**Measured 2026-10-15 with the live session and proxy:** `Likes` is in the registry
(`mu4us66Q6RRNOywVQhGtRg`) and the server answered with a **list** (107,736 bytes after
redaction; 19 posts / 20 tweet entries requested; one empty `tweet_results` skipped).
Provenance notes say so without hedging. Bookmarks likewise answered with a list
(146,490 bytes; 20 posts; bottom cursor present).

## What landed

- `internal/domain/source.go:47` — `Saved(ctx, SavedKind, Cursor) (PostPage, error)`.
- `internal/xsource/saved.go:15,30` — `opBookmarks`/`opLikes`, `(*Adapter).Saved`; names only, never query ids.
- `internal/xsource/home.go:62` — `WithViewerID` for Likes' measured `userId`.
- `internal/xsource/tweets.go:73,513` — shared entry walk via `timelineFrom` + `bookmark_timeline_v2`; empty tweet results skipped.
- `internal/xsource/testdata/fixture-{bookmarks,likes}.json` + provenance rows (redacted bodies/ids/handles; post text → same-length `x`).
- `internal/xsource/capture_test.go` — `TestCaptureBookmarks` / `TestCaptureLikes` gated on `X_TERMINAL_CAPTURE=1`.
- `internal/xsource/saved_test.go` — decoder + request variables + `TestPinPathIsBothHalvesOfXAPI04` (wrong pin → `OutcomeUnknownOperation` + empty page; right pin → posts; missing op → `ErrUnknownOperation`, call count 0).
- `internal/fake/saved.go:69,121` — `Saved`, `FailSaved`, `Unavailable`, `StallSavedWith`; `pages-saved.json` parity with bookmarks fixture parse.

## Decisions the plan missed

1. Bookmarks envelope is `data.bookmark_timeline_v2`, not `data.user…` — extended `ParseTimeline` rather than inventing a second walk.
2. Likes needs `userId`; added `WithViewerID` so the port stays one method and one GraphQL call.
3. Live Likes can carry an empty `tweet_results.result`; skip+count instead of refusing the page.
