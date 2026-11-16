package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 编辑器面包屑条右侧的工具(设计稿 ide-md):Markdown 的视图分段、自动换行、格式化、语言与编码。
// 自动换行与格式化对 diff 标签同样可用(见 ide_diff.go 顶部条)。

// toggleWrap 切换「自动换行」偏好(代码编辑器与差异视图共用,设置里同一个键)。
func (a *App) toggleWrap() {
	on := !a.prefs.Bool(pEdWrap, false)
	if on {
		a.prefs.Set(pEdWrap, true)
	} else {
		a.prefs.Reset(pEdWrap)
	}
	if on {
		a.toast = "已开启自动换行"
	} else {
		a.toast = "已关闭自动换行"
	}
	uxLog.Info("切换自动换行", "开", on)
	if tb := a.ide.cur(); tb != nil && tb.st != nil {
		a.applyEditorPrefs(tb)
	}
}

// crumbTools 画面包屑条右侧的一组工具。
func (a *App) crumbTools(c *ui.Context, tb *ideTab) {
	t := a.tk
	ui.Row(c).Shrink(0).Gap(2).AlignItems(ui.Center).Children(func() {
		md := tb.buf != nil && isMarkdownFile(tb.path)
		src := a.mdSource(tb)
		if md {
			on := 0
			if src {
				on = 1
			}
			if i := kit.Seg(c, []kit.SegItem{{Text: "预览", Tip: "渲染后的文档,可直接编辑"}, {Text: "源码", Tip: "显示 Markdown 原文"}}, on, kit.SegOpt{Key: "md-view"}); i >= 0 && i != on {
				a.setMdMode(tb, i == 1)
			}
			ui.Box(c).Size(6, 1)
		}
		if !md || src { // 所见即所得里文字本来就按宽度折行,换行开关只对源码有意义
			wrap := a.prefs.Bool(pEdWrap, false)
			tip := "自动换行 ⌥Z"
			if wrap {
				tip = "关闭自动换行 ⌥Z"
			}
			if kit.Ib(c, "wrap", kit.IbOpt{Small: true, On: wrap, Tip: tip}).Clicked() {
				a.toggleWrap()
			}
		}
		if tb.buf != nil && tb.virt == "" && (!md || src) {
			if kit.Ib(c, "align", kit.IbOpt{Small: true, Tip: "格式化文档 ⇧⌥F"}).Clicked() {
				a.formatDoc()
			}
		}
		if a.wsNarrowMM() { // 中档(<600)起收起面包屑右侧的语言 / 编码
			return
		}
		if l := langName(tb.path); l != "" {
			ui.Text(c, l).TextColor(t.Tx4).Margin(0, 0, 0, 6)
		}
		ui.Text(c, "UTF-8").TextColor(t.Tx4).Margin(0, 0, 0, 4)
	})
}
