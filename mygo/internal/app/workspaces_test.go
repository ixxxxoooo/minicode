package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/projects"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// 演示工作区:进入后文件树是多根、树上的状态色带前缀、活动仓库可切换,退出后还原单项目。
func TestDemoWorkspaceLifecycle(t *testing.T) {
	stressDemo(t)
	a, tt := newTestApp(t)
	before := a.ide.fs
	a.ActivateWorkspace("ws-fullstack")
	tt.Frame()
	if !a.multiRoot() || len(a.repos) != 5 {
		t.Fatalf("应进入 3 个文件夹(+2 个嵌套仓库)的多根工作区:%d", len(a.repos))
	}
	m, ok := a.ide.fs.(*workspace.Multi)
	if !ok || m.Roots()[0].Name != "router-demo" {
		t.Fatal("文件系统应是多根,主文件夹在最前")
	}
	if a.ide.git["web-dashboard/src/charts.ts"] != "M" || a.ide.git["router-demo/src/router.rs"] != "M" {
		t.Fatalf("树上的状态色应带文件夹前缀:%v", a.ide.git)
	}
	if a.activeRepo().name != "router-demo" || a.branch != "main" {
		t.Fatalf("默认活动仓库是主文件夹:%s %s", a.activeRepo().name, a.branch)
	}
	// 活动仓库切换:分支、提交目标跟着变
	a.setActiveRepo(1)
	if a.branch != "feat/charts" || a.repoBranchLabel() != "web-dashboard · feat/charts" {
		t.Fatalf("切到 web-dashboard:%q %q", a.branch, a.repoBranchLabel())
	}
	if a.treePath("src/charts.ts") != "web-dashboard/src/charts.ts" {
		t.Fatal("活动仓库内路径应加文件夹前缀")
	}
	// 文件树路径 → 仓库:非活动仓库也能取到 HEAD 内容(行内追溯 / 差异用)
	if s, ok := a.headOfRepo("router-demo/src/router.rs"); !ok || !strings.Contains(s, "self.routes.get(path)") {
		t.Fatal("非活动仓库的文件也应能读到 HEAD")
	}
	if _, _, _, n := a.repoSummary(0); n < 300 {
		t.Fatalf("router-demo 总览行应含 bulk 等多项改动(≥300):%d", n)
	}
	// 非仓库文件夹没有仓库
	if r, _ := a.repoAt("nope/x"); r != nil {
		t.Fatal("不在任何文件夹的路径没有仓库")
	}
	a.leaveWorkspace()
	if a.multiRoot() || a.ide.fs != before {
		t.Fatal("退出后应还原单项目文件系统")
	}
}

// 打开不属于工作区成员的会话时退出工作区;成员的会话则留在工作区里。
func TestOpenSessionLeavesWorkspace(t *testing.T) {
	a, _ := newTestApp(t)
	a.ActivateWorkspace("ws-fullstack")
	a.OpenSession("browser") // web-dashboard 是成员
	if !a.multiRoot() {
		t.Fatal("成员文件夹的会话应留在工作区")
	}
	a.OpenSession("new:/demo/scenario-lab")
	if a.multiRoot() {
		t.Fatal("别的项目的会话应退出工作区")
	}
}

// 侧栏:成员文件夹折进工作区,手动添加的项目补在列表里。
func TestSidebarFoldsMembers(t *testing.T) {
	a, _ := newTestApp(t)
	if !a.memberOfWorkspace("/demo/web-dashboard") || a.memberOfWorkspace("/demo/scenario-lab") {
		t.Fatal("成员判断不对")
	}
	found := false
	for _, p := range a.sideProjects() {
		if p.Path == "/demo/design-assets" {
			found = true
		}
	}
	if !found {
		t.Fatal("手动添加的项目应出现在侧栏")
	}
	if a.isRepoDir("/demo/design-assets") {
		t.Fatal("design-assets 不是仓库")
	}
}

