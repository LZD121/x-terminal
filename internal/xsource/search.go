package xsource

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LZD121/x-terminal/internal/domain"
)

// opSearch is the search operation, by the name the registry holds. The id it resolves to is the
// registry's business, never a literal here (XAPI-02).
const opSearch = "SearchTimeline"

// searchPath is where the search answer's instructions live: the container the research recorded
// for SearchTimeline (.planning/research/ARCHITECTURE.md). The gated capture in capture_test.go
// is what verifies it against a live body — the path is the parser's literal and the test's, and
// the two move together if x.com nests it differently.
const searchPath = "data.search_by_raw_query.search_timeline.timeline.instructions"

// searchEnvelope is the part of a SearchTimeline response that carries the timeline: the search
// container, nested the way searchPath spells it. Only the fields this package reads are
// declared, exactly as in tweets.go.
type searchEnvelope struct {
	Data struct {
		SearchByRawQuery *struct {
			SearchTimeline *timelineNode `json:"search_timeline"`
		} `json:"search_by_raw_query"`
	} `json:"data"`
}

// ParseSearch reads one page of search results out of a SearchTimeline response body, into domain
// posts in the answer's order.
//
// The entries and cursors are read by the same instruction walk the user timelines feed
// (walkTimeline in tweets.go): a cursor and a tweet entry mean the same thing here, and a body
// with neither posts nor cursors is a typed error naming the path, never an empty page.
func ParseSearch(raw []byte) (Page, error) {
	var envelope searchEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Page{}, unreadable("the response body", err)
	}
	if envelope.Data.SearchByRawQuery == nil || envelope.Data.SearchByRawQuery.SearchTimeline == nil {
		return Page{}, missing(searchPath)
	}

	var page Page
	if err := page.walkTimeline(searchPath, envelope.Data.SearchByRawQuery.SearchTimeline); err != nil {
		return Page{}, err
	}
	return page, nil
}

// Search returns one page of a keyword search.
//
// The variables are the measured request's own four: the query, the count one page asks for, the
// product in the wire's own spelling, and the cursor (empty for page one). Nothing else is sent:
// an invented key is a claim about the upstream's contract that nothing here measured.
//
// This operation answers only over POST — measured 2026-10-15, with GET answering 404 and a
// zero-byte body and POST answering 200 with real results under the same id, variables, features,
// field toggles and headers. The method is the transport's decision, not this adapter's: it is a
// property of the operation, and internal/xweb/graphql.go is the one place that holds it.
func (a *Adapter) Search(ctx context.Context, query string, p domain.SearchProduct, c domain.Cursor) (domain.PostPage, error) {
	product, err := wireProduct(p)
	if err != nil {
		return domain.PostPage{}, err
	}

	raw, err := a.client.GraphQL(ctx, opSearch, map[string]any{
		"rawQuery": query,
		"count":    pageCount,
		"product":  product,
		"cursor":   string(c),
	})
	if err != nil {
		return domain.PostPage{}, err
	}

	page, err := ParseSearch(raw)
	if err != nil {
		return domain.PostPage{}, err
	}
	return domain.PostPage{
		Posts:     page.Posts,
		Next:      domain.Cursor(page.BottomCursor),
		Requested: page.Requested,
		FetchedAt: now(),
	}, nil
}

// wireProduct is the product in the wire's own spelling. The display words stay above the port
// (domain.SearchProduct.String); the spellings below it are the operation's contract. A product
// this build does not declare is an error, never a silently wrong ranking.
func wireProduct(p domain.SearchProduct) (string, error) {
	switch p {
	case domain.Latest:
		return "Latest", nil
	case domain.Top:
		return "Top", nil
	case domain.People:
		return "People", nil
	}
	return "", fmt.Errorf("xsource: search product %d is not one this build can ask for", int(p))
}

// peoplePage is one page of accounts read from a SearchTimeline People answer. It is a sibling
// of Page that collects users instead of posts, and it folds entries through the SAME
// instruction walk (walkInstructions / foldEntry) so the cursor typename has one reader.
type peoplePage struct {
	Users        []domain.User
	TopCursor    string
	BottomCursor string
	Requested    int
}

func (p *peoplePage) setTopCursor(v string)    { p.TopCursor = v }
func (p *peoplePage) setBottomCursor(v string) { p.BottomCursor = v }
func (p *peoplePage) empty() bool {
	return len(p.Users) == 0 && p.TopCursor == "" && p.BottomCursor == ""
}

// addItem reads one TimelineUser item into the page. An item that is not an account is skipped
// (tweet cards land in People answers the same way user cards land in tweet answers). An
// account entry whose node is unreadable is a typed error naming its JSON path.
func (p *peoplePage) addItem(entryID string, content itemContent) error {
	if content.Typename != "TimelineUser" {
		return nil
	}
	// Counted before anything can fail. Requested is what the answer carried.
	p.Requested++

	path := entryID + ": content.itemContent.user_results.result"
	if content.UserResults.Result == nil {
		return missing(path)
	}
	user, err := userFrom(path, content.UserResults.Result)
	if err != nil {
		return err
	}
	p.Users = append(p.Users, user)
	return nil
}

// ParseSearchPeople reads one page of accounts out of a SearchTimeline People response body.
// The entries and cursors ride the SAME instruction walk ParseSearch uses — one entry loop,
// one cursor reader — so a shape change in the walk has exactly one home. A body with neither
// accounts nor cursors is a typed error naming the path, never an empty page.
func ParseSearchPeople(raw []byte) (domain.UserPage, error) {
	var envelope searchEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return domain.UserPage{}, unreadable("the response body", err)
	}
	if envelope.Data.SearchByRawQuery == nil || envelope.Data.SearchByRawQuery.SearchTimeline == nil {
		return domain.UserPage{}, missing(searchPath)
	}

	var page peoplePage
	if err := walkInstructions(searchPath, envelope.Data.SearchByRawQuery.SearchTimeline, &page); err != nil {
		return domain.UserPage{}, err
	}
	return domain.UserPage{
		Rows:      page.Users,
		Next:      domain.Cursor(page.BottomCursor),
		Requested: page.Requested,
	}, nil
}

// SearchPeople returns one page of accounts matching the query under the People product.
// The body is read by ParseSearchPeople over the same instruction walk ParseSearch uses.
func (a *Adapter) SearchPeople(ctx context.Context, query string, c domain.Cursor) (domain.UserPage, error) {
	product, err := wireProduct(domain.People)
	if err != nil {
		return domain.UserPage{}, err
	}
	raw, err := a.client.GraphQL(ctx, opSearch, map[string]any{
		"rawQuery": query,
		"count":    pageCount,
		"product":  product,
		"cursor":   string(c),
	})
	if err != nil {
		return domain.UserPage{}, err
	}

	page, err := ParseSearchPeople(raw)
	if err != nil {
		return domain.UserPage{}, err
	}
	page.FetchedAt = now()
	return page, nil
}
