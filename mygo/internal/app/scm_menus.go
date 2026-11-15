package app

import (
	"path"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// 源代码管理 / 图谱的右键菜单与背后的确认流程(设计稿 menus「SCM · 图谱 · 搜索 · 菜单的具体行为」)。

// scmDiscardAsk 放弃更改前先确认:悬停按钮与菜单走同一个入口,写明丢什么。
func (a *App) scmDiscardAsk(chs []vcs.Change) {
	if len(chs) == 0 {
		return
	}
	untracked := 0
	for _, c := range chs {
		if c.Status == 'U' {
			untracked++
		}
	}
	title := "放弃 " + path.Base(chs[0].Path) + " 的更改?"
	if len(chs) > 1 {
		title = "放弃 " + itoa(len(chs)) + " 个文件的更改?"
	}
	body := "工作区里的修改会恢复成暂存区 / HEAD 的样子,无法撤销。"
	if untracked > 0 {
		body += "其中 " + itoa(untracked) + " 个未跟踪文件会被删除。"
	}
	paths := make([]string, len(chs))
	for i, c := range chs {
		paths[i] = c.Path
	}
	a.confirmDlg(title, body, "放弃更改", true, func() {
		diag.L("scm").Info("放弃更改", "files", len(paths))
		a.scmAct("放弃更改", func(r vcs.Repo) error {
			for _, p := range paths {
				if err := r.Discard(p); err != nil {
					return err
				}
			}
			return nil
		})
	})
}

// scmFileMenu 是 SCM 文件行的右键菜单:打开更改 / 打开文件 / 暂存 / 放弃 / 历史 / 复制路径。
func (a *App) scmFileMenu(m *ui.Menu, grp string, ch vcs.Change) {
	if menuItem(m, "打开更改").Chosen() {
		a.scmOpenFile(grp, ch, true)
	}
	gone := ch.Status == 'D' && grp != "staged"
	if menuItem(m, "打开文件").Disabled(gone).Tooltip(map[bool]string{true: "文件已被删除"}[gone]).Chosen() {
		a.ide.openFile(a.treePath(ch.Path), true)
	}
	m.Separator()
	if grp == "staged" {
		if menuItem(m, "取消暂存").Chosen() {
			a.scmUnstage(ch.Path)
		}
	} else {
		if menuItem(m, "暂存更改").Disabled(grp == "merge").Tooltip(map[bool]string{true: "先解决冲突(打开更改)"}[grp == "merge"]).Chosen() {
			a.scmStage(ch.Path)
		}
		if menuItem(m, "放弃更改…").Danger().Disabled(grp == "merge").Tooltip(map[bool]string{true: "冲突文件请在合并视图里处理"}[grp == "merge"]).Chosen() {
			a.scmDiscardAsk([]vcs.Change{ch})
		}
	}
	m.Separator()
	if menuItem(m, "显示文件历史").Disabled(ch.Status == 'U' || ch.Status == 'A' && grp == "staged").Tooltip(map[bool]string{true: "新文件还没有历史"}[ch.Status == 'U' || ch.Status == 'A' && grp == "staged"]).Chosen() {
		a.ide.sel = a.treePath(ch.Path)
		a.ide.sections["时间线"] = true
		a.nav, a.sideOpen = "files", true
	}
	if menuItem(m, "复制路径").Chosen() {
		a.copyOut(a.absPath(a.treePath(ch.Path)), "路径")
	}
}

// scmGroupMenu 是分组标题(更改 / 暂存的更改)的右键菜单。
func (a *App) scmGroupMenu(m *ui.Menu, id string, cs []vcs.Change) {
	staged, work := a.sc.split()
	paths := func(l []vcs.Change) []string {
		out := make([]string, len(l))
		for i, c := range l {
			out[i] = c.Path
		}
		return out
	}
	none := func(n int) string { return map[bool]string{true: "没有可操作的文件"}[n == 0] }
	if menuItem(m, "全部暂存").Disabled(len(work) == 0).Tooltip(none(len(work))).Chosen() {
		a.scmStage(paths(work)...)
	}
	if menuItem(m, "全部取消暂存").Disabled(len(staged) == 0).Tooltip(none(len(staged))).Chosen() {
		a.scmUnstage(paths(staged)...)
	}
	var disc []vcs.Change // 只放弃「更改」分组里的,冲突文件与暂存区不动
	for _, c := range work {
		if c.Status != vcs.StatusConflict {
			disc = append(disc, c)
		}
	}
	if menuItem(m, "放弃全部更改…").Danger().Disabled(len(disc) == 0).Tooltip(none(len(disc))).Chosen() {
		a.scmDiscardAsk(disc)
	}
	m.Separator()
	if menuItem(m, "储藏所有更改").Disabled(len(a.sc.changes) == 0).Tooltip(none(len(a.sc.changes))).Chosen() {
		a.scmStashPush()
	}
	if menuItem(m, "刷新").Chosen() {
		a.sc.last = time.Time{}
	}
}

// commitMenu 是图谱 / SCM 里提交行的右键菜单。
func (a *App) commitMenu(m *ui.Menu, cm vcs.Commit) {
	// 当前提交(HEAD)上:检出 / 与 HEAD 比较都没有意义,置灰并写明原因(检出还会让 HEAD 游离)
	isHead := len(a.sc.log) > 0 && cm.Hash == a.sc.log[0].Hash
	headWhy := map[bool]string{true: "这就是当前提交(HEAD)", false: ""}[isHead]
	if menuItem(m, "检出此提交").Disabled(isHead).Tooltip(headWhy).Chosen() {
		a.scmCommitOp(cm, "checkout")
	}
	if menuItem(m, "从此提交创建分支…").Chosen() {
		a.scmCommitOp(cm, "branch")
	}
	m.Separator()
	if menuItem(m, "拣选(cherry-pick)").Chosen() {
		a.scmCommitOp(cm, "pick")
	}
	if menuItem(m, "还原(revert)").Chosen() {
		a.scmCommitOp(cm, "revert")
	}
	m.Separator()
	if menuItem(m, "与工作区比较…").Chosen() {
		a.scmCompareWork(cm)
	}
	if menuItem(m, "与 HEAD 比较…").Disabled(isHead).Tooltip(headWhy).Chosen() {
		a.scmCompareHead(cm)
	}
	picks := a.graphPicks()
	if menuItem(m, "比较这两个提交…").Disabled(len(picks) != 2).Tooltip(map[bool]string{true: "按住 ⌘ 再点一条提交以多选", false: ""}[len(picks) != 2]).Chosen() {
		a.scmCompareTwo(picks[0], picks[1])
	}
	if menuItem(m, "复制哈希").Chosen() {
		a.scmCommitOp(cm, "copy")
	}
	m.Separator()
	n, why := a.resetInfo(cm)
	if menuItem(m, "回退到这里…").Danger().Disabled(why != "").Tooltip(why).Chosen() {
		a.askReset(cm, n)
	}
}

// resetInfo 回退到 cm 会移除几个提交;不能回退时返回原因(游离 HEAD、已经在这里、不在当前分支上)。
func (a *App) resetInfo(cm vcs.Commit) (int, string) {
	s := &a.sc
	if len(s.log) == 0 || a.repo == nil {
		return 0, "还没有提交"
	}
	if isHexName(a.branch) {
		return 0, "游离 HEAD 上不能回退,先切到分支"
	}
	head := s.log[0].Hash
	if cm.Hash == head {
		return 0, "已经是当前提交"
	}
	byHash := map[string]vcs.Commit{}
	for _, c := range s.log {
		byHash[c.Hash] = c
	}
	reach := func(from string) map[string]bool {
		seen := map[string]bool{}
		stack := []string{from}
		for len(stack) > 0 {
			h := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if seen[h] {
				continue
			}
			seen[h] = true
			stack = append(stack, byHash[h].Parents...)
		}
		return seen
	}
	fromHead := reach(head)
	if !fromHead[cm.Hash] {
		return 0, "这个提交不在当前分支上"
	}
	fromTarget := reach(cm.Hash)
	n := 0
	for h := range fromHead {
		if !fromTarget[h] {
			n++
		}
	}
	return n, ""
}

func isHexName(s string) bool {
	if len(s) < 7 || len(s) > 40 {
		return false
	}
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

// askReset 确认后把当前分支回退到 cm(reset --keep:不丢工作区,冲突则中止)。
func (a *App) askReset(cm vcs.Commit, n int) {
	body := cm.Short() + " 之后的 " + itoa(n) + " 个提交会从 " + a.branch + " 上移除(仍可从 reflog 找回)。未提交的改动保留;如果与回退的内容冲突,操作会中止并告诉你原因。"
	if n > a.sc.ahead { // 要移除的比「领先上游」的多,说明含已推送的提交
		body += "\n其中含已推送的提交:回退后需要强制推送,本应用不提供强制推送。"
	}
	a.confirmDlg("把 "+a.branch+" 回退到 "+cm.Short()+"?", body, "回退", true, func() {
		diag.L("scm").Info("回退分支", "to", cm.Short(), "branch", a.branch, "removed", n)
		a.scmAct("回退", func(r vcs.Repo) error { return r.ResetKeep(cm.Hash) }, func() {
			a.sc.expanded = ""
			a.toast = "已把 " + a.branch + " 回退到 " + cm.Short()
		})
	})
}

// scmCompareWork 「与工作区比较」:先列出这个提交涉及的文件,选一个在编辑区开「提交 ↔ 工作区」差异。
func (a *App) scmCompareWork(cm vcs.Commit) {
	repo := a.repo
	if repo == nil {
		return
	}
	a.bg("scm.cmpfiles", func() func() {
		fs, err := repo.CommitFiles(cm.Hash)
		return func() {
			if repo != a.repo {
				return
			}
			switch {
			case err != nil:
				a.toast = "读取提交文件失败:" + err.Error()
			case len(fs) == 0:
				a.toast = "这个提交没有文件改动"
			default:
				a.qp = &qpState{kind: "cmpfiles", lastSel: -1, cmpC: cm, cmpFiles: fs}
			}
		}
	})
}

// openCompareWork 开「提交 ↔ 工作区」的只读差异标签(p 是仓库内路径)。
func (a *App) openCompareWork(cm vcs.Commit, p string) {
	repo := a.repo
	tp := a.treePath(p)
	id := "diff:cmp-" + cm.Short() + ":" + tp
	if t := a.ide.tab(id); t != nil {
		a.ide.active, a.ide.sel = id, tp
		return
	}
	a.bg("scm.cmpdiff", func() func() {
		old, _ := repo.Show(cm.Hash, p)
		cur, _ := a.ide.fs.Read(tp)
		return func() {
			a.ide.addDiffTab(id, tp, cm.Short()+" ↔ 工作区", old, cur, true)
			diag.L("scm").Info("与工作区比较", "commit", cm.Short(), "path", tp)
		}
	})
}

// 搜索结果的右键菜单:匹配行 / 文件行各一份。
func (a *App) searchMatchMenu(m *ui.Menu, mt searchMatch) {
	s := &a.sr
	if menuItem(m, "打开").Chosen() {
		a.ide.navMark()
		a.ide.openAt(mt.Path, mt.Line, mt.Col, mt.End, false)
		a.ide.sel = mt.Path
		a.ide.navPush(navPos{mt.Path, mt.Line, mt.Col})
	}
	if menuItem(m, "在侧边预览").Chosen() {
		a.searchPreviewSide(mt)
	}
	label := "忽略此结果"
	if s.ignored[mkey(mt)] {
		label = "恢复此结果"
	}
	if menuItem(m, label).Chosen() {
		k := mkey(mt)
		s.ignored[k] = !s.ignored[k]
		s.rows = s.flatten()
	}
	m.Separator()
	if menuItem(m, "复制路径与行号").Chosen() {
		a.copyOut(mt.Path+":"+itoa(mt.Line+1)+":"+itoa(mt.Col+1), "路径与行号")
	}
	if menuItem(m, "复制匹配行").Chosen() {
		a.copyOut(strings.TrimSpace(mt.Text), "匹配行")
	}
}

func (a *App) searchFileMenu(m *ui.Menu, p string) {
	s := &a.sr
	label := "折叠"
	if s.folded[p] {
		label = "展开"
	}
	if menuItem(m, label).Chosen() {
		s.folded[p] = !s.folded[p]
		s.rows = s.flatten()
	}
	if menuItem(m, "打开文件").Chosen() {
		a.ide.openFile(p, false)
	}
	if menuItem(m, "在文件树中定位").Chosen() {
		a.revealInTree(p)
	}
	m.Separator()
	if menuItem(m, "复制路径").Chosen() {
		a.copyOut(p, "路径")
	}
}
