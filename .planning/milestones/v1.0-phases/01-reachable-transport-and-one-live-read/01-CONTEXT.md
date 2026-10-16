# Phase 1: Reachable Transport and One Live Read - Context

**Gathered:** 2026-10-14
**Status:** Ready for planning
**Mode:** Smart discuss (autonomous) — proposals accepted in batch, no per-area interruption

<domain>
## Phase Boundary

The client reaches x.com through the user's proxy and completes one real authenticated read,
and every way that can fail is a named, distinct state rather than an empty timeline.

Delivers: the Go module and package skeleton, config handling, session-cookie handling
(import, storage, redaction), a proxied HTTP transport, a named connectivity self-check, the
response classifier, the request-discipline choke point, the volatile operation registry
(operation ids and their feature switches as data), a GraphQL read path proven against the
live site, and two runnable subcommands — `doctor` and `whoami` — plus the first-run account
risk statement.

Does NOT deliver: any TUI screen, any rendering, the timeline list, media, or the store.
Those are later phases; this one exists to make "the upstream answers from this host" true
and to capture the real response shapes that later phases parse.
</domain>

<decisions>
## Implementation Decisions

### Transport and proxy

- Proxy configuration comes from the config file **and** the standard environment variables,
  with environment variables taking precedence; the URL scheme decides HTTP(S) vs SOCKS5, so
  one setting covers both (`http://127.0.0.1:7897` and `socks5h://127.0.0.1:7897` both work on
  this host's Clash Verge mixed port).
- `ALL_PROXY` must be read explicitly: Go's stdlib honours only `HTTP_PROXY`/`HTTPS_PROXY`/
  `NO_PROXY`, and `net/http` natively accepts `socks5`/`socks5h` proxy URLs.
- Connectivity failures are **not retried**. The self-check runs once and reports; retrying a
  dead proxy only burns the account's request budget.
- With no proxy configured the client dials directly — the tool must not be welded to one
  network environment.
- The self-check reports five distinct named outcomes: no route to x.com, proxy refused,
  proxy reached but upstream blocked, TLS/Cloudflare challenge, authenticated read OK. A dead
  proxy must never be reported as an expired session.

### Session and secrets

- The session file is `KEY=VALUE` lines (`auth_token`, `ct0`) at
  `$XDG_CONFIG_HOME/x-terminal/session` — matching the file already created at
  `~/.config/x-terminal/session` on this machine (mode 0600, directory 0700).
- Overrides: `X_TERMINAL_SESSION` (path) and `X_TERMINAL_AUTH_TOKEN` / `X_TERMINAL_CT0`
  (values). Values win over the file.
- A session file with permissions looser than 0600 is **refused**, with the exact `chmod`
  command printed; a symlink is refused outright.
- The cookie is wrapped in a redacting type (`String`, `GoString` and `MarshalJSON` all return
  a placeholder) and a leak test greps captured error/log output for a fixture cookie.

### Upstream access

- The operation registry is a **single JSON file** carrying `queryId`, `operationType` and the
  `featureSwitches`/`fieldToggles` each operation requires — the live bundle ships all four
  together, so the same artifact answers the "features blob drifted / `(336)`" failure mode.
- The checked-in snapshot is authoritative. A runtime refresh may only **add or update**
  entries; it never removes or silently substitutes one, and an unknown operation is a typed
  error rather than a fallback to a stale id.
- A per-operation **manual pin** override exists (`doctor --pin <op> <queryId>`), because the
  bundle cannot reveal lazily-loaded operations such as `HomeTimeline`.
- The transaction-id signer (`x-client-transaction-id`) is **not implemented in this phase**:
  measured on this host, `UserByScreenName` and `UserTweets` both answered `200` without it.
  It stays behind a `Signer` interface with a no-op implementation and a per-operation
  re-test, rather than being designed around up front.
- Request discipline is enforced at one choke point: single-flight, an inter-request floor of
  1.5 s, no retry on `429`, and a session cap of 200 requests.

### Classifier and CLI

- The classifier's status table is built from measurement, not assumption:
  `403` with an empty body = missing or incoherent auth/signer; `404` with an empty body =
  unknown operation or stale query id; `404` **with** a JSON body = retired endpoint
  (`/1.1/account/verify_credentials.json` answered exactly this); `429` = rate limited (read
  the reset header); `5xx`/load-shed = bounded backoff with a distinct message.
- An empty body must never render as "no posts", in any state.
- The spike is a throwaway `cmd/spike` binary: read the session, issue one authenticated
  GraphQL call, print parsed author/text/status. Recorded responses become fixtures under
  `internal/xsource/testdata/`, and the spike is deleted once Phase 2's screens replace it.
- Phase 1 ships two subcommands: `doctor` (proxy reachability, session validity, signer
  self-test, snapshot age, request budget — presence facts only, never a token value) and
  `whoami` (the handle the session belongs to, plus session status).
- The first run prints the account-risk statement once and records that it was shown.
- Go module path: `github.com/LZD121/x-terminal`.

### Claude's Discretion

Layout details inside the packages, test organisation, and the exact wording of the risk
statement are at the implementer's discretion within the decisions above.
</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets

- Nothing exists in this repository yet except `.planning/`. The sibling project
  `/home/user/repos/rss-tm` (Go 1.27 + Bubble Tea v2, 16 `internal/` packages) is a **pattern
  reference only** — no shared code, by explicit decision. Its conventions worth copying:
  a pure `Update` returning effects, an AST import-boundary test with anti-vacuity arms,
  golden frames as the view specification, ordered migrations, and a consumer-side port
  instead of an interface defined by the producer.
- Live evidence already gathered for this phase (read them, do not re-derive):
  `.planning/research/UPSTREAM-PROBE.md` and `.planning/research/x-ops-snapshot.json`
  (104 operations with query ids and required feature switches),
  `.planning/research/evidence/fixture-usertweets.json` (a real `UserTweets` response),
  `.planning/research/SUMMARY.md` (the architecture the roadmap is built on).

### Established Patterns

- Measured request mechanics that the transport must reproduce:
  - An authenticated **navigation** request that carries `Authorization: Bearer` is rejected
    with `401` and an empty body; the same request with only cookies succeeds. The header set
    must match the request class.
  - API requests need `authorization` (web bearer), the cookie pair, `x-csrf-token` mirroring
    `ct0`, `x-twitter-auth-type: OAuth2Session`, `x-twitter-active-user: yes`.
  - The response schema is the **new** one (`core` / `avatar` / `id`); there is no
    `legacy.followers_count`. Parsers that assume `legacy` silently return nothing.
- Environment facts: proxy `127.0.0.1:7897` (Clash Verge mixed port, HTTP and SOCKS5), Go
  1.27.2 installed, no Rust toolchain, `sqlite3` present, terminal Ghostty 1.3.1.

### Integration Points

- `.planning/config.json` — granularity `standard`, parallelization on, verifier and plan
  check enabled.
- The session file already exists on this machine at `~/.config/x-terminal/session` (0600);
  `doctor` should treat it as valid input, not as something to import.
</code_context>

<specifics>
## Specific Ideas

- The `doctor` output should be quotable in a bug report: one line per check, a named outcome
  each, and no token material anywhere.
- `whoami` exists because a wrong-account session is the most confusing failure mode — the
  user must be able to confirm which account the cookie belongs to before blaming the app.
</specifics>

<deferred>
## Deferred Ideas

- A cookie-import helper that reads a browser profile: explicitly out of scope for the product
  (see REQUIREMENTS.md Out of Scope); extraction stays a manual, operator-side step.
- Implementing the transaction-id signer: deferred until a specific operation is measured to
  require it.
- Discovering the following-timeline operation id from the route chunk: a Phase 1 task, but the
  *phase* must not block on it — an external pinned source covers the gap meanwhile.
</deferred>
