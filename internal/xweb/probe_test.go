package xweb

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// probeBodyMarker is the needle the two marker tests below search for: a string that appears in
// a response BODY and in no field of Probe. It is deliberately not the errors[].message string:
// Detail carries that sentence on purpose now (it is the upstream's own name for what the
// request shape got wrong), so a marker placed there would fail the very behaviour this task
// added. It sits in the data half of a body this client never reads into a Probe.
const probeBodyMarker = "PROBE-BODY-MARKER-MUST-NOT-LEAK"

func TestProbeIssuesExactlyOneRequestIncluding429(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("x-rate-limit-remaining", "3")
		w.Header().Set("x-rate-limit-reset", fmt.Sprintf("%d", time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC).Unix()))
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"errors":[{"code":88,"message":"rate limited"}],"data":{"marker":"`+probeBodyMarker+`"}}`)
	}))
	defer srv.Close()

	client := testClient(t, srv, fixtureSession())
	probe, err := client.Probe(context.Background(), "Viewer", map[string]any{})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("Probe issued %d requests, want exactly 1", got)
	}
	if probe.Outcome != OutcomeRateLimited {
		t.Fatalf("outcome = %q, want %q", probe.Outcome, OutcomeRateLimited)
	}
	if !probe.RemainingKnown || probe.Remaining != 3 {
		t.Fatalf("remaining = %d known=%v", probe.Remaining, probe.RemainingKnown)
	}
	if probe.Reset.IsZero() {
		t.Fatal("reset missing")
	}
	// The upstream's own sentence is the one thing Detail is for, and it is the errors[] message
	// rather than the body: a body that carried more than a message contributes only the message.
	if probe.Detail != "rate limited" {
		t.Errorf("Detail = %q, want the upstream's own errors[] message", probe.Detail)
	}
	v := reflect.ValueOf(probe)
	for i := range v.NumField() {
		field := v.Field(i)
		if field.Kind() == reflect.String && strings.Contains(field.String(), probeBodyMarker) {
			t.Fatalf("Probe field %s carries the body marker", v.Type().Field(i).Name)
		}
	}
	if strings.Contains(fmt.Sprintf("%+v", probe), probeBodyMarker) {
		t.Fatal("Probe string form carries the body marker")
	}
}

// TestProbeDetailCarriesTheUpstreamsSentenceAndNothingElse is the arm T-r3a-02 asks for: on a
// rejected request shape the probe names the variable the upstream named, and it carries no
// byte of the rest of the body.
func TestProbeDetailCarriesTheUpstreamsSentenceAndNothingElse(t *testing.T) {
	const upstream = `Variable '$includePromotedContent' of required type 'Boolean!' was not provided`
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"errors":[{"message":"`+upstream+`","extensions":{"value":0}}],"data":null,"extra":"`+probeBodyMarker+`"}`)
	}))
	defer srv.Close()

	client := testClient(t, srv, fixtureSession())
	probe, err := client.Probe(context.Background(), "Viewer", map[string]any{})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if probe.Outcome != OutcomeBadRequest {
		t.Fatalf("outcome = %q, want %q", probe.Outcome, OutcomeBadRequest)
	}
	if probe.Detail != upstream {
		t.Fatalf("Detail = %q, want the upstream's own sentence %q", probe.Detail, upstream)
	}
	if strings.Contains(probe.Detail, probeBodyMarker) {
		t.Fatal("Detail carried body content beyond the upstream's message")
	}
}

func TestProbeDiscardsSuccessBodyMarker(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("content-type", "application/json")
		w.Header().Set("x-rate-limit-remaining", "19")
		fmt.Fprint(w, `{"data":{"viewer":{"marker":"`+probeBodyMarker+`"}}}`)
	}))
	defer srv.Close()

	client := testClient(t, srv, fixtureSession())
	probe, err := client.Probe(context.Background(), "Viewer", nil)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d", calls.Load())
	}
	if !probe.Outcome.Success() {
		t.Fatalf("outcome = %q", probe.Outcome)
	}
	if strings.Contains(fmt.Sprintf("%#v", probe), probeBodyMarker) {
		t.Fatal("success Probe retained the body marker")
	}
}

func TestSignerNameIsNoneWithNoopSigner(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	client := testClient(t, srv, fixtureSession())
	if got := client.SignerName(); got != "none" {
		t.Fatalf("SignerName = %q, want none", got)
	}
}
