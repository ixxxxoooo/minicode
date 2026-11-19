package app

import (
	"path"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// csv / tsv 源码的「彩虹列」:同一列的字用同一种颜色,相邻列颜色不同,一眼对齐列。
// 做法与括号着色同源:自带一个 Tokenizer 按分隔符切列,循环使用 8 种 Token 种类,
// 再用 csvTheme 把这 8 种种类的颜色换成彩虹色(不改共享的基础配色)。

// csvKinds 是列依次使用的 Token 种类(循环使用);颜色由 csvTheme 指定。
var csvKinds = [...]editor.TokenKind{
	editor.TokenInvalid, editor.TokenInserted, editor.TokenDeleted, editor.TokenLink,
	editor.TokenKeyword, editor.TokenNumber, editor.TokenType, editor.TokenConstant,
}

var csvCols = [len(csvKinds)]ui.Color{
	ui.RGBA(196, 72, 52, 255),  // 红
	ui.RGBA(184, 128, 0, 255),  // 琥珀
	ui.RGBA(40, 140, 78, 255),  // 绿
	ui.RGBA(0, 144, 150, 255),  // 青
	ui.RGBA(30, 110, 214, 255), // 蓝
	ui.RGBA(90, 82, 200, 255),  // 靛
	ui.RGBA(158, 76, 198, 255), // 紫
	ui.RGBA(200, 60, 140, 255), // 粉
}

var csvColsDark = [len(csvKinds)]ui.Color{
	ui.RGBA(255, 128, 108, 255),
	ui.RGBA(255, 210, 80, 255),
	ui.RGBA(110, 214, 140, 255),
	ui.RGBA(100, 220, 220, 255),
	ui.RGBA(120, 180, 255, 255),
	ui.RGBA(160, 150, 255, 255),
	ui.RGBA(206, 150, 255, 255),
	ui.RGBA(255, 130, 190, 255),
}

// csvTokenizer 按分隔符把每一行切成列:引号里的分隔符与换行不算分列(跨行的带引号字段用状态带到下一行)。
// 状态 = 0 正常;否则 1 | 列号<<1,表示这一行从引号字段中间开始。
type csvTokenizer struct{ sep byte }

func (t csvTokenizer) Tokenize(line string, state uint64, tokens []editor.Token) ([]editor.Token, uint64) {
	inq := state&1 == 1
	col := int(state >> 1)
	atStart := !inq // 字段开头才算引号开始(5" 这种中间的引号不是)
	start := 0
	emit := func(end int, k editor.TokenKind) {
		if end > start {
			tokens = append(tokens, editor.Token{End: end, Kind: k})
			start = end
		}
	}
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if inq {
			if ch == '"' {
				if i+1 < len(line) && line[i+1] == '"' {
					i++
				} else {
					inq = false
				}
			}
			continue
		}
		switch {
		case ch == '"' && atStart:
			inq, atStart = true, false
		case ch == t.sep:
			emit(i, csvKinds[col%len(csvKinds)])
			emit(i+1, editor.TokenPunctuation)
			col++
			atStart = true
		default:
			atStart = false
		}
	}
	emit(len(line), csvKinds[col%len(csvKinds)])
	if inq {
		return tokens, 1 | uint64(col)<<1
	}
	return tokens, 0
}

var csvLangs struct {
	mu sync.Mutex
	m  map[byte]*editor.Language
}

// csvLang 返回以 sep 分列的彩虹语言(每种分隔符一个,重开文件复用同一个指针)。
func csvLang(sep byte) *editor.Language {
	c := &csvLangs
	c.mu.Lock()
	defer c.mu.Unlock()
	if l := c.m[sep]; l != nil {
		return l
	}
	if c.m == nil {
		c.m = map[byte]*editor.Language{}
	}
	l := &editor.Language{Name: "CSV", Tokenizer: csvTokenizer{sep: sep}}
	c.m[sep] = l
	return l
}

func isCSVLang(l *editor.Language) bool {
	if l == nil {
		return false
	}
	_, ok := l.Tokenizer.(csvTokenizer)
	return ok
}

// csvSep 按扩展名与开头几行猜分隔符:tsv 是制表符;csv 在逗号 / 分号 / 制表符 / 竖线里取出现最多的。
func csvSep(p, text string) byte {
	if strings.EqualFold(path.Ext(p), ".tsv") {
		return '\t'
	}
	head := text
	if len(head) > 8192 {
		head = head[:8192]
	}
	lines := strings.SplitN(head, "\n", 6)
	best, bestN := byte(','), 0
	for _, c := range []byte{',', ';', '\t', '|'} {
		n := 0
		for _, ln := range lines {
			n += strings.Count(ln, string(c))
		}
		if n > bestN {
			best, bestN = c, n
		}
	}
	return best
}

// langForFile 是打开文件时用的语言:csv / tsv 用彩虹列,其余按文件名认。
func langForFile(p, text string) *editor.Language {
	switch strings.ToLower(path.Ext(p)) {
	case ".csv", ".tsv":
		return csvLang(csvSep(p, text))
	}
	return editor.LanguageFor(path.Base(p))
}

// csvTheme 是在编辑器配色基础上把列色换成彩虹色的主题(随主题缓存)。
func (a *App) csvTheme() *editor.Theme {
	if a.csvTh != nil && a.csvThKey == a.pal.ID {
		return a.csvTh
	}
	th := *a.editorTheme()
	cols := csvCols
	if a.pal.Dark {
		cols = csvColsDark
	}
	for i, k := range csvKinds {
		th.Tokens[k] = editor.TokenStyle{Color: cols[i]}
	}
	a.csvTh, a.csvThKey = &th, a.pal.ID
	return a.csvTh
}
