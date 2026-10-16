---
phase: 01-reachable-transport-and-one-live-read
plan: 02
subsystem: network-transport
tags: [go, http-transport, proxy, socks5, no-proxy, rate-limit, single-flight, request-budget, status-classifier, doctor, httptest, fmt-formatter, credential-redaction, logout]

# Dependency graph
requires:
  - phase: 01-reachable-transport-and-one-live-read
    provides: "internal/config (proxy precedence incl. ALL_PROXY, NO_PROXY passthrough, RequestFloor, SessionRequestCap, SessionPath), internal/session (Secret/Session/Load/Summary), internal/cli (the command table and exit codes)"
provides:
  - "internal/xweb: NewTransport/NewClient — one proxied http.Transport for the whole program (http, https, socks5, socks5h; NO_PROXY honoured; 10s dial and TLS-handshake timeouts; ForceAttemptHTTP2; redirects left visible)"
  - "internal/xweb.Limiter: the single request choke point (single-flight, 1.5s floor, 200-request session cap, 429 counter, Budget) with an injected clock"
  - "internal/xweb.Outcome + Classify: the measured status table as a typed state with a Remedy sentence"
  - "internal/netcheck: the five-outcome connectivity self-check (route / proxy / upstream / session) with bounded Detail lines and a session stage that is skipped, not failed"
  - "internal/cli: a real doctor (one line per link, session summary, budget line, --json, exit codes) and logout (deletes exactly the session file)"
  - "internal/session.Secret.Format: the %d family of format verbs now yields <redacted>"
affects: [01-03, phase-2, phase-6]

# Actuals — pairs with the plan's estimate to calibrate future estimates.
# chars/4 over the 111,544 bytes written: 13 new files (106,144 bytes) plus ~5,400
# bytes added to internal/cli/cli.go, internal/session/session.go and leak_test.go.
actuals:
  tokens: 27886
  tasks: 4
  commits: 0

# Tech tracking
tech-stack:
  added: []          # stdlib only, as the phase requires
  patterns:
    - "one transport, one client, one choke point: nothing elsewhere in the program opens a socket to x.com"
    - "measured outcome tables: status + body shape + two headers, with a Remedy sentence per state"
    - "injected seams (clock, sleep, resolver, dialer, http.Client) so behaviour is tested without real time or real network"
    - "self-checks that report presence facts only, with every line bounded to 200 bytes"
    - "a residual leak pinned by count rather than ignored"

key-files:
  created:
    - internal/xweb/limiter.go
    - internal/xweb/limiter_test.go
    - internal/xweb/proxy.go
    - internal/xweb/proxy_test.go
    - internal/xweb/transport.go
    - internal/xweb/classify.go
    - internal/xweb/classify_test.go
    - internal/netcheck/netcheck.go
    - internal/netcheck/netcheck_test.go
    - internal/cli/doctor.go
    - internal/cli/doctor_test.go
    - internal/cli/session.go
    - internal/cli/session_test.go
  modified:
    - internal/cli/cli.go
    - internal/session/session.go
    - internal/session/leak_test.go

key-decisions:
  - "Stage 4 reads https://x.com/home with the cookie and nothing else: the endpoint the plan named (/i/api/1.1/account/settings.json) answers 404 with a JSON body — the v1.1 API is retired wholesale — and the GraphQL registry is plan 03's artifact, so the one authenticated read available here is the navigation one"
  - "With a proxy configured, stage 1 resolves x.com and leaves the connect to stage 2; a direct connect to x.com:443 succeeds on this host (210ms) while the TLS/HTTP layer then hangs, so probing it would report a failure the program never meets"
  - "NO_PROXY matching is implemented in xweb: config.Load hands the proxy over directly, so net/http's own NO_PROXY handling (which only runs on the ProxyFromEnvironment path) would never see it"
  - "Secret.Format covers every verb fmt consults — v, +v, #v, s, q, d, x, X, c, U, t, b, o, f, e — but %p cannot be covered by any type: fmt handles it before method lookup. The leak test removes exactly that placeholder, counts it, and fails on anything else"
  - "The command table in internal/cli/cli.go became a function: as a package-level initializer it was a cycle (commands -> Doctor -> usage -> commands) that Go rejects at compile time"
  - "The doctor's session stage reports 'session not configured' and exits 0: a missing cookie is a state, not a failure of the network path, and only a session file that is there and unusable exits non-zero"
  - "XAPI-06 names 422 as its own outcome and the plan's table has no row for it, so OutcomeBadRequest exists for 400/422 rather than folding a rejected request shape into 'unauthorized'"
  - "A body with no declared Content-Length is described as 'at least N bytes (the body declared no length)' — the first live run showed the fetch clamping the unknown length to the 4096-byte peek, which printed the peek as if it were the body"

