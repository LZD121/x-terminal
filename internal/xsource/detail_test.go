package xsource

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

// The test-local bodies are built from the recorded answer's own envelope and item nesting rather
// than written out whole: the shapes under test here are entry classes and result types, and a test
// about one of them should not be able to fail because of a brace in the wrapper around it.
const (
	threadEnvelopePrefix = `{"data":{"threaded_conversation_with_injections_v2":{"instructions":[{"type":"TimelineClearCache"},{"type":"TimelineAddEntries","entries":[`
	threadEnvelopeSuffix = `]}]}}}`
)

// threadBody wraps entries in the envelope a TweetDetail answer has.
func threadBody(entries ...string) []byte {
	return []byte(threadEnvelopePrefix + strings.Join(entries, ",") + threadEnvelopeSuffix)
}

// threadModule is one conversation module carrying one tweet item, nested exactly the way the
// recorded answer nests it: content.items[].item.itemContent.tweet_results.result.
func threadModule(moduleID, itemID, display, result string) string {
	return `{"entryId":"` + moduleID + `","content":{"__typename":"TimelineTimelineModule","items":[{"entryId":"` + itemID + `","item":{"itemContent":{"__typename":"TimelineTweet","tweetDisplayType":"` + display + `","tweet_results":{"result":` + result + `}}}}]}}`
}

// rawPost is one post a body carries, with the entry it arrived in and whether that entry marked it
// as an advertisement or as a post from the recommendation rail.
type rawPost struct {
	Node    rawTweet
	EntryID string
	Ad      bool
	Rail    bool
}

// rawPosts is every post a body carries, keyed by the post's own id: the top-level entries' tweets
// and the module items' tweets, both.
func rawPosts(t *testing.T, entries []rawEntry) map[domain.PostID]rawPost {
	t.Helper()
	posts := map[domain.PostID]rawPost{}
	add := func(entry rawEntry, item rawItemContent, itemEntryID string) {
		node := rawTweetOf(t, item.TweetResults.Result)
		id := domain.PostID(node.RestID)
		if id == "" {
			// A tombstone is the one result with no rest_id of its own, and its identity is the
			// entry it arrived in — derived here rather than asked of the decoder, because this
			// helper is what the decoder's answers are compared against.
			if node.Typename != "TweetTombstone" {
				return
			}
			tail := itemEntryID
			if i := strings.LastIndex(tail, tweetEntryPrefix); i >= 0 {
				tail = tail[i+len(tweetEntryPrefix):]
			}
			id = domain.PostID(tail)
		}
		posts[id] = rawPost{
			Node:    node,
			EntryID: itemEntryID,
			Ad:      strings.Contains(itemEntryID, promotedItemMark),
			Rail:    strings.HasPrefix(entry.EntryID, railEntryPrefix),
		}
	}
	for _, entry := range entries {
		if entry.Content.ItemContent != nil && entry.Content.ItemContent.Typename == "TimelineTweet" {
			add(entry, *entry.Content.ItemContent, entry.EntryID)
		}
		for _, item := range entry.Content.Items {
			if item.Item.ItemContent.Typename == "TimelineTweet" {
				add(entry, item.Item.ItemContent, item.EntryID)
			}
		}
	}
	return posts
}

// promotedModule reports whether a module's items are advertisements rather than this
// conversation's: the mark is in the item's own entry id, which is where the upstream puts it.
func promotedModule(entry rawEntry) bool {
	for _, item := range entry.Content.Items {
		if strings.Contains(item.EntryID, promotedItemMark) {
			return true
		}
	}
	return false
}

// rawBottomCursor is the value of the answer's bottom cursor, read from the body rather than from
// the page: a decoder that dropped the cursor has to fail against the body, not against itself.
func rawBottomCursor(t *testing.T, entries []rawEntry) string {
	t.Helper()
	for _, entry := range entries {
		if entry.Content.Typename == "TimelineTimelineCursor" && entry.Content.CursorType == bottomCursor {
			return entry.Content.Value
		}
	}
	return ""
}

// allPosts is every post a page holds at the top level: the focal post, the chain above it and the
// posts of its groups. A quoted post is not included — it belongs to another conversation on
// purpose, and the one body here that carries a quote is a synthetic one.
func allPosts(page domain.ThreadPage) []domain.Post {
	posts := make([]domain.Post, 0, 1+len(page.Ancestors)+len(page.Groups))
	if page.Focal.ID != "" {
		posts = append(posts, page.Focal)
	}
	posts = append(posts, page.Ancestors...)
	for _, group := range page.Groups {
		posts = append(posts, group.Posts...)
	}
	return posts
}

