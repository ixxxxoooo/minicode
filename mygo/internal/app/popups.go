package app

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// popOpen 把「当前打开的弹层名」与 ui.Popover 的 *bool 对接:
// 返回 open 指针给 Popover,Popover 因点外部 / Esc 关闭时回写成空;关闭时先播 kit.PopOut。
func (a *App) popOpen(key string, fn func(open *bool)) {
	if a.popClosing == key && time.Since(a.popCloseAt) >= kit.DurFast {
		a.popClosing, a.popSeen = "", ""
	}
	if a.popClosing == key && time.Since(a.popCloseAt) >= kit.DurFast {
		a.popClosing, a.popSeen = "", ""
	}
	hold := a.popup == key
	if a.popClosing == key && time.Since(a.popCloseAt) < kit.DurFast {
		hold = true
	}
	if a.popup == key && a.popSeen != key {
		a.popSeen, a.popAt = key, time.Now()
	}
	fn(&hold)
	if !hold && a.popup == key {
		a.popClosing, a.popCloseAt, a.popup = key, time.Now(), ""
		a.frameSoon()
	}
}

// togglePop 是弹层入口按钮的点击:再点同一个入口就收起。
func (a *App) togglePop(key string) {
	if a.popup == key {
		a.popup = ""
	} else {
		a.popup = key
	}
}

// popPanel 给弹层面板上设计稿的外观:raised 底、大圆角、第二层阴影、4px 内边距。
func (a *App) popPanel(panel ui.Element, w float32) ui.Element {
	t := a.tk
	panel.Width(w).Background(t.Raised).Radius(t.RLg()).Padding(4).Border(1, t.Bd1)
	for _, s := range t.Shadow2() {
		panel.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	return a.popIn(panel)
}

// popIn 是弹层的入场 / 退场动画。
func (a *App) popIn(panel ui.Element) ui.Element {
	if a.popClosing != "" && time.Since(a.popCloseAt) < kit.DurFast {
		a.frameSoon()
		return kit.PopOut(panel, a.popCloseAt)
	}
	if time.Since(a.popAt) < kit.DurFast {
		a.frameSoon()
	}
	return kit.PopIn(panel, a.popAt)
}

// mi 是菜单项(.mi:26 高,选中 ac-soft 底 + ac-tx 字 + 对勾)。
func (a *App) mi(c *ui.Context, key, icon, text, right string, on bool) ui.Element {
	t := a.tk
	b := ui.ButtonBase(c).Key("mi-" + key).Height(26).PaddingX(8).Gap(8).Radius(t.RSm()).AlignItems(ui.Center).Justify(ui.Start).
		Cursor(ui.CursorDefault).Transition(kit.Fade)
	switch {
	case on:
		b.Background(t.AcSoft).TextColor(t.AcTx)
	case b.Hovered():
		b.Background(t.Fill1)
	}
	return b.Children(func() {
		if icon != "" {
			kit.Ic(c, icon, kit.S12)
		}
		ui.Text(c, text).Grow(1).SingleLine()
		if right != "" {
			ui.Text(c, right).FontSize(theme.FsCap).TextColor(t.Tx3)
		}
		if on {
			kit.Ic(c, "check", kit.S12)
		}
	})
}

// ───────── 运行配置弹层 ─────────

// cfgPopup 在 anchor 上方弹出:权限模式 + 模型 + 思考强度(状态栏入口 compact 时不带强度)。
func (a *App) cfgPopup(c *ui.Context, key string, anchor ui.Element, compact bool) {
	a.popOpen(key, func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 264).AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(0, 0, 6, 0)
			kit.Cap(c, "权限模式")
			modes := []struct {
				m    core.PermissionMode
				icon string
			}{{core.ModeDefault, "shield"}, {core.ModeAcceptEdits, "edit"}, {core.ModePlan, "list"}, {core.ModeBypassPermissions, "bolt"}}
			for _, md := range modes {
				if a.mi(c, "mode-"+md.m.CLI(), md.icon, md.m.Label(), "", a.st.Mode == md.m).Clicked() {
					a.send(core.CmdSetMode{Mode: md.m})
					a.st.Mode = md.m // 乐观更新,后端确认后以事件为准
				}
			}
			kit.Sep(c)
			kit.Cap(c, "模型")
			for _, m := range a.st.Models {
				if a.mi(c, "model-"+m.ID, "", m.Name, shortDesc(m), a.st.Model == m.ID).Clicked() {
					a.pickModel(m.ID)
				}
			}
			if compact {
				return
			}
			kit.Sep(c)
			ui.Row(c).Padding(4, 8, 6, 8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "思考强度").FontSize(theme.FsCap).FontWeight(500).TextColor(t3(a)).Grow(1)
				efforts := []core.Effort{core.EffortLow, core.EffortMedium, core.EffortHigh, core.EffortXHigh}
				items := make([]kit.SegItem, len(efforts))
				on := 1
				for i, e := range efforts {
					items[i] = kit.SegItem{Text: effortLabel(e)}
					if e == a.st.Effort {
						on = i
					}
				}
				if i := kit.Seg(c, items, on, kit.SegOpt{Key: "effortseg"}); i >= 0 {
					a.pickEffort(efforts[i])
				}
			})
		})
	})
}

