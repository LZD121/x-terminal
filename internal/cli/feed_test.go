package cli

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/fake"
	"github.com/LZD121/x-terminal/internal/netcheck"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/store"
	"github.com/LZD121/x-terminal/internal/tui"
	"github.com/LZD121/x-terminal/internal/xsource"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// counting is the fake Source with every read issued through the limiter the status line reads: the
// number on screen and the reads the shell makes are then the same number, which is what OPS-02 is
// about. The limiter is a real *xweb.Limiter and its own function does nothing — no socket is
// touched and the fake is what answers.
type counting struct {
	*fake.Source
	threads *fake.Thread
	limiter *xweb.Limiter
}

func (c counting) Home(ctx context.Context, kind domain.HomeKind, cursor domain.Cursor) (domain.PostPage, error) {
	if _, err := c.limiter.Do(ctx, func() (*http.Response, error) { return nil, nil }); err != nil {
		return domain.PostPage{}, err
	}
	return c.Source.Home(ctx, kind, cursor)
}

func (c counting) Thread(ctx context.Context, id domain.PostID, cursor domain.Cursor) (domain.ThreadPage, error) {
	if c.threads == nil {
		return domain.ThreadPage{}, nil
	}
	if _, err := c.limiter.Do(ctx, func() (*http.Response, error) { return nil, nil }); err != nil {
		return domain.ThreadPage{}, err
	}
	return c.threads.Thread(ctx, id, cursor)
}

// feedOptions are the knobs a saved-frame test needs before the verb runs. The zero value is the
// production saved side: the verb's own openSaved opens saved.db where the config puts it, and the
// verb's own reachability check is what the frame consults.
type feedOptions struct {
	// saved is the mirror the composition opens. Nil means openSaved runs, which is what a test
	// about a saved.db that cannot be opened wants: it places the file and the real store reports it.
	saved domain.Mirror
	// reach is the page's reachability check. Nil means the verb's own, which opens a socket to
	// x.com; no test here takes that one, and the two that exercise a failure state replace it.
	reach func(context.Context) []netcheck.Result
	// data is XDG_DATA_HOME for this run, for a test that has a saved.db to put where the verb will
	// read it. Empty means hermeticConfig's own temp directory, with no mirror in it.
	data string
}

// feedComposed runs the `feed` verb and captures the model it built, so the whole composition —
// config, session, registry, transport, one limiter, the adapter, the shell — is the real path while
// no terminal is opened: the seams the verb carries (feedSource, threadSource and runProgram) are
// what the test replaces, the saved side is whatever opts says, and the real limiter is handed back
// with the model.
func feedComposed(t *testing.T, src *fake.Source, opts feedOptions) (tui.Model, *xweb.Limiter) {
	t.Helper()
	hermeticConfig(t, writeSession(t, t.TempDir(), "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n"))
	if opts.data != "" {
		// hermeticConfig points the data directory at one of its own; this test filled another one.
		t.Setenv("XDG_DATA_HOME", opts.data)
	}

	var (
		model    tui.Model
		limiter  *xweb.Limiter
		language string
	)
	th := fake.ThreadShapes()
	previousSource, previousThread, previousRun, previousTranslate := feedSource, threadSource, runProgram, feedTranslate
	feedSource = func(client *xweb.Client, _ session.Session) domain.Source {
		limiter = client.Limiter
		return counting{Source: src, threads: th, limiter: client.Limiter}
	}
	threadSource = func(client *xweb.Client, _ session.Session) domain.Threads {
		return counting{Source: src, threads: th, limiter: client.Limiter}
	}
	// The translation port is replaced for the same reason the two reads are: the composition is the
	// real path, but the request behind the port is a socket this test must not open. The language the
	// run resolved is recorded here, which is how the setting's own wiring is asserted from outside
	// the shell — a tui.Model's Deps are not read from here.
	feedTranslate = func(_ *xweb.Client, _ session.Session, cfg config.Config) domain.Translator {
		language = cfg.TranslateLanguage
		return &fake.Translator{}
	}
	runProgram = func(m tui.Model) error {
		model = m
		return nil
	}
	seams := []func(){
		func() {
			feedSource, threadSource, runProgram, feedTranslate = previousSource, previousThread, previousRun, previousTranslate
		},
	}
	if opts.saved != nil {
		previousSaved := openSaved
		openSaved = func(config.Config, store.SavedFetcher, store.Viewer) (domain.Mirror, func() error, error) {
			return opts.saved, func() error { return nil }, nil
		}
		seams = append(seams, func() { openSaved = previousSaved })
	}
	if opts.reach != nil {
		previousReach := feedReach
		feedReach = func(*xweb.Client, config.Config) func(context.Context) []netcheck.Result {
			return opts.reach
		}
		seams = append(seams, func() { feedReach = previousReach })
	}
	t.Cleanup(func() {
		for _, restore := range seams {
			restore()
		}
	})

	code, _, stderr := run(t, Feed)
	if code != ExitOK {
		t.Fatalf("feed exited %d (%s)", code, stderr)
	}
	if limiter == nil {
		t.Fatalf("the verb ran no composition: there is no limiter and no model to drive")
	}
	if language == "" {
		t.Fatalf("the composition resolved no translation language: the pane was handed the empty setting")
	}
	return model, limiter
}

