package app

import (
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/claude"
)

// 行内补全(✦):库用 SetSuggestion 画幽灵文字,Tab 接受。
type suggestSt struct {
	at      int
	prefix  string
	text    string
	busy    bool
	tab     string
	lastKey string // path+word 去重
}

// suggestTick 在停顿后请求 Haiku 补全或演示占位。
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
	pos := tb.buf.Pos(cur)
	line := tb.buf.Line(pos.Line)
	if strings.TrimSpace(line) == "" {
		return
	}
	// 幽灵文字只在光标后面没有内容时出现(行尾),否则会盖住光标右侧的已有文字
	if col := pos.Col; col >= 0 && col < len(line) && strings.TrimSpace(line[col:]) != "" {
		tb.st.SetSuggestion(0, "")
		return
	}
	w, _, wEnd := wordUnder(tb, cur)
	if len(w) < 2 {
		return
	}
	// 光标在单词中间时不给建议:幽灵文字会插进词里(Handl|handler|er),只在词尾 / 行尾补全
	if cur < wEnd {
		tb.st.SetSuggestion(0, "")
		return
	}
	key := tb.id + "|" + w + "|" + line
	if a.sg != nil && a.sg.busy && a.sg.tab == tb.id {
		return
	}
	if a.sg == nil {
		a.sg = &suggestSt{}
	}
	if a.sg.busy && a.sg.lastKey == key {
		return
	}
	if a.sg.lastKey == key && a.sg.text != "" {
		tb.st.SetSuggestion(cur, a.sg.text)
		return
	}
	ghost := placeholderSuggest(w, line)
	if a.proj != "" {
		a.sg.busy, a.sg.tab, a.sg.at, a.sg.prefix, a.sg.lastKey = true, tb.id, cur, w, key
		ctx := a.beginClaudeJob()
		path, ln, word := tb.path, line, w
		a.bg("claude.suggest", func() func() {
			defer a.endClaudeJob()
			out, err := claude.RunPromptModelCtx(ctx, a.proj, suggestPrompt(path, ln, word), "plan", haikuInline, 45*time.Second)
			if err != nil {
				claudeEditLog.Warn("行内补全失败", "path", path, "err", err)
				out = placeholderSuggest(word, ln)
			} else {
				out = strings.TrimSpace(out)
				if strings.HasPrefix(out, word) {
					out = strings.TrimPrefix(out, word)
				}
			}
			if out == "" {
				out = placeholderSuggest(word, ln)
			}
			sgText, sgAt := out, cur
			return func() {
				if a.sg == nil || a.sg.tab != tb.id {
					return
				}
				a.sg.busy = false
				a.sg.text, a.sg.at = sgText, sgAt
				if t := a.ide.tab(tb.id); t != nil && t.st != nil {
					t.st.SetSuggestion(sgAt, sgText)
				}
			}
		})
		return
	}
	if ghost == "" {
		return
	}
	a.sg.lastKey, a.sg.text, a.sg.at = key, ghost, cur
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
