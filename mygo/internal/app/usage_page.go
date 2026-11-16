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
	rng           int // 0 = 7 天 / 1 = 30 天 / 2 = 本月
	set           bool
	limits        []core.Limit
	limitsAt      int64
	limitsState   core.LimitsState
	limitsUpdated string
	loadEmpty     bool      // 上一帧仍在等统计
	revealAt      time.Time // 骨架 → 内容交叉淡入起点
	chartAt       time.Time // 柱状生长动画起点
	lastRng       int       // 上次渲染的范围(切换时重置柱动画)
}

func (a *App) usage() core.UsageReport {
	var u core.UsageReport
	if a.usageSrc != nil {
		u = a.usageSrc(a.use.rng)
	} else if a.cuDemo() && a.demoPgUsage != 0 {
		u = demo.UsageFor(a.demoPgUsage, a.use.rng)
	} else {
		u = demo.Usage(a.use.rng)
	}
	a.usageLimitsFromSession()
	if len(a.use.limits) > 0 {
		u.Limits = a.use.limits
		u.LimitsUpdated = relLimitsAge(a.use.limitsAt)
	} else if !a.cuDemo() && a.use.limitsState == core.LimitsUnavailable {
		u.Limits = nil
		u.LimitsUpdated = ""
	}
	return u
}

// SetUsageSource 注入真实的用量数据源(按范围出报告);未就绪时可返回没有 Days 的空报告。
func (a *App) SetUsageSource(f func(rng int) core.UsageReport) { a.usageSrc = f }

// SetUsageRange 切换用量范围(测试 / 截图用)。
func (a *App) SetUsageRange(r int) { a.use.rng, a.use.set = r, true }

// closeUsage 关闭用量整页,回到工作台(没有侧栏图标入口,这是它唯一的显式出口)。
func (a *App) closeUsage() {
	if a.screen != ScreenUsage {
		return
	}
	log.Info("关闭用量页")
	a.screen = ScreenWork
}

// usageCloseBtn 是页头最右的「×」关闭按钮(设计稿 usage.js:关闭 Esc)。
func (a *App) usageCloseBtn(c *ui.Context) {
	kit.Ib(c, "x", kit.IbOpt{Tip: "关闭 Esc"}).OnClick(a.closeUsage)
}

func (a *App) usagePage(c *ui.Context) {
	t := a.tk
	if !a.use.set {
		a.use.rng, a.use.set = 1, true // 默认 30 天
	}
	if !a.cuDemo() {
		a.usageRefreshLimits(false)
	}
	u := a.usage()
	if len(u.Days) == 0 {
		a.use.loadEmpty = true
		msg := "正在统计会话记录…"
		if a.cuDemo() && a.demoPgUsage == 1 {
			msg = "新账户还没有可统计的会话费用(演示空数据)。"
		}
		a.pane(c).Grow(1).Children(func() {
			a.pageHead(c, "用量与成本", "本机各会话的 token 与费用估算。", func() { a.usageCloseBtn(c) })
			if msg == "正在统计会话记录…" {
				ui.Row(c).Gap(10).Padding(16, 16, 8, 16).AlignItems(ui.Center).Children(func() {
					kit.Ring(c, -1, t.Ac)
					ui.Text(c, msg).TextColor(t.Tx3)
				})
				kit.SkelRows(c, 5)
			} else {
				ui.Text(c, msg).TextColor(t.Tx3).Padding(16)
			}
		})
		if !(a.cuDemo() && a.demoPgUsage == 1) {
			a.redrawSoon()
		}
		return
	}
	if a.use.loadEmpty {
		a.use.loadEmpty = false
		if a.use.revealAt.IsZero() {
			a.use.revealAt = time.Now()
			a.frameSoon()
		}
	}
	reveal := !a.use.revealAt.IsZero()
	a.pane(c).Grow(1).Children(func() {
		root := ui.Box(c).Grow(1)
		root.Children(func() {
			var skel, body ui.Element
			if reveal {
				skel = ui.Column(c)
				skel.Children(func() {
					ui.Row(c).Gap(10).Padding(8, 12).AlignItems(ui.Center).Children(func() {
						kit.SkelLine(c, .5, 14)
					})
					kit.SkelRows(c, 4)
				})
				body = ui.Column(c).Grow(1)
			} else {
				body = ui.Column(c).Grow(1)
			}
			body.Children(func() { a.usageBody(c, u) })
			if reveal {
				kit.Crossfade(skel, body, a.use.revealAt, 3200*time.Millisecond, a.frameSoon)
			}
		})
	})
}

