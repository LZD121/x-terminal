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

**X lists are not readable, and not because a screen is missing.** Measured on 2026-10-17 against the
web build this tree pins: the bundle carries no operation that returns a list's timeline (the only
`list`-named reads are `ListSearchTimeline`, `UserSessionsList` and an audio-space privacy update),
and `GenericTimelineById` — the generic path — answers `Internal server error at timeline` for a
real list id in both `list-<id>` and bare forms. Nothing here draws a list screen because nothing
here could fill one.

## What it does

- **Timelines** — the Following and For-you lists, keyset-paginated, refreshed by hand.
- **Threads** — a post's ancestors, the focal post, its self-thread and its replies, with tombstones,
  limited-visibility posts and partially-reachable threads labelled honestly.
- **Search and profiles** — keyword search with a latest/top toggle, people lookup, and a profile's
  bio and recent posts.
- **Bookmarks and likes** — read from the session, mirrored to a local `saved.db` so they open with
  the network down. The store holds those two lists and nothing else.
- **Explore** — `x` lists what is trending, and `enter` on a row searches it. One request, one
  screen; the list is not paginated because Explore's own is not.
- **What a post names** — `L` lists the URLs, hashtags and accounts in the selected post's body,
  numbered, and a digit follows one: a URL leaves through the same opener the permalink uses, a tag
  opens a search, a mention opens that profile. Refused under a cover, where a list of addresses is
  exactly what the disguise exists to keep off screen.
- **One account's lists** — from a profile, `f` opens who it follows, `e` who follows it, `v` the
  verified among them, and `M` its media tab. Each is its own keyset-paginated frame; `q` walks back
  without spending a request.
- **Pictures in the reading pane** — in the second display, a post opened in the reading column has
  its first image drawn in the terminal, through the Kitty graphics protocol with Unicode
  placeholders (`TERM`/`TERM_PROGRAM`, or tmux's client term name behind a multiplexer). Anything the
  terminal cannot draw, and every video or gif, stays a line of text with the external-open hint.
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

## Install from a release

One command, which works out the platform, downloads the matching release asset and its published
digest, verifies the digest, and installs the binary into `$HOME/.local/bin`:

```sh
curl -fsSLO https://github.com/LZD121/x-terminal/releases/latest/download/install.sh
sh install.sh
```

`--prefix DIR` installs somewhere else, `--version v1.4.0` pins a release, and `PREFIX` /
`X_TERMINAL_VERSION` do the same through the environment. It needs no root, writes one file, and never
touches the session or the configuration. A proxy is whatever `curl` already honours from
`https_proxy`; the script never guesses one.

The script is the **one installer-shaped file this project permits**, and `internal/audit`'s
`distribution` check names it by exact path and prints it in its promise. Everything else in that
family — a package recipe, an image, a second `install-*.sh` — is still a finding.

If you would rather not run a script, these are the same steps by hand:

```sh
cd "$(mktemp -d)"
curl -LO https://github.com/LZD121/x-terminal/releases/latest/download/SHA256SUMS
asset=x-terminal_v1.4.0_linux_amd64          # or _linux_arm64 / _darwin_arm64
curl -LO "https://github.com/LZD121/x-terminal/releases/latest/download/$asset"
curl -LO "https://github.com/LZD121/x-terminal/releases/latest/download/$asset.sha256"
sha256sum -c "$asset.sha256"                  # macOS: shasum -a 256 -c
install -Dm755 "$asset" ~/.local/bin/x-terminal
x-terminal --version
```

After that, `x-terminal update` upgrades in place and verifies the published digest itself; `--check`
asks without installing.

## Build and run

```sh
CGO_ENABLED=0 go build -o x-terminal ./cmd/x-terminal

x-terminal login           # read the two session cookies once, check them, write the file at 0600
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

## Accounts

More than one x.com account can live in `$XDG_CONFIG_HOME/x-terminal/sessions/`, one `0600` file per
name, with the original single `session` file kept as the account called `default`:

```sh
x-terminal login --as work        # write a named account (--browser works the same way)
x-terminal logout --as work       # remove one
account = "work"                  # config.toml: which one to start on
```

`A` in the pane switches to the next account. A switch is a new shell, not a mutation: the frame
stack, the walks, the translations and the failure are dropped, because every one of them belongs to
the account that produced them; the client is rebuilt with that session's own request budget; and the
status line names the account outside a cover (a covered line carries no identity at all). The saved
mirror is one file with one owner — reading it as another account reports `wrong account` rather than
showing someone else's list.

`login` is the guided path. By default it names the two cookies a logged-in browser already has and
reads them with no echo; `login --browser` instead opens a throwaway Chromium window on the login
page, lets you sign in there, and reads the cookies out of that window's own jar over a DevTools pipe
— never a debugging port, and never your own profile, which is neither read nor locked. Either way it
spends one request to check the pair against x.com and writes the file only if that request is
accepted. `--no-verify` writes without the check; `--replace` writes over a pair that is already
there.

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
