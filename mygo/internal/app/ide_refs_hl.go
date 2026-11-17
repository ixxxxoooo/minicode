package app

import (
	"fmt"
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// refLineSpans 给引用面板一行源码着色,并在 [hitLo,hitHi) 叠命中底色。
func (a *App) refLineSpans(filePath, text string, hitLo, hitHi int) []ui.Span {
	const size = 11.5
	if text == "" {
		return []ui.Span{{Text: " ", Font: "monospace", Size: size}}
	}
	th := a.editorTheme()
	key := fmt.Sprintf("%s|%v|%v|%v", a.pal.ID, a.tk.Ac, size, path.Ext(filePath))
	if a.refHlKey != key || len(a.refHlCache) > 4000 {
		a.refHlKey, a.refHlCache = key, map[string][]ui.Span{}
	}
	ck := filePath + "\x00" + text
	if sp, ok := a.refHlCache[ck]; ok {
		return applyRefHit(sp, text, hitLo, hitHi, size, a.tk.Ac)
	}
	lang := editor.LanguageFor(path.Base(filePath))
	sp := tokenLineSpans(lang, text, th, size)
	a.refHlCache[ck] = sp
	return applyRefHit(sp, text, hitLo, hitHi, size, a.tk.Ac)
}

func tokenLineSpans(lang *editor.Language, text string, th *editor.Theme, size float32) []ui.Span {
	if lang == nil || lang.Tokenizer == nil {
		return []ui.Span{{Text: text, Font: "monospace", Size: size, Color: th.Text}}
	}
	buf := editor.NewBuffer(text)
	h := editor.NewTokenHighlighter(lang.Tokenizer, buf.Snapshot())
	toks, _ := h.Tokens(0, text, nil)
	return tokensToSpans(text, toks, th, size)
}

// tokensToSpans 把一行文本的词法结果按编辑器配色转成等宽 span(引用面板与 Markdown 围栏代码块共用)。
// 参数 text 是该行文本,toks 是词法器给出的该行词(End 为字节偏移),th 取色,size 是字号。
// 返回值:覆盖整行的 span;没有词法结果时整行取正文色。
func tokensToSpans(text string, toks []editor.Token, th *editor.Theme, size float32) []ui.Span {
	if len(toks) == 0 {
		return []ui.Span{{Text: text, Font: "monospace", Size: size, Color: th.Text}}
	}
	var out []ui.Span
	at := 0
	for _, tk := range toks {
		end := tk.End
		if end > len(text) {
			end = len(text)
		}
		if at >= end {
			continue
		}
		if at < end {
			st := th.Tokens[tk.Kind]
			col := st.Color
			if col.A == 0 {
				col = th.Text
			}
			w := 0
			if st.Bold {
				w = 600
			}
			out = append(out, ui.Span{Text: text[at:end], Font: "monospace", Size: size, Color: col, Italic: st.Italic, Weight: w})
		}
		at = end
	}
	if at < len(text) {
		out = append(out, ui.Span{Text: text[at:], Font: "monospace", Size: size, Color: th.Text})
	}
	return out
}

// applyRefHit 在已着色的 span 上叠加命中词底色(只改 hit 区间内的片段)。
func applyRefHit(spans []ui.Span, text string, hitLo, hitHi int, size float32, accent ui.Color) []ui.Span {
	if hitLo < 0 || hitHi <= hitLo || hitHi > len(text) {
		return spans
	}
	if len(spans) == 0 {
		return []ui.Span{{
			Text: text, Font: "monospace", Size: size, Color: accent,
			Background: accent.Alpha(.22),
		}}
	}
	bg := accent.Alpha(.22)
	var out []ui.Span
	pos := 0
	for _, sp := range spans {
		part := sp.Text
		if part == "" {
			continue
		}
		start, end := pos, pos+len(part)
		pos = end
		if end <= hitLo || start >= hitHi {
			out = append(out, sp)
			continue
		}
		lo := max(hitLo, start) - start
		hi := min(hitHi, end) - start
		if lo > 0 {
			out = append(out, spWithText(sp, part[:lo]))
		}
		mid := part[lo:hi]
		hit := spWithText(sp, mid)
		hit.Background = bg
		if strings.TrimSpace(mid) != "" {
			hit.Color = accent
			hit.Weight = 600
		}
		out = append(out, hit)
		if hi < len(part) {
			out = append(out, spWithText(sp, part[hi:]))
		}
	}
	return out
}

func spWithText(sp ui.Span, text string) ui.Span {
	sp.Text = text
	return sp
}
