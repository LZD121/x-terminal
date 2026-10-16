---
phase: 01-reachable-transport-and-one-live-read
verified: 2026-10-15T06:22:11Z
status: passed
score: 13/14 must-haves verified
covered_files:
  - .planning/phases/01-reachable-transport-and-one-live-read/01-01-PLAN.md
  - .planning/phases/01-reachable-transport-and-one-live-read/01-01-SUMMARY.md
  - .planning/phases/01-reachable-transport-and-one-live-read/01-02-PLAN.md
  - .planning/phases/01-reachable-transport-and-one-live-read/01-02-SUMMARY.md
  - .planning/phases/01-reachable-transport-and-one-live-read/01-03-PLAN.md
  - .planning/phases/01-reachable-transport-and-one-live-read/01-03-SUMMARY.md
  - cmd/spike/main.go
  - cmd/x-opsgen/main.go
  - cmd/x-terminal/main.go
  - go.mod
  - internal/cli/cli.go
  - internal/cli/cli_test.go
  - internal/cli/doctor.go
  - internal/cli/doctor_test.go
  - internal/cli/leak_test.go
  - internal/cli/notice.go
  - internal/cli/notice_test.go
  - internal/cli/session.go
  - internal/cli/session_test.go
  - internal/cli/whoami.go
  - internal/cli/whoami_test.go
  - internal/config/config.go
  - internal/config/config_test.go
  - internal/domain/domain.go
  - internal/netcheck/netcheck.go
  - internal/netcheck/netcheck_test.go
  - internal/opsgen/opsgen.go
  - internal/opsgen/opsgen_test.go
  - internal/opsgen/testdata/bundle-slice.js
  - internal/session/leak_test.go
  - internal/session/session.go
  - internal/session/session_test.go
  - internal/xsource/testdata/fixture-usertweets.json
  - internal/xsource/tweets.go
  - internal/xsource/user.go
  - internal/xsource/xsource_test.go
  - internal/xweb/classify.go
  - internal/xweb/classify_test.go
  - internal/xweb/graphql.go
  - internal/xweb/graphql_test.go
  - internal/xweb/limiter.go
  - internal/xweb/limiter_test.go
  - internal/xweb/proxy.go
  - internal/xweb/proxy_test.go
  - internal/xweb/transport.go
  - internal/xweb/volatile/pins.go
  - internal/xweb/volatile/registry.go
  - internal/xweb/volatile/registry.json
  - internal/xweb/volatile/registry_test.go
covered_digest: "v3:sha256:6ae7695cb7b5fae792992d10efe94fa0b8372c46cf60b4bc3e8cad5ee3aa99d3"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 13/14
  gaps_closed: []
  gaps_remaining: []
  regressions: []
deferred:
  - truth: "`cmd/spike` completes one authenticated read against the live site and prints the parsed author and text"
    addressed_in: "human check"
    evidence: "Present, substantive and wired: cmd/spike/main.go builds in `go build ./...`, the parsers it drives pass their own tests, and the recorded fixture is byte-identical to .planning/research/evidence/fixture-usertweets.json (130,245 bytes, sha256 2f268f5e…, re-checked with `cmp` this pass). Its live claim needs a route to x.com and a working proxy; this pass had neither, and internal/xsource/capture_test.go's live legs skip unless X_TERMINAL_CAPTURE=1 (6 such skips in both suite runs). Deferred to the live re-run in Human Verification, neither verified nor a gap."
---

# Phase 1: Reachable Transport and One Live Read — Verification Report

**Phase Goal:** The client reaches x.com through the user's proxy and completes one real authenticated read, and every way that can fail is a named, distinct state rather than an empty timeline.
**Verified:** 2026-10-15T06:22:11Z
**Status:** passed
**Re-verification:** Yes — fifth pass, on the frozen tree, after the milestone's narrow fix round landed on three files this report covers (`internal/config/config.go`, `internal/config/config_test.go`, `internal/xweb/volatile/pins.go`). All fourteen truths were re-adjudicated from the top; **no truth changed state**, no new gap appeared, and one new P3 note was recorded (the pins-name coupling is pinned by a name literal, not by an arm that would catch a hardcoded path). The `whoami` gap stays closed.

**Method.** `go build ./...` and `go vet ./...` both exit 0 (re-run this pass). The suite was run once as a whole (`go test ./... -count=1 -json`: 1579 test cases = 699 top-level `func Test` plus 880 subtests, 22 packages, 0 failures) and once inside a network namespace with only loopback up (`unshare -rn` + `ip link set lo up`): identical — 1579 cases, 22 packages, 0 failures — so nothing skipped itself into a pass. The phase's own packages were run again on their own (`go test ./internal/{session,xweb,netcheck,cli,config,xsource,opsgen,domain}/... -count=1`: 436 tests, 9 packages, all `ok`). Eight tests skip in both runs — six `internal/xsource` live/fixture legs gated on `X_TERMINAL_CAPTURE=1`, `TestNoFixtureCarriesViewerIdentity` (which skips on the *first* gate, naming the temp-dir session path, so the operator's real session was never opened), and the two `internal/render` legs that need a bare Ghostty window — and none of them is one of this phase's fourteen truths.

**Digest.** The stored digest (`v3:sha256:74130648…`) no longer matched: recomputing over the previous pass's 49-path list gives `v3:sha256:6ae7695c…`, so this report was stale before this pass, exactly as the assignment said. The set is **unchanged at 49 paths — nothing added**. It is adjusted path-by-path against the covered-file mtimes: the only covered files whose mtime is later than the previous pass (2026-10-15T02:59:46Z / 10:59:46 +0800) are `internal/config/config.go` (14:05:38), `internal/config/config_test.go` (14:07:44) and `internal/xweb/volatile/pins.go` (14:08:09) — precisely the fix round's footprint inside this set, which is why the digest moved and nothing else did. Three files the fix round also touched were judged not to belong, for the same reason the previous pass gave:
`internal/cli/feed.go` (saved-frame capability predicate) names phase-2/7 states and no phase-1 truth, artifact or key-link; its own arms are `internal/cli/feed_test.go`, and this pass re-checked the predicate from the phase-1 CLI surface without covering the file (arms 18, 19 below). `internal/keys/keys.go`'s comment is phase 3's keymap table (`ByKeystroke` has no production caller: `grep -rn ByKeystroke internal/` finds its definition and `keys_test.go` only). `internal/domain/page.go`, which carries the `PostPage.Requested` doc the fix round corrected, is phase 2's port (`domain.HomeKind`/`Cursor`/`PostPage`, used by `xsource/home.go`, `profile.go`, `saved.go`); the phase-1 file in this set is `internal/domain/domain.go`, the `User`/`Post`/`MediaRef` vocabulary the phase-1 parsers produce.

