// Package tui is the shell: the Bubble Tea model the binary runs, the frame stack it navigates,
// and the boundary that keeps it unable to touch anything but the port it was handed.
//
// The discipline is the phase's. Update and View are pure functions of the model: every read is a
// command value (a closure returning a message), nothing here opens a socket, reads a file, runs a
// program or touches a database, and the only clock and the only http client in the program live
// outside this package. What the shell does own is the geometry — the frame stack, the anchor and
// the translation of screen.List's effects into commands — which is why the phase's honesty rules
// (the selection stays where the reader left it, a late page moves nothing, a stopped session
// issues nothing) are properties of values this package can be driven through with no terminal.
//
// It adds no words and no bytes of its own: the frame is exactly what internal/render produces
// from the state, the status line's fields are read off the list and the limiter, and every
// user-visible string is a constant in internal/render/copy.go or a row of internal/keys.
package tui

import (
	"context"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/media"
	"github.com/LZD121/x-terminal/internal/netcheck"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/screen"
)

// State is which of the phase's three display states the shell is in. They are exclusive and total,
// in the order panic > immersive > normal: normal is the plain reader, immersive is the same feed
// rendered through a cover, and panic replaces the whole pane with one constant frame.
//
// The vocabulary is the file's own words — internal/config accepts exactly these three strings for
// display.state — and the two are joined by one test (TestDisplayVocabularyAgrees) rather than by
// anything either package could keep in step by hand.
type State uint8

const (
	// StateNormal is the plain reader, byte for byte Phase 2's frame.
	StateNormal State = iota
	// StateImmersive is the same list under a cover: same rows, same selection, different grammar.
	StateImmersive
	// StateVivid is the second presentation: the same read-only data, laid out as a navigation
	// column, a list and a reading pane, with images drawn in the terminal. It is entered from the
	// normal state by the entry sequence and from nowhere else — it is deliberately absent from
	// StateOf, because StateOf is the config file's vocabulary and the file is not a way in.
	StateVivid
	// StatePanic is the one constant frame, and it is an overlay rather than a screen: it is entered
	// from either other state with one key and left with the same key.
	StatePanic
)

// String is the state's own name: the word the config file holds, so a value read from the file and
// a value written to it are the same word.
func (s State) String() string {
	switch s {
	case StateImmersive:
		return "immersive"
	case StateVivid:
		return "vivid"
	case StatePanic:
		return "panic"
	}
	return "normal"
}

// enterVivid is the one way into the vivid presentation. It has no counterpart in the config file
// and no flag: the design's rule is that the entry sequence is the only door, so this method is
// called by the matcher and by nothing else. The cover style is left alone — vivid mode wears no
// cover, and the remembered style is what a return to immersive will want.
func (m Model) enterVivid() Model {
	m.state = StateVivid
	return m
}

// leaveVivid is the ordinary way back to the plain reader, and it discards everything vivid-only:
// the marks and the image bookkeeping belong to the presentation that is leaving, not to the frame
// beneath it.
func (m Model) leaveVivid() Model {
	m.state = StateNormal
	m.vivid = vividState{}
	return m
}

// StateOf reads a state's name, and reports whether it is one at all. An unknown name — including
// the empty string — is "not a state", so a caller keeps the default it already had rather than
// guessing at a state this build has never heard of.
func StateOf(name string) (State, bool) {
	switch name {
	case "normal":
		return StateNormal, true
	case "immersive":
		return StateImmersive, true
	case "panic":
		return StatePanic, true
	}
	return StateNormal, false
}

// Display is what the shell starts in: the state the pane opens in and the cover style the
// immersive state wears. The config's two display keys resolve onto it, and it is a value the
// composition root builds because the file's vocabulary is not this package's.
type Display struct {
	State State
	Cover render.CoverStyle
}

// Term is the terminal-side effect port: a sink the shell hands values to and reads nothing back
// from. It matches internal/term.Sink's two methods, and it is declared here because the shell may
// not import the one package of the module that runs a program — the boundary test says so, and
// this interface is what makes the boundary crossable.
//
// It is only ever called from a command. Update and View are pure: a state change returns the
// command that performs its effects, and the command is what calls the port.
type Term interface {
	// EnterPanic replaces the pane: the frame's cells, then the scrollback's clear, then the title,
	// in one write, and then the multiplexer's own clear and the window name.
	EnterPanic(frame, title string)
	// SetWindow renames the multiplexer's window to the state's title, and does nothing on a bare
	// terminal.
	SetWindow(name string)
}

