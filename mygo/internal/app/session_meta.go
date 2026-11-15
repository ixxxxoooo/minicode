package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/sessmeta"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 会话行的置顶 / 归档 / 重命名 / 删除 / 复制为 Markdown(设计稿 menus「会话行」一节)。
// 置顶 / 归档 / 自定义标题只存本机整理信息(sessmeta),不改 Claude 的会话文件;删除才动磁盘。

// sessRename 是侧栏行内重命名的状态。
type sessRename struct {
	id     string
	buf    string
	frames int // 已渲染的帧数:输入框拿到焦点之前不判「点别处取消」
}

// SetSessMeta 装配会话整理信息的存储(真实模式落盘,演示用内存)。
func (a *App) SetSessMeta(m *sessmeta.Store) {
	a.meta = m
	a.projects = a.applyMeta(a.rawProj)
}

// SetSessionDeleter 装配「删除会话」的磁盘动作(真实模式)。
func (a *App) SetSessionDeleter(f func(id string) error) { a.sessDel = f }

func (a *App) sessMeta() *sessmeta.Store {
	if a.meta == nil {
		a.meta = sessmeta.NewMem()
	}
	return a.meta
}

// applyMeta 把整理信息套到项目的会话上:自定义标题、置顶 / 归档标记(不改变时间顺序,置顶在侧栏渲染时才提前)。
func (a *App) applyMeta(in []core.ProjectInfo) []core.ProjectInfo {
	out := make([]core.ProjectInfo, len(in))
	for i, p := range in {
		ss := make([]core.SessionInfo, len(p.Sessions))
		for k, s := range p.Sessions {
			m := a.sessMeta().Get(s.ID)
			if m.Title != "" {
				s.Title = m.Title
			}
			s.Pinned, s.Archived = m.Pinned, m.Archived
			ss[k] = s
		}
		p.Sessions = ss
		out[i] = p
	}
	return out
}

// reapplyMeta 整理信息变了以后重新套用。
func (a *App) reapplyMeta() { a.projects = a.applyMeta(a.rawProj) }

// sessionByID 按 id 找会话与它所在的项目(取套用整理信息后的)。
func (a *App) sessionByID(id string) (core.SessionInfo, core.ProjectInfo, bool) {
	for _, p := range a.projects {
		for _, s := range p.Sessions {
			if s.ID == id {
				return s, p, true
			}
		}
	}
	return core.SessionInfo{}, core.ProjectInfo{}, false
}

// rawTitle 会话原本的标题(没套用自定义标题之前)。
func (a *App) rawTitle(id string) string {
	for _, p := range a.rawProj {
		for _, s := range p.Sessions {
			if s.ID == id {
				return s.Title
			}
		}
	}
	return ""
}

// sessionsSplit 把项目的会话分成正常的(置顶的排最前,其余保持时间顺序)与已归档的。
func sessionsSplit(p core.ProjectInfo) (live, archived []core.SessionInfo) {
	var pinned []core.SessionInfo
	for _, s := range p.Sessions {
		switch {
		case s.Archived:
			archived = append(archived, s)
		case s.Pinned:
			pinned = append(pinned, s)
		default:
			live = append(live, s)
		}
	}
	return append(pinned, live...), archived
}

// sessionRows 画一个项目下的会话(最多 4 条)与底部的「已归档 N」折叠行。
func (a *App) sessionRows(c *ui.Context, p core.ProjectInfo) {
	live, arch := sessionsSplit(p)
	for i, s := range live {
		if i >= 4 {
			break
		}
		a.sessRow(c, s, true)
	}
	if len(arch) == 0 {
		return
	}
	t := a.tk
	key := "arch:" + p.ID
	open := a.projOpen[key]
	r := kit.RowEl(c, false).Key("archhead-" + p.ID).Height(24).PaddingStart(25).Gap(5).TextColor(t.Tx3)
	r.OnClick(func() { a.projOpen[key] = !open })
	r.Children(func() {
		rot := float32(0)
		if open {
			rot = 90
		}
		car := kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4)
		car.Rotate(car.AnimateWith("rot", rot, kit.DurBase, kit.EaseOut))
		kit.Ic(c, "archive", kit.S12)
		ui.Text(c, "已归档").Grow(1)
		ui.Text(c, itoa(len(arch))).FontSize(theme.FsMicro).TextColor(t.Tx4).FontFeatures("tnum")
	})
	if open {
		for _, s := range arch {
			a.sessRow(c, s, true)
		}
	}
}

