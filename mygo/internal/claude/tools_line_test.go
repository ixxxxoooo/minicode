package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 编辑块的起始行要按文件内容定位(真实后端此前一律是 0,审查页 / 改动卡 / 编辑器叠层行号全错成 1)。
func TestBuildToolEditLine(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.go")
	src := "package a\n\n// 注释\nfunc f() {\n\tx := 1\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	line := func(tc *core.ToolCall) int {
		for _, c := range tc.Content {
			if d, ok := c.(*core.DiffBlock); ok {
				return d.Line
			}
		}
		return -1
	}
	// 实时:文件里还是 old
	tc := BuildTool("1", "Edit", obj{"file_path": p, "old_string": "\tx := 1", "new_string": "\tx := 2"}, dir)
	if got := line(tc); got != 5 {
		t.Errorf("实时编辑行号应为 5,得 %d", got)
	}
	// 回放:文件已经是 new
	tc = BuildTool("2", "Edit", obj{"file_path": p, "old_string": "\tx := 0", "new_string": "\tx := 1"}, dir)
	if got := line(tc); got != 5 {
		t.Errorf("回放应在现文件里找 new,得 %d", got)
	}
	// 相对路径按 cwd 解析;找不到 / 文件不存在 → 0
	tc = BuildTool("3", "Edit", obj{"file_path": "a.go", "old_string": "func f()", "new_string": "func g()"}, dir)
	if got := line(tc); got != 4 {
		t.Errorf("相对路径行号应为 4,得 %d", got)
	}
	tc = BuildTool("4", "Edit", obj{"file_path": filepath.Join(dir, "none.go"), "old_string": "a", "new_string": "b"}, dir)
	if got := line(tc); got != 0 {
		t.Errorf("文件不存在应为 0,得 %d", got)
	}
}
