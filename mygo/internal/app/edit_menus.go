package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// 各处「···」与右键菜单(设计稿 drag-more「···」菜单盘点)。
// 原则:菜单只放这个对象上真的能做的事,顺序「主要动作 → 导航 → 复制 / 访达 → 危险操作」;
// 后端还没接的动作置灰并写明原因,不隐藏;菜单文案要稳定(选完后菜单会按同一份 build 重建)。

// ───────── 编辑器标签 ─────────

// closeTabsWhere 关掉满足条件的标签(从后往前,保证下标稳定),返回关了几个。
func (a *App) closeTabsWhere(keep func(tb *ideTab) bool) int {
	n := 0
	for pane := int8(0); pane <= 1; pane++ {
		if pane == 1 && a.ide.splitDir == 0 {
			continue
		}
		tabs := a.ide.paneTabs(pane)
		for i := len(tabs) - 1; i >= 0; i-- {
			if !keep(tabs[i]) {
				a.closeWsTab(tabs[i].id)
				n++
			}
		}
	}
	return n
}

// tabIndex 是当前焦点组里标签的下标(找不到 -1)。
func (a *App) tabIndex(id string) int {
	tabs := a.ide.paneTabs(a.ide.paneSel)
	for i, t := range tabs {
		if t.id == id {
			return i
		}
	}
	return -1
}

// isFileTab 是不是对应磁盘上一个文件的标签(文件或差异,不含虚拟标签)。
func isFileTab(tb *ideTab) bool { return tb != nil && tb.virt == "" && tb.path != "" }

// reopenClosed 重新打开最近关闭的一个文件标签(⇧⌘T / 标签条菜单)。
func (a *App) reopenClosed() {
	d := a.ide
	for len(d.closed) > 0 {
		p := d.closed[len(d.closed)-1]
		d.closed = d.closed[:len(d.closed)-1]
		if d.tab(p) != nil {
			continue
		}
		d.openFile(p, true)
		uxLog.Info("重新打开已关闭的标签", "path", p)
		return
	}
	a.toast = "没有可以重新打开的标签"
}

// tabStripMenu 是标签条右侧「···」的菜单(与标签右键菜单里「关闭」一组一致)。
func (a *App) tabStripMenu(m *ui.Menu) {
	d := a.ide
	saved := 0
	for _, tb := range d.tabs {
		if !tb.dirty() {
			saved++
		}
	}
	if menuItem(m, "关闭已保存").Disabled(saved == 0).Chosen() {
		a.closeTabsWhere(func(tb *ideTab) bool { return tb.dirty() })
	}
	if menuItem(m, "关闭全部").Disabled(len(d.tabs) == 0).Tooltip("有未保存修改的标签也会关闭").Chosen() {
		a.closeTabsWhere(func(*ideTab) bool { return false })
	}
	m.Separator()
	mdSrc := a.prefs.Int(pMdView, 0) == 1
	if menuItem(m, "默认以预览打开 Markdown").Checked(!mdSrc).Chosen() {
		if mdSrc {
			a.prefs.Reset(pMdView)
		} else {
			a.prefs.Set(pMdView, 1)
		}
		a.toast = map[bool]string{true: "Markdown 默认以预览打开", false: "Markdown 默认以源码打开"}[mdSrc]
		for _, tb := range d.tabs { // 默认变了:各标签放弃自己的手动选择,统一跟随
			tb.mdMode = 0
		}
	}
	if menuItem(m, "自动换行").Trailing("⌥Z").Checked(a.prefs.Bool(pEdWrap, false)).Chosen() {
		a.toggleWrap()
	}
	m.Separator()
	if menuItem(m, "重新打开已关闭的标签").Trailing("⇧⌘T").Disabled(len(d.closed) == 0).Chosen() {
		a.reopenClosed()
	}
	m.Separator()
	if menuItem(m, "向右拆分").Trailing("⌘\\").Chosen() {
		a.splitEditor(1)
	}
	if menuItem(m, "向下拆分").Chosen() {
		a.splitEditor(2)
	}
	if tb := d.cur(); tb != nil && isFileTab(tb) {
		if menuItem(m, "在侧边打开").Trailing("⌘⏎").Chosen() {
			a.openSide(tb.path, true)
		}
	}
	a.mdMenuItems(m)
	m.Separator()
	if menuItem(m, "复制所有标签路径").Disabled(len(d.tabs) == 0).Chosen() {
		var ps []string
		for _, tb := range d.tabs {
			if isFileTab(tb) {
				ps = append(ps, a.absPath(tb.path))
			}
		}
		a.copyText = strings.Join(ps, "\n")
		a.toast = "已复制 " + itoa(len(ps)) + " 个路径"
	}
}

