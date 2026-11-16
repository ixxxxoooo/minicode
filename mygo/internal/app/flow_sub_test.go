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
