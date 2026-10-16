package media

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"testing"
)

type clipCall struct {
	name  string
	args  []string
	stdin string
}

type clipRecorder struct {
	mu    sync.Mutex
	calls []clipCall
	err   error
}

func (r *clipRecorder) Run(ctx context.Context, name string, args []string, stdin string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, clipCall{
		name:  name,
		args:  append([]string(nil), args...),
		stdin: stdin,
	})
	return r.err
}

func (r *clipRecorder) seen() []clipCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]clipCall, len(r.calls))
	copy(out, r.calls)
	return out
}

func TestClipboardOutcomeWords(t *testing.T) {
	for _, tc := range []struct {
		o    ClipboardOutcome
		want string
	}{
		{ClipboardNone, "none"},
		{Copied, "copied"},
		{CopiedWithWLCopy, "copied with wl-copy"},
		{NoClipboard, "no clipboard"},
		{ClipboardOutcome(99), "clipboard outcome"},
	} {
		if got := tc.o.String(); got != tc.want {
			t.Errorf("%v.String() = %q, want %q", tc.o, got, tc.want)
		}
	}
}

func TestCopyOSC52UnderColourTERM(t *testing.T) {
	const permalink = "https://x.com/aergon/status/1812041"
	var buf bytes.Buffer
	c := Clipboard{
		Out: &buf,
		Env: func(k string) string {
			if k == "TERM" {
				return "xterm-256color"
			}
			return ""
		},
	}
	if got := c.Copy(context.Background(), permalink); got != Copied {
		t.Fatalf("outcome = %v, want Copied", got)
	}
	raw := buf.String()
	if strings.Count(raw, "\x1b]52;c;") != 1 {
		t.Fatalf("want exactly one OSC 52 write, got %q", raw)
	}
	if !strings.HasPrefix(raw, "\x1b]52;c;") || !strings.HasSuffix(raw, "\x1b\\") {
		t.Fatalf("escape shape: %q", raw)
	}
	payload := strings.TrimSuffix(strings.TrimPrefix(raw, "\x1b]52;c;"), "\x1b\\")
	dec, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("base64: %v", err)
	}
	if string(dec) != permalink {
		t.Fatalf("decoded %q, want permalink only", dec)
	}
}

func TestCopyNoWriteUnderDumbOrEmptyTERM(t *testing.T) {
	const permalink = "https://x.com/i/status/1812041"
	for _, term := range []string{"dumb", ""} {
		var buf bytes.Buffer
		c := Clipboard{
			Out: &buf,
			Env: func(k string) string {
				if k == "TERM" {
					return term
				}
				return ""
			},
		}
		if got := c.Copy(context.Background(), permalink); got != NoClipboard {
			t.Fatalf("TERM=%q: outcome = %v, want NoClipboard", term, got)
		}
		if buf.Len() != 0 {
			t.Fatalf("TERM=%q: wrote %q", term, buf.String())
		}
	}
}

func TestCopyWithWLCopyBothLinks(t *testing.T) {
	const permalink = "https://x.com/aergon/status/9"
	var buf bytes.Buffer
	rec := &clipRecorder{}
	c := Clipboard{
		Out: &buf,
		Env: func(k string) string {
			switch k {
			case "TERM":
				return "xterm-256color"
			case "WAYLAND_DISPLAY":
				return "wayland-0"
			}
			return ""
		},
		LookPath: func(name string) (string, error) {
			if name == "wl-copy" {
				return "/usr/bin/wl-copy", nil
			}
			return "", errors.New("not found")
		},
		Run: rec.Run,
	}
	if got := c.Copy(context.Background(), permalink); got != CopiedWithWLCopy {
		t.Fatalf("outcome = %v, want CopiedWithWLCopy", got)
	}
	if !strings.Contains(buf.String(), "\x1b]52;c;") {
		t.Fatal("OSC 52 was not written alongside wl-copy")
	}
	got := rec.seen()
	if len(got) != 1 || got[0].name != "wl-copy" || got[0].stdin != permalink {
		t.Fatalf("wl-copy calls = %#v", got)
	}
	if len(got[0].args) != 0 {
		t.Fatalf("wl-copy argv = %#v, want empty (URL on stdin)", got[0].args)
	}
}

