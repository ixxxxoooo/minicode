package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// changesCard 是回合末尾的「改动卡」(对标 Cursor):左上「N 个文件已改动」,右上纯文字「撤销 / 审查」;
// 每个文件一行(类型图标 + 文件名,右侧是该文件的 +增 −删),整行点击 = 打开文件;
// 超过 3 个折成「··· 还有 N 个文件」,点开就地展开,再点收起。入口「审查」只是跳转,
// 逐块处理的家是工作区的「审查」标签。
func (a *App) changesCard(c *ui.Context) {
	t := a.tk
	files := a.changedFiles()
	if len(files) == 0 {
		return
	}
	box := ui.Column(c).Key("chg-card").Background(t.Raised).Radius(t.RMd()).Border(1, t.Bd2).Padding(3).Margin(2, 0)
	kit.Lift(box)
	box.Children(func() {
		ui.Row(c).Height(theme.Px(28)).PaddingX(8).Gap(2).AlignItems(ui.Center).Children(func() {
			ui.Text(c, itoa(len(files))+" 个文件已改动").FontSize(theme.FsCap).FontWeight(500).TextColor(t.Tx2).SingleLine().Grow(1).MinWidth(0)
			a.changesCardUndoBtn(c)
			a.chgLink(c, "chg-review", "审查", t.Tx2, a.OpenReview)
		})
		shown := files
		if !a.rv.chgAll && len(files) > chgCardMax {
			shown = files[:chgCardMax]
		}
		narrow := a.chatTier() == tierNarrow // 窄档:只留「N 个文件已改动 + 审查」,文件列表在审查页里看
		if !narrow {
			for i, f := range shown {
				a.changesFileRow(c, i, f)
			}
		}
		if n := len(files) - chgCardMax; n > 0 && !narrow {
			r := kit.RowEl(c, false).Key("chg-more").Height(theme.Px(26)).Gap(8).Cursor(ui.CursorPointer)
			if r.Clicked() {
				a.rv.chgAll = !a.rv.chgAll
			}
			r.Children(func() {
				kit.Ic(c, "more", kit.S12).TextColor(t.Tx4)
				txt := "还有 " + itoa(n) + " 个文件"
				if a.rv.chgAll {
					txt = "收起"
				}
				ui.Text(c, txt).FontSize(theme.FsCap).SingleLine()
			})
		}
	})
	_ = core.PhaseRunning
}

// chgCardMax 是改动卡折叠时直接列出的文件数。
const chgCardMax = 3

// chgLink 是标题行右侧的纯文字链接(悬停才出现浅底),颜色默认 col,悬停变主文字色。
func (a *App) chgLink(c *ui.Context, key, text string, col ui.Color, on func()) {
	t := a.tk
	b := ui.ButtonBase(c).Key(key).Height(theme.Px(22)).PaddingX(7).Radius(t.RSm()).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Transition(kit.Fade)
	if b.Hovered() {
		b.Background(t.Fill1)
		col = t.Tx1
	}
	b.OnClick(on)
	b.Children(func() {
		ui.Text(c, text).FontSize(theme.FsCap).TextColor(col).SingleLine()
	})
}

// changesFileRow 是改动卡里的一个文件:文件类型图标 + 文件名 + 右侧该文件的增删行数,
// 整行可点 = 打开这个文件(定位到第一处改动)。目录不写在行里,悬停提示给完整路径。
func (a *App) changesFileRow(c *ui.Context, i int, f chFile) {
	t := a.tk
	name := baseName(f.path)
	r := kit.RowEl(c, false).Key("chg-f-" + itoa(i)).Height(theme.Px(26)).Gap(8).Cursor(ui.CursorPointer)
	if a.ide != nil && (a.ide.active == a.wsRel(f.path) || a.ide.sel == a.wsRel(f.path)) {
		// 正在编辑器里看这个文件:选中外观。改动卡本身就是白底卡,方案 B 的白色浮起卡看不出来,这里统一用强调色淡染底
		r.Background(t.Sel).FontWeight(500).TextColor(t.Tx1)
	} else if r.Hovered() {
		r.Background(t.Fill2) // 悬停:与列表行一致的 8.5% 墨色
	}
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
		if f.add > 0 {
			ui.Text(c, "+"+itoa(f.add)).Font(theme.CodeFont()).FontSize(theme.FsCap).TextColor(t.Ok).Shrink(0)
		}
		if f.del > 0 {
			ui.Text(c, "−"+itoa(f.del)).Font(theme.CodeFont()).FontSize(theme.FsCap).TextColor(t.Err).Shrink(0)
		}
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
