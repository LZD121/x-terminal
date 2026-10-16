package screen

import (
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

var profileNow = time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)

func TestProfileOpenIssuesOnlyTheHeader(t *testing.T) {
	p, effect := Open("lzd")
	if effect.Kind != EffectPage {
		t.Fatalf("open effect is %v, want page", effect.Kind)
	}
	req, ok := effect.Req.(ProfileRequest)
	if !ok || req.Handle != "lzd" {
		t.Fatalf("open request is %#v, want ProfileRequest{lzd}", effect.Req)
	}
	if _, have := p.Header(); have {
		t.Fatal("a fresh profile already has a header")
	}
	if p.Walk().Loaded() != 0 {
		t.Fatal("a fresh profile already has posts")
	}
}

func TestProfileHeaderFetchedIssuesUserPostsByNumericID(t *testing.T) {
	p, _ := Open("lzd")
	header := domain.Profile{User: domain.User{ID: "42", ScreenName: "lzd", Name: "Maria"}}
	p, effect := p.Update(HeaderFetched{Header: header}, profileNow)
	if effect.Kind != EffectPage {
		t.Fatalf("after header effect is %v, want page", effect.Kind)
	}
	req, ok := effect.Req.(UserPostsRequest)
	if !ok || req.ID != "42" || req.Cursor != "" {
		t.Fatalf("posts request is %#v, want UserPostsRequest{42,\"\"}", effect.Req)
	}
	got, have := p.Header()
	if !have || got.User.ID != "42" {
		t.Fatalf("header not stored: have=%v got=%+v", have, got)
	}
}

func TestProfileHeaderFailedSpendsNoPostsRequest(t *testing.T) {
	p, _ := Open("lzd")
	p, effect := p.Update(HeaderFailed{Failure: domain.Failure{Kind: domain.FailureUnclassified}}, profileNow)
	if effect.Kind != EffectNone {
		t.Fatalf("failed header issued %v, want nothing", effect.Kind)
	}
	if p.HeaderFailure().Kind != domain.FailureUnclassified {
		t.Fatalf("header failure not recorded: %+v", p.HeaderFailure())
	}
	if _, have := p.Header(); have {
		t.Fatal("a failed header must not mark the header as present")
	}
}

func TestProfileRefreshIsTheTwoRequestPair(t *testing.T) {
	p, _ := Open("lzd")
	header := domain.Profile{User: domain.User{ID: "42", ScreenName: "lzd"}}
	p, _ = p.Update(HeaderFetched{Header: header}, profileNow)
	page := domain.PostPage{
		Posts: []domain.Post{{ID: "1", Author: header.User, Text: "one"}},
		Next:  "cursor-1",
	}
	p, _ = p.Update(PostsFetched{Page: page}, profileNow)

	p, effect := p.Update(ProfileRefreshRequested{}, profileNow)
	if effect.Kind != EffectRefresh {
		t.Fatalf("refresh effect is %v, want refresh", effect.Kind)
	}
	if _, ok := effect.Req.(ProfileRequest); !ok {
		t.Fatalf("refresh request is %#v, want ProfileRequest", effect.Req)
	}
	if !p.Refreshing() {
		t.Fatal("refreshing flag not set")
	}
	if _, again := p.Update(ProfileRefreshRequested{}, profileNow); again.Kind != EffectNone {
		t.Fatalf("second refresh while in flight issued %v", again.Kind)
	}

	p, effect = p.Update(HeaderFetched{Header: header}, profileNow)
	req := effect.Req.(UserPostsRequest)
	if req.ID != "42" {
		t.Fatalf("refresh posts id=%q", req.ID)
	}
	newer := domain.PostPage{
		Posts: []domain.Post{
			{ID: "0", Author: header.User, Text: "newer"},
			{ID: "1", Author: header.User, Text: "one"},
		},
	}
	p, _ = p.Update(PostsRefreshed{Page: newer}, profileNow)
	if p.Walk().NewRows() != 1 {
		t.Fatalf("new rows=%d, want 1", p.Walk().NewRows())
	}
	if p.Refreshing() {
		t.Fatal("still refreshing after posts answer")
	}
}
