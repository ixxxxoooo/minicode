package app

import (
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// popOpen 把「当前打开的弹层名」与 ui.Popover 的 *bool 对接:
// 返回 open 指针给 Popover,Popover 因点外部 / Esc 关闭时回写成空;关闭时先播 kit.PopOut。
func (a *App) popOpen(key string, fn func(open *bool)) {
	if a.popClosing == key && kit.Since(a.popCloseAt) >= kit.DurFast {
		a.popClosing, a.popSeen = "", ""
	}
	hold := a.popup == key
	if a.popClosing == key && kit.Since(a.popCloseAt) < kit.DurFast {
		hold = true
	}
	if a.popup == key && a.popSeen != key {
		a.popSeen, a.popAt = key, kit.Now()
	}
	fn(&hold)
	if !hold && a.popup == key {
		a.popClosing, a.popCloseAt, a.popup = key, kit.Now(), ""
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
	if a.popClosing != "" && kit.Since(a.popCloseAt) < kit.DurFast {
		a.frameSoon()
		return kit.PopOut(panel, a.popCloseAt)
	}
	if kit.Since(a.popAt) < kit.DurFast {
		a.frameSoon()
	}
	return kit.PopIn(panel, a.popAt)
}

// mi 是菜单项(.mi:26 高,选中 ac-soft 底 + ac-tx 字 + 对勾)。
func (a *App) mi(c *ui.Context, key, icon, text, right string, on bool) ui.Element {
	return a.miTint(c, key, icon, nil, text, right, on)
}

// miTint 同 mi,图标可以单独上色(tint 为 nil = 跟随文字色)。
func (a *App) miTint(c *ui.Context, key, icon string, tint *ui.Color, text, right string, on bool) ui.Element {
	t := a.tk
	b := ui.ButtonBase(c).Key("mi-" + key).Height(theme.Px(26)).PaddingX(8).Gap(8).Radius(t.RSm()).AlignItems(ui.Center).Justify(ui.Start).
		Cursor(ui.CursorPointer).Transition(kit.Fade)
	switch {
	case on:
		b.Background(t.AcSoft).TextColor(t.AcTx)
	case b.Hovered():
		b.Background(t.Fill1)
	}
	return b.Children(func() {
		if strings.HasPrefix(icon, "logo:") { // 模型品牌 logo;品牌为空 = 占位图标
			kit.Logo(c, strings.TrimPrefix(icon, "logo:"), 14, t.Tx2)
		} else if icon != "" {
			ic := kit.Ic(c, icon, kit.S12)
			if tint != nil {
				ic.TextColor(*tint)
			}
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
			hints := map[core.PermissionMode]string{
				core.ModeDefault: "每次都问", core.ModeAcceptEdits: "命令仍会问", core.ModePlan: "只读规划", core.ModeBypassPermissions: "全部放行",
			}
			for _, md := range modes {
				tint := a.modeTint(md.m)
				right := hints[md.m]
				blocked := md.m == core.ModeBypassPermissions && a.NoBypassPermissions()
				if blocked { // 设置里禁止了绕过:不再让人点了没反应,直接写明原因,点击去设置里解除
					right = "设置中已禁止"
				}
				if a.miTint(c, "mode-"+md.m.CLI(), md.icon, &tint, md.m.Label(), right, a.st.Mode == md.m && !blocked).Clicked() {
					if blocked {
						log.Info("跳过权限确认被设置禁止,转到权限设置")
						a.popup = ""
						a.openSettings("permissions")
						a.toast = "已在设置里禁止绕过权限模式,关掉「禁止绕过权限模式」后才能选"
						continue
					}
					a.send(core.CmdSetMode{Mode: md.m})
					a.st.Mode = md.m // 乐观更新,后端确认后以事件为准
					a.rememberPermMode(md.m)
				}
			}
			kit.Sep(c)
			kit.Cap(c, "模型")
			a.cfgModels(c)
			if a.mi(c, "prov-manage", "sliders", "管理服务商…", "", false).Clicked() {
				a.popup = ""
				a.openSettings("models")
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
	a.acctRefresh(false)
	a.acctApplyUsage() // 用量扫描比资料读取晚完成,每次打开都按最新扫描结果取本月费用(有缓存,开销小)
	a.popOpen("acct@"+which, func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 252).Padding(0).Clip().AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(0, 0, 6, 0)
			d := a.acctMenuData()
			ui.Row(c).Gap(9).Padding(10, 12).AlignItems(ui.Center).Children(func() {
				a.avatarSized(c, 30, 13, d.initial)
				ui.Column(c).Grow(1).Children(func() {
					if d.loading && !a.cuDemo() {
						ui.Column(c).Gap(4).Grow(1).Children(func() {
							kit.SkelLine(c, .55, 12)
							kit.SkelLine(c, .85, 10)
						})
					} else {
						ui.Text(c, d.name).FontWeight(500)
						ui.Text(c, d.subtitle).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
					}
				})
			})
			ui.Column(c).Gap(7).Padding(2, 12, 10, 12).FontSize(theme.FsCap).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "本月费用").Width(62).TextColor(t.Tx3)
					if d.loading && !a.cuDemo() {
						kit.Spin(c, t.Tx3)
					} else if d.monthCost != "" {
						ui.Text(c, d.monthCost).FontWeight(600).FontFeatures("tnum")
					} else {
						ui.Text(c, "暂无数据").TextColor(t.Tx3)
					}
				})
				if d.fivePct >= 0 {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						lbl := d.fiveLabel
						if lbl == "" {
							lbl = "5 小时"
						}
						ui.Text(c, lbl).Width(62).TextColor(t.Tx3)
						kit.Meter(c, float32(d.fivePct)).Grow(1)
						ui.Text(c, itoa(d.fivePct)+"%").TextColor(t.Tx3).FontFeatures("tnum")
					})
				} else if !d.loading || a.cuDemo() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "账户限额").Width(62).TextColor(t.Tx3)
						ui.Text(c, "暂无数据").TextColor(t.Tx3)
					})
				}
			})
			ui.Column(c).Padding(4).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1).Children(func() {
				item := func(key, icon, text, right string, fn func()) {
					b := a.mi(c, "acct-"+key, icon, text, right, false).Height(theme.Px(28))
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
				item("about", "info", "关于 MiniCode", strings.TrimSuffix(Version, "-mygo"), func() { a.openSettings("about") })
			})
		})
	})
}

