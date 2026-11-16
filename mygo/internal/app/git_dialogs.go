package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// notePushFail 记录推送失败原因,供 SCM 顶部的引导条使用。
func (a *App) notePushFail(err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "non-fast-forward") || strings.Contains(msg, "rejected") || strings.Contains(msg, "拒绝") {
		a.sc.pushGuide = friendlyGitErr(msg)
		log.Warn("推送被拒", "err", msg)
		return
	}
}

func (a *App) clearPushGuide() { a.sc.pushGuide = "" }

// opWord 进行中的多步操作的中文名("" 表示没有)。
func (a *App) opWord() string {
	switch {
	case a.sc.merging:
		return "合并"
	case a.sc.rebase:
		return "变基"
	}
	return ""
}

// scmPushGuide 推送被拒后的引导条;合并 / 变基冲突进行中也在这里给出口(打开冲突 / 中止)。
func (a *App) scmPushGuide(c *ui.Context) {
	if a.sc.pushGuide == "" && !a.sc.rebase && !a.sc.merging {
		return
	}
	t := a.tk
	msg := a.sc.pushGuide
	title := "推送被拒绝"
	if msg == "" { // 没有被拒的推送,只是冲突还没处理完
		title = "正在" + a.opWord()
		msg = a.opWord() + "未完成:请解决冲突,或中止" + a.opWord() + "。"
	}
	ui.Column(c).Padding(6, 8).Gap(6).Background(t.WarnSoft).Radius(t.RSm()).Margin(0, 2, 4, 2).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
			ui.Text(c, title).FontWeight(600).TextColor(t.Tx1)
			ui.Text(c, msg).FontSize(theme.FsCap).TextColor(t.Tx2).Grow(1)
			kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭"}).OnClick(a.clearPushGuide)
		})
		busy := a.sc.busy != ""
		if a.sc.pushGuide != "" {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				kit.Btn(c, "拉取并变基", kit.BtnOpt{
					Small: true, Variant: kit.BtnPri, Icon: "download",
					Disabled: busy, Loading: busy && a.sc.busy == "拉取并变基",
				}).OnClick(func() {
					a.clearPushGuide()
					a.scmPullRebase()
				})
				kit.Btn(c, "拉取并合并", kit.BtnOpt{
					Small: true, Variant: kit.BtnSoft,
					Disabled: busy, Loading: busy && a.sc.busy == "拉取并合并",
				}).OnClick(func() {
					a.clearPushGuide()
					a.scmPullMerge()
				})
				kit.Btn(c, "强制推送(带租约)", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Disabled: busy}).OnClick(func() {
					a.confirmDlg("强制推送到远端?", "使用 --force-with-lease,仅当你确认没有人在基于远端独有提交工作时才继续。", "强制推送", true, func() {
						a.clearPushGuide()
						a.scmAct("强制推送", func(r vcs.Repo) error { return r.PushForce() }, nil)
					})
				})
			})
		}
		if a.pushGuideHasConflict() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				word := a.opWord()
				if word == "" {
					word = "变基"
				}
				ui.Text(c, word+"产生冲突:在源代码管理打开冲突文件,或中止"+word+"。").FontSize(theme.FsCap).TextColor(t.Tx2).Grow(1)
				kit.Btn(c, "打开合并冲突", kit.BtnOpt{Small: true, Icon: "merge"}).OnClick(func() {
					a.OpenSCM()
					for _, ch := range a.sc.changes {
						if ch.Status == vcs.StatusConflict {
							a.OpenWorkspaceTab(vConflict, a.treePath(ch.Path))
							return
						}
					}
				})
				if a.sc.merging {
					kit.Btn(c, "中止合并", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Disabled: busy, Loading: busy && a.sc.busy == "中止合并"}).OnClick(a.scmMergeAbort)
				} else {
					kit.Btn(c, "中止变基", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(a.scmRebaseAbort)
				}
			})
		}
	})
}

// switchBranchNamed 切换分支(脏工作区时先问储藏)。
func (a *App) switchBranchNamed(name string) {
	run := func() { a.gitOp("切换分支", "已切换到 "+name, func() error { return a.repo.Switch(name) }) }
	if a.repo == nil {
		return
	}
	if a.branchDirty() {
		a.pd = projDlg{kind: "branch-dirty", open: true, title: "工作区有未提交的更改", body: "切换分支前可以储藏、带着改动切换,或取消。", branchName: name, branchFn: run}
		return
	}
	run()
}

func (a *App) createBranchNamed(name string) {
	run := func() {
		if h := a.sc.createAt; h != "" {
			a.sc.createAt = ""
			a.scmAct("创建分支", func(r vcs.Repo) error { return r.CreateAt(name, h) }, func() { a.toast = "已从该提交创建并切换到 " + name })
			return
		}
		a.gitOp("创建分支", "已创建并切换到 "+name, func() error { return a.repo.Create(name) })
	}
	if a.repo == nil {
		return
	}
	if a.branchDirty() {
		a.pd = projDlg{kind: "branch-dirty", open: true, title: "工作区有未提交的更改", body: "创建并切换分支前可以储藏、带着改动继续,或取消。", branchName: name, branchFn: run}
		return
	}
	run()
}

func (a *App) branchDirty() bool {
	if a.repo == nil {
		return false
	}
	ch, err := a.repo.Status()
	if err != nil {
		return false
	}
	return len(ch) > 0
}

func (a *App) branchDirtyDialog(c *ui.Context) {
	d := &a.pd
	a.dlgFrame(c, "branch-dirty", 420, d.title, func() {
		t := a.tk
		ui.Text(c, d.body).TextColor(t.Tx2).FontSize(theme.FsBody).LineHeight(1.55)
		ui.Row(c).Gap(8).Margin(8, 0, 0, 0).Children(func() {
			kit.Btn(c, "取消", kit.BtnOpt{Variant: kit.BtnGhost}).OnClick(a.cancelProjDlg)
			kit.Btn(c, "带着改动切换", kit.BtnOpt{Variant: kit.BtnSoft}).OnClick(func() {
				fn := d.branchFn
				a.closeProjDlg()
				fn()
			})
			kit.Btn(c, "储藏并切换", kit.BtnOpt{Variant: kit.BtnPri, Icon: "layers"}).OnClick(func() {
				fn := d.branchFn
				a.scmAct("储藏", func(r vcs.Repo) error { return r.StashPush("") }, func() {
					a.closeProjDlg()
					fn()
				})
			})
		})
	})
}