// tabMenu 是一个标签的右键菜单。
func (a *App) tabMenu(m *ui.Menu, tb *ideTab) {
	d := a.ide
	ix := a.tabIndex(tb.id)
	if menuItem(m, "关闭").Trailing("⌘W").Chosen() {
		a.closeWsTab(tb.id)
	}
	if menuItem(m, "关闭其他").Disabled(len(d.tabs) < 2).Chosen() {
		a.closeTabsWhere(func(x *ideTab) bool { return x.id == tb.id })
	}
	if menuItem(m, "关闭右侧").Disabled(ix < 0 || ix >= len(d.tabs)-1).Chosen() {
		a.closeTabsWhere(func(x *ideTab) bool { return a.tabIndex(x.id) <= ix })
	}
	if menuItem(m, "向右拆分").Trailing("⌘\\").Chosen() {
		a.splitEditor(1)
	}
	if menuItem(m, "向下拆分").Chosen() {
		a.splitEditor(2)
	}
	if menuItem(m, "固定标签").Disabled(!tb.preview).Chosen() {
		tb.preview = false
	}
	if menuItem(m, "在侧边打开").Trailing("⌘⏎").Chosen() {
		a.openSide(tb.path, true)
	}
	if menuItem(m, "在侧边预览").Chosen() {
		a.openSide(tb.path, false)
	}
	if menuItem(m, "关闭已保存").Chosen() {
		a.closeTabsWhere(func(x *ideTab) bool { return x.dirty() })
	}
	if tb.isVirt() {
		m.Separator()
		a.virtMenu(m, tb)
		return
	}
	if !isFileTab(tb) {
		return
	}
	m.Separator()
	if menuItem(m, "引用到对话").Chosen() {
		a.dropRefs(refText(tb.path, false))
	}
	a.pathMenu(m, tb.path)
	it := menuItem(m, "在文件树中定位")
	if workspace.IsExternalPath(tb.path) {
		it.Disabled(true).Tooltip("工作区外的文件不在文件树里")
	} else if it.Chosen() {
		a.revealInTree(tb.path)
	}
	m.Separator()
	proj := a.proj
	if menuItem(m, "在新窗口中打开").Disabled(proj == "").Tooltip(map[bool]string{true: "演示模式没有真实项目目录"}[proj == ""]).Chosen() {
		a.SpawnNewWindow(proj)
	}
}

// pathMenu 是「复制路径 / 复制相对路径 / 在访达中显示」三项(标签、文件树共用)。
func (a *App) pathMenu(m *ui.Menu, rel string) {
	if menuItem(m, "复制路径").Chosen() {
		a.copyText = a.absPath(rel)
		a.toast = "已复制路径"
	}
	it := menuItem(m, "复制相对路径")
	if workspace.IsExternalPath(rel) {
		it.Disabled(true).Tooltip("已是绝对路径")
	} else if it.Chosen() {
		a.copyText = rel
		a.toast = "已复制相对路径"
	}
	dis := a.proj == ""
	if workspace.IsExternalPath(rel) {
		dis = false
	}
	if menuItem(m, "在访达中显示").Disabled(dis).Tooltip(map[bool]string{true: "演示模式没有真实目录"}[a.proj == "" && !workspace.IsExternalPath(rel)]).Chosen() {
		a.revealInFinder(a.absPath(rel))
	}
}