patterns-established:
  - "Every upstream request passes xweb.Client.Do, so Limiter.Budget() is the truth about what a session spent — the doctor prints it from the same limiter the check used"
  - "Classify is the only place a status becomes a name, and its Remedy sentence is the user-facing answer: no state exists that means 'no posts'"
  - "Diagnostic Detail lines are clipped to 200 bytes on a rune boundary, in both the terminal report and the JSON one"
  - "Refusals and details never echo the thing they are about: the proxy setting, the session path's contents, the body of an API error"
  - "Tests point the same code at an httptest.Server through injected seams; the whole suite passes inside a network namespace with no external route"

requirements-completed: [NET-01, NET-02, NET-03, XAPI-06, XAPI-07, AUTH-04]

# Coverage metadata — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "One proxied transport for every request: http/https/socks5/socks5h proxy settings, NO_PROXY honoured, 10s dial and TLS-handshake timeouts, ForceAttemptHTTP2, and redirects left visible"
    requirement: NET-01
    verification:
      - kind: unit
        ref: "internal/xweb/proxy_test.go#TestNewTransportAcceptsBothMeasuredProxySchemes, #TestNewTransportRefusesAProxyItCannotUseWithoutEchoingIt, #TestNewTransportCarriesTheTimeouts, #TestNoProxyExemptsTheHostsItNames"
        status: pass
      - kind: manual_procedural
        ref: "live: X_TERMINAL_PROXY=http://127.0.0.1:7897 /tmp/x-terminal doctor -> exit 0, 'authenticated read OK'; and with socks5h://127.0.0.1:7897 -> exit 0, the same two stages"
        status: pass
    human_judgment: false
  - id: D2
    description: "One request choke point: single-flight (a second concurrent call waits), a 1.5s inter-request floor, a 200-request session cap returning typed ErrRequestCap, a 429 counter, and Budget()"
    requirement: XAPI-07
    verification:
      - kind: unit
        ref: "internal/xweb/limiter_test.go#TestLimiterSpacesRequestsByTheFloor, #TestLimiterStopsAtExactlyTheCap, #TestLimiterMakesTheSecondConcurrentCallWait, #TestLimiterCountsRateLimitedAnswersAndNeverRetriesThem"
        status: pass
    human_judgment: false
  - id: D3
    description: "The measured status table as a typed outcome with Error and Remedy: 401, 403-with-empty-body, 404-empty vs 404-with-JSON, 429, soft ban, load shed, bad request, empty body, edge interstitial, expired-session redirect — and no outcome that means 'no posts'"
    requirement: XAPI-06
    verification:
      - kind: unit
        ref: "internal/xweb/classify_test.go#TestClassifyTheMeasuredTable (22 cases, every row of the plan's table), #TestClassifyKeepsTheTwoEmptyBodiedRefusalsApart, #TestClassifyEmptyBodyIsNeverASuccess (statuses 100-599), #TestClassifyNeverAdvisesARetryFor429, #TestEveryOutcomeExplainsItself"
        status: pass
    human_judgment: false
  - id: D4
    description: "A five-outcome connectivity self-check — no route to x.com, proxy refused, proxy reached but upstream blocked, TLS/Cloudflare challenge, authenticated read OK — that stops at the first broken link and skips the session stage when there is no session"
    requirement: NET-02
    verification:
      - kind: unit
        ref: "internal/netcheck/netcheck_test.go#TestCheckReportsAuthenticatedReadOK (anti-vacuity: exactly / then /home were requested, cookie on the authenticated read only, no auth header anywhere, Elapsed > 0, budget = 2), #TestCheckNamesNoRoute (both arms), #TestCheckNamesProxyRefused, #TestCheckNamesUpstreamBlocked, #TestCheckNamesTheCloudflareChallenge, #TestCheckNamesARejectedSession, #TestCheckSkipsTheSessionStageWithoutOne, #TestCheckReportsAnUnusableProxySetting, #TestClipBoundsEveryLine, #TestSizeDoesNotPassThePeekOffAsTheBody, #TestCheckReportsABodyWithoutADeclaredLengthWithoutInventingOne, #TestCheckSpendsExactlyTwoRequestsOnASuccessfulRun"
        status: pass
      - kind: manual_procedural
        ref: "live: proxy up -> exit 0 naming 'authenticated read OK'; X_TERMINAL_PROXY=http://127.0.0.1:9 -> 'proxy refused', exit 1, 0 requests spent; no proxy at all -> 'no route to x.com', exit 1 (30s client timeout)"
        status: pass
    human_judgment: false
  - id: D5
    description: "A real doctor: one line per link with a named outcome, the session summary from session.Summary() (lengths and path only), the budget line 'requests: N used, M limited, cap 200', --json that parses with every message bounded, non-zero exit when a link failed, and the two plan-03 flags recognised but unwired"
    requirement: NET-03
    verification:
      - kind: manual_procedural
        ref: "live: doctor --json parsed with python3 json (top-level config/session/checks/budget/ok); longest string value 119 chars; no value >200 that is not a path or URL; no base64url-shaped run of 30+ chars; the live cookie's first 8 characters appear nowhere in the report or the binary"
        status: pass
      - kind: unit
        ref: "internal/cli/doctor_test.go#TestDoctorRefersTheUnwiredFlagsToPlan03, #TestDoctorRejectsWhatItCannotParse"
        status: pass
      - kind: manual_procedural
        ref: "live: doctor --refresh-ids -> 'not implemented yet', exit 2; doctor --pin HomeTimeline <id> -> 'not implemented yet', exit 2"
        status: pass
    human_judgment: false
  - id: D6
    description: "logout deletes exactly the one session file: it announces the path and the session's state first, refuses a symlink and a non-regular file, never touches the directory or a bystander file, prints 'no session file at <path>' with exit 0 when there is nothing to delete, and prints no cookie"
    requirement: AUTH-04
    verification:
      - kind: unit
        ref: "internal/cli/session_test.go#TestLogoutDeletesTheSessionFileAndNothingElse, #TestLogoutRefusesASymlink (real symlink in a temp dir), #TestLogoutOnAMissingFileIsNotAFailure, #TestLogoutRefusesADirectory, #TestLogoutTakesNoArguments"
        status: pass
      - kind: manual_procedural
        ref: "live against the real path: run 1 printed 'about to delete /home/user/.config/x-terminal/session' + 'session: present (auth_token 40 bytes, ct0 160 bytes, source ...)' + 'deleted ...', exit 0, file gone, directory intact; run 2 printed 'no session file at ...', exit 0; the file was then restored byte-for-byte (sha256 match, mode 600, 217 bytes) and doctor ran green against it"
        status: pass
    human_judgment: false
  - id: D7
    description: "Secret.Format closes the format-verb leak: %d, %x, %c, %U (and every other verb fmt consults) yield <redacted>, v/s/q still delegate to String so the plan-01 revert check keeps working"
    requirement: ""
    verification:
      - kind: unit
        ref: "internal/session/leak_test.go#TestSecretCannotLeak with the sink list extended by %d, %x, %c, %U, %p (+ %t, %b, %o), four anti-vacuity arms intact, a new arm asserting no other verb falls through to a fmt placeholder, and the one %p placeholder removed by count"
        status: pass
      - kind: manual_procedural
        ref: "revert check re-run after the change: String() temporarily returning string(s) made TestSecretCannotLeak fail reporting both credentials at offsets 11 and 116; reverted, test green again"
        status: pass
    human_judgment: true
    rationale: "The plan's acceptance asks that %p contain no raw value. Measurement (fmt's own source and a probe over five representations) shows fmt handles %p before it looks for a method, so %p renders %!p(session.Secret=<value>) for any string-kind type and no method of this package can intercept it; the test now pins that single residual by count instead. Accepting the boundary — or asking for a non-string representation, which would cost len/==/conversion — is a human call, and the same applies to %p on a credential being an unusual verb to begin with."

