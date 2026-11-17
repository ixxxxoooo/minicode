package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// 储藏展开后文件行的 diff 标签 id 要能与活动标签对上(选中态据此高亮)。
func TestStashDiffActiveID(t *testing.T) {
	stressDemo(t)
	a, tt := demoIDE(t)
	a.OpenSCM()
	tt.Frame()
	tt.Frame()
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
