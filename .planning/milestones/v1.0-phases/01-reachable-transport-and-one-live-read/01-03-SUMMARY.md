---
phase: 01-reachable-transport-and-one-live-read
plan: 03
subsystem: upstream-api
tags: [go, graphql, query-id-registry, go-embed, operation-pins, response-parsers, fixtures, whoami, first-run-notice, live-verification]

# Dependency graph
requires:
  - phase: 01-reachable-transport-and-one-live-read
    provides: "internal/xweb (NewClient/transport/Limiter/Classify/Outcome), internal/config (proxy precedence, XDG paths, RequestFloor/SessionRequestCap/SessionPath), internal/session (Secret/Session/Load/Summary), internal/netcheck (the four-stage self-check), internal/cli (command table, exit codes, doctor)"
provides:
  - "internal/xweb/volatile: the operation registry as data — registry.json (104 operations with queryId, operationType, featureSwitches, fieldToggles and provenance), embedded with //go:embed; Lookup/Load with pin precedence; FeaturesJSON/TogglesJSON; Merge that can only add or update; Write; Pin"
  - "internal/xweb: WebBearer, Signer/NoopSigner, APIError, Client.GraphQL — the measured GET request (queryId from the registry, variables/features/fieldToggles URL-encoded, the six measured headers) through the limiter and the classifier; Client.WithSession/WithSigner/WithRegistry"
  - "internal/domain: User and Post — the leaf vocabulary the parsers return"
  - "internal/xsource: ParseUser (the new core/avatar/banner schema, never legacy) and ParseTimeline (entries, modules, cursors, repost collapse, note_tweet text), with typed ParseError naming the missing path"
  - "internal/opsgen: bundle record parsing, main-bundle discovery, the live refresh through the configured transport, and RegistryPath — shared by cmd/x-opsgen and `doctor --refresh-ids`"
  - "cmd/spike: the throwaway end-to-end prover (Viewer + UserTweets + budget line, -record for bodies), proven live"
  - "internal/cli: a real whoami, the once-only account-risk statement, and doctor's --refresh-ids / --pin / --show-notice wired"
affects: [phase-2 (timeline screens parse the shapes this plan recorded), phase-4 (registry refresh), phase-6]

# Actuals — pairs with the plan's estimate to calibrate future estimates.
# chars/4 over the 856,378 bytes written: 20 new files, 3 modified (cli.go, doctor.go,
# transport.go, config.go, session_test.go — five, of which three were named in the plan).
actuals:
  tokens: 214094
  tasks: 3
  commits: 0

# Tech tracking
tech-stack:
  added: []          # stdlib only, as the phase requires
  patterns:
    - "operation ids as data with provenance, resolved through one accessor that has no stale-id fallback"
    - "a registry merge (and a refresh) that can only add or update, and that leaves a user's pin alone"
    - "parsers that fail loudly on a path that moved instead of returning a zero value"
    - "a recorded live response as the parser's fixture, plus a guard test proving ids are not literals"
    - "one seam per verb (a package-level constructor variable) so a real verb runs against httptest"

key-files:
  created:
    - internal/xweb/volatile/registry.json
    - internal/xweb/volatile/registry.go
    - internal/xweb/volatile/pins.go
    - internal/xweb/volatile/registry_test.go
    - internal/xweb/graphql.go
    - internal/xweb/graphql_test.go
    - internal/domain/domain.go
    - internal/xsource/user.go
    - internal/xsource/tweets.go
    - internal/xsource/xsource_test.go
    - internal/xsource/testdata/fixture-usertweets.json
    - internal/opsgen/opsgen.go
    - internal/opsgen/opsgen_test.go
    - internal/opsgen/testdata/bundle-slice.js
    - cmd/x-opsgen/main.go
    - cmd/spike/main.go
    - internal/cli/whoami.go
    - internal/cli/whoami_test.go
    - internal/cli/notice.go
    - internal/cli/notice_test.go
  modified:
    - internal/xweb/transport.go
    - internal/cli/doctor.go
    - internal/cli/cli.go
    - internal/config/config.go
    - internal/cli/session_test.go
    - internal/netcheck/netcheck.go
    - internal/netcheck/netcheck_test.go
    - internal/cli/doctor_test.go

