package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/canvas"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 画布整页(设计稿 pages.js):文档(编辑 / 预览)+ 右侧大纲、版本、让 Claude 修改。
// 文档与版本在 internal/canvas;「让 Claude 修改」把提示词交给对话(等 Claude 后端接上后由它改画布)。

type canvasState struct {
	preview     bool
	code        bool // 预览时的源码视图
	st          ui.ScrollState
	pos         map[string]float32 // 块 ID → 相对文档顶端的 Y(上一帧量到的)
	outline     string             // 大纲里当前项
	ask         string
	apply       string // 粘贴 Claude 回复以写回画布
	selBlk      string // 让 Claude 修改时引用的块
	verIx       int    // 版本步进器(0=最新)
	drawBot     bool   // 窄窗右栏落底部
	listRenID   string
	listRenName string
	undo        kit.UndoSlot
}

func (a *App) cvStore() *canvas.Store {
	if a.demoExtra {
		_ = a.cvLib()
		if a.canv == nil {
			if s, err := a.cvLibData.ActiveStore(); err == nil {
				a.canv = s
			}
		}
		return a.canv
	}
	l := a.cvLib()
	if a.canv == nil {
		if s, err := l.ActiveStore(); err == nil {
			a.canv = s
		}
	}
	return a.canv
}

func (a *App) canvasPage(c *ui.Context) {
	t := a.tk
	s := a.cvStore()
	v := &a.cv
	if v.pos == nil {
		v.pos = map[string]float32{}
	}
	a.canvasViewerOverlay(c)
	narrow := a.winW > 0 && a.winW < 900
	a.pane(c).Grow(1).Children(func() {
		// 顶栏
		ui.Row(c).Height(40).Padding(0, 16).Gap(8).AlignItems(ui.Center).Shrink(0).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
			kit.Ic(c, "canvas", kit.S12).TextColor(t.Ac)
			ui.Text(c, s.Doc.Title).FontWeight(600).SingleLine()
			kit.Tag(c, s.Doc.Status)
			v0 := s.Versions[0]
			ui.Text(c, "v"+itoa(v0.N)+" · "+canvas.Ago(a.now(), v0.At)+"由 "+v0.By+" 修改").FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Grow(1).MinWidth(0)
			mode := 0
			if v.code {
				mode = 2
			} else if v.preview {
				mode = 1
			}
			if i := kit.Seg(c, []kit.SegItem{{Text: "编辑"}, {Text: "预览"}, {Text: "源码"}}, mode, kit.SegOpt{Key: "cv-mode"}); i >= 0 {
				switch i {
				case 0:
					v.preview, v.code = false, false
				case 1:
					if !v.preview && s.Commit("你", "手动编辑", a.now()) {
						log.Info("画布保存版本", "n", s.Versions[0].N)
						a.canvasPersist()
					}
					v.preview, v.code = true, false
				case 2:
					v.preview, v.code = true, true
				}
			}
			if v.verIx >= len(s.Versions) {
				v.verIx = 0
			}
			if len(s.Versions) > 1 {
				kit.Ib(c, "chevL", kit.IbOpt{Small: true, Tip: "上一版"}).OnClick(func() {
					if v.verIx < len(s.Versions)-1 {
						v.verIx++
					}
				})
				ui.Text(c, "v"+itoa(s.Versions[v.verIx].N)).FontFeatures("tnum")
				kit.Ib(c, "chevR", kit.IbOpt{Small: true, Tip: "下一版"}).OnClick(func() {
					if v.verIx > 0 {
						v.verIx--
					}
				})
			}
			kit.Btn(c, "复制源码", kit.BtnOpt{Small: true, Icon: "copy"}).OnClick(func() {
				a.copyText = s.Doc.Markdown()
				a.toast = "已复制 Markdown 源码"
			})
			kit.Btn(c, "导出", kit.BtnOpt{Small: true, Icon: "download"}).OnClick(a.exportCanvas)
			kit.Btn(c, "让 Claude 修改", kit.BtnOpt{Small: true, Variant: kit.BtnSoft, Icon: "spark"}).OnClick(func() {
				ref := s.Doc.Title
				if v.selBlk != "" {
					if b := s.Doc.Block(v.selBlk); b != nil {
						ref = b.Text
						if b.Anchor != "" {
							ref = b.Anchor
						}
					}
				}
				msg := "请修改画布「" + s.Doc.Title + "」"
				if v.selBlk != "" {
					msg += "(块:" + ref + ")"
				}
				a.AskClaude(msg + ":")
			})
		})
		body := func() { a.canvasDoc(c, s) }
		drawer := func() { a.canvasDrawer(c, s) }
		if narrow {
			ui.Column(c).Grow(1).MinHeight(0).Children(func() {
				ui.Row(c).Grow(1).MinHeight(0).Children(func() {
					if !a.demoExtra || len(a.cvLib().IDs()) > 1 {
						a.canvasList(c)
					}
					ui.Scroll(c).Key("cv-main").Grow(1).TrackScroll(&v.st).Children(func() {
						ui.Column(c).Padding(4, 20, 12, 20).Children(body)
					})
				})
				ui.Column(c).Key("cv-drawer-b").MaxHeight(220).Padding(12, 16).Gap(8).BorderWidth(1, 0, 0, 0).BorderColor(a.tk.Bd1).Children(drawer)
			})
		} else {
			ui.Row(c).Grow(1).MinHeight(0).Children(func() {
				if !a.demoExtra || len(a.cvLib().IDs()) > 1 {
					a.canvasList(c)
				}
				a.pageSplitW(c, "cv", 280, &v.st, body, drawer)
			})
		}
	})
}