// keptPostIDs is every post id a page holds, at every quote depth up to the decoder's own bound: a
// test that some post did not reach the page has to look everywhere a post can be.
func keptPostIDs(page domain.ThreadPage) map[domain.PostID]bool {
	ids := map[domain.PostID]bool{}
	var walk func(post domain.Post, depth int)
	walk = func(post domain.Post, depth int) {
		if post.ID == "" {
			return
		}
		ids[post.ID] = true
		if post.Quote != nil && depth < maxQuoteDepth {
			walk(*post.Quote, depth+1)
		}
	}
	for _, post := range allPosts(page) {
		walk(post, 1)
	}
	return ids
}

// pagePostOf finds one post the page holds at the top level.
func pagePostOf(page domain.ThreadPage, id domain.PostID) (domain.Post, bool) {
	for _, post := range allPosts(page) {
		if post.ID == id {
			return post, true
		}
	}
	return domain.Post{}, false
}

// fixtureFocalID is the post a fixture's body is about, taken from the provenance row rather than
// from a literal in this file: the row is the one place that says which post a body is about, and a
// test carrying its own copy of that fact would keep passing after the row moved.
func fixtureFocalID(t *testing.T, name string) domain.PostID {
	t.Helper()
	row := rowFor(t, provenance(t), name)
	if row.FocalID == "" {
		t.Fatalf("%s records no focalId for %s", provenancePath, name)
	}
	return domain.PostID(row.FocalID)
}

// mustParseThread parses a body this test expects to read.
func mustParseThread(t *testing.T, raw []byte, focal domain.PostID) domain.ThreadPage {
	t.Helper()
	page, err := ParseThread(raw, focal)
	if err != nil {
		t.Fatalf("ParseThread: %v", err)
	}
	return page
}

// firstEntryID is the id of a body's first entry, or an empty string for a body with none.
func firstEntryID(entries []rawEntry) string {
	if len(entries) == 0 {
		return ""
	}
	return entries[0].EntryID
}

// TestTheRecordedThreadIsOnePage is the census the rest of the phase rests on: the recorded answer
// is one page — the focal post, the chain above it (empty here), the reply groups in the answer's
// own order, the answer's own next cursor — and the entry classes that are not posts are counted
// rather than rendered or refused.
func TestTheRecordedThreadIsOnePage(t *testing.T) {
	raw := readFixture(t, recordedDetailPath)
	focalID := fixtureFocalID(t, recordedDetailPath)
	page := mustParseThread(t, raw, focalID)
	entries := detailEntries(t, raw)

	if page.Focal.ID != focalID {
		t.Errorf("the focal post is %q, want the post the request was for (%q)", page.Focal.ID, focalID)
	}
	if page.Focal.Text == "" || page.Focal.Author.ScreenName == "" || page.Focal.CreatedAt.IsZero() {
		t.Errorf("the focal post is %+v, want a body, a byline and a date: the page carries the post in full, not a placeholder", page.Focal)
	}
	// The chain above the focal post is what the answer put before the focal entry, so "empty" is a
	// claim about this body rather than about the decoder.
	if first := firstEntryID(entries); first != tweetEntryPrefix+string(focalID) {
		t.Errorf("the answer's first entry is %q, want the focal post's entry: everything before that entry is the chain above it", first)
	}
	if len(page.Ancestors) != 0 {
		t.Errorf("the page carries %d ancestors, want none: the recorded answer's focal entry comes first", len(page.Ancestors))
	}

	// The groups, counted from the body: one per conversationthread module that is not an
	// advertisement. The measured answer carries 41 modules and 5 of them are advertisements from
	// five other conversations, so 36 groups — and the arithmetic is done here from the raw JSON
	// rather than written down, so a decoder that dropped a module cannot pass.
	modules, wantGroups := 0, 0
	for _, entry := range entries {
		if !strings.HasPrefix(entry.EntryID, threadEntryPrefix) {
			continue
		}
		modules++
		if promotedModule(entry) {
			continue
		}
		wantGroups++
	}

	switch {
	case modules != 41:
		t.Errorf("the body carries %d conversationthread modules, want the measured 41", modules)
	case wantGroups != 36:
		t.Errorf("the body carries %d reply modules that are not advertisements, want the measured 36", wantGroups)
	case len(page.Groups) != wantGroups:
		t.Errorf("the page carries %d groups and the body carries %d reply modules: a module cannot go missing between the two", len(page.Groups), wantGroups)
	case page.Next != domain.Cursor(rawBottomCursor(t, entries)):
		t.Errorf("Next is %q, want the body's own bottom cursor (%q)", page.Next, rawBottomCursor(t, entries))
	case page.Next == "":
		t.Error("Next is empty, want the cursor the recorded answer carried: without it the walk cannot take a second step")
	case page.Requested != 42:
		t.Errorf("Requested is %d, want the measured 42 tweet items (the focal post and the 41 module items)", page.Requested)
	case page.Skipped != 6:
		t.Errorf("Skipped is %d, want the measured 6 (five advertisements and the answer's bottom cursor)", page.Skipped)
	case !page.FetchedAt.IsZero():
		t.Errorf("ParseThread stamped the page with %s: the parser reads no clock, the adapter does", page.FetchedAt)
	}
}

