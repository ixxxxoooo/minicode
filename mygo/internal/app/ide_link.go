package app

import (
	"math"
	"sort"

	"github.com/charmbracelet/x/ansi"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/lsp"
	"github.com/ixxxxoooo/minicode/internal/symbols"
)

// ⌘ 悬停可跳转(设计稿 ide-nav「⌘ 悬停可跳转」):按住 ⌘,指针下能跳转的标识符
// 加下划线 + 手型光标,⌘点击(库自带)直接转到定义。没有定义的词不加下划线。
// 位置按「行号栏宽 + 显示列 × 字宽」估算(与 codeLens / 追溯卡同一套常量),
// 自动换行时不画(列与屏幕位置对不上)。

var linkLog = diag.L("ide-link")

// edLink 是当前被指的可跳转单词。
type edLink struct {
	word   string
	line   int
	c0, c1 int // 显示列(Tab / 宽字符已换算)
}

// linkMemo 缓存「单词 → 是否有定义」,避免指针每动一下就扫一遍符号索引 / LSP。
type linkMemo struct {
	word string
	path string
	off  int
	ok   bool
	at   uint64 // 缓存时的缓冲区版本,变了就重算
	lsp  int8   // -1 未查,0 无定义,1 有定义
}

// edTextPad 是行号栏右边到第一列字符之间的内边距(实测 5~6pt)。
const edTextPad = 6.0

// edGutterW 复现库的 gutterWidth:行号(至少 3 位)+ 折叠箭头 + Git 竖条 + 4px 留白。
// 行号位数随总行数变,所以不能用常量 edGutter(那只是 3 位数、无 Git 条时的近似)。
func edGutterW(tb *ideTab) float32 {
	if tb == nil || tb.st == nil || tb.buf == nil {
		return edGutter
	}
	cw := edChar * edScale
	w := float32(0)
	if tb.st.LineNumbers != editor.LineNumbersOff {
		digits := len(itoa(tb.buf.LineCount()))
		if digits < 3 {
			digits = 3
		}
		w += float32(digits)*cw + cw*1.5
	}
	if !tb.st.NoFolding {
		w += 20 * edScale * 0.8
	}
	if tb.st.DiffBase != nil {
		w += 6
	}
	if w > 0 {
		w += 4
	}
	return float32(math.Ceil(float64(w)))
}

// displayCols 把一行里 [b0,b1) 字节区间换算成显示列(Tab 对齐到 tabW,宽字符占 2 格)。
func displayCols(line string, b0, b1, tabW int) (c0, c1 int) {
	if tabW <= 0 {
		tabW = 4
	}
	cell := 0
	for i, r := range line {
		if i == b0 {
			c0 = cell
		}
		if i >= b1 {
			break
		}
		if r == '\t' {
			cell += tabW - cell%tabW
		} else {
			cell += ansi.StringWidth(string(r))
		}
		c1 = cell
	}
	if b0 >= len(line) {
		c0 = cell
	}
	if b1 >= len(line) {
		c1 = cell
	}
	return
}

// byteAtCell 把显示列换成该行的字节列(落在 Tab / 宽字符中间算它本身)。
func byteAtCell(line string, cell, tabW int) int {
	if tabW <= 0 {
		tabW = 4
	}
	cur := 0
	for i, r := range line {
		w := 0
		if r == '\t' {
			w = tabW - cur%tabW
		} else {
			w = ansi.StringWidth(string(r))
		}
		if cell < cur+w {
			return i
		}
		cur += w
	}
	return len(line)
}

// linkLineAt 按指针 y(相对编辑器上沿)找行:LineTop 单调不减,二分取最后一个顶不低于指针的行。
func linkLineAt(tb *ideTab, y float32) (int, bool) {
	n := tb.buf.LineCount()
	if n == 0 {
		return 0, false
	}
	yy := y + float32(tb.st.ScrollY())
	i := sort.Search(n, func(l int) bool { return float32(tb.st.LineTop(l)) > yy })
	l := i - 1
	if l < 0 {
		return 0, false
	}
	top := float32(tb.st.LineTop(l))
	if yy >= top+edLineH*edScale { // 超出最后一行
		return 0, false
	}
	return l, true
}