func (a *App) canvasPersist() {
	if a.demoExtra || a.cvLibData == nil {
		return
	}
	if err := a.cvLibData.SaveActive(); err != nil {
		log.Warn("画布保存失败", "err", err)
	}
}

func (a *App) exportCanvas() {
	s := a.cvStore()
	p, err := writeTemp("canvas.md", s.Doc.Markdown())
	if err != nil {
		a.toast = "导出失败:" + err.Error()
		return
	}
	a.toast = "已导出为 Markdown:" + p
}

// canvasDoc 是文档正文:居中 500px 的一列;main 在 pageSplit 的滚动容器里,这里自己再包一层以便定位大纲。
func (a *App) canvasDoc(c *ui.Context, s *canvas.Store) {
	t := a.tk
	v := &a.cv
	if v.preview && v.code {
		kit.Mono(c, s.Doc.Markdown(), t.Tx2).LineHeight(1.5)
		return
	}
	ui.Row(c).Justify(ui.Center).Children(func() {
		col := ui.Column(c).Key("cv-col").Width(620).Shrink(1).MinWidth(0).Padding(18, 0, 24, 0)
		col.Children(func() {
			top := col.Bounds().Y
			title := ui.Column(c)
			title.Children(func() {
				if v.preview {
					ui.Text(c, s.Doc.Title).FontSize(22).FontWeight(700)
				} else {
					ui.TextInput(c.Key("cv-title"), &s.Doc.Title).FontSize(22).FontWeight(700).Background(ui.Transparent).BorderWidth(0).Padding(0)
				}
			})
			v.pos["title"] = title.Bounds().Y - top
			for i := range s.Doc.Blocks {
				b := &s.Doc.Blocks[i]
				el := ui.Column(c.Key("cvb-" + b.ID))
				el.Children(func() { a.canvasBlock(c, b) })
				v.pos[b.ID] = el.Bounds().Y - top
			}
		})
	})
	_ = t
}

func (a *App) canvasBlock(c *ui.Context, b *canvas.Block) {
	t := a.tk
	v := &a.cv
	switch b.Kind {
	case canvas.Heading:
		if v.preview {
			ui.Text(c, b.Text).FontSize(15).FontWeight(700).Padding(18, 0, 8, 0)
		} else {
			ui.TextInput(c.Key("cvh-"+b.ID), &b.Text).FontSize(15).FontWeight(700).Background(ui.Transparent).BorderWidth(0).Padding(18, 0, 8, 0)
		}
	case canvas.Para:
		if v.preview {
			ui.Text(c, b.Text).TextColor(t.Tx2).LineHeight(1.65).Padding(6, 0, 4, 0)
		} else {
			ui.TextArea(c.Key("cvp-"+b.ID), &b.Text).Lines(1, 12).MinHeight(20).Background(ui.Transparent).BorderWidth(0).Padding(6, 0, 4, 0).TextColor(t.Tx2).LineHeight(1.65)
		}
	case canvas.DiagramBlock:
		a.canvasDiagram(c, b)
	case canvas.TableBlock:
		a.canvasTable(c, b.Table)
	case canvas.TodoBlock:
		ui.Column(c).Gap(7).Padding(2, 0).Children(func() {
			for i := range b.Todos {
				td := &b.Todos[i]
				r := ui.ButtonBase(c).Key("cvt-" + b.ID + "-" + itoa(i)).Gap(8).AlignItems(ui.Center).Justify(ui.Start).Cursor(ui.CursorPointer)
				r.OnClick(func() { td.State = (td.State + 1) % 3 })
				r.Children(func() {
					kit.Chk(c, td.State)
					x := ui.Text(c, td.Text)
					switch td.State {
					case 2:
						x.TextColor(t.Tx4).Strikethrough()
					case 1:
						x.FontWeight(500)
					default:
						x.TextColor(t.Tx2)
					}
				})
			}
		})
	}
}

