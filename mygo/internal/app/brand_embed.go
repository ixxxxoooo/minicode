package app

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	"image/png"
	"math"
)

// brandassets 由 assets/icon/build_all.sh 同步生成:托盘各状态 / 样式的 PNG 与 512px 应用图标。
// 内嵌之后,不管从访达、open 还是终端启动(工作目录各不相同),托盘与 Dock 图标都能找到素材。
//
//go:embed brandassets/*.png
var brandFS embed.FS

// embeddedIcon 读内嵌素材;name 形如 "tray-idle.png"、"app-style2-dark.png"。
func embeddedIcon(name string) ([]byte, bool) {
	b, err := brandFS.ReadFile("brandassets/" + name)
	if err != nil || len(b) == 0 {
		return nil, false
	}
	return b, true
}

// embeddedHas 内嵌素材是否存在。
func embeddedHas(name string) bool {
	_, ok := embeddedIcon(name)
	return ok
}

// 徽标状态的几何(36 画布,与 assets/icon/gen.py::tray_badge 一致):圆心与半径。
const (
	trayBadgeX = 28.2
	trayBadgeY = 8.0
)

// trayBadgeSpec 返回某托盘模式的彩色徽标半径与颜色;没有徽标的模式 ok 为 false。
// 颜色用 macOS 系统色(菜单栏是系统表面,不跟应用主题),与设计稿 brand.js::BADGE 一致。
func trayBadgeSpec(mode string) (r float64, c color.NRGBA, ok bool) {
	switch mode {
	case "waiting":
		return 6.4, color.NRGBA{0xff, 0x9f, 0x0a, 0xff}, true
	case "unread":
		return 4.6, color.NRGBA{0x30, 0xd1, 0x58, 0xff}, true
	case "error":
		return 6.4, color.NRGBA{0xff, 0x45, 0x3a, 0xff}, true
	case "update":
		return 6.4, color.NRGBA{0x0a, 0x84, 0xff, 0xff}, true
	}
	return 0, color.NRGBA{}, false // paused / demo 本来就是单色表达,idle / running 没有徽标
}

// tintTrayPNG 把单色模板 PNG 变成「彩色徽标版」:徽标圆盘染成状态色(里面的符号留白色),
// 其余主体按菜单栏深浅染成白或黑。结果必须按「非模板图」绘制,所以调用方要用 Tray.Dark() 判断深浅,
// 并在外观变化时重画。
func tintTrayPNG(data []byte, mode string, dark bool) ([]byte, bool) {
	r, badge, ok := trayBadgeSpec(mode)
	if !ok {
		return nil, false
	}
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	b := src.Bounds()
	scale := float64(b.Dx()) / 36 // 素材是 36px,@2x 时尺寸翻倍,几何随之放大
	fg := color.NRGBA{0, 0, 0, 0xff}
	if dark {
		fg = color.NRGBA{0xff, 0xff, 0xff, 0xff}
	}
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a16 := src.At(x, y).RGBA()
			a := float64(a16) / 65535
			d := math.Hypot((float64(x-b.Min.X)+0.5)/scale-trayBadgeX, (float64(y-b.Min.Y)+0.5)/scale-trayBadgeY)
			// 圆盘覆盖率:边缘半像素抗锯齿
			cov := clamp01(r + 0.5/scale - d)
			if cov > 0 {
				// 圆盘里:底色 = 状态色;原图在符号处是透明(a≈0),按透明度混入白色
				glyph := 1 - a
				if d > r-0.4 { // 圆盘最外沿不画符号
					glyph = 0
				}
				col := mixNRGBA(badge, color.NRGBA{0xff, 0xff, 0xff, 0xff}, glyph)
				col.A = uint8(math.Round(cov * 255))
				out.SetNRGBA(x, y, col)
				continue
			}
			c := fg
			c.A = uint8(math.Round(a * 255))
			out.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, false
	}
	return buf.Bytes(), true
}

// fadeTrayPNG 整体乘以透明度(「呼吸」动画的各帧),保持模板图性质。
func fadeTrayPNG(data []byte, k float64) []byte {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}
	b := src.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			c.A = uint8(math.Round(float64(c.A) * clamp01(k)))
			out.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return data
	}
	return buf.Bytes()
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

func mixNRGBA(a, b color.NRGBA, t float64) color.NRGBA {
	t = clamp01(t)
	m := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.NRGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 0xff}
}
