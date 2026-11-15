package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/projects"
	"github.com/ixxxxoooo/minicode/internal/vcs"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// 工作区与多仓库(设计稿「项目与工作区」「多仓库 Git」两页)。
//
//	单项目:a.ide.fs 是一个文件夹,a.repo / a.sc 就是它的仓库与状态(原有行为不变)。
//	工作区:a.ide.fs 是 workspace.Multi(顶层 = 各文件夹),a.repos 每个文件夹一项;
//	        a.repo / a.sc 始终是「活动仓库」的,切换活动仓库 = 换掉它们并重读。

// repoEntry 是工作区里一个文件夹的仓库信息。
type repoEntry struct {
	name    string // 顶层节点名(= Multi 根名,也是路径第一段);嵌套仓库是「父文件夹内相对路径」
	prefix  string // 文件树路径前缀(根 = name;嵌套 = 父文件夹名/相对路径)
	kind    string // 嵌套仓库的类别:"子模块"(.gitmodules 里有)/"嵌套",根为空
	parent  int    // 嵌套仓库所属根的序号(根为 0,无意义)
	dir     string // 绝对目录(演示是虚拟路径)
	repo    vcs.Repo
	checked bool // 已经探测过是不是仓库(真实模式异步探测)

	branch        string
	ahead, behind int
	changes       []vcs.Change // 非活动仓库的最近一次状态(树上的状态色、总览行的改动数)
	loaded        bool
	loading       bool
	last          time.Time
	msg           string // 该仓库未提交的说明草稿(切换活动仓库时保留)
	busy          string // 正在对这个仓库执行的操作(拉取 / 推送 / 同步 / 储藏),空 = 空闲
	opErr         string // 最近一次操作失败的原因(总览行显示「重试」)
	retry         string // 失败操作的名字,点「重试」再做一遍
}

// soloState 是进入工作区前的单项目现场,退出工作区时还原。
type soloState struct {
	fs   workspace.FS
	git  map[string]string
	head func(string) (string, bool)
	repo vcs.Repo
	proj string
	sc   scmState
}

// SetRegistry 注入项目与工作区存储(真实:磁盘;演示:内存)。
func (a *App) SetRegistry(r *projects.Store) { a.reg = r }

// demoWs 工作区数据是不是演示的(内存存储 = 演示或测试,成员取自 demo 数据而不是磁盘)。
func (a *App) demoWs() bool { return !a.registry().Persistent() }

// SeedDemoRegistry 演示模式启动时调用:内存存储里放演示工作区。
func (a *App) SeedDemoRegistry() { a.seedDemoRegistry() }

// registry 保证有存储(测试里没注入时用内存的)。
func (a *App) registry() *projects.Store {
	if a.reg == nil {
		a.reg = projects.NewMem()
	}
	return a.reg
}

// multiRoot 当前是不是多根工作区。
func (a *App) multiRoot() bool { return a.ws != nil && len(a.repos) > 0 }

// activeWsID 当前工作区的 id("" = 没有)。
func (a *App) activeWsID() string {
	if a.ws == nil {
		return ""
	}
	return a.ws.ID
}

// ───────── 演示数据 ─────────

// seedDemoRegistry 演示 / 截图:内存存储里放「全栈工作区」(3 个文件夹)与一个手动添加的非仓库项目。
func (a *App) seedDemoRegistry() {
	r := projects.NewMem()
	known := map[string]bool{}
	for _, f := range demo.Discovered() {
		known[projects.Clean(f.Path)] = true
	}
	r.Exists = func(p string) bool { return strings.HasPrefix(p, "/demo/") || known[p] } // 演示:只认样例路径,其余走「找不到」
	_, _ = r.AddProject("/demo/design-assets")
	_, _ = r.SaveWorkspace(projects.Workspace{ID: "ws-fullstack", Name: "全栈工作区", Folders: []projects.Folder{
		{Path: "/demo/router-demo"}, {Path: "/demo/web-dashboard"}, {Path: "/demo/docs-site"},
	}})
	a.reg = r
	a.projOpen["ws-fullstack"] = true
}

