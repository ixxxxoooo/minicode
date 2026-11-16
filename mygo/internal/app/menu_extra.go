package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/projects"
)

// 右键菜单补全(设计稿 menus / drag-more 页):Markdown 标签「···」附加项、
// 多根工作区顶层文件夹的专属项、会话流步骤行右键。规则同 menu_item.go:不可用的项置灰并写明原因。

// ───────── Markdown 标签 ···(同标签条,附加) ─────────

// mdMenuItems 在标签条「···」里追加 Markdown 专属项:切换所见即所得 / 源码、只读预览状态、格式化文档。
// 当前标签不是 Markdown 时什么也不加。
func (a *App) mdMenuItems(m *ui.Menu) {
	tb := a.ide.cur()
	if tb == nil || tb.buf == nil || !isMarkdownFile(tb.path) {
		return
	}
	m.Separator()
	src := a.mdSource(tb)
	label := "切换为源码"
	if src {
		label = "切换为预览"
	}
	if menuItem(m, label).Chosen() {
		a.setMdMode(tb, !src)
	}
	autoRO := tb.md != nil && tb.md.unsafe != ""
	manualRO := tb.md != nil && tb.md.readOnly
	roOn := !src && (autoRO || manualRO)
	why := "开启后预览只浏览、不编辑;再次点击恢复"
	if autoRO {
		why = "含 " + tb.md.unsafe + ",在预览里编辑会改写它,所以自动只读;切到源码再编辑"
	}
	it := menuItem(m, "只读预览").Icon(kit.MenuIcon("eye")).Checked(roOn).Tooltip(why)
	if src {
		it = it.Disabled(true).Tooltip("源码视图下请直接编辑文本")
	} else if autoRO {
		it = it.Disabled(true)
	} else if it.Chosen() {
		a.toggleMdReadOnly(tb)
	}
	if menuItem(m, "格式化文档  ⇧⌥F").Chosen() {
		a.formatDoc()
	}
}

// ───────── 多根:顶层文件夹 ─────────

// rootRepoIdx 文件树顶层节点名 → 仓库条目序号(只认工作区成员本身,不含嵌套仓库);-1 = 不是顶层文件夹。
func (a *App) rootRepoIdx(name string) int {
	for i, e := range a.repos {
		if e.kind == "" && e.name == name {
			return i
		}
	}
	return -1
}

// rootFolderItems 是多根顶层文件夹右键菜单的专属上半段:在源代码管理中查看。
func (a *App) rootFolderItems(m *ui.Menu, name string) {
	i := a.rootRepoIdx(name)
	if i < 0 {
		return
	}
	e := a.repos[i]
	noRepo := e.checked && e.repo == nil
	scmWhy := map[bool]string{true: "这个文件夹还不是 Git 仓库(在源代码管理总览里可以初始化)"}[noRepo]
	if menuItem(m, "在源代码管理中查看").Disabled(!e.checked).Tooltip(map[bool]string{true: "正在探测这个文件夹是不是仓库"}[!e.checked]).Chosen() {
		a.setActiveRepo(i)
		a.OpenSCM()
		if scmWhy != "" {
			a.toast = scmWhy
		}
	}
}

// rootFolderRemove 是多根顶层文件夹右键菜单最底部的危险项:从工作区移出(只改工作区列表,不删磁盘文件)。
func (a *App) rootFolderRemove(m *ui.Menu, name string) {
	i := a.rootRepoIdx(name)
	if i < 0 || a.ws == nil {
		return
	}
	why := ""
	switch {
	case len(a.ws.Folders) <= 2:
		why = "工作区至少需要 2 个文件夹"
	}
	e := a.repos[i]
	if menuItem(m, "从工作区移出…").Danger().Disabled(why != "").Tooltip(why).Chosen() {
		a.confirmDlg("把「"+e.name+"」移出工作区?", "只会从工作区里去掉这个文件夹,磁盘上的文件、会话记录与 Git 仓库都不会动。之后可以在「编辑工作区」里重新添加。", "移出", true, func() {
			a.removeFolderFromWs(e.dir)
		})
	}
}

// wsFolderIdx 在使用中的工作区里按路径找文件夹序号(找不到 -1)。
func (a *App) wsFolderIdx(w projects.Workspace, dir string) int {
	for i, f := range w.Folders {
		if projects.Clean(f.Path) == projects.Clean(dir) {
			return i
		}
	}
	return -1
}

