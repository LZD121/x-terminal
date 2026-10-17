package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/keys"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/vivid"
)

// Render composes the frame: the pane's blocks from the top frame's list, the one status line from
// the state, and render.Frame's own row and cell arithmetic. In panic it is the panic frame and
// nothing else, whatever the model underneath holds.
//
// The shell adds no bytes. The frame is exactly the engine's output for the state — which is what
// makes the stored golden frames the specification of the shipped screen rather than a picture of
// something close to it. w and h are the pane's size; the model's own size comes from the terminal
// and the two are the same in production, while a test renders any state at any size.
func (m Model) Render(w, h int) string {
	if w <= 0 || h <= 0 {
		// The terminal has not said how big it is yet: there is no frame to draw into, and a frame
		// of zero rows is not a frame.
		return ""
	}
	if m.state == StatePanic {
		// The whole terminal, with no status line and no list: the frame owns every row.
		return render.PanicFrame(w, h)
	}
	if m.state == StateVivid {
		// The second presentation is its own renderer: two columns and a bar, not the plain
		// reader's single pane. Its own package composes it, so the frame's bytes are a function
		// of the value the shell hands it and nothing else.
		return vivid.Render(m.vividView())
	}
	return render.Frame(m.pane(w), m.status(), w, h)
}

// View is the pane as the program draws it: the whole terminal, repainted in full every time, in
// the alternate screen so the timeline never lands in the terminal's scrollback and the panic
// entry has less to clear.
//
// The title is a function of the state — the plain reader's name, the active style's name, or the
// panic frame's name — and the view carries it, which is what makes the toolkit write it, and what
// makes a bare terminal's title cleared on exit: the toolkit clears a title it set.
//
// Focus reporting is enabled so that losing the window can cover the pane. A terminal that does not
// report focus never sends the message, and nothing on screen claims it will.
//
// Two things are deliberately still not set. Mouse reporting is never enabled, so the terminal keeps
// its own selection and middle-click paste. No progress bar is reported, and nothing here rings a
// bell.
func (m Model) View() tea.View {
	view := tea.NewView(m.Render(m.w, m.h))
	view.AltScreen = true
	view.MouseMode = tea.MouseModeNone
	view.ReportFocus = true
	view.WindowTitle = m.title()
	if m.prompt != nil && m.state != StatePanic && m.w > 0 {
		// The prompt's caret is the terminal's own cursor, never a byte of the frame: the shell
		// computes the one cell (row 1, PromptRow's column, zero-based for the toolkit) and the
		// toolkit places it. It is the only place the program positions a cursor.
		_, caret := render.PromptRow(m.grammar(), m.prompt.Product(), m.prompt.Text(), m.w)
		view.Cursor = tea.NewCursor(caret-1, 0)
	}
	return view
}

// title is the state's name for the thing on screen. It is a function of the state in every state,
// so panic ensures the value rather than being the only place it is set, and no state's title ever
// lies about its frame: the plain reader's own name, the active style's own name, or the panic
// frame's.
func (m Model) title() string {
	switch m.state {
	case StateImmersive:
		return m.style.Title()
	case StateVivid:
		return render.CopyTitleVivid
	case StatePanic:
		return render.CopyTitlePanic
	}
	return render.CopyTitlePlain
}

// covered is the style the pane is drawn under: the active style in immersive, and the zero style —
// the plain reader — in normal. It is the one place the state decides whether a cover is worn, so
// the status line, the empty state and the help overlay all read the same answer.
func (m Model) covered() render.CoverStyle {
	if m.state == StateImmersive {
		return m.style
	}
	return ""
}

// grammar is the display style the one composer is handed: the active cover's grammar, or the plain
// reader's. A cover is a value, not a second renderer.
func (m Model) grammar() render.Grammar {
	if style := m.covered(); style != "" {
		return style.Grammar()
	}
	return render.Plain
}