func t3(a *App) ui.Color { return a.tk.Tx3 }

// shortDesc 取模型说明的第一个词组,放在行尾(「最强」「最快」)。
func shortDesc(m core.ModelInfo) string {
	switch m.ID {
	case "opus":
		return "最强"
	case "haiku":
		return "最快"
	}
	return ""
}

// ───────── 状态栏弹层:上下文 / 更新 ─────────

func (a *App) ctxPopup(c *ui.Context, anchor ui.Element) {
	a.ctxAnchorPopup(c, "ctx", anchor, ui.AnchorTopRight)
}

// ctxBar 是上下文分类条:系统提示 / 消息 / 工具定义 / MCP 各占一段,下面是图例。
func (a *App) ctxBar(c *ui.Context) {
	t := a.tk
	cats := []struct {
		name string
		col  ui.Color
		pct  float32
		v    string
	}{{"系统提示", t.Info, 14, "28k"}, {"消息", t.Ok, 11, "22k"}, {"工具定义", t.Warn, 9, "18k"}, {"MCP", t.Vio, 4, "8k"}}
	ui.Row(c).Height(6).Radius(3).Background(t.Fill2).Clip().Children(func() {
		for _, k := range cats {
			ui.Box(c).WidthPercent(k.pct).FillHeight().Background(k.col).Shrink(0)
		}
	})
	for _, k := range cats {
		ui.Row(c).Gap(8).AlignItems(ui.Center).FontSize(theme.FsCap).Children(func() {
			ui.Box(c).Size(6, 6).Radius(3).Background(k.col)
			ui.Text(c, k.name).Grow(1)
			ui.Text(c, k.v).TextColor(t.Tx3).FontFeatures("tnum")
		})
	}
	kit.Sep(c)
}

func (a *App) bigRing(c *ui.Context, p float32) {
	t := a.tk
	e := ui.Box(c).Size(34, 34).Shrink(0)
	e.Draw(func(pt *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		const rad = 14.0
		arc := func(frac float32, col ui.Color) {
			n := max(2, int(frac*60))
			pts := make([][2]float32, 0, n+1)
			for i := 0; i <= n; i++ {
				ang := -1.5707963 + float64(i)/float64(n)*float64(frac)*6.2831853
				pts = append(pts, [2]float32{cx + rad*float32(math.Cos(ang)), cy + rad*float32(math.Sin(ang))})
			}
			pt.StrokePolyline(pts, 5, col)
		}
		arc(1, t.Fill3)
		if p > 0 {
			arc(min(p, .999), t.Ac)
		}
	})
}