// Deps is everything the shell needs from outside itself: the port it reads through, the translation
// read, the two external facts a frame prints (the request budget and the clock), the mapper that
// turns an error into the state the screen shows, and the display's three — the terminal sink, the
// state the pane opens in and the writer of that state.
//
// Every field is required to be a value the shell may call. The shell defaults none of them: a nil
// source, a nil clock or a nil sink is a wiring mistake, and a shell that papered one over would be a
// shell whose frame claims a side effect that never happened — a panic that painted nothing, or a
// display state the next run does not start in. The three presence facts — Check, Saved and
// Translate — are the exception, and each says in its own comment what its absence means.
type Deps struct {
	// Source is the read side of x.com. The shell calls it from a command and nowhere else.
	Source domain.Source
	// Threads is the conversation read. The shell calls it from a command and nowhere else.
	Threads domain.Threads
	// Budget reports the requests the process issued and the answers that came back rate limited,
	// from the one limiter every request goes through, so the line cannot drift from the requests
	// actually issued. The reset it returns is the limiter's own spacing window, which is not the
	// account's rate limit and is deliberately not printed: the state line's reset is the
	// upstream's own, carried on the failure.
	Budget func() (used, limited int, reset time.Time)
	// Failure maps one error into the state the status line prints. It is the composition root's
	// (internal/cli/failure.go), so the words on screen and the classifier's own sentences cannot
	// drift apart.
	Failure func(error, []netcheck.Result) domain.Failure
	// Check runs the reachability check's unauthenticated stages — route, proxy, upstream — and
	// returns their report. The shell runs it from the request command when, and only when, the
	// mapper cannot name the error at all, because finding out that the network is down must not
	// cost a session and must not spend an account request: the stages it does run are the cheap
	// ones, and the client behind them is the one the pane reads through.
	//
	// A nil Check is a shell with no check to run: an error no mapper rule names then stays
	// unclassified, which is what an honest pane can still say.
	Check func(context.Context) []netcheck.Result
	// Now is the program's one clock: it stamps the frame, it is what the rate-limited state's
	// reset is read against, and it is injected so that a frame is reproducible.
	Now func() time.Time
	// Term is the terminal-side effect port every state change writes its side effects through. The
	// shell calls it from a command and nowhere else.
	Term Term
	// Display is the state and cover style the shell starts in, resolved from the config file by the
	// composition root. The zero value is the plain reader.
	Display Display
	// Persist writes the display state and cover style back to the config file. It is the shell's own
	// write, wired to internal/config.SetDisplay by the composition root, and the shell calls it only
	// from a command: a transition that wrote a file in Update would be a transition that blocks the
	// frame the user is waiting for.
	//
	// It is called by the deliberate transitions only. A panic entered by a key or by focus loss
	// writes nothing at all: an involuntary transition must not become tomorrow's start-up state, and
	// no file is written while someone is walking up.
	Persist func(state, cover string)
	// Saved is the mirror the composition root builds. The shell reads pages and status from it and
	// drives Begin/Step/Abandon for the pane's r; nil means no saved lists in this build.
	Saved domain.Mirror
	// SavedCapability is the fact this build can decide before any request: whether the list's
	// operation is in the registry. CapNone when the operation is present.
	SavedCapability func(kind domain.SavedKind) domain.Capability
	// SavedError maps an error from the mirror or the walk into a capability. CapNone means the
	// error is a network failure Deps.Failure should name instead.
	SavedError func(error) domain.Capability
	// MirrorPath is the file's path, which two of the three capability bodies print.
	MirrorPath string
	// Media is the renderer seam that opens attachments. Required like every other
	// field: a nil seam is a wiring mistake. The shell imports the interface, not a
	// process — os/exec lives on the far side of one tea.Cmd.
	Media media.Renderer
	// Clip is the REND-07 clipboard chain (OSC 52, then wl-copy). media.Clipboard
	// satisfies it; tests use *fake.Clipboard. A nil Clip is treated as NoClipboard.
	Clip interface {
		Copy(ctx context.Context, url string) media.ClipboardOutcome
	}
	// Translate is the translation read: one post's body in another language, as x.com's own
	// translation service answers it. It is called from a command and nowhere else, so the request
	// is issued where every other request is.
	//
	// A nil Translate is a shell with no translation endpoint — the Saved/Check precedent, a presence
	// fact rather than a wiring mistake that panics: the `T` arm then says so in the notice field and
	// spends nothing.
	Translate domain.Translator
	// TranslateLang is the language a translation is rendered in, resolved once by the composition
	// root (config file, then the locale, then the default). The shell carries it because the port
	// takes it as a parameter; it is printed nowhere — no frame, notice or error in this build
	// carries a configured value that is not a proper noun.
	//
	// An empty language is the same presence fact a nil port is: nothing is asked and the notice
	// field says why.
	TranslateLang string
}