// feedUnderTest is feedComposed with the verb's own saved side, for the frames that never open one.
func feedUnderTest(t *testing.T, src *fake.Source) (tui.Model, *xweb.Limiter) {
	t.Helper()
	return feedComposed(t, src, feedOptions{})
}

// feedKey builds the key press a terminal reports for one letter.
func feedKey(letter byte) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: rune(letter), Text: string(letter)})
}

// feedRun executes a command and answers everything it leads to, which is the loop the program
// itself runs: a command is a value until something calls it, and a batch is the commands the shell
// runs next — which is what opening a saved frame returns, its page read and its status read.
func feedRun(t *testing.T, m tui.Model, cmd tea.Cmd) tui.Model {
	t.Helper()
	var queue []tea.Cmd
	if cmd != nil {
		queue = append(queue, cmd)
	}
	for i := 0; len(queue) > 0; i++ {
		if i > 16 {
			t.Fatalf("the shell answered commands with commands %d times over", i)
		}
		cmd, queue = queue[0], queue[1:]
		answer := cmd()
		if answer == nil {
			continue
		}
		if batch, ok := answer.(tea.BatchMsg); ok {
			for _, batched := range batch {
				if batched != nil {
					queue = append(queue, batched)
				}
			}
			continue
		}
		next, asked := feedUpdate(t, m, answer)
		m = next
		if asked != nil {
			queue = append(queue, asked)
		}
	}
	return m
}

// feedUpdate applies one message and returns the command it asks for.
func feedUpdate(t *testing.T, m tui.Model, msg tea.Msg) (tui.Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	typed, ok := next.(tui.Model)
	if !ok {
		t.Fatalf("the shell's Update returned %T, want tui.Model", next)
	}
	return typed, cmd
}

// feedPress presses one letter and answers everything it leads to.
func feedPress(t *testing.T, m tui.Model, letter byte) tui.Model {
	t.Helper()
	next, cmd := feedUpdate(t, m, feedKey(letter))
	return feedRun(t, next, cmd)
}

// feedPane and feedStatus split a frame: the status line is its last row.
func feedPane(frame string) string {
	rows := strings.Split(frame, "\n")
	return strings.Join(rows[:len(rows)-1], "\n")
}

func feedStatus(frame string) string {
	rows := strings.Split(frame, "\n")
	return rows[len(rows)-1]
}

