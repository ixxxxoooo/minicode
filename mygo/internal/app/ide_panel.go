package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// bottomPanel 是编辑区底部面板:终端 / 问题 / 输出。
func (a *App) bottomPanel(c *ui.Context) {
	t := a.tk
	h := a.panelHeight()
	box := ui.Column(c).Key("bpanel").BorderWidth(1, 0, 0, 0).BorderColor(t.Bd2).Transition(a.mt())
	if a.ly.panelMax {
		box.Grow(1).MinHeight(0)
	} else {
		box.Height(h).Shrink(0)
	}
	if b := box.Bounds(); b.H > 0 {
		a.ide.panelTop = b.Y // 上一帧量到的面板上缘(测试与浮层定位用)
	}
	box.Children(func() {
		def := float32(172)
		if a.ide.bottomTab == "problems" || a.ide.bottomTab == "refs" {
			def = 236
		}
		hi := max(120, a.winH-theme.HBar-theme.HTab-120)
		if a.ly.panelMax {
			// 最大化时把手仍在面板上缘:向下拖一点就还原成拖到的高度
			a.resizeHandle(c, lyPanel, "t", hi, def, 96, hi, func(v float32) {
				if v < hi-12 {
					a.ly.panelMax, a.ly.panel = false, v
				}
			})
		} else {
			// 向上拖到头(离工作区顶部只剩一点)就吸附成最大化:终端占满整个工作区
			a.resizeHandle(c, lyPanel, "t", h, def, 96, hi, func(v float32) {
				a.ly.panel = v
				if v >= hi-1 {
					a.ly.panelMax = true
				}
			})
		}
		items := []kit.TabItem{
			{ID: "term", Text: "终端", Icon: "terminal"},
			{ID: "problems", Text: "问题", Count: map[bool]int{true: 0, false: 3}[a.proj != ""]},
			{ID: "output", Text: "输出"},
		}
		if r := a.ide.refs; r != nil {
			items = append(items, kit.TabItem{ID: "refs", Text: "引用", Count: len(r.items)})
		}
		if id := kit.Tabs(c, items, a.ide.bottomTab, 28, func() {
			ui.Row(c).Gap(1).AlignItems(ui.Center).Children(func() {
				if a.ide.bottomTab == "refs" {
					if r := a.ide.refs; r != nil {
						ui.Text(c, "「"+r.word+"」").Font("monospace").FontSize(theme.FsCap).TextColor(t.Tx3).Margin(0, 6, 0, 0)
					}
				} else if a.ide.bottomTab == "problems" {
					kit.Chip(c, "全部", kit.ChipOpt{Icon: "filter", On: true})
					kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "重新检查"})
				} else {
					if a.proj != "" {
						n := len(a.ide.tps)
						if n == 0 {
							n = 1
						}
						for i := 0; i < n; i++ {
							i := i
							name := shellName()
							if i > 0 {
								name += " " + itoa(i+1)
							}
							on := a.ide.termSel == "" && a.ide.tpSel == i
							kit.Chip(c, name, kit.ChipOpt{Icon: "terminal", On: on}).OnClick(func() {
								a.ide.termSel, a.ide.tpSel = "", i
								a.ide.tpc().focusIt = true
							})
							if on && n > 1 {
								kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭这个终端"}).OnClick(func() { a.closeTermTab(i) })
							}
						}
					} else {
						kit.Chip(c, shellName(), kit.ChipOpt{Icon: "terminal", On: a.ide.termSel == ""}).OnClick(func() { a.ide.termSel = "" })
					}
					for _, v := range a.ide.cmds {
						kit.Chip(c, v.title, kit.ChipOpt{Icon: "lock", On: a.ide.termSel == v.id, Tip: "只读 · Claude 运行的命令"}).OnClick(func() { a.ide.termSel = v.id })
					}
					kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "新建终端"}).OnClick(func() {
						if a.proj != "" {
							a.newTerm()
						}
					})
					gear := kit.Ib(c, "gear", kit.IbOpt{Small: true, Tip: "终端设置", On: a.popup == "termset"})
					gear.OnClick(func() { a.togglePop("termset") })
					a.termSettings(c, gear)
					kit.Ib(c, "more", kit.IbOpt{Small: true, Tip: "更多操作"}).Menu(a.termMoreMenu)
				}
				mi, mt := "max", "最大化面板 ⌘⇧J"
				if a.ly.panelMax {
					mi, mt = "min", "还原面板 ⌘⇧J"
				}
				kit.Ib(c, mi, kit.IbOpt{Small: true, Tip: mt, On: a.ly.panelMax}).OnClick(a.togglePanelMax)
				kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭面板 ⌘J"}).OnClick(func() { a.ide.bottomOpen, a.ly.panelMax = false, false })
			})
		}); id != "" {
			a.ide.bottomTab = id
		}
		switch a.ide.bottomTab {
		case "problems":
			a.problems(c)
		case "refs":
			a.refsPanel(c)
		case "output":
			a.outPanel(c)
		default:
			a.terminal(c)
		}
	})
}

