package app

import (
	"path"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// 源代码管理(对标 VS Code SCM):状态、操作与图谱数据。界面在 scm_view.go。
// 所有 git 调用都在后台线程跑;结果带「仓库身份 + 代号」,过期的(切了仓库、又发起了新刷新)直接丢弃。

type scmState struct {
	msg           string
	viewMode      string // list / tree(偏好 scm.viewMode)
	dirFold       map[string]bool
	changes       []vcs.Change
	log           []vcs.Commit
	rows          []vcs.GraphRow
	lanes         int // 图谱最大泳道数
	ahead, behind int
	loaded        bool
	loading       bool
	last          time.Time
	gen           int
	err           string
	commitErr     string          // 最近一次提交失败的原因(逐个提交向导显示)
	folded        map[string]bool // staged / changes / graph
	expanded      string          // 展开的提交
	files         map[string][]vcs.Change
	busy          string // 正在进行的操作名(提交 / 拉取 / 推送),空 = 空闲
	amend         bool
	stashes       []vcs.Stash
	genning       bool // 正在让模型生成提交说明
	genGen        int
	createAt      string    // 非空:分支选择器里「创建并切换」从这个提交起
	pushGuide     string    // 推送被拒时的引导文案(空 = 不显示)
	rebase        bool      // 变基冲突中(可中止)
	merging       bool      // 合并冲突中(可中止,git merge --abort)
	graphShown    bool      // 图谱的提交行已经出现过(数据晚于导航切换才到时,图谱自己再落一遍,不整块蹦出)
	graphAt       time.Time // 图谱数据首次出现的时刻
	stashOpen     map[int]bool
	stashFiles    map[int][]vcs.Change
	stashBusy     map[int]bool // 正在读取储藏文件列表
	stashFilesAll map[int]bool // true = 展开全部文件(默认只显示前几条)
	splitH        float32      // 「更改列表 + 图形」区域上一帧高度
	contentH      float32      // 更改列表内容的自然高度(上一帧)
	logLimit      int          // 提交图谱已请求的条数上限(0 = 首批 scmLogFirst);翻页 = 加大它后整体重读,见 scm_log_page.go
	logMore       bool         // 还有更早的提交没读(读回条数 ≥ 请求条数)
	logPaging     bool         // 正在读取更早的提交
	logErr        string       // 读取更早提交失败的原因(空 = 没有失败)
	sideList      ui.ListState // 左栏「图形」分区的虚拟列表
	sideSt        ui.ScrollState
	graphList     ui.ListState // 工作区「Git 图谱」整页的虚拟列表
	graphSt       ui.ScrollState
}

// scmRefreshEvery 是面板可见时自动重读状态的间隔(git status 很轻,另外文件监视接上后可去掉)。
const scmRefreshEvery = 4 * time.Second

// bg 在后台跑 work,返回的 apply 回到界面线程执行。没有窗口(测试)时同步执行。
func (a *App) bg(name string, work func() func()) {
	if a.win == nil {
		if ap := work(); ap != nil {
			ap()
		}
		return
	}
	go func() {
		defer diag.Recover("app." + name)
		t0 := time.Now()
		ap := work()
		diag.PerfLog("app", "后台任务", "name", name, "耗时", time.Since(t0))
		if ap != nil {
			a.win.Update(ap)
		}
	}()
}

// scmTick 每帧调用(面板可见时):首次与定时刷新。
func (a *App) scmTick() {
	s := &a.sc
	if a.repo == nil || s.loading {
		return
	}
	if !s.loaded || time.Since(s.last) > scmRefreshEvery {
		a.scmRefresh()
	}
}

// scmRefresh 重读改动、提交图谱与领先 / 落后数。
func (a *App) scmRefresh() {
	repo := a.repo
	if repo == nil {
		return
	}
	s := &a.sc
	s.gen++
	gen := s.gen
	s.loading, s.last = true, time.Now()
	limit := a.scmLogLimit() // 在界面线程取:后台只读这个快照
	a.bg("scm", func() func() {
		end := diag.SpanSlow("scm", "读取仓库状态", 300*time.Millisecond)
		cs, err := repo.Status()
		var lg []vcs.Commit
		var lerr error
		if err == nil {
			lg, lerr = repo.Log(limit)
		}
		ah, bh := repo.AheadBehind()
		ss, _ := repo.Stashes()
		op := repo.OpState() // 重启后也能接上进行中的合并 / 变基
		end()
		return func() {
			if gen != a.sc.gen || repo != a.repo {
				return
			}
			s := &a.sc
			s.loading, s.loaded = false, true
			if err != nil {
				s.err = err.Error()
				log.Warn("读取仓库状态失败", "err", err)
				return
			}
			s.err = ""
			sortChanges(cs)
			s.changes, s.ahead, s.behind, s.stashes = cs, ah, bh, ss
			s.rebase, s.merging = op == "rebase", op == "merge"
			a.scmApplyLog(lg, lerr, limit)
			a.syncTreeGit()
			a.refreshDiffTabs()
		}
	})
}

// sortChanges 按路径排序,稳定显示。
func sortChanges(cs []vcs.Change) {
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].Path < cs[j].Path })
}

