package app

import (
	"path/filepath"

	"github.com/ixxxxoooo/minicode/internal/lsp"
)

// lspHub 的大纲缓存附在会话外,按文件路径存。
type outlineEntry struct {
	ver  uint64
	syms []symbol
	busy bool
}

// symbolsForFile 优先用语言服务器的 documentSymbol,失败退回本地索引。
func (a *App) symbolsForFile(p string) []symbol {
	if e, ok := a.lspH.outline[p]; ok && !e.busy && len(e.syms) > 0 {
		if tb := a.ide.tab(p); tb != nil && tb.buf != nil && e.ver == tb.buf.Version() {
			return e.syms
		}
	}
	if syms := a.ide.fileSyms(p); len(syms) > 0 {
		a.scheduleLspSymbols(p)
		return syms
	}
	a.scheduleLspSymbols(p)
	if e, ok := a.lspH.outline[p]; ok && len(e.syms) > 0 {
		return e.syms
	}
	return a.ide.fileSyms(p)
}

func (a *App) scheduleLspSymbols(rel string) {
	s := a.lspFor(rel)
	if s == nil || s.state != lspReady {
		return
	}
	if a.lspH.outline == nil {
		a.lspH.outline = map[string]outlineEntry{}
	}
	ent, _ := a.lspH.outline[rel]
	if ent.busy {
		return
	}
	ent.busy = true
	a.lspH.outline[rel] = ent
	uri := lsp.PathToURI(filepath.Join(a.proj, rel))
	a.lspJob("outline", s, func() func() {
		raw, err := s.c.DocumentSymbols(uri)
		syms := lspDocSymbols(raw)
		return func() {
			ver := uint64(0)
			if tb := a.ide.tab(rel); tb != nil && tb.buf != nil {
				ver = tb.buf.Version()
			}
			if err != nil || len(syms) == 0 {
				a.lspH.outline[rel] = outlineEntry{ver: ver, busy: false}
				return
			}
			a.lspH.outline[rel] = outlineEntry{ver: ver, syms: syms, busy: false}
			a.redraw()
		}
	})
}

func lspDocSymbols(items []lsp.DocSymbol) []symbol {
	var out []symbol
	var walk func(list []lsp.DocSymbol, parent int)
	walk = func(list []lsp.DocSymbol, parent int) {
		for _, it := range list {
			kind := lspSymKind(it.Kind)
			col := lsp.ColFromUTF16("", it.Range.Start.Character)
			idx := len(out)
			out = append(out, symbol{
				Name: it.Name, Kind: kind, Line: it.Range.Start.Line, Col: col,
				End: it.Range.End.Line, Parent: parent, Detail: it.Detail,
			})
			if len(it.Children) > 0 {
				walk(it.Children, idx)
			}
		}
	}
	walk(items, -1)
	return out
}

func lspSymKind(k int) string {
	switch k {
	case 12, 5:
		return "fn"
	case 23:
		return "h"
	case 11:
		return "const"
	case 10:
		return "enum"
	case 8, 22:
		return "struct"
	case 13:
		return "mod"
	default:
		return "type"
	}
}
