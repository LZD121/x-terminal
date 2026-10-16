# Phase 8: Operations, Repair Path and Audit - Context

**Gathered:** 2026-10-14
**Status:** Ready for planning
**Mode:** Smart discuss (autonomous) — proposals accepted in batch under the run-wide policy

<domain>
## Phase Boundary

When the upstream breaks — a rotated query id, a broken signer, an expired cookie, a rate limit —
the user can find out exactly what broke and repair it without touching code.

Delivers: an extended `doctor` (the mirror's presence facts, a one-operation probe, and the
runbook's path on any failing line); `RUNBOOK.md` with one section per failure case OPS-06 names;
a repo-wide audit that can fail, as both a verb and a test, covering the read-only promise, the
one-persistence-file promise, the no-timer promise, the no-distribution promise and the cookie
sweep; and the DSGT-08 re-audit that puts every screen built since Phase 3 back under the cover,
inside and outside tmux.

Does NOT deliver: any new reading surface or fetch feature; a UI for operation pinning (the
mechanism is `doctor --pin`, delivered in Phase 1); an automated repair that rewrites the registry
without a human deciding; release tooling, packaging, container or hosted mode (forbidden by the
project's own scope); or CI configuration — the audits run as `go test ./...` and as one command.

</domain>

<decisions>
## Implementation Decisions

### `doctor`: extended, not rebuilt

- The existing verb is the shape to keep: one line per link of the path, presence facts only, a
  text report and a JSON twin, exit non-zero when a link failed. Phase 8 **adds lines and one flag**;
  it does not restructure what Phase 1 built, because that shape is already the repair path's front
  door.
- **Added: the mirror line.** Path, present-or-absent, schema version, owner handle, row counts per
  kind, how many rows are marked absent, the last sync's time and stop reason — read through the
  store's read-only path (`query_only`), never a post body, never a row's contents. A mirror that
  does not exist yet is a presence fact ("no mirror yet"), not a failure.
- **Added: `doctor --probe <operation>`.** Exactly one request for the named operation, through the
  same choke point (single-flight, the 1.5 s floor, the session cap) and the same classifier, and it
  prints the resulting state word plus the rate-limit headers (remaining and reset) — **never a
  response body**. This is what turns the runbook's verification step into a command rather than a
  guess: after pinning an operation id, after a features change, after re-exporting a session, the
  user runs one command and reads one word. It is the only new request path this phase adds, and it
  is deliberately one operation per invocation, never a sweep.
- **Added: the runbook's path on a failing line.** When a check fails, the line names `RUNBOOK.md`
  and the section for that failure. The doctor is the entry point to the repair path, and a
  diagnosis that does not say where the repair is written is half a diagnosis.
- Unchanged and re-asserted: **presence facts only, never a token value** — the token sweep is
  re-run across the new lines, including the JSON twin (SEC-01, OPS-03).

### The runbook (`RUNBOOK.md`)

- **One file, at the repository root** — the repo has no docs directory today and this is a
  single-user tool: the runbook sits beside `AGENTS.md` where it is found rather than filed. It is
  the only document this project adds.
- **One section per case, and the cases are OPS-06's list plus the two this build adds**:
  operation-id rotation; features-blob drift; signer break; session expiry; `429`; soft ban;
  accidental distribution; plus the mirror's owner/schema (Phase 6's account-change repair) and the
  likes endpoint's absence.
- **Each section has the same four parts, and only the fourth is optional to skim**: the symptom
  (the exact words the status line shows, quoted from the classifier rather than paraphrased);
  what `doctor` shows for that case; the repair, which is one command or one file edit; and the
  verification, which is a command whose output is the answer (`doctor --probe <op>`, `audit`,
  `go test ./...`) — never "look at the UI and see whether it seems fine".
- **The request discipline is stated in the file**, near the top, and repeated in the `429` and
  soft-ban sections: a repair is one request, there is no retry loop, and a repair that fails is a
  second diagnosis rather than a second attempt. The moment a user needs a runbook is the moment
  they are most likely to hammer the upstream, which is how a soft ban becomes a hard one.
- **Accidental distribution** is the section that says: unpublish or archive the artifact, rotate
  the credential (log out everywhere, or change the password — the session cookie is a bearer token
  for the whole account), re-read what the terms say about internal endpoints, then re-run `audit`
  and the leak sweep. It is a repair instruction, not a policy essay: the project's position on
  distribution lives in `PROJECT.md`.
- **The runbook cannot rot silently.** A test asserts that every failure state the classifier can
  produce has a section in the file, so adding a state without documenting its repair fails the
  build. This is the cheapest thing that keeps a document honest in a repository with one
  contributor and no review.

### The audits, and how they can fail

- **A new package, `internal/audit`**, holding each check as a pure function over a file list, plus
  `x-terminal audit` — one new verb in the existing command table — which prints one line per check
  with the number of files it inspected and any findings (file and line), and exits non-zero when
  anything fails. Running the same checks as `go test ./internal/audit/...` keeps them from rotting;
  running them as a verb makes them usable in the repair path and before anything leaves the
  machine.
- **Every check has an anti-vacuity arm and a deliberate-failure test.** A scan that inspected no
  files, or that would not have noticed the thing it looks for, is worse than no scan; so each check
  reports what it inspected, and each has a test that feeds it a synthetic violation and asserts the
  finding.
- **The checks, each traceable to a promise:**
  1. **Read-only (SEC-03).** The set of operation names appearing as string literals in non-test Go
     source is collected and compared against a table inside the audit package that classifies each
     as a read or a write. An unclassified name fails; a write name fails; an empty set fails
     (anti-vacuity). The registry JSON *is* data and carries the whole bundle's map, writes
     included — it is exempt, because pruning it would violate XAPI-03's add-only refresh rule, and
     because what matters is that no *code* can name a write.
  2. **The transport cannot express a mutation.** No request method other than GET is constructed
     anywhere in the tree, and no package outside `internal/xweb` builds an `*http.Request`.
  3. **No timers.** No `time.NewTicker`, `NewTimer`, `AfterFunc`, `Tick`, or a sleep outside the
     limiter's own inter-request floor, in non-test source. The upstream must only ever be reached
     because a human pressed a key.
  4. **One persistence file.** Every write-capable call site (`os.Create`, `os.OpenFile`,
     `os.WriteFile`, a store DSN) resolves to exactly: the session file's create-if-absent template,
     `pins.json`, the first-run marker, and `saved.db` with its WAL siblings. A new one fails — this
     is the check that makes "no timeline on disk" a property of the tree rather than of intent.
  5. **No distribution artifact and no hosted mode.** No `Dockerfile`, `PKGBUILD`, `.goreleaser*`,
     install or release script, or publishing CI workflow; no `net.Listen`, `http.Server` or bind
     address anywhere. The project's scope says not distributed; the audit is where that stops
     being a sentence.
  6. **One account.** Exactly one session-path resolution and one credential pair in the tree; no
     `--account`/`--user` flag and no per-account config key.
  7. **The cookie sweep, re-run.** The fixture cookie appears nowhere in `doctor`'s text and JSON
     output, the sync output, the audit's own output, the media notices, the store's error strings,
     or a recovered panic's output. SEC-01's test exists; this extends its surface list and is the
     audit's own proof that it does not leak what it scans.
  8. **The import boundaries still hold.** `internal/tui` cannot reach `net/http`, `os/exec`,
     `database/sql`, `internal/store` or `internal/xsource`; `os/exec` is confined to
     `internal/media` (and the tmux escape's home); the volatile vocabulary (`queryId`, the feature
     names, the bundle hosts) appears only under `internal/xweb/volatile`.
  - The boundary checks re-assert Phase 2's and Phase 7's tests rather than replacing them: the
    audit is the tree-wide version, and either failing is a finding.
- **The audit's output is a report, not a log**: no post text, no token, no path outside the
  repository — the same rule the doctor lives by, checked by check 7.

### The DSGT-08 re-audit: everything since Phase 3 under the cover

- **The golden corpus grows to every screen × every cover style.** Timeline, thread, search
  results, people, profile, bookmarks, likes and every detail frame, rendered under the plain
  grammar and under each of Phase 3's four covers at 80 and 120 columns, with the adversarial
  content cases each screen can carry. The stored frames are the specification, so a screen that
  was never rendered under a cover shows up as a missing file rather than as a silent gap.
- **Assertions over the corpus, not over intentions**: no covered frame contains a real handle,
  display name, product name or purpose-naming copy from the content it was built from; the keymap
  table is byte-identical across display states (so "unchanged keys" is a comparison of two frames'
  help output, not a promise); the three cover-breaking actions remain gated (Phase 7's recording
  renderer and effect sink, re-run with a covered frame on screen); and the covered refusal line and
  the covered help rows for the new actions exist.
- **The panic path is re-verified with the probe harness, not by eye**: `wtype` injecting the keys
  and `grim` capturing the window for the bare-Ghostty leg, `tmux capture-pane -S -N` for the
  multiplexer leg, with the new screens on screen when panic is pressed. Phase 3 measured the
  mechanism; this phase proves it still holds for the surface that arrived afterwards.

### Claude's Discretion

- The audit's exact check ordering, its report wording, and whether a finding is one line or a few;
  the runbook's section order and how long each section is; whether `--probe` prints the
  rate-limit headers in a fixed format or as the doctor's own budget line; whether the mirror line
  lists counts per kind or a single total; and the corpus's directory layout under
  `internal/render/testdata/`.
- **Nothing in this phase needs the user**: every choice above is an implementation detail of a
  repair path whose cases OPS-06 already names, and the one product-level question this project has
  left open (Phase 6's mirror ownership after an account change) is already flagged in that phase's
  file, with its repair written into this phase's runbook.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets

- `internal/cli/doctor.go` — the whole shape to extend: the staged connectivity check, the session
  summary, the registry line (snapshot age, pin count), the budget line, the JSON twin, the
  `--refresh-ids` path, and the `90s` backstop. The mirror line and `--probe` are additions to this
  file's report, not a second diagnostic.
- `internal/netcheck` — the five named connectivity outcomes the doctor already prints.
- `internal/xweb` — `Client.GraphQL` (the choke point every request must pass, including
  `--probe`), the measured classifier, the limiter's budget read-out, and
  `volatile.Registry`/`pins.json` (the repair target for rotation).
- `internal/store` (Phase 6) — a `query_only` reader and a `meta` row holding the schema version and
  the mirror's owner; the doctor reads exactly those two things plus row counts.
- `internal/session` — `Summary()` and the leak test (`internal/session/leak_test.go`), which this
  phase extends rather than replaces.
- `internal/render` — the golden corpus, its capture harnesses inside and outside tmux, and the
  adversarial-content audit (Phase 2, plan 05); the DSGT-08 re-audit is a superset of that corpus.
- `internal/keys` — the one table that makes "unchanged keys under a cover" a comparison of two
  renderings rather than a claim.
- Phase 3's `Grammar` values, alias function and marker sets; Phase 4–7's frames, actions and
  literals — everything this phase audits.
- The evidence files, for the runbook's own statements: `.planning/research/UPSTREAM-PROBE.md`
  (measured failure shapes, the external pin source, the optional features bag),
  `.planning/research/TERMINAL-MEDIA-PROBE.md` (the probe method the panic re-verification reuses),
  and `.planning/research/PITFALLS.md`'s recovery table (the source of the runbook's repair steps).

### Established Patterns

- Presence facts only, never a value that could be a credential; a report a user can paste into a
  bug report.
- Tests that can fail for the right reason: anti-vacuity arms, deliberate-failure fixtures, and a
  count of what was inspected.
- Golden frames as the view specification, with the capture harnesses (tmux and a fixed-size PTY)
  as the terminal's own agreement with the model.
- The import-boundary scan as the mechanised form of an architectural decision, with the volatile
  vocabulary confined to one package.
- One verb per job in the single command table; a verb's body owns its diagnostics and its exit
  code; stdout carries only the verb's own output.

### Integration Points

- `internal/cli` — one new row (`audit`) and the doctor's new lines and flag; the command table
  remains the whole surface.
- `internal/audit` — a new package that reads the tree and the binary's module graph facts it can
  read without running the build; it must not import `internal/store`, `internal/xweb` or the shell,
  so that a broken build can still be audited.
- `RUNBOOK.md` — a new file at the root, referenced by the doctor's failing lines and by the audit's
  report heading.
- `internal/render/testdata/` — the corpus grows; the missing-file failure mode is the point.
- Phase 6's mirror flag and Phase 7's gate both resolve in this phase's runbook and corpus, which is
  why this phase is last: it needs the store's schema and a failure or two to render from.

</code_context>

<specifics>
## Specific Ideas

- The doctor's output should stay quotable in a bug report: one line per check, a named outcome
  each, the mirror's facts as numbers, and no token material anywhere — the new lines must not break
  that property.
- `--probe` is the repair loop's closing bracket: pin, probe, read one word. It should feel like
  asking a question, not like starting a test suite.
- The runbook should read like the tool's own voice — terse, factual, one command per step — because
  it is written for the person who is already annoyed.
- `audit` should print what it inspected, not only what it found: "no findings" over an empty file
  list is the one output that would make the whole audit worthless.
- The cover re-audit should be able to say "every screen this build has, under every cover this
  build has, at both widths, stored and compared" in one sentence — that is what makes DSGT-08 a
  property rather than a hope.

</specifics>

<deferred>
## Deferred Ideas

- A per-operation pin UI (the research's item, and the roadmap's own note): the mechanism is
  `doctor --pin`, it is one command, and a screen for it would be configuration surface for one
  user's rare repair.
- An automated repair that rewrites a registry entry without a human: the add-only pin path exists
  precisely because the human's decision is the safety property.
- Release tooling, packaging, a container image or a hosted mode: forbidden by the project's scope,
  and the audit checks for their absence rather than helping produce them.
- CI configuration: `go test ./...` and `x-terminal audit` are the checks; a CI service would add a
  second place where the credential story has to be reasoned about, and this project has no server.
- Telemetry or metrics: the budget line and the sync runs table are the only measurements this tool
  keeps, and they stay local.
- A startup self-check on every launch: the doctor is the deliberate act; the launch path stays
  quiet.
- An in-app "explain this state" help page: the runbook and the classifier's own words cover it
  without a second copy of the same sentences.
</deferred>