// TestFeedOpensOnFollowingAndSpendsFromTheLine is FEED-01's default and OPS-02's wiring: the first
// request is for Following, and the budget line is the limiter the verb built — the one every
// request of the composition passes through.
func TestFeedOpensOnFollowingAndSpendsFromTheLine(t *testing.T) {
	src := fake.Adversarial()
	model, _ := feedUnderTest(t, src)

	if status := feedStatus(model.Render(80, 24)); !strings.Contains(status, "0 requests, 0 rate-limited") {
		t.Errorf("the first frame's status line is %q, want it to report nothing spent yet", status)
	}

	model = feedRun(t, model, model.Init())
	if kind, ok := src.LastKind(); !ok || kind != domain.Following {
		t.Fatalf("the first request was for %v (reported %v), want Following", kind, ok)
	}
	if !strings.Contains(feedPane(model.Render(80, 24)), "a_15_cell_handl") {
		t.Errorf("the Following pane does not render the first page")
	}
	if status := feedStatus(model.Render(80, 24)); !strings.Contains(status, "1 request, 0 rate-limited") {
		t.Errorf("after one request the status line is %q, want it to report one request", status)
	}
}

// TestTheToggleSwitchesFeedsAndReturns: `t` renders the other feed's frame and `t` again restores
// the first one byte for byte.
func TestTheToggleSwitchesFeedsAndReturns(t *testing.T) {
	src := fake.Adversarial()
	model, _ := feedUnderTest(t, src)
	model = feedRun(t, model, model.Init())
	before := feedPane(model.Render(80, 24))

	model = feedPress(t, model, 't')
	status := feedStatus(model.Render(80, 24))
	if !strings.Contains(status, render.CopyListForYou) {
		t.Errorf("the toggled frame's status line is %q, want it to name the other list", status)
	}
	if kind, _ := src.LastKind(); kind != domain.ForYou {
		t.Errorf("the toggled frame read %v, want ForYou", kind)
	}

	model = feedPress(t, model, 't')
	// The pane is what must come back byte for byte; the budget line is not part of the restore, and
	// it has moved: the toggle spent one request on the other feed's first page.
	if got := feedPane(model.Render(80, 24)); got != before {
		t.Errorf("returning from the other feed is not byte-identical:\n%s\n---\n%s", before, got)
	}
}

// TestASoftBanStopsTheSessionAndKeepsThePosts is OPS-04 through the verb's own composition: the
// classifier's soft ban reaches the status line in the classifier's words, the pane keeps its posts,
// and no key issues a request again.
func TestASoftBanStopsTheSessionAndKeepsThePosts(t *testing.T) {
	src := fake.Adversarial()
	model, _ := feedUnderTest(t, src)
	model = feedRun(t, model, model.Init())

	pane := feedPane(model.Render(80, 24))
	calls := src.Calls()
	src.FailNext(&xweb.APIError{Operation: "HomeLatestTimeline", Outcome: xweb.OutcomeSoftBan, Status: 200})

	model = feedPress(t, model, 'r')
	status := feedStatus(model.Render(80, 24))
	if want := render.StateWord(domain.FailureSoftBan); !strings.Contains(status, want) {
		t.Errorf("the soft ban's status line is %q, want it to carry %q", status, want)
	}
	if want := render.Remedy(domain.FailureSoftBan); !strings.Contains(status, want) {
		t.Errorf("the soft ban's status line is %q, want it to carry %q", status, want)
	}
	if got := feedPane(model.Render(80, 24)); got != pane {
		t.Errorf("the soft ban cleared the posts:\n%s\n---\n%s", pane, got)
	}

	for _, letter := range []byte{'r', 'j', 't'} {
		model = feedPress(t, model, letter)
	}
	if src.Calls() != calls+1 {
		t.Errorf("the stopped session issued %d further requests", src.Calls()-(calls+1))
	}
}

