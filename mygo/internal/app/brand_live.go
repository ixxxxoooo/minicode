package app

import (
	"image/color"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/macos"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 设置 → 应用图标 / 菜单栏图标里的「动态预览」:
// 用真实素材与真实偏好(样式 / 外观 / 配色 / 动画 / 彩色徽标 / Dock 角标与弹跳 / 显示数字)按状态循环播放,
// 所见即「Dock 与菜单栏里实际会出现的样子」。时间走 kit.Now(),所以测试与截图里可复现。

const (
	brandLiveSlot  = 2800 * time.Millisecond // 自动循环时每个状态停留的时间
	brandLiveFrame = 180 * time.Millisecond  // 菜单栏动画换帧间隔(与真实托盘的定时器一致)
	brandLiveTick  = 33 * time.Millisecond   // 驱动重绘的节拍(约 30 fps)
)

// brandLiveClock 动态预览的当前时刻:状态序号、状态内已过的毫秒、24 帧循环里的帧号(与托盘 trayFrame 同源)。
type brandLiveClock struct {
	state int
	inMs  int64
	f24   int
}

func (a *App) brandLiveNow() brandLiveClock {
	if !kit.MotionOn() || kit.SystemReduceMotion() {
		return brandLiveClock{}
	}
	ms := kit.Now().UnixMilli()
	slot := brandLiveSlot.Milliseconds()
	return brandLiveClock{
		state: int(ms / slot % int64(len(trayStateList))),
		inMs:  ms % slot,
		f24:   int(ms / brandLiveFrame.Milliseconds() % 24),
	}
}

// brandTraySrc 取某样式、某状态、某动画帧的菜单栏图标(已按「运行中动画 / 彩色徽标」偏好处理);dark 为深色菜单栏。
// 与真实托盘共用 trayAssetFor + trayRender,所以预览和菜单栏里一模一样。
func (a *App) brandTraySrc(style int, mode string, f24 int, dark bool) ui.ImageSource {
	static := a.trayAnimStatic()
	animMode := a.trayAnimMode()
	name := trayAssetFor(style, mode, f24, mode == "running" && !static && animMode == 0)
	breath := -1.0
	if mode == "running" && !static && animMode == 1 {
		breath = trayBreathK(f24)
	}
	color := a.trayWantsColor()
	key := "live|" + name + "|" + strconv.FormatFloat(breath, 'f', 2, 64) + "|" + strconv.FormatBool(color) + "|" + strconv.FormatBool(dark)
	brandImgCache.Lock()
	if s, ok := brandImgCache.m[key]; ok {
		brandImgCache.Unlock()
		return s
	}
	brandImgCache.Unlock()
	var src ui.ImageSource
	if data, ok := embeddedIcon(name); ok {
		out, tmpl := trayRender(data, mode, breath, color, dark)
		if tmpl && dark { // 模板图在深色菜单栏上由系统反白;预览里自己染成白色
			out = recolorInk(out, whiteNRGBA)
		}
		if img, err := ui.DecodeImage(out); err == nil {
			src = img
		}
	}
	brandImgCache.Lock()
	brandImgCache.m[key] = src
	brandImgCache.Unlock()
	return src
}

// brandTrayLiveImg 画一枚菜单栏图标;素材缺失画占位。
func (a *App) brandTrayLiveImg(c *ui.Context, style int, mode string, f24 int, dark bool, px float32) {
	if src := a.brandTraySrc(style, mode, f24, dark); src != nil {
		ui.Image(c, src).Size(px, px).Fit(ui.Contain)
		return
	}
	ui.Box(c).Size(px, px).Radius(4).Background(a.tk.Fill2)
}

// brandMenuBarStrip 一条模拟菜单栏:右侧放托盘图标与(可选的)数字。
func (a *App) brandMenuBarStrip(c *ui.Context, style int, mode string, f24 int, dark bool) {
	bg, fg := ui.RGB(0xee, 0xee, 0xee), ui.RGB(0x22, 0x22, 0x24)
	if dark {
		bg, fg = ui.RGB(0x2a, 0x2a, 0x2c), ui.RGB(0xf2, 0xf2, 0xf4)
	}
	n := ""
	if a.prefs.Bool(pTrayCount, true) {
		switch mode {
		case "waiting":
			n = "2"
		case "running":
			n = "1"
		}
	}
	ui.Row(c).Height(28).Radius(6).Background(bg).PaddingX(10).Gap(5).Justify(ui.End).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(5, 5).Radius(3).Background(fg.Alpha(.22)) // 其它菜单栏图标的陪衬
		ui.Box(c).Size(5, 5).Radius(3).Background(fg.Alpha(.22))
		a.brandTrayLiveImg(c, style, mode, f24, dark, 20)
		if n != "" {
			ui.Text(c, n).FontSize(theme.FsCap).FontWeight(500).TextColor(fg).FontFeatures("tnum")
		}
	})
}

