package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// wsEmpty 是工作区没有任何标签时的「起始页」。
// 业界做法(Cursor / Codex 侧栏、VS Code 欢迎页):空面板不留白,而是列出这个面板能放什么,一步直达。
// 每一项只是「跳转」,内容的家仍在原处:改动在审查标签、计划 / 子 Agent / 任务是虚拟标签、
// 文件 / 搜索 / 源代码管理在导航栏、终端在底部面板。
func (a *App) wsEmpty(c *ui.Context) {
	t := a.tk
	type tile struct {
		key, icon, title, sub string
		off                   bool
		run                   func()
	}
	files := a.changedFiles()
	plan := a.latestPlan()
	agents, shells := a.dockTasks()
	var tiles []tile

	sub := "还没有改动"
	if n := len(files); n > 0 {
		sub = itoa(n) + " 个文件待审查"
	}
	tiles = append(tiles, tile{"review", "eye", "审查本轮改动", sub, len(files) == 0, a.OpenReview})

	psub := "还没有计划"
	if plan != nil {
		psub = "最新一份计划"
	}
	tiles = append(tiles, tile{"plan", "list", "计划", psub, plan == nil, func() {
		id := ""
		if plan != nil {
			id = plan.ID
		}
		a.OpenWorkspaceTab(vPlan, id)
	}})

	tsub := "没有运行中的任务"
	if n := len(agents) + len(shells); n > 0 {
		tsub = itoa(n) + " 个子 Agent / 后台命令"
	}
	tiles = append(tiles, tile{"tasks", "activity", "子 Agent 与任务", tsub, len(agents)+len(shells) == 0, func() {
		if len(agents) > 0 {
			a.openTaskTarget(agents[0])
		} else if len(shells) > 0 {
			a.openTaskTarget(shells[0])
		}
	}})
	tiles = append(tiles,
		tile{"term", "terminal", "终端", "在项目目录里打开 shell  ⌘J", false, func() {
			a.ide.bottomOpen, a.ide.bottomTab = true, "term"
			if a.ide.tpc() != nil {
				a.ide.tpc().focusIt = true
			}
		}},
		tile{"files", "folder", "文件", "在导航栏浏览项目文件", false, func() { a.nav, a.sideOpen = "files", true }},
		tile{"scm", "branch", "源代码管理", "暂存、提交、图谱  ⌃⇧G", false, a.OpenSCM},
		tile{"search", "search", "在项目中搜索", "⇧⌘F", false, a.OpenSearch},
		tile{"quick", "cmd", "命令面板", "快速打开文件与命令  ⌘K", false, a.openPalette},
	)

	ui.Scroll(c).Key("ws-empty").Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(22, 20).Gap(14).AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).Gap(3).Children(func() {
				ui.Text(c, "工作区").FontSize(theme.FsTitle).FontWeight(600).TextColor(t.Tx1)
				ui.Text(c, "改动、计划、任务、文件和终端都从这里进入;打开的内容会成为标签。").TextColor(t.Tx3).FontSize(theme.FsCap)
			})
			for i := 0; i < len(tiles); i += 2 {
				row := tiles[i:min(i+2, len(tiles))]
				ui.Row(c).Gap(8).AlignItems(ui.Stretch).Children(func() {
					for _, tl := range row {
						b := ui.ButtonBase(c).Key("wse-"+tl.key).Grow(1).Basis(0).MinWidth(0).Padding(10, 12).Gap(10).Radius(t.RMd()).
							Border(1, t.Bd1).AlignItems(ui.Center).Cursor(ui.CursorPointer).Transition(kit.Fade)
						if tl.off {
							b.Cursor(ui.CursorDefault)
						} else if b.Hovered() {
							b.Background(t.Fill1).BorderColor(t.Bd2)
						}
						if !tl.off {
							b.OnClick(tl.run)
						}
						b.Children(func() {
							col := t.Tx2
							if tl.off {
								col = t.Tx4
							}
							kit.Ic(c, tl.icon, kit.S16).TextColor(col).Shrink(0)
							ui.Column(c).Gap(1).MinWidth(0).Children(func() {
								ui.Text(c, tl.title).FontWeight(500).TextColor(map[bool]ui.Color{true: t.Tx3, false: t.Tx1}[tl.off]).SingleLine()
								ui.Text(c, tl.sub).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
							})
						})
					}
					if len(row) == 1 {
						ui.Spacer(c).Grow(1).Basis(0)
					}
				})
			}
		})
	})
}
