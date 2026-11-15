package app

import (
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/projects"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 侧栏的「项目与工作区」区块:标题行「+」(添加项目… / 新建工作区…)、工作区行(成员文件夹 + 主文件夹的会话)、
// 项目行,全部带右键菜单。设计稿:「项目与工作区」「右键菜单与动作矩阵」页。

// projectsHeader 是区块标题行:文字 + 「+」入口弹层。
func (a *App) projectsHeader(c *ui.Context) {
	t := a.tk
	ui.Row(c).Key("projhdr").Height(24).PaddingX(8).Margin(6, 0, 0, 0).AlignItems(ui.Center).Gap(4).Shrink(0).Children(func() {
		ui.Text(c, "项目与工作区").Grow(1).FontSize(theme.FsMicro).FontWeight(600).TextColor(t.Tx3)
		a.addMenuButton(c)
	})
}

// addMenuButton 是「+」按钮与它的弹层(方案 C 的侧栏标题行也用它)。
func (a *App) addMenuButton(c *ui.Context) {
	b := kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "添加项目或工作区", On: a.popup == "addmenu"})
	b.OnClick(func() { a.togglePop("addmenu") })
	a.popOpen("addmenu", func(open *bool) {
		ui.PopoverBase(c, b, open, func(panel ui.Element) {
			a.popPanel(panel, 220).AttachTo(b, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(2, 0, 0, 0)
			item := func(key, icon, text, right string, fn func()) {
				if a.mi(c, "add-"+key, icon, text, right, false).Clicked() {
					a.popup = ""
					fn()
				}
			}
			item("proj", "folder", "添加项目…", "⌘ O", a.openAddProject)
			item("ws", "layers", "新建工作区…", "", func() { a.openWsDialog("", "") })
			if len(a.sideProjects()) > 0 {
				kit.Sep(c)
				ui.Text(c, "项目是一个文件夹;工作区是一组文件夹").FontSize(theme.FsMicro).TextColor(a.tk.Tx4).Padding(4, 8)
			}
		})
	})
}

// projectsList 画工作区行(置顶)与项目行;成员文件夹折进它所在的工作区,不再单独列出。
func (a *App) projectsList(c *ui.Context) {
	if a.projScan && len(a.sideProjects()) > 0 {
		kit.SkelRows(c, 4)
		return
	}
	for _, w := range a.registry().Workspaces() {
		a.wsRow(c, w)
	}
	for _, p := range a.sideProjects() {
		if a.memberOfWorkspace(p.Path) {
			continue
		}
		a.projRow(c, p)
		if a.projOpen[p.ID] {
			a.sessionRows(c, p)
		}
	}
}

// primaryProject 工作区主文件夹对应的侧栏项目(没有会话的文件夹也有一条空项目)。
func (a *App) primaryProject(w projects.Workspace) core.ProjectInfo {
	if p, ok := a.projectByPath(w.Primary()); ok {
		return p
	}
	return core.ProjectInfo{ID: "path:" + w.Primary(), Name: filepath.Base(w.Primary()), Path: w.Primary()}
}

func (a *App) wsRow(c *ui.Context, w projects.Workspace) {
	t := a.tk
	open := a.projOpen[w.ID]
	on := a.ws != nil && a.ws.ID == w.ID
	r := kit.RowEl(c, false).Key("ws-" + w.ID).Gap(5).TextColor(t.Tx1).FontWeight(500)
	r.Drag(wsDrag{ID: w.ID}) // 拖动换位
	if d, ok := ui.Drop[wsDrag](r); ok {
		a.moveWorkspace(d.ID, w.ID)
	}
	d, over := ui.DragOver[wsDrag](r)
	over = over && d.ID != w.ID
	hov := r.Hovered()
	r.OnClick(func() {
		a.projOpen[w.ID] = !open
		if !open || !on {
			a.ActivateWorkspace(w.ID)
		}
	})
	r.ContextMenu(func(m *ui.Menu) { a.wsMenu(m, w) })
	r.Children(func() {
		if over {
			dropLine(c, t)
		}
		rot := float32(0)
		if open {
			rot = 90
		}
		car := kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4)
		car.Rotate(car.AnimateWith("rot", rot, kit.DurBase, kit.EaseOut))
		ic := kit.Ic(c, "layers", kit.S12).TextColor(t.Tx3)
		if on {
			ic.TextColor(t.Ac)
		}
		ui.Text(c, w.Name).Grow(1).SingleLine()
		if hov {
			kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "在主文件夹新建会话"}).Margin(0, -4, 0, 0).OnClick(func() { a.NewSessionIn(a.primaryProject(w)) })
		} else {
			ui.Text(c, itoa(len(w.Folders))+" 个文件夹").FontSize(theme.FsMicro).FontWeight(400).TextColor(t.Tx4)
		}
	})
	if !open {
		return
	}
	for i, f := range w.Folders {
		a.memberRow(c, w, i, f)
	}
	prim := a.primaryProject(w)
	a.sessionRows(c, prim)
}

