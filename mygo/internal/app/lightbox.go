package app

import (
	"math"
	"os"
	"os/exec"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

var lbLog = diag.L("app.lightbox")

var lbInk = ui.Color{R: 255, G: 255, B: 255, A: 255}

func lbInkA(a float32) ui.Color { return lbInk.Alpha(a) }

// lightboxState 当前打开的灯箱(nil 表示关闭)。
type lightboxState struct {
	paths      []string // 绝对路径
	names      []string // 显示名
	idx        int
	zoom       float32
	offX, offY float32
	natW, natH float32
	loaded     bool
	miss       bool // 当前图读不到
	drag       *lbDrag
}

type lbDrag struct {
	sx, sy, ox, oy float32
	moved          bool
}

func (a *App) openLightbox(paths []string, names []string, idx int) {
	paths = append([]string(nil), paths...)
	names = append([]string(nil), names...)
	if len(paths) == 0 {
		return
	}
	if idx < 0 || idx >= len(paths) {
		idx = 0
	}
	for i := range names {
		if names[i] == "" {
			names[i] = baseName(paths[i])
		}
	}
	lb := &lightboxState{paths: paths, names: names, idx: idx, zoom: 1}
	a.lb = lb
	a.lbRefreshNatural()
	a.frameSoon()
	lbLog.Info("打开灯箱", "path", paths[idx], "n", len(paths), "idx", idx)
}

func (a *App) closeLightbox() {
	if a.lb != nil {
		lbLog.Info("关闭灯箱")
	}
	a.lb = nil
}

func (a *App) lbRefreshNatural() {
	if a.lb == nil || a.lb.idx < 0 || a.lb.idx >= len(a.lb.paths) {
		return
	}
	p := a.lb.paths[a.lb.idx]
	a.lb.miss = false
	a.lb.loaded = false
	if _, err := os.Stat(p); err != nil {
		a.lb.miss = true
		a.lb.natW, a.lb.natH = 1200, 900
		a.lb.loaded = true
		return
	}
	if src := a.localImage(p); src != nil {
		iw, ih := ui.ImageSize(src)
		if iw > 0 && ih > 0 {
			a.lb.natW, a.lb.natH = iw, ih
			a.lb.loaded = true
			return
		}
	}
	a.lb.natW, a.lb.natH = 1200, 900
	a.lb.loaded = true
}

func (a *App) lightboxOverlay(c *ui.Context) {
	lb := a.lb
	if lb == nil {
		return
	}
	t := a.tk
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(t.Scrim).Grow(1)
		closeFn := func() { a.closeLightbox() }
		backdrop.OnClick(func() {
			if lb.drag != nil && lb.drag.moved {
				return
			}
			closeFn()
		})
		panel.Grow(1).Column()
		panel.Children(func() {
			panel.OnShortcut(0, ui.KeyEscape, closeFn)
			panel.OnShortcut(0, ui.KeyEqual, func() { a.lbZoomBy(1.25) })
			panel.OnShortcut(ui.Shift, ui.KeyEqual, func() { a.lbZoomBy(1.25) })
			panel.OnShortcut(0, ui.KeyMinus, func() { a.lbZoomBy(1 / 1.25) })
			panel.OnShortcut(0, ui.Key0, func() { lb.zoom, lb.offX, lb.offY = 1, 0, 0 })
			if len(lb.paths) > 1 {
				panel.OnShortcut(0, ui.KeyLeft, func() { a.lbStep(-1) })
				panel.OnShortcut(0, ui.KeyRight, func() { a.lbStep(1) })
			}
			vw, vh := panel.Bounds().W, panel.Bounds().H
			if vw <= 0 || vh <= 0 {
				vw, vh = a.winW, a.winH
			}
			name := lb.names[lb.idx]
			if len(lb.paths) > 1 {
				name = name + " · " + itoa(lb.idx+1) + " / " + itoa(len(lb.paths))
			}
			ui.Row(c).Height(48).Padding(0, 16).AlignItems(ui.Center).Children(func() {
				ui.Text(c, name).FontSize(theme.FsBody).TextColor(lbInkA(.85)).Grow(1).SingleLine()
				a.lbTopBtn(c, "lb-x", "x", "关闭 (Esc)", closeFn)
			})
			area := ui.Box(c).Grow(1).MinHeight(0)
			area.HandleInput(lbPanZoomInput(lb))
			dw, dh := lightboxDisplaySize(lb.natW, lb.natH, vw, vh, lb.zoom)
			if area.Pressed() {
				if dx, dy, ok := area.Dragged(); ok {
					if lb.drag == nil {
						lb.drag = &lbDrag{ox: lb.offX, oy: lb.offY}
					}
					d := lb.drag
					if !d.moved && (dx*dx+dy*dy) >= lbDragSlop*lbDragSlop {
						d.moved = true
					}
					if d.moved {
						lb.offX = d.ox + dx
						lb.offY = d.oy + dy
					}
				}
			} else if lb.drag != nil {
				lb.drag = nil
			}
			lx := (vw-dw)/2 + lb.offX
			ly := (vh-dh)/2 + lb.offY
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				_ = p
				_ = r
			})
			area.Children(func() {
				if lb.miss {
					ui.Column(c).Absolute().Left(lx).Top(ly).Width(max(dw, 280)).Padding(24).Gap(12).
						Background(ui.Color{R: 0, G: 0, B: 0, A: 140}).Radius(t.RLg()).AlignItems(ui.Center).Children(func() {
						kit.Ic(c, "image", kit.S20).TextColor(lbInkA(.6))
						ui.Text(c, baseName(lb.paths[lb.idx])).FontSize(theme.FsBody).TextColor(lbInkA(.85))
						ui.Text(c, "找不到图片文件").FontSize(theme.FsCap).TextColor(lbInkA(.55))
						kit.Btn(c, "在访达中显示", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
							_ = exec.Command("open", "-R", lb.paths[lb.idx]).Start()
						})
					})
					return
				}
				src := a.localImage(lb.paths[lb.idx])
				imgBox := ui.Box(c).Absolute().Left(lx).Top(ly).Size(dw, dh)
				if lb.drag != nil && lb.drag.moved {
					imgBox.Cursor(ui.CursorGrabbing)
				} else {
					imgBox.Cursor(ui.CursorGrab)
				}
				imgBox.OnClick(func() { /* 阻止冒泡到 backdrop */ })
				imgBox.Children(func() {
					if src != nil {
						ui.Image(c, src).Size(dw, dh).Fit(ui.Contain)
					} else {
						ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(8).Children(func() {
							kit.Spin(c, lbInkA(.7))
							ui.Text(c, "载入中…").FontSize(theme.FsCap).TextColor(lbInkA(.6))
						})
					}
				})
			})
			pct := lightboxZoomPct(lb.natW, lb.natH, vw, vh, lb.zoom)
			ui.Row(c).Padding(0, 0, 24, 0).Justify(ui.Center).Children(func() {
				bar := ui.Row(c).Gap(4).Padding(8, 12).Radius(t.RLg()).
					Background(ui.Color{R: 0, G: 0, B: 0, A: 160}).Border(1, lbInkA(.12)).AlignItems(ui.Center)
				bar.OnClick(func() {})
				bar.Children(func() {
					a.lbBarBtn(c, "lb-out", "minus", "缩小 (-)", func() { a.lbZoomBy(1 / 1.25) })
					ui.Text(c, itoa(pct)+"%").Width(52).TextAlign(ui.Center).FontSize(theme.FsCap).TextColor(lbInkA(.85))
					a.lbBarBtn(c, "lb-in", "plus", "放大 (+)", func() { a.lbZoomBy(1.25) })
					a.lbBarBtn(c, "lb-fit", "maximize", "适应窗口 (0)", func() { lb.zoom, lb.offX, lb.offY = 1, 0, 0 })
					ui.Box(c).Width(1).Height(20).Background(lbInkA(.15)).Margin(0, 4)
					a.lbBarBtn(c, "lb-open", "ext", "在预览中打开", func() {
						_ = exec.Command("open", lb.paths[lb.idx]).Start()
					})
					a.lbBarBtn(c, "lb-cp", "copy", "复制路径", func() {
						a.copyOut(lb.paths[lb.idx], "路径")
					})
				})
			})
		})
	})
	if !open {
		a.closeLightbox()
	}
}