func TestCopyWLCopyNonZeroIsEscapeOnly(t *testing.T) {
	const permalink = "https://x.com/aergon/status/9"
	var buf bytes.Buffer
	rec := &clipRecorder{err: errors.New("exit 1")}
	c := Clipboard{
		Out: &buf,
		Env: func(k string) string {
			switch k {
			case "TERM":
				return "xterm-256color"
			case "WAYLAND_DISPLAY":
				return "wayland-0"
			}
			return ""
		},
		LookPath: func(string) (string, error) { return "/usr/bin/wl-copy", nil },
		Run:      rec.Run,
	}
	if got := c.Copy(context.Background(), permalink); got != Copied {
		t.Fatalf("outcome = %v, want Copied", got)
	}
	if len(rec.seen()) != 1 {
		t.Fatal("wl-copy should still have been attempted")
	}
}

func TestCopyWLCopyUnresolvableIsEscapeOnly(t *testing.T) {
	const permalink = "https://x.com/aergon/status/9"
	var buf bytes.Buffer
	started := false
	c := Clipboard{
		Out: &buf,
		Env: func(k string) string {
			switch k {
			case "TERM":
				return "xterm-256color"
			case "WAYLAND_DISPLAY":
				return "wayland-0"
			}
			return ""
		},
		LookPath: func(string) (string, error) { return "", errors.New("not found") },
		Run: func(context.Context, string, []string, string) error {
			started = true
			return nil
		},
	}
	if got := c.Copy(context.Background(), permalink); got != Copied {
		t.Fatalf("outcome = %v, want Copied", got)
	}
	if started {
		t.Fatal("Run must not be called when wl-copy is missing")
	}
}

func TestCopyNeitherLinkIsNoClipboard(t *testing.T) {
	const permalink = "https://x.com/i/status/1"
	var buf bytes.Buffer
	c := Clipboard{
		Out: &buf,
		Env: func(k string) string {
			if k == "TERM" {
				return "dumb"
			}
			return ""
		},
	}
	if got := c.Copy(context.Background(), permalink); got != NoClipboard {
		t.Fatalf("outcome = %v, want NoClipboard", got)
	}
	if buf.Len() != 0 {
		t.Fatalf("wrote %q", buf.String())
	}
}
func TestCopyWLCopyOnlyUnderDumbTERM(t *testing.T) {
	const permalink = "https://x.com/aergon/status/9"
	for _, term := range []string{"dumb", ""} {
		var buf bytes.Buffer
		rec := &clipRecorder{}
		c := Clipboard{
			Out: &buf,
			Env: func(k string) string {
				switch k {
				case "TERM":
					return term
				case "WAYLAND_DISPLAY":
					return "wayland-0"
				}
				return ""
			},
			LookPath: func(name string) (string, error) {
				if name == "wl-copy" {
					return "/usr/bin/wl-copy", nil
				}
				return "", errors.New("not found")
			},
			Run: rec.Run,
		}
		if got := c.Copy(context.Background(), permalink); got != Copied {
			t.Fatalf("TERM=%q: outcome = %v, want Copied (wl-only success)", term, got)
		}
		if buf.Len() != 0 {
			t.Fatalf("TERM=%q: wrote OSC %q", term, buf.String())
		}
		got := rec.seen()
		if len(got) != 1 || got[0].name != "wl-copy" || got[0].stdin != permalink {
			t.Fatalf("TERM=%q: wl-copy calls = %#v", term, got)
		}
	}
}

func TestCopyPayloadIsPermalinkOnly(t *testing.T) {
	const permalink = "https://x.com/aergon/status/1812041"
	var buf bytes.Buffer
	rec := &clipRecorder{}
	c := Clipboard{
		Out: &buf,
		Env: func(k string) string {
			switch k {
			case "TERM":
				return "xterm-256color"
			case "WAYLAND_DISPLAY":
				return "wayland-0"
			}
			return ""
		},
		LookPath: func(string) (string, error) { return "/bin/wl-copy", nil },
		Run:      rec.Run,
	}
	_ = c.Copy(context.Background(), permalink)
	raw := buf.String()
	payload := strings.TrimSuffix(strings.TrimPrefix(raw, "\x1b]52;c;"), "\x1b\\")
	dec, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatal(err)
	}
	if string(dec) != permalink {
		t.Fatalf("escape carried %q", dec)
	}
	if got := rec.seen(); len(got) != 1 || got[0].stdin != permalink {
		t.Fatalf("stdin = %#v", got)
	}
}
