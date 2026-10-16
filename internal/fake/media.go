package fake

import (
	"context"
	"sync"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/media"
)

// Renderer is the recording media.Renderer the phase's tests are driven by: it
// records every ref and URL it was handed and answers a scripted outcome, so a
// test can assert what a keypress did without spawning anything. It is a test
// double and nothing else — the package breaks the boundary rule deliberately:
// the shell may not import it, and internal/tui's boundary test already forbids
// the import. Answering without spawning is what makes the gate in plan 03
// provable as zero calls.
type Renderer struct {
	mu      sync.Mutex
	Seen    []domain.MediaRef
	SeenURL []string
	Answer  media.Result
	// Answers, when non-nil, is consumed in call order before Answer. It lets a
	// mixed-success test script per-attachment outcomes without a second type.
	Answers []media.Result
	Err     error
}

// Open records the ref and returns the scripted answer.
func (r *Renderer) Open(ctx context.Context, ref domain.MediaRef) (media.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Seen = append(r.Seen, ref)
	if len(r.Answers) > 0 {
		out := r.Answers[0]
		r.Answers = r.Answers[1:]
		return out, r.Err
	}
	return r.Answer, r.Err
}

// OpenURL records the URL and returns the scripted answer.
func (r *Renderer) OpenURL(ctx context.Context, url string) (media.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.SeenURL = append(r.SeenURL, url)
	if len(r.Answers) > 0 {
		out := r.Answers[0]
		r.Answers = r.Answers[1:]
		return out, r.Err
	}
	return r.Answer, r.Err
}

// Calls returns a copy of every ref Open was handed, in call order.
func (r *Renderer) Calls() []domain.MediaRef {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.MediaRef, len(r.Seen))
	copy(out, r.Seen)
	return out
}

// Count is how many times Open was called.
func (r *Renderer) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.Seen)
}

// URLCount is how many times OpenURL was called.
func (r *Renderer) URLCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.SeenURL)
}

// Reset clears the recorded calls.
func (r *Renderer) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Seen = nil
	r.SeenURL = nil
}

// Clipboard is the recording media clipboard the gate and copy-link tests drive:
// Copy counts and answers a scripted outcome without writing an escape or spawning.
// Default Answer is NoClipboard so an unset script still reports honestly.
type Clipboard struct {
	mu     sync.Mutex
	N      int
	Last   string
	Answer media.ClipboardOutcome
}

// Copy increments the call count, records the URL, and returns Answer (or
// NoClipboard when Answer is the zero value).
func (c *Clipboard) Copy(ctx context.Context, url string) media.ClipboardOutcome {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.N++
	c.Last = url
	if c.Answer == media.ClipboardNone {
		return media.NoClipboard
	}
	return c.Answer
}

// Count is how many times Copy was called.
func (c *Clipboard) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.N
}

// Reset clears the recorded calls.
func (c *Clipboard) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.N = 0
	c.Last = ""
}
