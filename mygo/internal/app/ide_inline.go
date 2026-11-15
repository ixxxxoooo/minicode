package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 行内编辑块与 Claude 改动行的审查条。
type inlinePatch struct {
	tab    string
	start  int
	end    int
	old    string
	new    string
	prompt string
}

func (a *App) submitInlineEdit(tb *ideTab, prompt string) {
	if tb == nil || tb.st == nil || tb.buf == nil {
		return
	}
	s := tb.st.Selection()
	lo, hi := s.Anchor, s.Head
	if hi < lo {
		lo, hi = hi, lo
	}
	if lo == hi {
		a.toast = "先选中要修改的代码"
		return
	}
	old := tb.buf.Slice(lo, hi)
	newText := demoInlineRewrite(old, prompt)
	if newText == "" {
		newText = old + " // " + prompt
	}
	if a.inl == nil {
		a.inl = &inlinePatch{}
	}
	*a.inl = inlinePatch{tab: tb.id, start: lo, end: hi, old: old, new: newText, prompt: prompt}
	a.closeOv()
	a.toast = "已生成行内修改预览 · Tab 或点「接受」写回"
}

func demoInlineRewrite(src, prompt string) string {
	p := strings.ToLower(prompt)
	if strings.Contains(p, "忽略大小写") {
		return strings.ReplaceAll(src, "ToLower", "EqualFold")
	}
	if strings.Contains(p, "注释") {
		lines := strings.Split(src, "\n")
		for i, l := range lines {
			if strings.TrimSpace(l) != "" && !strings.HasPrefix(strings.TrimSpace(l), "//") {
				lines[i] = "// " + l
			}
		}
		return strings.Join(lines, "\n")
	}
	return ""
}

func (a *App) acceptInline() {
	p := a.inl
	if p == nil || p.tab == "" {
		return
	}
	tb := a.ide.tab(p.tab)
	if tb == nil || tb.buf == nil || tb.st == nil {
		a.inl = nil
		return
	}
	tb.buf.Edit(editor.Edit{Start: p.start, End: p.end, Text: p.new})
	tb.st.Select(p.start, p.start+len(p.new))
	a.inl = nil
	a.toast = "已接受行内修改"
}

func (a *App) revertInline() {
	a.inl = nil
	a.toast = "已撤消行内修改预览"
}

// inlinePatchUI 在选区下方画预览条。
func (a *App) inlinePatchUI(c *ui.Context, tb *ideTab) {
	p := a.inl
	if p == nil || p.tab != tb.id {
		return
	}
	t := a.tk
	s := tb.st.Selection()
	pos := tb.buf.Pos(s.Head)
	x, y := cursorXY(tb, pos.Line, pos.Col)
	row := ui.Row(c).Key("inl-bar").Absolute().Left(x).Top(y+28).Padding(6, 8).Gap(6).AlignItems(ui.Center).
		Background(t.Raised).Radius(t.RMd()).Border(1, t.Ac).FontSize(theme.FsCap)
	row.Children(func() {
		ui.Text(c, "行内修改预览").TextColor(t.Ac)
		kit.Btn(c, "接受", kit.BtnOpt{Small: true, Variant: kit.BtnPri}).OnClick(a.acceptInline)
		kit.Btn(c, "撤销", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(a.revertInline)
		ui.Text(c, ellipsize(p.new, 40)).TextColor(t.Tx3).Font("monospace")
	})
	// 用半透明紫底标出将被替换的范围
	tb.st.TextBackgrounds = append(tb.st.TextBackgrounds, editor.TextBackground{
		Start: p.start, End: p.end, Color: t.Ac.Alpha(.22),
	})
}

func ellipsize(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// reviewHover 光标在 Claude 改动行时,行尾显示接受 / 撤销。
func (a *App) reviewHover(c *ui.Context, tb *ideTab) {
	lines := a.claudeLines(tb)
	if len(lines) == 0 {
		return
	}
	line := tb.st.CursorPos().Line
	var hit *[2]int
	for _, g := range lines {
		if line >= g[0] && line <= g[1] {
			hit = &g
			break
		}
	}
	if hit == nil {
		return
	}
	t := a.tk
	_, y := cursorXY(tb, line, 0)
	bar := ui.Row(c).Key("rv-hov").Absolute().Right(a.mmWidth()+mmRulW+20).Top(y).Height(22).Gap(6).PaddingX(8).
		Background(t.Raised).Radius(t.RSm()).Border(1, t.Bd1).FontSize(theme.FsMicro)
	bar.Children(func() {
		ui.Text(c, "Claude 改动").TextColor(t.Ac)
		kit.Btn(c, "接受", kit.BtnOpt{Small: true}).OnClick(func() { a.toast = "已标记接受(演示)" })
		kit.Btn(c, "撤销", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.toast = "已标记撤销(演示)" })
	})
}
