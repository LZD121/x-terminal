package xsource

import (
	"context"
	"errors"
	"testing"
)

func TestViewerReadsTheSessionAccount(t *testing.T) {
	body := `{"data":{"viewer":{"user_results":{"result":{"__typename":"User","rest_id":"783214","core":{"screen_name":"X","name":"X"}}}}}}`
	client := &recordingClient{body: []byte(body)}
	user, err := New(client).Viewer(context.Background())
	if err != nil {
		t.Fatalf("Viewer: %v", err)
	}
	if client.op != "Viewer" {
		t.Fatalf("operation = %q, want Viewer", client.op)
	}
	if client.vars != nil {
		t.Fatalf("variables = %#v, want nil", client.vars)
	}
	if user.ScreenName != "X" || user.ID != "783214" {
		t.Fatalf("user = %+v, want @X / 783214", user)
	}
}

func TestViewerPassesTransportErrorsThrough(t *testing.T) {
	want := errors.New("transport refused")
	client := &recordingClient{err: want}
	_, err := New(client).Viewer(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Viewer = %v, want transport error", err)
	}
	if client.op != "Viewer" {
		t.Fatalf("operation = %q, want Viewer even on error", client.op)
	}
}
