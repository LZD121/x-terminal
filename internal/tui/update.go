package tui

import (
	"context"
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/keys"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/screen"
)

// pageMsg is one answer to a request the shell issued.
//
// Frame names the frame instance the request was for and Kind the feed it was for. The frame is
// what the answer is matched on: the toggle discards a frame and pushes another of the same feed
// kind, and an answer that finds no frame of that id is dropped rather than absorbed by the frame
// that replaced it. Refresh travels with the answer because it decides which event absorbs it: a
// refresh's page is prepended and a page's is appended (screen.Refreshed against screen.PageFetched),
// and screen.List keeps that latch to itself. Cursor is the position the request was made with: it is
// carried so that an answer can be named for what it was a request for.
type pageMsg struct {
	Frame   string
	Page    domain.PostPage
	Cursor  domain.Cursor
	Refresh bool
}

// threadMsg is one answer to a conversation request the shell issued.
type threadMsg struct {
	Frame   string
	Page    domain.ThreadPage
	Cursor  domain.Cursor
	Refresh bool
}

// searchMsg is one answer to a search request the shell issued: the frame that asked, the page,
// and whether it was a refresh — the same shape as the other two answers, because the walk that
// absorbs it is the same walk.
type searchMsg struct {
	Frame   string
	Page    domain.PostPage
	Cursor  domain.Cursor
	Refresh bool
}

// peopleMsg is one answer to a people-search request: the same shape as searchMsg over accounts.
type peopleMsg struct {
	Frame   string
	Page    domain.UserPage
	Cursor  domain.Cursor
	Refresh bool
}

// profileHeaderMsg is the header answer for a profile frame.
type profileHeaderMsg struct {
	Frame  string
	Header domain.Profile
}

// profilePostsMsg is one posts-page answer for a profile frame.
type profilePostsMsg struct {
	Frame   string
	Page    domain.PostPage
	Cursor  domain.Cursor
	Refresh bool
}

// failureMsg is a request that did not deliver data. Frame names the frame instance that asked, so
// the failure reaches the frame that issued the request and no other; the process's own state is
// set from Failure whatever frame is left. Failure is the mapper's answer, reached in the command
// that issued the request (Model.failureOf) rather than here, because naming a broken path may run
// the reachability check, and the check is a request like any other: it is made where requests are.
type failureMsg struct {
	Frame   string
	Failure domain.Failure
}

// fetchPage returns the command that issues one home request. This is one of the shell's two ways
// to a source: nothing in Update or View calls it, and every answer arrives as a pageMsg or a
// failureMsg.
//
// The context is Background because the shell has no cancellation to offer: one request is bounded
// by the transport's own timeout and by the limiter, and a quit that arrives while one is out ends
// the program rather than the request. A request that is answered after the program is gone is
// simply delivered to nobody.
func (m Model) fetchPage(id string, c domain.Cursor, k domain.HomeKind, refresh bool) tea.Cmd {
	return func() tea.Msg {
		page, err := m.deps.Source.Home(context.Background(), k, c)
		if err != nil {
			return failureMsg{Frame: id, Failure: m.failureOf(err)}
		}
		return pageMsg{Frame: id, Page: page, Cursor: c, Refresh: refresh}
	}
}

// fetchThread returns the command that issues one conversation request.
func (m Model) fetchThread(id string, focal domain.PostID, c domain.Cursor, refresh bool) tea.Cmd {
	return func() tea.Msg {
		page, err := m.deps.Threads.Thread(context.Background(), focal, c)
		if err != nil {
			return failureMsg{Frame: id, Failure: m.failureOf(err)}
		}
		return threadMsg{Frame: id, Page: page, Cursor: c, Refresh: refresh}
	}
}

// fetchSearch returns the command that issues one search request: the query, the product and the
// cursor are the request's own, and the answer arrives as a searchMsg — or as a failureMsg named
// by the same mapper the other two reads use.
func (m Model) fetchSearch(id string, req screen.SearchRequest, refresh bool) tea.Cmd {
	return func() tea.Msg {
		page, err := m.deps.Source.Search(context.Background(), req.Query, req.Product, req.Cursor)
		if err != nil {
			return failureMsg{Frame: id, Failure: m.failureOf(err)}
		}
		return searchMsg{Frame: id, Page: page, Cursor: req.Cursor, Refresh: refresh}
	}
}

// fetchPeople returns the command that issues one SearchPeople request.
func (m Model) fetchPeople(id string, req screen.PeopleRequest, refresh bool) tea.Cmd {
	return func() tea.Msg {
		page, err := m.deps.Source.SearchPeople(context.Background(), req.Query, req.Cursor)
		if err != nil {
			return failureMsg{Frame: id, Failure: m.failureOf(err)}
		}
		return peopleMsg{Frame: id, Page: page, Cursor: req.Cursor, Refresh: refresh}
	}
}

func (m Model) fetchProfile(id string, handle string) tea.Cmd {
	return func() tea.Msg {
		header, err := m.deps.Source.Profile(context.Background(), handle)
		if err != nil {
			return failureMsg{Frame: id, Failure: m.failureOf(err)}
		}
		return profileHeaderMsg{Frame: id, Header: header}
	}
}

func (m Model) fetchUserPosts(id string, req screen.UserPostsRequest, refresh bool) tea.Cmd {
	return func() tea.Msg {
		page, err := m.deps.Source.UserPosts(context.Background(), req.ID, req.Cursor)
		if err != nil {
			return failureMsg{Frame: id, Failure: m.failureOf(err)}
		}
		return profilePostsMsg{Frame: id, Page: page, Cursor: req.Cursor, Refresh: refresh}
	}
}

// Update is the tea.Model entry point: it applies one message and returns the next model with the
// command that message asks for.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	return next, cmd
}

