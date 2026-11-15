package app

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const ctxCompressThr = 0.85

// ctxState 推断上下文弹层的展示态(设计稿六种状态)。
func (a *App) ctxState() string {
	u := a.st.Usage
	if a.ctxCompacting {
		return "busy"
	}
	if a.ctxLoading {
		return "load"
	}
	if u.LimitsState == core.LimitsChecking {
		return "load"
	}
	if u.LimitsState == core.LimitsUnavailable && u.FiveHourPct == nil && u.WeeklyPct == nil {
		return "none"
	}
	if u.LimitsState == core.LimitsUnavailable && u.LimitsAt == 0 && len(u.ContextCategories) == 0 {
		return "none"
	}
	r := u.ContextRatio()
	if r >= 0.95 {
		return "warn"
	}
	if r >= ctxCompressThr {
		return "warn"
	}
	return "ok"
}

func (a *App) ctxRingColor(p float32) ui.Color {
	t := a.tk
	if p >= 0.95 {
		return t.Err
	}
	if p >= ctxCompressThr {
		return t.Warn
	}
	return t.Ac
}

// ctxAnchorPopup 输入框 / 状态栏共用的上下文弹层。
func (a *App) ctxAnchorPopup(c *ui.Context, key string, anchor ui.Element, attach ui.Anchor) {
	a.ctxEnsureLoaded()
	a.popOpen(key, func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			panel = a.popPanel(panel, 300).Padding(10, 12).Gap(9)
			switch attach {
			case ui.AnchorTopRight:
				panel.AttachTo(anchor, ui.AnchorTopRight, ui.AnchorBottomRight).Margin(0, 0, 6, 0)
			default:
				panel.AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(0, 0, 6, 0)
			}
			panel.Children(func() { a.ctxPanelBody(c) })
		})
	})
}

func (a *App) ctxEnsureLoaded() {
	if len(a.st.Usage.ContextCategories) > 0 {
		return
	}
	if a.be != nil && a.be.Name() == "claude" {
		a.send(core.CmdRefreshUsage{Force: false})
		a.ctxLoading = true
	} else if a.be != nil && a.be.Name() == "demo" {
		a.send(core.CmdRefreshUsage{Force: true})
	}
}

// ctxPanelBody 上下文弹层内容(与状态栏 / 输入框共用)。
func (a *App) ctxPanelBody(c *ui.Context) {
	t := a.tk
	st := a.ctxState()
	u := a.st.Usage
	p := u.ContextRatio()
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		a.bigRingColored(c, p)
		ui.Column(c).Grow(1).Children(func() {
			ui.Row(c).Gap(4).Children(func() {
				ui.Text(c, fmtTok(u.ContextUsed)+" / "+fmtTok(u.ContextSize)).FontWeight(600).FontFeatures("tnum")
				col := t.Tx3
				if st == "warn" {
					col = t.Warn
				}
				ui.Text(c, "· "+itoa(int(p*100+.5))+"%").TextColor(col)
			})
			ui.Text(c, a.modelName()+" · 自动压缩阈值 85%").FontSize(theme.FsCap).TextColor(t.Tx3)
		})
		switch st {
		case "busy":
			kit.Btn(c, "压缩中…", kit.BtnOpt{Small: true, Variant: kit.BtnGhost})
		case "warn":
			kit.Btn(c, "压缩", kit.BtnOpt{Small: true, Variant: kit.BtnPri}).OnClick(a.requestCompact)
		default:
			kit.Btn(c, "压缩", kit.BtnOpt{Small: true}).OnClick(a.requestCompact)
		}
	})
	if st == "warn" {
		ui.Row(c).Gap(6).Padding(6, 8).Radius(t.RSm()).Background(t.WarnSoft).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
			ui.Text(c, fmt.Sprintf("已用 %d%%,下一轮可能触发自动压缩。现在压缩可保留更多细节。", int(p*100+.5))).FontSize(theme.FsCap).Grow(1)
		})
	}
	if st == "busy" {
		ui.Row(c).Gap(6).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(theme.FsCap).Children(func() {
			kit.Spin(c, t.Tx3)
			ui.Text(c, "正在压缩上下文…完成后这里会更新占用。")
		})
		ui.Box(c).Height(3).Radius(2).Background(t.Fill2).Children(func() {
			ui.Box(c).HeightPercent(50).WidthPercent(40).Radius(2).Background(t.Ac)
		})
	}
	if st == "load" {
		ui.Column(c).Gap(6).Children(func() {
			ui.Box(c).Height(6).Radius(3).Background(t.Fill2)
			for _, w := range []float32{.7, .55, .62, .48} {
				ui.Box(c).Height(10).WidthPercent(w * 100).Radius(t.RXs()).Background(t.Fill2)
			}
			ui.Text(c, "正在读取分类明细…").FontSize(theme.FsCap).TextColor(t.Tx4)
		})
	} else {
		a.ctxBarLive(c, u)
	}
	a.ctxLimitsBlock(c, st, u)
	ui.Row(c).AlignItems(ui.Center).FontSize(theme.FsCap).TextColor(t.Tx3).Gap(6).Children(func() {
		ui.Text(c, a.ctxUpdatedText(st, u)).Grow(1)
		kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "刷新"}).OnClick(func() {
			a.ctxLoading = len(u.ContextCategories) == 0
			a.send(core.CmdRefreshUsage{Force: true})
		})
		kit.Btn(c, "用量与成本", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "arrow"}).OnClick(func() {
			a.popup = ""
			a.screen = ScreenUsage
		})
	})
}