key-decisions:
  - "The response schema is the one *recorded from the live site*, not the one the research named: UserTweets nests the timeline at data.user.result.timeline.timeline, and a repost's original tweet lives under legacy.retweeted_status_result (18 of one account's 20 entries). Both were found live and both are read; the plan's timeline_v2 path and the root-level retweet wrapper are still accepted because both have been seen"
  - "UserTweets needs the full variable set the two maintained clients send (userId, count, includePromotedContent, withQuickPromoteEligibilityTweetFields, withVoice). userId+count — the probe's curl shape — answers 422 with an empty body from the live site"
  - "features and fieldToggles are sent only when the operation declares flags, mirroring the bundle's own request builder (`_&&(u.features=…)`); the values are objects of {name:true}, as the plan says"
  - "A new package, internal/opsgen, holds the refresh logic; cmd/x-opsgen and `doctor --refresh-ids` share it, because a main package cannot be imported and duplicating the bundle parser would be the one untested path in the tool"
  - "netcheck gained a fifth stage: the registry-driven Viewer read, with its own outcomes (API read OK / API read failed). Stage 4's cookie-only navigation read is kept exactly as plan 02 built it — the user's instruction was to keep the cookie-only navigation behaviour — so doctor now prints both, and the API line names the account"
  - "The guard test proves ids are not literals in two arms: a 20+ base64url run beside the word queryId in any non-test Go file, and any of the 104 known ids anywhere in one. Test files are exempt by design — graphql_test.go pins the Viewer id, which is the opposite of a fallback"
  - "The registry's Source field (snapshot|pin) is an addition to the plan's Operation struct: 'a pin is marked as a pin' needs somewhere to be marked, and a refresh must not rewrite what the user pinned"
  - "The first-run statement goes to stderr (stdout belongs to the verb's output); `doctor --show-notice` prints it on stdout because it was asked for, and --show-notice with --json is refused rather than splicing prose into the JSON object"

patterns-established:
  - "Every id that can go stale is data with provenance; volatile.Lookup is the only way to name one, and an unknown name is a typed error that says how to fix it"
  - "A refresh adds and updates, never deletes, and prints a diff; a pinned entry is the user's and is never overwritten"
  - "A parser that cannot find a field names the dotted path and returns an error — no zero values, no empty timelines"
  - "Every verb that reads has an injectable client seam, so the tests drive the whole path against httptest rather than the live site"
  - "Live evidence lives in the repository as a fixture (or is deliberately kept out of it, when it is the account's own data)"

requirements-completed: [XAPI-01, XAPI-02, XAPI-03, XAPI-05, AUTH-03, OPS-05]

