package app

import (
	"path"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// bottomPanel 是编辑区底部面板:终端 / 问题 / 输出。
func (a *App) bottomPanel(c *ui.Context) {
	t := a.tk
	h := a.panelHeight()
	open := a.ide.bottomOpen
	// 开合(设计稿 .bpanel):高度 260ms ease-io、透明度 120ms ease;收起时内容保留到高度归零,展开时从 0 长出来。
	live := a.keepWhile("bpanel", open, kit.DurSlow)
	// 高度自己用 AnimateWith 推进(不用 Transition{Height}:只过渡高度的元素第一帧宽度会量成 0,终端也就起不来)。
	// 宽度不做任何过渡:父级工作区的宽度已经是逐帧动画值,面板宽度必须与它同帧一致;
	// 曾经给宽度挂 440 ms 的元素过渡,工作区一变宽,底部面板的宽度要再追约 0.5 秒才跟上(日志里面板宽度落后工作区几百 px),
	// 表现为「终端比编辑器窄一截、旁边露出空白」。位置也不过渡(否则上缘会滞后)。
	box := ui.Column(c).Key("bpanel").Transition(noMotion)
	if live {
		// 不透明底色:拖动分隔线时编辑器区域哪怕某一帧落后,也不会透到面板标签条上
		box.Background(t.BG).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd2)
	}
	target, dur := float32(0), kit.DurSlow
	if open {
		target = h
	}
	if a.instantLayout() {
		dur = time.Millisecond // 拖动分隔线 / 拖窗口边缘:高度直接跟手
	}
	a.probe(pzPanel, box)
	hv := box.AnimateWith("h", target, dur, kit.EaseIO)
	if a.instantLayout() {
		hv = target // 拖动分隔线:直接取目标高度,不用落后一帧的动画值
	}
	switch {
	case open && a.ly.panelMax:
		box.Grow(1).MinHeight(0)
	default:
		box.Height(hv).Shrink(0)
	}
	box.Opacity(box.AnimateWith("op", map[bool]float32{true: 1, false: 0}[open], kit.DurFast, kit.EaseCSS))
	if !live {
		box.Children(func() {})
		return
	}
	a.zoneMarkPanel(box)
	if b := box.Bounds(); b.H > 0 && open {
		a.ide.panelTop = b.Y // 上一帧量到的面板上缘(测试与浮层定位用)
	}
	box.Children(func() {
		if bw := box.Bounds(); bw.W > 0 {
			a.ide.panelW = bw.W
		}
		a.panelFlash(c, box)
		// 内容固定在展开后的高度,外层高度变化只是把它「裁出来 / 裁进去」(终端不会在动画里反复改行数)
		body := ui.Column(c).Key("bp-body")
		if open && a.ly.panelMax {
			body.Grow(1).MinHeight(0)
		} else {
			body.Height(h).Shrink(0)
		}
		body.Children(func() {
			a.bottomPanelBody(c, h)
		})
	})
}