// revealInTree 在左栏文件树里定位文件:切到「文件」、展开各级目录并选中。
func (a *App) revealInTree(rel string) {
	if a.mode != ModeIDE {
		a.SetMode(ModeIDE)
	}
	a.nav, a.sideOpen = "files", true
	parts := strings.Split(rel, "/")
	for i := 1; i < len(parts); i++ {
		a.ide.open[strings.Join(parts[:i], "/")] = true
	}
	a.ide.sel = rel
	uxLog.Info("在文件树中定位", "path", rel)
}

// ───────── 文件树 ─────────

// treeMenu 是文件树里一个文件 / 文件夹的右键菜单。会改磁盘的动作(新建 / 重命名 / 删除)待补确认对话框,先不放。
func (a *App) treeMenu(m *ui.Menu, rel string, dir bool) {
	_, ro := a.mutable()
	into := rel // 文件夹里新建;文件上右键 = 建在同级
	if !dir {
		into = parentDir(rel)
	}
	top := !strings.Contains(rel, "/") && a.multiRoot() // 多根时顶层是工作区文件夹,只能在里面新建
	if menuItem(m, "新建文件…").Disabled(ro != "").Tooltip(ro).Chosen() {
		a.beginNew(into, false)
	}
	if menuItem(m, "新建文件夹…").Disabled(ro != "").Tooltip(ro).Chosen() {
		a.beginNew(into, true)
	}
	m.Separator()
	if top { // 多根顶层文件夹:先给「在源代码管理中查看」
		a.rootFolderItems(m, rel)
	}
	if !dir {
		a.treeFileActions(m, rel, false)
	} else {
		label := "在文件夹中搜索…"
		top := !strings.Contains(rel, "/")
		if a.multiRoot() && top {
			label = "仅在此文件夹搜索"
		}
		if menuItem(m, label).Chosen() {
			if a.multiRoot() { // 多根:第一段是文件夹名,先限定范围;更深的文件夹再叠一个包含规则
				first, _, _ := strings.Cut(rel, "/")
				a.setSearchScope(first)
				a.sr.inc, a.sr.showGlobs = "", false
				if !top {
					a.sr.inc, a.sr.showGlobs = rel+"/**", true
				}
			} else {
				a.sr.inc, a.sr.showGlobs = rel+"/**", true
			}
			a.OpenSearch()
		}
		if menuItem(m, "在此处打开终端").Disabled(a.proj == "").Tooltip(map[bool]string{true: "演示模式没有真实终端目录"}[a.proj == ""]).Chosen() {
			a.openTermAt(rel)
		}
	}
	m.Separator()
	if menuItem(m, "引用到对话").Chosen() {
		a.dropRefs(refText(rel, dir))
	}
	m.Separator()
	a.treeClipMenu(m, rel, dir)
	m.Separator()
	if !top {
		if menuItem(m, "重命名…").Trailing("F2").Disabled(ro != "").Tooltip(ro).Chosen() {
			a.beginRenameFile(rel)
		}
		if menuItem(m, "移到废纸篓…").Trailing("⌘⌫").Disabled(ro != "").Tooltip(ro).Danger().Chosen() {
			a.askDelete(rel, dir)
		}
		m.Separator()
	}
	a.pathMenu(m, rel)
	if top { // 顶层文件夹没有「移到废纸篓」,最底部是危险项「从工作区移出」
		m.Separator()
		a.rootFolderRemove(m, rel)
	}
}

// openTermAt 在项目里某个目录新开一个终端标签并显示底部终端面板。
func (a *App) openTermAt(rel string) {
	a.newTerm()
	a.ide.tpc().cwd = a.absPath(rel)
	a.ide.bottomOpen, a.ide.bottomTab = true, "term"
	uxLog.Info("在目录中打开终端", "dir", rel)
}