func (a *App) memberRow(c *ui.Context, w projects.Workspace, i int, f projects.Folder) {
	t := a.tk
	key := "mem-" + w.ID + "-" + itoa(i)
	r := kit.RowEl(c, false).Key(key).Height(22).PaddingStart(34).Gap(5).TextColor(t.Tx3)
	r.Drag(memDrag{WS: w.ID, Idx: i})
	if d, ok := ui.Drop[memDrag](r); ok && d.WS == w.ID {
		a.moveFolder(w.ID, d.Idx, i)
	}
	d, over := ui.DragOver[memDrag](r)
	over = over && d.WS == w.ID && i >= 1 && d.Idx != i && d.Idx >= 1 // 主文件夹行不可放
	r.OnClick(func() { a.locateFolder(w, f) })
	r.ContextMenu(func(m *ui.Menu) { a.memberMenu(m, w, i, f) })
	r.Children(func() {
		if over {
			dropLine(c, t)
		}
		kit.Ic(c, "folder", kit.S12)
		ui.Text(c, f.Label()).Grow(1).SingleLine()
		if i == 0 {
			kit.Tag(c, "主", kit.KAc)
		} else if !a.isRepoDir(f.Path) {
			kit.Tag(c, "非仓库")
		}
	})
}

// locateFolder 在文件树里定位到成员文件夹(必要时先进入这个工作区)。
func (a *App) locateFolder(w projects.Workspace, f projects.Folder) {
	a.ActivateWorkspace(w.ID)
	if a.ws == nil {
		return
	}
	a.nav = "files"
	for _, e := range a.repos {
		if e.dir == f.Path {
			a.ide.open[e.name] = true
			a.ide.sel = e.name
		}
	}
}

func (a *App) wsMenu(m *ui.Menu, w projects.Workspace) {
	if m.Item("新对话(主文件夹)").Chosen() {
		a.NewSessionIn(a.primaryProject(w))
	}
	a.wsMoveItems(m, w)
	m.Separator()
	if m.Item("重命名…").Chosen() || m.Item("编辑文件夹…").Chosen() {
		a.openWsDialog(w.ID, "")
	}
	if m.Item("添加文件夹…").Chosen() {
		a.pickFolders(true, func(ps []string) { a.wsAddFolders(w.ID, ps) })
	}
	m.Separator()
	if m.Item("在访达中显示主文件夹").Chosen() {
		a.revealInFinder(w.Primary())
	}
	if m.Item("复制主文件夹路径").Chosen() {
		a.copyOut(w.Primary(), "路径")
	}
	m.Separator()
	if m.Item("从列表移除工作区").Chosen() {
		a.confirmDlg("移除工作区「"+w.Name+"」?", "只会从列表里移除这个工作区,磁盘上的文件夹与会话记录都不会动。", "移除", true, func() { a.removeWorkspace(w.ID) })
	}
}

func (a *App) memberMenu(m *ui.Menu, w projects.Workspace, i int, f projects.Folder) {
	if m.Item("在文件树中定位").Chosen() {
		a.locateFolder(w, f)
	}
	if i > 0 && m.Item("设为主文件夹").Chosen() {
		a.wsMutate(w.ID, func(w *projects.Workspace) {
			w.Folders[0], w.Folders[i] = w.Folders[i], w.Folders[0]
		})
	}
	a.memberMoveItems(m, w, i)
	m.Separator()
	if m.Item("在源代码管理中查看").Chosen() {
		a.locateFolder(w, f)
		for k, e := range a.repos {
			if e.dir == f.Path {
				a.setActiveRepo(k)
			}
		}
		a.nav = "scm"
		a.SetMode(ModeIDE)
	}
	if m.Item("在访达中显示").Chosen() {
		a.revealInFinder(f.Path)
	}
	if m.Item("复制路径").Chosen() {
		a.copyOut(f.Path, "路径")
	}
	if i > 0 {
		m.Separator()
		if m.Item("从工作区移出").Chosen() {
			a.wsMutate(w.ID, func(w *projects.Workspace) { w.Folders = append(w.Folders[:i:i], w.Folders[i+1:]...) })
		}
	}
}

// wsMutate 改一个已保存的工作区并保存;正在使用的工作区重新进入使文件树生效。
func (a *App) wsMutate(id string, f func(w *projects.Workspace)) {
	w, ok := a.registry().Workspace(id)
	if !ok {
		return
	}
	f(&w)
	if _, err := a.registry().SaveWorkspace(w); err != nil {
		a.toast = err.Error()
		return
	}
	if a.ws != nil && a.ws.ID == id {
		a.leaveWorkspace()
		a.ActivateWorkspace(id)
	}
}

func (a *App) wsAddFolders(id string, paths []string) {
	a.wsMutate(id, func(w *projects.Workspace) {
		for _, p := range paths {
			w.Folders = append(w.Folders, projects.Folder{Path: p})
		}
	})
}

