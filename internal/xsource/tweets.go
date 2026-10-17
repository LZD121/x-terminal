package xsource

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// timelineTime is the format created_at is written in: "Tue Oct 13 22:16:43 +0000 2026", the
// Twitter format, in the upstream's own clock (UTC in every measured response).
const timelineTime = "Mon Jan 02 15:04:05 -0700 2006"

// Page is one page of a timeline: the posts it carried, the cursors a caller scrolls with, and
// how much the answer carried. A page with no posts and no cursors is not a page, and
// ParseTimeline refuses to return one.
type Page struct {
	Posts        []domain.Post
	TopCursor    string
	BottomCursor string
	// Requested is how many tweet entries the answer carried: counted as the entries are read,
	// before a repost is collapsed into the post it reposts and before anything above this
	// package de-duplicated them. It is what the answer cost, not what it contributed, which is
	// exactly what a caller needs to tell "the answer carried 20 entries and none of them were
	// new" from "the answer carried nothing at all".
	Requested int
}

// ParseTimeline reads a timeline out of a response body — UserTweets and its siblings — into
// domain posts.
//
// The entries live at data.user.result.timeline.timeline.instructions in the responses measured
// on 2026-10-14. data.user.result.timeline_v2.timeline.instructions is accepted as well: it is
// the path the earlier research named, the same endpoint has answered under both, and refusing
// one of two shapes this parser can read would be a way to lose a whole timeline to a
// deployment nobody announced.
//
// data.home.home_timeline_urt.instructions is the third envelope accepted here, and it is the one
// the two home timelines answer under: measured 2026-10-15, HomeLatestTimeline (the Following
// feed, which bare x-terminal opens on) and HomeTimeline (For-you) both answered 200 with their
// instructions at that path. Its instructions are NOT behind a `timeline` object — they are a
// direct sibling of its metadata — which is the second nesting timelineNode reads. Same entry
// grammar either way (TimelineAddEntries with TimelineTimelineItem/TimelineTweet items, a Top and
// a Bottom TimelineTimelineCursor, TimelineTimelineModule entries), so it is an envelope arm and
// not a second walk.
//
// What it takes from the instructions:
//
//   - the cursor entries, into TopCursor and BottomCursor, because a caller that cannot page
//     cannot read a timeline;
//   - the tweet entries, in the order the upstream sent them, including the ones inside a
//     conversation module (the upstream puts a self-thread there, and dropping them would drop
//     posts from a timeline while still calling the result complete);
//   - a repost, collapsed into the post it reposts, with RepostedBy naming who reposted it.
//
// Anything else in an entry — a user suggestion, an entry type this build has never seen — is
// skipped, and an unknown *tweet* shape is an error: the difference is that an entry which is
// not a post is not evidence of a missing post, while a post whose body did not parse is.
func ParseTimeline(raw []byte) (Page, error) {
	var envelope timelineEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Page{}, unreadable("the response body", err)
	}

	timeline, path := timelineFrom(envelope)
	if timeline == nil {
		return Page{}, missing("data.user.result.timeline.timeline.instructions (and data.user.result.timeline_v2.timeline.instructions, and data.bookmark_timeline_v2.timeline.instructions, and data.home.home_timeline_urt.instructions)")
	}

	var page Page
	if err := page.walkTimeline(path, timeline); err != nil {
		return Page{}, err
	}
	return page, nil
}

// timelineFrom picks the instructions container the body carried: a user timeline
// (UserTweets / Likes), the bookmarks list, or one of the two home timelines. The entry walk is
// the same either way; only the envelope path differs.
//
// The home arm is the one the two feeds answer under: measured 2026-10-15, HomeLatestTimeline (the
// Following feed) and HomeTimeline (For-you) each answered 200 with their instructions at
// data.home.home_timeline_urt.instructions — the same path for both operations, which is why one
// arm covers them. That container carries no `timeline` object: its instructions are a direct
// sibling of its metadata, which is the second nesting timelineNode reads. Before the arm existed,
// bare x-terminal failed in this function while the request itself was accepted: the entries were
// the grammar the walker already handles and only the envelope was unknown.
func timelineFrom(envelope timelineEnvelope) (*timelineNode, string) {
	if timeline := envelope.Data.User.Result.Timeline; timeline != nil {
		return timeline, "data.user.result.timeline.timeline.instructions"
	}
	if timeline := envelope.Data.User.Result.TimelineV2; timeline != nil {
		return timeline, "data.user.result.timeline_v2.timeline.instructions"
	}
	if timeline := envelope.Data.BookmarkTimelineV2; timeline != nil {
		return timeline, "data.bookmark_timeline_v2.timeline.instructions"
	}
	if timeline := envelope.Data.Home.HomeTimelineUrt; timeline != nil {
		return timeline, "data.home.home_timeline_urt.instructions"
	}
	return nil, ""
}