# Coverage metadata — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "The operation registry as data: registry.json embedded with //go:embed (104 operations with their feature switches and field toggles, provenance recorded), Lookup with no stale-id fallback, FeaturesJSON/TogglesJSON in the wire shape"
    requirement: XAPI-02
    verification:
      - kind: unit
        ref: "internal/xweb/volatile/registry_test.go#TestTheEmbeddedSnapshotCarriesEveryMinedOperation, #TestAnUnknownOperationIsTypedAndSaysHowToFixIt, #TestFeaturesJSONIsTheWireObject, #TestTheSnapshotKeepsEveryMinedOperation"
        status: pass
      - kind: manual_procedural
        ref: "grep -c '\"queryId\"' internal/xweb/volatile/registry.json -> 104"
        status: pass
    human_judgment: false
  - id: D2
    description: "A guard test that fails if an operation id appears anywhere outside the registry data (two arms: an id-shaped literal beside the word queryId, and any of the 104 known ids), proven able to fail"
    requirement: XAPI-02
    verification:
      - kind: unit
        ref: "internal/xweb/volatile/registry_test.go#TestTheOperationIDsLiveOnlyInThisPackagesData"
        status: pass
      - kind: manual_procedural
        ref: "anti-vacuity arm A: added `var queryIdLookalike = \"AAAAAAAAAAAAAAAAAAAAAAAAxxxx\"` to internal/xweb/classify.go -> the guard failed naming that line; arm B: added `var borrowedID = \"TpQxQtWMTMnIAMPlMGMqNw\"` -> the guard failed naming the file and the id; classify.go restored byte-identical (cmp) both times"
        status: pass
    human_judgment: false
  - id: D3
    description: "A refresh that can only add or update (never delete), reports a diff summary, leaves pins alone, and writes atomically; driven from a bundle file or from the live site"
    requirement: XAPI-03
    verification:
      - kind: unit
        ref: "internal/xweb/volatile/registry_test.go#TestAMergeAddsAndUpdatesButNeverDeletes, #TestAMergeLeavesAPinAlone, #TestPinWritesAtomicallyAndKeepsItsNeighbours, #TestWriteRoundTripsTheTable, #TestTheDiffSummaryIsTheLineTheToolPrints; internal/opsgen/opsgen_test.go#TestParseReadsTheRealBundleShape (a verbatim slice of the real 1.1MB bundle), #TestBundleURLReadsBothShellShapes, #TestBundleURLRefusesALoggedOutShell, #TestFetchReadsTheShellAndTheBundle"
        status: pass
      - kind: manual_procedural
        ref: "go run ./cmd/x-opsgen --from-file /tmp/main-01-03.js --merge --registry /tmp/reg-copy.json (the real bundle, sha256 da9a4bb9…) -> '+0 added, ~0 updated, 104 unchanged'; on a synthetic bundle with one new and one changed operation -> '+1 added, ~1 updated, 1 unchanged (105 operations)', UserTweets survived"
        status: pass
    human_judgment: false
  - id: D4
    description: "A user pin wins over the snapshot and is written atomically to <datadir>/pins.json; a broken pins file is a warning, never silently ignored, and is never overwritten by a pin"
    requirement: XAPI-03
    verification:
      - kind: unit
        ref: "internal/xweb/volatile/registry_test.go#TestAPinWinsOverTheSnapshot, #TestAPinsFileThatCannotBeReadIsAWarningNotSilence, #TestPinRefusesWhatItCannotPin, #TestPinRefusesToReplacePinsItCannotRead; internal/cli/doctor_test.go#TestDoctorPinsAnOperationWithoutTouchingTheNetwork, #TestDoctorRefusesAPinItCannotWrite"
        status: pass
      - kind: manual_procedural
        ref: "live: XDG_DATA_HOME=<temp> go run ./cmd/x-terminal doctor --pin HomeTimeline 1u0Wlkw6Ru1NwBUD-pDiww -> exit 0, pins.json written 0600 with that entry"
        status: pass
    human_judgment: false
  - id: D5
    description: "The registry-driven GraphQL client: the measured GET (queryId from the registry, URL-encoded variables/features/fieldToggles) with the measured six headers and the web bearer, through the limiter, with every non-success outcome becoming an APIError that carries the outcome and the upstream's own words"
    requirement: XAPI-01
    verification:
      - kind: unit
        ref: "internal/xweb/graphql_test.go#TestGraphQLBuildsTheMeasuredRequest (path, the three parameters, all eight headers, no x-client-transaction-id), #TestGraphQLRefusesAnEmptyBodiedRefusalAsASignerOrAuthFailure, #TestGraphQLCarriesTheUpstreamsOwnMessage, #TestGraphQLRefusesWhatItCannotIssue (unknown operation, no session, every mutation), #TestGraphQLRunsTheSignerSeam, #TestGraphQLFallsBackToTheEmbeddedSnapshot"
        status: pass
      - kind: manual_procedural
        ref: "the live spike's Viewer call: 200 with the real account; a 403-empty answer is reported as OutcomeSignerOrAuth (unit-pinned)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The signer seam: type Signer {Header(ctx, *http.Request) error} with NoopSigner, documented from measurement (two operations answered 200 without x-client-transaction-id on 2026-10-14), so a per-operation requirement is one implementation"
    requirement: XAPI-05
    verification:
      - kind: unit
        ref: "internal/xweb/graphql_test.go#TestGraphQLRunsTheSignerSeam (a signer's header reaches the wire; a failing signer aborts before the request)"
        status: pass
      - kind: manual_procedural
        ref: "live: both spike calls (Viewer, UserTweets) answered 200 with no transaction id sent"
        status: pass
    human_judgment: false
  - id: D7
    description: "Parsers for the live response shapes: ParseUser (core/avatar/banner/rest_id/has_graduated_access, never legacy) and ParseTimeline (entries, conversation modules, cursors, note_tweet text, repost collapse), each failing with a typed error naming the path"
    requirement: XAPI-01
    verification:
      - kind: unit
        ref: "internal/xsource/xsource_test.go#TestParseTimelineReadsTheRecordedResponse (the 130,245-byte recorded response: >=20 posts with id/author/text/date, pinned first post, both cursors, a quoted id), #TestParseTimelineRefusesATruncatedResponse, #TestParseTimelineRefusesAnAnswerWithNoEntries, #TestParseTimelineUnwrapsAConversationModule, #TestParseTimelineCollapsesARepost (both measured nestings), #TestParseTimelinePrefersTheLongFormText, #TestParseTimelineRefusesATweetItCannotRead, #TestParseUserDoesNotFallBackToTheRetiredSchema, #TestParseUserRefusesAUserWithoutAnIdentity, #TestTheFixtureIsTheWholeRecordedResponse (>100 KB)"
        status: pass
      - kind: manual_procedural
        ref: "live: the recorded Viewer response is data.viewer.user_results.result.{core,avatar,banner,rest_id,id,has_graduated_access} — exactly what ParseUser reads — and the live timeline is data.user.result.timeline.timeline.instructions with legacy.retweeted_status_result"
        status: pass
    human_judgment: false
  - id: D8
    description: "cmd/spike: one authenticated read end to end against the live site, printing the viewer, one line per post (author, date, text), the request budget and the elapsed time — and nothing at all on stdout when a step fails"
    requirement: XAPI-01
    verification:
      - kind: manual_procedural
        ref: "live, proxy http://127.0.0.1:7897: `go run ./cmd/spike` -> 'viewer: @example-viewer (Example Viewer)', 20 posts with author/date/text (reposts printed as 'RT @x (reposted by @example-viewer)'), 'requests: 2 used, 0 limited, cap 200 · 4.6s', exit 0"
        status: pass
    human_judgment: false
  - id: D9
    description: "whoami: names the account the session belongs to, with the session's own state beside it, and a named outcome plus a non-zero exit when the session is missing, refused or unreadable"
    requirement: AUTH-03
    verification:
      - kind: unit
        ref: "internal/cli/whoami_test.go#TestWhoamiNamesTheAccount, #TestWhoamiReportsARejectedSession, #TestWhoamiWithoutASessionSaysSo, #TestWhoamiRefusesAnAnswerWithoutAHandle, #TestWhoamiTakesNoArguments, #TestTheFixtureHasTheRealCookieShape"
        status: pass
      - kind: manual_procedural
        ref: "live: `go run ./cmd/x-terminal whoami` -> '@example-viewer — Example Viewer' + 'session: present (auth_token 40 bytes, ct0 160 bytes, source /home/user/.config/x-terminal/session)', exit 0"
        status: pass
    human_judgment: false
  - id: D10
    description: "The account-risk statement: five lines or fewer, printed once on the first successful run and recorded at <datadir>/notice-shown (0600), reprintable with doctor --show-notice"
    requirement: OPS-05
    verification:
      - kind: unit
        ref: "internal/cli/notice_test.go#TestTheStatementFitsTheFiveLinesItPromises, #TestShowNoticeOncePrintsItOnceAndRecordsIt, #TestShowNoticeForcesTheStatement, #TestAnUnwritableMarkerStillPrintsTheStatement"
        status: pass
      - kind: manual_procedural
        ref: "live: rm ~/.local/share/x-terminal/notice-shown && whoami -> the statement on stderr, once; the next whoami printed only the handle and the session line"
        status: pass
    human_judgment: false
  - id: D11
    description: "doctor reports the operation table and the registry-driven API read: 'authenticated read OK' (the cookie-only navigation read) followed by 'API read OK' naming the account, plus the request budget"
    requirement: XAPI-01
    verification:
      - kind: unit
        ref: "internal/netcheck/netcheck_test.go#TestCheckReportsAuthenticatedReadOK (three requests, the cookie on both authenticated reads, no bearer on the navigation one, the API line naming the account), #TestCheckSpendsExactlyThreeRequestsOnASuccessfulRun, #TestCheckReportsABodyWithoutADeclaredLengthWithoutInventingOne"
        status: pass
      - kind: manual_procedural
        ref: "live: `go run ./cmd/x-terminal doctor` -> exit 0 with 'session authenticated read OK 1.472s' and 'session API read OK 1.474s the Viewer operation answered 200 for @example-viewer (Example Viewer)', 'requests: 3 used, 0 limited, cap 200'"
        status: pass
    human_judgment: false