// ───────── 侧栏数据 ─────────

// sideProject 是侧栏里的一行项目:Claude 索引里的项目 + 手动添加但还没有会话的文件夹。
func (a *App) sideProjects() []core.ProjectInfo {
	out := append([]core.ProjectInfo(nil), a.projects...)
	have := map[string]bool{}
	for _, p := range out {
		have[projects.Clean(p.Path)] = true
	}
	for _, p := range a.registry().Projects() {
		if !have[p] {
			out = append(out, core.ProjectInfo{ID: "path:" + p, Name: filepath.Base(p), Path: p})
		}
	}
	return out
}

// memberOfWorkspace 某个文件夹属于哪些工作区的成员(侧栏里成员文件夹折进工作区,不再单独列)。
func (a *App) memberOfWorkspace(path string) bool {
	return len(a.registry().WorkspacesOf(path)) > 0
}

// projectByPath 按目录找侧栏项目(没有会话的文件夹也能找到)。
func (a *App) projectByPath(path string) (core.ProjectInfo, bool) {
	c := projects.Clean(path)
	for _, p := range a.sideProjects() {
		if projects.Clean(p.Path) == c {
			return p, true
		}
	}
	return core.ProjectInfo{}, false
}

// isRepoDir 这个文件夹是不是仓库(侧栏的「非仓库」标签)。演示按沙盒数据,真实看 .git。
func (a *App) isRepoDir(path string) bool {
	if a.demoWs() {
		for _, f := range demo.WorkspaceFolders() {
			if f.Path == path {
				return f.Repo != nil
			}
		}
		return true
	}
	return dirHasGit(path)
}

// ───────── 激活与退出 ─────────

// folderSrc 是构建多根文件系统时每个文件夹的来源。
type folderSrc struct {
	name, path string
	fs         workspace.FS
	repo       vcs.Repo
	git        map[string]string
	head       func(string) (string, bool)
	nested     []demo.WsNested
}

// ActivateWorkspace 切到某个工作区:文件树 / 搜索变成多根,源代码管理按仓库分开。
func (a *App) ActivateWorkspace(id string) {
	w, ok := a.registry().Workspace(id)
	if !ok {
		a.toast = "工作区不存在"
		return
	}
	if a.ws != nil && a.ws.ID == id {
		return
	}
	defer diag.SpanSlow("app", "切换工作区", 300*time.Millisecond)()
	if a.ws != nil {
		a.leaveWorkspace()
	}
	log.Info("进入工作区", "id", w.ID, "name", w.Name, "folders", len(w.Folders))
	var src []folderSrc
	if a.demoWs() { // 演示:成员从演示数据里按路径取
		all := demo.WorkspaceFolders()
		for _, f := range w.Folders {
			for _, d := range all {
				if d.Path == f.Path {
					src = append(src, folderSrc{name: f.Label(), path: d.Path, fs: d.FS, repo: d.Repo, git: d.Git, head: d.Head, nested: d.Nested})
				}
			}
		}
	} else {
		for _, f := range w.Folders {
			src = append(src, folderSrc{name: f.Label(), path: f.Path, fs: workspace.NewOS(f.Path)})
		}
	}
	if len(src) == 0 {
		a.toast = "这个工作区里的文件夹都找不到"
		return
	}
	a.sr.scope = ""
	a.solo = &soloState{fs: a.ide.fs, git: a.ide.git, head: a.ide.headOf, repo: a.repo, proj: a.proj, sc: a.sc}
	roots := make([]workspace.Root, len(src))
	for i, s := range src {
		roots[i] = workspace.Root{Name: s.name, FS: s.fs}
	}
	multi := workspace.NewMulti(w.Name, roots)
	a.ws = &w
	a.repos = nil
	for i, s := range src {
		nm := multi.Roots()[i].Name
		e := &repoEntry{name: nm, prefix: nm, dir: s.path, repo: s.repo, checked: s.repo != nil || a.demoWs()}
		if s.repo != nil {
			e.branch = s.repo.Current()
		}
		a.repos = append(a.repos, e)
	}
	if !a.demoWs() {
		a.proj = w.Primary()
		a.lspReset()
		a.closeTerm()
	}
	a.ide.useFS(multi, map[string]string{}, a.headOfRepo)
	a.ide.termLines = nil
	a.repoAct = 0
	a.sc = scmState{gen: a.sc.gen + 1}
	a.SetRepo(a.repos[0].repo)
	if a.demoWs() { // 演示:树上的状态色一开始就齐
		for i, s := range src {
			a.repos[i].changes = demoChanges(s.repo)
			for _, n := range s.nested {
				ne := a.addNested(i, n.Rel, n.Repo, n.Kind)
				ne.checked, ne.changes = true, demoChanges(n.Repo)
			}
		}
	}
	a.syncTreeGit()
	a.openRepoProbes()
	a.discoverNested()
	a.projOpen[w.ID] = true
	a.ide.open[a.repos[0].name] = true
	if len(a.repos) > 1 {
		a.ide.open[a.repos[1].name] = true
	}
}

