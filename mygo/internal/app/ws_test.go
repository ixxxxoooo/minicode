package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// 区域模型:Agent 的工作区默认收起,开虚拟标签会展开它;切到 IDE 再切回,各形态的开合各自记住。
func TestWorkspaceRegionModel(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	if a.mode != ModeAgent || a.rightOn {
		t.Fatalf("Agent 默认工作区应收起:mode=%v rightOn=%v", a.mode, a.rightOn)
	}
	a.OpenWorkspaceTab(vPlan, "")
	tt.Frame()
	if !a.rightOn || a.ide.active != virtID(vPlan, "") {
		t.Fatalf("开计划标签应展开工作区并激活它:rightOn=%v active=%q", a.rightOn, a.ide.active)
	}
	// 同一个内容只有一个标签
	a.OpenWorkspaceTab(vPlan, "")
	if n := len(a.ide.tabs); n != 1 {
		t.Fatalf("重复打开不该多出标签:%d", n)
	}
	a.SetMode(ModeIDE)
	if !a.rightOn || !a.ide.bottomOpen {
		t.Fatalf("IDE 默认对话栏与底部面板展开:rightOn=%v bottom=%v", a.rightOn, a.ide.bottomOpen)
	}
	a.toggleRight() // IDE 里收起对话栏
	a.SetMode(ModeAgent)
	if !a.rightOn {
		t.Fatalf("回到 Agent 应恢复它自己记住的工作区开合")
	}
	a.closeWsTab(virtID(vPlan, ""))
	if a.rightOn {
		t.Fatalf("Agent 里最后一个标签关掉后工作区应收起")
	}
}

func TestReviewHunkRevert(t *testing.T) {
	a, tt := quickApp(t)
	tt.Frame()
	files := a.changedFiles()
	if len(files) == 0 || len(files[0].blocks) == 0 {
		t.Fatalf("演示会话里应有改动文件")
	}
	f := files[0]
	before, _ := a.ide.fs.Read(a.wsRel(f.path))
	a.revertHunk(f, 0)
	after, _ := a.ide.fs.Read(a.wsRel(f.path))
	if a.rv.hunk[hunkKey(f.path, 0)] != "rev" && a.toast == "" {
		t.Fatalf("撤销后应标记已撤销,或给出原因")
	}
	if a.rv.hunk[hunkKey(f.path, 0)] == "rev" && before == after {
		t.Fatalf("撤销应改动文件")
	}
}
