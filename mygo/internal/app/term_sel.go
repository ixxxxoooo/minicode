package app

import (
	"math"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/egoist/mygo/ui"

	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/term"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 终端的鼠标选择 / 复制、搜索、可点击链接(对标 VS Code 终端与 iTerm2):
//   - 拖动选择,双击选词,三击选行;⌘C 复制选区,⌘A 全选;右键菜单有复制 / 粘贴 / 全选 / 搜索 / 清屏。
//   - ⌘F 在终端里搜索(含历史),Enter 下一个,⇧Enter 上一个,Esc 关闭;命中全部高亮,当前命中更亮。
//   - ⌘ 悬停在 URL 或「路径:行:列」上出现下划线与手型,⌘ 点击:URL 用系统浏览器打开,文件在编辑器里跳到那一行。

// cellPos 是画面里的一格:row 是 Screen.Lines 的下标(含历史),col 是列。
type cellPos struct{ row, col int }

// termSel 是鼠标选区(锚点 a 到当前点 b,含两端)。
type termSel struct {
	on   bool
	a, b cellPos
	drag bool
}

// termHit 是一处搜索命中(c1 不含)。
type termHit struct{ row, c0, c1 int }

// termFind 是终端内搜索的状态。
type termFind struct {
	open      bool
	openedAt  time.Time
	closingAt time.Time
	query     string
	cs        bool // 区分大小写
	word      bool // 全词
	re        bool // 正则
	hits      []termHit
	cur       int
	key       string // 上次计算命中时的「查询|大小写|画面版本」
	focus     bool   // 下一帧把焦点给搜索框
}

// termLink 是一处可点击的链接。
type termLink struct {
	kind      string // url / file
	target    string // URL 或相对项目的路径
	line, col int
	c0, c1    int // 所在行的列范围(c1 不含)
}

// lineCells 把一行拆成「每列一个字符」:宽字符占两列(第二列为空串),组合字符并入前一格。
func lineCells(ln term.Line) []string {
	var cells []string
	for _, r := range ln {
		for _, ch := range r.Text {
			g := string(ch)
			w := ansi.StringWidth(g)
			switch {
			case w == 0:
				if n := len(cells); n > 0 {
					cells[n-1] += g
				}
			default:
				cells = append(cells, g)
				for i := 1; i < w; i++ {
					cells = append(cells, "")
				}
			}
		}
	}
	return cells
}

// norm 返回按先后排好的选区两端。
func (s termSel) norm() (from, to cellPos) {
	from, to = s.a, s.b
	if to.row < from.row || to.row == from.row && to.col < from.col {
		from, to = to, from
	}
	return
}

// span 是选区在第 row 行上覆盖的列范围(c1 不含);没有覆盖返回 false。
func (s termSel) span(row, cols int) (c0, c1 int, ok bool) {
	if !s.on {
		return 0, 0, false
	}
	from, to := s.norm()
	if row < from.row || row > to.row {
		return 0, 0, false
	}
	c0, c1 = 0, cols
	if row == from.row {
		c0 = from.col
	}
	if row == to.row {
		c1 = to.col + 1
	}
	return c0, c1, c1 > c0
}

// selectedText 取选区文字:每行去掉尾部空白,行间用换行。
func (tp *termPanel) selectedText() string {
	if !tp.sel.on || tp.scr == nil {
		return ""
	}
	from, to := tp.sel.norm()
	var rows []string
	for y := from.row; y <= to.row && y < len(tp.scr.Lines); y++ {
		cells := lineCells(tp.scr.Lines[y])
		c0, c1 := 0, len(cells)
		if y == from.row {
			c0 = from.col
		}
		if y == to.row {
			c1 = min(to.col+1, len(cells))
		}
		var s string
		if c0 < c1 && c0 < len(cells) {
			s = strings.Join(cells[c0:c1], "")
		}
		rows = append(rows, strings.TrimRight(s, " "))
	}
	return strings.Join(rows, "\n")
}

// rowsText 取 [from, to) 行的文字:每行去尾部空白,首尾空行去掉。
func (tp *termPanel) rowsText(from, to int) string {
	if tp.scr == nil {
		return ""
	}
	from, to = max(from, 0), min(to, len(tp.scr.Lines))
	var rows []string
	for y := from; y < to; y++ {
		rows = append(rows, strings.TrimRight(strings.Join(lineCells(tp.scr.Lines[y]), ""), " "))
	}
	return strings.Trim(strings.Join(rows, "\n"), "\n")
}

// copyCmdOutput 复制一条命令的输出(命令装饰圆点的点击动作)。
func (a *App) copyCmdOutput(tp *termPanel, from, to int) {
	txt := tp.rowsText(from, to)
	if txt == "" {
		a.toast = "这条命令没有输出"
		return
	}
	a.copyText = txt
	a.redraw() // 点圆点 / 菜单里没有上下文可直接写剪贴板,要一次重绘让 shortcuts 去写
	a.toast = "已复制命令输出(" + itoa(strings.Count(txt, "\n")+1) + " 行)"
	log.Info("复制命令输出", "lines", strings.Count(txt, "\n")+1)
}

func (tp *termPanel) selectAll() {
	if tp.scr == nil || len(tp.scr.Lines) == 0 {
		return
	}
	last := len(tp.scr.Lines) - 1
	tp.sel = termSel{on: true, a: cellPos{0, 0}, b: cellPos{last, max(0, len(lineCells(tp.scr.Lines[last]))-1)}}
}

// wordAt 选中 (row, col) 所在的「词」(字母数字与路径 / URL 常见符号连在一起)。
func (tp *termPanel) wordAt(p cellPos) {
	if tp.scr == nil || p.row >= len(tp.scr.Lines) {
		return
	}
	cells := lineCells(tp.scr.Lines[p.row])
	isW := func(i int) bool {
		if i < 0 || i >= len(cells) {
			return false
		}
		g := cells[i]
		if g == "" { // 宽字符后半格:跟随前一格
			return isW2(cells, i)
		}
		r, _ := utf8.DecodeRuneInString(g)
		return r > 127 || r == '_' || r == '-' || r == '.' || r == '/' || r == ':' || r == '~' || r == '@' || r == '%' || r == '+' || r == '=' || r == '?' || r == '&' || r == '#' ||
			(r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
	}
	if !isW(p.col) {
		tp.sel = termSel{on: true, a: p, b: p}
		return
	}
	l, r := p.col, p.col
	for isW(l - 1) {
		l--
	}
	for isW(r + 1) {
		r++
	}
	tp.sel = termSel{on: true, a: cellPos{p.row, l}, b: cellPos{p.row, r}}
}

func isW2(cells []string, i int) bool {
	for j := i - 1; j >= 0; j-- {
		if cells[j] != "" {
			r, _ := utf8.DecodeRuneInString(cells[j])
			return r > 127
		}
	}
	return false
}

func (tp *termPanel) selectLine(p cellPos) {
	if tp.scr == nil || p.row >= len(tp.scr.Lines) {
		return
	}
	n := len(lineCells(tp.scr.Lines[p.row]))
	tp.sel = termSel{on: true, a: cellPos{p.row, 0}, b: cellPos{p.row, max(0, n-1)}}
}

// termCell 把面板内的坐标换成画面格(含滚动偏移与内边距)。
func (a *App) termCell(tp *termPanel, x, y float32) cellPos {
	const padX, padY = 10, 6
	if tp.scr == nil || tp.cellW <= 0 {
		return cellPos{}
	}
	row := int(math.Floor(float64((y + tp.st.Y - padY) / termLineH)))
	col := int(math.Floor(float64((x - padX) / tp.cellW)))
	return cellPos{row: min(max(row, 0), max(0, len(tp.scr.Lines)-1)), col: max(col, 0)}
}

// ───────── 链接 ─────────

var (
	reURL  = regexp.MustCompile(`https?://[^\s<>"'` + "`" + `)\]}]+`)
	reFile = regexp.MustCompile(`(?:[A-Za-z0-9_.~-]*/)*[A-Za-z0-9_.-]+\.[A-Za-z0-9]{1,8}(?::\d+(?::\d+)?)?`)
)

// linkAt 找 (row, col) 处的链接:先看 URL,再看存在于项目里的「路径[:行[:列]]」。
func (a *App) linkAt(tp *termPanel, p cellPos) *termLink {
	if tp.scr == nil || p.row >= len(tp.scr.Lines) {
		return nil
	}
	cells := lineCells(tp.scr.Lines[p.row])
	// 字节偏移 → 列
	var sb strings.Builder
	var colAt []int // 每个字节所属的列
	for c, g := range cells {
		for range []byte(g) {
			colAt = append(colAt, c)
		}
		sb.WriteString(g)
	}
	text := sb.String()
	toCols := func(b0, b1 int) (int, int) {
		if b0 >= len(colAt) {
			return len(cells), len(cells)
		}
		c1 := len(cells)
		if b1 < len(colAt) {
			c1 = colAt[b1]
		}
		return colAt[b0], c1
	}
	for _, m := range reURL.FindAllStringIndex(text, -1) {
		c0, c1 := toCols(m[0], m[1])
		if p.col >= c0 && p.col < c1 {
			return &termLink{kind: "url", target: strings.TrimRight(text[m[0]:m[1]], ".,;:"), c0: c0, c1: c1}
		}
	}
	for _, m := range reFile.FindAllStringIndex(text, -1) {
		c0, c1 := toCols(m[0], m[1])
		if p.col < c0 || p.col >= c1 {
			continue
		}
		raw := text[m[0]:m[1]]
		path, line, col := splitPos(raw)
		if rel, ok := a.projectRel(path); ok {
			return &termLink{kind: "file", target: rel, line: line, col: col, c0: c0, c1: c1}
		}
	}
	return nil
}

// splitPos 拆开「路径:行:列」。
func splitPos(s string) (path string, line, col int) {
	parts := strings.Split(s, ":")
	path = parts[0]
	if len(parts) > 1 {
		line = atoi(parts[1])
	}
	if len(parts) > 2 {
		col = atoi(parts[2])
	}
	return
}

// projectRel 把终端里的路径换成相对项目的路径,且文件必须真的存在(免得把版本号、域名当成文件)。
func (a *App) projectRel(p string) (string, bool) {
	p = strings.TrimPrefix(p, "./")
	if a.proj != "" && filepath.IsAbs(p) {
		r, err := filepath.Rel(a.proj, p)
		if err != nil || strings.HasPrefix(r, "..") {
			return "", false
		}
		p = r
	}
	if p == "" || a.ide.fs == nil {
		return "", false
	}
	if _, err := a.ide.fs.Read(p); err != nil {
		return "", false
	}
	return p, true
}

// openLink 打开链接:URL 交给系统浏览器,文件在编辑器里跳到行列。
func (a *App) openLink(l *termLink) {
	switch l.kind {
	case "url":
		if err := exec.Command("open", l.target).Start(); err != nil {
			a.toast = "无法打开链接:" + err.Error()
		}
	case "file":
		a.revealWorkspace()
		line := max(l.line, 1)
		a.jumpTo(l.target, line, max(l.col, 1)-1, max(l.col, 1)-1)
	}
}

// ───────── 搜索 ─────────

// refreshFind 按当前画面重算命中(查询、大小写、画面版本变了才算)。
func (tp *termPanel) refreshFind() {
	f := &tp.find
	if !f.open || tp.scr == nil {
		return
	}
	key := f.query + "|" + map[bool]string{true: "1", false: "0"}[f.cs] +
		map[bool]string{true: "1", false: "0"}[f.word] +
		map[bool]string{true: "1", false: "0"}[f.re] + "|" + itoa(int(tp.scr.Ver))
	if key == f.key {
		return
	}
	f.key = key
	f.hits = f.hits[:0]
	if f.query == "" {
		f.cur = 0
		return
	}
	var re *regexp.Regexp
	if f.re && f.query != "" {
		pat := f.query
		if !f.cs {
			pat = "(?i)" + pat
		}
		var err error
		re, err = regexp.Compile(pat)
		if err != nil {
			f.cur = 0
			return
		}
	}
	for y, ln := range tp.scr.Lines {
		cells := lineCells(ln)
		// 逐格拼出文本,同时记下每个字符起始的列
		var sb strings.Builder
		var cols []int
		for c, g := range cells {
			if g == "" {
				continue
			}
			s := g
			if !f.cs {
				s = strings.ToLower(s)
			}
			for range []rune(s) {
				cols = append(cols, c)
			}
			sb.WriteString(s)
		}
		text := sb.String()
		matchAt := func(b0, b1 int) {
			c0 := 0
			if b0 < len(cols) {
				c0 = cols[b0]
			}
			c1 := len(cells)
			if b1 < len(cols) {
				c1 = cols[b1]
			}
			f.hits = append(f.hits, termHit{y, c0, c1})
		}
		if re != nil {
			locs := re.FindAllStringIndex(text, -1)
			for _, m := range locs {
				matchAt(m[0], m[1])
			}
			continue
		}
		q := f.query
		if !f.cs {
			q = strings.ToLower(q)
		}
		rs := []rune(text)
		qr := []rune(q)
		qc := len(qr)
		for i := 0; i+qc <= len(rs); i++ {
			if string(rs[i:i+qc]) != string(qr) {
				continue
			}
			if f.word && !wordBound(rs, i, i+qc) {
				continue
			}
			c0 := cols[i]
			c1 := len(cells)
			if i+qc < len(cols) {
				c1 = cols[i+qc]
			}
			f.hits = append(f.hits, termHit{y, c0, c1})
			i += qc - 1
		}
	}
	if f.cur >= len(f.hits) {
		f.cur = max(0, len(f.hits)-1)
	}
}

// findStep 跳到下一个(dir=1)/ 上一个(dir=-1)命中并把它滚进视野。
func (a *App) findStep(tp *termPanel, dir int) {
	f := &tp.find
	tp.refreshFind()
	if len(f.hits) == 0 {
		return
	}
	f.cur = (f.cur + dir + len(f.hits)) % len(f.hits)
	h := f.hits[f.cur]
	top := float32(h.row)*termLineH + 6
	tp.st.Y = max(0, top-tp.viewH/2)
}

func (a *App) openTermFind() {
	tp := a.ide.tpc()
	if tp == nil {
		return
	}
	tp.find.open, tp.find.focus, tp.find.openedAt = true, true, a.now()
	tp.find.closingAt = time.Time{}
	if tp.sel.on { // 选中了文字:作为初始查询(同 VS Code)
		if s := tp.selectedText(); s != "" && !strings.Contains(s, "\n") {
			tp.find.query = s
		}
	}
	tp.find.key = ""
}

func (a *App) closeTermFind() {
	tp := a.ide.tpc()
	if tp.find.open && tp.find.closingAt.IsZero() {
		tp.find.closingAt = a.now()
		a.frameSoon()
	}
	if tp.find.closingAt.IsZero() || time.Since(tp.find.closingAt) >= kit.DurFast {
		tp.find.open = false
		tp.find.openedAt, tp.find.closingAt = time.Time{}, time.Time{}
		tp.focusIt = true
		return
	}
	tp.focusIt = true
}

// termFindBar 是终端右上角的搜索条。
func (a *App) termFindBar(c *ui.Context, tp *termPanel) {
	t := a.tk
	f := &tp.find
	// 退场动画放完要真正收尾(open=false):原先只在「再调一次 closeTermFind」时才收尾,
	// 搜索条画没了,状态还是打开 → 命中高亮残留、Esc 仍被拦截
	if f.open && !f.closingAt.IsZero() && time.Since(f.closingAt) >= kit.DurFast {
		f.open = false
		f.openedAt, f.closingAt = time.Time{}, time.Time{}
		tp.focusIt = true
	}
	tp.refreshFind()
	if !f.open && f.closingAt.IsZero() {
		return
	}
	bar := ui.Row(c).Key("tfind").Absolute().Top(6).Right(16).Height(theme.Px(30)).Padding(0, 4, 0, 8).Gap(4).AlignItems(ui.Center).
		Background(t.Raised).Radius(t.RMd()).Border(1, t.Bd2)
	if !f.closingAt.IsZero() {
		bar = kit.PopOut(bar, f.closingAt)
		a.frameSoon()
	} else {
		bar = kit.PopIn(bar, f.openedAt)
	}
	defer func() { tp.addOverlay(bar.Bounds()) }()
	bar.Children(func() {
		in := ui.TextInputBase(c.Key("tfind-in"), &f.query).Width(150).FontSize(theme.FsBody).Placeholder("在终端中查找").PlaceholderColor(t.Tx4)
		if f.focus {
			in.Focus()
			f.focus = false
		}
		in.OnSubmit(func() { a.findStep(tp, 1) })
		cnt := "无结果"
		switch {
		case f.query == "":
			cnt = ""
		case len(f.hits) > 0:
			cnt = itoa(f.cur+1) + "/" + itoa(len(f.hits))
		}
		ui.Text(c, cnt).FontSize(theme.FsCap).TextColor(map[bool]ui.Color{true: t.Err, false: t.Tx3}[cnt == "无结果"]).FontFeatures("tnum").MinWidth(40)
		a.toggleBtn(c, "Aa", &f.cs, "区分大小写")
		a.toggleBtn(c, "ab", &f.word, "全词")
		a.toggleBtn(c, ".*", &f.re, "正则")
		if kit.Ib(c, "chevD", kit.IbOpt{Small: true, Tip: "下一个 Enter"}).Clicked() {
			a.findStep(tp, 1)
		}
		if kit.Ib(c, "chevU", kit.IbOpt{Small: true, Tip: "上一个 ⇧Enter"}).Clicked() {
			a.findStep(tp, -1)
		}
		if kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭 Esc"}).Clicked() {
			a.closeTermFind()
		}
	})
}

// rowDeco 是一行上要叠的装饰:选区、搜索命中、链接下划线(列范围,c1 不含)。
type rowDeco struct {
	sel  [2]int
	hits []rowHit
	link [2]int
	cmd  int8 // 命令装饰:0 无 / 1 成功 / -1 失败
	// 命令装饰的详情:悬停提示的退出码、点击复制该命令输出
	cmdExit int
	cmdCopy func()
}

type rowHit struct {
	c0, c1 int
	cur    bool
}

func (a *App) rowDeco(tp *termPanel, row, cols int) rowDeco {
	var d rowDeco
	if c0, c1, ok := tp.sel.span(row, cols); ok {
		d.sel = [2]int{c0, c1}
	}
	if tp.find.open {
		for i, h := range tp.find.hits {
			if h.row == row {
				d.hits = append(d.hits, rowHit{h.c0, h.c1, i == tp.find.cur})
			}
		}
	}
	if tp.link != nil && tp.hoverOn && tp.hover.row == row {
		d.link = [2]int{tp.link.c0, tp.link.c1}
	}
	if tp.t != nil && tp.scr != nil {
		abs := tp.scr.LineBase + row
		for _, b := range tp.t.CommandBlocks() {
			// 圆点落在命令行(输出第一行的上一行)上;命令还在跑的块先不画
			if !b.Done || b.StartRow-1 != abs {
				continue
			}
			// 清屏(⌃L)后旧块指向的行已经是空白,不画
			if row < len(tp.scr.Lines) && strings.TrimSpace(strings.Join(lineCells(tp.scr.Lines[row]), "")) == "" {
				break
			}
			d.cmd, d.cmdExit = 1, b.Exit
			if !b.ExitOK {
				d.cmd = -1
			}
			from, to := b.StartRow-tp.scr.LineBase, b.EndRow-tp.scr.LineBase
			d.cmdCopy = func() { a.copyCmdOutput(tp, from, to) }
			break
		}
	}
	return d
}

func wordBound(rs []rune, a, b int) bool {
	isW := func(r rune) bool {
		return r == '_' || r == '-' || unicode.IsLetter(r) || unicode.IsDigit(r)
	}
	if a > 0 && isW(rs[a-1]) {
		return false
	}
	if b < len(rs) && isW(rs[b]) {
		return false
	}
	return true
}

// termMenu 是终端的右键菜单。
func (a *App) termMenu(c *ui.Context, body ui.Element, tp *termPanel) {
	body.ContextMenu(func(m *ui.Menu) {
		has := tp.selectedText() != ""
		live := tp.t != nil
		if menuItem(m, "复制").Trailing("⌘C").Disabled(!has).Tooltip(map[bool]string{true: "先在终端里选中文字"}[!has]).Chosen() {
			a.copyText = tp.selectedText()
			a.toast = "已复制选中的终端文字"
			a.redraw()
		}
		if menuItem(m, "粘贴").Trailing("⌘V").Disabled(!live).Tooltip(map[bool]string{true: "终端还没启动"}[!live]).Chosen() && live {
			a.termPasteReq = true
			a.redraw()
		}
		if menuItem(m, "全选").Trailing("⌘A").Chosen() {
			tp.selectAll()
		}
		m.Separator()
		if menuItem(m, "查找…").Trailing("⌘F").Chosen() {
			a.openTermFind()
		}
		if menuItem(m, "清屏").Trailing("⌘K").Disabled(!live).Tooltip(map[bool]string{true: "终端还没启动"}[!live]).Chosen() && live {
			tp.t.ClearCmdBlocks()
			tp.t.SendText("\x0c") // Ctrl-L
		}
		m.Separator()
		if menuItem(m, "新建终端").Disabled(a.proj == "").Tooltip(map[bool]string{true: "演示模式没有真实终端"}[a.proj == ""]).Chosen() {
			a.newTerm()
		}
		if menuItem(m, "关闭终端").Disabled(len(a.ide.tps) < 2).Tooltip(map[bool]string{true: "至少保留一个终端"}[len(a.ide.tps) < 2]).Chosen() {
			a.askCloseTermTab(a.ide.tpSel)
		}
		m.Separator()
		dir := tp.cwd
		if dir == "" {
			dir = a.proj
		}
		if menuItem(m, "在访达中打开当前目录").Disabled(dir == "").Tooltip(map[bool]string{true: "演示模式没有真实目录"}[dir == ""]).Chosen() {
			a.revealInFinder(dir)
		}
		if menuItem(m, "终端设置…").Chosen() {
			a.openSettings("terminal")
		}
	})
}

// termShortcuts 终端相关的快捷键:⌘F 查找、Esc 关查找、⇧Enter 上一个、⌘+ / ⌘- / ⌘0 缩放字号。
func (a *App) termShortcuts(c *ui.Context) bool {
	if !a.ide.bottomOpen || a.screen != ScreenWork || len(a.ide.tps) == 0 && a.proj == "" {
		return false
	}
	tp := a.ide.tpc()
	if tp == nil {
		return false
	}
	switch {
	case tp.find.open && c.Shortcut(0, ui.KeyEscape):
		a.closeTermFind()
	case tp.find.open && c.Shortcut(ui.Shift, ui.KeyEnter):
		a.findStep(tp, -1)
	case (a.termHasFocus || tp.find.open) && a.shortcutHit(c, "termFind", ui.Cmd, ui.KeyF):
		a.openTermFind()
	case a.termHasFocus && c.Shortcut(ui.Cmd, ui.KeyEqual), a.termHasFocus && c.Shortcut(ui.Cmd|ui.Shift, ui.KeyEqual):
		a.setTermFont(a.prefs.Int(pTermFont, 12) + 1)
	case a.termHasFocus && c.Shortcut(ui.Cmd, ui.KeyMinus):
		a.setTermFont(a.prefs.Int(pTermFont, 12) - 1)
	case a.termHasFocus && c.Shortcut(ui.Cmd, ui.Key0):
		a.setTermFont(12)
	default:
		return false
	}
	return true
}

// setTermFont 改终端字号(9–24),写偏好并立刻生效。
func (a *App) setTermFont(n int) {
	n = min(max(n, 9), 24)
	a.prefs.Set(pTermFont, n)
	a.applyBehavior()
}
