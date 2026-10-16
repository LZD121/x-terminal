---
phase: 261015-r3a
plan: 01
status: complete
type: execute
commits:
  - 7844942
  - 9c9915d
  - 04c3251
  - b143138
deviations: 2
---

# 261015-r3a — the x.com GraphQL upstream drift in the read path

## Objective, as executed

Repair the read path this build actually uses, after the upstream drifted under it: the wire
method became a per-operation property (`SearchTimeline` answers GET with a 404 and a zero-byte
body), and three operations began demanding variables this client was not sending. Both drifts are
invisible to an operation-id refresh.

## Task 1 — the wire method becomes a per-operation property (commit `7844942`)

**Where it lives and why.** One table, `postOperations` in `internal/xweb/graphql.go`, read by one
function `wireMethod` that returns `http.MethodGet` for every operation the table does not name.
Not the registry: the bundle `cmd/x-opsgen` mines carries `{queryId, operationName, operationType,
metadata}` and no method, `Registry.Merge` replaces a whole entry with what the bundle carries, and
`Write` marshals the `Operation` struct — so a method kept there would be dropped silently by the
next `x-terminal doctor --refresh-ids`. Not a pin either: a pin is an operator's own override and
the shipped build has to search without one.

**One place decides the parameters.** `graphQLParams` (built by `buildGraphQLParams`) encodes the
three parameters once; `query()` renders the GET query string, `body()` renders the POST JSON body.
The rule that decides whether `features`/`fieldToggles` are present at all — sent when the
operation declares flags, omitted when it declares none — is therefore decided in exactly one
place, and it is identical in both forms.

**The POST shape is the body form, and it is measured, not preferred.** The live proof is Task 2's
`SearchTimeline` probe row, which drives this exact request builder and answers `ok` over POST.

**The audit keeps its promise with the POST named.** `checkTransport` keeps the printed promise
`no mutation is expressible over the wire` and now says what carries it:

- only `GET` and `POST` may be spelled anywhere (a method literal or `http.Method*` constant that is
  neither is a finding: PUT, DELETE, PATCH, HEAD, OPTIONS, CONNECT, TRACE);
- `POST` exists in exactly one declared file, named in the check as `methodExemptionFile`
  (`internal/xweb/graphql.go`) and printed in the promise: `no mutation is expressible over the
  wire (POST exemption internal/xweb/graphql.go:213)`;
- a POST anywhere else fails, including in `internal/xweb/x.go`;
- anti-vacuity in the check's existing idiom: a tree whose exemption file is missing is a finding
  (`the declared POST exemption … is not in the tree that was read`), and it still fails when it
  found no `http.NewRequest*` call site at all.

The check compares methods as its own string constants (`methodGet`, `methodPost`) rather than as
`http.Method*` selectors, because `internal/audit/readonly.go` is itself scanned by this check and
a checker that reports itself is a checker nobody can read.

**Observed plant, then reverted.** A new `internal/cli/zzplant.go` holding
`http.NewRequest(http.MethodPost, "https://x.com/", nil)` made `x-terminal audit` print
`10 checks, 3 findings` and exit 1; the file was deleted and `10 checks, 0 findings` returned.
`git diff --stat internal/xweb/volatile/registry.json` is empty.

**Tests.** `TestSearchTimelineGoesOutAsAPostWithItsParametersInTheBody` pins the method, the absent
query string, and a JSON body whose `variables`/`features`/`fieldToggles` equal the registry's own
rendering for that operation (read from the registry at run time). `TestViewerStillGoesOutAsAGetWithItsParametersInTheQuery`
pins the contrast. `TestTransportFailsNonGET` splits into: POST inside the exemption passes,
POST outside fails (as method and as constant), `http.MethodDelete` fails anywhere, a `"PUT"`
literal fails, and a tree missing the exemption file fails.

## Task 2 — `doctor --probe` names the missing variable, and the measurement (commit `9c9915d`)

