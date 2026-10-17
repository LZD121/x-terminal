# RUNBOOK.md

Repair path for this build. `doctor` names the section; the section names one
command; the command's output is the answer.

## Request discipline

A repair is one request. There is no retry loop. A failed repair is a second diagnosis rather than a second attempt. The moment this file is needed is the moment hammering the upstream turns a soft ban into a hard one.

<!-- anchors match the doctor's failing-line pointers (RUNBOOK.md#…) -->

## No path to x.com

**Symptom.** `no route to x.com`, `proxy refused`, `proxy reached but upstream blocked`, `TLS/Cloudflare challenge`, or `edge challenge`.

**What doctor shows.** The staged connectivity lines stop at the first break: route, proxy, upstream, or challenge. The config line's first hop is redacted.

**Repair.** Fix the path in `config.toml` (proxy host/port or `*_PROXY`), start the proxy if it is down, or clear the edge challenge in a browser on this host. One change, then re-check.

**Verify.** `x-terminal doctor`

## Operation id rotation

**Symptom.** `unknown operation`.

**What doctor shows.** The registry line (snapshot age, pin count) and, on `--probe`, the state word `unknown operation`.

**Repair.** Run `x-terminal doctor --refresh-ids` first — a bundle change is the common cause. If the operation is still missing from the new bundle, pin it by hand with `x-terminal doctor --pin <operation> <query-id>` into `pins.json`. Take the id from the operation's own document or a sibling project's snapshot, never from a guess. Pins are read at every start; no rebuild.

**Verify.** `x-terminal doctor --probe <operation>`

## Request shape drift

**Symptom.** `bad request` after an upstream change, or an operation that answers `404` with a zero-byte body where it used to answer.

**What doctor shows.** `--probe` reporting `bad request` with the upstream's own sentence, which names the variable it is missing — `must be defined at variable.includePromotedContent` is the whole shape of it — or an empty-bodied `404` for an operation that used to answer, which is what `SearchTimeline` did over GET until 2026-10-15.

**Repair.** `x-terminal doctor --probe <operation>` first, and read the name it prints. Run `x-terminal doctor --refresh-ids` only when the probe says the operation is unknown: a current id is not the failure here, and a refresh was measured reporting `+0 added, ~0 updated, 104 unchanged` for exactly this drift. A missing variable is a source change in the read layer (`internal/xsource`); a moved wire method is a change in the request builder (`internal/xweb/graphql.go`, whose `postOperations` table is the one place a method is decided). One request this build makes is in neither the registry nor the bundle's operation map — x.com's translation endpoint, reached by `T` — and none of that applies to it: there is no operation id to refresh and no variable a probe can report, so `internal/xweb/translate.go`'s URL *is* the whole claim about the upstream, and the measurement that settles it is the gated test `X_TERMINAL_CAPTURE=1 go test ./internal/xsource/ -run TestCaptureTranslation -count=1 -v`, which prints the path's own status beside the suffixed control that must answer `404`. Stop and report rather than retrying: a repair is one request, and the drift measurement needed four seconds between requests because a rapid burst is answered `401 {"code":89}`.

**Verify.** `x-terminal doctor --probe <operation>`

## Signer break

**Symptom.** `signer or auth rejected`.

**What doctor shows.** The signer line and the authenticated-read stage in the same run; session summary from `session.Load`.

**Repair.** Confirm the session is not the cause (`x-terminal doctor`), re-export the session and re-run; then `x-terminal doctor --refresh-ids`; then `x-terminal doctor --probe <operation>`. If all three fail, the operation needs a header this build does not send — stop and report. A source change would go in `internal/xweb/graphql.go`'s request builder; that is not a configuration step.

**Verify.** `x-terminal doctor`

## Session expiry

**Symptom.** `auth rejected` / no session file — `auth_token` and `ct0` are session cookies and die with the browser session.

**What doctor shows.** The session summary line (`session.Summary()`): path, presence, mode. A file anyone can read is a failure, not a state — `session.Load` refuses modes other than `0600`.

**Repair.** `x-terminal logout` (deletes the local copy; the browser session is untouched), then `x-terminal login`: it names the two cookies, reads them with no echo, and writes the session file at `0600` only after one check request accepts them. Doing it by hand — `auth_token=…` and `ct0=…` in `$XDG_CONFIG_HOME/x-terminal/session`, mode `0600` — still works and is what the verb automates.

**Verify.** `x-terminal doctor`

## 429

**Symptom.** `rate limited`, with the reset instant the upstream sent.

**What doctor shows.** The budget line and, on `--probe`, the remaining/reset headers. The status line carries the same reset.

**Repair.** Read the reset instant off the status line or `x-terminal doctor --probe <operation>`, wait for it, then one probe. A repair is one request; there is no retry loop; a failed repair is a second diagnosis.

**Verify.** `x-terminal doctor --probe <operation>`

## Soft ban

**Symptom.** `soft ban` — `code 88` while the budget still has room.

**What doctor shows.** The classifier's own row mapped through `internal/cli/failure.go`; the stop-the-session rule.

**Repair.** Stop for the session. A soft ban with budget remaining is the upstream holding the account back for reasons this client cannot see, so waiting for a reset is not the remedy. A repair is one request; there is no retry loop; a failed repair is a second diagnosis.

