package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// repoOverview 是多仓库时源代码管理顶部的「仓库」总览(设计稿「多仓库 Git」页):
// 每个文件夹一行(分支 · 领先落后 · 改动数),点一行切换活动仓库;非仓库的文件夹带「初始化仓库」。
func (a *App) repoOverview(c *ui.Context) {
	t := a.tk
	if a.sc.folded == nil {
		a.sc.folded = map[string]bool{}
	}
	open := !a.sc.folded["repos"]
	ui.Column(c).Shrink(0).Padding(0, 0, 4, 0).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Margin(0, 0, 6, 0).Children(func() {
		ui.Row(c).Height(theme.Px(24)).PaddingX(4).Gap(2).AlignItems(ui.Center).Children(func() {
			tog := ui.ButtonBase(c).Key("scm-h-repos").Grow(1).Height(theme.Px(24)).Justify(ui.Start).Gap(4).AlignItems(ui.Center).
				TextColor(t.Tx3).FontSize(theme.FsMicro).FontWeight(500).Cursor(ui.CursorPointer).Tooltip(map[bool]string{true: "收起仓库", false: "展开仓库"}[open])
			tog.OnClick(func() { a.sc.folded["repos"] = !open })
			tog.Children(func() {
				kit.Ic(c, map[bool]string{true: "chevD", false: "chevR"}[open], kit.S12)
				ui.Text(c, "仓库")
			})
			ui.Text(c, itoa(len(a.repos))).FontSize(theme.FsMicro).TextColor(t.Tx4).Margin(0, 4, 0, 0)
			if a.scmExpandAll() {
				kit.Tag(c, "全部展开", kit.KAc)
			}
			kit.Ib(c, "download", kit.IbOpt{Small: true, Tip: "全部拉取"}).OnClick(func() { a.syncAllRepos(true) })
			kit.Ib(c, "sync", kit.IbOpt{Small: true, Tip: "全部同步(先拉取再推送)"}).OnClick(func() { a.syncAllRepos(false) })
			kit.Ib(c, "more", kit.IbOpt{Small: true, Tip: "仓库视图与批量操作"}).Menu(a.repoOverviewMenu)
		})
		if !open {
			return
		}
		// 仓库块固定在顶部:最多显示 scmRepoMaxRows 行,超出在块内滚动,不把下面的提交框挤出去
		kit.VScroll(c).Key("scm-repos").MaxHeight(scmRepoMaxRows*scmRepoRowH + 4).Children(func() {
			ui.Column(c).Children(func() { a.repoRows(c) })
		})
	})
}

// repoRows 画仓库总览里的各仓库行(嵌套仓库紧跟在父文件夹后面,缩进显示)。
func (a *App) repoRows(c *ui.Context) {
	t := a.tk
	{
		for _, i := range a.repoOrder() {
			i, e := i, a.repos[i]
			var indent float32
			if e.kind != "" {
				indent = 16
			}
			if e.repo == nil && e.checked {
				r := kit.RowEl(c, false).Key("repo-"+itoa(i)).Height(theme.Px(scmRepoRowH)).Gap(6).TextColor(t.Tx3).Margin(0, 0, 0, indent)
				on := i == a.repoAct
				if on {
					r.Background(t.Fill1)
				}
				r.OnClick(func() { a.setActiveRepo(i) })
				r.ContextMenu(func(m *ui.Menu) { a.repoRowMenu(m, i) })
				r.Children(func() {
					kit.Ic(c, "folder", kit.S12)
					ui.Text(c, e.name).Grow(1).SingleLine()
					kit.Btn(c, "初始化仓库", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.initRepo(i) })
				})
				continue
			}
			br, ah, bh, n := a.repoSummary(i)
			conf, staged := a.repoFlags(i)
			r := kit.RowEl(c, i == a.repoAct).Key("repo-"+itoa(i)).Height(theme.Px(scmRepoRowH)).Gap(6).Margin(0, 0, 0, indent)
			hov := r.Hovered()
			r.OnClick(func() { a.setActiveRepo(i) })
			r.ContextMenu(func(m *ui.Menu) { a.repoRowMenu(m, i) })
			r.Children(func() {
				kit.Ic(c, "branch", kit.S12).TextColor(t.Tx3)
				ui.Row(c).Grow(1).MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, e.name).FontWeight(500).SingleLine().Shrink(0)
					ui.Text(c, br).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Shrink(1).MinWidth(0)
					if e.kind != "" {
						kit.Tag(c, e.kind)
					}
				})
				switch {
				case e.busy != "": // 这一行正在拉取 / 推送 / 同步
					kit.Spin(c, t.Ac)
				case e.opErr != "": // 失败:一键重试,不影响其它仓库
					ui.Text(c, "失败").FontSize(theme.FsMicro).TextColor(t.Err).Shrink(0).Tooltip(e.opErr)
					kit.Btn(c, "重试", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.repoRetry(i) })
				case hov: // 悬停:这一行仓库自己的动作(与是否活动仓库无关)
					kit.Ib(c, "download", kit.IbOpt{Small: true, Tip: "拉取 " + e.name}).OnClick(func() { a.repoSync(i, true, false) })
					kit.Ib(c, "sync", kit.IbOpt{Small: true, Tip: "同步 " + e.name}).OnClick(func() { a.repoSync(i, true, true) })
					kit.Ib(c, "branch", kit.IbOpt{Small: true, Tip: "切换分支…"}).OnClick(func() { a.setActiveRepo(i); a.OpenQuick("branch", "") })
				default:
					if ah > 0 {
						ui.Text(c, "↑"+itoa(ah)).FontSize(theme.FsMicro).TextColor(t.AcTx).FontFeatures("tnum").Shrink(0)
					}
					if bh > 0 {
						ui.Text(c, "↓"+itoa(bh)).FontSize(theme.FsMicro).TextColor(t.Tx3).FontFeatures("tnum").Shrink(0)
					}
				}
				if conf > 0 { // 有合并冲突:红色徽标,「全部同步」会跳过这个仓库
					kit.Tag(c, "冲突 "+itoa(conf), kit.KErr)
				}
				if staged && i != a.repoAct { // 其它仓库里有已暂存、还没提交的改动
					ui.Box(c).Size(6, 6).Radius(3).Background(t.Ac).Tooltip("有已暂存、未提交的改动")
				}
				if n > 0 && !(hov && e.busy == "" && e.opErr == "") {
					kit.Cnt(c, itoa(n), false)
				}
			})
		}
	}
}