func demoChanges(r vcs.Repo) []vcs.Change {
	if r == nil {
		return nil
	}
	cs, _ := r.Status()
	return cs
}

// openRepoProbes 真实模式:后台探测每个文件夹是不是仓库。
func (a *App) openRepoProbes() {
	if a.demoWs() {
		return
	}
	ws := a.ws
	for i, e := range a.repos {
		i, e := i, e
		a.bg("probe-repo", func() func() {
			cli, err := vcs.Open(e.dir)
			return func() {
				if a.ws != ws || i >= len(a.repos) || a.repos[i] != e {
					return
				}
				e.checked = true
				if err != nil {
					return
				}
				e.repo = cli
				e.branch = cli.Current()
				if i == a.repoAct {
					a.SetRepo(cli)
					a.sc = scmState{gen: a.sc.gen + 1, msg: a.sc.msg}
				}
			}
		})
	}
}

// leaveWorkspace 退出工作区,还原进入前的单项目现场。
func (a *App) leaveWorkspace() {
	if a.ws == nil {
		return
	}
	log.Info("退出工作区", "id", a.ws.ID)
	a.sr.scope = ""
	a.ws, a.repos, a.repoAct = nil, nil, 0
	s := a.solo
	a.solo = nil
	if s == nil {
		return
	}
	if s.proj == "" { // 演示:原样还原(含未提交的演示改动)
		a.ide.useFS(s.fs, s.git, s.head)
		a.SetRepo(s.repo)
		a.sc = s.sc
		return
	}
	a.proj = ""
	a.SetProject(s.proj)
}

// ───────── 活动仓库 ─────────

// setActiveRepo 切换源代码管理操作的仓库。
func (a *App) setActiveRepo(i int) {
	if !a.multiRoot() || i < 0 || i >= len(a.repos) || i == a.repoAct {
		return
	}
	cur := a.repos[a.repoAct]
	cur.msg = a.sc.msg
	if a.sc.loaded { // 没读完的状态不覆盖旧快照
		cur.changes, cur.ahead, cur.behind, cur.branch, cur.loaded = a.sc.changes, a.sc.ahead, a.sc.behind, a.branch, true
	}
	a.repoAct = i
	e := a.repos[i]
	log.Info("切换活动仓库", "repo", e.name)
	a.sc = scmState{gen: a.sc.gen + 1, msg: e.msg}
	a.SetRepo(e.repo)
	a.syncTreeGit()
}

// activeRepo 活动仓库项(单项目时 nil)。
func (a *App) activeRepo() *repoEntry {
	if !a.multiRoot() || a.repoAct >= len(a.repos) {
		return nil
	}
	return a.repos[a.repoAct]
}

