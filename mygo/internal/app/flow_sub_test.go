package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

// 主会话流里的子 Agent 只是精简调用卡:带「查看过程」入口,点了在工作区开子 Agent 标签。
func TestSubagentCompactCardRenders(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	demo.Replay(a.State(), demo.FixBug(false))
	card := core.NewSubagent("sub1", "code-reviewer", "审查 handlers", core.SubRunning)
	card.Prompt = "只读扫描 unwrap"
	a.State().Apply(core.EvAppend{Item: card})
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	if err := tt.Click("查看过程"); err != nil {
		t.Fatalf("调用卡应含「查看过程」:%v", err)
	}
	tt.Frame()
}

// 切会话后,绑定旧会话的子 Agent / 任务输出标签要关掉,不留「已不在当前会话里」的空壳。
func TestSessionSwitchClosesSubTabs(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	a.ide.openVirtual(vSub, "sub-x", true)
	a.ide.openVirtual(vPlan, "", true)
	a.OpenSession("")
	if a.ide.tab(virtID(vSub, "sub-x")) != nil {
		t.Fatal("子 Agent 标签应随会话切换关闭")
	}
	if a.ide.tab(virtID(vPlan, "")) == nil {
		t.Fatal("计划标签应保留")
	}
}