// frameKind is which surface a frame shows: a home timeline, a conversation, or one query's
// results.
type frameKind uint8

const (
	frameHome frameKind = iota
	frameThread
	frameSearch
	frameProfile
	frameSaved
)

// frameKinds is every surface a frame can show, in the order the consts above declare them. It is
// the shell's own enumeration rather than a list a test keeps, and `cover_e2e_test.go`'s
// TestEveryFrameKindIsCheckedUnderACover is the join: that test's table has exactly one row per kind
// returned here, and each row walks the shipped keys to the surface and requires the covered frame
// of it to be the cover's own rendering (a different frame, no word of the product, its status
// field in the style's words). A kind added here without a row there fails that test, and a row for
// a kind this function does not return fails it too, so neither list can grow past the other.
//
// It is a function and not a map for the reason render.CoverStyles is: the order is a fact about
// the program and not about a hash table.
func frameKinds() []frameKind {
	return []frameKind{frameHome, frameThread, frameSearch, frameProfile, frameSaved}
}

// frame is one screen state on the back stack. ID is this instance's own name — unique for the
// life of the process so an answer finds the frame that asked and not a later one of the same
// kind — and Kind says which of the two surfaces it holds. The posts live in List or Thread, the
// place the pane is at is one of two facts: the frame has just opened and shows its own first row,
// or the reader has moved and Anchor — the selected block's head row's distance from the pane's
// first row, in the rows of the grammar the frame is painted with, measured with blocksAt and never
// with a grammar the pane does not wear — is where they left it. Read records that an answer has
// come back for this frame, which is what tells an empty frame that was never read from an empty
// read that succeeded.
type frame struct {
	ID      string
	Kind    frameKind
	List    screen.List
	Thread  screen.Thread
	Search  screen.Results[domain.Post, domain.PostPage]
	People  screen.Results[domain.User, domain.UserPage]
	Profile screen.Profile
	Saved   screen.Saved
	// Moved records that the reader has moved the selection in this frame: from the first such move
	// the anchor below is the reader's place, and a page, a refresh, a cover round trip or a resize
	// never changes it. Until then the pane presents the frame's own first row (model.go's skip),
	// which is what puts a profile's header — and a conversation's ancestors — on screen when the
	// frame opens; the flag is not stored as a second anchor, because the two facts are one: the
	// pane is either at the frame's top or at the place the anchor names.
	Moved bool
	// Anchor is the selected block's head row's distance from the pane's first row, in the rows of
	// the grammar the frame is painted with. It is the reader's place from their first move on, and
	// it is never read while Moved is false.
	Anchor int
	Read   bool
	// Fall is the per-post URL fallback map for this frame. It is per frame, not
	// per process: a frame the user returns to is byte-identical to the frame they
	// left (P3), and a fallback belongs to the post it was raised for. Nothing
	// removes an entry but the next successful media action on that post.
	Fall render.Fallbacks
	// syncRun is the in-flight Begin/Step walk for a saved frame. It lives on the
	// frame so a pop can Abandon it; it is not part of the pure state machine.
	syncRun domain.SyncRun
}

// selectionOf is the row the cursor is on, as the renderer's own value: the screen's Target converted at
// the seam, so internal/render knows nothing about how the walk names and keys a row and this package
// knows nothing about how a row is painted.
func selectionOf(f frame) render.Selection {
	sel := f.selected()
	out := render.Selection{ID: domain.PostID(sel.ID)}
	for _, id := range sel.Path {
		out.Path = append(out.Path, domain.PostID(id))
	}
	return out
}