// update is Update with the shell's own type, which is what the tests drive. A message this build
// does not know is a no-op: a shell that guessed at an unknown event is a shell whose tests prove
// nothing.
//
// The two focus messages are the terminal's own report of the window's focus. Losing it is T8 and
// is the same entry the key asks for; regaining it is T9 and changes nothing, because un-covering is
// only safe as a deliberate act. A terminal that does not report focus simply never sends either
// message — the model never enters panic by itself, nothing on screen claims it will, and the key
// still works. (Under a multiplexer the events reach a pane only when the server's own
// `focus-events` option is on, and this program does not turn it on: it is a server-wide setting.)
func (m Model) update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		return m.press(msg)
	case tea.BlurMsg:
		return m.enterPanic()
	case tea.FocusMsg:
		return m, nil
	case pageMsg:
		return m.answered(msg), nil
	case threadMsg:
		return m.threadAnswered(msg), nil
	case searchMsg:
		return m.searchAnswered(msg), nil
	case peopleMsg:
		return m.peopleAnswered(msg), nil
	case profileHeaderMsg:
		return m.profileHeaderAnswered(msg)
	case profilePostsMsg:
		return m.profilePostsAnswered(msg), nil
	case failureMsg:
		return m.failed(msg), nil
	case savedPageMsg:
		return m.savedPageAnswered(msg), nil
	case savedStatusMsg:
		return m.savedStatusAnswered(msg), nil
	case savedFailureMsg:
		return m.savedFailed(msg)
	case syncStartedMsg:
		return m.syncStarted(msg)
	case syncStepMsg:
		return m.syncStepped(msg)
	case mediaMsg:
		return m.mediaAnswered(msg)
	case usersMsg:
		return m.accountsAnswered(msg), nil
	case mediaTabMsg:
		return m.mediaTabAnswered(msg), nil
	case mediaExitMsg:
		return m.mediaExitAnswered(msg), nil
	case copyLinkMsg:
		return m.copyLinkAnswered(msg), nil
	case openPostMsg:
		return m.openPostAnswered(msg)
	case translatedMsg:
		return m.translateAnswered(msg), nil
	}
	return m, nil
}

// press routes one key press through the one table.
//
// While panicked exactly two bindings exist: `ctrl+p` leaves and `ctrl+c` quits. Every other key —
// `q`, `esc`, `?` and every movement key among them — is consumed with a no-op, so an accidental
// `esc` can never uncover, and the help overlay does not open. A static frame that reacted to a key
// would be stranger than one that does not, and a hung app must stay quittable.
//
// While the help overlay is open the keymap is consulted for one thing only: `help` and `back`
// close the overlay and every other action is ignored, which is what shadowing means here — no key
// reaches the list under it.
func (m Model) press(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.state == StatePanic {
		action, ok := keyAction(msg, screenName(m.top().Kind))
		if !ok {
			return m, nil
		}
		switch action {
		case keys.ActionPanic:
			return m.leavePanic()
		case keys.ActionQuit:
			return m, tea.Quit
		}
		return m, nil
	}
	if m.prompt != nil {
		// While the prompt is open every printable key is text — `?`, `/`, `u`, `s` and `a`
		// among them — and the keymap is shadowed except for the field's own five keys. The
		// lookup is deliberately not consulted first: an unbound letter is still text, and a
		// bound one (`g`, `c`, `r`) must not reach its action.
		return m.promptKey(msg)
	}
	// The entry sequence is consulted before the keymap, in the two states where a key means what
	// the table says. It holds at most one key, and only after eight arrows have arrived in order:
	// the eight themselves pass through, because movement costs nothing and a reader must not feel
	// a delay for it. See konami.go.
	if m.state == StateNormal || m.state == StateVivid {
		var effect konamiEffect
		m.konami, effect = m.konami.advance(msg.Keystroke())
		switch effect {
		case konamiHeld:
			return m, nil
		case konamiEnter:
			m = m.enterVivid()
			return m, m.change()
		case konamiReplay:
			// The held key was bookmarks all along. Replaying it goes back through press with a
			// `b`, which cannot re-enter this arm: a `b` with nothing matched is an ordinary key.
			held, cmd := m.press(tea.KeyPressMsg(tea.Key{Code: 'b', Text: "b"}))
			m = held
			if action, ok := keyAction(msg, screenName(m.top().Kind)); ok {
				state, second := m.dispatch(action)
				return state, tea.Batch(cmd, second)
			}
			return m, cmd
		}
	}
	action, ok := keyAction(msg, screenName(m.top().Kind))
	if !ok {
		return m, nil
	}
	// The vivid presentation's `esc` is its own way out, and it is not the plain reader's `back`:
	// one press closes the reading column, the next leaves the presentation. `q` keeps the plain
	// reader's meaning, so quitting from the first frame still quits.
	if m.state == StateVivid && (msg.Keystroke() == "esc" || msg.String() == "esc") {
		if m.vivid.reading != "" {
			m.vivid.reading = ""
			return m, nil
		}
		m = m.leaveVivid()
		return m, m.change()
	}
	if m.help {
		if action == keys.ActionHelp || action == keys.ActionBack {
			m.help = false
		}
		return m, nil
	}
	// These are the actions that would hand a string identifying the real upstream
	// to a program outside this process or to the terminal's clipboard, so they are
	// refused by what they would reveal, not by which URL form they happen to carry.
	if m.state != StateNormal && coverBreaking(action) {
		return m.setNotice(render.NoticeGated, 0), nil
	}

	return m.dispatch(action)
}

// dispatch is the armed switch: one action, one effect. press reaches it for an ordinary key, and
// once more for a replayed one — the held key is dispatched without a second copy of the switch,
// which is what keeps the dispatch join's one-arm-per-action rule true.
func (m Model) dispatch(action keys.Action) (Model, tea.Cmd) {
	switch action {
	case keys.ActionQuit:
		return m, tea.Quit
	case keys.ActionHelp:
		m.help = true
		return m, nil
	case keys.ActionBack:
		return m.back()
	case keys.ActionSwitchList:
		return m.toggleFeed()
	case keys.ActionSearchPosts:
		return m.openPrompt(domain.Latest)
	case keys.ActionSearchPeople:
		return m.openPrompt(domain.People)
	case keys.ActionSearchOrder:
		return m.switchOrder()
	case keys.ActionOpen:
		return m.open()
	case keys.ActionOpenAuthor:
		return m.openAuthor()
	case keys.ActionRefresh:
		return m.refresh()
	case keys.ActionBookmarks:
		return m.savedAction(action)
	case keys.ActionLikes:
		return m.savedAction(action)
	case keys.ActionMedia:
		return m.openMedia()
	case keys.ActionCopyLink:
		return m.copyLink()
	case keys.ActionOpenPost:
		return m.openPost()
	case keys.ActionTranslate:
		return m.translate()
	case keys.ActionFollowing:
		return m.pushAccounts(domain.FollowingList)
	case keys.ActionFollowers:
		return m.pushAccounts(domain.FollowersList)
	case keys.ActionVerifiedFollowers:
		return m.pushAccounts(domain.VerifiedFollowersList)
	case keys.ActionProfileMedia:
		return m.pushMedia()
	case keys.ActionCover:
		return m.toggleCover()
	case keys.ActionNextCover:
		return m.wear(stepCover(m.style))
	case keys.ActionCover1:
		return m.numberedCover(action)
	case keys.ActionCover2:
		return m.numberedCover(action)
	case keys.ActionCover3:
		return m.numberedCover(action)
	case keys.ActionCover4:
		return m.numberedCover(action)
	case keys.ActionPanic:
		return m.enterPanic()
	case keys.ActionNext:
		return m.move(action)
	case keys.ActionPrev:
		return m.move(action)
	case keys.ActionHalfDown:
		return m.move(action)
	case keys.ActionHalfUp:
		return m.move(action)
	case keys.ActionPageDown:
		return m.move(action)
	case keys.ActionPageUp:
		return m.move(action)
	case keys.ActionFirst:
		return m.move(action)
	case keys.ActionLast:
		return m.move(action)
	}
	return m, nil
}

