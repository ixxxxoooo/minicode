package app

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

func projSessIDs(p core.ProjectInfo) string {
	var s []string
	for _, x := range p.Sessions {
		s = append(s, x.ID)
	}
	return strings.Join(s, ",")
}

// 工作区的会话 = 所有成员文件夹的会话合在一起(平的,没有主文件夹)。
func TestWsProjectMergesAllFolders(t *testing.T) {
	a, _ := newTestApp(t)
	w, _ := a.registry().Workspace("ws-fullstack")
	ids := projSessIDs(a.wsProject(w))
	for _, want := range []string{"fixbug", "browser", "subagent"} {
		if !strings.Contains(ids, want) {
			t.Fatalf("工作区会话应包含 %s,实际 %s", want, ids)
		}
	}
}

// 选中项目:退出工作区,聊天栏切到该项目最近的会话。
func TestSelectProjectLeavesWorkspace(t *testing.T) {
	a, _ := newTestApp(t)
	a.ActivateWorkspace("ws-fullstack")
	if a.ws == nil {
		t.Fatal("应已进入工作区")
	}
	p, ok := a.projectByPath("/demo/scenario-lab")
	if !ok || len(p.Sessions) == 0 {
		t.Fatal("演示数据里应有 scenario-lab 与它的会话")
	}
	a.selectProject(p)
	if a.ws != nil {
		t.Fatal("选中别的项目应退出工作区")
	}
	if a.curSess != p.Sessions[0].ID {
		t.Fatalf("应切到该项目最近的会话 %s,实际 %q", p.Sessions[0].ID, a.curSess)
	}
	if !a.ctxIsProject(p) {
		t.Fatal("该项目应是当前上下文")
	}
}

// 选中工作区:当前会话已在某个成员文件夹里就不动;不在则切到工作区最近的会话。
func TestSelectWorkspaceFollowsSession(t *testing.T) {
	a, _ := newTestApp(t)
	w, _ := a.registry().Workspace("ws-fullstack")
	a.selectWorkspace(w) // 当前会话 fixbug 在 router-demo,是成员
	if a.ws == nil || a.ws.ID != w.ID || a.curSess != "fixbug" {
		t.Fatalf("成员会话不应被换掉:ws=%v sess=%q", a.ws != nil, a.curSess)
	}
	// 在成员 web-dashboard 的会话里选中工作区:仍然保留该会话
	a.OpenSession("browser")
	a.selectWorkspace(w)
	if a.curSess != "browser" {
		t.Fatalf("成员文件夹里的会话都算工作区的会话:%q", a.curSess)
	}
	// 当前会话在工作区之外:选中工作区后切到工作区最近的会话
	a.leaveWorkspace()
	p, _ := a.projectByPath("/demo/scenario-lab")
	a.selectProject(p)
	a.selectWorkspace(w)
	first, _ := firstLive(a.wsProject(w).Sessions)
	if a.ws == nil || a.curSess != first.ID {
		t.Fatalf("应切到工作区最近的会话 %s,实际 %q", first.ID, a.curSess)
	}
}

// 工作区新会话沿用当前所在的成员文件夹,而不是写死第一个(没有「主文件夹」)。
func TestWsStartDirFollowsCurrentFolder(t *testing.T) {
	a, _ := newTestApp(t)
	w, _ := a.registry().Workspace("ws-fullstack")
	a.OpenSession("browser") // web-dashboard
	if got := a.wsStartDir(w); got != "/demo/web-dashboard" {
		t.Fatalf("应沿用当前文件夹,实际 %s", got)
	}
	a.OpenSession("fixbug")
	if got := a.wsStartDir(w); got != "/demo/router-demo" {
		t.Fatalf("应沿用当前文件夹,实际 %s", got)
	}
}

// 启动会话的 --add-dir:在工作区任一成员文件夹启动,都授权其余文件夹。
func TestWorkspaceOthers(t *testing.T) {
	a, _ := newTestApp(t)
	w, _ := a.registry().Workspace("ws-fullstack")
	got := strings.Join(w.Others("/demo/web-dashboard"), ",")
	if got != "/demo/router-demo,/demo/docs-site" {
		t.Fatalf("Others = %s", got)
	}
	a.ActivateWorkspace("ws-fullstack")
	if _, ok := a.WorkspaceFor("/demo/docs-site"); !ok {
		t.Fatal("使用中的工作区里任一成员都应命中")
	}
}

// 点侧栏的项目行 = 选中该项目;再点已选中的 = 收起 / 展开。
func TestSideProjectRowClickSelects(t *testing.T) {
	a, tt := newTestApp(t)
	a.ActivateWorkspace("ws-fullstack")
	tt.Frame()
	if err := tt.Click("scenario-lab"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.ws != nil {
		t.Fatal("点别的项目应退出工作区")
	}
	p, _ := a.projectByPath("/demo/scenario-lab")
	if !a.ctxIsProject(p) {
		t.Fatalf("scenario-lab 应是当前上下文,会话 %q", a.curSess)
	}
	open := a.projOpen[p.ID]
	if !open {
		t.Fatal("选中后应展开")
	}
	if err := tt.Click("scenario-lab"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.projOpen[p.ID] {
		t.Fatal("已选中再点应收起")
	}
}

// 侧栏头部(新对话 / 项目与工作区 + 添加)固定,滚动只动列表。
func TestSideHeaderStaysWhenListScrolls(t *testing.T) {
	a, tt := newTestApp(t)
	for _, p := range a.projects {
		a.projOpen[p.ID] = true
	}
	tt.Frame()
	before, ok := tt.Find("新对话")
	if !ok {
		t.Fatal("侧栏应有「新对话」")
	}
	hb, _ := tt.Find("项目与工作区")
	lb, _ := tt.Find("scenario-lab")
	tt.Move(before.X+40, before.Y+200)
	tt.Wheel(before.X+40, before.Y+200, 0, 400)
	tt.Frame()
	tt.Advance(400 * time.Millisecond)
	tt.Frame()
	after, _ := tt.Find("新对话")
	ha, _ := tt.Find("项目与工作区")
	if la, _ := tt.Find("scenario-lab"); la == lb {
		t.Fatal("列表本身应该滚动了(否则这个测试没有意义)")
	}
	if before != after || hb != ha {
		t.Fatalf("滚动列表后头部不应移动:%v→%v / %v→%v", before, after, hb, ha)
	}
}

// 方案 B(浮岛):项目条能切换也能添加,下拉里不逐个列成员文件夹。
func TestSchemeBPickerHasAddAndStaysShort(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "b", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	a.SeedDemoExtras()
	tt := ui.NewTester(a.View, 1280, 800)
	tt.Frame()
	a.popup = "projpick"
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"添加项目…", "新建工作区…", "全栈工作区"} {
		if !tt.HasText(want) {
			t.Fatalf("下拉里应有「%s」", want)
		}
	}
	if tt.HasText("成员") {
		t.Fatal("下拉里不应再逐个列出成员文件夹")
	}
}