// selected is the row the cursor is on in this frame.
func (f frame) selected() screen.Target {
	switch f.Kind {
	case frameThread:
		return f.Thread.Selected()
	case frameSearch:
		if f.isPeople() {
			return f.People.Walk().Selected()
		}
		return f.Search.Walk().Selected()
	case frameProfile:
		return f.Profile.Walk().Selected()
	case frameSaved:
		return f.Saved.List().Selected()
	}
	return f.List.Selected()
}

// targets is the frame's own row sequence.
func (f frame) targets() []screen.Target {
	switch f.Kind {
	case frameThread:
		return f.Thread.Targets()
	case frameSearch:
		if f.isPeople() {
			return screen.UserTargets(f.People.Walk().Rows())
		}
		return screen.PostTargets(f.Search.Walk().Rows())
	case frameProfile:
		return screen.PostTargets(f.Profile.Walk().Rows())
	case frameSaved:
		return f.Saved.List().Targets()
	}
	return f.List.Targets()
}

// post resolves the post a target names in this frame.
func (f frame) post(t screen.Target) (domain.Post, bool) {
	switch f.Kind {
	case frameThread:
		return f.Thread.Post(t)
	case frameSearch:
		return screen.PostFor(f.Search.Walk().Rows(), t)
	case frameProfile:
		return screen.PostFor(f.Profile.Walk().Rows(), t)
	case frameSaved:
		return f.Saved.List().Post(t)
	}
	return f.List.Post(t)
}

// isPeople reports whether this results frame is the people product. The people
// instantiation lives beside Search on the same frame kind so the pane dispatch,
// the back stack and the key applicability stay one path.
func (f frame) isPeople() bool {
	return f.Kind == frameSearch && f.People.Product() == domain.People
}

// account resolves the account a target names in a people results frame.
func (f frame) account(t screen.Target) (domain.User, bool) {
	if !f.isPeople() {
		return domain.User{}, false
	}
	for _, u := range f.People.Walk().Rows() {
		if u.ID == t.ID {
			return u, true
		}
	}
	return domain.User{}, false
}

// screenName is the keys table's screen for this frame kind.
func screenName(kind frameKind) string {
	switch kind {
	case frameThread:
		return "thread"
	case frameSearch:
		return "search"
	case frameProfile:
		return "profile"
	case frameSaved:
		return "home"
	}
	return "home"
}

// Model is the shell's whole state: the frame stack, the size of the pane, the display the pane is
// in, the three things the UI-SPEC says are the process's rather than a frame's — the sticky
// failure, the refresh notice and the in-flight bookkeeping — and the session's memory of what the
// translation service answered, which is a property of the selection rather than of any frame.
type Model struct {
	deps   Deps
	frames []frame
	// seq is the last frame id handed out. Ids are never reused, so an answer carrying an id the
	// stack no longer holds belongs to a frame the user has left, and it is dropped.
	seq  uint64
	help bool
	// prompt is the query field while it is open, and nil when it is not. It is model state beside
	// the help overlay, not a frame: nothing is pushed while it is open, no request is issued, and
	// the status line is composed exactly as before, which is what keeps the budget on screen.
	prompt *screen.Prompt
	w, h   int

	// state is which of the three display states the pane is in, and style the cover it wears when
	// it is immersive — remembered in normal too, where it is not worn, because the key that puts
	// the cover on has to know which one to put on.
	state State
	style render.CoverStyle
	// from is the state panic was entered from, and it is what leaving panic restores. It is a
	// remembered state rather than a stack because panic is an overlay of exactly one level: a
	// second blur while panicked enters nothing, and there is no second panic to nest.
	from State
	// entryPending is a start-up panic whose entry has not run yet. The frame is built from the
	// pane's size, so the sequence waits for the first size the terminal reports and runs after the
	// first frame: nothing above the frame may remain reachable either, because the shell line that
	// launched this program is above it.
	entryPending bool

	// failure is the state the status line prints, sticky until a fetch succeeds: it is the
	// process's, so a frame the user returns to still reports a failure raised while another frame
	// was on top, and no timer clears it.
	failure domain.Failure
	// notice is the process's announcement of the answer to an action the user just
	// took (a refresh, a media open, a "does not apply"), and noticeCount the number
	// it counted. Phase 2's rule: it is the first field dropped under width pressure.
	notice      render.NoticeKind
	noticeCount int

	// pending is how many requests this process has issued and not yet seen an answer for. A
	// count rather than a latch: a feed switch can leave two requests out — the limiter is what
	// keeps them one at a time on the wire — and both the shell's debounce and the pane's loading
	// line are read from it. It is process-wide: an answer is routed by frame id, not by feed slot.
	pending int

	// translations is the session's memory of what the translation service answered, keyed by post
	// id. It is the whole of the translated-vs-original choice: a row renders translated exactly
	// when it is the row the selection names AND its id is in this map, so moving the selection off
	// that post reverts it by construction — there is no second fact that could fall out of step —
	// and re-reaching it spends nothing, because the map is looked up before any request is built.
	//
	// It is memory only, session-scoped, and written nowhere: the audit's persistence check still
	// enumerates its six destinations and this adds none. What it holds is the answer, not a flag,
	// so "asked, and the service carries nothing for this post" is remembered as itself rather than
	// as an empty translation.
	//
	// Model is copied by value everywhere, so the map is never mutated in place: an answer builds a
	// fresh map with withTranslation, the way setTop writes a frame back by index rather than
	// through a pointer into a slice it may append to. A model value held before an answer therefore
	// still renders the original.
	translations map[domain.PostID]translation
	// vivid is everything the vivid presentation owns and nothing else does: the marks it draws on
	// the list, the nav destination it is showing, and the images it has placed. It is a value, so a
	// model copy taken before an answer still renders the previous presentation, and leaving vivid
	// mode zeroes it — the marks and the image ids are not facts about the plain reader.
	vivid vividState
	// konami is the entry sequence's progress, at rest in every state but the two it watches.
	konami konami
}