// TestTheFeedClassifiesAReachabilityFailure is OPS-01's third named state through the verb's own
// composition: a request that cannot reach the upstream is not printed as an unclassified transport
// error — the reachability check runs, and the state word, the remedy and the pane's body are the
// check's own words. The check itself is a seam here (the real one opens a socket to x.com), and what
// the test holds is that the composition root runs it and hands its report to the one mapper.
func TestTheFeedClassifiesAReachabilityFailure(t *testing.T) {
	previousReach := feedReach
	t.Cleanup(func() { feedReach = previousReach })

	checks := 0
	detail := "x.com did not resolve: lookup x.com: no such host"
	feedReach = func(*xweb.Client, config.Config) func(context.Context) []netcheck.Result {
		return func(context.Context) []netcheck.Result {
			checks++
			return []netcheck.Result{{Outcome: netcheck.OutcomeNoRoute, Detail: detail}}
		}
	}

	src := fake.Adversarial()
	model, _ := feedUnderTest(t, src)
	src.FailNext(errors.New("the request did not complete"))
	model = feedRun(t, model, model.Init())

	frame := model.Render(80, 24)
	status := feedStatus(frame)
	if want := render.StateWord(domain.FailureNoRoute); !strings.Contains(status, want) {
		t.Errorf("the status line is %q, want it to carry %q", status, want)
	}
	if want := render.Remedy(domain.FailureNoRoute); !strings.Contains(status, want) {
		t.Errorf("the status line is %q, want it to carry the remedy %q", status, want)
	}
	if pane := feedPane(frame); !strings.Contains(pane, detail) {
		t.Errorf("the pane is %q, want the check's own sentence", pane)
	}
	if checks != 1 {
		t.Errorf("the reachability check ran %d times for one unnameable failure, want once", checks)
	}
}

// TestTheReachabilityCheckRefusesToReportAPathItCannotMeasure: the check's own request goes through
// the limiter the budget line reads, so a limiter that has already spent its cap answers it with a
// refusal rather than with a path. A refusal is not a route: the check reports nothing instead, and
// the mapper's own answer — unclassified, with the transport's error — stands. Nothing is dialed here,
// because the decision is made before the first stage.
func TestTheReachabilityCheckRefusesToReportAPathItCannotMeasure(t *testing.T) {
	client := &xweb.Client{HTTP: &http.Client{}, Limiter: xweb.NewLimiter(0, 0)}
	if results := feedReach(client, config.Config{})(context.Background()); results != nil {
		t.Errorf("the check reported %v with its own request refused by the cap, want no report at all", results)
	}
}

// TestASavedFrameNamesATransportFailureAsTheOrdinaryState is the defect this pair of tests exists
// for: a request that never reached x.com was rendered as a fact about saved.db — `mirror
// unreadable`, remedy `move saved.db aside, then sync` — for a mirror that reads perfectly well,
// and the ordinary failure path never ran. It has to run now: the frame takes the state the
// reachability check reports, with the check's own sentence in the pane.
func TestASavedFrameNamesATransportFailureAsTheOrdinaryState(t *testing.T) {
	mirror := fake.NewMirror()
	mirror.FailPage(domain.Bookmark, refusedProxyError(t))

	detail := "proxy refused: dial tcp 127.0.0.1:1: connect: connection refused"
	checks := 0
	model, _ := feedComposed(t, fake.Adversarial(), feedOptions{
		saved: mirror,
		reach: func(context.Context) []netcheck.Result {
			checks++
			return []netcheck.Result{{Outcome: netcheck.OutcomeProxyRefused, Detail: detail}}
		},
	})
	model = feedRun(t, model, model.Init())
	// `b` opens the bookmarks frame; its page read is the one the mirror answers with that error.
	model = feedPress(t, model, 'b')

	frame := model.Render(80, 24)
	status := feedStatus(frame)
	if checks != 1 {
		t.Errorf("the reachability check ran %d times for a saved frame's transport failure, want once: the frame did not take the ordinary failure path", checks)
	}
	if want := render.StateWord(domain.FailureProxyRefused); !strings.Contains(status, want) {
		t.Errorf("the status line after a transport failure on a saved frame is %q, want it to carry %q", status, want)
	}
	if want := render.Remedy(domain.FailureProxyRefused); !strings.Contains(status, want) {
		t.Errorf("the status line is %q, want the path failure's own remedy %q", status, want)
	}
	if pane := feedPane(frame); !strings.Contains(pane, detail) {
		t.Errorf("the pane is %q, want the check's own sentence", pane)
	}
	if strings.Contains(frame, render.CopyCapMirrorUnreadable) || strings.Contains(frame, render.CopyCapRemedyAside) {
		t.Errorf("a transport failure rendered the mirror as the problem:\n%s", frame)
	}
}