// pane is the frame's block list: the help overlay while it is open, one state block when the list
// has nothing to show, and the list's own blocks — scrolled to the anchor — otherwise. A failure
// with posts already loaded changes nothing here: the posts stay exactly where they were and the
// status line carries the state.
//
// Wearing a cover changes exactly one thing here: the grammar the one composer is handed, and it is
// the same grammar the geometry is measured with — m.blocksAt is that one value (model.go), so the
// rows the pane paints are the rows its scroll was computed from. The order, the selection and the
// reader's place in the list are the list's own and are the same in both states; the rows those facts
// land on are the frame's, because a cover's record is not the plain reader's height.
func (m Model) pane(w int) []render.Block {
	f := m.top()
	switch {
	case m.links != nil:
		return []render.Block{{Rows: render.Links(m.grammar(), w, m.links.links)}}
	case m.help:
		// The overlay is the pane's whole height: the keymap is taller than a 24-row terminal, and
		// render.Help names the rows it cannot hold rather than dropping them silently.
		return []render.Block{{Rows: render.Help(helpRows(m.covered() != "", screenName(f.Kind)), w, m.paneRows())}}
	case m.prompt != nil:
		// The prompt replaces the pane, not the status row: row 1 is the field, the rows below
		// it are blank (emitted, as always, so the frame is H×W), and the budget stays on screen.
		row, _ := render.PromptRow(m.grammar(), m.prompt.Product(), m.prompt.Text(), w)
		return []render.Block{{Rows: []string{row}}}
	case f.Kind == frameThread:
		return m.scrolled(m.blocksAt(f, w), f)
	case f.Kind == frameSearch:
		if f.isPeople() {
			if len(f.People.Walk().Rows()) == 0 {
				return m.searchState(f, w)
			}
			return m.scrolled(m.blocksAt(f, w), f)
		}
		if len(f.Search.Walk().Rows()) == 0 {
			return m.searchState(f, w)
		}
		return m.scrolled(m.blocksAt(f, w), f)
	case f.Kind == frameProfile:
		if _, ok := f.Profile.Header(); !ok {
			return m.profileState(f, w)
		}
		return m.scrolled(m.blocksAt(f, w), f)
	case f.Kind == frameSaved:
		return m.savedPane(f, w)
	case f.Kind == frameAccounts:
		if len(f.Accounts.Walk().Rows()) == 0 {
			return m.searchState(f, w)
		}
		return m.scrolled(m.blocksAt(f, w), f)
	case f.Kind == frameTrends:
		if len(f.Trends) == 0 {
			return m.searchState(f, w)
		}
		return m.scrolled(m.blocksAt(f, w), f)
	case f.Kind == frameMedia:
		if len(f.Media.Walk().Rows()) == 0 {
			return m.searchState(f, w)
		}
		return m.scrolled(m.blocksAt(f, w), f)
	case len(f.List.Posts()) == 0:
		return []render.Block{m.emptyBlock(f, w)}
	}
	return m.scrolled(m.blocksAt(f, w), f)
}

// searchState is the results frame's pane when it has no rows to show. The honesty rule is the
// phase's own, extended to this frame kind: a failure is never rendered as an empty read, a frame
// that was never read says it is loading, and a successful read with zero rows is an EMPTY pane —
// the words for that one live in the status line's notice field, not in the pane.
func (m Model) searchState(f frame, w int) []render.Block {
	switch {
	case m.failure.Kind != domain.FailureNone:
		word, body := render.FailureWords(m.covered(), m.failure.Kind, m.failure.Detail)
		return []render.Block{render.StateBlock(m.grammar(), w, word, body)}
	case !f.Read:
		return []render.Block{render.StateBlock(m.grammar(), w, m.grammar().Markers.LoadingResults, "")}
	}
	return nil
}

