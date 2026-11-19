package app

import (
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// 括号配对着色:与 VS Code「Bracket Pair Colorization」/ Zed 的 colorize_brackets 一致——
// 括号字符本身按嵌套深度换前景色(像字体颜色),不加背景。
//
// 做法:库的 Token 只有「种类」,颜色由主题按种类决定。所以
//  1. 给代码类语言包一层高亮器(Language.Highlighter),把括号字符的 Token 拆出来,
//     按嵌套深度改成一组「括号深度种类」;
//  2. 这几个种类取自代码类语言的分词器从不产出的种类(Markdown / Diff 才会用,
//     这两种语言不包装),再在「每个标签自己的主题」里给它们配括号色。
//
// 字符串、注释、正则里的括号因为本来就不是标点 Token,自然被跳过(比旧的逐字节扫描准)。

// bracketKinds 是括号深度依次使用的 Token 种类(循环使用)。
var bracketKinds = [...]editor.TokenKind{
	editor.TokenInvalid, editor.TokenInserted, editor.TokenDeleted, editor.TokenLink,
}

// 括号色:浅色主题要压深一点,否则金色在白底上发虚。
var bracketCols = [len(bracketKinds)]ui.Color{
	ui.RGBA(184, 128, 0, 255),
	ui.RGBA(158, 76, 198, 255),
	ui.RGBA(30, 110, 214, 255),
	ui.RGBA(0, 148, 150, 255),
}
var bracketColsDark = [len(bracketKinds)]ui.Color{
	ui.RGBA(255, 210, 80, 255),
	ui.RGBA(206, 150, 255, 255),
	ui.RGBA(120, 180, 255, 255),
	ui.RGBA(100, 220, 220, 255),
}

// bracketMaxLines 超过这个行数的文件不做括号着色(第一次绘制要从头累计嵌套深度)。
const bracketMaxLines = 30000

// bracketSkipLang 不包装的语言:标记语言的尖括号不算括号,Markdown / Diff 自己用了借来的种类。
var bracketSkipLang = map[string]bool{
	"Markdown": true, "Diff": true, "CSV": true, "HTML": true, "XML": true, "ERB": true, "Svelte": true, "Jinja": true,
}

// bracketLangs 缓存包装后的语言(每种基础语言一个,换标签 / 重开文件复用同一个指针)。
var bracketLangs struct {
	mu   sync.Mutex
	wrap map[*editor.Language]*editor.Language
	base map[*editor.Language]*editor.Language
}

// wrapBracketLang 返回 l 的「括号着色版」;不适用(纯文本 / 标记语言)时原样返回。
func wrapBracketLang(l *editor.Language) *editor.Language {
	if l == nil || l.Tokenizer == nil || l.Highlighter != nil || bracketSkipLang[l.Name] {
		return l
	}
	m := &bracketLangs
	m.mu.Lock()
	defer m.mu.Unlock()
	if w, ok := m.wrap[l]; ok {
		return w
	}
	if m.wrap == nil {
		m.wrap, m.base = map[*editor.Language]*editor.Language{}, map[*editor.Language]*editor.Language{}
	}
	w := *l
	tk := l.Tokenizer
	w.Highlighter = func(text editor.Snapshot, _ func()) editor.Highlighter {
		return &bracketHL{base: editor.NewTokenHighlighter(tk, text), text: text}
	}
	m.wrap[l], m.base[&w] = &w, l
	return &w
}

// unwrapBracketLang 是 wrapBracketLang 的逆:包装版换回基础语言,其余原样返回。
func unwrapBracketLang(l *editor.Language) *editor.Language {
	m := &bracketLangs
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.base[l]; ok {
		return b
	}
	return l
}

// isBracketLang 报告 l 是不是括号着色版。
func isBracketLang(l *editor.Language) bool {
	m := &bracketLangs
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.base[l]
	return ok
}

// syncBracketLang 让标签缓冲区的语言与偏好一致(开关随时生效),并给标签配上对应的主题。
// 每帧调用;语言没变时只是几次指针比较。
func (a *App) syncBracketLang(tb *ideTab) {
	if tb == nil || tb.buf == nil || tb.st == nil {
		return
	}
	cur := tb.buf.Language()
	base := unwrapBracketLang(cur)
	want := base
	if a.prefs.Bool(pBracketColor, true) && tb.buf.LineCount() <= bracketMaxLines {
		want = wrapBracketLang(base)
	}
	if want != cur {
		tb.buf.SetLanguage(want)
	}
	if isBracketLang(want) {
		tb.st.Theme = a.bracketTheme()
	} else if isCSVLang(want) {
		tb.st.Theme = a.csvTheme()
	}
}

// bracketTheme 是在编辑器配色基础上多出「括号深度色」的主题(随主题缓存)。
func (a *App) bracketTheme() *editor.Theme {
	if a.brTheme != nil && a.brThemeKey == a.pal.ID {
		return a.brTheme
	}
	th := *a.editorTheme() // 值拷贝:不改动共享的基础配色
	cols := bracketCols
	if a.pal.Dark {
		cols = bracketColsDark
	}
	for i, k := range bracketKinds {
		th.Tokens[k] = editor.TokenStyle{Color: cols[i]}
	}
	a.brTheme, a.brThemeKey = &th, a.pal.ID
	return a.brTheme
}

// bracketHL 包着库自带的分词高亮器:Token 原样转发,只把括号字符按深度改种类。
type bracketHL struct {
	base editor.Highlighter
	text editor.Snapshot
	// depth[i] = 第 i 行开头的嵌套深度;未定稿行也缓存(增量),避免 scanBracket 每行重复 O(n) 累计。
	depth      []int
	depthFinal int // 行 [0,depthFinal) 在写入 depth 时基础高亮已 final;其后再确认会截断并重算
	scratch    []editor.Token
	out        []editor.Token
}

func (h *bracketHL) Edit(old, new editor.Snapshot, changes []editor.Change) {
	h.base.Edit(old, new, changes)
	h.text = new
	if len(changes) == 0 {
		return
	}
	// 第一处修改所在行之前的行不受影响;该行开头的深度也没变,之后的作废。
	keep := old.LineOf(changes[0].OldStart) + 1
	if keep < len(h.depth) {
		h.depth = h.depth[:keep]
	}
	if keep < h.depthFinal {
		h.depthFinal = keep
	}
}

func (h *bracketHL) Highlight(last int, budget time.Duration) bool {
	done := h.base.Highlight(last, budget)
	h.syncDepthFinal()
	return done
}

// syncDepthFinal 随基础高亮推进,把已 final 的前缀记下来;超出前缀的临时深度丢弃,下回按新 token 重算。
func (h *bracketHL) syncDepthFinal() {
	n := h.text.LineCount()
	for h.depthFinal < n {
		line := h.text.Line(h.depthFinal)
		_, fin := h.base.Tokens(h.depthFinal, line, h.scratch[:0])
		h.scratch = h.scratch[:0]
		if !fin {
			break
		}
		h.depthFinal++
	}
	keep := h.depthFinal + 1
	if len(h.depth) > keep {
		h.depth = h.depth[:keep]
	}
}

func (h *bracketHL) Tokens(i int, text string, dst []editor.Token) ([]editor.Token, bool) {
	// 先算深度(会借用 scratch 去取前面各行的 Token),再取本行。
	d, ok := h.depthAt(i)
	toks, final := h.base.Tokens(i, text, h.scratch[:0])
	h.scratch = toks[:0]
	if !ok {
		return append(dst, toks...), final
	}
	dst, _ = splitBrackets(text, toks, d, dst, true)
	return dst, final
}

// depthAt 返回第 i 行开头的嵌套深度:缓存里没有就从已知的最后一行累计过来(只增不减,摊还 O(1))。
func (h *bracketHL) depthAt(i int) (int, bool) {
	if i > bracketMaxLines || i >= h.text.LineCount() {
		return 0, false
	}
	if i < len(h.depth) {
		return h.depth[i], true
	}
	if len(h.depth) == 0 {
		h.depth = append(h.depth, 0)
	}
	d := h.depth[len(h.depth)-1]
	for j := len(h.depth) - 1; j < i; j++ {
		line := h.text.Line(j)
		toks, _ := h.base.Tokens(j, line, h.scratch[:0])
		h.scratch = toks[:0]
		_, d = splitBrackets(line, toks, d, nil, false)
		h.depth = append(h.depth, d)
	}
	return d, true
}

// splitBrackets 把一行的 Token 里的括号字符拆成单字符 Token,种类按嵌套深度循环取;
// 返回拆分后的 Token 与行尾深度。emit 为 false 时只算深度。
func splitBrackets(line string, toks []editor.Token, depth int, dst []editor.Token, emit bool) ([]editor.Token, int) {
	start := 0
	for _, tk := range toks {
		end := min(tk.End, len(line))
		switch tk.Kind {
		case editor.TokenPunctuation, editor.TokenText, editor.TokenOperator:
		default:
			if emit {
				dst = append(dst, tk)
			}
			start = tk.End
			continue
		}
		seg := start // 本 Token 里还没输出的起点
		for k := start; k < end; k++ {
			var kind editor.TokenKind
			switch line[k] {
			case '(', '[', '{':
				kind = bracketKinds[depth%len(bracketKinds)]
				depth++
			case ')', ']', '}':
				depth = max(depth-1, 0)
				kind = bracketKinds[depth%len(bracketKinds)]
			default:
				continue
			}
			if emit {
				if k > seg {
					dst = append(dst, editor.Token{End: k, Kind: tk.Kind})
				}
				dst = append(dst, editor.Token{End: k + 1, Kind: kind})
			}
			seg = k + 1
		}
		if emit && seg < tk.End {
			dst = append(dst, editor.Token{End: tk.End, Kind: tk.Kind})
		}
		start = tk.End
	}
	return dst, depth
}
