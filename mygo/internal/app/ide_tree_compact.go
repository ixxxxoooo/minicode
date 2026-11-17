package app

import (
	"path"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// compactFolderTail 若从 start 起只有单子文件夹链,返回链尾路径与合并标签 a/b/c。
func (d *ide) compactFolderTail(start string) (tail, label string, ok bool) {
	if d.fltActive() { // 过滤时展开结构,不合并
		return start, "", false
	}
	cur := start
	parts := []string{path.Base(start)}
	for {
		es := d.listing(cur)
		var only *workspace.Entry
		files := 0
		for i := range es {
			e := es[i]
			if e.Dir {
				if only != nil {
					only = nil
					break
				}
				only = &e
			} else {
				files++
			}
		}
		if only == nil || files > 0 {
			break
		}
		parts = append(parts, only.Name)
		cur = only.Path
	}
	if len(parts) < 2 {
		return start, "", false
	}
	return cur, strings.Join(parts, "/"), true
}

func (a *App) treeCompactRow(c *ui.Context, tail, label string, depth int) {
	t := a.tk
	d := a.ide
	expanded := d.open[tail]
	on := d.sel == tail || strings.HasPrefix(d.sel, tail+"/")
	if d.revealTo != "" && (d.revealTo == tail || strings.HasPrefix(d.revealTo, tail+"/")) {
		d.treeHit = d.treeIdx
	}
	d.treeIdx++
	r := a.navDrop(kit.RowEl(c, on).Key("tr-cmp-" + tail)).PaddingStart(6 + float32(depth)*14).Gap(6)
	r.ContextMenu(func(m *ui.Menu) { a.treeMenu(m, tail, true) })
	if r.Clicked() {
		a.filesFocus = true
		d.sel = tail
		d.open[tail] = !expanded
	}
	r.Children(func() {
		if on {
			kit.SelBar(c)
		}
		icon := "chevR"
		if expanded {
			icon = "chevD"
		}
		kit.Ic(c, icon, kit.S12).TextColor(t.Tx3)
		kit.Ic(c, "folder", kit.S14)
		ui.Text(c, label).Grow(1).SingleLine().FontFeatures("tnum")
	})
	if expanded {
		a.treeDir(c, tail, depth) // 链内仍走普通树(文件与同层夹断的文件夹)
	}
}
