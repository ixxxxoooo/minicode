package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const (
	mmZoomMin = 0.1
	mmZoomMax = 8
)

// closeMmViewer 关闭图表全屏:有动效时先播淡出再销毁(重复调用不重置),没有动效立刻关。
func (a *App) closeMmViewer() {
	v := &a.mmView
	if !v.open {
		return
	}
	if !kit.MotionOn() {
		v.open = false
		return
	}
	if v.closeAt.IsZero() {
		v.closeAt = kit.Now()
		a.frameSoon()
	}
}

// mmProgress 返回入场 / 退场进度:p 是遮罩与内容的不透明度,s 是内容缩放系数(与图片灯箱一致)。
func (v *mmViewer) mmProgress() (p, s float32) {
	if !kit.MotionOn() {
		return 1, 1
	}
	if !v.closeAt.IsZero() {
		q := min(max(float32(kit.Since(v.closeAt))/float32(lbCloseDur), 0), 1)
		return 1 - kit.EaseOut(q), 1 - .03*kit.EaseOut(q)
	}
	e := kit.EaseOut(min(max(float32(kit.Since(v.at))/float32(lbOpenDur), 0), 1))
	return e, .94 + .06*e
}

// mmZoomBy 以画布区域中心为中心缩放 f 倍(按钮 / 键盘用;滚轮与捏合以光标为中心,见 viewerInput)。
func (a *App) mmZoomBy(f float32) {
	v := &a.mmView
	v.fit = false
	v.scale, v.panX, v.panY = zoomAt(v.scale, v.panX, v.panY, v.aw/2, v.ah/2, f, mmZoomMin, mmZoomMax)
}