// TestTheRecordedAnswerKeepsOnlyItsOwnConversation proves the drop is complete rather than merely
// quiet: every post the page holds belongs to the focal post's conversation, and the five posts the
// answer carried from five other conversations are exactly the ones that were dropped.
func TestTheRecordedAnswerKeepsOnlyItsOwnConversation(t *testing.T) {
	raw := readFixture(t, recordedDetailPath)
	page := mustParseThread(t, raw, fixtureFocalID(t, recordedDetailPath))
	entries := detailEntries(t, raw)

	if page.Focal.ConversationID == "" {
		t.Fatal("the focal post carries no conversation id, so nothing below could be checked against it")
	}
	// The claim is about every post the page holds, and it is complete for this body because the
	// recorded answer carries no quote and no repost at all: the reply stream is flat.
	if bytes.Contains(raw, []byte("quoted_status_result")) || bytes.Contains(raw, []byte("retweeted_status_result")) {
		t.Fatal("the recorded body carries a quote or a repost, so 'every kept post' is not the whole page and this assertion has to be widened")
	}
	for _, post := range allPosts(page) {
		if post.ConversationID != page.Focal.ConversationID {
			t.Errorf("the page carries post %s from conversation %q, want the focal post's %q: the answer's own conversation is the whole thread", post.ID, post.ConversationID, page.Focal.ConversationID)
		}
	}

	posts := rawPosts(t, entries)
	ads, kept := 0, keptPostIDs(page)
	for id, post := range posts {
		if !post.Ad {
			continue
		}
		ads++
		if kept[id] {
			t.Errorf("the advertised post %s reached the page: it belongs to another conversation", id)
		}
		if post.Node.Legacy != nil && post.Node.Legacy.ConversationIDStr == page.Focal.ConversationID {
			t.Errorf("the advertised post %s claims the focal post's conversation, so the drop is not the conversation rule this test says it is", id)
		}
	}
	if ads != 5 {
		t.Errorf("the body carries %d advertised posts, want the measured 5", ads)
	}
}

// TestTheOrderIsTheAnswersOrder is the anti-tree assertion: the groups come out in the order the
// answer ranked them, each group's post is the post its entry carried, and every post's parent is
// the reply-to id the body itself wrote. The decoder neither re-sorts the answer nor derives a tree
// from it.
func TestTheOrderIsTheAnswersOrder(t *testing.T) {
	raw := readFixture(t, recordedDetailPath)
	page := mustParseThread(t, raw, fixtureFocalID(t, recordedDetailPath))
	entries := detailEntries(t, raw)

	// The payload's own order and parentage, taken from the body: the conversation modules that are
	// not advertisements, in the order they arrived, each with the post its item carries.
	var wantOrder []domain.PostID
	replyTo := map[domain.PostID]string{}
	for id, post := range rawPosts(t, entries) {
		if post.Node.Legacy != nil {
			replyTo[id] = post.Node.Legacy.InReplyToStatusIDStr
		}
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.EntryID, threadEntryPrefix) || promotedModule(entry) {
			continue
		}
		for _, item := range entry.Content.Items {
			if item.Item.ItemContent.Typename != "TimelineTweet" {
				continue
			}
			node := rawTweetOf(t, item.Item.ItemContent.TweetResults.Result)
			wantOrder = append(wantOrder, domain.PostID(node.RestID))
			// The module's own entry id carries the post id as its tail, which is the identity the
			// group is keyed by upstream, and the item's own rest_id is the post's. This test's
			// order is built from the item, so the two disagreeing would mean the body changed.
			if tail := strings.TrimPrefix(entry.EntryID, threadEntryPrefix); tail != node.RestID {
				t.Errorf("the module entry id %s names %q and its item's post is %s", entry.EntryID, tail, node.RestID)
			}
		}
	}
	if len(wantOrder) != len(page.Groups) {
		t.Fatalf("the body carries %d non-advertised modules and the page carries %d groups", len(wantOrder), len(page.Groups))
	}
	for i, group := range page.Groups {
		if len(group.Posts) != 1 {
			t.Fatalf("group %d carries %d posts, want the measured 1 (the recorded answer puts one post per module)", i, len(group.Posts))
		}
		post := group.Posts[0]
		if post.ID != wantOrder[i] {
			t.Errorf("group %d is post %s and the answer's own order has %s there: the decoder re-sorted the answer", i, post.ID, wantOrder[i])
		}
		if want := domain.PostID(replyTo[post.ID]); post.ReplyTo != want {
			t.Errorf("post %s answers %q and the body says %q: the parent comes from the payload, never from a tree built over it", post.ID, post.ReplyTo, want)
		}
	}
	for _, post := range allPosts(page) {
		if want := domain.PostID(replyTo[post.ID]); post.ReplyTo != want {
			t.Errorf("post %s answers %q and the body says %q", post.ID, post.ReplyTo, want)
		}
	}
}