// syncTreeGit 把改动同步给文件树的状态色(暂存与工作区合并;删除不在树里)。
func (a *App) syncTreeGit() {
	m := map[string]string{}
	add := func(prefix string, cs []vcs.Change) {
		for _, c := range cs {
			if c.Status == 'D' {
				continue
			}
			k := prefix + c.Path
			if _, ok := m[k]; !ok || !c.Staged {
				m[k] = string(c.Status)
			}
		}
	}
	if a.multiRoot() { // 多根:每个仓库的改动加上文件夹名前缀
		for i, e := range a.repos {
			cs := e.changes
			if i == a.repoAct && a.sc.loaded {
				cs = a.sc.changes
			}
			add(e.pre()+"/", cs)
		}
	} else {
		add("", a.sc.changes)
	}
	// 改动里出现 / 消失了文件(Claude 或外部新建、删除):文件树的目录缓存已经过期,清掉才会重新读盘。
	// 没有这一步,新建的文件只在「源代码管理」里有,文件树要手动点刷新才出现。
	if treeKeysChanged(a.ide.git, m) {
		a.ide.dropLists()
	}
	a.ide.git = m
	a.rebuildTreeDirGit(m)
}

// treeKeysChanged 两份改动表的路径集合是否不同(只比路径,不比状态字母)。
func treeKeysChanged(old, cur map[string]string) bool {
	if len(old) != len(cur) {
		return true
	}
	for k := range cur {
		if _, ok := old[k]; !ok {
			return true
		}
	}
	return false
}

// staged / unstaged 拆分。
func (s *scmState) split() (staged, work []vcs.Change) {
	for _, c := range s.changes {
		if c.Staged {
			staged = append(staged, c)
		} else {
			work = append(work, c)
		}
	}
	return
}

// scmAct 执行一次会改仓库的操作:后台执行,完成后提示并立刻刷新。
func (a *App) scmAct(what string, f func(r vcs.Repo) error, onOK ...func()) {
	repo := a.repo
	if repo == nil {
		a.toast = "当前项目不是 Git 仓库"
		return
	}
	s := &a.sc
	if s.busy != "" {
		return
	}
	s.busy = what
	a.bg("scm.op", func() func() {
		end := diag.SpanSlow("scm", what, time.Second)
		err := f(repo)
		end()
		return func() {
			a.sc.busy = ""
			if repo != a.repo {
				return
			}
			if err != nil {
				if what == "提交" {
					a.sc.commitErr = "提交失败:" + friendlyGitErr(err.Error())
				}
				a.outAppend(outGit, what+"失败:"+err.Error())
				log.Warn(what+"失败", "err", err)
				a.toast = what + "失败:" + friendlyGitErr(err.Error())
				if what == "推送" || what == "同步" || what == "提交并同步" {
					a.notePushFail(err)
				}
				if what == "拉取并合并" && (strings.Contains(err.Error(), "冲突") || strings.Contains(strings.ToLower(err.Error()), "conflict")) {
					a.sc.merging = true
				}
				if what == "拉取并变基" {
					a.notePushFail(err)
					if strings.Contains(err.Error(), "冲突") || strings.Contains(strings.ToLower(err.Error()), "conflict") {
						a.sc.rebase = true
					}
				}
			} else {
				a.outAppend(outGit, what+"完成")
				log.Info(what)
				for _, f := range onOK {
					f()
				}
				if what == "推送" || what == "同步" || what == "提交并同步" || what == "强制推送" {
					a.clearPushGuide()
				}
			}
			a.branch = repo.Current()
			a.sc.last = time.Time{} // 立即重读
			a.sc.loading = false
			a.sc.gen++ // 作废进行中的旧刷新
		}
	})
}

