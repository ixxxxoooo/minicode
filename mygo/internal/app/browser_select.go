package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/browser"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

type brSelect struct {
	open    bool
	startX  float32
	startY  float32
	endX    float32
	endY    float32
	drag    bool
	section int
	snip    string
}

func (a *App) browserReaderSelectLayer(c *ui.Context, pg browser.Page) {
	v := &a.brs
	sel := &a.brSel
	if pg.URL == "" {
		return
	}
	t := a.tk
	layer := ui.Box(c).Grow(1)
	layer.HitTest(func(x, y float32, _ ui.Rect) bool { return sel.open || sel.drag })
	layer.Draw(func(p *ui.Painter, r ui.Rect) {
		if !sel.drag && !sel.open {
			return
		}
		x0, y0 := min(sel.startX, sel.endX), min(sel.startY, sel.endY)
		x1, y1 := max(sel.startX, sel.endX), max(sel.startY, sel.endY)
		if x1-x0 < 4 && y1-y0 < 4 {
			return
		}
		p.Fill(ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}, t.AcSoft, 2)
		p.Stroke(ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}, t.Ac, 2, 1)
	})
	if layer.Pressed() {
		if px, py, ok := layer.PointerPosition(); ok {
			if !sel.drag {
				sel.drag = true
				sel.startX, sel.startY = px, py
			}
			sel.endX, sel.endY = px, py
		}
	}
	if layer.PressedOutside() || (!layer.Pressed() && sel.drag) {
		if sel.drag {
			sel.drag = false
			sel.open = sel.endX-sel.startX > 6 || sel.endY-sel.startY > 6
			if sel.open {
				sel.section, sel.snip = a.pickBrowserSnippet(pg, sel.startY, sel.endY)
			}
		}
	}
	if sel.open && sel.snip != "" {
		ui.Row(c).Absolute().Bottom(12).Left(12).Right(12).Padding(8, 10).Gap(8).Radius(t.RMd()).
			Background(t.Raised).Border(1, t.Bd2).Children(func() {
			ui.Text(c, "已选取段落").FontWeight(500)
			ui.Text(c, sel.snip).FontSize(theme.FsCap).LineHeight(1.4).Grow(1).MinWidth(0).MaxHeight(48)
			kit.Btn(c, "发给 Claude", kit.BtnOpt{Small: true, Variant: kit.BtnPri, Icon: "chat"}).OnClick(func() {
				ref := fmt.Sprintf("@浏览器:%s#%d", browser.Normalize(v.url), sel.section)
				a.draft = appendRefs(a.draft, ref)
				a.AskClaude("请根据页面选区:\n" + sel.snip)
				sel.open = false
				log.Info("浏览器选区引用", "url", v.url, "section", sel.section)
			})
			kit.Ib(c, "x", kit.IbOpt{Small: true}).OnClick(func() { sel.open = false })
		})
	}
}

func (a *App) pickBrowserSnippet(pg browser.Page, y0, y1 float32) (int, string) {
	v := &a.brs
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	best, bestIx := -1, -1
	for i, y := range v.pos {
		if y >= y0-20 && (best < 0 || y < v.pos[bestIx]) {
			bestIx, best = i, int(y)
		}
	}
	if bestIx < 0 || bestIx >= len(pg.Sections) {
		return 0, ""
	}
	s := pg.Sections[bestIx]
	text := strings.TrimSpace(s.Heading + "\n" + stripMD(s.Body))
	if len([]rune(text)) > 400 {
		text = string([]rune(text)[:400]) + "…"
	}
	return bestIx, text
}

func stripMD(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "`", "")
	return strings.TrimSpace(s)
}
