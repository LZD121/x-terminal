# Milestones

## v1.0 v1.0 (Shipped: 2026-10-15)

**Phases completed:** 8 phases, 27 plans, 15 tasks

**Key accomplishments:**
- The repository is now a buildable Go module whose credential path is closed by construction: a `0600`-only, symlink-refusing session loader wrapped in a `Secret` type that renders as `<redacted>` through every formatting path, plus CLI dispatch and a fully tested config resolver that reads `ALL_PROXY` because the standard library does not.
- One proxied transport, one request choke point and one measured status table behind a `doctor` that names exactly which link of the path is broken — verified live over both the HTTP and the SOCKS5 form of this host's proxy, and over a dead port and no proxy at all, with the credential package's last formatting hole closed and a `logout` that deletes exactly one file.
- The measured request path as code: operation ids as embedded data with pins and a refresh tool, a registry-driven GraphQL client, parsers for the shapes the live site actually sends — proven by a live authenticated read from this host that printed `@example-viewer`, twenty posts with author, date and text, and a budget of two requests.
- The read seam landed: `domain.Source` with the content model and the closed failure taxonomy, `xsource.Adapter` over `xweb.Client.GraphQL`, a fixture-backed `fake.Source` proven equal to the recorded response, and `screen.List`'s pure reducers — pagination that cannot duplicate a post or loop on a repeated cursor, a refresh diff that reports "N new posts" without moving the selection, and a soft ban that ends the session for the process.
- `internal/text` lands the program's one width model — Width/Wrap/Pad/Truncate dispatching on `$TMUX`, with an A4 agreement suite over a 19-entry adversarial corpus in both environments and four proven-failing anti-vacuity arms.
- In-place URL fallbacks replace failed attachment nouns, wrap without truncation, and stay out of covered frames while notices name the outcome.
- `y`/`o` copy or open the permalink through injected Clip/Media seams; one dispatcher gate refuses all three cover-breaking actions under immersive with a grammar-local notice and zero side effects.
- `internal/audit` proves the tree's promises as pure checks over a file list, and `x-terminal audit` prints what each check inspected and exits non-zero on any finding.
- `doctor` reports the mirror, signer, media openers and budget as presence facts; `--probe` spends exactly one request; every failing line names a `RUNBOOK.md` section.
- `RUNBOOK.md` is the repair path's written half; every failing classifier state has a section; every screen since Phase 3 is stored under every cover at both widths.

---
