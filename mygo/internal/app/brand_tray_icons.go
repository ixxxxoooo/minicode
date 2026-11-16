package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sync"

	"github.com/egoist/mygo"
)

// 托盘菜单里会话行前的小标记(参考 Cursor 的会话菜单):
//   运行中 = 四点(模板图,跟随菜单文字色)· 待确认 = 橙点 · 未读 / 刚完成 = 蓝点 · 其余 = 同尺寸透明占位(文字左缘对齐)

var (
	trayMarkMu    sync.Mutex
	trayMarkCache = map[string]*mygo.MenuIcon{}
)

// trayMarkIcon 返回标记图标;kind:run / wait / unread / blank。生成失败返回 nil(该行不显示标记)。
func trayMarkIcon(kind string) *mygo.MenuIcon {
	trayMarkMu.Lock()
	defer trayMarkMu.Unlock()
	if ic, ok := trayMarkCache[kind]; ok {
		return ic
	}
	const n = 32
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	template := false
	switch kind {
	case "run":
		template = true
		for _, c := range [][2]float64{{16, 8.5}, {8.5, 16}, {23.5, 16}, {16, 23.5}} {
			drawDisc(img, c[0], c[1], 3.2, color.NRGBA{A: 255})
		}
	case "wait":
		drawDisc(img, 16, 16, 7, color.NRGBA{R: 0xF5, G: 0x9E, B: 0x0B, A: 255})
	case "unread":
		drawDisc(img, 16, 16, 7, color.NRGBA{R: 0x2F, G: 0x7F, B: 0xDB, A: 255})
	}
	var b bytes.Buffer
	var out *mygo.MenuIcon
	if err := png.Encode(&b, img); err != nil {
		brandLog.Warn("托盘标记图编码失败", "kind", kind, "err", err)
	} else if ic, err := mygo.NewMenuIcon(b.Bytes(), template); err != nil {
		brandLog.Warn("托盘标记图创建失败", "kind", kind, "err", err)
	} else {
		out = ic
	}
	trayMarkCache[kind] = out
	return out
}

// drawDisc 画一个抗锯齿实心圆(按到圆心的距离做 1px 过渡)。
func drawDisc(img *image.NRGBA, cx, cy, r float64, c color.NRGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			cov := math.Max(0, math.Min(1, r-d+0.5))
			if cov <= 0 {
				continue
			}
			a := uint8(float64(c.A) * cov)
			if old := img.NRGBAAt(x, y); old.A >= a {
				continue
			}
			img.SetNRGBA(x, y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: a})
		}
	}
}

// trayMarkText 图标不可用时的文字标记(彩色圆点字符在所有系统版本都能显示);
// 没有状态的行用等宽的空格占位,让标题左缘对齐。
func trayMarkText(kind string) string {
	switch kind {
	case "run":
		return "⁘\u2002"
	case "wait":
		return "🟠\u2002"
	case "unread":
		return "🔵\u2002"
	}
	return "\u2003\u2002"
}

// trayMarkAsText 托盘菜单一律用文字标记:库的菜单图标(MenuItem.Icon)在托盘菜单里实机不渲染(两次实测),
// 改环境变量 MINI_TRAY_ICON=1 可强制改用图标,方便以后复测。
func trayMarkAsText() bool { return os.Getenv("MINI_TRAY_ICON") != "1" }