`Probe` gains `Detail string`, set from the same bounded reader `GraphQL` already uses
(`apiMessage`) and only when the outcome is not a success. `doctor --probe` prints it on the text
line after the outcome word, and as `detail` in `--json`; the `ok` line is unchanged byte for byte.
`probeVariables` became a table (`probeRequests`) of every operation this build reads, and the
usage error's list of known operations is derived from that table rather than typed out beside it.

### Deviation 1 (Rule 1, auto-fixed): the reader could not read the body the task is built on

The measured 422 body is

```
{"errors":[{"code":"GRAPHQL_VALIDATION_FAILED","extensions":{"code":"GRAPHQL_VALIDATION_FAILED"},
"message":"must be defined","path":["variable","includePromotedContent"]}]}
```

Two things in it defeated the existing reader, and both are silent rather than loud:

1. `errors[].code` arrives as a **string** here, where every previously recorded body carries a
   number. Decoding straight into an `int` made the whole `errors[]` array fail to decode, which is
   indistinguishable from an absent array: a body saying exactly what is wrong classified as though
   it said nothing.
2. The name of the missing variable is in the error's **`path`**, not in its `message` (`must be
   defined` names nothing).

Fix: `apiCode` keeps whichever spelling the upstream sent (the number or the name), `apiError`
keeps `path`, and `apiMessage` prints the path beside the message — so `--probe` prints
`must be defined at variable.includePromotedContent`. Pinned by
`TestParseErrorsReadsTheMeasuredValidationBody` and a new row in the classifier table.

### The measurement, run for real

One request at a time, four seconds apart, through the operator's own session and the proxy at
`http://127.0.0.1:7897`. `Bookmarks` is the unchanged control.

| operation | variables sent | observed |
|---|---|---|
| `HomeLatestTimeline` | `count,cursor,seenTweetIds` | `422`, `path:["variable","includePromotedContent"]` |
| `HomeLatestTimeline` | `…,includePromotedContent:false` | `200`, 983,277 bytes (and `true` answered `200` too) |
| `HomeTimeline` | `count,cursor,seenTweetIds` | `422`, `path:["variable","includePromotedContent"]` |
| `HomeTimeline` | `…,includePromotedContent:false` | `200` |
| `UserTweets` | `userId,count,cursor` | `422`, `path:["variable","includePromotedContent"]` |
| `UserTweets` | `…,includePromotedContent:false` | `422`, `path:["variable","withVoice"]` |
| `UserTweets` | `…,includePromotedContent:false,withVoice:false` | `200` |
| `TweetDetail` | `focalTweetId,cursor` | `422`, `path:["variable","includePromotedContent"]` |
| `TweetDetail` | `…,includePromotedContent:false` | `422`, `path:["variable","withBirdwatchNotes"]` |
| `TweetDetail` | `…,withBirdwatchNotes:false` | `422`, `path:["variable","withVoice"]` |
| `TweetDetail` | `…,withVoice:false` | `200` |
| `Bookmarks` | unchanged | `200`, unchanged — the control |
| `Likes`, `Viewer` | unchanged | `200`, unchanged |

Names added, in the order the upstream asked for them, with the value each got:

- `HomeLatestTimeline`, `HomeTimeline`: `includePromotedContent` = `false`
- `UserTweets`: `includePromotedContent` = `false`, then `withVoice` = `false`
- `TweetDetail`: `includePromotedContent` = `false`, then `withBirdwatchNotes` = `false`, then
  `withVoice` = `false`

`includePromotedContent` `true` was also measured answering `200` on the home timelines, so `false`
is this client's choice rather than a value the upstream demanded. No key was invented: every name
above is one the upstream's own 422 named, and every other key was already carried under a measured
200.

**The wire-method measurement, same day, same session.** With the same query id, variables,
features, field toggles and headers: `SearchTimeline` over `GET` → `404`, 0 bytes; over `POST` →
`200`, 123,860 bytes with real results.

**Final probe sweep, 4 s apart:** `SearchTimeline` `ok`, `HomeLatestTimeline` `ok`, `HomeTimeline`
`ok`, `UserTweets` `ok`, `TweetDetail` `ok`, `Bookmarks` `ok` (control), `Likes` `ok`, `Viewer`
`ok` — all exit 0.

