package render

import (
	"strings"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// ListSaved is the status line's list word for a saved kind, named the way the
// style on screen names it. Under a cover the product words `bookmarks` /
// `likes` are replaced by each style's own literals; the zero style keeps the
// plain names Phase 6 stored.
func ListSaved(cover CoverStyle, k domain.SavedKind) string {
	switch cover {
	case CoverServiceLog:
		if k == domain.Like {
			return CoverServiceLogListLikes
		}
		return CoverServiceLogListBookmarks
	case CoverGitLog:
		if k == domain.Like {
			return CoverGitListLikes
		}
		return CoverGitListBookmarks
	case CoverCI:
		if k == domain.Like {
			return CoverCIListLikes
		}
		return CoverCIListBookmarks
	case CoverInternal:
		if k == domain.Like {
			return CoverInternalListLikes
		}
		return CoverInternalListBookmarks
	}
	if k == domain.Like {
		return CopyListLikes
	}
	return CopyListBookmarks
}

// CapabilityWord is the state column for a capability, with the list's word
// substituted into "bookmarks unavailable" / "likes unavailable".
//
// Under a cover it is the cover's own word for the same state. The plain reader's two of its three
// words name the mirror and the file the account's own list lives in, and its unavailable form
// names the list by its product word — none of which a covered frame may hold — so a covered frame
// reads its state word out of the cover's vocabulary instead. Which state it is survives the
// substitution: the three covered words are distinct and each says what happened.
func CapabilityWord(cover CoverStyle, c domain.Capability, list string) string {
	if cover != "" {
		return coverCapabilityWord(c)
	}
	switch c {
	case domain.CapOperationMissing:
		if list == "" {
			list = "list"
		}
		return strings.ReplaceAll(CopyCapUnavailable, "{list}", list)
	case domain.CapWrongAccount:
		return CopyCapWrongAccount
	case domain.CapMirrorUnreadable:
		return CopyCapMirrorUnreadable
	default:
		return ""
	}
}

func coverCapabilityWord(c domain.Capability) string {
	switch c {
	case domain.CapOperationMissing:
		return CopyCoverCapUnavailable
	case domain.CapWrongAccount:
		return CopyCoverCapWrongAccount
	case domain.CapMirrorUnreadable:
		return CopyCoverCapUnreadable
	default:
		return ""
	}
}

// CapabilityRemedy is the terse remedy column for a capability, and under a cover the cover's own
// line for it: the plain three name the program that pins an operation and the file two of them
// tell the reader to move or delete.
func CapabilityRemedy(cover CoverStyle, c domain.Capability) string {
	if cover != "" {
		return coverCapabilityRemedy(c)
	}
	switch c {
	case domain.CapOperationMissing:
		return CopyCapRemedyPin
	case domain.CapWrongAccount:
		return CopyCapRemedyRepoint
	case domain.CapMirrorUnreadable:
		return CopyCapRemedyAside
	default:
		return ""
	}
}

func coverCapabilityRemedy(c domain.Capability) string {
	switch c {
	case domain.CapOperationMissing:
		return CopyCoverCapRemedyPin
	case domain.CapWrongAccount:
		return CopyCoverCapRemedyRepoint
	case domain.CapMirrorUnreadable:
		return CopyCoverCapRemedyAside
	default:
		return ""
	}
}

// CapabilityFacts are the values a capability pane block substitutes. They are the plain reader's
// values: a covered block substitutes none of them, because the state's own sentence is the
// cover's, and the path and the owner's handle are exactly what it must not print.
type CapabilityFacts struct {
	Path, Owner, List string
}

// CapabilityBlock is the pane's state block for a capability: the state word as the head line and
// the capability's own sentence as the body, in the shape and the words of the grammar the pane is
// painted with. A covered frame's head and body both come from the cover's own vocabulary
// (CapabilityWord, capabilityBody), so the mirror's path, the account the mirror was filled from
// and the program that pins an operation cannot reach a covered pane through this state.
func CapabilityBlock(g Grammar, c domain.Capability, f CapabilityFacts, w int) []Block {
	cover, _ := CoverOfGrammar(g)
	return []Block{StateBlock(g, w, CapabilityWord(cover, c, f.List), capabilityBody(cover, c, f))}
}

func capabilityBody(cover CoverStyle, c domain.Capability, f CapabilityFacts) string {
	if cover != "" {
		return coverCapabilityBody(c)
	}
	switch c {
	case domain.CapOperationMissing:
		return strings.ReplaceAll(CopyCapBodyMissing, "{list}", f.List)
	case domain.CapWrongAccount:
		s := strings.ReplaceAll(CopyCapBodyWrongAccount, "{owner}", f.Owner)
		return strings.ReplaceAll(s, "{path}", f.Path)
	case domain.CapMirrorUnreadable:
		return strings.ReplaceAll(CopyCapBodyUnreadable, "{path}", f.Path)
	default:
		return ""
	}
}

func coverCapabilityBody(c domain.Capability) string {
	switch c {
	case domain.CapOperationMissing:
		return CopyCoverCapBodyMissing
	case domain.CapWrongAccount:
		return CopyCoverCapBodyWrongAccount
	case domain.CapMirrorUnreadable:
		return CopyCoverCapBodyUnreadable
	default:
		return ""
	}
}

// SavedEmptyState is the empty frame's blocks for a saved list, in the grammar's own shape and
// words. The plain reader distinguishes the three cases it has — a likes list the upstream may no
// longer serve, a store nothing has ever filled, and a store that was read and held nothing —
// because each names a different fact. Under a cover the words are the cover's own and the cases are
// the same three, in the same order and with the same shapes: the likes list's one hedged line is
// one line under every grammar (the ambiguity is about the emptiness itself, so a body saying the
// read succeeded would resolve what the line exists to leave open), and the bookmarks empty read is
// the cover's heading over the cover's body — the never-filled case keeping the sentence that says
// so. A covered likes-empty frame is therefore *not* the home feeds' empty state: it is the one
// state whose answer cannot be read as "nothing was there".
func SavedEmptyState(g Grammar, kind domain.SavedKind, synced bool, w int) []Block {
	cover, _ := CoverOfGrammar(g)
	if kind == domain.Like {
		if cover != "" {
			return []Block{StateBlock(g, w, CopyCoverEmptyHedged, "")}
		}
		return []Block{StateBlock(g, w, CopyEmptyLikesHedged, "")}
	}
	if cover != "" {
		body := CopyCoverEmptyBody
		if !synced {
			body = CopyCoverEmptyNeverMirrored
		}
		return []Block{StateBlock(g, w, CopyCoverEmptyHeading, body)}
	}
	body := CopyEmptyBody
	if !synced {
		body = strings.ReplaceAll(CopyEmptyBodyNeverSynced, "{list}", kind.List())
	}
	return []Block{StateBlock(g, w, CopyEmptyHeadingBookmarks, body)}
}

// StateBlockWithMirror is Phase 2's two-line state block plus the third line naming the empty
// mirror, rendered only when the state is a network failure and the mirror held no rows for that
// list. The grammar is read twice: its shape composes the block, and — where it is a cover — its
// own words replace the classifier's sentence, the mirror's name and the list's product word.
func StateBlockWithMirror(g Grammar, f domain.Failure, kind domain.SavedKind, mirrored bool, w int) []Block {
	cover, _ := CoverOfGrammar(g)
	word, body := FailureWords(cover, f.Kind, f.Detail)
	b := StateBlock(g, w, word, body)
	if mirrored || f.Kind == domain.FailureNone {
		return []Block{b}
	}
	for _, line := range text.Wrap(savedNeverMirrored(cover, kind), g.measure(w, 0, 0)) {
		b.Rows = append(b.Rows, mirrorThirdRow(g, cover, w, line))
	}
	return []Block{b}
}

// savedNeverMirrored is the third line's sentence for a style: the plain reader's names the list
// and what it is missing (`nothing is mirrored for bookmarks yet`), and a cover's says the same
// fact in its own words with neither.
func savedNeverMirrored(cover CoverStyle, kind domain.SavedKind) string {
	if cover != "" {
		return CopyCoverEmptyNeverMirrored
	}
	return strings.ReplaceAll(CopyEmptyBodyNeverSynced, "{list}", kind.List())
}

// mirrorThirdRow is that sentence as a row of the state block. A cover's row is the one its own
// body rows have — its gutter and its indent, which is what StateBlock draws for the two body
// lines above it — and the plain reader's is the two cells Phase 2's frame for this state draws,
// left exactly as that frame has them.
func mirrorThirdRow(g Grammar, cover CoverStyle, w int, line string) string {
	if cover == "" {
		return "  " + line
	}
	b := &builder{grammar: g, width: w, pal: For(RungFromEnv())}
	return b.bodyRow(" ", "", line)
}

// SyncedStamp is the resolution-falls-with-age form for the synced field.
// today -> 15:04 (12 cells with "synced "), earlier this year -> Jan 02,
// earlier year -> Jan  2006.
func SyncedStamp(at, now time.Time) string {
	if at.IsZero() {
		return ""
	}
	local := at.In(now.Location())
	n := now.In(now.Location())
	if local.Year() == n.Year() && local.YearDay() == n.YearDay() {
		return local.Format("15:04")
	}
	if local.Year() == n.Year() {
		return local.Format("Jan 02")
	}
	return local.Format("Jan  2006")
}

// GrammarFor returns the grammar a cover style wears, or Plain for the reader.
func GrammarFor(cover CoverStyle) Grammar {
	if cover == "" {
		return Plain
	}
	return cover.Grammar()
}
