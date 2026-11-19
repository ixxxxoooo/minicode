package app

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/tabular"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 表格查看器(csv · tsv · xlsx):只读,但可搜索 / 筛选 / 排序 / 隐藏列 / 选中并复制。
// 数据永远不改:筛选与排序只算出「显示哪些行、什么顺序」(tableView.rows),界面按它取格子。

type tblOp int

const (
	tblContains tblOp = iota
	tblEq
	tblNe
	tblPrefix
	tblSuffix
	tblEmpty
	tblNotEmpty
	tblGt
	tblLt
)

var tblOpNames = []string{"包含", "等于", "不等于", "开头是", "结尾是", "为空", "不为空", "大于", "小于"}

func (o tblOp) needsValue() bool { return o != tblEmpty && o != tblNotEmpty }

// tblFilter 是一条列筛选条件;多条之间是「且」。
type tblFilter struct {
	col int
	op  tblOp
	val string
}

// tableView 是某张表的显示状态:列定义、搜索 / 筛选 / 排序、选中。换工作表时整个重建。
type tableView struct {
	sheet  int
	header bool
	start  int // 数据从第几行开始(首行作表头时为 1)

	list    *ui.ListState
	sel     ui.Selection[int] // 选中的行(键 = 表里的行下标,筛选 / 排序后仍然跟着行走)
	allCols []ui.TableColumn  // 每个数据列的定义(下标 = 数据列号)
	vis     []int             // 显示出来的数据列号(隐藏的不在里面)
	cols    []ui.TableColumn  // 交给 ui.Table 的列:「#」+ 显示的列
	hidden  map[int]bool

	sort        ui.SortOrder
	q           string      // 搜索框内容
	filters     []tblFilter // 列筛选条件
	showFilters bool        // 筛选条显示与否(有条件时总是显示)

	rows []int    // 当前显示的行(表里的行下标,已筛选、已排序)
	sig  string   // 上次算 rows 时的条件签名,变了才重算
	low  []string // 每行小写后拼起来,搜索用(第一次搜索时才算)
}

// buildTableView 按当前表与表头设置建显示状态。
func buildTableView(sh *tabular.Sheet, sheet int, header bool) *tableView {
	tv := &tableView{sheet: sheet, list: &ui.ListState{}, hidden: map[int]bool{}}
	tv.list.CellSelect = true
	tv.list.Selection = &tv.sel
	tv.list.Key = func(i int) any {
		if i >= 0 && i < len(tv.rows) {
			return tv.rows[i]
		}
		return -1 - i
	}
	tv.list.Sort = &tv.sort
	tv.setHeader(sh, header)
	return tv
}

// setHeader 切换「首行作表头」:重算列标题与列宽,保留搜索 / 筛选 / 排序。
func (tv *tableView) setHeader(sh *tabular.Sheet, header bool) {
	tv.header = header
	tv.start = 0
	if header && len(sh.Rows) > 0 {
		tv.start = 1
	}
	tv.allCols = tv.allCols[:0]
	sample := len(sh.Rows)
	if sample > 200 {
		sample = 200
	}
	for j := 0; j < sh.Cols; j++ {
		title := tabular.ColName(j)
		if tv.start == 1 {
			if h := strings.TrimSpace(sh.Cell(0, j)); h != "" {
				title = clipRunes(h, 40)
			}
		}
		u := textWidthUnits(title) + 3 // 留出排序箭头的位置
		for i := 0; i < sample; i++ {
			if n := textWidthUnits(sh.Cell(i, j)); n > u {
				u = n
			}
		}
		w := float32(u)*7 + 28
		if w < 72 {
			w = 72
		}
		if w > 320 {
			w = 320
		}
		tv.allCols = append(tv.allCols, ui.TableColumn{Title: title, ID: "c" + strconv.Itoa(j), Width: w, MinWidth: 40, MaxWidth: 800, Sortable: true})
	}
	tv.rebuildCols()
	tv.sig = ""
}

// rebuildCols 按隐藏情况重排交给 Table 的列。
func (tv *tableView) rebuildCols() {
	tv.vis = tv.vis[:0]
	tv.cols = append(tv.cols[:0], ui.TableColumn{Title: "#", ID: "#", Width: 52, Fixed: true, Align: ui.End, RowHeader: true})
	for j := range tv.allCols {
		if !tv.hidden[j] {
			tv.vis = append(tv.vis, j)
			tv.cols = append(tv.cols, tv.allCols[j])
		}
	}
}

