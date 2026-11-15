package core

import (
	"strings"
	"testing"
)

func TestExportMarkdown(t *testing.T) {
	items := []Item{
		&User{ID: "u1", Text: "路由匹配对大小写敏感"},
		&Thinking{ID: "t1", Text: "不应出现"},
		&ToolCall{ID: "c1", Name: "Edit", Title: "编辑 src/router.rs"},
		&Assistant{ID: "a1", Text: "  问题出在路由表。 "},
		&Notice{ID: "n1", Text: "不应出现"},
		&Changes{ID: "ch", Files: []ChangedFile{{Path: "src/router.rs", Added: 2, Removed: 1}}},
	}
	got, n := ExportMarkdown("修复", items)
	if n != 4 {
		t.Fatalf("条目数:%d\n%s", n, got)
	}
	for _, want := range []string{"## 修复", "**你**\n路由匹配对大小写敏感", "> 编辑 src/router.rs", "**Claude**\n问题出在路由表。", "> 本轮改动:src/router.rs(+2 −1)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("缺少 %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "不应出现") {
		t.Fatal("思考与提示不应导出")
	}
}