// TestTheSyntheticShapesAreAllRead is the other half of the decoder's coverage: the shapes no
// recording carries — a chain above the focal post, a self-thread, a module with its own handle, a
// tombstone, a limited reply, a withheld reply and a quote two levels deep — read as themselves.
func TestTheSyntheticShapesAreAllRead(t *testing.T) {
	shapes := readFixture(t, shapesPath)
	focalID := fixtureFocalID(t, shapesPath)
	page := mustParseThread(t, shapes, focalID)
	entries := detailEntries(t, shapes)
	posts := rawPosts(t, entries)

	// The chain above the focal post: the tweet entries before it, in the answer's order, root
	// first. Both sides are read from the body.
	var wantAncestors []domain.PostID
	for _, entry := range entries {
		if !strings.HasPrefix(entry.EntryID, tweetEntryPrefix) {
			continue
		}
		if strings.TrimPrefix(entry.EntryID, tweetEntryPrefix) == string(focalID) {
			break
		}
		node := rawTweetOf(t, entry.Content.ItemContent.TweetResults.Result)
		wantAncestors = append(wantAncestors, domain.PostID(node.RestID))
	}
	if len(wantAncestors) != 3 {
		t.Fatalf("the body carries %d ancestors above the focal post, want the 3 it was written with", len(wantAncestors))
	}
	if len(page.Ancestors) != len(wantAncestors) {
		t.Fatalf("the page carries %d ancestors, want %d", len(page.Ancestors), len(wantAncestors))
	}
	for i, ancestor := range page.Ancestors {
		if ancestor.ID != wantAncestors[i] {
			t.Errorf("ancestor %d is %s and the answer's own order has %s there", i, ancestor.ID, wantAncestors[i])
		}
	}
	// The focal post answers the last ancestor, which is what makes it the last link of the chain
	// rather than a post that answers nothing.
	if want := wantAncestors[len(wantAncestors)-1]; page.Focal.ReplyTo != want {
		t.Errorf("the focal post answers %q, want the ancestor above it (%q)", page.Focal.ReplyTo, want)
	}

	// The self-thread: one group, kind SelfThread, two posts, both by the focal post's author.
	selfThreads := 0
	for _, group := range page.Groups {
		if group.Kind != domain.SelfThread {
			continue
		}
		selfThreads++
		if len(group.Posts) != 2 {
			t.Errorf("the self-thread group carries %d posts, want the body's 2", len(group.Posts))
		}
		for _, post := range group.Posts {
			if post.Author.ScreenName != page.Focal.Author.ScreenName {
				t.Errorf("the self-thread carries a post by %q and the focal post is by %q: that is a stranger's reply, not a continuation", post.Author.ScreenName, page.Focal.Author.ScreenName)
			}
		}
	}
	if selfThreads != 1 {
		t.Errorf("the page carries %d self-thread groups, want the 1 the body was written with", selfThreads)
	}

	// The quote two levels deep, read from the body: the reply that quotes, the post it quotes and
	// that post's own quoted post.
	var quotingID, middleID, innerID domain.PostID
	for id, post := range posts {
		if post.Node.QuotedStatusResult == nil {
			continue
		}
		quotingID = id
		middle := rawTweetOf(t, post.Node.QuotedStatusResult.Result)
		middleID = domain.PostID(middle.RestID)
		if middle.QuotedStatusResult == nil {
			continue
		}
		innerID = domain.PostID(rawTweetOf(t, middle.QuotedStatusResult.Result).RestID)
	}
	if quotingID == "" || middleID == "" || innerID == "" {
		t.Fatalf("the body carries the quote chain %q -> %q -> %q, want a reply that quotes a post that quotes another", quotingID, middleID, innerID)
	}
	quoting, ok := pagePostOf(page, quotingID)
	if !ok {
		t.Fatalf("the quoting reply %s is not in the page", quotingID)
	}
	switch {
	case quoting.Quote == nil:
		t.Fatalf("the quoting reply carries no quote, want the post it quotes (%s)", middleID)
	case quoting.Quote.ID != middleID:
		t.Fatalf("the quoting reply's quote is %s, want the post it quotes (%s)", quoting.Quote.ID, middleID)
	case quoting.Quote.Quote == nil:
		t.Fatalf("the quoted post carries no quote, want the post it quotes (%s): the chain is two levels deep", innerID)
	case quoting.Quote.Quote.ID != innerID:
		t.Fatalf("the quoted post's own quote is %s, want the post it quotes (%s)", quoting.Quote.Quote.ID, innerID)
	case quoting.Quote.Quote.Quote != nil:
		t.Errorf("the page carries a third level of quote, want the two the body carries")
	}

	// The module's own handle: the group that holds it, and the page's own next cursor when the
	// answer carried no bottom cursor.
	var wantHandle string
	for _, entry := range entries {
		for _, item := range entry.Content.Items {
			if item.Item.ItemContent.Typename == "TimelineTimelineCursor" {
				wantHandle = item.Item.ItemContent.Value
			}
		}
	}
	if wantHandle == "" {
		t.Fatal("the body carries no module cursor, so the handle assertions below prove nothing")
	}
	if got := rawBottomCursor(t, entries); got != "" {
		t.Fatalf("the body carries a bottom cursor (%q), so it is not the body behind the handle path", got)
	}
	if page.Next != domain.Cursor(wantHandle) {
		t.Errorf("Next is %q, want the module's own handle %q: a page with no bottom cursor continues with a handle a group still holds", page.Next, wantHandle)
	}
	held := false
	for _, group := range page.Groups {
		if group.More == domain.Cursor(wantHandle) {
			held = true
		}
	}
	if !held {
		t.Errorf("no group holds the handle %q: a cursor the answer carried cannot be dropped", wantHandle)
	}

	// The visibility cases the synthetic set carries and the recording does not.
	for _, want := range []struct {
		typ  string
		vis  domain.Visibility
		name string
	}{
		{"TweetTombstone", domain.Unavailable, "the tombstone reply"},
		{"TweetWithVisibilityResults", domain.Limited, "the limited reply"},
	} {
		found := 0
		for id, post := range posts {
			if post.Node.Typename != want.typ {
				continue
			}
			found++
			decoded, ok := pagePostOf(page, id)
			if !ok {
				t.Errorf("%s (%s) is not in the page", want.name, id)
				continue
			}
			if decoded.Visibility != want.vis {
				t.Errorf("%s (%s) decoded as %s, want %s", want.name, id, decoded.Visibility, want.vis)
			}
		}
		if found != 1 {
			t.Errorf("the body carries %d posts of type %s, so %s proves nothing", found, want.typ, want.name)
		}
	}
	// The withheld reply: the one visibility rule with no measured shape. It is found by the field
	// the rule reads.
	withheld := 0
	for id, post := range posts {
		if post.Node.Legacy == nil || len(post.Node.Legacy.WithheldInCountries) == 0 {
			continue
		}
		withheld++
		decoded, ok := pagePostOf(page, id)
		if !ok {
			t.Fatalf("the withheld post %s is not in the page: a withheld post is a post, not an error", id)
		}
		if decoded.Visibility != domain.Withheld {
			t.Errorf("the withheld post %s decoded as %s, want Withheld", id, decoded.Visibility)
		}
	}
	if withheld != 1 {
		t.Errorf("the body carries %d posts with withheld_in_countries, want the 1 it was written with", withheld)
	}

	// A tombstone's identity is its entry id: it has no rest_id of its own, which is why the entry
	// it arrived in is where its id lives.
	tombstones := 0
	for _, post := range posts {
		if post.Node.Typename != "TweetTombstone" {
			continue
		}
		tombstones++
		want := domain.PostID(strings.TrimPrefix(post.EntryID, tweetEntryPrefix))
		if _, ok := pagePostOf(page, want); !ok {
			t.Errorf("the tombstone entry %s names the post %s and the page does not carry it", post.EntryID, want)
		}
	}
	if tombstones != 1 {
		t.Errorf("the body carries %d tombstones, want the 1 it was written with", tombstones)
	}

	// What the answer cost, and what was dropped: 12 tweet items (the focal post, three ancestors,
	// two self-thread posts, the quoting reply, the cursor module's reply, the tombstone, the
	// limited reply, the withheld reply and the advertisement) and five entries that are not content
	// (the rail, the message prompt, the composer slot, the advertisement and the entry class this
	// build has never seen).
	if page.Requested != 12 {
		t.Errorf("Requested is %d, want 12: every tweet-bearing item the answer carried, before anything was dropped", page.Requested)
	}
	if page.Skipped != 5 {
		t.Errorf("Skipped is %d, want 5: the rail, the message prompt, the composer slot, the advertisement and the unknown entry class", page.Skipped)
	}
}

