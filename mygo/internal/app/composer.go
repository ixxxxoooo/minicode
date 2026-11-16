package app

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// composer 是输入框(设计稿 .cmp):文本区 + 工具栏(附件 / 运行配置胶囊 / MCP / 上下文环 / 发送)。
func (a *App) composer(c *ui.Context) {
	t := a.tk
	r := t.RLg()
	if t.Scheme.ID == "b" {
		r = t.RXl()
	} else if t.Scheme.ID == "c" {
		r = t.RMd()
	}
	box := ui.Column(c).Key("composer").Background(t.Raised).Radius(r).Border(1, t.Bd2).Transition(ui.ElementTransition{Colors: true, Shadows: true, Duration: kit.DurBase, Ease: kit.EaseOut})
	if !a.composerInAt.IsZero() {
		box = kit.PopIn(box, a.composerInAt)
	}
	focus := box.FocusWithin() || box.Hovered()
	if focus {
		box.BorderColor(t.AcLine)
		box.Shadow(0, 0, 0, 3, t.Ring())
	}
	for _, s := range t.Shadow1() {
		box.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	_ = a.chatDrop(box)
	anchor := box
	box.Children(func() {
		a.composerRefChips(c)
		a.composerAttachRow(c)
		in := ui.TextArea(c.Key("draft"), &a.draft).Lines(1, 8).Placeholder(a.composerHint()).
			PlaceholderColor(t.Tx3).Background(ui.Transparent).BorderWidth(0).Padding(9, 12, 3, 12).FontSize(a.FlowFontSize()).LineHeight(1.55).
			MinHeight(36).TextColor(t.Tx1).FocusRing(false)
		if a.welcomeFocus {
			in.AutoFocus()
		}
		if a.ux.focusDraft { // 拖入引用之后接着写
			in.Focus()
			a.ux.focusDraft = false
		}
		in.OnPaste(func(text string) bool {
			return a.HandleComposerPaste(text)
		})
		in.OnTextKey(func(k ui.TextKey) bool {
			if a.slashTextKey(k) {
				return true
			}
			if k.Key != ui.KeyEnter || (k.Mods != 0 && k.Mods != ui.Cmd) {
				return false
			}
			cmdSend := a.prefs.Int(pSend, 0) == 1 // 「⌘ Enter 发送」:Enter 只换行
			if cmdSend && k.Mods == 0 {
				return false
			}
			// 回合进行中:发送键按「再发消息」偏好排队或引导;另一个修饰键(Enter 发送模式下的 ⌘Enter)取相反
			steer := a.prefs.Int(pBusy, 0) == 1
			if !cmdSend && k.Mods == ui.Cmd {
				steer = !steer
			}
			txt := k.Text()
			k.SetText("")
			a.submitKey(txt, steer)
			return true
		})
		a.composerBar(c)
	})
	if a.slashOpen() {
		a.slashPopup(c, anchor)
	}
	a.cfgPopup(c, "cfg", box, false)
}

// submitKey 回车发送:回合进行中 Enter 排队、⌘Enter 立即引导;空闲时都是直接发送。
func (a *App) submitKey(text string, steer bool) {
	if trim(text) == "" && len(parseDraftAttaches(text)) == 0 {
		return
	}
	if a.st.Phase == core.PhaseRunning {
		if steer {
			a.steerSend(text)
		} else if a.q.add(a.queueDraftFor(text)) {
			log.Info("排队", "len", len(text), "队列", a.q.len())
		}
		return
	}
	a.submitPrompt(text, false)
}

// submit 发送一条消息。
func (a *App) submit(text string) { a.submitPrompt(text, false) }

func (a *App) composerBar(c *ui.Context) {
	t := a.tk
	wide := a.chatTier() != tierNarrow // 窄档:省略「思考」、百分比与 MCP 字样
	ui.Row(c).Padding(3, 5, 5, 5).Gap(2).AlignItems(ui.Center).Children(func() {
		plus := kit.Ib(c, "plus", kit.IbOpt{Tip: "添加附件", On: a.popup == "cmp-attach"})
		plus.OnClick(func() { a.togglePop("cmp-attach") })
		a.plusAttachMenu(c, plus)
		a.cfgChip(c, wide)
		mcpAnchor := a.mcpChip(c, wide)
		a.mcpPopup(c, mcpAnchor)
		ui.Spacer(c).Grow(1)
		// 上下文环(点击展开与状态栏同内容的弹层)
		pct := a.st.Usage.ContextRatio()
		ctxBtn := ui.ButtonBase(c).Key("ctx-chip").Height(22).PaddingX(7).Gap(4).Radius(t.RSm()).AlignItems(ui.Center).Cursor(ui.CursorPointer).Transition(kit.Fade)
		if ctxBtn.Hovered() || a.popup == "ctx-cmp" {
			ctxBtn.Background(t.Fill2)
		}
		ctxBtn.OnClick(func() { a.togglePop("ctx-cmp") })
		ctxBtn.Children(func() {
			if wide {
				col := t.Tx2
				if pct >= a.ctxWarnRatio() {
					col = t.Warn
				}
				if pct >= 0.95 {
					col = t.Err
				}
				ui.Text(c, itoa(int(pct*100+.5))+"%").FontSize(11.5).TextColor(col).FontFeatures("tnum")
			}
			a.ringColored(c, pct)
		})
		a.ctxAnchorPopup(c, "ctx-cmp", ctxBtn, ui.AnchorTopRight)
		// 发送 / 停止是 24×24 的圆形按钮(半径 12)
		if a.st.Phase == core.PhaseRunning {
			b := kit.Ib(c, "stop", kit.IbOpt{Tip: "停止", NoHover: true}).Background(t.Tx1).Radius(12).TextColor(t.BG)
			b.OnClick(func() { a.send(core.CmdCancel{}) })
		} else {
			b := kit.Ib(c, "send", kit.IbOpt{Tip: "发送 ⏎", NoHover: true}).Background(t.Ac).Radius(12).TextColor(t.OnAc)
			if b.Hovered() {
				b.Background(t.Ac.Mix(t.Tx1, .10))
			}
			b.OnClick(func() {
				txt := a.draft
				a.submitKey(txt, false)
			})
		}
	})
}

// cfgChip 是「运行配置胶囊」:权限模式 · 模型 · 思考强度合在一个弹层入口里。
func (a *App) cfgChip(c *ui.Context, wide bool) {
	t := a.tk
	b := ui.ButtonBase(c).Height(22).PaddingX(7).Gap(5).Radius(t.RSm()).AlignItems(ui.Center).FontSize(11.5).TextColor(t.Tx2).Cursor(ui.CursorPointer).
		MinWidth(0).Clip().Transition(kit.Fade)
	if b.Hovered() || a.popup == "cfg" {
		b.Background(t.Fill2).TextColor(t.Tx1)
	}
	b.OnClick(func() { a.togglePop("cfg") })
	if tip := a.pvModelChipTip(); tip != "" {
		b.Tooltip(tip)
	}
	b.Children(func() {
		div := func() { ui.Box(c).Size(1, 10).Background(t.Bd3).Shrink(0) }
		kit.Ic(c, modeIcon(a.st.Mode), kit.S12).TextColor(a.modeTint(a.st.Mode))
		ui.Text(c, a.st.Mode.Label()).SingleLine().Shrink(0)
		div()
		if prov, model := a.pvModelLabel(); prov != "" {
			_, pr := a.pvActive()
			pb := ""
			if pr != nil {
				pb = provBrand(pr.Name, pr.BaseURL)
			}
			kit.Logo(c, modelBrand(model, pb), 13, t.Tx2)
		} else {
			kit.Logo(c, "claude", 13, t.Tx2)
		}
		ui.Text(c, a.modelName()).SingleLine().Shrink(1)
		if wide {
			div()
			kit.Ic(c, "brain", kit.S12)
			ui.Text(c, effortLabel(a.st.Effort)).SingleLine().Shrink(0)
		}
		kit.Ic(c, "chevD", kit.S12).Opacity(.5).Margin(0, 0, 0, -1)
	})
}

// modeTint 权限模式的语义色:逐项确认=强调色、自动接受编辑=绿、计划模式=紫、跳过权限确认=警示橙
// (设计稿 providers.js 的 perm 列表);其余模式沿用最接近的一种。
func (a *App) modeTint(m core.PermissionMode) ui.Color {
	t := a.tk
	switch m {
	case core.ModeAcceptEdits:
		return t.Ok
	case core.ModePlan:
		return t.Vio
	case core.ModeBypassPermissions:
		return t.Warn
	}
	return t.Ac
}

func modeIcon(m core.PermissionMode) string {
	switch m {
	case core.ModeAcceptEdits:
		return "edit"
	case core.ModePlan:
		return "list"
	case core.ModeBypassPermissions:
		return "bolt"
	}
	return "shield"
}

func effortLabel(e core.Effort) string {
	switch e {
	case core.EffortLow:
		return "低"
	case core.EffortMedium:
		return "中"
	case core.EffortHigh:
		return "高"
	case core.EffortXHigh:
		return "超高"
	case core.EffortMax:
		return "最大"
	}
	return "中"
}

// pvModelChipTip 第三方胶囊 tooltip;官方返回空。
func (a *App) pvModelChipTip() string {
	_, tip := a.pvModelChipText()
	return tip
}

// modelName 当前模型的显示名。
func (a *App) modelName() string {
	if prov, _ := a.pvModelLabel(); prov != "" { // 第三方:缩写或简短模型名(见 providers.js cfgChipHtml)
		short, _ := a.pvModelChipText()
		return short
	}
	for _, m := range a.st.Models {
		if m.ID == a.st.Model {
			return m.Name
		}
	}
	return core.TurnStats{Model: a.st.Model}.ModelLabel() // 认不出的原样显示
}

// ringColored 是 16px 的圆环进度(上下文占用)。
func (a *App) ringColored(c *ui.Context, p float32) {
	t := a.tk
	col := a.ctxRingColor(p)
	e := ui.Box(c).Size(16, 16).Shrink(0)
	e.Draw(func(pt *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		const rad = 6.4
		arc := func(frac float32, c ui.Color) {
			n := max(2, int(frac*40))
			pts := make([][2]float32, 0, n+1)
			for i := 0; i <= n; i++ {
				ang := -math.Pi/2 + float64(i)/float64(n)*float64(frac)*2*math.Pi
				pts = append(pts, [2]float32{cx + rad*float32(math.Cos(ang)), cy + rad*float32(math.Sin(ang))})
			}
			pt.StrokePolyline(pts, 3, c)
		}
		arc(1, t.Fill3)
		if p > 0 {
			arc(min(p, .999), col)
		}
	})
}

// composerHint 输入框占位:回合进行中写明 ⏎ / ⌘⏎ 各自的含义,不靠用户记快捷键。
func (a *App) composerHint() string {
	if a.st.Phase == core.PhaseRunning {
		e, m := a.busyKeys()
		return "Claude 正在工作 · ⏎ " + e + ",⌘⏎ " + m
	}
	return "描述任务,@ 引用文件,/ 调用命令"
}
