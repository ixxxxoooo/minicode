package app

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/lsp"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

var wsLog = diag.L("lsp.workspace")

const wsDiagTimeout = 30 * time.Second

// lspWorkspaceScan 按各会话能力启动工作区诊断(服务器 pull / 逐文件拉取 / didOpen 扫盘),彼此独立。
func (a *App) lspWorkspaceScan() {
	if a.proj == "" || !a.lspOn() || !a.prefs.Bool("lsp.workspaceDiag", true) {
		return
	}
	limit := a.prefs.Int("lsp.workspaceDiagLimit", 2000)
	if limit < 1 {
		limit = 2000
	}
	includeIgnored := a.prefs.Bool("lsp.workspaceDiagGitignore", false)
	cands := a.lspWorkspaceCandidates(includeIgnored)
	groups := a.ignoreGroups(cands, includeIgnored)
	if len(groups) == 0 {
		a.lspWorkspaceStart(capFiles(cands, limit), limit)
		return
	}
	// 按仓库的 .gitignore 过滤要起 git 子进程,放后台;结果回主线程再启动扫描
	a.bg("lsp.ws.ignore", func() func() {
		defer diag.Recover("lsp.workspace.ignore")
		end := diag.SpanSlow("lsp", "工作区诊断·忽略过滤", 2*time.Second, "files", len(cands))
		kept := dropIgnored(cands, groups)
		end()
		return func() {
			if n := len(cands) - len(kept); n > 0 {
				wsLog.Info("工作区诊断按 .gitignore 跳过文件", "skipped", n, "kept", len(kept))
			}
			a.lspWorkspaceStart(capFiles(kept, limit), limit)
		}
	})
}

// dropIgnored 去掉被 .gitignore 忽略的文件(会起 git 子进程,不要在界面线程调用)。
// 某个仓库读规则失败时只记警告、该仓库的文件原样保留。
func dropIgnored(cands []string, groups []ignoreGroup) []string {
	drop := map[string]bool{}
	for _, g := range groups {
		hit, err := g.ck.Ignored(g.rels)
		if err != nil {
			wsLog.Warn("读取 .gitignore 规则失败,该仓库按常见目录兜底", "err", err)
			continue
		}
		for i, rel := range g.rels {
			if hit[rel] {
				drop[g.trees[i]] = true
			}
		}
	}
	kept := make([]string, 0, len(cands))
	for _, p := range cands {
		if !drop[p] {
			kept = append(kept, p)
		}
	}
	return kept
}

// ignoreGroup 是同一个仓库里待判定的文件:rels 是相对仓库的路径,trees 是对应的文件树路径(多根时带文件夹前缀)。
type ignoreGroup struct {
	ck    vcs.IgnoreChecker
	rels  []string
	trees []string
}

// ignoreGroups 把候选文件按仓库分组;只有真实 git 的仓库(实现了 IgnoreChecker)才参与,
// 演示 / 非仓库 / 用户选择「包含被忽略的文件」时返回空,调用方直接用硬编码目录过滤后的结果。
func (a *App) ignoreGroups(files []string, includeIgnored bool) []ignoreGroup {
	if includeIgnored {
		return nil
	}
	idx := map[vcs.IgnoreChecker]int{}
	var out []ignoreGroup
	for _, p := range files {
		repo, rel := a.repoAt(p)
		ck, ok := repo.(vcs.IgnoreChecker)
		if !ok || repo == nil {
			continue
		}
		i, seen := idx[ck]
		if !seen {
			i = len(out)
			idx[ck] = i
			out = append(out, ignoreGroup{ck: ck})
		}
		out[i].rels = append(out[i].rels, rel)
		out[i].trees = append(out[i].trees, p)
	}
	return out
}

// capFiles 取前 limit 个(已排序)。
func capFiles(files []string, limit int) []string {
	if len(files) > limit {
		return files[:limit]
	}
	return files
}

