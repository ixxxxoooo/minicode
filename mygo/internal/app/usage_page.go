package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 用量与成本整页(设计稿 usage.js):范围切换会重算图表与汇总;右侧是账户限额与建议。
// 数据源是 a.usageSrc(演示用 demo.Usage,真实后端接上后换成扫描 ~/.claude/projects 的结果)。

type usageState struct {
	rng int // 0 = 7 天 / 1 = 30 天 / 2 = 本月
	set bool
}

func (a *App) usage() core.UsageReport {
	if a.usageSrc != nil {
		return a.usageSrc(a.use.rng)
	}
	return demo.Usage(a.use.rng)
}

// SetUsageSource 注入真实的用量数据源(按范围出报告);未就绪时可返回没有 Days 的空报告。
func (a *App) SetUsageSource(f func(rng int) core.UsageReport) { a.usageSrc = f }

// SetUsageRange 切换用量范围(测试 / 截图用)。
func (a *App) SetUsageRange(r int) { a.use.rng, a.use.set = r, true }

func (a *App) usagePage(c *ui.Context) {
	t := a.tk
	if !a.use.set {
		a.use.rng, a.use.set = 1, true // 默认 30 天
	}
	u := a.usage()
	if len(u.Days) == 0 {
		a.pane(c).Grow(1).Children(func() {
			a.pageHead(c, "用量与成本", "本机各会话的 token 与费用估算。", nil)
			ui.Text(c, "正在统计会话记录…").TextColor(t.Tx3).Padding(16)
		})
		a.redrawSoon()
		return
	}
	a.pane(c).Grow(1).Children(func() {
		a.pageHead(c, "用量与成本", "本机各会话的 token 与费用估算(取自会话记录,第三方服务商没有费用时只显示 token),以及账户限额。", func() {
			items := make([]kit.SegItem, 0, 3)
			for _, n := range demo.UsageRanges() {
				items = append(items, kit.SegItem{Text: n})
			}
			if i := kit.Seg(c, items, a.use.rng, kit.SegOpt{Key: "use-range"}); i >= 0 {
				a.use.rng = i
			}
			kit.Btn(c, "导出 CSV", kit.BtnOpt{Icon: "download"}).OnClick(func() { a.exportUsage(u) })
		})
		a.pageSplit(c, "use", func() {
			ui.Row(c).Gap(10).AlignItems(ui.Stretch).Children(func() {
				delta := "↓ " + itoa(int(abs(u.DeltaPercent)+.5)) + "% 较上一段"
				col := t.Ok
				if u.DeltaPercent >= 0 {
					delta = "↑ " + itoa(int(u.DeltaPercent+.5)) + "% 较上一段"
					col = t.Warn
				}
				a.stat(c, core.FormatMoney(u.Total), u.RangeName+"费用", delta, col)
				a.stat(c, u.InputTokens, "输入 token", "", col)
				a.stat(c, u.OutputTokens, "输出 token", "", col)
				a.stat(c, itoa(u.Sessions), "会话", "缓存命中 "+itoa(u.CacheHit)+"%", t.Ok)
			})
			ui.Box(c).Height(10)
			a.usageChart(c, u)
			ui.Box(c).Height(10)
			ui.Row(c).Gap(10).AlignItems(ui.Stretch).Children(func() {
				a.usageModels(c, u)
				a.usageProjects(c, u)
			})
			ui.Box(c).Height(10)
			a.usageTop(c, u)
		}, func() { a.usageDrawer(c, u) })
	})
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// stat 是汇总卡(.stat-c):大数字 + 说明 + 可选的小字。
func (a *App) stat(c *ui.Context, v, label, sub string, subCol ui.Color) {
	t := a.tk
	a.box(c).Grow(1).Basis(0).Padding(14, 16).Gap(3).Children(func() {
		ui.Text(c, v).FontSize(20).FontWeight(600).FontFeatures("tnum")
		ui.Text(c, label).FontSize(theme.FsCap).TextColor(t.Tx3)
		if sub != "" {
			ui.Text(c, sub).FontSize(theme.FsMicro).TextColor(subCol)
		}
	})
}

// usageChart 是每日费用的柱状图;悬停看每天的数值,今天(最后一根)用强调色深一档。
func (a *App) usageChart(c *ui.Context, u core.UsageReport) {
	t := a.tk
	mx := u.MaxDay()
	a.box(c).Padding(12, 14).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(0, 0, 8, 0).Children(func() {
			ui.Text(c, "每日费用").FontWeight(600)
			ui.Text(c, "悬停看每天的数值").FontSize(theme.FsCap).TextColor(t.Tx3)
			ui.Spacer(c).Grow(1)
			ui.Box(c).Size(6, 6).Radius(3).Background(t.Ac)
			ui.Text(c, "费用").FontSize(theme.FsCap).TextColor(t.Tx3)
		})
		ui.Row(c).Height(132).Gap(float32(map[bool]int{true: 4, false: 9}[len(u.Days) > 20])).AlignItems(ui.End).Children(func() {
			for i, d := range u.Days {
				h := float32(max(4, d.Cost/mx*100))
				col := t.Ac.Mix(t.BG, .42)
				if i == len(u.Days)-1 {
					col = t.Ac
				}
				b := ui.Box(c.Key("bar-" + itoa(i))).Grow(1).Basis(0).HeightPercent(h).Radius(2).Background(col).Tooltip(d.Label + " · " + core.FormatMoney(d.Cost))
				if b.Hovered() {
					b.Background(t.Ac)
				}
			}
		})
		ui.Row(c).Padding(4, 2, 0, 2).FontSize(theme.FsMicro).TextColor(t.Tx4).Children(func() {
			ui.Text(c, u.Days[0].Label)
			ui.Spacer(c).Grow(1)
			ui.Text(c, u.Days[len(u.Days)/2].Label)
			ui.Spacer(c).Grow(1)
			ui.Text(c, u.Days[len(u.Days)-1].Label)
		})
	})
}