// TestAnUnreadableMirrorStillNamesTheCapability is the other direction, and the reason the
// predicate above is not a catch-all: a saved.db that exists and cannot be opened is exactly what
// `mirror unreadable` is for, and it still has to say so — with the file's own path, and without
// spending a request on a path that is not what failed.
func TestAnUnreadableMirrorStillNamesTheCapability(t *testing.T) {
	data := t.TempDir()
	dir := filepath.Join(data, "x-terminal")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("creating the data directory: %v", err)
	}
	path := filepath.Join(dir, "saved.db")
	if err := os.WriteFile(path, []byte("not a sqlite file"), 0o600); err != nil {
		t.Fatalf("writing the unreadable mirror: %v", err)
	}

	checks := 0
	model, _ := feedComposed(t, fake.Adversarial(), feedOptions{
		data: data,
		reach: func(context.Context) []netcheck.Result {
			checks++
			return []netcheck.Result{{Outcome: netcheck.OutcomeNoRoute, Detail: "not measured"}}
		},
	})
	model = feedRun(t, model, model.Init())
	model = feedPress(t, model, 'b')

	frame := model.Render(80, 24)
	status := feedStatus(frame)
	if want := render.CopyCapMirrorUnreadable; !strings.Contains(status, want) {
		t.Errorf("the status line for an unreadable mirror is %q, want it to carry %q", status, want)
	}
	if want := render.CopyCapRemedyAside; !strings.Contains(status, want) {
		t.Errorf("the status line is %q, want the capability's own remedy %q", status, want)
	}
	// The block's body wraps at the pane's width, so the path is read out of the pane with the
	// wrapping whitespace taken out: the wrap is the pane's business, and the claim is that the block
	// names the file rather than saying something generic about it.
	if pane := feedPane(frame); !strings.Contains(strings.Join(strings.Fields(pane), ""), path) {
		t.Errorf("the pane is %q, want the capability block to name %s", pane, path)
	}
	if checks != 0 {
		t.Errorf("the reachability check ran %d times for an unreadable mirror, want never: the state is about the file, so no request is spent on the path", checks)
	}
}

// driftedTimelineBody is the body of a 200 answer whose shape this build cannot read: the envelope
// is a timeline's and the entry is a tweet's, and the one field that drifted is the date — the
// upstream's own format replaced by an HTTP date. The classifier has no row for it (it is not an
// error envelope), so the answer reaches the parser and the parser refuses it: that is the
// registry-drift case the saved walk's fetch meets in production.
func driftedTimelineBody() []byte {
	return []byte(`{"data":{"user":{"result":{"timeline":{"timeline":{"instructions":[{"type":"TimelineAddEntries","entries":[{"entryId":"tweet-1","content":{"__typename":"TimelineTimelineItem","itemContent":{"__typename":"TimelineTweet","tweet_results":{"result":{"__typename":"Tweet","rest_id":"9","legacy":{"full_text":"drifted","created_at":"Mon, 13 Oct 2026 22:16:43 GMT"},"core":{"user_results":{"result":{"__typename":"User","rest_id":"1","core":{"screen_name":"mira"}}}}}}}}}]}]}}}}}}`)
}

// driftedFetch is the saved walk's fetch with exactly one thing replaced: the body the answer
// carried. It runs the production parser over that body — internal/xsource.ParseTimeline is what
// the adapter's own Saved does with a 200 — so the error the walk hands back is the parser's own,
// and no socket is touched: the adapter's transport only ever talks to x.com.
type driftedFetch struct{ body []byte }

func (f driftedFetch) Saved(context.Context, domain.SavedKind, domain.Cursor) (domain.PostPage, error) {
	_, err := xsource.ParseTimeline(f.body)
	return domain.PostPage{}, err
}

