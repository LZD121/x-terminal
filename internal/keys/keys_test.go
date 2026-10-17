package keys

import (
	"strings"
	"testing"
)

// TestTheTableIsTheUISpecsRows pins the size the three surfaces are built on: a row added to the
// table without being meant, or a row lost from it, is a key that either works undocumented or is
// documented without working. The count is Phase 2's thirteen rows, Phase 3's seven, Phase 4's one
// (`enter`), Phase 5's three (`/`, `s`, `a`), Phase 6's two (`b`, `l`), Phase 7's four
// (`i`, `y`, `o`, `T`).
func TestTheTableIsTheUISpecsRows(t *testing.T) {
	if got := len(Rows()); got != 38 {
		t.Fatalf("the keymap table holds %d rows, want the UI-SPEC's 38", got)
	}
}

// TestTheDisguisesKeysResolve is this phase's rows through the matcher, with the spellings the
// decoder reports for them: a letter is the letter the user typed, a digit is the digit, and
// `ctrl+p` is the modifier spelling the decoder produces.
func TestTheDisguisesKeysResolve(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want Action
	}{
		{"m", ActionCover},
		{"c", ActionNextCover},
		{"1", ActionCover1},
		{"2", ActionCover2},
		{"3", ActionCover3},
		{"4", ActionCover4},
		{"ctrl+p", ActionPanic},
	} {
		got, ok := ByKeystroke(tc.key)
		if !ok || got != tc.want {
			t.Errorf("the keystroke %q resolves to %q (found %v), want %q", tc.key, got, ok, tc.want)
		}
	}
}

// TestTheSearchRowsResolve is this phase's rows through the matcher, with the help texts the
// UI-SPEC's table fixes: `/` opens the posts prompt, `s` is the Latest/Top toggle and `a` opens
// the selected post's author, on every screen, and all three spellings are the table's own.
func TestTheSearchRowsResolve(t *testing.T) {
	for _, tc := range []struct {
		key     string
		want    Action
		help    string
		covered string
	}{
		{"/", ActionSearchPosts, "search posts", "search records"},
		{"u", ActionSearchPeople, "find people", ""},
		{"s", ActionSearchOrder, "search order: latest / top", "switch the search order"},
		{"a", ActionOpenAuthor, "open the selected post's author", "open the selected record's author"},
	} {
		got, ok := ByKeystroke(tc.key)
		if !ok || got != tc.want {
			t.Errorf("the keystroke %q resolves to %q (found %v), want %q", tc.key, got, ok, tc.want)
		}
		for _, screen := range []string{"home", "search", "thread", "profile"} {
			if _, ok := Lookup(tc.key, screen); !ok {
				t.Errorf("the keystroke %q does not resolve on screen %q: its row is applicable everywhere", tc.key, screen)
			}
		}
		row, found := Row{}, false
		for _, r := range Rows() {
			if r.Action == tc.want {
				row, found = r, true
			}
		}
		if !found {
			t.Errorf("no row carries %q", tc.want)
			continue
		}
		if row.Help != tc.help {
			t.Errorf("the %q row's help is %q, want %q", tc.want, row.Help, tc.help)
		}
		if row.HelpCovered != tc.covered {
			t.Errorf("the %q row's covered help is %q, want %q", tc.want, row.HelpCovered, tc.covered)
		}
	}
}

