package app

import (
	"path/filepath"

	"github.com/ixxxxoooo/minicode/internal/lsp"
)

// codeActs 是最近一次 ⌘. 请求到的 LSP 代码操作(异步填充)。
func (a *App) refreshCodeActions(tb *ideTab) {
	a.codeActs = nil
	s := a.lspReadyFor(tb.path)
	if s == nil || tb.st == nil || tb.buf == nil {
		return
	}
	uri := lsp.PathToURI(filepath.Join(a.proj, tb.path))
	off := tb.st.Cursor()
	rng := lspRange(tb, off, off+1)
	a.lspJob("codeAction", s, func() func() {
		acts, err := s.c.CodeAction(uri, rng)
		return func() {
			if err != nil {
				return
			}
			a.codeActs = acts
		}
	})
}

func (a *App) applyCodeAction(i int) {
	if i < 0 || i >= len(a.codeActs) {
		return
	}
	act := a.codeActs[i]
	if act.Edit == nil || len(act.Edit.Changes) == 0 {
		a.toast = act.Title
		return
	}
	n, f, err := a.lspApplyChanges(act.Edit.Changes)
	if err != nil {
		a.toast = "代码操作失败:" + err.Error()
		return
	}
	a.toast = act.Title + " · " + itoa(n) + " 处(" + itoa(f) + " 个文件)"
	a.codeActs = nil
}

func lspRange(tb *ideTab, start, end int) lsp.Range {
	return lsp.Range{Start: lspPos(tb, start), End: lspPos(tb, end)}
}
