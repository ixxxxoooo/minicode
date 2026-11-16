package app

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/projects"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 「编辑项目」对话框(设计稿「项目与工作区」页的项目行右键 → 编辑项目…):
// 改侧栏里的显示名、查看路径 / 仓库状态 / 所属工作区,并提供在访达中显示、复制路径、从列表移除。
// 显示名只存在 ~/.minicode/workspaces.json 的 names 里,不改文件夹,也不动 Claude 的会话文件。

// openProjDialog 打开某个项目的编辑对话框。
func (a *App) openProjDialog(p core.ProjectInfo) {
	a.pd = projDlg{kind: "proj", open: true, projPath: projects.Clean(p.Path), name: p.Name}
	log.Info("打开编辑项目", "path", a.pd.projPath)
}

// submitProject 保存显示名;与文件夹名相同或留空 = 恢复文件夹名。
func (a *App) submitProject() {
	d := &a.pd
	name := strings.TrimSpace(d.name)
	if len([]rune(name)) > 60 {
		d.err = "名称太长了(最多 60 个字符)"
		return
	}
	if err := a.registry().SetProjectName(d.projPath, name); err != nil {
		d.err = "保存失败:" + err.Error()
		return
	}
	a.closeProjDlg()
	if name == "" || name == filepath.Base(d.projPath) {
		a.toast = "已恢复为文件夹名"
	} else {
		a.toast = "已保存项目名称"
	}
}

// removeProjectAsk 从列表移除项目前的确认(只对手动添加的项目)。
func (a *App) removeProjectAsk(p core.ProjectInfo) {
	body := "只会从列表里移除这个项目,磁盘上的文件夹不会动。"
	if n := len(a.registry().WorkspacesOf(p.Path)); n > 0 {
		body = "它属于 " + itoa(n) + " 个工作区,移除只影响这个列表,工作区里的引用保留。\n" + body
	}
	a.confirmDlg("移除项目「"+p.Name+"」?", body, "移除", true, func() {
		_ = a.registry().RemoveProject(p.Path)
		a.toast = "已移除项目"
		log.Info("移除项目", "path", p.Path)
	})
}

func (a *App) projectDialog(c *ui.Context) {
	t := a.tk
	d := &a.pd
	base := filepath.Base(d.projPath)
	removable := a.registry().HasProject(d.projPath)
	a.dlgFrame(c, "proj", 460, "编辑项目", func() {
		ui.Text(c, "项目 = 一个文件夹。这里只改它在侧栏里显示的名字,不会重命名磁盘上的文件夹。").FontSize(theme.FsCap).TextColor(t.Tx3)
		ui.Text(c, "显示名称").FontSize(theme.FsCap).TextColor(t.Tx3).Margin(4, 0, 0, 0)
		ui.Row(c).Children(func() { a.dlgInput(c, "proj-name", &d.name, base, true, a.submitProject) })
		ui.Text(c, "留空 = 使用文件夹名「"+base+"」").FontSize(theme.FsMicro).TextColor(t.Tx4)

		ui.Text(c, "位置").FontSize(theme.FsCap).TextColor(t.Tx3).Margin(6, 0, 0, 0)
		ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(4, 8).Radius(t.RSm()).Background(t.Fill1).Children(func() {
			kit.Ic(c, "folder", kit.S12).TextColor(t.Tx3)
			ui.Text(c, tildePath(d.projPath)).Font("monospace").FontSize(theme.FsCap).TextColor(t.Tx2).SingleLine().Grow(1).Shrink(1)
			if a.isRepoDir(d.projPath) {
				kit.Tag(c, "Git 仓库", kit.KOk)
			} else {
				kit.Tag(c, "非仓库")
			}
			kit.Ib(c, "folder", kit.IbOpt{Small: true, Tip: "在访达中显示"}).OnClick(func() { a.revealInFinder(d.projPath) })
			kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制路径"}).OnClick(func() { a.copyOut(d.projPath, "路径") })
		})

		ws := a.registry().WorkspacesOf(d.projPath)
		ui.Text(c, "所属工作区").FontSize(theme.FsCap).TextColor(t.Tx3).Margin(6, 0, 0, 0)
		if len(ws) == 0 {
			ui.Text(c, "不属于任何工作区").FontSize(theme.FsCap).TextColor(t.Tx4)
		} else {
			ui.Row(c).Gap(6).Wrap().Children(func() {
				for _, w := range ws {
					w := w
					r := kit.RowEl(c, false).Key("proj-ws-" + w.ID).Height(theme.Px(24)).Gap(5).PaddingX(8).Radius(t.RSm()).Background(t.Fill1)
					r.OnClick(func() { a.openWsDialog(w.ID, "") })
					r.Children(func() {
						kit.Ic(c, "layers", kit.S12).TextColor(t.Tx3)
						ui.Text(c, w.Name).SingleLine()
					})
				}
			})
			ui.Text(c, "点工作区名可直接编辑它").FontSize(theme.FsMicro).TextColor(t.Tx4)
		}

		a.dlgMsg(c)
		ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(8, 0, 0, 0).Children(func() {
			if removable {
				kit.Btn(c, "从列表移除", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "trash", Tip: "只移出侧栏列表,不删除文件夹"}).OnClick(func() {
					p, ok := a.projectByPath(d.projPath)
					if !ok {
						p = core.ProjectInfo{Name: base, Path: d.projPath}
					}
					a.removeProjectAsk(p)
				})
			} else {
				ui.Text(c, "来自 Claude 会话记录,随会话存在,不能从列表移除").FontSize(theme.FsMicro).TextColor(t.Tx4).Shrink(1)
			}
			ui.Spacer(c).Grow(1)
			kit.Btn(c, "取消", kit.BtnOpt{Variant: kit.BtnGhost}).OnClick(a.cancelProjDlg)
			kit.Btn(c, "保存", kit.BtnOpt{Variant: kit.BtnPri}).OnClick(a.submitProject)
		})
	})
}