# Metrics
duration: ~20min
completed: 2026-10-14
status: complete
---

# Phase 1: Reachable Transport and One Live Read Summary

**One proxied transport, one request choke point and one measured status table behind a `doctor` that names exactly which link of the path is broken — verified live over both the HTTP and the SOCKS5 form of this host's proxy, and over a dead port and no proxy at all, with the credential package's last formatting hole closed and a `logout` that deletes exactly one file.**

## Performance

- **Duration:** ~20 min total (two live probe passes settled the design first; the first file was written at 13:34:54 and the last verification finished at 13:46:06)
- **Started:** 2026-10-14T13:26+08:00 (approximate — the first measurement request)
- **Completed:** 2026-10-14T13:46:06+08:00
- **Tasks:** 4
- **Files modified:** 16 (13 created, 3 modified)

## Accomplishments

- **The network path is one object.** `internal/xweb.NewTransport` builds the single `http.Transport` from the *resolved configuration* (not from the environment), accepting `http`, `https`, `socks5` and `socks5h` settings and handing the URL to net/http rather than writing a dialer; `NO_PROXY` is matched here because `config.Load` hands the proxy over directly and net/http's own matcher only runs on the `ProxyFromEnvironment` path. Ten-second dial and handshake timeouts, `ForceAttemptHTTP2`, and redirects deliberately left visible (`NoRedirect`: an expired session answers 307 into the login flow, and following it would bury the expiry under the login page's 200).
- **Every request goes through one choke point.** `xweb.Limiter` enforces single-flight (a second call waits, and can give up with its context), a 1.5s inter-request floor, a 200-request session cap (`ErrRequestCap`), and never retries a 429. Its clock and its wait are injected, so the tests exercise the floor, the cap boundary (200 succeeds, 201 does not) and the concurrency case without real seconds; `limiter.go` is the only file in the repository that waits between requests.
- **The classifier is the measured table, not a guess.** `Classify(status, headers, body)` returns one of thirteen named states with a `Remedy` sentence each: 403-with-empty-body (signer/auth) and 404-with-empty-body (unknown operation) are different outcomes, 404-with-JSON is a retired endpoint, 429 is a rate limit whose remedy never says "retry", `errors[].code == 88` splits into a soft ban or a rate limit by `x-rate-limit-remaining`, and a zero-byte body is `OutcomeEmptyBody` at *every* status from 100 to 599 — there is no outcome meaning "no posts".
- **`doctor` names the broken link, live.** Four stages (route, proxy, upstream, session), one line each with a named outcome, the session summary from `session.Summary()` (lengths and path only), and a closing `requests: N used, M limited, cap 200` line read from the limiter the check itself spent on. Live: proxy up → exit 0 naming `authenticated read OK`; `X_TERMINAL_PROXY=http://127.0.0.1:9` → `proxy refused`, exit 1, zero requests spent; no proxy at all → `no route to x.com`, exit 1. `--json` emits the same data, parses cleanly, and carries nothing longer than a sentence.
- **`logout` deletes exactly one file.** It announces the path and the session's state (lengths only) before deleting, refuses a symlink and anything that is not a regular file, leaves the directory and bystanders alone, prints `no session file at <path>` with exit 0 when there is nothing to do, and never prints the cookie. Verified live against the real session file, which was then restored byte-for-byte (sha256 match, mode 600).
- **The credential package's formatting hole is closed.** `Secret.Format` answers every verb fmt consults — `%d`, `%x`, `%c`, `%U` and the rest — with `<redacted>`, while `v`, `s` and `q` still delegate to `String` (flags and width included), so the plan-01 revert check still fails when `String()` is broken: re-run and watched, both credentials reported at the same offsets (11 and 116), then reverted.

