package demo

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID: "side-run", Project: "status-lab", Title: "演示 · 运行中(后台保活)",
		Label: "刚刚", Live: true,
		Build: buildSideRun,
	})
	Register(Scenario{
		ID: "side-wait", Project: "status-lab", Title: "演示 · 等待权限",
		Label: "2 分钟", Live: true, Auto: "请运行 cargo test 并停在权限卡",
		Build: buildSideWait,
	})
	Register(Scenario{
		ID: "side-err", Project: "status-lab", Title: "演示 · 刚出错",
		Label: "8 分钟",
		Build: buildSideErr,
	})
}

func buildSideRun() []Step {
	s := NewScript("side-run", "运行中")
	_ = s.User("在后台持续编译,不要结束回合。")
	live := core.NewToolCall("live-build", "Bash", "cargo build").WithInput(map[string]any{"command": "cargo build"})
	s.Add(0, core.EvAppend{Item: live})
	s.Add(2*time.Second, core.EvToolUpdate{ID: "live-build", Patch: core.ToolPatch{
		LiveOutput: core.Str("   Compiling router-demo v0.3.1\n    Building [==============>     ] 88/120\n"),
	}})
	s.Say("编译仍在进行,Live 剧本保持回合运行中。")
	return s.Steps()
}

func buildSideWait() []Step {
	s := NewScript("side-wait", "等待权限")
	_ = s.User("运行测试并请求权限。")
	return append(s.Steps(), PendingPermission()...)
}

func buildSideErr() []Step {
	s := NewScript("side-err", "刚出错")
	u := s.User("运行测试。")
	failOut := "test result: FAILED. 2 passed; 1 failed\n"
	s.Tool(core.NewToolCall("", "Bash", "cargo test").WithInput(map[string]any{"command": "cargo test"}),
		core.ToolFailed, 1800, &core.TerminalBlock{Command: "cargo test", Output: failOut, ExitCode: core.Int(1)})
	s.Say("测试失败:请检查断言。")
	s.End(u, 8, core.StopError, DefaultStats(3))
	return s.Steps()
}
