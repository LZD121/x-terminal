---
phase: 261016-0mu
plan: 01
type: summary
status: complete
quick-task: translate-the-selected-post-through-x-co
commits:
  - d52fa1c  # 261016-0mu-01, the endpoint, the parser and the port (previous executor)
  - 1b71178  # 261016-0mu-02, the key, the selection-scoped view and the two notices
  - 42175be  # 261016-0mu-03, the language, the composition root, the doctor fact and the repair sentence
verified: |
  CGO_ENABLED=0 go test ./... -count=1          1670 passed in 22 packages
  CGO_ENABLED=0 go vet ./...                    clean
  gofmt -l internal cmd                         empty
  go run ./cmd/x-terminal audit                 10 checks, 0 findings, 951 files inspected
  git diff --stat go.mod go.sum                 empty
  git diff --name-only d5d63a5 -- internal/render/testdata/TestGoldenFrames
                                                10 files, all help_*, none else (0 non-help)
  git diff --name-only d5d63a5 -- internal/tui/testdata
                                                empty (the shell's own corpus holds no help frame)
  x-terminal keys                               prints "T  translate  translate the selected post"
  (built binary)                                translate zh-CN on the doctor's config line
unverified: |
  Nothing this plan asked for is unrun. The pane proof ran on this host at 2026-10-16 02:14-02:16
  local (2026-10-15T18:14-18:16Z) against ~/.local/bin/x-terminal rebuilt by this executor, and the
  gated upstream measurement behind Task 1 was run by the previous executor on 2026-10-16 and is on
  the record in .planning/research/UPSTREAM-PROBE.md.
  The four "one row per capture" pane rows below are the rows that carry the evidence; the seven
  complete captures (40 rows each) are at /tmp/xt-tr-proof/c*.txt on the host, with the mtimes
  recorded beside them, and capture 3 is transcribed in full.
---

# 261016-0mu SUMMARY — one key translates the selected post, and moving reverts it

## 0. What shipped

| Task | Commit | What |
|---|---|---|
| 1 | `d52fa1c` | `internal/xweb/translate.go` (the measured REST GET), `apiHeaders` extracted out of `graphql.go`, `internal/xsource/translate.go` (`ParseTranslation` + the adapter), `internal/domain/translate.go` (the port), `+5` lines of `TranslateLanguage` plumbing in `config.go`, the dated record in `.planning/research/UPSTREAM-PROBE.md` |
| 2 | `1b71178` | `T` as `ActionTranslate` in the one keymap table (31 rows), `Model.translations` + the swap in `blocksIn`, `internal/tui/translate.go`, the `update.go` arm/key/handled set, `NoticeTranslated`/`NoticeNoTranslation` with a covered pair per style, `internal/fake/translate.go`, `internal/tui/translate_test.go`, the 10 regenerated help goldens |
| 3 | `42175be` | `translate.language` resolved in `internal/config` (file → `LC_ALL`/`LANG` → `zh-CN`, with the line-numbered refusal), the `feedTranslate` composition seam + `Deps.Translate`/`Deps.TranslateLang`, the doctor's `translate <lang>` presence fact, `RUNBOOK.md#request-shape-drift`'s amended Repair, the README bullet, the live pane proof |

## 1. The measurement trail (Task 1, run by the previous executor on 2026-10-16)

Every request went through `internal/xweb`'s own client (`Client.Do`, so through the limiter), never
through curl with hand-assembled headers. The gated test printed, verbatim:

```
translate_test.go:326: the language this host resolves: zh-CN
translate_test.go:344: product path: Available=true, translation runes=117 (never the text)
translate_test.go:355: suffixed form: HTTP 404, classified "endpoint retired", body 68 bytes
```

Command: `X_TERMINAL_CAPTURE=1 HTTPS_PROXY=http://127.0.0.1:7897 CGO_ENABLED=0 go test ./internal/xsource/ -run TestCaptureTranslation -count=1 -v`.
The test skips without the gate, so a normal `go test ./...` opens no socket.

The URL as this build assembles it (`internal/xweb/translate.go`, `translateURL`), for the post the
record is about and the language the config resolves:

```
GET https://api.x.com/1.1/strato/column/None/tweetId=2110709021544153425,destinationLanguage=Some(zh-CN),translationSource=Some(X),feature=None,timeout=None,onlyCached=None/translation/service/translateTweet
```

No `.json` suffix — the suffixed form answered `404 {"code":34,"message":"Sorry, that page does not
exist"}`, and the empty fourth argument of the bundle's own `apiClient.get(path, {}, headers, "")` is
what removes it. Measured statuses: `200` for `zh-CN`/`de`/`ja` on a live post, `200` with
`translationState:"Missing"` and an empty `translation` for the id `1`, `404 {"code":34}` for the
suffixed path. No transaction-id header was needed; the header set is the eight `apiHeaders` sets for
every authenticated API request of this client. Full record: `.planning/research/UPSTREAM-PROBE.md`
(section dated 2026-10-16).

## 2. The live pane proof (Task 3, run by this executor)

Binary: `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o ~/.local/bin/x-terminal ./cmd/x-terminal`,
rebuilt at 2026-10-16 02:14 local before the first capture.

Pane: a tmux server on its own socket (`tmux -L xttr -f <conf> new-session -d -x 120 -y 40` with
`set-option -g status off`), so the operator's own tmux was untouched and the server was killed at the
end. Environment for every capture, recorded verbatim:

```
XDG_CONFIG_HOME=/tmp/xt-tr-proof/config        # scratch, holding translate.language = "zh-CN"
XDG_DATA_HOME=/tmp/xt-tr-proof/data            # scratch, so no saved.db of the operator's is opened
X_TERMINAL_SESSION=$HOME/.config/x-terminal/session       # the operator's real session (0600)
X_TERMINAL_PROXY=http://127.0.0.1:7897
$HOME/.local/bin/x-terminal
```

The scratch config home is deliberate (see deviation D3): the operator's own `config.toml` says
`display.state = "immersive"` / `display.cover = "gitlog"`, and the cover keys write both display
keys back — running against the real config would have rewritten the operator's start-up state. The
scratch config also exercises the new file key in a live run. The operator's `config.toml` was
verified byte-identical after the proof.

### Capture 1 — baseline, 02:14:49 local (18:14:49Z)

Launch, Following loaded. Status line, verbatim:

```
following  1/103+  1 request, 0 rate-limited
```

**The baseline request count is 1.**

### Capture 2 — the search, 02:15:13 local (18:15:13Z)

`/`, the query `日本語` typed with `tmux send-keys -l`, `Enter`. Status line, verbatim:

```
search: latest "日本語"  1/20+  2 requests, 0 rate-limited
```

The cursor's own block (`>` marks the selection), verbatim:

```
>BigSandyAgain    Oct 16 02:14
>  @RollingJK69 @SeekerOfLight81 @AioN191398 日本語の表現が不自由なのはだーれ、君かｗ 誰が言ってる
>  かにもよる。「法的に問題無い」と君が思っても、弁護士が「いや、ちょっとまずいかと・・・」てなるこ
>  ともあるし、また相談する弁護士によって意見が分かれることも多々ある。パターンの話だ。いちいち説明
>  さすな😄
```

**Request count 2** (1 baseline + 1 search). The post under the cursor is Japanese; the configured
language is `zh-CN`.

### Capture 3 — `T`, 02:15:19 local (18:15:19Z)

One `T`, then a 6-second wait (the limiter's floor is 1.5 s and this is one request). **The whole
pane, verbatim** — the same list, the same rows, the same frame, with the selected post's body in
Chinese and every other row untouched:

```
>BigSandyAgain    Oct 16 02:14
>  谁的日语表达能力很差？是你吗？哈哈 这取决于谁说的。即使你认为“法律上没问题”，律师也可能会说，“不
>  ，这有点问题……”，而且不同的律师意见往往也不一样。这是个规律问题。别每次都让我解释😄

 AliceKiriame     Oct 16 02:14
   @Rsantoissho 日本語アカウントは日本語で遊ぶはあたりまえじゃん？
   まぁ私日独仏英中広閩も理解できるし、別にわたしに対して言語の壁がないと思いますよ。
   ただ、自分のせいで募集主の攻略法を理解（特に初週）していないなら退室しますよ。

 kyantake0714     Oct 16 02:14
   洋楽ソングと思わせといて実は歌詞の半分くらいは日本語だったりする曲
   歌詞が気になる人は公式チャンネルを要チェックや
 > Hennessy1118     Oct 15 09:10
 >   前に好きなBGM聞かれて「TuNGSTeN」 って答えたら何それってなったのは泣いた🤤
 >   神ってるの多すぎて毎回悩むけどなんだかんだこれだなって毎回なる
 >   そして最後にラピが「今から地上を奪還します」って言うところまでがセット🤤 https://t.co/sDPBiZpLl0
 >   [1] video

 nekodaizin       Oct 16 02:13
   双子に、長めの本を1ヶ月かけて読んで、毎日ストーリーと感想を書く宿題が出ているのですが、本は日本
   語訳で読んでいるけど、ストーリーの要約の英文を私が考えないといけなくて、完全に私の宿題です。

 inosuke_beast    Oct 16 02:13
   これマヂですごい！ハーバードのコンピューターサイエンス学科のcs249rクラスで使用されている内容＋リ
   ンクも無料だし日本語もアリ。
 > Ryrenz           Oct 15 11:05
 >   📙 又挖到一本狠教材《Machine Learning Systems》，讲的不是怎么调模型，而是怎么把 AI
 >   真正做成能跑的系统。 GitHub 上 28988 star、3643 个 fork，由 Vijay Janapa Reddi
 >   主导开发，在哈佛的课堂上讲过好几个学期，2026 年还要由 MIT Press 出纸质版。 学 AI

 kasiwa_mama      Oct 16 02:12
   まぁ、私は日本語しかしゃべらないですよ〜という立場表明しとかんとね、あちこちの言葉で話しかけられ
   がちだからね（お前が翻訳機使って分かれという態度） それが、日本人運営以外のサイトだと、まぁ……し
   かたない一面もあるけど（同時翻訳機能の話もあるが、私がお邪魔する立場かどうかもあるし）

 k1a_vi4          Oct 16 02:12
   @428mint2 芋の韓国人日本語まぁまぁ出来るけど拙い感じの人多いから好き Toxicは嫌い

 gomang367391     Oct 16 02:11
showing a translation  search: latest "日本語"  1/20+  3 requests, 0 rate-limited
```

**Request count 3 — exactly one higher than the baseline.** The notice field reads
`showing a translation`, the block list, the position (`1/20+`) and the frame are the text's only
neighbours: nothing was pushed, and no other row moved.

### Capture 4 — `j` (selection off the post), 02:15:27 local (18:15:27Z)

Selected block, verbatim:

```
 BigSandyAgain    Oct 16 02:14
   @RollingJK69 @SeekerOfLight81 @AioN191398 日本語の表現が不自由なのはだーれ、君かｗ 誰が言ってる
   かにもよる。「法的に問題無い」と君が思っても、弁護士が「いや、ちょっとまずいかと・・・」てなるこ
   ともあるし、また相談する弁護士によって意見が分かれることも多々ある。パターンの話だ。いちいち説明
   さすな😄

>AliceKiriame     Oct 16 02:14
```

Status line, verbatim:

```
showing a translation  search: latest "日本語"  2/20+  3 requests, 0 rate-limited
```

The translated post is back to its original Japanese **while still on the pane**, and the selection
mark is on the next post. The notice field is the process's own last announcement and survives a
move (existing notice semantics — `page` is what clears it). **Request count 3: the move cost
nothing.**

### Capture 5 — `k` back onto the post, 02:15:33 local (18:15:33Z)

Selected block, verbatim:

```
>BigSandyAgain    Oct 16 02:14
>  谁的日语表达能力很差？是你吗？哈哈 这取决于谁说的。即使你认为“法律上没问题”，律师也可能会说，“不
>  ，这有点问题……”，而且不同的律师意见往往也不一样。这是个规律问题。别每次都让我解释😄
```

Status line, verbatim:

```
showing a translation  search: latest "日本語"  1/20+  3 requests, 0 rate-limited
```

The translated body is back and **the request count is unchanged at 3** — the second visit to the
same post spent no second request. (The hermetic twin of this is
`TestReachingThePostAgainSpendsNothing`, over the fake's own call count.)

### Capture 6 — covered, `m` then `T`, 02:15:44 local (18:15:44Z)

`m` puts the remembered style on (the scratch config names none, so `servicelog`), then `T` again on
the same post. Selected block and its neighbours, verbatim:

```
>2026-10-15T18:14:17Z INFO  svc=sentry-decode-99 seq=6800099
>  谁的日语表达能力很差？是你吗？哈哈 这取决于谁说的。即使你认为“法律上没问题”，律师也可能会说，“不
>  ，这有点问题……”，而且不同的律师意见往往也不一样。这是个规律问题。别每次都让我解释😄
 2026-10-15T18:14:12Z INFO  svc=probe-session-73 seq=1186185
   @Rsantoissho 日本語アカウントは日本語で遊ぶはあたりまえじゃん？
```

Status line, verbatim:

```
payload rendered  filter=latest "日本語"  1/20+  3 requests, 0 rate-limited
```

The translated body is inside the cover's own grammar, the notice is the style's own covered wording
(`payload rendered`, not `showing a translation`), the list field is the cover's (`filter=`), and
**the request count is still 3** — `T` under a cover is not gated and, with the answer remembered,
spent nothing.

### Capture 7 — a second style (`2` → gitlog), 02:15:56 local (18:15:56Z)

```
>commit b0addb595d36
>Author: Yuki Krishnan <yuki.krishnan99@corp.internal>
>Date:   Fri Oct 16 02:14:17 2026 +0800

>    谁的日语表达能力很差？是你吗？哈哈 这取决于谁说的。即使你认为“法律上没问题”，律师也可能会说，“不
>    ，这有点问题……”，而且不同的律师意见往往也不一样。这是个规律问题。别每次都让我解释😄
```

Status line, verbatim:

```
message rendered  grep: latest "日本語"  1/20+  3 requests, 0 rate-limited
```

### The two request counts, side by side

| moment | count |
|---|---|
| launch (Following page one) | 1 |
| after the search | 2 |
| after `T` (one translation) | **3** |
| after `j` (move off) | 3 |
| after `k` (back on) | 3 |
| after `m` + `T` (covered) | 3 |
| after `2` (style) | 3 |

One translation cost exactly one request; moving and returning cost none.

### The doctor's own line (installed binary, same scratch config)

```
config     /tmp/xt-tr-proof/config/x-terminal/config.toml   proxy http://127.0.0.1:7897   media xdg-open (images, links), mpv (video, gif)   translate zh-CN
```

It is a presence fact on the config line (no new report line, `doctor --json`'s object unchanged), and
it spends no account request: naming whether `api.x.com` answers this endpoint would.

## 3. `x-terminal keys` and the help overlay

```
$ x-terminal keys | grep translate
T	translate	translate the selected post
```

`internal/cli/keys.go` was not edited — the subcommand reads the one table, so the row appeared for
free.

## 4. The golden frames

Regenerated with `CGO_ENABLED=0 go test ./internal/render/ -run TestGoldenFrames -count=1 -update`, and
measured against the pre-change commit `d5d63a5` rather than the working tree, so the check cannot pass
by vacuity:

```
cover_ci_help_120.golden       cover_ci_help_80.golden
cover_gitlog_help_120.golden   cover_gitlog_help_80.golden
cover_internal_help_120.golden cover_internal_help_80.golden
cover_servicelog_help_120.golden cover_servicelog_help_80.golden
help_120.golden                help_80.golden
```

Exactly ten files, all `help_*`, one line changed in each (the overlay's row list). **No other stored
frame moved**: `git diff --name-only d5d63a5 -- internal/tui/testdata` is empty (the shell's corpus
holds no help frame), and the swap is a no-op with an empty map, so every post frame is
byte-identical — which is why the eleven other `internal/render/testdata/TestGoldenFrames` files and
the whole `internal/tui/testdata` corpus are untouched.

## 5. The hermetic arms behind the pane

`internal/tui/translate_test.go` (10 functions; the plan's eight plus two) and
`internal/xsource/translate_test.go` + `internal/xweb/translate_test.go` (Task 1) cover, in order:

1. the translated body replaces the original in the same block of the same list, same frame, selection
   mark on the same pane row, exactly one port call with the resolved language;
2. `j` reverts the post and the translated text leaves the frame;
3. `k` back re-renders it with the fake's call count unchanged;
4. `T` twice on one post is one call and the second press changes no byte of the frame;
5. `Available: false` renders `no translation available` and keeps the post's own body — never a blank
   body presented as a translation — and a second press still spends nothing;
6. a refusal (`*xweb.APIError`) sets the process's failure through the mapper with an empty notice, and
   the posts stay where they were;
7. under each of the four covers the translated body is in the cover's grammar and the notice is that
   style's own covered wording, with none of the plain wording in the frame;
8. a model value held before the answer still renders the original (the map is never mutated in place);
9. a nil port and an empty language each report `no translation available` and issue nothing;
10. a frame whose selection names no post (`New(deps)` with no first page) reports `nothing to open`
    and issues nothing.

`internal/tui/dispatch_test.go` was **not edited**: the keymap-dispatch join now holds in both
directions by construction (31 rows, 31 handled actions, `keys.ActionTranslate` as a case label).

## 6. Deviations from the plan

**D1 — `internal/config/config.go` was touched in Task 1, not only in Task 3.** Task 1 landed
`DefaultTranslateLanguage`, the `Config.TranslateLanguage` field and its seeding in `Load` (4 lines of
plumbing, needed by Task 1's own gated test, which reads `cfg.TranslateLanguage`). Task 3 therefore
owed — and delivered — the `LC_ALL`/`LANG` reader (`localeLanguage`, `languageTag`), the
`translate.language` arm in `applySettings` with its line-numbered refusal, and the three config tests
(resolution order; the locale reader's own table; a malformed value keeping the value it would
otherwise have had, plus the Redacted counter-arm). The seeding in `Load` changed from
`DefaultTranslateLanguage` to `localeLanguage()` in Task 3.

**D2 — Task 1's "empty diff on the read adapters" criterion could not hold literally.**
`internal/xsource/home.go` carries the `Adapter` struct, so `WithTranslator` necessarily adds a
4-line `translator Translator` field there. No request, variable, header, method or read-adapter call
site changed: `git diff --stat` for that file is `+4 -0`.

**D3 — the live pane proof ran with a scratch `XDG_CONFIG_HOME`/`XDG_DATA_HOME` and the operator's
real session.** The operator's own `config.toml` says `display.state = "immersive"` /
`display.cover = "gitlog"`, and every cover transition writes both display keys back
(`Deps.Persist` → `config.SetDisplay`), so running the proof against the real config home would have
rewritten the operator's start-up state. The scratch config holds only
`translate.language = "zh-CN"`, which also puts the new file key through a live run; the session
(`X_TERMINAL_SESSION`) and the proxy are the operator's own, and the binary is the installed one. The
operator's `config.toml` was verified byte-identical after the proof.

**D4 — two extra hermetic arms and one test-seam choice.** The plan's eight arms are all present;
arms 9 (the two presence facts) and 10 (nothing under the cursor) were added because the plan's Task 2
text requires those rules and nothing else asserted them. The refusal arm uses the tests' own failure
seam (`testFailure`) with a real `*xweb.APIError` built in the test rather than `internal/cli`'s
mapper, which `internal/tui` may not import (the boundary test forbids it): the arm asserts the shell
asked the mapper and printed its answer — the state word and the remedy — with an empty notice, and it
shows the refusal reaches the classifier's detail rather than inventing a word.

**D5 — the harness gained the fake and the shifted-T keystroke.** `internal/tui/tui_test.go` gained
`shell.tr`, the `translateLang`/`translatedBody` constants, the `T` arm in `press` (the decoder's real
shape: `Code: 't'`, `ShiftedCode: 'T'`, `Mod: tea.ModShift`, `Text: "T"`), and the two Deps fields in
`newShellIn`. `TestTheKeymapMatchesWhatTheDecoderReports` was left alone; `press`'s renamed helper
handles both shifted letters.

**D6 — `RUNBOOK.md` was amended in place.** The Repair paragraph of `## Request shape drift` keeps its
existing sentences and gains the one about the endpoint that is in neither the registry nor the
bundle: `internal/xweb/translate.go` is the file whose URL is the claim, `--refresh-ids` cannot repair
it, and the gated test is the command that measures it. Four parts, their order, and the Symptom's
verbatim `bad request` are unchanged, so `internal/cli/runbook_test.go` passes unmodified. No new
section, no new `domain.FailureKind`, no new doctor anchor.

## 7. What this feature deliberately did not do

- No registry entry, no `postOperations` row, no signature call for the translation endpoint: it is not
  a GraphQL operation and no bundle carries a query id for it.
- `coverBreaking` is byte-identical: a translation hands no string to any other program and is an
  in-process request about a post the account already fetched.
- Nothing about a translation reaches disk: the cache is a field of a value type in `internal/tui`,
  rebuilt by copy on every answer, and `x-terminal audit` still prints its six persistence destinations.
- No module added: `go.mod`/`go.sum` untouched, asserted by the plan's own verification line.
