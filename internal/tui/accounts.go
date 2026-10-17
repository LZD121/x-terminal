package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/render"
)

// The account switch: `A` re-reads the pane as the next session in the store.
//
// A switch is a new shell, not a mutation of this one. Every cached fact — the frame stack, the
// walks, the translations, the vivid marks, the failure — belongs to the account that produced it,
// and carrying any of it across would show one account's timeline under another's session. So the
// port rebuilds the dependencies and the shell starts again from an empty frame, keeping only what
// is about the terminal rather than the account: its size, its display state and the cover it wears.

// switchAccount is `A`: the next account, or a notice when there is nothing to switch to.
func (m Model) switchAccount() (Model, tea.Cmd) {
	if m.deps.SwitchAccount == nil || len(m.deps.Accounts) < 2 {
		return m.setNotice(render.NoticeNoAccount, 0), nil
	}
	next := m.nextAccount()
	if next == "" {
		return m.setNotice(render.NoticeNoAccount, 0), nil
	}
	deps, err := m.deps.SwitchAccount(next)
	if err != nil {
		// The store's own sentence: a missing file, a looser mode, a symlink. Nothing has changed
		// in the pane, so the reader keeps reading as the previous account.
		return m.setNotice(render.NoticeAccountFailed, 0), nil
	}
	restarted := m.restarted(deps)
	return restarted, restarted.Init()
}

// nextAccount is the name after the active one, wrapping. It is the cycle's whole rule, and it is a
// function so the key's test can hold it without a shell.
func (m Model) nextAccount() string {
	if len(m.deps.Accounts) == 0 {
		return ""
	}
	at := -1
	for i, a := range m.deps.Accounts {
		if a.Name == m.deps.Active {
			at = i
		}
	}
	return m.deps.Accounts[(at+1+len(m.deps.Accounts))%len(m.deps.Accounts)].Name
}

// restarted is this shell's terminal-shaped state on the new account's dependencies: the size, the
// display state, the cover and the account list. Everything else is New's.
func (m Model) restarted(deps Deps) Model {
	next := New(deps)
	next.w, next.h = m.w, m.h
	next.state, next.style = m.state, m.style
	next.from = m.from
	next.entryPending = m.entryPending
	return next
}

// accountField is the status line's account field: the active session's name when there is more
// than one and the pane is not covered, and nothing otherwise.
func (m Model) accountField() string {
	if m.covered() != "" || len(m.deps.Accounts) < 2 {
		return ""
	}
	return m.deps.Active
}