// avatarSized 是圆形头像(账号菜单头部用 30px)。
func (a *App) avatarSized(c *ui.Context, size, font float32, initial string) ui.Element {
	t := a.tk
	if initial == "" {
		initial = "?"
	}
	return ui.Row(c).Size(size, size).Radius(size / 2).Background(t.AcSoft).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Text(c, initial).FontSize(font).FontWeight(500).TextColor(t.AcTx)
	})
}

// openSettings 打开设置并跳到某个分类。
func (a *App) openSettings(tab string) {
	a.screen = ScreenSettings
	if tab != "" {
		a.setTab = resolveSetTabID(tab)
	} else if a.setTab != "" {
		a.setTab = resolveSetTabID(a.setTab)
	}
	a.set.query = ""
}

// dialogOpen 把居中对话框的 open 与 DialogOut 退场挂钩:关时先播动画再卸。
func (a *App) dialogOpen(key string, wantOpen bool, fn func(open *bool)) {
	if a.dlgClosing == key && kit.Since(a.dlgCloseAt) >= kit.DurFast {
		a.dlgClosing, a.dlgSeen = "", ""
	}
	hold := wantOpen
	if a.dlgClosing == key && kit.Since(a.dlgCloseAt) < kit.DurFast {
		hold = true
		a.frameSoon()
	} else if !wantOpen && a.dlgSeen == key && a.dlgClosing == "" {
		a.dlgClosing, a.dlgCloseAt = key, kit.Now()
		hold = true
		a.frameSoon()
	}
	if hold && a.dlgSeen != key {
		a.dlgSeen, a.dlgAt = key, kit.Now()
	}
	fn(&hold)
	if !hold && wantOpen && a.dlgClosing == "" {
		a.dlgClosing, a.dlgCloseAt = key, kit.Now()
		a.frameSoon()
	}
}

// dialogIn 是居中浮层(命令面板、快速选择)的入场 / 退场动画。
func (a *App) dialogIn(key string, backdrop, panel ui.Element) {
	if a.dlgClosing == key && kit.Since(a.dlgCloseAt) < kit.DurFast {
		kit.DialogOut(backdrop, panel, a.dlgCloseAt, a.frameSoon)
		return
	}
	if a.dlgSeen != key {
		a.dlgSeen, a.dlgAt = key, kit.Now()
	}
	kit.DialogIn(backdrop, panel, a.dlgAt, a.frameSoon)
}