# Metrics
duration: ~75min
completed: 2026-10-14
status: complete
---

# Phase 1: Reachable Transport and One Live Read Summary

**The measured request path as code: operation ids as embedded data with pins and a refresh tool, a registry-driven GraphQL client, parsers for the shapes the live site actually sends — proven by a live authenticated read from this host that printed `@example-viewer`, twenty posts with author, date and text, and a budget of two requests.**

## Performance

- **Duration:** ~75 min (first file written at 13:52, last live check at 15:07)
- **Started:** 2026-10-14T13:5x+08:00
- **Completed:** 2026-10-14T15:07+08:00
- **Tasks:** 3
- **Files modified:** 28 (20 created, 8 modified)

## Accomplishments

- **The operation table is data, and provably not code.** `internal/xweb/volatile/registry.json` holds all 104 operations from the mined bundle — `queryId`, `operationType`, the feature switches and the field toggles each operation ships with — plus provenance (bundle URL, sha256, byte count, method, mining date). It is embedded with `//go:embed`, resolved through `Lookup` (no stale-id fallback: an unknown name is a typed error naming the operation and saying how to fix it), overridden by pins at `<datadir>/pins.json`, and refreshed by `cmd/x-opsgen` (bundle file or live site) which can only add or update. Live: `--from-file` on the real 1.1MB bundle reported `+0 added, ~0 updated, 104 unchanged`, which is the strongest statement available that the reshaped data is complete and unaltered.
- **A guard test that proves the property, and that was made to fail twice.** It has two arms: a 20+ character base64url-shaped literal on a line mentioning `queryId` in any non-test Go file outside this package, and any of the 104 known ids anywhere in one. Both arms were exercised by planting the offending line in `internal/xweb/classify.go`, watching the guard fail with an actionable message, and restoring the file byte-identically.
- **A registry-driven GraphQL client.** `Client.GraphQL` builds the measured GET — the id from the registry, `variables`/`features`/`fieldToggles` URL-encoded through `net/url.Values`, and the eight measured headers including the public web bearer — sends it through the one limiter, and runs the answer through `Classify`; anything that is not data becomes an `*APIError` carrying the outcome, the status and the upstream's own (bounded) words. The transaction-id signer stays a seam with a documented no-op, because two operations were measured answering 200 without it.
- **Parsers for the shapes the live site really sends.** `ParseUser` reads the new `core`/`avatar`/`banner`/`rest_id` schema from either container (`data.user.result` or `data.viewer.user_results.result`) and never reads `legacy`; `ParseTimeline` reads entries, conversation modules, cursors, `note_tweet` text and reposts, and fails with a typed error naming the dotted path when a field moved. Both are pinned by the 130,245-byte recorded `UserTweets` response — including an anti-vacuity arm that the fixture is still over 100 KB.
- **Live, from this host, through the proxy.** `go run ./cmd/spike` printed `viewer: @example-viewer (Example Viewer)`, twenty posts (each with author, date and text; reposts rendered as the original post with `(reposted by @example-viewer)`), and `requests: 2 used, 0 limited, cap 200 · 4.6s`, exit 0. `x-terminal whoami` printed the same account with the session's own state, and printed the account-risk statement exactly once. `x-terminal doctor` exited 0 with `authenticated read OK` and `API read OK` naming the account.
- **Nothing else can spend a request.** Every request in this plan went through `xweb.Client`'s limiter: two for the spike, one for each `whoami`, three for `doctor`. No loops, no retries.

