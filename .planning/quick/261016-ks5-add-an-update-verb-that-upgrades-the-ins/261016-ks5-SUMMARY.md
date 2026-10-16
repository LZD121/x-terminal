---
phase: 261016-ks5
plan: 01
type: summary
status: complete
quick-task: add-an-update-verb-that-upgrades-the-installed-binary
commits:
  - a7c872c  # 261016-ks5 task 1, the release wire, the verb's row, --check, the transport allowlist's fourth root
  - f94608c  # 261016-ks5 task 2, the download, the published digest, the atomic replacement, the seventh persistence destination
  - 17f7a08  # 261016-ks5 task 3, the RUNBOOK section and the coverage rows
scope: |
  Tasks 1, 2 and 3 in full, including Task 3's live gate. The tag v1.2.0 and its Release were
  published by the operator/dispatcher, not by this executor, which pushed nothing, created no tag
  and touched no file outside this working tree and /tmp. Section 1 records the read half of the
  gate against the v1.1.0 Release; section 2 records the install half, the live v1.1.0 -> v1.2.0
  upgrade, with both --version lines and both digests. ~/.local/bin/x-terminal now holds the
  published v1.2.0 asset byte for byte (digest c451ac5d...), where it held a working-tree build.
verified: |
  CGO_ENABLED=0 go test ./... -count=1            1738 passed in 23 packages, no failures
  CGO_ENABLED=0 go vet ./...                      clean
  gofmt -l internal cmd                           empty
  go run ./cmd/x-terminal audit                   10 checks, 0 findings, 978 files inspected
  git diff --stat go.mod go.sum                   empty (no module added: stdlib only)
  git diff --numstat -- internal/audit/guards.go   12 added, 4 deleted (bounds: <=14 / <=6)
  planted transport violation                     internal/cli/planted_violation.go reported at :5,
                                                  audit 2 findings; file removed, back to 0
  planted persistence violation                   internal/release/other.go reported at :5, audit
                                                  2 findings; file removed, back to 0
  x-terminal update --check (live, anonymous)     section 1, exit 0
  x-terminal update (live, anonymous)             section 2, exit 0 on attempt 16; the installed
                                                  digest equals the published v1.2.0 digest
  x-terminal --version, after the upgrade         x-terminal v1.2.0 (6a184c6393f0) go1.27.2 linux/amd64
  second x-terminal update (live, anonymous)      "this build (v1.2.0) is the latest release", exit 0,
                                                  installed digest unchanged
unverified: |
  Nothing the plan's live gate asked for is unrun. One literal step is impossible rather than
  unproven: the published v1.1.0 asset cannot run `update` at all (`unknown command: update`, exit
  2), because it was built from a1c2e1b, before the verb existed. The upgrade was therefore
  performed from a v1.1.0-stamped build of master's tip — the plan's own Task 1 rig, and the only
  shape in which an old -> new upgrade of the shipped verb is observable. Section 2 says so, records
  what the 16 attempts measured, and names the environment fact behind them (the transport's 10 s
  TLS handshake budget against a path whose handshakes measure 5.8-37.7 s through this proxy).
---

# 261016-ks5 SUMMARY — `x-terminal update` reads the Release, verifies the digest it publishes, and replaces the running binary atomically

## 0. What shipped

