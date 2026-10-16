package xsource

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

// translatePostID is the post the parser's bodies are about. It is a synthetic id, not a real
// post's: this file pins the shape of an answer and never a post.
const translatePostID = domain.PostID("2110709021544153425")

// The synthetic bodies the parser is read against. Their field names and states are the ones the
// live endpoint was measured answering with on 2026-10-16 (`.planning/research/UPSTREAM-PROBE.md`)
// — `id_str`, `translation`, `translationState` — and none of them is a real post's text: the
// translation here is ASCII this file wrote, because a real post's body must never land in the tree.
//
// Why a synthetic body and not a recording. Every other parser in this package is driven by a
// checked-in recording, and this one cannot be: a recording of a real translation *is* a real post's
// body in another language, and this project's rule is that no real post text lands in a checked-in
// fixture (internal/xsource/testdata/provenance.json). What the live measurement therefore records
// is a *shape* — the field names, the states, the status codes — and this file pins that shape.
const (
	translationSuccessBody = `{"id":2110709021544153425,"id_str":"2110709021544153425",` +
		`"translation":"a synthetic translated body","translationState":"Success",` +
		`"sourceLanguage":"ja","localizedSourceLanguage":"ja",` +
		`"translationSource":"Google","destinationLanguage":"zh-CN","entities":[]}`

	// translationMissingBody is the shape the service answers with when it has no translation to
	// give: measured 2026-10-16, a post id that does not exist answered HTTP 200 with an empty
	// translation and `translationState:"Missing"` — under a 200, which is the reason the parser has
	// an Available flag at all. The measured body carried `"id_str":"1"`, the id it was asked about,
	// so that body's own arm is the different-id case below; this is the same shape carrying this
	// file's post, which is the state a post with nothing to translate for the target language
	// answers.
	translationMissingBody = `{"id":2110709021544153425,"id_str":"2110709021544153425","translation":"","translationState":"Missing",` +
		`"sourceLanguage":"ja","localizedSourceLanguage":"ja",` +
		`"translationSource":"","destinationLanguage":"zh-CN","entities":[]}`

	// translationMeasuredMissingBody is the measured body itself, byte for byte as far as the fields
	// this build reads: `id_str` is `1`, the nonexistent id the measurement asked about. It is here
	// as a case of its own because it is the one Missing answer that was really observed, and it
	// reaches the parser through the different-id arm.
	translationMeasuredMissingBody = `{"id":1,"id_str":"1","translation":"","translationState":"Missing",` +
		`"destinationLanguage":"zh-CN"}`

	// translationOtherPostBody is about a different post than the one asked for.
	translationOtherPostBody = `{"id_str":"9999999999999999999","translation":"somebody else's words",` +
		`"translationState":"Success"}`

	// translationUnknownStateBody is a state this build has not measured. It is treated exactly as
	// Missing is: the service answered and carries no translation, so it is Available false with no
	// error rather than a guess at what the word means.
	translationUnknownStateBody = `{"id_str":"2110709021544153425","translation":"","translationState":"Unmeasured"}`

	// translationEmptySuccessBody is the case a state check alone would let through: a `Success`
	// state with nothing in the body. It must not render as a translation.
	translationEmptySuccessBody = `{"id_str":"2110709021544153425","translation":"","translationState":"Success"}`
)

