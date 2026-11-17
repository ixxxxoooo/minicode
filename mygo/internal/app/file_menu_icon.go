package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// fileIconID 与文件树 fileIc 一致的判断键(单测与菜单对齐用):徽标类为 badge:标签,否则为线性图标名。
func fileIconID(name string) string {
	if k, ok := fileKindOf(name); ok {
		return "badge:" + k.label
	}
	return fileIcon(name)
}

var (
	fileMenuMu    sync.Mutex
	fileMenuCache = map[string]ui.MenuIcon{}
)

// fileMenuIcon 面包屑 / 右键等菜单项用的文件图标(与左侧文件树同一套规则)。
func (a *App) fileMenuIcon(name string) ui.MenuIcon {
	key := fileIconID(name)
	fileMenuMu.Lock()
	defer fileMenuMu.Unlock()
	if ic, ok := fileMenuCache[key]; ok {
		return ic
	}
	var ic ui.MenuIcon
	if strings.HasPrefix(key, "badge:") {
		if k, ok := fileKindOf(name); ok {
			ic = menuFileBadgeIcon(k.label, a.fileKindColor(k.tone))
		}
	} else {
		ic = kit.MenuIcon(key)
	}
	fileMenuCache[key] = ic
	return ic
}

// mItemFile 带与文件树一致图标的菜单项。
func (a *App) mItemFile(m *ui.Menu, name string) *ui.MenuItem {
	return m.Item(name).Icon(a.fileMenuIcon(name))
}

func menuFileBadgeIcon(label string, col ui.Color) ui.MenuIcon {
	r, g, b, a := colorRGBA(col)
	fileMenuMu.Lock()
	cacheKey := "badgepx:" + label + ":" + itoa(int(r)) + "," + itoa(int(g)) + "," + itoa(int(b)) + "," + itoa(int(a))
	if ic, ok := fileMenuCache[cacheKey]; ok {
		fileMenuMu.Unlock()
		return ic
	}
	fileMenuMu.Unlock()
	const n = 32
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	fg := color.NRGBA{R: r, G: g, B: b, A: a}
	drawBadgeText(img, label, fg)
	var buf bytes.Buffer
	var out ui.MenuIcon
	if err := png.Encode(&buf, img); err == nil {
		if ic, err := mygo.NewMenuIcon(buf.Bytes(), false); err == nil && ic != nil {
			out = ic
		}
	}
	fileMenuMu.Lock()
	fileMenuCache[cacheKey] = out
	fileMenuMu.Unlock()
	return out
}

func colorRGBA(c ui.Color) (r, g, b, a uint8) {
	r, g, b, a = c.R, c.G, c.B, c.A
	if a == 0 {
		a = 255
	}
	return r, g, b, a
}

func drawBadgeText(img *image.NRGBA, label string, fg color.NRGBA) {
	// 简易 5×7 位图字体,只画徽标短标签(GO/JS/{}/…)
	scale := 2
	if len([]rune(label)) >= 3 {
		scale = 1
	}
	w := len(label) * (6*scale + scale)
	x0 := (img.Bounds().Dx() - w) / 2
	y0 := (img.Bounds().Dy() - 7*scale) / 2
	for i, ch := range label {
		glyph := badgeGlyph(ch)
		x := x0 + i*(6*scale+scale)
		for row := 0; row < 7; row++ {
			bits := glyph[row]
			for cx := 0; cx < 5; cx++ {
				if bits&(1<<(4-cx)) != 0 {
					for dy := 0; dy < scale; dy++ {
						for dx := 0; dx < scale; dx++ {
							img.Set(x+cx*scale+dx, y0+row*scale+dy, fg)
						}
					}
				}
			}
		}
	}
}

func badgeGlyph(ch rune) [7]byte {
	switch ch {
	case 'G':
		return [7]byte{0x0E, 0x11, 0x10, 0x17, 0x11, 0x11, 0x0E}
	case 'O':
		return [7]byte{0x0E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E}
	case 'J':
		return [7]byte{0x04, 0x04, 0x04, 0x04, 0x04, 0x14, 0x08}
	case 'S':
		return [7]byte{0x0E, 0x11, 0x02, 0x0C, 0x10, 0x11, 0x0E}
	case 'T':
		return [7]byte{0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04}
	case 'Y':
		return [7]byte{0x11, 0x11, 0x0A, 0x04, 0x04, 0x04, 0x04}
	case 'M':
		return [7]byte{0x11, 0x1B, 0x15, 0x11, 0x11, 0x11, 0x11}
	case 'D':
		return [7]byte{0x1E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x1E}
	case 'R':
		return [7]byte{0x1E, 0x11, 0x11, 0x1E, 0x14, 0x12, 0x11}
	case 'P':
		return [7]byte{0x1E, 0x11, 0x11, 0x1E, 0x10, 0x10, 0x10}
	case 'H':
		return [7]byte{0x11, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11}
	case 'C':
		return [7]byte{0x0E, 0x11, 0x10, 0x10, 0x10, 0x11, 0x0E}
	case '+':
		return [7]byte{0x00, 0x04, 0x04, 0x1F, 0x04, 0x04, 0x00}
	case '#':
		return [7]byte{0x0A, 0x0A, 0x1F, 0x0A, 0x1F, 0x0A, 0x0A}
	case '{', '}':
		return [7]byte{0x06, 0x04, 0x04, 0x0C, 0x04, 0x04, 0x06}
	case 'V':
		return [7]byte{0x11, 0x11, 0x11, 0x11, 0x0A, 0x0A, 0x04}
	case 'U':
		return [7]byte{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E}
	case 'L':
		return [7]byte{0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1F}
	case 'X':
		return [7]byte{0x11, 0x11, 0x0A, 0x04, 0x0A, 0x11, 0x11}
	case 'Q':
		return [7]byte{0x0E, 0x11, 0x11, 0x11, 0x15, 0x12, 0x0D}
	case 'I':
		return [7]byte{0x0E, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E}
	case 'F':
		return [7]byte{0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x10}
	case 'W':
		return [7]byte{0x11, 0x11, 0x11, 0x15, 0x15, 0x1B, 0x11}
	case 'E':
		return [7]byte{0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x1F}
	case 'A':
		return [7]byte{0x0E, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11}
	case 'B':
		return [7]byte{0x1E, 0x11, 0x11, 0x1E, 0x11, 0x11, 0x1E}
	case 'N':
		return [7]byte{0x11, 0x19, 0x15, 0x13, 0x11, 0x11, 0x11}
	default:
		return [7]byte{0x0E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E}
	}
}
