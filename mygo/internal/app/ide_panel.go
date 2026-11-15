package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// bottomPanel 是编辑区底部面板:终端 / 问题 / 输出。
func (a *App) bottomPanel(c *ui.Context) {
	t := a.tk
	h := float32(172)
	if a.ide.bottomTab == "problems" {
		h = 236
	}
	ui.Column(c).Key("bpanel").Height(h).Shrink(0).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd2).Transition(morph).Children(func() {
		items := []kit.TabItem{
			{ID: "term", Text: "终端", Icon: "terminal"},
			{ID: "problems", Text: "问题", Count: 3},
			{ID: "output", Text: "输出"},
		}
		if id := kit.Tabs(c, items, a.ide.bottomTab, 28, func() {
			ui.Row(c).Gap(1).AlignItems(ui.Center).Children(func() {
				if a.ide.bottomTab == "problems" {
					kit.Chip(c, "全部", kit.ChipOpt{Icon: "filter", On: true})
					kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "重新检查"})
				} else {
					kit.Chip(c, shellName(), kit.ChipOpt{Icon: "terminal", On: true})
					kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: map[bool]string{true: "重启终端", false: "新建终端"}[a.proj != ""]}).OnClick(func() {
						if a.proj != "" {
							a.restartTerm()
						}
					})
					kit.Ib(c, "cols", kit.IbOpt{Small: true})
					kit.Ib(c, "gear", kit.IbOpt{Small: true, Tip: "终端设置"})
				}
				kit.Ib(c, "x", kit.IbOpt{Small: true}).OnClick(func() { a.ide.bottomOpen = false })
			})
		}); id != "" {
			a.ide.bottomTab = id
		}
		switch a.ide.bottomTab {
		case "problems":
			a.problems(c)
		case "output":
			ui.Column(c).Grow(1).Padding(6, 10).Children(func() {
				ui.Text(c, "暂无输出").TextColor(t.Tx3)
			})
		default:
			a.terminal(c)
		}
	})
}

func (a *App) terminal(c *ui.Context) {
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
	data := []struct {
		file  string
		items []problem
	}{
		{"router.rs", []problem{{"warn", "`fallback` 字段从未被读取", "rust-analyzer", 5}, {"warn", "未使用的变量:`h`", "rust-analyzer", 11}}},
		{"router_test.rs", []problem{{"err", "找不到函数 `normalize`,需要 `use crate::router::normalize`", "rust-analyzer", 8}}},
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
					kit.RowEl(c, false).Gap(6).Children(func() {
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