// ───────── 动作 ─────────

func (a *App) pinSession(id string, v bool) {
	if err := a.sessMeta().SetPinned(id, v); err != nil {
		a.toast = "保存失败:" + err.Error()
		log.Warn("置顶保存失败", "id", id, "err", err)
		return
	}
	log.Info("置顶会话", "id", id, "pinned", v)
	a.reapplyMeta()
	a.toast = map[bool]string{true: "已置顶", false: "已取消置顶"}[v]
}

func (a *App) archiveSession(id string, v bool) {
	if err := a.sessMeta().SetArchived(id, v); err != nil {
		a.toast = "保存失败:" + err.Error()
		log.Warn("归档保存失败", "id", id, "err", err)
		return
	}
	log.Info("归档会话", "id", id, "archived", v)
	a.reapplyMeta()
	if v { // 归档后让它在「已归档」里可见一次
		if _, p, ok := a.sessionByID(id); ok {
			a.projOpen["arch:"+p.ID] = a.projOpen["arch:"+p.ID] || false
		}
	}
	a.toast = map[bool]string{true: "已归档(在项目底部的「已归档」里)", false: "已取消归档"}[v]
}

// beginRename 进入行内重命名:展开所在项目、打开侧栏,输入框预填当前标题。
func (a *App) beginRename(id string) {
	s, p, ok := a.sessionByID(id)
	if !ok {
		return
	}
	a.sideOpen = true
	if a.nav != "" && a.nav != "sessions" {
		a.nav = ""
	}
	a.projOpen[p.ID] = true
	if s.Archived {
		a.projOpen["arch:"+p.ID] = true
	}
	a.ren = sessRename{id: id, buf: s.Title}
}

func (a *App) commitRename() {
	id, title := a.ren.id, strings.TrimSpace(a.ren.buf)
	a.ren = sessRename{}
	if id == "" {
		return
	}
	if title == a.rawTitle(id) { // 与原标题一样 = 恢复原标题
		title = ""
	}
	if err := a.sessMeta().SetTitle(id, title); err != nil {
		a.toast = "保存失败:" + err.Error()
		return
	}
	log.Info("重命名会话", "id", id, "reset", title == "")
	a.reapplyMeta()
	if title == "" {
		a.toast = "已恢复原标题"
	}
}

// renameInput 行内重命名的输入框:⏎ 保存,Esc 或点别处取消,留空恢复原标题。
func (a *App) renameInput(c *ui.Context, id string) {
	t := a.tk
	a.ren.frames++
	box := ui.Row(c).Key("ren-box-"+id).Grow(1).MinWidth(0).Height(24).PaddingX(8).Gap(4).AlignItems(ui.Center).
		Background(t.Fill1).Radius(t.RSm()).Border(1, t.Ac)
	box.Children(func() {
		in := ui.TextInputBase(c.Key("ren-"+id), &a.ren.buf).Grow(1).MinWidth(0).FontSize(theme.FsBody).
			Background(ui.Transparent).BorderWidth(0).Padding(0).FocusRing(false).AutoFocus()
		in.OnSubmit(a.commitRename)
		in.OnShortcut(0, ui.KeyEscape, func() { a.ren = sessRename{} })
	})
	if a.ren.frames > 3 && !box.FocusWithin() { // 点了别处:取消
		a.ren = sessRename{}
	}
}

// copySessionMarkdown 复制当前会话的 Markdown(只对已打开的会话可用)。
func (a *App) copySessionMarkdown(id, title string) {
	if id != a.curSess {
		return
	}
	md, n := core.ExportMarkdown(title, a.st.Items)
	a.copyOut(md, "为 Markdown · "+itoa(n)+" 条")
}

// sessionRunning 会话正在运行(当前会话回合进行中,或停在后台继续跑)。
func (a *App) sessionRunning(id string) bool {
	if id == "" {
		return false
	}
	if id == a.curSess && a.st.Phase == core.PhaseRunning {
		return true
	}
	_, ok := a.live[id]
	return ok
}

