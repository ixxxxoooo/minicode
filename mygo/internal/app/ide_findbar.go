package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 编辑器内查找 / 替换条:用应用自己的配色与中文,替换行用左边的「展开」箭头折起 / 展开(设计稿:替换行默认折起)。
// 只读文件不能替换:箭头置灰,条里直接给「解除只读」入口,点了就能替换。
// 库的 FindBar 负责匹配、跳转与替换本身(含正则 $1 展开、全部替换算一步撤销)。

func (a *App) findBar(c *ui.Context, tb *ideTab, bar ui.Element, f *editor.FindBar) {
	t := a.tk
	bar.Gap(4).Padding(6).Background(t.Raised).Border(1, t.Bd2).Radius(t.RMd())
	for _, s := range t.Shadow2() {
		bar.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	field := func(value *string, ph string, w float32) ui.Element {
		return ui.TextInputBase(c, value).Width(w).Placeholder(ph).Padding(3, 8).Background(t.BG).Border(1, t.Bd2).
			Radius(t.RSm()).FontSize(theme.FsCap + 1).Label(ph).TextColor(t.Tx1)
	}
	// icBtn 小图标按钮:不抢编辑器 / 输入框的焦点
	icBtn := func(icon, tip string, on bool) ui.Element {
		b := ui.ButtonBase(c).Size(22, 22).Radius(t.RSm()).Justify(ui.Center).AlignItems(ui.Center).Tooltip(tip).Label(tip).KeepFocus().Cursor(ui.CursorPointer)
		col := t.Tx3
		if on {
			b.Background(t.Fill2)
			col = t.Tx1
		} else if b.Hovered() {
			b.Background(t.Fill1)
			col = t.Tx1
		}
		b.Children(func() { kit.Ic(c, icon, kit.S12).TextColor(col) })
		return b
	}
	txtBtn := func(label, tip string, primary bool) ui.Element {
		b := ui.ButtonBase(c).Height(22).PaddingX(8).Radius(t.RSm()).Justify(ui.Center).AlignItems(ui.Center).Tooltip(tip).Label(tip).KeepFocus().Cursor(ui.CursorPointer)
		bg, col := t.Fill1, t.Tx2
		if primary {
			bg, col = t.Fill2, t.Tx1
		}
		if b.Hovered() {
			bg = t.Fill2
		}
		b.Background(bg)
		b.Children(func() { ui.Text(c, label).FontSize(theme.FsCap).TextColor(col) })
		return b
	}
	toggle := func(on *bool, label, tip string) {
		b := ui.ToggleBase(c, on).Size(24, 22).Radius(t.RSm()).Justify(ui.Center).AlignItems(ui.Center).Tooltip(tip).Label(tip).KeepFocus()
		col := t.Tx3
		if *on {
			b.Background(t.AcSoft)
			col = t.AcTx
		} else if b.Hovered() {
			b.Background(t.Fill1)
		}
		b.Children(func() { ui.Text(c, label).FontSize(theme.FsCap).TextColor(col).FontFeatures("tnum") })
	}

	ro := !f.CanReplace()
	bar.Children(func() {
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			// 展开 / 折起替换行
			arrow := "chevR"
			if f.Replacing() {
				arrow = "chevD"
			}
			tip := "显示替换 (⌥⌘F)"
			if f.Replacing() {
				tip = "收起替换"
			}
			if ro {
				tip = "只读文件不能替换,先解除只读"
			}
			ab := icBtn(arrow, tip, f.Replacing())
			if ro {
				ab.Opacity(.4)
			} else if ab.Clicked() {
				f.SetReplacing(!f.Replacing())
			}
			f.QueryInput(field(f.Query(), "查找", 200))
			toggle(f.CaseSensitive(), "Aa", "区分大小写")
			toggle(f.WholeWord(), "ab", "全词匹配")
			toggle(f.Regexp(), ".*", "正则表达式")
			toggle(f.InSelection(), "[ ]", "仅在选区内查找")
			st := f.Status()
			col := t.Tx3
			if f.Err() != nil {
				st, col = "正则无效", t.Err
			}
			ui.Text(c, st).FontSize(theme.FsCap).TextColor(col).FontFeatures("tnum").MinWidth(56)
			if icBtn("chevU", "上一个 (⇧↩)", false).Clicked() {
				f.Previous()
			}
			if icBtn("chevD", "下一个 (↩)", false).Clicked() {
				f.Next()
			}
			if icBtn("x", "关闭 (Esc)", false).Clicked() {
				f.Close()
			}
		})
		if ro && tb != nil && tb.diskRO { // 只读的项目外文件:直接给解除只读的入口,替换就能用了
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "只读文件,不能替换").FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1)
				if txtBtn("解除只读以替换", "解除只读后可编辑、替换并保存", true).Clicked() {
					a.unlockReadOnly(tb)
				}
			})
		}
		if f.Replacing() {
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Width(22).Shrink(0)
				f.ReplacementInput(field(f.Replacement(), "替换为", 200))
				if txtBtn("替换", "替换当前匹配并跳到下一个 (↩)", false).Clicked() {
					f.Replace()
				}
				if txtBtn("全部替换", "替换全部匹配,算一步撤销 (⌘↩)", true).Clicked() {
					f.ReplaceAll()
				}
			})
		}
	})
}
