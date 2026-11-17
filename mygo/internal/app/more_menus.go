package app

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// 各处「···」与右键菜单的补全(设计稿 menus、drag-more):SCM 标题、差异标签、终端标签条、
// 问题 / 输出行、虚拟标签、自定义详情。规则:不可用的项置灰并写明原因,不隐藏。

// scmHeadMenu 是 SCM 标题「···」:拉取 / 推送 / 同步 / 储藏 / 刷新。
func (a *App) scmHeadMenu(m *ui.Menu) {
	s := &a.sc
	if menuItem(m, "拉取").Chosen() {
		a.scmSync(true, false)
	}
	if menuItem(m, "推送").Chosen() {
		a.scmSync(false, true)
	}
	if menuItem(m, "同步").Chosen() {
		a.scmSync(true, true)
	}
	m.Separator()
	none := len(s.changes) == 0
	if menuItem(m, "储藏所有更改").Disabled(none).Tooltip(map[bool]string{true: "没有可储藏的更改"}[none]).Chosen() {
		a.scmStashPush()
	}
	noStash := len(s.stashes) == 0
	if menuItem(m, "弹出最近的储藏").Disabled(noStash).Tooltip(map[bool]string{true: "没有储藏"}[noStash]).Chosen() {
		a.scmStashApply(0, true)
	}
	m.Separator()
	tree := a.scmViewMode() == "tree"
	lbl := "切换为树形视图"
	if tree {
		lbl = "切换为列表视图"
	}
	if menuItem(m, lbl).Trailing(map[bool]string{true: "✓"}[tree]).Chosen() {
		a.toggleScmViewMode()
	}
	m.Separator()
	if menuItem(m, "刷新").Chosen() {
		s.last = time.Time{}
	}
}

// diffMoreMenu 是差异标签头部「···」。
func (a *App) diffMoreMenu(m *ui.Menu, tb *ideTab) {
	wrap := "自动换行"
	if a.prefs.Bool(pEdWrap, false) {
		wrap = "关闭自动换行"
	}
	if menuItem(m, wrap).Trailing("⌥Z").Chosen() {
		a.toggleWrap()
	}
	narrow := a.wsNarrow()
	if menuItem(m, "切换左右 / 统一").Disabled(narrow).Tooltip(map[bool]string{true: "工作区太窄,已强制统一视图"}[narrow]).Chosen() {
		if a.diffLayout(tb, false) == 0 { // 左右 → 统一
			tb.lay = 2
		} else {
			tb.lay = 1
		}
		a.prefs.Set("scm.diff", int(tb.lay)-1)
		a.diffPref = int(tb.lay) - 1
	}
	m.Separator()
	gone := tb.dk != "" && tb.dv != nil && tb.dv.New.String() == "" && tb.dv.Old.String() != ""
	if menuItem(m, "打开文件").Disabled(gone).Tooltip(map[bool]string{true: "文件已被删除"}[gone]).Chosen() {
		a.jumpTo(tb.path, 0, 0, 0)
	}
	if menuItem(m, "复制路径").Chosen() {
		a.copyOut(a.absPath(tb.path), "路径")
	}
	if menuItem(m, "显示文件历史").Chosen() {
		a.ide.sel = tb.path
		a.ide.sections["时间线"] = true
		a.nav, a.sideOpen = "files", true
	}
	m.Separator()
	ign := "忽略空白差异"
	if a.diffIgnoreWS() {
		ign = "显示空白差异"
	}
	if menuItem(m, ign).Chosen() {
		a.toggleDiffIgnoreWS(tb)
	}
}