// walkTimeline folds one timeline container's instructions into the page: the cursor entries into
// the page's two cursors, the tweet entries in the answer's order. It is the ONE entry loop every
// timeline envelope feeds — the user timelines ParseTimeline reads and the search container
// ParseSearch reads — so a cursor and an entry mean the same thing on every list the program
// shows, and a shape change in the entry walk has exactly one home.
//
// path is where the container's instructions live in the response, which is what an error names.
// An answer that carried nothing — no post, no cursor — is not an empty timeline: it is
// indistinguishable from a body whose shape moved, and no screen in this program may show it as
// "no posts".
func (p *Page) walkTimeline(path string, timeline *timelineNode) error {
	return walkInstructions(path, timeline, p)
}

// entrySink is what the shared instruction walk folds entries into: cursors and items. Page and
// peoplePage are the two sinks; the cursor arm lives once, in foldEntry, so the bottom/top
// cursor typename has exactly one reader.
type entrySink interface {
	setTopCursor(string)
	setBottomCursor(string)
	addItem(entryID string, content itemContent) error
	empty() bool
}

func (p *Page) setTopCursor(v string)    { p.TopCursor = v }
func (p *Page) setBottomCursor(v string) { p.BottomCursor = v }
func (p *Page) empty() bool {
	return len(p.Posts) == 0 && p.TopCursor == "" && p.BottomCursor == ""
}

// walkInstructions is the ONE entry loop: every timeline envelope feeds it. The sink decides
// what an item means (a post, an account); the cursors are folded here and nowhere else.
func walkInstructions(path string, timeline *timelineNode, sink entrySink) error {
	for _, instruction := range timeline.instructionList() {
		if instruction.Type != "TimelineAddEntries" {
			continue
		}
		for _, entry := range instruction.Entries {
			if err := foldEntry(entry, sink); err != nil {
				return err
			}
		}
	}

	if sink.empty() {
		return missing(path + ": the instructions carried no entries")
	}
	return nil
}

// foldEntry folds one entry into the sink: a cursor, an item, or a module of them. An entry type
// this build does not know is skipped rather than refused — the timeline sends entry types that
// are not posts (user suggestions, ad slots, and whatever ships next), and a timeline that lost
// its suggestions is still a timeline. The cursor typename arm lives here and only here.
func foldEntry(entry entryNode, sink entrySink) error {
	content := entry.Content
	switch {
	case content.Typename == "TimelineTimelineCursor":
		switch content.CursorType {
		case "Top":
			sink.setTopCursor(content.Value)
		case "Bottom":
			sink.setBottomCursor(content.Value)
		}
		return nil
	case content.ItemContent != nil:
		return sink.addItem(entry.EntryID, *content.ItemContent)
	}

	for _, module := range content.Items {
		if module.Item.ItemContent == nil {
			continue
		}
		if err := sink.addItem(module.EntryID, *module.Item.ItemContent); err != nil {
			return err
		}
	}
	return nil
}

// addItem reads one item of an entry or of a module. An item that is not a tweet is skipped:
// the conversation modules carry user cards beside their tweets.
func (p *Page) addItem(entryID string, content itemContent) error {
	if content.Typename != "TimelineTweet" {
		return nil
	}
	// An entry whose tweet_results.result is empty is not a post this build can show
	// (the likes list carries one of these beside ordinary tweets). Skip and count it
	// the same way a non-tweet entry is skipped, rather than refusing the page.
	if len(content.TweetResults.Result) == 0 {
		p.Requested++
		return nil
	}
	// Counted before anything can fail. Requested is what the answer carried, and an entry that
	// turned out to be unreadable was still carried.
	p.Requested++

	path := entryID + ": content.itemContent.tweet_results.result"
	node, visibility, err := unwrapTweet(path, content.TweetResults.Result)
	if err != nil {
		return err
	}

	// A repost is the post it reposts. The outer tweet names the account that reposted it and
	// carries the tweet that was reposted; the post is built from the inner one and marked with who
	// reposted it. When the answer carries no inner tweet at all (the upstream omits it for a post
	// it will not serve), what is left is a post whose text begins "RT @…", which is the honest
	// reading of the answer rather than an invented one.
	if reposted := node.repost(); reposted != nil {
		innerPath := path + ": retweeted_status_result.result"
		original, originalVisibility, err := unwrapTweet(innerPath, reposted.Result)
		if err != nil {
			return err
		}
		post, err := postFrom(entryID, innerPath, original, originalVisibility, 1)
		if err != nil {
			return err
		}
		reposter, err := authorOf(path, node)
		if err != nil {
			return err
		}
		post.RepostedBy = &reposter
		p.Posts = append(p.Posts, post)
		return nil
	}

	post, err := postFrom(entryID, path, node, visibility, 1)
	if err != nil {
		return err
	}
	p.Posts = append(p.Posts, post)
	return nil
}