// repoFlags 总览行用:冲突文件数、是否有已暂存的改动(活动仓库取实时状态)。
func (a *App) repoFlags(i int) (conflicts int, staged bool) {
	e := a.repos[i]
	cs := e.changes
	if i == a.repoAct && a.sc.loaded {
		cs = a.sc.changes
	}
	for _, c := range cs {
		if c.Status == vcs.StatusConflict {
			conflicts++
		}
		if c.Staged {
			staged = true
		}
	}
	return
}

// repoOp 对第 i 个仓库后台执行一个操作:行上转圈,完成后清掉、失败留「重试」,不影响其它仓库。
// done 在主线程里、成功时调用。
func (a *App) repoOp(i int, what string, f func(r vcs.Repo) error, done func()) {
	if i < 0 || i >= len(a.repos) || a.repos[i].repo == nil || a.repos[i].busy != "" {
		return
	}
	e, ws := a.repos[i], a.ws
	e.busy, e.opErr, e.retry = what, "", ""
	repo := e.repo
	a.bg("repo-op", func() func() {
		end := diag.SpanSlow("scm", what+" "+e.name, 2*time.Second)
		err := f(repo)
		end()
		return func() {
			e.busy = ""
			if a.ws != ws {
				return
			}
			if err != nil {
				e.opErr, e.retry = err.Error(), what
				log.Warn(what+"失败", "repo", e.name, "err", err)
				a.outAppend(outGit, e.name+" "+what+"失败:"+err.Error())
			} else {
				log.Info(what, "repo", e.name)
				a.outAppend(outGit, e.name+" "+what+"完成")
				if done != nil {
					done()
				}
			}
			e.last = e.last.Add(-time.Hour) // 让总览尽快重读
			if i == a.repoAct {
				a.sc.last = a.sc.last.Add(-scmRefreshEvery * 2)
				a.branch = e.repo.Current()
			}
		}
	})
}

// repoSync 拉取 / 推送 / 同步某一个仓库。
func (a *App) repoSync(i int, pull, push bool) {
	what := map[[2]bool]string{{true, false}: "拉取", {false, true}: "推送", {true, true}: "同步"}[[2]bool{pull, push}]
	a.repoOp(i, what, func(r vcs.Repo) error {
		if pull {
			if err := r.Pull(); err != nil {
				return err
			}
		}
		if push {
			return r.Push()
		}
		return nil
	}, nil)
}

// repoRetry 重试失败的那一步。
func (a *App) repoRetry(i int) {
	if i < 0 || i >= len(a.repos) {
		return
	}
	switch a.repos[i].retry {
	case "拉取":
		a.repoSync(i, true, false)
	case "推送":
		a.repoSync(i, false, true)
	case "储藏":
		a.repoStash(i)
	default:
		a.repoSync(i, true, true)
	}
}

