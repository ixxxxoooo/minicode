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
		y := float32(tb.st.LineTop(s.Line)-tb.st.ScrollY())*edLineH*edScale - 16*edScale
		if y < codelensTop || y > 600 {
			continue
		}
		n := len(a.ide.findRefs(s.Name))
		lbl := itoa(n) + " 个引用"
		if tc := a.testsForSymbol(tb, s); tc > 0 {
			lbl += " · " + itoa(tc) + " 测试"
		}
		row := ui.Row(c).Key("cl-" + itoa(s.Line)).Absolute().Left(edGutter * edScale).Top(y).Height(14 * edScale).Gap(8).AlignItems(ui.Center).
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