// handleViewer is the walk's one read of the session's identity: the account this mirror belongs to.
type handleViewer struct{ handle string }

func (v handleViewer) Viewer(context.Context) (domain.User, error) {
	return domain.User{ID: "1", ScreenName: v.handle}, nil
}

// driftedMirror is a real mirror over a real saved.db whose walk meets the drifted answer: the
// store, its migrations, its reader pool and its walk, with only the response body replaced.
func driftedMirror(t *testing.T, body []byte) *store.Mirror {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "saved.db"))
	if err != nil {
		t.Fatalf("opening a store for the drifted walk: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return store.NewMirror(st, driftedFetch{body: body}, handleViewer{handle: "mira"})
}

// TestAShapeDriftedAnswerIsNotAFactAboutTheFile is the blacklist's blind spot, closed: the walk's
// fetch reaches internal/xsource.ParseTimeline, whose failures are its own *ParseError with no
// wrapping, and a predicate that read "not one of the request path's four named shapes" as "the
// mirror's own failure" rendered every one of them as `mirror unreadable` / `move saved.db aside,
// then sync` — on a saved.db that reads perfectly well. A 200 whose body this build cannot read is
// a state about the answer, so the pane prints the mapper's own words for it and nothing about the
// file.
func TestAShapeDriftedAnswerIsNotAFactAboutTheFile(t *testing.T) {
	body := driftedTimelineBody()

	// The body is one this build cannot read, and the parser says so in its own type: this is the
	// failure the walk's fetch reaches, taken from the production parser rather than written by hand.
	if _, err := xsource.ParseTimeline(body); err == nil {
		t.Fatalf("the drifted body parsed: %s", body)
	} else {
		var parse *xsource.ParseError
		if !errors.As(err, &parse) {
			t.Fatalf("the drifted body answered %T (%v), want the parser's own *xsource.ParseError", err, err)
		}
	}

	// The walk over a real store hands that error back as the fetch left it: nothing in the store
	// relabels it as a failure of the file.
	mirror := driftedMirror(t, body)
	ctx := context.Background()
	run, err := mirror.Begin(ctx, domain.Bookmark, false)
	if err != nil {
		t.Fatalf("beginning the walk: %v", err)
	}
	if _, err := run.Step(ctx); err == nil {
		t.Fatal("a step over a shape-drifted answer succeeded")
	} else {
		var file *store.FileError
		if errors.As(err, &file) {
			t.Errorf("the walk marked %v as a failure of the file", err)
		}
		if savedMirrorError(err) {
			t.Errorf("a shape-drifted answer (%v) was read as a failure of the mirror: the pane would tell the reader to move a healthy saved.db aside", err)
		}
	}

	// The pane: the same walk through the verb and the real capability mapping, with the
	// reachability check reporting the path itself fine — which is what the mapper falls back on
	// when nothing named the state.
	checks := 0
	model, _ := feedComposed(t, fake.Adversarial(), feedOptions{
		saved: mirror,
		reach: func(context.Context) []netcheck.Result {
			checks++
			return []netcheck.Result{{Outcome: netcheck.OutcomeRouteOK, Detail: "route to x.com OK"}}
		},
	})
	model = feedRun(t, model, model.Init())
	model = feedPress(t, model, 'b')
	model = feedPress(t, model, 'r')

	frame := model.Render(80, 24)
	status := feedStatus(frame)
	if want := render.StateWord(domain.FailureUnclassified); !strings.Contains(status, want) {
		t.Errorf("the status line is %q, want the state for a shape this build has no rule for (%q)", status, want)
	}
	if strings.Contains(frame, render.CopyCapMirrorUnreadable) || strings.Contains(frame, render.CopyCapRemedyAside) {
		t.Errorf("a shape-drifted answer rendered the mirror as the problem:\n%s", frame)
	}
	if pane := strings.Join(strings.Fields(feedPane(frame)), ""); !strings.Contains(pane, "legacy.created_at") {
		t.Errorf("the pane is %q, want the parser's own sentence naming the field this build cannot read", feedPane(frame))
	}
	if checks != 1 {
		t.Errorf("the reachability check ran %d times for a shape-drifted answer, want once: the state is unnamed, so the path is what the mapper asks about", checks)
	}
}

// refusedProxyClient is the production client this file's error producers share: the production
// constructor, given a proxy address where nothing is listening and a budget of cap requests, with a
// session from a fixture. Every error it answers with is one production produces, and nothing it
// does leaves this machine — the only address dialed is a closed loopback port.
func refusedProxyClient(t *testing.T, cap int) (*xweb.Client, session.Session) {
	t.Helper()
	hermeticConfig(t, writeSession(t, t.TempDir(), "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n"))
	sess, err := session.Load(writeSession(t, t.TempDir(), "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n"))
	if err != nil {
		t.Fatalf("loading the fixture session: %v", err)
	}
	client, err := xweb.NewClient(config.Config{
		DataDir:           t.TempDir(),
		Proxy:             "http://127.0.0.1:1",
		SessionRequestCap: cap,
	})
	if err != nil {
		t.Fatalf("building the client: %v", err)
	}
	return client, sess
}

// refusedProxyError is the error the mirror's fetch hands back when the request never leaves this
// machine, and it is not a stand-in: net/http's own answer — a *url.Error wrapping the refused
// proxyconnect, wrapped the way xweb wraps every failed operation — is what comes back, and the
// check below is what makes that premise, rather than this test's convenience, hold.
func refusedProxyError(t *testing.T) error {
	t.Helper()
	client, sess := refusedProxyClient(t, 10)
	_, err := client.WithSession(sess).GraphQL(context.Background(), "Bookmarks", nil)
	if err == nil {
		t.Fatal("a request through a proxy that is not there answered successfully")
	}
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		t.Fatalf("the refused proxy produced %T (%v), want net/http's *url.Error: the error this test uses would not be the shape production produces", err, err)
	}
	return err
}

