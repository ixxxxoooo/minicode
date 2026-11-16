package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// T 是当前令牌的简写。
func (a *App) t() *theme.Tokens { return a.tk }

// root 是窗口根:标题栏 + 主体(图标轨 / 导航栏 / 舞台)+ 状态栏。
func (a *App) root(c *ui.Context) {
	t := a.tk
	root := ui.Column(c).Fill().FontSize(theme.FsBody).TextColor(t.Tx1)
	if t.Scheme.ID == "b" {
		// 方案 B:底色带一抹强调色光晕,画在 winGlow 里(root.Children 的第一个子元素)
		root.Background(t.Chrome).Clip()
	} else {
		root.Background(t.Chrome)
	}
	// 「你在哪」(⌘W 关标签还是关窗口,见 close_ctx.go):根上先记成「其它」,工作区 / 底部面板再各自覆盖
	root.ObserveInput(a.zoneObserve(zoneOther))
	a.zoneBegin()
	defer func() { a.zoneSync(root.FocusWithin()) }()
	root.Children(func() {
		a.winGlow(c)
		a.guard(c, "标题栏", "titlebar", guardChip, func() { a.titlebar(c) })
		if a.screen == ScreenSettings {
			a.guard(c, "设置", "settings", guardCard, func() { a.settingsPage(c) }) // 设置是整窗页面:没有导航栏与舞台
		} else {
			a.body(c)
		}
		a.guard(c, "状态栏", "statusbar", guardChip, func() { a.statusbar(c) })
		a.guard(c, "命令面板", "palette", guardSilent, func() { a.paletteView(c) })
		a.guard(c, "快速选择", "quick", guardSilent, func() { a.quickView(c) })
		a.guard(c, "项目对话框", "projdlg", guardSilent, func() { a.projDialogs(c) })
		a.guard(c, "浮层", "overlay", guardSilent, func() { a.overlayLayer(c) })
		a.guard(c, "图表查看器", "mermaid", guardSilent, func() { a.mermaidViewerOverlay(c) })
		if a.toast != "" { // 一句话轻提示统一走通知样式
			a.flash(a.toast)
			a.toast = ""
		}
		a.guard(c, "通知", "toasts", guardSilent, func() { a.noticeToasts(c) })
		a.morphShield(c)
		if a.delUndo.Active() {
			ui.Row(c).Justify(ui.Center).Padding(0, 0, 8, 0).Children(func() {
				kit.UndoBar(c, &a.delUndo)
			})
		}
	})
}

// body 是标题栏与状态栏之间的主体。
func (a *App) body(c *ui.Context) {
	t := a.tk
	gap := t.Scheme.GapPane
	ui.Row(c).Grow(1).MinHeight(0).PaddingX(gap).AlignItems(ui.Stretch).Children(func() {
		if t.Scheme.ID == "c" {
			a.rail(c)
		}
		a.guard(c, "导航栏", "side", guardCard, func() { a.side(c) })
		a.stage(c)
	})
}

// pane 是舞台里的一块面板:底色 BG;方案 A / C 左侧一条细线,方案 B 是带阴影的圆角岛。
func (a *App) pane(c *ui.Context) ui.Element {
	t := a.tk
	p := ui.Column(c).Background(t.BG).MinWidth(0).Clip()
	switch t.Scheme.ID {
	case "b":
		p.Radius(t.RLg())
		for _, s := range t.Shadow1() {
			p.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
	default:
		p.BorderWidth(0, 0, 0, 1).BorderColor(t.Bd2)
	}
	return p
}

// 面板过渡:宽度 / 外边距 440 ms 变形曲线(Agent ⇄ IDE)。
var morph = ui.ElementTransition{Size: true, Position: true, Duration: kit.DurMorph, Ease: kit.EaseMorph}

func (a *App) pagePlaceholder(c *ui.Context) {
	a.pane(c).Grow(1).Children(func() {
		ui.Text(c, string(a.screen)).Margin(24)
	})
}

// winGlow 画方案 B 窗口底色上的强调色光晕,与设计稿 shell.css 的
// radial-gradient(120% 90% at 0% 0%, 强调色 9%, 底色 55%) 一致:左上角为中心的椭圆,
// 横向半径 = 窗口宽 × 1.2,纵向半径 = 窗口高 × 0.9,随窗口大小一起缩放。
// 界面库的径向渐变只有正圆,所以先在一个更高的盒子里画半径 1.2 × 宽 的圆,再以左上角为原点纵向压扁成椭圆。
// 参数 c:界面上下文。无返回值;其它方案或窗口尺寸未知时不画。
func (a *App) winGlow(c *ui.Context) {
	t := a.tk
	w, h := a.winW, a.winH
	if t.Scheme.ID != "b" || w <= 0 || h <= 0 {
		return
	}
	rx, ry := w*1.2, h*.9
	sy := ry / rx // 纵向压缩比
	ui.Box(c).Key("win-glow").Absolute().ZIndex(-1).Left(0).Top(0).Size(w, h/sy).PassThrough().
		TransformOrigin(0, 0).Scale(1, sy).
		RadialGradient(ui.RadialGradient{
			// 中心写零值会被当成「盒子正中」,所以用一个极小的非零值表示左上角
			Center: ui.Point{X: 1e-6, Y: 1e-6}, Radius: rx,
			Stops: []ui.GradientStop{{Color: t.Chrome.Mix(t.Ac, .09), At: 0}, {Color: t.Chrome, At: .55}},
		})
}
