package xsource

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

func TestParseProfileReadsBioCountsAndJoined(t *testing.T) {
	raw, err := os.ReadFile("testdata/fixture-profile-header.json")
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseProfile(raw)
	if err != nil {
		t.Fatalf("ParseProfile: %v", err)
	}
	if p.User.ID != "783214" || p.User.ScreenName != "X" {
		t.Fatalf("identity is %+v, want rest_id 783214 / X", p.User)
	}
	if p.Bio == "" {
		t.Fatal("bio is empty")
	}
	if p.Followers == nil || *p.Followers == 0 {
		t.Fatalf("followers absent or zero: %v", p.Followers)
	}
	if p.Following == nil {
		t.Fatal("following absent")
	}
	if p.Posts == nil || *p.Posts == 0 {
		t.Fatalf("posts absent or zero: %v", p.Posts)
	}
	wantJoined := time.Date(2007, time.February, 20, 14, 35, 54, 0, time.UTC)
	if !p.Joined.Equal(wantJoined) {
		t.Fatalf("joined is %s, want %s", p.Joined, wantJoined)
	}
}

func TestParseProfileAbsentVersusZeroCounts(t *testing.T) {
	raw, err := os.ReadFile("testdata/fixture-profile-header.json")
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	node := envelope["data"].(map[string]any)["user"].(map[string]any)["result"].(map[string]any)

	delete(node, "relationship_counts")
	delete(node, "tweet_counts")
	delete(node, "profile_bio")
	absentBody, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	absent, err := ParseProfile(absentBody)
	if err != nil {
		t.Fatalf("absent: %v", err)
	}
	if absent.Followers != nil || absent.Following != nil || absent.Posts != nil {
		t.Fatalf("absent counts must be nil, got %+v", absent)
	}
	if absent.Bio != "" {
		t.Fatalf("absent bio must be empty, got %q", absent.Bio)
	}
	if absent.User.ScreenName != "X" {
		t.Fatalf("identity must survive: %+v", absent.User)
	}

	zero := 0
	node["relationship_counts"] = map[string]any{"followers": zero, "following": zero}
	node["tweet_counts"] = map[string]any{"tweets": zero}
	zeroBody, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseProfile(zeroBody)
	if err != nil {
		t.Fatalf("zero: %v", err)
	}
	if got.Followers == nil || *got.Followers != 0 {
		t.Fatalf("followers want &0, got %v", got.Followers)
	}
	if got.Following == nil || *got.Following != 0 {
		t.Fatalf("following want &0, got %v", got.Following)
	}
	if got.Posts == nil || *got.Posts != 0 {
		t.Fatalf("posts want &0, got %v", got.Posts)
	}
}

func TestAdapterProfileAndUserPostsAskByName(t *testing.T) {
	header, err := os.ReadFile("testdata/fixture-profile-header.json")
	if err != nil {
		t.Fatal(err)
	}
	posts, err := os.ReadFile("testdata/fixture-profile-posts.json")
	if err != nil {
		t.Fatal(err)
	}

	client := &recordingClient{body: header}
	adapter := New(client)
	if _, err := adapter.Profile(context.Background(), "X"); err != nil {
		t.Fatalf("Profile: %v", err)
	}
	if client.op != opUser {
		t.Errorf("Profile asked for %q, want %q", client.op, opUser)
	}
	if got := client.vars["screen_name"]; got != "X" {
		t.Errorf("screen_name=%v, want X", got)
	}

	client = &recordingClient{body: posts}
	adapter = New(client)
	if _, err := adapter.UserPosts(context.Background(), "783214", "cursor-1"); err != nil {
		t.Fatalf("UserPosts: %v", err)
	}
	if client.op != opUserTweets {
		t.Errorf("UserPosts asked for %q, want %q", client.op, opUserTweets)
	}
	want := map[string]any{
		"userId": "783214",
		"count":  pageCount,
		"cursor": "cursor-1",
		// The two names the upstream's own 422 asked for on 2026-10-15, with the value the
		// measurement settled on: UserTweets answered 422 naming includePromotedContent and then
		// withVoice, and 200 once both were present. A later edit that drops one of them fails
		// here rather than on a live profile read.
		"includePromotedContent": false,
		"withVoice":              false,
	}
	for key, value := range want {
		if client.vars[key] != value {
			t.Errorf("%s=%v, want %v", key, client.vars[key], value)
		}
	}
	if got := len(client.vars); got != len(want) {
		t.Errorf("UserPosts sent %d variables (%v), want exactly the %d measured ones", got, client.vars, len(want))
	}
	if _, ok := client.vars["screen_name"]; ok {
		t.Error("UserPosts must not send a handle")
	}
}

func TestUserPostsParseEqualsParseTimeline(t *testing.T) {
	raw, err := os.ReadFile("testdata/fixture-profile-posts.json")
	if err != nil {
		t.Fatal(err)
	}
	want, err := ParseTimeline(raw)
	if err != nil {
		t.Fatal(err)
	}
	client := &recordingClient{body: raw}
	got, err := New(client).UserPosts(context.Background(), "783214", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Posts) != len(want.Posts) {
		t.Fatalf("posts %d, want %d", len(got.Posts), len(want.Posts))
	}
	for i := range want.Posts {
		if got.Posts[i].ID != want.Posts[i].ID || got.Posts[i].Text != want.Posts[i].Text {
			t.Fatalf("post %d differs: got %+v want %+v", i, got.Posts[i].ID, want.Posts[i].ID)
		}
	}
	if string(got.Next) != want.BottomCursor {
		t.Fatalf("next=%q, want %q", got.Next, want.BottomCursor)
	}
	_ = domain.Profile{}
}
