package xsource

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/LZD121/x-terminal/internal/domain"
)

// opTweetDetail is the one operation this file reads, by the name the registry holds. It is a name
// rather than a query id on purpose: an id is only valid until x.com ships a new bundle, and a
// literal one here would be a stale id waiting to be compiled in (XAPI-02). The id lives in
// internal/xweb/volatile/registry.json and nowhere else in Go.
const opTweetDetail = "TweetDetail"

// detailPath is where a TweetDetail answer keeps its instructions. It is one path rather than two
// candidates, unlike the timeline's: this operation has answered under this key in every measured
// response, and a second candidate would be a second shape to keep in step for a body nobody has
// seen.
const detailPath = "data.threaded_conversation_with_injections_v2.instructions"

// The entry-id marks and item values this walker classifies by. They are the upstream's own
// vocabulary, which is exactly why they live here and not in internal/domain: nothing above this
// package has ever seen an entry id, a module or a cursor type, and nothing above it may.
const (
	// tweetEntryPrefix marks an entry that is a post outside a module: the focal post itself, a
	// post in the chain above it, or a post the answer carried in the reply stream on its own.
	tweetEntryPrefix = "tweet-"
	// threadEntryPrefix marks a conversation module: the upstream's own block for a reply and,
	// when the answer ranked it, the handle into more of that block.
	threadEntryPrefix = "conversationthread-"
	// railEntryPrefix marks the related-tweets rail: a recommendation surface this program drops
	// rather than renders (04-CONTEXT.md), and the decoder classifies it so that the drop is
	// counted instead of silent.
	railEntryPrefix = "tweetdetailrelatedtweets"
	// promptEntryPrefix marks a direct-message prompt slot.
	promptEntryPrefix = "messageprompt-"
	// composerEntryMark marks a composer entry. It is a mark rather than a prefix because the
	// entry ids that carry it are longer than the word: what identifies it is that it mentions a
	// composer at all.
	composerEntryMark = "tweetcomposer"
	// promotedItemMark marks an item that is an advertisement placed in a conversation module: an
	// entry id carrying it is not this conversation's content, whichever conversation it is from.
	promotedItemMark = "-promoted-tweet-"
	// selfThreadDisplay is what an item's tweetDisplayType says when the focal post is continuing
	// its own thought rather than answering someone else.
	selfThreadDisplay = "SelfThread"
	// bottomCursor is the cursor type that says where this answer stops. The other cursor types a
	// module can carry are its own handles.
	bottomCursor = "Bottom"
	// The two cursor types that are a module's own handle into more of that block.
	showMoreCursor        = "ShowMore"
	showMoreThreadsCursor = "ShowMoreThreads"
)

// Adapter implements the port's thread read. The assertion is here rather than in the tests
// because a signature change in one of the two packages has to fail a build rather than a screen
// run.
var _ domain.Threads = (*Adapter)(nil)

// Thread returns one page of the conversation containing a post: the chain above it, the post
// itself and one page of its replies.
//
// One request is one page. The focal id and the cursor are the call-specific part of the request;
// the query id, the feature switches and the field toggles come from the volatile registry inside
// the transport (XAPI-02); and the three variables below are what the upstream requires of ANY
// call to this operation, whatever it is about — measured 2026-10-15, TweetDetail with only
// focalTweetId and cursor answered 422 GRAPHQL_VALIDATION_FAILED naming includePromotedContent,
// then the same for withBirdwatchNotes, then the same for withVoice, and answered 200 once all
// three were present. They are sent as false: the value the upstream was measured to require
// rather than one it was shown to demand, and the one that keeps promoted posts out of a
// conversation this renderer marks no advertisement in.
//
// Nothing here expands a cursor into a further request: the page's own Next is what a later call
// is made with, and the walk above this package decides when to make it.
//
// A first page that carries no focal post is refused: an answer without the post the request was
// about is an answer to a different question, and rendering it would show an empty conversation
// with no sign that anything went wrong. A later page may carry none — the screen still holds the
// focal post it read from page one — and is returned as it is.
func (a *Adapter) Thread(ctx context.Context, id domain.PostID, c domain.Cursor) (domain.ThreadPage, error) {
	raw, err := a.client.GraphQL(ctx, opTweetDetail, map[string]any{
		"focalTweetId":           string(id),
		"cursor":                 string(c),
		"includePromotedContent": false,
		"withBirdwatchNotes":     false,
		"withVoice":              false,
	})
	if err != nil {
		// Passed through unchanged: the transport has already classified it into a typed error
		// naming its state, and re-wrapping it would put this package between the caller and the
		// sentence the user needs.
		return domain.ThreadPage{}, err
	}

	page, err := ParseThread(raw, id)
	if err != nil {
		return domain.ThreadPage{}, err
	}
	page.FetchedAt = now()

	if c == "" && page.Focal.ID == "" {
		return domain.ThreadPage{}, &ParseError{
			Path:   detailPath,
			Detail: "carries no entry for the post " + string(id) + ", which is the post this first page was asked for",
		}
	}
	return page, nil
}

