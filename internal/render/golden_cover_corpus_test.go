package render

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/screen"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// coverSavedPath is the store's path as the composition root hands it to the shell: the directory
// the program is named after, and the file two of the three capability bodies print. A covered
// frame may hold neither half of it, and the covered capability frames carry the string the shipped
// program really has rather than a placeholder, so the absence scan is armed against it.
const coverSavedPath = "/home/user/.local/share/x-terminal/saved.db"

// coveredSafeList is the adversarial feed with every body that would print a
// URL or a product noun under a cover rewritten to safe prose. Handles and
// account ids stay so the alias-presence arm remains armed against timeline_80.
func coveredSafeList(t *testing.T) screen.List {
	t.Helper()
	list := absorbed(t)
	posts := append([]domain.Post(nil), list.Posts()...)
	for i := range posts {
		posts[i].Text = scrubCoverBody(posts[i].Text)
		if posts[i].Quote != nil {
			q := *posts[i].Quote
			q.Text = scrubCoverBody(q.Text)
			if q.Quote != nil {
				inner := *q.Quote
				inner.Text = scrubCoverBody(inner.Text)
				q.Quote = &inner
			}
			posts[i].Quote = &q
		}
	}
	page := domain.PostPage{Posts: posts, Next: list.Next()}
	out := screen.List{Kind: domain.Following}
	next, _ := out.Update(screen.PageFetched{Page: page}, corpusNow)
	if len(posts) > 1 {
		return selected(t, next, posts[1].ID)
	}
	return next
}

// scrubCoverBody rewrites out of the writers' text the one vocabulary a cover's own chrome must not
// carry: the program's words for what it stores, what a post is and what it attaches. A covered
// frame may hold the accounts' words — that is the feed it is a rendering of — but not the
// program's, and the corpus's writers use both. The substitutions are the covered vocabulary's own
// (copy.go's `record`, `file`, `cache`), so the text a covered frame carries reads the way its own
// chrome does, and the whole-corpus absence scans are scans of the program's words rather than of
// the feed's.
//
// The rewrites are applied in this order and all of them apply: a sentence of the feed's may name
// more than one of these words (a post that was saved to a mirror), and a text that kept one of
// them because another matched first would be a covered frame holding a word the cover has not got.
func scrubCoverBody(s string) string {
	if s == "" {
		return s
	}
	if strings.Contains(strings.ToLower(s), "http") {
		return "the link that runs past the measure: example.invalid/2026/10/14/a-path-that-cannot-fit-on-one-line/a-fragment"
	}
	s = strings.ReplaceAll(s, "mirror", "cache")
	s = strings.ReplaceAll(s, "attachment", "file")
	s = strings.ReplaceAll(s, "attachments", "files")
	s = strings.ReplaceAll(s, "post", "record")
	s = strings.ReplaceAll(s, "Post", "Record")
	return s
}

// coveredSavedPosts is the Phase 6 saved corpus with product nouns scrubbed so
// a covered bookmarks/likes frame can carry the writers' text without failing
// the no-real-identity product-noun arm.
func coveredSavedPosts() []domain.Post {
	posts := savedCorpusPosts()
	for i := range posts {
		posts[i].Text = scrubCoverBody(posts[i].Text)
		posts[i].Text = strings.ReplaceAll(posts[i].Text, "bookmark", "record")
		posts[i].Text = strings.ReplaceAll(posts[i].Text, "likes", "stars")
		posts[i].Text = strings.ReplaceAll(posts[i].Text, "like", "star")
	}
	return posts
}