// TestParseTranslationIsTheHonestyRule pins the parser's three arms, and the one of them that the
// whole feature turns on: a 200 whose translationState is not Success, or whose translation is
// empty, is Available false with **no error**, because that answer is the service answering. A pane
// that rendered those bytes as a translation would put an empty body under the reader's post.
func TestParseTranslationIsTheHonestyRule(t *testing.T) {
	for _, tc := range []struct {
		name          string
		body          string
		wantAvailable bool
		wantText      string
		wantErrPath   string
	}{
		{
			name:          "about this post and carrying text",
			body:          translationSuccessBody,
			wantAvailable: true,
			wantText:      "a synthetic translated body",
		},
		{
			name:          "a Missing state is an answer with no translation",
			body:          translationMissingBody,
			wantAvailable: false,
		},
		{
			name:        "the measured Missing body is about the id it was asked for, so it reaches the different-id arm",
			body:        translationMeasuredMissingBody,
			wantErrPath: "id_str",
		},
		{
			name:          "a state this build has not measured is treated the same way",
			body:          translationUnknownStateBody,
			wantAvailable: false,
		},
		{
			name:          "Success with an empty body is not a translation",
			body:          translationEmptySuccessBody,
			wantAvailable: false,
		},
		{
			name:        "an answer about a different post is an answer to a different question",
			body:        translationOtherPostBody,
			wantErrPath: "id_str",
		},
		{
			name:        "a body that is not JSON names the path",
			body:        `not json at all`,
			wantErrPath: "the translation answer",
		},
		{
			name:        "an empty body names the path",
			body:        ``,
			wantErrPath: "the translation answer",
		},
		{
			name:        "a JSON body with no id_str names the path",
			body:        `{"translation":"x","translationState":"Success"}`,
			wantErrPath: "id_str",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTranslation([]byte(tc.body), translatePostID)
			if tc.wantErrPath != "" {
				var parseErr *ParseError
				if !errors.As(err, &parseErr) {
					t.Fatalf("ParseTranslation reported %v, want a *ParseError naming %s", err, tc.wantErrPath)
				}
				if !strings.Contains(parseErr.Error(), tc.wantErrPath) {
					t.Errorf("the error is %q, want it to name %q", parseErr.Error(), tc.wantErrPath)
				}
				if got.Available || got.Text != "" {
					t.Errorf("a refused body produced %+v, want nothing", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTranslation: %v", err)
			}
			if got.Available != tc.wantAvailable {
				t.Errorf("Available is %v, want %v", got.Available, tc.wantAvailable)
			}
			if got.Text != tc.wantText {
				t.Errorf("Text is %q, want %q", got.Text, tc.wantText)
			}
			// The invariant the two fields hold between them: text is present exactly when the
			// service said it had some.
			if (got.Text != "") != got.Available {
				t.Errorf("the answer is %+v: Text and Available disagree about whether there is a translation", got)
			}
		})
	}
}

// recordingTranslator is the adapter arm's double: it records the one (id, language) it was handed
// and answers a scripted body or error. It is what makes "one call, the language handed through
// verbatim, the refusal passed through unchanged" assertable without a socket.
type recordingTranslator struct {
	body []byte
	err  error

	calls int
	id    string
	lang  string
}

func (r *recordingTranslator) Translate(_ context.Context, id, lang string) ([]byte, error) {
	r.calls++
	r.id, r.lang = id, lang
	return r.body, r.err
}

// TestTheAdapterAsksOnceAndPassesTheLanguageThrough pins the adapter's own contract: exactly one
// call per Translate, the post id and the language handed through verbatim (the language is a
// setting resolved above this package, and this package must not normalise it), and the parsed
// answer returned.
func TestTheAdapterAsksOnceAndPassesTheLanguageThrough(t *testing.T) {
	rec := &recordingTranslator{body: []byte(translationSuccessBody)}
	got, err := New(nil).WithTranslator(rec).Translate(context.Background(), translatePostID, "de")
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if rec.calls != 1 {
		t.Fatalf("the adapter issued %d calls, want exactly 1: a refusal is a second diagnosis, not a second attempt", rec.calls)
	}
	if rec.id != string(translatePostID) {
		t.Errorf("the translator was handed the id %q, want %q", rec.id, translatePostID)
	}
	if rec.lang != "de" {
		t.Errorf("the translator was handed the language %q, want it verbatim", rec.lang)
	}
	if !got.Available || got.Text != "a synthetic translated body" {
		t.Errorf("Translate returned %+v, want the answer's translation", got)
	}
}

// TestTheAdapterPassesARefusalThroughUnchanged: a classified refusal is already a state with a
// sentence, and re-wrapping it here would put this package between the caller and the words the user
// needs. The same *xweb.APIError must come back.
func TestTheAdapterPassesARefusalThroughUnchanged(t *testing.T) {
	refusal := &xweb.APIError{
		Operation: "translate",
		Outcome:   xweb.OutcomeRateLimited,
		Status:    429,
		Detail:    "slow down",
		Reset:     time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC),
	}
	rec := &recordingTranslator{err: refusal}
	got, err := New(nil).WithTranslator(rec).Translate(context.Background(), translatePostID, "de")
	var apiErr *xweb.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Translate reported %v, want the *xweb.APIError back", err)
	}
	if apiErr != refusal {
		t.Errorf("the error is %+v, want the identical value the translator answered with", apiErr)
	}
	if got.Available || got.Text != "" {
		t.Errorf("a refused call produced %+v, want nothing", got)
	}
	// And the refusal is classified, not re-read: no second call.
	if rec.calls != 1 {
		t.Errorf("the adapter issued %d calls for a refusal, want 1", rec.calls)
	}
}

