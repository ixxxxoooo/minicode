package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

const pScmExpandAll = "scm.expandAll"

func (a *App) scmExpandAll() bool { return a.prefs.Bool(pScmExpandAll, false) }

func (a *App) toggleScmExpandAll() {
	on := !a.scmExpandAll()
	a.prefs.Set(pScmExpandAll, on)
	a.toast = map[bool]string{true: "源代码管理:全部展开", false: "源代码管理:活动仓库"}[on]
	log.Info("切换多仓 SCM 模式", "expandAll", on)
}

// repoOverviewMenu 总览标题行「···」:全部展开 / 活动仓库与批量同步。
func (a *App) repoOverviewMenu(m *ui.Menu) {
	exp := a.scmExpandAll()
	lbl := "全部展开"
	if exp {
		lbl = "活动仓库模式"
	}
	if menuItem(m, lbl).Checked(exp).Chosen() {
		a.toggleScmExpandAll()
	}
	m.Separator()
	if menuItem(m, "全部拉取").Chosen() {
		a.syncAllRepos(true)
	}
	if menuItem(m, "全部同步").Chosen() {
		a.syncAllRepos(false)
	}
}

// scmExpandAllView 「全部展开」:每个仓库一块完整 SCM(设置页偏好 scm.expandAll,默认关)。
func (a *App) scmExpandAllView(c *ui.Context) {
	t := a.tk
	if a.sc.folded == nil {
		a.sc.folded = map[string]bool{}
	}
	ui.Column(c).Grow(1).MinHeight(0).Children(func() {
		ui.Row(c).Height(theme.Px(24)).PaddingX(4).Gap(2).AlignItems(ui.Center).Shrink(0).Children(func() {
			ui.Text(c, "仓库").Grow(1).FontSize(theme.FsMicro).FontWeight(700).TextColor(t.Tx3)
			ui.Text(c, "全部展开").FontSize(theme.FsMicro).TextColor(t.Ac).Shrink(0)
			kit.Ib(c, "more", kit.IbOpt{Small: true, Tip: "更多"}).Menu(a.repoOverviewMenu)
		})
		kit.VScroll(c).Key("scm-expand").Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Padding(0, 0, 8, 0).Gap(10).Children(func() {
				for _, i := range a.repoOrder() {
					i := i
					e := a.repos[i]
					if e.repo == nil {
						continue
					}
					a.scmExpandBlock(c, i)
				}
			})
		})
	})
}

func (a *App) scmExpandBlock(c *ui.Context, i int) {
	t := a.tk
	e := a.repos[i]
	br, ah, bh, n := a.repoSummary(i)
	ui.Column(c).Key("scm-bl-"+itoa(i)).Gap(4).Radius(t.RMd()).Border(1, t.Bd1).Padding(4, 6, 8, 6).Children(func() {
		r := kit.RowEl(c, i == a.repoAct).Height(theme.Px(32)).Gap(8).PaddingX(4)
		r.OnClick(func() { a.setActiveRepo(i) })
		r.Children(func() {
			kit.Ic(c, "branch", kit.S12).TextColor(t.Tx3)
			ui.Text(c, e.name).FontWeight(700).Shrink(0)
			ui.Text(c, br).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Shrink(1)
			if ah > 0 {
				ui.Text(c, "↑"+itoa(ah)).FontSize(theme.FsMicro).TextColor(t.AcTx).Shrink(0)
			}
			if bh > 0 {
				ui.Text(c, "↓"+itoa(bh)).FontSize(theme.FsMicro).TextColor(t.Tx3).Shrink(0)
			}
			kit.Cnt(c, itoa(n), false)
			if i == a.repoAct {
				kit.Tag(c, "活动", kit.KAc)
			}
		})
		restore := a.pushScmRepo(i)
		defer restore()
		if i == a.repoAct {
			a.wizardBar(c)
		}
		a.scmCommitBox(c, a.scmStagedCount())
		a.scmExpandBody(c)
	})
}

func (a *App) scmStagedCount() int {
	st, _ := a.sc.split()
	return len(st)
}

func (a *App) scmExpandBody(c *ui.Context) {
	s := &a.sc
	staged, work := s.split()
	var conf []vcs.Change
	for i := 0; i < len(work); {
		if work[i].Status == vcs.StatusConflict {
			conf = append(conf, work[i])
			work = append(work[:i:i], work[i+1:]...)
			continue
		}
		i++
	}
	if len(conf) > 0 {
		a.scmGroup(c, "merge", "合并冲突", conf, "", "")
	}
	if len(staged) > 0 {
		a.scmGroup(c, "staged", "暂存的更改", staged, "minus", "全部取消暂存")
	}
	a.scmGroup(c, "changes", "更改", work, "plus", "全部暂存")
	if len(staged)+len(work)+len(conf) == 0 && s.loaded {
		ui.Text(c, "没有未提交的更改").FontSize(theme.FsCap).TextColor(a.tk.Tx4).Padding(6, 8)
	}
}

// pushScmRepo 临时把源代码管理切到第 i 个仓库,返回还原函数(只用于全部展开模式的渲染)。
func (a *App) pushScmRepo(i int) func() {
	if !a.multiRoot() || i < 0 || i >= len(a.repos) {
		return func() {}
	}
	if i == a.repoAct {
		a.scmTick()
		return func() {}
	}
	act := a.repoAct
	cur := a.repos[act]
	if a.sc.loaded {
		cur.msg = a.sc.msg
		cur.changes = a.sc.changes
		cur.ahead, cur.behind, cur.branch, cur.loaded = a.sc.ahead, a.sc.behind, a.branch, true
	}
	e := a.repos[i]
	sc0 := a.sc
	repo0 := a.repo
	br0 := a.branch
	a.repoAct = i
	a.sc = scmState{
		msg: e.msg, changes: append([]vcs.Change(nil), e.changes...),
		ahead: e.ahead, behind: e.behind, loaded: e.loaded,
		folded: a.sc.folded, gen: sc0.gen,
	}
	a.SetRepo(e.repo)
	a.branch = e.branch
	if !e.loaded && e.repo != nil {
		end := diag.SpanSlow("scm", "展开块读状态", 300*time.Millisecond)
		cs, _ := e.repo.Status()
		ah, bh := e.repo.AheadBehind()
		end()
		a.sc.changes, a.sc.ahead, a.sc.behind, a.sc.loaded = cs, ah, bh, true
		e.changes, e.ahead, e.behind, e.loaded = cs, ah, bh, true
	}
	return func() {
		e.msg = a.sc.msg
		e.changes = append([]vcs.Change(nil), a.sc.changes...)
		e.ahead, e.behind, e.branch, e.loaded = a.sc.ahead, a.sc.behind, a.branch, a.sc.loaded
		a.repoAct = act
		a.sc = sc0
		a.SetRepo(repo0)
		a.branch = br0
	}
}
