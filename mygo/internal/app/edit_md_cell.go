package app

import (
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// Markdown 预览里表格单元格的「渲染态」(设计稿 ide-md「表格:就地编辑」):
// 没聚焦的单元格按 Markdown 渲染(行内代码胶囊 / 粗体 / 斜体 / 链接,长内容折行、整行变高),
// 点击或 Tab / ⏎ 走到的那一格才是输入框、显示 Markdown 源码;离开后渲染回去。做法对标 Typora / Obsidian 实时预览。
//
// @author ixxxxoooo

// mdCellOptions 返回(并缓存)单元格用的 Markdown 选项:字号固定取预览的 13px(不随会话字号缩放),块间距为 0。
func (a *App) mdCellOptions() *markdown.Options {
	base := a.mdOptions() // 先让主题键保持最新
	key := a.mdThemeKey + "|cell"
	if a.mdCell == nil || a.mdCellKey != key {
		th := a.mdTheme()
		th.FontSize = theme.FsRead
		th.LineHeight = 1.5
		th.BlockSpacing = 0
		// Options 含 sync.Map,不能整体复制:新建并带上基础选项的回调。
		a.mdCell = &markdown.Options{Theme: th, OnLink: base.OnLink, LinkMenu: base.LinkMenu}
		a.mdCellKey = key
	}
	return a.mdCell
}

// mdCellBlockStart 匹配会被当成块级语法的单元格开头(标题 / 引用 / 列表 / 围栏):单元格里只该有行内语法。
var mdCellBlockStart = regexp.MustCompile("^([#>]|[-+*](\\s|$)|\\d+[.)](\\s|$)|```|~~~)")

// mdCellSource 把单元格源码整理成可渲染的 Markdown:还原转义的竖线,把块级开头转义成普通文字,
// 表头整体加粗,并给行内代码两侧补窄空格(同会话流,让底色比文字宽出一点)。
// 参数 raw 是单元格源码,header 表示是不是表头。返回值是交给 markdown.View 的文本;空单元格返回空串。
func mdCellSource(raw string, header bool) string {
	s := strings.TrimSpace(strings.ReplaceAll(raw, `\|`, "|"))
	if s == "" {
		return ""
	}
	if mdCellBlockStart.MatchString(s) {
		s = `\` + s
	}
	if header && !strings.Contains(s, "**") {
		s = "**" + s + "**"
	}
	return padCode(s)
}

// mdCellView 画一个渲染态的单元格:内容是渲染后的 Markdown,上面盖一层透明热区,点一下就进入编辑。
// 参数 key 是单元格键;raw 是源码;header 是不是表头;align 是列对齐('c' 居中 / 'r' 靠右);
// last 为真表示这一列没有右边框;grow 是这一列的宽度权重(mdColWeights);enter 在被点击时调用。
func (a *App) mdCellView(c *ui.Context, key, raw string, header bool, align byte, last bool, grow float32, enter func()) {
	t := a.tk
	box := ui.Column(c.Key(key+"-v")).Grow(grow).Basis(0).MinWidth(0).MinHeight(mdCellH).Padding(5, 8).Justify(ui.Center)
	if !last {
		box.BorderWidth(0, 1, 0, 0).BorderColor(t.Bd1)
	}
	box.Children(func() {
		if src := mdCellSource(raw, header); src != "" {
			v := markdown.View(c, src, a.mdCellOptions())
			switch align {
			case 'c':
				v.AlignSelf(ui.Center)
			case 'r':
				v.AlignSelf(ui.End)
			}
		}
		hit := ui.Box(c.Key(key + "-hit")).Absolute().Left(0).Right(0).Top(0).Bottom(0).Cursor(ui.CursorText)
		if hit.Clicked() {
			enter()
		}
	})
}

// mdGoCell 让表格转到某个单元格的编辑态(下一次构建里聚焦它的输入框)。参数 key 是目标单元格键。
func (md *mdState) mdGoCell(key string) {
	md.tblEdit, md.tblFocus, md.tblHad = key, key, false
}

// mdCellNeighbor 返回 Tab 方向上的下一个单元格坐标(行 r、列 col;r = -1 是表头);走到头返回 ok=false。
// 参数 rows / cols 是表格的行数(不含表头)与列数,back 为真表示 ⇧Tab(往前)。
func mdCellNeighbor(r, col, rows, cols int, back bool) (nr, nc int, ok bool) {
	if back {
		col--
		if col < 0 {
			r--
			col = cols - 1
		}
		return r, col, r >= -1
	}
	col++
	if col >= cols {
		r++
		col = 0
	}
	return r, col, r < rows
}

// mdCellTab 处理编辑态单元格里的 Tab / ⇧Tab:走到下一格(行尾换行、走到头不动),与设计稿一致。
// 参数 rr / cc 是当前格的行列(rr = -1 是表头),keyOf 把行列换成单元格键。
func (a *App) mdCellTab(c *ui.Context, md *mdState, tbl *mdTable, rr, cc int, keyOf func(r, col int) string) {
	back := false
	hit := c.Shortcut(0, ui.KeyTab)
	if !hit {
		back = c.Shortcut(ui.Shift, ui.KeyTab)
		hit = back
	}
	if !hit {
		return
	}
	if nr, nc, ok := mdCellNeighbor(rr, cc, len(tbl.rows), tbl.cols(), back); ok {
		md.mdGoCell(keyOf(nr, nc))
		a.frameSoon()
	}
}

// mdColPad 是每列权重里留给单元格内边距与行内代码胶囊的富余(按字符数计)。
const mdColPad = 6

// mdColWeights 按各列最长内容给列宽分配权重:内容越长的列越宽(内容宽度夹在 [8, 40] 之间,避免一列独占或被压成一条缝),
// 而不是所有列等宽、长说明被挤成一窄条。参数 tbl 是表格;返回值每列一个权重,表头也参与计算。
func mdColWeights(tbl *mdTable) []float32 {
	n := tbl.cols()
	w := make([]float32, n)
	for i := range w {
		w[i] = 8 + mdColPad
	}
	see := func(col int, s string) {
		if col < n {
			w[col] = max(w[col], min(float32(mdWidth(s)), 40)+mdColPad)
		}
	}
	for col, h := range tbl.head {
		see(col, h)
	}
	for _, r := range tbl.rows {
		for col, cell := range r {
			see(col, cell)
		}
	}
	return w
}
