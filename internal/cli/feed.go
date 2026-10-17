package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/media"
	"github.com/LZD121/x-terminal/internal/netcheck"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/store"
	"github.com/LZD121/x-terminal/internal/term"
	"github.com/LZD121/x-terminal/internal/tui"
	"github.com/LZD121/x-terminal/internal/xsource"
	"github.com/LZD121/x-terminal/internal/xweb"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

// feedSource is the domain.Source the timeline reads through: the seam this package's tests replace
// with internal/fake, so that a test drives the model this verb builds with no network, no session
// cookie and no terminal. It is a variable for the same reason newAPIClient is (whoami.go), and it
// is the only place the production adapter is named: this verb is the composition root, and it is
// the only package that names config, session, xweb, xsource and the shell together.
var feedSource = func(client *xweb.Client, sess session.Session) domain.Source {
	return xsource.New(client.WithSession(sess))
}

// feedDirectory is the domain.Directory the profile's account lists and media tab read through:
// the same adapter as feedSource, over one client.WithSession copy, so the four reads go through the
// one limiter the status line's budget already reads. A variable for feedSource's reason — a test
// drives the shell's account lists with internal/fake and no network.
var feedDirectory = func(client *xweb.Client, sess session.Session) domain.Directory {
	return xsource.New(client.WithSession(sess))
}

// feedImage is the reading column's image read: the bytes of one attachment, through the same
// proxy-aware transport, with no cookie and a cap. It is the only in-process fetch of something
// other than JSON, and it is a variable for feedSource's reason.
var feedImage = func(client *xweb.Client, cfg config.Config) func(context.Context, string) ([]byte, string, error) {
	fetcher, err := xweb.NewImageFetcher(cfg, 8<<20)
	if err != nil {
		return nil
	}
	return fetcher.Get
}

// feedImages reports whether this terminal can draw a picture. Inside a multiplexer the process's
// own TERM names tmux, not the terminal behind it, so the multiplexer is asked for the client's term
// name; outside one, the environment is the whole answer.
func feedImages() bool {
	if os.Getenv("TMUX") == "" {
		return media.KittySupported(os.Getenv)
	}
	return media.KittySupportedName(term.New(os.Stdout).ClientTermName())
}

// threadSource is the domain.Threads the conversation reads through: the same adapter as
// feedSource, declared beside it so a test can replace both seams and no test reaches the network.
var threadSource = func(client *xweb.Client, sess session.Session) domain.Threads {
	return xsource.New(client.WithSession(sess))
}

// feedTranslate is the domain.Translator the pane reads translations through: the same adapter as
// feedSource, over one client.WithSession copy, so a translation is one more request through the one
// limiter the status line's budget already reads. It is a variable for the same reason feedSource is
// — a test drives the shell's translation path with internal/fake and no network — and it is the last
// place the production translator is named.
//
// The config is part of the seam's shape rather than something the adapter below needs: the port
// takes the language per call, because the setting is resolved once (Config.TranslateLanguage) and
// read as a value, and a seam is not the place to resolve it a second time. What the parameter buys
// is that a test replacing this seam can read the setting the run resolved, which is how the
// language's wiring is asserted from outside the shell.
var feedTranslate = func(client *xweb.Client, sess session.Session, cfg config.Config) domain.Translator {
	c := client.WithSession(sess)
	return xsource.New(c).WithTranslator(c)
}

