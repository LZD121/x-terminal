---
phase: 04-post-detail-threads-and-quotes
plan: 01
type: summary
status: complete
requirements: [THRD-01]
verified: |
  go build ./... && go vet ./...                 clean
  gofmt -l internal cmd                          empty
  go test ./... -count=1                         703 passed in 19 packages
  sha256sum internal/xsource/testdata/fixture-tweetdetail.json .planning/research/evidence/fixture-tweetdetail.json   526ca1f8… both, 202462 bytes
  go test ./internal/xsource -count=1 -run 'TestEveryFixtureIsProvenanced|TestTheRecordedFixtureNamesItsSource|TestTheRecordedFixtureIsTheOneWeThinkItIs|TestNoFixtureCarriesViewerIdentity' -v   3 PASS, 1 SKIP (the identity arm: no identity in the session, no X_TERMINAL_VIEWER_* set)
  go test ./internal/xsource -count=1 -run 'TestCaptureTweetDetail|TestEveryFixtureIsProvenanced' -v   SKIP (gate off) + PASS
  go test ./internal/domain ./internal/xsource -count=1 && go vet ./internal/domain ./internal/xsource   ok, clean
---
# 04-01 SUMMARY — the recorded thread, its provenance, and the entry grammar

## 1. The recorded body and its record (Task 1)

`internal/xsource/testdata/fixture-tweetdetail.json` is the evidence file copied byte for byte
(**202462 bytes**, `sha256 526ca1f8081f1cbb42335b148f55997d6e48cdfd2ce84d5ae69bc372222597f3` on both
sides). `testdata/provenance.json` carries one row per fixture in `testdata/` — this body, Phase 1's
`fixture-usertweets.json` (required by the plan's own rule that *every* fixture is provenanced; a
timeline has no focal post, so its `focalId` is empty and its `sourceFile` is the Phase 1 evidence
file), and the three synthetic bodies — with `file, operation, queryId, focalId, capturedAt, bytes,
sha256, source, sourceFile, redaction, notes` in that order.

The census, taken from the raw JSON by `jq` and independently by `TestTheRecordedFixtureIsTheOneWeThinkItIs`:

| measured | 43 entries | 41 `conversationthread` | 1 `tweet` | 1 `cursor` | 41 module items | 42 tweet items | 5 promoted | focal `reply_count` 8121 |
|---|---|---|---|---|---|---|---|---|

Four tests pin it: `TestEveryFixtureIsProvenanced` (digest and byte count against the file on disk, no
row naming a file that is not there, `synthetic` ⇒ non-empty `notes`, and which bodies are recorded by
name), `TestTheRecordedFixtureNamesItsSource` (the evidence path, the operation, the registry's own
`QueryID` for it, and the copy's digest read from the module root), `TestTheRecordedFixtureIsTheOneWeThinkItIs`
(the census above), `TestNoFixtureCarriesViewerIdentity` (see §2), plus
`TestEveryProvenanceRowResolvesThroughTheRegistry` over every row.

## 2. The capture harness and the synthetic set (Task 2)

`internal/xsource/capture_test.go`, gated on `X_TERMINAL_CAPTURE=1` (a normal run prints
`--- SKIP: TestCaptureTweetDetail` naming the variable). When gated it builds the client the way
`internal/cli` does, issues **one** `TweetDetail` call through a counting `GraphQLer` (`focalTweetId`
from the provenance row, empty `cursor`), rewrites the viewer's identity, writes
`testdata/fixture-tweetdetail.json` and rewrites that body's provenance row (`bytes`, `sha256`,
`capturedAt` from the package clock, the redaction sentence). No socket was opened here: the live path
is unexercised.

Three deviations, each reported rather than hidden:

1. **`internal/session` carries the two cookies and no identity** (deliberately, SEC-01), so "the
   viewer's identity, read from the loaded session" is read from the session *file*'s own
   `twid`/`id`/`screen_name` lines, with `X_TERMINAL_VIEWER_ID` and `X_TERMINAL_VIEWER_HANDLE`
   overriding them. With neither, the harness **fails closed** rather than writing a body it cannot
   clean.
2. **The account id is rewritten wherever its digits appear; the handle only where it is a whole JSON
   string value.** A handle can be one character long, and a bare replacement would rewrite that
   character everywhere — a body destroyed rather than redacted. The row's `redaction` sentence names
   exactly what was applied, including "nothing rewritten" when the answer carried neither.
3. **A freshly captured body clears the row's `sourceFile`** (it is no longer the evidence file's copy)
   and the harness says so in its log: copying the new body into `.planning/research/evidence/` and
   re-pointing the row is the operator's step, and the source test fails until it happens.

The three synthetic bodies (15, 1 and 3 entries) carry the shapes the recording has none of: the
ancestor chain, the focal post, a `SelfThread` module of two items, a quote two levels deep, a module
with its own `ShowMoreThreads` handle, a tombstone reply, a `TweetWithVisibilityResults` reply, a
reply whose `legacy.withheld_in_countries` is non-empty, the rail, the message prompt, the composer
slot, a promoted module and a `who-to-follow-` entry carrying an item class this build has never seen.
All three rows are `source: "synthetic"` with `sourceFile` empty and notes naming every shape.

## 3. The port and the entry grammar (Task 3)

`internal/domain/thread.go` — `GroupKind` (`SelfThread` zero, `Replies`), `Group{Kind, Posts, More}`,
`ThreadPage{Focal, Ancestors, Groups, Next, Requested, Skipped, FetchedAt}` and the `Threads` port,
declared beside `Source` rather than inside it. `internal/xsource/detail.go` — `ParseThread`,
`(*Adapter).Thread`, `var _ domain.Threads = (*Adapter)(nil)`, and the walker.

