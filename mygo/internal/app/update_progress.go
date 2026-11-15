package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// updPhase 是更新胶囊的四态 + 可重启。
type updPhase int

const (
	updNone updPhase = iota
	updDownload
	updVerify
	updUnpack
	updReady
)

type updState struct {
	Phase   updPhase
	Percent int // 0..100
	Version string
}

func (a *App) updLabel() string {
	if a.updSt.Phase == updNone {
		return a.update // 兼容旧的演示字符串
	}
	switch a.updSt.Phase {
	case updDownload:
		return fmt.Sprintf("下载更新 %d%%", a.updSt.Percent)
	case updVerify:
		return "校验中…"
	case updUnpack:
		return "解包中…"
	case updReady:
		return "重启更新"
	}
	return ""
}

func (a *App) updSidebarCapsule(c *ui.Context) {
	if a.updSt.Phase == updNone && a.update == "" {
		return
	}
	t := a.tk
	lbl := a.updLabel()
	row := ui.ButtonBase(c).Key("upd-cap").Height(26).PaddingX(8).Gap(6).Radius(t.RSm()).Background(t.AcSoft).AlignItems(ui.Center).Margin(4, 0, 0, 0).Cursor(ui.CursorPointer)
	row.OnClick(func() {
		if a.updSt.Phase == updReady {
			a.toast = "更新已就绪(开发版需手动替换包)"
			return
		}
		a.togglePop("upd")
	})
	row.Children(func() {
		ic := "download"
		switch a.updSt.Phase {
		case updVerify:
			ic = "shieldCk"
		case updUnpack:
			ic = "cube"
		case updReady:
			ic = "refresh"
		}
		kit.Ic(c, ic, kit.S12).TextColor(t.Ac)
		ui.Text(c, lbl).TextColor(t.Ac).Grow(1).SingleLine()
		if a.updSt.Phase == updDownload && a.updSt.Percent > 0 {
			kit.Meter(c, float32(a.updSt.Percent)).Width(54)
		}
	})
}

func (a *App) updPopup(c *ui.Context, anchor ui.Element) {
	t := a.tk
	a.popOpen("upd", func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 260).Padding(10, 12).Gap(8).AttachTo(anchor, ui.AnchorTopRight, ui.AnchorBottomRight).Margin(0, 0, 6, 0)
			ver := a.updSt.Version
			if ver == "" {
				ver = "0.2.1"
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "发现新版本 "+ver).FontWeight(600).Grow(1)
				if a.updSt.Phase == updDownload {
					kit.Tag(c, itoa(a.updSt.Percent)+"%", kit.KAc)
				}
			})
			switch a.updSt.Phase {
			case updDownload:
				kit.Meter(c, float32(a.updSt.Percent))
				ui.Text(c, "正在后台下载,完成后提示重启。").FontSize(theme.FsCap).TextColor(t.Tx3)
			case updVerify:
				kit.Meter(c, 100)
				ui.Text(c, "正在校验下载包…").FontSize(theme.FsCap).TextColor(t.Tx3)
			case updUnpack:
				kit.Meter(c, 100)
				ui.Text(c, "正在解包到临时目录…").FontSize(theme.FsCap).TextColor(t.Tx3)
			case updReady:
				ui.Text(c, "更新已准备好,重启后生效。").FontSize(theme.FsCap).TextColor(t.Tx3)
				kit.Btn(c, "重启更新", kit.BtnOpt{Variant: kit.BtnPri, Small: true, Icon: "refresh"}).OnClick(func() {
					a.toast = "开发版请用 scripts/dev.sh 重新编译"
				})
			default:
				kit.Meter(c, 42)
				ui.Text(c, "正在后台下载,完成后提示重启。").FontSize(theme.FsCap).TextColor(t.Tx3)
			}
		})
	})
}

// SeedUpdateDemo 演示四态(截图 / 调试)。
func (a *App) SeedUpdateDemo(phase string, pct int) {
	switch phase {
	case "verify":
		a.updSt = updState{Phase: updVerify, Version: "0.2.1"}
	case "unpack":
		a.updSt = updState{Phase: updUnpack, Version: "0.2.1"}
	case "ready":
		a.updSt = updState{Phase: updReady, Version: "0.2.1"}
	default:
		a.updSt = updState{Phase: updDownload, Percent: pct, Version: "0.2.1"}
	}
	a.update = a.updLabel()
}