// translation is this package's own value for one answered post: the body the service gave, and
// whether it gave one at all. A string-valued map would have to say "asked, none available" with an
// empty string, and the pane must not have to look the difference between three states up in two
// places.
type translation struct {
	// Text is the translated body. It is non-empty exactly when Available is true.
	Text string
	// Available reports whether the service answered with a translation of the post asked for. It is
	// never the difference between "no answer" and "an empty answer": a refusal crosses the seam as
	// an error and reaches the state field, not this value.
	Available bool
}

// withTranslation returns the model with one post's answer remembered. The map is rebuilt rather
// than written through, because Model is a value: a model held before the answer — by a test, or by
// a command that captured one — is left rendering exactly what it rendered.
func (m Model) withTranslation(id domain.PostID, tr domain.Translation) Model {
	next := make(map[domain.PostID]translation, len(m.translations)+1)
	for k, v := range m.translations {
		next[k] = v
	}
	next[id] = translation{Text: tr.Text, Available: tr.Available}
	m.translations = next
	return m
}

// New returns the shell at its start: one frame, the Following list, nothing loaded, in the display
// state and cover style the composition root resolved from the config file. The algorithmic feed's
// frame does not exist yet — the toggle creates it — and the process opens by asking for page one of
// Following, which is why that request is counted as outstanding from here: Init is where it goes
// out, and Init has no model to write to.
//
// A start-up state of panic paints the frame from the first View and commands the entry once the
// terminal has said how big the pane is: the frame is built from the pane's size, so there is
// nothing to write before there is a size to write into.
func New(d Deps) Model {
	m := Model{
		deps:    d,
		seq:     1,
		frames:  []frame{{ID: homeID(domain.Following, 1), Kind: frameHome, List: screen.List{Kind: domain.Following}}},
		state:   d.Display.State,
		style:   d.Display.Cover,
		pending: 1,
	}
	if m.state == StatePanic {
		// The state is already panic, so the first frame the renderer paints is the frame; the entry
		// itself is what waits.
		m.entryPending = true
		m.from = StateNormal
	}
	return m
}

// failureOf maps one failed request into the state the status line prints.
//
// The mapper is the only classifier and is asked first: an error it names — a classified answer, a
// spent budget, a missing session — is already a state, and no request is spent asking the network
// about a path that is not what failed. Only an error it cannot name at all sends the reachability
// check out, whose unauthenticated stages name the broken link, so the pane shows `no route to
// x.com` rather than an unclassified transport error. A check that reports nothing leaves the
// mapper's own answer standing.
func (m Model) failureOf(err error) domain.Failure {
	failure := m.deps.Failure(err, nil)
	if m.deps.Check == nil || failure.Kind != domain.FailureUnclassified {
		return failure
	}
	return m.deps.Failure(err, m.deps.Check(context.Background()))
}

