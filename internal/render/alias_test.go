package render

// This file is DSGT-07's evidence: the alias function's exact outputs, its determinism and its
// totality, and — the structural half, which is the one that cannot be checked by reading a call
// site — the proof that the package holding it cannot write anything, so a package that cannot write
// cannot write an alias table.

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/text"

	"github.com/LZD121/x-terminal/internal/domain"
)

// TestCoverAliasWorkedExamples is the UI-SPEC's own worked-example table, byte for byte: the fixture
// pair's two accounts, in every style. The style is part of the signature and not of the mapping, so
// the same four projections come back for all four — that is the test of the claim, and it is one
// loop rather than four expectations.
func TestCoverAliasWorkedExamples(t *testing.T) {
	tests := []struct {
		name string
		user domain.User
		want CoverAlias
	}{
		{
			name: "aergon",
			user: coverAuthor,
			want: CoverAlias{
				Handle: "agent-rollback-70",
				Name:   "Anya Karlsson",
				Login:  "anya.karlsson",
				Email:  "anya.karlsson70@corp.internal",
			},
		},
		{
			name: "lzd",
			user: coverQuoted,
			want: CoverAlias{
				Handle: "guard-migrate-48",
				Name:   "Clara Sorensen",
				Login:  "clara.sorensen",
				Email:  "clara.sorensen48@corp.internal",
			},
		},
	}
	for _, tc := range tests {
		for _, style := range CoverStyles() {
			if got := CoverAliasOf(style, tc.user); got != tc.want {
				t.Errorf("CoverAliasOf(%q, %s) = %+v, want %+v", style, tc.name, got, tc.want)
			}
		}
		// The plain reader's identity is its own seam and not this function's: Plain projects the
		// handle the account really has, which is what makes a covered frame's identity the only
		// identity on screen that is not the account's own.
		if plain := Plain.Identity(tc.user); plain.Handle != tc.user.ScreenName {
			t.Errorf("the plain identity of %s is %q, want the account's own handle %q", tc.name, plain.Handle, tc.user.ScreenName)
		}
	}
}

// TestCoverAliasIsStableAndTotal is the function's two properties as a table.
//
// Stable: the same account maps to the same four projections however it is asked, in whatever order,
// however often — the mapping is a function of the id alone, so a restart, a second process and a
// second call are the same call. Total: every account renders four non-empty fields, including one
// with no id and no handle at all, which the anonymous key covers with a fixed alias rather than with
// an empty identity column.
//
// And the absence that makes DSGT-06 and DSGT-07 compatible: the account's own strings — its handle,
// its display name, its id — are inputs to the fingerprint and in none of its projections. The
// fixture's accounts really do carry the handle and the name the walk below searches for.
func TestCoverAliasIsStableAndTotal(t *testing.T) {
	fixture := []domain.User{
		coverAuthor, coverQuoted,
		{ID: "7", ScreenName: "someone", Name: "Some Body"},
		{ID: "9", ScreenName: "MixedCase"},
		{ScreenName: "@with-sigil"},
		{},
	}
	secrets := 0
	for _, u := range fixture {
		if u.ScreenName != "" {
			secrets++
		}
		if u.Name != "" {
			secrets++
		}
	}
	if secrets == 0 {
		t.Fatal("no account under test carries a handle or a name: the absence below would be vacuous")
	}

	for _, u := range fixture {
		for _, style := range append([]CoverStyle{""}, CoverStyles()...) {
			first := CoverAliasOf(style, u)
			for _, again := range []CoverAlias{CoverAliasOf(style, u), CoverAliasOf(style, u)} {
				if again != first {
					t.Fatalf("the account %q renders %+v and then %+v: the alias is not a function of the account", u.ScreenName, first, again)
				}
			}
			for _, field := range []struct{ name, value string }{
				{"Handle", first.Handle}, {"Name", first.Name}, {"Login", first.Login}, {"Email", first.Email},
			} {
				if strings.TrimSpace(field.value) == "" {
					t.Errorf("the account %q renders an empty %s under %q, so an identity column can be empty", u.ScreenName, field.name, style)
				}
			}
			if style == "" {
				continue
			}
			rendered := first.Handle + " " + first.Name + " " + first.Email
			for _, secret := range []string{u.ScreenName, strings.TrimPrefix(u.ScreenName, "@"), u.Name, u.ID} {
				if len(secret) < 3 {
					continue // a one-digit id is in every number a reference token prints, and is not an identity
				}
				if strings.Contains(rendered, secret) {
					t.Errorf("the account %q renders its own %q under %q: %q", u.ScreenName, secret, style, rendered)
				}
			}
		}
	}

	// The anonymous key, byte for byte: an author the upstream named nowhere still renders one fixed
	// alias, and it is this one.
	anonymous := CoverAliasOf(CoverServiceLog, domain.User{})
	want := CoverAlias{
		Handle: "relay-lease-65",
		Name:   "Ines Adeyemi",
		Login:  "ines.adeyemi",
		Email:  "ines.adeyemi65@corp.internal",
	}
	if anonymous != want {
		t.Fatalf("an account with no id and no handle renders %+v, want the fixed %+v", anonymous, want)
	}

	// The two halves of the key rule: an id wins over the handle, and a handle is lower-cased with its
	// sigil off, so the same account cannot render two aliases because the upstream spelled it two ways.
	if got, want := CoverAliasOf(CoverGitLog, domain.User{ID: coverQuoted.ID, ScreenName: "anything at all"}),
		CoverAliasOf(CoverGitLog, coverQuoted); got != want {
		t.Errorf("an account with an id renders %+v when its handle is spelled differently, want %+v: the id is the key", got, want)
	}
	if got, want := CoverAliasOf(CoverGitLog, domain.User{ScreenName: "@LZD"}),
		CoverAliasOf(CoverGitLog, domain.User{ScreenName: "lzd"}); got != want {
		t.Errorf("the handle @LZD renders %+v and lzd renders %+v: the key is lower-cased and bare", got, want)
	}

	// Two accounts are two machine names in one frame: the fixture's pair, and the anonymous key, are
	// three distinct aliases — which is what makes a covered list followable.
	handles := map[string]string{}
	for _, u := range []domain.User{coverAuthor, coverQuoted, {}} {
		handle := CoverAliasOf(CoverCI, u).Handle
		if other, seen := handles[handle]; seen {
			t.Errorf("the accounts %q and %q share the machine name %q", other, u.ScreenName, handle)
		}
		handles[handle] = u.ScreenName
	}
}

