package app

import (
	"strings"

	"github.com/egoist/mygo/ui/richtext"
)

// Markdown 预览里的表格编辑(设计稿 ide-md「表格:就地编辑」)。
//
// richtext 编辑器没有「表格」这种块:读入时表格会被拍成用 Tab 分隔的普通段落,序列化回来就丢了 |---| 分隔行。
// 所以含(顶层)表格的文档按块拆成若干「段」:正文段仍交给 richtext,表格段是单独的可编辑网格,
// 每段各自写回,表格前后的正文原样不动。

// mdSeg 是文档里的一段:正文(富文本)或一张表格(网格)。
// pre / raw / post 三者首尾相接恰好是原文:pre = 开头的空行,post = 末尾的换行与空行,raw = 中间的内容。
type mdSeg struct {
	table bool
	pre   string
	raw   string
	post  string

	doc   *richtext.Document // 正文段的富文本文档
	split mdBodySplit        // 正文段读入时切的块(写回时只替换变了的块)
	h     float32            // 正文段编辑器当前高度(随内容量出来,逐帧收敛)
	tbl   *mdTable           // 表格段的数据
	dirty bool               // 这一段的内容被改过(写回时才补段间空行)
}

// mdTable 是一张 GFM 表格。单元格内容按 Markdown 源码原样保存(**粗体**、`代码`、链接都不被改写)。
type mdTable struct {
	head   []string
	align  []byte // 每列:0 默认 / 'l' / 'c' / 'r'
	rows   [][]string
	edited bool // 单元格或结构改过 → 写回时重新序列化(否则保持原字节)
}

// mdLinesKeep 按行切开,每行保留自己的换行符。
func mdLinesKeep(s string) []string {
	var out []string
	for len(s) > 0 {
		k := strings.IndexByte(s, '\n')
		if k < 0 {
			out = append(out, s)
			break
		}
		out = append(out, s[:k+1])
		s = s[k+1:]
	}
	return out
}

// mdIndent 是行首的缩进空格数;遇到 Tab 按 4 计。
func mdIndent(s string) int {
	n := 0
	for _, r := range s {
		switch r {
		case ' ':
			n++
		case '\t':
			return 4
		default:
			return n
		}
	}
	return n
}

// mdSplitRow 把一行表格切成单元格:去掉首尾的 |,按「未转义的 |」切开(\| 原样留在单元格里),单元格去首尾空白。
func mdSplitRow(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	if strings.HasSuffix(s, "|") && !strings.HasSuffix(s, `\|`) {
		s = s[:len(s)-1]
	}
	var cells []string
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '\\' && i+1 < len(s) && s[i+1] == '|' {
			cur.WriteString(`\|`)
			i++
			continue
		}
		if ch == '|' {
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(ch)
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

// mdDelimCell 是不是分隔行里的一格(:?-+:?)。
func mdDelimCell(c string) bool {
	c = strings.TrimPrefix(c, ":")
	c = strings.TrimSuffix(c, ":")
	return c != "" && strings.Trim(c, "-") == ""
}

// mdTableAt 判断第 i 行是不是顶层 GFM 表格的表头:下一行是列数相同的分隔行。返回表格结束的行号(不含),0 = 不是。
// 只认顶层(缩进为 0)的表格:列表 / 引用里的表格写回时缩进会丢,仍走只读预览。
func mdTableAt(lines []string, i int) int {
	if i+1 >= len(lines) {
		return 0
	}
	h := strings.TrimRight(lines[i], "\r\n")
	if mdIndent(h) != 0 || !strings.Contains(h, "|") || strings.TrimSpace(h) == "" {
		return 0
	}
	d := strings.TrimSpace(lines[i+1])
	if !strings.Contains(d, "-") || mdIndent(strings.TrimRight(lines[i+1], "\r\n")) != 0 {
		return 0
	}
	hc, dc := mdSplitRow(h), mdSplitRow(d)
	if len(hc) != len(dc) {
		return 0
	}
	for _, c := range dc {
		if !mdDelimCell(c) {
			return 0
		}
	}
	j := i + 2
	for j < len(lines) {
		r := strings.TrimRight(lines[j], "\r\n")
		tr := strings.TrimSpace(r)
		if tr == "" || !strings.Contains(r, "|") || mdIndent(r) != 0 || strings.HasPrefix(tr, "```") || strings.HasPrefix(tr, "~~~") {
			break
		}
		j++
	}
	return j
}

// parseMdTable 把表头 + 分隔行 + 数据行读成数据;行内单元格不足补空、多出的丢弃(GFM 本来就忽略)。
func parseMdTable(lines []string) *mdTable {
	t := &mdTable{head: mdSplitRow(strings.TrimRight(lines[0], "\r\n"))}
	n := len(t.head)
	t.align = make([]byte, n)
	for i, c := range mdSplitRow(strings.TrimRight(lines[1], "\r\n")) {
		if i >= n {
			break
		}
		l, r := strings.HasPrefix(c, ":"), strings.HasSuffix(c, ":")
		switch {
		case l && r:
			t.align[i] = 'c'
		case l:
			t.align[i] = 'l'
		case r:
			t.align[i] = 'r'
		}
	}
	for _, ln := range lines[2:] {
		cells := mdSplitRow(strings.TrimRight(ln, "\r\n"))
		row := make([]string, n)
		copy(row, cells)
		t.rows = append(t.rows, row)
	}
	return t
}

// mdWidth 是单元格在等宽字体里的显示宽度(中日韩与全角字符算 2),只用来把序列化出的竖线对齐。
func mdWidth(s string) int {
	w := 0
	for _, r := range s {
		switch {
		case r >= 0x1100 && r <= 0x115F, r >= 0x2E80 && r <= 0xA4CF, r >= 0xAC00 && r <= 0xD7A3,
			r >= 0xF900 && r <= 0xFAFF, r >= 0xFE30 && r <= 0xFE6F, r >= 0xFF00 && r <= 0xFF60, r >= 0xFFE0 && r <= 0xFFE6:
			w += 2
		default:
			w++
		}
	}
	return w
}

// mdEscCell 把单元格里没转义的 | 转义,换行换成空格(表格一行写不下换行)。
func mdEscCell(s string) string {
	var b strings.Builder
	prev := byte(0)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '\n' || ch == '\r':
			ch = ' '
		case ch == '|' && prev != '\\':
			b.WriteByte('\\')
		}
		b.WriteByte(ch)
		prev = ch
	}
	return b.String()
}