// profileState is the profile pane while the header has not arrived: loading, or a failure.
func (m Model) profileState(f frame, w int) []render.Block {
	switch {
	case m.failure.Kind != domain.FailureNone:
		word, body := render.FailureWords(m.covered(), m.failure.Kind, m.failure.Detail)
		return []render.Block{render.StateBlock(m.grammar(), w, word, body)}
	case f.Profile.HeaderFailure().Kind != domain.FailureNone:
		word, body := render.FailureWords(m.covered(), f.Profile.HeaderFailure().Kind, f.Profile.HeaderFailure().Detail)
		return []render.Block{render.StateBlock(m.grammar(), w, word, body)}
	default:
		line := m.grammar().Markers.LoadingProfile
		if line == "" {
			line = render.MarkerLoadingProfile
		}
		return []render.Block{render.StateBlock(m.grammar(), w, line, "")}
	}
}

// emptyBlock is the pane when the list has nothing to show, in the block shape every post uses so
// that the pane never looks like a different kind of screen. The body is wrapped to the pane's own
// measure, which is why the width comes with the frame.
//
// The three cases are the honesty rule of this phase, in order: a failure is never rendered as an
// empty read, a list that was never read is never rendered as an empty read, and the empty-state
// copy appears only after a read that succeeded and brought nothing.
//
// Under a cover the shape is the cover grammar's — its gutter and its indent — and the words are
// the cover's, reached through the render package's own lookups so a covered frame cannot print the
// plain reader's words by accident. The failure state is the one case whose words come from the
// classifier rather than from the copy table, and render.FailureWords is the one place that is
// decided: the plain reader gets the state's own name and the classifier's own sentence verbatim,
// and a covered pane gets that name with the host substituted and the cover's own line. The
// classifier's sentence never reaches a covered pane, so nothing it says about the upstream — its
// hostname, its operation ids, the product's own words — can be read off a covered screen.
func (m Model) emptyBlock(f frame, w int) render.Block {
	cover := m.covered()
	switch {
	case m.failure.Kind != domain.FailureNone:
		word, body := render.FailureWords(cover, m.failure.Kind, m.failure.Detail)
		return render.StateBlock(m.grammar(), w, word, body)
	case !f.Read:
		return render.StateBlock(m.grammar(), w, m.loadingLine(f), "")
	}
	return render.StateBlock(m.grammar(), w, m.emptyHeading(f), render.EmptyBody(cover))
}

// loadingLine and emptyHeading are the pane's own two messages, chosen by the feed on screen and
// worded by the cover: the same two states, in the words the tool on screen uses.
func (m Model) loadingLine(f frame) string {
	return render.LoadingLine(m.covered(), f.List.Kind)
}

func (m Model) emptyHeading(f frame) string {
	return render.EmptyHeading(m.covered(), f.List.Kind)
}

// helpRows is the one keymap table mapped into the overlay's rows: the keystrokes through the
// table's own join rule, the help text rendered verbatim and the group carried along (the overlay
// does not print it). This is the third surface, and it reads the same rows the input handling
// does, so a new key is one row.
//
// The wording is the row's own decision (keys.Row.HelpText): under a cover the rows whose plain
// wording names the product are re-worded, and the overlay's shape — a key list — is the one thing
// a cover does not disguise, because the user asked for it.
func helpRows(covered bool, screen string) []render.HelpRow {
	src := keys.ForScreen(screen)
	rows := make([]render.HelpRow, 0, len(src))
	for _, row := range src {
		rows = append(rows, render.HelpRow{Keys: row.KeysJoined(), Text: row.HelpText(covered), Group: row.Group})
	}
	return rows
}