## Task Commits

**None made by this executor.** The orchestrator commits this plan's files; no git command was run and no hashes are claimed.

## Files Created/Modified

- `internal/xweb/volatile/registry.json` — the embedded table: 104 operations with provenance.
- `internal/xweb/volatile/registry.go` — `Operation`, `Registry`, `Lookup` (package-level and method), `Load` (pins over snapshot), `ReadFile`, `Merge` (add/update only, pins untouched), `Write` (atomic), `FeaturesJSON`/`TogglesJSON`, `RecordRefresh`, `Provenance`, `Warnings`.
- `internal/xweb/volatile/pins.go` — `Pin` (validate, read, merge, atomic write at 0600), `readPins`, `applyPins` (per-field override), `PinsFileName`.
- `internal/xweb/volatile/registry_test.go` — the data, the pins, the merge rules, the wire objects, the round trip, the diff line, the pruning guard against `.planning/research/x-ops-snapshot.json`, and the two-armed id-literal guard.
- `internal/xweb/graphql.go` — `WebBearer`, `Signer`/`NoopSigner`, `APIError`, `Client.GraphQL`, the request builder, `apiMessage`/`bounded`.
- `internal/xweb/graphql_test.go` — the request as it goes out, the refusals, the signer seam, the snapshot fallback.
- `internal/domain/domain.go` — `User`, `Post`.
- `internal/xsource/user.go` — the package doc, `ParseError`, `ParseUser`.
- `internal/xsource/tweets.go` — `Page`, `ParseTimeline`, the entry/module walk, `postFrom`, `authorOf`, `unwrapTweet`, `tweet.repost`, the response shapes.
- `internal/xsource/xsource_test.go` — the recorded response, the refusals, the module, the repost (both nestings), the long-form text, the identity rules, the anti-legacy arm, the fixture floor.
- `internal/xsource/testdata/fixture-usertweets.json` — the real response, copied unchanged (sha256 `2f268f5e…`).
- `internal/opsgen/opsgen.go` — `Parse`, `BundleURL`, `Fetch`, `Operations`, `ShellURL`, `RegistryPath`.
- `internal/opsgen/opsgen_test.go` + `testdata/bundle-slice.js` — a verbatim slice of the real bundle and the request/cookie/host assertions of a refresh.
- `cmd/x-opsgen/main.go` — the refresh tool (preview by default, `--merge` to write, `--registry` to choose the file).
- `cmd/spike/main.go` — the throwaway prover, with `-record`.
- `internal/cli/whoami.go`, `internal/cli/notice.go` (+ their tests) — the verb and the once-only statement.
- `internal/xweb/transport.go` (modified) — `Client` gained `session`/`registry`/`signer` with `WithSession`/`WithSigner`/`WithRegistry`, and `NewClient` now loads the registry with pins.
- `internal/cli/doctor.go` (modified) — the three flags wired, the registry line, the notice on success, the API stage's line.
- `internal/netcheck/netcheck.go` + tests (modified) — the fifth stage: the registry-driven Viewer read, with `API read OK` / `API read failed`.
- `internal/cli/cli.go` (modified) — `whoami` registered, the stub and `ExitUnimplemented` removed.
- `internal/config/config.go` (modified) — `PinsPath()`, `NoticePath()`.
- `internal/cli/session_test.go` (modified) — the hermetic helper now isolates `XDG_DATA_HOME` too.

