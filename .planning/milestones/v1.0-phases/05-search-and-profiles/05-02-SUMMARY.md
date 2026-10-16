---
phase: 05-search-and-profiles
plan: 02
type: summary
status: complete
requirements: [DISC-03]
verified: |
  gofmt -l internal cmd                          empty
  go test ./... -count=1                         860 passed in 19 packages
  go test ./internal/xsource -count=1 -run TestEveryFixtureIsProvenanced PASS
  go test ./internal/tui -count=1 -run 'TestProfile|TestOpenAuthor' 3 PASS
  grep -c 'Walk\[' internal/screen/profile.go    >=1
  grep -n 'UserHead' internal/render/grammar.go internal/render/profile.go finds both
unverified: |
  Live UserByScreenName+UserTweets capture (X_TERMINAL_CAPTURE=1) not run here.
  fixture-profile-header.json is synthetic (lifted from fixture-usertweets user
  node into a UserByScreenName envelope); fixture-profile-posts.json is
  byte-identical to fixture-usertweets.json (recorded). Help overlay at H=24
  clips the last keymap row (`ctrl+p`) — stored help_* goldens keep corpusH.
---
# 05-02 SUMMARY — open author profile: two requests, header block, same walk

## 1. Profile value and port

`domain.Profile` = `{User, Bio, Followers/Following/Posts *int, Joined time.Time}`.
`domain.User` was not edited (UI-SPEC). Counts are pointers so absent ≠ zero.
`Source.Profile(ctx, handle)` and `Source.UserPosts(ctx, id, cursor)`.

## 2. Wire decode and fixtures

`ParseProfile` reads identity via `userFrom`, then `profile_bio.description`,
`relationship_counts.followers/following`, `tweet_counts.tweets`,
`core.created_at` (Twitter layout → `Joined`). Absent fields stay nil/empty/zero
time; explicit `0` sets a pointer to zero.

`Adapter.Profile` → `UserByScreenName` + `screen_name`. `Adapter.UserPosts` →
`UserTweets` + `{userId, count:pageCount, cursor}` and `ParseTimeline` (one walk).

Provenance: header `source=synthetic` (lifted envelope); posts `source=recorded`.
`pages-profile.json` written from the posts fixture via `-update-profile` parity.

## 3. Frame reducer — one walk, ordered pair

`screen.Open(handle)` returns only `ProfileRequest`. `HeaderFetched` stores the
header and returns `UserPostsRequest{ID: header.User.ID}`. `HeaderFailed` issues
nothing further. Posts events fold into `Walk`; refresh is header then page one
with `RefreshPrepend`. No local cursor/dedupe/stall fields.

## 4. Render

`ProfileHeader` / `Profile`: identity from `UserHead` (`{handle:16} {name}`),
bio wrapped every line, facts clauses in order `followers/following/posts/joined`
omitting absents. Header + blank separator on first post block + `Blocks(posts)`.
`Stamp.MonthYear=Jan 2006`, `AccountFacts`, `ListLabelProfile`, notices
`loading posts` / `no posts` / `no author to open`, marker `loading profile...`.
Verification prints nowhere.

## 5. Shell

`frameProfile` on the existing stack. `a` → no author: notice + 0; same author on
top: no-op + 0; else push + header, then frame Update issues posts. Status
`profile: {handle}`; position counts posts only. Failed header → 1 request;
failed posts keep header on screen.

## 6. Tests that assert the counts

- `TestProfileOpensInTwoRequests` — exactly 2, then 0 on second `a`
- `TestProfileHeaderArrivesBeforeThePosts` — header kept after posts failure
- `TestOpenAuthorWithoutAnAuthor` — 0 spend + `NoticeNoAuthor`
- `TestProfileHeaderFetchedIssuesUserPostsByNumericID` / `HeaderFailed…`
- `TestParseProfileAbsentVersusZeroCounts` — nil vs `&0`
- `TestAdapterProfileAndUserPostsAskByName` — variables key-by-key
