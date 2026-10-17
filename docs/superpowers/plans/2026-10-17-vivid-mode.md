# Vivid Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a second, image-capable presentation of the same read-only x.com data, reachable only by typing the Konami sequence in plain mode, laid out as a left navigation column plus a list/thread reading area plus a status bar.

**Architecture:** A new `internal/vivid` package owns the frame geometry, the nav column, the post list and the reading pane as pure functions of `[]render.Block` plus a small view-model; `internal/tui` gains a fourth `State` and routes to it. Images are fetched in `internal/xweb` (the only package the transport audit lets build requests) on demand, decoded with the standard library, and placed by a new Kitty renderer behind the existing `media.Renderer` seam. Panic gains one extra step that deletes every placed image.

**Tech Stack:** Go 1.27, `charm.land/bubbletea/v2` v2.1.0, `charm.land/lipgloss/v2` (already a transitive dep — do not add it as a direct dep unless Task 7 needs styling the existing `render.Palette` cannot do), `golang.org/x/image` **only if** Task 6's JPEG path needs it (it does not; stdlib `image/jpeg` decodes).

**Spec:** `docs/superpowers/specs/2026-10-17-vivid-mode-design.md`

## Global Constraints

- Read-only: `internal/audit`'s read-only and transport checks pass **unmodified**. No write operation name appears in non-test source.
- No persistence: nothing about the feed, images or vivid-mode marks is written to disk. The audit's persistence check enumerates its destinations and this plan adds none.
- `internal/tui` may not import `net/http` and may not name `os/exec` — the module boundary tests enforce both (`TestProcessSpawnsAreOnlyInTermAndMedia`, and the import allow-list). All new subprocess and HTTP work lives in `internal/term` / `internal/media` / `internal/xweb`.
- All x.com and `*.twimg.com` traffic goes through the existing `xweb` transport (proxy-aware, budgeted). A missing proxy must fail loudly, as it already does.
- No timeline data on disk; the session cookie is never sent to `*.twimg.com` and never printed.
- English-only UI text.
- `TERM`/`TERM_PROGRAM` is the only graphics detection signal; never `KITTY_WINDOW_ID`. Under `$TMUX`, every escape is DCS-wrapped with its ESCs doubled.
- The plain mode, the four covers and the panic frame are unchanged, byte for byte. Their golden corpus must not move.
- No flag, config key or env var selects vivid mode. Its key sequence is not listed by `--help`, `?` or `x-terminal keys`.
- Every new screen gets golden-frame tests; every new gate gets a synthetic-violation test **and** a planted-in-tree proof, following the pattern in `internal/audit`.
- Fixtures and docs carry no real account name, address or machine path (use `$HOME`).

## Review Focus

Inputs the spec implies but does not spell out, most likely to bite first. Each has a test in the owning task.

1. **Keys typed mid-sequence in plain mode** — `b` opens bookmarks and `a` opens the author today. A half-typed Konami sequence must not fire either. (Task 3)
2. **Narrow terminal under vivid mode** — below the two-column floor the layout must degrade to one column rather than truncate or panic. (Task 4)
3. **Panic while an image is on screen** — the feed must be gone *and* the terminal's image store empty. (Task 8)
4. **A terminal with no graphics support** — a post with an image must print a placeholder and still be usable, in and out of tmux. (Task 6)
5. **Image fetch failure or oversize payload** — must surface as a placeholder, never as a hung request or unbounded memory. (Task 5)

---

## File Structure