**On the record.** Appended to `.planning/research/UPSTREAM-PROBE.md` under
`## Two request-shape drifts, measured 2026-10-15`: the per-operation table with observed statuses,
the two drifts, the exact command that reads a missing variable's name out of the upstream, the
4-second spacing and the transient `401 {"code":89}` it exists for, and the fact that
`--refresh-ids` reports `+0 added, ~0 updated, 104 unchanged` for a drift it cannot see. No cookie,
no post text, no id beyond the query ids the registry already carries in the tree.

### Deviation 1b: the fallback for want of a bundle

The plan's value rule prefers "the value the x.com bundle's own call site uses, where the bundle
has one". The mined bundle (`main.77cd09844e96c189a.js`) is not on disk in this tree and is not
reproducible here without a fresh authenticated bundle fetch, so the plan's stated fallback
governs: `false` for a promotion switch, with the comment saying the value is the measured-required
one and not a value the upstream was shown to demand.

## Task 3 — the measured variables at the read call sites (commit `04c3251`)

- `internal/xsource/home.go` (`Home`, both feeds): `includePromotedContent: false`, with the comment
  citing the 422, the date and why `false` rather than `true`.
- `internal/xsource/profile.go` (`UserPosts`): `includePromotedContent: false`, `withVoice: false`.
- `internal/xsource/detail.go` (`Thread`): `includePromotedContent: false`,
  `withBirdwatchNotes: false`, `withVoice: false`, and its doc comment now explains the half it was
  missing — the variables the upstream requires regardless of the call.
- `internal/xsource/search.go` (`Search`): no variable change (its four are measured); its doc
  comment gains the wire-method sentence.
- `internal/xsource/saved.go`, the parsers, `GraphQLer` and `domain.Source`: untouched.

**Tests pin the new keys and the exact variable count**
(`TestAdapterAsksForTheRegisteredOperations`, `TestTheAdapterAsksForOnePage`,
`TestAdapterProfileAndUserPostsAskByName`), so a later edit that drops a required variable fails a
test rather than a live read. The suite stays hermetic: fixtures and `stubClient` are the only
bodies in play.

**RUNBOOK.md.** `## Features blob drift` became `## Request shape drift` (anchor
`request-shape-drift`), its four parts rewritten in the order and vocabulary the runbook test
asserts, its Symptom carrying the classifier's own word `bad request`, and its Repair no longer
reaching for `--refresh-ids` first, because a refresh was measured reporting `+0 added, ~0 updated,
104 unchanged` for exactly this drift. Both anchor tables — the doctor's `runbookAnchors` and
`runbook_test.go`'s `stateToAnchor` — are re-pointed in the same change.

## Deviation 2 (out of scope, recorded rather than guessed): the feeds' response envelope

The request-shape repair works — the live pane now spends its two requests with `0 rate-limited` —
but bare `x-terminal` (the Following feed) still fails, now in the **reader**:

```
 unclassified
   xsource: missing data.user.result.timeline.timeline.instructions (and
   data.user.result.timeline_v2.timeline.instructions, and
   data.bookmark_timeline_v2.timeline.instructions)
```

Measured 2026-10-15, both home timelines answer `200` with their instructions at
`data.home.home_timeline_urt.instructions` — a third, response-side drift that `timelineFrom` in
`internal/xsource/tweets.go` does not accept (it reads `data.user.result.timeline[._v2]` and
`data.bookmark_timeline_v2`). The home entries themselves are the grammar the walker already reads
(`TimelineTimelineItem` / `TimelineTweet` / `TimelineTimelineCursor`, 112 entries: 92 `tweet-`,
15 `promoted-`, 3 `home-conversation-` modules, with `TweetWithVisibilityResults` wrappers).