func (a *App) removeWorkspace(id string) {
	if a.ws != nil && a.ws.ID == id {
		a.leaveWorkspace()
	}
	if err := a.registry().RemoveWorkspace(id); err != nil {
		a.toast = "移除失败:" + err.Error()
		return
	}
	a.toast = "已移除工作区"
}

// projMenu 是项目行的右键菜单。
func (a *App) projMenu(m *ui.Menu, p core.ProjectInfo) {
	if m.Item("在此项目新对话").Chosen() {
		a.NewSessionIn(p)
	}
	ws := a.registry().Workspaces()
	if len(ws) > 0 {
		m.Submenu("加入工作区…", func(sm *ui.Menu) {
			for _, w := range ws {
				if w.Has(p.Path) {
					continue
				}
				if sm.Item(w.Name).Chosen() {
					a.wsAddFolders(w.ID, []string{p.Path})
					a.toast = "已加入「" + w.Name + "」"
				}
			}
			sm.Separator()
			if sm.Item("新建工作区…").Chosen() {
				a.openWsDialog("", p.Path)
			}
		})
	} else if m.Item("加入新工作区…").Chosen() {
		a.openWsDialog("", p.Path)
	}
	m.Separator()
	if m.Item("在访达中显示").Chosen() {
		a.revealInFinder(p.Path)
	}
	if m.Item("复制路径").Chosen() {
		a.copyOut(p.Path, "路径")
	}
	if a.registry().HasProject(p.Path) { // 只有手动添加的能移除;Claude 会话索引里的项目随会话存在
		m.Separator()
		if m.Item("从列表移除项目").Chosen() {
			body := "只会从列表里移除这个项目,磁盘上的文件夹不会动。"
			if n := len(a.registry().WorkspacesOf(p.Path)); n > 0 {
				body = "它属于 " + itoa(n) + " 个工作区,移除只影响这个列表,工作区里的引用保留。\n" + body
			}
			a.confirmDlg("移除项目「"+p.Name+"」?", body, "移除", true, func() {
				_ = a.registry().RemoveProject(p.Path)
				a.toast = "已移除项目"
			})
		}
	}
}

// ───────── 拖拽排序 ─────────

// wsDrag / memDrag 是侧栏里工作区行 / 成员文件夹行的拖拽载荷。
type wsDrag struct{ ID string }
type memDrag struct {
	WS  string
	Idx int
}

// dropLine 是拖拽经过时行顶部的 2px 插入线。
func dropLine(c *ui.Context, t *theme.Tokens) {
	ui.Box(c).Absolute().Left(0).Right(0).Top(0).Height(2).Background(t.Ac).PassThrough()
}

// moveWorkspace 把工作区 id 放到 before 之前(顺序落盘)。
func (a *App) moveWorkspace(id, before string) {
	if err := a.registry().MoveWorkspace(id, before); err != nil {
		a.toast = "保存失败:" + err.Error()
		return
	}
	log.Info("拖动工作区", "id", id, "before", before)
}

// moveFolder 把工作区里第 from 个文件夹放到第 before 个之前;第一个是主文件夹,不参与换位。
func (a *App) moveFolder(wsID string, from, before int) {
	if from < 1 || before < 1 {
		a.toast = "第一个是主文件夹,用「设为主文件夹」更换"
		return
	}
	a.wsMutate(wsID, func(w *projects.Workspace) { w.Folders, _ = projects.MoveBefore(w.Folders, from, before) })
}

// wsMoveItems 工作区行右键菜单里的「上移 / 下移」。
func (a *App) wsMoveItems(m *ui.Menu, w projects.Workspace) {
	all := a.registry().Workspaces()
	pos := -1
	for i, x := range all {
		if x.ID == w.ID {
			pos = i
		}
	}
	if pos < 0 || len(all) < 2 {
		return
	}
	m.Separator()
	if m.Item("上移").Disabled(pos == 0).Chosen() {
		a.moveWorkspace(w.ID, all[pos-1].ID)
	}
	if m.Item("下移").Disabled(pos == len(all)-1).Chosen() {
		before := ""
		if pos+2 < len(all) {
			before = all[pos+2].ID
		}
		a.moveWorkspace(w.ID, before)
	}
}

// memberMoveItems 成员文件夹右键菜单里的「上移 / 下移」(主文件夹固定在第一个)。
func (a *App) memberMoveItems(m *ui.Menu, w projects.Workspace, i int) {
	if len(w.Folders) < 3 || i == 0 {
		return
	}
	m.Separator()
	if m.Item("上移").Disabled(i <= 1).Tooltip(map[bool]string{true: "再往上就是主文件夹,用「设为主文件夹」更换"}[i <= 1]).Chosen() {
		a.moveFolder(w.ID, i, i-1)
	}
	if m.Item("下移").Disabled(i >= len(w.Folders)-1).Chosen() {
		a.moveFolder(w.ID, i, i+2)
	}
}

// SetProjScanning 供 main 的定时重扫调用:重扫期间侧栏显示骨架。
func (a *App) SetProjScanning(on bool) { a.projScan = on }