// scrolled drops the rows that sit above the pane's first row: the list's blocks are laid out from
// the top, and skip (model.go) is what says where the pane starts — the frame's own top while the
// reader has not moved in it, and the anchor's place from their first move on. The rows of a block
// that is partly scrolled out are its own — a slice of them, never a re-wrap — and the whole block
// list was built at the pane's current width, so a resize re-measures rather than reusing lines. The
// block list is always rows of the list in order, never re-sorted.
//
// It measures the very block list it is given — the one blocksAt built for the grammar this pane is
// about to be painted with — so the scroll and the rows are one value's: no grammar can move the
// selection out of the pane by being counted with another one's rows.
func (m Model) scrolled(blocks []render.Block, f frame) []render.Block {
	rows, selected := geometry(blocks, f)
	if selected < 0 {
		return blocks
	}
	skip := m.skip(f, rows[selected])
	if skip <= 0 {
		return blocks
	}
	out := make([]render.Block, 0, len(blocks))
	for _, block := range blocks {
		switch {
		case skip >= len(block.Rows):
			skip -= len(block.Rows)
		case skip > 0:
			out = append(out, render.Block{PostID: block.PostID, Rows: block.Rows[skip:]})
			skip = 0
		default:
			out = append(out, block)
		}
	}
	return out
}

// status is the one line every frame carries: the state and its remedy from the process's failure,
// the notice, the list, the position and the budget. It is composed from values this package reads
// rather than owns — domain.Failure's own kind, the list's own accessors and the limiter behind
// Deps.Budget — so no field of the line is a second copy of a fact.
//
// Wearing a cover changes the line through one field: StatusFields.Cover. The state word's host, the
// list's name and the notice's words are the style's, and nothing else about the line moves — not
// the field order, not the drop rule, and not the budget, which is visible in every frame.
func (m Model) status() render.StatusFields {
	f := m.top()
	// The limiter's own reset is deliberately not read: the reset the state line prints is the
	// upstream's own (x-rate-limit-reset, carried on the failure), and the limiter's window is a
	// spacing rule rather than the account's rate limit.
	used, limited, _ := m.deps.Budget()

	fields := render.StatusFields{
		Failure: m.failure,
		// The account field exists only where more than one session does, and only outside a
		// cover: the name is the operator's own word for the file, and a covered line carries no
		// identity at all.
		Account:     m.accountField(),
		Notice:      m.noticeFor(f),
		NoticeCount: m.noticeCount,
		Cover:       m.covered(),
		Used:        used,
		Limited:     limited,
		Now:         m.deps.Now(),
		Gated:       m.grammar().Markers.Gated,
	}
	if f.Kind == frameThread {
		selected, loaded := f.Thread.Position()
		fields.List = render.ThreadListName(m.covered())
		fields.Selected = selected
		fields.Loaded = loaded
		fields.HasMore = f.Thread.HasMore()
		return fields
	}
	if f.Kind == frameSearch {
		if f.isPeople() {
			fields.List = render.ListLabelPeople(m.grammar(), f.People.Query())
			fields.Selected = f.People.Walk().Index(f.People.Walk().Selected().Top().ID) + 1
			fields.Loaded = f.People.Walk().Loaded()
			fields.HasMore = f.People.Walk().HasMore()
			return fields
		}
		fields.List = render.ListLabelSearch(m.grammar(), f.Search.Product(), f.Search.Query())
		fields.Selected = f.Search.Walk().Index(f.Search.Walk().Selected().Top().ID) + 1
		fields.Loaded = f.Search.Walk().Loaded()
		fields.HasMore = f.Search.Walk().HasMore()
		return fields
	}
	if f.Kind == frameProfile {
		if acct, ok := m.profileAccount(f); ok {
			fields.List = render.ListLabelProfile(m.grammar(), acct)
		}
		if f.Profile.Walk().Loaded() > 0 {
			fields.Selected = f.Profile.Walk().Index(f.Profile.Walk().Selected().Top().ID) + 1
		}
		fields.Loaded = f.Profile.Walk().Loaded()
		fields.HasMore = f.Profile.Walk().HasMore()
		return fields
	}
	if f.Kind == frameTrends {
		fields.List = render.ListLabelTrends(m.grammar())
		if len(f.Trends) > 0 {
			fields.Selected = f.TrendSel + 1
		}
		fields.Loaded = len(f.Trends)
		return fields
	}
	if f.Kind == frameAccounts {
		fields.List = render.ListLabelAccounts(m.grammar(), f.AccountsKind, f.Owner)
		if loaded := f.Accounts.Walk().Loaded(); loaded > 0 {
			fields.Selected = f.Accounts.Walk().Index(f.Accounts.Walk().Selected().Top().ID) + 1
		}
		fields.Loaded = f.Accounts.Walk().Loaded()
		fields.HasMore = f.Accounts.Walk().HasMore()
		return fields
	}
	if f.Kind == frameMedia {
		fields.List = render.ListLabelMedia(m.grammar(), f.Owner)
		if loaded := f.Media.Walk().Loaded(); loaded > 0 {
			fields.Selected = f.Media.Walk().Index(f.Media.Walk().Selected().Top().ID) + 1
		}
		fields.Loaded = f.Media.Walk().Loaded()
		fields.HasMore = f.Media.Walk().HasMore()
		return fields
	}
	if f.Kind == frameSaved {
		kind, _ := f.Saved.Kind().Saved()
		fields.List = render.ListSaved(m.covered(), kind)
		fields.Synced = render.SyncedField{On: true, At: f.Saved.Synced()}
		fields.Capability = f.Saved.Capability()
		fields.CapabilityList = kind.List()
		l := f.Saved.List()
		if l.Loaded() > 0 {
			fields.Selected = l.Index(l.Selected().Top().ID) + 1
		}
		fields.Loaded = l.Loaded()
		fields.HasMore = l.HasMore()
		// Capability occupies the state slots; process failure reappears on other frames.
		if fields.Capability != domain.CapNone {
			fields.Failure = domain.Failure{}
		}
		return fields
	}
	fields.List = render.ListName(m.covered(), f.List.Kind)
	fields.Selected = f.List.Index(f.List.Selected().Top().ID) + 1
	fields.Loaded = f.List.Loaded()
	fields.HasMore = f.List.HasMore()
	return fields
}