func limitText(p *float32) string {
	if p == nil {
		return "—"
	}
	return itoa(int(*p+.5)) + "%"
}

// fmtTok 把 token 数写成 76k / 1.2M。
func fmtTok(n uint64) string {
	switch {
	case n >= 1_000_000:
		return trimZero(float64(n)/1e6) + "M"
	case n >= 1000:
		return trimZero(float64(n)/1e3) + "k"
	}
	return itoa(int(n))
}

func trimZero(f float64) string {
	i := int(f)
	if f == float64(i) || f >= 100 {
		return itoa(int(f + .5))
	}
	d := int((f-float64(i))*10 + .5)
	if d == 10 {
		return itoa(i + 1)
	}
	return itoa(i) + "." + itoa(d)
}

// ───────── 账号菜单 ─────────

func (a *App) acctPopup(c *ui.Context, which string, anchor ui.Element) {
	t := a.tk
	a.popOpen("acct@"+which, func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 252).Padding(0).Clip().AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(0, 0, 6, 0)
			ui.Row(c).Gap(9).Padding(10, 12).AlignItems(ui.Center).Children(func() {
				a.avatarSized(c, 30, 13)
				ui.Column(c).Grow(1).Children(func() {
					ui.Text(c, "demo").FontWeight(500)
					ui.Text(c, "Claude Max · 官方账号").FontSize(theme.FsCap).TextColor(t.Tx3)
				})
			})
			ui.Column(c).Gap(7).Padding(2, 12, 10, 12).FontSize(theme.FsCap).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "本月费用").Width(62).TextColor(t.Tx3)
					ui.Text(c, "$12.40").FontWeight(600).FontFeatures("tnum")
				})
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "5 小时").Width(62).TextColor(t.Tx3)
					kit.Meter(c, 38).Grow(1)
					ui.Text(c, "38%").TextColor(t.Tx3).FontFeatures("tnum")
				})
			})
			ui.Column(c).Padding(4).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1).Children(func() {
				item := func(key, icon, text, right string, fn func()) {
					b := a.mi(c, "acct-"+key, icon, text, right, false).Height(28)
					if b.Clicked() {
						a.popup = ""
						fn()
					}
				}
				item("settings", "sliders", "设置", "⌘ ,", func() { a.screen = ScreenSettings })
				item("usage", "chart", "用量与成本", "", func() { a.screen = ScreenUsage })
				item("keys", "kbd", "快捷键", "", func() { a.openSettings("shortcuts") })
				kit.Sep(c)
				item("update", "refresh", "检查更新", "", func() { a.openSettings("about") })
				item("about", "info", "关于 MiniCode", "0.2.0", func() { a.openSettings("about") })
			})
		})
	})
}

// avatarSized 是圆形头像(账号菜单头部用 30px)。
func (a *App) avatarSized(c *ui.Context, size, font float32) ui.Element {
	t := a.tk
	return ui.Row(c).Size(size, size).Radius(size / 2).Background(t.AcSoft).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Text(c, "L").FontSize(font).FontWeight(600).TextColor(t.AcTx)
	})
}

// openSettings 打开设置并跳到某个分类。
func (a *App) openSettings(tab string) {
	a.screen = ScreenSettings
	if tab != "" {
		a.setTab = tab
	}
	a.set.query = ""
}

// dialogIn 是居中浮层(命令面板、快速选择)的入场 / 退场动画。
func (a *App) dialogIn(key string, backdrop, panel ui.Element) {
	if a.dlgClosing == key && time.Since(a.dlgCloseAt) < kit.DurFast {
		kit.DialogOut(backdrop, panel, a.dlgCloseAt, a.frameSoon)
		return
	}
	if a.dlgSeen != key {
		a.dlgSeen, a.dlgAt = key, time.Now()
	}
	kit.DialogIn(backdrop, panel, a.dlgAt, a.frameSoon)
}
