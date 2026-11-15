package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// changesCard 是回合末尾的「改动卡」:这一轮改了几个文件、增删多少,入口「审查」只是跳转,
// 逐块处理的家是工作区的「审查」标签。
func (a *App) changesCard(c *ui.Context) {
	t := a.tk
	files := a.changedFiles()
	if len(files) == 0 {
		return
	}
	add, del := 0, 0
	for _, f := range files {
		add += f.add
		del += f.del
	}
	box := ui.Column(c).Key("chg-card").Background(t.Raised).Radius(t.RMd()).Padding(8, 11).Gap(2).Margin(2, 0)
	for _, s := range t.Shadow1() {
		box.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	box.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "diff", kit.S12).TextColor(t.Ac)
			ui.Text(c, itoa(len(files))+" 个文件已改动").FontWeight(600)
			ui.Text(c, "+"+itoa(add)).Font("monospace").FontSize(theme.FsCap).TextColor(t.Ok)
			ui.Text(c, "−"+itoa(del)).Font("monospace").FontSize(theme.FsCap).TextColor(t.Err)
			ui.Spacer(c).Grow(1)
			a.changesCardUndoBtn(c)
			kit.Btn(c, "审查", kit.BtnOpt{Variant: kit.BtnSoft, Small: true, Icon: "eye"}).OnClick(a.OpenReview)
		})
		for i, f := range files {
			if i >= 3 {
				break
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).PaddingStart(20).Children(func() {
				ui.Text(c, baseName(f.path)).FontSize(theme.FsCap).TextColor(t.Tx2).SingleLine().Shrink(0)
				ui.Text(c, trimSlash(dirOf(a.wsRel(f.path)))).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().Grow(1)
			})
		}
	})
	_ = core.PhaseRunning
}

// viewInPanel 在底部终端面板开一个只读标签,查看这条命令的输出。
func (a *App) viewInPanel(tb *core.TerminalBlock) {
	id := "cmd:" + tb.Command
	found := false
	for _, v := range a.ide.cmds {
		if v.id == id {
			found = true
		}
	}
	if !found {
		a.ide.cmds = append(a.ide.cmds, cmdView{id: id, title: tb.Command, output: tb.Output, exit: tb.ExitCode})
	}
	a.ide.termSel = id
	a.ide.bottomOpen, a.ide.bottomTab = true, "term"
	a.screen = ScreenWork
}