func lbPanZoomInput(lb *lightboxState) func(ev ui.InputEvent) bool {
	return func(ev ui.InputEvent) bool {
		switch ev.Kind {
		case ui.InputZoom:
			if ev.Scale <= 0 {
				return false
			}
			lb.zoom = clamp(lb.zoom*ev.Scale, lbZoomMin, lbZoomMax)
			if math.Abs(float64(lb.zoom-1)) < 0.01 {
				lb.zoom = 1
			}
			return true
		case ui.InputScroll:
			if ev.DY == 0 {
				return false
			}
			cx, cy := ev.X, ev.Y
			f := lightboxWheelFactor(ev.DY)
			old := lb.zoom
			lb.zoom = clamp(lb.zoom*f, lbZoomMin, lbZoomMax)
			if lb.zoom != old {
				r := lb.zoom / old
				lb.offX = cx - (cx-lb.offX)*r
				lb.offY = cy - (cy-lb.offY)*r
			}
			return true
		}
		return false
	}
}

func (a *App) lbZoomBy(f float32) {
	if a.lb == nil {
		return
	}
	a.lb.zoom = clamp(a.lb.zoom*f, lbZoomMin, lbZoomMax)
	if math.Abs(float64(a.lb.zoom-1)) < 0.01 {
		a.lb.zoom = 1
	}
}

