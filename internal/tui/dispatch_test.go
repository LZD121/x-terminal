package tui

import (
	"go/ast"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/keys"
)

// This file is the keymap-dispatch join, the key side of the guard cli_test.go's
// TestEveryVerbBodyIsReachableThroughCommands is for the verb table. The defect class is the one
// AUTH-03 shipped: `whoami` was implemented and its row in the table was deleted, so the binary
// answered `unknown command: whoami` for a verb it carried. Here the drift runs both ways across
// one seam — internal/keys/keys.go's table is what a key means, and internal/tui/update.go's
// press is the dispatcher that ends in `return m, nil` — and nothing joins them but this test:
//
//   - a row whose action nothing handles is a key the table documents and the shell ignores, a
//     documented no-op;
//   - a handler for an action no row binds is an arm no key can reach, dead code a reader meets
//     as if it were live.
//
// handledActions, next to the dispatcher in update.go, is the join point the test reads from
// both directions; the source sweep below is what keeps the enumeration from going stale, the way
// the cli guard's sweep of verb bodies does — it reads the action declarations themselves out of
// keys.go, so an action added without a row, without a handler and without an enumeration entry
// fails here at once.
//
// The one shape this package could hold that legitimately needs no key — a screen-local callback —
// does not exist in this build: the prompt's five editing keys and the panic gate's two are the
// table's own rows consulted under shadowed states, and focus loss enters panic through the same
// arm the key does. Were one added, the honest place for it is handledActions' comment, naming it
// and its reason — not a silent absence this test cannot tell from a forgotten one.

// actionDecl finds the action constants: an identifier beginning Action whose value is a string
// literal, which is the one shape keys.go declares actions in.
var actionDecl = regexp.MustCompile(`(?m)^\s*(Action[A-Za-z0-9]+)\s+Action\s*=\s*"`)

// TestEveryBoundActionIsHandledAndEveryHandledActionIsBound is the join, both directions.
func TestEveryBoundActionIsHandledAndEveryHandledActionIsBound(t *testing.T) {
	handled := handledActions
	inSet := make(map[keys.Action]bool, len(handled))
	for _, action := range handled {
		if inSet[action] {
			t.Errorf("handledActions names %q twice: the set is a lie about the switch it counts", action)
		}
		inSet[action] = true
	}

	// The table binds one action per row — no row shares its action, or the join by action would
	// be ambiguous — and every row's action is one the shell handles.
	bound := make(map[keys.Action]bool, len(keys.Rows()))
	for _, row := range keys.Rows() {
		if bound[row.Action] {
			t.Errorf("the keymap table binds %q twice", row.Action)
		}
		bound[row.Action] = true
		if !inSet[row.Action] {
			t.Errorf("the keymap binds %q (%s) and press has no arm for it: the key is a documented no-op",
				row.Action, row.KeysJoined())
		}
	}

	// Direction two: every action the shell handles is one the table binds. The test names the
	// action rather than an index, so the failure reads as the thing that drifted.
	for _, action := range handled {
		if !bound[action] {
			t.Errorf("press handles %q and no row of the keymap binds it: the arm is unreachable", action)
		}
	}

	// Anti-vacuity: both ends of the join are the size this build ships, or the test compared two
	// empty or halved sets and called it a join.
	if len(keys.Rows()) != len(handled) {
		t.Errorf("the table holds %d rows and handledActions names %d actions: the join is being read off two different tables",
			len(keys.Rows()), len(handled))
	}
	if len(handled) < 30 {
		t.Fatalf("handledActions names %d actions, fewer than the keymap's 30 rows: it is not the dispatcher's set", len(handled))
	}

	// HandledActions is the exported read of the same set, and it is a copy: a caller that writes
	// it must not move the shell's own enumeration.
	exported := HandledActions()
	if !slices.Equal(exported, handled) {
		t.Errorf("HandledActions() = %v, want the dispatcher's own %v", exported, handled)
	}
	exported[0] = keys.Action("scribbled")
	if handled[0] == keys.Action("scribbled") {
		t.Errorf("writing HandledActions()'s slice moved handledActions: it is not a copy")
	}
}

