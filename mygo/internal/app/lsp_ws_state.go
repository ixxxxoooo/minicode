package app

import (
	"path/filepath"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/lsp"
)

// lspWsScan 记录单个(项目根 + 服务器)的工作区诊断进度与结果摘要。
type lspWsScan struct {
	Scanning  bool
	Checked   int
	Total     int
	Truncated bool
	Source    string // 服务器 pull / 逐文件拉取 / 逐个打开扫描
	Err       string
	Server    string
	Canceled  bool
}

func (a *App) lspWsScanSt(key string) *lspWsScan {
	if a.lspH.wsScan == nil {
		a.lspH.wsScan = map[string]*lspWsScan{}
	}
	if st := a.lspH.wsScan[key]; st != nil {
		return st
	}
	st := &lspWsScan{}
	a.lspH.wsScan[key] = st
	return st
}

func (a *App) lspWsMode(s *lspSess) string {
	if s.c == nil {
		return "didopen"
	}
	if s.c.Diag.WorkspacePull {
		return "pull"
	}
	if s.c.Diag.DocumentPull {
		return "docpull"
	}
	return "didopen"
}

func (a *App) lspWsSourceLabel(mode string) string {
	switch mode {
	case "pull":
		return "服务器 pull"
	case "docpull":
		return "逐文件拉取"
	default:
		return "逐个打开扫描"
	}
}

// lspRelFromURI 把 LSP URI 换成项目相对路径;失败返回空。
func (a *App) lspRelFromURI(uri string) string {
	abs := lsp.URIToPath(uri)
	if abs == "" || a.proj == "" {
		return ""
	}
	rel, err := filepath.Rel(a.proj, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	return rel
}