## Task Commits

**None made by this executor.** The orchestrator commits this plan's files; no git command was run and no hashes are claimed.

## Files Created/Modified

- `internal/xweb/limiter.go` — `ErrRequestCap`, `Limiter` with injected clock and sleep, `Do` (single-flight → cap → floor → issue → count), `Budget`.
- `internal/xweb/limiter_test.go` — floor spacing with a fake clock (including the "a later request does not wait" arm), the cap at exactly 200, the concurrent second call waiting, and the 429 counter with no retry.
- `internal/xweb/proxy.go` — `NewTransport`, `ProxyAddress`, proxy-scheme validation that never echoes the setting, the `NO_PROXY` matcher (`*`, domain, host:port, literal, CIDR).
- `internal/xweb/proxy_test.go` — the four measured schemes, refusals that leak no credential, the timeouts, and eleven `NO_PROXY` rows.
- `internal/xweb/transport.go` — the package doc (the measured mechanics), `BaseURL`/`APIBaseURL`/`UserAgent`, `Client` with `Do` and `ProxyLine`, `NewClient`, `NoRedirect`.
- `internal/xweb/classify.go` — `Outcome` with `Error`/`Remedy`/`Success`, `Classify`, and the small parsers behind it (`parseErrors`, `rateLimitRemaining`, `looksLikeInterstitial`, `expiredSessionRedirect`, load-shed codes and messages).
- `internal/xweb/classify_test.go` — the 22-case measured table plus the four structural rules (the two empty-bodied refusals apart, empty body never a success, 429 never advises a retry, every outcome explains itself).
- `internal/netcheck/netcheck.go` — `Outcome` (the five promised names), `Result`, `Target`/`NewTarget`/`Client`, `Check`/`CheckWith`, the four stages, `fetch` (4KB peek, no invented length) and `clip`.
- `internal/netcheck/netcheck_test.go` — every outcome driven through an `httptest` server, the anti-vacuity arms (exactly two requests, both for `x.com`, cookie only on the authenticated one, non-zero elapsed), and the two size-wording cases.
- `internal/cli/doctor.go` — the verb: flag handling, config and session load, the check, and the text and JSON reports with bounded messages.
- `internal/cli/doctor_test.go` — the unwired flags return before anything runs; a bad invocation prints the usage.
- `internal/cli/session.go` — the `logout` verb.
- `internal/cli/session_test.go` — delete-and-nothing-else (with a bystander file and the directory checked), symlink refusal, missing file, directory refusal, no arguments.
- `internal/cli/cli.go` — the table now registers `doctor` and `logout`, and is a function rather than a variable (see decisions).
- `internal/session/session.go` — `Secret.Format` and `formatVerb`, plus the doc that records the `%p` boundary.
- `internal/session/leak_test.go` — the verb sinks (`%d`, `%x`, `%c`, `%U`, `%p`, `%t`, `%b`, `%o`), the `%p`-by-count removal, and the new anti-vacuity arm against fmt's placeholder.

