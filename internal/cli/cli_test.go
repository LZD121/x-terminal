package cli

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/fake"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/tui"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// TestABareInvocationOpensTheFeed is FEED-01's shape at the process's front door: the binary with no
// arguments opens the Following timeline in the pane — the one verb that opens a screen is the
// program's own surface — while the two invocations that are not a request for the timeline, `--help`
// and an unknown word, still print the usage and keep their exit codes. The composition's seams are
// held still here (the same two feedUnderTest replaces), so no terminal is opened and no socket is
// touched.
func TestABareInvocationOpensTheFeed(t *testing.T) {
	hermeticConfig(t, writeSession(t, t.TempDir(), "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n"))

	var (
		model   tui.Model
		opened  int
		limiter *xweb.Limiter
	)
	th := fake.ThreadShapes()
	previousSource, previousThread, previousRun := feedSource, threadSource, runProgram
	feedSource = func(client *xweb.Client, _ session.Session) domain.Source {
		limiter = client.Limiter
		return counting{Source: fake.Adversarial(), threads: th, limiter: client.Limiter}
	}
	threadSource = func(client *xweb.Client, _ session.Session) domain.Threads {
		return counting{Source: fake.Adversarial(), threads: th, limiter: client.Limiter}
	}
	runProgram = func(m tui.Model) error {
		opened, model = opened+1, m
		return nil
	}
	t.Cleanup(func() { feedSource, threadSource, runProgram = previousSource, previousThread, previousRun })

	if code := Main(nil); code != ExitOK {
		t.Fatalf("a bare invocation exited %d, want %d: it opens the feed rather than printing the usage", code, ExitOK)
	}
	if opened != 1 {
		t.Fatalf("a bare invocation opened the program %d times, want once", opened)
	}
	if limiter == nil {
		t.Fatalf("a bare invocation ran no composition: there is no model to drive")
	}
	// And it opened on Following: the first request the composition issues is for that list.
	model = feedRun(t, model, model.Init())
	status := feedStatus(model.Render(80, 24))
	if !strings.Contains(status, render.CopyListFollowing) {
		t.Errorf("the bare invocation's status line is %q, want it to name the Following list", status)
	}

	// The shapes that are not a bare invocation keep the usage path they had: `--help` is the one
	// requested usage, and an unknown word is a mistake.
	if code := Main([]string{"--help"}); code != ExitOK {
		t.Errorf("x-terminal --help exited %d, want %d", code, ExitOK)
	}
	if code := Main([]string{"frobnicate"}); code != ExitUsage {
		t.Errorf("x-terminal frobnicate exited %d, want %d", code, ExitUsage)
	}
}

// verbBodies is this package's verb surface written out by hand: one entry per exported body the
// table has to dispatch. It is spelled out rather than derived, so a body renamed, deleted or
// replaced breaks this test by name, and it is read from both directions below — a body with no row
// fails, and a row naming something that is not one of these fails.
//
// ShowNoticeOnce is deliberately absent, and so is ShowNotice. They are helpers rather than verbs:
// they take the marker path they write instead of argv, they return no exit code for a row to hand
// the process, and they are called by a body rather than by Main. The source sweep below is written
// against a verb body's own signature, so it does not see them either.
var verbBodies = []struct {
	name string
	body Handler
}{
	{name: "doctor", body: Doctor},
	{name: "logout", body: Logout},
	{name: "whoami", body: Whoami},
	{name: "feed", body: Feed},
	{name: "keys", body: Keys},
	{name: "sync", body: Sync},
	{name: "audit", body: Audit},
}