// The set the dispatch join holds against the keymap is only as good as its weakest link, and
// this is it: the arm below, planted in the switch above as
// `case keys.ActionThisPackageDeclaresNoSuchAction: return m, nil`, fails the build with
// "undefined: keys.ActionThisPackageDeclaresNoSuchAction". The compiler refusing a bogus case
// label is the property the dispatch join's source sweep leans on when it reads handledActions
// against the table — one action per arm is what keeps that sweep a line match rather than a
// parser.

// coverBreaking is the one predicate the dispatcher gate uses: media, copy-link
// and open-post would reveal the real upstream to a program or the clipboard.
func coverBreaking(action keys.Action) bool {
	switch action {
	case keys.ActionMedia, keys.ActionCopyLink, keys.ActionOpenPost:
		return true
	}
	return false
}

// keyAction looks one key press up in the table. It asks twice because a shifted letter has two
// spellings in Bubble Tea v2: the decoder lower-cases an upper-case ASCII byte, keeps the shifted
// character in the event and marks it shifted, so Keystroke() reports `shift+g` while String()
// reports the character the user typed — which is the spelling the table, the overlay and the
// subcommand all use.
func keyAction(msg tea.KeyPressMsg, screen string) (keys.Action, bool) {
	if action, ok := keys.Lookup(msg.Keystroke(), screen); ok {
		return action, true
	}
	return keys.Lookup(msg.String(), screen)
}

// move is one movement key. The selection walks the frame's own row order — every post and then the
// posts it quotes, so one step can land on a quoted row inside the block it is in — and the anchor is put
// back so that the pane keeps the scroll position it had: a move inside the visible area moves nothing
// but the mark, and a move past the pane's edge scrolls by exactly as much as it has to.
//
// `g` is the one exception, and it is the key's own meaning rather than a second movement rule: it
// asks for the frame's beginning, and the frame's beginning is the state it opened in. The frame is
// therefore left unmoved (Moved false) and the anchor is set aside, so the opening rule (model.go's
// skip) presents the frame's own first row again — a profile's header, a conversation's ancestors,
// the top of the list. That is what 05-UI-SPEC's "the header is always one key away" is: `g` after
// any scroll puts the frame's first row back as the pane's first row, without a second rule for
// where the top is.
func (m Model) move(action keys.Action) (Model, tea.Cmd) {
	f := m.top()
	rows := f.targets()
	if len(rows) == 0 {
		return m, nil
	}

	forward, step := true, 0
	switch action {
	case keys.ActionNext:
		step = 1
	case keys.ActionPrev:
		forward, step = false, -1
	case keys.ActionFirst:
		forward, step = false, -len(rows)
	case keys.ActionLast:
		step = len(rows)
	case keys.ActionHalfDown:
		step = m.step(f, 1, m.paneRows()/2)
	case keys.ActionHalfUp:
		forward, step = false, m.step(f, -1, m.paneRows()/2)
	case keys.ActionPageDown:
		step = m.step(f, 1, m.paneRows())
	case keys.ActionPageUp:
		forward, step = false, m.step(f, -1, m.paneRows())
	default:
		return m, nil
	}

	if step != 0 {
		scroll, before := m.scroll(f, m.w), f.selected()
		f = m.moveFrame(f, step)
		switch {
		case action == keys.ActionFirst:
			// The frame is at its beginning again: unmoved, so the opening rule decides the pane
			// and the anchor the reader left is not read (model.go's skip reads it only while the
			// frame is moved). Anchor is cleared with the flag rather than left stale, because the
			// two facts are one fact: either the pane is at the frame's top or it is at the
			// anchor.
			f.Moved, f.Anchor = false, 0
			m = m.setTop(f)
		case !f.selected().Equal(before):
			// The reader has now moved in this frame, so the anchor is theirs: reanchor puts the newly
			// selected block's head where the pane's own scroll leaves it, and the frame's opening
			// rule (model.go's skip) never applies to it again. A move that changes nothing — a `k`
			// at the first row — leaves the frame unmoved, and the pane exactly where it was.
			f.Anchor, f.Moved = m.reanchor(f, scroll, m.w), true
			m = m.setTop(f)
		}
	}
	if forward {
		return m.page()
	}
	return m, nil
}

func (m Model) moveFrame(f frame, step int) frame {
	switch f.Kind {
	case frameThread:
		f.Thread = f.Thread.Move(step)
		return f
	case frameSearch:
		if f.isPeople() {
			f.People = f.People.Move(f.targets(), step)
			return f
		}
		f.Search = f.Search.Move(f.targets(), step)
		return f
	case frameProfile:
		f.Profile = f.Profile.Move(f.targets(), step)
		return f
	case frameSaved:
		l := f.Saved.List().Move(step)
		f.Saved = f.Saved.WithList(l)
		return f
	}
	f.List = f.List.Move(step)
	return f
}

