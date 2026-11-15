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
		if p.Col == 0 && p.Line > 0 {
			prevLine := p.Line - 1
			ln := tb.buf.Line(prevLine)
			trim := ln
			if len(trim) > 0 && trim[len(trim)-1] == '\n' {
				trim = trim[:len(trim)-1]
			}
			return lsp.Position{
				Line:      prevLine,
				Character: lsp.ColToUTF16(trim, len(trim)),
			}
		}
	}
	return lspPos(tb, off)
}