// frameIndexID is the stack index of the frame instance with this id, or -1 when the stack no
// longer holds it: the answer's frame was discarded before the answer arrived.
func (m Model) frameIndexID(id string) int {
	for i, f := range m.frames {
		if f.ID == id {
			return i
		}
	}
	return -1
}

// homeID names one home-frame instance.
func homeID(kind domain.HomeKind, seq uint64) string {
	return "home:" + kind.String() + "#" + strconv.FormatUint(seq, 10)
}

// threadID names one conversation-frame instance.
func threadID(focal domain.PostID, seq uint64) string {
	return "thread:" + string(focal) + "#" + strconv.FormatUint(seq, 10)
}

// searchID names one results-frame instance: the query and product it searches, and the sequence
// number that keeps two frames of one query distinct. The query is in the id because the id names
// the frame; it is memory only and is never written anywhere.
func searchID(query string, p domain.SearchProduct, seq uint64) string {
	return "search:" + p.String() + ":" + query + "#" + strconv.FormatUint(seq, 10)
}

// profileID names one profile-frame instance.
func profileID(handle string, seq uint64) string {
	return "profile:" + handle + "#" + strconv.FormatUint(seq, 10)
}

// savedID names one saved-frame instance.
func savedID(kind domain.ListKind, seq uint64) string {
	return "saved:" + kind.String() + "#" + strconv.FormatUint(seq, 10)
}

// WithFailure returns the model with a failure already on it: the state a feed opens in when there
// is nothing to read with at all — no session file, say.
//
// It is the same state a failureMsg leaves behind, reached without a request: the frames the shell
// already has are told, so their own gates and the process's agree, and no request is left counted
// as outstanding, because the count is what the debounce reads and the request the shell will not
// make is the same fact.
func (m Model) WithFailure(f domain.Failure) Model {
	m.failure = f
	if !m.requestAllowed(m.deps.Now()) {
		m.pending = 0
	}
	for i := range m.frames {
		next, _ := m.frames[i].List.Update(screen.PageFailed{Failure: f}, m.deps.Now())
		m.frames[i].List = next
	}
	return m
}

// Init is the first request: page one of the Following list. It is the only page the shell asks for
// outside its own Update — screen.List's pagination trigger needs a cursor, and page one has none —
// and it is not issued at all when the state already forbids a request (a session that is not there,
// a soft ban preset at construction).
func (m Model) Init() tea.Cmd {
	if !m.requestAllowed(m.deps.Now()) {
		return nil
	}
	return m.fetchPage(m.frames[0].ID, "", domain.Following, false)
}

// requestAllowed reports whether the process may issue a request at now.
//
// The rules are screen.List's own (absorb.go's blocked), read one level up, and they are here
// because the failure state is the process's rather than a frame's: a feed switch puts a frame on
// screen that never saw the failure, and OPS-04's "stop for the session" has to hold there too.
// Phase 4 replaces the kind-keyed frames with opaque frame ids; until then this is the one place
// the process-level gate lives.
func (m Model) requestAllowed(now time.Time) bool {
	switch {
	case m.failure.Kind == domain.FailureNone:
		return true
	case m.failure.Stopped:
		// A soft ban stops the session: no page and no refresh again for the life of the process.
		return false
	case m.failure.Kind == domain.FailureNoSession:
		// There is no session to read with: a request would be refused before it left the machine,
		// and the state line already names what to write.
		return false
	case m.failure.Retry:
		return true
	case m.failure.Kind == domain.FailureRateLimited:
		// The one state with an end the upstream itself names: unblocked at its reset. A zero reset
		// is "no retry; reset unknown" — a wait with no end to wait for — and stays blocked.
		return !m.failure.Reset.IsZero() && !now.Before(m.failure.Reset)
	default:
		return false
	}
}

// outstanding is how many requests the process has issued and not yet been answered for.
func (m Model) outstanding() int { return m.pending }

// top is the frame that renders: the last one pushed.
func (m Model) top() frame { return m.frames[len(m.frames)-1] }