// detailEnvelope is the shape of a TweetDetail answer: one instruction list under one key. The
// entry, item and module shapes under it are the timeline decoder's own — one description of the
// wire in this package, not two.
type detailEnvelope struct {
	Data struct {
		Threaded struct {
			Instructions []instruction `json:"instructions"`
		} `json:"threaded_conversation_with_injections_v2"`
	} `json:"data"`
}

// ParseThread reads a TweetDetail answer into one page of a conversation.
//
// The answer is a ranked flat list of entries, not a tree, and this walker keeps it flat: what it
// produces is the answer's own order — the chain above the focal post in the order it arrived
// (root first), the focal post, and one group per reply module — with every post's parent left as
// the reply-to id the body carried. Nothing here re-sorts and nothing here derives parentage: a
// tree rebuilt from the reply-to ids would look right on the recorded body and be wrong on the
// next one, because the upstream's list is ranked rather than threaded.
//
// focal is the post the request was for, and it is an argument rather than something derived from
// the body because the body is exactly what cannot say it: nothing in an answer marks the post
// that was asked for as the last link of a chain, and a walker that guessed at it would be
// deriving the tree it refuses to derive.
//
// The two rules about what it reads are different on purpose. An entry class this build does not
// know is skipped and counted: the entry stream is a ranked, ad-laden container that ships new
// decorations without notice, and a skipped decoration must not become a blank row. A post whose
// body, author or date moved is a typed error naming the JSON path, because that is the silent
// wrongness this package refuses to guess at.
func ParseThread(raw []byte, focal domain.PostID) (domain.ThreadPage, error) {
	var envelope detailEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return domain.ThreadPage{}, unreadable("the response body", err)
	}
	instructions := envelope.Data.Threaded.Instructions
	if instructions == nil {
		return domain.ThreadPage{}, missing(detailPath)
	}

	walk := threadWalker{focal: focal}
	for _, instruction := range instructions {
		// Only TimelineAddEntries carries entries. A TimelineClearCache or a
		// TimelineTerminateTimeline says something about the connection, not about the page.
		if instruction.Type != "TimelineAddEntries" {
			continue
		}
		for _, entry := range instruction.Entries {
			if err := walk.entry(entry); err != nil {
				return domain.ThreadPage{}, err
			}
		}
	}

	// Next is the page's own bottom cursor when the answer carried one, and otherwise the first
	// unspent handle a group carries. Either way it is opaque here and the screen only stores it;
	// a cursor the answer carried is never dropped, which is why the fallback is a search for one
	// rather than a look at the first group alone.
	switch {
	case walk.bottom != "":
		walk.page.Next = domain.Cursor(walk.bottom)
	case walk.handle != "":
		walk.page.Next = domain.Cursor(walk.handle)
	}
	return walk.page, nil
}

// threadWalker is the state one answer is read into: the page being built, the post the request
// was for, and the two cursors an answer can offer.
type threadWalker struct {
	focal domain.PostID
	page  domain.ThreadPage
	// seenFocal is whether the focal entry has arrived. It is what tells the chain above the
	// focal post from the posts the answer carried after it.
	seenFocal bool
	// bottom is the page's own cursor, when the answer carried one; handle is the first unspent
	// handle a group carried, when it did not.
	bottom string
	handle string
}

// entry classifies one entry of the answer by its id and its content, in that order of cheapness,
// and folds it into the page or counts it.
func (w *threadWalker) entry(entry entryNode) error {
	content := entry.Content
	switch {
	case content.Typename == "TimelineTimelineCursor":
		return w.cursor(content)
	case strings.HasPrefix(entry.EntryID, tweetEntryPrefix):
		return w.tweetEntry(entry.EntryID, content)
	case strings.HasPrefix(entry.EntryID, threadEntryPrefix):
		return w.module(content)
	case strings.HasPrefix(entry.EntryID, railEntryPrefix),
		strings.HasPrefix(entry.EntryID, promptEntryPrefix),
		strings.Contains(entry.EntryID, composerEntryMark):
		// The named decorations. A rail is an algorithmic recommendation surface and rendering it
		// would put posts the reader did not ask for into the conversation; a message prompt and a
		// composer slot are furniture. None of them is content, and each is counted so that one
		// appearing where it did not before is visible rather than invisible.
		w.page.Skipped++
		return nil
	}
	// Everything else the entry stream ships: an entry class this build has never seen, a module
	// whose id names no block this program reads, a user suggestion. Skipped and counted, never
	// refused and never rendered as a blank row.
	w.page.Skipped++
	return nil
}

// cursor folds a cursor entry into the page. Every cursor entry is counted as skipped, including
// the bottom one that sets the page's own cursor: a cursor is a position, not content, and a page
// that counted only its posts would hide the entry that told the walk where to stop.
func (w *threadWalker) cursor(content entryBody) error {
	w.page.Skipped++
	if content.CursorType == bottomCursor {
		w.bottom = content.Value
	}
	return nil
}