| File | Responsibility |
|---|---|
| `internal/vivid/model.go` | The vivid view-model: nav index, split geometry, mark sets. Pure. |
| `internal/vivid/nav.go` | The nav column's rows and hit order. |
| `internal/vivid/list.go` | The post list column: row assembly from `render.Block`s plus marks. |
| `internal/vivid/reading.go` | The reading pane: thread blocks, and where an image block sits. |
| `internal/vivid/statusbar.go` | The bottom bar's fields and their drop order. |
| `internal/vivid/katakana.go` | **Not created** — no invented vocabulary lives here; vivid mode reuses `render`'s copy. |
| `internal/keys/keys.go` | Add the `vivid` and `plain` actions; keep them out of `Rows()`' documented output only if Task 3's test proves the listing rule needs it. |
| `internal/tui/konami.go` | The sequence matcher and the held-key rule. |
| `internal/tui/vivid.go` | Wiring: enter/leave, per-screen vivid assembly, image request commands. |
| `internal/tui/model.go` | `StateVivid`, `vivid` marks, `images` bookkeeping fields. |
| `internal/xweb/image.go` | `ImageFetcher`: GET a `twimg` URL through the budgeted transport, no cookie, caps, returns bytes. |
| `internal/media/kitty.go` | Kitty placement: encode, transmit, place by placeholders, delete all. |
| `internal/media/kitty_payload.go` | The pure escape-string builders (transmit, place, delete), testable without a terminal. |
| `internal/term/term.go` | `EnterPanic` gains the image-delete step, DCS-wrapped. |
| `internal/render/vivid_test.go` | **No** — vivid frames are golden-tested from `internal/vivid`. |

Task order: 1 (state + entry) → 2 (matcher) → 3 (keys) → 4 (layout) → 5 (fetch) → 6 (kitty) → 7 (wire screens) → 8 (panic) → 9 (audit, docs).

---

### Task 1: The fourth state and its entry point

**Files:**
- Modify: `internal/tui/model.go:38-100` (State consts, `String`, `StateOf`, `Display`)
- Modify: `internal/tui/view.go:70-90` (`title`, `covered`, `grammar`, `Render`)
- Test: `internal/tui/state_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `tui.StateVivid`; `func (m Model) enterVivid() Model`; `func (m Model) leaveVivid() Model`. `StateOf("vivid")` must **not** accept the name: `StateOf` is the config file's vocabulary and `Display.State` is what the shell starts in, so accepting it there would make a config key a way in. The state is reachable only through `enterVivid`, and Task 9 adds the audit check that says so. Its `String()` still names it `"vivid"` for test failure messages.

- [ ] **Step 1: Write the failing test**

```go
func TestVividIsAStateButNotAStartingState(t *testing.T) {
	m := newTestModel(t)
	if m.state != StateNormal { t.Fatal("must start plain") }
	got := m.enterVivid()
	if got.state != StateVivid { t.Fatalf("state = %v", got.state) }
	if !strings.Contains(got.Render(100, 30), "") { /* frame renders */ }
	if _, ok := StateOf("vivid"); !ok { t.Fatal("StateOf must name it") }
	// and it must not survive as a display: config vocabulary is separate — see Task 9.
}
```

- [ ] **Step 2: Run it, expect failure**

Run: `CGO_ENABLED=0 go test ./internal/tui/ -run TestVividIsAStateButNotAStartingState -count=1`
Expected: `undefined: StateVivid`.

- [ ] **Step 3: Implement the state and the two transitions**

`StateVivid` after `StateImmersive`, before `StatePanic` (panic must stay last: `from` restores into any of the others). `title()` returns `render.CopyTitleVivid` — a new constant in `internal/render/panic.go` beside the other two, value `"viewer"` (a command name that reads as work software; the builder may pick another noun, but it must be a real command name and must not be a product name). `covered()` returns `""` in vivid mode (no cover is worn); `grammar()` returns a new `render.Vivid` grammar value declared in `internal/render/grammar.go` — same fields as `Plain` but `ShowCounts: true`, `MeasureCap: 0`, `Separator: false`, and `Identity` printing the name and handle as the plain one does.

- [ ] **Step 4: Run the test, expect pass**

- [ ] **Step 5: Commit**

```bash
git add internal/tui/model.go internal/tui/view.go internal/tui/state_test.go internal/render/grammar.go internal/render/panic.go
git commit -m "feat(vivid): a fourth display state, plain at start"
```

---

### Task 2: The Konami matcher and the held-key rule

**Files:**
- Create: `internal/tui/konami.go`
- Test: `internal/tui/konami_test.go`

**Interfaces:**
- Produces: `type konami struct{ ... }` with `func newKonami() konami`, `func (k konami) advance(key string) (konami, konamiEffect)` where `konamiEffect` is `konamiNone | konamiHeld | konamiEnter | konamiReplay`. Method `func (k konami) advance` is pure: a model value in, a model value out.

- [ ] **Step 1: Write the failing test — the full table**

```go
func TestKonamiHoldsOnlyTheFinalB(t *testing.T) {
	cases := []struct{ typed []string; want konamiEffect; replay []string }{
		{[]string{"up","up","down","down","left","right","left","right","b"}, konamiHeld, nil},
		{[]string{"up","up","down","down","left","right","left","right","b","a"}, konamiEnter, nil},
		{[]string{"up","up","down","down","left","right","left","right","b","c"}, konamiReplay, []string{"b"}},
		{[]string{"b"}, konamiNone, nil},
		{[]string{"up","up","x"}, konamiNone, nil},
	}
	...
}

