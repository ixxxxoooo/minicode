package app

import (
	"path/filepath"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/lsp"
)

var probLog = diag.L("problems")

// recheckProblems 按当前分段触发诊断刷新(已打开 = 同步打开文档;工作区 = 新扫描)。
func (a *App) recheckProblems() {
	if a.proj == "" {
		if a.wsDiagDemo != "" || a.probScope() == "ws" {
			a.wsDiagDemo = "scan"
			a.ide.probScope = "ws"
			a.redraw()
		}
		return
	}
	probLog.Info("重新检查问题", "scope", a.probScope())
	if a.probScope() == "ws" {
		if a.prefs.Bool("lsp.workspaceDiag", true) {
			a.lspH.wsDiags = nil
			for k := range a.lspH.wsScan {
				a.lspH.wsScan[k] = &lspWsScan{}
			}
			a.lspWorkspaceScan()
		}
		a.toast = "正在检查工作区…"
		a.redraw()
		return
	}
	for _, tb := range a.ide.tabs {
		if tb.buf != nil && !tb.diff && tb.virt == "" {
			delete(a.lspH.outline, tb.path)
			if s := a.lspFor(tb.path); s != nil && s.state == lspReady {
				a.lspTouchDoc(tb.path)
			}
		}
	}
	a.toast = "正在重新检查已打开文件…"
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
