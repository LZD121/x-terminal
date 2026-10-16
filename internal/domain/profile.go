package domain

import (
	"strconv"
	"time"
)

// SearchProduct is which ranking a search page was asked under. Latest is the zero
// value because reverse-chronological is the posture a `/` search starts in; Top is
// the engagement-ranked product the `s` toggle reaches; People is the account
// product `u` opens. There is deliberately no Media value: a product no screen can
// reach is a value nothing may construct.
//
// String returns the display words the status line and the prompt print (`latest`,
// `top`, `people`). The wire spellings (`Latest`, `Top`, `People`) stay below the
// port.
type SearchProduct uint8

const (
	// Latest is reverse-chronological search results.
	Latest SearchProduct = iota
	// Top is engagement-ranked search results.
	Top
	// People is account search results: one row per matching account.
	People
)

// String names the product in the words the status line and the prompt print.
func (p SearchProduct) String() string {
	switch p {
	case Latest:
		return "latest"
	case Top:
		return "top"
	case People:
		return "people"
	}
	return "search product " + strconv.Itoa(int(p))
}

// Profile is one account's header: the account itself plus the facts the profile
// screen prints. domain.User stays the post author's shape; these header facts
// live here and nowhere else, because the screen that prints them is the only
// consumer and because absent counts must stay distinct from zero counts.
//
// The three counts are pointers on purpose: a nil pointer is "the answer did not
// carry this", a pointer to zero is "the answer carried zero", and the facts row
// prints them differently (omit, never zero an absent value). Joined is the
// account's creation instant, parsed below the port — never a string above it.
type Profile struct {
	User      User
	Bio       string
	Followers *int
	Following *int
	Posts     *int
	Joined    time.Time
}

// UserPage is one page of accounts matching a people search: the accounts in the
// answer's order, where the walk continues, and what the answer cost. It mirrors
// PostPage's shape on purpose — the walk is entity-agnostic — and carries nothing
// about the wire: no product string, no cursor type, no sortIndex.
type UserPage struct {
	// Rows are the accounts the answer carried, in the answer's order. That order is
	// the list's order and no caller may re-sort it.
	Rows []User
	// Next is where the walk continues, empty at end-of-list.
	Next Cursor
	// Requested is the count of account entries the answer carried, as the adapter
	// counted them. It is what the answer cost, not what it contributed: a page whose
	// Requested is 20 and which adds zero unseen accounts is exactly the stall case.
	Requested int
	// FetchedAt is when the answer arrived, from the caller's clock.
	FetchedAt time.Time
}