func TestKonamiIsNotReachableByPaste(t *testing.T) {
	// tea.KeyPressMsg carries no paste flag; a paste arrives as a bracketed-paste message the
	// shell already routes elsewhere. The test asserts the matcher is only fed from press().
}
```

- [ ] **Step 2: Run, expect failure**

- [ ] **Step 3: Implement**

The state is the count of matched keys, 0..9. Every key that matches the next expected key advances; anything else resets to 0 **except** that a `b` at index 8 is *held*: `advance` reports `konamiHeld` and the caller does not dispatch it. If the next key is not `a`, the caller replays the held `b` (i.e. dispatches `ActionBookmarks`) and then processes the new key normally. A `b` that does not follow the eight arrows is an ordinary key.

- [ ] **Step 4: Run, expect pass**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(vivid): the entry sequence, with only its last key held"
```

---

### Task 3: Keys routed through the matcher without changing the keymap listing

**Files:**
- Modify: `internal/tui/update.go:229-250` (`press`)
- Modify: `internal/keys/keys.go` (add `ActionVivid`, `ActionPlain`; **not** in `rows`)
- Test: `internal/tui/dispatch_test.go`, `internal/keys/keys_test.go`

**Interfaces:**
- Consumes: `konami` from Task 2.
- Produces: `keys.ActionVivid`, `keys.ActionPlain`.

- [ ] **Step 1: Write the failing test**

```go
func TestTypingTheSequenceDoesNotOpenBookmarks(t *testing.T) {
	// Drive the model with the eight arrows and `b` using the real press(); assert no frame was
	// pushed, no request cmd was returned, and the top frame is still home.
}
func TestTheListingDoesNotMentionVivid(t *testing.T) {
	for _, r := range keys.Rows() { if r.Action == keys.ActionVivid || r.Action == keys.ActionPlain { t.Fatal("documented") } }
	if strings.Contains(keysJoinedForCLI(), "vivid") { t.Fatal("keys subcommand must not print it") }
}
```

- [ ] **Step 2: Run, expect failure**

- [ ] **Step 3: Implement**

In `press`, before the `keyAction` lookup and only while `m.state == StateNormal` and `m.prompt == nil`: feed the keystroke to the matcher. `konamiHeld` returns `(m, nil)`. `konamiEnter` returns `m.enterVivid()`. `konamiReplay` dispatches the normal action for `b` and falls through for the current key. In vivid mode, `ActionPlain` (bound to `esc`) returns to plain, and `ActionPanic` behaves as it does today (Task 8 adds the image delete).

