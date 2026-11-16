package app

import (
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// scmNestedHints 子模块脏工作区提示与父仓库的指针变更说明。
func (a *App) scmNestedHints(c *ui.Context) {
	if !a.multiRoot() && len(a.repos) <= 1 {
		return
	}
	t := a.tk
	for i, e := range a.repos {
		if e.kind != "子模块" || e.repo == nil {
			continue
		}
		// 只读缓存:这里在构建界面里每帧都会走,绝不能同步起 git 子进程(此前每帧对每个子模块跑一次
		// git status -uall,多仓库工作区里主线程被 fork / wait4 占满,IDE 形态卡顿、拖动卡)。
		// 非活动仓库由 reposTick 在后台刷新;单根工程没有 reposTick,首次在后台读一次。
		ch := e.changes
		if i == a.repoAct {
			ch = a.sc.changes
		}
		if !e.loaded && !e.loading && i != a.repoAct {
			a.loadRepoEntryAsync(i, e)
		}
		if len(ch) == 0 {
			continue
		}
		ui.Row(c).Key("scm-sub-"+itoa(i)).Padding(4, 8).Gap(6).AlignItems(ui.Center).Background(t.WarnSoft).Radius(t.RSm()).Margin(0, 2, 4, 2).Children(func() {
			kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
			ui.Text(c, e.name+" 子模块有未提交的改动("+itoa(len(ch))+" 项)").FontSize(theme.FsCap).TextColor(t.Tx1).Grow(1)
			kit.Btn(c, "查看", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
				a.setActiveRepo(i)
			})
		})
	}
	for _, ch := range a.sc.changes {
		if a.isSubmodulePath(ch.Path) {
			a.scmSubmoduleRow(c, ch)
		}
	}
}

func (a *App) isSubmodulePath(p string) bool {
	for _, e := range a.repos {
		if e.kind != "子模块" {
			continue
		}
		if p == e.name || strings.HasPrefix(p, e.pre()+"/") || strings.HasPrefix(p, e.name+"/") {
			return true
		}
	}
	return false
}

// scmSubmoduleRow 父仓库中子模块指针变更条目。
func (a *App) scmSubmoduleRow(c *ui.Context, ch vcs.Change) {
	t := a.tk
	ui.Row(c).Key("scm-submod-"+ch.Path).Height(theme.Px(24)).PaddingX(8).Gap(6).AlignItems(ui.Center).Background(t.Fill1).Radius(t.RSm()).Margin(0, 2, 2, 2).Children(func() {
		a.fileIc(c, path.Base(ch.Path))
		ui.Text(c, ch.Path).FontWeight(500).SingleLine().Grow(1)
		ui.Text(c, "子模块指针变更").FontSize(theme.FsMicro).TextColor(t.Tx3)
		kit.Tag(c, "M", kit.KWarn)
	})
}
