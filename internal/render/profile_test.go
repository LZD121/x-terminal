package render

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

func profileFixture() domain.Profile {
	followers, following, posts := 1204, 312, 8901
	return domain.Profile{
		User:      domain.User{ID: "42", ScreenName: "lzd", Name: "Maria"},
		Bio:       "writing about terminals, long-running jobs, and the page cache.",
		Followers: &followers,
		Following: &following,
		Posts:     &posts,
		Joined:    time.Date(2011, time.March, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestProfileHeaderFullBareBioFacts(t *testing.T) {
	plainEnv(t)
	full := ProfileHeader(profileFixture(), Plain, 80)
	if len(full.Rows) < 3 {
		t.Fatalf("full header has %d rows, want identity+bio+facts", len(full.Rows))
	}
	if full.PostID != "" {
		t.Fatal("header block must not be selectable")
	}
	identity := ansi.Strip(full.Rows[0])
	if !strings.Contains(identity, "lzd") || !strings.Contains(identity, "Maria") {
		t.Fatalf("identity row %q", identity)
	}
	facts := ansi.Strip(full.Rows[len(full.Rows)-1])
	if !strings.Contains(facts, "followers: 1204") || !strings.Contains(facts, "joined: Mar 2011") {
		t.Fatalf("facts row %q", facts)
	}

	bare := ProfileHeader(domain.Profile{User: domain.User{ScreenName: "lzd", Name: "Maria"}}, Plain, 80)
	if len(bare.Rows) != 1 {
		t.Fatalf("bare header has %d rows, want 1", len(bare.Rows))
	}
	if strings.Contains(ansi.Strip(bare.Rows[0]), "0") {
		t.Fatalf("bare header carries a zeroed count: %q", ansi.Strip(bare.Rows[0]))
	}

	bioOnly := ProfileHeader(domain.Profile{
		User: domain.User{ScreenName: "lzd", Name: "Maria"},
		Bio:  "just a bio",
	}, Plain, 80)
	if len(bioOnly.Rows) < 2 {
		t.Fatalf("bio-only header has %d rows", len(bioOnly.Rows))
	}

	zero := 0
	factsOnly := ProfileHeader(domain.Profile{
		User:      domain.User{ScreenName: "lzd", Name: "Maria"},
		Followers: &zero,
		Joined:    time.Date(2011, time.March, 1, 0, 0, 0, 0, time.UTC),
	}, Plain, 80)
	if len(factsOnly.Rows) != 2 {
		t.Fatalf("facts-only header has %d rows, want 2", len(factsOnly.Rows))
	}
	if !strings.Contains(ansi.Strip(factsOnly.Rows[1]), "followers: 0") {
		t.Fatalf("explicit zero must print: %q", ansi.Strip(factsOnly.Rows[1]))
	}
}

func TestProfileHeaderBioWrapsFully(t *testing.T) {
	plainEnv(t)
	bio := strings.Repeat("a", 5000)
	p := domain.Profile{User: domain.User{ScreenName: "lzd", Name: "Maria"}, Bio: bio}
	block := ProfileHeader(p, Plain, 80)
	var cells int
	for _, row := range block.Rows[1:] {
		line := strings.TrimSpace(ansi.Strip(row))
		cells += text.Width(line)
	}
	if cells < 5000 {
		t.Fatalf("bio cells printed=%d, want all 5000", cells)
	}
}

func TestProfileSeparatesHeaderFromPosts(t *testing.T) {
	plainEnv(t)
	p := profileFixture()
	posts := []domain.Post{{
		ID: "1", Author: p.User, CreatedAt: specNow, Text: "a post",
	}}
	blocks := Profile(p, posts, Plain, 80, Selection{ID: "1"}, specNow)
	if len(blocks) != 2 {
		t.Fatalf("blocks=%d, want header + one post", len(blocks))
	}
	if len(blocks[1].Rows) == 0 || blocks[1].Rows[0] != "" {
		t.Fatalf("first post block must start with the blank separator: %#v", blocks[1].Rows)
	}
}