// TestNoPostFromTheRelatedRailOrAnAdvertisementReachesThePage is the rail's own guard: the answer
// carries a recommended post and an advertisement, the walker counts both, and neither reaches any
// part of the page — not the groups, not the ancestors, not the focal post, not a quote.
func TestNoPostFromTheRelatedRailOrAnAdvertisementReachesThePage(t *testing.T) {
	shapes := readFixture(t, shapesPath)
	page := mustParseThread(t, shapes, fixtureFocalID(t, shapesPath))
	posts := rawPosts(t, detailEntries(t, shapes))

	// The arm: the body has to carry both, or this test would pass on a body that has neither.
	rail, adverts := 0, 0
	for _, post := range posts {
		switch {
		case post.Rail:
			rail++
		case post.Ad:
			adverts++
		}
	}
	if rail != 1 || adverts != 1 {
		t.Fatalf("the body carries %d recommended posts and %d advertisements, want one of each", rail, adverts)
	}

	kept := keptPostIDs(page)
	for id, post := range posts {
		if !post.Rail && !post.Ad {
			continue
		}
		what := "advertised"
		if post.Rail {
			what = "recommended"
		}
		if kept[id] {
			t.Errorf("the %s post %s reached the page: neither a rail nor an advertisement is conversation content", what, id)
		}
	}
}