## Decisions Made

1. **The live response, not the research, settled two shapes.** `UserTweets` nests its timeline at `data.user.result.timeline.timeline` (not `timeline_v2`), and a repost's original tweet is under `legacy.retweeted_status_result` (18 of one account's 20 entries), not at the tweet's root. Both paths are read; both alternatives remain accepted because both have been seen. The first live spike run printed `RT @…` text attributed to the reposter — a post that looked like a post and was not one — and that is what caught it.
2. **`UserTweets` needs the full variable set** (`userId`, `count`, `includePromotedContent`, `withQuickPromoteEligibilityTweetFields`, `withVoice`). The probe's curl shape (userId+count) answers a `422` with an empty body from the live site; the two maintained clients both send the five. Recorded in the spike next to the call.
3. **`features`/`fieldToggles` are sent only when the operation declares flags** — mirroring the bundle's own builder (`_&&(u.features=JSON.stringify(_))`), with objects of `{name:true}` rather than the arrays the registry stores.
4. **`internal/opsgen` is a new package.** `cmd/x-opsgen` and `doctor --refresh-ids` must run the same refresh; a main package cannot be imported, and a second copy of the bundle parser would have been the one untested path in the credential-adjacent refresh.
5. **`netcheck` gained a fifth stage rather than losing its fourth.** Stage 4 keeps the cookie-only navigation read plan 02 built and documented (a bearer on a navigation request is answered `401`), and the new stage proves the API path and names the account. `doctor` therefore spends three requests; the budget line says so.
6. **`Operation.Source` is an addition to the plan's struct.** "A pin is marked as a pin" needs somewhere to be marked, and a refresh must leave a pinned entry exactly as the user wrote it.
7. **The guard exempts test files, deliberately.** `graphql_test.go` pins the Viewer id, which is the point of a wire-format test; the acceptance grep in the plan does the same. The guard's second arm (any of the 104 ids anywhere in a non-test file) is far stronger than the literal reading of the first.
8. **The risk statement goes to stderr.** stdout belongs to the verb's output; `doctor --show-notice` prints it to stdout because it was asked for, and `--show-notice --json` is refused rather than splicing prose into the JSON object.

