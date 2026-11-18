package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/canvas"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func (a *App) canvasListRow(c *ui.Context, l *canvas.Library, id string, s *canvas.Store, on bool) {
	t := a.tk
	v := &a.cv
	title := s.Doc.Title
	if v.listRenID == id {
		ui.Column(c.Key("cv-ren-"+id)).Gap(4).Padding(4, 0).Children(func() {
			ui.TextInput(c.Key("cv-ren-in"), &v.listRenName).Height(theme.Px(26))
			ui.Row(c).Gap(6).Children(func() {
				kit.Btn(c, "保存", kit.BtnOpt{Small: true, Variant: kit.BtnPri}).OnClick(func() {
					old := title
					if err := l.Rename(id, v.listRenName); err != nil {
						a.toast = err.Error()
						return
					}
					if on && a.canv != nil {
						a.canv.Doc.Title = v.listRenName
					}
					if a.cvLibData != nil {
						_ = a.cvLibData.SaveActive()
					}
					v.undo.Begin("已重命名画布", func() {
						_ = l.Rename(id, old)
						if on && a.canv != nil {
							a.canv.Doc.Title = old
						}
						if a.cvLibData != nil {
							_ = a.cvLibData.SaveActive()
						}
					})
					v.listRenID = ""
					a.toast = "已重命名"
				})
				kit.Btn(c, "取消", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { v.listRenID = "" })
			})
		})
		return
	}
	// 两行内容(标题 + 日期):RowEl 默认是单行行高(方案 B 为 26),内容会溢出选中底色,所以显式给两行的高度
	row := kit.RowEl(c, on).Key("cv-id-"+id).Padding(5, 8).Height(theme.Px(42)).Justify(ui.Start)
	row.OnClick(func() {
		l.SetActive(id)
		a.canv = s
		v.preview = false
	})
	kit.CtxMenu(row, func(m *ui.Menu) { a.canvasListMenu(m, l, id, title) })
	row.Children(func() {
		kit.Ic(c, "fileText", kit.S12)
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, title).SingleLine()
			at := a.now()
			if len(s.Versions) > 0 {
				at = s.Versions[0].At
			}
			ui.Text(c, canvas.Ago(a.now(), at)).FontSize(theme.FsMicro).TextColor(t.Tx4)
		})
		more := kit.Ib(c, "more", kit.IbOpt{Small: true, Tip: "更多"})
		kit.MenuBtn(more, func(m *ui.Menu) { a.canvasListMenu(m, l, id, title) }) // 左键点「···」弹出(行本身右键同一份菜单)
	})
}

func (a *App) canvasListMenu(m *ui.Menu, l *canvas.Library, id, title string) {
	v := &a.cv
	if menuItem(m, "重命名").Chosen() {
		v.listRenID, v.listRenName = id, title
	}
	if menuItem(m, "导出为 .mccanvas").Tooltip("写到「下载」并在访达里显示;双击该文件可导入成新画布").Chosen() {
		a.exportCanvasFile(l, id, title)
	}
	m.Separator()
	if menuItem(m, "删除…").Chosen() {
		st, err := l.Store(id)
		if err != nil {
			return
		}
		wasDoc := st.Doc.Clone()
		a.confirmDlg("删除画布", "删除「"+title+"」?可从撤销条恢复(本页)。", "删除", true, func() {
			if err := l.Delete(id); err != nil {
				a.toast = err.Error()
				return
			}
			if l.Active() != "" {
				if ns, e := l.ActiveStore(); e == nil {
					a.canv = ns
				}
			} else {
				a.canv = nil
			}
			v.undo.Begin("已删除画布", func() {
				if _, _, e := l.New(title, wasDoc, "你", "撤销删除", a.now()); e == nil {
					if ns, e2 := l.ActiveStore(); e2 == nil {
						a.canv = ns
					}
				}
			})
			a.toast = "已删除"
			log.Info("画布删除", "id", id)
		})
	}
}

func (a *App) canvasUndoBar(c *ui.Context) {
	v := &a.cv
	if !v.undo.Active() {
		return
	}
	t := a.tk
	ui.Row(c).Gap(8).Padding(6, 8).Radius(t.RSm()).Background(t.Fill1).Border(1, t.Bd2).Margin(0, 0, 6, 0).Children(func() {
		ui.Text(c, v.undo.Label).Grow(1).FontSize(theme.FsCap)
		kit.Btn(c, "撤销", kit.BtnOpt{Small: true}).OnClick(func() {
			fn := v.undo.Undo
			v.undo.Clear()
			if fn != nil {
				fn()
			}
		})
	})
}