// TestEveryRowIsWellFormed is the table's shape: an action, at least one keystroke, a help string,
// a group from the UI-SPEC's four, and a covered help string that is present exactly when the
// covered-wording table re-words the row. An action with no keystroke is a binding nothing can
// reach, and a covered text that merely repeats the plain one is not a re-wording.
func TestEveryRowIsWellFormed(t *testing.T) {
	actions := map[Action]bool{}
	helps := map[string]Action{}
	covered := map[string]Action{}
	groups := map[string]bool{GroupNavigate: true, GroupFeed: true, GroupView: true, GroupApp: true, GroupLists: true, GroupMedia: true}

	for _, row := range Rows() {
		if actions[row.Action] {
			t.Errorf("the action %q is in the table twice", row.Action)
		}
		actions[row.Action] = true
		if len(row.Keys) == 0 {
			t.Errorf("the action %q has no keystroke", row.Action)
		}
		if row.Help == "" {
			t.Errorf("the action %q has no help text", row.Action)
		}
		if previous, seen := helps[row.Help]; seen {
			t.Errorf("the help %q is used by both %q and %q", row.Help, previous, row.Action)
		}
		helps[row.Help] = row.Action
		if row.HelpCovered != "" {
			if row.HelpCovered == row.Help {
				t.Errorf("the action %q carries the same words under a cover: %q is not a re-wording", row.Action, row.HelpCovered)
			}
			if previous, seen := covered[row.HelpCovered]; seen {
				t.Errorf("the covered help %q is used by both %q and %q", row.HelpCovered, previous, row.Action)
			}
			covered[row.HelpCovered] = row.Action
		}
		if !groups[row.Group] {
			t.Errorf("the action %q is in the group %q, which is not one of the UI-SPEC's four", row.Action, row.Group)
		}
		for _, key := range row.Keys {
			if key == "" {
				t.Errorf("the action %q holds an empty keystroke", row.Action)
			}
		}
		if len(row.Screens) == 0 {
			t.Errorf("the action %q names no screens", row.Action)
		}
		for _, screen := range row.Screens {
			switch screen {
			case "all", "home", "search", "thread", "profile":
			default:
				t.Errorf("the action %q names the screen %q, which is not one of the UI-SPEC's screens", row.Action, screen)
			}
		}
	}
	if len(actions) != len(Rows()) {
		t.Fatalf("the table holds %d actions for %d rows", len(actions), len(Rows()))
	}
}

// TestLookupFiltersByScreen is the screen field's contract: a keystroke whose row excludes the
// frame on top is not found, and the same keystroke is found on a screen the row names.
func TestLookupFiltersByScreen(t *testing.T) {
	if _, ok := Lookup("t", "thread"); ok {
		t.Errorf("t resolves on a thread frame: its row is home-only")
	}
	if action, ok := Lookup("t", "home"); !ok || action != ActionSwitchList {
		t.Errorf("t on home resolves to %q (found %v), want switch-list", action, ok)
	}
	if action, ok := Lookup("enter", "thread"); !ok || action != ActionOpen {
		t.Errorf("enter on thread resolves to %q (found %v), want open", action, ok)
	}
	if action, ok := Lookup("enter", "home"); !ok || action != ActionOpen {
		t.Errorf("enter on home resolves to %q (found %v), want open", action, ok)
	}
	if rows := ForScreen("thread"); len(rows) == 0 {
		t.Fatal("ForScreen(thread) returned no rows")
	} else {
		foundOpen := false
		for _, row := range rows {
			if row.Action == ActionSwitchList {
				t.Errorf("ForScreen(thread) listed switch-list")
			}
			if row.Action == ActionOpen {
				foundOpen = true
			}
		}
		if !foundOpen {
			t.Fatal("ForScreen(thread) did not list open")
		}
	}
}

// TestNoKeystrokeIsBoundTwice is what stops a row shadowing another: a key in two rows would make
// the matcher's answer depend on the table's order rather than on the row's action.
func TestNoKeystrokeIsBoundTwice(t *testing.T) {
	seen := map[string]Action{}
	for _, row := range Rows() {
		for _, key := range row.Keys {
			if previous, twice := seen[key]; twice {
				t.Errorf("the keystroke %q does %q and %q", key, previous, row.Action)
			}
			seen[key] = row.Action
		}
	}
}

// TestEveryKeyRoundTripsThroughTheMatcher is the property the three surfaces rest on: the matcher
// reads the same table the overlay and the subcommand do, so a keystroke the table lists does the
// action the table says, with no second table to fall out of step.
func TestEveryKeyRoundTripsThroughTheMatcher(t *testing.T) {
	for _, row := range Rows() {
		for _, key := range row.Keys {
			action, ok := ByKeystroke(key)
			if !ok {
				t.Errorf("the keystroke %q is in the table for %q and the matcher does not know it", key, row.Action)
				continue
			}
			if action != row.Action {
				t.Errorf("the keystroke %q does %q through the matcher and %q in the table", key, action, row.Action)
			}
		}
	}
}

