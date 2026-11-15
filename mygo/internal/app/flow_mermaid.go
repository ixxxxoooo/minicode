package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/mermaid"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const mmCacheMax = 64

type mmCached struct {
	scene mermaid.Scene
	err   error
	ready bool
}

type mmViewer struct {
	open  bool
	title string
	src   string
	scale float32
	panX  float32
	panY  float32
	fit   bool
}

func mmKey(src string) string {
	h := sha256.Sum256([]byte(src))
	return hex.EncodeToString(h[:16])
}

func (a *App) mmCacheInit() {
	if a.mmCache == nil {
		a.mmCache = map[string]*mmCached{}
	}
}

func (a *App) mmEnsure(src string) *mmCached {
	a.mmCacheInit()
	k := mmKey(src)
	if e, ok := a.mmCache[k]; ok {
		return e
	}
	if len(a.mmCache) >= mmCacheMax {
		a.mmCache = map[string]*mmCached{}
	}
	e := &mmCached{}
	a.mmCache[k] = e
	body := src
	a.bg("mermaid-render", func() func() {
		defer diag.SpanSlow("mermaid", "Render", 80)()
		sc, err := mermaid.Render(body)
		return func() {
			e.scene = sc
			if err != nil {
				e.err = err
			}
			e.ready = true
		}
	})
	return e
}

func (a *App) mmCardTitle(src string) string {
	if k, ok := mermaid.KindTitle(src); ok {
		return "Mermaid · " + k
	}
	return "Mermaid"
}

func (a *App) foldMermaidBlock(c *ui.Context, b *markdown.CodeBlockView) {
	src := strings.TrimRight(b.Code(), "\n")
	t := a.tk
	keyBase := foldKey("cb-mm", itoa(b.Index))
	srcKey := keyBase + ":src"
	showSrc := a.foldExpanded(srcKey)
	tile := a.mmCardTitle(src)
	entry := a.mmEnsure(src)
	a.box(c).Background(t.Sunken).Radius(t.RMd()).Border(1, t.Bd1).Clip().Children(func() {
		ui.Row(c).Height(28).PaddingX(8).Gap(6).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
			kit.Ic(c, "layers", kit.S12).TextColor(t.Tx3)
			ui.Text(c, tile).FontSize(theme.FsCap).TextColor(t.Tx3)
			ui.Text(c, mmHeadline(src)).Grow(1).SingleLine().FontSize(theme.FsCap).TextColor(t.Tx2)
			kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制源码"}).OnClick(func() { a.copyOut(src, "Mermaid 源码") })
			if entry.ready && entry.err == nil {
				kit.Ib(c, "max", kit.IbOpt{Small: true, Tip: "全屏查看"}).OnClick(func() {
					a.openMmViewer(tile, src)
				})
			}
			kit.Btn(c, map[bool]string{true: "查看图", false: "查看源码"}[showSrc], kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
				a.setFold(srcKey, !showSrc)
			})
		})
		if showSrc || (entry.ready && entry.err != nil) {
			a.mmSourceBody(c, src, entry.err)
		} else if !entry.ready {
			ui.Column(c).Height(120).Padding(12).Gap(8).Children(func() {
				kit.SkelRows(c, 3)
				ui.Text(c, "正在排版…").FontSize(theme.FsCap).TextColor(t.Tx3)
			})
		} else {
			a.mmSceneCard(c, &entry.scene, 1, false)
		}
	})
}

func mmHeadline(src string) string {
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%%") || line == "---" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "title ") {
			return strings.TrimSpace(line[6:])
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return strings.Join(parts[1:], " ")
		}
		break
	}
	return "图表"
}

func (a *App) mmSourceBody(c *ui.Context, src string, err error) {
	t := a.tk
	ui.Column(c).Padding(8).Gap(6).Children(func() {
		if err != nil {
			msg := err.Error()
			if re, ok := err.(mermaid.RenderError); ok {
				msg = re.Error()
			}
			ui.Text(c, msg+",以下为源码").FontSize(theme.FsCap).TextColor(t.Warn)
		}
		key := foldKey("cb-mm-src", mmKey(src))
		a.foldBody(c, key, src, false, true, func() { a.copyOut(src, "Mermaid 源码") })
	})
}

func (a *App) mmSceneCard(c *ui.Context, sc *mermaid.Scene, scale float32, wheelZoom bool) {
	if sc == nil || scale <= 0 {
		scale = 1
	}
	const maxH = float32(360)
	// 绘制层必须有明确尺寸:文字层是绝对定位,撑不开高度,否则整张图高度为 0 看不见
	w, h := a.mmSceneSize(sc, scale)
	if w <= 0 || h <= 0 {
		return
	}
	box := ui.Box(c).Clip()
	if wheelZoom {
		box.Grow(1)
	} else {
		box.Height(min(h+16, maxH)).Cursor(ui.CursorPointer)
	}
	box.Children(func() {
		layer := ui.Box(c).Size(w, h).Margin(8)
		layer.Draw(func(p *ui.Painter, r ui.Rect) {
			a.paintMmScene(p, sc, scale, r.X, r.Y)
		})
		layer.Children(func() {
			a.mmTextLayer(c, sc, scale)
		})
	})
}