// step is half a pane or a whole pane of rows, measured in the rows the blocks actually render: a step
// counted in rows of the list would be one row in one frame and twenty in another. It returns how many
// of the frame's own rows to move by, which is the step the walk then clamps.
//
// The selection lands on the last row whose block head is within the step, and on the next row at least
// when the first one is already further away than the step — so a pane-height step through a list of tall
// blocks still moves, rather than standing still.
func (m Model) step(f frame, dir, rows int) int {
	seq := f.targets()
	from := indexOfTarget(seq, f.selected())
	if from < 0 {
		return 0
	}
	heads, _ := geometry(m.blocksAt(f, m.w), f)
	block := blockIndex(f, seq[from].Top().ID)
	if block < 0 || block >= len(heads) {
		return 0
	}
	// The walk below steps in blocks and lands on rows: the head rows are the blocks' own, and the
	// sequence runs through one block's rows before the next block's, so the block index moves with the
	// row index. A quoted row is a row of the block above it, which is what keeps a step inside one block
	// from leaving it.
	target, at := from, block
	if dir > 0 {
		for i := from + 1; i < len(seq); i++ {
			if !seq[i].Top().Equal(seq[i-1].Top()) {
				if at+1 >= len(heads) {
					break
				}
				at++
			}
			if heads[at]-heads[block] > rows {
				break
			}
			target = i
		}
	} else {
		for i := from - 1; i >= 0; i-- {
			if !seq[i].Top().Equal(seq[i+1].Top()) {
				if at == 0 {
					break
				}
				at--
			}
			if heads[block]-heads[at] > rows {
				break
			}
			target = i
		}
	}
	if target == from {
		target = from + dir
	}
	if target < 0 || target >= len(seq) {
		return 0
	}
	return target - from
}

// indexOfTarget is where a row sits in a frame's own row sequence, or -1 when it is not one of them. The
// movement itself is the walk's (Move); this is the shell reading its own geometry against the sequence
// it just asked for.
func indexOfTarget(seq []screen.Target, t screen.Target) int {
	for i := range seq {
		if seq[i].Equal(t) {
			return i
		}
	}
	return -1
}

func blockIndex(f frame, id string) int {
	switch f.Kind {
	case frameThread:
		// Position is 1-based for the selected block; find by walking blocks via Targets tops.
		seen := map[string]int{}
		n := 0
		for _, t := range f.Thread.Targets() {
			top := t.Top().ID
			if _, ok := seen[top]; ok {
				continue
			}
			seen[top] = n
			if top == id {
				return n
			}
			n++
		}
		return -1
	case frameSearch:
		if f.isPeople() {
			return f.People.Walk().Index(id)
		}
		return f.Search.Walk().Index(id)
	case frameProfile:
		idx := f.Profile.Walk().Index(id)
		if idx < 0 {
			return -1
		}
		// The header is block 0; posts follow it.
		return idx + 1
	}
	return f.List.Index(id)
}

// page is the pagination trigger: the reader has reached the last loaded post and the walk has
// somewhere to go. The list's own single-flight latch is what stops a slow page turning a
// held-down key into a burst, and the process's outstanding count stops this request being stacked
// behind another feed's.
func (m Model) page() (Model, tea.Cmd) {
	f := m.top()
	if m.outstanding() > 0 || !m.requestAllowed(m.deps.Now()) {
		return m, nil
	}
	var effect screen.Effect
	switch f.Kind {
	case frameThread:
		if f.Thread.Read() && !f.Thread.AtEnd() {
			return m, nil
		}
		next, e := f.Thread.Update(screen.ThreadRequested{}, m.deps.Now())
		if e.Kind != screen.EffectPage {
			return m, nil
		}
		f.Thread = next
		effect = e
	case frameSearch:
		if f.isPeople() {
			if !f.People.AtEnd() {
				return m, nil
			}
			next, e := f.People.Update(screen.ResultsRequested{}, m.deps.Now())
			if e.Kind != screen.EffectPage {
				return m, nil
			}
			f.People = next
			effect = e
			break
		}
		if !f.Search.AtEnd() {
			return m, nil
		}
		next, e := f.Search.Update(screen.ResultsRequested{}, m.deps.Now())
		if e.Kind != screen.EffectPage {
			return m, nil
		}
		f.Search = next
		effect = e
	case frameProfile:
		if !f.Profile.AtEnd() {
			return m, nil
		}
		next, e := f.Profile.Update(screen.PostsRequested{}, m.deps.Now())
		if e.Kind != screen.EffectPage {
			return m, nil
		}
		f.Profile = next
		effect = e
	case frameSaved:
		return m.pageSaved()
	default:
		if !f.List.AtEnd() {
			return m, nil
		}
		next, e := f.List.Update(screen.PageRequested{}, m.deps.Now())
		if e.Kind != screen.EffectPage {
			return m, nil
		}
		f.List = next
		effect = e
	}
	m = m.setTop(f)
	m = m.setNotice(render.NoticeNone, 0)
	return m.request(f, effect, false)
}

// refresh is the explicit refresh: page one again, to find out what is new at the top. It is
// debounced twice over — the list refuses while its own request is out, and the process refuses
// while any request is out or while the state forbids one — because a held-down `r` must not spend
// the budget on a burst and a stopped session must not be asked again.
func (m Model) refresh() (Model, tea.Cmd) {
	if m.outstanding() > 0 || !m.requestAllowed(m.deps.Now()) {
		return m, nil
	}
	f := m.top()
	var effect screen.Effect
	switch f.Kind {
	case frameThread:
		next, e := f.Thread.Update(screen.ThreadRefreshRequested{}, m.deps.Now())
		if e.Kind != screen.EffectRefresh {
			return m, nil
		}
		f.Thread = next
		effect = e
	case frameSearch:
		if f.isPeople() {
			next, e := f.People.Update(screen.ResultsRefreshRequested{}, m.deps.Now())
			if e.Kind != screen.EffectRefresh {
				return m, nil
			}
			f.People = next
			effect = e
			break
		}
		next, e := f.Search.Update(screen.ResultsRefreshRequested{}, m.deps.Now())
		if e.Kind != screen.EffectRefresh {
			return m, nil
		}
		f.Search = next
		effect = e
	case frameProfile:
		next, e := f.Profile.Update(screen.ProfileRefreshRequested{}, m.deps.Now())
		if e.Kind != screen.EffectRefresh {
			return m, nil
		}
		f.Profile = next
		effect = e
	case frameSaved:
		return m.refreshSaved()
	default:
		next, e := f.List.Update(screen.RefreshRequested{}, m.deps.Now())
		if e.Kind != screen.EffectRefresh {
			return m, nil
		}
		f.List = next
		effect = e
	}
	m = m.setTop(f)
	m = m.setNotice(render.NoticeNone, 0)
	return m.request(f, effect, true)
}