// termMoreMenu 是终端标签条「···」:清屏 / 查找 / 新建 / 关闭 / 设置。
func (a *App) termMoreMenu(m *ui.Menu) {
	tp := a.ide.tpc()
	live := tp != nil && tp.t != nil
	why := map[bool]string{true: "终端还没启动"}[!live]
	if menuItem(m, "清屏").Trailing("⌘K").Disabled(!live).Tooltip(why).Chosen() {
		tp.t.ClearCmdBlocks()
		tp.t.SendText("\x0c")
	}
	if menuItem(m, "查找…").Trailing("⌘F").Disabled(!live).Tooltip(why).Chosen() {
		a.openTermFind()
	}
	m.Separator()
	if menuItem(m, "重命名…").Disabled(a.ide.tpSel < 0 || a.ide.tpSel >= len(a.ide.tps)).Chosen() {
		a.beginTermRename(a.ide.tpSel)
	}
	if menuItem(m, "新建终端").Disabled(a.proj == "").Tooltip(map[bool]string{true: "演示模式没有真实终端"}[a.proj == ""]).Chosen() {
		a.newTerm()
	}
	if menuItem(m, "关闭终端").Disabled(len(a.ide.tps) < 2).Tooltip(map[bool]string{true: "至少保留一个终端"}[len(a.ide.tps) < 2]).Chosen() {
		a.askCloseTermTab(a.ide.tpSel)
	}
	if menuItem(m, "关闭其他").Disabled(len(a.ide.tps) < 2).Chosen() {
		a.closeOtherTermTabs(a.ide.tpSel)
	}
	cur := a.prefs.Int(pTermTabs, termTabAuto)
	m.Separator()
	if menuItem(m, "标签排列:横向").Trailing(map[bool]string{true: "✓"}[cur == 0]).Chosen() {
		a.setTermTabsMode(termTabHorz)
	}
	if menuItem(m, "标签排列:竖向").Trailing(map[bool]string{true: "✓"}[cur == 1]).Chosen() {
		a.setTermTabsMode(termTabVert)
	}
	if menuItem(m, "标签排列:自动").Trailing(map[bool]string{true: "✓"}[cur == 2]).Chosen() {
		a.setTermTabsMode(termTabAuto)
	}
	m.Separator()
	if menuItem(m, "终端设置…").Chosen() {
		a.openSettings("terminal")
	}
}

// problemMenu 是问题行的右键菜单;all 给出「复制全部」的文本。
func (a *App) problemMenu(m *ui.Menu, file string, line int, msg string, all func() string) {
	full := a.resolveFile(file)
	if menuItem(m, "转到位置").Disabled(full == "").Tooltip(map[bool]string{true: "找不到这个文件"}[full == ""]).Chosen() {
		a.jumpTo(full, max(line-1, 0), 0, 0)
	}
	if menuItem(m, "让 Claude 修复").Chosen() {
		a.draft = strings.TrimSpace(a.draft + "\n请修复这个问题:" + file + ":" + itoa(line) + " " + msg)
		if a.mode == ModeIDE {
			a.rightOn = true
		}
		a.toast = "已把问题放进输入框"
	}
	m.Separator()
	if menuItem(m, "复制消息").Chosen() {
		a.copyOut(msg, "消息")
	}
	if menuItem(m, "复制全部").Chosen() {
		a.copyOut(all(), "全部问题")
	}
}

// resolveFile 把问题里的文件名(可能只有文件名)对到项目里的相对路径;找不到返回空。
func (a *App) resolveFile(f string) string {
	if _, err := a.ide.fs.Read(f); err == nil {
		return f
	}
	for _, p := range a.ide.allFiles() {
		if path.Base(p) == path.Base(f) {
			return p
		}
	}
	return ""
}

// outputMenu 是输出面板里一行的右键菜单。
func (a *App) outputMenu(m *ui.Menu, line string, lines []string) {
	if menuItem(m, "复制此行").Chosen() {
		raw := line
		if i := strings.Index(line, "  "); i >= 0 {
			raw = strings.TrimSpace(line[i+2:])
		}
		a.copyOut(raw, "此行")
	}
	if menuItem(m, "复制全部").Chosen() {
		a.copyOut(strings.Join(lines, "\n"), "全部输出")
	}
	m.Separator()
	app := a.out.ch == outApp
	if menuItem(m, "清空输出").Disabled(app).Tooltip(map[bool]string{true: "应用日志来自日志文件,不能清空"}[app]).Danger().Chosen() {
		a.out.lines[a.out.ch] = nil
	}
}

