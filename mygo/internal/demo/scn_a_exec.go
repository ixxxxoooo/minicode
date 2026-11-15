package demo

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func init() {
	Register(Scenario{
		ID: "ci", Title: "工具卡 · 命令执行:修复 CI", Project: "router-demo",
		Label: "昨天", Group: "昨天", Clock: "11:02",
		Build: buildScnCI,
	})
	Register(Scenario{
		ID: "tools-bash", Title: "工具卡 · 命令:长输出 / ANSI / 多样态", Project: "",
		Build: buildScnToolsBash,
	})
	Register(Scenario{
		ID: "tools-bash-live", Title: "工具卡 · 命令:实时输出进行中",
		Live:  true,
		Build: buildScnToolsBashLive,
	})
}

func buildScnCI() []Step {
	s := NewScript("ci", "工具卡 · 命令执行:修复 CI")
	u := s.User("CI 挂了,先跑集成测试看错误,修好后再跑一遍。")
	s.Think("失败 → 改测 → 通过,用两三段 Bash 讲完整故事。", 3)

	failOut := "running tests::integration::routes ... FAILED\nassertion `left == right` failed\n  left: \"/Users\"\n right: \"/users\"\ntest result: FAILED. 1 failed; 11 passed\n"
	s.Tool(core.NewToolCall("", "Bash", "运行 cargo test --test integration").WithInput(map[string]any{
		"command": "cargo test --test integration",
	}), core.ToolFailed, 2400, &core.TerminalBlock{
		Command: "cargo test --test integration", Output: failOut, ExitCode: core.Int(101),
	})

	fix := `#[test]
fn routes_case_insensitive() {
    assert_eq!(normalize("/Users"), "users");
}
`
	s.Tool(core.NewToolCall("", "Edit", "编辑 tests/integration/ci.rs").
		WithInput(map[string]any{"file_path": "tests/integration/ci.rs"}).WithLocation("tests/integration/ci.rs"),
		core.ToolCompleted, 520, &core.DiffBlock{Path: "tests/integration/ci.rs", Old: "", New: fix, Line: 1})

	okOut := "   Compiling router-demo v0.3.1\n     Running tests/integration/ci.rs\nrunning 12 tests ... ok\ntest result: ok. 12 passed; 0 failed in 0.58s\n"
	s.Tool(core.NewToolCall("", "Bash", "再次运行 cargo test --test integration").WithInput(map[string]any{
		"command": "cargo test --test integration",
	}), core.ToolCompleted, 3100, &core.TerminalBlock{
		Command: "cargo test --test integration", Output: okOut, ExitCode: core.Int(0),
	})

	s.Say("集成测试先因大小写失败,补上归一化断言后已全部通过。")
	s.End(u, 78, core.StopEndTurn, DefaultStats(8))
	return s.Steps()
}

func buildScnToolsBash() []Step {
	s := NewScript("tools-bash", "工具卡 · 命令:长输出 / ANSI / 多样态")
	u := s.User("演示 Bash 成功、失败、无输出、长输出、ANSI、多行命令、后台、超时与被中断。")

	s.Tool(core.NewToolCall("", "Bash", "运行 true").WithInput(map[string]any{"command": "true"}),
		core.ToolCompleted, 120, &core.TerminalBlock{Command: "true", Output: "", ExitCode: core.Int(0)})

	s.Tool(core.NewToolCall("", "Bash", "运行 false").WithInput(map[string]any{"command": "false"}),
		core.ToolFailed, 90, &core.TerminalBlock{Command: "false", Output: "command failed\n", ExitCode: core.Int(1)})

	ml := "cargo test router \\\n  && cargo clippy \\\n  && cargo fmt --check"
	s.Tool(core.NewToolCall("", "Bash", "多行命令").WithInput(map[string]any{"command": ml}),
		core.ToolCompleted, 1800, &core.TerminalBlock{Command: ml, Output: "all checks passed\n", ExitCode: core.Int(0)})

	s.Tool(core.NewToolCall("", "Bash", "长输出测试").WithInput(map[string]any{"command": "cargo test -- --nocapture"}),
		core.ToolCompleted, 4200, &core.TerminalBlock{Command: "cargo test -- --nocapture", Output: longBashOutput(), ExitCode: core.Int(0)})

	s.Tool(core.NewToolCall("", "Bash", "彩色编译输出").WithInput(map[string]any{"command": "cargo test"}),
		core.ToolCompleted, 900, &core.TerminalBlock{Command: "cargo test", Output: bashANSI, ExitCode: core.Int(0)})

	bg := core.NewToolCall("bg1", "Bash", "后台 watch 测试").WithInput(map[string]any{
		"command": "cargo watch -x test", "run_in_background": true,
	})
	s.Add(ms(100), core.EvAppend{Item: bg})
	now := time.Now().UnixMilli()
	s.Add(ms(400), core.EvToolUpdate{ID: "bg1", Patch: core.ToolPatch{
		Status: core.Status(core.ToolCompleted), TaskID: "demo-bg-watch", DurationMs: 200,
	}})
	s.Add(0, core.EvTaskStarted{Task: core.NewBackgroundTask(now-8000, "demo-bg-watch", core.TaskShell, "cargo watch -x test")})

	s.Tool(core.NewToolCall("", "Bash", "超时命令").WithInput(map[string]any{"command": "sleep 120"}),
		core.ToolFailed, 60_000, &core.TerminalBlock{Command: "sleep 120", Output: "Command timed out after 60s", ExitCode: core.Int(124)})

	run := core.NewToolCall("intr", "Bash", "长时间任务").WithInput(map[string]any{"command": "cargo build --workspace"})
	s.Add(ms(150), core.EvAppend{Item: run})
	s.Add(ms(300), core.EvToolUpdate{ID: "intr", Patch: core.ToolPatch{LiveOutput: core.Str("   Compiling serde v1.0\n")}})
	s.Say("下一条事件将取消回合,上面运行中的 Bash 应变为被中断。")
	s.End(u, 12, core.StopCancelled, DefaultStats(9))
	return s.Steps()
}

func buildScnToolsBashLive() []Step {
	s := NewScript("tools-bash-live", "工具卡 · 命令:实时输出进行中")
	u := s.User("启动开发服务器,让输出实时滚动。")
	s.Think("Live 场景故意不收尾,停在运行中与 LiveOutput。", 2)
	live := core.NewToolCall("live1", "Bash", "npm run dev").WithInput(map[string]any{"command": "npm run dev"})
	s.Add(ms(200), core.EvAppend{Item: live})
	s.Add(ms(800), core.EvToolUpdate{ID: "live1", Patch: core.ToolPatch{LiveOutput: core.Str("> dev\n> vite --port 5173\n\n  VITE v5 ready in 420 ms\n")}})
	s.Add(ms(1200), core.EvToolUpdate{ID: "live1", Patch: core.ToolPatch{LiveOutput: core.Str("> dev\n> vite --port 5173\n\n  VITE v5 ready in 420 ms\n  ➜  Local: http://localhost:5173/\n  ➜  watching for file changes...\n")}})
	s.Say("服务器已启动,命令仍在运行中(Live=true 剧本未结束回合)。")
	// 故意不 End,保持 PhaseRunning 与运行中 Bash
	_ = u
	return s.Steps()
}