// repoSummary 总览行用:分支、领先 / 落后、改动数(活动仓库取实时状态)。
func (a *App) repoSummary(i int) (branch string, ahead, behind, n int) {
	e := a.repos[i]
	cs, ah, bh, br := e.changes, e.ahead, e.behind, e.branch
	if i == a.repoAct && a.sc.loaded { // 活动仓库读完后取实时状态,之前用激活时的快照
		cs, ah, bh, br = a.sc.changes, a.sc.ahead, a.sc.behind, a.branch
	} else if i == a.repoAct {
		br = a.branch
	}
	seen := map[string]bool{}
	for _, c := range cs {
		seen[c.Path] = true
	}
	return br, ah, bh, len(seen)
}

// treePath 活动仓库内的相对路径 → 文件树路径(多根时加文件夹名前缀)。
func (a *App) treePath(rel string) string {
	if e := a.activeRepo(); e != nil {
		return e.pre() + "/" + rel
	}
	return rel
}

// repoAt 文件树路径 → 它所在的仓库与仓库内相对路径(单项目时就是 a.repo)。
func (a *App) repoAt(p string) (vcs.Repo, string) {
	if a.multiRoot() {
		if i, rest, ok := a.repoIndexOf(p); ok {
			return a.repos[i].repo, rest
		}
		return nil, p
	}
	return a.repo, p
}

// reposTick 每帧调用(多根时):隔一段时间在后台轻量刷新非活动仓库的状态。
func (a *App) reposTick() {
	if !a.multiRoot() {
		return
	}
	for i, e := range a.repos {
		if i == a.repoAct || e.repo == nil || e.loading || time.Since(e.last) < 8*time.Second {
			continue
		}
		i, e, ws := i, e, a.ws
		e.loading, e.last = true, time.Now()
		a.bg("repo-light", func() func() {
			cs, _ := e.repo.Status()
			ah, bh := e.repo.AheadBehind()
			br := e.repo.Current()
			return func() {
				e.loading = false
				if a.ws != ws || i >= len(a.repos) || a.repos[i] != e {
					return
				}
				e.changes, e.ahead, e.behind, e.branch, e.loaded = cs, ah, bh, br, true
				a.syncTreeGit()
			}
		})
	}
}

// initRepo 给非仓库文件夹 `git init`(演示:换成内存空仓库)。
func (a *App) initRepo(i int) {
	if !a.multiRoot() || i >= len(a.repos) {
		return
	}
	e := a.repos[i]
	if a.demoWs() {
		m := vcs.NewMem()
		m.SetBranch("main", 0)
		e.repo, e.branch = m, "main"
	} else {
		dir := e.dir
		a.bg("git-init", func() func() {
			cli, err := vcs.Init(dir)
			return func() {
				if err != nil {
					a.toast = "初始化失败:" + err.Error()
					return
				}
				e.repo, e.branch = cli, cli.Current()
				a.outAppend(outGit, "git init "+e.name+" 完成")
				a.toast = "已在 " + e.name + " 初始化 Git 仓库"
				if i == a.repoAct {
					a.SetRepo(cli)
					a.sc = scmState{gen: a.sc.gen + 1, msg: a.sc.msg}
				}
			}
		})
		return
	}
	if i == a.repoAct {
		a.SetRepo(e.repo)
		a.sc = scmState{gen: a.sc.gen + 1, msg: a.sc.msg}
	}
	a.outAppend(outGit, "git init "+e.name+" 完成(演示)")
	a.toast = "已在 " + e.name + " 初始化 Git 仓库"
}

// wsRelMulti 工具调用里的绝对路径 → 文件树路径(多根):找到包含它的文件夹。
func (a *App) wsRelMulti(p string) (string, bool) {
	for _, e := range a.repos {
		if e.kind != "" { // 嵌套仓库的目录在父文件夹里,由父文件夹那一项换算
			continue
		}
		if p == e.dir || strings.HasPrefix(p, e.dir+"/") {
			return e.name + strings.TrimPrefix(p, e.dir), true
		}
	}
	return p, false
}