// virtText 虚拟标签的文本内容(计划 / 子 Agent / 任务);没有可复制文本的返回空。
func (a *App) virtText(tb *ideTab) string {
	switch tb.virt {
	case vPlan:
		if p := a.latestPlan(); p != nil {
			return p.Markdown
		}
	case vSub:
		if s := a.findSub(tb.ref); s != nil {
			return strings.TrimSpace(s.Description + "\n" + s.Summary)
		}
	case vTask:
		if k := a.findTask(tb.ref); k != nil {
			return strings.TrimSpace(k.Description + "\n" + k.Summary)
		}
	}
	return ""
}

// virtMenu 是虚拟标签(计划 / 审查 / 任务…)右键菜单里关闭之外的几项。
func (a *App) virtMenu(m *ui.Menu, tb *ideTab) {
	txt := a.virtText(tb)
	why := map[bool]string{true: "这个标签没有可复制的文本"}[txt == ""]
	if menuItem(m, "复制内容").Disabled(txt == "").Tooltip(why).Chosen() {
		a.copyOut(txt, "内容")
	}
	virtWin := menuItem(m, "在新窗口中打开")
	if why2 := a.virtNewWindowWhy(tb.virt); why2 != "" {
		virtWin.Disabled(true).Tooltip(why2)
	} else if virtWin.Chosen() {
		a.SpawnVirtWindow(tb)
	}
	m.Separator()
	if menuItem(m, "引用到对话").Disabled(txt == "").Tooltip(why).Chosen() {
		q := txt
		if r := []rune(q); len(r) > 1500 {
			q = string(r[:1500]) + "…"
		}
		a.draft = strings.TrimSpace(a.draft + "\n> " + strings.ReplaceAll(q, "\n", "\n> "))
		if a.mode == ModeIDE {
			a.rightOn = true
		}
		a.toast = "已引用到对话"
	}
}

// cuMoreMenu 是自定义详情头部「···」:复制名称 / 访达 / 删除。
func (a *App) cuMoreMenu(m *ui.Menu, st *customize.Store, it *customize.Item) {
	if menuItem(m, "复制名称").Chosen() {
		a.copyOut(it.Name, "名称")
	}
	real := it.Path != "" && !a.cuDemo()
	why := "这一项没有对应的文件(或在演示数据里)"
	canEdit := real && a.cuOpenOK(it)
	if menuItem(m, "在访达中显示文件").Disabled(!real).Tooltip(why).Chosen() {
		p := it.Path
		if strings.HasPrefix(p, "~") {
			if h, err := os.UserHomeDir(); err == nil {
				p = filepath.Join(h, strings.TrimPrefix(p, "~/"))
			}
		} else if a.proj != "" && !filepath.IsAbs(p) {
			p = filepath.Join(a.proj, p)
		}
		if a.cuCat() == customize.Skills {
			p = filepath.Join(p, "SKILL.md")
		}
		a.revealInFinder(p)
	}
	editWhy := why
	if !canEdit {
		editWhy = "没有可打开的源文件"
	}
	if menuItem(m, "用编辑器打开").Disabled(!canEdit).Tooltip(editWhy).Chosen() {
		a.cuOpenPath(it.Path)
	}
	m.Separator()
	verb := "删除"
	if a.cuCat() == customize.Plugins {
		verb = "卸载"
	}
	if menuItem(m, verb+"…").Danger().Chosen() {
		a.cuDeleteAsk(st, it, verb)
	}
}

var _ = vcs.StatusConflict