| Task | Commit | What |
|---|---|---|
| 1 | `a7c872c` | `internal/release/release.go` (the GET-only release wire, tag ordering, the release's own asset picker, `session.Secret` token, `Client.Format`), `internal/cli/update.go` + its row in `internal/cli/cli.go`, `--check` end to end, `internal/audit/readonly.go`'s fourth builder root with `TestTransportPassesTheReleaseRoot`, and the verb list row in `internal/cli/cli_test.go` |
| 2 | `f94608c` | `internal/release/install.go` (resolve, writability check, digest parse, verify, three shape checks, atomic replace), `Client.Fetch`, the verb's install path and `--force`, `internal/audit/guards.go`'s seventh destination and its anchor, `persistenceDestinationNames`/`persistenceBaseTree`/`TestPersistenceFailsAWriteBesideTheInstaller` |
| 3 | `17f7a08` | `RUNBOOK.md`'s `## The update path` section and `internal/cli/runbook_test.go`'s three joins (coverage rows, `builtCommands`, `classifierWords`); the live gate is recorded in sections 1 and 2 |

### The shipped surface

`internal/release`'s exported API (`go doc ./internal/release`), all of it used by the verb:

```
func New(cfg config.Config, token session.Secret) (*Client, error)
func (c *Client) Latest(ctx context.Context) (Release, error)
func (c *Client) Fetch(ctx context.Context, url string, limit int64) ([]byte, error)
func SetIdentity(v string)  /  func UserAgent() string
type Release / Asset / HTTPError (Status, URL, Body) / Version / Client
func ParseTag(tag string) (Version, error)  /  func CompareTags(a, b string) (int, error)
func AssetName(tag, goos, goarch string) string
func AssetFor(assets []Asset, tag, goos, goarch string) (Asset, error)
func AssetNamed(assets []Asset, name string) (Asset, error)
func ResolveExecutable(path string) (string, error)
func CheckReplaceable(target string) error
func ParseDigest(body []byte, wantName string) ([32]byte, error)
func Verify(payload []byte, want [32]byte) error
func LooksLikeThisBuild(payload []byte, tag, goos string, size int64) error
func Replace(target string, payload []byte) error
```

The verb's five states, each printed as
`update: <state>: <detail> (RUNBOOK.md#the-update-path)` on stderr, with the anchor
`the-update-path` = `## The update path`:

| State constant | Text |
|---|---|
| `stateNotReachable` | `the release is not reachable` |
| `stateNoAsset` | `no release asset for this platform` |
| `stateNotVerified` | `the download did not verify` |
| `stateNotWritable` | `the install path is not writable` |
| `stateNotARelease` | `this build is not a release` |

`updateStates` is the single declaration the verb prints from, the runbook join maps through, and
`classifierWords()` reads — so the section and the code cannot drift apart silently.

The seventh persistence destination, as `x-terminal audit` prints it (`17f7a08`):

```
every write resolves to one of the 7 named destinations: config, pins, notice, saved, update (the running tool); registry, spike-recording (a development tool) [spike-recording@cmd/spike/main.go:176, notice@internal/cli/notice.go:64, config@internal/config/config.go:548, config@internal/config/config.go:607, update@internal/release/install.go:78, update@internal/release/install.go:237, saved@internal/store/read.go:198, saved@internal/store/schema.go:50, saved@internal/store/schema.go:56, pins@internal/xweb/volatile/pins.go:81, registry@internal/xweb/volatile/registry.go:403, registry@internal/xweb/volatile/registry.go:414]
```

The transport allowlist (`requestBuilderOK`) now names four roots — `internal/xweb`,
`internal/netcheck`, `internal/opsgen`, `internal/release` — with the fourth's justification in the
comment in the voice of the other three, no exemption, no widened method arm, and no method
constant other than the existing GET spelling.

## 1. The live gate, read half — 2026-10-16, anonymously (`env -u GH_TOKEN -u GITHUB_TOKEN`), proxy `http://127.0.0.1:7897`

Every attempt below is a hand-made attempt with several seconds between attempts; none of them is a
loop, and every failure recorded here is the measured proxy flakiness rather than a verb defect.

| # | Step | Attempt | Result |
|---|---|---|---|
| 1 | `curl ... https://github.com/LZD121/x-terminal` | 1 | `200` (10.1 s) |
| 2 | `curl ... https://api.github.com/repos/LZD121/x-terminal/releases/latest` | 1 | curl exit 28: timed out after the body was fully received (8439 bytes); `-w` printed `200` |
| 2b | same | 2 | `200` in 8.4 s, tag `v1.1.0`, 7 assets |
| 3 | `curl ... /releases/download/v1.1.0/SHA256SUMS` | 1 | `302` (no `-L`: the signed object URL is the next hop) |
| 3b | same, `-L` | 2 | curl exit 28: 1 redirect followed, 0 bytes, timed out at 45.6 s |
| 3c | same, `-L` | 3 | `200` in 23.6 s after 1 redirect |
| 4 | `/tmp/ks5-check update --check` (build stamped `-X ...version=v1.1.0`) | 1 | TLS handshake timeout → `the release is not reachable` state with its pointer, exit 1 |
| 4b | same | 2 | the two expected lines, exit 0 |
| 5 | `/tmp/ks5-old update --check` (stamped `v1.0.0`) | 1 | the two expected lines, exit 0 |
| 6 | working-tree build (`go run`), `update --check` | 1 | release line + `this build is not a release`, exit 1 |

The releases API's own answer, verbatim from attempt 2 (`/tmp/ks5-latest2.json`):

```
tag: v1.1.0 | assets: 7
  SHA256SUMS                                       289  uploaded
  x-terminal_v1.1.0_darwin_arm64              13627762  uploaded
  x-terminal_v1.1.0_darwin_arm64.sha256             97  uploaded
  x-terminal_v1.1.0_linux_amd64               14352544  uploaded
  x-terminal_v1.1.0_linux_amd64.sha256              96  uploaded
  x-terminal_v1.1.0_linux_arm64               13566112  uploaded
  x-terminal_v1.1.0_linux_arm64.sha256              96  uploaded
```

The `SHA256SUMS` body fetched anonymously on attempt 3, verbatim:

```
b84294e07b4031810ad139b15b35af1fa7ed69fd24d7e3fb0bb1f344752a6045  x-terminal_v1.1.0_darwin_arm64
ac4057ceb27f8dac69d1bd779f6b01425735e6354829fc77cdd5c106b4b06faf  x-terminal_v1.1.0_linux_amd64
accbf86f5ba19492bdfacab0b310abf490534c89747d44d5f302b00169af419d  x-terminal_v1.1.0_linux_arm64
```

The verb's own output, verbatim.

Step 4b — `/tmp/ks5-check`, stamped `v1.1.0` the way the workflow stamps an asset:

```
release v1.1.0, linux/amd64 asset x-terminal_v1.1.0_linux_amd64
this build (v1.1.0) is the latest release
exit 0
```

Step 5 — the same source stamped `v1.0.0`:

```
release v1.1.0, linux/amd64 asset x-terminal_v1.1.0_linux_amd64
x-terminal v1.1.0 is available (this build is v1.0.0)
exit 0
```

Step 6 — the working-tree build (no `-X` stamp at all):

```
release v1.1.0, linux/amd64 asset x-terminal_v1.1.0_linux_amd64
update: this build is not a release: install a published asset, or build with the release workflow's -ldflags stamps (RUNBOOK.md#the-update-path)
exit 1
```

Step 4's first attempt, kept because it is the failure state working live:

```
update: the release is not reachable: release: GET https://api.github.com/repos/LZD121/x-terminal/releases/latest: Get "https://api.github.com/repos/LZD121/x-terminal/releases/latest": net/http: TLS handshake timeout (RUNBOOK.md#the-update-path)
exit 1
```

## 2. The live old → new upgrade proof, v1.1.0 → v1.2.0

Run on 2026-10-16, 08:19–08:35Z, from the repository root, anonymously (`env -u GH_TOKEN -u
GITHUB_TOKEN`), through the same proxy as section 1. `v1.2.0` is the operator's tag at `6a184c6`;
this executor pushed nothing. Every attempt below is a hand-made, spaced attempt — none of them is a
loop — and every one that failed is recorded with what it printed.

One convention in the transcripts below: the verb prints the path it replaces as the absolute path it
resolved, so its own output carries the operator's home directory. Those lines are shown with
`$HOME` in place of it, for the same reason three planning documents were redacted before this
repository was made public; nothing else in a quoted line is altered.

### 2.1 The "before" state, installed by hand (the README's documented path)

The published `v1.1.0` asset was fetched anonymously and verified before it was installed:

- `x-terminal_v1.1.0_linux_amd64.sha256` → `200` in 27.0 s after 1 redirect (attempt 1)
- `x-terminal_v1.1.0_linux_amd64` → `200` in 20.7 s, **14,352,544 bytes** after 1 redirect (attempt
  1) — the size the v1.1.0 API publishes for it
- `sha256sum -c x-terminal_v1.1.0_linux_amd64.sha256` → `x-terminal_v1.1.0_linux_amd64: OK`
- `install -m 0755 … ~/.local/bin/x-terminal` → `x-terminal v1.1.0 (a1c2e1be5d41) go1.27.2
  linux/amd64`, digest `ac4057ceb27f8dac69d1bd779f6b01425735e6354829fc77cdd5c106b4b06faf`. What it
  replaced: the working-tree build, `x-terminal (working tree build) go1.27.2 linux/amd64`, digest
  `c879808449a7093c999d38806e8c86bd2aa5810866c0d7fe78fd145ccdd03aef`

**Measured fact that changes the plan's step 2:** the published `v1.1.0` asset does not carry the verb
at all — it is the build of `a1c2e1b`, which predates it:

```
$ x-terminal update
unknown command: update
usage: x-terminal [command] [args...]
exit 2
```

So an upgrade cannot start from that binary, and the plan's "before" state is observable only as a
`v1.1.0`-stamped build **of a tree that has the verb**. That is the plan's own Task 1 rig, and it is
what this proof's "before" state is:

```
$ CGO_ENABLED=0 go build -trimpath -ldflags "-X github.com/LZD121/x-terminal/internal/cli.version=v1.1.0 \
    -X github.com/LZD121/x-terminal/internal/cli.commit=6a184c6393f0" -o /tmp/ks5-live/before-v1.1.0-stamped ./cmd/x-terminal
$ install -m 0755 /tmp/ks5-live/before-v1.1.0-stamped ~/.local/bin/x-terminal
$ x-terminal --version
x-terminal v1.1.0 (6a184c6393f0) go1.27.2 linux/amd64
$ sha256sum ~/.local/bin/x-terminal
f3e478ee2b8d271fa3f2e20cb238816104d6fcee27a1e12da2efa52ca84e63e2  $HOME/.local/bin/x-terminal
```

### 2.2 `x-terminal update` — the upgrade, verbatim

Succeeded on **attempt 16** of 16 spaced attempts (the fifteen before it are in 2.4):

```
$ env -u GH_TOKEN -u GITHUB_TOKEN x-terminal update
x-terminal v1.2.0 is available (this build is v1.1.0)
replacing $HOME/.local/bin/x-terminal (this build v1.1.0) with v1.2.0
verified against x-terminal_v1.2.0_linux_amd64.sha256
replaced $HOME/.local/bin/x-terminal with v1.2.0
the process that ran this is still the old image: the next launch reports v1.2.0
v1.1.0 -> v1.2.0
exit 0
```

Both digests — the expected value fetched from the Release by hand, and the installed file read
afterwards — are the same 64 hex digits. This is the only thing that proves the bytes installed are
the bytes published:

```
published  (x-terminal_v1.2.0_linux_amd64.sha256, fetched anonymously):
c451ac5d7c418135d06484c7153212f683d4868df1d7435c3523acc94ecd4488  x-terminal_v1.2.0_linux_amd64
observed   (sha256sum ~/.local/bin/x-terminal):
c451ac5d7c418135d06484c7153212f683d4868df1d7435c3523acc94ecd4488  $HOME/.local/bin/x-terminal
```

### 2.3 The next launch, and the second run

```
$ x-terminal --version                 # a new process, not the one that ran the update
x-terminal v1.2.0 (6a184c6393f0) go1.27.2 linux/amd64
```

`6a184c6393f0` is the tag's own commit (the release workflow stamps `git rev-parse --short=12 HEAD`),
and `-rwxr-xr-x 14397600` is the size the v1.2.0 API publishes for `x-terminal_v1.2.0_linux_amd64`
(14,397,600): the installed file is the published asset, byte for byte.

```
$ env -u GH_TOKEN -u GITHUB_TOKEN x-terminal update    # second run, answered on attempt 4 of 4
this build (v1.2.0) is the latest release
exit 0
$ sha256sum ~/.local/bin/x-terminal                    # unchanged
c451ac5d7c418135d06484c7153212f683d4868df1d7435c3523acc94ecd4488  $HOME/.local/bin/x-terminal
```

Acceptance 1 live: the already-latest line, exit 0, no `replacing` line (so no download), and the
installed bytes identical before and after.

### 2.4 What the fifteen failed attempts measured

They are the proxy flakiness the plan and RUNBOOK.md's first section are written around, and they are
also the evidence that a failed update is inert: through all fifteen, `~/.local/bin/x-terminal` stayed
`f3e478ee2b8d271fa3f2e20cb238816104d6fcee27a1e12da2efa52ca84e63e2` and
`ls -a ~/.local/bin | grep '^\.x-terminal-update-'` counted **0** entries — no partial binary and no
temporary file left behind, exactly as `install.go` promises.

| Attempts | Stage that timed out | Printed state |
|---|---|---|
| 1, 4, 8, 9, 12, 13, 14 | `api.github.com` handshake | `the release is not reachable` |
| 2, 3, 5, 11, 15 | `github.com` digest handshake (3 was `unexpected EOF`) | `the download did not verify` |
| 7, 10 | `release-assets.githubusercontent.com` handshake (the 302's target) | `the download did not verify` |
| second run's 1, 2, 3 | `api.github.com` handshake | `the release is not reachable` |

Every failure is `net/http: TLS handshake timeout`, and the cause is measurable: through this proxy a
TLS handshake costs **5.8–37.7 s** (`curl -w '%{time_appconnect}'`, spaced samples: 5.8, 10.5, 11.0,
11.2, 13.8, 15.2, 16.0, 17.1, 17.2, 21.9, 23.1, 29.7, 37.7 s), while `xweb.NewTransport` sets
`TLSHandshakeTimeout = 10 * time.Second` — a value measured for the x.com path on 2026-10-14 and
shared by every request in this program, not something the update verb introduced. One run needs
three of those handshakes (api.github.com, github.com, and the CDN host the 302 names) to land under
the budget in a row, which is what attempt 16 was. `socks5://127.0.0.1:7897` was measured as the
alternative path and is no better (13.8 s against 17.2 s for the same URL). Nothing in the product was
changed to make this pass: the attempts are the measurement.

## 3. Deviations from the plan

1. **`internal/cli/cli_test.go` was edited** (one line: `{name: "update", body: Update}` in the
   hand-written `verbBodies` list). The plan said `cli_test.go` needs no edit because it "derives
   the verb set from `commands()`", but direction three of
   `TestEveryVerbBodyIsReachableThroughCommands` counts the package's exported verb-signature
   functions against that hand-written list, so adding `Update` without the row fails the build.
   The file was not in the plan's `files_modified`. — Rule 3 (blocking).
2. **`updateClient` returns an interface (`releaseWire`), not `*release.Client`.** The plan fixes the
   signature as `func(cfg, token) (*release.Client, error)` *and* requires the verb's tests to drive
   an `httptest`-backed fake, but `release.Client`'s fields are unexported by the same plan's own
   decision, so a `cli` test cannot point one anywhere. The interface is the seam the plan says the
   tests need; the wire's request shape (path, method, the three headers, the token's absence and
   presence) is asserted inside `internal/release` against `httptest`, where the client is built, and
   the verb's own tests drive a map-backed fake that reaches no network at all. — Rule 3 (blocking),
   resolved toward the plan's stated intent (no exported way to redirect the production type).
3. **`Client.Fetch`, `readBounded`, `statusError` and `clip` landed in Task 1's commit** rather than
   Task 2's: they are part of the wire file the package is, and Task 1's `Latest` shares the bounded
   read with them. Task 2 is the commit that gives them a caller (the install path) and their tests.
4. **Task 1's commit answers and cannot install.** Since the plan splits the verb across two commits,
   Task 1's non-`--check` path printed, for the length of one commit, that this build answers the
   question and cannot install yet (with `--check` named as the invocation that answers). Task 2
   (`f94608c`) removed that branch and replaced it with the install path; the tree at HEAD has no
   such text, and no test of that interim shape survives.
5. **`update --check` on a Release with no asset for this platform exits before the version
   comparison.** The plan lists the not-a-release refusal among the states and the asset pick as
   `--check`'s one printed fact; the two can both apply, and printing the fact the answer rests on
   (or the state that says there is nothing to install) before the ordering question is the
   narrower-first order. In install mode the comparison is still the first thing the free question
   asks, as the plan's sequence requires.
6. **The `A bare update` arm at `f94608c` folds `--force` into the not-a-release case**: with
   `--force` and no stamp the verb takes the resolved release rather than refusing, which is what
   the plan's "unless `--force` was passed" says, and it is asserted.

## 4. Deliberately not amended (as the plan requires)

- The `account` check and its promise (`exactly one session path and one credential pair`): no token
  is required, none is written to a file, a keyring or a config key, and the token is a
  `session.Secret` — redaction by type rather than by care. `go run ./cmd/x-terminal audit` prints
  the promise unchanged.
- The `distribution` check's comment and promise line, `README.md` and `.planning/PROJECT.md`: the
  concurrent quick task `261016` recorded the public repository there. That is why the verb's
  user-facing documentation is its usage line and its `RUNBOOK.md` section.
- `orphanExempt` still names exactly one section (`accidental-distribution`), and the doctor's own
  `RunbookAnchors` table is untouched; no other RUNBOOK.md section's bytes moved.

## 5. Notes for the next reader

- `Client.Format` exists because fmt prints an *unexported* field's value without consulting that
  field's `String` method: measured on this toolchain (2026-10-16), `%v` of a `Client` holding a
  `session.Secret` printed the token while a `Session`, whose credential fields are exported,
  rendered `session.Redacted`. The client answers for itself, and the test asserts all six verbs
  render the placeholder.
- `User-Agent` is required by `api.github.com` (403 without one) and carries the build's own
  identity: `internal/release` cannot import `internal/cli`, so the verb hands `Version()` over
  through `release.SetIdentity`.
- `Fetch` refuses a URL that is not under the release origin (`https://github.com/`), so a Release
  whose asset list named another host would be a named error rather than a download.
- `internal/release` is a new package root and the audit's `transport` check knows it: run
  `go run ./cmd/x-terminal audit` after any change to its request-building code.

---
*Phase: 261016-ks5 (quick task)*
*Executed: 2026-10-16*
*Live upgrade proof: complete — v1.1.0 → v1.2.0 on 2026-10-16 (16 attempts), installed digest equal to the published digest, and the second run up to date.*
