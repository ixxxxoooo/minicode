package app

import (
	"path"

	"github.com/egoist/mygo/ui"

	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// openEditorTabList 列出已打开的文件标签(两组编辑器去重,保持标签条顺序)。
func (d *ide) openEditorTabList() []*ideTab {
	seen := map[string]bool{}
	var out []*ideTab
	for _, sl := range [][]*ideTab{d.tabs, d.tabs2} {
		for _, tb := range sl {
			if tb == nil || !isFileTab(tb) || tb.diff || seen[tb.path] {
				continue
			}
			seen[tb.path] = true
			out = append(out, tb)
		}
	}
	return out
}

// openedEditorsSection 文件树根区块上方的「打开的编辑器」(设计稿三区清单)。
func (a *App) openedEditorsSection(c *ui.Context) {
	d := a.ide
	tabs := d.openEditorTabList()
	if len(tabs) == 0 {
		return
	}
	t := a.tk
	if d.sections == nil {
		d.sections = map[string]bool{}
	}
	open := d.sections["打开的编辑器"]
	head := ui.ButtonBase(c).Key("sec-open-editors").Height(theme.Px(26)).PaddingX(8).Justify(ui.Start).Gap(5).AlignItems(ui.Center).
		TextColor(t.Tx3).FontSize(theme.Px(11.5)).FontWeight(500).Cursor(ui.CursorDefault)
	if head.Clicked() {
		d.sections["打开的编辑器"] = !open
	}
	head.Children(func() {
		icon := "chevR"
		if open {
			icon = "chevD"
		}
		kit.Ic(c, icon, kit.S12)
		ui.Text(c, "打开的编辑器").Grow(1).SingleLine()
		ui.Text(c, itoa(len(tabs))).FontSize(theme.FsCap).TextColor(t.Tx4).Shrink(0)
	})
	if !open {
		return
	}
	for _, tb := range tabs {
		a.openedEditorRow(c, tb)
	}
}

func (a *App) openedEditorRow(c *ui.Context, tb *ideTab) {
	t := a.tk
	d := a.ide
	on := d.active == tb.id || d.active2 == tb.id
	r := ui.ButtonBase(c).Key("oe-" + tb.id).Height(theme.Px(26)).PaddingStart(22).PaddingEnd(6).Gap(6).
		Justify(ui.Start).AlignItems(ui.Center).Grow(1).TextColor(t.Tx2).Cursor(ui.CursorDefault)
	if on {
		r.Background(t.Fill1)
	}
	if r.Clicked() {
		if t := d.tab(tb.id); t != nil {
			if d.tabOn(tb.id, d.tabs2) != nil {
				d.paneSel, d.active2 = 1, tb.id
			} else {
				d.paneSel, d.active = 0, tb.id
			}
			d.sel = tb.path
		}
	}
	r.Children(func() {
		a.fileIc(c, path.Base(tb.path))
		ui.Text(c, path.Base(tb.path)).Grow(1).SingleLine()
		slot := ui.Row(c).Size(14, 14).Justify(ui.Center).AlignItems(ui.Center).Shrink(0)
		if tb.dirty() {
			slot.Children(func() { ui.Box(c).Size(6, 6).Radius(3).Background(t.Tx3) })
			return
		}
		x := slot.Key("oe-x-" + tb.id)
		if x.Hovered() {
			x.Background(t.Fill2)
		}
		if x.Clicked() {
			a.closeWsTab(tb.id)
		}
		x.Children(func() { kit.Ic(c, "x", kit.S12).TextColor(t.Tx3) })
	})
}