// feedReach is the reachability check the pane consults when a request fails with an error no
// classifier rule names, so that the state on the line is the check's own verdict — no route to
// x.com, a refused proxy, a blocked upstream — rather than a transport error rendered as an
// unclassified shape. The stages it runs are the unauthenticated ones: the zero session stops the
// check before any read that would spend an account request, and the client is the one the pane reads
// through, so the budget line counts what the check spent. It is a variable for the same reason
// feedSource is: a test drives the pane's no-route state without a socket.
var feedReach = func(client *xweb.Client, cfg config.Config) func(context.Context) []netcheck.Result {
	target, err := netcheck.NewTargetWithClient(cfg, client)
	return func(ctx context.Context) []netcheck.Result {
		if err != nil {
			// The path could not be built at all — an unusable proxy setting. Nothing was dialed, and
			// that is the one result the check itself reports for this case (Check rebuilds the path
			// and fails the same way).
			return []netcheck.Result{{Outcome: netcheck.OutcomeConfigFailed, Detail: err.Error()}}
		}
		if used, _, _ := client.Limiter.Budget(); used >= cfg.SessionRequestCap {
			// The check's own request would be refused by the cap this process set on itself rather
			// than measure anything, and a stage that never ran must not report where the path broke.
			return nil
		}
		return netcheck.CheckWith(ctx, target, session.Session{})
	}
}

// feedTerm is the terminal-side sink this verb builds: the one package of the module allowed to
// write to the terminal or to run a program. It is a variable for the same reason feedSource is —
// a test drives the whole verb without a subprocess — and the program's own output file is what the
// panic entry is written to, so the entry's bytes land in the same stream the renderer paints.
var feedTerm = term.New

// feedMediaLookPath and feedMediaRun are the process seams External is wired
// with. They are variables for the same reason feedSource is: a test drives the
// verb with a recorder and no subprocess. Production uses media.DefaultLookPath
// and media.StartAndReap so this file never imports os/exec — the boundary holds
// that import to internal/term and internal/media.
var feedMediaLookPath = media.DefaultLookPath
var feedMediaRun = media.StartAndReap
var feedClipboardRun = media.RunWithStdin

// openSaved is the composition root's store seam: open cfg.SavedDBPath, wrap it
// with store.NewMirror, and return a closer. When the open fails the verb does
// not fail — it fills Deps.Saved with a mirror that returns the open error from
// every method so the frame can render "mirror unreadable". The error store.Open
// answers with is its own mark on the file (store.FileError), so the frame's
// mapping can tell a failure of the file from a failure of the request path.
var openSaved = func(cfg config.Config, fetch store.SavedFetcher, viewer store.Viewer) (domain.Mirror, func() error, error) {
	st, err := store.Open(cfg.SavedDBPath())
	if err != nil {
		return unreadableMirror{err: err}, func() error { return nil }, err
	}
	m := store.NewMirror(st, fetch, viewer)
	return m, st.Close, nil
}

// unreadableMirror is the offline stand-in when saved.db cannot be opened: every
// method returns the store's own marked error (store.FileError, which is what
// wants CapMirrorUnreadable and nothing else) so the frame renders the mirror
// unreadable state with the file's own remedy.
type unreadableMirror struct{ err error }

func (u unreadableMirror) Page(ctx context.Context, kind domain.SavedKind, c domain.Cursor) (domain.PostPage, error) {
	return domain.PostPage{}, u.err
}
func (u unreadableMirror) Status(ctx context.Context, kind domain.SavedKind) (domain.MirrorStatus, error) {
	return domain.MirrorStatus{}, u.err
}
func (u unreadableMirror) Begin(ctx context.Context, kind domain.SavedKind, full bool) (domain.SyncRun, error) {
	return nil, u.err
}

// savedMirrorError reports whether err is the mirror's own failure — the store's own operation on
// the file, which is what CapMirrorUnreadable is for — rather than a state of the request path that
// Deps.Failure already names.
//
// It is a whitelist, and deliberately a narrow one: internal/store marks exactly the failures of
// opening, migrating and reading saved.db with store.FileError, and nothing else wears that mark.
// Everything else the walk can answer with is a fact about a request or about the shape of an
// answer, not about the file: a classified outcome (*xweb.APIError, which carries every state the
// classifier has a row for), the request cap this process set on itself, a session that is missing
// or unusable, a transport error that never left this machine, and — the case a blacklist of those
// four missed — a *xsource.ParseError, which is how the walk's fetch answers a 200 whose body this
// build cannot read. None of them is a fact about saved.db, each has a state and a remedy the pane
// already prints, and a frame that wore CapMirrorUnreadable for one would be telling the reader to
// move a healthy file aside. An error this predicate has never seen is not the file's either: the
// pane prints the mapper's own state for it rather than a claim about saved.db.
func savedMirrorError(err error) bool {
	var file *store.FileError
	return errors.As(err, &file)
}