func (a *App) usageBody(c *ui.Context, u core.UsageReport) {
	t := a.tk
	a.pageHead(c, "用量与成本", "本机各会话的 token 与费用估算(取自会话记录,第三方服务商没有费用时只显示 token),以及账户限额。", func() {
		items := make([]kit.SegItem, 0, 3)
		for _, n := range demo.UsageRanges() {
			items = append(items, kit.SegItem{Text: n})
		}
		if i := kit.Seg(c, items, a.use.rng, kit.SegOpt{Key: "use-range"}); i >= 0 {
			a.use.rng = i
		}
		kit.Btn(c, "导出 CSV", kit.BtnOpt{Icon: "download"}).OnClick(func() { a.exportUsage(u) })
		a.usageCloseBtn(c)
	})
	main := func() {
		pct := func(f float64) string { // 极端涨幅(上一段几乎为 0)写成「>999%」,免得读起来像 bug
			if f > 999 {
				return ">999"
			}
			return itoa(int(f + .5))
		}
		delta := "↓ " + pct(abs(u.DeltaPercent)) + "% 较上一段"
		col := t.Ok
		if u.DeltaPercent >= 0 {
			delta = "↑ " + pct(u.DeltaPercent) + "% 较上一段"
			col = t.Warn
		}
		stats := func() {
			if a.pageNarrowMid() {
				ui.Column(c).Gap(10).Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Stretch).Children(func() {
						a.stat(c, core.FormatMoney(u.Total), u.RangeName+"费用", delta, col)
						a.stat(c, u.InputTokens, "输入 token", "", col)
					})
					ui.Row(c).Gap(10).AlignItems(ui.Stretch).Children(func() {
						a.stat(c, u.OutputTokens, "输出 token", "", col)
						a.stat(c, itoa(u.Sessions), "会话", "缓存命中 "+itoa(u.CacheHit)+"%", t.Ok)
					})
				})
				return
			}
			ui.Row(c).Gap(10).AlignItems(ui.Stretch).Children(func() {
				a.stat(c, core.FormatMoney(u.Total), u.RangeName+"费用", delta, col)
				a.stat(c, u.InputTokens, "输入 token", "", col)
				a.stat(c, u.OutputTokens, "输出 token", "", col)
				a.stat(c, itoa(u.Sessions), "会话", "缓存命中 "+itoa(u.CacheHit)+"%", t.Ok)
			})
		}
		stats()
		ui.Box(c).Height(10)
		a.usageChart(c, u)
		ui.Box(c).Height(10)
		if a.pageNarrowMid() {
			ui.Column(c).Gap(10).Children(func() {
				a.usageModels(c, u)
				a.usageProjects(c, u)
			})
		} else {
			ui.Row(c).Gap(10).AlignItems(ui.Stretch).Children(func() {
				a.usageModels(c, u)
				a.usageProjects(c, u)
			})
		}
		ui.Box(c).Height(10)
		a.usageTop(c, u)
		if a.pageNarrowMid() {
			ui.Box(c).Height(12)
			a.usageDrawer(c, u)
		}
	}
	if a.pageNarrowMid() {
		kit.VScroll(c).Key("use-main-n").Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Padding(4, 20, 24, 20).Children(main)
		})
	} else {
		a.pageSplit(c, "use", main, func() { a.usageDrawer(c, u) })
	}
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
		ui.Text(c, v).FontSize(theme.Px(20)).FontWeight(600).FontFeatures("tnum")
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
	if a.use.lastRng != a.use.rng {
		a.use.lastRng = a.use.rng
		a.use.chartAt = time.Time{}
	}
	if a.use.chartAt.IsZero() {
		a.use.chartAt = time.Now()
		a.frameSoon()
	}
	frac := float32(1)
	dur := kit.DurSlow + time.Duration(len(u.Days))*24*time.Millisecond
	if kit.MotionOn() {
		frac = float32(time.Since(a.use.chartAt)) / float32(dur)
		if frac < 1 {
			a.frameSoon()
		}
	}
	a.box(c).Padding(12, 14).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(0, 0, 8, 0).Children(func() {
			ui.Text(c, "每日费用").FontWeight(500)
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
				kit.GrowY(b, frac, time.Duration(i)*24*time.Millisecond, dur)
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
		ui.Text(c, "按模型").FontWeight(500)
		for _, m := range u.Models {
			col := []ui.Color{t.Info, t.Vio, t.Ok}[m.Kind%3]
			ui.Row(c).Height(theme.Px(28)).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, m.Name).Width(96).SingleLine()
				ui.Box(c).Grow(1).Height(6).Radius(3).Background(t.Fill2).Clip().Children(func() {
					ui.Box(c).HeightPercent(100).WidthPercent(float32(m.Fraction * 100)).Radius(3).Background(col)
				})
				ui.Text(c, usageModelCost(m.Name, m.Cost)).Width(58).TextColor(t.Tx3).FontFeatures("tnum").TextAlign(ui.End)
			})
		}
	})
}

