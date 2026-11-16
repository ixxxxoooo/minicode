package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const maxMcCursors = 1000

var mcLog = diag.L("ide.mc")

// tbCodeEditable 当前标签是否可用代码编辑器里的多光标与块编辑(与设计稿「不可用」一致)。
func (a *App) tbCodeEditable(tb *ideTab) (bool, string) {
	if tb == nil || tb.st == nil || tb.buf == nil {
		return false, "没有打开的编辑器"
	}
	if tb.diff {
		return false, "差异标签的此侧不可编辑"
	}
	if tb.virt != "" {
		return false, "虚拟标签不可编辑"
	}
	if tb.st.ReadOnly {
		return false, "只读"
	}
	if isMarkdownFile(tb.path) && !a.mdSource(tb) {
		return false, "只读 Markdown 预览"
	}
	return true, ""
}

func (tb *ideTab) mcCount() int {
	if tb == nil || tb.st == nil {
		return 0
	}
	return len(tb.st.Selections())
}

func (a *App) mcRun(tb *ideTab, cmd editor.Command) {
	if ok, why := a.tbCodeEditable(tb); !ok {
		a.toast = why
		return
	}
	before := tb.mcCount()
	if !tb.st.Run(cmd) {
		if cmd == editor.SelectNext {
			a.toast = "没有更多匹配"
		}
		return
	}
	a.mcClamp(tb, before)
}

func (a *App) mcClamp(tb *ideTab, before int) {
	n := tb.mcCount()
	if n <= maxMcCursors {
		return
	}
	a.toast = "已达 1000 个光标上限"
	sels := tb.st.Selections()
	if len(sels) > maxMcCursors {
		sels = sels[:maxMcCursors]
		tb.st.SetSelections(sels...)
	}
	mcLog.Info("多光标触及上限", "path", tb.path, "count", n, "before", before)
}

func (a *App) mcSingle(tb *ideTab) {
	if tb == nil || tb.st == nil || tb.mcCount() <= 1 {
		return
	}
	mcLog.Info("回到单光标", "path", tb.path, "count", tb.mcCount())
	tb.st.Run(editor.Cancel)
}

func (a *App) mcLineEnds(tb *ideTab) {
	if ok, why := a.tbCodeEditable(tb); !ok {
		a.toast = why
		return
	}
	r := tb.buf
	seen := map[int]bool{}
	var offs []int
	add := func(off int) {
		if !seen[off] {
			seen[off] = true
			offs = append(offs, off)
		}
	}
	for _, s := range tb.st.Selections() {
		lo, hi := s.Anchor, s.Head
		if lo > hi {
			lo, hi = hi, lo
		}
		pa, pb := r.Pos(lo), r.Pos(hi)
		for line := pa.Line; line <= pb.Line; line++ {
			ln := r.Line(line)
			add(r.Offset(editor.Pos{Line: line, Col: len(ln)}))
		}
	}
	if len(offs) == 0 {
		return
	}
	if len(offs) > maxMcCursors {
		a.toast = "已达 1000 个光标上限"
		offs = offs[:maxMcCursors]
	}
	sels := make([]editor.Selection, len(offs))
	for i, o := range offs {
		sels[i] = editor.Selection{Anchor: o, Head: o}
	}
	tb.st.SetSelections(sels...)
	mcLog.Info("在行尾添加光标", "path", tb.path, "count", len(offs))
}

func (a *App) edMcInput(tb *ideTab, ev ui.InputEvent) bool {
	if ev.Kind != ui.InputKeyDown || ev.Key == ui.KeyUnknown {
		return false
	}
	var cmd editor.Command
	var customLineEnds bool
	switch {
	case ev.Mods == ui.Cmd && ev.Key == ui.KeyD:
		cmd = editor.SelectNext
	case ev.Mods == (ui.Cmd|ui.Shift) && ev.Key == ui.KeyL:
		cmd = editor.SelectAllMatches
	case ev.Mods == (ui.Cmd|ui.Alt) && ev.Key == ui.KeyUp:
		cmd = editor.AddCursorAbove
	case ev.Mods == (ui.Cmd|ui.Alt) && ev.Key == ui.KeyDown:
		cmd = editor.AddCursorBelow
	case ev.Mods == (ui.Alt|ui.Shift) && ev.Key == ui.KeyI:
		customLineEnds = true
	default:
		return false
	}
	if ok, why := a.tbCodeEditable(tb); !ok {
		if customLineEnds || cmd != 0 {
			a.toast = why
		}
		return true
	}
	if customLineEnds {
		a.mcLineEnds(tb)
		return true
	}
	a.mcRun(tb, cmd)
	return true
}

