package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 设置页「应用图标 / 菜单栏图标」两组里的真实图标预览:用内嵌素材画,不再是带文字的彩色方块。

var brandImgCache = struct {
	sync.Mutex
	m map[string]ui.ImageSource
}{m: map[string]ui.ImageSource{}}

// brandImage 解码内嵌素材(带缓存);dark 仅对托盘模板图有意义:深色菜单栏上把黑色主体染成白色。
func brandImage(name string, whiteInk bool) ui.ImageSource {
	key := name
	if whiteInk {
		key += "|w"
	}
	brandImgCache.Lock()
	defer brandImgCache.Unlock()
	if s, ok := brandImgCache.m[key]; ok {
		return s
	}
	data, ok := embeddedIcon(name)
	if !ok {
		brandImgCache.m[key] = nil
		return nil
	}
	if whiteInk {
		data = recolorInk(data, color.NRGBA{0xff, 0xff, 0xff, 0xff})
	}
	src, err := ui.DecodeImage(data)
	if err != nil {
		src = nil
	}
	brandImgCache.m[key] = src
	return src
}

// recolorInk 把所有像素的颜色换成 c,透明度按原图乘以 c.A(模板图在深色底上显示成白色)。
func recolorInk(data []byte, c color.NRGBA) []byte {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}
	b := src.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			a := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA).A
			a = uint8(uint32(a) * uint32(c.A) / 255) // c.A 作整体不透明度(空状态的 25% 单色图形)
			out.SetNRGBA(x, y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: a})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return data
	}
	return buf.Bytes()
}

// appIconName 某样式在当前配色与外观偏好下的素材名;配色素材缺失时退回默认配色。
func (a *App) appIconName(style int) string {
	return a.appIconNameFor(style, a.appIconPal(), a.tk != nil && a.tk.Palette.Dark)
}

// appIconNameFor 指定配色与「自动外观下的深浅」时的素材名(配色色板的小预览用)。
func (a *App) appIconNameFor(style int, pal string, dark bool) string {
	suffix, _ := a.dockIconSuffix(dark)
	if pal != "" {
		if name := appIconAsset(style, pal, suffix); embeddedHas(name) {
			return name
		}
	}
	return appIconAsset(style, "", suffix)
}

// brandAppImg 画一枚应用图标;素材缺失时画占位方块(不会空白)。
func (a *App) brandAppImg(c *ui.Context, name string, px float32) {
	if src := brandImage(name, false); src != nil {
		ui.Image(c, src).Size(px, px).Fit(ui.Contain)
		return
	}
	ui.Box(c).Size(px, px).Radius(px / 4).Background(a.tk.Fill2)
}

// brandPreviewDock 应用图标组顶部的预览:同一枚图标在 Dock / ⌘Tab / 关于 里的三种尺寸(即时反映偏好)。
func (a *App) brandPreviewDock(c *ui.Context) {
	t := a.tk
	name := a.appIconName(a.iconStyleIndex())
	ui.Row(c).Gap(22).Padding(4, 0).AlignItems(ui.End).Children(func() {
		for _, it := range []struct {
			label string
			px    float32
		}{{"Dock", 56}, {"⌘Tab", 40}, {"关于", 28}} {
			ui.Column(c).Gap(5).AlignItems(ui.Center).Children(func() {
				a.brandAppImg(c, name, it.px)
				ui.Text(c, it.label).FontSize(theme.FsMicro).TextColor(t.Tx4)
			})
		}
	})
}

// brandStyleTile 一张可点的样式卡:上面是图标预览,下面是名称;选中时描边。
// preview 的参数是「鼠标正悬停在这张卡上」:悬停时卡里的菜单栏图标播放「运行中」动画,不用先选中就能看到各样式动起来的样子。
func (a *App) brandStyleTile(c *ui.Context, key string, on bool, w, h float32, name string, preview func(hov bool)) bool {
	t := a.tk
	b := ui.ButtonBase(c).Key(key).Size(w, h).Radius(t.RMd()).Cursor(ui.CursorPointer).Shrink(0)
	hov := b.Hovered()
	switch {
	case on:
		b.Background(t.AcSoft).Border(1.5, t.Ac)
	case hov:
		b.Background(t.Fill1).Border(1, t.Bd2)
	default:
		b.Border(1, t.Bd1)
	}
	if hov && kit.MotionOn() && !kit.SystemReduceMotion() {
		_ = b.Loop("tile-anim", brandLiveTick, ui.Linear)
	}
	b.Children(func() {
		ui.Column(c).Gap(5).AlignItems(ui.Center).Children(func() {
			preview(hov)
			ui.Text(c, name).FontSize(theme.FsCap).FontWeight(map[bool]int{true: 650, false: 500}[on]).
				TextColor(map[bool]ui.Color{true: t.AcTx, false: t.Tx2}[on]).SingleLine()
		})
	})
	return b.Clicked()
}