func (tv *tableView) colTitle(j int) string {
	if j >= 0 && j < len(tv.allCols) {
		return tv.allCols[j].Title
	}
	return tabular.ColName(j)
}

// colOfID 把列 ID(c3)转成数据列号,不是数据列返回 -1。
func colOfID(id string) int {
	if !strings.HasPrefix(id, "c") {
		return -1
	}
	n, err := strconv.Atoi(id[1:])
	if err != nil {
		return -1
	}
	return n
}

// parseNum 把「1,234」「12%」「¥30」这类格子读成数字。
func parseNum(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	s = strings.NewReplacer(",", "", "¥", "", "$", "", "%", "", " ", "").Replace(s)
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

func (f tblFilter) match(cell, lowVal string) bool {
	switch f.op {
	case tblEmpty:
		return strings.TrimSpace(cell) == ""
	case tblNotEmpty:
		return strings.TrimSpace(cell) != ""
	}
	if f.val == "" {
		return true // 条件还没填:不过滤
	}
	switch f.op {
	case tblGt, tblLt:
		a, ok1 := parseNum(cell)
		b, ok2 := parseNum(f.val)
		if ok1 && ok2 {
			if f.op == tblGt {
				return a > b
			}
			return a < b
		}
		if f.op == tblGt {
			return strings.ToLower(cell) > lowVal
		}
		return strings.ToLower(cell) < lowVal
	}
	lc := strings.ToLower(cell)
	switch f.op {
	case tblEq:
		return lc == lowVal
	case tblNe:
		return lc != lowVal
	case tblPrefix:
		return strings.HasPrefix(lc, lowVal)
	case tblSuffix:
		return strings.HasSuffix(lc, lowVal)
	}
	return strings.Contains(lc, lowVal)
}

func (tv *tableView) signature() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%t|%s|%s|%t|%s", tv.header, tv.q, tv.sort.Column, tv.sort.Descending, "")
	for _, f := range tv.filters {
		fmt.Fprintf(&b, "|%d:%d:%s", f.col, f.op, f.val)
	}
	return b.String()
}

// refresh 条件变了就重算要显示的行(筛选 → 排序),没变什么都不做。
func (tv *tableView) refresh(sh *tabular.Sheet) {
	sig := tv.signature()
	if sig == tv.sig && tv.rows != nil {
		return
	}
	tv.sig = sig
	q := strings.ToLower(strings.TrimSpace(tv.q))
	if q != "" && tv.low == nil {
		tv.low = make([]string, len(sh.Rows))
		for i, r := range sh.Rows {
			tv.low[i] = strings.ToLower(strings.Join(r, "\x00"))
		}
	}
	lows := make([]string, len(tv.filters))
	for i, f := range tv.filters {
		lows[i] = strings.ToLower(f.val)
	}
	rows := make([]int, 0, len(sh.Rows))
	for r := tv.start; r < len(sh.Rows); r++ {
		if q != "" && !strings.Contains(tv.low[r], q) {
			continue
		}
		ok := true
		for i, f := range tv.filters {
			if !f.match(sh.Cell(r, f.col), lows[i]) {
				ok = false
				break
			}
		}
		if ok {
			rows = append(rows, r)
		}
	}
	if j := colOfID(tv.sort.Column); j >= 0 && j < sh.Cols {
		numeric := true
		for _, r := range rows {
			if s := sh.Cell(r, j); strings.TrimSpace(s) != "" {
				if _, ok := parseNum(s); !ok {
					numeric = false
					break
				}
			}
		}
		desc := tv.sort.Descending
		sort.SliceStable(rows, func(x, y int) bool {
			a, b := sh.Cell(rows[x], j), sh.Cell(rows[y], j)
			ea, eb := strings.TrimSpace(a) == "", strings.TrimSpace(b) == ""
			if ea || eb { // 空值总排在最后,升降序都一样
				return !ea && eb
			}
			var less, greater bool
			if numeric {
				fa, _ := parseNum(a)
				fb, _ := parseNum(b)
				less, greater = fa < fb, fa > fb
			} else {
				la, lb := strings.ToLower(a), strings.ToLower(b)
				less, greater = la < lb, la > lb
			}
			if desc {
				return greater
			}
			return less
		})
	}
	tv.rows = rows
}

func (tv *tableView) clearFilters() {
	tv.filters = nil
	tv.q = ""
}