// canvasDiagram 画流程图卡片:头部(说明、Mermaid 标签、全屏、源码)+ 方框与箭头。
func (a *App) canvasDiagram(c *ui.Context, b *canvas.Block) {
	d := b.Diagram
	t := a.tk
	const sc = 1.0 // 520 宽的设计坐标缩到正文列宽
	a.box(c).Background(t.Sunken).Clip().Margin(4, 0).Children(func() {
		ui.Row(c).Height(30).Padding(0, 10).Gap(8).AlignItems(ui.Center).FontSize(theme.FsCap).TextColor(t.Tx3).Children(func() {
			kit.Ic(c, "layers", kit.S12)
			ui.Text(c, d.Caption).Grow(1).SingleLine()
			kit.Tag(c, "Mermaid")
			kit.Ib(c, "max", kit.IbOpt{Small: true, Tip: "全屏查看"}).OnClick(func() { a.openCvViewer(b.ID) })
			kit.Ib(c, "code", kit.IbOpt{Small: true, Tip: "查看源码"}).OnClick(func() {
				a.copyText = "```mermaid\n" + d.Caption + "\n```"
				a.toast = "已复制图表源码"
			})
		})
		cv := ui.Box(c).Height(150*sc).Margin(0, 0, 8, 0)
		cv.Draw(func(p *ui.Painter, r ui.Rect) {
			ox, oy := r.X, r.Y
			pt := func(x, y float32) (float32, float32) { return ox + x*sc, oy + y*sc }
			for _, e := range d.Edges {
				pts := make([][2]float32, len(e))
				for i, q := range e {
					x, y := pt(q[0], q[1])
					pts[i] = [2]float32{x, y}
				}
				p.StrokePolyline(pts, 1.5, t.Tx3)
				if n := len(pts); n >= 2 { // 箭头
					x1, y1 := pts[n-1][0], pts[n-1][1]
					x0, y0 := pts[n-2][0], pts[n-2][1]
					dx, dy := float32(0), float32(0)
					switch {
					case x1 > x0:
						dx = 1
					case x1 < x0:
						dx = -1
					case y1 > y0:
						dy = 1
					default:
						dy = -1
					}
					p.StrokePolyline([][2]float32{{x1 - 6*dx - 3.5*dy, y1 - 6*dy - 3.5*dx}, {x1, y1}, {x1 - 6*dx + 3.5*dy, y1 - 6*dy + 3.5*dx}}, 1.5, t.Tx3)
				}
			}
		})
		// 方框:用绝对定位的元素,文字才好排
		cv.Children(func() {
			for i, n := range d.Nodes {
				bx := ui.Row(c.Key("dgn-"+itoa(i))).Absolute().Top(n.Y*sc).Size(n.W*sc, 30*sc).Radius(7).Justify(ui.Center).AlignItems(ui.Center).
					Background(t.Raised).Border(1, t.Bd3).FontSize(theme.FsCap)
				bx.Left((520*sc-0)*0 + n.X*sc + 0)
				if n.Hot {
					bx.Background(t.AcSoft).BorderColor(t.AcLine).TextColor(t.AcTx)
				}
				bx.Children(func() { ui.Text(c, n.Label).SingleLine() })
			}
		})
	})
}

