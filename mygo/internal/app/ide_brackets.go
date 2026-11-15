package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// 括号配对着色:库自带匹配高亮;这里按深度给括号字符染色(非 tree-sitter 语义)。
var bracketCols = []ui.Color{
	ui.RGBA(230, 180, 60, 255),
	ui.RGBA(180, 120, 220, 255),
	ui.RGBA(90, 150, 230, 255),
	ui.RGBA(60, 190, 200, 255),
}
var bracketColsDark = []ui.Color{
	ui.RGBA(255, 210, 80, 255),
	ui.RGBA(200, 150, 255, 255),
	ui.RGBA(120, 180, 255, 255),
	ui.RGBA(100, 220, 220, 255),
}

func (a *App) bracketDecorate(tb *ideTab) []editor.TextBackground {
	if tb == nil || tb.st == nil || tb.buf == nil || !a.prefs.Bool(pBracketColor, true) {
		return nil
	}
	src := tb.buf.String()
	cols := bracketCols
	if a.pal.Dark {
		cols = bracketColsDark
	}
	var tbs []editor.TextBackground
	depth := 0
	pairs := map[byte]byte{'(': ')', '[': ']', '{': '}'}
	closers := map[byte]bool{')': true, ']': true, '}': true}
	inStr, inCom := false, false
	for i := 0; i < len(src); i++ {
		ch := src[i]
		if inCom {
			if ch == '\n' {
				inCom = false
			}
			continue
		}
		if inStr {
			if ch == '"' && (i == 0 || src[i-1] != '\\') {
				inStr = false
			}
			continue
		}
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '/' {
			inCom = true
			continue
		}
		if ch == '"' {
			inStr = true
			continue
		}
		if pairs[ch] != 0 {
			col := cols[depth%len(cols)].Alpha(.85)
			tbs = append(tbs, editor.TextBackground{Start: i, End: i + 1, Color: col})
			depth++
		} else if closers[ch] {
			depth = max(depth-1, 0)
			col := cols[depth%len(cols)].Alpha(.85)
			tbs = append(tbs, editor.TextBackground{Start: i, End: i + 1, Color: col})
		}
		if len(tbs) > 800 {
			break
		}
	}
	return tbs
}