// lspWorkspaceStart 对已定下来的文件列表按会话分组并启动扫描(主线程)。
func (a *App) lspWorkspaceStart(files []string, limit int) {
	bySess := map[string][]string{}
	for _, rel := range files {
		s := a.lspFor(rel)
		if s == nil {
			continue
		}
		if s.state != lspReady {
			continue
		}
		key := s.root + "|" + s.spec.ID
		bySess[key] = append(bySess[key], rel)
	}
	gen := a.lspH.workspaceGen + 1
	a.lspH.workspaceGen = gen
	for key, list := range bySess {
		s := a.lspH.sess[key]
		if s == nil {
			continue
		}
		total := len(list)
		wsMode := a.lspWsMode(s)
		if wsMode == "pull" {
			total = len(files) // 工作区 pull 以全项目文件数为分母
		}
		a.onUI(func() {
			st := a.lspWsScanSt(key)
			st.Scanning, st.Checked, st.Total = true, 0, total
			st.Truncated = len(list) >= limit && wsMode != "pull"
			st.Source = a.lspWsSourceLabel(wsMode)
			st.Server = s.spec.Name
			st.Err, st.Canceled = "", false
		})
		key, list, s := key, list, s
		a.bg("lsp.ws."+s.spec.ID, func() func() {
			defer diag.Recover("lsp.workspace")
			return a.lspWsRun(s, key, gen, list, limit)
		})
	}
	a.redraw()
}

// lspWorkspaceCandidates 枚举工作区诊断的候选文件(已排序、未截断):有语言支持的文件,
// 关闭「包含被忽略的文件」时先去掉 target / node_modules 等常见产物目录(再由 .gitignore 精筛)。
func (a *App) lspWorkspaceCandidates(includeIgnored bool) []string {
	var files []string
	for _, p := range a.ide.allFiles() {
		if a.lspFor(p) == nil && symbolLang(p) == "" {
			continue
		}
		if !includeIgnored && skipWorkspacePath(p) {
			continue
		}
		files = append(files, p)
	}
	sort.Strings(files)
	return files
}

func skipWorkspacePath(rel string) bool {
	parts := strings.Split(rel, "/")
	for _, seg := range parts {
		switch seg {
		case "target", "node_modules", "dist", "build", ".git":
			return true
		}
	}
	return false
}

func (a *App) lspWsRun(s *lspSess, key string, gen int, files []string, limit int) func() {
	mode := a.lspWsMode(s)
	end := diag.SpanSlow("lsp", "工作区诊断", 8*time.Second, "server", s.spec.Name, "mode", mode)
	var (
		checked  int
		errMsg   string
		canceled bool
	)
	switch mode {
	case "pull":
		checked, errMsg = a.lspWsServerPull(s, key, gen)
	case "docpull":
		checked, errMsg, canceled = a.lspWsDocPull(s, key, gen, files)
	default:
		checked, errMsg, canceled = a.lspWsDidOpen(s, key, gen, files)
	}
	end()
	source := a.lspWsSourceLabel(mode)
	didFiles := append([]string(nil), files...)
	return func() {
		if gen != a.lspH.workspaceGen {
			return
		}
		if mode == "didopen" {
			for _, rel := range didFiles {
				if ds := a.lspH.diags[rel]; len(ds) > 0 {
					a.lspWsSetFile(key, rel, ds)
				} else {
					a.lspWsClearFile(rel)
				}
			}
		}
		st := a.lspWsScanSt(key)
		st.Scanning = false
		st.Checked = checked
		st.Source = source
		if errMsg != "" {
			st.Err = errMsg
		}
		st.Canceled = canceled
		if checked > 0 && errMsg == "" {
			wsLog.Info("工作区诊断完成", "server", s.spec.Name, "mode", mode, "files", checked, "source", source)
		}
		a.lspH.gen++
		a.redraw()
	}
}

func (a *App) lspWsCanceled(gen int, s *lspSess) bool {
	if gen != a.lspH.workspaceGen {
		if s.c != nil {
			s.c.CancelRequest(s.c.LastRequestID())
		}
		return true
	}
	return false
}

func (a *App) lspWsServerPull(s *lspSess, key string, gen int) (int, string) {
	if s.c == nil {
		return 0, "语言服务器未就绪"
	}
	prev := s.wsPrevID
	var res lsp.WorkspacePullResult
	var err error
	done := make(chan struct{})
	s.do(func() {
		defer close(done)
		res, err = s.c.WorkspaceDiagnostic(s.c.Diag, prev, wsDiagTimeout)
	})
	select {
	case <-done:
	case <-time.After(wsDiagTimeout + 2*time.Second):
		err = errTimeout("workspace/diagnostic")
		s.c.CancelRequest(s.c.LastRequestID())
	}
	if a.lspWsCanceled(gen, s) {
		return res.FilesChecked, ""
	}
	if err != nil {
		wsLog.Warn("工作区 pull 失败", "server", s.spec.Name, "err", err.Error())
		return res.FilesChecked, err.Error()
	}
	s.wsPrevID = res.ResultID
	a.lspWsMergePull(s, key, res.ByURI)
	return max(res.FilesChecked, len(res.ByURI)), ""
}

