package app

import (
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

// 回归:回合折叠(「已处理」)与探索组里的嵌套容器(`nest`)没有 MinWidth(0),
// 超长的命令 / 输出行会把组内所有卡片撑出会话流内容列,一路铺到窗口右缘。
// 这里造一个已结束的回合,展开「已处理」,检查命令卡头部的「通过」仍落在内容列内。
func TestFoldedTurnLongLinesStayInColumn(t *testing.T) {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	long := strings.Repeat("/Users/demo/MiniCode/mygo/internal/app/very_long_name_", 40)
	cmd := "git diff -U1 -- " + long + " | head -30"
	out := "- " + long + "\n+ " + long + "\n"
	bash := core.NewToolCall("b1", "Bash", cmd).WithInput(map[string]any{"command": cmd})
	steps := []demo.Step{
		{After: 0, Ev: core.EvReady{SessionID: "t-overflow", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault}},
		{After: 0, Ev: core.EvAppend{Item: &core.User{ID: "u1", Text: "看一下改动"}}},
		{After: ms(1), Ev: core.EvTurnStarted{}},
		{After: ms(1), Ev: core.EvAppend{Item: bash}},
		{After: ms(1), Ev: core.EvToolUpdate{ID: "b1", Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 100,
			AppendContent: []core.ToolContent{&core.TerminalBlock{Command: cmd, Output: out, ExitCode: core.Int(0)}}}}},
		{After: ms(1), Ev: core.EvAppend{Item: &core.Assistant{ID: "a1", Text: "看完了。"}}},
		{After: 0, Ev: core.EvTurnEnded{Reason: core.StopEndTurn}},
	}
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	a.Attach(&fakeBE{})
	demo.Replay(a.st, steps)
	a.folds["turnfold:u1"] = true // 展开「已处理」
	tt := ui.NewTester(a.View, 2048, 1100)
	tt.Frame()
	tt.Frame()

	r, ok := tt.Find("通过")
	if !ok {
		t.Fatalf("展开「已处理」后命令卡头部应显示「通过」;现有文本:%q", tt.Texts())
	}
	// 内容列最宽约 760;左缘之后留足余量,仍远小于窗口宽 2048
	if r.X+r.W > 1500 {
		t.Fatalf("折叠组里的命令卡被超长行撑出内容列:「通过」右缘 %.0f", r.X+r.W)
	}
}

// 命令卡头部的命令按宽度换行(而不是单行压到右侧按钮底下):文字高度应是多行,
// 且右侧「通过」仍在命令文字右边,不与文字重叠。
func TestTermCardCommandWraps(t *testing.T) {
	cmd := "git status --short && echo '--- diff stat' && git diff --stat && echo '--- ahead/behind' && git rev-list --left-right --count origin/feat/mygo...HEAD 2>&1 && " +
		"cd /Users/demo/MiniCode/mygo && unset GOROOT && gofmt -l . 2>&1 | head -20"
	bash := core.NewToolCall("b1", "Bash", "运行检查命令").WithInput(map[string]any{"command": cmd})
	steps := []demo.Step{
		{After: 0, Ev: core.EvReady{SessionID: "t-wrap", Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault}},
		{After: 0, Ev: core.EvAppend{Item: &core.User{ID: "u1", Text: "看一下"}}},
		{After: time.Millisecond, Ev: core.EvTurnStarted{}},
		{After: time.Millisecond, Ev: core.EvAppend{Item: bash}},
		{After: time.Millisecond, Ev: core.EvToolUpdate{ID: "b1", Patch: core.ToolPatch{Status: core.Status(core.ToolCompleted), DurationMs: 100,
			AppendContent: []core.ToolContent{&core.TerminalBlock{Command: cmd, Output: "ok\n", ExitCode: core.Int(0)}}}}},
	}
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	a.Attach(&fakeBE{})
	demo.Replay(a.st, steps)
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	tt.Frame()
	if os.Getenv("MINI_DUMP") != "" {
		f, _ := os.Create(os.Getenv("MINI_DUMP"))
		_ = png.Encode(f, tt.Image())
		f.Close()
	}
	pass, ok1 := tt.Find("通过")
	body, ok2 := tt.Find("ok")
	if !ok1 || !ok2 {
		t.Fatalf("应显示头部「通过」与输出「ok」;文本 %q", tt.Texts())
	}
	// 单行头部 24 高:「通过」到输出首行约 30px;命令换成多行后头部长高,间距明显更大
	if body.Y-pass.Y < 45 {
		t.Fatalf("命令没有换行:「通过」Y=%.0f,输出首行 Y=%.0f", pass.Y, body.Y)
	}
}
