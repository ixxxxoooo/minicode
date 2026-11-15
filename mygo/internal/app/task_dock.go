package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// dockTasks 返回运行中的子 Agent 与后台 shell 任务(供工作区概览等)。
func (a *App) dockTasks() (agents, shells []*core.BackgroundTask) {
	g := core.GroupTasks(a.st.Tasks)
	return g[core.DockAgent], g[core.DockShell]
}

// taskDock 是输入框上方的任务坞(设计稿 .tdock)。
func (a *App) taskDock(c *ui.Context) {
	groups := core.GroupTasks(a.st.Tasks)
	if len(groups) == 0 {
		return
	}
	narrow := a.chatTier() == tierNarrow
	ui.Row(c).Gap(4).Margin(0, 0, 6, 0).Children(func() {
		if narrow {
			n := 0
			for _, list := range groups {
				if len(list) > 0 {
					n += len(list)
				}
			}
			a.taskDockChip(c, "all", fmt.Sprintf("任务 %d", n), "activity", groups)
			return
		}
		order := []core.TaskDockGroup{core.DockAgent, core.DockShell, core.DockMonitor, core.DockWorkflow, core.DockOther, core.DockFinished}
		for _, g := range order {
			list := groups[g]
			if len(list) == 0 {
				continue
			}
			lbl := fmt.Sprintf("%s %d", g.Label(), len(list))
			a.taskDockChip(c, g.Key(), lbl, g.Icon(), map[core.TaskDockGroup][]*core.BackgroundTask{g: list})
		}
	})
}

func (a *App) taskDockChip(c *ui.Context, key, text, icon string, groups map[core.TaskDockGroup][]*core.BackgroundTask) {
	ch := kit.Chip(c, text, kit.ChipOpt{Icon: icon, Line: true, On: a.popup == "tpop:"+key})
	ch.OnClick(func() { a.togglePop("tpop:" + key) })
	a.taskDockPopup(c, key, ch, groups)
}

func (a *App) taskDockPopup(c *ui.Context, key string, anchor ui.Element, groups map[core.TaskDockGroup][]*core.BackgroundTask) {
	t := a.tk
	a.popOpen("tpop:"+key, func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 320).AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(0, 0, 6, 0)
			draw := func(cap string, list []*core.BackgroundTask) {
				if len(list) == 0 {
					return
				}
				kit.Cap(c, cap)
				for _, tk := range list {
					a.taskDockRow(c, tk)
				}
			}
			if key == "all" {
				for _, g := range []core.TaskDockGroup{core.DockAgent, core.DockShell, core.DockMonitor, core.DockWorkflow, core.DockOther} {
					draw(g.Label(), groups[g])
				}
				draw("已结束", groups[core.DockFinished])
				return
			}
			for g, list := range groups {
				draw(g.Label(), list)
			}
			_ = t
		})
	})
}

func (a *App) taskDockRow(c *ui.Context, tk *core.BackgroundTask) {
	t := a.tk
	stale := tk.Status == core.TaskRunning && tk.IdleMs(a.st.Now()) >= core.StallMs
	r := kit.RowEl(c, false).Key("tp-"+tk.TaskID).MinHeight(36).Padding(5, 8).Gap(8).AlignItems(ui.Start)
	r.OnClick(func() {
		a.popup = ""
		a.openTaskTarget(tk)
	})
	r.Children(func() {
		ui.Row(c).Margin(2, 0, 0, 0).Shrink(0).Children(func() {
			switch {
			case tk.Status.Finished():
				kit.Ic(c, "check", kit.S12).TextColor(t.Ok)
			case stale:
				kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
			case tk.Status == core.TaskRunning:
				kit.Spin(c, t.ToolAgent)
			default:
				kit.Ic(c, core.DockGroupOf(tk).Icon(), kit.S12).TextColor(t.Tx3)
			}
		})
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Text(c, tk.Description).TextColor(t.Tx1).SingleLine().Grow(1)
				kit.Tag(c, core.DockGroupOf(tk).Label(), kit.KAc)
			})
			sum := tk.Summary
			if stale && tk.Status == core.TaskRunning {
				sum = fmt.Sprintf("已 %s 没有新进展,可能卡住了", core.FormatElapsed(tk.IdleMs(a.st.Now())))
			}
			if sum == "" && tk.LastTool != "" {
				sum = tk.LastTool
			}
			if sum != "" {
				ui.Text(c, sum).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
			}
			if tk.RunID != "" || tk.ProcessID > 0 {
				meta := ""
				if tk.RunID != "" {
					meta = "run " + shortID(tk.RunID)
				}
				if tk.ProcessID > 0 {
					if meta != "" {
						meta += " · "
					}
					meta += fmt.Sprintf("pid %d", tk.ProcessID)
				}
				ui.Text(c, meta).FontSize(theme.FsMicro).TextColor(t.Tx4)
			}
		})
		ui.Text(c, core.FormatElapsed(tk.ElapsedMs(a.st.Now()))).FontSize(theme.FsMicro).TextColor(t.Tx4).FontFeatures("tnum")
		if tk.Status == core.TaskRunning {
			kit.Ib(c, "stop", kit.IbOpt{Small: true, Tip: "停止"}).OnClick(func() {
				a.send(core.CmdStopTask{TaskID: tk.TaskID})
			})
		}
	})
	ui.Row(c).Padding(4, 8, 0, 36).Gap(4).Children(func() {
		if tk.Kind == core.TaskShell || tk.Output != "" || tk.OutputFile != "" {
			kit.Btn(c, "在终端面板中打开", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "terminal"}).OnClick(func() {
				a.viewTaskInPanel(tk)
			})
		}
		if tk.Status == core.TaskRunning || tk.Status == core.TaskStopped || tk.Status == core.TaskFailed {
			dis, reason := a.taskRestartDisabled()
			opt := kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "refresh", Disabled: dis, Tip: reason}
			btn := kit.Btn(c, "重启", opt)
			if !dis {
				btn.OnClick(func() { a.send(core.CmdRestartTask{TaskID: tk.TaskID}) })
			}
		}
	})
}

func shortID(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8]
}

// viewTaskInPanel 在底部终端面板开只读命令标签查看任务输出。
func (a *App) viewTaskInPanel(tk *core.BackgroundTask) {
	title := tk.Description
	if title == "" {
		title = "后台任务"
	}
	id := "cmd:" + tk.TaskID
	out := strings.TrimRight(tk.Output, "\n")
	found := false
	for _, v := range a.ide.cmds {
		if v.id == id {
			found = true
			break
		}
	}
	if !found {
		a.ide.cmds = append(a.ide.cmds, cmdView{id: id, title: title, output: out})
	} else {
		for i := range a.ide.cmds {
			if a.ide.cmds[i].id == id {
				a.ide.cmds[i].output = out
				break
			}
		}
	}
	a.ide.termSel = id
	a.ide.bottomOpen, a.ide.bottomTab = true, "term"
	a.screen = ScreenWork
}