## Deviations from Plan

1. **Files the plan did not list were touched**, all of them because the plan's own instructions required it:
   - `internal/xweb/transport.go` — `Client` had to carry the session and the registry for `GraphQL`'s signature (`GraphQL(ctx, opName, variables)`) to have anything to authenticate with.
   - `internal/cli/doctor.go` and `internal/cli/cli.go` — Task 3 says to wire `doctor --refresh-ids`, `doctor --pin` and to register `whoami`; both live there. The `stub` body and `ExitUnimplemented` were removed with them (dead weight once every verb has a body).
   - `internal/config/config.go` — `PinsPath()`/`NoticePath()` so the refresh tool, the client and the doctor cannot disagree about where a pin is written and read.
   - `internal/netcheck/netcheck.go` (+ its tests) — the fifth stage, per the task brief's "prefer wiring Viewer through the registry".
   - `internal/cli/session_test.go` — the hermetic helper had to isolate `XDG_DATA_HOME`, or the new tests would have written pins and a notice marker into the developer's real data directory.
   - `internal/xsource/testdata/fixture-viewer.json` was **not** added: the Viewer response carries this account's own profile (19.4 KB), so it stays out of the repository. Its shape was recorded to a temp directory, inspected path by path, and the parser's test fixtures are written from that shape.
