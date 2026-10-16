package render

import (
	"reflect"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
)

func TestPermalinkWithHandle(t *testing.T) {
	p := domain.Post{
		ID:     "1812041",
		Author: domain.User{ScreenName: "aergon"},
	}
	want := "https://x.com/aergon/status/1812041"
	if got := Permalink(p); got != want {
		t.Fatalf("Permalink = %q, want %q", got, want)
	}
}

func TestPermalinkStripsLeadingAt(t *testing.T) {
	p := domain.Post{
		ID:     "1812041",
		Author: domain.User{ScreenName: "@aergon"},
	}
	want := "https://x.com/aergon/status/1812041"
	if got := Permalink(p); got != want {
		t.Fatalf("Permalink = %q, want %q", got, want)
	}
}

func TestPermalinkNoHandle(t *testing.T) {
	p := domain.Post{ID: "1812041"}
	want := "https://x.com/i/status/1812041"
	if got := Permalink(p); got != want {
		t.Fatalf("Permalink = %q, want %q", got, want)
	}
}

func TestPermalinkNotStoredOnPost(t *testing.T) {
	// The template is presentation: domain.Post carries no Permalink field.
	var zero domain.Post
	tf := reflect.TypeOf(zero)
	for i := range tf.NumField() {
		if tf.Field(i).Name == "Permalink" {
			t.Fatal("domain.Post must not store a Permalink field")
		}
	}
}
