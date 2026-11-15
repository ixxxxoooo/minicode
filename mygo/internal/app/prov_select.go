package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// selItem 是下拉里的一项。
type selItem struct {
	id, label string
	hint      string // 行尾灰字(说明或置灰原因)
	disabled  bool
	mono      bool
	logo      bool   // 行首画品牌 logo
	brand     string // logo 的品牌(空 = 占位图标)
}

// selSearchMin 选项超过这个数量时,下拉顶部出现搜索框。
const selSearchMin = 8

// selectBox 是通用的下拉选择框(设计稿 .sel):按钮显示当前项,点开弹出可滚动的列表,
// 选项多时顶部带搜索;置灰项显示原因且不可点;当前项带对勾。
// key 要在同一页内唯一;cur 是当前项的 id(找不到时显示 empty);选中后回调 onPick。
func (a *App) selectBox(c *ui.Context, key string, w float32, items []selItem, cur, empty string, mono bool, onPick func(id string)) {
	t := a.tk
	label := empty
	var curItem *selItem
	for i, it := range items {
		if it.id == cur {
			label = it.label
			curItem = &items[i]
		}
	}
	pk := "sel@" + key
	b := ui.ButtonBase(c).Key("sel-"+key).Width(w).Height(24).PaddingX(8).Gap(6).Radius(t.RSm()).AlignItems(ui.Center).
		Background(t.Raised).Border(1, t.Bd2).Cursor(ui.CursorPointer).FontSize(12).Shrink(0).Transition(kit.Fade)
	if b.Hovered() || a.popup == pk {
		b.Background(t.Raised.Mix(t.Tx1, .04))
	}
	if a.popup == pk {
		b.BorderColor(t.Ac)
	}
	b.OnClick(func() {
		a.set.pv.selQ = ""
		a.togglePop(pk)
	})
	b.Children(func() {
		if curItem != nil && curItem.logo {
			kit.Logo(c, curItem.brand, 14, t.Tx2)
		}
		tx := ui.Text(c, label).SingleLine().Grow(1).MinWidth(0).Justify(ui.Start)
		if mono {
			tx.Font("monospace")
		}
		if cur == "" || label == empty {
			tx.TextColor(t.Tx3)
		}
		kit.Ic(c, "chevD", kit.S12).TextColor(t.Tx3)
	})
	a.popOpen(pk, func(open *bool) {
		ui.PopoverBase(c, b, open, func(panel ui.Element) {
			pw := max(w, 220)
			a.popPanel(panel, pw).AttachTo(b, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(4, 0, 0, 0)
			q := strings.ToLower(strings.TrimSpace(a.set.pv.selQ))
			if len(items) > selSearchMin {
				ui.Row(c).Height(26).Margin(2, 2, 4, 2).PaddingX(8).Gap(6).AlignItems(ui.Center).Background(t.Fill1).Radius(t.RSm()).Children(func() {
					kit.Ic(c, "search", kit.S12).TextColor(t.Tx3)
					ui.TextInputBase(c.Key("sel-q-"+key), &a.set.pv.selQ).Grow(1).FontSize(12).Placeholder("搜索").AutoFocus()
				})
			}
			shown := 0
			ui.Scroll(c).Key("sel-sc-" + key).MaxHeight(260).Children(func() {
				ui.Column(c).Children(func() {
					for _, it := range items {
						if q != "" && !strings.Contains(strings.ToLower(it.label+" "+it.id), q) {
							continue
						}
						shown++
						a.selRow(c, key, it, it.id == cur, func() {
							a.popup = ""
							onPick(it.id)
						})
					}
					if shown == 0 {
						ui.Text(c, "没有匹配的选项").FontSize(theme.FsCap).TextColor(t.Tx3).Padding(8, 8)
					}
				})
			})
		})
	})
}

// selRow 是下拉里的一行:当前项 ac-soft 底 + 对勾,置灰项降低透明度并写原因。
func (a *App) selRow(c *ui.Context, key string, it selItem, on bool, pick func()) {
	t := a.tk
	if it.disabled {
		ui.Row(c).Height(26).PaddingX(8).Gap(8).AlignItems(ui.Center).Opacity(.5).Children(func() {
			if it.logo {
				kit.Logo(c, it.brand, 14, t.Tx2)
			}
			tx := ui.Text(c, it.label).SingleLine().Grow(1)
			if it.mono {
				tx.Font("monospace")
			}
			if it.hint != "" {
				ui.Text(c, it.hint).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
			}
		})
		return
	}
	b := ui.ButtonBase(c).Key("selrow-" + key + "-" + it.id).Height(26).PaddingX(8).Gap(8).Radius(t.RSm()).AlignItems(ui.Center).
		Justify(ui.Start).Cursor(ui.CursorDefault).Transition(kit.Fade)
	switch {
	case on:
		b.Background(t.AcSoft).TextColor(t.AcTx)
	case b.Hovered():
		b.Background(t.Fill1)
	}
	b.OnClick(pick)
	b.Children(func() {
		if it.logo {
			kit.Logo(c, it.brand, 14, t.Tx2)
		}
		tx := ui.Text(c, it.label).SingleLine().Grow(1).MinWidth(0)
		if it.mono {
			tx.Font("monospace")
		}
		if it.hint != "" {
			ui.Text(c, it.hint).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
		}
		if on {
			kit.Ic(c, "check", kit.S12)
		}
	})
}