// TestCoverWordLists is the word lists' own contract: the four frozen lengths, every word ASCII and
// space-free, and the two widest projections the identity columns are sized against — the longest
// machine name (18 cells, which is why a cover's identity field is 20) and the longest login (17).
// A substituted word recomputes these numbers and the worked examples in the same commit.
func TestCoverWordLists(t *testing.T) {
	for _, list := range []struct {
		name  string
		words []string
		want  int
	}{
		{"ROLES", coverRoles[:], 24},
		{"NOUNS", coverNouns[:], 64},
		{"FIRST", coverFirst[:], 48},
		{"SURNAME", coverSurnames[:], 64},
	} {
		if len(list.words) != list.want {
			t.Errorf("the %s list holds %d words, want %d", list.name, len(list.words), list.want)
			continue
		}
		seen := map[string]bool{}
		for _, word := range list.words {
			if word == "" || !isPlainASCII(word) || strings.ContainsAny(word, " \t") {
				t.Errorf("the %s list holds %q, which is not a plain ASCII word", list.name, word)
			}
			if seen[word] {
				t.Errorf("the %s list holds %q twice: the index of a word behind it would move", list.name, word)
			}
			seen[word] = true
		}
	}

	// The widest machine name and the widest login, over the whole cross product the projection walks.
	slug, login := 0, 0
	for _, role := range coverRoles {
		for _, noun := range coverNouns {
			slug = max(slug, text.Width(role+"-"+noun+"-99"))
		}
	}
	for _, first := range coverFirst {
		for _, surname := range coverSurnames {
			login = max(login, text.Width(first+"."+surname))
		}
	}
	if slug != 18 {
		t.Errorf("the widest machine name is %d cells, want the 18 the 20-cell identity field is sized against: %v", slug, coverRoles)
	}
	if login != 17 {
		t.Errorf("the widest login is %d cells, want the 17 the phase documents", login)
	}
	if slug >= 20 || login >= 20 {
		t.Errorf("a projection is %d or %d cells, which the covers' 20-cell identity fields cannot hold: %d/%d", slug, login, slug, login)
	}
}

// isPlainASCII reports whether s is made of the printable ASCII a terminal draws one cell each.
func isPlainASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x21 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

