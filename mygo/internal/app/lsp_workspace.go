package app

import (
	"sort"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// lspWorkspaceScan 对项目内源文件发轻量 didOpen,触发工作区级诊断(有上限;不支持 pull 的服务器也走此路径)。
func (a *App) lspWorkspaceScan() {
	if a.proj == "" || !a.lspOn() || !a.prefs.Bool("lsp.workspaceDiag", true) {
		return
	}
	limit := a.prefs.Int("lsp.workspaceDiagLimit", 80)
	if limit < 1 {
		limit = 80
	}
	var files []string
	for _, p := range a.ide.allFiles() {
		if a.lspFor(p) == nil && symbolLang(p) == "" {
			continue
		}
		files = append(files, p)
	}
	sort.Strings(files)
	if len(files) > limit {
		files = files[:limit]
	}
	gen := a.lspH.workspaceGen + 1
	a.lspH.workspaceGen = gen
	a.bg("lsp.workspace", func() func() {
		defer diag.Recover("lsp.workspace")
		end := diag.SpanSlow("lsp", "工作区诊断扫盘", 8*time.Second, "files", len(files))
		n := 0
		for _, rel := range files {
			if gen != a.lspH.workspaceGen {
				break
			}
			s := a.lspFor(rel)
			if s == nil || s.state != lspReady {
				continue
			}
			a.lspTouchDoc(rel)
			n++
		}
		end()
		return func() {
			if gen != a.lspH.workspaceGen {
				return
			}
			if n > 0 {
				a.toast = "工作区诊断:已检查 " + itoa(n) + " 个文件"
			}
			a.redraw()
		}
	})
}
