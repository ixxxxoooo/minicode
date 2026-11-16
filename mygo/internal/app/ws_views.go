package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// virtView 画虚拟标签的内容。
func (a *App) virtView(c *ui.Context, tb *ideTab) {
	switch tb.virt {
	case vReview:
		a.reviewView(c)
	case vPlan:
		a.planView(c)
	case vSub:
		a.subView(c, tb.ref)
	case vTask:
		a.taskView(c, tb.ref)
	case vConflict:
		a.conflictView(c, tb)
	case vGraph:
		a.graphView(c)
	}
}

// wsHead 是虚拟标签顶部的标题条(36 高):图标 + 标题 + 状态,右侧是操作。
func (a *App) wsHead(c *ui.Context, icon string, iconCol ui.Color, title string, left, right func()) {
	t := a.tk
	ui.Row(c).Height(36).Shrink(0).PaddingX(12).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		kit.Ic(c, icon, kit.S12).TextColor(iconCol)
		ui.Text(c, title).FontWeight(700).SingleLine().Shrink(1)
		if left != nil {
			left()
		}
		ui.Spacer(c).Grow(1)
		if right != nil {
			right()
		}
	})
}

// ───────── 计划 ─────────

// planView 是计划标签:头部固定(标题 + 状态 + Build),下面是执行进度与计划全文。
func (a *App) planView(c *ui.Context) {
	t := a.tk
	p := a.latestPlan()
	if p == nil {
		ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "还没有计划").TextColor(t.Tx3)
		})
		return
	}
	a.wsHead(c, "list", t.Warn, planTitle(p.Markdown), func() {
		switch {
		case p.Streaming:
			kit.Tag(c, "生成中…")
		case p.Approved == nil:
			kit.Tag(c, "待批准", kit.KWarn)
		case *p.Approved:
			kit.Tag(c, "已批准", kit.KOk)
		default:
			kit.Tag(c, "已拒绝")
		}
	}, func() {
		if !p.Streaming && p.Approved == nil {
			resolve := func(ok bool, m core.PermissionMode) {
				mm := m
				a.send(core.CmdResolvePlan{ID: p.ID, Approve: ok, Mode: &mm})
			}
			kit.Btn(c, "继续规划", kit.BtnOpt{Variant: kit.BtnGhost, Small: true}).OnClick(func() { resolve(false, a.st.Mode) })
			kit.Btn(c, "自动模式", kit.BtnOpt{Small: true}).OnClick(func() { resolve(true, core.ModeAuto) })
			kit.Btn(c, "开始实施", kit.BtnOpt{Variant: kit.BtnPri, Small: true, Icon: "play"}).OnClick(func() { resolve(true, core.ModeDefault) })
		}
	})
	kit.VScroll(c).Key("planview").Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(14, 18, 20, 18).Gap(12).MaxWidth(720).Children(func() {
			// 执行进度:最新一份待办
			if td := a.latestTodos(); td != nil {
				done := 0
				for _, it := range td.Items {
					if it.Status == core.TodoCompleted {
						done++
					}
				}
				ui.Column(c).Gap(6).Padding(10, 12).Background(t.Sunken).Radius(t.RMd()).Border(1, t.Bd1).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "执行进度").FontWeight(600)
						ui.Text(c, itoa(done)+" / "+itoa(len(td.Items))).TextColor(t.Tx3).FontFeatures("tnum")
						ui.Spacer(c).Grow(1)
						ui.Column(c).Width(120).Children(func() { kit.Meter(c, float32(done)/float32(max(1, len(td.Items)))*100) })
					})
					a.todoBody(c, td)
				})
			}
			if strings.TrimSpace(p.Markdown) != "" {
				a.planBody(c, p.ID, p.Markdown, p.Streaming)
			}
		})
	})
}

// latestTodos 时间线里最新的一份待办清单。
func (a *App) latestTodos() *core.Todos {
	for i := len(a.st.Items) - 1; i >= 0; i-- {
		if td, ok := a.st.Items[i].(*core.Todos); ok {
			return td
		}
	}
	return nil
}

// ───────── 子 Agent ─────────