func (tv *tableView) addFilter(col int, op tblOp, val string) {
	tv.filters = append(tv.filters, tblFilter{col: col, op: op, val: val})
	tv.showFilters = true
}

// cellsText 把选中的格子拼成制表符分隔的文本(表格软件能直接粘贴)。
func (tv *tableView) cellsText(sh *tabular.Sheet) (text string, rows, cols int) {
	var js []int
	for _, id := range tv.list.SelectedColumns() {
		if j := colOfID(id); j >= 0 {
			js = append(js, j)
		}
	}
	if len(js) == 0 {
		return "", 0, 0
	}
	var lines []string
	for _, r := range tv.rows {
		if !tv.sel.Has(r) {
			continue
		}
		cells := make([]string, len(js))
		for k, j := range js {
			cells[k] = sh.Cell(r, j)
		}
		lines = append(lines, strings.Join(cells, "\t"))
	}
	return strings.Join(lines, "\n"), len(lines), len(js)
}

// selStat 选中数字格的统计(个数 / 求和 / 平均);没有数字返回 ok=false。
func (tv *tableView) selStat(sh *tabular.Sheet) (n int, sum float64, ok bool) {
	if tv.sel.Len() == 0 {
		return
	}
	for _, id := range tv.list.SelectedColumns() {
		j := colOfID(id)
		if j < 0 {
			continue
		}
		for _, r := range tv.rows {
			if !tv.sel.Has(r) {
				continue
			}
			if f, isNum := parseNum(sh.Cell(r, j)); isNum {
				n++
				sum += f
			}
		}
	}
	return n, sum, n > 0
}