// TestTheEmptyAnswerIsAFirstPageWithNoReplies is the body a cold thread read can honestly be
// answered with: the focal post, no replies, and the two facts the honest partial-thread state
// reads — a reply count above zero, and a post above that the page does not hold.
func TestTheEmptyAnswerIsAFirstPageWithNoReplies(t *testing.T) {
	empty := readFixture(t, emptyPath)
	focalID := fixtureFocalID(t, emptyPath)
	page := mustParseThread(t, empty, focalID)
	entries := detailEntries(t, empty)

	if page.Focal.ID != focalID {
		t.Errorf("the focal post is %q, want %q", page.Focal.ID, focalID)
	}
	if len(page.Groups) != 0 {
		t.Errorf("the page carries %d groups, want none: the body carries no module", len(page.Groups))
	}
	if page.Next != "" {
		t.Errorf("Next is %q, want empty: the body carries no cursor, and an empty Next is the end of the walk rather than a failure", page.Next)
	}
	if page.Focal.Counts.Replies <= 0 {
		t.Errorf("the focal post reports %d replies, want the body's count above zero: it is the evidence behind the third partial-thread condition", page.Focal.Counts.Replies)
	}
	if page.Focal.ReplyTo == "" {
		t.Error("the focal post answers nothing, but the body has it answering a post: the chain-cut condition reads this")
	}

	// The id the focal post answers is one the body does not carry at all, which is what makes the
	// chain above it unreachable rather than merely empty.
	kept := keptPostIDs(page)
	if kept[page.Focal.ReplyTo] {
		t.Errorf("the focal post answers %q and the page carries that post, so this body is not the chain-cut case", page.Focal.ReplyTo)
	}
	if page.Requested != 1 || page.Skipped != 0 {
		t.Errorf("the body reports Requested %d and Skipped %d, want 1 and 0: one tweet item and no cursor or decoration", page.Requested, page.Skipped)
	}

	// The arm: the id the focal post answers is nowhere else in the body either.
	for id := range rawPosts(t, entries) {
		if id == page.Focal.ReplyTo {
			t.Errorf("the body carries the post %s the focal post answers, so this body is not the chain-cut case", id)
		}
	}
}

