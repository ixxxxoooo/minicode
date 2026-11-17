package app

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

var scopeLog = diag.L("ide-scope")

const scopeBarW = 2.0

func (a *App) scopeOn() bool {
	return a.prefs.Bool(pEdScope, true)
}

func (a *App) scopeWithBg() bool {
	return a.prefs.Int(pEdScopeStyle, 0) == 1
}

// scopeBarX 行号槽最左竖条列(在追溯栏右侧,与 Git 改动条分列)。
func scopeBarX(tb *ideTab, a *App) float32 {
	x := float32(0)
	if a != nil && tb != nil && tb.st != nil && !tb.diff && tb.virt == "" &&
		a.prefs.Bool(pBlameGutter, false) && !tb.st.SoftWrap {
		x = 88
	}
	return x
}

// scopeSnapshot 返回光标与悬停所在范围(符号表按文件版本缓存,只按行号查链)。
func (a *App) scopeSnapshot(tb *ideTab) scopeSnap {
	out := scopeSnap{inner: -1, hover: -1}
	if tb == nil || tb.buf == nil || tb.st == nil || tb.diff || tb.virt != "" {
		return out
	}
	ver := tb.buf.Version()
	line := tb.st.CursorPos().Line
	if tb.scopeCacheVer == ver && tb.scopeCacheLn == line && tb.scopeHoverLn == tb.scopeHoverLnCache {
		return tb.scopeCache
	}
	syms := a.symbolsForFile(tb.path)
	if len(syms) == 0 {
		syms = a.ide.fileSyms(tb.path)
	}
	out.syms = syms
	if len(syms) > 0 {
		if e, ok := a.lspH.outline[tb.path]; ok && !e.busy && len(e.syms) > 0 && e.ver == ver {
			out.source = "lsp"
		} else {
			out.source = "local"
		}
	}
	out.chain = parentChain(syms, line)
	out.inner = innermostSymbol(syms, line)
	if tb.scopeDbgHover && out.inner >= 0 {
		if p := syms[out.inner].Parent; p >= 0 {
			out.hover = p
		}
	} else if tb.scopeHoverLn >= 0 {
		out.hover = innermostSymbol(syms, tb.scopeHoverLn)
	}
	tb.scopeCacheVer = ver
	tb.scopeCacheLn = line
	tb.scopeHoverLnCache = tb.scopeHoverLn
	tb.scopeCache = out
	return out
}

func (a *App) scopeLineBackgrounds(tb *ideTab) []editor.LineBackground {
	if !a.scopeOn() || !a.scopeWithBg() {
		return nil
	}
	snap := a.scopeSnapshot(tb)
	if snap.inner < 0 {
		return nil
	}
	s := snap.syms[snap.inner]
	return []editor.LineBackground{{First: s.Line, Last: s.End, Color: a.tk.Ac.Alpha(.06)}}
}

