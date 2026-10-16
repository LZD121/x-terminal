// Package keys is the program's one keymap table: what a keystroke means, what the key is called
// and what it does, in one place.
//
// Three surfaces read it and none of them may keep a second copy: input handling (internal/tui),
// the `?` help overlay (which internal/tui maps into render.HelpRow) and the `keys` subcommand's
// output. Adding a key is one row here; a row here works and is documented the moment it is added,
// and a key that is not a row here does nothing and is documented nowhere.
//
// Nothing in this package does anything: no tea, no I/O, no context, no clock. It is data and a
// lookup, and every string in it is the UI-SPEC's own (`## Keymap Contract`), which is why this
// package composes no sentence: the help text is rendered verbatim and the key list is joined by
// the one documented rule below.
package keys

import "strings"

// Action is what a key does. It is a typed string rather than a number so that a table row, a test
// failure and a log line all read as the thing they name.
type Action string

// The phase's actions: one per row of the UI-SPEC's keymap table. Pagination deliberately has no
// action of its own — reaching the last loaded post is the trigger — and media or a link are later
// phases' rows, so no key is documented here that does not work. `open` is the conversation's one
// new row: it opens the selected post or the selected quoted post as its own conversation.
//
// The disguise's actions are the ones Phase 3 added, in the same table: `cover` wears the feed as
// work-software output and takes it off again, `next-cover` walks the four styles, `cover-1` to
// `cover-4` name one of them directly, and `panic` replaces the whole pane. Their names are code
// and no surface renders them — the `keys` subcommand prints the plain help wording, because the
// CLI runs outside any cover.
const (
	ActionNext       Action = "next"
	ActionPrev       Action = "prev"
	ActionHalfDown   Action = "half-down"
	ActionHalfUp     Action = "half-up"
	ActionPageDown   Action = "page-down"
	ActionPageUp     Action = "page-up"
	ActionFirst      Action = "first"
	ActionLast       Action = "last"
	ActionRefresh    Action = "refresh"
	ActionSwitchList Action = "switch-list"
	ActionOpen       Action = "open" // ActionOpen Action = "open"
	ActionHelp       Action = "help"
	ActionBack       Action = "back"
	// ActionSearchPosts is `/`: open the query prompt for posts. ActionSearchOrder is `s`: the
	// Latest/Top toggle on a results frame, and one honest notice where there is no order to
	// switch. ActionOpenAuthor is `a`: the selected post's author as a profile. ActionSearchPeople
	// is `u`: open the query prompt for accounts.
	ActionSearchPosts  Action = "search-posts"
	ActionSearchOrder  Action = "search-order"
	ActionOpenAuthor   Action = "open-author"
	ActionSearchPeople Action = "search-people"

	ActionQuit Action = "quit"

	ActionCover     Action = "cover"
	ActionNextCover Action = "next-cover"
	ActionCover1    Action = "cover-1"
	ActionCover2    Action = "cover-2"
	ActionCover3    Action = "cover-3"
	ActionCover4    Action = "cover-4"
	ActionPanic     Action = "panic"

	ActionBookmarks Action = "bookmarks"
	ActionLikes     Action = "likes"

	// ActionMedia opens every attachment of the selected post as a set. Its key is `i`
	// rather than `m` because Phase 3's cover binds `m` in every state, and it is not a
	// digit row because Phase 3's T5 binds `1`–`4` to the cover styles from normal as
	// well as immersive — Phase 3 lands first and a collision is resolved in the cover's favour.
	ActionMedia Action = "media"
	// ActionCopyLink copies a permalink to the selected post (OSC 52, then wl-copy).
	ActionCopyLink Action = "copy-link"
	// ActionOpenPost opens the selected post's permalink in an external opener.
	ActionOpenPost Action = "open-post"
	// ActionTranslate is `T` (shift-t): show the selected post's body in the configured language,
	// through x.com's own translation service.
	//
	// The key is a shifted letter because every lower-case letter this list could reach for is
	// spoken for: `t` switches the list, `r` refreshes, `s`/`a`/`u` are the search rows, `i`/`y`/`o`
	// are the media rows and `m`/`c`/`1`-`4`/`ctrl+p` are the disguise's. A shifted letter is a
	// distinct binding in this table — the shell looks a keystroke up twice, once as the decoder's
	// Keystroke() and once as the character the user typed — which the `G` row already proves end to
	// end.
	ActionTranslate Action = "translate"
)