// removeFolderFromWs 把 dir 对应的文件夹从使用中的工作区移出。
func (a *App) removeFolderFromWs(dir string) {
	id := a.activeWsID()
	if id == "" {
		return
	}
	done := false
	a.toast = ""
	a.wsMutate(id, func(w *projects.Workspace) {
		if k := a.wsFolderIdx(*w, dir); k >= 0 && len(w.Folders) > 2 {
			w.Folders = append(w.Folders[:k:k], w.Folders[k+1:]...)
			done = true
		}
	})
	if done && a.toast == "" {
		uxLog.Info("从工作区移出文件夹", "dir", dir)
		a.toast = "已从工作区移出"
	}
}

// ───────── 会话流:步骤行右键 ─────────

// stepHasMenu 步骤行有没有可右键的动作(没有就不挂菜单,免得弹出空菜单)。
func stepHasMenu(sp step) bool {
	return sp.path != "" || sp.term != nil || sp.url != "" || sp.copyStr != ""
}

// stepMenu 是会话流步骤行的右键菜单(设计稿 menus「会话流里的动作」表的「右键」列):
// 读取文件 = 复制路径 · 在访达中显示;编辑 / 写入 = 同上 + 查看差异 / 在审查标签中查看;
// 运行命令 = 复制命令 · 复制输出;搜索 = 复制搜索词;抓取网页 = 复制网址。
func (a *App) stepMenu(m *ui.Menu, sp step) {
	noReal := map[bool]string{true: "演示模式没有真实目录"}[a.proj == ""]
	if sp.path != "" {
		if menuItem(m, "打开文件").Chosen() && !a.openPathRef(joinRef(sp.path, sp.line)) {
			a.toast = "文件已不存在:" + baseName(sp.path)
		}
		if sp.diffAs {
			if menuItem(m, "查看差异").Chosen() {
				a.OpenDiffTab(sp.path)
			}
			if menuItem(m, "在审查标签中查看").Chosen() {
				a.OpenReview()
			}
		}
	}
	if sp.term != nil {
		if menuItem(m, "在终端面板查看").Chosen() {
			a.viewInPanel(sp.term)
		}
	}
	if sp.url != "" {
		if menuItem(m, "在浏览器中打开").Chosen() {
			a.openURL(sp.url)
		}
	}
	m.Separator()
	switch {
	case sp.term != nil:
		if menuItem(m, "复制命令").Chosen() {
			a.copyOut(sp.term.Command, "命令")
		}
		empty := strings.TrimSpace(sp.term.Output) == ""
		if menuItem(m, "复制输出").Disabled(empty).Tooltip(map[bool]string{true: "这条命令没有输出"}[empty]).Chosen() {
			a.copyOut(sp.term.Output, "输出")
		}
	case sp.path != "":
		if menuItem(m, "复制路径").Chosen() {
			a.copyOut(sp.path, "路径")
		}
		if menuItem(m, "在访达中显示").Disabled(a.proj == "").Tooltip(noReal).Chosen() {
			a.revealInFinder(a.stepAbs(sp.path))
		}
	case sp.url != "":
		if menuItem(m, "复制网址").Chosen() {
			a.copyOut(sp.url, "网址")
		}
	case sp.copyStr != "":
		what := "搜索词"
		if sp.fam == famExec {
			what = "命令"
		}
		if menuItem(m, "复制"+what).Chosen() {
			a.copyOut(sp.copyStr, what)
		}
	}
}

// stepAbs 步骤里的路径(可能是绝对、也可能相对项目根)→ 绝对路径,供「在访达中显示」。
func (a *App) stepAbs(p string) string {
	if strings.HasPrefix(p, "/") {
		return p
	}
	return a.absPath(a.wsRel(p))
}

// askEchoMenu 回答卡片的右键菜单。
func (a *App) askEchoMenu(m *ui.Menu, q *core.QuestionRequest) {
	if menuItem(m, "复制回答").Chosen() {
		log.Info("复制回答", "id", q.ID, "题数", len(q.Questions))
		a.copyOut(askAnswersText(q), "回答")
	}
}

// askAnswersText 把「问题 → 回答」拼成纯文本,供复制。
func askAnswersText(q *core.QuestionRequest) string {
	var b strings.Builder
	for i, qu := range q.Questions {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(qu.Question)
		b.WriteString(" → ")
		ans := []string(nil)
		if i < len(q.Answers) {
			ans = q.Answers[i]
		}
		if len(ans) == 0 {
			b.WriteString("已跳过")
		} else {
			b.WriteString(strings.Join(ans, ", "))
		}
	}
	return b.String()
}
