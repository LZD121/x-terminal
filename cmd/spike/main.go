// Command spike is a throwaway: it proves that one authenticated read from this host works end to
// end — configuration, transport, session, registry, the GraphQL request, the response shapes and
// the parsers — and prints what it read. It is deleted once Phase 2's screens replace it.
//
// It is deliberately not a subcommand of x-terminal. It reads the user's own account, it prints
// raw-ish post text, and it exists for one phase; a verb of the product would have to promise all
// three of those forever.
//
// What it does, in order: load the configuration and the session; call Viewer, which answers with
// the account the session belongs to; call UserTweets for that account's own id with count=20;
// print the viewer, one line per post, and the request budget this session spent. If any step
// fails it prints the classified outcome and exits non-zero — and prints nothing at all on stdout
// on the way out, because a blank timeline is exactly what this project refuses to show when the
// upstream said no.
//
// It never prints the cookie, and it makes exactly two requests: no retry, no loop, no refresh.
// With -record it also writes the two response bodies it read into a directory of the user's
// choosing, so that the parse layer can be pinned against a real body; those bodies carry whatever
// the account's own response carries, which is why nothing recorded this way is committed without
// being looked at first.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xsource"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// spikeTimeout bounds the whole run. Two requests: one measured at 0.38s and one at 0.81s. The rest
// of the budget is for a slow proxy, not for a retry.
const spikeTimeout = 90 * time.Second

// postsWanted is the count the plan asks UserTweets for.
const postsWanted = 20

// previewRunes is how much of a post's text one line of this output carries.
const previewRunes = 100

func main() {
	os.Exit(run(os.Stdout, os.Stderr, os.Args[1:]))
}

func run(stdout, stderr io.Writer, args []string) int {
	flags := flag.NewFlagSet("spike", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("record", "", "directory to write the response bodies into as they arrive (they are not to be committed unread)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "spike takes no arguments\n")
		return 2
	}

	started := time.Now()
	cfg, err := config.Load()
	if err != nil {
		return fail(stderr, err)
	}
	for _, warning := range cfg.Warnings {
		fmt.Fprintf(stderr, "config warning: %s\n", warning)
	}

	sess, err := session.Load(cfg.SessionPath)
	if err != nil {
		return fail(stderr, err)
	}
	client, err := xweb.NewClient(cfg)
	if err != nil {
		return fail(stderr, err)
	}
	api := client.WithSession(sess)

	ctx, cancel := context.WithTimeout(context.Background(), spikeTimeout)
	defer cancel()
	// Each body is written as it arrives rather than at the end: a run that fails at the second
	// call is exactly the run whose first answer is worth reading.
	record := recorder(*dir, stderr)

	viewerBody, err := api.GraphQL(ctx, "Viewer", nil)
	if err != nil {
		return fail(stderr, err)
	}
	record("viewer.json", viewerBody)

	viewer, err := xsource.ParseUser(viewerBody)
	if err != nil {
		return fail(stderr, err)
	}

	// The variables the two maintained clients send for this operation. userId and count alone were
	// enough when the probe drove curl on 2026-10-14, and the live run answered a 422 with an empty
	// body to exactly that pair — which is what a required-but-unstated Boolean looks like from the
	// outside, and why the shape is written out here rather than guessed at a third time.
	tweetsBody, err := api.GraphQL(ctx, "UserTweets", map[string]any{
		"userId":                                 viewer.ID,
		"count":                                  postsWanted,
		"includePromotedContent":                 false,
		"withQuickPromoteEligibilityTweetFields": true,
		"withVoice":                              true,
	})
	if err != nil {
		return fail(stderr, err)
	}
	record("tweets.json", tweetsBody)

	page, err := xsource.ParseTimeline(tweetsBody)
	if err != nil {
		return fail(stderr, err)
	}

	// Nothing above this line wrote to stdout, and nothing below it can fail: a read that did not
	// work prints its outcome on stderr and leaves stdout empty.
	var out strings.Builder
	fmt.Fprintf(&out, "viewer: @%s (%s)\n\n", viewer.ScreenName, viewer.Name)
	for _, post := range page.Posts {
		by := post.Author.ScreenName
		if post.RepostedBy != nil {
			by += " (reposted by @" + post.RepostedBy.ScreenName + ")"
		}
		fmt.Fprintf(&out, "@%s  %s  %s\n", by, post.CreatedAt.Format("2006-01-02"), preview(post.Text))
	}
	used, limited, _ := client.Limiter.Budget()
	fmt.Fprintf(&out, "\n%d posts  ·  requests: %d used, %d limited, cap %d  ·  %s\n",
		len(page.Posts), used, limited, cfg.SessionRequestCap, time.Since(started).Round(time.Millisecond))
	fmt.Fprint(stdout, out.String())

	return 0
}

// fail reports one failed step on stderr and keeps stdout empty. The message is the classified
// outcome and its remedy when the failure came from the upstream, so the run says what to do
// rather than only what went wrong.
func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "spike: %v\n", err)
	return 1
}

// preview is one post's text on one line: whitespace collapsed, cut to previewRunes runes. A post
// with a newline in it would otherwise turn one line of output into several.
func preview(text string) string {
	line := strings.Join(strings.FieldsFunc(text, unicode.IsSpace), " ")
	runes := []rune(line)
	if len(runes) <= previewRunes {
		return line
	}
	return string(runes[:previewRunes]) + "…"
}

// recorder returns the function that saves one response body. An empty dir means the caller did not
// ask for one, and a body that cannot be written is reported and does not stop the run: the read
// already happened, and its result is what the run is for.
//
// 0600, in a directory of the caller's choosing: a response this account's session fetched is this
// account's data, which is why nothing recorded this way is committed without being read first.
func recorder(dir string, stderr io.Writer) func(name string, body []byte) {
	if dir == "" {
		return func(string, []byte) {}
	}
	return func(name string, body []byte) {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			fmt.Fprintf(stderr, "spike: recording: %v\n", err)
			return
		}
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
			fmt.Fprintf(stderr, "spike: recording: %v\n", err)
		}
	}
}