// focusRepoOf 返回文件树路径在其仓库里的相对路径;多根且这个文件不属于活动仓库时先切过去,
// 这样随后的 scmAct(作用在活动仓库上)落在正确的仓库。
func (a *App) focusRepoOf(p string) string {
	if !a.multiRoot() {
		return p
	}
	i, rest, ok := a.repoIndexOf(p)
	if !ok {
		return p
	}
	a.setActiveRepo(i)
	return rest
}

// leaveWorkspaceKeep 退出工作区但不还原单项目现场(调用方马上要切到另一个项目)。
func (a *App) leaveWorkspaceKeep() {
	if a.ws == nil {
		return
	}
	log.Info("退出工作区(切到别的项目)", "id", a.ws.ID)
	a.ws, a.repos, a.repoAct, a.solo = nil, nil, 0, nil
	a.proj = "" // 让随后的 SetProject 一定走完整切换
}

// dirHasGit 文件夹里有没有 .git(目录或 worktree 的文件都算)。
func dirHasGit(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// maybeLeaveWorkspace 打开的会话不属于当前工作区的任何成员文件夹时退出工作区。
// 真实模式由 SetProject 在会话打开后统一处理(会话所在目录要读索引才知道);这里只管演示数据。
func (a *App) maybeLeaveWorkspace(id string) {
	if a.ws == nil || !a.demoWs() || id == "" {
		return
	}
	path, ok := strings.CutPrefix(id, "new:")
	if !ok {
		for _, p := range a.projects {
			for _, s := range p.Sessions {
				if s.ID == id {
					path = p.Path
				}
			}
		}
	}
	if path != "" && !a.ws.Has(path) {
		a.leaveWorkspace()
	}
}

// OpenProjDialog 直接打开某个对话框(截图 / 调试用):add / ws / wsedit / confirm。
func (a *App) OpenProjDialog(kind string) {
	switch kind {
	case "add":
		a.openAddProject()
	case "ws":
		a.openWsDialog("", "")
	case "wsedit":
		a.openWsDialog("ws-fullstack", "")
	case "confirm":
		a.confirmDlg("移除工作区「全栈工作区」?", "只会从列表里移除这个工作区,磁盘上的文件夹与会话记录都不会动。", "移除", true, nil)
	}
}

// SetAddProjectPath 截图用:给添加项目对话框预填路径并提交(演示校验错误)。
func (a *App) SetAddProjectPath(p string) {
	a.pd.path = p
	a.submitAddProject()
}

// SetActiveRepoIndex 截图 / 调试用:切换活动仓库。
func (a *App) SetActiveRepoIndex(i int) { a.setActiveRepo(i) }

// rootBadges 多根文件树里顶层文件夹行右侧的徽标:「主」、该仓库的改动数、非仓库。
func (a *App) rootBadges(c *ui.Context, name string) {
	t := a.tk
	for i, e := range a.repos {
		if e.name != name {
			continue
		}
		if i == 0 {
			kit.Tag(c, "主", kit.KAc)
		}
		if e.repo == nil && e.checked {
			kit.Tag(c, "非仓库")
			return
		}
		if _, _, _, n := a.repoSummary(i); n > 0 {
			ui.Text(c, itoa(n)).FontSize(theme.FsMicro).FontWeight(700).TextColor(t.Warn).FontFeatures("tnum").Shrink(0)
		}
		return
	}
}

// repoBranchLabel 状态栏的分支文字:多仓库时带上仓库名("web-dashboard · feat/charts"),让人知道切的是哪个。
func (a *App) repoBranchLabel() string {
	if e := a.activeRepo(); e != nil {
		return e.name + " · " + a.branch
	}
	return a.branch
}