// coveredCrossProduct is every later screen × every CoverStyle × both audit
// widths. A missing covered frame is a missing file and fails TestGoldenFrames.
func coveredCrossProduct(t *testing.T) []frame {
	t.Helper()
	safe := coveredSafeList(t)
	expired := domain.Failure{Kind: domain.FailureSessionExpired, Detail: xweb.OutcomeAuthRejected.Remedy(), Retry: true}
	threadBase := threadSpecConversation()
	counts := engagement(t)
	threadBase.Ancestors[0].Counts = counts
	threadBase.Focal.Counts = counts
	for i := range threadBase.Continued {
		threadBase.Continued[i].Counts = counts
	}
	for i := range threadBase.Replies {
		threadBase.Replies[i].Counts = counts
	}
	threadBase.Ancestors[0].Text = scrubCoverBody(threadBase.Ancestors[0].Text)
	threadBase.Focal.Text = scrubCoverBody(threadBase.Focal.Text)
	if threadBase.Focal.Quote != nil {
		q := *threadBase.Focal.Quote
		q.Text = scrubCoverBody(q.Text)
		threadBase.Focal.Quote = &q
	}
	for i := range threadBase.Continued {
		threadBase.Continued[i].Text = scrubCoverBody(threadBase.Continued[i].Text)
	}
	for i := range threadBase.Replies {
		threadBase.Replies[i].Text = scrubCoverBody(threadBase.Replies[i].Text)
	}
	threadState := state{
		pane: paneThread, conversation: threadBase,
		threadSel: Selection{ID: threadBase.Focal.ID}, threadHasMore: true,
		used: 12, limited: 0, failure: expired,
	}
	posts := coveredSavedPosts()
	var saved screen.Saved
	saved = screen.NewSaved(domain.ListBookmarks, domain.CapNone)
	saved = saved.Update(screen.MirrorPage{Page: domain.PostPage{Posts: posts, Next: "more"}})
	saved = saved.Update(screen.StatusRead{
		Owner: "mira", LastSuccess: time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC),
	})
	if len(posts) > 1 {
		next, _ := saved.List().Update(screen.Select{Target: screen.Target{ID: string(posts[1].ID)}}, corpusNow)
		saved = saved.WithList(next)
	}
	bookmarkState := state{
		pane: paneSaved, saved: saved, savedKind: domain.Bookmark,
		syncedOn: true, syncedAt: time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC),
		failure: expired, used: 12, limited: 0,
	}
	likesSaved := screen.NewSaved(domain.ListLikes, domain.CapNone)
	likesSaved = likesSaved.Update(screen.MirrorPage{Page: domain.PostPage{Posts: posts, Next: "more"}})
	likesSaved = likesSaved.Update(screen.StatusRead{
		Owner: "mira", LastSuccess: time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC),
	})
	if len(posts) > 1 {
		next, _ := likesSaved.List().Update(screen.Select{Target: screen.Target{ID: string(posts[1].ID)}}, corpusNow)
		likesSaved = likesSaved.WithList(next)
	}
	likesState := state{
		pane: paneSaved, saved: likesSaved, savedKind: domain.Like,
		syncedOn: true, syncedAt: time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC),
		failure: expired, used: 12, limited: 0,
	}
	author := domain.User{ID: "u-aergon", ScreenName: "aergon", Name: "name of aergon"}
	detailPost := domain.Post{
		ID: "cover-media-detail", Author: author, CreatedAt: corpusNow.Add(-time.Hour),
		Text: "two clips from the archive; one of them needs a player.",
		Media: []domain.MediaRef{
			{Kind: domain.Image, URL: "https://pbs.twimg.com/media/opened.jpg"},
			{Kind: domain.Video, URL: "https://pbs.twimg.com/media/poster.jpg", Playable: "https://video.twimg.com/opened.mp4"},
		},
	}
	detailList := absorbPosts(t, []domain.Post{detailPost}, detailPost.ID)
	detailState := state{list: detailList, used: 12, limited: 0, failure: expired}
	helpState := state{pane: paneHelp, list: safe, failure: expired, used: 200, limited: 12}

	// The saved surface's non-row states: the three capability states, the empty read of a store
	// nothing has ever filled, and the failure that has nothing to show. They are the states this
	// frame's own three branches draw, and each carries the values the plain reader's words spell —
	// the store's path, the account the cache was filled from, the list's product word and the
	// reachability check's sentence with the host in it — so the covered absence over this corpus is
	// a scan over armed frames rather than over frames with nothing to hide.
	savedFacts := func(kind domain.SavedKind, owner string) CapabilityFacts {
		return CapabilityFacts{Path: coverSavedPath, Owner: owner, List: kind.List()}
	}
	// The capability states carry no process failure: the shell suppresses it while a capability
	// occupies the state slot (view.go's status), so this is the state the frame is really in — a
	// capability is a fact about the build or the store, not a failed request.
	capState := func(cap domain.Capability) state {
		return state{
			pane: paneSaved, savedKind: domain.Bookmark, syncedOn: true, syncedAt: corpusNow,
			capability: cap, capFacts: savedFacts(domain.Bookmark, "mira"),
			used: 12, limited: 0,
		}
	}
	emptySavedState := state{
		pane: paneSaved, savedKind: domain.Bookmark, syncedOn: true,
		used: 12, limited: 0,
	}
	failedSavedState := state{
		pane: paneSaved, savedKind: domain.Bookmark, syncedOn: true,
		failure: domain.Failure{Kind: domain.FailureNoRoute, Detail: routeSentence(), Retry: true},
		used:    12, limited: 0,
	}
	// The likes list's ambiguous empty answer: a store holding no rows for a list the upstream may
	// no longer serve. Its copy is the cover's own hedge, one line and no body — the state whose
	// answer cannot be read as "nothing was there" (06-UI-SPEC:263-266) — and it is a state of its
	// own rather than a variant of `saved_empty`, so the two frames can be confused neither in the
	// corpus nor by a reader of it.
	likesHedgedState := state{
		pane: paneSaved, savedKind: domain.Like, syncedOn: true, syncedAt: corpusNow,
		used: 12, limited: 0,
	}
	// The four notices a saved frame's `r` leaves, in the words the style uses for them. They are
	// the only reading of these words on screen under a cover, and the corpus carried none of them.
	noticeState := func(notice NoticeKind, count int) state {
		return state{
			pane: paneSaved, saved: saved, savedKind: domain.Bookmark,
			syncedOn: true, syncedAt: corpusNow,
			notice: notice, noticeCount: count,
			used: 12, limited: 0,
		}
	}

	screens := []struct {
		name   string
		base   state
		widths []int
	}{
		{name: "thread", base: threadState},
		{name: "bookmarks", base: bookmarkState},
		{name: "likes", base: likesState},
		{name: "media_detail", base: detailState},
		{name: "help", base: helpState},
		{name: "saved_unavailable", base: capState(domain.CapOperationMissing)},
		{name: "saved_wrong_account", base: capState(domain.CapWrongAccount)},
		{name: "saved_unreadable", base: capState(domain.CapMirrorUnreadable)},
		{name: "saved_empty", base: emptySavedState},
		{name: "saved_failed", base: failedSavedState},
		{name: "likes_hedged", base: likesHedgedState},
		// The notice frames are stored at 120 only: the notice is the first field the drop rule
		// removes, and the covered saved line at 80 is already tight enough that the notice is what
		// goes (`cover_*_bookmarks_80` holds neither the notice, the list nor synced). A covered
		// notice's words are on screen where the line has room for them, and the audit asserts them
		// in the frames that hold them.
		{name: "sync_notice", base: noticeState(NoticeSyncing, 40), widths: []int{120}},
		{name: "absent_notice", base: noticeState(NoticeNoLongerSaved, 3), widths: []int{120}},
		{name: "complete_notice", base: noticeState(NoticeMirrorComplete, 0), widths: []int{120}},
		{name: "stalled_notice", base: noticeState(NoticeSyncStalled, 0), widths: []int{120}},
	}
	out := make([]frame, 0, len(screens)*len(CoverStyles())*2+len(CoverStyles()))
	for _, sc := range screens {
		widths := sc.widths
		if widths == nil {
			widths = []int{80, 120}
		}
		for _, style := range CoverStyles() {
			for _, w := range widths {
				name := "cover_" + string(style) + "_" + sc.name + "_" + strconv.Itoa(w)
				f := frame{
					name: name, w: w, env: envPlain,
					state: withCover(sc.base, style),
				}
				if sc.name == "help" {
					// 30 keymap rows need a taller pane so the three media HelpCovered
					// rows are on screen for the DSGT-08 key-column comparison.
					f.h = 36
				}
				out = append(out, f)
			}
		}
	}
	// Gated refusals at 120 (80 already in mediaFrames) complete the width cross-product.
	for _, style := range CoverStyles() {
		out = append(out, frame{
			name: "media_gated_" + string(style) + "_120", w: 120, env: envPlain,
			state: withCover(state{list: safe, notice: NoticeGated, used: 12, limited: 0}, style),
		})
	}
	return out
}
