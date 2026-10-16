---
phase: 07-media-links-and-clipboard
plan: 01
type: summary
status: complete
requirements: [MEDIA-01, MEDIA-02]
verified: |
  gofmt -l internal cmd                         empty
  CGO_ENABLED=0 go build ./...                  ok
  CGO_ENABLED=0 go test ./... -count=1          all packages ok
  go test ./internal/media/... -count=1         ok (dispatch, no-shell, reaper)
  go test ./internal/tui/... -run 'TestProcessSpawnsAreOnlyInTermAndMedia|TestTheShellCannotReachIO|TestMediaAction' -count=1  ok
  planted os/exec in internal/render/copy.go → TestProcessSpawnsAreOnlyInTermAndMedia FAIL
    ("internal/render/copy.go imports os/exec, and only internal/term and internal/media may");
    revert → PASS
  grep -rn os/exec internal/ --include=*.go (non-test) → internal/term/tmux.go, internal/media/spawn.go only
  grep xdg-open|mpv internal/tui internal/keys → empty
  only internal/cli/feed.go constructs media.External
  git diff --stat internal/config go.mod go.sum → empty
unverified: |
  MEDIA-03 URL fallback rows (plan 02); REND-07 clipboard/browser and DSGT-10 gate (plan 03);
  live xdg-open/mpv smoke on a real pane (tests inject Run/LookPath).
---
# 07-01 SUMMARY — media seam and external open (MEDIA-01, MEDIA-02)

## 1. Renderer seam

`internal/media/renderer.go` — `Outcome` (OpenedImage/Video/GIF/Poster, NoOpener,
NoProgram, SpawnFailed; zero is not success), `Result{Outcome, Rows}`,
`Renderer.Open(ctx, ref) (Result, error)`. Doc cites TERMINAL-MEDIA-PROBE.md and
names `kitty.go` + one config value + one feed wiring line as the v1.x addition.
v1 ships no in-terminal image code; External returns no rows.

## 2. External dispatch

`internal/media/external.go` — `External{Opener, Env, LookPath, Run}`; image →
Opener + `[url]`; video/gif → fixed `mpv` + `[--no-terminal, url]`; `target(ref)`
prefers Playable. `spawn.go` — `StartAndReap` / `DefaultLookPath` (os/exec only
here beside term). Missing PATH → NoProgram (nil error); spawn/nonzero →
SpawnFailed. Reaper: Start returns, Wait in a goroutine.

## 3. Key, shell, wiring

`keys`: `ActionMedia`/`GroupMedia`, key `i` (avoids Phase 3 `m` and `1`–`4`).
`tui`: `Deps.Media`, `openMedia` / `mediaMsg` / notice map; empty media →
`no media in this post`. Notice copy: opened image/video/gif/poster frame,
`no opener - URL shown`, `no media in this post`. `cli/feed.go` builds
`media.External` from `cfg.MediaOpener` + StartAndReap.

## 4. Playable URL + recording renderer

`domain.MediaRef.Playable`; `xsource.playableOf` — highest-bitrate `video/mp4`,
else first HLS, else empty; missing variant url is a path-carrying ParseError.
`fake.Renderer` records refs for gate/tests. Fixture pages regenerated with
`-update` / `-update-saved` / `-update-profile` / thread `-update`.

## 5. Boundary

`TestProcessSpawnsAreOnlyInTermAndMedia` (was TestOsExecIsOnlyInInternalTerm):
both packages required present; planted third package observed failing; empty
importer anti-vacuity exercised.

Decision the plan settled: CONTEXT wins over UI-SPEC — no `video_opener` key;
mpv is fixed for this phase.