func (a *App) ctxUpdatedText(st string, u core.Usage) string {
	if st == "load" {
		return "更新中…"
	}
	if st == "none" {
		return "分类明细随回合自动更新"
	}
	if u.LimitsAt == 0 {
		return "暂无账户限额数据"
	}
	return "更新于 " + relTime(a.now(), time.Unix(u.LimitsAt, 0))
}

func (a *App) ctxLimitsBlock(c *ui.Context, st string, u core.Usage) {
	t := a.tk
	if st == "load" {
		return
	}
	if st == "none" {
		ui.Row(c).Gap(6).Padding(6, 8).Radius(t.RSm()).Background(t.Fill1).Children(func() {
			kit.Ic(c, "info", kit.S12).TextColor(t.Tx3)
			ui.Text(c, "当前使用第三方服务商,没有账户限额数据。").FontSize(theme.FsCap).TextColor(t.Tx3)
		})
		return
	}
	if u.LimitsState == core.LimitsUnavailable && u.LimitsAt != 0 {
		ui.Row(c).Gap(6).Padding(6, 8).Radius(t.RSm()).Background(t.ErrSoft).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "alert", kit.S12).TextColor(t.Err)
			ui.Text(c, "账户限额查询失败,分类明细不受影响。").FontSize(theme.FsCap).Grow(1)
			kit.Btn(c, "重试", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
				a.send(core.CmdRefreshUsage{Force: true})
			})
		})
		return
	}
	if u.FiveHourPct == nil && u.WeeklyPct == nil {
		ui.Row(c).Gap(6).Padding(6, 8).Children(func() {
			kit.Ic(c, "info", kit.S12).TextColor(t.Tx3)
			ui.Text(c, "无账户限额数据(真实模式需接 get_usage)。").FontSize(theme.FsCap).TextColor(t.Tx3)
		})
		return
	}
	ui.Column(c).Gap(8).Children(func() {
		if u.FiveHourPct != nil {
			a.ctxLimitRow(c, "5 小时限额", int(*u.FiveHourPct+.5), u.FiveHourResets)
		}
		if u.WeeklyPct != nil {
			a.ctxLimitRow(c, "本周限额", int(*u.WeeklyPct+.5), u.WeeklyResets)
		}
	})
}

func (a *App) ctxLimitRow(c *ui.Context, name string, pct int, resetUnix int64) {
	t := a.tk
	sub := "—"
	if resetUnix > 0 {
		sub = relTime(a.now(), time.Unix(resetUnix, 0)) + "后重置"
	}
	ui.Column(c).Gap(4).Children(func() {
		ui.Row(c).Children(func() {
			ui.Text(c, name).Grow(1)
			ui.Text(c, itoa(pct)+"%").FontFeatures("tnum").TextColor(t.Tx3)
		})
		kit.Meter(c, float32(pct))
		ui.Text(c, sub).FontSize(theme.FsMicro).TextColor(t.Tx4)
	})
}

