package app

import (
	"github.com/ixxxxoooo/minicode/internal/theme"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

const (
	mdSegSlack = 36 // 正文段编辑器在内容下方多留的高度:新打一行不用先滚动,下一帧再收敛
	mdSegMin   = 30 // 正文段最小高度(空段也要能点进去输入)
	mdCellH    = 32 // 表格一行的高度
	mdDelW     = 26 // 表格每行最右的「删除行」列宽
)

// mdPend 是表格上排队等执行的结构修改(增删行列、对齐):不在渲染中途改表格,下一次构建开头统一执行。
type mdPend struct {
	seg *mdSeg
	fn  func()
}

// mdQueue 排一个对表格结构的修改,并请求下一帧。
func (a *App) mdQueue(md *mdState, s *mdSeg, op string, fn func()) {
	md.pend = append(md.pend, mdPend{seg: s, fn: func() {
		uxLog.Info("预览表格编辑", "操作", op)
		fn()
	}})
	a.frameSoon()
}

// mdRunPend 执行排队的修改并写回缓冲区。
func (a *App) mdRunPend(tb *ideTab, md *mdState) {
	if len(md.pend) == 0 {
		return
	}
	todo := md.pend
	md.pend = nil
	md.tblEdit, md.tblHad = "", false // 结构变了,旧的编辑键可能指到别的格子;操作要聚焦的格子由它自己再设
	touched := map[*mdSeg]bool{}
	for _, p := range todo {
		p.fn()
		touched[p.seg] = true
	}
	for _, s := range md.segs {
		if touched[s] {
			a.mdWriteSegs(tb, md, s)
		}
	}
}

// mdWriteSegs 把改过的一段写回缓冲区(其余段原样拼回);s 为 nil 时只拼接。
func (a *App) mdWriteSegs(tb *ideTab, md *mdState, s *mdSeg) {
	if s != nil {
		switch {
		case s.table:
			if s.tbl.edited {
				s.raw = s.tbl.markdown()
				s.tbl.edited = false
				s.dirty = true
			}
		case s.doc != nil:
			merged := strings.TrimRight(mdPreserveBody(s.raw, s.split, s.doc.Text().Markdown()), "\n")
			if merged != s.raw {
				s.raw, s.split, s.dirty = merged, mdSplitBodyBlocks(merged), true
			}
		}
	}
	out := md.front + md.segsText()
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	if out == tb.buf.String() {
		return
	}
	tb.buf.Set(out)
	md.ver = tb.buf.Version()
	md.edited = true
	uxLog.Debug("Markdown 分段写回", "path", tb.path, "段数", len(md.segs))
}

// mdSegsView 是含表格的 Markdown 的预览:一个整页滚动容器,里面按顺序放正文编辑器与表格网格。
func (a *App) mdSegsView(c *ui.Context, tb *ideTab, md *mdState) {
	a.mdRunPend(tb, md)
	kit.VScroll(c).Key("mdsegs-" + tb.id).Grow(1).MinHeight(0).Children(func() {
		ui.Row(c).Justify(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).MaxWidth(900).Padding(14, 34, 24, 34).Children(func() {
				var changed *mdSeg
				for i, s := range md.segs {
					if s.table {
						if a.mdTableView(c, tb, md, i, s) {
							changed = s
						}
						continue
					}
					if s.doc == nil {
						continue
					}
					ed := richtext.EditorBase(c.Key("mdseg-"+tb.id+"-"+itoa(i)), s.doc, richtext.EditorOptions{
						Label:        "Markdown 文档",
						Font:         rtFont(),
						Theme:        a.rtTheme(),
						Spans:        a.rtSpans,
						SpansVersion: 1,
						OnLink:       a.openMdLink,
						NoScrollbar:  true,
					}).FillWidth().Height(s.h)
					if ed.Changed() {
						changed = s
					}
					a.mdSegFit(s)
				}
				if changed != nil {
					a.mdWriteSegs(tb, md, changed)
				}
			})
		})
	})
}

// mdSegFit 按上一帧量到的段落位置,把正文段编辑器的高度调到刚好放下全部内容(整页统一滚动,编辑器自己不滚)。
// 还没放下(最后一段落没出现)就放大再量,所以第一次显示要几帧收敛。
func (a *App) mdSegFit(s *mdSeg) {
	n := s.doc.Text().ParagraphCount()
	boxes := s.doc.VisibleParagraphs()
	if len(boxes) == 0 {
		return
	}
	last := boxes[len(boxes)-1]
	var want float32
	if last.Index >= n-1 {
		bottom := last.Top + last.Height
		if f := boxes[0]; f.Index == 0 && f.Top < 0 {
			bottom -= f.Top // 编辑器为了露出光标自己滚过一点
		}
		want = bottom + mdSegSlack
	} else {
		want = s.h*1.5 + 120
	}
	want = max(want, mdSegMin)
	if d := want - s.h; d > .5 || d < -.5 {
		s.h = want
		a.frameSoon()
	}
}