// The four groups the UI-SPEC's table sorts its rows into. Nothing prints them in this phase — the
// help overlay is one flat list in table order, and a group heading would be a heading — but they
// are part of the row so that a later surface (a grouped overlay, a completion script) does not
// have to re-derive them from the action names.
const (
	GroupNavigate = "navigate"
	GroupFeed     = "feed"
	GroupView     = "view"
	GroupApp      = "app"
	GroupLists    = "lists"
	GroupMedia    = "media"
)

// KeyJoin is how a row's keystrokes are written when more than one key does the same thing. The
// help overlay's key column and the `keys` subcommand's first field both write the row this way,
// so the same binding reads the same on both surfaces and neither writes the join by hand.
const KeyJoin = ", "

// Row is one binding: what it does, every keystroke that does it, the help text rendered verbatim
// by the overlay, which part of the surface it belongs to, and which screens the row applies to.
//
// Keys holds the keystroke spellings a Bubble Tea v2 KeyPressMsg reports for the key, and they are
// also what the overlay prints: `down`, `pgdown`, `up` and `pgup` are what the decoder calls the
// arrow and page keys, `ctrl+d` and `esc` are the modifier and escape spellings it produces, and a
// letter is the letter the user typed. `G` is the shift spelling: the decoder lower-cases an
// upper-case byte and marks the event shifted, so the shell matches the row through
// KeyPressMsg.String() — which reports the shifted character itself — as well as through
// KeyPressMsg.Keystroke().
//
// Screens []string is which frames the row applies to: `all`, or a list drawn from `{home, search, thread,
// profile}`. A row that does not apply to the top frame is not dispatched (a silent no-op) and is
// not listed by the `?` overlay. Applicability is not printed by the `keys` subcommand, because a
// listing has no frame context.
type Row struct {
	Action Action
	Keys   []string
	Help   string
	// HelpCovered is the same row's help text under a cover, and empty means "the same words". Only
	// the rows whose plain wording names the product, a feed or a post are re-worded — the wording is
	// a field of the row rather than a second table, so adding a key is still one row — and a row
	// whose covered text is empty has exactly one string to keep in step with itself.
	HelpCovered string
	Group       string
	Screens     []string // Screens []string
}

// KeysJoined is the row's keystrokes as one string, joined the way both surfaces write them.
func (r Row) KeysJoined() string { return strings.Join(r.Keys, KeyJoin) }

// HelpText is the row's help text as a rendering surface prints it: the covered wording when the
// surface is drawn under a cover and this row has a covered wording, and the plain wording
// everywhere else. A surface asks this rather than choosing between the two fields itself, so the
// choice exists in one place — and the `keys` subcommand, which runs outside any cover, passes
// false and prints the plain wording.
func (r Row) HelpText(covered bool) string {
	if covered && r.HelpCovered != "" {
		return r.HelpCovered
	}
	return r.Help
}

