package app

import (
	"path"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

var refsLog = diag.L("refs")

// refsState 是底部「引用」分段的状态。
type refsState struct {
	word    string
	items   []refItem
	folded  map[string]bool
	list    ui.ListState
	scroll  ui.ScrollState
	selKey  string
	loading bool
	err     string
	gen     int
	rows    []refFlatRow
}

func (r *refsState) ensureFolded() {
	if r.folded == nil {
		r.folded = map[string]bool{}
	}
}

func (r *refsState) rebuildRows() {
	r.rows = refFlatten(r.items, r.folded)
}

// refsHeadChrome 画引用标签下标题栏右侧:符号摘要、全部折叠/展开、重新查找、清除。
func (a *App) refsHeadChrome(c *ui.Context, r *refsState) {
	t := a.tk
	if r == nil {
		return
	}
	nHit, nFile := len(r.items), refFileCount(r.items)
	summary := itoa(nHit) + " 处引用 · " + itoa(nFile) + " 个文件"
	tip := "「" + r.word + "」的所有引用"
	if a.lspOn() {
		tip += "(语言服务器 gopls)"
	} else {
		tip += "(本地索引)"
	}
	ui.Row(c).Gap(6).AlignItems(ui.Center).Margin(0, 4, 0, 0).Children(func() {
		kit.Chip(c, r.word, kit.ChipOpt{Line: true, Tip: tip})
		ui.Text(c, summary).Font("monospace").FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
	})
	if kit.Ib(c, "collapseAll", kit.IbOpt{Small: true, Tip: "全部折叠"}).Clicked() {
		a.refsCollapseAll()
	}
	if kit.Ib(c, "tree", kit.IbOpt{Small: true, Tip: "全部展开"}).Clicked() {
		a.refsExpandAll()
	}
	if kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "重新查找"}).Clicked() {
		a.refreshRefs()
	}
	if kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "清除结果"}).Clicked() {
		a.clearRefs()
	}
}

func (a *App) refsCollapseAll() {
	r := a.ide.refs
	if r == nil {
		return
	}
	r.ensureFolded()
	seen := map[string]bool{}
	for _, it := range r.items {
		seen[it.path] = true
	}
	for p := range seen {
		r.folded[p] = true
	}
	r.rebuildRows()
	refsLog.Debug("引用全部折叠", "word", r.word)
}

func (a *App) refsExpandAll() {
	r := a.ide.refs
	if r == nil {
		return
	}
	r.folded = map[string]bool{}
	r.rebuildRows()
	refsLog.Debug("引用全部展开", "word", r.word)
}

func (a *App) refreshRefs() {
	r := a.ide.refs
	if r == nil || r.word == "" {
		return
	}
	refsLog.Info("重新查找引用", "word", r.word)
	a.showRefs(r.word)
}

func (a *App) clearRefs() {
	refsLog.Info("清除引用结果")
	a.ide.refs = nil
	if a.ide.bottomTab == "refs" {
		a.ide.bottomTab = "term"
	}
}

func (a *App) refsBeginSearch(word string) *refsState {
	gen := 1
	if r := a.ide.refs; r != nil {
		gen = r.gen + 1
	}
	r := &refsState{word: word, folded: map[string]bool{}, gen: gen}
	a.ide.refs = r
	a.ide.bottomOpen, a.ide.bottomTab = true, "refs"
	return r
}

func (a *App) refsFinishLocal(r *refsState, items []refItem) {
	r.loading = false
	r.items = items
	sort.SliceStable(r.items, func(i, j int) bool {
		if r.items[i].path != r.items[j].path {
			return r.items[i].path < r.items[j].path
		}
		if r.items[i].line != r.items[j].line {
			return r.items[i].line < r.items[j].line
		}
		return r.items[i].col < r.items[j].col
	})
	r.rebuildRows()
}

// refsDemoItems 演示模式下的引用数据。
func (a *App) refsDemoItems(word string) []refItem {
	if hits, ok := demo.RefPanelDemo(word); ok {
		var out []refItem
		for _, h := range hits {
			out = append(out, refItem{path: h.Path, line: h.Line, col: h.Col, text: h.Text})
		}
		return out
	}
	return a.ide.findRefs(word)
}

func (a *App) refsSyncFromEditor() {
	r := a.ide.refs
	if r == nil || len(r.items) == 0 || a.ide.bottomTab != "refs" {
		return
	}
	tb := a.ide.cur()
	if tb == nil || tb.st == nil || tb.buf == nil {
		return
	}
	p := tb.st.CursorPos()
	want := refFindSel(r.items, r.word, tb.path, p.Line, p.Col)
	if want < 0 {
		return
	}
	k := refItemKey(r.items[want], r.word)
	if k == r.selKey {
		return
	}
	r.selKey = k
	for i, row := range r.rows {
		if row.kind == refRowHit && row.ix == want {
			r.list.Selected = &i
			break
		}
	}
}