// setTop replaces the top frame. A frame is a value in the stack, so the shell writes one back by
// index rather than holding a pointer into a slice it may append to.
func (m Model) setTop(f frame) Model {
	m.frames[len(m.frames)-1] = f
	return m
}

// paneRows is how many rows the pane has: the frame's last row is the status line.
func (m Model) paneRows() int { return m.h - 1 }

// clamp keeps v inside [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// blocksIn is the frame's own block list at the pane's width, in the grammar handed in: one
// conversation's blocks or one list's. It is the only place the pane's rows are built, and the
// grammar is a parameter rather than a reading of the state, so the rows a frame measures and the
// rows it paints are one value and no call site can choose between two of them.
//
// The rows are the block builder's own output at the pane's current width, so a resize re-measures
// the list rather than reusing a measurement: no height and no wrap point is cached anywhere.
//
// Each arm's posts pass through the translation swap on their way in — translated,
// translatedConversation — which is where the translated-vs-original choice is applied (translate.go).
// With nothing remembered for the selected row, `translated` hands back the slice it was given: the
// ordinary path allocates nothing, and every stored frame stays byte-identical.
func (m Model) blocksIn(g render.Grammar, f frame, w int) []render.Block {
	falls := m.fallbacksFor(f)
	sel := selectionOf(f)
	switch f.Kind {
	case frameThread:
		return render.ThreadBlocks(m.translatedConversation(conversationOf(f.Thread), sel), g, w, sel, m.deps.Now(), falls)
	case frameSearch:
		if f.isPeople() {
			sel := ""
			if t := f.People.Walk().Selected(); t.ID != "" {
				sel = t.ID
			}
			// A people row is an account: nothing about it is a post, so no translation can apply
			// to it — the `T` arm reports that there is nothing to translate on this frame.
			return render.UserRows(f.People.Walk().Rows(), g, w, sel)
		}
		return render.Blocks(m.translated(f.Search.Walk().Rows(), sel), g, w, sel, m.deps.Now(), falls)
	case frameProfile:
		header, ok := f.Profile.Header()
		if !ok {
			return nil
		}

		return render.Profile(header, m.translated(f.Profile.Walk().Rows(), sel), g, w, sel, m.deps.Now())
	case frameSaved:
		return render.Blocks(m.translated(f.Saved.List().Posts(), sel), g, w, sel, m.deps.Now(), falls)
	}
	return render.Blocks(m.translated(f.List.Posts(), sel), g, w, sel, m.deps.Now(), falls)
}

// fallbacksFor is the per-post URL fallback map the pane hands the composer: the frame's own in the
// plain reader, and the zero map under a cover.
//
// DSGT-09: a fallback's rows are the URL of a thing that could not be opened — a pbs.twimg.com or
// x.com address — and a covered screen may not carry one. The state keeps the map either way (a frame
// the user returns to is byte-identical, so nothing here is dropped from the model); the drop is at
// the seam, and it is the *state's* rule rather than each cover grammar happening not to print an
// attachment row. That distinction is the whole of this function: every cover this build ships sets
// AttachmentRow false, so the rows would not appear even if the map were handed over, and a rule that
// rested on that would stop holding the day a cover grew one. TestCoveredPaneDropsTheFallbackMapByState
// is the arm that keeps it a rule.
func (m Model) fallbacksFor(f frame) render.Fallbacks {
	if m.state == StateNormal {
		return f.Fall
	}
	return nil
}

// blocksAt is blocksIn in the grammar the pane is painted with — the cover the state wears, or the
// plain reader's. view.pane paints these rows and the geometry below measures these rows, which is
// what keeps the measuring grammar and the painting grammar the same value by construction.
func (m Model) blocksAt(f frame, w int) []render.Block { return m.blocksIn(m.grammar(), f, w) }

// geometry is the list's own geometry over a block list: the head row of every block, in pane rows
// counted from the list's first row, plus the index of the block the selection is in (or -1 when the
// list does not hold it).
//
// It takes the blocks rather than a frame because the rows it measures have to be the rows the pane
// paints. A cover is not the plain reader's height for the same conversation — a git record is
// taller than a plain post — and a pane that measured one grammar's rows while painting another's
// would scroll the cursor out of its own pane.
//
// The unit is the head row and not the block's first row, and the difference is one row per block:
// a block that follows another begins with the blank separator row between them, and the anchor is
// the selected *head*'s distance from the pane's first row. The separator is the empty row the block
// builder writes, so the head is the block's first non-empty row.
func geometry(blocks []render.Block, f frame) (rows []int, selected int) {
	rows, selected, at := make([]int, len(blocks)), -1, 0
	selTop := domain.PostID(f.selected().Top().ID)
	for i, block := range blocks {
		head := at
		for head < at+len(block.Rows) && block.Rows[head-at] == "" {
			head++
		}
		rows[i] = head
		if block.PostID == selTop {
			selected = i
		}
		at += len(block.Rows)
	}
	return rows, selected
}

