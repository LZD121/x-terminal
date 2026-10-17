package render

import (
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
)

// TestVividIsADeclaredException is the exception written down as a test rather than as a comment.
//
// The plain reader, the four covers and the panic frame carry no product vocabulary and no real
// identity: that is what makes them a disguise, and the corpus tests hold every one of their frames
// to it. The vivid presentation is worn where no disguise is needed, so it prints who actually wrote
// a post — and it is the only grammar this build ships that may.
//
// The exception is narrow in the way that matters: it is a different grammar, not a flag on the plain
// one, so no plain or covered frame can acquire the exception by accident.
func TestVividIsADeclaredException(t *testing.T) {
	user := domain.User{ScreenName: "someone", Name: "Some One"}

	// The plain reader prints the identity too: it is not a disguise, it is the reader. The covers
	// are the disguise, and every one of them aliases it.
	if got := Plain.Identity(user); got.Handle != user.ScreenName || got.Name != user.Name {
		t.Fatalf("the plain reader hides the identity: %+v", got)
	}
	for _, style := range CoverStyles() {
		if got := style.Grammar().Identity(user); got.Handle == user.ScreenName || got.Name == user.Name {
			t.Fatalf("%s no longer hides the identity: %+v", style, got)
		}
	}

	// The vivid grammar prints it, and that is the whole of the exception.
	got := Vivid.Identity(user)
	if got.Handle != user.ScreenName || got.Name != user.Name {
		t.Fatalf("the vivid grammar hides the identity: %+v", got)
	}
	if Vivid.Name == Plain.Name {
		t.Fatal("the vivid grammar has the plain reader's own name")
	}
	// It is still a grammar, not a cover: nothing here renames an author to a machine's word.
	if strings.Contains(strings.ToLower(Vivid.Name), "cover") {
		t.Fatalf("the vivid grammar is named like a cover: %q", Vivid.Name)
	}
}

// TestTheVividGrammarKeepsThePlainMeasure is the one field a careless edit would get wrong: a
// MeasureCap of zero would make every body line zero cells wide, and the presentation would render
// nothing. It is asserted here rather than only in the vivid package's goldens so a change to this
// value fails in the package that holds it.
func TestTheVividGrammarKeepsThePlainMeasure(t *testing.T) {
	if Vivid.MeasureCap < 1 {
		t.Fatalf("MeasureCap = %d: a cap below one makes every body line zero cells", Vivid.MeasureCap)
	}
	if Vivid.Separator {
		t.Fatal("the vivid grammar keeps the plain reader's separator row: the list is dense by design")
	}
}