// request is the shell's whole dispatch over the request union: the effect names which read is wanted and
// with what arguments, and this is where that becomes a command.
//
// A frame kind added later adds a case here rather than a field on every effect. The conversation's own
// arm — its read, the frame the answer lands in, and the notices it prints — arrives with the frame that
// can ask for one: nothing in this build produces a ThreadRequest, and its case is declared in
// screen/walk.go for that plan.
func (m Model) request(f frame, effect screen.Effect, refresh bool) (Model, tea.Cmd) {
	switch req := effect.Req.(type) {
	case screen.FeedRequest:
		m.pending++
		return m, m.fetchPage(f.ID, req.Cursor, req.Kind, refresh)
	case screen.ThreadRequest:
		m.pending++
		return m, m.fetchThread(f.ID, req.Focal, req.Cursor, refresh)
	case screen.SearchRequest:
		m.pending++
		return m, m.fetchSearch(f.ID, req, refresh)
	case screen.PeopleRequest:
		m.pending++
		return m, m.fetchPeople(f.ID, req, refresh)
	case screen.ProfileRequest:
		m.pending++
		return m, m.fetchProfile(f.ID, req.Handle)
	case screen.UserPostsRequest:
		m.pending++
		return m, m.fetchUserPosts(f.ID, req, refresh)
	case screen.UsersRequest:
		m.pending++
		return m, m.fetchAccounts(f.ID, req, refresh)
	case screen.MediaRequest:
		m.pending++
		return m, m.fetchMedia(f.ID, req, refresh)
	}
	return m, nil
}

// toggleFeed is `t`: the other feed, one key away and one key back.
//
// The push creates a fresh frame for the other feed and asks for its page one, because a frame
// showing an empty list that was never read would be exactly the lie this phase exists to prevent
// — the empty-state copy says an empty read succeeded, and no read happened. Pressing the key again
// pops the frame, so the key is a true toggle, and nothing is fetched for the frame being returned
// to: the frame that comes back is byte-identical to the frame that was left.
func (m Model) toggleFeed() (Model, tea.Cmd) {
	if len(m.frames) > 1 {
		return m.pop(), nil
	}
	kind := m.otherFeed()
	m.seq++
	id := homeID(kind, m.seq)
	m.frames = append(m.frames, frame{ID: id, Kind: frameHome, List: screen.List{Kind: kind}})
	if !m.requestAllowed(m.deps.Now()) {
		return m, nil
	}
	m.pending++
	return m, m.fetchPage(id, "", kind, false)
}

// otherFeed is the feed the top frame is not showing. This phase has exactly two, and the toggle is
// the only producer of a second frame.
func (m Model) otherFeed() domain.HomeKind {
	if m.top().List.Kind == domain.ForYou {
		return domain.Following
	}
	return domain.ForYou
}

// openPrompt is `/` or `u`: the query field opens, naming the product it will search. Nothing is
// pushed, no request is issued and the status line is composed exactly as before — the budget
// included. Over a results frame the field is seeded with that frame's own query, which is the
// recovery path for a query the status field cut short.
func (m Model) openPrompt(product domain.SearchProduct) (Model, tea.Cmd) {
	seed := ""
	if f := m.top(); f.Kind == frameSearch {
		if f.isPeople() {
			seed = f.People.Query()
		} else {
			seed = f.Search.Query()
		}
	}
	prompt := screen.NewPrompt(product, seed)
	m.prompt = &prompt
	return m, nil
}

// promptKey is one key while the prompt is open. The keymap is shadowed: every printable key is
// text, `enter` submits, `esc` cancels, and the three editing keys edit. Nothing else acts, and
// nothing here can issue a request — only a submitted query can.
func (m Model) promptKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	prompt := *m.prompt
	switch {
	case msg.Code == tea.KeyEscape:
		// The pane is restored byte-identically: the prompt is model state, and closing it
		// changes nothing else.
		m.prompt = nil
		return m, nil
	case msg.Code == tea.KeyEnter:
		return m.submitSearch()
	case msg.Code == tea.KeyBackspace:
		prompt = prompt.Backspace()
	case msg.Mod == tea.ModCtrl && msg.Code == 'w':
		prompt = prompt.DeleteWord()
	case msg.Mod == tea.ModCtrl && msg.Code == 'u':
		prompt = prompt.Clear()
	case msg.Text != "":
		prompt = prompt.Insert(msg.Text)
	}
	m.prompt = &prompt
	return m, nil
}

// submitSearch is `enter` in the prompt: a real query closes the prompt, pushes the results frame
// and issues the frame's one opening effect through the single command path. An empty or
// all-space query is the documented no-op: the prompt stays open and nothing is spent.
func (m Model) submitSearch() (Model, tea.Cmd) {
	query, ok := m.prompt.Submitted()
	if !ok {
		return m, nil
	}
	product := m.prompt.Product()
	m.prompt = nil
	if product == domain.People {
		results, effect := screen.NewPeopleResults(query)
		m.seq++
		m.frames = append(m.frames, frame{ID: searchID(query, domain.People, m.seq), Kind: frameSearch, People: results})
		if !m.requestAllowed(m.deps.Now()) {
			return m, nil
		}
		m = m.setNotice(render.NoticeNone, 0)
		return m.request(m.top(), effect, false)
	}
	results, effect := screen.NewPostResults(query, domain.Latest)
	m.seq++
	m.frames = append(m.frames, frame{ID: searchID(query, domain.Latest, m.seq), Kind: frameSearch, Search: results})
	if !m.requestAllowed(m.deps.Now()) {
		return m, nil
	}
	m = m.setNotice(render.NoticeNone, 0)
	return m.request(m.top(), effect, false)
}

// switchOrder is `s`: the Latest/Top toggle, a frame toggle and not an in-place rerun.
//
// From a `latest` results frame it pushes a `top` frame for the SAME query and issues exactly one
// request: switching under a new product's name while the old product's rows stay on screen would
// be a lie about what is shown, so the new product gets its own frame. From a `top` frame it
// pops, which returns to the frozen frame it came from with the rows it had and zero requests.
// Anywhere there is no order to switch, it says so in the notice field and issues nothing.
func (m Model) switchOrder() (Model, tea.Cmd) {
	f := m.top()
	if f.Kind != frameSearch || f.isPeople() {
		return m.setNotice(render.NoticeNothingToSwitch, 0), nil
	}
	if f.Search.Product() == domain.Top {
		// A `top` frame only ever sits on the frame it was pushed from, so the pop is the way
		// back — byte-identical, because a non-top frame is frozen.
		if len(m.frames) > 1 {
			return m.pop(), nil
		}
		return m.setNotice(render.NoticeNothingToSwitch, 0), nil
	}
	results, effect := screen.NewPostResults(f.Search.Query(), domain.Top)
	m.seq++
	m.frames = append(m.frames, frame{ID: searchID(f.Search.Query(), domain.Top, m.seq), Kind: frameSearch, Search: results})
	if !m.requestAllowed(m.deps.Now()) {
		return m, nil
	}
	m = m.setNotice(render.NoticeNone, 0)
	return m.request(m.top(), effect, false)
}