// storeOpenError is a real store error: a saved.db that is there and is not a database, opened by
// the store itself.
func storeOpenError(t *testing.T) error {
	t.Helper()
	path := filepath.Join(t.TempDir(), "saved.db")
	if err := os.WriteFile(path, []byte("not a sqlite file"), 0o600); err != nil {
		t.Fatalf("writing the unreadable mirror: %v", err)
	}
	st, err := store.Open(path)
	if err == nil {
		_ = st.Close()
		t.Fatalf("opening %s succeeded: this test has no store error to check", path)
	}
	return err
}

// requestPathState is one error the request path answers a saved frame with, the state Deps.Failure
// names for it, and how many times the reachability check is asked to look at the path for it.
type requestPathState struct {
	name   string
	err    error
	want   domain.FailureKind
	checks int
	// verdict is what the reachability check reports when it is asked. The states the failure
	// mapper names by itself never ask, so only a row that expects a check carries one.
	verdict netcheck.Result
}

// requestPathErrors produces each of those errors from the production code that produces it: the cap
// this process set on itself, exhausted by real requests; a session the client cannot be issued
// without; and a refused proxy. The proxy is the only address dialed, and it is a closed loopback
// port.
func requestPathErrors(t *testing.T) []requestPathState {
	t.Helper()

	// The first call is refused by the proxy and the second by this process's own cap: the limiter
	// counts a request as it is issued, not as it succeeds.
	capped, sess := refusedProxyClient(t, 1)
	if _, err := capped.WithSession(sess).GraphQL(context.Background(), "Bookmarks", nil); err == nil {
		t.Fatal("the first request through a proxy that is not there answered successfully")
	}
	_, capErr := capped.WithSession(sess).GraphQL(context.Background(), "Bookmarks", nil)
	if !errors.Is(capErr, xweb.ErrRequestCap) {
		t.Fatalf("the second request answered %v, want this process's own request cap", capErr)
	}

	// No session at all, which the client refuses before it builds a request.
	direct, _ := refusedProxyClient(t, 10)
	_, sessionErr := direct.GraphQL(context.Background(), "Bookmarks", nil)
	if !errors.Is(sessionErr, session.ErrNoSession) {
		t.Fatalf("a call without a session answered %v, want session.ErrNoSession", sessionErr)
	}

	return []requestPathState{
		{name: "a spent request cap", err: capErr, want: domain.FailureBudgetSpent},
		{name: "a missing session", err: sessionErr, want: domain.FailureNoSession},
		// The transport error is the one unnamed failure of the three: nothing names it until the
		// reachability check says where the path broke, which is why only this row carries a verdict
		// and expects the check to run.
		{
			name:    "a refused proxy",
			err:     refusedProxyError(t),
			want:    domain.FailureProxyRefused,
			checks:  1,
			verdict: netcheck.Result{Outcome: netcheck.OutcomeProxyRefused, Detail: "proxy refused: dial tcp 127.0.0.1:1: connect: connection refused"},
		},
	}
}

