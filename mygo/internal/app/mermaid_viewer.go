package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

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
	sc := entry.scene
	t := a.tk
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(t.Scrim).Grow(1)
		closeFn := func() { v.open = false }
		backdrop.OnClick(closeFn)
		panel.Grow(1).Background(t.BG).Column()
		panel.Children(func() {
			ui.Row(c).Height(44).Padding(0, 16).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, v.title).FontWeight(600).Grow(1).SingleLine()
				kit.Btn(c, "适应", kit.BtnOpt{Small: true}).OnClick(func() { v.fit = true })
				kit.Btn(c, "100%", kit.BtnOpt{Small: true}).OnClick(func() { v.fit = false; v.scale = 1 })
				kit.Btn(c, "＋", kit.BtnOpt{Small: true}).OnClick(func() { v.fit = false; v.scale = clamp(v.scale*1.15, 0.25, 4) })
				kit.Btn(c, "－", kit.BtnOpt{Small: true}).OnClick(func() { v.fit = false; v.scale = clamp(v.scale/1.15, 0.25, 4) })
				kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭 Esc"}).OnClick(closeFn)
			})
			panel.OnShortcut(0, ui.KeyEscape, closeFn)
			for _, sk := range []struct {
				key ui.Key
				dx  float32
				dy  float32
			}{
				{ui.KeyLeft, -24, 0}, {ui.KeyRight, 24, 0}, {ui.KeyUp, 0, -24}, {ui.KeyDown, 0, 24},
			} {
				sk := sk
				panel.OnShortcut(ui.Shift, sk.key, func() {
					v.fit = false
					v.panX += sk.dx * 2
					v.panY += sk.dy * 2
				})
				panel.OnShortcut(0, sk.key, func() {
					v.fit = false
					v.panX += sk.dx
					v.panY += sk.dy
				})
			}
			cv := ui.Box(c).Grow(1).Cursor(ui.CursorGrab)
			// 滚轮以光标为中心缩放、触控板两指平移 / 捏合缩放(与 100% / ± 按钮同一套缩放范围的放宽版)
			cv.HandleInput(viewerInput(&v.fit, &v.scale, &v.panX, &v.panY, 0.2, 6))
			cv.Draw(func(p *ui.Painter, r ui.Rect) {
				if cv.Pressed() {
					if dx, dy, ok := cv.Dragged(); ok {
						v.fit = false
						v.panX += dx
						v.panY += dy
					}
				}
				w, h := r.W, r.H
				if w > 0 && h > 0 && v.fit && sc.Width > 0 && sc.Height > 0 {
					v.scale = min(w/sc.Width, h/sc.Height) * 0.9
					v.panX = (w - sc.Width*v.scale) / 2
					v.panY = (h - sc.Height*v.scale) / 2
				}
				a.paintMmScene(p, &sc, v.scale, r.X+v.panX, r.Y+v.panY)
			})
			cv.Children(func() {
				// 文字层与画布同步平移
				row := ui.Box(c).Absolute().Top(v.panY).Left(v.panX)
				row.Children(func() { a.mmTextLayer(c, &sc, v.scale) })
			})
			ui.Row(c).Height(theme.Px(22)).Padding(0, 12).Gap(8).AlignItems(ui.Center).FontSize(theme.FsMicro).TextColor(t.Tx4).Children(func() {
				ui.Text(c, fmt.Sprintf("缩放 %s · 拖拽平移 · 滚轮缩放", fmtMmPct(v.scale)))
				ui.Spacer(c).Grow(1)
				ui.Text(c, "⇧方向键大步").SingleLine()
			})
		})
	})
	if !open {
		v.open = false
	}
}
