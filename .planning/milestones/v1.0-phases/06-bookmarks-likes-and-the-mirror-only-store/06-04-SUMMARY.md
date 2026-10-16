---
phase: 06-bookmarks-likes-and-the-mirror-only-store
plan: 04
type: summary
status: complete
requirements: [SAVE-02, SAVE-03]
verified: |
  gofmt -l internal cmd                         empty
  go vet ./...                                  ok
  go test ./... -count=1                        1017 tests / 0 fail / 8 skip
  unshare -rn (lo up) go test ./internal/tui/ -run TestSavedFrameIssuesNoSourceRequest  ok
  planted bookmarks_80.golden synced→XXXXXXXXXXXX  FAIL then revert PASS
  grep store.Open in internal/{tui,screen,render}  empty
unverified: |
  Live `go run ./cmd/x-terminal feed` with proxy stopped — not run in this worker;
  behavioural zero-request proof is the fake-mirror tui test (+ unshare).
decision: |
  Capability frames at ≤80 columns may drop `synced` (06-UI-SPEC named compromise);
  corpus requires the capability word there instead of the synced form.
  tea.BatchMsg unwrapped in the tui test harness so mirror Page+Status both land.
  Help overlay test height raised 26→30 for the two new keymap rows.
---
# 06-04 SUMMARY — saved frame, synced field, sync on `r` (SAVE-02, SAVE-03)

## Zero requests on open

`internal/keys/keys.go:164-165` — `b`/`l` → `ActionBookmarks`/`ActionLikes`, help
`open bookmarks` / `open likes`, group `lists`.
`internal/screen/saved.go` — `Saved` + Mirror/Sync/Status/Capability events;
`NoticeFor` table.
`internal/fake/mirror.go` — scripted `domain.Mirror` with Page/Begin/Step/Abandon counters.
`internal/tui/saved.go` — `pushSaved` issues only `Saved.Page`/`Status`; Source untouched.
`TestSavedFrameIssuesNoSourceRequest` — call count 0 across open/page/pop; byte-identical restore.

## One sync implementation

`r` on a saved frame drives `Saved.Begin` → `Step` loop → re-`Page` from "";
pop mid-walk calls `Abandon(StopAbandoned)` once. Same `store.Mirror` the verb uses.

## Capability + copy

`internal/cli/feed.go` — `openSaved` seam; `SavedCapability`/`SavedError`/`MirrorPath`
wired only at the composition root (`store.Open` nowhere else in tui/screen/render).
`internal/render/{copy,saved,status,grammar}.go` — synced field after `list`;
drop order notice→position→list→synced; capability words/remedies/pane blocks;
likes hedge one line; `Markers.Sync`/`SyncNever` on every grammar.

## Goldens

24 new frames under `testdata/TestGoldenFrames/` (bookmarks/likes/capability/syncing/
absent/color_*_saved + twins). Corpus asserts synced presence/absence both ways;
cover leak scan for `x.com`/`doctor`/`saved.db`/`pins.json`.