func fmtNum(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// ───────── 界面 ─────────

func (a *App) tableView(c *ui.Context, tb *ideTab) {
	v, t := tb.vw, a.tk
	wb := v.wb
	if wb == nil || len(wb.Sheets) == 0 {
		a.viewerFail(c, tb, "这个文件里没有可显示的内容")
		return
	}
	if v.sheet < 0 || v.sheet >= len(wb.Sheets) {
		v.sheet = 0
	}
	sh := &wb.Sheets[v.sheet]
	if v.tv == nil || v.tv.sheet != v.sheet {
		v.tv = buildTableView(sh, v.sheet, !v.noHeader)
	} else if v.tv.header != !v.noHeader {
		v.tv.setHeader(sh, !v.noHeader)
	}
	tv := v.tv
	tv.refresh(sh)
	total := len(sh.Rows) - tv.start
	if total < 0 {
		total = 0
	}
	n := len(tv.rows)
	ext := strings.ToLower(path.Ext(tb.path))
	isText := ext == ".csv" || ext == ".tsv"
	nFilter := len(tv.filters)
	if strings.TrimSpace(tv.q) != "" {
		nFilter++
	}

	a.viewerBar(c, func() {
		if k := len(wb.Sheets); k > 1 {
			if k <= 5 {
				items := make([]kit.SegItem, k)
				for i := range wb.Sheets {
					items[i] = kit.SegItem{Text: clipRunes(wb.Sheets[i].Name, 12)}
				}
				if i := kit.Seg(c, items, v.sheet, kit.SegOpt{Key: "tbl-sheets"}); i >= 0 && i != v.sheet {
					v.sheet = i
				}
			} else {
				kit.MenuBtn(kit.Btn(c, clipRunes(sh.Name, 16)+"  ▾", kit.BtnOpt{Small: true, Tip: "切换工作表"}), func(m *ui.Menu) {
					for i := range wb.Sheets {
						if menuItem(m, wb.Sheets[i].Name).Chosen() {
							v.sheet = i
						}
					}
				})
			}
		}
		stat := fmt.Sprintf("%d 行 × %d 列", total, sh.Cols)
		if n != total {
			stat = fmt.Sprintf("%d / %d 行 × %d 列", n, total, sh.Cols)
		}
		if sh.Truncated {
			stat = fmt.Sprintf("仅显示前 %d 行 × %d 列", tabular.MaxRows, tabular.MaxCols)
			if n != total {
				stat = fmt.Sprintf("%d / %d 行(仅载入前 %d 行 × %d 列)", n, total, tabular.MaxRows, tabular.MaxCols)
			}
		}
		if hid := len(tv.allCols) - len(tv.vis); hid > 0 {
			stat += fmt.Sprintf(" · 隐藏 %d 列", hid)
		}
		ui.Text(c, stat+" · "+v.info).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
		if v.note != "" {
			ui.Text(c, "· "+v.note).FontSize(theme.FsCap).TextColor(t.Warn).SingleLine()
		}
		if cnt, sum, ok := tv.selStat(sh); ok {
			s := fmt.Sprintf("选中 %d 个数字 · 求和 %s · 平均 %s", cnt, fmtNum(sum), fmtNum(sum/float64(cnt)))
			ui.Text(c, s).FontSize(theme.FsCap).TextColor(t.AcTx).SingleLine()
		}
	}, func() {
		// 搜索框:所有列里包含这段文字的行
		q := ui.TextInputBase(c.Key("tbl-q-"+tb.id), &tv.q).Width(170).Placeholder("在表格中搜索").Padding(2, 8).Background(t.BG).Border(1, t.Bd2).
			Radius(t.RSm()).FontSize(theme.FsCap + .5).Label("在表格中搜索").TextColor(t.Tx1)
		_ = q
		if tv.q != "" {
			if kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "清除搜索"}).Clicked() {
				tv.q = ""
			}
		}
		fl := kit.Btn(c, "筛选", kit.BtnOpt{Small: true, Icon: "filter", Tip: "按列设置筛选条件"})
		if nFilter > 0 {
			fl = kit.Btn(c, fmt.Sprintf("筛选 · %d", len(tv.filters)), kit.BtnOpt{Small: true, Icon: "filter", Variant: kit.BtnSoft, Tip: "按列设置筛选条件"})
		}
		if fl.Clicked() {
			tv.showFilters = !tv.showFilters
			if tv.showFilters && len(tv.filters) == 0 && len(tv.vis) > 0 {
				tv.addFilter(tv.vis[0], tblContains, "")
			}
		}
		if hid := len(tv.allCols) - len(tv.vis); hid > 0 {
			kit.MenuBtn(kit.Btn(c, "列", kit.BtnOpt{Small: true, Icon: "eyeOff", Tip: "显示 / 隐藏列"}), func(m *ui.Menu) {
				a.tblColumnsMenu(m, tv)
			})
		} else if len(tv.allCols) > 1 {
			kit.MenuBtn(kit.Btn(c, "列", kit.BtnOpt{Small: true, Icon: "eye", Tip: "显示 / 隐藏列"}), func(m *ui.Menu) {
				a.tblColumnsMenu(m, tv)
			})
		}
		ui.Text(c, "首行作表头").FontSize(theme.FsCap).TextColor(t.Tx3)
		if kit.Tog(c, !v.noHeader, "tbl-hdr") {
			v.noHeader = !v.noHeader
		}
		if isText {
			a.tableModeSeg(c, tb)
		}
		kit.Ib(c, "folder", kit.IbOpt{Small: true, Tip: "在访达中显示", Disabled: a.proj == ""}).OnClick(func() { a.revealInFinder(a.absPath(tb.path)) })
	})
	if tv.showFilters || len(tv.filters) > 0 {
		a.tblFilterBar(c, tb, tv, sh)
	}
	if total == 0 {
		ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "这张表是空的").TextColor(t.Tx3)
		})
		return
	}
	if n == 0 {
		ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Text(c, "没有符合条件的行").TextColor(t.Tx3)
			if kit.Btn(c, "清除搜索与筛选", kit.BtnOpt{Small: true}).Clicked() {
				tv.clearFilters()
			}
		})
		return
	}

	tv.list.HeaderMenu = func(m *ui.Menu, col int) { a.tblHeaderMenu(m, tv, sh, col) }
	tv.list.CellMenu = func(m *ui.Menu, row, col int) { a.tblCellMenu(m, tv, sh, row, col) }
	// Key 必须在构造 Table 之前给(构造之后再 .Key 会 panic:Key on a Table, whose state was initialized)
	ui.Table(c.Key("tbl-"+tb.id+"-"+strconv.Itoa(tv.sheet)), tv.list, tv.cols, n, func(row, col int) {
		if row < 0 || row >= len(tv.rows) {
			return
		}
		r := tv.rows[row]
		if col == 0 {
			ui.Text(c, strconv.Itoa(r+1)).FontSize(theme.FsCap).TextColor(t.Tx4).SingleLine()
			return
		}
		if col-1 >= len(tv.vis) {
			return
		}
		ui.Text(c, sh.Cell(r, tv.vis[col-1])).FontSize(theme.Px(12)).TextColor(t.Tx1).SingleLine()
	}).Grow(1).MinHeight(0)
	if tv.list.Shortcut(c, ui.Cmd, ui.KeyC) {
		a.tblCopy(tv, sh)
	}
}

