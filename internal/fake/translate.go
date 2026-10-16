package fake

import (
	"context"
	"sync"

	"github.com/LZD121/x-terminal/internal/domain"
)

// TranslateCall is one recorded translation request: the post it was about and the language it was
// asked for, which are the two things the port takes. The language is recorded rather than assumed
// because it is the setting the composition root resolved, and a double that dropped it could not
// tell a shell that handed the wrong one from a shell that handed none.
type TranslateCall struct {
	ID   domain.PostID
	Lang string
}

// Translator is the recording domain.Translator this feature's tests are driven by: it records the
// (id, language) of every call and answers a scripted domain.Translation, so a test can assert what
// a keypress did — and, more to the point, how many times it did it — without a socket. It is a test
// double and nothing else: the shell may not import this package, which internal/tui's boundary test
// already forbids, and the count it keeps is what the "a post already translated costs no second
// request" acceptance is asserted against.
//
// It imports internal/domain only: the answer it hands back is the port's own value, so nothing about
// the wire (a status, a body, an upstream state word) can leak into a shell test through it.
type Translator struct {
	mu sync.Mutex
	// seen is every call in order, and it is the record Count and Calls report.
	seen []TranslateCall
	// Answer is what every call returns. Available true with a Text is the service answering with a
	// translation; the zero value is the service answering with nothing to give — the state the
	// honest-notice arm is about — and it is deliberately a real answer rather than an error.
	Answer domain.Translation
	// Err, when non-nil, is returned with the zero Translation.
	Err error
	// failNext is the one-shot error FailNext plants, and it is consumed by the first call to see it.
	failNext error
}

// Translate records the call and returns the scripted answer.
func (t *Translator) Translate(ctx context.Context, id domain.PostID, lang string) (domain.Translation, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.seen = append(t.seen, TranslateCall{ID: id, Lang: lang})
	if t.failNext != nil {
		err := t.failNext
		t.failNext = nil
		return domain.Translation{}, err
	}
	if t.Err != nil {
		return domain.Translation{}, t.Err
	}
	return t.Answer, nil
}

// FailNext makes the next call fail with err, and only that call: one failure is what a refusal is,
// and a double that failed every call could not tell a shell that handled one from a shell that
// stopped asking.
func (t *Translator) FailNext(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failNext = err
}

// Calls returns a copy of every recorded call, in call order.
func (t *Translator) Calls() []TranslateCall {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]TranslateCall, len(t.seen))
	copy(out, t.seen)
	return out
}

// Count is how many times Translate was called.
func (t *Translator) Count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.seen)
}

// Reset clears the recorded calls.
func (t *Translator) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.seen = nil
}

// Adapter implements the port, so a test that hands this double to the shell cannot be holding a
// value whose signature drifted from the one the shell calls.
var _ domain.Translator = (*Translator)(nil)