// conversationOf is the renderer's view of a thread frame.
func conversationOf(t screen.Thread) render.Conversation {
	return render.Conversation{
		Ancestors:     t.Ancestors(),
		Focal:         t.Focal(),
		Continued:     t.Continued(),
		Replies:       t.Replies(),
		Loading:       t.Loading(),
		Read:          t.Read(),
		Stalled:       t.PartialStopped(),
		NoReplies:     t.Read() && len(t.Replies()) == 0 && !t.PartialHidden(),
		PartialAbove:  t.PartialAbove(),
		PartialHidden: t.PartialHidden(),
	}
}

// rowsAbove is how many pane rows the list has above the selected block's head: the head row's own
// offset, which is every row of every block before it plus the selected block's separator row. The
// selection is a post the list holds whenever the list holds posts — screen.List establishes that
// with the first page — so the walk always reaches it.
//
// The block the selection is inside is the one its top-level id names, which is what makes the reader's
// place stay put when the cursor steps into a quoted post: the quote is a row of the block above it.
//
// The rows are blocksAt's — the pane's own, in the grammar the pane is painted with — so the
// geometry the shell anchors against is the geometry the reader sees.
func (m Model) rowsAbove(f frame, w int) int {
	rows, selected := geometry(m.blocksAt(f, w), f)
	if selected < 0 {
		return 0
	}
	return rows[selected]
}

// scroll is how many rows of the list sit above the pane's first row: the pane's window into a list
// taller than it. It is derived, never stored — skip is the one rule — and it is read by move, which
// keeps the scroll position a selection move leaves the pane at.
func (m Model) scroll(f frame, w int) int { return m.skip(f, m.rowsAbove(f, w)) }

// skip is how many rows of the list the pane drops above its first row, measured from the selected
// head's own offset: the opening rule while the reader has not moved in this frame, and the anchor's
// rule from their first move on. The two rules are one function because the pane the shell measures
// (scroll, in move) and the pane it paints (scrolled, in view) have to be the same pane; the anchor
// is never read while the frame is unmoved.
//
// The opening rule is the phase's: the offset is 0 at the frame's top, so a frame that has just
// opened — or whose content a first page has just replaced, the frame being unmoved in both cases —
// presents its own first row at the pane's first row whatever the selection does below it. That is
// what puts a profile's header, and a conversation's ancestors, on screen: the frame composes the
// header as its first block while the selection is the first post, so measuring only from the
// selection would drop the header the frame had just built (05-UI-SPEC: "the offset is 0 at the
// frame's top and the header is simply the first block there"). When the selected head is below the
// pane's last row at that offset, the head is the pane's first row instead — 04-UI-SPEC's one rule
// for where a frame opens, so that a frame never opens with its selection off screen (an ancestor
// chain taller than the pane). The two cases agree wherever the frame holds nothing above the
// selection, which is why no plain timeline frame moves under it.
//
// A list shorter than the anchor's promise cannot be scrolled above its own first block, which is
// what the anchor rule's floor says.
func (m Model) skip(f frame, above int) int {
	if !f.Moved {
		if above < m.paneRows() {
			return 0
		}
		return above
	}
	return max(0, above-f.Anchor)
}

// reanchor is the anchor rule. A selection move keeps the scroll position the pane already had —
// the rows above the pane's first row do not change because the reader pressed `j` — so the new
// anchor is where the newly selected block's head row falls from there, clamped into the pane: an
// anchor outside [0, paneRows-1] would either hide the selection above the pane's first row or ask
// the frame to scroll past the top of the list, and render.Frame does neither.
func (m Model) reanchor(f frame, scroll, w int) int {
	return clamp(m.rowsAbove(f, w)-scroll, 0, max(0, m.paneRows()-1))
}