// open is enter: on a people row push the account's profile; otherwise push the selected post
// or quoted post as its own conversation. Pushing a conversation costs no request of its own;
// page one is asked for from ThreadRequested's effect. Opening a profile issues the header
// request through the single command path (05-02).
func (m Model) open() (Model, tea.Cmd) {
	if m.state == StateVivid {
		// The presentation's `enter` fills the reading column and leaves the list where it is —
		// that is what the second column is for.
		return m.openIntoReading()
	}
	f := m.top()
	sel := f.selected()
	if sel.ID == "" {
		return m.setNotice(render.NoticeNothingToOpen, 0), nil
	}
	if f.isPeople() {
		return m.openAccount(sel)
	}
	post, ok := f.post(sel)
	if !ok {
		return m.setNotice(render.NoticeNothingToOpen, 0), nil
	}
	if f.Kind == frameThread && post.ID == f.Thread.Focal().ID && len(sel.Path) == 0 {
		return m, nil
	}
	m.seq++
	id := threadID(post.ID, m.seq)
	th := screen.NewThread(post)
	m.frames = append(m.frames, frame{ID: id, Kind: frameThread, Thread: th})
	if !m.requestAllowed(m.deps.Now()) {
		return m, nil
	}
	next, effect := th.Update(screen.ThreadRequested{}, m.deps.Now())
	if effect.Kind != screen.EffectPage {
		return m, nil
	}
	m.frames[len(m.frames)-1].Thread = next
	m = m.setNotice(render.NoticeNone, 0)
	return m.request(m.frames[len(m.frames)-1], effect, false)
}

// openAuthor is `a`: on a people row the account IS the author, so it opens the same profile
// path. On a post row it opens the selected post's author's profile.
func (m Model) openAuthor() (Model, tea.Cmd) {
	f := m.top()
	sel := f.selected()
	if sel.ID == "" {
		return m.setNotice(render.NoticeNoAuthor, 0), nil
	}
	if f.isPeople() {
		return m.openAccount(sel)
	}
	post, ok := f.post(sel)
	if !ok || post.Author.ScreenName == "" {
		return m.setNotice(render.NoticeNoAuthor, 0), nil
	}
	handle := post.Author.ScreenName
	if f.Kind == frameProfile && f.Profile.Handle() == handle {
		return m, nil
	}
	return m.pushProfile(handle)
}

// openAccount pushes the selected account's profile from a people results frame.
func (m Model) openAccount(sel screen.Target) (Model, tea.Cmd) {
	f := m.top()
	user, ok := f.account(sel)
	if !ok || user.ScreenName == "" {
		return m.setNotice(render.NoticeNoAuthor, 0), nil
	}
	if top := m.top(); top.Kind == frameProfile && top.Profile.Handle() == user.ScreenName {
		return m, nil
	}
	return m.pushProfile(user.ScreenName)
}

func (m Model) pushProfile(handle string) (Model, tea.Cmd) {
	profile, effect := screen.Open(handle)
	m.seq++
	id := profileID(handle, m.seq)
	m.frames = append(m.frames, frame{ID: id, Kind: frameProfile, Profile: profile})
	if !m.requestAllowed(m.deps.Now()) {
		return m, nil
	}
	m = m.setNotice(render.NoticeNone, 0)
	return m.request(m.frames[len(m.frames)-1], effect, false)
}

func (m Model) profileHeaderAnswered(msg profileHeaderMsg) (Model, tea.Cmd) {
	m = m.release()
	i := m.frameIndexID(msg.Frame)
	if i < 0 {
		return m, nil
	}
	m.frames[i].Read = true
	next, effect := m.frames[i].Profile.Update(screen.HeaderFetched{Header: msg.Header}, m.deps.Now())
	m.frames[i].Profile = next
	m.failure = domain.Failure{}
	if effect.Kind == screen.EffectNone {
		return m, nil
	}
	return m.request(m.frames[i], effect, false)
}

func (m Model) profilePostsAnswered(msg profilePostsMsg) Model {
	m = m.release()
	i := m.frameIndexID(msg.Frame)
	if i < 0 {
		return m
	}
	m.frames[i].Read = true
	wasRefreshing := m.frames[i].Profile.Refreshing()
	var event screen.Event = screen.PostsFetched{Page: msg.Page}
	if wasRefreshing {
		event = screen.PostsRefreshed{Page: msg.Page}
	}
	next, _ := m.frames[i].Profile.Update(event, m.deps.Now())
	m.frames[i].Profile = next
	m.failure = domain.Failure{}
	if wasRefreshing {
		if n := next.Walk().NewRows(); n > 0 {
			m = m.setNotice(render.NoticeNewPosts, n)
		} else {
			m = m.setNotice(render.NoticeNoNewPosts, 0)
		}
	}
	return m
}

// pop drops the top frame. The frames below are untouched, which is what makes the frame the user
// returns to byte-identical to the frame they left: a non-top frame is frozen, and a refresh
// applies to the top frame only.
func (m Model) pop() Model {
	m.frames = m.frames[:len(m.frames)-1]
	return m
}

// back is `q`/`esc`: pop one frame, or quit at the bottom one.
func (m Model) back() (Model, tea.Cmd) {
	if len(m.frames) > 1 {
		top := m.top()
		cmd := m.abandonSavedSync(top)
		m = m.pop()
		return m, cmd
	}
	return m, tea.Quit
}

// answered absorbs one answer. The frame it belongs to is found by the id the request carried — the
// instance that asked, not merely one of its feed kind — and an answer whose frame is gone is
// dropped: the toggle discarded it, and absorbing it into the frame that replaced it would read as a
// source that repeated itself, which is the one state that ends a walk. A dropped answer marks
// nothing read either: the pane has no answer for the frame it replaced, so the empty-state copy's
// claim ("an empty read succeeded") is still not true of it.
func (m Model) answered(msg pageMsg) Model {
	m = m.release()
	i := m.frameIndexID(msg.Frame)
	if i < 0 {
		return m
	}
	m.frames[i].Read = true

	event := screen.Event(screen.PageFetched{Page: msg.Page})
	if msg.Refresh {
		event = screen.Refreshed{Page: msg.Page}
	}
	next, _ := m.frames[i].List.Update(event, m.deps.Now())
	m.frames[i].List = next
	m.failure = domain.Failure{}

	if msg.Refresh {
		if n := next.NewPosts(); n > 0 {
			m = m.setNotice(render.NoticeNewPosts, n)
		} else {
			m = m.setNotice(render.NoticeNoNewPosts, 0)
		}
	}
	return m
}