// TestTheShellDeclaresOnlyActionsTheKeymapDeclares is direction three, the sweep that keeps the
// enumeration honest: every action constant keys.go declares is a row's action, a handled action
// and a case label somewhere in the shell's sources — and the shell names no action constant the
// keymap package does not declare. The constants are typed strings, so a name that is not declared
// is already the compiler's to refuse (update.go's planted arm proves it does); this sweep is for
// the drift the compiler cannot see: declared, handled, bound, or enumerated in only some of the
// three places.
func TestTheShellDeclaresOnlyActionsTheKeymapDeclares(t *testing.T) {
	declared := declaredActions(t)
	if len(declared) < 30 {
		t.Fatalf("the sweep read %d action declarations out of keys.go, fewer than the table's rows: it is not reading the source", len(declared))
	}

	rows := keys.Rows()
	byAction := make(map[keys.Action]bool, len(rows))
	for _, row := range rows {
		byAction[row.Action] = true
	}
	inSet := make(map[keys.Action]bool, len(handledActions))
	for _, action := range handledActions {
		inSet[action] = true
	}

	// Every declared action is bound, handled and enumerated.
	for _, name := range declared {
		action := keys.Action(actionValue(t, name))
		if !byAction[action] {
			t.Errorf("keys.go declares %s (%q) and no row binds it: an action with no key is not an action", name, action)
		}
		if !inSet[action] {
			t.Errorf("keys.go declares %s and handledActions does not name it: the enumeration went stale", name)
		}
	}

	// Every constant the shell's sources name is one the keymap declares, bound and handled above.
	// The floor is this build's own count rather than the declaration count: an action declared
	// with no reference anywhere is a failure the arms above already name, not a sweep that read
	// nothing.
	used := shellActionRefs(t)
	if len(used) < 30 {
		t.Fatalf("the sweep found %d action references in the shell, fewer than this build's 30: it is not reading the sources",
			len(used))
	}
	declaredSet := make(map[string]bool, len(declared))
	for _, name := range declared {
		declaredSet[name] = true
	}
	for name := range used {
		if !declaredSet[name] {
			t.Errorf("the shell names keys.%s, which keys.go does not declare", name)
		}
	}

	// And every declared action is a case label the shell dispatches, not merely a name the
	// enumeration mentions: handledActions is data, and data without a switch arm is the silent
	// no-op this join exists to fail. The dispatcher's own arms, the cover gate and the saved
	// routing are all case labels over the action; an action named nowhere as a case is handled
	// by nothing.
	cases := shellActionCases(t)
	for _, name := range declared {
		if !cases[name] {
			t.Errorf("keys.%s is no case label in the shell: nothing dispatches it, and handledActions is data without an arm", name)
		}
	}
}

// declaredActions reads internal/keys/keys.go for the action constants, in declaration order. The
// path is resolved from this test's directory, the way boundary_test.go resolves the module root.
func declaredActions(t *testing.T) []string {
	t.Helper()
	src := readSource(t, filepath.Join("..", "keys", "keys.go"))
	var names []string
	for _, match := range actionDecl.FindAllStringSubmatch(src, -1) {
		names = append(names, match[1])
	}
	if len(names) == 0 {
		t.Fatal("the sweep found no action declarations in keys.go: the pattern is not reading the source")
	}
	return names
}

// actionValue reads one declaration's literal back out of keys.go, so the sweep compares actions
// by the value the table carries rather than by a lower-cased guess at the identifier.
func actionValue(t *testing.T, name string) string {
	t.Helper()
	src := readSource(t, filepath.Join("..", "keys", "keys.go"))
	match := regexp.MustCompile(`(?m)^\s*` + name + `\s+Action\s*=\s*"([^"]*)"`).FindStringSubmatch(src)
	if match == nil {
		t.Fatalf("keys.go declares no string value for %s", name)
	}
	return match[1]
}

// isActionConst is one action constant's name: the Action prefix, and more than the bare type
// name, which appears in signatures and the coverDigits slice type without being an action.
func isActionConst(name string) bool {
	return name != "Action" && strings.HasPrefix(name, "Action")
}

// shellActionRefs reads the shell's non-test sources for the action constants they name. The walk
// is over syntax, the way the compiler reads them: prose — including the planted arm's obituary in
// update.go — is not a reference.
func shellActionRefs(t *testing.T) map[string]bool {
	t.Helper()
	files := parsePackage(t, ".")
	if len(files) == 0 {
		t.Fatal("the sweep parsed no non-test sources in internal/tui: it is not reading the package")
	}
	found := make(map[string]bool)
	for _, parsed := range files {
		ast.Inspect(parsed.file, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "keys" && isActionConst(sel.Sel.Name) {
					found[sel.Sel.Name] = true
				}
			}
			return true
		})
	}
	return found
}

// shellActionCases is the same read restricted to case labels: the dispatcher's arms and every
// routed switch's. An action that appears only as data — the enumeration's own slice — is named
// here only if something dispatches it, which is the difference between handled and mentioned.
func shellActionCases(t *testing.T) map[string]bool {
	t.Helper()
	files := parsePackage(t, ".")
	if len(files) == 0 {
		t.Fatal("the sweep parsed no non-test sources in internal/tui: it is not reading the package")
	}
	found := make(map[string]bool)
	for _, parsed := range files {
		ast.Inspect(parsed.file, func(n ast.Node) bool {
			clause, ok := n.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, expr := range clause.List {
				ast.Inspect(expr, func(m ast.Node) bool {
					if sel, ok := m.(*ast.SelectorExpr); ok {
						if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "keys" && isActionConst(sel.Sel.Name) {
							found[sel.Sel.Name] = true
						}
					}
					return true
				})
			}
			return true
		})
	}
	return found
}

func readSource(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}
