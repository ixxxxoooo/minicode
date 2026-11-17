package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 设计稿 flow-detail「什么时候成一组」:已处理的权限请求里装的也是那次工具调用,不能把相邻的一串从中间截断;
// 待处理的权限请求仍然是一张卡(断开成组)。
func TestGroupingUnwrapsResolvedPermission(t *testing.T) {
	edit := core.NewToolCall("e", "Edit", "编辑").WithInput(map[string]any{"file_path": "a.go"})
	read := core.NewToolCall("r", "Read", "读取").WithInput(map[string]any{"file_path": "a.go"})
	done := &core.PermissionRequest{ID: "p1", Tool: edit, Resolved: "allow"}
	pending := &core.PermissionRequest{ID: "p2", Tool: edit}
	if tc, ok := mergeable(done); !ok || tc != edit {
		t.Fatal("已处理的权限请求应并入精简摘要")
	}
	if _, ok := mergeable(pending); ok {
		t.Fatal("待处理的权限请求不能并入(它是一张要你决定的卡)")
	}
	if tc, ok := isExplore(&core.PermissionRequest{ID: "p3", Tool: read, Resolved: "allow"}); !ok || tc != read {
		t.Fatal("已处理的权限请求里的读取应并入探索组")
	}
	if _, ok := isExplore(done); ok {
		t.Fatal("编辑不属于探索组")
	}
}

// 设计稿 flow 页「工具令牌」:后台 Bash / Monitor / 联网搜索 / Glob 各有动词与图标,失败给退出码,被拒绝给「已拒绝」。
func TestToolTokens(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	done := func(tc *core.ToolCall) *core.ToolCall { tc.Status = core.ToolCompleted; return tc }

	bg := done(core.NewToolCall("b", "Bash", "cargo watch").WithInput(map[string]any{"command": "cargo watch -x test", "run_in_background": true}))
	if s := a.presentTool(bg); s.verb != "后台运行" || s.icon != "activity" || s.right != "已启动" {
		t.Errorf("后台 Bash:%+v", s)
	}
	mon := done(core.NewToolCall("m", "Monitor", "监听 target/dev.log").WithInput(map[string]any{"command": "tail -f target/dev.log"}))
	if s := a.presentTool(mon); s.verb != "监听" || s.icon != "eye" || s.obj != "tail -f target/dev.log" {
		t.Errorf("Monitor:%+v", s)
	}
	ws := done(core.NewToolCall("w", "WebSearch", "搜索网页").WithInput(map[string]any{"query": "axum case insensitive route"}))
	ws.Content = []core.ToolContent{&core.TextBlock{Text: `Links: [{"title":"a","url":"https://a"},{"title":"b","url":"https://b"}]`}}
	if s := a.presentTool(ws); s.verb != "联网搜索" || s.obj != "axum case insensitive route" || s.right != "2 条结果" || !s.plainObj {
		t.Errorf("WebSearch:%+v", s)
	}
	glob := done(core.NewToolCall("g", "Glob", "查找").WithInput(map[string]any{"pattern": "**/*_test.rs"}))
	glob.Content = []core.ToolContent{&core.TextBlock{Text: "a_test.rs\nb_test.rs\nc_test.rs\n"}}
	if s := a.presentTool(glob); s.right != "3 个文件" {
		t.Errorf("Glob 右侧应是文件数:%+v", s)
	}

	fail := core.NewToolCall("f", "Bash", "cargo clippy").WithInput(map[string]any{"command": "cargo clippy"})
	fail.Status = core.ToolFailed
	fail.Content = []core.ToolContent{&core.TerminalBlock{Command: "cargo clippy", Output: "error", ExitCode: core.Int(101)}}
	if s := a.presentTool(fail); s.right != "退出码 101" || s.icon != "errc" {
		t.Errorf("失败的命令应显示退出码:%+v", s)
	}
	deny := core.NewToolCall("d", "Bash", "rm -rf target").WithInput(map[string]any{"command": "rm -rf target"})
	deny.Status = core.ToolDenied
	if s := a.presentTool(deny); s.right != "已拒绝" || s.icon != "lock" || s.state != "deny" {
		t.Errorf("被拒绝的调用:%+v", s)
	}
}

// 设计稿 flow-detail:运行中的命令展开实时输出,结束后不自动收起(记进手动展开状态);用户运行期间自己收起过的不覆盖。
func TestRunningCommandStaysOpenAfterFinish(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetDetailLevel(1)
	run := core.NewToolCall("r1", "Bash", "cargo test").WithInput(map[string]any{"command": "cargo test"})
	run.Content = []core.ToolContent{&core.TerminalBlock{Command: "cargo test", Output: "running\n"}}
	if !a.presentTool(run).open {
		t.Fatal("运行中的命令默认展开")
	}
	run.Status = core.ToolCompleted
	run.Content = []core.ToolContent{&core.TerminalBlock{Command: "cargo test", Output: "ok\n", ExitCode: core.Int(0)}}
	if open, set := a.folds["r1"]; !set || !open {
		t.Fatal("运行中展开过的命令,结束后应保持展开(folds 里记着)")
	}

	a.folds["r2"] = false // 用户在运行期间主动收起
	run2 := core.NewToolCall("r2", "Bash", "sleep 9").WithInput(map[string]any{"command": "sleep 9"})
	run2.Content = []core.ToolContent{&core.TerminalBlock{Command: "sleep 9"}}
	a.presentTool(run2)
	if a.folds["r2"] {
		t.Fatal("用户自己收起的不应被改回展开")
	}
}