// noticeFor is the notice field: the last refresh's answer when there is one, and otherwise what
// the list itself says — a page in flight, or a walk that ended because the cursor stopped moving.
// The refresh's answer wins because it is the answer to the most recent deliberate action; a page
// request clears it (update.go's page), which is what lets the walk's own state be read afterwards.
func (m Model) noticeFor(f frame) render.NoticeKind {
	switch {
	case m.notice != render.NoticeNone:
		return m.notice
	case f.Kind == frameThread && f.Thread.LoadingOlder():
		return render.NoticeLoadingReplies
	case f.Kind == frameThread && f.Thread.Stalled():
		return render.NoticeStalled
	case f.Kind == frameSearch && f.isPeople() && f.People.Walk().Stalled():
		return render.NoticeStalled
	case f.Kind == frameSearch && f.isPeople() && f.People.Walk().LoadingOlder():
		return render.NoticeLoadingOlder
	case f.Kind == frameSearch && f.isPeople() && f.Read && f.People.Walk().Loaded() == 0 && m.failure.Kind == domain.FailureNone:
		return render.NoticeNoResults
	case f.Kind == frameSearch && !f.isPeople() && f.Search.Walk().Stalled():
		return render.NoticeStalled
	case f.Kind == frameSearch && !f.isPeople() && f.Search.Walk().LoadingOlder():
		return render.NoticeLoadingOlder
	case f.Kind == frameSearch && !f.isPeople() && f.Read && f.Search.Walk().Loaded() == 0 && m.failure.Kind == domain.FailureNone:
		return render.NoticeNoResults
	case f.Kind == frameProfile && f.Profile.Walk().Stalled():
		return render.NoticeStalled
	case f.Kind == frameProfile && f.Profile.Walk().Loaded() == 0:
		// A profile that holds no post yet is loading no *older* page of anything: the one request it
		// has out is its first post page, and the header's own two states are named here, ahead of the
		// walk's "older" wording, for that reason. The in-flight state is the walk's own latch, which
		// the header answer sets when it asks for the posts (screen.Profile.Update's HeaderFetched):
		// without it the pane would call an outstanding request an empty read that succeeded.
		if _, ok := f.Profile.Header(); ok {
			switch {
			case f.Profile.Walk().InFlight():
				return render.NoticeLoadingPosts
			case f.Read && m.failure.Kind == domain.FailureNone:
				return render.NoticeNoPosts
			}
		}
	case f.Kind == frameProfile && f.Profile.Walk().LoadingOlder():
		return render.NoticeLoadingOlder
	case f.Kind != frameThread && f.Kind != frameSearch && f.Kind != frameProfile && f.Kind != frameSaved && f.List.Stalled():
		return render.NoticeStalled
	case f.Kind != frameThread && f.Kind != frameSearch && f.Kind != frameProfile && f.Kind != frameSaved && f.List.LoadingOlder():
		return render.NoticeLoadingOlder
	case f.Kind == frameSaved && f.Saved.List().Stalled():
		return render.NoticeStalled
	case f.Kind == frameSaved && f.Saved.List().LoadingOlder():
		return render.NoticeLoadingOlder
	}
	return render.NoticeNone
}

