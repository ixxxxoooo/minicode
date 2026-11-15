package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// CodeLens 叠层:库没有独立 CodeLens 行,在符号声明行上方画可点胶囊。
func (a *App) codeLens(c *ui.Context, tb *ideTab) {
	if tb == nil || tb.st == nil || tb.buf == nil || tb.diff || tb.virt != "" {
		return
	}
	if !a.prefs.Bool("editor.codelens", true) {
		return
	}
	first, last := tb.st.VisibleLines()
	if last <= first {
		return
	}
	stickyRows := a.stickyRows(tb, first)
	codelensTop := 8 + float32(stickyRows)*24*edScale
	syms := a.ide.fileSyms(tb.path)
	t := a.tk
	for _, s := range syms {
		if s.Kind == "h" || s.Line < first-1 || s.Line > last+1 {
			continue
		}
		if s.Kind != "fn" && s.Kind != "struct" && s.Kind != "class" && s.Kind != "type" && s.Kind != "trait" {
			continue
		}
		// LineTop / ScrollY 单位就是 DIP(已含字号缩放),不能再乘行高。
		// 库没有行间小部件,胶囊画在声明行右端(不盖住上一行的注释)。
		lineH := edLineH * edScale
		y := float32(tb.st.LineTop(s.Line)-tb.st.ScrollY()) + (lineH-14*edScale)/2
		// 声明行太长会和胶囊相撞:按字符数估算,放不下就不画(宁缺勿叠)。
		avail := float32(420)
		if a.wsW > 0 {
			avail = a.wsW - edGutter*edScale - a.codeLensRight() - 150
		}
		if float32(len([]rune(tb.buf.Line(s.Line))))*7.3*edScale > avail {
			continue
		}
		if y < codelensTop-lineH || s.Line < first || s.Line > last {
			continue
		}
		n := 0
		for _, r := range a.ide.findRefs(s.Name) {
			if r.path == tb.path && r.line == s.Line { // 声明行自己不算引用
				continue
			}
			n++
		}
		lbl := itoa(n) + " 个引用"
		if tc := a.testsForSymbol(tb, s); tc > 0 {
			lbl += " · " + itoa(tc) + " 测试"
		}
		row := ui.Row(c).Key("cl-" + itoa(s.Line)).Absolute().Right(a.codeLensRight()).Top(y).Height(14 * edScale).Gap(8).AlignItems(ui.Center).
			FontSize(theme.FsMicro).TextColor(t.Tx4)
		row.Children(func() {
			link := ui.Text(c, lbl).Cursor(ui.CursorPointer).TextColor(t.Tx3)
			if link.Clicked() {
				a.showRefs(s.Name)
			}
			ask := ui.Text(c, "问 Claude").Cursor(ui.CursorPointer).TextColor(t.Ac)
			if ask.Clicked() {
				a.draft = appendRefs(a.draft, refText(tb.path, false))
				a.draft = strings.TrimSpace(a.draft + "\n解释一下 " + s.Name)
				a.ux.focusDraft = true
			}
		})
	}
}

// codeLensRight 胶囊离编辑区右缘的距离:让开迷你地图 / 概览标尺与滚动条。
func (a *App) codeLensRight() float32 {
	if !a.wsNarrowMM() && a.prefs.Bool(pMinimap, true) {
		return 110
	}
	return 24
}
