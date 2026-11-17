package app

import (
	"path"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// crumbMenuTitle 下拉顶部的灰色标题行(不可选)。
func crumbMenuTitle(m *ui.Menu, title string) {
	if title == "" {
		return
	}
	m.Item(title).Disabled(true)
}

// crumbPathMenuTitle 路径段同级下拉的标题(所在文件夹名)。
func crumbPathMenuTitle(fs workspace.FS, parts []string, partIx int) string {
	if partIx <= 0 || partIx >= len(parts) {
		return fs.Name()
	}
	return parts[partIx-1]
}

func sortCrumbEntries(es []workspace.Entry) {
	sort.Slice(es, func(i, j int) bool {
		if es[i].Dir != es[j].Dir {
			return es[i].Dir
		}
		return strings.ToLower(es[i].Name) < strings.ToLower(es[j].Name)
	})
}

// crumbPartMenu 面包屑路径段的同级下拉:目录在前,文件在后。
func (a *App) crumbPartMenu(m *ui.Menu, tb *ideTab, partIx int) {
	parts := strings.Split(tb.path, "/")
	if partIx < 0 || partIx >= len(parts) {
		return
	}
	parent := strings.Join(parts[:partIx], "/")
	cur := parts[partIx]
	crumbMenuTitle(m, crumbPathMenuTitle(a.ide.fs, parts, partIx))
	es, err := a.ide.fs.List(parent)
	if err != nil {
		m.Item("读取失败").Disabled(true)
		return
	}
	if len(es) == 0 {
		m.Item("没有同级项").Disabled(true)
		return
	}
	sortCrumbEntries(es)
	for _, e := range es {
		p := e.Path
		ic := "code"
		if e.Dir {
			ic = "folder"
		}
		it := mItem(m, ic, e.Name)
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

// crumbSegment 渲染可点击的路径段。
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
		if last && tb.virt == "" && !tb.diff {
			a.fileIc(c, name)
		}
		x := ui.Text(c, name)
		if last {
			x.TextColor(t.Tx1).FontWeight(500)
		}
	})
}

// crumbSymSegment 符号段同级下拉(LSP 大纲数据与本地符号索引同源)。
func (a *App) crumbSymSegment(c *ui.Context, tb *ideTab, syms []symbol, symIx int) {
	if symIx < 0 || symIx >= len(syms) {
		return
	}
	sy := syms[symIx]
	t := a.tk
	active := a.scopeOn() && symIx == len(enclosing(syms, tb.st.CursorPos().Line))-1
	b := ui.ButtonBase(c).Key("crumb-sym-" + tb.path + "-" + itoa(symIx)).Height(18).PaddingX(3).Gap(3).
		Radius(t.RXs()).AlignItems(ui.Center).Cursor(ui.CursorPointer).Menu(func(m *ui.Menu) {
		a.crumbSymMenu(m, tb, syms, symIx)
	})
	if active {
		b.Tooltip(scopeTip(sy))
	}
	if b.Hovered() {
		b.Background(t.Fill1)
	}
	b.Children(func() {
		kit.Ic(c, "cube", kit.S12).TextColor(t.AcTx)
		lbl := ui.Text(c, sy.Name)
		if active {
			lbl.TextColor(t.Ac).FontWeight(500)
		}
	})
}

func (a *App) crumbSymMenu(m *ui.Menu, tb *ideTab, syms []symbol, symIx int) {
	if len(syms) == 0 {
		m.Item("没有同级项").Disabled(true)
		return
	}
	sibs := siblingSymbols(syms, symIx)
	if len(sibs) == 0 {
		m.Item("没有同级项").Disabled(true)
		return
	}
	crumbMenuTitle(m, "同级符号")
	sort.Slice(sibs, func(i, j int) bool { return syms[sibs[i]].Line < syms[sibs[j]].Line })
	cur := syms[symIx]
	for _, ix := range sibs {
		s := syms[ix]
		it := mItem(m, "cube", s.Name)
		if s.Name == cur.Name && s.Line == cur.Line {
			it.Disabled(true)
		}
		if it.Chosen() {
			a.gotoSymbol(tb, s)
		}
		if s.Line >= 0 {
			it.Trailing(itoa(s.Line + 1))
		}
	}
}

// siblingSymbols 返回与 symIx 同父的符号下标(含自身)。
func siblingSymbols(syms []symbol, symIx int) []int {
	if symIx < 0 || symIx >= len(syms) {
		return nil
	}
	parent := syms[symIx].Parent
	var out []int
	for i, s := range syms {
		if s.Parent == parent {
			out = append(out, i)
		}
	}
	return out
}

func (a *App) gotoSymbol(tb *ideTab, s symbol) {
	if tb == nil || tb.st == nil || tb.buf == nil {
		return
	}
	p0 := editor.Pos{Line: s.Line, Col: s.Col}
	tb.st.GoTo(p0)
	endCol := s.Col
	if s.Line < tb.buf.LineCount() {
		_, endCol, _ = wordAt(tb.buf.Line(s.Line), s.Col)
		if endCol <= s.Col {
			endCol = s.Col + len(s.Name)
		}
	}
	tb.st.Select(tb.buf.Offset(p0), tb.buf.Offset(editor.Pos{Line: s.Line, Col: endCol}))
}

// dbgCrumb 打开文件并展开路径上的文件夹;-ed crumb:file|sym
func (a *App) dbgCrumb(arg string) {
	p := "src/router.rs"
	switch arg {
	case "sym":
		p = "src/router.rs"
	case "file":
		p = "src/router.rs"
	case "":
	default:
		p = arg
	}
	a.SetMode(ModeIDE)
	dir := path.Dir(p)
	for dir != "." && dir != "/" && dir != "" {
		a.ide.open[strings.ReplaceAll(dir, "\\", "/")] = true
		dir = path.Dir(dir)
	}
	a.ide.openFile(p, true)
	a.ide.sel = p
	if arg == "sym" {
		if tb := a.ide.cur(); tb != nil && tb.st != nil {
			tb.st.GoTo(editor.Pos{Line: 16, Col: 0})
		}
	}
}