This was outside Task 3's variable-and-method scope and was NOT guessed at there. It is **no longer
open**: Task 4 below closed it — the envelope is read, the nesting it uses is read, a recording of
the live answer is checked in so the arm can fail in a test instead of a pane, and the Following
feed renders in a real pane. One thing this deviation predicted did not hold: it expected 112
entries with 15 `promoted-` entries and "the promoted-result wrapper" to need reading. The
recording's own census is 111 entries — 94 `tweet-`, 12 `promoted-`, 3 `home-conversation-`, 2
`cursor-` — and its only wrapper is `TweetWithVisibilityResults` (6 of its 112 tweet items), which
`unwrapTweet` already read. No new tweet shape was needed; the promoted entries ARE there, and
Task 4 records what the walk does with them.

## The live pty proof

Built with `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o ~/.local/bin/x-terminal
./cmd/x-terminal` and captured through a real pty on a private tmux socket with the operator's own
tmux untouched (the `internal/render/capture_tmux_test.go` recipe).

**Search** — `/` + `sea otter` + Enter rendered real results:

```
>neulsi           Oct 15 20:06
>  can we spread the jay gardiner is a sea otter agenda
 ShadowOfPhaaze   Oct 15 18:35
   The sea otter
 > orpheuzir        Oct 15 00:54
 >   these two in a fight whos winning https://t.co/aQEVj5PiVm
```

**Thread** — Enter on the selected post opened the conversation:

```
>neulsi           Oct 15 20:06  focal
>  can we spread the jay gardiner is a sea otter agenda
     no replies
     this is an empty read that succeeded - press r to refresh
     partial thread - the reply walk stopped here
no older posts (cursor stalled)  thread  1/1  4 requests, 0 rate-limited
```

Status lines: `thread` with a non-zero loaded count (`1/1`), no failure sentence, `0 rate-limited`.

**Feed** — NOT captured in this round: the request shape was accepted but the reader refused the
answer's envelope (Deviation 2 above). Task 4 repaired the reader and captured it: bare
`x-terminal` rendered the Following timeline with real posts and the status line
`following  1/168+  1 request, 0 rate-limited`.

**Profile** — `a` on the selected post's author opened the profile: its facts header rendered
(`followers: 164  following: 162  posts: 1681  joined: Nov 2025`), the conversation stayed on the
pane, and the author's own timeline loaded below it with reposts attributed by name:

```
 neulsi           austin abrams shady facts
   every timeline bends toward byler
   followers: 164  following: 162  posts: 1681  joined: Nov 2025
>neulsi           Oct 15 20:06
>  can we spread the jay gardiner is a sea otter agenda
 halfofheartisin  Oct 14 22:49  reposted by neulsi
   이제 앓다 죽을 제이 https://t.co/qFcZbFBTuq
   [1] image, [2] image, [3] image, [4] image
 kittyynaim       Oct 15 08:27  reposted by neulsi
   Jay Gardiner get behind me #austinabrams #whalefall https://t.co/sG5AV7EIi8
   [1] video
 S0ULMATES_95     Oct 14 11:02  reposted by neulsi
```

Observed verbatim:

- **thread** — status line `no older posts (cursor stalled)  thread  1/1  4 requests, 0 rate-limited`
  (the renderer's own list name `thread`, a non-zero loaded count `1/1`, no failure sentence)
- **profile** — the facts header and the author's own timeline rendered below the conversation,
  with reposts attributed by name (`reposted by neulsi`); no failure sentence anywhere in the pane
- **search** — the results pane rendered six real posts from the live upstream, one carrying media
  refs (`[1] image, [2] image`) and one a gif
- **feed** — NOT captured in this round (the reader drift of Deviation 2); captured in Task 4, whose
  section quotes the Following timeline's own status line with real posts on the pane

The search and profile panes' own status lines were not among the captured rows (the capture was
taken from the top of each pane). Task 4's section quotes all four surfaces' status lines, each
read from the bottom row of its pane.

## Verification

At the three-commit round (Tasks 1–3):