// tweetEntry folds one post the answer carried outside a module into the page.
//
// Three cases, and the entry's own id is what tells them apart: the entry for the post the request
// was for is the focal post; a post entry before that one is a link of the chain above it,
// appended in the answer's own order, which is root first; and a post entry after it is a post the
// answer carried in the reply stream on its own, which is a group of one — dropping it would drop
// a post the answer carried, and nesting it into another post would be the tree this walker does
// not build.
func (w *threadWalker) tweetEntry(entryID string, content entryBody) error {
	item := content.ItemContent
	if item == nil || item.Typename != "TimelineTweet" {
		w.page.Skipped++
		return nil
	}
	path := entryID + ": content.itemContent.tweet_results.result"

	switch id := strings.TrimPrefix(entryID, tweetEntryPrefix); {
	case id == string(w.focal):
		if w.seenFocal {
			// The answer carried the focal entry twice. One post, one place: the second copy is a
			// decoration and is counted rather than rendered as a second focal post.
			w.page.Skipped++
			return nil
		}
		w.seenFocal = true
		post, err := w.post(entryID, path, *item)
		if err != nil {
			return err
		}
		w.page.Focal = post
		return nil
	case !w.seenFocal:
		post, err := w.post(entryID, path, *item)
		if err != nil {
			return err
		}
		w.page.Ancestors = append(w.page.Ancestors, post)
		return nil
	default:
		post, err := w.post(entryID, path, *item)
		if err != nil {
			return err
		}
		w.page.Groups = append(w.page.Groups, domain.Group{Kind: domain.Replies, Posts: []domain.Post{post}})
		return nil
	}
}

// module folds one conversation module into a group: the posts it carries, whether the block is
// the focal post continuing itself or the conversation under it, and — when the answer ranked it —
// the module's own cursor, which is a handle into more of that block.
//
// A module whose item is a promoted tweet is an advertisement from another conversation: it is
// counted and dropped, and it forms no group, so nothing from it can reach a pane. A module that
// yields neither a post nor a handle is a decoration of some class this build does not read, and
// is counted too.
func (w *threadWalker) module(content entryBody) error {
	group := domain.Group{Kind: domain.Replies}
	posts, ads, handle := 0, 0, ""

	for i, item := range content.Items {
		content := item.Item.ItemContent
		if content == nil {
			continue
		}
		switch content.Typename {
		case "TimelineTimelineCursor":
			if handle == "" && (content.CursorType == showMoreCursor || content.CursorType == showMoreThreadsCursor) {
				handle = content.Value
			}
		case "TimelineTweet":
			if strings.Contains(item.EntryID, promotedItemMark) {
				// An advertisement: the answer carried it, so it is part of what the answer cost,
				// and it is not this conversation, so it is not content.
				w.page.Requested++
				ads++
				continue
			}
			path := item.EntryID + ": content.items[" + strconv.Itoa(i) + "].item.itemContent.tweet_results.result"
			post, err := w.post(item.EntryID, path, *content)
			if err != nil {
				return err
			}
			group.Posts = append(group.Posts, post)
			posts++
			if content.TweetDisplayType == selfThreadDisplay {
				group.Kind = domain.SelfThread
			}
		}
	}

	w.page.Skipped += ads
	if posts == 0 && handle == "" {
		if ads == 0 {
			// A module of user cards, of a class this build has never seen, or of nothing at all.
			w.page.Skipped++
		}
		return nil
	}
	group.More = domain.Cursor(handle)
	if handle != "" && w.handle == "" {
		w.handle = handle
	}
	w.page.Groups = append(w.page.Groups, group)
	return nil
}

// post builds one post out of one item's tweet result, through the same decoder the timeline uses:
// unwrapTweet for the visibility the result type grants, postFrom for the post and its quote chain
// (bounded by the existing maxQuoteDepth, which is where a nested quote comes from), countsOf for
// the counters a screen consults and never prints, and authorOf for the byline.
//
// Requested is counted before anything can fail: it is what the answer carried, not what the page
// could read.
func (w *threadWalker) post(entryID, path string, content itemContent) (domain.Post, error) {
	w.page.Requested++

	node, granted, err := unwrapTweet(path, content.TweetResults.Result)
	if err != nil {
		return domain.Post{}, err
	}
	post, err := postFrom(entryID, path, node, granted, 1)
	if err != nil {
		return domain.Post{}, err
	}
	if withholds(node) {
		// The one visibility rule this build has no measured shape for (see withholds).
		post.Visibility = domain.Withheld
	}
	return post, nil
}

// withholds reports whether a node's own legacy says the upstream will not serve this post
// everywhere.
//
// This is the one visibility state no recording in this project carries, so the rule here is an
// inference and it is the narrowest one available: a result whose legacy carries a non-empty
// withheld_in_countries is domain.Withheld, whether or not the result is also a tombstone, and any
// other unknown result type stays a typed error from unwrapTweet rather than a guess. The
// synthetic shapes fixture is the only body that exercises it, and its provenance row says so.
//
// The visibility is applied to the post after it is built rather than passed into postFrom, because
// the grant and the post's shape are two different things: a tombstone's body is built from its
// entry id whatever it is also withheld in, and that is why the withheld tombstone reads as a post
// with an id and no body rather than as an error.
func withholds(node tweet) bool {
	return node.Legacy != nil && len(node.Legacy.WithheldInCountries) > 0
}
