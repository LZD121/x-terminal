---
phase: 08-operations-repair-path-and-audit
plan: 01
type: summary
status: complete
requirements: [SEC-03]
subsystem: audit
tags: [audit, sec-03, go-ast, cli]
requires: []
provides:
  - internal/audit package with eight pure checks over a file list
  - x-terminal audit verb (--json twin)
  - extended session leak surface list
affects: [08-02, 08-03, repair-path]
actuals:
  tokens: 18000
  tasks: 3
  commits: 0
tech-stack:
  added: []
  patterns: [pure-check-over-tree, anti-vacuity-arm, synthetic-plus-planted-proof]
key-files:
  created:
    - internal/audit/audit.go
    - internal/audit/readonly.go
    - internal/audit/guards.go
    - internal/audit/leak.go
    - internal/cli/audit.go
  modified:
    - internal/cli/cli.go
    - internal/session/leak_test.go
key-decisions:
  - "transport request-builder allowlist includes internal/netcheck and internal/opsgen beside internal/xweb; both are GET-only and already in the tree"
  - "audit package sources are skipped by checks that would otherwise fail on their own classification tables and needle lists"
  - "account check counts Config.SessionPath defaults in config.go plus session.Load; a third SessionPath key-value is the planted second-account arm"
patterns-established:
  - "Every check reports Files inspected; empty tree/list is a finding"
  - "audit.Run recovers panics into findings"
  - "cli auditRun package-level seam mirrors feedSource"
requirements-completed: [SEC-03]
coverage:
  - id: D1
    description: "internal/audit Load/Run/Checks with File/Tree/Finding/Report; package imports neither store nor xweb nor cli"
    requirement: SEC-03
    verification:
      - kind: unit
        ref: "internal/audit/audit_test.go#TestLoad|TestPackageImportsNeitherStoreNorXweb|TestRunRealTree"
        status: pass
    human_judgment: false
  - id: D2
    description: "readonly + transport checks with synthetic and planted failure arms"
    requirement: SEC-03
    verification:
      - kind: unit
        ref: "internal/audit/readonly_test.go"
        status: pass
      - kind: manual_procedural
        ref: "plant CreateTweet / MethodPost → audit exit 1; revert"
        status: pass
    human_judgment: false
  - id: D3
    description: "timers, persistence, distribution, account, boundaries, leak checks"
    requirement: SEC-03
    verification:
      - kind: unit
        ref: "internal/audit/guards_test.go|leak_test.go"
        status: pass
      - kind: manual_procedural
        ref: "six planted violations observed non-zero then reverted"
        status: pass
    human_judgment: false
  - id: D4
    description: "x-terminal audit verb with text/--json, non-zero on findings"
    requirement: SEC-03
    verification:
      - kind: unit
        ref: "internal/cli/audit_test.go#TestAudit"
        status: pass
      - kind: other
        ref: "go run ./cmd/x-terminal audit → exit 0, 8 checks, 0 findings"
        status: pass
    human_judgment: false
verified: |
  gofmt -l internal cmd                         empty
  CGO_ENABLED=0 go build ./...                  ok
  CGO_ENABLED=0 go test ./... -count=1          1153 passed in 22 packages
  go run ./cmd/x-terminal audit                 exit 0; 8 checks, 0 findings, 782 files
  go run ./cmd/x-terminal audit --json          findings=0, 8 checks
  report contains no absolute repository prefix
  go.mod / go.sum                               unchanged
planted: |
  1-readonly: exit=1; internal/xsource/home.go:123: write operation name "CreateTweet" is spelled in source
  2-transport: exit=1; internal/cli/keys.go:31: http.NewRequest builds an *http.Request outside the GET request-builder packages; request method POST is not GET
  3-timers: exit=1; internal/cli/keys.go:31: banned timer call time.NewTicker
  4-persistence: exit=1; internal/cli/keys.go:31: write-capable call os.WriteFile resolves to a fifth destination
  5-distribution: exit=1; Dockerfile: distribution artifact "Dockerfile" exists
  6-account: exit=1; internal/session/session.go:176: second session-path resolution: session loader (session.Load)
  after each plant: source restored; final audit exit 0; no _auditPlant leftovers; no Dockerfile
unverified: |
  OPS-03 doctor extensions (plan 02); OPS-06 RUNBOOK (plan 03); DSGT-08 cover re-audit.
duration: 90min
completed: 2026-10-15
---
# 08-01 SUMMARY — repo-wide audit package and verb (SEC-03)

**`internal/audit` proves the tree's promises as pure checks over a file list, and `x-terminal audit` prints what each check inspected and exits non-zero on any finding.**

## Accomplishments

- Eight checks: readonly, transport, timers, persistence, distribution, account, boundaries, leak — each with Files count, anti-vacuity, synthetic tests, and (for the six plantable ones) a real-tree planted failure observed then reverted.
- `audit.Load` walks non-test `.go` plus manifests; test files excluded on purpose.
- `x-terminal audit` / `--json` wired as one new command-table row; report is repository-relative and token-free.
- `internal/session/leak_test.go` surface list extended (doctor/sync/audit/media/store/panic) without replacing existing arms.

## Planted-violation observations

1. **readonly** — plant `CreateTweet` in `internal/xsource/home.go` → exit 1; `internal/xsource/home.go:123: write operation name "CreateTweet" is spelled in source` → reverted.
2. **transport** — plant `http.NewRequest(http.MethodPost, …)` in `internal/cli/keys.go` → exit 1; outside-builder + non-GET findings at keys.go:31 → reverted.
3. **timers** — plant `time.NewTicker` in `internal/cli/keys.go` → exit 1; `banned timer call time.NewTicker` → reverted.
4. **persistence** — plant `os.WriteFile("/tmp/…")` in `internal/cli/keys.go` → exit 1; `fifth destination` → reverted.
5. **distribution** — plant root `Dockerfile` → exit 1; `distribution artifact "Dockerfile" exists` → removed.
6. **account** — plant third `Config{SessionPath:…}` in `internal/config/config.go` → exit 1; `second session-path resolution` → reverted.

## Decisions

- Transport allowlists GET request builders in `internal/xweb`, `internal/netcheck`, and `internal/opsgen` (all already GET-only in the tree); the plan's wording named only xweb.
- The audit package skips its own sources where self-referential tables would trip the checks they implement.

## Files

- `internal/audit/{audit,readonly,guards,leak}.go` + tests
- `internal/cli/{audit.go,audit_test.go,cli.go}`
- `internal/session/leak_test.go`