func (a *App) tblCopy(tv *tableView, sh *tabular.Sheet) {
	text, r, k := tv.cellsText(sh)
	if text == "" {
		a.toast = "先点选要复制的单元格"
		return
	}
	a.copyOut(text, fmt.Sprintf(" %d 行 × %d 列", r, k))
}

// tblColumnsMenu 是「列」菜单:逐列勾选显示 / 隐藏。
func (a *App) tblColumnsMenu(m *ui.Menu, tv *tableView) {
	if len(tv.allCols) > len(tv.vis) {
		if menuItem(m, "显示全部列").Chosen() {
			tv.hidden = map[int]bool{}
			tv.rebuildCols()
		}
		m.Separator()
	}
	max := len(tv.allCols)
	if max > 60 {
		max = 60
	}
	for j := 0; j < max; j++ {
		it := m.Item(clipRunes(tv.allCols[j].Title, 30)).Checked(!tv.hidden[j])
		if it.Chosen() {
			if tv.hidden[j] {
				delete(tv.hidden, j)
			} else if len(tv.vis) > 1 { // 至少留一列
				tv.hidden[j] = true
			}
			tv.rebuildCols()
		}
	}
}

func (a *App) tblHeaderMenu(m *ui.Menu, tv *tableView, sh *tabular.Sheet, col int) {
	if col <= 0 || col-1 >= len(tv.vis) {
		if len(tv.allCols) > len(tv.vis) && menuItem(m, "显示全部列").Chosen() {
			tv.hidden = map[int]bool{}
			tv.rebuildCols()
		}
		return
	}
	j := tv.vis[col-1]
	id := tv.allCols[j].ID
	if m.Item("升序").Checked(tv.sort.Column == id && !tv.sort.Descending).Chosen() {
		tv.sort = ui.SortOrder{Column: id}
	}
	if m.Item("降序").Checked(tv.sort.Column == id && tv.sort.Descending).Chosen() {
		tv.sort = ui.SortOrder{Column: id, Descending: true}
	}
	if tv.sort.Column != "" && menuItem(m, "取消排序").Chosen() {
		tv.sort = ui.SortOrder{}
	}
	m.Separator()
	if menuItem(m, "筛选此列…").Chosen() {
		tv.addFilter(j, tblContains, "")
	}
	if menuItem(m, "复制整列").Chosen() {
		var lines []string
		for _, r := range tv.rows {
			lines = append(lines, sh.Cell(r, j))
		}
		a.copyOut(strings.Join(lines, "\n"), fmt.Sprintf("「%s」列(%d 行)", tv.allCols[j].Title, len(lines)))
	}
	if len(tv.vis) > 1 && menuItem(m, "隐藏此列").Chosen() {
		tv.hidden[j] = true
		tv.rebuildCols()
	}
	if len(tv.allCols) > len(tv.vis) && menuItem(m, "显示全部列").Chosen() {
		tv.hidden = map[int]bool{}
		tv.rebuildCols()
	}
}

func (a *App) tblCellMenu(m *ui.Menu, tv *tableView, sh *tabular.Sheet, row, col int) {
	if row < 0 || row >= len(tv.rows) {
		return
	}
	r := tv.rows[row]
	if tv.sel.Len() > 1 {
		if menuItem(m, "复制所选  ⌘C").Chosen() {
			a.tblCopy(tv, sh)
		}
	}
	if col <= 0 || col-1 >= len(tv.vis) {
		if menuItem(m, "复制整行").Chosen() {
			a.copyOut(strings.Join(sh.Rows[r], "\t"), "整行")
		}
		return
	}
	j := tv.vis[col-1]
	val := sh.Cell(r, j)
	if menuItem(m, "复制单元格").Chosen() {
		a.copyOut(val, "单元格")
	}
	if menuItem(m, "复制整行").Chosen() {
		a.copyOut(strings.Join(sh.Rows[r], "\t"), "整行")
	}
	m.Separator()
	if strings.TrimSpace(val) == "" {
		if menuItem(m, "只看此列为空的行").Chosen() {
			tv.addFilter(j, tblEmpty, "")
		}
		return
	}
	short := clipRunes(val, 16)
	if menuItem(m, "只看「"+short+"」").Chosen() {
		tv.addFilter(j, tblEq, val)
	}
	if menuItem(m, "排除「"+short+"」").Chosen() {
		tv.addFilter(j, tblNe, val)
	}
}