// TestTheAdapterWithoutATranslatorSaysSo: a build with no translation endpoint is a wiring fact, and
// AskTheAdapterWithoutATranslator must answer the typed error rather than panicking in a keypress.
func TestTheAdapterWithoutATranslatorSaysSo(t *testing.T) {
	rec := &recordingTranslator{body: []byte(translationSuccessBody)}
	for _, tc := range []struct {
		name    string
		adapter *Adapter
	}{
		{"built without one", New(nil)},
		{"a nil adapter", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.adapter.Translate(context.Background(), translatePostID, "de")
			if !errors.Is(err, ErrNoTranslator) {
				t.Fatalf("Translate reported %v, want ErrNoTranslator", err)
			}
			if got.Available || got.Text != "" {
				t.Errorf("the absent translator produced %+v, want nothing", got)
			}
		})
	}
	// An adapter that was handed one is not in this state, so the assertion above is not vacuous.
	if _, err := New(nil).WithTranslator(rec).Translate(context.Background(), translatePostID, "de"); err != nil {
		t.Fatalf("an adapter with a translator reported %v, want the answer", err)
	}
	if rec.calls != 1 {
		t.Errorf("the control issued %d calls, want 1", rec.calls)
	}
}

// TestWithTranslatorDoesNotMutateTheAdapter is the copy contract WithViewerID established: a
// composition root builds the adapter once and hands the same one to both reads, so the wiring must
// be a copy rather than a mutation.
func TestWithTranslatorDoesNotMutateTheAdapter(t *testing.T) {
	base := New(nil)
	wired := base.WithTranslator(&recordingTranslator{body: []byte(translationSuccessBody)})
	if _, err := wired.Translate(context.Background(), translatePostID, "de"); err != nil {
		t.Fatalf("the wired copy reported %v, want the answer", err)
	}
	if _, err := base.Translate(context.Background(), translatePostID, "de"); !errors.Is(err, ErrNoTranslator) {
		t.Fatalf("the adapter the copy was made from reports %v, want ErrNoTranslator: WithTranslator mutated it", err)
	}
}

// TestCaptureTranslation is the live measurement this feature exists to close: it issues the
// product's own path for one post and one language through the product's own client, and then —
// after the measured spacing — issues the *suffixed* form of the same path, so both statuses the
// record carries are observed in one run.
//
// It is skipped unless X_TERMINAL_CAPTURE=1, so a normal `go test ./...` opens no socket. It prints
// the parsed Translation's Available and the text's **rune count**, never the text: a real post's
// translation must not land in a checked-in file, a log or a summary.
//
// Measurement discipline, and it is the reason this test exists rather than a curl: a statement
// about this upstream must come out of the product's own read path — internal/xweb's client, through
// the limiter — because curl with hand-assembled headers produced spurious `401 {"code":89}` on
// requests this client answered `200`. A curl result that disagrees is the curl's problem.
//
// The spacing is not decoration either: a rapid burst on this host answers
// `401 {"code":89,"message":"Invalid or expired token"}` (measured 2026-10-15), and a single 401 is
// retried **once** after another wait and never more than once — no loop, here or in the product.
func TestCaptureTranslation(t *testing.T) {
	if os.Getenv(captureGate) != "1" {
		t.Skipf("%s is not 1: this test issues two live translate requests and prints their outcomes, so it runs only when an operator asks for it", captureGate)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("the configuration this host reads with: %v", err)
	}
	for _, warning := range cfg.Warnings {
		t.Logf("config warning: %s", warning)
	}
	if _, err := volatile.Load(cfg.PinsPath()); err != nil {
		t.Fatalf("the operation table: %v", err)
	}
	sess, err := session.Load(cfg.SessionPath)
	if err != nil {
		t.Fatalf("the session this measurement reads with: %v", err)
	}
	for _, warning := range sess.Warnings {
		t.Logf("session warning: %s", warning)
	}
	client, err := xweb.NewClient(cfg)
	if err != nil {
		t.Fatalf("the transport: %v", err)
	}

	// The product's own path: the adapter, over the client, asked exactly the way the shell asks it.
	adapter := New(client.WithSession(sess)).WithTranslator(client.WithSession(sess))
	lang := cfg.TranslateLanguage
	t.Logf("the language this host resolves: %s", lang)

	translated, err := adapter.Translate(context.Background(), domain.PostID(captureTranslateID), lang)
	if err != nil {
		if isTransient401(err) {
			t.Logf("the first answer was the transient 401 this upstream answers inside the floor: waiting it out and asking once more, which is the one retry this harness is allowed")
			time.Sleep(captureSpacing)
			translated, err = adapter.Translate(context.Background(), domain.PostID(captureTranslateID), lang)
		}
		if err != nil {
			t.Fatalf("the product's own path did not answer: %v", err)
		}
	}
	// The first answer must carry a translation: "the endpoint answers" is asserted rather than
	// observed, so a run whose endpoint moved fails here instead of printing a quiet negative.
	if !translated.Available {
		t.Fatalf("the product's own path answered with no translation for the post the record names: Available is false, so this measurement proves nothing about the endpoint")
	}
	t.Logf("product path: Available=%v, translation runes=%d (never the text)", translated.Available, len([]rune(translated.Text)))

	// Then the control: the same path with the suffix appended, built here rather than by the
	// product, so the `404 {"code":34}` the record carries is observed rather than quoted. The wait
	// before it is the measured floor: a burst on this host is answered `401 {"code":89}`, and this
	// run has to be as far from a burst as the record's own run was.
	time.Sleep(captureSpacing)
	suffixed, status, outcome, err := captureSuffixed(client, sess, captureTranslateID, lang)
	if err != nil {
		t.Fatalf("the suffixed form's request: %v", err)
	}
	t.Logf("suffixed form: HTTP %d, classified %q, body %d bytes", status, outcome, len(suffixed))
	if status != 404 {
		t.Errorf("the suffixed form answered HTTP %d, want the measured 404: the record says appending `.json` is a false negative and this run must observe it", status)
	}
	if !strings.Contains(string(suffixed), `"code":34`) {
		t.Errorf("the suffixed form's body is %q, want the measured `{\"code\":34,...}` refusal", suffixed)
	}
}

