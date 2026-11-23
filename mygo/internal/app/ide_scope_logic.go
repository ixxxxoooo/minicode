package app

import "github.com/ixxxxoooo/minicode/internal/symbols"

// 所在范围(光标 / 悬停):用符号表的行范围找最内层块,按缓冲区版本缓存,光标行变化才重算链。

// scopeSnap 是一次查询的快照(syms 切片只读引用缓存,勿修改)。
type scopeSnap struct {
	syms   []symbols.Symbol
	chain  []int  // 从外到内的符号下标
	inner  int    // 最内层下标,-1 = 无
	hover  int    // 悬停预览的最内层,-1 = 无
	source string // "lsp" | "local" | ""
}

func innermostSymbol(syms []symbols.Symbol, line int) int {
	ch := symbols.Enclosing(syms, line)
	if len(ch) == 0 {
		return -1
	}
	return ch[len(ch)-1]
}

// symbolAtLine 返回包含 line 的最内层符号下标(与 innermostSymbol 相同,别名便于测试)。
func symbolAtLine(syms []symbols.Symbol, line int) int { return innermostSymbol(syms, line) }

// parentChain 返回从外到内、包含 line 的整条符号链下标。
func parentChain(syms []symbols.Symbol, line int) []int {
	ix := innermostSymbol(syms, line)
	if ix < 0 {
		return nil
	}
	var chain []int
	for k := ix; k >= 0; k = syms[k].Parent {
		chain = append([]int{k}, chain...)
	}
	return chain
}

// scopeDepth 返回 symIx 在 chain 里的层级(0 = 最内层)。
func scopeDepth(chain []int, symIx int) int {
	for i, k := range chain {
		if k == symIx {
			return len(chain) - 1 - i
		}
	}
	return -1
}

// scopeIndentLevel 估算符号块主体的缩进列(用于对齐缩进参考线预览)。
func scopeIndentLevel(syms []symbols.Symbol, symIx int, lineText func(int) string) int {
	if symIx < 0 || symIx >= len(syms) {
		return 0
	}
	ln := syms[symIx].Line
	if lineText == nil {
		return 0
	}
	return symbols.IndentOf(lineText(ln))
}