func (a *App) mcSync(tb *ideTab) {
	if tb == nil || tb.st == nil {
		return
	}
	a.mcClamp(tb, tb.mcCount())
}

func (a *App) mcStatusChip(c *ui.Context, tb *ideTab) {
	n := tb.mcCount()
	if n <= 1 {
		return
	}
	chip := kit.Chip(c, itoa(n)+" 个光标", kit.ChipOpt{Line: true, On: a.popup == "mcsel"})
	chip.OnClick(func() { a.togglePop("mcsel") })
	a.mcPop(c, chip, tb)
}

func (a *App) mcPop(c *ui.Context, anchor ui.Element, tb *ideTab) {
	a.popOpen("mcsel", func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 228).AttachTo(anchor, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(0, 0, 4, 0)
			panel.Column().Gap(2).Children(func() {
				if a.mcPopItem(c, "mc-x", "x", "回到单光标", "Esc").Clicked() {
					a.mcSingle(tb)
					a.popup = ""
				}
				if a.mcPopItem(c, "mc-all", "layers", "选中所有匹配", "⇧", "⌘", "L").Clicked() {
					a.mcRun(tb, editor.SelectAllMatches)
					a.popup = ""
				}
				if a.mcPopItem(c, "mc-end", "edit", "在行尾添加光标", "⇧", "⌥", "I").Clicked() {
					a.mcLineEnds(tb)
					a.popup = ""
				}
			})
		})
	})
}

func (a *App) mcPopItem(c *ui.Context, key, icon, text string, keys ...string) ui.Element {
	t := a.tk
	b := ui.ButtonBase(c).Key(key).Height(theme.Px(26)).PaddingX(8).Gap(8).Radius(t.RSm()).AlignItems(ui.Center).
		Cursor(ui.CursorDefault).Transition(kit.Fade)
	if b.Hovered() {
		b.Background(t.Fill1)
	}
	return b.Children(func() {
		kit.Ic(c, icon, kit.S12)
		ui.Text(c, text).Grow(1).FontSize(theme.FsBody)
		if len(keys) > 0 {
			kit.Kbd(c, keys...)
		}
	})
}

// mcEditMenu 在不可编辑时于右键菜单底部列出多光标项(置灰 + 原因)。
func (a *App) mcEditMenu(tb *ideTab, m *ui.Menu) {
	ok, why := a.tbCodeEditable(tb)
	if ok {
		return
	}
	m.Separator()
	m.Submenu("选择", func(m *ui.Menu) {
		for _, it := range []struct {
			label, key string
		}{
			{"添加光标", "⌥ 点击"},
			{"添加下一个匹配", "⌘D"},
			{"选中所有匹配", "⇧⌘L"},
		} {
			menuItem(m, it.label+"  "+it.key).Disabled(true).Tooltip(why)
		}
	})
}

func (a *App) dbgMultiCursor(tb *ideTab) {
	if tb == nil || tb.buf == nil || tb.st == nil {
		return
	}
	const word = "normalize"
	var sels []editor.Selection
	for i := 0; i < tb.buf.LineCount() && len(sels) < 3; i++ {
		ln := tb.buf.Line(i)
		base := tb.buf.Offset(editor.Pos{Line: i, Col: 0})
		for _, c := range wordHits(ln, word) {
			off := base + c
			sels = append(sels, editor.Selection{Anchor: off, Head: off + len(word)})
			if len(sels) >= 3 {
				break
			}
		}
	}
	if len(sels) > 0 {
		tb.st.SetSelections(sels...)
	}
	mcLog.Info("调试多光标", "path", tb.path, "count", len(sels))
}