// mermaidViewerOverlay 图表全屏查看:深色遮罩盖住整个窗口(与图片灯箱同一套),图画在一张底卡上;
// 拖动平移、滚轮以光标为中心缩放、触控板两指平移 / 捏合、底部缩放工具条、Esc / 点空白处关闭。
func (a *App) mermaidViewerOverlay(c *ui.Context) {
	v := &a.mmView
	if !v.open || v.src == "" {
		return
	}
	entry := a.mmEnsure(v.src)
	if !entry.ready {
		return
	}
	if entry.err != nil {
		v.open = false
		return
	}
	if !v.closeAt.IsZero() && kit.Since(v.closeAt) >= lbCloseDur {
		v.open = false
		return
	}
	alpha, _ := v.mmProgress()
	if alpha < 1 || !v.closeAt.IsZero() {
		a.frameSoon() // 动画期间持续出帧
	}
	sc := entry.scene
	t := a.tk
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(lbScrim).Grow(1).Opacity(alpha)
		closeFn := func() { a.closeMmViewer() }
		backdrop.OnClick(func() {
			if v.drag != nil && v.drag.moved {
				return
			}
			closeFn()
		})
		panel.Grow(1).AlignSelf(ui.Stretch).Column() // 铺满整个窗口
		panel.Children(func() {
			panel.OnShortcut(0, ui.KeyEscape, closeFn)
			panel.OnShortcut(0, ui.KeyEqual, func() { a.mmZoomBy(1.25) })
			panel.OnShortcut(ui.Shift, ui.KeyEqual, func() { a.mmZoomBy(1.25) })
			panel.OnShortcut(0, ui.KeyMinus, func() { a.mmZoomBy(1 / 1.25) })
			panel.OnShortcut(0, ui.Key0, func() { v.fit = true })
			for _, sk := range []struct {
				key    ui.Key
				dx, dy float32
			}{
				{ui.KeyLeft, -24, 0}, {ui.KeyRight, 24, 0}, {ui.KeyUp, 0, -24}, {ui.KeyDown, 0, 24},
			} {
				sk := sk
				panel.OnShortcut(0, sk.key, func() { v.fit, v.panX, v.panY = false, v.panX+sk.dx, v.panY+sk.dy })
				panel.OnShortcut(ui.Shift, sk.key, func() { v.fit, v.panX, v.panY = false, v.panX+sk.dx*2, v.panY+sk.dy*2 })
			}

			ui.Row(c).Height(48).Padding(0, 16).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Text(c, v.title).FontSize(theme.FsBody).TextColor(lbInkA(.85)).SingleLine()
				ui.Text(c, "拖拽平移 · 滚轮缩放 · Esc 关闭").FontSize(theme.FsCap).TextColor(lbInkA(.45)).Grow(1).SingleLine()
				a.lbTopBtn(c, "mm-x", "x", "关闭 (Esc)", closeFn)
			})

			area := ui.Box(c).Grow(1).MinHeight(0).Clip()
			// 滚轮以光标为中心缩放、触控板两指平移 / 捏合缩放
			area.HandleInput(viewerInput(&v.fit, &v.scale, &v.panX, &v.panY, mmZoomMin, mmZoomMax))
			if v.drag != nil && v.drag.moved {
				area.Cursor(ui.CursorGrabbing)
			} else {
				area.Cursor(ui.CursorGrab)
			}
			if b := area.Bounds(); b.W > 0 && b.H > 0 {
				v.aw, v.ah = b.W, b.H
				if v.fit && sc.Width > 0 && sc.Height > 0 {
					v.scale = clamp(min((b.W-48)/sc.Width, (b.H-48)/sc.Height), mmZoomMin, mmZoomMax)
					v.panX = (b.W - sc.Width*v.scale) / 2
					v.panY = (b.H - sc.Height*v.scale) / 2
				}
			}
			if area.Pressed() {
				if dx, dy, ok := area.Dragged(); ok {
					// Dragged 返回的是「自上一帧起」指针挪动的量:要累加,越过阈值后才算拖动
					if v.drag == nil {
						v.drag = &lbDrag{}
					}
					d := v.drag
					d.sx += dx
					d.sy += dy
					if !d.moved && (d.sx*d.sx+d.sy*d.sy) >= lbDragSlop*lbDragSlop {
						d.moved = true
						dx, dy = d.sx, d.sy
					}
					if d.moved {
						v.fit = false
						v.panX += dx
						v.panY += dy
					}
				}
			} else if v.drag != nil {
				v.drag = nil
			}
			sw, sh := sc.Width*v.scale, sc.Height*v.scale
			// 点图外的空白关闭,点图本身(含拖动结束)不关
			area.OnClick(func() {
				if v.drag != nil && v.drag.moved {
					return
				}
				if x, y, ok := area.PointerPosition(); ok && x >= v.panX && x <= v.panX+sw && y >= v.panY && y <= v.panY+sh {
					return
				}
				closeFn()
			})
			const cardPad = float32(16)
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				// 底卡:图的配色是按应用背景设计的,放在深色遮罩上要垫一张底
				card := ui.Rect{X: r.X + v.panX - cardPad, Y: r.Y + v.panY - cardPad, W: sw + cardPad*2, H: sh + cardPad*2}
				p.Fill(card, t.BG.Alpha(alpha), t.RLg())
				a.paintMmScene(p, &sc, v.scale, r.X+v.panX, r.Y+v.panY)
			})
			area.Children(func() {
				// 文字层与画布同步平移
				ui.Box(c).Absolute().Top(v.panY).Left(v.panX).Opacity(alpha).Children(func() { a.mmTextLayer(c, &sc, v.scale) })
			})

			ui.Row(c).Padding(0, 0, 24, 0).Justify(ui.Center).Children(func() {
				bar := ui.Row(c).Gap(4).Padding(8, 12).Radius(t.RLg()).
					Background(ui.Color{R: 0, G: 0, B: 0, A: 160}).Border(1, lbInkA(.12)).AlignItems(ui.Center)
				bar.OnClick(func() {})
				bar.Children(func() {
					a.lbBarBtn(c, "mm-out", "minus", "缩小 (-)", func() { a.mmZoomBy(1 / 1.25) })
					ui.Text(c, fmtMmPct(v.scale)).Width(52).TextAlign(ui.Center).FontSize(theme.FsCap).TextColor(lbInkA(.85))
					a.lbBarBtn(c, "mm-in", "plus", "放大 (+)", func() { a.mmZoomBy(1.25) })
					a.lbBarBtn(c, "mm-fit", "max", "适应窗口 (0)", func() { v.fit = true })
					ui.Box(c).Width(1).Height(20).Background(lbInkA(.15)).Margin(0, 4)
					a.lbBarBtn(c, "mm-copy", "copy", "复制源码", func() { a.copyOut(v.src, "Mermaid 源码") })
				})
			})
		})
	})
	if !open {
		a.closeMmViewer()
	}
}
