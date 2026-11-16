package app

import (
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 设计稿 scroll-spec「▾ 全部标签」:弹层 = 图标 + 标题(+ 淡色的所在目录)、未保存圆点、当前标签打勾;
// 点一行切换并把标签滚到可见;超出高度时弹层内滚动。

const etabListKey = "etab-list"

// tabListDir 是标签行里淡色显示的所在目录:项目内取相对目录,项目外的绝对路径把家目录写成 ~,
// 仍然过长的由行内「中间省略」收短;根目录下的文件不显示。
func tabListDir(p string) string {
	if p == "" {
		return ""
	}
	dir := path.Dir(p)
	if dir == "." || dir == "/" {
		return ""
	}
	if strings.HasPrefix(dir, "/") {
		dir = tildePath(dir)
	}
	return dir
}

// etabListBtn 是标签条工具区的「全部标签」按钮,点击弹出 etabListPopup。
func (a *App) etabListBtn(c *ui.Context) {
	d := a.ide
	btn := kit.Ib(c, "chevD", kit.IbOpt{Small: true, Tip: "全部标签(" + itoa(len(d.tabs)) + ")", On: a.popup == etabListKey})
	btn.OnClick(func() { a.togglePop(etabListKey) })
	a.etabListPopup(c, btn)
}

// etabListPopup 在按钮下方右对齐弹出全部标签。
func (a *App) etabListPopup(c *ui.Context, anchor ui.Element) {
	t := a.tk
	d := a.ide
	a.popOpen(etabListKey, func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 320).AttachTo(anchor, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(6, 0, 0, 0)
			kit.VScroll(c).Key("etab-list-scroll").MaxHeight(360).Children(func() {
				ui.Column(c).Children(func() {
					for _, tb := range d.tabs {
						tb := tb
						on := tb.id == d.active
						row := ui.ButtonBase(c).Key("etab-list-" + tb.id).Height(28).PaddingX(8).Gap(8).Radius(t.RSm()).
							AlignItems(ui.Center).Justify(ui.Start).Cursor(ui.CursorDefault).Transition(kit.Fade)
						switch {
						case on:
							row.Background(t.AcSoft).TextColor(t.AcTx)
						case row.Hovered():
							row.Background(t.Fill1)
						}
						row.Children(func() {
							if tb.virt == "" {
								a.fileIc(c, path.Base(tb.path))
							} else {
								kit.Ic(c, a.tabIcon(tb), kit.S12).Shrink(0)
							}
							ui.Text(c, a.tabTitle(tb)).SingleLine().Shrink(0).MaxWidth(190)
							dir := ui.Text(c, tabListDir(tb.path)).FontSize(theme.FsCap).SingleLine().TruncateMiddle().Grow(1).MinWidth(0)
							if on {
								dir.TextColor(t.AcTx.Alpha(.7))
							} else {
								dir.TextColor(t.Tx4)
							}
							if tb.dirty() {
								ui.Box(c).Size(6, 6).Radius(3).Background(t.Tx3).Shrink(0)
							}
							slot := ui.Row(c).Size(12, 12).Shrink(0)
							slot.Children(func() {
								if on {
									kit.Ic(c, "check", kit.S12)
								}
							})
						})
						if row.Clicked() {
							d.active, d.paneSel = tb.id, 0
							if tb.path != "" {
								d.sel = tb.path
							}
							d.tabSeen = "" // 让标签条把它滚进可见区
							a.popup = ""
						}
					}
				})
			})
		})
	})
}

// onMiddleClick 让元素响应鼠标中键「点击」:在它上面按下、再在它上面抬起才触发 fn(拖到别处松手不算)。
// key 区分同时存在的多个目标;事件坐标相对元素自己,用上一帧量到的尺寸判断是否仍在内部。
func (a *App) onMiddleClick(el ui.Element, key string, fn func()) {
	d := a.ide
	b := el.Bounds() // 上一帧的矩形(事件回调里不能再读元素)
	el.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Button != 2 {
			return false
		}
		switch ev.Kind {
		case ui.InputPointerDown:
			d.midDown = key
			return true
		case ui.InputPointerUp:
			inside := ev.X >= 0 && ev.X <= b.W && ev.Y >= 0 && ev.Y <= b.H
			hit := d.midDown == key && inside
			d.midDown = ""
			if hit {
				fn()
				return true
			}
		}
		return false
	})
}
