package domain

import "context"

// Translation is one post's body in another language, as the translation service answered it.
//
// Available is the whole of what this package may say about the answer's state. `true` means the
// service answered with a translation of the post asked for; `false` means the service answered and
// carries no translation for it — a real answer, from a service that had nothing to give. It never
// means "the request failed": a failure crosses this seam as an error, and a caller that read
// `Available: false` as a failure would print a state for an answer that arrived.
//
// No upstream word appears in this package. The service answers `translationState` as one of its own
// strings (`Success`, `Missing`, and states this build has not measured), and none of them is
// spelled here: the words a user reads are internal/render's, and the state a failure puts on the
// status line is the classifier's. An adapter that leaked a `translationState` through this type
// would make every screen carry a vocabulary this build cannot keep in step.
type Translation struct {
	// Text is the translated body, and it is non-empty exactly when Available is true.
	Text string
	// Available reports whether the service answered with a translation of the post that was asked
	// for. It is never the difference between "no answer" and "an empty answer" — see above.
	Available bool
}

// Translator is the translation read, phrased in this package's terms: one post's body, rendered in
// one language, as the upstream's own translation service answers it.
//
// It is declared here, by the code that consumes it, for the reason Source is (source.go): internal
// xsource implements it over the REST client and internal/fake implements it in tests, and the two
// packages that name it are the composition root and the shell. Like Source, it can say nothing
// about the wire: no HTTP status, no response body, no endpoint and no upstream state word crosses
// this seam.
//
// The language is a parameter rather than a field of the implementation because it is a setting, and
// a setting is resolved once at the composition root (internal/cli) — not carried in a value that is
// built per read and could then disagree with the file the user wrote.
type Translator interface {
	// Translate returns the post `id`'s body rendered in `lang`. An answer the service carries no
	// translation in is Translation{Available: false} with a nil error; a refusal or an unreadable
	// answer is an error, classified by the same code the rest of the read path uses.
	Translate(ctx context.Context, id PostID, lang string) (Translation, error)
}
