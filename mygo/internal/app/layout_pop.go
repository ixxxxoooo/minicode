package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// @author ixxxxoooo

// 布局浮窗里每个方案的一句话(与设置 → 外观的布局卡同源)。
var layTagline = map[string][2]string{
	"a": {"Ink", "同一平面,细线分隔"},
	"b": {"Float", "面板浮在底色上,圆润柔和"},
	"c": {"Rail", "图标轨常驻,步骤串成时间轴"},
}

// layoutPopup 是标题栏「布局」图标点开的浮窗:墨线 / 浮岛 / 轨道三行(微缩示意 + 名称 + 一句话 + 当前对勾),点哪行立即切换。
// 参数:anchor 是触发它的图标按钮。弹层开关走 a.popup("layout"),点外部 / Esc / 再点图标收起。
func (a *App) layoutPopup(c *ui.Context, anchor ui.Element) {
	t := a.tk
	a.popOpen("layout", func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 320).Padding(6).AttachTo(anchor, ui.AnchorTopRight, ui.AnchorBottomRight).Margin(0, 0, 6, 0)
			ui.Row(c).Gap(8).Padding(4, 8, 6, 8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "布局").FontWeight(500)
				ui.Text(c, "点一下立即切换,Agent 与 IDE 同步").FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Shrink(1).MinWidth(0)
			})
			for _, s := range theme.Schemes {
				a.layoutRow(c, s)
			}
			ui.Text(c, "也可在命令面板输入「布局」,或在 设置 → 外观 里切换").FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(1.4).
				Padding(8, 8, 2, 8).Margin(4, 0, 0, 0).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1)
		})
	})
}

// layoutRow 画浮窗里的一行方案。
func (a *App) layoutRow(c *ui.Context, s theme.Scheme) {
	t := a.tk
	on := t.Scheme.ID == s.ID
	row := ui.ButtonBase(c).Key("layrow-"+s.ID).Padding(6, 8).Gap(10).Radius(t.RMd()).AlignItems(ui.Center).Cursor(ui.CursorPointer).Transition(kit.Fade)
	switch {
	case on:
		row.Background(t.AcSoft).Border(1.5, t.Ac)
	case row.Hovered():
		row.Background(t.Fill1)
	}
	row.OnClick(func() {
		a.popup = ""
		if !on {
			log.Info("切换布局", "from", t.Scheme.ID, "to", s.ID)
			a.setScheme(s.ID)
		}
	})
	row.Children(func() {
		layPreviewSized(a, c, s.ID, 96, 56)
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, s.Name).FontWeight(500)
				ui.Text(c, layTagline[s.ID][0]).FontSize(theme.FsMicro).TextColor(t.Tx4)
			})
			ui.Text(c, layTagline[s.ID][1]).FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(1.4)
		})
		if on {
			kit.Ic(c, "check", kit.S14).TextColor(t.Ac)
		}
	})
}