**The real session directory was not touched by this pass.** `/home/user/.config/x-terminal/` holds no `session` file (checked once, by directory listing only). Every binary run and every test run set `XDG_CONFIG_HOME` and `XDG_DATA_HOME` to a temp dir (`/tmp/p1v-etc`, `/tmp/p1v-data`), so `config.Load()` resolved a temp path in every run; the only session files this pass read or deleted were ones it wrote itself under those temp paths. Nothing read, printed or copied the operator's credential, and no command ran `logout` against a real path.

**What the fix round does to this phase, one by one.** (1) `PinsPath` is now built from `volatile.PinsFileName` (`config.go:183`, `filepath.Join(c.DataDir, volatile.PinsFileName)`) — the claim is true in the code, and it is now *checked*: renaming the constant makes `TestPinsPathIsTheRegistersName` fail naming the rename as a migration for the operator's existing file, and an offline `doctor --pin` run still writes `pins.json` (0600, 182 bytes) and reports `107 operations, mined 2026-10-14, 1 pinned`. What the test does *not* catch is a hardcoded name that keeps a dangling reference to the constant (arm 17b, recorded as a P3 note; the reference itself is load-bearing for the build — arm 17 fails to compile with an unused import). The new dependency is one hop (`go list -deps ./internal/config` → `internal/xweb/volatile`, and nothing in-module below it) and the audit's `boundaries` check passes. (2) `internal/cli/feed.go`'s saved-frame predicate is now `errors.As(err, &store.FileError{})`, a whitelist; `savedTransportError` and the blacklist are gone from the tree (`grep -rn savedTransportError internal/` finds nothing, and `store.FileError` is set only by the store's own open/page/status reads). Both failure directions were re-falsified from the CLI surface (arms 18, 19). (3) The `keys` comment now states the matcher's real purpose, which is what the code says too. (4) The `opnames` audit check is new and the tree's check count grew to 10; it reads the audit package's own `opClass` table plus `registry.json`, and the audit still runs clean (`x-terminal audit` → 10 checks, 0 findings, 893 files inspected, exit 0).

## Goal Achievement

### Observable Truths