func (a *App) usageModels(c *ui.Context, u core.UsageReport) {
	t := a.tk
	a.box(c).Grow(1).Basis(0).Padding(10, 14).Gap(4).Children(func() {
		ui.Text(c, "按模型").FontWeight(600)
		for _, m := range u.Models {
			col := []ui.Color{t.Info, t.Vio, t.Ok}[m.Kind%3]
			ui.Row(c).Height(28).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, m.Name).Width(96).SingleLine()
				ui.Box(c).Grow(1).Height(6).Radius(3).Background(t.Fill2).Clip().Children(func() {
					ui.Box(c).HeightPercent(100).WidthPercent(float32(m.Fraction * 100)).Radius(3).Background(col)
				})
				ui.Text(c, core.FormatMoney(m.Cost)).Width(58).TextColor(t.Tx3).FontFeatures("tnum").TextAlign(ui.End)
			})
		}
	})
}

func (a *App) usageProjects(c *ui.Context, u core.UsageReport) {
	t := a.tk
	a.box(c).Grow(1).Basis(0).Padding(10, 14).Gap(4).Children(func() {
		ui.Text(c, "按项目").FontWeight(600)
		for _, p := range u.Projects {
			ui.Row(c).Height(28).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, p.Name).Grow(1).SingleLine().MinWidth(0)
				ui.Text(c, itoa(p.Sessions)+" 个会话").FontSize(theme.FsCap).TextColor(t.Tx3).FontFeatures("tnum").Shrink(0)
				ui.Text(c, core.FormatMoney(p.Cost)).Width(54).FontFeatures("tnum").TextAlign(ui.End)
			})
		}
	})
}

func (a *App) usageTop(c *ui.Context, u core.UsageReport) {
	t := a.tk
	a.box(c).Padding(10, 6).Children(func() {
		ui.Text(c, "最贵的会话").FontWeight(600).Padding(0, 8, 4, 8)
		for i, s := range u.TopSessions {
			r := ui.ButtonBase(c).Key("top-" + itoa(i)).Height(30).PaddingX(8).Gap(10).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer)
			if r.Hovered() {
				r.Background(t.Fill1)
			}
			r.OnClick(func() { a.toast = "已在 Agent 形态打开该会话" })
			r.Children(func() {
				ui.Text(c, s.Title).Grow(1).SingleLine().MinWidth(0)
				ui.Text(c, s.Project).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Shrink(0)
				kit.Tag(c, s.Model)
				ui.Text(c, core.FormatMoney(s.Cost)).Width(50).FontFeatures("tnum").TextAlign(ui.End)
			})
		}
	})
}