What the recorded body decodes to: **focal** = the row's `focalId`, **ancestors 0**, **groups 36**,
`Next` = the body's own bottom-cursor value, **Requested 42**, **Skipped 6**. The group order is the
answer's order, every kept post's `ConversationID` is the focal post's, the five advertised posts
(the ones whose conversation differs) appear nowhere in the page, and every post's `ReplyTo` is the
`in_reply_to_status_id_str` the body wrote.

The classifier, by entry id and content: `TimelineTimelineCursor` → the page's cursor (`Bottom`) or a
module's `More` (`ShowMore`/`ShowMoreThreads`), counted; `tweet-<focalId>` → the focal post;
`tweet-` before it → an ancestor; `tweet-` after it → a reply group of one; `conversationthread-` →
a group (`SelfThread` from the item's `tweetDisplayType`, else `Replies`), its promoted items dropped
and counted; the rail, the prompt and the composer → counted; anything else → counted. Posts are built
by `unwrapTweet`/`postFrom`/`countsOf`/`authorOf` — no second tweet decoder, and `ParseTimeline` and
`timelineEnvelope` are untouched.

Four decisions the plan left open, all documented in the code:

| # | Reading taken | Why |
|---|---|---|
| a | the page carries **36** groups, not the 41 the plan's one behavior line names | the same line's `Requested` 42 / `Skipped` 6 means the 41 modules include the 5 promoted items; the plan's must_haves ("42 posts out of 43 entries") and the acceptance's raw-count comparison agree, and the count is taken from the raw JSON anyway |
| b | `ParseThread(raw, focal)` takes the focal id | nothing in an answer marks which `tweet-` entry is the post that was asked for once ancestors precede it; deriving it (the deepest link) is the tree this walker refuses to build |
| c | `Next` is the first group **that has** an unspent handle | the plan also says no cursor the answer carried may be dropped, and the synthetic handle module is not the first group |
| d | a standalone `tweet-` entry *after* the focal entry is a group of one | the shapes the plan lists there (the tombstone, the limited reply) are required to decode; skipping them is the silent drop this phase guards against |
| e | withheld = `legacy.withheld_in_countries` non-empty, applied to the built post | the plan's own narrowest rule, recorded as an inference; a withheld tombstone still reads as a post (id from its entry id) rather than an error |

`detail.go` added two fields to the shared wire structs in `tweets.go`
(`itemContent.TweetDisplayType/CursorType/Value`, `tweetLegacy.WithheldInCountries`) rather than
describing the item shape twice.

## 4. Commands the work was accepted on

```
go build ./... && go vet ./...                      clean
gofmt -l internal cmd                               empty
go test ./... -count=1                              703 passed in 19 packages
sha256sum testdata/fixture-tweetdetail.json evidence/fixture-tweetdetail.json   526ca1f8…, identical
go test ./internal/xsource -run 'TestEveryFixtureIsProvenanced|TestTheRecordedFixtureNamesItsSource|TestTheRecordedFixtureIsTheOneWeThinkItIs|TestNoFixtureCarriesViewerIdentity' -v   3 PASS, 1 SKIP
go test ./internal/xsource -run 'TestCaptureTweetDetail|TestEveryFixtureIsProvenanced' -v   SKIP + PASS
grep -c 'func ParseThread' internal/xsource/detail.go                    1
grep -c 'func (a \*Adapter) Thread' internal/xsource/detail.go           1
grep -rl '3BlMtudy4N-d78ch7zTLZQ' internal/xsource/*.go | wc -l          0
grep -c '2106019976381689871' internal/xsource/detail_test.go            0
```

Break-once checks, each run and reverted (the decoder's own tests, not the fixture layer's):

| Planted | Test | Result |
|---|---|---|
| advertisements kept (`if false && …promotedItemMark`) | 6 tests | 41 groups vs 36, five other-conversation posts on the page, `Skipped` 4, the advertised post in the shapes page |
| the module-handle fallback for `Next` removed | `TestTheSyntheticShapesAreAllRead` | `Next is "", want the module's own handle "SYNTHETIC-SHOW-MORE-THREADS"` |
| the first-page refusal removed | `TestAMissingFocalFirstPageIsRefused` | `reported <nil>, want a *ParseError` |
| one entry deleted from the recorded fixture | census + provenance + decoder | `42 entries, want the measured 43`, digest mismatch, 40 modules; restored, digest identical |
| an identity named (`X_TERMINAL_VIEWER_HANDLE`, then `X_TERMINAL_VIEWER_ID`) | `TestNoFixtureCarriesViewerIdentity` | both arms failed on the fixtures that carry it, rather than skipping |

## 5. Reported, not implemented

- **The capture harness was not run**: no live request was issued from this host, so its live path
  (session, limiter, classifier, write, row rewrite) is unexercised; its own redaction and identity
  helpers are covered by tests that need no socket.
- **The identity arm of `TestNoFixtureCarriesViewerIdentity` skips here**: the session on this host
  loads but names no account id or handle, and no `X_TERMINAL_VIEWER_*` is set. The always-on arms
  ran: every fixture is scanned for `auth_token`/`ct0` keys, the session's own two values are searched
  for in every fixture, and every row's `redaction` is asserted non-empty. The skip message names the
  session path and what would make the body scan run.
- **The plan's "41 reply groups"** is superseded by 36 (see §3a); if a later reader wants 41 groups,
  the promoted items would have to become groups, which would put five other conversations on screen.
- `provenance.json` records a query id, which is data rather than Go: the repo-wide XAPI-02 guard
  (`TestTheOperationIDsLiveOnlyInThisPackagesData`) scans non-test `.go` files and is unaffected; the
  plan's own acceptance asks for the id in this file.