func (a *App) usageProjects(c *ui.Context, u core.UsageReport) {
	t := a.tk
	a.box(c).Grow(1).Basis(0).Padding(10, 14).Gap(4).Children(func() {
		ui.Text(c, "按项目").FontWeight(500)
		for _, p := range u.Projects {
			ui.Row(c).Height(theme.Px(28)).Gap(8).AlignItems(ui.Center).Children(func() {
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
		ui.Text(c, "最贵的会话").FontWeight(500).Padding(0, 8, 4, 8)
		for i, s := range u.TopSessions {
			r := ui.ButtonBase(c).Key("top-" + itoa(i)).Height(theme.Px(30)).PaddingX(8).Gap(10).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer)
			if r.Hovered() {
				r.Background(t.Fill1)
			}
			r.OnClick(func() {
				if s.SessionID != "" {
					a.OpenSession(s.SessionID)
					return
				}
				a.toast = "找不到该会话的 id"
			})
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
		ui.Text(c, "账户限额").FontWeight(500).Grow(1)
		if a.use.limitsState == core.LimitsChecking {
			kit.Ring(c, -1, t.Ac)
			ui.Text(c, "更新中…").FontSize(theme.FsCap).TextColor(t.Tx4)
		} else if u.LimitsUpdated != "" {
			ui.Text(c, "更新于 "+u.LimitsUpdated).FontSize(theme.FsCap).TextColor(t.Tx4)
		}
		kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "刷新"}).OnClick(func() { a.usageRefreshLimits(true) })
	})
	if len(u.Limits) == 0 {
		msg := "当前版本无法读取账户限额(需要已连接的 claude 会话发送 get_usage)。本页仍显示本机会话统计。"
		if a.cuDemo() {
			msg = "演示模式:限额为示例数据。"
		} else if a.be == nil {
			msg = "还没有打开会话,无法查询账户限额。请先在工作台打开一个会话,或稍后再试。"
		} else if a.use.limitsState == core.LimitsChecking {
			msg = "正在查询账户限额…"
		} else if a.use.limitsState == core.LimitsUnavailable && a.use.limitsAt != 0 {
			msg = "当前服务商不提供账户限额(例如 DeepSeek 等第三方网关)。本页仍会显示本机会话 token 与费用估算;无价目表的模型显示为「未知费用」。"
		} else if a.use.limitsState == core.LimitsUnavailable {
			msg = "查询账户限额失败。请确认会话仍连接并重试。"
		}
		ui.Text(c, msg).FontSize(theme.FsCap).TextColor(t.Tx4).LineHeight(1.45)
		if !a.cuDemo() && a.use.limitsState == core.LimitsUnavailable && a.use.limitsAt == 0 {
			kit.Btn(c, "重试", kit.BtnOpt{Small: true, Icon: "refresh"}).OnClick(func() { a.usageRefreshLimits(true) })
		}
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
	ui.Text(c, "建议").FontWeight(500)
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
