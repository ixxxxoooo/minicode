package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
)

// 回归:MCP 工具返回的一整行长 JSON / 无空格长串,结果卡不能撑出一大片空白
// (Markdown 的双向滚动按「折行后」量高、实际却不折行,卡被撑到 480 上限)。
func TestToolResultLongLineNoBlank(t *testing.T) {
	a, _ := newTestApp(t)
	oneLineJSON := `{"field_descriptions":{"query_columns":"结果列名列表","query_results":"建表语句结果行"},"query_columns":["Table","Create Table"],"query_results":[["tblBwDataSet","CREATE TABLE ` +
		strings.Repeat("`col_x` varchar(64) NOT NULL DEFAULT '' COMMENT '字段说明',", 80) + `"]]}`

	endY := func(name, text string) float32 {
		tc := core.NewToolCall("id-"+name, "mcp__journey__db_schema", "调用").WithContent(&core.TextBlock{Text: text})
		tc.Status = core.ToolCompleted
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Width(700).Children(func() {
				a.toolResult(c, tc)
				ui.Text(c, "END-MARK")
			})
		}, 900, 900)
		tt.Frame()
		tt.Frame()
		r, ok := tt.Find("END-MARK")
		if !ok {
			t.Fatalf("%s:没找到结尾标记", name)
		}
		return r.Y
	}

	for name, text := range map[string]string{
		"一整行 JSON":   oneLineJSON,
		"无空格长 ASCII": strings.Repeat("abcdefghij,", 300),
	} {
		y := endY(name, text)
		t.Logf("%s:卡下沿 y=%v", name, y)
		if y >= foldExpandMax {
			t.Errorf("%s:结果卡被撑到上限(y=%v),下面是一大片空白", name, y)
		}
	}
	if y := endY("普通文本", "hello world"); y > 80 {
		t.Errorf("普通短文本不该很高:y=%v", y)
	}
}

func TestMonoResultText(t *testing.T) {
	if s, ok := monoResultText(`{"a":1,"b":[1,2]}`); !ok || !strings.Contains(s, "\n  \"a\": 1") {
		t.Fatalf("JSON 应缩进成多行:%v %q", ok, s)
	}
	if _, ok := monoResultText("# 标题\n\n正常的 markdown 段落,有空格 and words."); ok {
		t.Fatal("普通 Markdown 不该走等宽卡")
	}
	if _, ok := monoResultText(strings.Repeat("x", 200)); !ok {
		t.Fatal("超长无空白行应走等宽卡")
	}
	if _, ok := monoResultText("{not json"); ok {
		t.Fatal("不是合法 JSON、也没有超长行,不该走等宽卡")
	}
}
