package app

import (
	"os"
	"path/filepath"
	"testing"
)

// 演示工作区:web-dashboard 下有子模块 packages/ui 与嵌套仓库 vendor-kit/tool,
// 总览里缩进显示并带类别标签,文件按「离它最近的 .git」归属。
func TestNestedReposInWorkspace(t *testing.T) {
	a, tt := newTestApp(t)
	a.ActivateWorkspace("ws-fullstack")
	tt.Frame()
	var ui, tool int = -1, -1
	for i, e := range a.repos {
		switch e.name {
		case "packages/ui":
			ui = i
		case "vendor-kit/tool":
			tool = i
		}
	}
	if ui < 0 || tool < 0 {
		t.Fatalf("应发现两个嵌套仓库:%v", len(a.repos))
	}
	if a.repos[ui].kind != "子模块" || a.repos[tool].kind != "嵌套" {
		t.Fatalf("类别不对:%q %q", a.repos[ui].kind, a.repos[tool].kind)
	}
	// 归属:嵌套仓库里的文件属于嵌套仓库,其余属于外层文件夹
	i, rest, ok := a.repoIndexOf("web-dashboard/packages/ui/src/Button.tsx")
	if !ok || i != ui || rest != "src/Button.tsx" {
		t.Fatalf("归属错误:%d %q %v", i, rest, ok)
	}
	if i, rest, _ := a.repoIndexOf("web-dashboard/src/app.ts"); a.repos[i].name != "web-dashboard" || rest != "src/app.ts" {
		t.Fatalf("外层文件归属错误:%d %q", i, rest)
	}
	// 活动仓库切到子模块后,仓库内相对路径 → 树路径带完整前缀
	a.setActiveRepo(ui)
	if got := a.treePath("src/Button.tsx"); got != "web-dashboard/packages/ui/src/Button.tsx" {
		t.Fatalf("树路径:%q", got)
	}
	if a.ide.git["web-dashboard/packages/ui/src/Button.tsx"] != "M" {
		t.Fatalf("树上应标出子模块里的改动:%v", a.ide.git)
	}
	// 总览顺序:嵌套紧跟父文件夹
	ord := a.repoOrder()
	pos := map[int]int{}
	for k, v := range ord {
		pos[v] = k
	}
	if pos[ui] != pos[1]+1 {
		t.Fatalf("子模块应紧跟 web-dashboard:%v", ord)
	}
	a.nav = "scm"
	tt.Frame()
	if !tt.HasText("子模块") || !tt.HasText("嵌套") || !tt.HasText("packages/ui") {
		t.Fatal("总览应显示类别标签与嵌套仓库名")
	}
	// 审查分组:嵌套里的绝对路径归嵌套仓库
	if got := a.repoIdxOfFile("/demo/web-dashboard/packages/ui/src/Button.tsx"); got != ui {
		t.Fatalf("审查分组应归子模块:%d", got)
	}
}

// 真实目录里的发现:2 层内、跳过 node_modules / 隐藏目录,.gitmodules 里的标「子模块」。
func TestFindNested(t *testing.T) {
	dir := t.TempDir()
	mk := func(p string) {
		if err := os.MkdirAll(filepath.Join(dir, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mk("packages/ui/.git")
	mk("vendor-kit/tool/.git")
	mk("node_modules/x/.git")
	mk(".hidden/y/.git")
	mk("a/b/c/deep/.git") // 太深,不找
	os.WriteFile(filepath.Join(dir, ".gitmodules"), []byte("[submodule \"ui\"]\n\tpath = packages/ui\n\turl = x\n"), 0o644)
	got := findNested(dir)
	if len(got) != 2 || got[0].rel != "packages/ui" || got[0].kind != "子模块" || got[1].rel != "vendor-kit/tool" || got[1].kind != "嵌套" {
		t.Fatalf("发现结果不对:%v", got)
	}
}