// savedPane is the saved frame's pane: capability blocks, empty states, or the
// mirrored rows scrolled to the anchor.
func (m Model) savedPane(f frame, w int) []render.Block {
	if cap := f.Saved.Capability(); cap != domain.CapNone {
		kind, _ := f.Saved.Kind().Saved()
		facts := render.CapabilityFacts{
			Path:  m.deps.MirrorPath,
			Owner: f.Saved.Owner(),
			List:  kind.List(),
		}
		return render.CapabilityBlock(m.grammar(), cap, facts, w)
	}
	if len(f.Saved.List().Posts()) == 0 {
		if m.failure.Kind != domain.FailureNone {
			kind, _ := f.Saved.Kind().Saved()
			return render.StateBlockWithMirror(m.grammar(), m.failure, kind, false, w)
		}
		if !f.Read {
			return []render.Block{render.StateBlock(m.grammar(), w, render.SavedLoadingLine(m.covered()), "")}
		}
		kind, _ := f.Saved.Kind().Saved()
		return render.SavedEmptyState(m.grammar(), kind, !f.Saved.Synced().IsZero(), w)
	}
	return m.scrolled(m.blocksAt(f, w), f)
}

// profileAccount is the account a profile frame names in the status line's list field, and whether
// there is one to name yet.
//
// The header's account is authoritative once its answer has arrived: it is the identity the pane's
// own header row is painted with, so the two fields name one account one way.
//
// Before it has arrived the frame holds exactly one identity — the handle it was opened on — and
// that value is *not* the account to the alias function: a cover computes its alias from the
// account's id when it has one and from its handle when it has not, so a field built from the
// handle answers with one alias and the answered header with another. Measured under all four
// covers with the frame's own handle `alice` (header id `42`): `account: pump-sign-57` before the
// answer and `account: batch-restore-09` after it, on the same pane. A cover that renamed the
// account the moment its request answered would be a tell and a lie.
//
// The plain reader is not in that position: its field is the account's own handle, and the handle
// the frame was opened on is a true name for the account being read, so it prints it. A cover has
// no true name to print before the header, and a second alias is worse than none — so the field is
// empty until there is an account to name, exactly as the line's own drop rule leaves it empty when
// the line has no room for it.
func (m Model) profileAccount(f frame) (domain.User, bool) {
	if header, ok := f.Profile.Header(); ok {
		return header.User, true
	}
	if m.covered() == "" {
		return domain.User{ScreenName: f.Profile.Handle()}, true
	}
	return domain.User{}, false
}
