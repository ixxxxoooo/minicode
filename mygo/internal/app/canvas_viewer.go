package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/canvas"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

type cvViewer struct {
	open      bool
	block     string
	scale     float32
	panX      float32
	panY      float32
	fit       bool
	returnBlk string
	lastX     float32
	lastY     float32
}

func (a *App) openCvViewer(blockID string) {
	a.cvView = cvViewer{open: true, block: blockID, scale: 1, fit: true, returnBlk: a.cv.selBlk}
}

func (a *App) canvasViewerOverlay(c *ui.Context) {
	v := &a.cvView
	if !v.open {
		return
	}
	s := a.cvStore()
	var dg *canvas.Diagram
	for i := range s.Doc.Blocks {
		if s.Doc.Blocks[i].ID == v.block && s.Doc.Blocks[i].Diagram != nil {
			dg = s.Doc.Blocks[i].Diagram
			break
		}
	}
	if dg == nil {
		v.open = false
		return
	}
	t := a.tk
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(t.Scrim).Grow(1)
		closeFn := func() {
			v.open = false
			if v.returnBlk != "" {
				a.cv.selBlk = v.returnBlk
			}
		}
		backdrop.OnClick(closeFn)
		panel.Grow(1).Background(t.BG).Column()
		panel.Children(func() {
			ui.Row(c).Height(44).Padding(0, 16).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, dg.Caption).FontWeight(600).Grow(1).SingleLine()
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
			cv.HandleInput(viewerInput(&v.fit, &v.scale, &v.panX, &v.panY, 0.2, 6))
			const designW = float32(520)
			const designH = float32(150)
			cv.Draw(func(p *ui.Painter, r ui.Rect) {
				if cv.Pressed() {
					if dx, dy, ok := cv.Dragged(); ok {
						v.fit = false
						v.panX += dx
						v.panY += dy
					}
				}
				w, h := r.W, r.H
				sc := v.scale
				if w > 0 && h > 0 && v.fit {
					sc = min(w/designW, h/designH) * 0.9
					v.scale = sc
					v.panX = (w - designW*sc) / 2
					v.panY = (h - designH*sc) / 2
				}
				ox, oy := r.X+v.panX, r.Y+v.panY
				pt := func(x, y float32) (float32, float32) { return ox + x*sc, oy + y*sc }
				if cv.Pressed() {
					if cx, cy, ok := cv.PointerPosition(); ok {
						v.lastX, v.lastY = (cx-ox)/sc, (cy-oy)/sc
					}
				}
				for _, e := range dg.Edges {
					pts := make([][2]float32, len(e))
					for i, q := range e {
						x, y := pt(q[0], q[1])
						pts[i] = [2]float32{x, y}
					}
					p.StrokePolyline(pts, 1.5*sc, t.Tx3)
				}
			})
			cv.Children(func() {
				sc := v.scale
				for i, n := range dg.Nodes {
					bx := ui.Row(c.Key("cvn-"+itoa(i))).Absolute().Top(v.panY+n.Y*sc).Left(v.panX+n.X*sc).Size(n.W*sc, 30*sc).Radius(7).Justify(ui.Center).AlignItems(ui.Center).
						Background(t.Raised).Border(1, t.Bd3).FontSize(theme.FsCap)
					if n.Hot {
						bx.Background(t.AcSoft).BorderColor(t.AcLine).TextColor(t.AcTx)
					}
					bx.Children(func() { ui.Text(c, n.Label).SingleLine() })
				}
			})
			ui.Row(c).Height(22).Padding(0, 12).Gap(8).AlignItems(ui.Center).FontSize(theme.FsMicro).TextColor(t.Tx4).Children(func() {
				ui.Text(c, fmt.Sprintf("缩放 %.0f%% · 中心 (%.0f, %.0f)", v.scale*100, v.lastX, v.lastY))
				ui.Spacer(c).Grow(1)
				ui.Text(c, "拖拽平移 · 滚轮缩放 · ⇧方向键大步").SingleLine()
			})
		})
	})
	if !open {
		v.open = false
	}
}

func clamp(x, lo, hi float32) float32 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}
