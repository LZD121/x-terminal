package xweb

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/LZD121/x-terminal/internal/session"
)

// Probe is what one --probe request reports: the classified outcome, the
// rate-limit headers the answer carried, how many response bytes were seen, and,
// for a rejected request, the upstream's own bounded sentence.
//
// There is still no field for the body, and that is the point: Detail carries what
// apiMessage already returns to APIError — the upstream's own sentence from an
// errors[] message (bounded by errorDetailLimit runes), or the JSON body verbatim
// when a JSON answer carries no message. It is the sentence the classifier's remedy
// already promises to name ("fix the variables or features it names"), so a repair
// can read the variable name out of it instead of guessing one build at a time. It
// is never the body, never a post, never a token; the success path sets nothing,
// and probe_test.go fails if any field of this struct carries body content.
//
// A repair spends exactly one request and reads one word, the headers, and (when
// the shape was refused) the name of what the upstream wanted.
type Probe struct {
	Outcome        Outcome
	Remaining      int
	RemainingKnown bool
	Reset          time.Time
	Bytes          int
	Detail         string
}

// Probe issues exactly one request for opName through Client.GraphQL's own
// path: the same limiter (single-flight, the floor, the session cap), the same
// registry lookup, the same wire method, and the same classifier. The body is
// discarded inside the method and never returned.
//
// A non-success outcome returns the probe with a nil error, the classified
// outcome set (the caller prints the word) and Detail carrying the upstream's own
// bounded sentence about it. Only a request that could not be built at all (an
// unknown operation name, a missing session, a non-query operation) returns an
// error.
func (c *Client) Probe(ctx context.Context, opName string, variables map[string]any) (Probe, error) {
	op, err := c.operation(opName)
	if err != nil {
		return Probe{}, err
	}
	if op.OperationType != "query" {
		return Probe{}, fmt.Errorf("xweb: %s is a %s, and this client issues queries only", opName, op.OperationType)
	}
	if !c.session.Valid() {
		return Probe{}, fmt.Errorf("xweb: %s cannot be issued without a session: %w", opName, session.ErrNoSession)
	}

	req, err := c.graphQLRequest(ctx, opName, op, variables)
	if err != nil {
		return Probe{}, err
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return Probe{}, fmt.Errorf("xweb: %s: %w", opName, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody+1))
	if err != nil {
		return Probe{}, fmt.Errorf("xweb: %s: reading the answer: %w", opName, err)
	}
	if len(body) > maxResponseBody {
		return Probe{}, fmt.Errorf("xweb: %s answered more than the %d bytes this client will hold", opName, maxResponseBody)
	}

	outcome := Classify(resp.StatusCode, resp.Header, body)
	remaining, remainingKnown := rateLimitRemaining(resp.Header)
	reset, _ := rateLimitReset(resp.Header)
	// The upstream's own sentence travels only when the answer was not data: a successful
	// body contributes nothing to the probe, which is what keeps this field a diagnostic
	// rather than a second copy of the response.
	detail := ""
	if !outcome.Success() {
		detail = apiMessage(body)
	}
	return Probe{
		Outcome:        outcome,
		Remaining:      remaining,
		RemainingKnown: remainingKnown,
		Reset:          reset,
		Bytes:          len(body),
		Detail:         detail,
	}, nil
}
