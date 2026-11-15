package app

import "testing"

// 图谱标签的仓库下拉:点开选另一个仓库 = 切换活动仓库,标签名带仓库名,选中的提交清空。
func TestGraphRepoDropdown(t *testing.T) {
	a, tt := newTestApp(t)
	a.ActivateWorkspace("ws-fullstack")
	a.openGraph()
	tt.Frame()
	if a.tabTitle(&ideTab{id: virtID(vGraph, ""), virt: vGraph}) != "图谱 · router-demo" {
		t.Fatalf("多仓库时标签名应带仓库名:%q", a.tabTitle(&ideTab{id: virtID(vGraph, ""), virt: vGraph}))
	}
	a.graphTab.sel = "abc"
	a.popup = "fp-graph-repo"
	tt.Frame()
	if err := tt.Click("feat/charts"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.repoAct != 1 || a.graphTab.sel != "" || a.branch != "feat/charts" {
		t.Fatalf("应切到 web-dashboard 并清空选中:act=%d sel=%q branch=%s", a.repoAct, a.graphTab.sel, a.branch)
	}
	if a.repoCount() != 5 {
		t.Fatalf("演示工作区应有 5 个仓库(3 个文件夹 + 2 个嵌套):%d", a.repoCount())
	}
}

// 分支选择器(多仓库):顶部 chip 选仓库,点另一个仓库后列表换成它的分支,切分支作用在它身上。
func TestBranchPickerRepoChips(t *testing.T) {
	a, tt := newTestApp(t)
	a.ActivateWorkspace("ws-fullstack")
	a.OpenQuick("branch", "")
	tt.Frame()
	if !tt.HasText("router-demo") || !tt.HasText("web-dashboard") || tt.HasText("feat/charts") {
		t.Fatal("应显示仓库 chip,且当前仓库(router-demo)的分支列表里没有 feat/charts")
	}
	if err := tt.Click("web-dashboard"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.repoAct != 1 || a.qp == nil || a.qp.kind != "branch" {
		t.Fatalf("应切到 web-dashboard 且选择器仍开着:act=%d", a.repoAct)
	}
	if !tt.HasText("feat/charts") {
		t.Fatal("分支列表应换成 web-dashboard 的")
	}
	a.switchBranch("main")
	tt.Frame()
	if a.branch != "main" || a.repos[0].branch != "main" {
		t.Fatalf("切分支只作用于 web-dashboard:%s", a.branch)
	}
}