func (t *mdTable) cols() int { return len(t.head) }

// markdown 序列化成对齐过的 GFM 表格(不带结尾换行)。
func (t *mdTable) markdown() string {
	n := t.cols()
	w := make([]int, n)
	for i := range w {
		w[i] = 3
	}
	esc := func(r []string) []string {
		out := make([]string, n)
		for i := 0; i < n && i < len(r); i++ {
			out[i] = mdEscCell(r[i])
		}
		return out
	}
	all := [][]string{esc(t.head)}
	for _, r := range t.rows {
		all = append(all, esc(r))
	}
	for _, r := range all {
		for i, c := range r {
			if x := mdWidth(c); x > w[i] {
				w[i] = x
			}
		}
	}
	line := func(r []string) string {
		var b strings.Builder
		b.WriteString("|")
		for i, c := range r {
			b.WriteString(" " + c + strings.Repeat(" ", w[i]-mdWidth(c)) + " |")
		}
		return b.String()
	}
	delim := "|"
	for i := 0; i < n; i++ {
		d := strings.Repeat("-", w[i])
		switch t.align[i] {
		case 'l':
			d = ":" + d[1:]
		case 'r':
			d = d[1:] + ":"
		case 'c':
			d = ":" + d[2:] + ":"
		}
		delim += " " + d + " |"
	}
	lines := []string{line(all[0]), delim}
	for _, r := range all[1:] {
		lines = append(lines, line(r))
	}
	return strings.Join(lines, "\n")
}

// insertRow 在第 at 行(数据行序号,0 = 表头下面第一行)之前插入一行空行。
func (t *mdTable) insertRow(at int) {
	at = max(0, min(at, len(t.rows)))
	t.rows = append(t.rows, nil)
	copy(t.rows[at+1:], t.rows[at:])
	t.rows[at] = make([]string, t.cols())
	t.edited = true
}

func (t *mdTable) deleteRow(r int) {
	if r < 0 || r >= len(t.rows) {
		return
	}
	t.rows = append(t.rows[:r], t.rows[r+1:]...)
	t.edited = true
}

// insertCol 在第 at 列之前插入一列空列。
func (t *mdTable) insertCol(at int) {
	at = max(0, min(at, t.cols()))
	ins := func(r []string) []string {
		r = append(r, "")
		copy(r[at+1:], r[at:])
		r[at] = ""
		return r
	}
	t.head = ins(t.head)
	for i := range t.rows {
		t.rows[i] = ins(t.rows[i])
	}
	t.align = append(t.align, 0)
	copy(t.align[at+1:], t.align[at:])
	t.align[at] = 0
	t.edited = true
}

