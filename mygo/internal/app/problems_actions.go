package app

import (
	"path/filepath"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/lsp"
)

var probLog = diag.L("problems")

// recheckProblems 触发工作区级诊断重扫并刷新各语言服务器会话。
func (a *App) recheckProblems() {
	if a.proj == "" {
		return
	}
	probLog.Info("重新检查问题")
	for _, s := range a.lspH.sess {
		if s.state == lspReady {
			a.lspRetry(s)
		}
	}
	if a.prefs.Bool("lsp.workspaceDiag", true) {
		a.lspWorkspaceScan()
	}
	for _, tb := range a.ide.tabs {
		if tb.buf != nil && !tb.diff && tb.virt == "" {
			delete(a.lspH.outline, tb.path)
			if s := a.lspFor(tb.path); s != nil && s.state == lspReady {
				a.lspTouchDoc(tb.path)
			}
		}
	}
	a.toast = "正在重新检查…"
	a.redraw()
}

// lspTouchDoc 对未打开 / 已打开文档发 didOpen(强制刷新诊断)。
func (a *App) lspTouchDoc(rel string) {
	s := a.lspFor(rel)
	if s == nil || s.state != lspReady {
		return
	}
	text, ok := a.ide.fileText(rel)
	if !ok {
		return
	}
	uri := lsp.PathToURI(filepath.Join(a.proj, rel))
	ver := 1
	if d := s.docs[rel]; d != nil {
		ver = d.ver + 1
	} else {
		s.docs[rel] = &lspDoc{ver: ver}
	}
	s.do(func() { _ = s.c.DidOpen(uri, s.spec.LangID(rel), ver, text) })
}

func problemSeverityPass(sev int, errorsOnly bool) bool {
	if !errorsOnly {
		return true
	}
	return sev == 1
}
