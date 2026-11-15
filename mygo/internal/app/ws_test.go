package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
)

var coreTerm = core.TerminalBlock{Command: "cargo test", Output: "ok\n"}

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

// 对话栏三档按自己的宽度判断,不看窗口。
func TestChatTier(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	for _, c := range []struct {
		w    float32
		want int
	}{{352, tierNarrow}, {419, tierNarrow}, {420, tierMid}, {560, tierMid}, {640, tierWide}, {712, tierWide}} {
		a.chatW = c.w
		if got := a.chatTier(); got != c.want {
			t.Errorf("宽 %v:档位 %d,想要 %d", c.w, got, c.want)
		}
	}
}

// 会话流里的入口都落到工作区的虚拟标签。
func TestWorkspaceEntries(t *testing.T) {
	a, tt := quickApp(t)
	tt.Frame()
	a.OpenReview()
	if a.ide.active != virtID(vReview, "") || !a.rightOn {
		t.Fatalf("审查入口:active=%q rightOn=%v", a.ide.active, a.rightOn)
	}
	a.OpenDiffTab("src/router.rs")
	if a.ide.active != "diff:src/router.rs" {
		t.Fatalf("差异入口:active=%q", a.ide.active)
	}
	// 命令卡 → 终端面板里的只读标签
	for _, it := range a.st.Items {
		_ = it
	}
	n0 := len(a.ide.cmds)
	a.viewInPanel(&coreTerm)
	if a.ide.termSel == "" || !a.ide.bottomOpen || len(a.ide.cmds) != n0+1 {
		t.Fatalf("命令输出入口:sel=%q open=%v cmds=%d", a.ide.termSel, a.ide.bottomOpen, len(a.ide.cmds))
	}
	a.viewInPanel(&coreTerm)
	if len(a.ide.cmds) != n0+1 {
		t.Fatalf("同一条命令只开一个标签")
	}
}
