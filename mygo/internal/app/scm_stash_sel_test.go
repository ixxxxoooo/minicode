package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

func TestStashDiffActiveID(t *testing.T) {
	a := New(Options{Mode: ModeIDE, Demo: true})
	a.nav = "scm"
	a.ide.openFile("src/router.rs", true)
	a.scmTick()
	if len(a.sc.stashes) == 0 {
		t.Fatal("演示应有储藏")
	}
	a.sc.stashOpen = map[int]bool{0: true}
	a.sc.stashFiles = map[int][]vcs.Change{
		0: {{Path: "src/router.rs", Status: 'M'}},
	}
	tp := a.treePath("src/router.rs")
	want := "diff:stash-0:" + tp
	a.ide.active = want
	if a.ide.active != want {
		t.Fatalf("储藏 diff id:%q", a.ide.active)
	}
}