// writeImports is what the package may not import: os/exec is a subprocess (the phase's effects live
// behind one boundary and this package is not it), io and path/filepath are the filesystem's own
// packages, and math/rand and hash/maphash are the two ways a hash could stop being a function of its
// input — one random, one seeded per process. A path with one of these as its first element counts too,
// which is what catches io/fs and math/rand/v2: a second way to reach a file, run a command or draw a
// number is still a way. The list is the structural half of DSGT-07's no-alias-table proof.
//
// os is deliberately not on it. Phase 2's colour ladder reads TERM and NO_COLOR through os.Getenv, so a
// claim that the package does not import os would be false; what is true, and what the no-alias-table
// proof needs, is that os is only ever read from: readEnvCalls counts every os.Getenv and
// sourceFindings reports every other selector on the package. With io, path/filepath and os/exec
// unimportable, there is no way left to open, create, write or remove anything.
var writeImports = []string{"os/exec", "io", "path/filepath", "math/rand", "hash/maphash"}

// packageFiles is every non-test source file of this package, read from the directory rather than
// listed by hand, so that a file added to the package is scanned whether or not this test knows its
// name.
func packageFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading the package's directory: %v", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, name)
	}
	return files
}

// sourceFindings parses one Go source and reports, as the lines of a report, every import of a package
// that could write or that is the filesystem, every selector on the os package that is not os.Getenv,
// and every call to time.Now. It parses the source it is given and never the file it came from, which is
// what lets the self-test below feed it a string.
func sourceFindings(t *testing.T, name string, src []byte) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}
	var findings []string
	for _, imported := range file.Imports {
		path := strings.Trim(imported.Path.Value, `"`)
		for _, forbidden := range writeImports {
			if path == forbidden || strings.HasPrefix(path, forbidden+"/") {
				findings = append(findings, "imports "+path)
			}
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		switch {
		case ident.Name == "os" && selector.Sel.Name != "Getenv":
			findings = append(findings, "calls os."+selector.Sel.Name)
		case ident.Name == "time" && selector.Sel.Name == "Now":
			findings = append(findings, "calls time.Now")
		}
		return true
	})
	return findings
}

// readEnvCalls counts a source's os.Getenv calls: the arm under "os is only ever read from", so that the
// rule is checked against a call that is really in the package and not against nothing at all.
func readEnvCalls(src []byte) int {
	file, err := parser.ParseFile(token.NewFileSet(), "count.go", src, parser.SkipObjectResolution)
	if err != nil {
		return 0
	}
	calls := 0
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Getenv" {
			return true
		}
		if ident, ok := selector.X.(*ast.Ident); ok && ident.Name == "os" {
			calls++
		}
		return true
	})
	return calls
}

// TestRenderCannotWrite is the structural proof that the package holding the alias function cannot
// write anything, and therefore cannot write an alias table: no io, no path/filepath, no os/exec, no
// math/rand, no hash/maphash and no time.Now in any non-test source of the package, and its one os call
// is the colour ladder's read of the environment. The alias function's purity — no clock, no randomness,
// no process-seeded hash — is the same claim from the other side: those imports are how a package would
// read a clock or draw a random number, and the digest's own constants are what it uses instead.
//
// Five arms, because a scan that finds nothing proves nothing. The file set is the directory's and is
// named, so a package that lost its files fails. At least one file imports internal/text, so the scan
// really read this package's sources. At least one os.Getenv is in the package and nothing else on os is,
// so "os is only ever read from" is a statement about a call that is really there. The package's own test
// files — which do read files — are excluded, and the scanner itself is run against a source that carries
// os, os/exec and time.Now and against one that carries neither.
func TestRenderCannotWrite(t *testing.T) {
	files := packageFiles(t)
	want := []string{
		"alias.go", "block.go", "copy.go", "cover.go", "frame.go", "grammar.go", "help.go",
		"palette.go", "panic.go", "status.go",
	}
	having := map[string]bool{}
	for _, name := range files {
		having[name] = true
	}
	for _, name := range want {
		if !having[name] {
			t.Errorf("the package's source file %s is not in the scan: the directory read found %v", name, files)
		}
	}
	if len(files) < len(want) {
		t.Fatalf("the scan reads %d source files, fewer than the %d the package has", len(files), len(want))
	}

	readers, envReads := 0, 0
	for _, name := range files {
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		for _, finding := range sourceFindings(t, name, src) {
			t.Errorf("internal/render/%s %s", name, finding)
		}
		if strings.Contains(string(src), `"github.com/LZD121/x-terminal/internal/text"`) {
			readers++
		}
		envReads += readEnvCalls(src)
	}
	if readers == 0 {
		t.Fatal("no source file of the package imports internal/text: the scan did not read the package the alias function lives in")
	}
	if envReads == 0 {
		t.Fatal("no source file of the package reads the environment: the \"os is only ever read from\" arm has nothing to hold")
	}

	// The scanner's own test. Both sources are strings in this file: the check never writes a file to
	// look at one, which is the same discipline the package itself is held to.
	const dirty = `package render

import (
	"hash/maphash"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func f() {
	_ = os.Getenv("A")
	_ = os.WriteFile(filepath.Join("a", "b"), []byte("c"), 0o600)
	_, _ = io.WriteString(os.Stdout, "d")
	_ = rand.Intn(3)
	_ = maphash.MakeSeed()
	_, _ = exec.Command("true").Output()
	_ = time.Now()
}
`
	findings := sourceFindings(t, "dirty.go", []byte(dirty))
	for _, want := range []string{
		"imports io", "imports os/exec", "imports path/filepath", "imports math/rand", "imports hash/maphash",
		"calls os.WriteFile", "calls time.Now",
	} {
		found := false
		for _, finding := range findings {
			if finding == want {
				found = true
			}
		}
		if !found {
			t.Errorf("the scanner did not report %q in a source that carries it: %v", want, findings)
		}
	}
	if calls := readEnvCalls([]byte(dirty)); calls != 1 {
		t.Errorf("the scanner counted %d os.Getenv calls in a source with one", calls)
	}

	const clean = `package render

import "time"

func g(t time.Time) string { return t.Format("15:04") }
`
	if findings := sourceFindings(t, "clean.go", []byte(clean)); len(findings) != 0 {
		t.Errorf("the scanner reports %v in a source that neither writes nor reads the clock", findings)
	}
}

