package app

import (
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diff"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 迷你地图 = 缩略的「语法色块」+ 视口滑块;右侧另有一条概览标尺(整篇缩放到面板高度)。
// 地图左缘:Git 改动竖条;叠加:搜索命中(黄)、同词出现(蓝)、Claude 改过的行(紫)。
// 标尺:错误 / 警告 / 搜索 / Git / 光标的刻度,点任意位置跳转。

const (
	mmRulW  = 8 // 概览标尺宽
	mmRowH  = float32(3.5)
	mmChar  = float32(0.6)
	mmMaxPx = float32(52)

	mmThumbIdle = 3 * time.Second // 停止滚动后视口滑块再显示多久
)

// run 是一行里的一段同色文字(列按字节)。
type run struct {
	a, b int
	kind byte // k 关键字 / s 字符串 / c 注释 / t 类型 / f 函数 / 0 其它
}

// lineRuns 把一行切成着色段:够给缩略图用,不求与编辑器的高亮逐字一致。
func lineRuns(ln string) []run {
	var out []run
	i := 0
	for i < len(ln) {
		ch := ln[i]
		switch {
		case ch == ' ' || ch == '\t':
			i++
		case ch == '/' && i+1 < len(ln) && ln[i+1] == '/' || ch == '#' && strings.TrimSpace(ln)[0] == '#':
			out = append(out, run{i, len(ln), 'c'})
			return out
		case ch == '"':
			j := i + 1
			for j < len(ln) && ln[j] != '"' {
				if ln[j] == '\\' {
					j++
				}
				j++
			}
			if j < len(ln) {
				j++
			}
			out = append(out, run{i, min(j, len(ln)), 's'})
			i = min(j, len(ln))
		case isWordByte(ch):
			j := i
			for j < len(ln) && isWordByte(ln[j]) {
				j++
			}
			w := ln[i:j]
			k := byte(0)
			switch {
			case isKeyword(w):
				k = 'k'
			case w[0] >= 'A' && w[0] <= 'Z':
				k = 't'
			case j < len(ln) && ln[j] == '(':
				k = 'f'
			}
			out = append(out, run{i, j, k})
			i = j
		default:
			j := i
			for j < len(ln) && !isWordByte(ln[j]) && ln[j] != ' ' && ln[j] != '"' {
				j++
			}
			if j == i {
				j++
			}
			out = append(out, run{i, j, 'p'})
			i = j
		}
	}
	return out
}

// gitMarks 对比 HEAD 与当前文本,给每个改动行一个标记:+ 新增 / ~ 修改 / - 其后有删除。键是新文件里的 0 起行号。
func gitMarks(head, cur string) map[int]byte {
	out := map[int]byte{}
	ls := diff.Lines(head, cur, 1, 1)
	for i, l := range ls {
		switch l.Kind {
		case diff.Add:
			k := byte('+')
			if i > 0 && ls[i-1].Kind == diff.Del || i+1 < len(ls) && ls[i+1].Kind == diff.Del {
				k = '~'
			}
			out[l.NewNo-1] = k
		case diff.Del:
			// 紧挨着的新增行已标 ~;孤立的删除标在下一行
			if i+1 < len(ls) && ls[i+1].Kind == diff.Same && ls[i+1].NewNo > 0 {
				out[ls[i+1].NewNo-1] = '-'
			}
		}
	}
	return out
}

// mmInfo 是一个文件标签的地图叠加数据(随缓冲区版本 / 光标词 / 查找词缓存)。
type mmInfo struct {
	ver   uint64
	word  string
	query string
	cs    bool
	git   map[int]byte
	word_ []int // 同词出现的行
	found []int // 搜索命中的行
}

// overlayInfo 取(或重算)标签的地图叠加数据。
func (a *App) overlayInfo(tb *ideTab) *mmInfo {
	ver := tb.buf.Version()
	word := a.tabWord(tb)
	q, cs := tb.findQuery()
	if m := tb.mm; m != nil && m.ver == ver && m.word == word && m.query == q && m.cs == cs {
		return m
	}
	m := &mmInfo{ver: ver, word: word, query: q, cs: cs}
	if tb.mm != nil && tb.mm.ver == ver {
		m.git = tb.mm.git
	} else if head, ok := a.ide.headOf(tb.path); ok {
		m.git = gitMarks(head, tb.buf.String())
	}
	n := tb.buf.LineCount()
	for i := 0; i < n && i < 5000; i++ {
		ln := tb.buf.Line(i)
		if word != "" && len(wordHits(ln, word)) > 0 {
			m.word_ = append(m.word_, i)
		}
		if q != "" {
			hay, nd := ln, q
			if !cs {
				hay, nd = strings.ToLower(ln), strings.ToLower(q)
			}
			if strings.Contains(hay, nd) {
				m.found = append(m.found, i)
			}
		}
	}
	tb.mm = m
	return m
}

// tabWord 是光标所在的单词(关键字与过短的词不做同词高亮)。
func (a *App) tabWord(tb *ideTab) string {
	if tb.st == nil || tb.buf == nil {
		return ""
	}
	s := tb.st.Selection()
	if s.Anchor != s.Head { // 有选区时高亮选区文字本身(单个单词)
		return ""
	}
	w, _, _ := wordUnder(tb, s.Head)
	if len(w) < 2 || isKeyword(w) {
		return ""
	}
	return w
}

func (a *App) mmWidth() float32 {
	switch a.prefs.Int(pMmWidth, 1) {
	case 0:
		return 48
	case 2:
		return 80
	default:
		return 64
	}
}

// minimap 是迷你地图 + 概览标尺。
func (a *App) minimap(c *ui.Context, tb *ideTab) {
	t := a.tk
	st := tb.st
	mmW := a.mmWidth()
	n := tb.buf.LineCount() // 不再截到 2000 行:只画框里看得见的那一段(见下面的绘制范围),几何见 ide_minimap_geom.go
	info := a.overlayInfo(tb)
	total := float32(st.LineTop(n))
	view := float32(st.ViewportHeight())
	// 滚动中:滑块与行范围显示,停止后 3 秒隐藏(偏好「滑块 → 常驻」则一直显示滑块)
	now := a.now()
	if y := st.ScrollY(); !tb.mmInit {
		tb.mmY, tb.mmInit = y, true
	} else if y != tb.mmY {
		tb.mmY, tb.mmAt = y, now
	}
	scrolling := !tb.mmAt.IsZero() && now.Sub(tb.mmAt) < mmThumbIdle
	if scrolling {
		a.redrawSoon() // 持续出帧到过期那一刻,滑块才会按时收起
	}
	box := ui.Box(c).Key("mm-"+tb.id).Width(mmW+mmRulW).FillHeight().Shrink(0).BorderWidth(0, 0, 0, 1).BorderColor(t.Bd1).Clip() // 必须裁剪:色块按整篇行数画,长文件会溢出盖住底部面板
	// 悬停行只做地图上的浅色高亮,不再弹源码摘要(没人看)
	pg := mmGeometry(st, n, box.Bounds().H) // 交互用上一帧量到的框高;绘制时用本帧的
	hoverLine := -1
	if box.Hovered() {
		if _, y, ok := box.PointerPosition(); ok {
			hoverLine = int((y - mmPad + pg.off) / mmRowH)
		}
	}
	cursorLine := st.CursorPos().Line
	rl := a.claudeLines(tb)

	if box.Pressed() || box.Dragging() {
		if x, y, ok := box.PointerPosition(); ok && total > 0 {
			h := box.Bounds().H
			switch {
			case x >= mmW && h > 0: // 概览标尺:整篇缩放
				st.SetScrollY(float64(y/h*total - view/2))
			case h > 0:
				yc := y - mmPad // 地图内容坐标(框内)
				if !tb.mmDrag { // 刚按下:抓在滑块上就保持抓住的那一点,点在空白处把滑块中心移到指针
					tb.mmDrag = true
					tb.mmGrab = pg.h / 2
					if yc >= pg.top && yc <= pg.top+pg.h {
						tb.mmGrab = yc - pg.top
					}
					if line := int((yc + pg.off) / mmRowH); line >= 0 && line < n {
						if q, _ := tb.findQuery(); q != "" {
							unfoldLineForNav(tb, line) // 有查找词时点地图要先展开该行
						}
					}
				}
				st.SetScrollY(float64(pg.scrollFor(yc-tb.mmGrab, total, view)))
			}
		}
	} else {
		tb.mmDrag = false
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		g := mmGeometry(st, n, r.H)
		y0 := r.Y + mmPad - g.off // 内容比框高时,地图内容随滚动比例上移
		first := max(int(g.off/mmRowH)-1, 0)
		last := min(int((g.off+g.visH)/mmRowH)+2, n) // 只画框里看得见的行
		showThumb := a.prefs.Int(pMmSlider, 0) == 1 || box.Hovered() || box.Dragging() || scrolling
		if showThumb && total > 0 && view > 0 {
			if t.Glass() {
				// 流光:视口滑块是一块清透玻璃(左右各缩 2 带圆角,像浮在地图上),底下的缩略色块仍可读
				kit.GlassPaintClear(p, ui.Rect{X: r.X + 2, Y: r.Y + mmPad + g.top, W: mmW - 4, H: g.h}, 4)
			} else {
				p.Fill(ui.Rect{X: r.X, Y: r.Y + mmPad + g.top, W: mmW, H: g.h}, t.Fill2, 0)
			}
		}
		if hoverLine >= 0 && hoverLine < n {
			p.Fill(ui.Rect{X: r.X, Y: y0 + float32(hoverLine)*mmRowH - .5, W: mmW, H: mmRowH + 1}, t.Info.Alpha(.25), 0)
		}
		if a.scopeOn() {
			snap := a.scopeSnapshot(tb)
			if snap.inner >= 0 && snap.inner < len(snap.syms) {
				s := snap.syms[snap.inner]
				end := s.End
				if end >= n {
					end = n - 1
				}
				if s.Line <= end && s.Line < n {
					yy := y0 + float32(s.Line)*mmRowH
					hh := float32(end-s.Line+1) * mmRowH
					p.Fill(ui.Rect{X: r.X, Y: yy, W: 2, H: hh}, t.Ac.Alpha(.7), 0)
				}
			}
		}
		// 行底色:Claude 改过的行(紫)、搜索命中(黄)、同词(蓝)
		band := func(lines []int, col ui.Color) {
			for _, l := range lines {
				if l < n {
					p.Fill(ui.Rect{X: r.X, Y: y0 + float32(l)*mmRowH - .5, W: mmW, H: mmRowH}, col, 0)
				}
			}
		}
		band(info.word_, t.Info.Alpha(.20))
		band(info.found, t.Warn.Alpha(.38))
		for _, g := range rl {
			for l := g[0]; l <= g[1] && l < n; l++ {
				p.Fill(ui.Rect{X: r.X, Y: y0 + float32(l)*mmRowH - .5, W: mmW, H: mmRowH}, t.Ac.Alpha(.20), 0)
			}
		}
		// 语法色块(只画框里看得见的行,避免长文件每帧白画几千行)
		for i := first; i < last; i++ {
			ln := strings.TrimRight(tb.buf.Line(i), " \t\r")
			if ln == "" {
				continue
			}
			x0 := 0
			for _, rn := range lineRuns(ln) {
				col := t.Tx3.Alpha(.5)
				switch rn.kind {
				case 'k':
					col = t.SynKw.Alpha(.75)
				case 's':
					col = t.SynStr.Alpha(.75)
				case 'c':
					col = t.SynCom.Alpha(.55)
				case 't':
					col = t.SynType.Alpha(.75)
				case 'f':
					col = t.SynFn.Alpha(.75)
				case 'p':
					col = t.Tx3.Alpha(.35)
				}
				x := 6 + float32(rn.a+x0)*mmChar
				w := float32(rn.b-rn.a) * mmChar
				if x >= 6+mmMaxPx {
					break
				}
				p.Fill(ui.Rect{X: r.X + x, Y: y0 + float32(i)*mmRowH, W: float32(math.Min(float64(w), float64(6+mmMaxPx-x))), H: 2}, col, 1)
			}
		}
		// 左缘 Git 竖条
		gitM := info.git
		if !a.prefs.Bool(pMmGit, true) {
			gitM = nil
		}
		for l, k := range gitM {
			if l >= n {
				continue
			}
			col := t.Ok
			if k == '~' {
				col = t.Info
			} else if k == '-' {
				col = t.Err
			}
			p.Fill(ui.Rect{X: r.X, Y: y0 + float32(l)*mmRowH - .5, W: 2, H: mmRowH}, col, 0)
		}
		// 概览标尺:整篇缩放
		all := float32(tb.buf.LineCount())
		if all < 1 {
			all = 1
		}
		tick := func(l int, col ui.Color, left, w float32) {
			y := r.Y + float32(l)/all*r.H
			p.Fill(ui.Rect{X: r.X + mmW + left, Y: y, W: w, H: 2}, col, 0)
		}
		for l := range gitM {
			k := info.git[l]
			col := t.Ok
			if k == '~' {
				col = t.Info
			} else if k == '-' {
				col = t.Err
			}
			tick(l, col, 0, 2)
		}
		for _, l := range info.found {
			tick(l, t.Warn, 3, 3)
		}
		for _, g := range rl {
			tick(g[0], t.Ac, 2, 4)
		}
		for _, d := range st.Diagnostics() {
			col := t.Warn
			if d.Severity == 0 {
				col = t.Err
			}
			tick(tb.buf.Pos(d.Start).Line, col, 2, 4)
		}
		tick(cursorLine, t.Tx2, 0, mmRulW)
	})
}
