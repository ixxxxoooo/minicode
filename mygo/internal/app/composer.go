package app

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
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
	focus := box.FocusWithin() || box.Hovered()
	if focus {
		box.BorderColor(t.AcLine)
		box.Shadow(0, 0, 0, 3, t.Ring())
	}
	for _, s := range t.Shadow1() {
		box.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	box.Children(func() {
		in := ui.TextArea(c.Key("draft"), &a.draft).Lines(1, 8).Placeholder("描述任务,@ 引用文件,/ 调用命令").
			PlaceholderColor(t.Tx3).Background(ui.Transparent).BorderWidth(0).Padding(9, 12, 3, 12).FontSize(theme.FsRead).LineHeight(1.55).
			MinHeight(36).TextColor(t.Tx1).FocusRing(false)
		in.OnTextKey(func(k ui.TextKey) bool {
			if k.Key == ui.KeyEnter && k.Mods == 0 {
				txt := k.Text()
				k.SetText("")
				a.submit(txt)
				return true
			}
			return false
		})
		a.composerBar(c)
	})
}

// submit 发送一条消息。
func (a *App) submit(text string) {
	if text == "" {
		return
	}
	log.Info("发送提示", "len", len(text))
	a.send(core.CmdPrompt{Text: text})
}

func (a *App) composerBar(c *ui.Context) {
	t := a.tk
	wide := a.mode == ModeAgent // IDE 对话栏窄:省略「思考」与百分比
	ui.Row(c).Padding(3, 5, 5, 5).Gap(2).AlignItems(ui.Center).Children(func() {
		kit.Ib(c, "plus", kit.IbOpt{Tip: "添加附件"})
		a.cfgChip(c, wide)
		kit.Chip(c, "MCP "+itoa(len(a.st.Mcp)+3), kit.ChipOpt{Icon: "plug"})
		ui.Spacer(c).Grow(1)
		// 上下文环
		pct := int(a.st.Usage.ContextRatio()*100 + .5)
		ui.Row(c).Height(22).PaddingX(7).Gap(4).AlignItems(ui.Center).Children(func() {
			if wide {
				ui.Text(c, itoa(pct)+"%").FontSize(11.5).TextColor(t.Tx2).FontFeatures("tnum")
			}
			a.ring(c, float32(pct)/100)
		})
		if a.st.Phase == core.PhaseRunning {
			b := kit.Ib(c, "stop", kit.IbOpt{Tip: "停止", NoHover: true}).Background(t.Tx1).Radius(t.RMd()).TextColor(t.BG)
			b.OnClick(func() { a.send(core.CmdCancel{}) })
		} else {
			b := kit.Ib(c, "send", kit.IbOpt{Tip: "发送 ⏎", NoHover: true}).Background(t.Ac).Radius(t.RMd()).TextColor(t.OnAc)
			if b.Hovered() {
				b.Background(t.Ac.Mix(t.Tx1, .10))
			}
			b.OnClick(func() { a.submit(a.draft); a.draft = "" })
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
	b.OnClick(func() {
		if a.popup == "cfg" {
			a.popup = ""
		} else {
			a.popup = "cfg"
		}
	})
	b.Children(func() {
		div := func() { ui.Box(c).Size(1, 10).Background(t.Bd3).Shrink(0) }
		kit.Ic(c, modeIcon(a.st.Mode), kit.S12)
		ui.Text(c, a.st.Mode.Label()).SingleLine().Shrink(0)
		div()
		ui.Text(c, a.modelName()).SingleLine().Shrink(1)
		if wide {
			div()
			kit.Ic(c, "brain", kit.S12)
			ui.Text(c, effortLabel(a.st.Effort)).SingleLine().Shrink(0)
		}
		kit.Ic(c, "chevD", kit.S12).Opacity(.5).Margin(0, 0, 0, -1)
	})
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

// modelName 当前模型的显示名。
func (a *App) modelName() string {
	for _, m := range a.st.Models {
		if m.ID == a.st.Model {
			return m.Name
		}
	}
	return "Sonnet 5.5"
}

// ring 是 16px 的圆环进度(上下文占用):底圈 fill-3,进度弧强调色,从 12 点钟起顺时针。
func (a *App) ring(c *ui.Context, p float32) {
	t := a.tk
	e := ui.Box(c).Size(16, 16).Shrink(0)
	e.Draw(func(pt *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		const rad = 6.4
		arc := func(frac float32, col ui.Color) {
			n := max(2, int(frac*40))
			pts := make([][2]float32, 0, n+1)
			for i := 0; i <= n; i++ {
				ang := -math.Pi/2 + float64(i)/float64(n)*float64(frac)*2*math.Pi
				pts = append(pts, [2]float32{cx + rad*float32(math.Cos(ang)), cy + rad*float32(math.Sin(ang))})
			}
			pt.StrokePolyline(pts, 3, col)
		}
		arc(1, t.Fill3)
		if p > 0 {
			arc(min(p, .999), t.Ac)
		}
	})
}
