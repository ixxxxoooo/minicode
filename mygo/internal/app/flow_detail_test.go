package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

// 设计稿 flow-detail「默认展开矩阵」:标准档编辑 / 命令默认折叠(失败、被拒绝、运行中、非 0 退出码例外),
// 精简档一律折叠,详细档默认展开。
func TestToolDefaultOpenMatrix(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	edit := func(st core.ToolStatus) *core.ToolCall {
		tc := core.NewToolCall("e", "Edit", "编辑 router.rs").WithInput(map[string]any{"file_path": "src/router.rs"})
		tc.Status = st
		tc.Content = []core.ToolContent{&core.DiffBlock{Path: "src/router.rs", Old: "a\n", New: "b\n"}}
		return tc
	}
	run := func(st core.ToolStatus, exit *int) *core.ToolCall {
		tc := core.NewToolCall("r", "Bash", "cargo test").WithInput(map[string]any{"command": "cargo test"})
		tc.Status = st
		tc.Content = []core.ToolContent{&core.TerminalBlock{Command: "cargo test", Output: "ok\n", ExitCode: exit}}
		return tc
	}
	cases := []struct {
		name  string
		level int
		tc    *core.ToolCall
		open  bool
	}{
		{"精简·编辑", 0, edit(core.ToolCompleted), false},
		{"标准·编辑完成", 1, edit(core.ToolCompleted), false},
		{"标准·编辑失败", 1, edit(core.ToolFailed), true},
		{"标准·命令完成", 1, run(core.ToolCompleted, core.Int(0)), false},
		{"标准·命令运行中", 1, run(core.ToolRunning, nil), true},
		{"标准·命令非0退出码", 1, run(core.ToolCompleted, core.Int(101)), true},
		{"详细·编辑完成", 2, edit(core.ToolCompleted), true},
		{"详细·命令完成", 2, run(core.ToolCompleted, core.Int(0)), true},
	}
	for _, c := range cases {
		a.SetDetailLevel(c.level)
		if got := a.presentTool(c.tc).open; got != c.open {
			t.Errorf("%s:默认展开应为 %v,实际 %v", c.name, c.open, got)
		}
	}
}

// 精简档摘要:固定顺序、量词固定;窄档短句。
func TestMergeSummaryOrderAndNarrow(t *testing.T) {
	mk := func(name string) *core.ToolCall { return core.NewToolCall(name, name, name) }
	g := []*core.ToolCall{mk("Bash"), mk("Edit"), mk("Read"), mk("Read"), mk("Grep"), mk("Edit")}
	sum, _, failed := mergeSummary(g)
	if want := "读取了 2 个文件、搜索了 1 次、编辑了 2 个文件、运行了 1 条命令"; sum != want {
		t.Fatalf("摘要:想要 %q,实际 %q", want, sum)
	}
	if failed != 0 {
		t.Fatalf("没有失败的调用不应计失败:%d", failed)
	}
	if want := "读 2 · 搜 1 · 改 2 · 跑 1"; mergeSummaryNarrow(g) != want {
		t.Fatalf("窄档短句:想要 %q,实际 %q", want, mergeSummaryNarrow(g))
	}
	g[1].Status = core.ToolFailed
	if _, _, f := mergeSummary(g); f != 1 {
		t.Fatalf("应计 1 项失败,实际 %d", f)
	}
}

// 思考标题:第一句 ≤28 字;流式尚无正文写「思考中」。
func TestThinkTitle(t *testing.T) {
	if thinkTitle("") != "思考中" {
		t.Fatal("空正文应写「思考中」")
	}
	if got := thinkTitle("先看路由表。再看测试"); got != "先看路由表" {
		t.Fatalf("应取第一句,实际 %q", got)
	}
	long := "The user wants me to check whether the app router handles trailing slashes correctly"
	if r := []rune(thinkTitle(long)); len(r) > 29 {
		t.Fatalf("标题最多 28 字加省略号,实际 %d", len(r))
	}
}

// 回归:全新 App 第一次渲染流式思考行不能因跑马灯状态表未初始化而 panic(开发版启动即崩)。
func TestStreamingThinkingRendersOnFreshApp(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	a.State().Apply(core.EvAppend{Item: &core.Thinking{ID: "th-live", Text: "先看路由表。再看测试", Streaming: true}})
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	a.State().Apply(core.EvAppend{Item: &core.Thinking{ID: "th-done", Text: "已经想完了", Streaming: false}})
	tt.Frame()
}
