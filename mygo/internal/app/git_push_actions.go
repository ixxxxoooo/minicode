package app

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

var pushLog = diag.L("scm.push")

func (a *App) scmPullRebase() {
	a.scmAct("拉取并变基", func(r vcs.Repo) error { return r.PullRebase() }, func() {
		a.sc.rebase = false
		a.toast = "已拉取并变基"
	})
}

func (a *App) scmPullMerge() {
	a.scmAct("拉取并合并", func(r vcs.Repo) error { return r.PullMerge() }, func() {
		a.toast = "已拉取并合并"
	})
}

func (a *App) scmRebaseAbort() {
	a.scmAct("中止变基", func(r vcs.Repo) error { return r.RebaseAbort() }, func() {
		a.sc.rebase = false
		a.clearPushGuide()
		a.toast = "已中止变基"
		pushLog.Info("中止变基")
	})
}

func (a *App) pushGuideHasConflict() bool {
	g := a.sc.pushGuide
	return a.sc.rebase || strings.Contains(g, "冲突") || strings.Contains(strings.ToLower(g), "conflict") || strings.Contains(g, "变基")
}