// refsPanel 是「引用」分段主体。
func (a *App) refsPanel(c *ui.Context) {
	t := a.tk
	r := a.ide.refs
	if r == nil {
		return
	}
	r.ensureFolded()
	if r.rows == nil && !r.loading {
		r.rebuildRows()
	}
	a.refsSyncFromEditor()
	if r.loading {
		ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(8).Children(func() {
			kit.Spin(c, t.Tx3)
			ui.Text(c, "正在查找「"+r.word+"」的引用…").TextColor(t.Tx3)
		})
		return
	}
	if len(r.items) == 0 {
		ui.Column(c).Grow(1).Padding(16, 20).Gap(8).Children(func() {
			if r.err != "" {
				ui.Text(c, r.err).TextColor(t.Err).FontSize(theme.FsCap)
				kit.Btn(c, "重试", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "refresh"}).OnClick(a.refreshRefs)
				return
			}
			msg := "没有找到「" + r.word + "」的引用"
			hint := "把光标放在符号上按 ⇧F12 重新查找"
			if !a.lspOn() && a.proj != "" {
				hint = "语言服务器未就绪时仅搜索工作区内文件;请打开源文件并等待 gopls 启动后再试"
			}
			if a.proj == "" {
				hint = "演示模式:在 router.rs 里对 normalize 按 ⇧F12"
			}
			ui.Text(c, msg).TextColor(t.Tx3)
			ui.Text(c, hint).FontSize(theme.FsCap).TextColor(t.Tx4)
		})
		return
	}
	if r.err != "" {
		ui.Row(c).Padding(6, 10).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
			ui.Text(c, r.err).TextColor(t.Err).FontSize(theme.FsCap).Grow(1).SingleLine()
			kit.Btn(c, "重试", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "refresh"}).OnClick(a.refreshRefs)
		})
	}
	col := ui.Column(c).Grow(1).MinHeight(0)
	col.HandleInput(a.refsInput)
	col.Children(func() {
		scroll := kit.VScroll(c).Key("refs").TrackScroll(&r.scroll).Grow(1).MinHeight(0)
		if len(r.rows) == 0 {
			r.rebuildRows()
		}
		scroll.Children(func() {
			ui.List(c, &r.list, len(r.rows), func(i int) {
				if i >= len(r.rows) {
					return
				}
				a.refsRow(c, r, r.rows[i], i)
			})
		})
	})
}

func (a *App) refsInput(ev ui.InputEvent) bool {
	r := a.ide.refs
	if r == nil || ev.Kind != ui.InputKeyDown {
		return false
	}
	if ev.Key == ui.KeyEscape {
		r.list.Selected = nil
		return true
	}
	if r.list.Selected == nil {
		return false
	}
	sel := *r.list.Selected
	if sel < 0 || sel >= len(r.rows) {
		return false
	}
	row := r.rows[sel]
	switch ev.Key {
	case ui.KeyLeft:
		if row.kind == refRowFile && !r.folded[row.path] {
			r.folded[row.path] = true
			r.rebuildRows()
			return true
		}
	case ui.KeyRight:
		if row.kind == refRowFile && r.folded[row.path] {
			delete(r.folded, row.path)
			r.rebuildRows()
			return true
		}
	}
	return false
}

func (a *App) refsRow(c *ui.Context, r *refsState, row refFlatRow, i int) {
	if row.kind == refRowFile {
		a.refsFileRow(c, r, row.path, i)
		return
	}
	if row.ix < 0 || row.ix >= len(r.items) {
		return
	}
	a.refsHitRow(c, r, r.items[row.ix], row.ix, i)
}

func (a *App) refsFileRow(c *ui.Context, r *refsState, file string, i int) {
	t := a.tk
	folded := r.folded[file]
	n := 0
	for _, it := range r.items {
		if it.path == file {
			n++
		}
	}
	on := r.list.Selected != nil && *r.list.Selected == i
	btn := a.navDrop(ui.ButtonBase(c).Key("rg-" + file)).Height(theme.Px(24)).PaddingX(8).Gap(5).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer)
	if on {
		kit.SelStyle(btn)
	} else if btn.Hovered() {
		btn.Background(t.Fill1)
	} else if folded {
		btn.Background(t.Fill1.Over(t.BG)).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd2)
	}
	btn.OnClick(func() {
		r.folded[file] = !folded
		r.rebuildRows()
	})
	btn.ContextMenu(func(m *ui.Menu) { a.refsFileMenu(m, r, file) })
	btn.Children(func() {
		chev := map[bool]string{true: "chevR", false: "chevD"}[folded]
		kit.Ic(c, chev, kit.S12).TextColor(t.Tx3).Transition(kit.Fade)
		a.fileIc(c, path.Base(file))
		base := path.Base(file)
		dir := path.Dir(file)
		tc := t.Tx2
		if folded {
			tc = t.Tx4
		}
		ui.Text(c, base).FontWeight(600).TextColor(tc).Shrink(0).SingleLine()
		if dir != "." && dir != "" {
			ui.Text(c, dir).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().Shrink(1).MinWidth(0)
		}
		if workspace.IsExternalPath(file) {
			kit.Ic(c, "lock", kit.S12).TextColor(t.Tx4).Tooltip("项目外 · 只读打开")
		}
		if refIsTestFile(file) {
			kit.Tag(c, "测试", kit.KInfo)
		}
		ui.Spacer(c).Grow(1)
		accent := folded
		ui.Box(c).Tooltip("该文件里的引用数").Children(func() {
			kit.Cnt(c, itoa(n), accent)
		})
	})
}

