package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/store"
	"github.com/LZD121/x-terminal/internal/xsource"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// syncTimeout bounds one verb run. A sync that never returns is one of the ways
// this tool could stop telling the truth about what it is doing.
const syncTimeout = 10 * time.Minute

// openMirror is the seam this package's tests replace: open the mirror file,
// wrap it with the walk, and return a closer. Production builds the adapter over
// the shared client so Likes can pick up the viewer's id from Begin's Viewer read.
var openMirror = func(path string, fetch store.SavedFetcher, viewer store.Viewer) (*store.Mirror, io.Closer, error) {
	st, err := store.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return store.NewMirror(st, fetch, viewer), st, nil
}

// likesReady is the production fetcher+viewer: Begin's Viewer read names the
// session and stamps the adapter with WithViewerID so a later Likes page has the
// measured userId without a second identity call outside the walk.
type likesReady struct {
	mu sync.RWMutex
	a  *xsource.Adapter
}

// set points the adapter at another account's session. It is what an account switch calls: the
// mirror is built once for the process, and its reads follow whichever session is active now.
func (l *likesReady) set(a *xsource.Adapter) {
	l.mu.Lock()
	l.a = a
	l.mu.Unlock()
}

// current is the adapter under the lock: a switch and a sync can overlap.
func (l *likesReady) current() *xsource.Adapter {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.a
}

func (l *likesReady) Viewer(ctx context.Context) (domain.User, error) {
	u, err := l.current().Viewer(ctx)
	if err != nil {
		return domain.User{}, err
	}
	l.mu.Lock()
	l.a = l.a.WithViewerID(u.ID)
	l.mu.Unlock()
	return u, nil
}

func (l *likesReady) Saved(ctx context.Context, kind domain.SavedKind, c domain.Cursor) (domain.PostPage, error) {
	return l.current().Saved(ctx, kind, c)
}

// Sync is the `sync` verb: fill the saved mirror from x.com through the same
// Begin/Step/Abandon walk the pane's r will drive. It prints presence facts only
// — kind, pages, items, absent, stop reason, path — never a post body and never
// a token.
func Sync(ctx *Context, args []string) int {
	var (
		full     bool
		asJSON   bool
		quiet    bool
		kindFlag string
		kindSet  bool
	)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--full":
			full = true
		case "--json":
			asJSON = true
		case "--quiet":
			quiet = true
		case "--kind":
			if i+1 >= len(args) {
				fmt.Fprintln(ctx.Stderr, "sync --kind needs bookmarks or likes")
				usage(ctx.Stderr)
				return ExitUsage
			}
			kindFlag = args[i+1]
			kindSet = true
			i++
		default:
			fmt.Fprintf(ctx.Stderr, "sync: unknown argument %s\n", args[i])
			usage(ctx.Stderr)
			return ExitUsage
		}
	}
	if asJSON && quiet {
		fmt.Fprintln(ctx.Stderr, "sync: --json and --quiet cannot be combined")
		usage(ctx.Stderr)
		return ExitUsage
	}

	kinds, err := syncKinds(kindSet, kindFlag)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "sync: %v\n", err)
		usage(ctx.Stderr)
		return ExitUsage
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "sync: %v\n", err)
		return ExitError
	}
	for _, warning := range cfg.Warnings {
		fmt.Fprintf(ctx.Stderr, "config warning: %s\n", warning)
	}

	sess, err := session.Load(cfg.SessionPath)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "sync: %v\n", err)
		return ExitError
	}

	client, err := newAPIClient(cfg, sess)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "sync: %v\n", err)
		return ExitError
	}

	adapter := &likesReady{a: xsource.New(client)}
	if dir := filepath.Dir(cfg.SavedDBPath()); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			fmt.Fprintf(ctx.Stderr, "sync: %v\n", err)
			return ExitError
		}
	}
	mirror, closer, err := openMirror(cfg.SavedDBPath(), adapter, adapter)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "sync: %v\n", err)
		return ExitError
	}
	if closer != nil {
		defer closer.Close() //nolint:errcheck
	}

	runCtx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	path := mirror.Path()
	if path == "" {
		path = cfg.SavedDBPath()
	}

	ok := true
	pathPrinted := false
	for _, kind := range kinds {
		prog, walkErr := walkKind(runCtx, mirror, kind, full)
		if walkErr != nil {
			ok = false
			reportSyncErr(ctx, kind, walkErr)
			if syncStopsProcess(walkErr) {
				break
			}
			continue
		}
		if quiet {
			continue
		}
		if asJSON {
			writeSyncJSON(ctx.Stdout, kind, prog, path)
			continue
		}
		writeSyncLine(ctx.Stdout, kind, prog)
		if !pathPrinted {
			fmt.Fprintf(ctx.Stdout, "%-10s %s\n", "mirror", path)
			pathPrinted = true
		}
	}

	if ok {
		ShowNoticeOnce(ctx, cfg.NoticePath())
		return ExitOK
	}
	return ExitError
}

