package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 整页屏幕在面包屑里的名字。
var crumbName = map[Screen]string{
	ScreenAutomation: "自动化", ScreenCanvas: "画布", ScreenBrowser: "浏览器", ScreenCustomize: "自定义", ScreenUsage: "用量与成本",
}

// titlebar 是标题栏(36 高):三套方案的左 / 中 / 右内容不同,但 Agent|IDE 分段始终居中或居左。
func (a *App) titlebar(c *ui.Context) {
	t := a.tk
	bar := c.TitleBar()
	row := ui.Row(c).Height(theme.HBar).PaddingX(10).Gap(8).Shrink(0).DragWindow()
	if t.Scheme.ID != "b" {
		row.BorderWidth(0, 0, 1, 0).BorderColor(t.Bd2)
	}
	row.Children(func() {
		// 左
		ui.Row(c).Grow(1).Basis(0).MinWidth(0).Gap(4).Children(func() {
			a.lights(c, bar.Left)
			kit.Ib(c, "panelL", kit.IbOpt{Tip: "导航栏 ⌘B", On: false}).OnClick(func() { a.sideOpen = !a.sideOpen })
			switch t.Scheme.ID {
			case "b":
				a.modeSeg(c)
			default:
				a.crumb(c)
			}
		})
		// 中
		ui.Row(c).Gap(8).Shrink(0).Children(func() {
			switch t.Scheme.ID {
			case "a":
				a.modeSeg(c)
			case "b":
				a.cmdk(c)
			case "c":
				a.modeSeg(c)
				a.cmdk(c)
			}
		})
		// 右
		ui.Row(c).Grow(1).Basis(0).MinWidth(0).Gap(4).Justify(ui.End).Children(func() {
			if t.Scheme.ID == "a" {
				a.cmdk(c)
			}
			if a.mode == ModeIDE {
				kit.Ib(c, "panelB", kit.IbOpt{Tip: "终端 ⌃`"})
			} else {
				kit.Ib(c, "panelR", kit.IbOpt{Tip: "检查器 ⌥⌘B", On: false}).OnClick(func() { a.inspOpen = !a.inspOpen })
			}
			kit.Ib(c, "bell", kit.IbOpt{Tip: "通知"})
			if t.Scheme.ID == "b" {
				a.avatar(c).Margin(0, 0, 0, 4)
			}
		})
	})
}

// lights 给系统红绿灯留位;没有系统红绿灯(测试 / 截图)时自己画三个圆点。
func (a *App) lights(c *ui.Context, room float32) {
	if room > 0 {
		ui.Box(c).Width(max(0, room-10)).Height(1).Shrink(0)
		return
	}
	if !a.fakeLights {
		return
	}
	ui.Row(c).Gap(7).Padding(0, 6, 0, 2).Shrink(0).Children(func() {
		for _, col := range []string{"#FF5F57", "#FEBC2E", "#28C840"} {
			ui.Box(c).Size(11.5, 11.5).Radius(6).Background(ui.Hex(col))
		}
	})
}

// modeSeg 是 Agent | IDE 分段(⌘E)。
func (a *App) modeSeg(c *ui.Context) {
	on := 0
	if a.mode == ModeIDE {
		on = 1
	}
	i := kit.Seg(c, []kit.SegItem{{Icon: "bot", Text: "Agent", Tip: "⌘E"}, {Icon: "code", Text: "IDE", Tip: "⌘E"}}, on, kit.SegOpt{Large: true, W: 62, Key: "modeseg"})
	if i >= 0 {
		a.SetMode(Mode(i))
	}
}

// crumb 是面包屑:项目名 › 当前会话 / 页面名。
func (a *App) crumb(c *ui.Context) {
	t := a.tk
	title := core.ShortTitle(a.st.Title)
	if n, ok := crumbName[a.screen]; ok {
		title = n
	}
	ui.Row(c).Gap(5).PaddingStart(4).MinWidth(0).TextColor(t.Tx3).Children(func() {
		kit.Ic(c, "folder", kit.S12)
		ui.Text(c, "router-demo").SingleLine()
		kit.Ic(c, "chevR", kit.S12)
		ui.Text(c, title).SingleLine().FontWeight(500).TextColor(t.Tx1)
	})
}

// cmdk 是 ⌘K 命令栏。
func (a *App) cmdk(c *ui.Context) {
	t := a.tk
	b := ui.ButtonBase(c).Width(250).Height(24).Radius(t.RMd()).PaddingStart(8).PaddingEnd(6).Gap(6).AlignItems(ui.Center).
		FontSize(11.5).TextColor(t.Tx3).Cursor(ui.CursorPointer).Shrink(0)
	switch {
	case t.Scheme.ID == "b":
		b.Radius(theme.RFull).Background(t.Raised.Alpha(.7))
		for _, s := range t.Shadow1() {
			b.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
	case b.Hovered():
		b.Background(t.Fill2).Border(1, t.Bd2)
	default:
		b.Background(t.Fill1)
	}
	b.Transition(kit.Fade).OnClick(func() { a.paletteOn = true })
	b.Children(func() {
		kit.Ic(c, "search", kit.S12)
		ui.Text(c, "搜索会话、文件与命令").Grow(1).SingleLine()
		kit.Kbd(c, "⌘", "K")
	})
}

// avatar 是 20px 圆形头像(账号菜单入口)。
func (a *App) avatar(c *ui.Context) ui.Element {
	t := a.tk
	return ui.ButtonBase(c).Size(20, 20).Radius(10).Background(t.AcSoft).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).
		Tooltip("账号与设置").Label("账号与设置").Children(func() {
		ui.Text(c, "L").FontSize(10).FontWeight(600).TextColor(t.AcTx)
	})
}
