package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

const pBlameGutter = "editor.blameGutter"

// blameHashColor 提交哈希对应的追溯栏色条颜色。
func blameHashColor(h string, t *theme.Tokens) ui.Color {
	if h == "" {
		return t.Tx4
	}
	var n uint32
	for i := 0; i < len(h) && i < 8; i++ {
		n = n*31 + uint32(h[i])
	}
	pal := []ui.Color{t.Ac, t.Ok, t.Info, t.Warn, t.Err}
	return pal[int(n)%len(pal)]
}

// authorAbbr 作者缩写(首字母 / 前两字)。
func authorAbbr(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "?"
	}
	r := []rune(name)
	if len(r) <= 2 {
		return string(r)
	}
	return string(r[:2])
}

// blameCardOverlay 光标行追溯悬停提交卡(设计稿 commitCard)。
func (a *App) blameCardOverlay(c *ui.Context, tb *ideTab) {
	if tb == nil || tb.st == nil || tb.buf == nil || tb.diff || !a.prefs.Bool(pBlameInline, true) {
		return
	}
	g := a.gitInfoFor(tb.path)
	if g == nil {
		return
	}
	line := tb.st.CursorPos().Line
	bl, ok := a.blameLine(tb, g, line)
	if !ok {
		return
	}
	if !a.ide.blameHover {
		_, y := cursorXY(tb, line, 0)
		hit := ui.ButtonBase(c).Key("blame-hover").Absolute().Right(a.mmWidth() + mmRulW + 12).Top(y).Height(18).MinWidth(60).
			Cursor(ui.CursorPointer).Opacity(0.01)
		if hit.Hovered() {
			a.ide.blameHover = true
			a.ide.blameLine = line
		}
		return
	}
	if a.ide.blameLine != line {
		a.ide.blameHover = false
		return
	}
	_, y := cursorXY(tb, line, 0)
	a.drawCommitCard(c, tb, bl, a.mmWidth()+mmRulW+20, y+22)
}

func (a *App) drawCommitCard(c *ui.Context, tb *ideTab, bl vcs.BlameLine, right, top float32) {
	t := a.tk
	card := ui.Column(c).Key("blame-card").Absolute().Right(right).Top(top).Width(300).Padding(10, 12).Gap(6).
		Background(t.Raised).Radius(t.RLg()).Border(1, t.Bd1)
	for _, s := range t.Shadow2() {
		card.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	card.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(22, 22).Radius(11).Background(t.AcSoft).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Text(c, authorAbbr(bl.Author)).FontSize(10).FontWeight(700).TextColor(t.AcTx)
			})
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, bl.Author).FontWeight(600).SingleLine()
				ui.Text(c, bl.When).FontSize(theme.FsMicro).TextColor(t.Tx3).SingleLine()
			})
		})
		ui.Text(c, bl.Summary).FontSize(theme.FsCap).TextColor(t.Tx2)
		short := bl.Hash
		if len(short) > 7 {
			short = short[:7]
		}
		if short == "" {
			short = "未提交"
		}
		ui.Text(c, short).Font("monospace").FontSize(theme.FsMicro).TextColor(t.Tx4)
		ui.Row(c).Gap(8).Children(func() {
			kit.Btn(c, "复制哈希", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
				if bl.Hash != "" {
					a.copyOut(bl.Hash, "哈希")
				}
			})
			kit.Btn(c, "查看提交", kit.BtnOpt{Small: true, Icon: "commit"}).OnClick(func() {
				a.openGraph()
				if bl.Hash != "" {
					a.graphTab.sel = bl.Hash
					a.scmCommitFiles(bl.Hash)
				}
			})
		})
	})
}

// blameGutterOverlay 追溯栏:行号左侧(自动换行时隐藏)。
func (a *App) blameGutterOverlay(c *ui.Context, tb *ideTab) {
	if tb == nil || tb.st == nil || tb.buf == nil || tb.diff || tb.virt != "" {
		return
	}
	if !a.prefs.Bool(pBlameGutter, false) || tb.st.SoftWrap {
		return
	}
	g := a.gitInfoFor(tb.path)
	if g == nil || len(g.blame) == 0 {
		return
	}
	first, last := tb.st.VisibleLines()
	if first < 0 {
		first = 0
	}
	t := a.tk
	w := float32(88)
	col := ui.Column(c).Key("blame-gutter").Absolute().Left(0).Top(0).Width(w).MinHeight(0)
	col.Children(func() {
		for i := first; i <= last && i < len(g.blame); i++ {
			bl := g.blame[i]
			if i > 0 && i < len(g.blame) && g.blame[i].Hash == g.blame[i-1].Hash && g.blame[i].Author == g.blame[i-1].Author {
				ui.Box(c).Key(i).Height(monoLH * edScale).Width(w).Children(func() {
					ui.Box(c).Width(2).Height(monoLH * edScale).Background(blameHashColor(bl.Hash, t))
				})
				continue
			}
			top := float32(tb.st.LineTop(i)-tb.st.ScrollY()) + monoLH*edScale*0.1
			row := ui.Row(c).Key("bg-" + itoa(i)).Absolute().Left(2).Top(top).Width(w - 4).Height(monoLH * edScale).AlignItems(ui.Center).Gap(4)
			row.Children(func() {
				ui.Box(c).Width(2).Height(monoLH * edScale).Background(blameHashColor(bl.Hash, t)).Shrink(0)
				ui.Text(c, authorAbbr(bl.Author)).FontSize(9).TextColor(t.Tx3).SingleLine().Width(28).Shrink(0)
				ui.Text(c, bl.When).FontSize(9).TextColor(t.Tx4).SingleLine().Grow(1)
			})
		}
	})
}
