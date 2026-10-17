# x-terminal

A read-only x.com client that runs inside a terminal pane.

It renders timelines, threads, search results, profiles and the bookmark/like lists as plain text,
with no app-like chrome: it should read like a log viewer, not a social app. Single user, personal
tool — built for one person who already lives in a terminal and wants the feed without the browser.

## What this is not

**This is not a product.** It talks to x.com's internal web GraphQL endpoints with the browser's own
session cookie — the only zero-cost path that preserves full read fidelity. Using those endpoints is
against X's terms of service, so no container image, no hosted instance, no installer and no package
recipe exist or will be added, and `internal/audit`'s `distribution` check fails the build if one
appears.

Be clear about what is on this page, though: the repository and its Release are **public**, so anyone
can clone the source and download a ready-to-run binary for Linux or macOS, and there is no support,
no roadmap and no release cadence behind any of it. That is the author's own machine and account this
is written for — the public release is a convenience for the case where the author is on another
one, and its exposure was accepted deliberately. If X's terms matter to you, they matter here: this
is not offered as something for you to run.

It is also strictly **read-only**: no post, reply, like, repost, follow or bookmark write call exists
anywhere in the build, and a repo-wide audit proves it.

## What it does

- **Timelines** — the Following and For-you lists, keyset-paginated, refreshed by hand.
- **Threads** — a post's ancestors, the focal post, its self-thread and its replies, with tombstones,
  limited-visibility posts and partially-reachable threads labelled honestly.
- **Search and profiles** — keyword search with a latest/top toggle, people lookup, and a profile's
  bio and recent posts.
- **Bookmarks and likes** — read from the session, mirrored to a local `saved.db` so they open with
  the network down. The store holds those two lists and nothing else.
- **Media and links** — a post's images and video open in external programs (`xdg-open`, `mpv`), its
  permalink can be copied or opened. When no opener is available the URL is printed instead.
- **Translations** — `T` shows the selected post through x.com's own translator, in the language
  `config.toml`'s `translate.language` resolves. The translation is a property of the selection:
  moving off the post reverts it, and nothing about it is written anywhere.
- **A disguise** — three display states: normal, *immersive* (the same feed rendered through one of
  four work-software covers, with author identities aliased) and *panic* (the pane cleared, terminal
  scrollback and tmux history wiped). One key each way.
- **`doctor` and a runbook** — every way the upstream can break is a named state with a named repair
  in [`RUNBOOK.md`](RUNBOOK.md).

## Requirements

- **Go 1.27+** to build. The shipping build is `CGO_ENABLED=0` — a static binary, no cgo anywhere.
- **Linux** (developed and measured on Arch + Ghostty + tmux). macOS should build; nothing has been
  measured there.
- **A route to x.com.** On a network where x.com is unreachable directly, a local proxy (the author
  uses Clash/mihomo on `127.0.0.1:7890`-ish ports) and the session cookie from a logged-in browser.
- **A terminal that speaks the Kitty keyboard protocol** for the best behaviour; Ghostty is what the
  project was built against. Nothing here draws images inside the pane — media opens externally.

## Build and run

```sh
CGO_ENABLED=0 go build -o x-terminal ./cmd/x-terminal

x-terminal                 # the feed (runs the pane)
x-terminal doctor          # is the network path, the session and the budget OK?
x-terminal audit           # check the tree against every promise this build makes
x-terminal sync --kind bookmarks   # fill the local mirror
x-terminal whoami          # which account the session belongs to
x-terminal keys            # the keymap, tab-separated
```

## A second display

There is a second presentation besides the plain reader: a navigation column, a list and a reading
pane side by side, with images drawn in the terminal. It is meant for the places where the reader
needs no cover, and it is deliberately not documented further — this file does not say how to reach
it, and neither does `--help`, the `?` overlay or `x-terminal keys`.

What it is not: it is still read-only, it still keeps nothing on disk, and it still walks out through
the same proxy and the same request budget. The plain reader, the covers and the panic frame are
unchanged, and their golden frames are unmoved.

The session cookie goes in `$XDG_CONFIG_HOME/x-terminal/session` — `auth_token=…` and `ct0=…`, mode
`0600`, outside any editor round-trip. `X_TERMINAL_AUTH_TOKEN` / `X_TERMINAL_CT0` override it.
[`RUNBOOK.md`](RUNBOOK.md) has the one-command repairs for every failure state the classifier can
produce, including what to do when the cookie expires.

## CI and releases

- **`.github/workflows/ci.yml`** runs on every push and pull request: `go build`, `go vet`, `gofmt`,
  the full test suite with `CGO_ENABLED=0`, and `x-terminal audit` — the project's own promise
  checks, which is the gate that matters. No step touches x.com; the suite is hermetic.
- **`.github/workflows/release.yml`** runs on a `v*` tag: it re-runs the same gate, cross-compiles
  static binaries for Linux (amd64, arm64) and macOS (arm64), stamps each with the tag and commit it
  came from, writes per-asset `sha256` files and a combined `SHA256SUMS`, and attaches them to the
  repository's own GitHub Release. It builds no image, no package and no installer, on purpose.

A release binary says what it is:

```sh
$ x-terminal --version
x-terminal v1.0.0 (98ed0af72547) go1.27.2 linux/amd64
```

## Layout

```
cmd/x-terminal/       the process boundary only
cmd/x-opsgen/         a development tool: mines the operation-id registry
cmd/spike/            a development tool: the live probe that measured the payload shapes
internal/cli/         argument handling, dispatch and the verbs' bodies
internal/config/      XDG paths, the proxy block, the reader
internal/session/     the credential: a 0600 file, symlink-refusing, redacted by construction
internal/xweb/        the transport, the limiter, the classifier, the volatile registry
internal/xsource/     the GraphQL client and the parsers, with recorded fixtures
internal/domain/      the content model and the closed failure taxonomy
internal/screen/      the pure reducers: the list, threads, search, profiles, the saved frame
internal/render/      the width model's consumers: blocks, grammars, the covers, the golden frames
internal/tui/         the shell: the event loop, the frame stack, the keymap dispatch
internal/text/        the one width model (Width/Wrap/Pad/Truncate)
internal/media/       the external-opener seam and the clipboard chain
internal/store/       the mirror-only SQLite store (pure Go, no cgo)
internal/audit/       the tree's promise checks, as a package and a verb
internal/term/        the tmux/terminal escape seam
internal/fake/        the fixture-backed test double every screen test is driven by
RUNBOOK.md            the repair path: one section per failure state
AGENTS.md             the planning record and the research the design rests on
```

## Testing

```sh
go test ./... -count=1
```

Roughly 1,500 tests across 22 packages. The interesting ones are the ones that can fail for the right
reason: golden frames that the corpus audit checks over their stored bytes, absence arms that fail on
a planted leak, and the audits in `internal/audit` — every check there has both a synthetic-violation
test and a planted-in-tree proof.

Two things a test cannot do on a machine with no session and no display, and which therefore skip
with their reason printed:

```sh
X_TERMINAL_CAPTURE=1 go test ./internal/xsource/ -run TestCapture -v   # refresh the fixtures
go test ./internal/render -run TestCapturePanic -v                    # the panic leg in a real window
```

## License

None. This is a personal tool, published for its author's use; see "What this is not" above.