func (a *App) usageDrawer(c *ui.Context, u core.UsageReport) {
	t := a.tk
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "账户限额").FontWeight(600).Grow(1)
		if len(u.Limits) > 0 {
			ui.Text(c, "更新于 "+u.LimitsUpdated).FontSize(theme.FsCap).TextColor(t.Tx4)
			kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "刷新"}).OnClick(func() { a.toast = "已刷新账户限额" })
		}
	})
	if len(u.Limits) == 0 {
		ui.Text(c, "账户限额还没有接入(需要 claude 的 get_usage 控制请求),这里先只显示本机会话统计。").FontSize(theme.FsCap).TextColor(t.Tx4).LineHeight(1.45)
	}
	ui.Column(c).Gap(10).Children(func() {
		for _, l := range u.Limits {
			ui.Column(c).Gap(4).Children(func() {
				ui.Row(c).Children(func() {
					ui.Text(c, l.Name).FontWeight(500).Grow(1)
					ui.Text(c, itoa(l.Percent)+"%").TextColor(t.Tx3).FontFeatures("tnum")
				})
				m := kit.Meter(c, float32(l.Percent)).Height(6)
				_ = m
				ui.Text(c, l.Reset).FontSize(theme.FsMicro).TextColor(t.Tx4)
			})
		}
	})
	kit.Sep(c)
	ui.Text(c, "建议").FontWeight(600)
	ui.Column(c).Gap(10).Children(func() {
		for _, tip := range u.Tips {
			icon, col := []string{"okc", "alert", "info"}[tip.Level], []ui.Color{t.Ok, t.Warn, t.Info}[tip.Level]
			ui.Row(c).Gap(8).AlignItems(ui.Start).Children(func() {
				kit.Ic(c, icon, kit.S12).TextColor(col).Margin(2, 0, 0, 0)
				ui.Column(c).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, tip.Title).FontWeight(500)
					ui.Text(c, tip.Desc).FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(1.45)
				})
			})
		}
	})
	ui.Spacer(c).Grow(1)
	ui.Row(c).Gap(8).AlignItems(ui.Start).FontSize(theme.FsCap).TextColor(t.Tx3).Children(func() {
		kit.Ic(c, "info", kit.S12).Margin(2, 0, 0, 0)
		ui.Text(c, "限额只在打开此页或点「刷新」时查询,不轮询。").LineHeight(1.45).Grow(1)
	})
}

// exportUsage 把当前范围的每日费用写成 CSV(演示写到临时目录,真实模式让用户选位置)。
func (a *App) exportUsage(u core.UsageReport) {
	p, err := writeTemp("usage.csv", core.UsageCSV(u))
	if err != nil {
		log.Warn("导出用量失败", "err", err)
		a.toast = "导出失败:" + err.Error()
		return
	}
	log.Info("导出用量", "path", p, "days", len(u.Days))
	a.toast = "已导出到 " + p
}

// redrawSoon 0.3 秒后重画一帧(等后台统计完成;用标志位避免堆积定时器)。
func (a *App) redrawSoon() {
	if a.win == nil || a.redrawPending {
		return
	}
	a.redrawPending = true
	time.AfterFunc(300*time.Millisecond, func() {
		a.win.Update(func() { a.redrawPending = false })
	})
}

// frameSoon 请求下一帧(入场动画推进用,约 16 ms 后重绘一次)。
func (a *App) frameSoon() {
	if a.win == nil || a.framePending {
		return
	}
	a.framePending = true
	time.AfterFunc(16*time.Millisecond, func() {
		a.win.Update(func() { a.framePending = false })
	})
}