// TestTheTombstoneFocalIsAnUnavailableFocal is the adversarial frame: the post the thread is about
// is gone, and the conversation under it is still worth reading. Nothing about it is an error.
func TestTheTombstoneFocalIsAnUnavailableFocal(t *testing.T) {
	raw := readFixture(t, tombstonePath)
	focalID := fixtureFocalID(t, tombstonePath)
	page := mustParseThread(t, raw, focalID)
	entries := detailEntries(t, raw)

	if page.Focal.ID != focalID {
		t.Errorf("the focal post is %q, want the id its entry id names (%q)", page.Focal.ID, focalID)
	}
	if page.Focal.Visibility != domain.Unavailable {
		t.Errorf("the focal post is %s, want unavailable", page.Focal.Visibility)
	}
	if len(page.Groups) != 2 {
		t.Errorf("the page carries %d groups, want the body's 2 replies", len(page.Groups))
	}
	for _, group := range page.Groups {
		if len(group.Posts) != 1 {
			t.Errorf("a reply group carries %d posts, want 1", len(group.Posts))
		}
	}

	// The arm: the body's focal entry really is a tombstone, so the assertions above are about the
	// shape they name.
	if first := firstEntryID(entries); first != tweetEntryPrefix+string(focalID) {
		t.Fatalf("the body's first entry is %q, want the tombstone's own entry", first)
	}
	if got := rawTweetOf(t, entries[0].Content.ItemContent.TweetResults.Result); got.Typename != "TweetTombstone" {
		t.Fatalf("the body's focal entry carries a %s, want a TweetTombstone", got.Typename)
	}
	if page.Requested != 3 || page.Skipped != 0 {
		t.Errorf("the body reports Requested %d and Skipped %d, want 3 and 0", page.Requested, page.Skipped)
	}
}

// TestAMissingFocalFirstPageIsRefused is the one entry-level error this phase's read has: a first
// page without the post the request was about answered a different question, and rendering it would
// show an empty conversation with no sign that anything went wrong. A later page may carry none.
func TestAMissingFocalFirstPageIsRefused(t *testing.T) {
	raw := threadBody(threadModule("conversationthread-4242", "conversationthread-4242-tweet-4242", "Tweet", tweetResult("4242", "a reply with no focal post above it", "someone", "")))
	focal := domain.PostID("9999")

	// The parser itself does not refuse it: whether a page has to carry the focal post is a fact
	// about which page was asked for, and only the adapter knows that.
	if page := mustParseThread(t, raw, focal); page.Focal.ID != "" {
		t.Errorf("ParseThread found a focal post (%q) in a body that carries none", page.Focal.ID)
	}

	client := &stubClient{body: raw}
	page, err := New(client).Thread(context.Background(), focal, "")
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("a first page with no focal post reported %v, want a *ParseError", err)
	}
	if !strings.Contains(parseErr.Path, detailPath) {
		t.Errorf("the refusal names %q, want the path %q", parseErr.Path, detailPath)
	}
	if !strings.Contains(parseErr.Detail, string(focal)) {
		t.Errorf("the refusal says %q, want it to name the post that was asked for", parseErr.Detail)
	}
	if len(page.Groups) != 0 || page.Focal.ID != "" {
		t.Errorf("a refused page carried %+v, want nothing", page)
	}

	// A later page: the same body, asked for with a cursor, is a page of replies and no error — the
	// screen keeps the focal post it read from page one.
	client = &stubClient{body: raw}
	page, err = New(client).Thread(context.Background(), focal, "a-cursor-from-the-page-before")
	if err != nil {
		t.Fatalf("a later page with no focal post reported %v, want it accepted", err)
	}
	if page.Focal.ID != "" {
		t.Errorf("a later page carried the focal post %q, want none", page.Focal.ID)
	}
	if len(page.Groups) != 1 {
		t.Errorf("a later page carries %d groups, want the 1 reply the body carries", len(page.Groups))
	}
}

// TestAnUnknownTweetShapeIsATypedError is the contrast the package is built on: an entry class this
// build does not know is skipped and counted, while a post whose shape moved is a typed error
// naming the path. One is a decoration; the other is a post that would be rendered wrong.
func TestAnUnknownTweetShapeIsATypedError(t *testing.T) {
	// An entry class nobody has named: skipped, counted, not an error.
	unknownEntry := threadBody(
		`{"entryId":"who-to-follow-1","content":{"__typename":"TimelineTimelineItem","itemContent":{"__typename":"TimelineUser","user_results":{"result":{"__typename":"User","rest_id":"7","core":{"screen_name":"somebody"}}}}}}`,
	)
	page := mustParseThread(t, unknownEntry, domain.PostID("77"))
	if page.Skipped != 1 || len(page.Groups) != 0 || page.Focal.ID != "" {
		t.Errorf("an entry class this build does not know produced %+v, want one skip and nothing else", page)
	}

	for _, tc := range []struct {
		name  string
		body  []byte
		want  string
		entry string
	}{
		{
			name:  "a result type this build does not know",
			body:  threadBody(tweetItem("tweet-77", `{"__typename":"TweetUnavailable","rest_id":"77"}`)),
			want:  "is not a tweet shape this parser knows",
			entry: "tweet-77",
		},
		{
			name:  "a post with no legacy section",
			body:  threadBody(tweetItem("tweet-78", `{"__typename":"Tweet","rest_id":"78"}`)),
			want:  "legacy",
			entry: "tweet-78",
		},
		{
			name:  "an answer with no instructions at all",
			body:  []byte(`{"data":{"something_else":{}}}`),
			want:  detailPath,
			entry: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseThread(tc.body, domain.PostID(strings.TrimPrefix(tc.entry, tweetEntryPrefix)))
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("ParseThread reported %v, want a *ParseError", err)
			}
			if !strings.Contains(parseErr.Error(), tc.want) {
				t.Errorf("the error is %q, want it to carry %q", parseErr.Error(), tc.want)
			}
			if tc.entry != "" && !strings.Contains(parseErr.Error(), tc.entry) {
				t.Errorf("the error is %q, want it to name the entry %s it came from", parseErr.Error(), tc.entry)
			}
		})
	}
}