func (a *App) terminal(c *ui.Context) {
	if a.ide.termSel != "" {
		for _, v := range a.ide.cmds {
			if v.id == a.ide.termSel {
				a.cmdOutput(c, v)
				return
			}
		}
		a.ide.termSel = ""
	}
	if a.proj != "" {
		a.realTerminal(c)
		return
	}
	t := a.tk
	ui.Scroll(c).Key("term").TrackScroll(&a.ide.termSt).Grow(1).Children(func() {
		ui.Column(c).Padding(6, 10).Children(func() {
			for _, ln := range a.ide.termLines {
				var spans []ui.Span
				for _, s := range ln.segs {
					sp := ui.Span{Text: s.text, Font: "monospace", Size: 11.5, Color: t.Tx1}
					switch s.kind {
					case 'p':
						sp.Color = t.Ok
					case 'e':
						sp.Color = t.Err
					case 'd':
						sp.Color = t.Tx3
					case 'b':
						sp.Weight = 600
					}
					spans = append(spans, sp)
				}
				if len(spans) == 0 {
					spans = append(spans, ui.Span{Text: " ", Font: "monospace", Size: 11.5})
				}
				row := ui.Row(c).MinHeight(18.4).AlignItems(ui.Center)
				row.Children(func() {
					ui.RichText(c, spans...).LineHeight(1.6)
					if ln.segs != nil && ln.segs[len(ln.segs)-1].text == "" {
						kit.Caret(c)
					}
				})
			}
		})
	})
}

// problem 是问题列表里的一项(演示数据;真实模式由语言服务器填充)。
type problem struct {
	level, msg, src string
	line            int
}

