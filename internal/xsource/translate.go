package xsource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/LZD121/x-terminal/internal/domain"
)

// ErrNoTranslator is the typed "this build has no translation endpoint" state: the adapter was built
// without one, which is a wiring fact and not a failure of any request. A caller that receives it
// reports the absence in its own words rather than printing a state about the network.
var ErrNoTranslator = errors.New("xsource: this adapter has no translation endpoint; call WithTranslator before Translate")

// Translator is the one call the translation adapter makes of the transport: one REST GET by post id
// and language, answered with a response body or with an error that already names its state.
//
// It is a second, small interface rather than a widening of GraphQLer, and the reason is the one
// GraphQLer's own doc comment gives for itself: `GraphQL` is the one *GraphQL operation* call this
// adapter makes, and this is not a GraphQL operation. The translation service is a REST endpoint on
// another origin with no query id, no feature switches and no registry entry, so nothing about it
// belongs in the type that resolves operation names — and an adapter that reached it through
// GraphQLer would have to spell an operation name that no bundle carries.
type Translator interface {
	Translate(ctx context.Context, id, lang string) ([]byte, error)
}

// WithTranslator returns a copy of the adapter that reads translations through t.
//
// It is a copy rather than a mutation for the same reason WithViewerID is: a composition root can
// build the adapter and the client once and hand both the same session. A nil translator — a build
// with no translation endpoint — is not a panic: Translate answers the typed ErrNoTranslator, so the
// absence is a value a shell can report rather than a crash in a keypress.
func (a *Adapter) WithTranslator(t Translator) *Adapter {
	copied := *a
	copied.translator = t
	return &copied
}

// Translate reads one post's body in another language.
//
// Exactly one call goes out and its bytes go to the parser; nothing here retries. A refusal is a
// second diagnosis rather than a second attempt, which is the discipline RUNBOOK.md's first section
// states, and the transport has already classified the refusal into a typed *xweb.APIError that
// names its state.
func (a *Adapter) Translate(ctx context.Context, id domain.PostID, lang string) (domain.Translation, error) {
	if a == nil || a.translator == nil {
		return domain.Translation{}, ErrNoTranslator
	}
	raw, err := a.translator.Translate(ctx, string(id), lang)
	if err != nil {
		// Passed through unchanged, as every other read here passes a refusal through: re-wrapping
		// it would put this package between the caller and the sentence the user needs.
		return domain.Translation{}, err
	}
	return ParseTranslation(raw, id)
}

// translationAnswer is the shape of a translate answer, and it declares exactly the three fields
// this build reads: the id the answer is about, the text, and the service's own state word. The rest
// of the response (`sourceLanguage`, `localizedSourceLanguage`, `translationSource`,
// `destinationLanguage`, `entities`) is deliberately not spelled out: a struct that carried the
// whole response would be a second copy of a schema to keep in step with the upstream, and nothing
// in this program reads those fields.
//
// `id_str` and not `id`: measured 2026-10-16 the response carries both, and `id` is the same value
// as a JSON number. `id_str` is the shape every other parser in this package compares an identity
// in, so the check that the answer is about the post that was asked for reads one type.
type translationAnswer struct {
	IDStr            string `json:"id_str"`
	Translation      string `json:"translation"`
	TranslationState string `json:"translationState"`
}

// translationStateSuccess is the one `translationState` this build treats as an answer carrying text
// (measured 2026-10-16: a `Success` body for a live post, and a `Missing` body for a post id that
// does not exist — which arrived under a `200`).
const translationStateSuccess = "Success"

// ParseTranslation reads a translate answer into one domain.Translation. It is pure over the bytes,
// and it is the whole of the honesty rule this feature turns on: **a `200` does not mean a
// translation exists**.
//
// Measured 2026-10-16: a nonexistent post id answered `HTTP 200` with `"translation":""` and
// `"translationState":"Missing"`. So a pane that rendered the body's `translation` field on any
// successful read would present an empty post as a translation of the reader's post — the one
// outcome this function exists to refuse. Three arms, in order:
//
//   - A body that is not JSON, or that carries no `id_str`, is a typed error naming the JSON path.
//     This is detail.go's idiom (a first page that carries no focal post is refused for the same
//     reason): a body this build cannot read must never render as a post with no translation.
//   - A body whose `id_str` is not the post that was asked for is a typed error too, in detail.go's
//     own words — an answer about a different post is an answer to a different question, and
//     rendering it would put somebody else's words under this reader's selection.
//   - A body that is about the right post with `translationState == "Success"` and a non-empty
//     `translation` is `Available: true` with the text.
//
// Everything else — `Missing`, a state this build has not measured, `Success` with an empty
// translation — is `Available: false` with **no error**, because that answer *is* the service
// answering. It carries no translation for this post, and the notice field says so.
func ParseTranslation(raw []byte, id domain.PostID) (domain.Translation, error) {
	var answer translationAnswer
	if err := json.Unmarshal(raw, &answer); err != nil {
		return domain.Translation{}, unreadable("the translation answer", err)
	}
	if answer.IDStr == "" {
		return domain.Translation{}, missing("id_str")
	}
	if answer.IDStr != string(id) {
		return domain.Translation{}, &ParseError{
			Path:   "id_str",
			Detail: fmt.Sprintf("is %s, and the post this translation was asked for is %s: an answer about a different post is an answer to a different question", answer.IDStr, id),
		}
	}
	if answer.TranslationState != translationStateSuccess || answer.Translation == "" {
		return domain.Translation{Available: false}, nil
	}
	return domain.Translation{Text: answer.Translation, Available: true}, nil
}

// Adapter implements the port's translation read. The assertion is here rather than in the tests
// because a signature change in one of the two packages has to fail a build rather than a keypress.
var _ domain.Translator = (*Adapter)(nil)