// pinRow is one account's line of the hand-written pin table: the account's id and own handle, and
// the four projections the alias function makes of it.
type pinRow struct {
	id     string
	handle string
	alias  CoverAlias
}

// pinPath is the pin table. It is testdata rather than a literal in this file because it is meant to
// be read as a document: a human writes the identity down once, and a change to the word lists, the
// hash or the key rule shows up as a diff of that document in the commit that made it.
const pinPath = "testdata/cover-alias-pins.txt"

// readPinTable reads the pin table: every line that is neither blank nor a comment, split on tabs,
// with the account's id, its own handle, and then the four projections in the struct's own order.
func readPinTable(t *testing.T) []pinRow {
	t.Helper()
	body, err := os.ReadFile(pinPath)
	if err != nil {
		t.Fatalf("reading the pin table %s: %v", pinPath, err)
	}
	rows, err := parsePins(string(body))
	if err != nil {
		t.Fatalf("reading the pin table %s: %v", pinPath, err)
	}
	return rows
}

// parsePins is the pin table's own format, in one place: a comment or a blank line is skipped, and a
// line is six tab-separated fields. Both readers of the file — this test and the second process
// below — go through it, so a line that parses for one parses for the other.
func parsePins(body string) ([]pinRow, error) {
	rows := make([]pinRow, 0, 4)
	for i, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 6 {
			return nil, fmt.Errorf("line %d is %d fields, want the six the header names: %q", i+1, len(fields), line)
		}
		rows = append(rows, pinRow{
			id:     fields[0],
			handle: fields[1],
			alias:  CoverAlias{Handle: fields[2], Name: fields[3], Login: fields[4], Email: fields[5]},
		})
	}
	return rows, nil
}

// TestCoverAliasPinTable is the phase's stability promise against a document the code cannot move:
// the UI-SPEC's own worked examples are written into testdata by hand, and every style has to
// reproduce them. Determinism inside one process is TestCoverAliasIsStableAndTotal's business; this
// is the same value written down where a change to it would have to be committed.
func TestCoverAliasPinTable(t *testing.T) {
	rows := readPinTable(t)
	if len(rows) < 2 {
		t.Fatalf("the pin table holds %d accounts: too few to be the UI-SPEC's worked examples", len(rows))
	}
	// The pin table is the same pair of accounts the worked-example table names, so the two artifacts
	// cannot drift apart silently.
	for _, want := range []domain.User{coverAuthor, coverQuoted} {
		found := false
		for _, row := range rows {
			if row.id == want.ID && row.handle == want.ScreenName {
				found = true
			}
		}
		if !found {
			t.Errorf("the pin table does not hold %s (id %s): the worked examples and the pinned identities are two copies of one table", want.ScreenName, want.ID)
		}
	}

	for _, row := range rows {
		account := domain.User{ID: row.id, ScreenName: row.handle}
		for _, style := range CoverStyles() {
			if got := CoverAliasOf(style, account); got != row.alias {
				t.Errorf("CoverAliasOf(%q, id %s) = %+v, want the pinned %+v", style, row.id, got, row.alias)
			}
		}
		// The arm: every projection is a projection of the account and never the account itself —
		// non-empty, and carrying neither the handle nor the id it was made from.
		for _, field := range []string{row.alias.Handle, row.alias.Name, row.alias.Login, row.alias.Email} {
			if field == "" {
				t.Errorf("the pinned alias of id %s has an empty field: the assertion above would pass on an empty value", row.id)
			}
			if strings.Contains(strings.ToLower(field), strings.ToLower(row.handle)) || strings.Contains(field, row.id) {
				t.Errorf("the alias %q of id %s carries the account's own identity: an alias is a projection of the account and never the account", field, row.id)
			}
		}
	}
}