func (a *App) problems(c *ui.Context) {
	t := a.tk
	if a.proj != "" {
		a.realProblems(c)
		return
	}
	type group struct {
		file  string
		items []problem
	}
	var data []group
	for _, p := range demoProblems() {
		if n := len(data); n == 0 || data[n-1].file != p.file {
			data = append(data, group{file: p.file})
		}
		g := &data[len(data)-1]
		g.items = append(g.items, problem{p.level, p.msg, p.src, p.line})
	}
	ui.Scroll(c).Key("probs").Grow(1).Children(func() {
		ui.Column(c).Padding(2, 6).Children(func() {
			for _, g := range data {
				ui.Row(c).Height(24).PaddingX(8).Gap(5).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(10.5).FontWeight(700).Children(func() {
					kit.Ic(c, "chevD", kit.S12)
					kit.Ic(c, "code", kit.S12)
					ui.Text(c, g.file).Grow(1)
					kit.Cnt(c, itoa(len(g.items)), false)
				})
				for _, p := range g.items {
					p := p
					file := g.file
					row := kit.RowEl(c, false).Key("dp-" + file + itoa(p.line)).Gap(6)
					row.ContextMenu(func(m *ui.Menu) {
						a.problemMenu(m, file, p.line, p.msg, func() string {
							var sb strings.Builder
							for _, d := range demoProblems() {
								sb.WriteString(d.file + ":" + itoa(d.line) + " " + d.msg + "\n")
							}
							return strings.TrimSpace(sb.String())
						})
					})
					row.OnClick(func() {
						if full := a.resolveFile(file); full != "" {
							a.jumpTo(full, max(p.line-1, 0), 0, 0)
						}
					})
					row.Children(func() {
						icon, col := "alert", t.Warn
						if p.level == "err" {
							icon, col = "errc", t.Err
						}
						kit.Ic(c, icon, kit.S12).TextColor(col)
						ui.Text(c, p.msg).Grow(1).SingleLine()
						ui.Text(c, p.src).FontSize(theme.FsCap).TextColor(t.Tx4).NoWrap()
						ui.Text(c, "行 "+itoa(p.line)).FontSize(theme.FsMicro).TextColor(t.Tx4)
					})
				}
			}
		})
	})
}

// cmdOutput 显示一条只读命令的输出(Claude 在会话里运行的命令)。
func (a *App) cmdOutput(c *ui.Context, v cmdView) {
	t := a.tk
	ui.Scroll(c).Key("cmdout-" + v.id).Grow(1).Children(func() {
		ui.Column(c).Padding(6, 10).Children(func() {
			ui.Row(c).MinHeight(monoLH).AlignItems(ui.Center).Children(func() {
				ui.RichText(c, ui.Span{Text: "$ ", Font: "monospace", Size: 11.5, Color: t.Ok}, ui.Span{Text: v.title, Font: "monospace", Size: 11.5, Color: t.Tx1, Weight: 600}).NoWrap()
			})
			for i, ln := range strings.Split(strings.TrimRight(v.output, "\n"), "\n") {
				ui.Row(c).Key(i).MinHeight(monoLH).AlignItems(ui.Center).Children(func() {
					ui.RichText(c, a.termLine(ln)...).NoWrap().Clip()
				})
			}
		})
	})
}

// refsPanel 是「引用」分段:按文件分组,点击跳转并保持面板,方便逐个看。
func (a *App) refsPanel(c *ui.Context) {
	t := a.tk
	r := a.ide.refs
	if r == nil {
		return
	}
	if len(r.items) == 0 {
		ui.Column(c).Grow(1).Padding(8, 12).Children(func() {
			ui.Text(c, "没有找到「"+r.word+"」的引用").TextColor(t.Tx3)
		})
		return
	}
	ui.Scroll(c).Key("refs").Grow(1).Children(func() {
		ui.Column(c).Padding(2, 6).Children(func() {
			last := ""
			for i, it := range r.items {
				if it.path != last {
					last = it.path
					n := 0
					for _, x := range r.items {
						if x.path == it.path {
							n++
						}
					}
					ui.Row(c).Key("rg-" + it.path).Height(24).PaddingX(8).Gap(5).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(10.5).FontWeight(700).Children(func() {
						kit.Ic(c, "chevD", kit.S12)
						a.fileIc(c, it.path)
						ui.Text(c, it.path).Grow(1)
						kit.Cnt(c, itoa(n), false)
					})
				}
				i, it := i, it
				kit.RowEl(c, false).Key("rf-" + itoa(i)).Gap(8).Children(func() {
					ui.Text(c, itoa(it.line+1)).FontSize(theme.FsMicro).TextColor(t.Tx4).Width(28).NoWrap()
					ui.Text(c, it.text).Font("monospace").FontSize(11.5).Grow(1).SingleLine()
				}).OnClick(func() { a.jumpTo(it.path, it.line, it.col, it.col+len(r.word)) })
			}
		})
	})
}