func (m Model) threadAnswered(msg threadMsg) Model {
	m = m.release()
	i := m.frameIndexID(msg.Frame)
	if i < 0 {
		return m
	}
	m.frames[i].Read = true
	event := screen.Event(screen.ThreadFetched{Page: msg.Page})
	if msg.Refresh {
		event = screen.ThreadRefreshed{Page: msg.Page}
	}
	next, _ := m.frames[i].Thread.Update(event, m.deps.Now())
	m.frames[i].Thread = next
	m.failure = domain.Failure{}
	if msg.Refresh {
		if n := next.NewReplies(); n > 0 {
			m = m.setNotice(render.NoticeNewReplies, n)
		} else {
			m = m.setNotice(render.NoticeNoNewReplies, 0)
		}
	}
	return m
}

// searchAnswered absorbs one search answer, found by the frame id the request carried — an answer
// whose frame is gone is dropped, exactly as the other two reads' answers are. The event carries
// the walk's vocabulary read out of the page here, at the seam: the results frame is generic over
// the page value, and the shell is the one place that knows it is a PostPage.
func (m Model) searchAnswered(msg searchMsg) Model {
	m = m.release()
	i := m.frameIndexID(msg.Frame)
	if i < 0 {
		return m
	}
	m.frames[i].Read = true
	event := screen.Event(screen.ResultsFetched[domain.Post, domain.PostPage]{Page: msg.Page, Rows: msg.Page.Posts, Next: msg.Page.Next})
	if msg.Refresh {
		event = screen.ResultsRefreshed[domain.Post, domain.PostPage]{Page: msg.Page, Rows: msg.Page.Posts, Next: msg.Page.Next}
	}
	next, _ := m.frames[i].Search.Update(event, m.deps.Now())
	m.frames[i].Search = next
	m.failure = domain.Failure{}
	if msg.Refresh {
		if n := next.Walk().NewRows(); n > 0 {
			m = m.setNotice(render.NoticeNewPosts, n)
		} else {
			m = m.setNotice(render.NoticeNoNewPosts, 0)
		}
	}
	return m
}

// peopleAnswered absorbs one people-search answer into the people results frame.
func (m Model) peopleAnswered(msg peopleMsg) Model {
	m = m.release()
	i := m.frameIndexID(msg.Frame)
	if i < 0 {
		return m
	}
	m.frames[i].Read = true
	event := screen.Event(screen.ResultsFetched[domain.User, domain.UserPage]{Page: msg.Page, Rows: msg.Page.Rows, Next: msg.Page.Next})
	if msg.Refresh {
		event = screen.ResultsRefreshed[domain.User, domain.UserPage]{Page: msg.Page, Rows: msg.Page.Rows, Next: msg.Page.Next}
	}
	next, _ := m.frames[i].People.Update(event, m.deps.Now())
	m.frames[i].People = next
	m.failure = domain.Failure{}
	if msg.Refresh {
		if n := next.Walk().NewRows(); n > 0 {
			m = m.setNotice(render.NoticeNewPosts, n)
		} else {
			m = m.setNotice(render.NoticeNoNewPosts, 0)
		}
	}
	return m
}

// failed records a request that did not deliver data.
//
// The state is the process's, not the frame's, so it is stored on the model and the status line
// reads it whatever frame is on top. The frame the request belonged to is told as well, so that its
// own gate and its own latch agree with the state line; a failure whose frame is gone still reaches
// the state line, because a request whose failure nothing reports is the one thing a failure message
// must not become — and it does not reach the frame that replaced it, whose own request is still out.
func (m Model) failed(msg failureMsg) Model {
	m = m.release()
	m.failure = msg.Failure
	m = m.setNotice(render.NoticeNone, 0)
	if i := m.frameIndexID(msg.Frame); i >= 0 {
		switch m.frames[i].Kind {
		case frameThread:
			next, _ := m.frames[i].Thread.Update(screen.ThreadFailed{Failure: m.failure}, m.deps.Now())
			m.frames[i].Thread = next
		case frameSearch:
			if m.frames[i].isPeople() {
				next, _ := m.frames[i].People.Update(screen.ResultsFailed{Failure: m.failure}, m.deps.Now())
				m.frames[i].People = next
			} else {
				next, _ := m.frames[i].Search.Update(screen.ResultsFailed{Failure: m.failure}, m.deps.Now())
				m.frames[i].Search = next
			}
		case frameProfile:
			_, have := m.frames[i].Profile.Header()
			if !have {
				next, _ := m.frames[i].Profile.Update(screen.HeaderFailed{Failure: m.failure}, m.deps.Now())
				m.frames[i].Profile = next
			} else {
				next, _ := m.frames[i].Profile.Update(screen.PostsFailed{Failure: m.failure}, m.deps.Now())
				m.frames[i].Profile = next
			}
		default:
			next, _ := m.frames[i].List.Update(screen.PageFailed{Failure: m.failure}, m.deps.Now())
			m.frames[i].List = next
		}
	}
	return m
}

func (m Model) release() Model {
	if m.pending > 0 {
		m.pending--
	}
	return m
}

// setNotice sets the process's announcement.
func (m Model) setNotice(kind render.NoticeKind, count int) Model {
	m.notice, m.noticeCount = kind, count
	return m
}

// resize is the terminal's size arriving: once at start-up and again on every change.
//
// Nothing but the size and the anchor's clamp changes. No request is issued, the list, the
// selection and the walk are untouched, and the frame is re-rendered from the same state at the new
// width — which is why no wrapped line and no column width is cached anywhere (REND-03, and A5's
// test compares exactly this against a cold render).
//
// The one command it can return is a start-up panic's entry: the frame is built from the pane's
// size, so the sequence waits for the first size the terminal reports and then runs, after the first
// frame has been painted.
func (m Model) resize(w, h int) (Model, tea.Cmd) {
	m.w, m.h = w, h
	top := m.top()
	top.Anchor = clamp(top.Anchor, 0, max(0, m.paneRows()-1))
	m = m.setTop(top)
	if m.entryPending && w > 0 && h > 0 {
		m.entryPending = false
		return m, m.entry()
	}
	return m, nil
}

