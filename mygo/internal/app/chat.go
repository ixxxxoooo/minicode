package app

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// chatPane 是对话栏:标题行 + 会话流 + 任务坞 + 输入框。
func (a *App) chatPane(c *ui.Context, gap float32) {
	t := a.tk
	p := a.pane(c).Key("chat").Transition(morph)
	if a.mode == ModeIDE {
		// IDE 形态:对话栏停靠在右侧,固定宽度
		p.Width(theme.WDock).Shrink(0).Margin(0, 0, 0, gap)
	} else {
		p.Grow(1).Basis(0)
	}
	p.Children(func() {
		a.chatHead(c)
		// 会话流:停在底部时随新内容跟随,用户上滑后不再抢滚动位置
		if a.flowSt.Y >= a.flowSt.MaxY {
			a.flowSt.Y = math.MaxFloat32
		}
		ui.Scroll(c).Key("flow").TrackScroll(&a.flowSt).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).AlignItems(ui.Center).Children(func() {
				a.flowCol(c, func() { a.flow(c) })
			})
		})
		ui.Column(c).Padding(4, 0, 10, 0).Shrink(0).AlignItems(ui.Center).Children(func() {
			a.flowCol(c, func() {
				a.taskDock(c)
				a.composer(c)
			})
		})
		_ = t
	})
}

func (a *App) chatHead(c *ui.Context) {
	ui.Row(c).Height(theme.HTab).Padding(0, 8, 0, 14).Gap(6).Shrink(0).AlignItems(ui.Center).Children(func() {
		ui.Text(c, a.st.Title).FontWeight(600).SingleLine().Shrink(1)
		kit.Chip(c, "main", kit.ChipOpt{Icon: "branch"})
		ui.Spacer(c).Grow(1)
		kit.Ib(c, "history", kit.IbOpt{Small: true, Tip: "历史会话"})
		kit.Ib(c, "more", kit.IbOpt{Small: true})
	})
}

// taskDock 是输入框上方的任务坞:每类一个胶囊。
func (a *App) taskDock(c *ui.Context) {
	agents, shells := 0, 0
	for _, tk := range a.st.Tasks {
		if !tk.Visible() {
			continue
		}
		switch tk.Kind {
		case core.TaskAgent:
			agents++
		case core.TaskShell:
			if tk.Status == core.TaskRunning {
				shells++
			}
		}
	}
	if agents == 0 && shells == 0 {
		return
	}
	ui.Row(c).Gap(4).Margin(0, 0, 6, 0).Children(func() {
		if agents > 0 {
			kit.Chip(c, "子代理 "+itoa(agents), kit.ChipOpt{Icon: "bot", Line: true})
		}
		if shells > 0 {
			kit.Chip(c, "后台命令 "+itoa(shells), kit.ChipOpt{Icon: "terminal", Line: true})
		}
	})
}