// maxQuoteDepth bounds how far this parser follows a chain of quotes.
//
// Four levels is already more than any screen renders (the plain grammar caps the quote prefix
// at two), and it is a bound rather than plain recursion because the nesting is written by
// whoever wrote the post: a body can quote a post that quotes a post for as deep as the upstream
// will nest them, and a parser that followed that without a limit would be a stack overflow
// anyone could author.
const maxQuoteDepth = 4

// postFrom builds one post out of one tweet node and follows its quote chain up to
// maxQuoteDepth.
//
// entryID is the entry the post was read from, which is the only identity a tombstone has.
// path is where the node lives in the response, which is what an error from here names.
// visibility is what unwrapTweet made of the node's result type. depth is the nesting level of
// the post being built: 1 for a timeline entry, one deeper for each quote.
func postFrom(entryID, path string, node tweet, visibility domain.Visibility, depth int) (domain.Post, error) {
	post, err := postBody(entryID, path, node, visibility)
	if err != nil {
		return domain.Post{}, err
	}
	// An absent quote is not a parse failure. X answers `"quoted_status_result": {"result": null}`
	// for a post whose original has been deleted or withheld, and one such row must not cost the
	// whole page: measured on this host on 2026-10-17, a profile's first page carried exactly one
	// and the entire page was reported unreadable, so the profile showed "no posts" while the
	// account had thousands.
	if node.QuotedStatusResult == nil || isNullJSON(node.QuotedStatusResult.Result) {
		return post, nil
	}

	quotePath := path + ": quoted_status_result.result"
	quoted, quotedVisibility, err := unwrapTweet(quotePath, node.QuotedStatusResult.Result)
	if err != nil {
		return domain.Post{}, err
	}

	var nested domain.Post
	if depth >= maxQuoteDepth {
		// Past the bound the post is still a post — its id, its author and its body — and it
		// carries no quote of its own. Cutting the chain rather than the post is what keeps the
		// deepest link of a long chain from being dropped out of the answer entirely.
		nested, err = postBody(entryID, quotePath, quoted, quotedVisibility)
	} else {
		nested, err = postFrom(entryID, quotePath, quoted, quotedVisibility, depth+1)
	}
	if err != nil {
		return domain.Post{}, err
	}
	post.Quote = &nested
	return post, nil
}

// postBody builds everything a post has except its quote.
//
// The text is resolved in the order note_tweet.text → legacy.full_text: a post longer than the
// legacy limit is answered twice, with the truncated body in legacy and the whole one in
// note_tweet, and rendering the truncated one would cut the post off mid-sentence with no sign
// that anything was missing.
func postBody(entryID, path string, node tweet, visibility domain.Visibility) (domain.Post, error) {
	if visibility == domain.Unavailable {
		return tombstone(entryID, path, node)
	}
	if node.RestID == "" {
		return domain.Post{}, missing(path + ": rest_id")
	}
	if node.Legacy == nil {
		return domain.Post{}, missing(path + ": legacy")
	}

	text := node.Legacy.FullText
	if node.NoteTweet != nil && node.NoteTweet.Text != "" {
		text = node.NoteTweet.Text
	}
	if text == "" {
		return domain.Post{}, missing(path + ": legacy.full_text (and note_tweet.text)")
	}
	if node.Legacy.CreatedAt == "" {
		return domain.Post{}, missing(path + ": legacy.created_at")
	}
	created, err := time.Parse(timelineTime, node.Legacy.CreatedAt)
	if err != nil {
		// A date this build cannot read is not a date to guess at: a zero time sorts to the
		// bottom of every timeline and renders as a year nobody posted in.
		return domain.Post{}, &ParseError{Path: path + ": legacy.created_at", Detail: err.Error()}
	}

	author, err := authorOf(path, node)
	if err != nil {
		return domain.Post{}, err
	}
	media, err := mediaOf(path, node.Legacy)
	if err != nil {
		return domain.Post{}, err
	}
	counts, err := countsOf(path, node)
	if err != nil {
		return domain.Post{}, err
	}

	return domain.Post{
		ID:             domain.PostID(node.RestID),
		Author:         author,
		Text:           text,
		CreatedAt:      created,
		Lang:           node.Legacy.Lang,
		ConversationID: node.Legacy.ConversationIDStr,
		ReplyTo:        domain.PostID(node.Legacy.InReplyToStatusIDStr),
		Media:          media,
		Links:          linksOf(node.Legacy),
		Visibility:     visibility,
		Counts:         counts,
	}, nil
}