// toggleCover is T1 and T2: `m` puts the remembered style on and takes it off again. It is the one
// transition that does not change which style is remembered.
func (m Model) toggleCover() (Model, tea.Cmd) {
	if m.state == StateNormal {
		return m.wear(m.style)
	}
	m.state = StateNormal
	return m, m.change()
}

// wear is T1, T3, T4 and T5: the feed through a cover, with the state's title on the window, the
// terminal's title on the view, and both display keys written — and the rows on screen not moved.
// The same list, the same selection and the same anchor are rendered by a different grammar, which
// is the whole of what wearing a cover changes.
func (m Model) wear(style render.CoverStyle) (Model, tea.Cmd) {
	m.state, m.style = StateImmersive, style
	return m, m.change()
}

// numberedCover is T5: `1` through `4` wear the style that digit numbers. The order is
// render.CoverStyles', which is the cycle's own order, so a digit and a step of `c` cannot disagree
// about which style is which.
func (m Model) numberedCover(action keys.Action) (Model, tea.Cmd) {
	styles := render.CoverStyles()
	for i, digit := range coverDigits {
		if digit == action && i < len(styles) {
			return m.wear(styles[i])
		}
	}
	return m, nil
}

// coverDigits are the four style keys in the order they number the styles.
var coverDigits = []keys.Action{
	keys.ActionCover1, keys.ActionCover2, keys.ActionCover3, keys.ActionCover4,
}

// stepCover is the style the cycle key steps to, which is the next of the four and the first after
// the fourth. An unknown style — the zero value in a shell that never had one — starts the cycle.
func stepCover(style render.CoverStyle) render.CoverStyle {
	styles := render.CoverStyles()
	for i, known := range styles {
		if known == style {
			return styles[(i+1)%len(styles)]
		}
	}
	return styles[0]
}

// change is the command every non-panic transition asks for: the state's window name, then the
// display state written to the config file. They are one command rather than a batch so that their
// order is this file's rather than the runtime's scheduler's, and so that a test that runs the
// command sees both effects and not one of them.
//
// The name is read off the model before the closure is built: the closure runs later, and what it
// writes has to be the name of the state that was entered, not of whatever state is current when it
// happens to run.
func (m Model) change() tea.Cmd {
	name, state, style, deps := m.title(), m.state.String(), string(m.style), m.deps
	if m.state == StateVivid {
		// The vivid presentation is not a starting state: the config file's vocabulary does not hold
		// its name, StateOf refuses it on purpose, and the design says the sequence is the only way
		// in. Writing it would leave a value the next run cannot resolve — a silent fall back to the
		// plain reader with the file still claiming otherwise. The window name is still set, because
		// that is what is on screen.
		return func() tea.Msg {
			deps.Term.SetWindow(name)
			return nil
		}
	}
	return func() tea.Msg {
		deps.Term.SetWindow(name)
		deps.Persist(state, style)
		return nil
	}
}

// enterPanic is T6 and T8 in one function, used by all three callers: the key, the focus loss and
// the start-up state. It sets the state, remembers where it came from, and returns the single
// command that runs the entry.
//
// Nothing is cancelled and nothing is written: a request in flight stays in flight, panic is a
// render state rather than a network state, and an involuntary transition must not become
// tomorrow's start-up state.
func (m Model) enterPanic() (Model, tea.Cmd) {
	if m.state == StatePanic {
		// Already panicked: a second blur (or a key that cannot arrive) runs the entry once, not
		// twice.
		return m, nil
	}
	m.from, m.state = m.state, StatePanic
	return m, m.entry()
}

// entry is the panic entry as the shell's one command: the frame built from the pane's size and the
// title set to the frame's own name, handed to the sink — which writes them in the required order
// and then works the multiplexer. The model performs no terminal call itself, and the recording
// port a test hands it is empty until this command is run.
func (m Model) entry() tea.Cmd {
	term, w, h := m.deps.Term, m.w, m.h
	return func() tea.Msg {
		term.EnterPanic(render.PanicFrame(w, h), render.CopyTitlePanic)
		return nil
	}
}

// leavePanic is T7: back to the state the panic was entered from, with that state's title and window
// name, and no config write at all. A panic is not a state the user chose, so it is not one the next
// run starts in.
func (m Model) leavePanic() (Model, tea.Cmd) {
	m.state = m.from
	name, term := m.title(), m.deps.Term
	return m, func() tea.Msg {
		term.SetWindow(name)
		return nil
	}
}

// handledActions is the shell's dispatch set, spelled out next to the dispatcher whose switch it
// names: the actions press and update act on. press routes a key to an action through the one
// table and its switch ends in a no-op, so an action the keymap binds but nothing handles — and a
// handler for an action nothing binds — drifts with no compiler to stop it. The sweep in
// dispatch_test.go joins this set to the table from both directions and reads the sources for the
// action declarations themselves, so a new action lands here, in the table and in a handler arm at
// once or not at all.
//
// Nothing here is an action with no key: the prompt's five editing keys and the panic gate's two
// are the table's own rows under shadowed states, not a second keymap — the arms they close over
// are this set's.
var handledActions = []keys.Action{
	keys.ActionNext, keys.ActionPrev, keys.ActionHalfDown, keys.ActionHalfUp,
	keys.ActionPageDown, keys.ActionPageUp, keys.ActionFirst, keys.ActionLast,
	keys.ActionOpen, keys.ActionOpenAuthor,
	keys.ActionRefresh, keys.ActionSwitchList,
	keys.ActionSearchPosts, keys.ActionSearchPeople, keys.ActionSearchOrder,
	keys.ActionHelp, keys.ActionBack, keys.ActionQuit,
	keys.ActionCover, keys.ActionNextCover,
	keys.ActionCover1, keys.ActionCover2, keys.ActionCover3, keys.ActionCover4,
	keys.ActionPanic,
	keys.ActionBookmarks, keys.ActionLikes,
	keys.ActionMedia, keys.ActionCopyLink, keys.ActionOpenPost,
	keys.ActionFollowing, keys.ActionFollowers, keys.ActionVerifiedFollowers, keys.ActionProfileMedia,
	keys.ActionTranslate,
}

// HandledActions returns the actions the shell's dispatcher handles. It exists so the surfaces
// that read the keymap's other end — the `keys` subcommand, the help overlay's golden — can ask
// for the join from their own side, and so a test outside this package can hold the dispatcher to
// the table. The slice is a copy: a caller reads it and does not modify it.
func HandledActions() []keys.Action { return slices.Clone(handledActions) }
