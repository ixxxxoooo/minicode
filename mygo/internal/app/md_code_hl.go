package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// Markdown 围栏代码块的语法着色(设计稿 flow.css .md pre):
// 用编辑器同一套词法器与主题语法色(syn-*),所以会话流、只读预览里的代码和编辑器里的颜色一致。
//
// @author ixxxxoooo

const mdCodeHlMax = 300 // 缓存的代码块上限,超过就整体清空重来

// mdCodeLang 返回围栏信息串对应的语言;没写语言或不认识返回 nil(调用方按纯文本画)。
// 参数 info 是围栏后的整段信息串,如 "go" 或 "rust title=a.rs";取第一个词。
func mdCodeLang(info string) *editor.Language {
	f := strings.Fields(info)
	if len(f) == 0 {
		return nil
	}
	l := editor.LanguageNamed(strings.TrimPrefix(strings.Split(f[0], "{")[0], "."))
	if l == nil || l.Tokenizer == nil {
		return nil
	}
	return l
}

// mdCodeLines 把一段围栏代码按语言着色,返回逐行 span(行数与 strings.Split(代码, "\n") 一致)。
// 参数 lang 是围栏语言,code 是代码正文,size 是等宽字号,live 为真表示块还在流式写入(不进缓存)。
// 返回值:语言不认识时为 nil。词法状态跨行延续,所以多行注释 / 多行字符串能正确着色。
func (a *App) mdCodeLines(lang *editor.Language, code string, size float32, live bool) [][]ui.Span {
	if lang == nil {
		return nil
	}
	key := fmt.Sprintf("%s|%v|%v", a.pal.ID, a.tk.Ac, size)
	if a.mdCodeKey != key || len(a.mdCodeHl) > mdCodeHlMax {
		a.mdCodeKey, a.mdCodeHl = key, map[string][][]ui.Span{}
	}
	ck := lang.Name + "\x00" + code
	if !live {
		if hit, ok := a.mdCodeHl[ck]; ok {
			return hit
		}
	}
	th := a.editorTheme()
	lines := strings.Split(strings.TrimRight(code, "\n"), "\n")
	out := make([][]ui.Span, len(lines))
	state := uint64(0)
	for i, ln := range lines {
		if ln == "" {
			out[i] = []ui.Span{{Text: " ", Font: theme.CodeFont(), Size: size, Color: th.Text}}
			continue
		}
		if len(ln) > 2000 { // 压缩过的超长行不跑词法,免得每帧白耗时
			out[i] = []ui.Span{{Text: ln, Font: theme.CodeFont(), Size: size, Color: th.Text}}
			continue
		}
		var toks []editor.Token
		toks, state = lang.Tokenizer.Tokenize(ln, state, nil)
		out[i] = tokensToSpans(ln, toks, th, size)
	}
	if !live {
		a.mdCodeHl[ck] = out
	}
	return out
}