// tombstone builds the post a TweetTombstone stands for: a post that used to exist and that the
// upstream will no longer serve a body, an author or a date for.
//
// It is deliberately not an error. A tombstone in a timeline is evidence that a post was there —
// dropping it would shorten the timeline silently, which is the failure this whole package is
// written against — and it is a post with nothing to render, which is a state the screens have.
//
// Its identity is the one thing a tombstone must have, because everything above this package
// de-duplicates on it: the node's rest_id when the response carries one, and otherwise the entry
// id, which is where the upstream keeps it in every measured response (`tweet-<id>` and
// `profile-conversation-<id>-tweet-<id>` are the two shapes the recording holds).
func tombstone(entryID, path string, node tweet) (domain.Post, error) {
	id := node.RestID
	if id == "" {
		id = tombstoneID(entryID)
	}
	if id == "" {
		return domain.Post{}, missing(path + ": the tombstone's rest_id (and an entry id to read one from)")
	}
	return domain.Post{ID: domain.PostID(id), Visibility: domain.Unavailable}, nil
}

// tombstoneID reads a post id out of the entry a tombstone arrived in. When the entry id has no
// `tweet-` segment at all, the entry id itself is returned: it is still a value that names this
// entry and no other, which is what a post id is for above this package.
func tombstoneID(entryID string) string {
	if i := strings.LastIndex(entryID, "tweet-"); i >= 0 {
		return entryID[i+len("tweet-"):]
	}
	return entryID
}