// deleteCol 删掉一列;只剩一列时不删(表格至少要有一列)。
func (t *mdTable) deleteCol(c int) {
	if t.cols() <= 1 || c < 0 || c >= t.cols() {
		return
	}
	del := func(r []string) []string { return append(r[:c], r[c+1:]...) }
	t.head = del(t.head)
	for i := range t.rows {
		t.rows[i] = del(t.rows[i])
	}
	t.align = append(t.align[:c], t.align[c+1:]...)
	t.edited = true
}

func (t *mdTable) setAlign(c int, a byte) {
	if c >= 0 && c < len(t.align) && t.align[c] != a {
		t.align[c] = a
		t.edited = true
	}
}

// mdNewSeg 把一组行造成一段:开头的空行进 pre,末尾的空行与最后一行的换行进 post。
func mdNewSeg(lines []string, table bool) *mdSeg {
	a, b := 0, len(lines)
	for a < b && strings.TrimSpace(lines[a]) == "" {
		a++
	}
	for b > a && strings.TrimSpace(lines[b-1]) == "" {
		b--
	}
	s := &mdSeg{table: table, pre: strings.Join(lines[:a], ""), post: strings.Join(lines[b:], "")}
	mid := strings.Join(lines[a:b], "")
	if strings.HasSuffix(mid, "\n") {
		mid = mid[:len(mid)-1]
		s.post = "\n" + s.post
	}
	s.raw = mid
	if table {
		s.tbl = parseMdTable(lines[a:b])
	} else {
		s.split = mdSplitBodyBlocks(mid)
	}
	return s
}

// mdSplitSegs 把正文按「顶层表格 / 其余」切成段;围栏代码块里的 | 不算表格。
// 第一段和最后一段一定是正文段(必要时补空段),这样表格前后总有地方可以输入。
func mdSplitSegs(body string) []*mdSeg {
	lines := mdLinesKeep(body)
	var segs []*mdSeg
	var rich []string
	flush := func() {
		if len(rich) > 0 {
			segs = append(segs, mdNewSeg(rich, false))
			rich = nil
		}
	}
	fence := ""
	for i := 0; i < len(lines); {
		trim := strings.TrimSpace(lines[i])
		if fence != "" {
			if strings.HasPrefix(trim, fence) {
				fence = ""
			}
			rich = append(rich, lines[i])
			i++
			continue
		}
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			fence = trim[:3]
			rich = append(rich, lines[i])
			i++
			continue
		}
		if j := mdTableAt(lines, i); j > 0 {
			flush()
			segs = append(segs, mdNewSeg(lines[i:j], true))
			i = j
			continue
		}
		rich = append(rich, lines[i])
		i++
	}
	flush()
	if len(segs) > 0 && segs[0].table {
		segs = append([]*mdSeg{{}}, segs...)
	}
	if len(segs) > 0 && segs[len(segs)-1].table {
		segs = append(segs, &mdSeg{})
	}
	return segs
}

func mdSegsHaveTable(segs []*mdSeg) bool {
	for _, s := range segs {
		if s.table {
			return true
		}
	}
	return false
}

// segsText 是各段拼回的正文。改过的正文段会补足与相邻表格之间的空行(GFM 里表格后面紧跟的非空行会被当成表格的一行)。
func (md *mdState) segsText() string {
	var sb strings.Builder
	n := len(md.segs)
	for i, s := range md.segs {
		pre, post := s.pre, s.post
		if !s.table && s.dirty {
			if s.raw == "" {
				if i > 0 && i < n-1 && pre+post == "" {
					pre = "\n"
				}
			} else {
				if i > 0 && md.segs[i-1].table {
					if need := 2 - strings.Count(md.segs[i-1].post, "\n") - strings.Count(pre, "\n"); need > 0 {
						pre = strings.Repeat("\n", need) + pre
					}
				}
				if i < n-1 && md.segs[i+1].table {
					if need := 2 - strings.Count(post, "\n") - strings.Count(md.segs[i+1].pre, "\n"); need > 0 {
						post += strings.Repeat("\n", need)
					}
				}
			}
		}
		sb.WriteString(pre + s.raw + post)
	}
	return sb.String()
}
