package app

import (
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
)

// crumbPartMenu 面包屑路径段的同级下拉:目录在前,文件在后。
func (a *App) crumbPartMenu(m *ui.Menu, tb *ideTab, partIx int) {
	parts := strings.Split(tb.path, "/")
	if partIx < 0 || partIx >= len(parts) {
		return
	}
	parent := strings.Join(parts[:partIx], "/")
	cur := parts[partIx]
	for _, e := range a.ide.listing(parent) {
		p := e.Path
		it := m.Item(e.Name)
		if e.Name == cur {
			it.Disabled(true)
		}
		if e.Dir && it.Chosen() {
			a.ide.open[p] = true
			a.ide.sel = p
			a.revealInTree(p)
		}
		if !e.Dir && it.Chosen() {
			a.ide.openFile(p, partIx == len(parts)-1)
			a.ide.sel = p
			a.revealInTree(p)
		}
	}
}

// crumbSegment 渲染可点击的路径段(符号段仍走原有逻辑)。
func (a *App) crumbSegment(c *ui.Context, tb *ideTab, name string, partIx int, last bool) {
	t := a.tk
	b := ui.ButtonBase(c).Key("crumb-" + tb.path + "-" + itoa(partIx)).Height(18).PaddingX(3).Gap(3).
		Radius(t.RXs()).AlignItems(ui.Center).Cursor(ui.CursorPointer).Menu(func(m *ui.Menu) {
		a.crumbPartMenu(m, tb, partIx)
	})
	if b.Hovered() {
		b.Background(t.Fill1)
	}
	b.Children(func() {
		x := ui.Text(c, name)
		if last {
			x.TextColor(t.Tx1).FontWeight(500)
		}
	})
}

// dbgCrumb 打开文件并展开路径上的文件夹(截图面包屑)。
func (a *App) dbgCrumb(p string) {
	if p == "" {
		p = "src/router.rs"
	}
	a.SetMode(ModeIDE)
	dir := path.Dir(p)
	for dir != "." && dir != "/" && dir != "" {
		a.ide.open[strings.ReplaceAll(dir, "\\", "/")] = true
		dir = path.Dir(dir)
	}
	a.ide.openFile(p, true)
	a.ide.sel = p
}