// aliasHelperEnv is the marker the second process is started with. A test binary that finds it does
// not run the suite: it prints the aliases the pin table asks for and exits.
const aliasHelperEnv = "XTERM_COVER_ALIAS_HELPER"

// TestCoverAliasesAcrossProcesses is DSGT-07's "the same account keeps the same alias across
// sessions and restarts" claim made of two processes rather than one: this binary re-executes itself
// as a child, and the child — which shares no memory, no map order and no call order with this
// process — prints the aliases of the pin table's ids. The child's answers are compared with this
// process's own call and with the pinned table, so the alias is a function of the account and not of
// the process that happened to ask.
func TestCoverAliasesAcrossProcesses(t *testing.T) {
	if os.Getenv(aliasHelperEnv) == "1" {
		printAliasHelper()
		return
	}

	rows := readPinTable(t)
	if len(rows) == 0 {
		t.Fatal("the pin table holds no account: the second process would have nothing to answer for")
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestCoverAliasesAcrossProcesses")
	cmd.Env = append(os.Environ(), aliasHelperEnv+"=1")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("the second process: %v\n%s", err, out)
	}

	// The child's lines, parsed back: one per account and style.
	answered := make(map[string]CoverAlias, len(rows))
	lines := 0
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(line, "alias\t") {
			continue
		}
		fields := strings.Split(strings.TrimSuffix(line, "\r"), "\t")
		if len(fields) != 5 {
			t.Fatalf("the second process printed %q, want five tab-separated fields", line)
		}
		key := fields[1] + "/" + fields[2]
		got := CoverAlias{Handle: fields[3], Name: fields[4]}
		if previous, seen := answered[key]; seen && previous != got {
			t.Errorf("the second process answered %s twice, %+v then %+v", key, previous, got)
		}
		answered[key] = got
		lines++
	}
	if lines < len(rows)*len(CoverStyles()) {
		t.Fatalf("the second process answered %d times, want %d accounts in %d styles: the comparison below would be vacuous", lines, len(rows), len(CoverStyles()))
	}

	compared := 0
	for _, row := range rows {
		account := domain.User{ID: row.id, ScreenName: row.handle}
		for _, style := range CoverStyles() {
			got, seen := answered[row.id+"/"+string(style)]
			if !seen {
				t.Errorf("the second process answered nothing for %s in %q", row.id, style)
				continue
			}
			want := CoverAliasOf(style, account)
			if got != (CoverAlias{Handle: want.Handle, Name: want.Name}) {
				t.Errorf("the second process answered %+v for %s in %q, this process computes %+v", got, row.id, style, want)
				continue
			}
			if want != row.alias {
				t.Errorf("the pinned alias of %s is %+v and this process computes %+v", row.id, row.alias, want)
			}
			compared++
		}
	}
	if compared == 0 {
		t.Fatal("no account was compared: the second-process leg proves nothing")
	}
}

// printAliasHelper is the second process's whole job: read the pin table, print one line per account
// and style with the alias's handle and name, and exit. It prints rather than asserts, because the
// parent owns the comparison, and it prints the two projections that cannot contain a tab; the login
// and the email are derived from them and are compared in-process.
func printAliasHelper() {
	body, err := os.ReadFile(pinPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "the alias helper cannot read %s: %v\n", pinPath, err)
		os.Exit(1)
	}
	rows, err := parsePins(string(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "the alias helper cannot read %s: %v\n", pinPath, err)
		os.Exit(1)
	}
	out := bufio.NewWriter(os.Stdout)
	for _, row := range rows {
		account := domain.User{ID: row.id, ScreenName: row.handle}
		for _, style := range CoverStyles() {
			alias := CoverAliasOf(style, account)
			fmt.Fprintf(out, "alias\t%s\t%s\t%s\t%s\n", row.id, style, alias.Handle, alias.Name)
		}
	}
	_ = out.Flush()
	os.Exit(0)
}