// tblFilterBar 是筛选条:每条「列 · 条件 · 值」,之间是「且」。
func (a *App) tblFilterBar(c *ui.Context, tb *ideTab, tv *tableView, sh *tabular.Sheet) {
	t := a.tk
	ui.Column(c).Shrink(0).Gap(4).Padding(6, 10).Background(t.Fill1).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		del := -1
		for i := range tv.filters {
			f := &tv.filters[i]
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, map[bool]string{true: "条件", false: "且"}[i == 0]).FontSize(theme.FsCap).TextColor(t.Tx4).Width(28)
				kit.MenuBtn(kit.Btn(c, clipRunes(tv.colTitle(f.col), 18)+"  ▾", kit.BtnOpt{Small: true, Tip: "选择列"}), func(m *ui.Menu) {
					for j := range tv.allCols {
						if j >= 80 {
							break
						}
						if m.Item(clipRunes(tv.allCols[j].Title, 30)).Checked(j == f.col).Chosen() {
							f.col = j
						}
					}
				})
				kit.MenuBtn(kit.Btn(c, tblOpNames[f.op]+"  ▾", kit.BtnOpt{Small: true, Tip: "条件"}), func(m *ui.Menu) {
					for k, name := range tblOpNames {
						if m.Item(name).Checked(tblOp(k) == f.op).Chosen() {
							f.op = tblOp(k)
						}
					}
				})
				if f.op.needsValue() {
					ui.TextInputBase(c.Key("tbl-f-"+tb.id+"-"+strconv.Itoa(i)), &f.val).Width(180).Placeholder("值").Padding(2, 8).Background(t.BG).Border(1, t.Bd2).
						Radius(t.RSm()).FontSize(theme.FsCap + .5).Label("筛选值").TextColor(t.Tx1)
				}
				if kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "删除这条条件"}).Clicked() {
					del = i
				}
			})
		}
		if del >= 0 {
			tv.filters = append(tv.filters[:del], tv.filters[del+1:]...)
		}
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if kit.Btn(c, "添加条件", kit.BtnOpt{Small: true, Icon: "plus"}).Clicked() && len(tv.vis) > 0 {
				tv.addFilter(tv.vis[0], tblContains, "")
			}
			if len(tv.filters) > 0 || tv.q != "" {
				if kit.Btn(c, "清除全部", kit.BtnOpt{Small: true}).Clicked() {
					tv.clearFilters()
				}
			}
			ui.Box(c).Grow(1)
			if kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "收起筛选条(条件仍然生效)"}).Clicked() {
				tv.showFilters = false
			}
		})
	})
}

// tableModeSeg 是 csv / tsv 的「预览 | 源码」分段:预览 = 这个表格查看器,源码 = 文本编辑器(可编辑保存)。
func (a *App) tableModeSeg(c *ui.Context, tb *ideTab) {
	if i := kit.Seg(c, []kit.SegItem{{Text: "预览", Tip: "表格视图,可搜索、筛选、排序"}, {Text: "源码", Tip: "用文本编辑器打开原文,可以编辑与保存"}}, 0, kit.SegOpt{Key: "tbl-mode"}); i == 1 {
		a.viewerAsText(tb)
	}
}

// tableTextSeg 是 csv / tsv 在文本编辑器里的「预览 | 源码」分段,选「预览」回到表格查看器。
func (a *App) tableTextSeg(c *ui.Context, tb *ideTab) {
	if i := kit.Seg(c, []kit.SegItem{{Text: "预览", Tip: "表格视图,可搜索、筛选、排序"}, {Text: "源码", Tip: "显示 CSV 原文"}}, 1, kit.SegOpt{Key: "tbl-mode"}); i == 0 {
		a.textAsTable(tb)
	}
}

// textAsTable 把 csv / tsv 的文本标签换回表格查看器(查看器从磁盘读,有未保存修改时先要求保存)。
func (a *App) textAsTable(tb *ideTab) {
	if tb.buf != nil && tb.buf.String() != tb.saved {
		a.toast = "有未保存的修改,先保存(⌘S)再切到预览"
		return
	}
	viewLog.Info("改用表格预览", "path", tb.path)
	tb.viewer, tb.vw = "table", &viewerState{}
	tb.buf, tb.st = nil, nil
}
