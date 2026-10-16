package text

import (
	"slices"
	"strings"
	"testing"
)

// zwjFamily is one grapheme cluster drawn as three wide code points joined by zero-width joiners:
// the string that tells the two width semantics apart.
const zwjFamily = "\U0001F468\u200D\U0001F469\u200D\U0001F467"

// tmuxValue is the shape $TMUX has inside a pane (socket path, window and pane id). Nothing reads
// its value: any non-empty $TMUX means a multiplexer's grid.
const tmuxValue = "/tmp/tmux-1000/default,4242,0"

// envArms is the two environments the measure dispatches between, so that every
// environment-dependent expectation is written once per arm and both arms run in one test run.
func envArms() []struct {
	name string
	tmux string
} {
	return []struct {
		name string
		tmux string
	}{
		{"bare terminal", ""},
		{"tmux pane", tmuxValue},
	}
}

func TestWidth(t *testing.T) {
	cases := []struct {
		s    string
		want int
	}{
		{"", 0},
		{"hello", 5},
		{"日本語のテキスト", 16}, // eight wide characters, sixteen cells in either environment
		{"ab日", 4},
	}
	for _, env := range envArms() {
		t.Run(env.name, func(t *testing.T) {
			t.Setenv("TMUX", env.tmux)
			for _, c := range cases {
				if got := Width(c.s); got != c.want {
					t.Errorf("Width(%q) = %d, want %d", c.s, got, c.want)
				}
			}
		})
	}
}

// TestWidthDispatchesOnTMUX asserts both halves of the dispatch in one run: the same string is
// two cells in a bare terminal and six under a multiplexer, and the two do not coincide.
func TestWidthDispatchesOnTMUX(t *testing.T) {
	t.Setenv("TMUX", "")
	bare := Width(zwjFamily)
	t.Setenv("TMUX", tmuxValue)
	multiplexed := Width(zwjFamily)
	if bare != 2 {
		t.Errorf("Width(%q) with TMUX unset = %d, want 2: one grapheme cluster", zwjFamily, bare)
	}
	if multiplexed != 6 {
		t.Errorf("Width(%q) with TMUX set = %d, want 6: three wide code points", zwjFamily, multiplexed)
	}
	if bare == multiplexed {
		t.Errorf("Width(%q) = %d in both environments: the dispatch is not live", zwjFamily, bare)
	}
}

func TestPad(t *testing.T) {
	cases := []struct {
		s    string
		w    int
		want string
	}{
		{"ab", 5, "ab   "},
		{"abcde", 5, "abcde"},
		{"", 3, "   "},
		{"abcdef", 3, "abc"},
		{"ab日", 3, "ab "}, // 日 is two cells wide, so the cut lands a cell short and the fill closes it
		{"日", 1, " "},     // nothing of s fits in one cell
		{"ab", 0, ""},
		{"ab", -1, ""},
	}
	for _, env := range envArms() {
		t.Run(env.name, func(t *testing.T) {
			t.Setenv("TMUX", env.tmux)
			for _, c := range cases {
				if got := Pad(c.s, c.w); got != c.want {
					t.Errorf("Pad(%q, %d) = %q, want %q", c.s, c.w, got, c.want)
				}
			}
			// A field wider than its column is cut, not spilled: the content is exactly what
			// Truncate leaves, and the result reports the width that was asked for.
			if got, want := Pad("abcdef", 3), Truncate("abcdef", 3); got != want {
				t.Errorf("Pad(%q, 3) = %q, want Truncate(%q, 3) = %q", "abcdef", got, "abcdef", want)
			}
			for _, c := range cases {
				if got := Width(Pad(c.s, c.w)); got != max(c.w, 0) {
					t.Errorf("Width(Pad(%q, %d)) = %d, want exactly %d", c.s, c.w, got, max(c.w, 0))
				}
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		s    string
		w    int
		want string
	}{
		{"abcdef", 3, "abc"},
		{"abcdef", 6, "abcdef"},
		{"abcdef", 7, "abcdef"},
		{"ab日本", 4, "ab日"}, // the next cluster would make six
		{"ab日本", 5, "ab日"}, // a cluster is never half cut: the odd cell stays empty
		{"日本語のテキスト", 16, "日本語のテキスト"},
		{"abcdef", 0, ""},
		{"abcdef", -1, ""},
		{"", 4, ""},
		{"日", 1, ""},
	}
	for _, env := range envArms() {
		t.Run(env.name, func(t *testing.T) {
			t.Setenv("TMUX", env.tmux)
			for _, c := range cases {
				if got := Truncate(c.s, c.w); got != c.want {
					t.Errorf("Truncate(%q, %d) = %q, want %q", c.s, c.w, got, c.want)
				}
			}
		})
	}
}

// TestTruncateCutsAtAClusterBoundary checks the cut against the bytes of the input rather than
// against a number the package computed: the result has to be a prefix of s that ends exactly
// where the cluster it could not fit begins.
func TestTruncateCutsAtAClusterBoundary(t *testing.T) {
	const s = "ab日本" // six bytes: "ab", then two three-byte clusters
	for _, env := range envArms() {
		t.Run(env.name, func(t *testing.T) {
			t.Setenv("TMUX", env.tmux)
			got := Truncate(s, 5)
			if !strings.HasPrefix(s, got) {
				t.Fatalf("Truncate(%q, 5) = %q, which is not a prefix of s", s, got)
			}
			if got != "ab日" {
				t.Errorf("Truncate(%q, 5) = %q, want %q", s, got, "ab日")
			}
			if len(got) != 5 {
				t.Errorf("Truncate(%q, 5) = %q, %d bytes, want 5: the cut is the byte where %q starts", s, got, len(got), "本")
			}
		})
	}
}

func TestWrap(t *testing.T) {
	cases := []struct {
		name string
		s    string
		w    int
		want []string
	}{
		{"greedy on spaces", "one two three", 7, []string{"one two", "three"}},
		{"one token per line", "aaa bbb", 3, []string{"aaa", "bbb"}},
		{"token wider than the measure", "aaa bbbbbbbbbbbb ccc", 8, []string{"aaa bbbb", "bbbbbbbb", "ccc"}},
		{"measure of one", "ab cd", 1, []string{"a", "b", "c", "d"}},
		{"empty input", "", 8, nil},
		{"run of spaces", "   ", 8, nil},
		{"newline separates like a space", "one\ntwo", 7, []string{"one two"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Wrap(c.s, c.w); !slices.Equal(got, c.want) {
				t.Errorf("Wrap(%q, %d) = %q, want %q", c.s, c.w, got, c.want)
			}
		})
	}
	// w < 1 is treated as 1, not as nothing.
	if got, want := Wrap("ab cd", 0), Wrap("ab cd", 1); !slices.Equal(got, want) {
		t.Errorf("Wrap(%q, 0) = %q, want Wrap(%q, 1) = %q", "ab cd", got, "ab cd", want)
	}
}
