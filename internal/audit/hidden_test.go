package audit

import (
	"strings"
	"testing"
)

// hiddenTree is the two files the check reads, as a synthetic tree: the state's vocabulary and the
// composition root. It is small on purpose — the check's subject is two names, not a package.
func hiddenTree(model, feed string) Tree {
	return Tree{
		{Path: "internal/tui/model.go", Src: []byte(model)},
		{Path: "internal/cli/feed.go", Src: []byte(feed)},
	}
}

const clearModel = `package tui

func StateOf(name string) (State, bool) {
	switch name {
	case "normal":
		return StateNormal, true
	}
	return StateNormal, false
}
`

const clearFeed = `package cli

func display() tui.Display {
	return tui.Display{State: tui.StateNormal}
}
`

// TestHiddenModePassesOnAClearTree is the control: the check must not report the shape the shipped
// tree has.
func TestHiddenModePassesOnAClearTree(t *testing.T) {
	r := checkHiddenMode(hiddenTree(clearModel, clearFeed))
	if len(r.Findings) != 0 {
		t.Fatalf("a clear tree failed: %+v", r.Findings)
	}
	if r.Files != 2 {
		t.Fatalf("inspected %d files, want 2", r.Files)
	}
}

// TestHiddenModeFindsTheConfigName is the synthetic violation: a case for the name in StateOf is the
// config file becoming a way in.
func TestHiddenModeFindsTheConfigName(t *testing.T) {
	model := strings.Replace(clearModel, `case "normal":`, `case "vivid":
		return StateVivid, true
	case "normal":`, 1)
	r := checkHiddenMode(hiddenTree(model, clearFeed))
	if !findingsSay(r.Findings, "way in") {
		t.Fatalf("a config name passed: %+v", r.Findings)
	}
}

// TestHiddenModeFindsAStartupPath is the second arm: the composition root naming the state directly
// is a start-up path that does not read the file.
func TestHiddenModeFindsAStartupPath(t *testing.T) {
	feed := strings.Replace(clearFeed, "tui.StateNormal", "tui.StateVivid", 1)
	r := checkHiddenMode(hiddenTree(clearModel, feed))
	if !findingsSay(r.Findings, "composition root") {
		t.Fatalf("a start-up path passed: %+v", r.Findings)
	}
}

// TestHiddenModeFailsAnEmptyScan: a scan that read nothing proves nothing.
func TestHiddenModeFailsAnEmptyScan(t *testing.T) {
	r := checkHiddenMode(Tree{{Path: "README.md", Src: []byte("nothing")}})
	if !findingsSay(r.Findings, "nothing") {
		t.Fatalf("an empty scan passed: %+v", r.Findings)
	}
}