| # | Plan | Truth | Status | Evidence |
|---|------|-------|--------|----------|
| 1 | 01-01 | A session file with mode 0644 is refused with a message naming the exact chmod command | ✓ VERIFIED | `internal/session/session.go:186-189` refuses `perm&0o077 != 0` with `chmod 600 %s`. `TestLoadRefusesLooseMode` passes; re-falsified this pass (the perm check rewritten to never fire → FAIL at `session_test.go:97: Load accepted <path>`), reverted byte-identical. End to end again: `X_TERMINAL_SESSION=<0644 file> doctor` printed `session file <path> has mode 0644: users other than its owner can read the cookie; fix it with: chmod 600 <path>`, exit 1 (same through `whoami`). |
| 2 | 01-01 | A symlinked session path is refused | ✓ VERIFIED | `session.go:181-182` checks `Lstat` plus `ModeSymlink` before the regular-file and mode checks. `TestLoadRefusesSymlink` passes; re-falsified this pass (the symlink case deleted → FAIL at `session_test.go:108: the refusal does not mention "symlink"`), reverted. End to end: a symlink at `X_TERMINAL_SESSION` printed `session file is a symlink: <path>`, exit 1. |
| 3 | 01-01 | Environment values for auth_token/ct0 win over the file | ✓ VERIFIED | `session.go:218-223`; `TestEnvironmentOverridesWin` passes. Re-falsified this pass (both override blocks deleted → FAIL with three messages, incl. `session_test.go:129: Source = "<path>", want "environment"`), reverted. The sibling path override works too: `X_TERMINAL_SESSION=<0600 file> logout` printed `session: present (auth_token 40 bytes, ct0 160 bytes, source <overridden path>)`, deleted that file, exit 0, and left the XDG session file alone. |
| 4 | 01-01 | No code path prints the cookie, including %+v, %#v, JSON marshalling and errors | ✓ VERIFIED | Both arms re-falsified this pass in the scratch copy: `String()` returning the raw value made `TestSecretCannotLeak` FAIL naming both credentials (offsets 11 and 116); renaming `Format` away made it FAIL through the fmt-placeholder arm plus the two offsets (348, 413). My verb-matrix probe was re-run on this tree: 19 verbs (`%v %+v %#v %s %q %d %x %X %c %U %t %b %o %O %e %f %g %.3s %10s`), three struct-dump verbs and slog all yield the redacted literal; the value appears only under `%p`, the accepted boundary below. `Reveal()` still has four non-test call sites, all `Header.Set`; `internal/cli/leak_test.go`'s producer sweep passes. |
| 5 | 01-02 | With the proxy unset and x.com unreachable, the app reports "no route" — never "no posts" | ✓ VERIFIED (offline arm re-run; live leg out of scope) | Re-ran the built binary in a namespace with only loopback up and no proxy in the environment: `route  no route to x.com  1ms  x.com resolves to 2001:2::29, 198.18.0.41; [2001:2::29]:443 did not accept a TCP connection: dial tcp …: connect: network is unreachable`, `requests: 0 used, 0 limited, cap 200`, exit 1. Nothing can render that as an empty result: the only list-producing walk refuses an empty payload (`tweets.go:131`, `missing(path + ": the instructions carried no entries")`) and `Outcome.Failed()` is true for it. `TestCheckNamesNoRoute` (both arms) passes. |
| 6 | 01-02 | With the proxy pointed at a dead port, the app reports "proxy refused" rather than a session error | ✓ VERIFIED (offline arm re-run; live leg out of scope) | Re-ran the built binary with `X_TERMINAL_PROXY=http://127.0.0.1:9`: `route  route to x.com OK`, then `proxy  proxy refused  0s  dialing the proxy at 127.0.0.1:9 failed: dial tcp 127.0.0.1:9: connect: connection refused  see RUNBOOK.md#no-path-to-x.com`, exit 1. The verdict is the transport state, not the session: the same run's session line is a diagnostic before the verdict. `TestCheckNamesProxyRefused` passes. |
| 7 | 01-02 | With the proxy up, the self-check reports "authenticated read OK" only when a real API call returned 200 | ✓ VERIFIED (coupling re-derived offline; the live 200 was not re-run) | The code requires `classified.Success()` for the navigation line (`netcheck.go:275-289`) and a successful `GraphQL("Viewer")` plus `ParseUser` for the API line (`netcheck.go:301-325`). Re-derived adversarially with my own in-package probe over an httptest TLS server (`internal/netcheck/zz_probe_reverify_test.go`, scratch copy only): a zero-byte 200 on `/` is never `upstream reachable` (the probe fails with the `len(resp.prefix) > 0` clause removed, below), while its control — the same server answering a real page — reaches `upstream reachable`; the tree's own `TestCheckNamesUpstreamBlocked` (502 + "load shed") and `TestCheckReportsAuthenticatedReadOK` also pass. |
| 8 | 01-02 | A second request issued inside the 1.5s floor waits instead of failing | ✓ VERIFIED | `TestLimiterSpacesRequestsByTheFloor` (fake clock, exactly one wait of exactly 1500ms, plus its anti-vacuity arm) and `TestLimiterMakesTheSecondConcurrentCallWait` pass; re-falsified this pass (the floor's sleep disabled → FAIL at `limiter_test.go:55: waits = [], want exactly one: the second request was inside the floor`), reverted. `limiter.go:66` is still the tree's only timer/sleep outside tests (the audit's `timers` check names it as the single exemption) and `limiter.go:107-111` its only caller. |
| 9 | 01-02 | An HTTP 403 with a zero-byte body classifies as auth/signer, not as an empty result | ✓ VERIFIED | `classify.go:148-156` returns `OutcomeSignerOrAuth` for a 403 with an empty body and `OutcomeUnknownOperation` for a 404 with an empty body. Re-falsified this pass: remapping the 403-empty arm to `OutcomeEmptyBody` made `TestClassifyKeepsTheTwoEmptyBodiedRefusalsApart` FAIL at `classify_test.go:204: Classify gave ("empty response body", "unknown operation"), want ("signer or auth rejected", "unknown operation")`; reverted byte-identical. |
| 10 | 01-03 | `x-terminal whoami` prints the handle of the account the session belongs to, never the cookie | ✓ VERIFIED — the earlier pass's only failure; re-adjudicated again this pass | `internal/cli/cli.go:68` still carries the row (`{Name: "whoami", Usage: "whoami   name the account the loaded session belongs to", Run: Whoami}`), the file's package comment matches, and `--help` renders the row (re-run: the listing shows `doctor`, `logout`, `whoami`, `feed`, `keys`, `sync`, `audit`, exit 0). The guard is load-bearing in both directions, re-falsified this pass: deleting the row FAILs `TestEveryVerbBodyIsReachableThroughCommands` at `cli_test.go:127` (`Whoami is exported, implemented and unreachable from Main`) and `TestWhoamiIsDispatchableFromTheFrontDoor` at `cli_test.go:296` (`x-terminal whoami exited 2, want 0: unknown command: whoami`); planting a row whose body is another verb's FAILs the first test at `cli_test.go:140` (`commands() names "scratch", which is not a verb body in this package`). Through the built binary: no session → the typed `session: no session cookie found …` state, exit 1; with a session and a dead proxy → reaches `GET …/Viewer`, exit 1, no cookie anywhere in the output. |
| 11 | 01-03 | `cmd/spike` completes one authenticated read against the live site and prints the parsed author and text | ? UNCERTAIN — deferred | See `deferred:` and the Human Verification section. The artifact is present, substantive and wired; its live claim cannot be exercised in a pass with no route to x.com and no proxy. |
| 12 | 01-03 | Editing registry.json alone changes which operation id is used — no id literal exists in any other file | ✓ VERIFIED | Guard re-falsified both ways again this pass: planting `// scratch: queryId fallback` plus `var scratchQueryIdFallback = "AAAAAAAAAAAAAAAAAAAAAAAAxxxx"` in `internal/xweb/classify.go` FAILed arm A naming `registry_test.go:639: 1 line(s) outside internal/xweb/volatile spell an id next to queryId`; planting the real Viewer id `TpQxQtWMTMnIAMPlMGMqNw` there FAILed arm B with `registry_test.go:642: 1 file(s) hold an operation id from the registry`. (A concatenated spelling of the same id does not trip the byte-level scan — correct behaviour for a scan of contiguous values, and arm A is what catches a constructed call site.) `classify.go` restored byte-identical (`cp` from the repository, whole-tree `diff -r` clean). `grep -rn queryId internal/ cmd/ --include=*.go` finds only the registry package's own comments/regex; the runtime URL that `whoami` prints on the dead-proxy run carries the id because `volatile.Lookup` resolved it, not because a literal exists. |
| 13 | 01-03 | A user pin for an operation wins over the snapshot | ✓ VERIFIED | `TestAPinWinsOverTheSnapshot`, `TestAMergeLeavesAPinAlone`, `TestWriteRefusesATableWithTheUsersPinsInIt`, `TestWriteRoundTripsTheTable` all pass. Offline end to end through the built binary again this pass: `doctor --pin HomeTimeline og4a4SdSF3WiQkkwaPCdPg` wrote `pins.json` at 0600, 182 bytes, under the temp data dir, and the next run's line reads `registry   107 operations, mined 2026-10-14, 1 pinned`. The path it wrote is the one `PinsPath` builds from `volatile.PinsFileName` (the fix round's change, arm 16 below). |
| 14 | 01-03 | The first run prints the account-risk statement exactly once | ✓ VERIFIED (the production trigger not re-run live) | `TestShowNoticeOncePrintsItOnceAndRecordsIt` (marker written at 0600, second call prints nothing on either stream) and `TestAnUnwritableMarkerStillPrintsTheStatement` pass. Measured from outside the tests again this pass: `doctor --show-notice` printed the five-line statement on stdout and dialled **0** times against a counting socket on 127.0.0.1, exit 0; `--show-notice --json` exited 2 (usage) with 0 dials; a bare `doctor` through the same kind of socket dialled twice — so the flag spends no request. The statement's production trigger is a *successful* run (`doctor.go:213`, `whoami.go:95`), which needs the live leg. |

**Score:** 13/14 truths verified (0 failed, 0 present-behavior-unverified, 1 deferred to the live check)

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | `cmd/spike` completes one authenticated read against the live site and prints the parsed author and text | human check (live re-run) | Truth 11: the tree-side halves were re-verified here (the binary builds, the parsers refuse an empty payload, the fixture is byte-identical); the live half needs a working proxy. |
| 2 | The Following timeline as the default list | Phase 2 | ROADMAP FEED-01; the three operations are registered and `HomeLatestTimeline` was read live (`.planning/research/UPSTREAM-PROBE.md`) |
| 3 | A pin UI for operations bundle-scraping cannot see | Phase 6 | ROADMAP XAPI-04; the mechanism ships as data plus `doctor --pin` in Phase 1 |
| 4 | doctor's store-schema-version fact and the runbook | Phase 8 | ROADMAP OPS-03, OPS-06 |

### Advisory (New Scope, Unevidenced)

None. This pass re-ran recorded falsifications and re-adjudicated existing truths; it raised no new-scope finding.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `go.mod` | module `github.com/LZD121/x-terminal`, go 1.27, no third-party requires | ✓ EXISTS (module and Go version) / clause superseded | `module github.com/LZD121/x-terminal`, `go 1.27` — unchanged. The "no third-party requires" clause was this phase's own scope limit (plan 01 task 1: stdlib only) and later phases legitimately superseded it with the planned stack: Bubble Tea v2, `charmbracelet/x/*`, `modernc.org/sqlite` and their indirects now appear, with a `go.sum`. Recorded, not scored as a gap: it was never a standing phase-1 truth, and every file this phase owns still imports stdlib or in-module only. |
| `internal/session/session.go` | redacting credential type + loader with mode and symlink checks | ✓ EXISTS + SUBSTANTIVE | 344 lines, untouched this pass: `Redacted`, `Secret` with `String/GoString/Format/MarshalJSON/MarshalText/LogValue/Reveal`, `Session`, `Valid`, `Summary`, `ErrNoSession`, `Load`, `parseFile`, `keyName`. |
| `internal/config/config.go` | XDG paths, proxy block, TOML-free reader, `Redacted` | ✓ EXISTS + SUBSTANTIVE | 549 lines. Re-verified this pass through `internal/config`'s own tests: `TestProxyPrecedence` (`X_TERMINAL_PROXY` > file > `HTTPS_PROXY` > `HTTP_PROXY` > `ALL_PROXY`, upper-case before lower-case), `TestSessionPathPrecedence` (`X_TERMINAL_SESSION` beats the file), `TestNoProxyIsPassedThroughUnchanged`, `TestRedactedDropsProxyCredentials`. `PinsPath`/`NoticePath`/`SavedDBPath` all build under `DataDir` and return "" for a zero Config (`TestSavedDBPathBesidePinsAndNotice`); the pins name now comes from `volatile.PinsFileName` and is pinned by `TestPinsPathIsTheRegistersName`. |
| `cmd/x-terminal/main.go` + `internal/cli/cli.go` | subcommand dispatch | ✓ EXISTS + SUBSTANTIVE | 117 lines; usage renders from the table (re-run: 7 verbs listed), unknown verbs exit 2, `--help` exits 0, and the table holds all seven verbs including `whoami` (`cli.go:63-72`). One documented drift from later phases remains: a bare invocation opens the feed instead of printing usage (exit 1 with a named session state when there is no session), covered by `TestABareInvocationOpensTheFeed`. |
| `internal/cli/cli_test.go` | the surface guard: a body with no row, or a row with no body, is a test failure | ✓ EXISTS + SUBSTANTIVE | 327 lines; `TestEveryVerbBodyIsReachableThroughCommands` reads the table and the package's own sources in both directions (the defect AUTH-03 was), `TestWhoamiIsDispatchableFromTheFrontDoor` drives `Main`, `TestTheUsageListingIsAligned` checks the rendered listing. Falsified in both directions again this pass. |
| `internal/xweb/transport.go`, `proxy.go`, `classify.go`, `limiter.go` | one transport, one choke point, the measured table | ✓ EXISTS + SUBSTANTIVE | `NewClient` builds the single `http.Transport` (timeouts, `ForceAttemptHTTP2`, http/https/socks5/socks5h, `NO_PROXY`); `Client.Do` (`transport.go:132-140`) is still the only sender and routes through the limiter, and every request-issuing site goes through it (`graphql.go:141`, `netcheck.go:417`, `opsgen.go:260`, plus the later `probe.go:50`); 13 typed outcomes with `Error`/`Remedy`; limiter with injected clock, cap `ErrRequestCap`, `Budget`. |
| `internal/netcheck/netcheck.go` | the five-outcome self-check | ✓ EXISTS + SUBSTANTIVE | 445 lines; four stages plus the plan-03 API stage; the five promised outcomes verbatim (`no route to x.com`, `proxy refused`, `proxy reached but upstream blocked`, `TLS/Cloudflare challenge`, `authenticated read OK`) plus config/session/API states; stops at the first broken link in every run I made (the dead-proxy run stops at stage two, the namespace run at stage one). |
| `internal/cli/doctor.go` | a real `doctor` | ✓ EXISTS + SUBSTANTIVE | 832 lines; text and `--json` reports, `--pin`, `--refresh-ids`, `--show-notice`, `--probe`, non-zero exit on a failed link, `requests: N used, M limited, cap 200` taken from the limiter that spent them. Re-ran the failing path: valid JSON, `ok: false`, two check objects, `budget.used = 0`; the text run's first two stages render `route`/`proxy` with real dials. |
| `internal/xweb/volatile/registry.json` | operations as data with provenance | ✓ EXISTS + SUBSTANTIVE | 137,912 bytes, schema 1, provenance (`mined_at 2026-10-14`, bundle URL, sha256 `da9a4bb9…`, 1,157,073 bytes, method), **107** operations; `Viewer` `TpQxQtWMTMnIAMPlMGMqNw`, `UserByScreenName`, `UserTweets` present; three entries carry `"source": "external"` with no feature switches or toggles (`HomeTimeline og4a4SdSF3WiQkkwaPCdPg`, `HomeLatestTimeline OQPHTgwczzp9RMAPt6BH9A`, `Bookmarks -dgKZ58Dr9YSJYrcgEb5KA`). Identical to the phase-1 record. |
| `internal/xweb/volatile/registry.go` | per-entry provenance, a write that cannot launder a pin | ✓ EXISTS + SUBSTANTIVE | 469 lines: `Source` with `SourceSnapshot`/`SourcePin`/`SourceExternal`, `parse` naming the sources it refuses, `Merge` forcing `SourceSnapshot` and never deleting (re-falsified this pass), `Write` refusing a table that holds a pin. |
| `internal/xweb/graphql.go` | registry-driven authenticated request | ✓ EXISTS + SUBSTANTIVE | 268 lines; `GraphQL` → `operation()` → registry `Lookup` (no fallback id) → `url.Values`-encoded GET with the measured headers → `Client.Do` → `Classify`; non-success becomes `*APIError`; `Signer`/`NoopSigner` seam. |
| `internal/xsource/{user,tweets}.go` + `testdata/fixture-usertweets.json` | parsers for the live shapes, driven by a recorded fixture | ✓ EXISTS + SUBSTANTIVE | Fixture 130,245 bytes, sha256 `2f268f5ec2d7d89d4e37f170dd75f150e96f01cef8a14a2f2853a8b9eab2b759`, byte-identical again this pass (`cmp`, both paths). Both files grew in later phases (726 and 173 lines) and still refuse an empty payload rather than reporting an empty timeline (`walkInstructions` → `missing(path + ": the instructions carried no entries")`, `tweets.go:131`). |
| `cmd/spike/main.go` | throwaway binary that proves the path end to end | ✓ EXISTS + SUBSTANTIVE | Unchanged since phase 1; still builds; still documented as the throwaway whose fixtures move into `internal/xsource/testdata`. |
| `internal/opsgen/opsgen.go` + `cmd/x-opsgen/main.go` | the refresh tool | ✓ EXISTS + SUBSTANTIVE | Bundle record parser, add-or-update-only merge through `volatile.Merge`, preview by default; the only `queryId` occurrences outside the data file are its comments and the bundle regex. |

**Artifacts:** 13/13 present and substantive (one clause of `go.mod` recorded as superseded by design)

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `internal/cli` → `internal/config` → `internal/session` | session path → file contents | `config.Load()` then `session.Load(cfg.SessionPath)` | ✓ WIRED | `doctor.go:138,173`, `whoami.go`, `feed.go`, `sync.go`, `session.go` — all through `config.Load`; re-ran the override end to end (`X_TERMINAL_SESSION=<0600 file> logout` deleted exactly that file) |
| `internal/config` → `internal/xweb/volatile` | the pins file's name | `filepath.Join(c.DataDir, volatile.PinsFileName)` (`config.go:183`) | ✓ WIRED | One in-module hop (`go list -deps ./internal/config` names `internal/xweb/volatile` and nothing below it; no cycle, and the package is stdlib-only); `TestPinsPathIsTheRegistersName` passes and fails when the constant is renamed; `doctor --pin` writes at that path |
| `session.Secret` → every later package that needs the cookie | `.Reveal()` only where a request header is built | 4 non-test call sites, all `Header.Set`; no other package holds the cookie as a plain string | ✓ WIRED | Re-grepped this pass (`graphql.go:226,227`, `netcheck.go:414`, `opsgen.go:257`); `internal/cli/leak_test.go` sweeps the CLI's real producers for the fixture cookie and passes |
| `cli doctor` → `netcheck` → `xweb` transport | the self-check dials through the same transport the API uses | `netcheck.NewTarget(cfg)` → `xweb.NewClient(cfg)` → `CheckWith` | ✓ WIRED | `doctor.go`; the budget and first hop are read back from `target.Client()`, and the dead-proxy run's `requests: 0 used` comes from that limiter |
| every later `xsource` call → `xweb.Limiter.Do` | one request choke point | `Client.Do` → `Limiter.Do` → `HTTP.Do` | ✓ WIRED | `transport.go:139`; request-issuing sites `graphql.go:141`, `probe.go:50`, `netcheck.go:417`, `opsgen.go:260`, all via `Client.Do`. `limiter.go:66` is still the only timer/sleep outside tests. |
| `graphql.go` → `volatile.Lookup` | nothing else may name a queryId | `Client.operation()` | ✓ WIRED | `graphql.go:170-177`; the guard falsified both ways again this pass. |
| `xsource` → `graphql` → `xweb.Client` | the parse layer never touches http | imports | ✓ WIRED | `grep -rn '"net/http"\|"net/url"' internal/xsource/*.go internal/domain/*.go` (non-test) finds nothing (0 matches); the parsers take `[]byte`. |

**Wiring:** 7/7 connections verified

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `doctor` account line | `viewer.ScreenName` | `GraphQL("Viewer")` → `xsource.ParseUser` | Yes when the path is live: `API read OK … answered 200 for @handle (name)`; a rejected/empty answer exits non-zero with the named outcome (my probe arms) | ✓ FLOWING |
| `cmd/spike` post lines | `page.Posts` | `GraphQL("UserTweets", …)` → `ParseTimeline` | The parser refuses an empty payload with a typed error; the live 20-post run is the phase-1 record (row 11) | ✓ FLOWING (live leg not re-run) |
| `doctor` session lines | `netcheck.Result` per stage | real sockets through the shared client; budget read from the same limiter | Yes offline: `route`/`proxy` lines rendered with real dials (`requests: 0 used` on the refused-proxy run; 2 dials on the counting-socket control) | ✓ FLOWING |
| `doctor` registry line | `registry.Len()` / `Provenance()` / pin count | embedded `registry.json` (107 ops) + pins file | Yes: `107 operations, mined 2026-10-14` and `…, 1 pinned` with the pin written this pass | ✓ FLOWING |
| `whoami` output | `user.ScreenName` | `GraphQL("Viewer")` → `ParseUser` | Reachable from the CLI: `Main` dispatches the verb, the no-session run prints the typed state and exits 1, the loose-mode run prints the refusal, and the dead-proxy run reaches the Viewer request; the live 200 was not re-run | ✓ FLOWING (live leg not re-run) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Whole suite green | `go test ./... -count=1 -json` | 1579 test cases (699 `func Test` + 880 subtests), 22 packages, 0 failures; 8 test skips | ✓ PASS |
| Suite needs no real network | `unshare -rn bash -c 'ip link set lo up && go test ./... -count=1 -json'` | identical: 1579 cases, 22 packages, 0 failures, same 8 skips | ✓ PASS |
| Phase packages on their own | `go test ./internal/{session,xweb,netcheck,cli,config,xsource,opsgen,domain}/... -count=1` | 436 tests, 9 packages, all `ok` | ✓ PASS |
| Build and vet clean | `go build ./... && go vet ./...` | exit 0, exit 0 | ✓ PASS |
| Tree still audit-clean | `x-terminal audit` (temp XDG dirs) | 10 checks, 0 findings, 893 files inspected, exit 0 (incl. the new `opnames` and `readonly` checks) | ✓ PASS |
| `whoami` is dispatchable | `x-terminal --help`; `x-terminal whoami` (no session) | the listing names all seven verbs incl. `whoami`, exit 0; `whoami` prints the typed `session: no session cookie found …` state, exit 1 — not `unknown command` | ✓ PASS |
| `whoami` reaches the live path and names a transport failure | built binary, `whoami` with a session and `X_TERMINAL_PROXY=http://127.0.0.1:9` | reaches `GET …/Viewer`, reports the refused proxy, exit 1, no cookie in the output | ✓ PASS |
| Loose-mode session refused, end to end | built binary, `X_TERMINAL_SESSION=<0644 file> doctor` (and `whoami`) | `… has mode 0644 … chmod 600 <path>`, exit 1 | ✓ PASS |
| Symlinked session refused, end to end | built binary, `X_TERMINAL_SESSION=<symlink> doctor` | `session file is a symlink: <path>`, exit 1 | ✓ PASS |
| Session-path override honoured and only that file deleted | `X_TERMINAL_SESSION=<0600 file> logout` | `session: present (auth_token 40 bytes, ct0 160 bytes, source <that path>)`, deleted that path, left the XDG file in place, exit 0; a second `logout` with none: `no session file at …`, exit 0 | ✓ PASS |
| Dead proxy named, no request spent | `X_TERMINAL_PROXY=http://127.0.0.1:9 doctor` | `proxy refused`, exit 1; `--json` on the same path: valid JSON, `ok:false`, 2 checks, `budget.used=0`, cap 200 | ✓ PASS |
| No proxy + unreachable upstream is "no route" | `unshare -rn env -u … doctor` | `no route to x.com`, `requests: 0 used, 0 limited, cap 200`, exit 1 | ✓ PASS |
| Pin writes at the path the constant builds, and is reported | `doctor --pin HomeTimeline …` then a run | `pinned HomeTimeline to … in <DataDir>/pins.json`; `pins.json` 0600, 182 bytes; next line `107 operations, mined 2026-10-14, 1 pinned` | ✓ PASS |
| `--show-notice` spends nothing | counting socket on 127.0.0.1 + `doctor --show-notice` (and `--show-notice --json`), bare `doctor` as control | 0 dials, exit 0 (statement printed) / 0 dials, exit 2 / control: 2 dials | ✓ PASS |
| Redaction can fail (arm A) | scratch copy: `String()` returns the raw value | `TestSecretCannotLeak` FAILS naming both credentials at offsets 11 and 116 | ✓ PASS |
| Redaction can fail (arm B) | scratch copy: `Format` renamed away | FAILS through the fmt-placeholder arm (leak_test.go:160) and the two offsets (348, 413) | ✓ PASS |
| The verb matrix reaches no value | probe in the scratch copy: 19 verbs, three struct-dump verbs, slog | all yield the redacted literal; only `%p` renders the value, as `%!p(session.Secret=<value>)` (the accepted boundary) | ✓ PASS |
| The id guard can fail (both arms) | scratch copy: an id-shaped literal beside `queryId`; the real Viewer id | arm A FAILS naming the line outside the package; arm B FAILS with `1 file(s) hold an operation id from the registry`; reverted, whole-tree `diff -r` clean | ✓ PASS |
| The surface guard can fail (both directions) | scratch copy: the `whoami` row deleted; a row planted with a foreign body | FAILS at `cli_test.go:127` (`Whoami is exported, implemented and unreachable`) and `cli_test.go:140` (`"scratch" … is not a verb body`); reverted, `diff -r` clean | ✓ PASS |
| Loose-mode check is load-bearing | scratch copy: the perm check disabled | `TestLoadRefusesLooseMode` FAILS (`Load accepted <path>`) | ✓ PASS |
| Symlink check is load-bearing | scratch copy: the symlink case deleted | `TestLoadRefusesSymlink` FAILS (`does not mention "symlink"`) | ✓ PASS |
| Env override is load-bearing | scratch copy: both override blocks deleted | `TestEnvironmentOverridesWin` FAILS (three messages, incl. `Source = "<path>", want "environment"`) | ✓ PASS |
| Floor is load-bearing | scratch copy: the floor's sleep disabled | `TestLimiterSpacesRequestsByTheFloor` FAILS (`waits = [], want exactly one`) | ✓ PASS |
| 403-empty classification is load-bearing | scratch copy: 403-empty → `OutcomeEmptyBody` | `TestClassifyKeepsTheTwoEmptyBodiedRefusalsApart` FAILS | ✓ PASS |
| The success coupling is load-bearing (probe only) | scratch copy: `len(resp.prefix) > 0` removed from `netcheck.go:232` | the tree's own tests stay green (arm does not bite); my probe FAILS (`a zero-byte 200 on / produced "upstream reachable"`), and passes on the pristine tree with its control reaching `upstream reachable` | ✓ PASS |
| The pins name is pinned, and the coupling is checked | scratch copy: `volatile.PinsFileName` renamed to `pins-renamed.json` | `TestPinsPathIsTheRegistersName` FAILS naming the rename as a migration and printing `PinsPath = "/tmp/x-data/pins-renamed.json"`; reverted | ✓ PASS |
| Hardcoding the pins name is only half-caught | scratch copy: `PinsPath` hardcodes `"pins.json"` | dropping the reference FAILS the build (`volatile` imported and not used); keeping a dangling `_ = volatile.PinsFileName` keeps every config test green → P3 note | ✓ PASS (noted) |
| The saved-frame predicate is a whitelist (fix round) | scratch copy: widened to also accept `*xsource.ParseError` | `TestAShapeDriftedAnswerIsNotAFactAboutTheFile` FAILS (`a shape-drifted answer … was read as a failure of the mirror; the pane would tell the reader to move a healthy saved.db aside`), plus `feed_test.go:520,523`; reverted | ✓ PASS |
| The saved-frame predicate cannot be total (fix round) | scratch copy: `savedMirrorError` returns `err != nil` | `TestSavedMirrorErrorNamesOnlyTheFile` FAILS naming the spent cap, the missing session and the refused proxy as failures of the mirror; reverted | ✓ PASS |
| The anti-prune guard fails, not skips | scratch copy: `.planning/research/x-ops-snapshot.json` renamed away | `--- FAIL` at `registry_test.go:90` with the explanation, no SKIP line (0 SKIP occurrences); restored | ✓ PASS |
| The three external entries cannot be pruned | scratch copy: `Merge` patched to delete what the mined map omits | `TestAMergeAddsAndUpdatesButNeverDeletes` FAILS (`107 to 0`), `TestAnExternalEntrySurvivesARefresh` FAILS (`107 to 2 entries, want 108`); reverted | ✓ PASS |
| The same three cannot be dropped from the data | scratch copy: the three entries deleted (107→104) | `TestTheExternallySourcedOperationsAreInTheSnapshot` FAILS naming all three; restored | ✓ PASS |
| Scratch tree left pristine | `diff -r -x .git -x '*VERIFICATION.md' /tmp/p1v-2 <repo>` after every experiment | empty — the repository is byte-identical to the copy taken before this pass apart from this report | ✓ PASS |

### Probe Execution

None — the phase declares no probe scripts and no `scripts/*/tests/probe-*.sh` exists in the tree; its live checks are the CLI runs reproduced above, not probes. The two probes in the table above are mine, written for this pass in the `/tmp/p1v-2` copy (`internal/netcheck/zz_probe_reverify_test.go`, `internal/session/zz_probe_verbs_test.go`) and removed again before the final `diff -r`.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| AUTH-01 | 01-01 | 0600 file at the XDG path; looser mode and symlinks refused | ✓ SATISFIED | Truths 1, 2 — re-verified by tests, absence arms and the built binary |
| AUTH-02 | 01-01 | env override for path and cookie pair, precedence over file | ✓ SATISFIED | Truth 3 plus the `X_TERMINAL_SESSION` run; `TestSessionPathPrecedence` and the proxy-precedence tests pass |
| AUTH-03 | 01-03 | See which account the loaded session belongs to (`whoami`: handle and session status) | ✓ SATISFIED (restored by the earlier audit fix, re-adjudicated this pass) | Truth 10: the row is in `commands()`, `--help` lists it, `Main` dispatches it, and the surface guard makes a missing row a test failure in both directions |
| AUTH-04 | 01-01, 01-02 | one command deletes the local copy; the cookie is never written elsewhere | ✓ SATISFIED | `cli/session.go` refuses symlinks/non-regular files, deletes exactly the resolved path, reports "no session file" with exit 0 — re-run both ways this pass; the only other writers are the notice marker, the registry/pins and `spike -record` bodies (the audit's `persistence` check lists all six destinations) |
| NET-01 | 01-02 | proxy from config or HTTP(S)_PROXY/ALL_PROXY/NO_PROXY over HTTP(S) and SOCKS5 | ✓ SATISFIED | `proxy.go` accepts http/https/socks5/socks5h; `NO_PROXY` matched in-package; `X_TERMINAL_PROXY` and `ALL_PROXY` precedence re-verified in code and tests. The SOCKS5 leg itself was not re-run live |
| NET-02 | 01-02, 01-03 | one named connectivity outcome per failure, never an expired session for a dead proxy | ✓ SATISFIED | Truths 5, 6, 7, re-derived offline through the built binary and the httptest probe |
| NET-03 | 01-02 | fail loudly with that named state, never an empty list | ✓ SATISFIED | `doctor` exits non-zero on any failed stage; the parsers refuse empty payloads; `whoami` prints nothing on stdout when a step fails |
| XAPI-01 | 01-03 | read the Following timeline from the internal GraphQL with the session cookie, no API key | ✓ SATISFIED (live read not repeated) | The registry half re-verified (107 ops, the three external entries present, marked and inside the guard). The read itself is recorded in `.planning/research/UPSTREAM-PROBE.md` (`HomeLatestTimeline`, 200, 957,866 bytes, 157 posts); live calls were out of scope this pass |
| XAPI-02 | 01-03 | operation ids are data; no `queryId` literal elsewhere | ✓ SATISFIED | Truth 12, guard falsified both ways |
| XAPI-03 | 01-03 | pinned authoritative snapshot; refresh may only add or update | ✓ SATISFIED (refresh not re-run) | 107 operations with provenance; `Merge` leaves pins alone and never deletes (falsified both ways); the anti-prune guard fails rather than skipping when its input is gone; `--from-file`/`--live` refresh against the real bundle not re-run — it rewrites the checked-in file |
| XAPI-05 | 01-02, 01-03 | signer isolated behind one interface, broken signer its own error class | ✓ SATISFIED (with note) | `Signer`/`NoopSigner` seam with the measurement documented, `OutcomeSignerOrAuth` as its own class, `APIError` carrying it; the "startup self-test" remains the doctor's live read rather than a signer-specific probe |
| XAPI-06 | 01-02 | 404-empty, 422, 200-with-errors[], 429, load-shed as distinct outcomes | ✓ SATISFIED | 13 outcomes; truth 9's arm re-falsified; the table is unchanged in the branches the phase promised |
| XAPI-07 | 01-02 | one choke point: single-flight, >=1.5s floor, no 429 retry, session cap | ✓ SATISFIED | Truth 8 plus the wiring table; `limiter.go` is the only waiter (audit `timers` names it as the single exemption), cap trips at exactly 200 (`TestLimiterStopsAtExactlyTheCap`), 429 never retried |
| SEC-01 | 01-01, 01-02 | the cookie never appears in logs, errors, doctor output, request dumps or crash output | ✓ SATISFIED | Truth 4: two falsified arms, my 19-verb matrix, four `Reveal()` call sites all `Header.Set`, `internal/cli/leak_test.go`'s producer sweep, and no credential-shaped run in the re-run `doctor --json`. One documented residual: `%p` (Human item 2) |
| OPS-05 | 01-03 | account-risk statement shown on first run and recorded | ✓ SATISFIED (trigger not re-run live) | Truth 14 |

**Coverage:** 15/15 satisfied

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/cli/cli.go` | 79-85 | A bare invocation opens the feed instead of printing usage (exit 1 without a session) | ℹ️ Info | A deliberate later-phase change (FEED-01), covered by `TestABareInvocationOpensTheFeed`; it does not touch a phase-1 truth, but this report's earlier note that a bare invocation prints usage and exits 2 is stale |
| `internal/cli/whoami.go` | 74-81 | A transport failure prints the raw `xweb: Viewer: Get "https://x.com/i/api/graphql/<id>/Viewer?features=…"` string rather than a named network state | ℹ️ Info | Re-observed this pass on the dead-proxy run. Not a credential leak (the URL carries no cookie — checked) and not a phase-1 truth — the verb's own contract is "a session that is missing, unreadable or rejected is a named outcome", and the network path's named states belong to `doctor`. Worth the same `feedReach`-style treatment later; recorded as history rather than as a defect, and it is pre-existing, not introduced by the `whoami` row's restoration |
| `go.mod` | 5-27 | Six direct and seventeen indirect third-party requires, plus `go.sum` | ℹ️ Info | Supersedes plan 01's "stdlib only" clause with the project's planned stack; no phase-1 file gained a third-party import |
| `internal/config/config.go` | 183 | `PinsPath` builds the name from `volatile.PinsFileName`, so `internal/config` now depends on `internal/xweb/volatile` | ℹ️ Info | A deliberate fix-round change, argued in the file's own comment and one in-module hop deep (`volatile` imports nothing in-module, so there is no cycle and no layering inversion; the audit's `boundaries` check passes). Test strength, not the dependency, is the weak spot — see the P3 note below |
| `internal/session/session.go` / `leak_test.go` | 66-73 / 127-133 | Documented `%p` boundary: `fmt.Sprintf("%p", secret)` renders `%!p(session.Secret=<the value>)` | ⚠️ Warning | Re-measured on this tree (19 verbs, a struct dump and slog all yield the redacted literal; `%p` unchanged) and accepted by the orchestrator on 2026-10-14; no code path uses `%p` |

**Anti-patterns:** 5 found (0 blockers, 1 warning, 4 info). No must-have truth failed.

### Human Verification Required

#### 1. Re-run the live x.com leg, or accept the phase-1 record
**Test:** `X_TERMINAL_PROXY=<proxy> x-terminal doctor` (expect `authenticated read OK` plus `API read OK … @handle`, exit 0), `x-terminal whoami` (expect `@handle — name`), and the spike read (expect the viewer line plus its post lines, exit 0).
**Expected:** the named states and the account name as recorded on 2026-10-14.
**Why human:** live calls were out of scope for this pass, so rows #5-#7 rest on their offline arms plus the phase-1 live record, row #11's live half and row #14's live trigger are deferred here, and the restored `whoami` verb's success line has never been observed live in this tree.

#### 2. The `%p` residual — still accepted, still recorded
**Resolution:** Accepted by the orchestrator on 2026-10-14 as a documented, measured limitation; re-measured on this tree and unchanged (19 verbs, a struct dump and slog all yield the redacted literal; `%p` is intercepted by fmt before method lookup, so no type-level fix exists while `Secret` keeps its string representation). No caller in the tree uses `%p`.

#### 3. The Following-vs-UserTweets reading of the phase's read
**Resolution:** Closed by the orchestrator on 2026-10-14 with new evidence: `HomeTimeline`/`HomeLatestTimeline`/`Bookmarks` were taken from an external pinned source and registered (`registry.json`, 107 entries), and `HomeLatestTimeline` was read live (200, 957,866 bytes, 157 posts). This pass re-verified the registry half (the three entries present, marked external, inside the id guard, and still unprunable by a merge) but did not repeat the live read.

## Gaps Summary

**No gaps found. Phase goal achieved.** The one blocking gap this report has ever recorded (`whoami` unreachable) was closed by the earlier audit fix and stays closed: the row is in `internal/cli/cli.go`'s table, `Main` dispatches the verb, `--help` lists it, and `internal/cli/cli_test.go` makes both failure directions a test failure — re-falsified in both directions this pass. The fix round that made this report stale touched three covered files and changed no truth's state.

### Non-Critical Gaps (Can Defer)

1. **Row 11 (the live spike read) is deferred, not verified** — the artifact is present and wired, but its live claim needs a working proxy; it is carried in `deferred:` and Human item 1.
2. **`go.mod`'s "no third-party requires" clause** — superseded by the planned stack in later phases; recorded as history rather than as a defect.
3. **This report's earlier "bare invocation prints usage, exit 2"** — superseded by FEED-01; the current behaviour is covered by its own test.
4. **`whoami`'s transport failure prints a raw URL-shaped error** — see anti-patterns; no phase-1 truth covers it and no credential is involved.
5. **P3 — the pins-name coupling is pinned by a literal, not by an arm (new this pass).** `TestPinsPathIsTheRegistersName` asserts `volatile.PinsFileName != "pins.json"` and `PinsPath() == "/tmp/x-data/pins.json"`. Renaming the constant fails it (arm 16, good — that is the operator-migration alarm), but a `PinsPath` that hardcodes `"pins.json"` while keeping a dangling `_ = volatile.PinsFileName` reference passes every config test (arm 17b): the test would not catch the two names drifting apart, which is exactly the failure the fix round's comment claims is now impossible. The code as written is correct (`config.go:183` uses the constant), so this is a test-strength note, not a defect; hardening it means asserting the *derivation* (e.g. a test that renames the constant in a scratch copy, or a source scan) rather than the literal.
6. **P3 — the success coupling at `netcheck.go:232` has no tree test (recorded before, re-confirmed this pass).** Removing `&& len(resp.prefix) > 0` leaves every test in `internal/netcheck` green; only my probe (`internal/netcheck/zz_probe_reverify_test.go`, not committed) fails. Truth 7 itself is still covered by tree tests; this is the narrower clause that a zero-byte 200 is not a page.
7. **P3 — the `PinsPath` hardcode arm fails to compile rather than failing a test** (arm 17: `"internal/xweb/volatile" imported and not used`). That is an incidental guard — it only holds while the constant is the sole reference in the file — and note 5 is its real statement.

## Recommended Fix Plans

None for this phase. The earlier pass's recommendation (restore the `whoami` verb and pin the CLI surface) is implemented and stays verified; the notes above name their own one-line hardening or are history, and none of them blocks the milestone.

## Verification Metadata

**Verification approach:** Goal-backward re-adjudication of all 14 plan truths and the roadmap success criteria against the frozen tree, with every falsification the four earlier reports recorded re-run in a scratch copy (`/tmp/p1v-2`): 19 arms (the secret-formatting arms, the id guard's two arms, the registry/snapshot integrity arms, the mode/symlink/env-override arms, the limiter floor, the two-empty-bodied-refusals arm, the anti-prune guard, the surface guard's two directions, the pins-name rename, the pins hardcode, the saved-frame predicate's two directions) plus two probes of my own and 12 offline CLI runs.
**Must-haves source:** PLAN frontmatter of 01-01, 01-02 and 01-03 (truths, artifacts, key_links), plus the ROADMAP success criteria and REQUIREMENTS.md for coverage.
**Automated checks:** 13/14 truths verified, 0 failed, 0 present-behavior-unverified, 1 deferred. This pass: 1579 test cases (699 `func Test` + 880 subtests) in 22 packages, 0 failures, online and again inside `unshare -rn` with only loopback up; the phase's own packages again on their own (436 tests, 9 packages); `go build ./...`, `go vet ./...` and `x-terminal audit` clean; 19 falsification/absence experiments, each making a named test, the build or my probe fail as required and each reverted (verified with the whole-tree `diff -r`); 2 purpose-written probes (the zero-byte page + its control, and the verb matrix); 12 offline CLI runs including the counting-socket measurements.
**Covered files added this pass:** none — the 49-path set is unchanged, and the digest moved only because `internal/config/config.go`, `internal/config/config_test.go` and `internal/xweb/volatile/pins.go` changed. `internal/cli/feed.go`, `internal/keys/keys.go` and `internal/domain/page.go` (the fix round's other three files) were considered and judged not to belong, each with its reason stated in the Digest paragraph.
**Live checks:** none — live x.com calls were out of scope, and no account request was spent. Everything above ran offline or against sockets on 127.0.0.1 written for the purpose. No credential was read, printed or copied, and the operator's real session directory was never opened for reading or writing. The phase-1 record's live runs (3 CLI runs through `http://127.0.0.1:7897`, 6 account requests) are cited, not repeated.
**Human checks required:** 1 open item (re-run the live leg, or accept the phase-1 record); the items earlier passes raised remain closed and are recorded with their resolutions.
**Tree state:** the only file this pass writes inside the repository is this report. Every experiment ran in `/tmp/p1v-2`, a full copy, and after the last one `diff -r -x .git -x '*VERIFICATION.md'` between the repository and the copy was empty.

---

*Verified: 2026-10-15T06:22:11Z*
*Verifier: gsd-verifier (subagent)*

## Re-verification pass (2026-10-15T06:22:11Z)

**Why.** The milestone's narrow fix round changed files this report covers — `internal/config/config.go`, `internal/config/config_test.go` and `internal/xweb/volatile/pins.go` (mtimes 14:05, 14:07 and 14:08 +0800, after this report's previous timestamp) — so recomputing the fingerprint over the previously covered 49 paths gave `v3:sha256:6ae7695c…` against the stored `v3:sha256:74130648…`: the report was stale and the gate could not accept it. The tree was frozen for this pass; the only file written is this report.

**What was re-run.** `go build ./...` and `go vet ./...` (both exit 0); the whole suite online and inside `unshare -rn` with `lo` up (1579 test cases, 22 packages, 0 failures, same 8 skips in both); the phase's packages on their own (436 tests, 9 packages); `x-terminal audit` (10 checks, 0 findings, 893 files); the named tests of all fourteen rows; the fix round's three claims re-read in the code and re-checked end to end (`doctor --pin` writing `pins.json`, the saved-frame predicate's two failure directions, the `ByKeystroke` comment against its callers); and 19 falsification experiments plus two probes, each of which made its target fail — except where the arm itself showed the guard to be missing, which is recorded as a P3 note (the pins hardcode, the netcheck success clause) — and each of which was reverted (whole-tree `diff -r` clean).

**What changed in the verdict.** Nothing: 13/14 truths verified, `passed`, no gap. The earlier failure (`whoami` unreachable) stays closed and was re-adjudicated from the top rather than inherited; the credential path, the proxy wiring, the five named connectivity states, the request discipline, the registry-as-data guard, the pin path and the first-run statement all still hold; every falsification that proved them on 2026-10-14 still fails when the defect is re-planted. Three new P3 notes were recorded (items 5-7 above) and no must-have truth is touched by them.

**Not verified in this pass.** No live x.com request was made. That leaves the live halves of rows #5-#7, row #11's whole claim, the live trigger of row #14 and the restored `whoami`'s success line, the `--live`/`--refresh-ids` refresh against the real bundle (it rewrites the checked-in registry), the `--from-file` merge against the real 1.1 MB bundle, the SOCKS5 leg, `logout` against the real session file (deliberately not run — it deletes the user's credential, and there is no file at that path now), and the `HomeLatestTimeline` read recorded in `.planning/research/UPSTREAM-PROBE.md` (whose registry half was re-verified here). A human needs a working proxy plus the session file restored to close Human item 1.