// treeHeadMenu 是文件树标题「···」:刷新、全部折叠、在访达中显示项目。
func (a *App) treeHeadMenu(m *ui.Menu) {
	_, ro := a.mutable()
	if menuItem(m, "新建文件…").Disabled(ro != "").Tooltip(ro).Chosen() {
		a.beginNew(a.newTargetDir(), false)
	}
	if menuItem(m, "新建文件夹…").Disabled(ro != "").Tooltip(ro).Chosen() {
		a.beginNew(a.newTargetDir(), true)
	}
	m.Separator()
	if menuItem(m, "刷新").Chosen() {
		a.ide.listCache, a.ide.filesCache = nil, nil
	}
	if menuItem(m, "全部折叠").Disabled(a.treeOpenCount() == 0).Tooltip(map[bool]string{true: "没有展开的文件夹"}[a.treeOpenCount() == 0]).Chosen() {
		a.collapseTree()
	}
	m.Separator()
	if menuItem(m, "在访达中显示项目").Disabled(a.proj == "").Tooltip(map[bool]string{true: "演示模式没有真实目录"}[a.proj == ""]).Chosen() {
		a.revealInFinder(a.proj)
	}
}

// ───────── 会话 ─────────

const noBackend = "后端暂未支持"

// sessionMenu 是会话的「···」/ 右键菜单(导航栏会话行与对话栏头部共用)。
// 重命名 / 置顶 / 归档 / 复制为 Markdown / 删除见 session_meta.go;分叉走 CmdForkSession(演示 / 真实均已接)。
func (a *App) sessionMenu(m *ui.Menu, id, title string, inHead bool) {
	fresh := id == "" || strings.HasPrefix(id, "new:") // 新会话还没有 id,没有可整理的对象
	meta := a.sessMeta().Get(id)
	if !inHead {
		if mItem(m, "play", "打开").Trailing("⏎").Disabled(id == a.curSess).Chosen() {
			a.OpenSessionChecked(id)
		}
		m.Separator()
	}
	noID := map[bool]string{true: "新会话还没有 ID,发出第一条消息后才有"}[fresh]
	if menuItem(m, "重命名…").Disabled(fresh).Tooltip(noID).Chosen() {
		a.beginRename(id)
	}
	if menuItem(m, map[bool]string{true: "取消置顶", false: "置顶"}[meta.Pinned]).Disabled(fresh).Tooltip(noID).Chosen() {
		a.pinSession(id, !meta.Pinned)
	}
	if menuItem(m, map[bool]string{true: "取消归档", false: "归档"}[meta.Archived]).Disabled(fresh).Tooltip(noID).Chosen() {
		a.archiveSession(id, !meta.Archived)
	}
	m.Separator()
	if menuItem(m, "复制会话 ID").Disabled(fresh).Tooltip(noID).Chosen() {
		a.copyOut(id, "会话 ID")
	}
	if menuItem(m, "复制标题").Disabled(title == "").Chosen() {
		a.copyOut(title, "标题")
	}
	mdOff := id != a.curSess
	if menuItem(m, "复制为 Markdown").Disabled(mdOff).Tooltip(map[bool]string{true: "只能复制已打开的会话,先点开它"}[mdOff]).Chosen() {
		a.copySessionMarkdown(id, title)
	}
	m.Separator()
	run := a.sessionRunning(id)
	if menuItem(m, "复制会话链接").Disabled(fresh).Tooltip(noID).Chosen() {
		a.copySessionLink(id)
	}
	forkWhy := map[bool]string{true: noID}[fresh]
	if forkWhy == "" && run {
		forkWhy = "会话正在运行,先停止或等它结束再分叉"
	}
	if menuItem(m, "分叉会话…").Disabled(fresh || run).Tooltip(forkWhy).Chosen() {
		a.forkSessionAsk(id)
	}
	if inHead {
		if menuItem(m, "历史会话").Trailing("⇧⌘H").Chosen() {
			a.togglePop("hist")
		}
		return
	}
	m.Separator()
	if menuItem(m, "删除会话…").Danger().Disabled(fresh || run).Tooltip(map[bool]string{true: "会话正在运行,先停止或等它结束"}[run && !fresh]).Chosen() {
		a.deleteSessionAsk(id)
	}
}

// ───────── 小工具 ─────────