## Decisions Made

1. **Stage 4 reads `https://x.com/home` with the cookie and nothing else.** The plan named `/1.1/account/settings.json`; measured on 2026-10-14 it answers `404` with `{"errors":[{"message":"Sorry, that page does not exist","code":34}]}` — the v1.1 REST API is retired wholesale (`verify_credentials.json` too), and the registry the plan offers as the alternative is plan 03's artifact. The navigation read is the one authenticated read that exists here, and it *is* discriminating: with a dead cookie the same request answers `307`, zero bytes, `location: /i/jf/onboarding/web?redirect_after_login=%2Fhome&mode=login`, and `set-cookie` expiring `auth_token` and `ct0`. The header set is cookie-only because the measured navigation class rejects a bearer with `401` and an empty body.
2. **With a proxy configured, stage 1 resolves the name and leaves the connect to stage 2.** Measured live: a direct TCP connect to `x.com:443` succeeds in 210ms on this host and the TLS/HTTP layer then hangs for the full 30s client timeout. Probing that path would report a failure the program never meets, so the first hop is the proxy — which is stage 2's job, and its own outcome.
3. **`NO_PROXY` is matched in `xweb`, not delegated.** `config.Load` resolves the proxy itself, so net/http's `NO_PROXY` handling (which only runs on the `ProxyFromEnvironment` path) would never see it; ignoring a list the user wrote would be the silent wrongness this phase exists to remove. The matcher follows the standard library's rules and is covered by eleven cases.
4. **`Secret.Format` covers every verb fmt consults; `%p` is out of reach of any type.** fmt handles `%T` and `%p` before method lookup ("we always do them first"), so `%p` on a string-kind value renders `%!p(session.Secret=<the value>)`; measured over five representations (string, struct-with-string, slice, func, pointer) — only pointer-shaped kinds print an address. The leak test therefore removes exactly that placeholder and counts it (one per credential), then scans the rest, and the type's doc comment records the boundary. `internal/session/session.go:54-73` carries the reasoning.
5. **`internal/cli/cli.go` had to change, and its table had to become a function.** The plan's file list omits `cli.go`, but Task 3 says to replace the stub and Task 4 says to register the verb in the command table, which lives there. As a package-level slice the table is an initialization cycle (`commands` → `Doctor` → `usage` → `commands`) that Go refuses to compile; it is now `func commands()` with the doc comment saying why.
6. **`OutcomeBadRequest` exists for 400/422.** XAPI-06 names `422` as a state of its own and the plan's table has no row for it; folding a rejected request shape into "unauthorized" would lose the distinction the requirement is about.
7. **The session stage being skipped is not a failure.** `doctor` exits 0 when the network path is proven and there is simply no cookie, and prints "no session cookie is configured ... the authenticated read was not attempted". A session file that is *there and unusable* (a symlink, a loose mode) is a failure and exits 1.
8. **A body with no declared length is described as such.** The first live run printed the 4096-byte peek as the body's size, because `fetch` clamped an unknown `Content-Length` to what it had read; `fetch` now passes `resp.ContentLength` through and two tests (a unit case for both branches, one end-to-end chunked response) keep it honest.
9. **Every message in the JSON report is clipped to 200 bytes**, on a rune boundary, with the path, URL and session-summary fields left intact — the plan's acceptance makes length itself the proxy for "no token material".
10. **The report's first hop comes from the client, not from the config string.** `configLine` and the JSON `proxy` field print `Client.ProxyLine()` — the URL the transport actually resolved — so a line cannot describe a setting the client did not use; the redacted config value is the fallback for the case where the path could not be built at all. Live, this is what makes the SOCKS5 run legible as one.

