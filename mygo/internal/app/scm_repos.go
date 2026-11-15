package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// repoOverview 是多仓库时源代码管理顶部的「仓库」总览(设计稿「多仓库 Git」页):
// 每个文件夹一行(分支 · 领先落后 · 改动数),点一行切换活动仓库;非仓库的文件夹带「初始化仓库」。
func (a *App) repoOverview(c *ui.Context) {
	t := a.tk
	ui.Column(c).Shrink(0).Padding(0, 0, 4, 0).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Margin(0, 0, 6, 0).Children(func() {
		ui.Row(c).Height(24).PaddingX(4).Gap(2).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "仓库").Grow(1).FontSize(theme.FsMicro).FontWeight(700).TextColor(t.Tx3)
			ui.Text(c, itoa(len(a.repos))).FontSize(theme.FsMicro).TextColor(t.Tx4).Margin(0, 4, 0, 0)
			kit.Ib(c, "download", kit.IbOpt{Small: true, Tip: "全部拉取"}).OnClick(func() { a.syncAllRepos(true) })
			kit.Ib(c, "sync", kit.IbOpt{Small: true, Tip: "全部同步(先拉取再推送)"}).OnClick(func() { a.syncAllRepos(false) })
		})
		for i, e := range a.repos {
			i, e := i, e
			if e.repo == nil && e.checked {
				r := kit.RowEl(c, false).Key("repo-" + itoa(i)).Height(30).Gap(6).TextColor(t.Tx3)
				on := i == a.repoAct
				if on {
					r.Background(t.Fill1)
				}
				r.OnClick(func() { a.setActiveRepo(i) })
				r.Children(func() {
					kit.Ic(c, "folder", kit.S12)
					ui.Text(c, e.name).Grow(1).SingleLine()
					kit.Btn(c, "初始化仓库", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.initRepo(i) })
				})
				continue
			}
			br, ah, bh, n := a.repoSummary(i)
			r := kit.RowEl(c, i == a.repoAct).Key("repo-" + itoa(i)).Height(32).Gap(6)
			r.OnClick(func() { a.setActiveRepo(i) })
			r.Children(func() {
				kit.Ic(c, "branch", kit.S12).TextColor(t.Tx3)
				ui.Row(c).Grow(1).MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, e.name).FontWeight(600).SingleLine().Shrink(0)
					ui.Text(c, br).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Shrink(1).MinWidth(0)
				})
				if ah > 0 {
					ui.Text(c, "↑"+itoa(ah)).FontSize(theme.FsMicro).TextColor(t.AcTx).FontFeatures("tnum").Shrink(0)
				}
				if bh > 0 {
					ui.Text(c, "↓"+itoa(bh)).FontSize(theme.FsMicro).TextColor(t.Tx3).FontFeatures("tnum").Shrink(0)
				}
				if n > 0 {
					kit.Cnt(c, itoa(n), false)
				}
			})
		}
	})
}

// syncAllRepos 对每个仓库依次拉取(pullOnly)或拉取 + 推送,汇总结果提示;失败的仓库单独点名。
func (a *App) syncAllRepos(pullOnly bool) {
	type job struct {
		name string
		repo vcs.Repo
	}
	var jobs []job
	for _, e := range a.repos {
		if e.repo != nil {
			jobs = append(jobs, job{e.name, e.repo})
		}
	}
	if len(jobs) == 0 {
		a.toast = "工作区里没有 Git 仓库"
		return
	}
	a.sc.busy = map[bool]string{true: "拉取", false: "同步"}[pullOnly]
	ws := a.ws
	a.bg("sync-all", func() func() {
		var failed []string
		for _, j := range jobs {
			if err := j.repo.Pull(); err != nil {
				failed = append(failed, j.name+"("+err.Error()+")")
				continue
			}
			if !pullOnly {
				if err := j.repo.Push(); err != nil {
					failed = append(failed, j.name+"("+err.Error()+")")
				}
			}
		}
		return func() {
			a.sc.busy = ""
			msg := "已" + map[bool]string{true: "拉取", false: "同步"}[pullOnly] + " " + itoa(len(jobs)-len(failed)) + " 个仓库"
			if len(failed) > 0 {
				msg += ",失败:" + failed[0]
				if len(failed) > 1 {
					msg += " 等 " + itoa(len(failed)) + " 个"
				}
			}
			a.toast = msg
			if a.ws == ws {
				for _, e := range a.repos {
					e.last = e.last.Add(-time.Hour) // 让总览尽快重读
				}
				a.sc.last = a.sc.last.Add(-scmRefreshEvery * 2)
			}
		}
	})
}