2. **A new package (`internal/opsgen`) and two new test files** (`internal/opsgen/opsgen_test.go`, `internal/cli/whoami_test.go`, `internal/cli/notice_test.go`) — see decision 4 and the acceptance criteria in Tasks 1–3, which name behaviour that needs tests somewhere.
3. **`TestDoctorRefersTheUnwiredFlagsToPlan03` was replaced** by `TestDoctorPinsAnOperationWithoutTouchingTheNetwork` and friends: the flags it pinned as unwired are wired now.
4. **Two additions the plan did not name**: `cmd/spike -record` (bodies as they arrive, which is what made the second live finding diagnosable) and doctor's `registry` line (the "snapshot age" fact the phase's context promised, and the place a broken pins file is reported).

## Issues Encountered

- **The first live run failed with `422` and an empty body.** `UserTweets` with the probe's two variables was rejected; the variable set the maintained clients send fixed it. The failure was invisible because a 422 from this endpoint carries no body — `APIError.Detail` now also quotes a bounded JSON body when the answer is JSON but not an `errors[]` object, so the next such failure says something.
- **The second live run printed a lie.** Reposts came back as posts attributed to the reposter with `RT @…` text, because the recorded fixture had no repost at all and the research-era shape put `retweeted_status_result` at the tweet's root. The live body puts it under `legacy`. The parser now reads both and the test covers both nestings.
- **`internal/xweb/graphql.go` had to be repaired after a bad patch** (an accidental non-ASCII fragment and a duplicated statement); the guard test and the suite were re-run green afterwards.
- **`unshare -rn go test ./...` fails without `ip link set lo up`** (httptest servers bind to 127.0.0.1, which is down in a fresh network namespace): with loopback up and nothing else, the whole suite passes.

## User Setup Required

None. The proxy is passed with `X_TERMINAL_PROXY=http://127.0.0.1:7897` (nothing exports `*_PROXY` in a non-interactive shell, and no `config.toml` exists), and the session file was read only — never written, never printed. Live runs spent 11 requests in total on x.com (two spike runs, two `whoami`, one `doctor`) plus one GET of the public bundle from `abs.twimg.com`.

## Next Phase Readiness

- **Phase 2 can parse without the network.** `internal/xsource` returns `domain.User`/`domain.Post`, and its fixture is a real response; extending the parser (media entities, metrics, reply context) needs a new fixture and one function, not a live call.
- **`Client.GraphQL` is the only API path.** A screen asks for `UserTweets` or `HomeTimeline` by name and gets bytes or an `*APIError` whose `Outcome` names the state and whose `Error()` carries the remedy — the two things a UI must never invent.
- **The pins file is how a lazily-loaded operation gets an id.** `doctor --pin HomeTimeline <id>` writes it, the client reads it every start, and a refresh will not overwrite it. The timeline operation ids still have to come from an external source (they are absent from `main.js`), which is now a data problem rather than a code one.
- **Verified evidence, all green:** `go build ./...` exit 0; `go vet ./...` exit 0; `go test ./...` 203 tests in 12 packages, exit 0; `gofmt -l internal/ cmd/` empty; the whole suite also passes inside `unshare -rn` with only loopback up (the proof that no test touches the real network); live `spike`, `whoami` and `doctor` as recorded above.
- **One acceptance criterion is met in intent but not literally:** `grep -rn 'queryId' internal/ cmd/ --include=*.go | grep -v volatile | grep -v _test.go` is *not* empty — `internal/opsgen/opsgen.go` names the JS keys it parses (`queryId`, in its regex and in two comments). Those are field names, not ids; no id literal exists outside the registry data, and the guard test proves it in both arms.

---
*Phase: 01-reachable-transport-and-one-live-read*
*Completed: 2026-10-14*