// repoStash 储藏某个仓库的所有更改。
func (a *App) repoStash(i int) {
	a.repoOp(i, "储藏", func(r vcs.Repo) error { return r.StashPush("") }, func() { a.toast = "已储藏 " + a.repos[i].name + " 的全部更改" })
}

// syncAllRepos 对每个仓库依次拉取(pullOnly)或拉取 + 推送;每行各自转圈,失败的行留「重试」不中断其余;
// 有合并冲突的仓库跳过并点名(设计稿:冲突阻止「全部同步」)。
func (a *App) syncAllRepos(pullOnly bool) {
	var order []int
	var skipped []string
	for i, e := range a.repos {
		if e.repo == nil {
			continue
		}
		if conf, _ := a.repoFlags(i); conf > 0 {
			skipped = append(skipped, e.name)
			continue
		}
		order = append(order, i)
	}
	if len(order) == 0 {
		a.toast = map[bool]string{true: "没有可拉取的仓库", false: "没有可同步的仓库"}[len(skipped) == 0] + map[bool]string{true: "", false: "(" + strings.Join(skipped, "、") + " 有合并冲突)"}[len(skipped) == 0]
		if len(skipped) == 0 {
			a.toast = "工作区里没有 Git 仓库"
		}
		return
	}
	what := map[bool]string{true: "拉取", false: "同步"}[pullOnly]
	log.Info("全部"+what, "repos", len(order), "skipped", len(skipped))
	ws := a.ws
	done, failed := 0, 0
	var next func(k int)
	next = func(k int) { // 一个接一个:上一个结束再开下一个,每行状态独立
		if a.ws != ws {
			return
		}
		if k >= len(order) {
			msg := "已" + what + " " + itoa(done) + " 个仓库"
			if failed > 0 {
				msg += "," + itoa(failed) + " 个失败(行上可重试)"
			}
			if len(skipped) > 0 {
				msg += ",跳过 " + strings.Join(skipped, "、") + "(有合并冲突)"
			}
			a.toast = msg
			return
		}
		i := order[k]
		e := a.repos[i]
		e.busy, e.opErr = what, ""
		repo := e.repo
		a.bg("sync-all", func() func() {
			err := repo.Pull()
			if err == nil && !pullOnly {
				err = repo.Push()
			}
			return func() {
				e.busy = ""
				if err != nil {
					e.opErr, e.retry = err.Error(), what
					failed++
					log.Warn("全部"+what+":失败", "repo", e.name, "err", err)
				} else {
					done++
				}
				e.last = e.last.Add(-time.Hour)
				if i == a.repoAct {
					a.sc.last = a.sc.last.Add(-scmRefreshEvery * 2)
				}
				next(k + 1)
			}
		})
	}
	next(0)
}

// repoRowMenu 是总览行的右键菜单(设计稿「多仓库 Git」:每个动作只作用于这一行的仓库)。
func (a *App) repoRowMenu(m *ui.Menu, i int) {
	e := a.repos[i]
	has := e.repo != nil
	why := map[bool]string{true: "这个文件夹还不是 Git 仓库"}[!has]
	if menuItem(m, "设为活动仓库").Disabled(i == a.repoAct).Chosen() {
		a.setActiveRepo(i)
	}
	m.Separator()
	if menuItem(m, "拉取").Disabled(!has).Tooltip(why).Chosen() {
		a.repoSync(i, true, false)
	}
	if menuItem(m, "推送").Disabled(!has).Tooltip(why).Chosen() {
		a.repoSync(i, false, true)
	}
	if menuItem(m, "同步").Disabled(!has).Tooltip(why).Chosen() {
		a.repoSync(i, true, true)
	}
	if menuItem(m, "切换分支…").Disabled(!has).Tooltip(why).Chosen() {
		a.setActiveRepo(i)
		a.OpenQuick("branch", "")
	}
	m.Separator()
	if menuItem(m, "在图谱中查看").Disabled(!has).Tooltip(why).Chosen() {
		a.setActiveRepo(i)
		a.openGraph()
	}
	if menuItem(m, "储藏所有更改").Disabled(!has).Tooltip(why).Chosen() {
		a.repoStash(i)
	}
	m.Separator()
	real := !a.demoWs()
	nd := map[bool]string{true: "演示模式没有真实目录"}[!real]
	if menuItem(m, "在访达中显示").Disabled(!real).Tooltip(nd).Chosen() {
		a.revealInFinder(e.dir)
	}
	if menuItem(m, "在此处打开终端").Disabled(!real).Tooltip(nd).Chosen() {
		a.openTermAt(e.name)
	}
	if menuItem(m, "复制路径").Chosen() {
		a.copyOut(e.dir, "路径")
	}
}
