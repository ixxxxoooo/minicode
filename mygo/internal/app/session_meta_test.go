package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
)

func sessIDs(a *App, proj string) []string {
	var out []string
	for _, p := range a.projects {
		if p.ID == proj {
			for _, s := range p.Sessions {
				out = append(out, s.ID)
			}
		}
	}
	return out
}

// 侧栏:演示数据里「文件读写」置顶排最前,「已归档 2」折叠行在项目底部,展开后出现归档的会话。
func TestSessionRowsPinnedAndArchived(t *testing.T) {
	a, tt := newTestApp(t)
	tt.Frame()
	if !tt.HasText("已归档") || !tt.HasText("文件读写:梳理并整理路由层") {
		t.Fatal("侧栏应有置顶会话与「已归档」折叠行")
	}
	if tt.HasText("调研:axum 与 actix 的路由差异") {
		t.Fatal("归档的会话默认收起")
	}
	live, arch := sessionsSplit(a.projects[0])
	if live[0].ID != "files" || len(arch) != 2 {
		t.Fatalf("置顶应排最前、归档 2 个:%v %d", live[0].ID, len(arch))
	}
	// 时间顺序没变(历史列表等仍按时间)
	if ids := sessIDs(a, "router-demo"); ids[0] != "fixbug" {
		t.Fatalf("a.projects 不应被置顶打乱:%v", ids)
	}
	if err := tt.Click("已归档"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("调研:axum 与 actix 的路由差异") {
		t.Fatal("展开后应显示归档的会话")
	}
}

// 右键菜单:置顶 / 归档 / 复制为 Markdown / 删除的可用性与效果。
func TestSessionMenuActions(t *testing.T) {
	a, tt := newTestApp(t)
	if err := tt.RightClick("权限请求:修改配置与执行命令"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	menu := strings.Join(tt.Menu(), "|")
	for _, want := range []string{"重命名…", "置顶", "归档", "复制会话 ID", "复制标题", "复制为 Markdown", "删除会话…"} {
		if !strings.Contains(menu, want) {
			t.Fatalf("会话行右键菜单缺 %q:%s", want, menu)
		}
	}
	if err := tt.ChooseMenuItem("置顶"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.sessMeta().Get("perm").Pinned {
		t.Fatalf("应置顶:%+v", a.sessMeta().Get("perm"))
	}
	if live, _ := sessionsSplit(a.projects[0]); live[0].ID != "perm" && live[1].ID != "perm" {
		t.Fatalf("置顶的应排在前面:%v", live[0].ID)
	}
	// 归档后从列表里收进「已归档」
	a.archiveSession("perm", true)
	tt.Frame()
	if _, arch := sessionsSplit(a.projects[0]); len(arch) != 3 {
		t.Fatalf("归档后应有 3 个已归档:%d", len(arch))
	}
	a.archiveSession("perm", false)
	// 复制为 Markdown:当前会话可用,内容含用户消息与助手正文
	a.copySessionMarkdown("fixbug", "修复")
	if !strings.Contains(a.copyText, "**你**") || !strings.Contains(a.copyText, "**Claude**") || !strings.Contains(a.toast, "Markdown") {
		t.Fatalf("Markdown 内容不对:%q %q", a.copyText, a.toast)
	}
}

// 行内重命名:⏎ 保存、Esc 取消、与原标题相同(或清空)= 恢复原标题。
func TestSessionRename(t *testing.T) {
	a, tt := newTestApp(t)
	a.beginRename("perm")
	tt.Frame()
	if a.ren.id != "perm" || a.ren.buf != "权限请求:修改配置与执行命令" {
		t.Fatalf("应预填当前标题:%+v", a.ren)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.ren.id != "" {
		t.Fatal("Esc 应取消")
	}
	a.beginRename("perm")
	tt.Frame()
	a.ren.buf = "  改好的名字 "
	a.commitRename()
	if s, _, _ := a.sessionByID("perm"); s.Title != "改好的名字" {
		t.Fatalf("标题应更新:%q", s.Title)
	}
	if a.rawTitle("perm") != "权限请求:修改配置与执行命令" {
		t.Fatal("原标题应保留")
	}
	a.beginRename("perm")
	a.ren.buf = ""
	a.commitRename()
	if s, _, _ := a.sessionByID("perm"); s.Title != "权限请求:修改配置与执行命令" || a.toast != "已恢复原标题" {
		t.Fatalf("清空应恢复原标题:%q %q", s.Title, a.toast)
	}
}

// 删除:确认框 → 从列表移除并清掉整理信息;删当前会话切到同项目下一个;磁盘失败给出原因且不移除;运行中置灰。
func TestSessionDelete(t *testing.T) {
	a, tt := newTestApp(t)
	var gone []string
	a.SetSessionDeleter(func(id string) error { gone = append(gone, id); return nil })
	a.pinSession("perm", true)
	a.deleteSessionAsk("perm")
	tt.Frame()
	if !tt.HasText("删除这个会话?") {
		t.Fatal("应弹确认框")
	}
	if err := tt.Click("删除"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, id := range sessIDs(a, "router-demo") {
		if id == "perm" {
			t.Fatal("会话应已从列表移除")
		}
	}
	if len(gone) != 1 || gone[0] != "perm" || a.sessMeta().Get("perm").Pinned {
		t.Fatalf("应调用删除并清掉整理信息:%v", gone)
	}
	// 删除当前会话:切到同项目的下一个
	a.deleteSession("fixbug")
	if a.curSess == "fixbug" || a.curSess == "" {
		t.Fatalf("应切到同项目的下一个会话:%q", a.curSess)
	}
	// 磁盘失败:不移除、给原因
	a.SetSessionDeleter(func(id string) error { return errors.New("权限不足") })
	before := len(sessIDs(a, "router-demo"))
	a.deleteSession("ci")
	if len(sessIDs(a, "router-demo")) != before || !strings.Contains(a.toast, "权限不足") {
		t.Fatalf("失败时不应移除:%q", a.toast)
	}
	// 运行中的会话:菜单里「删除会话…」置灰
	a.curSess = "md"
	a.st.Phase = core.PhaseRunning
	if !a.sessionRunning("md") {
		t.Fatal("回合进行中的当前会话应判为运行中")
	}
	a.st.Phase = core.PhaseIdle
	if a.sessionRunning("md") {
		t.Fatal("空闲时不应判为运行中")
	}
}