func (a *App) lspWsDocPull(s *lspSess, key string, gen int, files []string) (int, string, bool) {
	if s.c == nil {
		return 0, "语言服务器未就绪", false
	}
	n := 0
	for i, rel := range files {
		if a.lspWsCanceled(gen, s) {
			return n, "", true
		}
		a.onUI(func() {
			st := a.lspWsScanSt(key)
			st.Checked = i
		})
		uri := lsp.PathToURI(filepath.Join(a.proj, rel))
		var docRes lsp.DocumentPullResult
		var err error
		done := make(chan struct{})
		s.do(func() {
			defer close(done)
			docRes, err = s.c.DocumentDiagnostic(uri, "", 15*time.Second)
		})
		select {
		case <-done:
		case <-time.After(17 * time.Second):
			err = errTimeout("textDocument/diagnostic")
			s.c.CancelRequest(s.c.LastRequestID())
		}
		if a.lspWsCanceled(gen, s) {
			return n, "", true
		}
		if err != nil {
			wsLog.Warn("逐文件拉取失败", "path", rel, "err", err.Error())
			return n, "超时 30 s", false
		}
		if len(docRes.Items) > 0 {
			a.lspWsSetFile(key, rel, docRes.Items)
		} else {
			a.lspWsClearFile(rel)
		}
		n++
	}
	a.onUI(func() {
		st := a.lspWsScanSt(key)
		st.Checked = n
	})
	return n, "", false
}

func (a *App) lspWsDidOpen(s *lspSess, key string, gen int, files []string) (int, string, bool) {
	n := 0
	for i, rel := range files {
		if a.lspWsCanceled(gen, s) {
			if s.c != nil {
				s.c.CancelRequest(s.c.LastRequestID())
			}
			return n, "", true
		}
		a.onUI(func() {
			st := a.lspWsScanSt(key)
			st.Checked = i
		})
		text, ok := a.ide.fileText(rel)
		if !ok {
			continue
		}
		uri := lsp.PathToURI(filepath.Join(a.proj, rel))
		ver := 1
		if d := s.docs[rel]; d != nil {
			ver = d.ver + 1
		} else {
			s.docs[rel] = &lspDoc{ver: ver}
		}
		done := make(chan struct{})
		s.do(func() {
			defer close(done)
			_ = s.c.DidOpen(uri, s.spec.LangID(rel), ver, text)
		})
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			s.c.CancelRequest(s.c.LastRequestID())
		}
		n++
	}
	return n, "", false
}

func (a *App) lspWsMergePull(s *lspSess, key string, byURI map[string][]lsp.Diagnostic) {
	a.onUI(func() {
		if a.lspH.wsDiags == nil {
			a.lspH.wsDiags = map[string][]lsp.Diagnostic{}
		}
		for uri, ds := range byURI {
			rel := a.lspRelFromURI(uri)
			if rel == "" {
				continue
			}
			if len(ds) == 0 {
				delete(a.lspH.wsDiags, rel)
			} else {
				a.lspH.wsDiags[rel] = ds
			}
		}
		a.redraw()
	})
	_ = s
	_ = key
}

func (a *App) lspWsSetFile(key, rel string, ds []lsp.Diagnostic) {
	a.onUI(func() {
		if a.lspH.wsDiags == nil {
			a.lspH.wsDiags = map[string][]lsp.Diagnostic{}
		}
		a.lspH.wsDiags[rel] = ds
		a.lspH.gen++
		a.redraw()
	})
	_ = key
}

func (a *App) lspWsClearFile(rel string) {
	a.onUI(func() {
		delete(a.lspH.wsDiags, rel)
		a.redraw()
	})
}

func errTimeout(method string) error {
	return &diagTimeout{method: method}
}

type diagTimeout struct{ method string }

func (e *diagTimeout) Error() string { return e.method + " 超时(30s)" }

// cancelWorkspaceDiag 取消当前代次并通知各服务器。
func (a *App) cancelWorkspaceDiag() {
	a.lspH.workspaceGen++
	for _, s := range a.lspH.sess {
		if s.c != nil {
			s.c.CancelRequest(s.c.LastRequestID())
		}
	}
	for _, st := range a.lspH.wsScan {
		if st.Scanning {
			st.Canceled = true
			st.Scanning = false
		}
	}
	wsLog.Info("用户取消工作区诊断")
	a.redraw()
}
