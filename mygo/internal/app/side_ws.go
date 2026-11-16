package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/projects"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 侧栏的「项目与工作区」区块:标题行「+」(添加项目… / 新建工作区…)、工作区行(所有成员文件夹的会话合在一起)、
// 项目行,全部带右键菜单。设计稿:「项目与工作区」「右键菜单与动作矩阵」页。

// projectsHeader 是区块标题行:文字 + 「+」入口弹层。
// 它固定在会话列表上方,不随列表滚动;withAdd 为假时不画「+」(方案 C 的侧栏标题行已经有)。
func (a *App) projectsHeader(c *ui.Context, withAdd bool) {
	t := a.tk
	ui.Row(c).Key("projhdr").Height(24).PaddingX(8).Margin(6, 0, 0, 0).AlignItems(ui.Center).Gap(4).Shrink(0).Children(func() {
		ui.Text(c, "项目与工作区").Grow(1).FontSize(theme.FsMicro).FontWeight(600).TextColor(t.Tx3)
		if withAdd {
			a.addMenuButton(c)
		}
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

// caretBtn 是行首的展开箭头:单独可点,只展开 / 收起,不改变当前选中的项目或工作区。
func (a *App) caretBtn(c *ui.Context, key string, open bool, toggle func()) {
	t := a.tk
	b := ui.ButtonBase(c).Key(key).Size(16, 18).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).Radius(4).Margin(0, -2, 0, -2).Cursor(ui.CursorPointer)
	b.OnClick(toggle)
	b.Children(func() {
		rot := float32(0)
		if open {
			rot = 90
		}
		car := kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4)
		car.Rotate(car.AnimateWith("rot", rot, kit.DurBase, kit.EaseOut))
	})
}

// wsRow 工作区行:点行 = 选中这个工作区(聊天栏与 IDE 一起切过来)并展开;已选中时再点 = 展开 / 收起;
// 点箭头只展开 / 收起。展开后列出所有成员文件夹的会话(工作区是平的,没有主文件夹)。
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
		if !on {
			a.selectWorkspace(w)
			return
		}
		a.projOpen[w.ID] = !open
	})
	r.ContextMenu(func(m *ui.Menu) { a.wsMenu(m, w) })
	kit.DragLift(r, r.Dragging())
	r.Children(func() {
		if over {
			dropLine(c, t)
		}
		a.caretBtn(c, "wscar-"+w.ID, open, func() { a.projOpen[w.ID] = !open })
		ic := kit.Ic(c, "layers", kit.S12).TextColor(t.Tx3)
		if on {
			ic.TextColor(t.Ac)
		}
		ui.Text(c, w.Name).Grow(1).SingleLine()
		if hov {
			kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "在此工作区新建会话"}).Margin(0, -4, 0, 0).OnClick(func() { a.NewSessionIn(a.startProject(w)) })
		} else {
			ui.Text(c, itoa(len(w.Folders))+" 个文件夹").FontSize(theme.FsMicro).FontWeight(400).TextColor(t.Tx4)
		}
	})
	if !open {
		return
	}
	// 展开后列出所有成员文件夹的会话;点会话时先进入这个工作区(它可能还没被选中)。
	a.sessCtxWs = w.ID
	a.sessionRowsN(c, a.wsProject(w), 6)
	a.sessCtxWs = ""
}

// folderItems 给「在访达中显示 / 复制路径」这类按文件夹动作的菜单项:只有一个文件夹直接是菜单项,多个就列成子菜单(平级,不分主次)。
func folderItems(m *ui.Menu, icon, label string, w projects.Workspace, fn func(f projects.Folder)) {
	if len(w.Folders) == 1 {
		if mItem(m, icon, label).Chosen() {
			fn(w.Folders[0])
		}
		return
	}
	m.Submenu(label, func(sm *ui.Menu) {
		for _, f := range w.Folders {
			if sm.Item(f.Label()).Chosen() {
				fn(f)
			}
		}
	}).Icon(kit.MenuIcon(icon))
}

