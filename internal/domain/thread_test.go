package domain_test

import (
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/xsource"
)

// The port is declared by its consumer and implemented at the composition root, so the one thing
// no single package can check on its own is that the two still fit: a signature that moves in
// internal/xsource and not here has to fail a test rather than a screen run. The assertion is a
// package-level declaration, so it is compiled whenever these tests are — and the test below
// exercises the same assignment in a function, where the failure names what moved.
var _ domain.Threads = (*xsource.Adapter)(nil)

// TestTheAdapterSatisfiesTheThreadsPort is the seam from the consumer's side. A nil transport is
// enough: this is about the method set, not about a request.
func TestTheAdapterSatisfiesTheThreadsPort(t *testing.T) {
	var threads domain.Threads = xsource.New(nil)
	if threads == nil {
		t.Fatal("the adapter did not satisfy domain.Threads")
	}
}

// TestGroupKindNamesItself pins the two words and the closed set: the words are what a log line
// and a test name read, and a third kind added without a word would print a number instead.
func TestGroupKindNamesItself(t *testing.T) {
	if got := domain.SelfThread.String(); got != "self-thread" {
		t.Errorf("SelfThread prints as %q, want self-thread", got)
	}
	if got := domain.Replies.String(); got != "replies" {
		t.Errorf("Replies prints as %q, want replies", got)
	}
	if domain.SelfThread != 0 {
		t.Errorf("SelfThread is %d, want the zero value: the continuation is the narrower claim", domain.SelfThread)
	}
	if got := domain.GroupKind(9).String(); !strings.Contains(got, "9") {
		t.Errorf("an unknown kind prints as %q, want a name carrying its number rather than an empty word", got)
	}
}

// TestThreadPageZeroValueIsNothingReadYet pins the zero value: a pane that has not read yet has to
// be distinguishable from one that read an empty conversation, which is the difference the whole
// phase's honesty rests on.
func TestThreadPageZeroValueIsNothingReadYet(t *testing.T) {
	var page domain.ThreadPage
	switch {
	case page.Focal.ID != "":
		t.Errorf("the zero page carries the focal post %q", page.Focal.ID)
	case len(page.Ancestors) != 0:
		t.Errorf("the zero page carries %d ancestors", len(page.Ancestors))
	case len(page.Groups) != 0:
		t.Errorf("the zero page carries %d groups", len(page.Groups))
	case page.Next != "":
		t.Errorf("the zero page carries the next cursor %q", page.Next)
	case page.Requested != 0 || page.Skipped != 0:
		t.Errorf("the zero page reports Requested %d and Skipped %d, want both zero", page.Requested, page.Skipped)
	case !page.FetchedAt.IsZero():
		t.Errorf("the zero page is stamped %s, want no clock read at all", page.FetchedAt)
	}

	// A group's zero value is a complete, empty block, and its kind is the continuation: the zero
	// value of a kind enum is a choice, and this one is the narrower claim of the two, spelled the
	// way Following is HomeKind's zero value. The walker sets the kind explicitly either way.
	var group domain.Group
	if group.Kind != domain.SelfThread {
		t.Errorf("a group's zero kind is %q, want SelfThread: it is the zero value of the enum", group.Kind)
	}
	if group.More != "" {
		t.Errorf("a group's zero handle is %q, want empty: an unspent handle is something the answer says", group.More)
	}
}
