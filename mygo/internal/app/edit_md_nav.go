package app

import (
	"strings"

	"github.com/egoist/mygo/ui/richtext"
	"github.com/ixxxxoooo/minicode/internal/symbols"
)

// mdScrollToLine 在所见即所得里滚到源码第 line(0 起)附近:按标题序号或行比例估算段落。
func (a *App) mdScrollToLine(tb *ideTab, md *mdState, line int) {
	if md == nil || md.doc == nil || line < 0 {
		return
	}
	_, body := mdSplitFront(tb.buf.String())
	off := mdHeadingOffset(body, line, md.doc.Text())
	if off < 0 {
		n := len(strings.Split(body, "\n"))
		if n <= 1 {
			off = 0
		} else {
			text := md.doc.Text()
			ratio := float64(line) / float64(n-1)
			off = int(float64(text.Len()) * ratio)
			if off >= text.Len() {
				off = max(0, text.Len()-1)
			}
		}
	}
	md.doc.SetSelection(off, off)
	uxLog.Debug("Markdown 大纲定位", "path", tb.path, "line", line+1, "off", off)
}

// mdHeadingOffset 在文档里找与源码第 line 行对应的标题段落起始(rune 偏移);失败 -1。
func mdHeadingOffset(body string, line int, doc richtext.Text) int {
	if line < 0 {
		return -1
	}
	lines := strings.Split(body, "\n")
	if line >= len(lines) {
		return -1
	}
	ln := lines[line]
	if m := symbols.MarkdownHeading.FindStringSubmatch(ln); m != nil {
		wantLvl := len(m[1])
		want := strings.TrimSpace(m[2])
		paras := doc.ParagraphCount()
		for p := 0; p < paras; p++ {
			off := paraStart(doc, p)
			st := doc.ParagraphStyleAt(off)
			if st.Heading == wantLvl {
				_, e := doc.ParagraphRange(off) // off = 该段落起始 rune
				if strings.Contains(doc.Slice(off, e).String(), want) {
					return off
				}
			}
		}
	}
	hix := -1
	for i, ln := range lines {
		if i > line {
			break
		}
		if m := symbols.MarkdownHeading.FindStringSubmatch(ln); m != nil {
			hix++
		}
	}
	if hix < 0 {
		return -1
	}
	k := 0
	paras := doc.ParagraphCount()
	for p := 0; p < paras; p++ {
		st := doc.ParagraphStyleAt(paraStart(doc, p))
		if st.Heading > 0 {
			if k == hix {
				return paraStart(doc, p)
			}
			k++
		}
	}
	return -1
}

func paraStart(doc richtext.Text, p int) int {
	n := 0
	for start := range doc.Paragraphs() {
		if n == p {
			return start
		}
		n++
	}
	return 0
}

// mdApplyReveal 处理标签上的 reveal(大纲 / 跳转):WYSIWYG 内滚动,不切源码。
func (a *App) mdApplyReveal(tb *ideTab, md *mdState) {
	if tb.reveal == nil || md == nil {
		return
	}
	line := tb.reveal[0]
	tb.reveal = nil
	a.mdScrollToLine(tb, md, line)
}