// TestTheMatcherBindsNothingElse is the anti-vacuity arm for the round trip above: if ByKeystroke
// answered with an action for anything it was handed, every round trip would pass for the wrong
// reason.
func TestTheMatcherBindsNothingElse(t *testing.T) {
	for _, key := range []string{"z", "", "ctrl+z", "shift+g", "j down"} {
		if action, ok := ByKeystroke(key); ok {
			t.Errorf("the matcher answers %q for %q, which is not a row of the table", action, key)
		}
	}
}

// TestTheHelpTextsAreASCII is the copy contract's chrome rule, asserted where the chrome is
// written: the keymap's own words are ASCII, whatever script the posts arrive in. Both wordings are
// the keymap's words, so both are held to it.
func TestTheHelpTextsAreASCII(t *testing.T) {
	for _, row := range Rows() {
		for _, text := range []string{row.Help, row.HelpCovered} {
			for _, r := range text {
				if r > 127 {
					t.Errorf("the help text %q carries %q, which is not ASCII", text, r)
				}
			}
		}
	}
}

// TestARowsKeysAreJoinedTheOneWay pins the join both surfaces write: the help overlay's key column
// and the `keys` subcommand's first field come from this method, so the two cannot spell the same
// row differently.
func TestARowsKeysAreJoinedTheOneWay(t *testing.T) {
	for _, row := range Rows() {
		joined := row.KeysJoined()
		if want := strings.Join(row.Keys, ", "); joined != want {
			t.Errorf("the row %q joins its keys as %q, want %q", row.Action, joined, want)
		}
		if got := strings.Count(joined, ", "); got != len(row.Keys)-1 {
			t.Errorf("the row %q has %d separators for %d keystrokes", row.Action, got, len(row.Keys))
		}
	}
}

// coveredHelp is the UI-SPEC's covered-wording table, written out here so the test that reads the
// table has something independent to read it against: only these rows are re-worded, in these words.
var coveredHelp = map[Action]string{
	ActionNext:        "next record",
	ActionPrev:        "previous record",
	ActionFirst:       "first loaded record",
	ActionLast:        "last loaded record",
	ActionRefresh:     "poll for new records, keep the place",
	ActionOpen:        "open the selected record",
	ActionOpenAuthor:  "open the selected record's author",
	ActionSwitchList:  "switch streams",
	ActionSearchPosts: "search records",
	ActionSearchOrder: "switch the search order",
	ActionCover:       "switch display mode",
	ActionNextCover:   "next display style",
	ActionCover1:      "display style 1",
	ActionCover2:      "display style 2",
	ActionCover3:      "display style 3",
	ActionCover4:      "display style 4",
	ActionPanic:       "switch to the idle screen",
	ActionBookmarks:   "open the saved stream",
	ActionLikes:       "open the starred stream",
	ActionMedia:       "open selected record files",
	ActionCopyLink:    "copy the record reference",
	ActionOpenPost:    "open the record externally",
	ActionTranslate:   "show the selected record in the configured language",
	// The four account reads. The covered wording names no account, no list and no post: "record"
	// and "references" are the disguise's own nouns, and the plain wording is the one that says
	// what the keys actually do.
	ActionFollowing:         "records this one references",
	ActionFollowers:         "records referencing this one",
	ActionVerifiedFollowers: "verified records referencing this one",
	ActionProfileMedia:      "files this record attached",
	ActionLinks:             "what this record references",
	ActionExplore:           "the trending list",
	ActionSwitchAccount:     "read as the next stored identity",
}

