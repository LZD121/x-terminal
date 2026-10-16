package xweb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ErrRequestCap is the typed "this session has spent its budget" state. It is a
// sentinel so a caller can test for it with errors.Is and render it in its own
// words; the wrapped message carries the numbers.
var ErrRequestCap = errors.New("xweb: session request cap reached")

// Limiter is the one place a request to the upstream is issued from. It holds the
// account to one request at a time (single-flight), spaces requests at least Floor
// apart, and refuses the request after the Cap'th.
//
// The clock and the wait are injected rather than read from the process, so the
// floor and the single-flight wait are exercised by the tests in microseconds
// instead of real seconds. This is also the only type in the repository that sleeps
// between requests: a caller that waited on its own would be spending budget this
// type counts but cannot see.
type Limiter struct {
	floor time.Duration
	cap   int

	now   func() time.Time
	sleep func(ctx context.Context, d time.Duration) error

	// busy is the single-flight slot — capacity one, so a second caller waits for the
	// first to finish rather than failing. A channel rather than a sync.Mutex, so a
	// caller whose context ends while it waits can leave instead of blocking until
	// the request in front of it does.
	busy chan struct{}

	mu      sync.Mutex
	used    int
	limited int
	last    time.Time
}

// NewLimiter returns a limiter that spaces requests at least floor apart and
// refuses the (requestCap+1)th request.
//
// A floor of zero means "no floor" and is what the tests that are not about timing
// use; config.Load refuses a non-positive floor from a configuration file, so a
// zero here can only have been written on purpose.
func NewLimiter(floor time.Duration, requestCap int) *Limiter {
	return &Limiter{
		floor: floor,
		cap:   requestCap,
		now:   time.Now,
		sleep: sleepContext,
		busy:  make(chan struct{}, 1),
	}
}

// sleepContext waits for d, or until ctx ends, whichever happens first.
func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Do issues one request. fn is what actually opens it, and it is called at most
// once: this limiter never retries, because a 429 is the upstream asking for less
// traffic rather than for another attempt.
//
// The rules are applied in the order that fails fastest: the single-flight slot is
// taken first, so a second caller waits; the cap is checked before the floor is
// waited out, so a spent session fails at once instead of sleeping first; then the
// floor is honoured, and only then is the request issued.
func (l *Limiter) Do(ctx context.Context, fn func() (*http.Response, error)) (*http.Response, error) {
	if fn == nil {
		return nil, errors.New("xweb: limiter.Do needs a function to call")
	}

	select {
	case l.busy <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-l.busy }()

	l.mu.Lock()
	if l.used >= l.cap {
		used, limit := l.used, l.cap
		l.mu.Unlock()
		return nil, fmt.Errorf("%w: %d requests issued, cap %d", ErrRequestCap, used, limit)
	}
	wait := time.Duration(0)
	if !l.last.IsZero() {
		if elapsed := l.now().Sub(l.last); elapsed < l.floor {
			wait = l.floor - elapsed
		}
	}
	l.mu.Unlock()

	if wait > 0 {
		if err := l.sleep(ctx, wait); err != nil {
			return nil, err
		}
	}

	// The request is counted as it is issued, not as it succeeds: a connect that times
	// out may well have reached the upstream's edge, and a budget that only counted
	// answers would under-report exactly when the account is in the most trouble.
	l.mu.Lock()
	l.used++
	l.last = l.now()
	l.mu.Unlock()

	resp, err := fn()
	if err == nil && resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		l.mu.Lock()
		l.limited++
		l.mu.Unlock()
	}
	return resp, err
}

// Budget reports what the session has spent: how many requests were issued, how many
// of the answers were 429, and the earliest time at which the next request may be
// issued. It is what the doctor's closing line prints.
func (l *Limiter) Budget() (used int, limited int, reset time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.last.IsZero() {
		return l.used, l.limited, l.now()
	}
	return l.used, l.limited, l.last.Add(l.floor)
}