// rows is the table, in the UI-SPEC's order: Phase 2's rows verbatim, then this phase's. It is
// package data rather than a literal rebuilt by Rows() so that the matcher, the overlay and the
// subcommand walk one slice; nothing here is computed, ordered or derived.
var rows = []Row{
	{Action: ActionNext, Keys: []string{"j", "down"}, Help: "next post", HelpCovered: "next record", Group: GroupNavigate, Screens: []string{"all"}},
	{Action: ActionPrev, Keys: []string{"k", "up"}, Help: "previous post", HelpCovered: "previous record", Group: GroupNavigate, Screens: []string{"all"}},
	{Action: ActionHalfDown, Keys: []string{"ctrl+d"}, Help: "half a pane down", Group: GroupNavigate, Screens: []string{"all"}},
	{Action: ActionHalfUp, Keys: []string{"ctrl+u"}, Help: "half a pane up", Group: GroupNavigate, Screens: []string{"all"}},
	{Action: ActionPageDown, Keys: []string{"pgdown"}, Help: "one pane down", Group: GroupNavigate, Screens: []string{"all"}},
	{Action: ActionPageUp, Keys: []string{"pgup"}, Help: "one pane up", Group: GroupNavigate, Screens: []string{"all"}},
	{Action: ActionFirst, Keys: []string{"g", "home"}, Help: "first loaded post", HelpCovered: "first loaded record", Group: GroupNavigate, Screens: []string{"all"}},
	{Action: ActionLast, Keys: []string{"G", "end"}, Help: "last loaded post", HelpCovered: "last loaded record", Group: GroupNavigate, Screens: []string{"all"}},
	{Action: ActionOpen, Keys: []string{"enter"}, Help: "open the selected post or account", HelpCovered: "open the selected record", Group: GroupNavigate, Screens: []string{"all"}},
	{Action: ActionOpenAuthor, Keys: []string{"a"}, Help: "open the selected post's author", HelpCovered: "open the selected record's author", Group: GroupNavigate, Screens: []string{"all"}},
	{Action: ActionRefresh, Keys: []string{"r"}, Help: "fetch newer posts, keep the place", HelpCovered: "poll for new records, keep the place", Group: GroupFeed, Screens: []string{"all"}},
	{Action: ActionSwitchList, Keys: []string{"t"}, Help: "switch what this list shows", HelpCovered: "switch streams", Group: GroupFeed, Screens: []string{"home"}},
	{Action: ActionSearchPosts, Keys: []string{"/"}, Help: "search posts", HelpCovered: "search records", Group: GroupFeed, Screens: []string{"all"}},
	{Action: ActionSearchPeople, Keys: []string{"u"}, Help: "find people", Group: GroupFeed, Screens: []string{"all"}},
	{Action: ActionSearchOrder, Keys: []string{"s"}, Help: "search order: latest / top", HelpCovered: "switch the search order", Group: GroupFeed, Screens: []string{"all"}},
	{Action: ActionHelp, Keys: []string{"?"}, Help: "show or hide this list", Group: GroupView, Screens: []string{"all"}},
	{Action: ActionBack, Keys: []string{"q", "esc"}, Help: "back, or quit at the start", Group: GroupApp, Screens: []string{"all"}},
	{Action: ActionQuit, Keys: []string{"ctrl+c"}, Help: "quit now", Group: GroupApp, Screens: []string{"all"}},

	// This phase's rows, in the UI-SPEC's table order. The covered wording is the same action in the
	// words a work tool uses, and it names no product, no feed and no handle: the help overlay is a
	// key list, and a key list can leak.
	{Action: ActionCover, Keys: []string{"m"}, Help: "wear the feed as work-software output, or take it off", HelpCovered: "switch display mode", Group: GroupView, Screens: []string{"all"}},
	{Action: ActionNextCover, Keys: []string{"c"}, Help: "next cover style", HelpCovered: "next display style", Group: GroupView, Screens: []string{"all"}},
	{Action: ActionCover1, Keys: []string{"1"}, Help: "cover style 1", HelpCovered: "display style 1", Group: GroupView, Screens: []string{"all"}},
	{Action: ActionCover2, Keys: []string{"2"}, Help: "cover style 2", HelpCovered: "display style 2", Group: GroupView, Screens: []string{"all"}},
	{Action: ActionCover3, Keys: []string{"3"}, Help: "cover style 3", HelpCovered: "display style 3", Group: GroupView, Screens: []string{"all"}},
	{Action: ActionCover4, Keys: []string{"4"}, Help: "cover style 4", HelpCovered: "display style 4", Group: GroupView, Screens: []string{"all"}},
	{Action: ActionPanic, Keys: []string{"ctrl+p"}, Help: "replace the screen now, and again to come back", HelpCovered: "switch to the idle screen", Group: GroupView, Screens: []string{"all"}},

	// Phase 6: the two saved lists. Two keys, not a toggle — each destination has its own literal
	// and the likes list is directly reachable even when the operation is missing from the build.
	{Action: ActionBookmarks, Keys: []string{"b"}, Help: "open bookmarks", HelpCovered: "open the saved stream", Group: GroupLists, Screens: []string{"all"}},
	{Action: ActionLikes, Keys: []string{"l"}, Help: "open likes", HelpCovered: "open the starred stream", Group: GroupLists, Screens: []string{"all"}},

	// Phase 7: open every attachment of the selected post, copy its permalink, or open
	// that permalink externally. Key `i` (not Phase 3's `m` cover) and not `1`–`9`
	// (Phase 3's cover-style rows win those collisions). The covered wording names no
	// purpose the action would reveal — browser, attachment, link, media, post.
	{Action: ActionMedia, Keys: []string{"i"}, Help: "open the post's media in an external viewer", HelpCovered: "open selected record files", Group: GroupMedia, Screens: []string{"all"}},
	{Action: ActionCopyLink, Keys: []string{"y"}, Help: "copy the post's link", HelpCovered: "copy the record reference", Group: GroupMedia, Screens: []string{"all"}},
	{Action: ActionOpenPost, Keys: []string{"o"}, Help: "open the post in the browser", HelpCovered: "open the record externally", Group: GroupMedia, Screens: []string{"all"}},

	// Phase 7's last row: show the selected post through x.com's own translator. `T` is the shifted
	// letter (see ActionTranslate above), and the row carries a covered wording because the plain one
	// names a post. The action itself reveals nothing a covered pane does not already carry: the body
	// it shows is the same post's, in another language, and nothing is handed to any other program.
	{Action: ActionTranslate, Keys: []string{"T"}, Help: "translate the selected post", HelpCovered: "show the selected record in the configured language", Group: GroupView, Screens: []string{"all"}},
}

