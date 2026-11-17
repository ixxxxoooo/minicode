package app

import (
	"path"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

var treeLog = diag.L("ide-tree")

// 文件树内联过滤(设计稿 ide-nav 三区清单「左栏 · 过滤文件」):
// 标题行 ⏷ 按钮或树内 ⌘F 展开过滤条,输入即缩小树——只留命中的文件和它们的祖先目录,目录一律展开;
// 文件夹名命中则整个文件夹的内容都留着。Esc 清空并收起,清空关键字树恢复原来的展开状态(不改用户的展开 / 折叠)。

// fltActive 过滤条已展开且有关键字。
func (d *ide) fltActive() bool { return d.fltOpen && strings.TrimSpace(d.flt) != "" }

// treeFilterStep 在过滤命中文件间上下移动(需先有过滤关键字)。
func (a *App) treeFilterStep(dy int) {
	d := a.ide
	if !d.fltActive() {
		return
	}
	d.fltCompute()
	var hits []string
	for _, p := range d.allFiles() {
		if d.fltVis[p] {
			hits = append(hits, p)
		}
	}
	if len(hits) == 0 {
		return
	}
	ix := 0
	for i, p := range hits {
		if p == d.sel {
			ix = i
			break
		}
	}
	ix += dy
	if ix < 0 {
		ix = len(hits) - 1
	}
	if ix >= len(hits) {
		ix = 0
	}
	d.sel = hits[ix]
	d.revealTo = hits[ix]
	a.filesFocus = true
	treeLog.Info("过滤命中间移动", "path", d.sel)
}

func (a *App) toggleTreeFilter() {
	d := a.ide
	d.fltOpen = !d.fltOpen
	if !d.fltOpen {
		d.flt = ""
	}
	treeLog.Info("文件树过滤", "展开", d.fltOpen)
}

// fltCompute 算出该显示哪些路径,按「关键字 + 文件清单」缓存。
func (d *ide) fltCompute() {
	q := strings.ToLower(strings.TrimSpace(d.flt))
	files := d.allFiles()
	key := q + "|" + itoa(len(files)) + "|" + d.fs.Name()
	if d.fltKey == key && d.fltVis != nil {
		return
	}
	vis := map[string]bool{}
	n := 0
	for _, p := range files {
		hit := false
		if strings.Contains(q, "/") { // 带斜杠:按整条路径匹配
			hit = strings.Contains(strings.ToLower(p), q)
		} else {
			// 文件名命中,或任一级目录名命中(整个文件夹都留着)
			for _, seg := range strings.Split(p, "/") {
				if strings.Contains(strings.ToLower(seg), q) {
					hit = true
					break
				}
			}
		}
		if !hit {
			continue
		}
		n++
		vis[p] = true
		for dir := path.Dir(p); dir != "." && dir != "/" && dir != ""; dir = path.Dir(dir) {
			vis[dir] = true
		}
	}
	d.fltKey, d.fltVis, d.fltN = key, vis, n
}

// treeFilterBar 过滤条:输入框 + 命中数 + 清除。
func (a *App) treeFilterBar(c *ui.Context) {
	d := a.ide
	if !d.fltOpen {
		return
	}
	t := a.tk
	ui.Row(c).Key("tree-flt").Height(theme.Px(28)).PaddingX(8).Gap(5).AlignItems(ui.Center).Children(func() {
		box := ui.Row(c).Key("tree-flt-box").Grow(1).MinWidth(0).Height(theme.Px(22)).PaddingX(6).Gap(5).AlignItems(ui.Center).
			Background(t.Fill1).Radius(t.RSm()).Border(1, t.Ac)
		box.Children(func() {
			kit.Ic(c, "filter", kit.S12).TextColor(t.Tx3)
			in := ui.TextInputBase(c.Key("tree-flt-in"), &d.flt).Grow(1).MinWidth(0).FontSize(theme.FsBody).
				Background(ui.Transparent).BorderWidth(0).Padding(0).FocusRing(false).AutoFocus().Placeholder("过滤文件名…")
			in.OnShortcut(0, ui.KeyEscape, a.toggleTreeFilter)
			if d.fltActive() {
				d.fltCompute()
				ui.Text(c, itoa(d.fltN)+" 项").FontSize(theme.FsCap).TextColor(t.Tx3).Shrink(0)
				kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "清空"}).OnClick(func() { d.flt = "" })
			}
		})
	})
}

// treeFiltered 与 treeDir 同形,但只画命中的路径、目录强制展开。
func (a *App) treeFiltered(c *ui.Context, dir string, depth int) {
	d := a.ide
	d.fltCompute()
	if dir == "" && d.fltN == 0 {
		ui.Column(c).Key("tree-flt-empty").PaddingX(14).PaddingY(10).Gap(6).Children(func() {
			ui.Text(c, "没有名称含「"+strings.TrimSpace(d.flt)+"」的文件").TextColor(a.tk.Tx3).FontSize(theme.FsCap)
			kit.Btn(c, "清空过滤", kit.BtnOpt{Small: true}).OnClick(func() { d.flt = "" })
		})
		return
	}
	for _, e := range d.listing(dir) {
		if !d.fltVis[e.Path] {
			continue
		}
		a.treeRow(c, e, depth)
		if e.Dir {
			a.treeFiltered(c, e.Path, depth+1)
		}
	}
}