func (a *App) ctxBarLive(c *ui.Context, u core.Usage) {
	t := a.tk
	cats := u.ContextCategories
	if len(cats) == 0 {
		a.ctxBar(c) // 演示占位
		return
	}
	var used uint64
	for _, k := range cats {
		if k.Kind == core.CtxUsed {
			used += k.Tokens
		}
	}
	total := u.ContextSize
	if total == 0 {
		total = used
	}
	ui.Row(c).Height(6).Radius(3).Background(t.Fill2).Clip().Children(func() {
		for _, k := range cats {
			if k.Kind != core.CtxUsed {
				continue
			}
			w := float32(k.Tokens) / float32(max(total, 1)) * 100
			ui.Box(c).WidthPercent(w).FillHeight().Background(ctxCatColor(t, k.Name)).Shrink(0)
		}
		// 阈值刻度 85%
	})
	for _, k := range cats {
		if k.Kind == core.CtxBuffer {
			continue
		}
		note := ""
		if k.Kind == core.CtxDeferred {
			note = "按需加载"
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).FontSize(theme.FsCap).Children(func() {
			ui.Box(c).Size(6, 6).Radius(3).Background(ctxCatColor(t, k.Name))
			ui.Text(c, k.Name).Grow(1)
			if note != "" {
				ui.Text(c, "· "+note).TextColor(t.Tx4)
			}
			ui.Text(c, fmtTok(k.Tokens)).TextColor(t.Tx3).FontFeatures("tnum")
		})
	}
	kit.Sep(c)
}

func ctxCatColor(t *theme.Tokens, name string) ui.Color {
	switch name {
	case "系统提示":
		return t.Info
	case "消息":
		return t.Ok
	case "工具定义":
		return t.Warn
	case "MCP", "MCP 工具":
		return t.Vio
	case "空闲":
		return t.Fill3
	}
	return t.Ac
}

func (a *App) bigRingColored(c *ui.Context, p float32) {
	col := a.ctxRingColor(p)
	a.bigRingColoredWith(c, p, col)
}

func (a *App) bigRingColoredWith(c *ui.Context, p float32, col ui.Color) {
	t := a.tk
	e := ui.Box(c).Size(34, 34).Shrink(0)
	e.Draw(func(pt *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		const rad = 14.0
		arc := func(frac float32, c ui.Color) {
			n := max(2, int(frac*60))
			pts := make([][2]float32, 0, n+1)
			for i := 0; i <= n; i++ {
				ang := -1.5707963 + float64(i)/float64(n)*float64(frac)*6.2831853
				pts = append(pts, [2]float32{cx + rad*float32(math.Cos(ang)), cy + rad*float32(math.Sin(ang))})
			}
			pt.StrokePolyline(pts, 5, c)
		}
		arc(1, t.Fill3)
		if p > 0 {
			arc(min(p, .999), col)
		}
	})
}

func (a *App) requestCompact() {
	if a.ctxCompacting {
		return
	}
	log.Info("请求压缩上下文")
	a.ctxCompacting = true
	if a.be != nil && a.be.Name() == "demo" {
		a.bg("ctx-compact-demo", func() func() {
			time.Sleep(1200 * time.Millisecond)
			u := a.st.Usage
			u.ContextUsed = u.ContextUsed * 6 / 10
			return func() {
				a.ctxCompacting = false
				a.st.Usage = u
				a.noticeInfo("已压缩上下文", "占用已更新(演示)")
			}
		})
		return
	}
	a.toast = "已请求压缩(等待 Claude 处理)"
	a.ctxCompacting = false
}

func (a *App) onUsageEvent(u core.Usage) {
	a.ctxLoading = false
	if a.ctxCompacting && u.ContextUsed < a.st.Usage.ContextUsed {
		a.ctxCompacting = false
	}
	a.usageLimitsFromSession()
}
