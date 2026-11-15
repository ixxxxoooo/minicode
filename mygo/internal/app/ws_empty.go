package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 工作区起始页(没有任何标签时,设计稿 v3.4「工作区起始页与多个终端」):
// 三组左对齐列表——本轮(只列有内容的:审查 / 计划 / 子 Agent 与任务)、终端(每个已开的终端一行 + 新建)、
// 打开(文件 / 源代码管理 / 搜索 / 命令面板)。每一行都只是「跳转」,内容的家仍在原处。
// 终端行与底部面板的终端标签是同一份状态(a.ide.tps / tpSel)。
//
// @author ixxxxoooo

// termEntry 是起始页「终端」组里的一行。
type termEntry struct {
	Name    string // zsh / zsh 2 …
	Sub     string // 状态:当前 / 后台运行 / 尚未启动
	Idx     int    // 对应 a.ide.tps 的下标
	Current bool
}

// termEntries 返回已开终端的列表(真实模式)。
// 用途:起始页终端组与单元测试;不会创建终端(a.ide.tpc() 才会惰性创建)。
// 返回:没有终端或演示模式时为空。
func (a *App) termEntries() []termEntry {
	if a.proj == "" {
		return nil
	}
	var out []termEntry
	for i, tp := range a.ide.tps {
		name := shellName()
		if i > 0 {
			name += " " + itoa(i+1)
		}
		cur := a.ide.termSel == "" && a.ide.tpSel == i
		sub := "后台运行"
		switch {
		case tp.err != "":
			sub = "启动失败"
		case tp.t == nil:
			sub = "尚未启动"
		case cur:
			sub = "当前"
		}
		out = append(out, termEntry{Name: name, Sub: sub, Idx: i, Current: cur})
	}
	return out
}

// revealTerm 展开底部面板并切到第 i 个终端、把焦点给它。
// 参数:i 终端下标(越界按最后一个处理)。
func (a *App) revealTerm(i int) {
	a.ide.bottomOpen, a.ide.bottomTab, a.ide.termSel = true, "term", ""
	if n := len(a.ide.tps); n > 0 {
		a.ide.tpSel = min(max(i, 0), n-1)
	}
	a.ide.tpc().focusIt = true
	log.Info("展开终端", "index", a.ide.tpSel)
}

// openNewTerm 新建一个终端并展开面板(还没有任何终端时就是打开第一个)。
func (a *App) openNewTerm() {
	if a.proj != "" && len(a.ide.tps) > 0 {
		a.newTerm() // 新建并选中最后一个
	}
	a.revealTerm(len(a.ide.tps) - 1)
}

// wsRow 是起始页里的一行。
type wsRow struct {
	key, icon, title, sub string
	keys                  []string
	chev                  bool
	run                   func()
}

