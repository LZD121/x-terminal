package audit

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"
)

// sourcePromise is the line this check prints. It stands first in report order
// because every Go scan behind it reads this same source: a file that does not
// parse is a file those scans cannot read, and this is the one place that says
// so, by name, instead of skipping it.
const sourcePromise = "every Go file the audit reads parses; a file that does not is reported here rather than skipped"

// checkSource is the load-level honesty check. It parses every non-test Go file
// the audit was handed and reports one finding per file that does not parse,
// naming the file, the parser's own message and the line it stopped at. More
// than one check reads the same source, so this finding is stated once, here,
// and never per check.
//
// The Go scans below count only the files they parsed among the files they
// inspected, so a file reported here is not counted as inspected anywhere: the
// drop is named in Files as well as in this finding.
func checkSource(t Tree) Report {
	const name = "source"
	goFiles := t.GoFiles()
	r := Report{Check: name, Promise: sourcePromise, Files: len(goFiles)}
	if len(goFiles) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "inspected no Go files: a scan of nothing proves nothing"))
		return r
	}
	for _, f := range goFiles {
		if err := parses(f); err != nil {
			r.Findings = append(r.Findings, parseFinding(name, f.Path, err))
		}
	}
	return r
}

// goSource is one Go file a check parsed: the file itself, the file set its
// positions resolve against, and the syntax.
type goSource struct {
	File
	FSet *token.FileSet
	AST  *ast.File
}

// parseGoSources parses the files one check will read and returns the ones that
// parsed. A file that does not parse is left out, not skipped in place: it is
// not among the files that check inspected, and checkSource carries the finding
// that names it.
func parseGoSources(files Tree, mode parser.Mode) []goSource {
	out := make([]goSource, 0, len(files))
	for _, f := range files {
		fset, file, err := parseGoSource(f, mode)
		if err != nil {
			continue
		}
		out = append(out, goSource{File: f, FSet: fset, AST: file})
	}
	return out
}

// parseGoSource parses one file as data. mode belongs to the caller: a check
// that only needs the imports parses with parser.ImportsOnly.
func parseGoSource(f File, mode parser.Mode) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, f.Path, f.Src, mode)
	if err != nil {
		return fset, nil, err
	}
	return fset, file, nil
}

// parses is the source check's question about one file: does it parse at all?
func parses(f File) error {
	_, _, err := parseGoSource(f, 0)
	return err
}

// parseFinding is the finding for a file the audit cannot parse: the file, the
// line the parser stopped at, and the parser's message. The message is the
// parser's own — it names tokens, not text, so a string literal's contents never
// enter the report — and the path it repeats is dropped, because the finding
// already carries it.
func parseFinding(check, path string, err error) Finding {
	line, msg := parseFailure(err)
	return fileFinding(check, path, "cannot parse: "+msg, line)
}

// parseFailure is the first failure the parser reported: its line and its
// message. A file can fail in more than one place; the first failure is the one
// that made the rest of the file unreadable.
func parseFailure(err error) (int, string) {
	var list scanner.ErrorList
	if errors.As(err, &list) && len(list) > 0 {
		return list[0].Pos.Line, oneLine(list[0].Msg)
	}
	var one *scanner.Error
	if errors.As(err, &one) {
		return one.Pos.Line, oneLine(one.Msg)
	}
	return 0, oneLine(err.Error())
}

// oneLine keeps a message one line long, so a finding is one line in the report.
func oneLine(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
}
