package domain

import (
	"reflect"
	"strings"
	"testing"
)

func TestSavedKindsAreTheClosedTable(t *testing.T) {
	kinds := []SavedKind{Bookmark, Like}
	if len(kinds) != int(Like)+1 {
		t.Fatalf("the table holds %d kinds and the last constant is %d: a kind was added or removed without a row here", len(kinds), Like)
	}
	seen := map[SavedKind]bool{}
	for i, kind := range kinds {
		if int(kind) != i {
			t.Errorf("kinds[%d] = %d (%s): index must equal value so an inserted constant renumbers and fails here", i, kind, kind)
		}
		if seen[kind] {
			t.Errorf("kind %d (%s) appears twice", kind, kind)
		}
		seen[kind] = true
	}
	if Bookmark != 0 {
		t.Errorf("Bookmark = %d, want 0: it is the list this phase can always reach", Bookmark)
	}
	if !Bookmark.Valid() || !Like.Valid() {
		t.Errorf("Bookmark.Valid()=%v Like.Valid()=%v, want both true", Bookmark.Valid(), Like.Valid())
	}
	if SavedKind(7).Valid() {
		t.Error("SavedKind(7).Valid() = true, want false")
	}
	if got := Bookmark.String(); got != "bookmark" {
		t.Errorf("Bookmark.String() = %q, want bookmark", got)
	}
	if got := Like.String(); got != "like" {
		t.Errorf("Like.String() = %q, want like", got)
	}
	if got := Bookmark.List(); got != "bookmarks" {
		t.Errorf("Bookmark.List() = %q, want bookmarks", got)
	}
	if got := Like.List(); got != "likes" {
		t.Errorf("Like.List() = %q, want likes", got)
	}
	if got := SavedKind(9).String(); !strings.Contains(got, "9") {
		t.Errorf("unknown SavedKind prints %q, want the number it is", got)
	}
}

func TestListKindsAreTheClosedTable(t *testing.T) {
	kinds := []ListKind{ListFollowing, ListForYou, ListBookmarks, ListLikes}
	if len(kinds) != int(ListLikes)+1 {
		t.Fatalf("the table holds %d kinds and the last constant is %d", len(kinds), ListLikes)
	}
	seen := map[ListKind]bool{}
	want := []string{"following", "for you", "bookmarks", "likes"}
	for i, kind := range kinds {
		if int(kind) != i {
			t.Errorf("kinds[%d] = %d (%s): index must equal value", i, kind, kind)
		}
		if seen[kind] {
			t.Errorf("kind %d appears twice", kind)
		}
		seen[kind] = true
		if got := kind.String(); got != want[i] {
			t.Errorf("%s.String() = %q, want %q", kind, got, want[i])
		}
	}
	if ListFollowing != 0 {
		t.Fatalf("ListFollowing = %d, want 0", ListFollowing)
	}
}

func TestListKindRoundTripsHomeAndSaved(t *testing.T) {
	if got := HomeList(Following); got != ListFollowing {
		t.Errorf("HomeList(Following) = %v, want ListFollowing", got)
	}
	if got := HomeList(ForYou); got != ListForYou {
		t.Fatalf("HomeList(ForYou) = %v, want ListForYou", got)
	}
	if got := SavedList(Bookmark); got != ListBookmarks {
		t.Fatalf("SavedList(Bookmark) = %v, want ListBookmarks", got)
	}
	if got := SavedList(Like); got != ListLikes {
		t.Fatalf("SavedList(Like) = %v, want ListLikes", got)
	}

	for _, k := range []HomeKind{Following, ForYou} {
		home, ok := HomeList(k).Home()
		if !ok || home != k {
			t.Errorf("HomeList(%v).Home() = (%v, %v), want (%v, true)", k, home, ok, k)
		}
		if _, ok := HomeList(k).Saved(); ok {
			t.Errorf("HomeList(%v).Saved() reported a saved kind", k)
		}
	}
	for _, k := range []SavedKind{Bookmark, Like} {
		saved, ok := SavedList(k).Saved()
		if !ok || saved != k {
			t.Errorf("SavedList(%v).Saved() = (%v, %v), want (%v, true)", k, saved, ok, k)
		}
		if _, ok := SavedList(k).Home(); ok {
			t.Errorf("SavedList(%v).Home() reported a home kind", k)
		}
	}
}

func TestStopReasonsAreTheClosedTable(t *testing.T) {
	reasons := []StopReason{
		StopNone,
		StopNoUnknownIDs,
		StopEndOfList,
		StopPageCap,
		StopFailure,
		StopAbandoned,
		StopStalled,
	}
	if len(reasons) != int(StopStalled)+1 {
		t.Fatalf("the table holds %d reasons and the last constant is %d", len(reasons), StopStalled)
	}
	seen := map[StopReason]bool{}
	want := []string{"", "no unknown ids", "end of list", "page cap", "failure", "abandoned", "cursor stalled"}
	for i, r := range reasons {
		if int(r) != i {
			t.Errorf("reasons[%d] = %d (%s): index must equal value", i, r, r)
		}
		if seen[r] {
			t.Errorf("reason %d appears twice", r)
		}
		seen[r] = true
		if got := r.String(); got != want[i] {
			t.Errorf("%v.String() = %q, want %q", r, got, want[i])
		}
	}
	if StopNone != 0 {
		t.Fatalf("StopNone = %d, want 0", StopNone)
	}
}

func TestCapabilityZeroIsNone(t *testing.T) {
	if CapNone != 0 {
		t.Fatalf("CapNone = %d, want 0", CapNone)
	}
	caps := []Capability{CapNone, CapOperationMissing, CapWrongAccount, CapMirrorUnreadable}
	if len(caps) != int(CapMirrorUnreadable)+1 {
		t.Fatalf("capability table length %d != last+1 %d", len(caps), CapMirrorUnreadable+1)
	}
	want := []string{"", "operation missing", "wrong account", "mirror unreadable"}
	for i, c := range caps {
		if int(c) != i {
			t.Errorf("caps[%d] = %d", i, c)
		}
		if got := c.String(); got != want[i] {
			t.Errorf("%v.String() = %q, want %q", c, got, want[i])
		}
	}
}

func TestMirrorFactsArePresenceOnly(t *testing.T) {
	rt := reflect.TypeOf(MirrorFacts{})
	forbidden := []string{"Body", "JSON", "Post", "Text", "Content", "Row"}
	for i := 0; i < rt.NumField(); i++ {
		name := rt.Field(i).Name
		for _, bad := range forbidden {
			if strings.Contains(name, bad) {
				t.Errorf("MirrorFacts field %q looks like row/body contents", name)
			}
		}
	}
	want := []string{"Path", "Exists", "SchemaVersion", "Newer", "Owner", "Kinds", "LastRun", "Stop"}
	if rt.NumField() != len(want) {
		t.Fatalf("MirrorFacts has %d fields, want %v", rt.NumField(), want)
	}
	for i, name := range want {
		if rt.Field(i).Name != name {
			t.Errorf("field %d = %s, want %s", i, rt.Field(i).Name, name)
		}
	}
}