// mItem 是带图标的菜单项(图标取设计稿的线性图标,窗口内自绘菜单才会显示)。
func mItem(m *ui.Menu, icon, label string) *ui.MenuItem {
	return m.Item(label).Icon(kit.MenuIcon(icon))
}

// wsMenu 是工作区行的右键菜单:主要动作是「编辑工作区…」(名称 / 文件夹 / 顺序都在那一个对话框里)。
func (a *App) wsMenu(m *ui.Menu, w projects.Workspace) {
	if mItem(m, "plus", "在此工作区新对话").Chosen() {
		a.NewSessionIn(a.startProject(w))
	}
	m.Separator()
	if mItem(m, "edit", "编辑工作区…").Tooltip("改名称、增删文件夹、调整顺序").Chosen() {
		a.openWsDialog(w.ID, "")
	}
	if mItem(m, "folder", "添加文件夹…").Chosen() {
		a.pickFolders(true, func(ps []string) { a.wsAddFolders(w.ID, ps) })
	}
	a.wsMoveItems(m, w)
	m.Separator()
	folderItems(m, "folder", "在访达中显示", w, func(f projects.Folder) { a.revealInFinder(f.Path) })
	folderItems(m, "copy", "复制路径", w, func(f projects.Folder) { a.copyOut(f.Path, "路径") })
	if mItem(m, "download", "导出为工作区文件…").Disabled(a.demoWs()).Tooltip(map[bool]string{true: "演示模式没有真实目录", false: "写到「下载」并在访达里显示;双击该文件可在别处导入"}[a.demoWs()]).Chosen() {
		a.exportWorkspaceFile(w)
	}
	m.Separator()
	if mItem(m, "trash", "从列表移除工作区").Danger().Tooltip("只移出列表,不删除磁盘上的文件").Chosen() {
		a.confirmDlg("移除工作区「"+w.Name+"」?", "只会从列表里移除这个工作区,磁盘上的文件夹与会话记录都不会动。", "移除", true, func() { a.removeWorkspace(w.ID) })
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

// projMenu 是项目行的右键菜单:主要动作是「编辑项目…」。
func (a *App) projMenu(m *ui.Menu, p core.ProjectInfo) {
	if mItem(m, "plus", "在此项目新对话").Chosen() {
		a.NewSessionIn(p)
	}
	m.Separator()
	if mItem(m, "edit", "编辑项目…").Tooltip("改侧栏里显示的名称,查看位置与所属工作区").Chosen() {
		a.openProjDialog(p)
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
		}).Icon(kit.MenuIcon("layers"))
	} else if mItem(m, "layers", "加入新工作区…").Chosen() {
		a.openWsDialog("", p.Path)
	}
	m.Separator()
	if mItem(m, "folder", "在访达中显示").Chosen() {
		a.revealInFinder(p.Path)
	}
	if mItem(m, "copy", "复制路径").Chosen() {
		a.copyOut(p.Path, "路径")
	}
	if a.registry().HasProject(p.Path) { // 只有手动添加的能移除;Claude 会话索引里的项目随会话存在
		m.Separator()
		if mItem(m, "trash", "从列表移除项目").Danger().Tooltip("只移出列表,不删除文件夹").Chosen() {
			a.removeProjectAsk(p)
		}
	}
}

// ───────── 拖拽排序 ─────────

// wsDrag 是侧栏里工作区行的拖拽载荷。
type wsDrag struct{ ID string }

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

// moveFolder 把工作区里第 from 个文件夹放到第 before 个之前(文件夹平级,谁都能换位)。
func (a *App) moveFolder(wsID string, from, before int) {
	if from < 0 || before < 0 {
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
	if mItem(m, "chevU", "上移").Disabled(pos == 0).Chosen() {
		a.moveWorkspace(w.ID, all[pos-1].ID)
	}
	if mItem(m, "chevD", "下移").Disabled(pos == len(all)-1).Chosen() {
		before := ""
		if pos+2 < len(all) {
			before = all[pos+2].ID
		}
		a.moveWorkspace(w.ID, before)
	}
}

// SetProjScanning 供 main 的定时重扫调用:重扫期间侧栏显示骨架。
func (a *App) SetProjScanning(on bool) { a.projScan = on }