func (a *App) refsHitRow(c *ui.Context, r *refsState, it refItem, ix, i int) {
	t := a.tk
	word := r.word
	key := refItemKey(it, word)
	on := r.selKey == key || (r.list.Selected != nil && *r.list.Selected == i)
	row := kit.RowEl(c, on).Key("rf-" + key).Gap(6).PaddingStart(22).MinHeight(theme.Px(22))
	row.ContextMenu(func(m *ui.Menu) { a.refsHitMenu(m, r, it) })
	row.OnClick(func() {
		r.selKey = key
		r.list.Selected = &i
		end := it.col + len(word)
		a.jumpTo(it.path, it.line, it.col, end)
	})
	row.Children(func() {
		if on && t.Scheme.ID != "b" {
			kit.SelBar(c)
		}
		lnCol := t.Tx4
		if on {
			lnCol = t.AcTx
		}
		ui.Text(c, itoa(it.line+1)).FontSize(theme.FsMicro).TextColor(lnCol).Width(32).TextAlign(ui.End).FontFeatures("tnum").NoWrap()
		raw := it.text
		hitLo, hitHi := it.col, it.col+len(word)
		if raw == "" {
			hitLo, hitHi = 0, 0
		}
		vis, vLo, vHi := refPreviewBytes(raw, hitLo, hitHi, 72)
		if vis == "" {
			vis = strings.TrimSpace(raw)
			vLo, vHi = hitLo, hitHi
		}
		ui.Box(c).Grow(1).Shrink(1).MinWidth(0).Clip().Children(func() {
			ui.RichText(c, a.refLineSpans(it.path, vis, vLo, vHi)...).NoWrap()
		})
		if refLooksLikeDef(raw, word) {
			kit.Tag(c, "定义", kit.KAc)
		}
		ui.Text(c, itoa(it.col+1)).FontSize(theme.FsMicro).TextColor(t.Tx4).FontFeatures("tnum").NoWrap()
	})
}

func (a *App) refsFileMenu(m *ui.Menu, r *refsState, file string) {
	folded := r.folded[file]
	label := "折叠"
	if folded {
		label = "展开"
	}
	if menuItem(m, label).Chosen() {
		r.folded[file] = !folded
		r.rebuildRows()
	}
	if menuItem(m, "折叠其它").Chosen() {
		for p := range r.folded {
			r.folded[p] = p != file
		}
		for _, it := range r.items {
			if it.path != file {
				r.folded[it.path] = true
			}
		}
		r.rebuildRows()
	}
	if menuItem(m, "在文件树中定位").Chosen() {
		if !workspace.IsExternalPath(file) {
			a.revealInTree(file)
		}
	}
	m.Separator()
	if menuItem(m, "复制路径").Chosen() {
		a.copyOut(file, "路径")
	}
	if menuItem(m, "复制全部引用").Chosen() {
		a.copyOut(a.refsAllText(r), "全部引用")
	}
}

func (a *App) refsHitMenu(m *ui.Menu, r *refsState, it refItem) {
	word := r.word
	if menuItem(m, "转到位置").Chosen() {
		a.jumpTo(it.path, it.line, it.col, it.col+len(word))
	}
	if menuItem(m, "在文件树中定位").Chosen() {
		if !workspace.IsExternalPath(it.path) {
			a.revealInTree(it.path)
		}
	}
	m.Separator()
	if menuItem(m, "复制路径与行号").Chosen() {
		a.copyOut(it.path+":"+itoa(it.line+1)+":"+itoa(it.col+1), "路径与行号")
	}
	if menuItem(m, "复制该行").Chosen() {
		a.copyOut(strings.TrimSpace(it.text), "该行")
	}
	if menuItem(m, "复制全部引用").Chosen() {
		a.copyOut(a.refsAllText(r), "全部引用")
	}
}

func (a *App) refsAllText(r *refsState) string {
	var sb strings.Builder
	for _, it := range r.items {
		sb.WriteString(it.path)
		sb.WriteByte(':')
		sb.WriteString(itoa(it.line + 1))
		sb.WriteByte(':')
		sb.WriteString(itoa(it.col + 1))
		sb.WriteByte(' ')
		sb.WriteString(strings.TrimSpace(it.text))
		sb.WriteByte('\n')
	}
	return strings.TrimSpace(sb.String())
}
