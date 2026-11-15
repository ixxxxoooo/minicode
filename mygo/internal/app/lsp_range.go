package app

import (
	"github.com/ixxxxoooo/minicode/internal/lsp"
)

// lspRange 把字节选区换成 LSP Range(End 为 exclusive,行尾按 UTF-16 列对齐)。
func lspRange(tb *ideTab, start, end int) lsp.Range {
	return lsp.Range{Start: lspPos(tb, start), End: lspPosEnd(tb, end)}
}

// lspPosEnd 把字节偏移换成 LSP 的 exclusive End(落在换行符上时收到上一行行尾 UTF-16 列)。
func lspPosEnd(tb *ideTab, off int) lsp.Position {
	if tb == nil || tb.buf == nil {
		return lsp.Position{}
	}
	n := tb.buf.Len()
	if off < 0 {
		off = 0
	}
	if off > n {
		off = n
	}
	if off > 0 && off <= n {
		p := tb.buf.Pos(off)
		if p.Col == 0 && off > 0 {
			prev := tb.buf.Pos(off - 1)
			ln := tb.buf.Line(prev.Line)
			if len(ln) > 0 && ln[len(ln)-1] == '\n' {
				return lsp.Position{
					Line:      prev.Line,
					Character: lsp.ColToUTF16(ln[:len(ln)-1], len(ln)-1),
				}
			}
		}
	}
	return lspPos(tb, off)
}