// scopeOverlay 画行号槽竖条、悬停热区与点击选中整块。
func (a *App) scopeOverlay(c *ui.Context, tb *ideTab, edEl ui.Element) {
	if !a.scopeOn() || tb == nil || tb.st == nil || tb.buf == nil || tb.diff || tb.virt != "" || tb.st.SoftWrap {
		return
	}
	snap := a.scopeSnapshot(tb)
	if len(snap.syms) == 0 {
		return
	}
	first, last := tb.st.VisibleLines()
	if last < first {
		return
	}
	t := a.tk
	bx := scopeBarX(tb, a) + scopeGutterPad - 2
	// 指针在代码区:按可见行更新悬停预览(符号表已缓存,只查该行最内层块)
	if edEl.Hovered() {
		if px, py, ok := edEl.PointerPosition(); ok && px >= edGutterW(tb) {
			ln := tb.st.CursorPos().Line
			for i := first; i <= last; i++ {
				y0 := float32(tb.st.LineTop(i) - tb.st.ScrollY())
				y1 := y0 + edLineH*edScale
				if py >= y0 && py < y1 {
					ln = i
					break
				}
			}
			if ln != tb.scopeHoverLn {
				tb.scopeHoverLn = ln
			}
		}
	} else if !tb.scopeDbgHover && tb.scopeHoverLn >= 0 {
		tb.scopeHoverLn = -1
	}
	snap = a.scopeSnapshot(tb)
	drawBar := func(symIx int, strong bool, hover bool) {
		if symIx < 0 || symIx >= len(snap.syms) {
			return
		}
		s := snap.syms[symIx]
		if s.End < first || s.Line > last {
			return
		}
		y0 := float32(tb.st.LineTop(s.Line)-tb.st.ScrollY()) + edLineH*edScale*0.05
		y1 := float32(tb.st.LineTop(s.End)-tb.st.ScrollY()) + edLineH*edScale*0.95
		col := t.Ac.Alpha(.28)
		if strong {
			col = t.Ac
		} else if hover {
			col = t.Info.Alpha(.55)
		}
		h := y1 - y0
		if h < edLineH*edScale {
			h = edLineH * edScale
		}
		key := fmt.Sprintf("sc-rail-%d-%d", symIx, boolInt(strong))
		z := ui.Box(c).Key(key).Absolute().Left(bx).Top(y0).Width(scopeBarW).Height(h)
		z.Background(col).Radius(1)
		// 首尾横杆
		cap := func(top bool) {
			yy, suffix := y0, "-cap-t"
			if !top {
				yy, suffix = y1-2, "-cap-b"
			}
			ui.Box(c).Key(key + suffix).Absolute().Left(bx - 1).Top(yy).Width(4).Height(2).Background(col).Radius(1)
		}
		cap(true)
		cap(false)
		// 点击竖条选中整块
		hit := ui.Box(c).Key(key + "-hit").Absolute().Left(bx - 2).Top(y0).Width(8).Height(h).Cursor(ui.CursorPointer)
		if hit.Clicked() {
			a.selectSymbolBlock(tb, s)
			scopeLog.Info("点击范围竖条选中块", "符号", s.Name, "行", s.Line+1)
		}
		if hit.Hovered() && !tb.scopeDbgHover {
			tb.scopeHoverLn = s.Line + (s.End-s.Line)/2
		}
	}
	// 外层变淡
	for _, ix := range snap.chain {
		if ix == snap.inner {
			continue
		}
		drawBar(ix, false, false)
	}
	if snap.hover >= 0 && snap.hover != snap.inner {
		drawBar(snap.hover, false, true)
	}
	if snap.inner >= 0 {
		drawBar(snap.inner, true, false)
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (a *App) selectSymbolBlock(tb *ideTab, s symbol) {
	if tb == nil || tb.st == nil || tb.buf == nil {
		return
	}
	start := tb.buf.Offset(editor.Pos{Line: s.Line, Col: 0})
	endLn := s.End
	if endLn >= tb.buf.LineCount() {
		endLn = tb.buf.LineCount() - 1
	}
	end := tb.buf.Offset(editor.Pos{Line: endLn, Col: len(tb.buf.Line(endLn))})
	tb.st.Select(start, end)
	tb.st.ScrollTo(s.Line)
}

// scopeStickyBar 在粘性滚动行左侧画竖条色标。
func (a *App) scopeStickyBar(c *ui.Context, sym symbol) {
	if !a.scopeOn() {
		return
	}
	ui.Box(c).Width(2).Height(theme.Px(18)).Background(a.tk.Ac).Radius(1).Shrink(0)
}

func (a *App) toggleScopeHighlight() {
	on := !a.scopeOn()
	if on {
		a.prefs.Set(pEdScope, true)
	} else {
		a.prefs.Reset(pEdScope)
	}
	if on {
		a.toast = "已开启高亮当前所在范围"
	} else {
		a.toast = "已关闭高亮当前所在范围"
	}
	scopeLog.Info("切换所在范围高亮", "开", on)
}

func scopeTip(sym symbol) string {
	return fmt.Sprintf("光标所在的符号 · 第 %d–%d 行", sym.Line+1, sym.End+1)
}

func parseScopeLineArg(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

// dbgScopeLine 把光标放到指定行并打开演示路由文件。
func (a *App) dbgScopeLine(line int) {
	a.SetMode(ModeIDE)
	tb := a.ide.cur()
	if tb == nil || tb.path != "src/router.rs" {
		a.ide.openFile("src/router.rs", true)
		tb = a.ide.cur()
	}
	if tb != nil && tb.st != nil {
		tb.scopeDbgHover = false
		tb.scopeHoverLn = -1
		tb.st.GoTo(editor.Pos{Line: line - 1, Col: 0})
	}
}

// dbgScopeHover 演示悬停预览:光标在内层 fn,悬停外层 impl。
func (a *App) dbgScopeHover() {
	a.SetMode(ModeIDE)
	a.ide.openFile("src/router.rs", true)
	tb := a.ide.cur()
	if tb != nil && tb.st != nil {
		tb.st.GoTo(editor.Pos{Line: 16, Col: 8})
		tb.scopeDbgHover = true
		tb.scopeHoverLn = -1
	}
}

// scopeIndentGuideTheme 略抬高活动缩进线对比度(仍走库绘制)。
func (a *App) scopeIndentGuideTheme(th *editor.Theme) {
	if !a.scopeOn() || th == nil {
		return
	}
	t := a.tk
	th.ActiveIndentGuide = t.Ac.Alpha(.45)
}