func (a *App) lbStep(d int) {
	if a.lb == nil || len(a.lb.paths) < 2 {
		return
	}
	a.lb.idx = (a.lb.idx + d + len(a.lb.paths)) % len(a.lb.paths)
	a.lb.zoom, a.lb.offX, a.lb.offY = 1, 0, 0
	a.lbRefreshNatural()
}

func (a *App) lbTopBtn(c *ui.Context, key, icon, tip string, fn func()) {
	b := ui.ButtonBase(c).Key(key).Size(32, 32).Radius(a.tk.RSm()).Justify(ui.Center).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	if b.Hovered() {
		b.Background(lbInkA(.16))
	}
	if b.Pressed() {
		b.Background(lbInkA(.28))
	}
	b.Tooltip(tip).OnClick(fn).Children(func() {
		kit.Ic(c, icon, kit.S16).TextColor(lbInk)
	})
}

func (a *App) lbBarBtn(c *ui.Context, key, icon, tip string, fn func()) {
	b := ui.ButtonBase(c).Key(key).Size(32, 32).Radius(a.tk.RSm()).Justify(ui.Center).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	if b.Hovered() {
		b.Background(lbInkA(.16))
	}
	if b.Pressed() {
		b.Background(lbInkA(.28))
	}
	b.Tooltip(tip).OnClick(fn).Children(func() {
		kit.Ic(c, icon, kit.S16).TextColor(lbInk)
	})
}

// openUserImageLightbox 打开某条用户消息里的第 i 张图(同消息多图可左右切换)。
func (a *App) openUserImageLightbox(u *core.User, i int) {
	var paths, names []string
	for _, raw := range u.Images {
		abs := a.resolveUserImage(raw)
		if abs == "" {
			abs = raw
		}
		paths = append(paths, abs)
		names = append(names, baseName(raw))
	}
	if len(paths) == 0 {
		return
	}
	if i < 0 || i >= len(paths) {
		i = 0
	}
	a.openLightbox(paths, names, i)
}

func (a *App) openImageLightboxPath(abs string) {
	if strings.TrimSpace(abs) == "" {
		return
	}
	a.openLightbox([]string{abs}, []string{baseName(abs)}, 0)
}
