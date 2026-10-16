package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xsource"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// whoamiTimeout bounds the one read this verb makes. A verb that never returns is one of the ways
// this tool could stop telling the truth about what it is doing.
const whoamiTimeout = 30 * time.Second

// newAPIClient is the seam this package's tests replace: the client the verbs read through, with
// the session they are to read as. It is a variable rather than a call at each verb for the same
// reason netcheck's Target takes its dependencies as fields — a test has to be able to point a
// real verb at a fake upstream.
var newAPIClient = func(cfg config.Config, sess session.Session) (*xweb.Client, error) {
	client, err := xweb.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return client.WithSession(sess), nil
}

// Whoami is the `whoami` verb: it names the account the loaded session belongs to.
//
// It exists because a session that belongs to the wrong account is the most confusing failure in
// this tool — content that is not the user's looks like a bug in the app, and the cookie is not
// something anybody can check by eye. The verb answers it with one authenticated read (the Viewer
// operation, which answers with the account the session belongs to), and prints the session's own
// state beside it: lengths, origin and path, never the cookie.
//
// A session that is missing, unreadable or rejected is a named outcome and a non-zero exit code.
// So is a response that does not carry a handle: "whoami failed" with no reason is the state this
// verb exists to remove.
func Whoami(ctx *Context, args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(ctx.Stderr, "whoami takes no arguments\n")
		usage(ctx.Stderr)
		return ExitUsage
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "whoami: %v\n", err)
		return ExitError
	}
	for _, warning := range cfg.Warnings {
		fmt.Fprintf(ctx.Stderr, "config warning: %s\n", warning)
	}

	sess, err := session.Load(cfg.SessionPath)
	if err != nil {
		// The message names the path and, for a missing session, what to put in it. It carries no
		// cookie: session.Load never puts one in an error.
		fmt.Fprintf(ctx.Stderr, "whoami: %v\n", err)
		return ExitError
	}

	client, err := newAPIClient(cfg, sess)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "whoami: %v\n", err)
		return ExitError
	}

	runCtx, cancel := context.WithTimeout(context.Background(), whoamiTimeout)
	defer cancel()

	raw, err := client.GraphQL(runCtx, "Viewer", nil)
	if err != nil {
		// The error carries the classified outcome and its remedy: refused credentials, an
		// unknown operation, a rate limit and an edge challenge are four different sentences, and
		// the user's next move is different for each.
		fmt.Fprintf(ctx.Stderr, "whoami: %v\n", err)
		return ExitError
	}

	user, err := xsource.ParseUser(raw)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "whoami: %v\n", err)
		return ExitError
	}

	fmt.Fprintf(ctx.Stdout, "@%s — %s\n", user.ScreenName, user.Name)
	fmt.Fprintf(ctx.Stdout, "%s\n", sess.Summary())

	// The first successful run of any verb is where the account-risk statement belongs: the user
	// has just seen the tool work, which is when a warning about what it does with their identity
	// is worth reading.
	ShowNoticeOnce(ctx, cfg.NoticePath())
	return ExitOK
}