// TestTheAdapterAsksForOnePage pins the request half of the port: one call per page, the operation
// named rather than an id, the focal post and the cursor as the only call-specific variables, and
// no request of the read path's own after the answer arrives.
func TestTheAdapterAsksForOnePage(t *testing.T) {
	frozen := time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)
	restore := now
	now = func() time.Time { return frozen }
	defer func() { now = restore }()

	raw := readFixture(t, recordedDetailPath)
	focalID := fixtureFocalID(t, recordedDetailPath)
	client := &stubClient{body: raw}

	page, err := New(client).Thread(context.Background(), focalID, "")
	if err != nil {
		t.Fatalf("Thread: %v", err)
	}
	switch {
	case client.calls != 1:
		t.Errorf("the read issued %d requests, want exactly 1: one request is one page", client.calls)
	case client.op != opTweetDetail:
		t.Errorf("the read named the operation %q, want %q", client.op, opTweetDetail)
	case client.vars["focalTweetId"] != string(focalID):
		t.Errorf("the request sent focalTweetId = %v, want the post it was asked for", client.vars["focalTweetId"])
	case client.vars["cursor"] != "":
		t.Errorf("the first page sent cursor = %v, want an empty one", client.vars["cursor"])
	case !page.FetchedAt.Equal(frozen):
		t.Errorf("the page is stamped %s, want the frozen clock's %s", page.FetchedAt, frozen)
	case page.Next == "":
		t.Error("the page carries no next cursor, want the body's own")
	}

	// The three variables the upstream requires of any call to this operation, with the value the
	// measurement settled on: each was named by a 422 on 2026-10-15, and the request answered 200
	// once all three were present. A later edit that drops one of them fails here rather than on a
	// live thread read.
	for _, name := range []string{"includePromotedContent", "withBirdwatchNotes", "withVoice"} {
		if value, ok := client.vars[name]; !ok {
			t.Errorf("the request does not send %s, which the upstream's own 422 named as required (2026-10-15)", name)
		} else if value != false {
			t.Errorf("the request sends %s = %v, want false", name, value)
		}
	}
	if got := len(client.vars); got != 5 {
		t.Errorf("the request sent %d variables (%v), want exactly the 5 measured ones", got, client.vars)
	}

	// The answer's own cursor is what the next request is made with, and asking for that page is one
	// more request — never a walk of the read path's own.
	nextPage, err := New(client).Thread(context.Background(), focalID, page.Next)
	if err != nil {
		t.Fatalf("Thread with a cursor: %v", err)
	}
	if client.calls != 2 {
		t.Errorf("asking for the next page issued %d requests in all, want 2", client.calls)
	}
	if got := client.vars["cursor"]; got != string(page.Next) {
		t.Errorf("the second request sent cursor = %v, want the page's own Next (%q)", got, page.Next)
	}
	if nextPage.Focal.ID != focalID {
		t.Errorf("the second page's focal post is %q, want the post the request named", nextPage.Focal.ID)
	}
}

// TestNoGoFileInThisPackageSpellsTheQueryID is XAPI-02 at this package's boundary: the adapter names
// the operation and the registry holds the id, so no file here may spell one. The needle is read
// from the registry at run time, which is what lets this test's own file pass it.
func TestNoGoFileInThisPackageSpellsTheQueryID(t *testing.T) {
	op, err := volatile.Lookup(opTweetDetail)
	if err != nil {
		t.Fatalf("the registry has no %s: %v", opTweetDetail, err)
	}
	if op.QueryID == "" {
		t.Fatalf("the registry pins no id for %s", opTweetDetail)
	}
	id := []byte(op.QueryID)

	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("listing this package: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("the scan found no files, so it cannot fail and proves nothing")
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		if bytes.Contains(data, id) {
			t.Errorf("%s spells the %s query id: operation ids are data (XAPI-02) and live in internal/xweb/volatile/registry.json, resolved through volatile.Lookup", path, opTweetDetail)
		}
	}
}
