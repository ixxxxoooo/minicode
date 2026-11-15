package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// 拉取并合并遇到冲突 → 进入「正在合并」→ 中止合并后状态与冲突分组都清空;没有进行中的合并时不显示。
func TestMergeConflictAbortFlow(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	m := vcs.NewMem()
	a.repo = m
	if a.sc.merging || a.opWord() != "" {
		t.Fatal("初始不应在合并中")
	}
	m.SetMergeConflict("合并冲突")
	a.scmPullMerge()
	if !a.sc.merging || a.opWord() != "合并" {
		t.Fatalf("冲突后应处于合并中: merging=%v word=%q", a.sc.merging, a.opWord())
	}
	if !a.pushGuideHasConflict() {
		t.Fatal("合并中应显示冲突出口")
	}
	// 刷新按仓库真实状态落定(重启后也能接上)
	a.scmRefresh()
	if !a.sc.merging {
		t.Fatal("刷新后仍应处于合并中")
	}
	a.scmMergeAbort()
	if a.sc.merging || m.OpState() != "" {
		t.Fatal("中止后应退出合并")
	}
	if a.toast != "已中止合并" {
		t.Fatalf("toast: %q", a.toast)
	}
	a.scmRefresh() // 真实运行时由下一帧的 scmTick 重读
	for _, c := range a.sc.changes {
		if c.Status == vcs.StatusConflict {
			t.Fatal("冲突分组应清空")
		}
	}
}