// Rows returns the whole keymap, in table order. The slice is the table's own: a caller reads it
// and does not modify it.
func Rows() []Row { return rows }

// Lookup returns the action a keystroke does on a given screen. It is a linear scan of the same
// table, which is what keeps the lookup and the table from drifting apart: a spelling the table
// does not hold is a keystroke this build does not bind, and a row whose Screens exclude the frame
// on top is a silent no-op — false is the whole of what either means.
//
// screen is one of `{home, search, thread, profile}` or empty (treated as home for callers that
// have not yet named a frame). A row listing `all` applies everywhere.
func Lookup(k, screen string) (Action, bool) {
	if screen == "" {
		screen = "home"
	}
	for _, row := range rows {
		if !rowApplies(row, screen) {
			continue
		}
		for _, key := range row.Keys {
			if key == k {
				return row.Action, true
			}
		}
	}
	return "", false
}

// ByKeystroke returns the action a keystroke does, without a screen filter. It is the keymap's own
// tests' matcher and has no production caller: input handling goes through Lookup, the help overlay
// through ForScreen and the `keys` subcommand through Rows(). The two tests that use it ask the
// question without a frame — every keystroke the table lists resolves to the row's action (the
// round trip), and a keystroke the table does not list resolves to nothing (the anti-vacuity arm).
func ByKeystroke(k string) (Action, bool) {
	for _, row := range rows {
		for _, key := range row.Keys {
			if key == k {
				return row.Action, true
			}
		}
	}
	return "", false
}

// ForScreen returns the rows that apply to a screen, in table order: what the help overlay lists.
func ForScreen(screen string) []Row {
	if screen == "" {
		screen = "home"
	}
	out := make([]Row, 0, len(rows))
	for _, row := range rows {
		if rowApplies(row, screen) {
			out = append(out, row)
		}
	}
	return out
}

func rowApplies(row Row, screen string) bool {
	for _, s := range row.Screens {
		if s == "all" || s == screen {
			return true
		}
	}
	return false
}