func (a *App) paintMmScene(p *ui.Painter, sc *mermaid.Scene, scale float32, ox, oy float32) {
	pt := func(x, y float32) (float32, float32) { return ox + x*scale, oy + y*scale }
	for _, it := range sc.Items {
		switch {
		case it.Rect != nil:
			r := it.Rect
			x, y := pt(r.X, r.Y)
			rr := ui.Rect{X: x, Y: y, W: r.W * scale, H: r.H * scale}
			fill := a.mmColor(r.Fill, true)
			if fill.A > 0 {
				p.Fill(rr, fill, r.R*scale)
			}
			if r.Stroke != nil {
				p.Stroke(rr, a.mmColor(r.Stroke, false), r.R*scale, r.Width*scale)
			}
		case it.Poly != nil:
			poly := it.Poly
			if len(poly.Pts) < 2 {
				continue
			}
			path := &ui.Path{}
			path.MoveTo(pt(poly.Pts[0][0], poly.Pts[0][1]))
			for _, q := range poly.Pts[1:] {
				path.LineTo(pt(q[0], q[1]))
			}
			if poly.Closed {
				path.Close()
				if poly.Fill != nil {
					p.FillPath(path, a.mmColor(poly.Fill, true))
				}
			}
			if poly.Stroke != nil {
				p.StrokePath(path, poly.Width*scale, a.mmColor(poly.Stroke, false))
			}
		}
	}
}

func (a *App) mmTextLayer(c *ui.Context, sc *mermaid.Scene, scale float32) {
	for i, it := range sc.Items {
		if it.Text == nil {
			continue
		}
		t := it.Text
		col := a.mmColor(&t.Paint, false)
		row := ui.Row(c.Key("mmt-" + itoa(i))).Absolute().
			Top(t.Y*scale - t.Size*scale*0.65).
			FontSize(t.Size * scale).
			TextColor(col).
			MinWidth(t.Size * scale * 8)
		switch t.Anchor {
		case mermaid.AnchorMiddle:
			row.Left(t.X*scale - 200*scale).Width(400 * scale).Justify(ui.Center)
		case mermaid.AnchorEnd:
			row.Left(t.X*scale - 400*scale).Width(400 * scale).Justify(ui.End)
		default:
			row.Left(t.X * scale)
		}
		if t.Bold {
			row.FontWeight(600)
		}
		row.Children(func() { ui.Text(c, t.Text).SingleLine() })
	}
}

func (a *App) mmColor(p *mermaid.Paint, fill bool) ui.Color {
	if p == nil {
		return ui.Transparent
	}
	t := a.tk
	if rgb, ok := p.RGB(); ok {
		return ui.RGBA(byte((rgb>>16)&0xff), byte((rgb>>8)&0xff), byte(rgb&0xff), 255)
	}
	if acc, ok := p.Accent(); ok {
		cols := []ui.Color{t.Ac, t.Info, t.Ok, t.Warn, t.Err, t.Vio, t.Cyan, t.Pink}
		return cols[acc%uint8(len(cols))]
	}
	if role, ok := p.Role(); ok {
		switch role {
		case mermaid.RoleSurface:
			return t.BG
		case mermaid.RoleNode:
			return t.Raised
		case mermaid.RoleBorder:
			return t.Bd3
		case mermaid.RoleLine:
			return t.Tx3
		case mermaid.RoleText:
			return t.Tx1
		case mermaid.RoleMuted:
			return t.Tx3
		case mermaid.RoleNote:
			return t.WarnSoft
		case mermaid.RoleGroup:
			return t.Fill2
		default:
			if fill {
				return t.Raised
			}
			return t.Tx3
		}
	}
	if fill {
		return t.Raised
	}
	return t.Tx1
}

func (a *App) openMmViewer(title, src string) {
	a.mmView = mmViewer{open: true, title: title, src: src, scale: 1, fit: true}
	diag.L("app").Info("打开 Mermaid 全屏", "title", title)
}

func isMermaidInfo(info string) bool {
	// 没写语言的围栏(``` 后面直接换行)info 为空,不能直接取第一个词,否则越界 panic
	f := strings.Fields(strings.TrimSpace(info))
	return len(f) > 0 && strings.ToLower(f[0]) == "mermaid"
}

func (a *App) mmSceneSize(sc *mermaid.Scene, scale float32) (float32, float32) {
	if sc == nil {
		return 0, 0
	}
	return sc.Width * scale, sc.Height * scale
}

func (a *App) mmWheelZoom(v *mmViewer, dy float32, cx, cy float32, sceneW, sceneH float32) {
	if dy == 0 {
		return
	}
	f := float32(1.1)
	if dy > 0 {
		f = 1 / f
	}
	old := v.scale
	v.fit = false
	v.scale = clamp(v.scale*f, 0.2, 6)
	ratio := v.scale / old
	v.panX = cx - (cx-v.panX)*ratio
	v.panY = cy - (cy-v.panY)*ratio
	_, _ = sceneW, sceneH
}

func fmtMmPct(s float32) string {
	return fmt.Sprintf("%.0f%%", s*100)
}