func (a *App) wsEmpty(c *ui.Context) {
	t := a.tk
	files := a.changedFiles()
	plan := a.latestPlan()
	agents, shells := a.dockTasks()

	var turn []wsRow
	if n := len(files); n > 0 {
		sub := itoa(n) + " 个文件待审查"
		if a.changesCommitted() {
			sub = itoa(n) + " 个文件 · 已提交到 Git,可回看"
		}
		turn = append(turn, wsRow{key: "review", icon: "eye", title: "审查本轮改动", sub: sub, chev: true, run: a.OpenReview})
	}
	if plan != nil {
		turn = append(turn, wsRow{key: "plan", icon: "list", title: "计划", sub: "最新一份计划", chev: true, run: func() { a.OpenWorkspaceTab(vPlan, plan.ID) }})
	}
	if n := len(agents) + len(shells); n > 0 {
		turn = append(turn, wsRow{key: "tasks", icon: "activity", title: "子 Agent 与任务", sub: itoa(n) + " 个运行中", chev: true, run: func() {
			if len(agents) > 0 {
				a.openTaskTarget(agents[0])
			} else {
				a.openTaskTarget(shells[0])
			}
		}})
	}

	var terms []wsRow
	ents := a.termEntries()
	for _, e := range ents {
		idx := e.Idx
		terms = append(terms, wsRow{key: "term" + itoa(idx), icon: "terminal", title: e.Name, sub: e.Sub, chev: true, run: func() { a.revealTerm(idx) }})
	}
	if len(ents) > 0 {
		terms = append(terms, wsRow{key: "termnew", icon: "plus", title: "新建终端", sub: "再开一个 shell", run: a.openNewTerm})
	} else {
		terms = append(terms, wsRow{key: "termnew", icon: "terminal", title: "打开终端", sub: "在项目目录里打开 shell", keys: []string{"⌘", "J"}, run: a.openNewTerm})
	}
	open := []wsRow{
		{key: "files", icon: "folder", title: "文件", sub: "在导航栏浏览项目文件", chev: true, run: func() { a.nav, a.sideOpen = "files", true }},
		{key: "scm", icon: "branch", title: "源代码管理", sub: "暂存、提交、图谱", keys: []string{"⌃", "⇧", "G"}, run: a.OpenSCM},
		{key: "search", icon: "search", title: "在项目中搜索", keys: []string{"⇧", "⌘", "F"}, run: a.OpenSearch},
		{key: "quick", icon: "cmd", title: "命令面板", sub: "快速打开文件与命令", keys: []string{"⌘", "K"}, run: a.openPalette},
	}

	group := func(title string, rows []wsRow, none string) {
		ui.Column(c).Gap(5).AlignItems(ui.Stretch).Children(func() {
			ui.Text(c, title).FontSize(10.5).FontWeight(500).LetterSpacing(.3).TextColor(t.Tx3).Margin(0, 0, 0, 4)
			ui.Column(c).Radius(t.RMd()).Background(t.Raised).Border(1, t.Bd1).AlignItems(ui.Stretch).Children(func() {
				if len(rows) == 0 {
					ui.Text(c, none).FontSize(theme.FsCap).TextColor(t.Tx3).Padding(10, 12)
					return
				}
				for i, r := range rows {
					if i > 0 {
						ui.Box(c).Height(1).Background(t.Bd1).Shrink(0)
					}
					b := ui.ButtonBase(c).Key("wse-" + r.key).Height(36).PaddingX(12).Gap(10).AlignItems(ui.Center).
						Justify(ui.Start).Cursor(ui.CursorPointer).Transition(kit.Fade)
					if b.Hovered() {
						b.Background(t.Fill1)
					}
					b.OnClick(r.run)
					b.Children(func() {
						kit.Ic(c, r.icon, kit.S16).TextColor(t.Tx2).Shrink(0)
						ui.Text(c, r.title).FontWeight(500).TextColor(t.Tx1).SingleLine().Shrink(0)
						ui.Text(c, r.sub).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Grow(1).MinWidth(0)
						if len(r.keys) > 0 {
							kit.Kbd(c, r.keys...)
						} else if r.chev {
							kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4).Shrink(0)
						}
					})
				}
			})
		})
	}

	termTitle := "终端"
	if n := len(ents); n > 1 {
		termTitle += " · " + itoa(n) + " 个"
	}
	ui.Scroll(c).Key("ws-empty").Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(22, 20).Gap(16).AlignItems(ui.Stretch).MaxWidth(560).MarginX(ui.Auto).Children(func() {
			ui.Column(c).Gap(3).Children(func() {
				ui.Text(c, "工作区").FontSize(theme.FsTitle).FontWeight(600).TextColor(t.Tx1)
				ui.Text(c, "改动、计划、任务、文件和终端都从这里进入;打开的内容会成为标签。").TextColor(t.Tx3).FontSize(theme.FsCap)
			})
			group("本轮", turn, "本轮还没有改动、计划或任务。")
			group(termTitle, terms, "")
			group("打开", open, "")
		})
	})
}