// runProgram hands the composed model to the terminal. It is a variable so that a test can drive
// the model this verb builds without opening one.
var runProgram = func(m tui.Model) error {
	_, err := tea.NewProgram(m).Run()
	return err
}

// Feed is the `feed` verb: the Following timeline in the pane. It is also what a bare invocation
// runs — no arguments is the program's own surface, and the surface is the timeline — while every
// other invocation shape keeps its usage path. `feed` opens on Following, which is the list the
// phase exists for.
//
// The composition is the whole stack and nothing else: the configuration, the operation table, the
// session, one transport, one limiter, one client, the adapter and the shell. The ordering is what
// makes the failures honest. The session is read before anything is requested, because its absence
// is a state the pane shows rather than a crash: the model is built with that state already on it
// and no request is issued for it. The warnings config and the pins file carry go to stderr before
// the program starts, never into the pane, because the pane is the timeline rather than a
// diagnostic stream.
func Feed(ctx *Context, args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(ctx.Stderr, "feed: unknown argument %s\n", args[0])
		usage(ctx.Stderr)
		return ExitUsage
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "feed: %v\n", err)
		return ExitError
	}
	for _, warning := range cfg.Warnings {
		fmt.Fprintf(ctx.Stderr, "config warning: %s\n", warning)
	}

	// The operation table is read here as well as inside the client, for one reason: a pins file the
	// user wrote and this run cannot read has to be visible. A pin that silently did nothing would
	// be the kind of quiet wrongness this tool exists to remove.
	registry, err := volatile.Load(cfg.PinsPath())
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "feed: registry: %v\n", err)
	} else {
		for _, warning := range registry.Warnings() {
			fmt.Fprintf(ctx.Stderr, "registry warning: %s\n", warning)
		}
	}

	sess, sessionErr := session.Load(cfg.SessionPath)
	for _, warning := range sess.Warnings {
		fmt.Fprintf(ctx.Stderr, "session warning: %s\n", warning)
	}

	client, err := xweb.NewClient(cfg)
	if err != nil {
		// The transport could not be built, so there is nothing for the shell to read through and no
		// frame that would be telling the truth: the error names the setting to fix and `doctor`
		// prints the whole path.
		fmt.Fprintf(ctx.Stderr, "feed: %v\n", err)
		return ExitError
	}

	// The display state the pane opens in, and the terminal sink its transitions write through. The
	// two keys are resolved through the shell's own vocabulary and the render package's own lookup:
	// an unknown value keeps the default, and the reader has already warned about it by name and line.
	display := tui.Display{State: tui.StateNormal, Cover: render.CoverServiceLog}
	if state, ok := tui.StateOf(cfg.DisplayState); ok {
		display.State = state
	}
	if cover, ok := render.CoverStyleOf(cfg.DisplayCover); ok {
		display.Cover = cover
	}
	sink := feedTerm(ctx.Stdout)
	// The window name to put back on exit, read once before the program starts. Under tmux only:
	// without a pane there is no window this process renames, so the value could never be used.
	window := sink.WindowName()

	// The saved mirror: open once here. A failure to open does not fail the verb.
	var savedCloser func() error
	adapter := &likesReady{a: xsource.New(client.WithSession(sess))}
	saved, closer, openErr := openSaved(cfg, adapter, adapter)
	savedCloser = closer
	_ = openErr
	defer func() {
		if savedCloser != nil {
			_ = savedCloser()
		}
	}()

	savedCapability := func(kind domain.SavedKind) domain.Capability {
		if registry == nil {
			return domain.CapNone
		}
		name := "Bookmarks"
		if kind == domain.Like {
			name = "Likes"
		}
		if _, err := registry.Lookup(name); err != nil {
			return domain.CapOperationMissing
		}
		return domain.CapNone
	}

	// savedError maps one error from the mirror or the walk into the capability the frame wears.
	// CapMirrorUnreadable is the narrow case and stays narrow: it is the store's own mark on the file
	// (store.FileError), which is the one state whose remedy is about saved.db. Every other error
	// the walk or the mirror can answer with is CapNone, so Deps.Failure names the real state and
	// the real remedy — a refused proxy, a spent cap, a missing session and a classified answer all
	// have one, and a 200 whose body this build cannot read prints the parser's own words — and none
	// of them is about the file. The wrong-account refusal is the mirror's own answer and has its own
	// capability, because a file that is healthy but another account's is neither.
	savedError := func(err error) domain.Capability {
		if errors.Is(err, store.ErrWrongOwner) {
			return domain.CapWrongAccount
		}
		if savedMirrorError(err) {
			return domain.CapMirrorUnreadable
		}
		return domain.CapNone
	}

	// One limiter, and the budget line reads that one: client.Limiter is what every request from the
	// adapter passes through, so the number on screen cannot drift from the requests issued.
	model := tui.New(tui.Deps{
		Source:          feedSource(client, sess),
		Threads:         threadSource(client, sess),
		Budget:          client.Limiter.Budget,
		Failure:         failureFor,
		Check:           feedReach(client, cfg),
		Now:             time.Now,
		Term:            sink,
		Display:         display,
		Saved:           saved,
		SavedCapability: savedCapability,
		SavedError:      savedError,
		MirrorPath:      cfg.SavedDBPath(),
		Media: media.External{
			Opener:   cfg.MediaOpener,
			Env:      os.Getenv,
			LookPath: feedMediaLookPath,
			Run:      feedMediaRun,
		},
		Clip: media.Clipboard{
			Out:      ctx.Stdout,
			Env:      os.Getenv,
			LookPath: feedMediaLookPath,
			Run:      feedClipboardRun,
		},

		// The translation read, built once here like every other port, and the language the config
		// resolved. The port is nil only if this seam returns nil, which is what a build with no
		// translation endpoint would look like: the `T` key then says so and spends nothing.
		Directory:     feedDirectory(client, sess),
		Image:         feedImage(client, cfg),
		Images:        feedImages(),
		Translate:     feedTranslate(client, sess, cfg),
		TranslateLang: cfg.TranslateLanguage,
		Persist: func(state, cover string) {
			// The display's own write, from a command: the file it names is the one the loader read,
			// and nothing else in the file is touched. A write that fails is not reported on screen —
			// the pane says what the reader is doing, not what this program's own bookkeeping did —
			// and the next deliberate change tries again.
			_ = config.SetDisplay(filepath.Join(cfg.ConfigDir, config.ConfigFileName), state, cover)
		},
	})
	if sessionErr != nil {
		// A missing or unusable session is a state, not a crash: the state line names it, the pane
		// shows it, and no request is issued for it — `?` and `q` still work on the frame that says
		// so. The message goes to stderr too, because it is the one that names the path to fix.
		fmt.Fprintf(ctx.Stderr, "feed: %v\n", sessionErr)
		model = model.WithFailure(failureFor(sessionErr, nil))
	}

	if err := runProgram(model); err != nil {
		fmt.Fprintf(ctx.Stderr, "feed: %v\n", err)
		return ExitError
	}

	// The program is gone, so the window name is put back to what it was. A bare terminal's title is
	// cleared by the toolkit itself — the view set one, and the toolkit clears a title it set — which
	// is why no pre-existing title is queried: there is no portable way to read one back.
	if window != "" {
		sink.SetWindow(window)
	}
	return ExitOK
}
