// Package domain is the vocabulary the rest of the program speaks: a user, a post, a page of
// posts. It is a leaf — nothing in it imports another package of this module — so that the
// parsers, the screens and the store can agree on these types without agreeing on anything
// else, and so that a shape change in the response can be absorbed in exactly one place
// (internal/xsource) rather than leaking into every caller.
//
// The fields are what the live responses carry, not what the research-era schema said: the
// upstream answers with `core` / `avatar` / `banner` and does not carry `legacy.followers_count`
// at all (measured 2026-10-14, .planning/research/UPSTREAM-PROBE.md §3). Anything a screen
// needs that is not here is missing on purpose, and the fix is to add it with the response
// that carries it in hand.
//
// This package is the content half of the phase's Content / Presentation Split: it owns what
// is true about a post, and no value in it is a string a renderer prints. The words for a
// marker, a stamp, a state or a list name live in internal/render, and a failure's own
// sentence comes from the classifier (`xweb.Outcome.Remedy`) or from a netcheck result.
package domain

import (
	"strconv"
	"time"
)

// User is an account as the upstream describes it.
type User struct {
	// ID is the numeric account id the upstream calls rest_id.
	ID string
	// ScreenName is the handle without the @.
	ScreenName string
	// Name is the display name, which is what a post's byline shows above the handle.
	Name string
	// AvatarURL and BannerURL are the profile images' URLs. They are carried from the start
	// because the parser reads them anyway, and re-reading the same response later to find
	// them again would be a second parse of a shape this package has already been taught.
	AvatarURL string
	BannerURL string
	// HasGraduatedAccess is one of the flags the current schema carries. Nothing in this
	// phase renders it; it is recorded rather than dropped so that a later screen does not
	// have to guess whether it was read.
	HasGraduatedAccess bool
}

// PostID is the identity of a post: the key that de-duplication, selection and the refresh
// diff are all built on. It is a string because the upstream's id is a decimal number wider
// than any int this program should round-trip.
type PostID string

// Visibility is what the upstream will let a reader see of a post. The words a user reads for
// these states are presentation; the four states are content, and each one means something
// different — a limited post still has its body, an unavailable one is a tombstone.
type Visibility uint8

const (
	// Visible is a post served in full, and it is the zero value: a post whose visibility the
	// parser never set is a post it read all of.
	Visible Visibility = iota
	// Limited is a post behind the TweetWithVisibilityResults wrapper: the body is there, and
	// the upstream is saying it is held back.
	Limited
	// Unavailable is a tombstone: a post that used to exist, with no author, no body and no
	// date. It is a post, not an error, and dropping it would be editing the timeline.
	Unavailable
	// Withheld is a post the upstream removed for legal reasons in the reader's country. No
	// recorded response carries one yet; the state is here because a parser that mapped it
	// onto Visible would be lying about what the answer said.
	Withheld
)

// String names the state in one lowercase word, for logs and for tests.
func (v Visibility) String() string {
	switch v {
	case Visible:
		return "visible"
	case Limited:
		return "limited"
	case Unavailable:
		return "unavailable"
	case Withheld:
		return "withheld"
	}
	return "visibility " + strconv.Itoa(int(v))
}

// MediaKind is what kind of attachment a MediaRef points at. The noun a renderer prints for it
// is presentation; the kind is content.
type MediaKind uint8

const (
	// Image is a still picture, and it is the zero value: a photo is the shape this program
	// meets most.
	Image MediaKind = iota
	// Video is a video attachment. The URL is its poster frame, not its stream: the response
	// carries no media bytes and this program never fetches them.
	Video
	// GIF is an animated_gif: a video the upstream loops without sound.
	GIF
)

// String names the kind in one lowercase word, for logs and for tests.
func (k MediaKind) String() string {
	switch k {
	case Image:
		return "image"
	case Video:
		return "video"
	case GIF:
		return "gif"
	}
	return "media kind " + strconv.Itoa(int(k))
}

// MediaRef is one attachment: what it is, where it is, and the alt text its poster wrote for
// it. URL is the media's own URL as the payload carried it (for a video, the poster frame) and
// is what a printer can always show; Playable, when non-empty, is what a player is handed. Nothing
// in domain chooses a variant: the choice is the decoder's, and this value carries the answer.
type MediaRef struct {
	Kind     MediaKind
	URL      string
	AltText  string
	Playable string
}

// Counts are a post's engagement numbers as the response reports them. Zero means the response
// carried no such count; the list grammar this phase ships prints none of them.
type Counts struct {
	Replies   int
	Reposts   int
	Likes     int
	Quotes    int
	Bookmarks int
	Views     int
}

// LinkKind is what one thing a post's body names is: a URL, a hashtag, or an account.
//
// The three are one type because the reader's act is one — follow the thing under the cursor —
// and three types would be three menus. They are three kinds because following each one lands in a
// different place: a URL leaves the program, a hashtag is a search, and a mention is a profile.
type LinkKind uint8

const (
	// LinkURL is an absolute http(s) address, the expanded form when the upstream carried one.
	LinkURL LinkKind = iota
	// LinkHashtag is a tag, stored without its leading '#'.
	LinkHashtag
	// LinkMention is an account's handle, stored without its leading '@'.
	LinkMention
)

// Link is one thing a post's body names, as the upstream's entities list it. It is a value rather
// than a string because what the reader can do with it depends on which of the three it is, and a
// string would make every caller re-parse the difference.
type Link struct {
	Kind   LinkKind
	Target string
}

// Post is one post as a timeline carries it.
type Post struct {
	// ID is the post's own id (the upstream's rest_id).
	ID PostID
	// Author is who wrote it. For a repost this is the original author, and RepostedBy says
	// who reposted it — a repost is the same post, not a post by the reposter.
	Author User
	// Text is the post's text, with the long-form body preferred over the truncated one when
	// the response carries both.
	Text string
	// CreatedAt is when the post was written, in the upstream's own clock (UTC).
	CreatedAt time.Time
	// RepostedBy is the account that reposted this post, nil when nobody did.
	RepostedBy *User
	// Lang is the language the upstream attributed to the post ("en", "ja", …). Nothing in
	// this phase reads it; it is carried because the response carries it, and so that a later
	// screen that wants to filter or label by language does not have to re-parse a body.
	Lang string
	// ConversationID is the conversation the post belongs to. Two posts with the same id are a
	// thread; the words for that are presentation, the relationship is content.
	ConversationID string
	// ReplyTo is the post this one answers, empty when it answers nothing. It equals the
	// conversation id for a self-thread's later posts.
	ReplyTo PostID
	// Quote is the post this one quotes, nil when it quotes none. The quoted body is nested in
	// the same response (bounded in xsource), so a renderer can show it without a second
	// request.
	Quote *Post
	// Links are the URLs, hashtags and accounts the body names, in the upstream's entities order:
	// URLs first, then tags, then mentions, each in the answer's own order. They are deduplicated by
	// kind and target, because a body that repeats a link names it once.
	Links []Link
	// Media are the post's attachments, in the order the upstream listed them. Refs only: no
	// bytes and no URL the renderer may print (Phase 7 owns opening media).
	//
	// It is nil when the post carries none, which is how a renderer tells "no attachment" from
	// "attachments that did not parse".
	Media []MediaRef
	// Visibility is what the upstream will let a reader see of this post.
	Visibility Visibility
	// Counts are the post's engagement numbers. They are parsed and retained, and the list
	// grammar this phase ships must not print them (REND-01).
	Counts Counts
}