- `CGO_ENABLED=0 go test ./... -count=1` — 1585 passed, 22 packages
- `CGO_ENABLED=0 go vet ./...` — clean
- `test -z "$(gofmt -l internal cmd)"` — prints nothing
- `go run ./cmd/x-terminal audit` — `10 checks, 0 findings, 906 files inspected`
- `git diff --stat go.mod go.sum` — empty (this plan adds no dependency)
- every operation this build reads answers `ok` to `doctor --probe` against the live upstream,
  with `Bookmarks` unchanged as the control

Re-run on the full four-commit work (Task 4 included):

- `CGO_ENABLED=0 go test ./... -count=1` — 1589 passed, 22 packages
- `CGO_ENABLED=0 go vet ./...` — clean
- `test -z "$(gofmt -l internal cmd)"` — prints nothing
- `go run ./cmd/x-terminal audit` — `10 checks, 0 findings, 906 files inspected`
- `git diff --stat go.mod go.sum` — empty
- `x-terminal` installed from this tree and driven in a real pane on all four surfaces

## What the next reader should know

1. The variable drift, the wire-method drift and the home timelines' response-envelope drift are
   all repaired and measured. The Following feed — the default surface — renders.
2. The home envelope's fixture is `internal/xsource/testdata/fixture-home.json`, recorded
   2026-10-15 by `TestCaptureHome`. A future drift in this envelope fails
   `TestParseTimelineReadsTheHomeFixture`, not a pane.
3. `doctor --probe` now prints the upstream's own sentence with the path it names, so the next
   drift of this class is one command away from being named instead of guessed at.
4. Open, measured and recorded in Task 4: the Following timeline shows the 12 `promoted-tweet-…`
   entries its answer carries as ordinary posts, because the shared entry walk has no promotion
   rule and the renderer marks no advertisement. `fixture-home.json` is what makes a change there
   testable when someone owns it.

## Task 4 — the home envelope is read, and a recording keeps it readable (commit `b143138`)

**Where the drift was.** `timelineFrom` in `internal/xsource/tweets.go` picked the instructions
container out of three envelopes (`data.user.result.timeline[._v2]`,
`data.bookmark_timeline_v2`), and the home timelines answer under a fourth,
`data.home.home_timeline_urt`. Measured 2026-10-15 on `HomeLatestTimeline`
(`OQPHTgwczzp9RMAPt6BH9A`) under this build's own variables: `200`, instructions at
`data.home.home_timeline_urt.instructions`, `TimelineAddEntries` with 111 entries — 106
`TimelineTimelineItem` (each `itemContent` a `TimelineTweet`: 100 plain `Tweet` results and 6
`TweetWithVisibilityResults` wrappers), 2 `TimelineTimelineCursor` (`Top` and `Bottom`) and 3
`TimelineTimelineModule` carrying 6 tweet items between them, so 112 tweet items in all — the
number `ParseTimeline` reports as `Requested`. `HomeTimeline` (For-you, `og4a4SdSF3WiQkkwaPCdPg`)
was measured answering under the same path, so one arm covers both.