// deleteSessionAsk 弹确认框,确认后删除会话。
func (a *App) deleteSessionAsk(id string) {
	s, _, ok := a.sessionByID(id)
	if !ok {
		return
	}
	a.confirmDlg("删除这个会话?", "「"+s.Title+"」的会话记录会从磁盘移除,无法恢复。", "删除", true, func() { a.deleteSession(id) })
}

func (a *App) deleteSession(id string) {
	s, p, ok := a.sessionByID(id)
	if !ok {
		return
	}
	if a.sessHide == nil {
		a.sessHide = map[string]time.Time{}
	}
	a.sessHide[id] = time.Now()
	a.frameSoon()
	del := a.sessDel
	snap := s
	log.Info("删除会话", "id", id, "title", s.Title, "persistent", del != nil)
	done := make(chan struct{})
	if a.sessDelWait == nil {
		a.sessDelWait = map[string]func(){}
	}
	a.sessDelWait[id] = func() { close(done) }
	a.delUndo.Begin("已删除「"+s.Title+"」", func() {
		delete(a.sessHide, id)
		if cancel := a.sessDelWait[id]; cancel != nil {
			cancel()
		}
		delete(a.sessDelWait, id)
	})
	a.bg("delete-session", func() func() {
		defer diag.SpanSlow("app", "删除会话", 300*1e6)()
		time.Sleep(kit.DurSlow)
		return func() {
			if _, folding := a.sessHide[id]; !folding {
				return
			}
			a.dropSession(id, p)
			delete(a.sessHide, id)
		}
	})
	a.bg("delete-session-disk", func() func() {
		select {
		case <-time.After(kit.UndoDuration):
		case <-done:
			return func() {
				delete(a.sessDelWait, id)
			}
		}
		var err error
		if del != nil {
			err = del(id)
		}
		return func() {
			delete(a.sessDelWait, id)
			if err != nil {
				a.insertSession(snap, p)
				a.delUndo.Clear()
				a.toast = "删除失败:" + err.Error()
				log.Warn("删除会话失败", "id", id, "err", err)
			}
		}
	})
}

// insertSession 把一条会话插回项目的侧栏列表(撤销删除用)。
func (a *App) insertSession(s core.SessionInfo, p core.ProjectInfo) {
	for i, pr := range a.rawProj {
		if pr.ID != p.ID {
			continue
		}
		for _, ex := range pr.Sessions {
			if ex.ID == s.ID {
				a.reapplyMeta()
				return
			}
		}
		a.rawProj[i].Sessions = append(a.rawProj[i].Sessions, s)
		a.rawProj[i].Count++
		a.reapplyMeta()
		a.OpenSession(s.ID)
		return
	}
}

// dropSession 从列表里移除一个已删除的会话;删的是当前会话就切到同项目的下一个,没有就开新会话。
func (a *App) dropSession(id string, p core.ProjectInfo) {
	_ = a.sessMeta().Forget(id)
	for i, pr := range a.rawProj {
		if pr.ID != p.ID {
			continue
		}
		ss := make([]core.SessionInfo, 0, len(pr.Sessions))
		for _, s := range pr.Sessions {
			if s.ID != id {
				ss = append(ss, s)
			}
		}
		a.rawProj[i].Sessions = ss
		if a.rawProj[i].Count > 0 {
			a.rawProj[i].Count--
		}
	}
	a.reapplyMeta()
	if id != a.curSess {
		return
	}
	for _, pr := range a.projects {
		if pr.ID != p.ID {
			continue
		}
		if live, _ := sessionsSplit(pr); len(live) > 0 {
			a.OpenSession(live[0].ID)
			return
		}
	}
	a.OpenSession("new:" + p.Path)
}

// SessionDeleter 是真实模式的删除实现:按索引找到会话所在目录再删文件。
func SessionDeleter(locate func(id string) (cwd string, ok bool)) func(id string) error {
	return func(id string) error {
		cwd, ok := locate(id)
		if !ok {
			return nil // 已经不在磁盘上
		}
		return claude.DeleteSession(claude.ProjectsDir(), cwd, id)
	}
}

// seedDemoSessMeta 演示 / 截图:置顶 3、归档 5、自定义标题 2(数据见 demo.SessMetaPresets)。
func (a *App) seedDemoSessMeta() {
	m := sessmeta.NewMem()
	demo.ApplySessMeta(m)
	a.meta = m
	a.reapplyMeta()
}
