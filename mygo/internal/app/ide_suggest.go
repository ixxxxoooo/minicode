package app

import (
	"strings"
)

// 行内补全(✦):库用 SetSuggestion 画幽灵文字,Tab 接受。
type suggestSt struct {
	at     int
	prefix string
	text   string
	busy   bool
	tab    string
}

// suggestTick 在停顿后请求占位补全(演示)或 claude -p(真实,未全量接入)。
func (a *App) suggestTick(tb *ideTab) {
	if tb == nil || tb.st == nil || tb.buf == nil || tb.diff || tb.ReadOnly() {
		return
	}
	if !a.prefs.Bool("editor.inlineSuggest", true) {
		tb.st.SetSuggestion(0, "")
		return
	}
	cur := tb.st.Cursor()
	sel := tb.st.Selection()
	if sel.Anchor != sel.Head {
		return
	}
	line := tb.buf.Line(tb.buf.Pos(cur).Line)
	if strings.TrimSpace(line) == "" {
		return
	}
	w, _, _ := wordUnder(tb, cur)
	if len(w) < 2 {
		return
	}
	if a.sg != nil && a.sg.busy && a.sg.tab == tb.id {
		return
	}
	ghost := placeholderSuggest(w, line)
	if ghost == "" {
		return
	}
	tb.st.SetSuggestion(cur, ghost)
}

func placeholderSuggest(word, line string) string {
	switch {
	case strings.Contains(line, "normalize"):
		return "d"
	case strings.HasSuffix(word, "Path"):
		return "Buf"
	case len(word) > 3:
		return strings.ToLower(word[:1]) + word[1:]
	}
	return ""
}

func (tb *ideTab) ReadOnly() bool {
	return tb.st != nil && tb.st.ReadOnly
}
