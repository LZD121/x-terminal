---
phase: 02-following-timeline-in-the-pane
plan: 01
subsystem: domain
tags: [domain-model, port, adapter, fake-source, reducers, keyset-pagination, dedupe, stall-detection, soft-ban]
requires:
  - phase: 01-reachable-transport-and-one-live-read
    provides: xweb.Client.GraphQL + volatile registry, xweb.Outcome/APIError + Classify, netcheck.Result.Detail, the recorded UserTweets fixture and the xsource parsers
provides:
  - domain.Source (Home/User) plus PostPage/Cursor/HomeKind, the extended content model and the closed FailureKind taxonomy
  - xsource.Adapter implementing the port over an injected GraphQLer, with HomeLatestTimeline/HomeTimeline/UserByScreenName and the measured variables
  - the parser reading quotes (bounded), media refs, visibility, conversation/reply/lang and counts, and counting the tweet entries per answer
  - fake.Source: the recorded fixture as pages (proven equal to the fixture's parse) plus the hand-authored adversarial set, with counters and failure/stall injection
  - screen.List: page cache, id-keyed dedupe, stall detection, refresh diff, single-flight latch, sticky failure and the stop rule, all as pure reducers
affects: [02-02, 02-03, 02-04, 02-05, phase 3]
actuals:
  tokens: 43000
  tasks: 3
  commits: 0
tech-stack:
  added: []
  patterns:
    - "The port is declared by its consumer in a leaf package (domain.Source) and satisfied structurally by the adapter and by a test double"
    - "The shell never calls a source: an update returns an Effect value and the composition root turns it into a request"
    - "Honesty rules as reducer properties (no duplicate / no loop / no silent loss / no retry after a stop), each proven by a test that fails when the behaviour is removed"
    - "A synthetic test double embedded from a checked-in document, with a parity test that fails when the document drifts from the recorded response"
key-files:
  created: [internal/domain/page.go, internal/domain/source.go, internal/domain/failure.go, internal/domain/domain_test.go, internal/xsource/home.go, internal/fake/fake.go, internal/fake/parity_test.go, internal/fake/testdata/pages.json, internal/screen/list.go, internal/screen/effect.go, internal/screen/absorb.go, internal/screen/list_test.go]
  modified: [internal/domain/domain.go, internal/xsource/tweets.go, internal/xsource/xsource_test.go]
key-decisions:
  - "EffectNone is the zero EffectKind, so 'no effect' is expressible: the plan named only EffectPage and EffectRefresh"
  - "List.Update(ev Event, now time.Time) (List, Effect) — a value receiver returning the next state and one effect; now is an argument only because the rate-limit row has an end"
  - "A page arriving into an empty list selects its first post, because a list with posts and no selection has no row for the selection column and nowhere for the movement keys to start"
  - "PageFetched and Refreshed each release the one latch that belongs to them (inflight = refreshPending / = loadingOlder), so a request that is still out is never reported as finished"
  - "Counts: an absent counter is zero (nothing in this phase prints one); views.count is read as raw JSON because the measured responses quote it"
  - "A TweetTombstone is a post with domain.Unavailable, identified by rest_id and otherwise by the entry id; an unknown media type is a typed ParseError, not a skip"
patterns-established:
  - "Effect-as-value: internal/screen returns Effect values and never performs a read, so every rule is testable with no socket, no clock and no terminal"
  - "Fake parity: internal/fake's pages are written from xsource.ParseTimeline over the recorded fixture and compared field by field against a fresh parse"
requirements-completed: [FEED-02, FEED-03, FEED-05, OPS-04]
coverage:
  - id: D1
    description: "The Source port, the content model the Content/Presentation Split owns, and the closed failure taxonomy"
    verification:
      - kind: unit
        ref: "internal/domain/domain_test.go#TestFailureKindsAreTheClosedTable"
        status: pass
      - kind: unit
        ref: "internal/domain/domain_test.go#TestPostCarriesTheContentTheSplitOwns"
        status: pass
      - kind: integration
        ref: "go list -deps ./internal/domain | grep -c '^github.com/LZD121' | grep -qx 1"
        status: pass
    human_judgment: false
  - id: D2
    description: "xsource.Adapter implements domain.Source over GraphQLer: HomeLatestTimeline/HomeTimeline/UserByScreenName, count 20, cursor, empty seenTweetIds, failures passed through"
    verification:
      - kind: unit
        ref: "internal/xsource/xsource_test.go#TestAdapterAsksForTheRegisteredOperations"
        status: pass
      - kind: unit
        ref: "internal/xsource/xsource_test.go#TestAdapterPassesAFailedAnswerThrough"
        status: pass
      - kind: unit
        ref: "internal/xsource/xsource_test.go#TestAdapterUserAsksByScreenName"
        status: pass
    human_judgment: false
  - id: D3
    description: "The parser reads nested quotes (bounded at 4), media refs, visibility (limited/tombstone), conversation/reply/lang and counts, and counts the tweet entries of an answer"
    verification:
      - kind: unit
        ref: "internal/xsource/xsource_test.go#TestParseTimelineReadsTheRecordedResponse"
        status: pass
      - kind: unit
        ref: "internal/xsource/xsource_test.go#TestParseTimelineReadsATombstoneAsAPost"
        status: pass
      - kind: unit
        ref: "internal/xsource/xsource_test.go#TestParseTimelineMarksALimitedPost"
        status: pass
      - kind: unit
        ref: "internal/xsource/xsource_test.go#TestParseTimelineBoundsTheQuoteChain"
        status: pass
      - kind: unit
        ref: "internal/xsource/xsource_test.go#TestParseTimelineReadsMediaRefs"
        status: pass
    human_judgment: false
  - id: D4
    description: "fake.Source serves the recorded fixture as three pages and the hand-authored adversarial set, and is proven equal to the fixture's parse"
    verification:
      - kind: unit
        ref: "internal/fake/parity_test.go#TestFakePagesMatchTheFixture"
        status: pass
      - kind: unit
        ref: "internal/fake/parity_test.go#TestAdversarialCarriesEveryRequiredCase"
        status: pass
    human_judgment: false
  - id: D5
    description: "FEED-02: pagination absorbs each post once and keeps the answers' order, including a page that repeats ids"
    requirement: FEED-02
    verification:
      - kind: unit
        ref: "internal/screen/list_test.go#TestPaginationAbsorbsEachPostOnce"
        status: pass
      - kind: unit
        ref: "internal/screen/list_test.go#TestAPageRequestWhileOneIsInFlightIsANoOp"
        status: pass
    human_judgment: false
  - id: D6
    description: "FEED-03: a repeated cursor or a page that adds nothing unseen stalls the walk, and no further page is ever requested"
    requirement: FEED-03
    verification:
      - kind: unit
        ref: "internal/screen/list_test.go#TestARepeatedCursorStallsTheWalk"
        status: pass
      - kind: unit
        ref: "internal/screen/list_test.go#TestAPageThatAddsNothingUnseenStallsTheWalk"
        status: pass
    human_judgment: false
  - id: D7
    description: "FEED-05: a failed page keeps every post, every page and the selection, and records one failure for the status line"
    requirement: FEED-05
    verification:
      - kind: unit
        ref: "internal/screen/list_test.go#TestAFailedPageKeepsEveryPostAndTheSelection"
        status: pass
    human_judgment: false
  - id: D8
    description: "OPS-04: after a soft ban no page request and no refresh is issued again for the life of the process, and the list carries the stopped state"
    requirement: OPS-04
    verification:
      - kind: unit
        ref: "internal/screen/list_test.go#TestASoftBanStopsTheSessionForTheProcess"
        status: pass
      - kind: unit
        ref: "internal/screen/list_test.go#TestARateLimitedFailureBlocksUntilItsReset"
        status: pass
    human_judgment: false
  - id: D9
    description: "A refresh prepends only unseen posts, reports the count as the notice, and leaves the selection and the walk's cursor where they were"
    verification:
      - kind: unit
        ref: "internal/screen/list_test.go#TestRefreshPrependsUnseenPostsAndReportsTheCount"
        status: pass
      - kind: unit
        ref: "internal/screen/list_test.go#TestRefreshWithNothingNewReportsZero"
        status: pass
    human_judgment: false
duration: 12min
completed: 2026-10-14
status: complete
---

# Phase 2: Following Timeline in the Pane Summary

**The read seam landed: `domain.Source` with the content model and the closed failure taxonomy, `xsource.Adapter` over `xweb.Client.GraphQL`, a fixture-backed `fake.Source` proven equal to the recorded response, and `screen.List`'s pure reducers — pagination that cannot duplicate a post or loop on a repeated cursor, a refresh diff that reports "N new posts" without moving the selection, and a soft ban that ends the session for the process.**

## Performance

- **Duration:** 12 min (first file write 14:53:29, last 15:04:31 +0800)
- **Started:** 2026-10-14T14:53:29+08:00
- **Completed:** 2026-10-14T15:04:31+08:00
- **Tasks:** 3
- **Files modified:** 15 (12 created, 3 modified)

## Accomplishments

- **`domain` is still a leaf and now carries the phase's content model**: `PostID`, `Cursor`, `HomeKind` (`following`/`for you`),
  `Visibility` (`Visible`/`Limited`/`Unavailable`/`Withheld`), `MediaKind` (`Image`/`Video`/`GIF`), `MediaRef`, `Counts`,
  `PostPage{Posts, Next, Requested, FetchedAt}`, `FailureKind` (19 states + `FailureNone`) and `Failure{Kind, Detail, Reset, Retry, Stopped}`.
  `go list -deps ./internal/domain` names the standard library and itself and nothing else.
- **`xsource.Adapter` is the only production implementation**, built from one injected `GraphQLer` (satisfied by `*xweb.Client` as it
  stands, verified against its exported signature), and the parser now reads what a screen needs: a nested quoted post (bounded at four
  levels), media refs from `extended_entities`/`entities`, the visibility wrapper and the tombstone, conversation/reply/lang and counts,
  plus the number of tweet entries an answer carried.
- **`fake.Source` drives every later screen offline**: three pages cut from the recorded fixture (8/8/4) with synthetic cursors, a
  hand-authored adversarial set (wide/narrow handles, CJK, emoji, ZWJ family, over-measure URL, exact/one-short measures, both quote
  depths, a continuation pair, a repost, a limited post, a tombstone, singular and plural attachments), counters, and failure/stall
  injection. `parity_test.go` proves the pages are the fixture's parse, and fails when the document drifts.
- **`screen.List` is the state machine the honesty rules live in**: dedupe keyed on the post id, stall detection on a repeated cursor or
  an unproductive page, single-flight latch, sticky failure, refresh diff, and the stop rule as a flag rather than a countdown.

## Task Commits

None made by this agent: the orchestrator commits. Each task's `<verify>` passed before the next began (see below).

## Files Created/Modified

- `internal/domain/domain.go` - `Post` extended (`ID PostID`, `Lang`, `ConversationID`, `ReplyTo`, `Quote *Post`, `Media`, `Visibility`, `Counts`; `QuotedID` gone) and the content value types with their `String()` methods
- `internal/domain/page.go` - `PostID`, `Cursor`, `HomeKind` + `String()`, `PostPage`
- `internal/domain/source.go` - the `Source` port: exactly `Home(ctx, kind, c)` and `User(ctx, handle)`
- `internal/domain/failure.go` - `FailureKind` (19 states + zero) and `Failure{Kind, Detail, Reset, Retry, Stopped}`
- `internal/domain/domain_test.go` - totality test over the kinds, zero-value tests, the content round trip
- `internal/xsource/tweets.go` - `Page.Requested`; `postFrom`/`postBody`, `tombstone`, `mediaOf`/`mediaKind`, `countsOf`, `unwrapTweet(path)` returning the visibility
- `internal/xsource/home.go` - `GraphQLer`, `Adapter`, `New`, `Home`, `User`, the compile-time `var _ domain.Source = (*Adapter)(nil)`
- `internal/xsource/xsource_test.go` - fixture assertions for the new fields, synthetic tombstone/limited/quote-bound/media tests, the adapter tests with a stub client
- `internal/fake/fake.go` - `Source` with `Recorded`, `Adversarial`, `Home`, `User`, `Calls`, `LastKind`, `FailNext`, `StallWith`, the embedded page document and the adversarial literals
- `internal/fake/parity_test.go` - the `-update` writer and the field-by-field parity proof
- `internal/fake/testdata/pages.json` - the recorded fixture's 20 posts as three pages (8/8/4), written by `-update`
- `internal/screen/list.go` - `List` and its accessors and lookups
- `internal/screen/effect.go` - `EffectKind` (`EffectNone`/`EffectPage`/`EffectRefresh`) and `Effect`
- `internal/screen/absorb.go` - the sealed `Event` set and `Update` with its rules
- `internal/screen/list_test.go` - 11 tests, driven by `internal/fake` and its call counter

## The `domain` shape as landed

```go
type PostID string
type Cursor string

type HomeKind uint8                  // Following = 0, ForYou; String(): "following" / "for you"
type Visibility uint8                // Visible = 0, Limited, Unavailable, Withheld
type MediaKind uint8                 // Image = 0, Video, GIF
type MediaRef struct{ Kind MediaKind; URL, AltText string }
type Counts struct{ Replies, Reposts, Likes, Quotes, Bookmarks, Views int }

type User struct{ ID, ScreenName, Name, AvatarURL, BannerURL string; HasGraduatedAccess bool } // unchanged from Phase 1

type Post struct {
	ID             PostID
	Author         User
	Text           string
	CreatedAt      time.Time
	RepostedBy     *User
	Lang           string
	ConversationID string
	ReplyTo        PostID
	Quote          *Post          // the quoted post's id is Quote.ID; QuotedID is gone
	Media          []MediaRef     // nil when there are none
	Visibility     Visibility
	Counts         Counts
}

type PostPage struct {
	Posts     []Post
	Next      Cursor      // "" means end-of-list
	Requested int         // the entries the answer carried, not what survived de-duplication
	FetchedAt time.Time
}

type Source interface {
	Home(ctx context.Context, kind HomeKind, c Cursor) (PostPage, error)
	User(ctx context.Context, handle string) (User, error)
}

type FailureKind uint8 // FailureNone = 0, then: SessionExpired, SignerOrAuth, OperationStale,
                       // EndpointRetired, RateLimited, SoftBan, LoadShed, BadRequest, EmptyBody,
                       // Challenge, Unauthorized, Unclassified, BudgetSpent, NoRoute, ProxyRefused,
                       // UpstreamBlocked, TLSChallenge, TransportUnusable, NoSession

type Failure struct {
	Kind    FailureKind
	Detail  string    // the caller copies it from xweb.Outcome.Remedy() or netcheck.Result.Detail
	Reset   time.Time // x-rate-limit-reset; zero when the header was absent
	Retry   bool      // false for the rows whose remedy is "do not retry"
	Stopped bool      // true only for the soft ban
}
```

No `Terminal()` helper was added: nothing in this phase calls one. `FailureKind`, `Visibility` and `MediaKind` have no user-facing
`String()` except the latter two (logs/tests, and an unmapped value names itself rather than passing as a mapped one).

## What `xsource.Adapter` sends

| Method | Operation | Variables |
|---|---|---|
| `Home(…, domain.Following, c)` | `HomeLatestTimeline` | `count: 20`, `cursor: string(c)` (empty on page one), `seenTweetIds: []string{}` |
| `Home(…, domain.ForYou, c)` | `HomeTimeline` | same three |
| `User(…, handle)` | `UserByScreenName` | `screen_name: handle` |

No query id is written anywhere in these four packages (`grep -rn queryId` is empty); the id, the feature switches and the field
toggles come from `internal/xweb/volatile`. A non-success answer is passed through as the transport's own typed error.
`Page.Next` is the answer's bottom cursor, `Page.Requested` is the count of tweet entries walked, and `FetchedAt` comes from the
package-level `var now = time.Now` (frozen in tests).

## The fake's page splits and counters

- `Recorded()`: `testdata/pages.json` (written by `go test ./internal/fake/... -update`, which re-parses
  `../xsource/testdata/fixture-usertweets.json` through `xsource.ParseTimeline`) embedded at build time. Split `splitRecorded`: chunks of
  `recordedPageSize = 8`, so the recorded 20 posts are pages of **8 / 8 / 4**. Cursors `recorded-1`, `recorded-2`; the last page's `Next`
  is `""`. `Requested` is set to the page's post count and `FetchedAt` to `frozenAt` (2026-10-14T09:12:00Z) when a page is served.
- `Adversarial()`: 3 pages of **9 / 5 / 2** posts (`adversarial-1`, `adversarial-2`, then end-of-list), the cases listed above.
- Counters and knobs: `Calls() int` (every `Home` and `User` call), `LastKind() (domain.HomeKind, bool)` (which feed the last `Home`
  asked for — needed by plan 04's feed test), `FailNext(err)` (the next call only), `StallWith(cursor)` (every page handed back
  carries that cursor as its `Next`). A cursor that names no page is an error, never an empty page.
- The fake has no I/O: the document is `//go:embed`ed, there is no clock, and its imports are the standard library plus `internal/domain`.

## The stall and stop rules, and the tests that pin them

| Rule | Where | Test (fails when the rule is removed — verified by hand) |
|---|---|---|
| A page never adds a post the list already holds (dedupe on `domain.PostID`) | `absorb.go` `mark`/`absorb` | `TestPaginationAbsorbsEachPostOnce` — removing the `seen` check: `the overlapping page added 5 posts, want the 2 …` and `the post "2097843483495035131" appears at index 13 and again at 16` |
| A page whose `Next` is the cursor it was asked with, or one already used, stalls | `absorb.go` `pageFetched` | `TestARepeatedCursorStallsTheWalk` — removing the stall block: `a page that hands back the cursor it was asked with did not stall the walk …` |
| A page that adds zero unseen posts stalls | `absorb.go` `pageFetched` | `TestAPageThatAddsNothingUnseenStallsTheWalk` — same removal: `a page that added no unseen post did not stall the walk` |
| A stalled walk never asks again (`HasMore()` is false) | `list.go` `HasMore`, `absorb.go` `pageRequested` | the same test loop over five further `PageRequested` events with the fake's call count flat |
| One request at a time: a press while a page is in flight has no effect | `absorb.go` `pageRequested` | `TestAPageRequestWhileOneIsInFlightIsANoOp` — removing the `inflight` case: `a press while a page was in flight issued a request for 8 posts` |
| A failure keeps everything and records one failure | `absorb.go` `pageFailed` | `TestAFailedPageKeepsEveryPostAndTheSelection` |
| A not-retryable failure blocks; the rate-limit row unblocks at `Reset`; no reset header means no retry | `absorb.go` `blocked` | `TestARateLimitedFailureBlocksUntilItsReset` — removing the `blocked` case: `screen.PageRequested issued a request for 8 posts before the reset` |
| A soft ban stops the session: no page request and no refresh, at any later time | `absorb.go` `pageRequested`/`refreshRequested`/`blocked` | `TestASoftBanStopsTheSessionForTheProcess` — the call count is flat over nine further events across three instants (an hour and a year later) |
| A refresh prepends only unseen posts, reports the count and moves nothing | `absorb.go` `refreshed` | `TestRefreshPrependsUnseenPostsAndReportsTheCount` (making `Refreshed` re-select: `the refresh moved the selection to "refresh-1", want "2108299253655257139"`), `TestRefreshWithNothingNewReportsZero` (prepending without the diff: `the notice would read 8 new posts, want 0`) |

## Decisions Made

- **`EffectNone` is the zero `EffectKind`.** The plan named `EffectPage` and `EffectRefresh`; without a third value, "no effect" and
  "fetch a page" would be the same value. `Update` returns `Effect{Kind: EffectNone}` for every no-op.
- **`List.Update(ev Event, now time.Time) (List, Effect)`** with a value receiver, and a sealed `Event` interface with exactly
  `PageRequested`, `PageFetched{Page}`, `PageFailed{Failure}`, `RefreshRequested`, `Refreshed{Page}`, `Select{ID}`. Plan 04 writes
  `screen.List.Update(PageRequested)`; the actual signature is this one — see *Next Phase Readiness*.
- **The first page selects its first post** (`PageFetched` only, and only when nothing is selected): a list holding posts with no
  selection has no row for the selection column and nowhere for `j` to move from.
- **Each answer releases its own latch** (`inflight = refreshPending` in `PageFetched`, `inflight = loadingOlder` in `Refreshed`), so
  `refreshPending` is load-bearing and a request that is still out is never reported as finished.
- **`Counts`: an absent counter is zero.** Counts are the one place a missing field is genuinely zero (nothing prints them this phase);
  everything identity-, body-, date- or media-shaped keeps the package's typed-error rule. A present but non-numeric `views.count` is an
  error, and `views.count` is read as raw JSON because every measured response quotes it.
- **A tombstone is a post, not an error** (`Visibility: Unavailable`), identified by `rest_id` and otherwise by the entry id, because
  everything above the port de-duplicates on the id and a dropped tombstone shortens the timeline silently.
- **An unknown media type or a media entry with no URL is a typed `*ParseError`**, not a skip: a skipped attachment is an attachment
  line that lies about how many there are.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `EffectNone` had to exist for "no effect" to be expressible**
- **Found during:** Task 3 (design of `effect.go`)
- **Issue:** the plan names two `EffectKind` values; three of the required tests ("returns no effect") and plan 04's "further presses
  return no effect" need the zero value to be distinguishable from a request.
- **Fix:** `EffectNone EffectKind = iota` before `EffectPage`/`EffectRefresh`; every no-op returns `Effect{Kind: EffectNone}`.
- **Files modified:** internal/screen/effect.go
- **Verification:** `TestAPageRequestWhileOneIsInFlightIsANoOp`, `TestASoftBanStopsTheSessionForTheProcess`.

**2. [Rule 2 - Missing Critical] `List.requested` field (the cursor the in-flight page asked for)**
- **Found during:** Task 3 (`pageFetched`)
- **Issue:** the plan's rule says a page "marks the cursor used" and stalls when its `Next` "equals the cursor just used", but the listed
  fields carry no record of what the in-flight request asked for.
- **Fix:** one unexported field `requested domain.Cursor`, set in `pageRequested`, consumed and cleared in `pageFetched`.
- **Files modified:** internal/screen/list.go, internal/screen/absorb.go
- **Verification:** `TestARepeatedCursorStallsTheWalk`.

**3. [Rule 2 - Missing Critical] The fake gained `LastKind()`**
- **Found during:** Task 2
- **Issue:** plan 04's `feed_test.go` asserts "the first request the fake sees is for `domain.Following`", and the plan's listed knobs
  (`Calls`, `FailNext`, `StallWith`) cannot answer which feed was asked for.
- **Fix:** `LastKind() (domain.HomeKind, bool)`, recorded on every `Home`.
- **Files modified:** internal/fake/fake.go
- **Verification:** `internal/fake/parity_test.go#TestTheFakeCountsAndRefusesWhatItCannotServe` (which also pins `Calls()` and `FailNext`'s one-shot behaviour).

**4. [Rule 1 - Bug] The plan's illustrative cursor names did not exist**
- **Found during:** Task 2
- **Issue:** naming cursors after the page they fetch (`recorded-2` first) meant the plan's own example `recorded-1` was never produced.
- **Fix:** a cursor names the position it stands for — the first page's `Next` is `recorded-1`, the second's `recorded-2`. No behaviour
  changed; the walk and the parity test are unaffected (both read cursors only by handing them back).
- **Files modified:** internal/fake/fake.go
- **Verification:** `TestFakePagesMatchTheFixture`, `TestWalkingTheAdversarialSetReachesItsEnd`.

**5. [Rule 1 - Bug] The test helper `tweetResult` splices at the tweet's root**
- **Found during:** Task 2 (`TestParseTimelineReadsMediaRefs` failed with 0 media refs)
- **Issue:** media, conversation ids and counters live under `legacy`; the existing helper's `extra` lands at the tweet's root, so a
  test written the obvious way would have asserted against a shape the upstream does not send.
- **Fix:** added `tweetResultWithLegacy` and used it for the media cases, leaving the existing helper and every existing assertion alone.
- **Files modified:** internal/xsource/xsource_test.go
- **Verification:** `TestParseTimelineReadsMediaRefs` passes, and the pre-existing repost/note-tweet tests still pass.

**Total deviations:** 5 auto-fixed (2 missing-critical, 2 bugs, 1 blocking).
**Impact on plan:** no scope growth: each fix is either a value the plan's own tests need to be expressible or a fidelity fix. The one
visible interface difference from the plan's prose is `Update`'s signature, recorded for plans 03–05 below.

## Issues Encountered

- `pages.json` embeds at build time, which means the package cannot compile before the file exists. The first `-update` run needed a
  placeholder document; the checked-in file is the generated one (22,922 bytes, 8/8/4).
- The fixture's `views.count` is a **quoted decimal string** (`"1075958"`) and `legacy` carries media under both `entities` and
  `extended_entities` (2 of 20 posts); both were handled and pinned by tests rather than assumed.
- A repair to `internal/fake/fake.go` dropped `Home`'s mutex lock and its `s.calls++`. The new
  `TestTheFakeCountsAndRefusesWhatItCannotServe` caught it (`reports 1 calls, want 3`) — until then every call-count assertion in
  `internal/screen` had been passing *vacuously*, because a counter that never moves also "stays flat". Restored, and every
  call-count arm re-run: the single-flight arm now fails on the count as well as on the effect, and the soft-ban test was shown to
  fail when all failure gating is removed at once.

## User Setup Required

None - no external service configuration required, no new dependency.

## Verification as run

Restricted to the packages this plan owns, as the shared context requires (`go build ./...`/`go test ./...` are the orchestrator's):

| Check | Result |
|---|---|
| `go build ./internal/{domain,xsource,fake,screen}/...` | exit 0 |
| `go vet ./internal/{domain,xsource,fake,screen}/...` | exit 0 |
| `go test ./internal/{domain,xsource,fake,screen}/... -count=1` | 54 tests in 4 packages, all pass (domain 5, xsource 35, fake 3, screen 11) |
| `gofmt -l internal/{domain,xsource,fake,screen}` | empty |
| `grep -rn "queryId" internal/domain internal/xsource internal/screen internal/fake` | no matches |
| `unshare -rn` + `ip link set lo up`, then the four packages' tests | all pass (no test touches a socket) |
| `go list -deps ./internal/domain \| grep -c '^github.com/LZD121'` | `1` (a leaf: no module package but itself) |
| no width measurement in `internal/domain`, `internal/screen` | no `unicode/utf8`, no `fmt` width verb, no `len()` on a string |
| anti-vacuity arms | dedupe, stall (both), refresh diff, refresh selection, single-flight, failure gating and the parity file each broken by hand, each test failed, each reverted (exact failures in the table above) |

## Next Phase Readiness

**What plans 02-05 need to know about this port.**
- **`screen.List` never calls a `domain.Source`.** Nothing in `internal/screen` imports one, holds one or takes a `context.Context`;
  `Update` returns an `Effect` and the shell performs it. `internal/screen`'s non-test files import only `internal/domain`, `slices` and `time`.
- **The signatures plans 03-05 will actually call:**
  - `func (l List) Update(ev Event, now time.Time) (List, Effect)` — value receiver, returns the next state *and* one effect.
  - Events: `PageRequested{}`, `PageFetched{Page}`, `PageFailed{Failure}`, `RefreshRequested{}`, `Refreshed{Page}`, `Select{ID}`.
  - `Effect{Kind, Feed, Cursor, Head}` with `EffectNone`/`EffectPage`/`EffectRefresh`; `Effect.Kind == EffectNone` is the "no request" test.
  - `List` fields: `Kind` exported, everything else unexported; accessors `Posts`, `Selected`, `Next`, `Inflight`, `Failure`, `NewPosts`,
    `LoadingOlder`, `Stalled`, `Loaded`, `HasMore`, `Index`, `FirstID`, `LastID`, `NextID`, `PrevID`, `Has`, `AtEnd`.
  - `Posts()` returns the list's own slice (no copy, it is read every frame); callers must not modify it.
- **Page one is the shell's own fetch.** `PageRequested` with an empty cursor is a no-op by design, so `tui.Model.Init()` must call the
  source with `cursor == ""` itself and feed the answer back as `PageFetched`; every later page goes through `PageRequested`.
- **The list sets the selection exactly once on its own**: the first post of the first non-empty page, when nothing is selected.
  A refresh never moves it, and `Select` ignores ids the list does not hold.
- **The status line's fields it can fill:** `position` from `Index(Selected())+1` and `Loaded()` with `+` while `HasMore()`, `notice` from
  `LoadingOlder()` / `Stalled()` / `NewPosts()`, `list` from `Kind`, `state`/`remedy` from `Failure()` (the words are `internal/render`'s).
- **`domain.FailureKind` has 20 constants, of which `FailureNone` is the zero value and not a row of either UI-SPEC table** — plan 03's
  totality test over the copy tables must skip it, and plan 04's `failureFor` must set `Retry` (only the soft-ban row sets `Stopped`).
- **`Failure.Retry` is what gates the reducers** (`blocked`): false blocks until something else resolves the state, except for
  `FailureRateLimited`, which unblocks at `Reset` and stays blocked when `Reset` is zero.

No blockers. The fake is ready to drive every frame in plans 03-05, and the recorded pages are proven equal to the fixture they came from.