func syncKinds(set bool, flag string) ([]domain.SavedKind, error) {
	if !set {
		return []domain.SavedKind{domain.Bookmark, domain.Like}, nil
	}
	switch flag {
	case "bookmarks":
		return []domain.SavedKind{domain.Bookmark}, nil
	case "likes":
		return []domain.SavedKind{domain.Like}, nil
	default:
		return nil, fmt.Errorf("--kind needs bookmarks or likes, got %q", flag)
	}
}

func walkKind(ctx context.Context, m *store.Mirror, kind domain.SavedKind, full bool) (domain.SyncProgress, error) {
	run, err := m.Begin(ctx, kind, full)
	if err != nil {
		return domain.SyncProgress{}, err
	}
	var last domain.SyncProgress
	for {
		p, err := run.Step(ctx)
		if err != nil {
			return p, err
		}
		last = p
		if p.Done {
			return last, nil
		}
	}
}

func writeSyncLine(w io.Writer, kind domain.SavedKind, p domain.SyncProgress) {
	fmt.Fprintf(w, "%-10s pages %d  items %d  absent %d  %s\n",
		kind.List(), p.Pages, p.Added, p.Absent, p.Stop.String())
}

type syncJSON struct {
	Kind   string `json:"kind"`
	Pages  int    `json:"pages"`
	Items  int    `json:"items"`
	Absent int    `json:"absent"`
	Stop   string `json:"stop"`
	Path   string `json:"path"`
}

func writeSyncJSON(w io.Writer, kind domain.SavedKind, p domain.SyncProgress, path string) {
	_ = json.NewEncoder(w).Encode(syncJSON{
		Kind:   kind.List(),
		Pages:  p.Pages,
		Items:  p.Added,
		Absent: p.Absent,
		Stop:   p.Stop.String(),
		Path:   path,
	})
}

func reportSyncErr(ctx *Context, kind domain.SavedKind, err error) {
	if errors.Is(err, store.ErrWrongOwner) {
		fmt.Fprintf(ctx.Stderr, "sync %s: %v\n", kind.List(), err)
		return
	}
	f := failureFor(err, nil)
	word := render.StateWord(f.Kind)
	if word == "" {
		word = "failure"
	}
	fmt.Fprintf(ctx.Stderr, "sync %s: %s\n", kind.List(), word)
	if f.Detail != "" && f.Detail != word {
		fmt.Fprintf(ctx.Stderr, "  %s\n", f.Detail)
	}
}

// syncStopsProcess reports whether the error ends the verb for every remaining
// kind: a soft ban, a spent budget, or a stopped session.
func syncStopsProcess(err error) bool {
	if errors.Is(err, xweb.ErrRequestCap) {
		return true
	}
	f := failureFor(err, nil)
	return f.Stopped || f.Kind == domain.FailureBudgetSpent || f.Kind == domain.FailureSessionExpired || f.Kind == domain.FailureNoSession
}