// mdTableView 画一张可编辑的表格,返回这一帧里有没有单元格被输入改动。
func (a *App) mdTableView(c *ui.Context, tb *ideTab, md *mdState, idx int, s *mdSeg) bool {
	t := a.tk
	tbl := s.tbl
	n := tbl.cols()
	changed := false
	base := "mdt-" + tb.id + "-" + itoa(idx) + "-"
	keyOf := func(r, col int) string { return base + itoa(r+1) + "-" + itoa(col) } // r = -1 是表头
	cellp := func(r, col int) *string {
		switch {
		case r < 0 && col < len(tbl.head):
			return &tbl.head[col]
		case r >= 0 && r < len(tbl.rows) && col < len(tbl.rows[r]):
			return &tbl.rows[r][col]
		}
		return nil
	}
	weights := mdColWeights(tbl)
	ui.Column(c).Key(base+"box").Margin(8, 0, 10, 0).Shrink(0).Gap(6).Children(func() {
		ui.Column(c).Key(base+"grid").Radius(t.RSm()).Border(1, t.Bd2).Clip().Children(func() {
			for r := -1; r < len(tbl.rows); r++ {
				rr := r
				row := ui.Row(c).MinHeight(mdCellH).Shrink(0).AlignItems(ui.Stretch)
				if rr < 0 {
					row.Background(t.Fill1)
				} else {
					row.BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1)
				}
				row.Children(func() {
					for col := 0; col < n; col++ {
						cc := col
						p := cellp(rr, cc)
						if p == nil {
							continue
						}
						key := keyOf(rr, cc)
						if md.tblEdit != key { // 渲染态:行内代码胶囊 / 粗体 / 链接,点一下进入编辑(edit_md_cell.go)
							a.mdCellView(c, key, *p, rr < 0, tbl.align[cc], cc == n-1, weights[cc], func() { md.mdGoCell(key); a.frameSoon() })
							continue
						}
						a.mdCellTab(c, md, tbl, rr, cc, keyOf)
						in := ui.TextInputBase(c.Key(key), p).Grow(weights[cc]).Basis(0).MinWidth(0).Height(mdCellH).PaddingX(8).FontSize(theme.Px(13)).FocusRing(false)
						if rr < 0 {
							in.FontWeight(650)
						}
						switch tbl.align[cc] {
						case 'c':
							in.TextAlign(ui.Center)
						case 'r':
							in.TextAlign(ui.End)
						}
						if cc < n-1 {
							in.BorderWidth(0, 1, 0, 0).BorderColor(t.Bd1)
						}
						if in.Focused() {
							in.Background(t.Ac.Alpha(.1))
							md.tblHad = true
						} else if md.tblHad { // 拿到过焦点又失焦:退回渲染态
							md.tblEdit, md.tblHad = "", false
							a.frameSoon()
						}
						if md.tblFocus == keyOf(rr, cc) {
							md.tblFocus = ""
							in.Focus()
						}
						if in.Changed() {
							changed = true
							tbl.edited = true
						}
						if in.Submitted() { // ⏎:跳到下一行同列;已是最后一行就新增一行
							if rr+1 < len(tbl.rows) {
								md.mdGoCell(keyOf(rr+1, cc))
								a.frameSoon()
							} else {
								a.mdQueue(md, s, "回车新增行", func() {
									tbl.insertRow(len(tbl.rows))
									md.mdGoCell(keyOf(len(tbl.rows)-1, cc))
								})
							}
						}
						in.ContextMenu(func(m *ui.Menu) { a.mdTableMenu(m, md, s, rr, cc) })
					}
					ui.Row(c).Width(mdDelW).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
						if rr >= 0 {
							kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "删除这一行", Color: t.Tx4}).OnClick(func() {
								a.mdQueue(md, s, "删除行", func() { tbl.deleteRow(rr) })
							})
						}
					})
				})
			}
		})
		ui.Row(c).Gap(6).Children(func() {
			kit.Btn(c, "添加行", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "plus"}).OnClick(func() {
				a.mdQueue(md, s, "添加行", func() { tbl.insertRow(len(tbl.rows)) })
			})
			kit.Btn(c, "添加列", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "plus"}).OnClick(func() {
				a.mdQueue(md, s, "添加列", func() { tbl.insertCol(tbl.cols()) })
			})
		})
	})
	return changed
}

// mdTableMenu 是单元格的右键菜单:行、列的增删与列对齐。r = -1 是表头。
func (a *App) mdTableMenu(m *ui.Menu, md *mdState, s *mdSeg, r, col int) {
	tbl := s.tbl
	if menuItem(m, "在上方插入行").Disabled(r < 0).Chosen() {
		a.mdQueue(md, s, "上方插入行", func() { tbl.insertRow(r) })
	}
	if menuItem(m, "在下方插入行").Chosen() {
		a.mdQueue(md, s, "下方插入行", func() { tbl.insertRow(r + 1) })
	}
	if menuItem(m, "删除行").Disabled(r < 0).Chosen() {
		a.mdQueue(md, s, "删除行", func() { tbl.deleteRow(r) })
	}
	m.Separator()
	if menuItem(m, "在左侧插入列").Chosen() {
		a.mdQueue(md, s, "左侧插入列", func() { tbl.insertCol(col) })
	}
	if menuItem(m, "在右侧插入列").Chosen() {
		a.mdQueue(md, s, "右侧插入列", func() { tbl.insertCol(col + 1) })
	}
	if menuItem(m, "删除列").Disabled(tbl.cols() <= 1).Chosen() {
		a.mdQueue(md, s, "删除列", func() { tbl.deleteCol(col) })
	}
	m.Separator()
	cur := byte(0)
	if col >= 0 && col < len(tbl.align) {
		cur = tbl.align[col]
	}
	for _, al := range []struct {
		name string
		v    byte
	}{{"本列靠左", 'l'}, {"本列居中", 'c'}, {"本列靠右", 'r'}} {
		if menuItem(m, al.name).Checked(cur == al.v).Chosen() {
			v := al.v
			if cur == v {
				v = 0 // 再选一次 = 取消对齐
			}
			a.mdQueue(md, s, "列对齐", func() { tbl.setAlign(col, v) })
		}
	}
}