**The second half of the arm, which the task brief did not name.** The home container puts its
instructions at a DIFFERENT nesting: `home_timeline_urt.instructions` is a direct sibling of
`home_timeline_urt.metadata`, with no `timeline` object in between. The first draft of the arm
reused `timelineNode` as-is and read `…timeline.instructions`, so it matched the container and then
found no entries: that draft's `ParseTimeline(fixture-home)` failed with `missing
data.home.home_timeline_urt.timeline.instructions: the instructions carried no entries`, which is
what a candidate path is worth without the nesting it arrives under. `timelineNode` gained the
unwrapped field and `instructionList` as the one place the nesting is unwrapped — the wrapped shape
wins when present, so the four containers that use it are untouched. Both nestings are pinned by a
test.

**The fixture, which is half the fix.** `internal/xsource/testdata/` had no home body at all —
`fixture-usertweets.json`, `fixture-tweetdetail.json` and the two saved-list recordings are user
timelines and saved lists — so the home envelope was the one arm no test could falsify. New
`TestCaptureHome` (gated on `X_TERMINAL_CAPTURE=1`, following the bookmarks/likes pattern) recorded
one live `HomeLatestTimeline` answer into `internal/xsource/testdata/fixture-home.json`:
**983,042 bytes**, 112 tweet items, both cursors, sha256
`f88f644aacae82c067f6aa61f71af970b2a074882491acadeba0568981cdf101`, its row in
`provenance.json`. 190 post bodies were replaced with synthetic text of the same lengths
(the operator's own Following line, so the checked-in body must not carry posts that account
happens to follow). The answer carried neither the viewer's account id nor their handle, and no
cookie or credential is in the fixture. The shared capture helper was renamed `captureSaved` →
`captureOne`: it is the capture for any list now, and the lists that first named it are not its
subject.

**A measured fact the fixture exposed: `includePromotedContent: false` does not keep advertisements
out.** The recorded Following body, requested with `false`, carries 12 `promoted-tweet-…` entries,
and the shared entry walk reads them as posts like any other (verified: their tweet ids are in
`ParseTimeline`'s output). `internal/xsource/home.go`'s comment said `false` is sent because this
renderer marks no advertisement — which is still true — but it sat next to an implication the
recording refutes, so the comment now states the measurement. The thread path already has a
promotion rule (`detail.go`'s `promotedItemMark` drops them inside a conversation module); a
timeline has no equivalent, and **nothing in this task changed that**: dropping them would change
the shared walk for the user timelines and search too, and the reader's own status line counts what
the answer cost. Left as an open observation for whoever owns the list grammar.

**Tests.** New `internal/xsource/home_test.go`: the fixture parses into real posts with both
cursors and a census of the body's tweet items matched against `Requested` (112/112); the
`missing(...)` error names the home path AND the path it names is one this parser reads; the
unwrapped and the wrapped nestings both parse; `Home` over the fixture returns the page the screens
read, with the bottom cursor carried into `Next`.

**Anti-vacuity, both mutations observed and reverted.** Removing the envelope arm fails all four
tests with `missing data.user.result.timeline.timeline.instructions (and …, and
data.home.home_timeline_urt.instructions)`. Removing only the nesting accessor (`instructionList` →
`Timeline.Instructions`) fails all four with `missing data.home.home_timeline_urt.instructions: the
instructions carried no entries`. `tweets.go` was byte-identical to its pre-mutation content
afterwards (`diff` empty).

**The live proof, all four surfaces on the installed binary.** Built with the plan's own command
and captured through a real pty on a private tmux socket (`xt-final-$$`), the operator's own tmux
untouched. Status lines verbatim:

```
following  1/168+  1 request, 0 rate-limited
search: latest "sea otter"  1/20+  2 requests, 0 rate-limited
no older posts (cursor stalled)  thread  1/1  3 requests, 0 rate-limited
profile: neulsi  1/20+  5 requests, 0 rate-limited
```

Bare `x-terminal` renders the Following timeline with real posts (e.g.
`>PaulSolt Oct 15 20:32 — iPhone Duo is launching tomorrow…` with its quoted post, and
`TV_N_korea … reposted by X_Global_tv` attributed by name). `/` + `sea otter` + Enter renders real
search results. Enter on the selected post opens the thread. `a` opens the author's profile with
its facts header (`followers: 164  following: 162  posts: 1682  joined: Nov 2025`) and the author's
own timeline below it. No surface shows a failure sentence, and every status line carries
`0 rate-limited`.

**The deviation this task corrects.** Task 3's record above called the feeds' envelope "outside
Task 3's variable-and-method scope" and "the remaining blocker". Both were true of Task 3; neither
is true of the build now: the Following feed is the default surface and it renders. The two
statements are kept there as the trail, with the correction pointing here.

**Gates after this change.** `CGO_ENABLED=0 go test ./... -count=1` — 1589 passed, 22 packages;
`CGO_ENABLED=0 go vet ./...` — clean; `test -z "$(gofmt -l internal cmd)"` — prints nothing;
`go run ./cmd/x-terminal audit` — `10 checks, 0 findings, 906 files inspected`;
`git diff --stat go.mod go.sum` — empty. No test makes a live request: the capture test skips
without the gate.
