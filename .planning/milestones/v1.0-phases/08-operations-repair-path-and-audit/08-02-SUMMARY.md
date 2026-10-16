---
phase: 08-operations-repair-path-and-audit
plan: 02
type: summary
status: complete
requirements: [OPS-03]
subsystem: cli
tags: [doctor, ops-03, probe, mirror-facts, runbook-pointers]
requires:
  - phase: 06-bookmarks-likes-and-the-mirror-only-store
    provides: saved.db schema, meta owner/version, sync_runs
  - phase: 01
    provides: doctor verb shape, exit rules, session summary
provides:
  - domain.MirrorFacts + store.ReadFacts read-only path
  - xweb.Client.Probe + SignerName
  - doctor mirror/signer/media/--probe + RunbookAnchors table
affects: [08-03, repair-path]
actuals:
  tokens: 27311
  tasks: 3
  commits: 0
tech-stack:
  added: []
  patterns: [read-only-facts-dsn, one-request-probe, failing-state-runbook-table]
key-files:
  created:
    - internal/store/facts_test.go
    - internal/xweb/probe.go
    - internal/xweb/probe_test.go
  modified:
    - internal/domain/saved.go
    - internal/domain/saved_test.go
    - internal/store/read.go
    - internal/store/schema.go
    - internal/store/migrate.go
    - internal/store/schema_test.go
    - internal/xweb/graphql.go
    - internal/cli/doctor.go
    - internal/cli/doctor_test.go
key-decisions:
  - "ReadFacts uses mode=ro + query_only DSN; missing file is Exists=false, nil error"
  - "Probe returns classified non-success with nil error; only unbuildable requests error"
  - "RunbookAnchors exported for plan 03; mirror missing maps to empty pointer"
  - "mediaFacts hardcodes mpv to avoid cli→media import cycle"
patterns-established:
  - "Doctor gains labelled lines and one flag; stage order and exit-code rule stay Phase 1's"
  - "Presence-facts proof covers text+JSON with fixture cookie control arm"
requirements-completed: [OPS-03]
coverage:
  - id: D1
    description: "MirrorFacts + ReadFacts presence facts; missing/newer/unreadable shapes"
    requirement: OPS-03
    verification:
      - kind: unit
        ref: "internal/store/facts_test.go#TestReadFacts"
        status: pass
      - kind: unit
        ref: "internal/cli/doctor_test.go#TestDoctorMirrorLineShapes"
        status: pass
    human_judgment: false
  - id: D2
    description: "--probe one request through choke point; SignerName; media facts; runbook table total"
    requirement: OPS-03
    verification:
      - kind: unit
        ref: "internal/xweb/probe_test.go#TestProbe"
        status: pass
      - kind: unit
        ref: "internal/cli/doctor_test.go#TestDoctorProbe|TestRunbookAnchorTableIsTotal"
        status: pass
      - kind: manual_procedural
        ref: "comment out soft-ban row → TestRunbookAnchorTableIsTotal FAIL; restore → PASS"
        status: pass
    human_judgment: false
  - id: D3
    description: "Text+JSON presence-facts only; exit codes unchanged for skipped session / missing vs unreadable mirror"
    requirement: OPS-03
    verification:
      - kind: unit
        ref: "internal/cli/doctor_test.go#TestDoctorPresenceFactsTextAndJSON"
        status: pass
      - kind: unit
        ref: "internal/session/... leak surfaces still green"
        status: pass
    human_judgment: false
verified: |
  gofmt -l internal cmd                         empty
  CGO_ENABLED=0 go build ./...                  ok
  CGO_ENABLED=0 go test ./... -count=1          1186 passed in 22 packages
  go.mod / go.sum / internal/config             unchanged (diff --stat empty)
  TestRunbookAnchorTableIsTotal deliberate fail observed then restored
unverified: |
  Live `go run ./cmd/x-terminal doctor` / `--probe Viewer` against a real session (machine-local; plan verification checklist).
  RUNBOOK.md anchors existence (plan 03).
duration: 90min
completed: 2026-10-15
---
# 08-02 SUMMARY — doctor repair-path front door (OPS-03)

**`doctor` reports the mirror, signer, media openers and budget as presence facts; `--probe` spends exactly one request; every failing line names a `RUNBOOK.md` section.**

## Accomplishments

- `domain.MirrorFacts` / `MirrorKindCount` — closed fact set; no body/row fields.
- `store.ReadFacts` — read-only open (`mode=ro` + `query_only`); missing → Exists=false; newer → Newer=true + version; corrupt → named error; exported-surface allowlist + anti-vacuity mutators arm.
- `Client.Probe` + `SignerName()` — one GraphQL path request; body discarded; noop signer → `none`.
- Doctor: mirror/signer/media lines; `--probe` + `probeVariables`; `RunbookAnchor`/`RunbookAnchors`; JSON twin fields; 90s backstop and exit rules unchanged.

## Deliberate failure

- Removed `soft ban` runbook row → `TestRunbookAnchorTableIsTotal` FAIL (`missing xweb row "soft ban"`) → restored → PASS.

## Decisions

- Media video program string is the literal `mpv` in `cli` (matches `internal/media.videoProgram`) to avoid an import cycle.
- APIRejected defaults to `#unclassified` unless Detail carries a classifier word the mapper can use via checkState.

## Files

- `internal/domain/saved.go`, `saved_test.go`
- `internal/store/{read,schema,migrate,schema_test,facts_test}.go`
- `internal/xweb/{probe,probe_test,graphql}.go`
- `internal/cli/{doctor,doctor_test}.go`