// TestCoveredHelpIsPresentExactlyWhereTheUISpecListsIt is the covered-wording table both ways: every
// row it names carries its words, every row it does not name carries none, and a row whose covered
// wording is absent is asked for the plain one even by a covered surface. The last arm is what keeps
// the `keys` subcommand's output — which passes false and prints the plain wording, because the CLI
// runs outside any cover — unchanged by this phase.
func TestCoveredHelpIsPresentExactlyWhereTheUISpecListsIt(t *testing.T) {
	seen := map[Action]bool{}
	for _, row := range Rows() {
		if want, ok := coveredHelp[row.Action]; ok {
			seen[row.Action] = true
			if row.HelpCovered != want {
				t.Errorf("the row %q is re-worded as %q under a cover, want %q", row.Action, row.HelpCovered, want)
			}
			if got := row.HelpText(true); got != want {
				t.Errorf("the row %q renders %q under a cover, want %q", row.Action, got, want)
			}
			if got := row.HelpText(false); got != row.Help {
				t.Errorf("the row %q renders %q outside a cover, want the plain %q", row.Action, got, row.Help)
			}
			continue
		}
		if row.HelpCovered != "" {
			t.Errorf("the row %q carries the covered help %q, and the UI-SPEC re-words nothing there", row.Action, row.HelpCovered)
		}
		if got := row.HelpText(true); got != row.Help {
			t.Errorf("the row %q renders %q under a cover, want the plain %q: it has no covered wording", row.Action, got, row.Help)
		}
	}
	for action := range coveredHelp {
		if !seen[action] {
			t.Fatalf("the UI-SPEC re-words %q, which is not a row of the table", action)
		}
	}
}

// leaks is the words a covered help string may not hold that the string does hold: the product's
// own vocabulary, the two feeds, the host, the words the disguise's action names use, and the
// account nouns a cover exists to keep off the screen. It is a function so that the arm below can be
// held to the same rule the table is.
func leaks(text string) []string {
	var found []string
	for _, word := range []string{
		"x.com", "twitter", "x-terminal",
		"following", "for you",
		"post", "timeline", "feed",
		"cover", "disguise", "panic",
		"handle", "account",
	} {
		if strings.Contains(strings.ToLower(text), word) {
			found = append(found, word)
		}
	}
	return found
}

// TestNoCoveredHelpLeaks is the rule the covered wording exists for. A key list is on screen under a
// cover, so the covered words are the one place the disguise could be undone by its own help text.
func TestNoCoveredHelpLeaks(t *testing.T) {
	for _, row := range Rows() {
		if row.HelpCovered == "" {
			continue
		}
		if found := leaks(row.HelpCovered); len(found) != 0 {
			t.Errorf("the covered help %q of %q holds %q", row.HelpCovered, row.Action, found)
		}
	}

	// Purpose words that name what the three media rows would reveal under a cover.
	for _, row := range Rows() {
		switch row.Action {
		case ActionMedia, ActionCopyLink, ActionOpenPost:
			lower := strings.ToLower(row.HelpCovered)
			for _, word := range []string{"browser", "attachment", "link", "media", "post"} {
				if strings.Contains(lower, word) {
					t.Errorf("the media covered help %q of %q holds purpose word %q", row.HelpCovered, row.Action, word)
				}
			}
		}
	}

	// The arm: the rule is over words a help string can really hold.
	if found := leaks("post unavailable - press cover to hide the feed"); len(found) == 0 {
		t.Fatalf("the leak check found nothing in a string holding %q", "post, cover, feed")
	}
}

// TestNoActionNameIsRendered pins the other half: the action names are code. The check is over the
// names that cannot be mistaken for prose — the hyphenated and numbered ones — because the UI-SPEC's
// own plain wording deliberately shares the words `next` and `cover` with the actions they name, and
// asserting over those would be asserting against the specification.
func TestNoActionNameIsRendered(t *testing.T) {
	var identifiers []string
	for _, row := range Rows() {
		if name := string(row.Action); strings.ContainsAny(name, "-0123456789") {
			identifiers = append(identifiers, name)
		}
	}
	if len(identifiers) < 5 {
		t.Fatalf("the table names %d identifier-shaped actions, too few for the check below", len(identifiers))
	}

	for _, row := range Rows() {
		for _, text := range []string{row.Help, row.HelpCovered} {
			for _, name := range identifiers {
				if strings.Contains(text, name) {
					t.Errorf("the help text %q carries the action name %q, which is code", text, name)
				}
			}
		}
	}
}

