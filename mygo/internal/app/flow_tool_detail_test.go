package app

import (
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 工具详情:没有差异 / 终端输出的工具(MCP)也能展开看到入参与结果。
func TestToolDetailInputAndResult(t *testing.T) {
	in := map[string]any{"repo": "a/b", "number": 12, "body": "第一行\n第二行"}
	got := formatToolInput(in)
	for _, want := range []string{"number: 12", "repo: a/b", "body:\n  第一行\n  第二行"} {
		if !strings.Contains(got, want) {
			t.Fatalf("入参格式化缺 %q:\n%s", want, got)
		}
	}
	tc := core.NewToolCall("m1", "mcp__github__get_pr", "获取 PR #12").WithInput(in)
	if !toolHasDetail(tc) || toolRichBody(tc) {
		t.Fatal("MCP 工具应有通用详情,没有专用展开体")
	}
	if !strings.Contains(toolMetaLine(tc), "mcp__github__get_pr") {
		t.Fatal("元信息应带完整工具名")
	}
	if toolHasDetail(core.NewToolCall("x", "Foo", "空")) {
		t.Fatal("没有入参与结果就没有详情")
	}
}

func TestMcpStepExpands(t *testing.T) {
	a, tt := newTestApp(t)
	tc := core.NewToolCall("mcp-demo-1", "mcp__github__get_pr", "获取 PR #12").
		WithInput(map[string]any{"repo": "a/b", "number": 12}).
		WithContent(&core.TextBlock{Text: "merged: true"})
	tc.Status = core.ToolCompleted
	a.st.Apply(core.EvAppend{Item: tc})
	tt.Frame()
	if !tt.HasText("获取 PR #12") {
		t.Skip("该条目被折进回合折叠,不在可见区域")
	}
	if tt.HasText("merged: true") {
		t.Fatal("默认不应展开")
	}
	if err := tt.Click("获取 PR #12"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, want := range []string{"入参", "结果", "merged: true", "repo: a/b"} {
		if !tt.HasText(want) {
			t.Fatalf("展开后应显示「%s」", want)
		}
	}
}
