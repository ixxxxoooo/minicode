package app

import (
	"strings"

	"github.com/egoist/mygo/ui/richtext"
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
	hix := -1
	for i, ln := range strings.Split(body, "\n") {
		if i > line {
			break
		}
		if m := reMd.FindStringSubmatch(ln); m != nil {
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
	for i := 0; i < p; i++ {
		_, e := doc.ParagraphRange(n)
		n = e
	}
	s, _ := doc.ParagraphRange(n)
	return s
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
