package xweb

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

// fakeClock is the limiter's injected time: now reads it, sleep advances it, and a test
// can move it forward by hand. Every wait the limiter takes is therefore a number in a
// slice rather than real seconds on the wall clock.
type fakeClock struct {
	at    time.Time
	slept []time.Duration
}

func newFakeClock() *fakeClock {
	return &fakeClock{at: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) install(l *Limiter) {
	l.now = func() time.Time { return c.at }
	l.sleep = func(_ context.Context, d time.Duration) error {
		c.slept = append(c.slept, d)
		c.at = c.at.Add(d)
		return nil
	}
}

func (c *fakeClock) advance(d time.Duration) { c.at = c.at.Add(d) }

// answered is one request that succeeds with a status.
func answered(status int) func() (*http.Response, error) {
	return func() (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: http.NoBody}, nil
	}
}

func TestLimiterSpacesRequestsByTheFloor(t *testing.T) {
	clock := newFakeClock()
	l := NewLimiter(1500*time.Millisecond, 200)
	clock.install(l)

	for i := 1; i <= 2; i++ {
		if _, err := l.Do(context.Background(), answered(200)); err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
	}
	// Anti-vacuity: the second request was issued inside the floor, so a limiter that
	// consults anything at all had to wait once. An empty list here would mean the
	// assertions below prove nothing about the floor.
	if len(clock.slept) != 1 {
		t.Fatalf("waits = %v, want exactly one: the second request was inside the floor", clock.slept)
	}
	if got := clock.slept[0]; got != 1500*time.Millisecond {
		t.Errorf("waited %s, want the whole floor, 1.5s", got)
	}
	atIssue := clock.at
	if used, limited, reset := l.Budget(); used != 2 || limited != 0 || !reset.Equal(atIssue.Add(1500*time.Millisecond)) {
		t.Errorf("Budget() = (%d used, %d limited, reset %s), want (2, 0, %s)", used, limited, reset, atIssue.Add(1500*time.Millisecond))
	}

	// A request issued after the floor has already passed waits for nobody: the floor is a
	// spacing, not a queue with a debt to pay off.
	clock.advance(10 * time.Second)
	if _, err := l.Do(context.Background(), answered(200)); err != nil {
		t.Fatalf("third request: %v", err)
	}
	if len(clock.slept) != 1 {
		t.Errorf("a request issued 10s after the previous one still waited: %v", clock.slept)
	}
}

func TestLimiterStopsAtExactlyTheCap(t *testing.T) {
	l := NewLimiter(0, 200)
	issued := 0
	for i := 1; i <= 200; i++ {
		_, err := l.Do(context.Background(), func() (*http.Response, error) {
			issued++
			return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
		})
		if err != nil {
			t.Fatalf("request %d of 200 was refused: %v", i, err)
		}
	}

	// The 200th call is the boundary on both sides: a cap that tripped at 199 would have
	// failed the loop above, and one that never trips would not fail here.
	_, err := l.Do(context.Background(), func() (*http.Response, error) {
		issued++
		return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
	})
	if !errors.Is(err, ErrRequestCap) {
		t.Fatalf("request 201: err = %v, want ErrRequestCap", err)
	}
	if issued != 200 {
		t.Errorf("the request function ran %d times, want 200: a refused request must not be sent", issued)
	}
	if used, _, _ := l.Budget(); used != 200 {
		t.Errorf("used = %d, want 200", used)
	}
}

func TestLimiterMakesTheSecondConcurrentCallWait(t *testing.T) {
	l := NewLimiter(0, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	entered, release := make(chan struct{}), make(chan struct{})
	first := make(chan error, 1)
	go func() {
		_, err := l.Do(ctx, func() (*http.Response, error) {
			close(entered)
			<-release
			return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
		})
		first <- err
	}()
	<-entered

	second := make(chan error, 1)
	go func() {
		_, err := l.Do(ctx, answered(200))
		second <- err
	}()

	select {
	case err := <-second:
		t.Fatalf("the second call returned while the first was still in flight (err = %v) — it must wait, not fail", err)
	case <-time.After(50 * time.Millisecond):
		// This 50ms is not the floor: it is how the test observes that the second call is
		// still blocked. The limiter's own waits are all on the fake clock in the tests above.
	}

	close(release)
	if err := <-first; err != nil {
		t.Errorf("the first call failed: %v", err)
	}
	select {
	case err := <-second:
		if err != nil {
			t.Errorf("the second call failed after waiting: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the second call never ran, even after the first finished")
	}
}

func TestLimiterCountsRateLimitedAnswersAndNeverRetriesThem(t *testing.T) {
	l := NewLimiter(0, 10)

	calls := 0
	if _, err := l.Do(context.Background(), func() (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 429, Body: http.NoBody}, nil
	}); err != nil {
		t.Fatalf("the 429 itself was an error to the limiter: %v", err)
	}
	if calls != 1 {
		t.Errorf("the request function ran %d times for one 429, want 1: the limiter retries nothing", calls)
	}
	if _, err := l.Do(context.Background(), answered(200)); err != nil {
		t.Fatalf("the following 200: %v", err)
	}

	if used, limited, _ := l.Budget(); used != 2 || limited != 1 {
		t.Errorf("Budget() = (%d used, %d limited), want (2, 1): only the 429 is a limited request", used, limited)
	}
}