// TestTheSavedListRowsResolve is Phase 6's two rows: b opens bookmarks, l opens
// likes, with the help literals the UI-SPEC fixes, and neither key is bound twice.
func TestTheSavedListRowsResolve(t *testing.T) {
	want := map[string]struct {
		action Action
		help   string
	}{
		"b": {ActionBookmarks, "open bookmarks"},
		"l": {ActionLikes, "open likes"},
	}
	for key, row := range want {
		got, ok := Lookup(key, "home")
		if !ok || got != row.action {
			t.Errorf("Lookup(%q) = %q, %v; want %q", key, got, ok, row.action)
		}
	}
	for _, r := range Rows() {
		switch r.Action {
		case ActionBookmarks, ActionLikes:
			if r.Help != want[r.Keys[0]].help {
				t.Errorf("%s help = %q, want %q", r.Action, r.Help, want[r.Keys[0]].help)
			}
			if r.Group != GroupLists {
				t.Errorf("%s group = %q, want lists", r.Action, r.Group)
			}
		}
	}
}

func TestTheMediaRowResolves(t *testing.T) {
	want := map[string]struct {
		action  Action
		help    string
		covered string
	}{
		"i": {ActionMedia, "open the post's media in an external viewer", "open selected record files"},
		"y": {ActionCopyLink, "copy the post's link", "copy the record reference"},
		"o": {ActionOpenPost, "open the post in the browser", "open the record externally"},
	}
	for key, tc := range want {
		got, ok := Lookup(key, "home")
		if !ok || got != tc.action {
			t.Errorf("Lookup(%s) = %q, %v; want %q", key, got, ok, tc.action)
		}
	}
	seen := map[Action]bool{}
	for _, r := range Rows() {
		switch r.Action {
		case ActionMedia, ActionCopyLink, ActionOpenPost:
			seen[r.Action] = true
			tc := want[r.Keys[0]]
			if r.Help != tc.help {
				t.Errorf("%s help = %q, want %q", r.Action, r.Help, tc.help)
			}
			if r.HelpCovered != tc.covered {
				t.Errorf("%s HelpCovered = %q, want %q", r.Action, r.HelpCovered, tc.covered)
			}
			if r.Group != GroupMedia {
				t.Errorf("%s group = %q, want media", r.Action, r.Group)
			}
			if len(r.Keys) != 1 {
				t.Errorf("%s keys = %v, want one key", r.Action, r.Keys)
			}
		}
	}
	for _, a := range []Action{ActionMedia, ActionCopyLink, ActionOpenPost} {
		if !seen[a] {
			t.Errorf("no %q row in the table", a)
		}
	}
}

// TestTheTranslateRowResolves is this feature's row: `T` is the shifted letter the decoder reports
// as `T` (String) and `shift+t` (Keystroke), the plain wording names a post, and the covered wording
// is the same action in the words a work tool uses.
func TestTheTranslateRowResolves(t *testing.T) {
	const (
		plain   = "translate the selected post"
		covered = "show the selected record in the configured language"
	)
	got, ok := Lookup("T", "home")
	if !ok || got != ActionTranslate {
		t.Errorf("Lookup(T) = %q, %v; want %q", got, ok, ActionTranslate)
	}
	// The shifted letter is a distinct binding: the unshifted one switches the list, and this is
	// where the two are proved to be two.
	if other, ok := Lookup("t", "home"); !ok || other != ActionSwitchList {
		t.Errorf("Lookup(t) = %q, %v; want %q — `T` must not have taken the unshifted key", other, ok, ActionSwitchList)
	}
	seen := false
	for _, r := range Rows() {
		if r.Action != ActionTranslate {
			continue
		}
		seen = true
		if len(r.Keys) != 1 || r.Keys[0] != "T" {
			t.Errorf("the translate row's keys are %v, want exactly [T]", r.Keys)
		}
		if r.Help != plain {
			t.Errorf("the translate row's help is %q, want %q", r.Help, plain)
		}
		if r.HelpCovered != covered {
			t.Errorf("the translate row's covered help is %q, want %q", r.HelpCovered, covered)
		}
		if r.Group != GroupView {
			t.Errorf("the translate row's group is %q, want view", r.Group)
		}
	}
	if !seen {
		t.Fatal("no translate row in the table")
	}
}