**Verify.** `x-terminal doctor`

## The mirror

**Symptom.** `wrong account` on the saved screens, a schema version newer than this build, or an unreadable `saved.db`.

**What doctor shows.** The mirror line: path, schema, owner, row counts, last sync. Newer and unreadable point here; missing is a presence fact, not a failure.

**Repair.** Owner mismatch: `mv saved.db saved.db.old` and `x-terminal sync --full` — the store refuses to re-point a mirror, and that refusal is a feature. Newer schema: update the build, or move the file aside if its contents are expendable (the move loses the mirror's data). Unreadable: the same move-aside.

**Verify.** `x-terminal doctor`, `x-terminal sync --full`

## The likes endpoint

**Symptom.** `unknown operation` / operation id stale on the likes list, with bookmarks working.

**What doctor shows.** Phase 6's `CapOperationMissing` on the likes frame; `--probe Likes` names the state word.

**Repair.** Pin the operation if the id is known: `x-terminal doctor --pin Likes <query-id>` into `pins.json`. Otherwise the mirror is still worth having with bookmarks alone — `x-terminal doctor --probe Likes` is how to tell which case this is.

**Verify.** `x-terminal doctor --probe Likes`

## Accidental distribution

**Symptom.** Something of this build (or of an account) left the machine.

**What doctor shows.** Not a classifier state — this section is the repair when an artifact, cookie, or binary escaped. See `.planning/PROJECT.md` for the project's position on distribution rather than restating it here.

**Repair.** Unpublish or archive the artifact. Rotate the credential (log out everywhere, or change the password — the session cookie is a bearer token for the whole account). Re-read what the terms say about internal endpoints. Then re-run `x-terminal audit` and the leak sweep.

**Verify.** `x-terminal audit` and `go test ./internal/session/...`

## Load shed

**Symptom.** `load shed`.

**What doctor shows.** The classifier's remedy: a bounded backoff, distinct from rate limiting.

**Repair.** Wait, then `x-terminal doctor --probe <operation>` once. Do not open a retry loop.

**Verify.** `x-terminal doctor --probe <operation>`

## Endpoint retired

**Symptom.** `endpoint retired`.

**What doctor shows.** A 404 that carried a JSON body; the classifier's remark: do not retry.

**Repair.** Do not retry. Record the shape with `x-terminal doctor`; the operation is gone upstream.

**Verify.** `x-terminal doctor`

## Empty response body

**Symptom.** `empty response body`.

**What doctor shows.** The doctor's route/proxy/upstream stages — a 2xx with no bytes is never an empty timeline.

**Repair.** Wait and probe once after confirming the path with `x-terminal doctor`. Do not render the answer as data.

**Verify.** `x-terminal doctor`

## Unclassified

**Symptom.** `unclassified`.

**What doctor shows.** The doctor's report, quotable as-is — a shape this build has never measured.

**Repair.** Do not retry. Keep the `x-terminal doctor` report; treat it as a second diagnosis before any change.

**Verify.** `x-terminal doctor` and the report itself

## The update path

**Symptom.** `the release is not reachable`, `no release asset for this platform`, `the download did not verify`, `the install path is not writable`, or `this build is not a release`. The first is the API or the download URL not answering — GitHub answers 404 both for a repository it cannot see and for a release that does not exist, so the line says which of the two it cannot tell apart. The second is the latest Release carrying no asset named for this `GOOS/GOARCH`; the published platforms are linux/amd64, linux/arm64 and darwin/arm64. The third is bytes that are not the bytes the Release publishes, or a payload that is not an object file carrying the tag it was published under: nothing was installed. The fourth is the directory the resolved binary lives in refusing the temporary file the replacement is made through. The fifth is a running binary with no version stamp, so there is no ordering between it and the Release. The verb also prints `this build (...) is the latest release`, which is not a failure and has no repair.

**What doctor shows.** Nothing about this: `doctor` names the path to x.com, the session and the request budget, and these states come from `api.github.com` and `github.com`, which the doctor's staged check does not touch. Read the verb's own line instead — it carries the state, the detail naming the release, path or platform, and its own `RUNBOOK.md#` pointer.

**Repair.** `the release is not reachable` — re-probe the path with `x-terminal doctor` first, because a proxy that is down looks exactly like a repository that is not visible; then export `GH_TOKEN` if the repository is private again, and re-run `x-terminal update --check`. `no release asset for this platform` — a platform with no asset is built from source, and `x-terminal update --check` names the asset it wanted. `the download did not verify` — do not retry in a loop: re-run `x-terminal update` once and treat a second failure as a diagnosis of the Release itself. `the install path is not writable` — the verb prints the resolved path; run the binary from a directory this user owns, or fix that directory's ownership. `this build is not a release` — install a published asset or build with the release workflow's `-ldflags` stamps, then `x-terminal update --check`. Two facts the verb owes the reader: the token is read from `GH_TOKEN` or `GITHUB_TOKEN` in the environment, is never written to a file, a keyring or a `config.toml` key, and is not needed at all for this repository; and a replacement takes effect at the next launch, not in the process that ran the verb. Nothing happens to the installed file on any failure path: every arm removes its temporary file and leaves the installed binary byte-identical.

**Verify.** `x-terminal update --check`