// bottomPanelBody 是底部面板里的标签条与当前分段的内容。
func (a *App) bottomPanelBody(c *ui.Context, h float32) {
	t := a.tk
	{
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
		if a.wsNarrow() { // 窄档:底部面板只留终端 / 问题(当前所在的分段仍保留,免得看不到自己在哪)
			keep := items[:0:0]
			for _, it := range items {
				if it.ID == "term" || it.ID == "problems" || it.ID == a.ide.bottomTab {
					keep = append(keep, it)
				}
			}
			items = keep
		}
		if id := kit.Tabs(c, items, a.ide.bottomTab, 28, func() {
			ui.Row(c).Gap(1).AlignItems(ui.Center).Children(func() {
				if a.ide.bottomTab == "refs" {
					if r := a.ide.refs; r != nil {
						ui.Text(c, "「"+r.word+"」").Font("monospace").FontSize(theme.FsCap).TextColor(t.Tx3).Margin(0, 6, 0, 0)
					}
				} else if a.ide.bottomTab == "problems" {
					kit.Chip(c, "全部", kit.ChipOpt{Icon: "filter", On: !a.ide.probErr}).OnClick(func() { a.ide.probErr = false })
					kit.Chip(c, "仅错误", kit.ChipOpt{On: a.ide.probErr}).OnClick(func() { a.ide.probErr = true })
					if kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "重新检查"}).Clicked() {
						a.recheckProblems()
					}
					if a.probScope() == "ws" && a.anyWsScanning() {
						kit.Btn(c, "取消", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.cancelWorkspaceDiag() })
					}
				} else {
					vert := a.ide.bottomTab == "term" && a.termTabsVertical(a.ide.panelW)
					if a.proj != "" && !vert {
						a.termTabBar(c)
					} else if a.proj == "" {
						kit.Chip(c, shellName(), kit.ChipOpt{Icon: "terminal", On: a.ide.termSel == ""}).OnClick(func() { a.ide.termSel = "" })
					}
					for _, v := range a.ide.cmds {
						kit.Chip(c, v.title, kit.ChipOpt{Icon: "lock", On: a.ide.termSel == v.id, Tip: "只读 · Claude 运行的命令"}).OnClick(func() { a.ide.termSel = v.id })
					}
					if !vert {
						tip := "新建终端"
						if a.proj == "" {
							tip = "演示模式没有真实终端"
						}
						kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: tip, Disabled: a.proj == ""}).OnClick(func() {
							if a.proj != "" {
								a.newTerm()
							}
						})
					}
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
	}
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
		if a.termTabsVertical(a.ide.panelW) && a.ide.termSel == "" {
			// Stretch:竖向列表要贴着面板顶部撑满高度(默认的居中对齐会让它漂在中间)
			ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Children(func() {
					a.realTerminal(c)
				})
				a.termVerticalList(c)
			})
		} else {
			a.realTerminal(c)
		}
		return
	}
	t := a.tk
	kit.VScroll(c).Key("term").TrackScroll(&a.ide.termSt).Grow(1).Children(func() {
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
	if a.proj != "" || a.wsDiagDemo != "" {
		a.realProblems(c)
		return
	}
	type group struct {
		file, rel string
		items     []problem
	}
	var data []group
	for _, p := range demo.Problems() {
		if n := len(data); n == 0 || data[n-1].rel != p.Path {
			data = append(data, group{file: path.Base(p.Path), rel: p.Path})
		}
		g := &data[len(data)-1]
		g.items = append(g.items, problem{p.Level, p.Msg, p.Source, p.Line})
	}
	kit.VScroll(c).Key("probs").Grow(1).Children(func() {
		ui.Column(c).Padding(2, 6).Children(func() {
			for _, g := range data {
				ui.Row(c).Height(theme.Px(24)).PaddingX(8).Gap(5).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(theme.Px(10.5)).FontWeight(500).Children(func() {
					kit.Ic(c, "chevD", kit.S12)
					a.fileIc(c, g.file)
					ui.Text(c, g.file).Grow(1)
					kit.Cnt(c, itoa(len(g.items)), false)
				})
				for _, p := range g.items {
					p := p
					if a.ide.probErr && p.level != "err" {
						continue
					}
					row := kit.RowEl(c, false).Key("dp-" + g.rel + itoa(p.line)).Gap(6)
					row.ContextMenu(func(m *ui.Menu) {
						a.problemMenu(m, g.rel, p.line, p.msg, func() string {
							var sb strings.Builder
							for _, item := range g.items {
								sb.WriteString(g.rel + ":" + itoa(item.line) + " " + item.msg + "\n")
							}
							return strings.TrimSpace(sb.String())
						})
					})
					row.OnClick(func() {
						a.jumpTo(g.rel, max(p.line-1, 0), 0, 0)
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
	kit.VScroll(c).Key("cmdout-" + v.id).Grow(1).Children(func() {
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
	kit.VScroll(c).Key("refs").Grow(1).Children(func() {
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
					disp := it.path
					if workspace.IsExternalPath(it.path) {
						disp = a.pathDisplay(it.path)
					}
					ui.Row(c).Key("rg-" + it.path).Height(theme.Px(24)).PaddingX(8).Gap(5).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(theme.Px(10.5)).FontWeight(500).Children(func() {
						kit.Ic(c, "chevD", kit.S12)
						a.fileIc(c, path.Base(it.path))
						ui.Text(c, disp).Grow(1)
						kit.Cnt(c, itoa(n), false)
					})
				}
				i, it := i, it
				raw := it.text
				hitLo, hitHi := it.col, it.col+len(r.word)
				if raw == "" {
					hitLo, hitHi = 0, 0
				}
				row := kit.RowEl(c, false).Key("rf-" + itoa(i)).Gap(8)
				row.OnClick(func() { a.jumpTo(it.path, it.line, it.col, it.col+len(r.word)) })
				row.Children(func() {
					ui.Text(c, itoa(it.line+1)).FontSize(theme.FsMicro).TextColor(t.Tx4).Width(28).NoWrap()
					ui.Box(c).Grow(1).Shrink(1).MinWidth(0).Clip().Children(func() {
						ui.RichText(c, a.refLineSpans(it.path, raw, hitLo, hitHi)...).NoWrap()
					})
				})
			}
		})
	})
}