// TestEveryVerbBodyIsReachableThroughCommands is the guard for the defect AUTH-03 was: `whoami` was
// implemented, documented and tested, and its row in the table had been deleted, so the binary
// answered `unknown command: whoami` and exited 2 for a verb it carried. A body nothing dispatches
// is not a verb — this package's own promise is that "a verb absent here cannot be dispatched" —
// and this test fails when the two drift apart in either direction.
func TestEveryVerbBodyIsReachableThroughCommands(t *testing.T) {
	table := commands()
	dispatched := make(map[string]Handler, len(table))
	for _, cmd := range table {
		if _, twice := dispatched[cmd.Name]; twice {
			t.Errorf("commands() names %q twice: the second row can never run", cmd.Name)
		}
		dispatched[cmd.Name] = cmd.Run
	}
	// A row's name is its body's own name in lower case — the one convention the table and the
	// bodies share — so the two maps can be read against each other by name.
	listed := make(map[string]Handler, len(verbBodies))
	for _, verb := range verbBodies {
		listed[verb.name] = verb.body
	}

	// Direction one: every verb body this package exports is dispatched, by its own name, to itself.
	for _, verb := range verbBodies {
		run, ok := dispatched[verb.name]
		if !ok {
			t.Errorf("commands() has no %q row: %s is exported, implemented and unreachable from Main",
				verb.name, handlerName(verb.body))
			continue
		}
		if handlerPointer(run) != handlerPointer(verb.body) {
			t.Errorf("commands()'s %q row runs %s, want %s", verb.name, handlerName(run), handlerName(verb.body))
		}
	}

	// Direction two: the table names nothing that is not one of those bodies, so a wishful or
	// misspelled row cannot pass for a verb.
	for name := range dispatched {
		if _, ok := listed[name]; !ok {
			t.Errorf("commands() names %q, which is not a verb body in this package", name)
		}
	}

	// Direction three, which is what keeps the enumeration above from going stale: the package's own
	// sources are read for functions with a verb body's signature, so a body added without a row —
	// the shape of the defect this test exists for — is caught whether or not anybody listed it.
	bodies := exportedVerbBodies(t)
	if len(bodies) != len(verbBodies) {
		t.Errorf("this package holds %d verb bodies %v and the list above names %d: a body was added or removed without this list changing",
			len(bodies), bodies, len(verbBodies))
	}
	for _, name := range bodies {
		if _, ok := listed[strings.ToLower(name)]; !ok {
			t.Errorf("%s has a verb body's signature and no row in the table: dispatch it from commands(), or give it a helper's signature", name)
		}
	}
}

// handlerName is a Handler's own name, for the messages that have to say which body is which.
func handlerName(h Handler) string {
	if fn := runtime.FuncForPC(handlerPointer(h)); fn != nil {
		return fn.Name()
	}
	return "a handler with no name"
}

// handlerPointer is the identity of a top-level function: two func values cannot be compared to
// each other, only to nil, so two Handlers are the same body when their entry points are.
func handlerPointer(h Handler) uintptr {
	return reflect.ValueOf(h).Pointer()
}

// exportedVerbBodies reads this package's own sources and answers the names of the exported
// functions whose signature is a verb body's. Tests are excluded: a test names what it drives, and
// a helper there is not part of the surface the table dispatches.
func exportedVerbBodies(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading this package's directory: %v", err)
	}
	fset := token.NewFileSet()
	var names []string
	for _, entry := range entries {
		file := entry.Name()
		if !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() || !isVerbBodySignature(fn.Type) {
				continue
			}
			names = append(names, fn.Name.Name)
		}
	}
	if len(names) == 0 {
		t.Fatal("the sweep found no verb bodies at all: it is not reading this package's sources")
	}
	return names
}

// isVerbBodySignature is the shape a verb body has: the streams and the verb's own arguments in,
// the exit code out. Main's own signature — ([]string) int — is deliberately not it, because Main
// is the dispatcher, which is the one thing that is not a verb.
func isVerbBodySignature(sig *ast.FuncType) bool {
	if sig.TypeParams != nil || sig.Params == nil || sig.Results == nil {
		return false
	}
	params := sig.Params.List
	if len(params) != 2 || len(params[0].Names) != 1 || len(params[1].Names) != 1 {
		return false
	}
	star, ok := params[0].Type.(*ast.StarExpr)
	if !ok || !isIdent(star.X, "Context") {
		return false
	}
	slice, ok := params[1].Type.(*ast.ArrayType)
	if !ok || slice.Len != nil || !isIdent(slice.Elt, "string") {
		return false
	}
	results := sig.Results.List
	return len(results) == 1 && isIdent(results[0].Type, "int")
}