// subView 是子 Agent 标签:任务、完整过程、结果(完成后也写在会话流的卡片上)。
func (a *App) subView(c *ui.Context, id string) {
	t := a.tk
	s := a.findSub(id)
	if s == nil {
		ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "这个子 Agent 已不在当前会话里").TextColor(t.Tx3)
		})
		return
	}
	run := s.Active()
	a.wsHead(c, "bot", t.ToolAgent, s.Description, func() {
		if run {
			kit.Tag(c, "运行中", kit.KInfo)
		} else {
			kit.Tag(c, "已完成", kit.KOk)
		}
	}, func() {
		el, _ := s.ElapsedMs(a.st.Now())
		meta := core.FormatElapsed(el) + " · " + itoa(s.Tools()) + " 个工具"
		if s.Tokens > 0 {
			meta += " · " + fmtTok(s.Tokens) + " token"
		}
		ui.Text(c, meta).TextColor(t.Tx3).FontSize(theme.FsCap).FontFeatures("tnum").SingleLine()
		if run {
			kit.Ib(c, "stop", kit.IbOpt{Small: true, Tip: "停止"}).OnClick(func() {
				if s.TaskID != "" {
					a.send(core.CmdStopTask{TaskID: s.TaskID})
				}
			})
		}
	})
	st := &a.subSt
	if run {
		st.Y = 1e9
	}
	kit.VScroll(c).Key("subview-" + id).TrackScroll(st).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(14, 18, 20, 18).Gap(10).MaxWidth(720).Children(func() {
			wsSection(c, t, "任务")
			ui.Column(c).Padding(8, 12).Background(t.Sunken).Radius(t.RMd()).Border(1, t.Bd1).Children(func() {
				ui.Text(c, s.AgentType+":"+s.Description).TextColor(t.Tx1).LineHeight(1.55)
			})
			wsSection(c, t, "过程")
			ui.Column(c).Gap(1).Children(func() {
				n := 0
				for _, ch := range s.Children {
					if tc, ok := ch.(*core.ToolCall); ok {
						a.stepRow(c, a.presentTool(tc), nil)
						n++
					}
				}
				if n == 0 {
					ui.Text(c, "还没有工具调用").TextColor(t.Tx4)
				}
			})
			wsSection(c, t, "结果(完成后写在这里,并回到主会话的子 Agent 卡片上)")
			ui.Column(c).Padding(8, 12).Background(t.Sunken).Radius(t.RMd()).Border(1, t.Bd1).Children(func() {
				switch {
				case strings.TrimSpace(s.Summary) != "":
					ui.Text(c, s.Summary).LineHeight(1.55)
				case run:
					ui.Text(c, "尚未完成 · 点上方「停止」可以随时终止,已读到的内容会保留。").TextColor(t.Tx3)
				default:
					ui.Text(c, "没有文字结果").TextColor(t.Tx3)
				}
			})
			ui.Row(c).Gap(6).Children(func() {
				kit.Btn(c, "引用到对话", kit.BtnOpt{Small: true, Icon: "quote"}).OnClick(func() {
					a.draft = strings.TrimSpace(a.draft + "\n> 子 Agent「" + s.Description + "」:" + s.Summary)
				})
				kit.Btn(c, "复制全部", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "copy"}).OnClick(func() {
					c.WriteClipboard(s.Description + "\n" + s.Summary)
				})
			})
		})
	})
	a.taskScrollBtn(c)
}

// wsSection 是工作区内容里的小标题。
func wsSection(c *ui.Context, t *theme.Tokens, s string) {
	ui.Text(c, s).FontSize(theme.FsCap).FontWeight(600).TextColor(t.Tx3).Margin(4, 0, 0, 0)
}

// ───────── 后台任务输出 ─────────

// taskView 是后台命令 / 监听的实时输出标签。
func (a *App) taskView(c *ui.Context, id string) {
	t := a.tk
	tk := a.findTask(id)
	if tk == nil {
		ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "这个任务已不在当前会话里").TextColor(t.Tx3)
		})
		return
	}
	run := tk.Status == core.TaskRunning
	a.wsHead(c, "terminal", t.ToolExec, tk.Description, func() {
		if run {
			kit.Tag(c, "运行中", kit.KInfo)
		} else {
			kit.Tag(c, "已结束")
		}
	}, func() {
		end := a.st.Now()
		if tk.EndedMs > 0 {
			end = tk.EndedMs
		}
		ui.Text(c, core.FormatElapsed(end-tk.StartedMs)+" · "+tk.TaskID).TextColor(t.Tx3).FontSize(theme.FsCap).FontFeatures("tnum").SingleLine()
		if run {
			kit.Ib(c, "stop", kit.IbOpt{Small: true, Tip: "停止"}).OnClick(func() { a.send(core.CmdStopTask{TaskID: tk.TaskID}) })
		}
	})
	ui.Row(c).Height(30).Shrink(0).PaddingX(12).Gap(8).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(theme.FsCap).Children(func() {
		ui.Text(c, "来自会话「"+core.ShortTitle(a.st.Title)+"」· 后台命令").Grow(1).SingleLine()
		ui.Text(c, "跟随输出")
		if kit.Tog(c, a.taskFollowOn(), "tfollow") {
			a.taskNoFollow = !a.taskNoFollow
		}
		kit.Btn(c, "在终端面板中打开", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "terminal"}).OnClick(func() {
			a.viewTaskInPanel(tk)
		})
	})
	st := &a.taskSt
	if a.taskFollowOn() {
		st.Y = 1e9
	}
	kit.VScroll(c).Key("taskout-" + id).TrackScroll(st).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Margin(0, 12, 12, 12).Padding(8, 12).Background(t.Sunken).Radius(t.RMd()).Border(1, t.Bd1).Children(func() {
			out := strings.TrimRight(tk.Output, "\n")
			if out == "" {
				ui.Text(c, "暂无输出").TextColor(t.Tx4)
				return
			}
			for i, ln := range strings.Split(out, "\n") {
				ui.Row(c).Key(i).MinHeight(monoLH).AlignItems(ui.Center).Children(func() {
					ui.RichText(c, a.termLine(ln)...).NoWrap().Clip()
				})
			}
		})
	})
	a.taskScrollBtn(c)
}

// taskFollowOn 跟随输出默认开启。
func (a *App) taskFollowOn() bool { return !a.taskNoFollow }