- [ ] **Step 4: Run, expect pass**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(vivid): entry and exit keys, absent from every listing"
```

---

### Task 4: The two-column frame

**Files:**
- Create: `internal/vivid/model.go`, `internal/vivid/nav.go`, `internal/vivid/list.go`, `internal/vivid/reading.go`, `internal/vivid/statusbar.go`
- Test: `internal/vivid/model_test.go` (geometry table), `internal/vivid/golden_test.go` (golden frames)
- Testdata: `internal/vivid/testdata/*.golden`

**Interfaces:**
- Consumes: `render.Block`, `render.Grammar`, `domain.Post`.
- Produces:
  ```go
  type View struct {
      Nav      []NavRow      // destination rows, in order
      Current  int           // index into Nav
      List     []render.Block
      Reading  []render.Block
      Marks    Marks
      Status   render.StatusFields
      Width    int
      Height   int
  }
  func Render(v View, pal render.Palette) string
  type Marks struct { New, Opened map[domain.PostID]bool }
  const MinTwoColumn int = 60 // cells; below this, one column
  ```

- [ ] **Step 1: Write the failing geometry test**

A table of `(w,h) → (navW, listW, readingW)` including `59 → one column`, `60 → 12/24/24`, `200 → 20/~90/~90`. Assert `navW+listW+readingW+2 == w` exactly (two separator columns) and that every produced row is exactly `w` cells at every case.

- [ ] **Step 2: Run, expect failure**

- [ ] **Step 3: Implement the split and the columns**

Nav column: one row per destination, current one marked with the selection mark, in the existing `render` vocabulary. List column: `render.Blocks` output, each row prefixed with a mark column when the post is new or opened (new = a `+` in the mark column; opened = the row painted in the muted role). Reading column: the thread's blocks, scrolled independently. Status bar: reuse `render.StatusLine` for the right half and put the nav-driven breadcrumb on the left.

- [ ] **Step 4: Write the golden test and record frames**

`TestVividGolden` renders a home frame, a thread frame and a narrow frame from fixtures and compares to `testdata/*.golden`, following `internal/render/golden_test.go`'s pattern.

- [ ] **Step 5: Run, expect pass**

- [ ] **Step 6: Commit**

```bash
git commit -am "feat(vivid): nav column, list, reading pane and status bar"
```

---

### Task 5: Fetching image bytes without the cookie

**Files:**
- Create: `internal/xweb/image.go`
- Test: `internal/xweb/image_test.go`

**Interfaces:**
- Consumes: `xweb.NewTransport`, `xweb.Config`.
- Produces:
  ```go
  type ImageFetcher struct { ... }
  func NewImageFetcher(cfg Config, maxBytes int64) (*ImageFetcher, error)
  func (f *ImageFetcher) Get(ctx context.Context, url string) ([]byte, string, error) // bytes, content type
  ```
  Errors: `ErrTooLarge`, `ErrNotAnImage`, and the transport's own.

- [ ] **Step 1: Write the failing tests** (an `httptest` server; no network)

```go
func TestImageFetchSendsNoCookie(t *testing.T)          // assert the request carries no Cookie and no Authorization
func TestImageFetchIsCapped(t *testing.T)               // a body over maxBytes returns ErrTooLarge, not a partial read
func TestImageFetchRejectsANonImageContentType(t *testing.T)
func TestImageFetchRefusesANonTwimgHost(t *testing.T)   // host allow-list: *.twimg.com only
```

- [ ] **Step 2: Run, expect failure**

- [ ] **Step 3: Implement** — using the module's existing transport constructor; `http.Client{Transport: …}` with the fetcher's own short timeout; `io.LimitReader(body, max+1)`; no cookie jar.

- [ ] **Step 4: Add the audit arm** — `internal/audit/readonly.go`'s transport check already enumerates request builders; `image.go` lives in an approved package. Add a test asserting `image.go` contains no `Cookie`/`Authorization` set and that the host check is present. (This is the planted-in-tree proof.)

- [ ] **Step 5: Run, expect pass**

- [ ] **Step 6: Commit**

```bash
git commit -am "feat(xweb): a cookie-free, capped image fetch behind the transport audit"
```

---

### Task 6: The Kitty renderer

**Files:**
- Create: `internal/media/kitty_payload.go`, `internal/media/kitty.go`
- Test: `internal/media/kitty_payload_test.go`, `internal/media/kitty_test.go`

**Interfaces:**
- Consumes: `media.Renderer` (already defined), `media.Result`.
- Produces:
  ```go
  type Kitty struct { Out io.Writer; Tmux bool; NextID func() int }
  func (k *Kitty) Open(ctx context.Context, ref domain.MediaRef) (Result, error) // returns Rows: none; the pane places
  func (k *Kitty) OpenURL(...) (Result, error)                                    // unsupported outcome, as today
  func (k *Kitty) Draw(w io.Writer, id int, png []byte, cols, rows int) error
  func (k *Kitty) DeleteAll(w io.Writer) error
  func KittyTransmit(id int, png []byte) string
  func KittyPlace(id, cols, rows int) string
  func KittyDeleteAll() string
  ```
  `kitty_payload.go` holds the three pure string builders; `kitty.go` does the writing and wrapping.

- [ ] **Step 1: Write the failing payload tests** — byte-exact expectations, both bare and `Tmux`:

```go
func TestTransmitIsChunkedAt4096(t *testing.T)
func TestPlaceUsesPlaceholders(t *testing.T)   // contains U+10EEEE, c=, r=, i=
func TestTmuxWrappingDoublesEveryEscape(t *testing.T)  // ESC P tmux; … and no undoubled ESC inside
func TestDeleteAllHasNoPlacement(t *testing.T)
```

- [ ] **Step 2: Run, expect failure**

- [ ] **Step 3: Implement the builders**, using the row/column diacritic table (16 rows, 16 columns is enough; the builder may use the full 255 of kitty's table but must document the index).

- [ ] **Step 4: Write the spike-verification test** — a `TestKittyPayloadRendersInTmux` that, when `X_TERMINAL_CAPTURE=1` **and** a display is present, writes a payload into a tmux pane and screenshots it, asserting the image is present and the row above/below is not overdrawn. Mirror `internal/render/capture_tmux_test.go`. This settles design risk (a) if a human runs it; it skips with its reason otherwise. A manual run of it is also the first thing to do after Step 5, per the spec's "first implementation step for images".

- [ ] **Step 5: Run, expect pass**

- [ ] **Step 6: Commit**

```bash
git commit -am "feat(media): Kitty placement by Unicode placeholder, tmux-wrapped"
```

---

### Task 7: Vivid screens wired to the existing data

**Files:**
- Create: `internal/tui/vivid.go`
- Modify: `internal/tui/model.go` (marks + image bookkeeping), `internal/tui/view.go` (`Render` for `StateVivid`)
- Test: `internal/tui/vivid_test.go`, extend `internal/tui/corpus_test.go`

**Interfaces:**
- Consumes: `vivid.View` (Task 4), `xweb.ImageFetcher` (Task 5), `media.Kitty` (Task 6).
- Produces: `func (m Model) vividView() vivid.View`; `type imageState struct { id int; rows int; post domain.PostID }`; `func (m Model) imageCmd(post domain.PostID) tea.Cmd`.

- [ ] **Step 1: Write the failing tests**

```go
func TestVividShowsTheSamePostsAsPlain(t *testing.T)      // same IDs, same order
func TestEveryScreenKindHasAVividFrame(t *testing.T)      // home, thread, search, profile, saved — mirrors cover_e2e's join
func TestImagesAreOnlyRequestedForTheReadingPane(t *testing.T)
func TestASecondOpenOfTheSamePostDoesNotRefetch(t *testing.T)
```

- [ ] **Step 2: Run, expect failure**

- [ ] **Step 3: Implement the wiring.** Each frame kind gets a vivid assembly: home/search/profile/saved = list + reading (reading empty until a post is opened); thread = list is the conversation's ancestors and reading holds the rest. Marks: on a refresh, `New[post] = true` for ids in `List.NewRows()`' range; on open, `Opened[post] = true`. Both cleared on leaving vivid mode.

- [ ] **Step 4: Run, expect pass, and run the whole suite**

Run: `CGO_ENABLED=0 go test ./... -count=1`
Expected: pass; `internal/render`'s goldens unmoved.

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(vivid): every screen, with new/opened marks and on-demand images"
```

---

### Task 8: Panic deletes the images

**Files:**
- Modify: `internal/term/term.go:81-100` (`Entry`/`EnterPanic`), `internal/tui/update.go` (panic path)
- Test: `internal/term/term_test.go`, `internal/render/panic_test.go` (extend)

**Interfaces:**
- Consumes: `media.KittyDeleteAll()`.
- Produces: `func (s Sink) EnterPanic(frame, title string)` unchanged in signature; its `Entry` gains the delete payload **before** ED 3.

- [ ] **Step 1: Write the failing test**

```go
func TestPanicEntryDeletesImagesBeforeClearingTheScreen(t *testing.T) {
	e := Entry(frame, title)
	if i, j := strings.Index(e, kittyDeleteAll), strings.Index(e, eraseScrollback); i < 0 || i > j {
		t.Fatal("delete must precede the scrollback clear")
	}
}
func TestPanicUnderTmuxWrapsTheDelete(t *testing.T)  // the delete payload is DCS-wrapped
```

- [ ] **Step 2: Run, expect failure**

- [ ] **Step 3: Implement** — `Entry` takes the delete string as a parameter (or the sink prepends it); the tmux wrapping reuses the same helper as Task 6 rather than a second copy.

- [ ] **Step 4: Run, expect pass**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(panic): the image store is emptied before the screen"
```

---

### Task 9: The declared exception, the docs, and the tree

**Files:**
- Modify: `internal/render/cover_test.go`, `corpus_test.go`, `status_test.go` (the banned-vocabulary tests) — add the vivid exception **as a named set of screens**, keeping every existing assertion for plain/cover/panic
- Modify: `.planning/PROJECT.md` (one paragraph naming the exception)
- Modify: `README.md` (a short "Vivid mode" section: what it is, how it is reached — **or deliberately not saying**, see below)
- Modify: `internal/audit/audit.go` + test: a new check that `StateOf` and the config loader cannot produce `StateVivid`
- Test: `internal/audit/audit_test.go` (synthetic violation + planted-in-tree)

**Interfaces:**
- Consumes: everything above.
- Produces: no API.

- [ ] **Step 1: Write the failing audit test** — a planted tree where `Display.State` can be `vivid` must fail; the real tree must pass.
- [ ] **Step 2: Run, expect failure**
- [ ] **Step 3: Implement the check and the doc edits.** Decision for the builder: `README.md` is public — the design says the entry is hidden. Write the section as a statement that a second mode exists and is not documented, **without** the sequence. Do not document the sequence anywhere in the repo.
- [ ] **Step 4: Run the whole gate**

Run: `CGO_ENABLED=0 go test ./... -count=1 && go vet ./... && gofmt -l internal cmd && go run ./cmd/x-terminal audit`
Expected: all pass; `audit` reports 11 checks, 0 findings.

- [ ] **Step 5: Commit**

```bash
git commit -am "docs(vivid): the declared exception, and a gate that keeps it hidden"
```

## Self-review notes

- Spec coverage: entry/exit/panic (Tasks 2, 3, 8), layout B (4), images (5, 6), all six screens (7), declared exception and hidden entry (9), read-only and no-persistence (5, 9 and the unmodified audit).
- Not a separate task, deliberately: the narrow fallback below the two-column floor is Task 4's Step 1 table.
- Two spec risks are not settled by code: scrolling an image out of the pane, and Ghostty's storage limit with many images. Task 6 Step 4's capture test is where the first gets settled; the second is bounded by Task 5's in-memory cap and is a manual check before release.
