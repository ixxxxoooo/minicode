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
			if i >= chgCardMax {
				break
			}
			f := f
			a.changesFileRow(c, i, f)
		}
		if n := len(files) - chgCardMax; n > 0 { // 超出的折叠成一行,点开 = 审查页(全部文件在那里)
			r := kit.RowEl(c, false).Key("chg-more").Height(22).Margin(0, 0, 0, 12).Gap(6).Cursor(ui.CursorPointer)
			if r.Clicked() {
				a.OpenReview()
			}
			r.Children(func() {
				ui.Box(c).Size(14, 14).Shrink(0) // 与上面文件行的图标对齐
				ui.Text(c, "还有 "+itoa(n)+" 个文件").FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
			})
		}
	})
	_ = core.PhaseRunning
}

// chgCardMax 是改动卡最多直接列出的文件数。
const chgCardMax = 3

// changesFileRow 是改动卡里的一个文件:文件类型图标 + 文件名,整行可点 = 打开这个文件(定位到第一处改动)。
// 目录不再写在行尾(太长、信息量低),悬停提示里给出完整路径。
func (a *App) changesFileRow(c *ui.Context, i int, f chFile) {
	name := baseName(f.path)
	r := kit.RowEl(c, false).Key("chg-f-"+itoa(i)).Height(22).Margin(0, 0, 0, 12).Gap(6).Cursor(ui.CursorPointer)
	r.Tooltip(a.shortPath(f.path) + " · 点击打开")
	if r.Clicked() {
		line := 0
		if len(f.blocks) > 0 {
			line = f.blocks[0].Line
		}
		log.Info("改动卡打开文件", "path", f.path, "line", line)
		if !a.openPathRef(joinRef(f.path, line)) {
			a.toast = "文件已不存在:" + name
		}
	}
	r.Children(func() {
		a.fileIc(c, name)
		ui.Text(c, name).FontSize(theme.FsCap).SingleLine().Grow(1).MinWidth(0)
	})
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
