package render

import (
	"strconv"
	"strings"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// NoticeKind is which notice the status line carries: the program's own announcement about the
// list, never a result the user asked for twice.
type NoticeKind uint8

const (
	// NoticeNone is no notice: the field is omitted from the line.
	NoticeNone NoticeKind = iota
	// NoticeNewPosts is a refresh that found posts the list did not have, counted in NoticeCount.
	NoticeNewPosts
	// NoticeNoNewPosts is a refresh that found nothing new: said out loud, because silence after a
	// deliberate refresh is indistinguishable from a refresh that did not happen.
	NoticeNoNewPosts
	// NoticeLoadingOlder is a page request in flight.
	NoticeLoadingOlder
	// NoticeStalled is a cursor that answered with itself: the walk ends here, and it is reported
	// rather than retried.
	NoticeStalled
	// NoticeLoadingReplies is a reply page of a conversation in flight: the thread frame's own
	// `loading older posts`, in a conversation's vocabulary.
	NoticeLoadingReplies
	// NoticeNewReplies is a conversation re-read that found replies the frame did not hold,
	// counted in NoticeCount.
	NoticeNewReplies
	// NoticeNoNewReplies is a conversation re-read that found nothing new: said out loud, because
	// silence after a deliberate refresh is indistinguishable from a refresh that did not happen.
	NoticeNoNewReplies
	// NoticeNothingToOpen is the one honest line an applicable key reports when its action has
	// nothing to act on. It is a notice and not a pane statement: the pane says what the
	// conversation is, the line says what the key did.
	NoticeNothingToOpen
	// NoticeNoResults is a search that read successfully and carried no rows: a successful outcome
	// in the notice field, never a failure and never a pane heading.
	NoticeNoResults
	// NoticeNothingToSwitch is `s` where there is no order to switch: the one honest line an
	// applicable key reports, in the same shape as NoticeNothingToOpen.
	NoticeNothingToSwitch
	// NoticeLoadingPosts is a profile whose header arrived and whose first posts page is out.
	NoticeLoadingPosts
	// NoticeNoPosts is a profile whose posts read succeeded and carried none.
	NoticeNoPosts
	// NoticeNoAuthor is `a` where the selected row has no author.
	NoticeNoAuthor
	// NoticeSyncing is a saved-frame sync walk in progress, counted in NoticeCount.
	NoticeSyncing
	// NoticeNoLongerSaved is a full re-list that marked rows absent.
	NoticeNoLongerSaved
	// NoticeMirrorComplete is a sync that reached the end of the list.
	NoticeMirrorComplete
	// NoticeSyncStalled is a sync that stopped on a repeated cursor.
	NoticeSyncStalled
	// NoticeOpenedImage is an image handed to the configured opener.
	NoticeOpenedImage
	// NoticeOpenedVideo is a video handed a playable URL to the player.
	NoticeOpenedVideo
	// NoticeOpenedGIF is an animated gif handed a playable URL to the player.
	NoticeOpenedGIF
	// NoticeOpenedPoster is a video or gif whose response carried no variant, so the
	// poster frame is what was handed over and the words say so.
	NoticeOpenedPoster
	// NoticeOpenedMedia is one action that opened several attachments: "opened {n} media".
	NoticeOpenedMedia
	// NoticeNoOpener is a missing opener; the pane prints the URL row beside it.
	NoticeNoOpener
	// NoticeNoMedia is `i` on a post with no attachments.
	NoticeNoMedia
	// NoticeOpenedBrowser is a permalink handed to the configured opener.
	NoticeOpenedBrowser
	// NoticeLinkCopied is a permalink written via OSC 52.
	NoticeLinkCopied
	// NoticeLinkCopiedWL is a permalink written via OSC 52 and also via wl-copy.
	NoticeLinkCopiedWL
	// NoticeNoClipboard is a copy that could write nowhere.
	NoticeNoClipboard
	// NoticeTranslated is a translation now on screen for the selected post: the program's own
	// announcement that the body it is showing in place of the original is the service's answer.
	NoticeTranslated
	// NoticeNoTranslation is a successful translation read that carried no translation for this post
	// — the service answered, and its answer was that it has nothing to give. It is not a failure:
	// a refusal is the classifier's, and it reaches the state field instead.
	NoticeNoTranslation
	// NoticeGated is a cover-breaking action refused under a cover. Its words come
	// from StatusFields.Gated (the active grammar's MarkerSet.Gated), not from
	// copy.go — empty in the plain grammar, where the notice is unreachable.
	NoticeGated
)

// SyncedField is the status line's synced field: present exactly when On,
// printing Markers.Sync plus the stamp, or Markers.SyncNever when At is zero.
type SyncedField struct {
	On bool
	At time.Time
}

// StatusFields is everything the status line prints, and nothing it does not: the state and its
// remedy, the notice, the list, the position and the budget.
type StatusFields struct {
	// Failure is the state to report, or a zero Failure for none. Its own words are printed: the
	// state word and the remedy come from copy.go, which state_test.go pins to the classifier.
	Failure domain.Failure
	// Notice is what to announce about the list.
	Notice NoticeKind
	// NoticeCount is the count the notice needs: the posts a refresh found.
	NoticeCount int
	// Cover is the display style the line is drawn under, or the zero value for the plain reader.
	// Every covered difference of the line is this field's: the state word's host, the list's name and
	// the notice's words. Nothing else about the line changes — not the field order, not the drop
	// rule, and not the remedy column, which stays the classifier's own instruction because a remedy
	// in the cover's vocabulary would be an instruction that does not work.
	Cover CoverStyle
	// List is the list field's own word: the list the frame is of, named the way the style on screen
	// names it. For a feed it is ListName's answer for that feed and for a conversation it is
	// ThreadListName's, so the thread frame prints `thread` and the feeds keep printing their own
	// words; the field stays droppable by the line's own declared rule whichever surface it names.
	List string
	// Selected and Loaded are the position: the selected post's place in the loaded list (1-based)
	// and how many posts are loaded. The field is omitted while nothing is loaded, because a
	// position in an empty list is not a position.
	Selected, Loaded int
	// HasMore is whether a next cursor is stored: it prints the position's `+`, so the pane never
	// implies there is more than the cursor says.
	HasMore bool
	// Used and Limited are the budget: the requests issued and the answers that came back rate
	// drift from the requests actually issued.
	Used, Limited int
	// Now is the clock the rate-limited state's reset instant is read against: the reset prints as
	// a wall-clock time when it falls on this clock's own local day, and as a dated time otherwise.
	// A zero Now has no day to compare against and takes the dated form, which is true whenever it
	// is read — the line never reads a clock of its own, because the program has exactly one.
	Now time.Time
	// Synced is the saved frame's provenance field: present exactly when On.
	Synced SyncedField
	// Capability is a list-local capability state. When set it occupies the state
	// and remedy slots and the failure's otherwise; the two are never both printed.
	Capability domain.Capability
	// CapabilityList is the list word substituted into the capability state word
	// ("bookmarks unavailable" / "likes unavailable").
	CapabilityList string
	// Gated is the refusal notice's words under a cover: filled from the active
	// Grammar's MarkerSet.Gated. Empty in the plain grammar (unreachable there);
	// NoticeGated is the one notice whose text comes from this field rather than
	// from copy.go.
	Gated string
}

// statusField is which field of the line a run belongs to. The order is the line's fixed order,
// and the drop rule is expressed over these values rather than over positions in a slice, so that
// dropping a field can never move another one into its place.
type statusField uint8

const (
	fieldState statusField = iota
	fieldRemedy
	fieldNotice
	fieldList
	fieldSynced
	fieldPosition
	fieldBudget
)

// statusGap is the separator between two fields: exactly two cells.
const statusGap = 2

// statusPart is one field's run: its text, and the role the palette paints it with.
type statusPart struct {
	field statusField
	role  Role
	text  string
}

// StatusLine composes the one status row for a set of fields, and is the only place the row's
// words are decided. Fields are separated by exactly two spaces; an empty field is omitted
// together with its separator, so no two adjacent gaps ever appear; and the row is one
// left-aligned run, so dropping a field never moves another.
//
// It never wraps. When the row does not fit w cells the fields are dropped in one fixed order —
// notice, then position, then list — and state, remedy and budget are never dropped: the budget
// is visible in every frame, every failure frame included. When all three droppable fields are
// gone and the row still does not fit, it is cut at w with the truncation mark in its last cell.
func StatusLine(f StatusFields, w int) string {
	return statusLine(f, w, For(RungFromEnv()))
}

// statusLine is StatusLine with the palette supplied, which is how the frame composer paints the
// status row without composing its words a second time.
func statusLine(f StatusFields, w int, pal Palette) string {
	parts := statusParts(f)
	droppable := []statusField{fieldNotice, fieldPosition, fieldList, fieldSynced}
	for dropped := 0; dropped <= len(droppable); dropped++ {
		kept := without(parts, droppable[:dropped])
		if text.Width(statusText(kept)) <= w {
			return paintStatus(kept, pal)
		}
	}
	return paintStatus(cutStatus(without(parts, droppable), w), pal)
}

// without returns the parts whose fields are not in dropped, in order.
func without(parts []statusPart, dropped []statusField) []statusPart {
	kept := make([]statusPart, 0, len(parts))
	for _, part := range parts {
		drop := false
		for _, field := range dropped {
			if part.field == field {
				drop = true
				break
			}
		}
		if !drop {
			kept = append(kept, part)
		}
	}
	return kept
}

// statusParts is the line's fields in their fixed order, with the empty ones left out: an empty
// field is not a field. The budget is always present, because the phase's budget is visible in
// every frame.
func statusParts(f StatusFields) []statusPart {
	parts := make([]statusPart, 0, 7)
	if f.Capability != domain.CapNone {
		// The capability state's two fields are the style's: under a cover the word and the remedy
		// are the cover's own, because the plain ones name the mirror, the file it lives in and the
		// program that pins an operation.
		if word := CapabilityWord(f.Cover, f.Capability, f.CapabilityList); word != "" {
			parts = append(parts, statusPart{fieldState, RoleState, word})
		}
		if word := CapabilityRemedy(f.Cover, f.Capability); word != "" {
			parts = append(parts, statusPart{fieldRemedy, RoleMuted, word})
		}
	} else {
		if word := stateField(f); word != "" {
			parts = append(parts, statusPart{fieldState, stateRole(f.Failure.Kind), word})
		}
		if word := remedyField(f.Failure); word != "" {
			parts = append(parts, statusPart{fieldRemedy, RoleMuted, word})
		}
	}
	if f.Notice == NoticeGated {
		if f.Gated != "" {
			parts = append(parts, statusPart{fieldNotice, RoleMuted, f.Gated})
		}
	} else if word := NoticeLine(f.Cover, f.Notice, f.NoticeCount); word != "" {
		parts = append(parts, statusPart{fieldNotice, RoleMuted, word})
	}
	if f.List != "" {
		parts = append(parts, statusPart{fieldList, RoleMuted, f.List})
	}
	if word := syncedField(f); word != "" {
		parts = append(parts, statusPart{fieldSynced, RoleMuted, word})
	}
	if word := positionField(f); word != "" {
		parts = append(parts, statusPart{fieldPosition, RoleMuted, word})
	}
	parts = append(parts, statusPart{fieldBudget, RoleMuted, budgetField(f)})
	return parts
}

// syncedField is the synced provenance field: Markers.Sync + stamp, or SyncNever.
func syncedField(f StatusFields) string {
	if !f.Synced.On {
		return ""
	}
	g := GrammarFor(f.Cover)
	if f.Synced.At.IsZero() {
		return g.Markers.SyncNever
	}
	return g.Markers.Sync + " " + SyncedStamp(f.Synced.At, f.Now)
}

// stateField is the state word, composed with the rate limit's reset instant when there is one.
// A zero reset adds no suffix: "rate limited" with no end to the wait is the truthful form, and
// the remedy next to it says so.
func stateField(f StatusFields) string {
	word := coverOr(f.Cover, HostSubstituted(StateWord(f.Failure.Kind)), StateWord(f.Failure.Kind))
	if word == "" {
		return ""
	}
	if f.Failure.Kind != domain.FailureRateLimited || f.Failure.Reset.IsZero() {
		return word
	}
	return word + " " + resetSuffix(f.Failure.Reset, f.Now)
}

// stateRole is the state word's role: the stop accent for the states whose session is over —
// expired, soft-banned, out of budget — and the recoverable accent for every other one. The two
// cases call for different human actions, and the words state the difference; the colour only
// repeats it.
func stateRole(kind domain.FailureKind) Role {
	switch kind {
	case domain.FailureSessionExpired, domain.FailureSoftBan, domain.FailureBudgetSpent:
		return RoleStateStop
	default:
		return RoleState
	}
}

// remedyField is the terse remedy column for the failure, with the rate-limited row's second
// form when no reset header was sent: a wait with no end is not the same instruction as a wait
// until a time.
func remedyField(f domain.Failure) string {
	if f.Kind == domain.FailureRateLimited && f.Reset.IsZero() {
		return CopyRemedyRateLimitedUnknown
	}
	return Remedy(f.Kind)
}

// resetSuffix renders the reset instant the way the state table writes it: the local wall-clock
// time when the reset falls on the clock's own local day, and the dated form otherwise. There is
// deliberately no countdown form — that would be a second clock in a program that has one, and a
// countdown is stale the moment the frame is read.
func resetSuffix(reset, now time.Time) string {
	zone := now.Location()
	local := reset.In(zone)
	if !now.IsZero() && local.Year() == now.Year() && local.YearDay() == now.YearDay() {
		return strings.ReplaceAll(CopyResetUntilClock, PlaceholderClock, local.Format("15:04"))
	}
	return strings.ReplaceAll(CopyResetUntilDate, PlaceholderDate, local.Format("Jan 02 15:04"))
}

// positionField is the position: the selected post's place in the loaded list, then the count of
// loaded posts, then `+` while a next cursor is stored. It is omitted while nothing is loaded.
func positionField(f StatusFields) string {
	if f.Loaded < 1 {
		return ""
	}
	selected := strconv.Itoa(f.Selected)
	loaded := strconv.Itoa(f.Loaded)
	if f.HasMore {
		line := strings.ReplaceAll(CopyPositionMore, PlaceholderSelected, selected)
		return strings.ReplaceAll(line, PlaceholderLoaded, loaded)
	}
	line := strings.ReplaceAll(CopyPosition, PlaceholderSelected, selected)
	return strings.ReplaceAll(line, PlaceholderLoaded, loaded)
}

// budgetField is the budget: the requests this process issued and the answers that came back rate
// limited, with the request noun singular at one. It is never dropped.
func budgetField(f StatusFields) string {
	template := CopyBudgetMany
	if f.Used == 1 {
		template = CopyBudgetOne
	}

	line := strings.ReplaceAll(template, PlaceholderCount, strconv.Itoa(f.Used))
	return strings.ReplaceAll(line, PlaceholderRateLimited, strconv.Itoa(f.Limited))
}

// statusText is the line's characters with no escapes: the two-space separators, then the fields
// in order. This is the string every width decision in this file is made on.
func statusText(parts []statusPart) string {
	var b strings.Builder
	for i, part := range parts {
		if i > 0 {
			b.WriteString(strings.Repeat(" ", statusGap))
		}
		b.WriteString(part.text)
	}
	return b.String()
}

// paintStatus paints each field's run with its role. The separators stay outside the runs: they
// are two cells of nothing and nothing paints them.
func paintStatus(parts []statusPart, pal Palette) string {
	var b strings.Builder
	for i, part := range parts {
		if i > 0 {
			b.WriteString(strings.Repeat(" ", statusGap))
		}
		b.WriteString(pal.paint(part.role, part.text))
	}
	return b.String()
}

// cutStatus cuts the composed fields so that the whole row fits w cells with the truncation mark
// in its last cell, keeping each surviving run's role: the last run is extended by the mark
// rather than by a role-less tail, so a cut state word keeps the state colour.
func cutStatus(parts []statusPart, w int) []statusPart {
	room := w - text.Width(MarkTruncation)
	if room < 0 {
		room = 0
	}
	kept := make([]statusPart, 0, len(parts))
	used := 0
	for i, part := range parts {
		if i > 0 {
			if used+statusGap > room {
				break
			}
			used += statusGap
		}
		width := text.Width(part.text)
		if used+width <= room {
			kept = append(kept, part)
			used += width
			continue
		}
		if room > used {
			kept = append(kept, statusPart{part.field, part.role, text.Truncate(part.text, room-used)})
		}
		break
	}
	if len(kept) == 0 {
		// The row is narrower than everything that goes in it: the mark is all there is room for.
		return []statusPart{{fieldBudget, RoleMuted, MarkTruncation}}
	}
	last := len(kept) - 1
	// The mark goes in the last cell the row occupies, and the spaces the cut left behind it are
	// not cells the row says anything in.
	kept[last].text = strings.TrimRight(kept[last].text, " ") + MarkTruncation
	return kept
}