// brandBounceDy Dock 弹跳的纵向位移(向上为负):once = 状态一开始跳一下;until = 持续跳到处理为止。
func brandBounceDy(mode int, inMs int64) float32 {
	const hop = 700.0
	var u float64
	switch mode {
	case 1:
		u = float64(inMs) / hop
	case 2:
		u = float64(inMs%1100) / hop
	default:
		return 0
	}
	if u < 0 || u >= 1 {
		return 0
	}
	return float32(-18 * 4 * u * (1 - u)) // 抛物线,最高 18px
}

// brandDockCell 动态预览里的 Dock:同一枚应用图标,按状态带角标并在需要确认时弹跳(受「Dock 角标 / 弹跳」偏好控制)。
func (a *App) brandDockCell(c *ui.Context, mode string, clk brandLiveClock) {
	t := a.tk
	style := a.iconStyleIndex()
	name := a.appIconName(style)
	badge := ""
	switch a.prefs.Int(pDockBadge, 1) {
	case 1:
		if mode == "waiting" {
			badge = "2"
		}
	case 2:
		if mode == "running" || mode == "waiting" {
			badge = "1"
		}
	}
	dy := float32(0)
	if mode == "waiting" {
		dy = brandBounceDy(a.prefs.Int(pDockBnce, 1), clk.inMs)
	}
	ui.Row(c).Size(112, 66).Radius(14).Background(t.Fill2).Justify(ui.Center).AlignItems(ui.End).Padding(0, 0, 6, 0).Children(func() {
		ui.Box(c).Size(52, 52).Translate(0, dy).Children(func() {
			a.brandAppImg(c, name, 52)
			if badge != "" {
				ui.Row(c).Absolute().Top(-1).Right(-1).Height(18).MinWidth(18).PaddingX(5).Radius(9).Background(ui.RGB(0xff, 0x3b, 0x30)).
					Justify(ui.Center).AlignItems(ui.Center).Children(func() {
					ui.Text(c, badge).FontSize(theme.FsCap).FontWeight(700).TextColor(ui.RGB(0xff, 0xff, 0xff)).FontFeatures("tnum")
				})
			}
		})
	})
}

// brandLivePreview 动态预览面板:Dock + 浅 / 深菜单栏三处同步播放当前样式,下面一排状态胶囊可固定看某一种。
func (a *App) brandLivePreview(c *ui.Context) {
	t := a.tk
	clk := a.brandLiveNow()
	style := a.iconStyleIndex()
	idx := clk.state
	pin := a.set.brandPin
	for i, st := range trayStateList {
		if st.id == pin {
			idx = i
		}
	}
	st := trayStateList[idx]
	card := ui.Column(c).Gap(10).Padding(12).Radius(t.RMd()).Background(t.Fill1).Border(1, t.Bd1)
	if kit.MotionOn() && !kit.SystemReduceMotion() {
		_ = card.Loop("brand-live", brandLiveTick, ui.Linear)
	}
	card.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "动态预览").FontWeight(600)
			ui.Text(c, "Dock 与菜单栏用的是同一个样式:选一次,两处同时换").FontSize(theme.FsCap).TextColor(t.Tx3)
		})
		ui.Row(c).Gap(14).Wrap().Children(func() {
			ui.Column(c).Gap(4).AlignItems(ui.Center).Children(func() {
				a.brandDockCell(c, st.id, clk)
				ui.Text(c, "Dock").FontSize(theme.FsMicro).TextColor(t.Tx4)
			})
			ui.Column(c).Gap(4).Grow(1).MinWidth(170).Children(func() {
				a.brandMenuBarStrip(c, style, st.id, clk.f24, false)
				ui.Text(c, "浅色菜单栏").FontSize(theme.FsMicro).TextColor(t.Tx4)
			})
			ui.Column(c).Gap(4).Grow(1).MinWidth(170).Children(func() {
				a.brandMenuBarStrip(c, style, st.id, clk.f24, true)
				ui.Text(c, "深色菜单栏").FontSize(theme.FsMicro).TextColor(t.Tx4)
			})
		})
		ui.Row(c).Gap(4).Wrap().Children(func() {
			on := pin == ""
			if kit.Chip(c, "自动循环", kit.ChipOpt{On: on, Line: !on}).Key("blv-auto").Clicked() {
				a.set.brandPin = ""
			}
			for _, s := range trayStateList {
				s := s
				cur := pin == s.id
				ch := kit.Chip(c, s.name, kit.ChipOpt{On: cur, Line: !cur}).Key("blv-" + s.id)
				if ch.Clicked() {
					if cur {
						a.set.brandPin = ""
					} else {
						a.set.brandPin = s.id
					}
				}
			}
		})
		ui.Text(c, st.name+":"+st.desc).FontSize(theme.FsCap).TextColor(t.Tx3)
		if a.prefs != nil && !a.prefs.Bool(pTrayDyn, true) {
			ui.Text(c, "「随会话状态变化」已关闭:菜单栏里实际始终显示空闲图标,上面是各状态本来的样子。").FontSize(theme.FsCap).TextColor(t.Warn)
		}
		if kit.SystemReduceMotion() || macos.ReduceMotion() {
			ui.Text(c, "系统开启了「减弱动态效果」:预览与菜单栏都停在静态图。").FontSize(theme.FsCap).TextColor(t.Tx3)
		}
	})
}

var whiteNRGBA = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
