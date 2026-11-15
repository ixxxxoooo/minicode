package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// rail 是方案 C 的图标轨(44 宽):导航分段与整页入口都在这里,两种形态都不动。
func (a *App) rail(c *ui.Context) {
	t := a.tk
	ui.Column(c).Width(theme.WRail).Shrink(0).AlignItems(ui.Center).Gap(2).Padding(4, 0, 8, 0).
		BorderWidth(0, 1, 0, 0).BorderColor(t.Bd1).Children(func() {
		ri := func(icon, tip string, on bool, badge int) ui.Element {
			b := ui.ButtonBase(c).Key("ri-"+icon).Size(32, 30).Radius(t.RMd()).Justify(ui.Center).AlignItems(ui.Center).TextColor(t.Tx3).Cursor(ui.CursorPointer).Tooltip(tip).Label(tip).Transition(kit.Fade)
			switch {
			case on:
				b.Background(t.Fill2).TextColor(t.Tx1)
			case b.Hovered():
				b.Background(t.Fill1).TextColor(t.Tx1)
			}
			b.Children(func() {
				// 选中指示条:左侧 2px 强调线,纵向缩放弹出
				ln := ui.Box(c).Absolute().Left(-6).Top(7).Bottom(7).Width(2).Radius(2).Background(t.Ac)
				s := float32(0)
				if on {
					s = 1
				}
				ln.Scale(1, ln.AnimateTo("s", s, ui.Animation{Duration: kit.DurBase, Ease: kit.EaseSpring}))
				kit.Ic(c, icon, kit.S16)
				if badge > 0 {
					kit.Cnt(c, itoa(badge), true).Absolute().Right(-1).Top(0)
				}
			})
			return b
		}
		for _, n := range navs {
			badge := 0
			if n.id == "scm" {
				badge = 3
			}
			if ri(n.icon, n.name, a.NavID() == n.id && a.screen == ScreenWork, badge).Clicked() {
				a.nav = n.id
				a.leavePage()
			}
		}
		ui.Box(c).Size(20, 1).Background(t.Bd1).Margin(6, 0)
		for _, p := range pages {
			if ri(p.icon, p.name, a.screen == p.screen, 0).Clicked() {
				a.togglePage(p.screen)
			}
		}
		ui.Spacer(c).Grow(1)
		g := ri("gear", "账号与设置", a.popup == "acct@gear", 0)
		g.OnClick(func() { a.togglePop("acct@gear") })
		a.acctPopup(c, "gear", g)
		a.avatar(c).Margin(4, 0, 0, 0)
	})
}
