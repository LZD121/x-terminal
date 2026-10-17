package tui

// The entry sequence. It is data and a counter, and nothing else in the package knows the order of
// the keys: the matcher is fed one keystroke at a time from press and answers with what to do with
// it.
//
// The rule that makes this safe to sit in front of the keymap is that it withholds exactly one key.
// Eight arrows are movements a reader makes without consequence, so they pass through untouched; the
// ninth key is `b`, which is bookmarks and would spend a request, so it is held back. If the next
// key is `a` the presentation opens and nothing was spent. If it is anything else, the held `b` is
// replayed as the bookmark key it always was and the new key is dispatched normally.

// konamiStep is one key of the sequence, spelled the way KeyPressMsg reports it.
var konamiSequence = [...]string{
	"up", "up", "down", "down", "left", "right", "left", "right", "b",
}

// konamiEffect is what the matcher says to do with the keystroke it was handed.
type konamiEffect uint8

const (
	// konamiNone is an ordinary key: the matcher does not want it and it was not part of a sequence
	// in progress.
	konamiNone konamiEffect = iota
	// konamiHeld is the one key the matcher keeps: the sequence is complete but unconfirmed, and
	// the key must not reach the keymap.
	konamiHeld
	// konamiEnter is a completed sequence: the presentation opens.
	konamiEnter
	// konamiReplay is a sequence that broke at the held key: the held key must be dispatched as the
	// action it always had, and the current key processed normally.
	konamiReplay
)

// konami is the sequence's state: how many of its keys have matched, in order, with no gap.
//
// A zero konami is the matcher at rest, which is what a model copy that never typed anything holds.
// The value is copied with the model and mutated only by advance, so it is never shared.
type konami struct {
	step int
}

// advance feeds one keystroke in and answers with the matcher's new value and what to do.
//
// The eight arrow keys are matched by their spellings and pass through normally: a reader who types
// three downs and stops is a reader who moved three rows. Only the ninth key is withheld, and only
// once the eight are already in — which is what keeps both halves honest: the sequence never costs a
// request, and ordinary movement never feels a delay.
func (k konami) advance(key string) (konami, konamiEffect) {
	// The held key is already complete; the next key either confirms it or breaks it. This arm is
	// checked first because `a` is not a key of the sequence.
	if k.step == len(konamiSequence) {
		if key == "a" {
			return konami{}, konamiEnter
		}
		return konami{}, konamiReplay
	}
	if key == konamiSequence[k.step] {
		next := konami{step: k.step + 1}
		if next.step == len(konamiSequence) {
			return next, konamiHeld
		}
		return next, konamiNone
	}
	// A mismatch restarts the match, and the key itself may be the sequence's first one: two ups in
	// a row are two ups, and the second one is not a failed restart.
	if k.step > 0 && key == konamiSequence[0] {
		return konami{step: 1}, konamiNone
	}
	return konami{}, konamiNone
}