## Deviations from Plan

1. **`internal/xweb/proxy_test.go` was added** (the plan's file list names only `limiter_test.go` for Task 1). The `NO_PROXY` matcher and the scheme validation are new code in `proxy.go`; leaving them untested would have been the one untested thing in the credential-adjacent path.
2. **`internal/cli/doctor_test.go` and `internal/cli/session_test.go` were added.** Task 4's acceptance requires logout tests ("covered by a test with a real symlink in a temp dir") and the plan lists no test file under `internal/cli` at all; the doctor test pins that the unwired flags return *before* any request is made.
3. **`internal/cli/cli.go` was edited** — see decision 5.
4. **`%p` is the one plan acceptance item not met as written.** The plan asks for a test asserting `%p` contains no raw value; no type can deliver that (decision 4). The test pins the residual by count instead, and the coverage entry D7 is flagged for human judgment rather than auto-passed.

## Issues Encountered

- **The live run caught a lie in my own code.** `netcheck.fetch` clamped an unknown `Content-Length` to the bytes it had read, so the doctor printed "answered 200 with 4096 bytes" for a gzip'd 300KB response. Fixed by passing the declared length through unchanged and describing the unknown case as "at least N bytes (the body declared no length)"; the end-to-end chunked test that first detected it was kept alongside a unit test of both wording branches.
- **The command table was an initialization cycle**, found by the compiler when the stubs were replaced by real bodies; resolved by making it a function (decision 5).
- **An `httptest` server cannot be made to answer without a `Content-Length` on demand** — the first attempt at the chunked test passed for the wrong reason. The test now covers both shapes explicitly.
- **The 1.5s floor is not exercised with real time anywhere**: the only wall-clock wait in the suite is a 50ms observation in the single-flight test, where it detects that a second call *has not* run, not that a floor has passed.

## User Setup Required

None. The proxy is reached through `X_TERMINAL_PROXY` (nothing exports `*_PROXY` in a non-interactive shell, and no `config.toml` exists), and the session file was left exactly as it was found: 217 bytes, mode 600, sha256-identical after the live logout check.

## Next Phase Readiness

- **Plan 01-03 can build straight on top.** `xweb.Client.Do` is the only request path, so a GraphQL call added to `internal/xweb/graphql.go` inherits the proxy, the limiter and the budget line for free; `Classify` is the only place a status becomes a name, and its `Remedy` is the sentence the UI will print; `Limiter.Budget()` is what `doctor` already reports.
- **The two `doctor` flags plan 03 owns are wired to say so**: `--refresh-ids` and `--pin <operation> <queryId>` are recognised, print `not implemented yet` and exit 2 — they return before `config.Load`, so a placeholder can never spend a request.
- **The stage-4 read should be replaced by a registry-driven GraphQL call** when the registry lands: the navigation read proves the cookie is accepted, while an authenticated API read would prove the API path itself. The measurement that forces this (the retired v1.1 endpoints) is recorded in the code next to the stage.
- **The `%p` boundary is documented, pinned and reported** — see decision 4 and coverage D7.
- **Verified evidence, all green:** `go build ./...` exit 0; `go vet ./...` exit 0; `go test ./... -count=1` 129 tests in 6 packages, exit 0; `gofmt -l internal/ cmd/` empty; the whole suite also passes inside `unshare -rn` with only loopback up, which is the proof that no test touches the real network; live `doctor` exit 0 naming `authenticated read OK` over `http://127.0.0.1:7897` and over `socks5h://127.0.0.1:7897`; live `doctor` naming `proxy refused` (exit 1, zero requests spent) and `no route to x.com` (exit 1); live `doctor --json` parsed and screened (longest value 119 characters, no token-shaped runs, the live cookie's first 8 characters absent from both the report and the binary).

---
*Phase: 01-reachable-transport-and-one-live-read*
*Completed: 2026-10-14*
