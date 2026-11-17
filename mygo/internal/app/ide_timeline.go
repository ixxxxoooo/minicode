package app

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/vcs"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// timelineSubject 时间线跟随的源文件路径(差异 / 预览标签也算)。
func (a *App) timelineSubject() string {
	tb := a.ide.cur()
	if tb == nil || tb.virt != "" {
		return ""
	}
	if tb.diff {
		return tb.path
	}
	if tb.path != "" {
		return tb.path
	}
	return ""
}

// timelinePickHash 当前对比标签应对应高亮的时间线条目。
func (a *App) timelinePickHash() string {
	tb := a.ide.cur()
	if tb == nil {
		return ""
	}
	if tb.diff {
		switch tb.dk {
		case dkHead:
			return "__local__"
		case dkTimeline:
			return tb.revHash
		case dkWork, dkStaged:
			return ""
		}
		if strings.HasPrefix(tb.id, "diff:rev-") {
			if rest, ok := strings.CutPrefix(tb.id, "diff:rev-"); ok {
				if short, _, ok2 := strings.Cut(rest, ":"); ok2 {
					for _, cm := range a.sc.log {
						if cm.Short() == short {
							return cm.Hash
						}
					}
				}
			}
		}
	}
	return ""
}

func (a *App) gitInfoLimit(p string) int {
	n := 40
	if a.ide.tlMore != nil {
		if m := a.ide.tlMore[p]; m > n {
			n = m
		}
	}
	return n
}

func (a *App) timelineReload(p string) {
	delete(a.gi, p)
}

func (a *App) timelineLoadMore(p string) {
	if a.ide.tlMore == nil {
		a.ide.tlMore = map[string]int{}
	}
	a.ide.tlMore[p] = a.gitInfoLimit(p) + 40
	a.timelineReload(p)
}

// openTimelineCommit 时间线单击:预览对比(父提交 ↔ 该提交)。
func (a *App) openTimelineCommit(c vcs.Commit, p string, pin bool) {
	repo := a.repo
	if repo == nil {
		return
	}
	parent := ""
	if len(c.Parents) > 0 {
		parent = c.Parents[0]
	}
	hash, short := c.Hash, c.Short()
	a.bg("timeline.diff", func() func() {
		nw, _ := repo.Show(hash, p)
		old := ""
		if parent != "" {
			old, _ = repo.Show(parent, p)
		}
		label := short + " 的改动"
		return func() {
			a.finishTimelineDiff(hash, short, p, old, nw, label, pin)
			ideLog.Info("打开时间线对比", "path", p, "commit", short, "preview", !pin)
		}
	})
}

func (a *App) finishTimelineDiff(hash, short, p, old, nw, label string, pin bool) {
	a.ide.openTimelineDiff(hash, short, p, old, nw, label, pin)
	if t := a.ide.tab(timelineDiffID(p)); t != nil && t.dv != nil {
		a.configureDiff(t)
		a.syncDiffAnns(t)
	}
}

// openTimelineVsWork 与工作区对比(提交版本 ↔ 工作区)。
func (a *App) openTimelineVsWork(c vcs.Commit, p string, pin bool) {
	repo := a.repo
	if repo == nil {
		return
	}
	hash, short := c.Hash, c.Short()
	a.bg("timeline.diff", func() func() {
		old, _ := repo.Show(hash, p)
		nw, _ := a.ide.fs.Read(p)
		label := short + " ↔ 工作区"
		return func() { a.finishTimelineDiff(hash, short, p, old, nw, label, pin) }
	})
}

// openTimelineVsParent 与上一版本对比(同单击默认)。
func (a *App) openTimelineVsParent(c vcs.Commit, p string, pin bool) {
	a.openTimelineCommit(c, p, pin)
}

func (a *App) timelineViewInSCM(hash string) {
	a.OpenSCM()
	a.sc.expanded = hash
	a.scmCommitFiles(hash)
}

func gitInfoUntracked(p string, git map[string]string) bool {
	st := git[p]
	return st == "U" || st == "A"
}

func timelineErrText(p string, g *gitInfo, untracked bool) string {
	if g == nil {
		return "不在 Git 仓库里"
	}
	if g.err != nil {
		return "读取失败:" + g.err.Error()
	}
	if untracked && len(g.log) == 0 && !g.busy {
		return "未跟踪的文件还没有历史"
	}
	if len(g.log) == 0 {
		if g.busy {
			return "读取中…"
		}
		return "这个文件还没有提交历史"
	}
	return ""
}

func (a *App) timelineFileGit(p string) bool {
	if workspace.IsExternalPath(p) {
		return false
	}
	repo, _ := a.repoAt(p)
	return repo != nil && p != ""
}
