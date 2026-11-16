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
	updChecking
	updLatest
	updFound
	updDownload
	updVerify
	updUnpack
	updReady
	updFail
	updNetErr
)

type updState struct {
	Phase   updPhase
	Percent int // 0..100
	Version string
	Title   string // 失败时的标题(空 = 按阶段用默认文案)
	Msg     string // 失败时的一句原因 + 下一步(空 = 用默认文案)
	// Manual 只用于 updFound:发现了新版本但「自动下载更新」关闭,等用户点「下载并安装」。
	Manual bool
}

func (a *App) updLabel() string {
	if a.updSt.Phase == updNone {
		return a.update // 兼容旧的演示字符串
	}
	switch a.updSt.Phase {
	case updChecking:
		return "检查更新…"
	case updLatest:
		return "已是最新"
	case updFound:
		return "发现新版本"
	case updDownload:
		return fmt.Sprintf("下载更新 %d%%", a.updSt.Percent)
	case updVerify:
		return "校验中…"
	case updUnpack:
		return "解包中…"
	case updReady:
		return "重启更新"
	case updFail:
		return a.updFailTitle("下载失败")
	case updNetErr:
		return a.updFailTitle("网络错误")
	}
	return ""
}

// updFailTitle 失败标题:带了具体标题就用它,否则用默认。
func (a *App) updFailTitle(def string) string {
	if a.updSt.Title != "" {
		return a.updSt.Title
	}
	return def
}

func (a *App) updSidebarCapsule(c *ui.Context) {
	if a.updSt.Phase == updNone && a.update == "" {
		return
	}
	t := a.tk
	lbl := a.updLabel()
	row := ui.ButtonBase(c).Key("upd-cap").Height(theme.Px(26)).PaddingX(8).Gap(6).Radius(t.RSm()).Background(t.AcSoft).AlignItems(ui.Center).Margin(4, 0, 0, 0).Cursor(ui.CursorPointer)
	row.OnClick(func() {
		if a.updSt.Phase == updReady {
			a.applyUpdateRestart()
			return
		}
		if a.updSt.Phase == updNone && a.upd == nil {
			a.checkUpdateAsync()
		}
		a.togglePop("upd")
	})
	row.Children(func() {
		ic := "download"
		switch a.updSt.Phase {
		case updChecking, updLatest:
			ic = "refresh"
		case updVerify:
			ic = "shieldCk"
		case updUnpack:
			ic = "cube"
		case updReady:
			ic = "refresh"
		case updFail, updNetErr:
			ic = "alert"
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
			title := "发现新版本 " + ver
			switch a.updSt.Phase {
			case updChecking:
				title = "正在检查更新…"
			case updLatest:
				title = "已是最新版本"
			case updFail:
				title = a.updFailTitle("下载失败")
			case updNetErr:
				title = a.updFailTitle("网络错误")
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, title).FontWeight(600).Grow(1)
				if a.updSt.Phase == updDownload {
					kit.Tag(c, itoa(a.updSt.Percent)+"%", kit.KAc)
				}
			})
			switch a.updSt.Phase {
			case updChecking:
				kit.Meter(c, 30)
				ui.Text(c, "正在查询 appcast…").FontSize(theme.FsCap).TextColor(t.Tx3)
			case updLatest:
				ui.Text(c, "当前版本已是最新。").FontSize(theme.FsCap).TextColor(t.Tx3)
			case updFound:
				if a.updSt.Manual {
					// 偏好「自动下载更新」关闭:不动网络,等用户确认
					ui.Text(c, "发现 "+ver+",没有自动下载(设置 → 关于)。").FontSize(theme.FsCap).TextColor(t.Tx3)
					kit.Btn(c, "下载并安装", kit.BtnOpt{Variant: kit.BtnPri, Small: true, Icon: "download"}).OnClick(a.startUpdateDownload)
				} else {
					kit.Meter(c, 5)
					ui.Text(c, "发现 "+ver+",准备下载…").FontSize(theme.FsCap).TextColor(t.Tx3)
				}
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
				kit.Btn(c, "重启更新", kit.BtnOpt{Variant: kit.BtnPri, Small: true, Icon: "refresh"}).OnClick(a.applyUpdateRestart)
			case updFail:
				msg := a.updSt.Msg
				if msg == "" {
					msg = "校验失败,安装包可能损坏。"
				}
				ui.Text(c, msg).FontSize(theme.FsCap).TextColor(t.Err)
				kit.Btn(c, "重试", kit.BtnOpt{Small: true, Icon: "refresh"}).OnClick(a.retryUpdate)
			case updNetErr:
				msg := a.updSt.Msg
				if msg == "" {
					msg = "网络连接丢失,稍后再试。"
				}
				ui.Text(c, msg).FontSize(theme.FsCap).TextColor(t.Err)
				kit.Btn(c, "重试", kit.BtnOpt{Small: true, Icon: "refresh"}).OnClick(a.retryUpdate)
			default:
				kit.Meter(c, 42)
				ui.Text(c, "正在后台下载,完成后提示重启。").FontSize(theme.FsCap).TextColor(t.Tx3)
			}
		})
	})
}

// updAboutHint 关于页版本卡下的更新状态文案(演示 / 真实更新流程共用)。
func (a *App) updAboutHint() (txt string, ok bool) {
	if a.updSt.Phase == updNone {
		return "", false
	}
	switch a.updSt.Phase {
	case updChecking:
		return "正在检查更新…", true
	case updLatest:
		return "已是最新", true
	case updDownload:
		return fmt.Sprintf("正在下载 %d%%", a.updSt.Percent), true
	case updVerify:
		return "校验中…", true
	case updUnpack:
		return "解包中…", true
	case updReady:
		return "可重启更新", true
	case updFail:
		return a.updFailTitle("下载失败"), true
	case updNetErr:
		return a.updFailTitle("网络错误"), true
	case updFound:
		if a.updSt.Manual {
			return "发现新版本 " + a.updSt.Version + "(未自动下载)", true
		}
		return "发现新版本 " + a.updSt.Version, true
	}
	return "", false
}

// SeedUpdateDemo 演示更新各阶段(截图 / 调试 / 命令面板)。
func (a *App) SeedUpdateDemo(phase string, pct int) {
	ver := "0.3.1"
	switch phase {
	case "check":
		a.updSt = updState{Phase: updChecking, Version: ver}
	case "latest":
		a.updSt = updState{Phase: updLatest, Version: ver}
	case "found":
		a.updSt = updState{Phase: updFound, Version: ver}
	case "verify":
		a.updSt = updState{Phase: updVerify, Version: ver}
	case "unpack":
		a.updSt = updState{Phase: updUnpack, Version: ver}
	case "ready":
		a.updSt = updState{Phase: updReady, Version: ver}
	case "fail":
		a.updSt = updState{Phase: updFail, Version: ver}
	case "neterr", "network":
		a.updSt = updState{Phase: updNetErr, Version: ver}
	default:
		if pct <= 0 {
			pct = 42
		}
		a.updSt = updState{Phase: updDownload, Percent: pct, Version: ver}
	}
	a.update = a.updLabel()
}
