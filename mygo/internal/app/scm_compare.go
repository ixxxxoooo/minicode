package app

import (
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// scmCompareHead 「与 HEAD 比较」:列出该提交涉及文件,右侧为 HEAD 版本。
func (a *App) scmCompareHead(cm vcs.Commit) {
	repo := a.repo
	if repo == nil {
		return
	}
	a.bg("scm.cmphead", func() func() {
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
				a.qp = &qpState{kind: "cmphead", lastSel: -1, cmpC: cm, cmpFiles: fs}
			}
		}
	})
}

// scmCompareTwo 比较两个提交:列出二者之间的差异文件。
func (a *App) scmCompareTwo(aHash, bHash string) {
	repo := a.repo
	if repo == nil || aHash == "" || bHash == "" {
		return
	}
	if aHash == bHash {
		a.toast = "请选择两个不同的提交"
		return
	}
	a.graphTab.picks = []string{aHash, bHash}
	a.bg("scm.cmp2", func() func() {
		fs, err := repo.CompareCommits(aHash, bHash)
		return func() {
			if repo != a.repo {
				return
			}
			switch {
			case err != nil:
				a.toast = "比较提交失败:" + err.Error()
			case len(fs) == 0:
				a.toast = "两个提交之间没有文件差异"
			default:
				a.qp = &qpState{
					kind: "cmp2", lastSel: -1,
					cmpA:     vcs.Commit{Hash: aHash},
					cmpB:     vcs.Commit{Hash: bHash},
					cmpFiles: fs,
				}
			}
		}
	})
}

func (a *App) openCompareHead(cm vcs.Commit, p string) {
	repo := a.repo
	tp := a.treePath(p)
	id := "diff:cmphead-" + cm.Short() + ":" + tp
	if t := a.ide.tab(id); t != nil {
		a.ide.active, a.ide.sel = id, tp
		return
	}
	a.bg("scm.cmphead.diff", func() func() {
		old, _ := repo.Show(cm.Hash, p)
		head, _ := repo.Show("HEAD", p)
		return func() {
			a.ide.addDiffTab(id, tp, cm.Short()+" ↔ HEAD", old, head, true)
			diag.L("scm").Info("与 HEAD 比较", "commit", cm.Short(), "path", tp)
		}
	})
}

func (a *App) openCompareTwo(aHash, bHash, p string) {
	repo := a.repo
	tp := a.treePath(p)
	shortA, shortB := shortenHash(aHash), shortenHash(bHash)
	id := "diff:cmp-" + shortA + "-" + shortB + ":" + tp
	if t := a.ide.tab(id); t != nil {
		a.ide.active, a.ide.sel = id, tp
		return
	}
	a.bg("scm.cmp2.diff", func() func() {
		old, _ := repo.Show(aHash, p)
		nw, _ := repo.Show(bHash, p)
		return func() {
			label := shortA + " ↔ " + shortB
			a.ide.addDiffTab(id, tp, label, old, nw, true)
			diag.L("scm").Info("比较两个提交", "a", shortA, "b", shortB, "path", tp)
		}
	})
}

func shortenHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}
