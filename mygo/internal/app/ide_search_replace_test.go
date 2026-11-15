package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/search"
)

func TestReplaceOneNoSelection(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	sel := -1
	a.sr.list.Selected = &sel
	a.replaceOne()
}

func TestReplaceOneNeedsMatchRow(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.sr.rows = []srow{{n: 1}}
	sel := 0
	a.sr.list.Selected = &sel
	a.replaceOne()
	a.sr.rows = []srow{{match: &search.Match{Path: "x", Line: 1, Col: 0}}}
	a.sr.list.Selected = &sel
	a.replaceOne()
}