func (a *App) scmStage(paths ...string) {
	a.scmAct("暂存", func(r vcs.Repo) error { return r.Stage(paths...) })
}
func (a *App) scmUnstage(paths ...string) {
	a.scmAct("取消暂存", func(r vcs.Repo) error { return r.Unstage(paths...) })
}
func (a *App) scmDiscard(p string) {
	a.scmAct("放弃更改", func(r vcs.Repo) error { return r.Discard(p) })
}

// scmCommit 提交暂存区;push 为真时提交后接着推送。
// scmCommitSync 提交后拉取再推送(提交并同步)。
func (a *App) scmCommitSync() {
	s := &a.sc
	staged, _ := s.split()
	msg := strings.TrimSpace(s.msg)
	if msg == "" && !s.amend {
		a.toast = "请先填写提交说明"
		return
	}
	if len(staged) == 0 && !s.amend {
		a.toast = "没有暂存的更改"
		return
	}
	amend := s.amend
	a.scmAct("提交并同步", func(r vcs.Repo) error {
		if err := r.Commit(msg, amend); err != nil {
			return err
		}
		if err := r.Pull(); err != nil {
			return err
		}
		return r.Push()
	}, func() {
		a.sc.msg, a.sc.amend, a.sc.commitErr = "", false, ""
		a.wizardAfterCommit()
	})
}

func (a *App) scmCommit(push bool) {
	s := &a.sc
	staged, _ := s.split()
	msg := strings.TrimSpace(s.msg)
	switch {
	case msg == "" && !s.amend:
		a.toast = "请先填写提交说明"
		return
	case len(staged) == 0 && !s.amend:
		a.toast = "没有暂存的更改:先在「更改」里点 + 暂存"
		return
	}
	amend := s.amend
	a.scmAct("提交", func(r vcs.Repo) error {
		if err := r.Commit(msg, amend); err != nil {
			return err
		}
		if push {
			return r.Push()
		}
		return nil
	}, func() {
		a.sc.msg, a.sc.amend, a.sc.commitErr = "", false, "" // 成功后才清空输入
		a.wizardAfterCommit()
	})
}

func (a *App) scmSync(pull, push bool) {
	what := map[[2]bool]string{{true, false}: "拉取", {false, true}: "推送", {true, true}: "同步"}[[2]bool{pull, push}]
	a.scmAct(what, func(r vcs.Repo) error {
		if pull {
			if err := r.Pull(); err != nil {
				return err
			}
		}
		if push {
			return r.Push()
		}
		return nil
	})
}

func draftCommitMsg(cs []vcs.Change) string {
	if len(cs) == 0 {
		return ""
	}
	typ, onlyTests, onlyDocs, anyNew, anyDel := "fix", true, true, false, false
	var names []string
	scope := ""
	for _, c := range cs {
		base := path.Base(c.Path)
		names = append(names, base)
		isTest := strings.Contains(c.Path, "test") || strings.HasSuffix(base, "_test.go")
		isDoc := strings.HasSuffix(base, ".md") || strings.HasPrefix(c.Path, "docs/")
		onlyTests = onlyTests && isTest
		onlyDocs = onlyDocs && isDoc
		anyNew = anyNew || c.Status == 'A' || c.Status == 'U'
		anyDel = anyDel || c.Status == 'D'
		if !isTest && !isDoc && scope == "" {
			scope = strings.TrimSuffix(base, path.Ext(base))
		}
	}
	switch {
	case onlyDocs:
		typ = "docs"
	case onlyTests:
		typ = "test"
	case anyNew && !anyDel && len(cs) == 1:
		typ = "feat"
	}
	verb := "更新"
	if anyNew && len(cs) == 1 {
		verb = "新增"
	}
	if len(names) > 3 {
		names = append(names[:3:3], "等 "+itoa(len(cs))+" 个文件")
	}
	head := typ
	if scope != "" && typ != "docs" && typ != "test" {
		head += "(" + scope + ")"
	}
	return head + ": " + verb + " " + strings.Join(names, "、")
}