// linksOf reads the three kinds of thing a body names, in the entities' order: URLs, then tags,
// then mentions, each in the answer's own order. Repeat names are dropped — a body that names one
// link twice names it once — and an entry with no target is skipped rather than carried as an empty
// row.
//
// The URL target is the expanded form when the answer carries one: the text holds t.co short links,
// and following one should land where the author meant, not in the shortener.
func linksOf(legacy *tweetLegacy) []domain.Link {
	if legacy == nil || legacy.Entities == nil {
		return nil
	}
	e := legacy.Entities
	out := make([]domain.Link, 0, len(e.URLs)+len(e.Hashtags)+len(e.UserMentions))
	seen := map[domain.Link]bool{}
	add := func(kind domain.LinkKind, target string) {
		if target == "" {
			return
		}
		link := domain.Link{Kind: kind, Target: target}
		if seen[link] {
			return
		}
		seen[link] = true
		out = append(out, link)
	}
	for _, u := range e.URLs {
		if u.ExpandedURL != "" {
			add(domain.LinkURL, u.ExpandedURL)
			continue
		}
		add(domain.LinkURL, u.URL)
	}
	for _, h := range e.Hashtags {
		add(domain.LinkHashtag, h.Text)
	}
	for _, m := range e.UserMentions {
		add(domain.LinkMention, m.ScreenName)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// mediaOf reads a post's attachments.
//
// extended_entities is preferred over entities and entities is the fallback: a post whose video
// was replaced by a longer cut carries the new media in extended_entities while entities still
// holds the old list, and both nestings have been the "current" one at some point. A post that
// has neither is a post with no attachment and gets no refs at all — an empty list would say
// "attachments, none of which I could read".
func mediaOf(path string, legacy *tweetLegacy) ([]domain.MediaRef, error) {
	var (
		media []mediaEntity
		name  string
	)
	switch {
	case legacy.ExtendedEntities != nil && len(legacy.ExtendedEntities.Media) > 0:
		media, name = legacy.ExtendedEntities.Media, "extended_entities"
	case legacy.Entities != nil && len(legacy.Entities.Media) > 0:
		media, name = legacy.Entities.Media, "entities"
	}
	if len(media) == 0 {
		return nil, nil
	}

	refs := make([]domain.MediaRef, 0, len(media))
	for i, media := range media {
		at := path + ": legacy." + name + ".media[" + strconv.Itoa(i) + "]"
		kind, err := mediaKind(at+".type", media.Type)
		if err != nil {
			return nil, err
		}
		if media.MediaURLHTTPS == "" {
			return nil, missing(at + ".media_url_https")
		}
		ref := domain.MediaRef{
			Kind:    kind,
			URL:     media.MediaURLHTTPS,
			AltText: media.ExtAltText,
		}
		if kind == domain.Video || kind == domain.GIF {
			playable, err := playableOf(at+".video_info", media.VideoInfo)
			if err != nil {
				return nil, err
			}
			ref.Playable = playable
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

// playableOf picks the URL a player is handed from video_info.variants.
//
// Among variants with content type video/mp4, take the one with the highest
// bitrate (a variant with no bitrate is ranked last and is still eligible when
// it is the only one); else take the first application/x-mpegURL variant; else
// leave Playable empty. mp4 needs no demuxer beyond what mpv ships with, and
// the manifest is the fallback because it is a stream this program does not
// otherwise reason about. Ties break by the response's own order (the first
// variant at the winning bitrate wins), so a re-parse of the same body gives
// the same bytes. An absent video_info is not an error — the poster is what
// OpenedPoster reports — but a variant whose url is missing is a parse error
// at its own path.
func playableOf(path string, info *videoInfo) (string, error) {
	if info == nil || len(info.Variants) == 0 {
		return "", nil
	}
	var (
		bestMP4    string
		bestRate   int
		haveMP4    bool
		bestHasBit bool
		hls        string
	)
	for i, v := range info.Variants {
		at := path + ".variants[" + strconv.Itoa(i) + "]"
		if v.URL == "" {
			return "", missing(at + ".url")
		}
		switch v.ContentType {
		case "video/mp4":
			rate := 0
			hasBit := v.Bitrate != nil
			if hasBit {
				rate = *v.Bitrate
			}
			switch {
			case !haveMP4:
				haveMP4, bestMP4, bestRate, bestHasBit = true, v.URL, rate, hasBit
			case hasBit && (!bestHasBit || rate > bestRate):
				bestMP4, bestRate, bestHasBit = v.URL, rate, true
			}
		case "application/x-mpegURL":
			if hls == "" {
				hls = v.URL
			}
		}
	}
	if haveMP4 {
		return bestMP4, nil
	}
	return hls, nil
}

// mediaKind maps the upstream's media type onto this program's. A type this build does not know
// is an error rather than a skip: a skipped attachment does not cost a line, it costs the whole
// attachment line, and a post that says it has one attachment when it has three is a lie of
// exactly the kind this package refuses.
func mediaKind(path, mediaType string) (domain.MediaKind, error) {
	switch mediaType {
	case "photo":
		return domain.Image, nil
	case "video":
		return domain.Video, nil
	case "animated_gif":
		return domain.GIF, nil
	}
	return 0, &ParseError{Path: path, Detail: "is " + mediaType + ", which is not a media type this build knows"}
}

// countsOf reads a post's engagement numbers.
//
// A counter the answer omits is zero: unlike an identity, a body or a date, an absent counter is
// not evidence that a shape moved, and nothing in this phase prints one. A counter that is there
// and is not a number is a shape this build cannot read, and says so by name.
func countsOf(path string, node tweet) (domain.Counts, error) {
	counts := domain.Counts{
		Replies:   node.Legacy.ReplyCount,
		Reposts:   node.Legacy.RetweetCount,
		Likes:     node.Legacy.FavoriteCount,
		Quotes:    node.Legacy.QuoteCount,
		Bookmarks: node.Legacy.BookmarkCount,
	}
	if node.Views == nil || len(node.Views.Count) == 0 || string(node.Views.Count) == "null" {
		return counts, nil
	}

	text := string(node.Views.Count)
	if text[0] == '"' {
		var quoted string
		if err := json.Unmarshal(node.Views.Count, &quoted); err != nil {
			return domain.Counts{}, unreadable(path+": views.count", err)
		}
		text = quoted
	}
	views, err := strconv.Atoi(text)
	if err != nil {
		return domain.Counts{}, &ParseError{Path: path + ": views.count", Detail: "is not a whole number"}
	}
	counts.Views = views
	return counts, nil
}

// authorOf reads the account a tweet is attributed to. A post without an author cannot be
// rendered as anything, so an absent one is an error rather than an empty byline.
func authorOf(path string, node tweet) (domain.User, error) {
	author := node.Core.UserResults.Result
	if author == nil {
		return domain.User{}, missing(path + ": core.user_results.result")
	}
	return userFrom(path+": core.user_results.result", author)
}

// unwrapTweet reads one tweet_results.result into a tweet node and the visibility the result type
// says the upstream is granting it.
//
// Three shapes are read here. The tweet itself; a TweetWithVisibilityResults wrapper, which means
// the post is limited (the wrapper is not one of the recorded fixture's shapes, and it is unwrapped
// rather than refused because the alternative to one extra branch is a timeline that silently lost
// a post); and a TweetTombstone, which is a post that used to exist and is returned as one with
// domain.Unavailable rather than as an error, because a timeline that dropped its tombstones
// would be a timeline the upstream never sent.
//
// A result that is none of those is an error naming the type it found: this build has no way to
// tell "not a post" from "a post I cannot read", and the second one is the dangerous case.
//
// path is where the result lives in the response, and it is what every error from here names.
// isNullJSON reports whether a raw value is absent or JSON's null. A nil RawMessage (the field was
// missing) and the four bytes `null` (the field was there and empty) are the same fact to a reader:
// there is no quoted post to show.
func isNullJSON(raw json.RawMessage) bool {
	return len(raw) == 0 || string(raw) == "null"
}

func unwrapTweet(path string, raw json.RawMessage) (tweet, domain.Visibility, error) {
	if len(raw) == 0 {
		return tweet{}, domain.Visible, missing(path)
	}
	var node tweet
	if err := json.Unmarshal(raw, &node); err != nil {
		return tweet{}, domain.Visible, unreadable(path, err)
	}

	switch node.Typename {
	case "Tweet":
		return node, domain.Visible, nil
	case "TweetWithVisibilityResults":
		if node.Tweet == nil {
			return tweet{}, domain.Visible, missing(path + ".tweet")
		}
		var inner tweet
		if err := json.Unmarshal(*node.Tweet, &inner); err != nil {
			return tweet{}, domain.Visible, unreadable(path+".tweet", err)
		}
		return inner, domain.Limited, nil
	case "TweetTombstone":
		return node, domain.Unavailable, nil
	default:
		return tweet{}, domain.Visible, &ParseError{
			Path:   path + ".__typename",
			Detail: "is " + node.Typename + ", which is not a tweet shape this parser knows",
		}
	}
}

// The response shapes below are the ones the recorded fixture has, field for field. Only the
// fields this package reads are declared: a response carries hundreds more, and a struct that
// spelled them all out would be a second copy of the schema to keep in step.
type timelineEnvelope struct {
	Data struct {
		User struct {
			Result struct {
				Timeline   *timelineNode `json:"timeline"`
				TimelineV2 *timelineNode `json:"timeline_v2"`
			} `json:"result"`
		} `json:"user"`
		// Home is the envelope the two home timelines answer under: HomeLatestTimeline (the
		// Following feed bare `x-terminal` opens on) and HomeTimeline (For-you). Measured
		// 2026-10-15: both answered 200 with their instructions at
		// data.home.home_timeline_urt.instructions under the variables this build sends
		// (count 20, empty cursor, empty seenTweetIds, includePromotedContent false), and the
		// path is the same for the two — one arm covers both.
		Home struct {
			HomeTimelineUrt *timelineNode `json:"home_timeline_urt"`
		} `json:"home"`
		BookmarkTimelineV2 *timelineNode `json:"bookmark_timeline_v2"`
	} `json:"data"`
}

// timelineNode is one timeline container: the instructions and the nesting the envelope put them
// under. Two nestings are read here, because the responses measured so far use both:
//
//   - timeline.instructions — the user timelines (UserTweets / Likes), the bookmarks list and the
//     search container all answer with their instructions behind a `timeline` object;
//   - instructions directly — the home timelines' container (data.home.home_timeline_urt) has no
//     `timeline` wrapper at all: measured 2026-10-15, its instructions are a sibling of its
//     `metadata`, and reading only the wrapped shape is what refused every home answer.
//
// Both are one type rather than two because nothing below this point can tell them apart: the
// entries and the cursors are the same grammar either way (TimelineAddEntries,
// TimelineTimelineItem/TimelineTweet, TimelineTimelineCursor, TimelineTimelineModule), and
// instructionList is the one place the nesting is unwrapped.
type timelineNode struct {
	Timeline struct {
		Instructions []instruction `json:"instructions"`
	} `json:"timeline"`
	// Instructions is the home container's own nesting: data.home.home_timeline_urt carries its
	// instructions as a direct sibling of its metadata, with no `timeline` object in between.
	// Measured 2026-10-15 on the live HomeLatestTimeline answer.
	Instructions []instruction `json:"instructions"`
}

// instructionList returns the container's instructions, whichever nesting the envelope used. The
// wrapped shape wins when it is present: it is what the three containers that have a `timeline`
// object answer with, and the home container never carries one.
func (n *timelineNode) instructionList() []instruction {
	if n.Timeline.Instructions != nil {
		return n.Timeline.Instructions
	}
	return n.Instructions
}

type instruction struct {
	Type    string      `json:"type"`
	Entries []entryNode `json:"entries"`
}

type entryNode struct {
	EntryID string    `json:"entryId"`
	Content entryBody `json:"content"`
}

type entryBody struct {
	Typename    string       `json:"__typename"`
	CursorType  string       `json:"cursorType"`
	Value       string       `json:"value"`
	ItemContent *itemContent `json:"itemContent"`
	Items       []moduleItem `json:"items"`
}

type moduleItem struct {
	EntryID string `json:"entryId"`
	Item    struct {
		ItemContent *itemContent `json:"itemContent"`
	} `json:"item"`
}

type itemContent struct {
	Typename string `json:"__typename"`
	// TweetDisplayType is the upstream's own name for the block an item belongs to: an item in a
	// conversation module that says SelfThread is the focal post continuing itself rather than a
	// stranger's reply. detail.go reads it and nothing else does.
	TweetDisplayType string `json:"tweetDisplayType"`
	// CursorType and Value are a module's own handle into more of its block: a conversation module
	// can carry a cursor item instead of a post, and which cursor types are handles is detail.go's
	// business rather than this struct's.
	CursorType   string `json:"cursorType"`
	Value        string `json:"value"`
	TweetResults struct {
		Result json.RawMessage `json:"result"`
	} `json:"tweet_results"`
	// Name and TrendURL are a TimelineTrend item's own two fields: the name Explore prints and the
	// link the web client follows. Nothing else in this parser reads them.
	Name string `json:"name"`
	// TrendURL is a shape whose spelling has moved: the live answer carries an object
	// (`{"url": "twitter://trending/<id>", "urlType": "DeepLink"}`) while an older one carried a
	// string. It stays raw here and the trends parser reads both, so neither shape is a decode
	// failure for a field only one reader uses.
	TrendURL json.RawMessage `json:"trend_url"`
	// UserResults is a TimelineUser item's account node. People search answers carry accounts
	// here; the same userNode reader that builds a post's author reads them, so the two cannot
	// drift. Empty on every TimelineTweet item.
	UserResults struct {
		Result *userNode `json:"result"`
	} `json:"user_results"`
}

type tweet struct {
	Typename string       `json:"__typename"`
	RestID   string       `json:"rest_id"`
	Legacy   *tweetLegacy `json:"legacy"`
	Core     struct {
		UserResults struct {
			Result *userNode `json:"result"`
		} `json:"user_results"`
	} `json:"core"`
	NoteTweet *struct {
		Text string `json:"text"`
	} `json:"note_tweet"`
	// Views carries the view count. It is declared as raw JSON because the count arrives quoted
	// as a decimal string in every measured response ("1075958"): read into a string it would
	// make an unquoted answer unreadable, and read into a number it would lose the measured one.
	Views *struct {
		Count json.RawMessage `json:"count"`
	} `json:"views"`
	QuotedStatusResult    *resultHolder `json:"quoted_status_result"`
	RetweetedStatusResult *resultHolder `json:"retweeted_status_result"`
	// Tweet is the inner tweet of a TweetWithVisibilityResults wrapper.
	Tweet *json.RawMessage `json:"tweet"`
}

// repost returns the tweet this one reposts, wherever the answer nested it.
//
// The two nestings have both been measured: the research-era shape puts retweeted_status_result
// at the tweet's root, and the response recorded from the live site on 2026-10-14 puts it under
// legacy (18 of one account's 20 entries). Reading only one of them renders a repost as a post
// whose text begins "RT @…" and whose author is the reposter — which looks like a post that
// exists and is not one.
func (t tweet) repost() *resultHolder {
	if t.Legacy != nil && t.Legacy.RetweetedStatusResult != nil {
		return t.Legacy.RetweetedStatusResult
	}
	return t.RetweetedStatusResult
}

type resultHolder struct {
	Result json.RawMessage `json:"result"`
}

type tweetLegacy struct {
	FullText      string `json:"full_text"`
	CreatedAt     string `json:"created_at"`
	IDStr         string `json:"id_str"`
	Lang          string `json:"lang"`
	ReplyCount    int    `json:"reply_count"`
	RetweetCount  int    `json:"retweet_count"`
	FavoriteCount int    `json:"favorite_count"`
	QuoteCount    int    `json:"quote_count"`
	BookmarkCount int    `json:"bookmark_count"`
	// ConversationIDStr is the conversation a post belongs to, and one of the two fields a
	// self-thread is told apart by (the other is in_reply_to_status_id_str).
	ConversationIDStr string `json:"conversation_id_str"`
	// InReplyToStatusIDStr is the post this one answers. It is absent from most posts, and an
	// absent one means "answers nothing" rather than "a field I could not find".
	InReplyToStatusIDStr string `json:"in_reply_to_status_id_str"`
	// Entities and ExtendedEntities both carry media, and which of the two is current has moved
	// before; mediaOf prefers extended_entities and falls back to entities. entities also carries
	// the three lists linksOf reads: a post's URLs, tags and mentions.
	Entities         *tweetEntities `json:"entities"`
	ExtendedEntities *mediaEntities `json:"extended_entities"`
	// RetweetedStatusResult is where a repost's original tweet lives in the current responses.
	RetweetedStatusResult *resultHolder `json:"retweeted_status_result"`
	// WithheldInCountries is the list of countries the upstream will not serve this post in. An
	// empty or absent list means the post is served everywhere; a non-empty one is the one
	// visibility state this build has no measured shape for, and detail.go is its only reader.
	WithheldInCountries []string `json:"withheld_in_countries"`
}

// mediaEntities is the part of extended_entities this package reads: that object carries media and
// nothing else.
type mediaEntities struct {
	Media []mediaEntity `json:"media"`
}

// tweetEntities is the part of legacy.entities this package reads: media, and the three kinds of
// thing a body names. The URL list is read for its expanded form — the text itself holds t.co short
// links, and a reader who follows one wants the destination the upstream already resolved, not a
// redirect hop through an ad domain.
type tweetEntities struct {
	Media        []mediaEntity   `json:"media"`
	URLs         []urlEntity     `json:"urls"`
	Hashtags     []hashtagEntity `json:"hashtags"`
	UserMentions []mentionEntity `json:"user_mentions"`
}

// urlEntity is one URL a body names.
type urlEntity struct {
	URL         string `json:"url"`
	ExpandedURL string `json:"expanded_url"`
}

// hashtagEntity is one tag a body names.
type hashtagEntity struct {
	Text string `json:"text"`
}

// mentionEntity is one account a body names.
type mentionEntity struct {
	ScreenName string `json:"screen_name"`
}

// mediaEntity is one attachment as the response carries it: the upstream's own type name, the
// URL it serves the attachment under (a video's poster frame), the alt text its poster wrote,
// and — for video and animated_gif — the variant list a player can take.
type mediaEntity struct {
	Type          string     `json:"type"`
	MediaURLHTTPS string     `json:"media_url_https"`
	ExtAltText    string     `json:"ext_alt_text"`
	VideoInfo     *videoInfo `json:"video_info"`
}

// videoInfo is the playable half of a video or animated_gif attachment.
type videoInfo struct {
	Variants []videoVariant `json:"variants"`
}

// videoVariant is one encoding of a video: an mp4 at a bitrate, or an HLS manifest.
type videoVariant struct {
	Bitrate     *int   `json:"bitrate"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"`
}