// brandAppStylePick 图标样式五选一:每张卡同时给出应用图标与它在浅 / 深菜单栏上的样子(同一个样式,一次换两处)。
func (a *App) brandAppStylePick(c *ui.Context) {
	cur := a.iconStyleIndex()
	ui.Row(c).Gap(6).Wrap().Children(func() {
		for i, name := range appStyles {
			i := i
			if a.brandStyleTile(c, "appsty-"+itoa(i), cur == i, 84, 104, name, func(hov bool) {
				a.brandAppImg(c, a.appIconName(i), 44)
				mode, f24 := "idle", 0
				if hov {
					mode, f24 = "running", a.brandLiveNow().f24
				}
				ui.Row(c).Gap(4).Children(func() {
					for _, dark := range []bool{false, true} {
						bg := ui.RGB(0xee, 0xee, 0xee)
						if dark {
							bg = ui.RGB(0x2a, 0x2a, 0x2c)
						}
						ui.Row(c).Size(34, 22).Radius(5).Background(bg).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
							a.brandTrayLiveImg(c, i, mode, f24, dark, 17)
						})
					}
				})
			}) {
				a.prefs.Set(pAppStyle, i)
				setLog.Info("选择图标样式(应用图标与菜单栏图标同步)", "style", name)
				a.syncBrand()
			}
		}
	})
}

// trayFileName 样式下标 + 状态 → 托盘素材名(样式 0 沿用 tray-<状态>.png)。
func trayFileName(style int, state string) string { return trayAssetFor(style, state, 0, false) }

// 设计稿 brand.js::TRAY_STATES:托盘各状态与说明。
var trayStateList = []struct{ id, name, desc string }{
	{"idle", "空闲", "没有运行中的回合,也没有待处理的事"},
	{"running", "运行中", "有会话在跑回合 / 后台任务;图标动画,不加徽标"},
	{"waiting", "待确认", "有权限请求或提问在等你:右上角「!」徽标,优先级最高"},
	{"unread", "已完成", "后台会话的回合刚结束、你还没看:实心小圆点"},
	{"error", "出错", "会话失败或 claude 进程崩溃且重启失败:「×」徽标"},
	{"update", "有更新", "已下载新版本、等你重启:向下箭头徽标"},
	{"paused", "已暂停", "「暂停通知」开启:整体半透明 + 暂停徽标"},
	{"demo", "演示模式", "演示后端:空心圆环徽标,避免把演示当真"},
}

// brandTrayStates 当前样式下全部状态的一览(浅 / 深菜单栏各一排),悬停看说明;
// 与真实菜单栏一样遵守「运行中动画」「彩色徽标」偏好,运行中那一格会动。
func (a *App) brandTrayStates(c *ui.Context) {
	t := a.tk
	style := a.iconStyleIndex()
	clk := a.brandLiveNow()
	for _, dark := range []bool{false, true} {
		bg := ui.RGB(0xee, 0xee, 0xee)
		fg := t.Tx3
		if dark {
			bg, fg = ui.RGB(0x2a, 0x2a, 0x2c), ui.RGB(0xb8, 0xb8, 0xbc)
		}
		row := ui.Row(c).Gap(2).Padding(6, 8).Radius(t.RSm()).Background(bg).Wrap()
		if kit.MotionOn() && !kit.SystemReduceMotion() {
			_ = row.Loop("tray-states-"+strconv.FormatBool(dark), brandLiveTick, ui.Linear)
		}
		row.Children(func() {
			for _, st := range trayStateList {
				ui.Column(c).Key("trst-" + st.id + map[bool]string{true: "d", false: "l"}[dark]).Gap(3).AlignItems(ui.Center).Width(54).Tooltip(st.name + ":" + st.desc).Children(func() {
					a.brandTrayLiveImg(c, style, st.id, clk.f24, dark, 20)
					ui.Text(c, st.name).FontSize(theme.FsMicro).TextColor(fg).SingleLine()
				})
			}
		})
	}
}

// brandInkImage 把托盘模板图染成 col(含不透明度)后缓存;空状态用它画「25% 的单色图形」。
func brandInkImage(name string, col ui.Color) ui.ImageSource {
	key := name + "|ink|" + itoa(int(col.R)) + "," + itoa(int(col.G)) + "," + itoa(int(col.B)) + "," + itoa(int(col.A))
	brandImgCache.Lock()
	defer brandImgCache.Unlock()
	if s, ok := brandImgCache.m[key]; ok {
		return s
	}
	var src ui.ImageSource
	if data, ok := embeddedIcon(name); ok {
		if img, err := ui.DecodeImage(recolorInk(data, color.NRGBA{R: col.R, G: col.G, B: col.B, A: col.A})); err == nil {
			src = img
		}
	}
	brandImgCache.m[key] = src
	return src
}

// brandEmptyMark 空状态的品牌小标:当前菜单栏图标样式的空闲图,单色 25%,不抢内容。
func (a *App) brandEmptyMark(c *ui.Context, px float32) {
	t := a.tk
	name := trayFileName(a.iconStyleIndex(), "idle")
	ink := ui.Color{R: t.Tx1.R, G: t.Tx1.G, B: t.Tx1.B, A: 64}
	if src := brandInkImage(name, ink); src != nil {
		ui.Image(c, src).Size(px, px).Fit(ui.Contain)
		return
	}
	kit.Ic(c, "mark", px*0.65).TextColor(t.Tx4)
}
