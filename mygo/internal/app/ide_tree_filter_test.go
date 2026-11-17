package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

func newFilterApp() *App {
	a := New(Options{Mode: ModeIDE})
	a.ide.fs = workspace.NewMem("demo", map[string]string{
		"src/router.rs":         "a",
		"src/handlers/users.rs": "b",
		"src/handlers/mod.rs":   "c",
		"docs/guide.md":         "d",
		"README.md":             "e",
	})
	a.ide.listCache, a.ide.filesCache = nil, nil
	return a
}

func TestTreeFilterCompute(t *testing.T) {
	a := newFilterApp()
	d := a.ide
	d.fltOpen, d.flt = true, "users"
	d.fltCompute()
	if d.fltN != 1 || !d.fltVis["src/handlers/users.rs"] || !d.fltVis["src/handlers"] || !d.fltVis["src"] {
		t.Fatalf("应只命中 users.rs 并带出祖先目录: n=%d vis=%v", d.fltN, d.fltVis)
	}
	if d.fltVis["README.md"] || d.fltVis["src/router.rs"] {
		t.Fatalf("不命中的文件不该显示: %v", d.fltVis)
	}
	// 文件夹名命中 = 整个文件夹都留着
	d.flt = "handlers"
	d.fltCompute()
	if d.fltN != 2 || !d.fltVis["src/handlers/mod.rs"] {
		t.Fatalf("目录名命中应留下其下全部文件: %v", d.fltVis)
	}
	// 不区分大小写;带斜杠按整条路径
	d.flt = "READ"
	d.fltCompute()
	if d.fltN != 1 {
		t.Fatalf("应不区分大小写: %d", d.fltN)
	}
	d.flt = "src/r"
	d.fltCompute()
	if d.fltN != 1 || !d.fltVis["src/router.rs"] {
		t.Fatalf("带斜杠按路径匹配: %v", d.fltVis)
	}
}

func TestTreeFilterRendersAndEmpty(t *testing.T) {
	a := newFilterApp()
	d := a.ide
	tt := ui.NewTester(func(c *ui.Context) { a.filesView(c) }, 300, 400)
	tt.Frame()
	if !tt.HasText("README.md") {
		t.Fatalf("未过滤时应显示全部: %v", tt.Texts())
	}
	a.toggleTreeFilter()
	d.flt = "users"
	tt.Frame()
	tt.Frame()
	// 命中的文字会被拆成「命中段(高亮)+ 其余」几段文本,所以按片段断言
	if !tt.HasText("users") || !tt.HasText(".rs") || tt.HasText("README.md") || tt.HasText("router.rs") {
		t.Fatalf("过滤后只应留下命中项: %v", tt.Texts())
	}
	if !tt.HasText("1 项") {
		t.Fatalf("应显示命中数: %v", tt.Texts())
	}
	d.flt = "zzzz-none"
	tt.Frame()
	tt.Frame()
	if !tt.HasText("没有名称含「zzzz-none」的文件") {
		t.Fatalf("无命中应显示空态: %v", tt.Texts())
	}
	tt.Click("清空过滤")
	tt.Frame()
	tt.Frame()
	if d.flt != "" || !tt.HasText("README.md") {
		t.Fatalf("清空后应恢复完整的树: flt=%q %v", d.flt, tt.Texts())
	}
	a.toggleTreeFilter() // 收起
	if d.fltOpen || d.flt != "" {
		t.Fatal("收起应清空关键字")
	}
}