// The two facts one live run needs: the post the record is about, and the spacing the measurement
// waits. The id is the public post the record names in `.planning/research/UPSTREAM-PROBE.md`; the
// spacing is the measured floor below which a burst is answered `401 {"code":89}`.
const (
	captureTranslateID = "2110709021544153425"
	captureSpacing     = 4 * time.Second
)

// captureSuffixed issues the same path with `.json` appended, through the product's own client, and
//
// It builds the URL and the request itself rather than asking the product for one, and that is the
// whole of the no-suffix rule: the product's builder must never be able to produce this form, so the
// only way to observe the measured false negative is for the harness to assemble it. The eight
// headers are set here from the exported values (WebBearer, UserAgent, the session's two cookies) —
// the same eight apiHeaders sets — because a request this arm sends without them would measure the
// authentication rather than the path. This file is not read by the audit's transport check (the
// audit reads non-test sources only), which is also why the shipped path carries no wait.
func captureSuffixed(client *xweb.Client, sess session.Session, id, lang string) (body []byte, status int, outcome string, err error) {
	raw := xweb.APIBaseURL + "/1.1/strato/column/None/tweetId=" + id +
		",destinationLanguage=Some(" + lang + "),translationSource=Some(X),feature=None,timeout=None,onlyCached=None" +
		"/translation/service/translateTweet.json"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, raw, nil)
	if err != nil {
		return nil, 0, "", err
	}
	req.Header.Set("authorization", "Bearer "+xweb.WebBearer)
	req.Header.Set("cookie", "auth_token="+sess.AuthToken.Reveal()+"; ct0="+sess.CTO.Reveal())
	req.Header.Set("x-csrf-token", sess.CTO.Reveal())
	req.Header.Set("x-twitter-auth-type", "OAuth2Session")
	req.Header.Set("x-twitter-active-user", "yes")
	req.Header.Set("x-twitter-client-language", "en")
	req.Header.Set("user-agent", xweb.UserAgent)
	req.Header.Set("content-type", "application/json")

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		return nil, 0, "", err
	}
	defer resp.Body.Close()
	answered, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, 0, "", err
	}
	return answered, resp.StatusCode, string(xweb.Classify(resp.StatusCode, resp.Header, answered)), nil
}

// isTransient401 reports whether one error is the transient 401 the measurement above retries once.
// It is the harness's own reading of the classified state: any 401 this upstream answers, named as
func isTransient401(err error) bool {
	var apiErr *xweb.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Status == 401
}
