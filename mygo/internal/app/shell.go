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
		// 方案 B:底色带一抹强调色光晕(径向渐变)
		root.Background(t.Chrome)
		root.RadialGradient(ui.RadialGradient{
			Center: ui.Point{X: 0, Y: 0}, Radius: 1500,
			Stops: []ui.GradientStop{{Color: t.Chrome.Mix(t.Ac, .09), At: 0}, {Color: t.Chrome, At: .55}},
		})
	} else {
		root.Background(t.Chrome)
	}
	// 「你在哪」(⌘W 关标签还是关窗口,见 close_ctx.go):根上先记成「其它」,工作区 / 底部面板再各自覆盖
	root.ObserveInput(a.zoneObserve(zoneOther))
	a.zoneBegin()
	defer func() { a.zoneSync(root.FocusWithin()) }()
	root.Children(func() {
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
		a.guard(c, "动效总览", "motion", guardSilent, func() { a.motionCatalogOverlay(c) })
		a.guard(c, "浮层", "overlay", guardSilent, func() { a.overlayLayer(c) })
		a.guard(c, "图表查看器", "mermaid", guardSilent, func() { a.mermaidViewerOverlay(c) })
		a.guard(c, "图片灯箱", "lightbox", guardSilent, func() { a.lightboxOverlay(c) })
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
