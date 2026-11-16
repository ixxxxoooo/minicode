package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
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

// brandTrayImg 在菜单栏小底板上画一枚托盘图标:dark 为深色菜单栏(主体染白)。
func (a *App) brandTrayImg(c *ui.Context, name string, px float32, dark bool) {
	if src := brandImage(name, dark); src != nil {
		ui.Image(c, src).Size(px, px).Fit(ui.Contain)
		return
	}
	ui.Box(c).Size(px, px).Radius(4).Background(a.tk.Fill2)
}

// brandPreviewDock 应用图标组顶部的预览:同一枚图标在 Dock / ⌘Tab / 关于 里的三种尺寸(即时反映偏好)。
func (a *App) brandPreviewDock(c *ui.Context) {
	t := a.tk
	name := a.appIconName(a.prefs.Int(pAppStyle, 0) % len(appStyles))
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
func (a *App) brandStyleTile(c *ui.Context, key string, on bool, w, h float32, name string, preview func()) bool {
	t := a.tk
	b := ui.ButtonBase(c).Key(key).Size(w, h).Radius(t.RMd()).Cursor(ui.CursorPointer).Shrink(0)
	switch {
	case on:
		b.Background(t.AcSoft).Border(1.5, t.Ac)
	case b.Hovered():
		b.Background(t.Fill1).Border(1, t.Bd2)
	default:
		b.Border(1, t.Bd1)
	}
	b.Children(func() {
		ui.Column(c).Gap(5).AlignItems(ui.Center).Children(func() {
			preview()
			ui.Text(c, name).FontSize(theme.FsCap).FontWeight(map[bool]int{true: 650, false: 500}[on]).
				TextColor(map[bool]ui.Color{true: t.AcTx, false: t.Tx2}[on]).SingleLine()
		})
	})
	return b.Clicked()
}

func (a *App) brandAppStylePick(c *ui.Context) {
	cur := a.prefs.Int(pAppStyle, 0)
	ui.Row(c).Gap(6).Wrap().Children(func() {
		for i, name := range appStyles {
			i := i
			if a.brandStyleTile(c, "appsty-"+itoa(i), cur == i, 72, 78, name, func() { a.brandAppImg(c, a.appIconName(i), 44) }) {
				a.prefs.Set(pAppStyle, i)
				setLog.Info("选择应用图标样式", "style", name)
				a.syncBrand()
			}
		}
	})
}

func (a *App) brandTrayStylePick(c *ui.Context) {
	cur := a.trayStyleIndex()
	ui.Row(c).Gap(6).Wrap().Children(func() {
		for i, name := range trayStyles {
			i := i
			file := trayFileName(i, "idle")
			if a.brandStyleTile(c, "trsty-"+itoa(i), cur == i, 72, 66, name, func() {
				// 浅 / 深两种菜单栏底各放一枚,与设计稿的预览一致
				ui.Row(c).Gap(4).Children(func() {
					for _, dark := range []bool{false, true} {
						bg := ui.RGB(0xee, 0xee, 0xee)
						if dark {
							bg = ui.RGB(0x2a, 0x2a, 0x2c)
						}
						ui.Row(c).Size(28, 24).Radius(5).Background(bg).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
							a.brandTrayImg(c, file, 18, dark)
						})
					}
				})
			}) {
				a.prefs.Set(pTrayStyle, i)
				setLog.Info("选择菜单栏图标样式", "style", name)
				a.syncBrand()
			}
		}
	})
}

// trayFileName 样式下标 + 状态 → 托盘素材名(样式 0 沿用 tray-<状态>.png)。
func trayFileName(style int, state string) string {
	if style <= 0 {
		return "tray-" + state + ".png"
	}
	return "tray-s" + itoa(style) + "-" + state + ".png"
}

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

// brandTrayStates 当前样式下全部状态的一览(浅 / 深菜单栏各一排),悬停看说明。
func (a *App) brandTrayStates(c *ui.Context) {
	t := a.tk
	style := a.trayStyleIndex()
	for _, dark := range []bool{false, true} {
		bg := ui.RGB(0xee, 0xee, 0xee)
		fg := t.Tx3
		if dark {
			bg, fg = ui.RGB(0x2a, 0x2a, 0x2c), ui.RGB(0xb8, 0xb8, 0xbc)
		}
		ui.Row(c).Gap(2).Padding(6, 8).Radius(t.RSm()).Background(bg).Wrap().Children(func() {
			for _, st := range trayStateList {
				ui.Column(c).Key("trst-" + st.id + map[bool]string{true: "d", false: "l"}[dark]).Gap(3).AlignItems(ui.Center).Width(54).Tooltip(st.name + ":" + st.desc).Children(func() {
					a.brandTrayImg(c, trayFileName(style, st.id), 20, dark)
					ui.Text(c, st.name).FontSize(theme.FsMicro).TextColor(fg).SingleLine()
				})
			}
		})
	}
	_ = kit.S12
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
	name := "tray-idle.png"
	if i := a.trayStyleIndex(); i > 0 {
		name = "tray-s" + itoa(i) + "-idle.png"
	}
	ink := ui.Color{R: t.Tx1.R, G: t.Tx1.G, B: t.Tx1.B, A: 64}
	if src := brandInkImage(name, ink); src != nil {
		ui.Image(c, src).Size(px, px).Fit(ui.Contain)
		return
	}
	kit.Ic(c, "mark", px*0.65).TextColor(t.Tx4)
}