// TestASavedFrameNamesTheRequestPathsOwnStates: the capability says something about the file, so
// only a failure of the file may wear it. A cap this process spent and a session that is missing are
// states the mapper already names, with their own remedies, and the frame has to print those instead
// of `move saved.db aside, then sync`. The check column is the other half of the same claim: a state
// the mapper names spends no request on the path, and only the unnamed transport failure does.
func TestASavedFrameNamesTheRequestPathsOwnStates(t *testing.T) {
	for _, state := range requestPathErrors(t) {
		t.Run(state.name, func(t *testing.T) {
			mirror := fake.NewMirror()
			mirror.FailBegin(domain.Bookmark, state.err)

			checks := 0
			model, _ := feedComposed(t, fake.Adversarial(), feedOptions{
				saved: mirror,
				reach: func(context.Context) []netcheck.Result {
					checks++
					return []netcheck.Result{state.verdict}
				},
			})
			model = feedRun(t, model, model.Init())
			// `b` opens the bookmarks frame and `r` starts its sync, which the mirror fails.
			model = feedPress(t, model, 'b')
			model = feedPress(t, model, 'r')

			frame := model.Render(80, 24)
			status := feedStatus(frame)
			if want := render.StateWord(state.want); !strings.Contains(status, want) {
				t.Errorf("the status line is %q, want it to carry %q", status, want)
			}
			if want := render.Remedy(state.want); !strings.Contains(status, want) {
				t.Errorf("the status line is %q, want the remedy %q", status, want)
			}
			if strings.Contains(frame, render.CopyCapMirrorUnreadable) || strings.Contains(frame, render.CopyCapRemedyAside) {
				t.Errorf("%s rendered the mirror as the problem:\n%s", state.name, frame)
			}
			if checks != state.checks {
				t.Errorf("the reachability check ran %d times, want %d", checks, state.checks)
			}
			// The pane wraps the check's sentence to its width, so the two are compared with the
			// wrapping whitespace taken out of both: the wrap is the pane's business.
			if pane := strings.Join(strings.Fields(feedPane(frame)), ""); checks == 1 && !strings.Contains(pane, strings.Join(strings.Fields(state.verdict.Detail), "")) {
				t.Errorf("the pane is %q, want the check's own sentence %q", feedPane(frame), state.verdict.Detail)
			}
		})
	}
}

// TestSavedMirrorErrorNamesOnlyTheFile is the predicate that decides which capability an error may
// wear, over errors that are not stand-ins: a real store error from a file that cannot be opened,
// and the request path's own errors, each from the production code that answers with it.
func TestSavedMirrorErrorNamesOnlyTheFile(t *testing.T) {
	if err := storeOpenError(t); !savedMirrorError(err) {
		t.Errorf("the store's own error (%v) is the mirror's and the predicate does not see it", err)
	}
	for _, state := range requestPathErrors(t) {
		if savedMirrorError(state.err) {
			t.Errorf("%s (%v) was read as a failure of the mirror", state.name, state.err)
		}
	}
	if savedMirrorError(nil) {
		t.Error("no error at all was read as a failure of the mirror")
	}
}