// 添加项目:校验错误给出下一步。
func TestSubmitAddProject(t *testing.T) {
	a, _ := newTestApp(t)
	a.openAddProject()
	a.pd.path = "~/Code/api-gatewy"
	a.submitAddProject()
	if a.pd.err == "" || !strings.Contains(a.pd.hint, "api-gateway") {
		t.Fatalf("应报找不到并给出相近路径:%q / %q", a.pd.err, a.pd.hint)
	}
	a.pd.hintDo()
	a.submitAddProject()
	if a.pd.kind != "" || !a.registry().HasProject("~/Code/api-gateway") {
		t.Fatalf("采纳建议后应添加成功:%+v", a.pd)
	}
	a.openAddProject()
	a.pd.path = "~/Code/api-gateway"
	a.submitAddProject()
	if !strings.Contains(a.pd.hint, "已经在列表里") {
		t.Fatalf("重复添加应提示:%q", a.pd.hint)
	}
}

// 工作区对话框:至少 2 个文件夹才能保存;保存后侧栏出现。
func TestSubmitWorkspace(t *testing.T) {
	a, _ := newTestApp(t)
	a.openWsDialog("", "")
	a.pd.name = "测试区"
	a.addWsFolder("/demo/router-demo")
	a.submitWorkspace()
	if a.pd.err == "" || a.pd.kind != "ws" {
		t.Fatal("只有 1 个文件夹不能保存")
	}
	a.addWsFolder("/demo/router-demo") // 重复
	if !strings.Contains(a.pd.err, "已经在") {
		t.Fatalf("重复的文件夹应提示:%q", a.pd.err)
	}
	a.addWsFolder("/demo/docs-site")
	a.submitWorkspace()
	// 首次保存会先出「附加文件夹权限」提示卡,选「我知道了」(读写)后才真正保存
	if a.pd.kind != "ws-perm" {
		t.Fatalf("首次保存应先弹权限提示卡:%q", a.pd.kind)
	}
	a.wsPermChoose(false)
	if a.pd.kind != "" {
		t.Fatalf("应保存成功:%q", a.pd.err)
	}
	n := 0
	for _, w := range a.registry().Workspaces() {
		if w.Name == "测试区" && len(w.Folders) == 2 {
			n++
		}
	}
	if n != 1 {
		t.Fatal("工作区没保存")
	}
}

func TestEditDistance(t *testing.T) {
	if editDistance("api-gatewy", "api-gateway") != 1 || editDistance("a", "a") != 0 || editDistance("", "abc") != 3 {
		t.Fatal("编辑距离不对")
	}
}

// 真实模式:磁盘上的两个文件夹(一个是 git 仓库、一个不是)组成工作区,多仓库探测与「初始化仓库」。
func TestRealWorkspace(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("没有 git")
	}
	root := t.TempDir()
	api, docs := filepath.Join(root, "api"), filepath.Join(root, "docs")
	for _, d := range []string{api, docs} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.WriteFile(filepath.Join(api, "main.go"), []byte("package main\n"), 0o644)
	_ = os.WriteFile(filepath.Join(docs, "a.md"), []byte("# a\n"), 0o644)
	run := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run(api, "init", "-q")
	run(api, "add", ".")
	run(api, "commit", "-q", "-m", "init")
	_ = os.WriteFile(filepath.Join(api, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644) // 一处改动

	a, tt := newTestApp(t)
	reg := projects.Open(filepath.Join(t.TempDir(), "ws.json"))
	w, err := reg.SaveWorkspace(projects.Workspace{Name: "真实", Folders: []projects.Folder{{Path: api}, {Path: docs}}})
	if err != nil {
		t.Fatal(err)
	}
	a.SetRegistry(reg)
	a.proj = api
	a.ActivateWorkspace(w.ID)
	tt.Frame()
	if !a.multiRoot() || len(a.repos) != 2 {
		t.Fatal("应进入真实工作区")
	}
	if a.repos[0].repo == nil || a.repos[1].repo != nil || !a.repos[1].checked {
		t.Fatalf("api 是仓库、docs 不是:%v %v", a.repos[0].repo != nil, a.repos[1].repo != nil)
	}
	if got := a.absPath("docs/a.md"); got != filepath.Join(docs, "a.md") {
		t.Fatalf("多根绝对路径:%s", got)
	}
	if rel := a.wsRel(filepath.Join(docs, "a.md")); rel != "docs/a.md" {
		t.Fatalf("工具调用绝对路径 → 树路径:%s", rel)
	}
	a.setActiveRepo(1)
	a.initRepo(1)
	if a.repos[1].repo == nil {
		t.Fatal("初始化后应成为仓库")
	}
	if _, err := os.Stat(filepath.Join(docs, ".git")); err != nil {
		t.Fatal("应真的执行了 git init")
	}
}