// linkJumpable 判断这个词能不能转到定义:本地索引或已确认的 LSP 答案;未确认前不加下划线。
func (a *App) linkJumpable(tb *ideTab, w string, off int) bool {
	if w == "" || symbols.IsKeyword(w) {
		return false
	}
	c := a.ide.linkMemo
	ver := tb.buf.Version()
	if c.word == w && c.path == tb.path && c.at == ver && c.off == off {
		if c.ok || c.lsp == 1 {
			return true
		}
		if c.lsp == 0 {
			return false
		}
	}
	if len(a.ide.findDefs(w, tb.path)) > 0 {
		a.ide.linkMemo = linkMemo{word: w, path: tb.path, off: off, ok: true, at: ver, lsp: 1}
		return true
	}
	a.ide.linkMemo = linkMemo{word: w, path: tb.path, off: off, ok: false, at: ver, lsp: -1}
	if s := a.lspReadyFor(tb.path); s != nil {
		a.linkProbeDef(tb, off, w, ver, s)
	}
	return false
}

// linkProbeDef 后台问 LSP 是否有定义,答了再画下划线(不误标关键字 / 无定义词)。
func (a *App) linkProbeDef(tb *ideTab, off int, w string, ver uint64, s *lspSess) {
	key := tb.path + ":" + itoa(off)
	if a.ide.linkProbe == key {
		return
	}
	a.ide.linkProbe = key
	uri := lsp.PathToURI(a.absPath(tb.path))
	pos := lspPos(tb, off)
	a.lspJob("link", s, func() func() {
		locs, err := s.c.Definition(uri, pos)
		has := err == nil && len(locs) > 0
		return func() {
			a.ide.linkProbe = ""
			if tb.buf == nil || tb.buf.Version() != ver {
				return
			}
			lv := int8(0)
			if has {
				lv = 1
			}
			a.ide.linkMemo = linkMemo{word: w, path: tb.path, off: off, ok: has, at: ver, lsp: lv}
			if has {
				linkLog.Debug("⌘ 悬停:有定义", "path", tb.path, "词", w)
			}
			a.frameSoon()
		}
	})
}

// linkHover 每帧根据 ⌘ 与指针位置更新 tb.link,并给编辑器元素设手型光标。
func (a *App) linkHover(c *ui.Context, tb *ideTab, ev ui.Element) {
	prev := tb.link
	tb.link = nil
	defer func() {
		if (prev == nil) != (tb.link == nil) || prev != nil && tb.link != nil && *prev != *tb.link {
			a.frameSoon()
		}
	}()
	if tb.st == nil || tb.buf == nil || tb.diff || a.prefs.Bool(pEdWrap, false) {
		return
	}
	if c.Modifiers()&ui.Cmd == 0 {
		return
	}
	x, y, ok := ev.PointerPosition()
	gw := edGutterW(tb) + edTextPad
	if !ok || x < gw {
		return
	}
	l, ok := linkLineAt(tb, y)
	if !ok {
		return
	}
	ln := tb.buf.Line(l)
	cell := int((x - gw) / (edChar * edScale))
	col := byteAtCell(ln, cell, tb.st.TabSize)
	w, b0, b1 := symbols.WordAt(ln, col)
	off := tb.buf.Offset(editor.Pos{Line: l, Col: col})
	if w == "" || col >= len(ln) || !a.linkJumpable(tb, w, off) {
		return
	}
	c0, c1 := displayCols(ln, b0, b1, tb.st.TabSize)
	tb.link = &edLink{word: w, line: l, c0: c0, c1: c1}
	ev.Cursor(ui.CursorPointer)
}

// linkUnderline 画下划线(叠在编辑器上,不挡事件)。yOff 是编辑器上沿相对本层的偏移。
func (a *App) linkUnderline(c *ui.Context, tb *ideTab, yOff float32) {
	k := tb.link
	if k == nil {
		return
	}
	t := a.tk
	top := float32(tb.st.LineTop(k.line)-tb.st.ScrollY()) + edLineH*edScale - 4*edScale + yOff
	left := edGutterW(tb) + edTextPad + float32(k.c0)*edChar*edScale
	w := float32(k.c1-k.c0) * edChar * edScale
	ui.Box(c).Absolute().Left(left).Top(top).Width(w).Height(1).Background(t.AcTx).HitTest(func(float32, float32, ui.Rect) bool { return false })
}