func (a *App) canvasTable(c *ui.Context, tb *canvas.Table) {
	t := a.tk
	w := []float32{2.2, 1.3, 1.6, 1.3}
	ui.Column(c).Padding(2, 0).Children(func() {
		ui.Row(c).Height(26).AlignItems(ui.Center).FontSize(theme.FsCap).TextColor(t.Tx3).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd2).Children(func() {
			for i, h := range tb.Head {
				ui.Text(c, h).Grow(w[i%4]).Basis(0)
			}
		})
		for _, r := range tb.Rows {
			ui.Row(c).Height(26).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
				for i, cell := range r {
					if i == len(r)-1 {
						ui.Row(c).Grow(w[i%4]).Basis(0).Children(func() {
							kit.Tag(c, cell, map[string]kit.TagKind{"已完成": kit.KOk, "进行中": kit.KAc}[cell])
						})
						continue
					}
					ui.Text(c, cell).Grow(w[i%4]).Basis(0).SingleLine()
				}
			})
		}
	})
}

func (a *App) canvasDrawer(c *ui.Context, s *canvas.Store) {
	t := a.tk
	v := &a.cv
	kit.Grp(c, "大纲")
	ui.Column(c).Margin(0, -6).Children(func() {
		for i, o := range s.Doc.Outline() {
			on := v.outline == o.ID || (v.outline == "" && i == 0)
			r := kit.RowEl(c, on).Key("ol-" + o.ID).PaddingStart(8 + float32(o.Level)*14).Justify(ui.Start)
			r.OnClick(func() {
				v.outline = o.ID
				v.selBlk = o.ID
				if y, ok := v.pos[o.ID]; ok {
					v.scrollTo(y)
				}
			})
			r.Children(func() {
				if o.Level == 0 {
					kit.Ic(c, "fileText", kit.S12)
				}
				ui.Text(c, o.Text).SingleLine()
			})
		}
	})
	kit.Sep(c)
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		kit.Grp(c, "版本")
		kit.Cnt(c, itoa(len(s.Versions)), false)
	})
	ui.Column(c).Margin(0, -6).Children(func() {
		for k, ver := range s.Versions {
			row := ui.Row(c.Key("ver-"+itoa(ver.N))).Padding(4, 8).Gap(8).AlignItems(ui.Start)
			row.Children(func() {
				kind := kit.KNeutral
				if k == 0 {
					kind = kit.KAc
				}
				kit.Tag(c, "v"+itoa(ver.N), kind)
				ui.Column(c).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, ver.By+" · "+ver.Note).SingleLine()
					ui.Text(c, canvas.Ago(a.now(), ver.At)).FontSize(theme.FsCap).TextColor(t.Tx4)
				})
				if k > 0 {
					kit.Ib(c, "rewind", kit.IbOpt{Small: true, Tip: "回到此版本"}).OnClick(func() {
						if err := s.Rewind(ver.N, a.now()); err == nil {
							a.toast = "已回到 v" + itoa(ver.N)
							a.canvasPersist()
							log.Info("画布回退", "to", ver.N)
						}
					})
				}
			})
		}
	})
	ui.Spacer(c).Grow(1)
	kit.Grp(c, "让 Claude 修改")
	chip := ""
	if v.selBlk != "" {
		if b := s.Doc.Block(v.selBlk); b != nil {
			if b.Anchor != "" {
				chip = "块:" + b.Anchor
			} else {
				chip = "块:" + b.Text
			}
		}
	}
	if v.apply != "" {
		if kit.Btn(c, "应用 Claude 回复到画布", kit.BtnOpt{Variant: kit.BtnSoft, Small: true}).Clicked() {
			if ok, msg := s.ApplyClaudeReply(v.apply, "Claude", "按对话修改", a.now()); ok {
				v.apply = ""
				a.canvasPersist()
				a.toast = "已写入新版本 v" + itoa(s.Versions[0].N)
			} else {
				a.toast = msg
			}
		}
	}
	ui.TextArea(c.Key("cv-apply"), &v.apply).Lines(2, 4).Placeholder("将 Claude 的 JSON 回复粘贴到这里,点上方按钮写回…").FontSize(theme.FsCap)
	a.miniComposer(c, "cv-ask", "描述要怎么改,例如「把里程碑改成表格」", &v.ask, chip, func(text string) {
		if text != "" {
			a.AskClaude("请修改画布「" + s.Doc.Title + "」(引用块 " + chip + "):" + text)
		}
	})
}

// scrollTo 让文档滚到某个 Y(留一点上边距)。
func (v *canvasState) scrollTo(y float32) { v.st.Y = max(0, y-12) }
