package app

import (
	"sort"

	"github.com/ixxxxoooo/minicode/internal/lsp"
)

func (a *App) probScope() string {
	if a.ide.probScope == "ws" {
		return "ws"
	}
	return "open"
}

// problemCount 统计某一分段的错误 + 警告条目数(不受「仅错误」筛选影响)。
func (a *App) problemCount(scope string) int {
	n := 0
	for _, ds := range a.problemDiags(scope) {
		for _, d := range ds {
			if d.Severity == 1 || d.Severity == 2 {
				n++
			}
		}
	}
	return n
}

func (a *App) problemDiags(scope string) map[string][]lsp.Diagnostic {
	out := map[string][]lsp.Diagnostic{}
	if scope == "ws" {
		if a.wsDiagDemo != "" {
			return a.demoWsDiags()
		}
		for k, v := range a.lspH.wsDiags {
			out[k] = v
		}
		return out
	}
	// 已打开:仅已在语言服务里同步过的文件
	for _, tb := range a.ide.tabs {
		if tb.buf == nil || tb.diff || tb.virt != "" {
			continue
		}
		if s := a.lspFor(tb.path); s == nil || s.docs[tb.path] == nil {
			continue
		}
		if ds := a.lspH.diags[tb.path]; len(ds) > 0 {
			out[tb.path] = ds
		}
	}
	return out
}

func (a *App) anyWsScanning() bool {
	for _, st := range a.lspH.wsScan {
		if st != nil && st.Scanning {
			return true
		}
	}
	return a.wsDiagDemo == "scan"
}

func (a *App) wsScanProgress() (checked, total int) {
	if a.wsDiagDemo == "scan" {
		return 128, 640
	}
	if a.wsDiagDemo == "trunc" {
		return 2000, 2000
	}
	if a.wsDiagDemo == "done" {
		return 640, 640
	}
	for _, st := range a.lspH.wsScan {
		if st == nil {
			continue
		}
		if st.Scanning || st.Checked > 0 || st.Total > 0 {
			checked += st.Checked
			total += st.Total
		}
	}
	return
}

func (a *App) wsScanSources() string {
	for _, st := range a.lspH.wsScan {
		if st != nil && st.Source != "" && st.Err == "" && !st.Scanning {
			return st.Source
		}
	}
	if a.wsDiagDemo == "done" {
		return "服务器 pull"
	}
	if a.wsDiagDemo == "trunc" || a.wsDiagDemo == "scan" {
		return "逐个打开扫描"
	}
	return ""
}

func (a *App) wsScanError() (server, msg string) {
	if a.wsDiagDemo == "err" {
		return "rust-analyzer", "超时 30 s"
	}
	for _, st := range a.lspH.wsScan {
		if st != nil && st.Err != "" {
			return st.Server, st.Err
		}
	}
	return "", ""
}

func (a *App) wsScanTruncated() bool {
	if a.wsDiagDemo == "trunc" {
		return true
	}
	for _, st := range a.lspH.wsScan {
		if st != nil && st.Truncated {
			return true
		}
	}
	return false
}

func (a *App) sortedProblemFiles(scope string) []string {
	m := a.problemDiags(scope)
	files := make([]string, 0, len(m))
	for f := range m {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

func (a *App) demoWsDiags() map[string][]lsp.Diagnostic {
	switch a.wsDiagDemo {
	case "empty", "scan", "err":
		return map[string][]lsp.Diagnostic{}
	case "trunc", "done":
		return map[string][]lsp.Diagnostic{
			"handlers/mod.rs": {{Severity: 2, Message: "未使用的 import", Source: "rust-analyzer", Range: lsp.Range{Start: lsp.Position{Line: 2}}}},
		}
	default:
		return nil
	}
}