// scmCommitFiles 取某次提交涉及的文件(缓存;后台读)。
func (a *App) scmCommitFiles(hash string) {
	s := &a.sc
	if s.files == nil {
		s.files = map[string][]vcs.Change{}
	}
	if _, ok := s.files[hash]; ok || a.repo == nil {
		return
	}
	s.files[hash] = nil // 占位,避免重复读
	repo := a.repo
	a.bg("scm.files", func() func() {
		fs, err := repo.CommitFiles(hash)
		return func() {
			if repo != a.repo {
				return
			}
			if err != nil {
				delete(a.sc.files, hash)
				return
			}
			a.sc.files[hash] = fs
		}
	})
}

// scmOpenCommitFile 在编辑区打开某次提交里某个文件的对比(父提交 ↔ 该提交)。
func (a *App) scmOpenCommitFile(c vcs.Commit, p string, pin bool) {
	repo := a.repo
	if repo == nil {
		return
	}
	parent := ""
	if len(c.Parents) > 0 {
		parent = c.Parents[0]
	}
	a.bg("scm.diff", func() func() {
		nw, _ := repo.Show(c.Hash, p)
		old := ""
		if parent != "" {
			old, _ = repo.Show(parent, p)
		}
		return func() { a.ide.openRevDiff(c.Short(), p, old, nw, pin) }
	})
}

// OpenSCM 打开左栏「源代码管理」(⌃⇧G)。
func (a *App) OpenSCM() {
	if a.mode != ModeIDE {
		a.SetMode(ModeIDE)
	}
	a.nav = "scm"
	a.sideOpen = true
}

// SetSCM 直接设置提交框与展开的提交(测试 / 截图用)。
func (a *App) SetSCM(msg, expanded string) {
	a.sc.msg = msg
	if expanded != "" {
		a.sc.expanded = expanded
		a.scmCommitFiles(expanded)
	}
}

// 储藏与提交上的操作(都走 scmAct:后台执行、完成后提示并刷新)。
func (a *App) scmStashPush() {
	a.scmAct("储藏", func(r vcs.Repo) error { return r.StashPush("") }, func() { a.toast = "已储藏全部更改" })
}
func (a *App) scmStashApply(i int, drop bool) {
	what := map[bool]string{true: "弹出储藏", false: "应用储藏"}[drop]
	a.scmAct(what, func(r vcs.Repo) error { return r.StashApply(i, drop) })
}
func (a *App) scmStashDrop(i int) {
	a.scmAct("删除储藏", func(r vcs.Repo) error { return r.StashDrop(i) })
}
func (a *App) scmUndoCommit() {
	a.scmAct("撤销上次提交", func(r vcs.Repo) error { return r.UndoCommit() }, func() { a.toast = "已撤销上次提交,改动保留在暂存区" })
}
func (a *App) scmCommitOp(cm vcs.Commit, op string) {
	h := cm.Hash
	switch op {
	case "checkout":
		a.scmAct("检出提交", func(r vcs.Repo) error { return r.Checkout(h) }, func() { a.toast = "已检出 " + cm.Short() + "(游离 HEAD)" })
	case "pick":
		a.scmAct("拣选", func(r vcs.Repo) error { return r.CherryPick(h) })
	case "revert":
		a.scmAct("还原提交", func(r vcs.Repo) error { return r.RevertCommit(h) })
	case "branch":
		a.sc.createAt = h
		a.OpenQuick("branch", "")
	case "copy":
		a.copyText = cm.Hash
		a.toast = "已复制哈希 " + cm.Short()
	}
}