// isIdent reports whether expr is the named identifier.
func isIdent(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

// TestTheUsageListingIsAligned: the listing is the surface's own copy, rendered from the table, and
// a row whose description begins in a different column than its neighbours' is the whole defect
// this asserts against. It is checked on the rendered text rather than on the table alone, because
// the `--help` row and the hint are printed beside the table's rows and have to line up with them.
func TestTheUsageListingIsAligned(t *testing.T) {
	var out bytes.Buffer
	usage(&out)
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	// The header, a blank line, one row per table entry, the --help and --version rows and the hint.
	if want := len(commands()) + 5; len(lines) != want {
		t.Fatalf("the usage listing is %d lines, want %d:\n%s", len(lines), want, out.String())
	}

	// Groups: the table's rows, then the flag rows. Every row's description is an identifier, a run of
	// spaces and the words, and a group's column is set by its own widest name — the flags are wider
	// than any verb, which is why they are a second group rather than a row among the first.
	columnOf := func(rows []string) int {
		text, indented := strings.CutPrefix(rows[0], "  ")
		if !indented {
			t.Errorf("the usage row %q is not indented by two spaces, so it cannot line up", rows[0])
			return -1
		}
		_, description, _ := strings.Cut(text, " ")
		description = strings.TrimLeft(description, " ")
		return strings.Index(text, description)
	}
	for _, group := range [][]string{lines[2 : 2+len(commands())], lines[2+len(commands()) : len(lines)-1]} {
		want := columnOf(group)
		for _, row := range group {
			text, indented := strings.CutPrefix(row, "  ")
			if !indented {
				continue
			}
			name, description, _ := strings.Cut(text, " ")
			description = strings.TrimLeft(description, " ")
			if description == "" {
				t.Errorf("the %q row of the usage listing has no description: %q", name, row)
				continue
			}
			if at := strings.Index(text, description); at != want {
				t.Errorf("the %q row begins its description at column %d, want %d: the listing's columns line up within a group", name, at, want)
			}
		}
	}

	// The listing is rendered from the table, so a verb added to one is in the other without a
	// second place to edit — which is the property the table's own doc comment promises.
	for _, cmd := range commands() {
		if !strings.Contains(out.String(), cmd.Usage) {
			t.Errorf("the usage listing does not render the %q row", cmd.Name)
		}
	}
}

// TestWhoamiIsDispatchableFromTheFrontDoor drives Main, which is the path the binary itself takes:
// the verb is only a verb if `x-terminal whoami` reaches its body. Main builds its streams from the
// process's own, so this test is those streams for the call; the session and the upstream are
// fixtures — nothing here reads the operator's session file or dials x.com.
func TestWhoamiIsDispatchableFromTheFrontDoor(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, viewerAnswer)
	}))
	defer srv.Close()
	hermeticConfig(t, writeSession(t, t.TempDir(), "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n"))
	pointAtTestServer(t, srv)

	code, printed := mainOutput(t, "whoami")
	if code != ExitOK {
		t.Fatalf("x-terminal whoami exited %d, want %d: %s", code, ExitOK, printed)
	}
	if strings.Contains(printed, "unknown command") {
		t.Errorf("the front door answered one of its own verbs with %q", printed)
	}
	if !strings.Contains(printed, "@X — X") {
		t.Errorf("x-terminal whoami printed %q, want the account the session belongs to", printed)
	}
}

// mainOutput runs Main with the process's own streams pointed at one file, which is where Main
// builds its Context from, and answers the exit code and everything it printed. The file is why
// this needs no pipe and cannot block on a full buffer.
func mainOutput(t *testing.T, args ...string) (int, string) {
	t.Helper()
	sink, err := os.CreateTemp(t.TempDir(), "main-output")
	if err != nil {
		t.Fatalf("creating the output file: %v", err)
	}
	stdout, stderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = sink, sink
	code := Main(args)
	os.Stdout, os.Stderr = stdout, stderr
	if err := sink.Close(); err != nil {
		t.Fatalf("closing the output file: %v", err)
	}
	printed, err := os.ReadFile(sink.Name())
	if err != nil {
		t.Fatalf("reading what Main printed: %v", err)
	}
	return code, string(printed)
}
