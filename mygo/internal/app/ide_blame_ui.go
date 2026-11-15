package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

const (
	pBlameGutter     = "editor.blameGutter"
	blameHoverDelay  = 300 * time.Millisecond
	blameGutterHintH = 22
)

var blameLog = diag.L("ide.blame")

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
	if len(r) <= 3 { // 三个字以内的短名原样显示
		return string(r)
	}
	return string(r[:2])
}

func (a *App) resetBlameCard() {
	a.ide.blameCardShow = false
	a.ide.blameHoverAt = time.Time{}
	a.ide.blameCardHover = false
}

func (a *App) blameCommitFor(hash string) vcs.Commit {
	for _, c := range a.sc.log {
		if c.Hash == hash {
			return c
		}
	}
	short := hash
	if len(short) > 7 {
		short = short[:7]
	}
	return vcs.Commit{Hash: hash, Subject: short}
}

func (a *App) blameCommitFileCount(hash string) (n int, ready bool) {
	if hash == "" {
		return 0, true
	}
	fs, ok := a.sc.files[hash]
	if ok {
		return len(fs), true
	}
	a.scmCommitFiles(hash)
	return 0, false
}

// blameCardOverlay 光标行追溯悬停提交卡(设计稿 commitCard)。
func (a *App) blameCardOverlay(c *ui.Context, tb *ideTab) {
	if tb == nil || tb.st == nil || tb.buf == nil || tb.diff || tb.virt != "" || !a.prefs.Bool(pBlameInline, true) {
		a.resetBlameCard()
		return
	}
	g := a.gitInfoFor(tb.path)
	if g == nil {
		return
	}
	line := tb.st.CursorPos().Line
	bl, ok := a.blameLine(tb, g, line)
	if !ok {
		a.resetBlameCard()
		return
	}
	if a.ide.blameCardShow && a.ide.blameLine != line {
		a.resetBlameCard()
	}
	_, y := cursorXY(tb, line, 0)
	rowH := monoLH * edScale
	hit := ui.ButtonBase(c).Key("blame-hover").Absolute().Right(a.mmWidth() + mmRulW + 12).Top(y).Height(rowH).MinWidth(120).
		Cursor(ui.CursorPointer).Opacity(0.01)
	if hit.Hovered() {
		a.ide.blameLine = line
		if a.ide.blameHoverAt.IsZero() {
			a.ide.blameHoverAt = time.Now()
		}
		if time.Since(a.ide.blameHoverAt) >= blameHoverDelay {
			a.ide.blameCardShow = true
		}
	} else if !a.ide.blameCardHover {
		a.resetBlameCard()
	}
	if !a.ide.blameCardShow {
		return
	}
	a.drawCommitCard(c, tb, bl, a.mmWidth()+mmRulW+20, y+rowH+4)
}

func (a *App) drawCommitCard(c *ui.Context, tb *ideTab, bl vcs.BlameLine, right, top float32) {
	t := a.tk
	card := ui.Column(c).Key("blame-card").Absolute().Right(right).Top(top).Width(300).Padding(10, 12).Gap(6).
		Background(t.Raised).Radius(t.RLg()).Border(1, t.Bd1)
	if card.Hovered() {
		a.ide.blameCardHover = true
	} else {
		a.ide.blameCardHover = false
	}
	for _, s := range t.Shadow2() {
		card.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	when := bl.When
	if bl.Uncommitted {
		when = when + "(未提交)"
	}
	card.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(22, 22).Radius(11).Background(t.AcSoft).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Text(c, authorAbbr(bl.Author)).FontSize(10).FontWeight(700).TextColor(t.AcTx)
			})
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, bl.Author).FontWeight(600).SingleLine()
				ui.Text(c, when).FontSize(theme.FsMicro).TextColor(t.Tx3).SingleLine()
			})
		})
		sum := bl.Summary
		if sum == "" {
			sum = "无提交说明"
		}
		ui.Text(c, sum).FontSize(theme.FsCap).TextColor(t.Tx2)
		short := bl.Hash
		if len(short) > 7 {
			short = short[:7]
		}
		meta := short
		if bl.Uncommitted {
			meta = "未提交"
		} else if short != "" {
			n, ready := a.blameCommitFileCount(bl.Hash)
			switch {
			case !ready:
				meta = short + " · 正在读取文件数…"
			case n == 1:
				meta = short + " · 本次修改影响 1 个文件"
			default:
				meta = fmt.Sprintf("%s · 本次修改影响 %d 个文件", short, n)
			}
		}
		ui.Text(c, meta).Font("monospace").FontSize(theme.FsMicro).TextColor(t.Tx4)
		ui.Row(c).Gap(8).Children(func() {
			kit.Btn(c, "查看提交", kit.BtnOpt{Small: true, Icon: "commit", Disabled: bl.Uncommitted || bl.Hash == "", Tip: map[bool]string{true: "未提交的改动没有提交", false: ""}[bl.Uncommitted]}).OnClick(func() {
				if bl.Hash == "" {
					return
				}
				a.openGraph()
				a.graphTab.sel = bl.Hash
				a.scmCommitFiles(bl.Hash)
				blameLog.Info("追溯卡查看提交", "hash", shortHash(bl.Hash))
			})
			kit.Btn(c, "差异", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "diff", Disabled: bl.Uncommitted || bl.Hash == "", Tip: map[bool]string{true: "未提交的改动请用差异标签", false: ""}[bl.Uncommitted]}).OnClick(func() {
				if bl.Hash == "" {
					return
				}
				cm := a.blameCommitFor(bl.Hash)
				rel := a.wsRel(tb.path)
				a.scmOpenCommitFile(cm, rel, true)
				blameLog.Info("追溯卡打开差异", "hash", shortHash(bl.Hash), "path", rel)
			})
			kit.Btn(c, "复制哈希", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "copy", Disabled: bl.Hash == ""}).OnClick(func() {
				if bl.Hash != "" {
					a.copyOut(bl.Hash, "哈希")
				}
			})
		})
	})
}

func shortHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}

// blameGutterWrapHint 自动换行开启时说明追溯栏不可用。
func (a *App) blameGutterWrapHint(c *ui.Context, tb *ideTab) {
	if tb == nil || tb.st == nil || !a.prefs.Bool(pBlameGutter, false) || !tb.st.SoftWrap {
		return
	}
	t := a.tk
	ui.Row(c).Key("blame-wrap-hint").Absolute().Left(0).Top(0).Height(blameGutterHintH).PaddingX(8).
		AlignItems(ui.Center).Gap(6).Background(t.Fill1.Alpha(.92)).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		kit.Ic(c, "info", kit.S12).TextColor(t.Tx3)
		ui.Text(c, "自动换行已开启,追溯栏已隐藏(关闭换行可恢复)").FontSize(theme.FsMicro).TextColor(t.Tx3).SingleLine()
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
	t := a.tk
	if g == nil {
		ui.Row(c).Key("blame-no-repo").Absolute().Left(0).Top(0).Width(88).Padding(4, 6).
			Children(func() {
				ui.Text(c, "不在 Git 仓库").FontSize(9).TextColor(t.Tx4)
			})
		return
	}
	if g.busy {
		ui.Row(c).Key("blame-gutter-busy").Absolute().Left(0).Top(0).Width(88).Padding(4, 6).Children(func() {
			ui.Text(c, "读取中…").FontSize(9).TextColor(t.Tx4)
		})
		return
	}
	if len(g.blame) == 0 && len(g.log) == 0 {
		ui.Row(c).Key("blame-gutter-empty").Absolute().Left(0).Top(0).Width(88).Padding(4, 6).Children(func() {
			ui.Text(c, "暂无历史").FontSize(9).TextColor(t.Tx4)
		})
		return
	}
	first, last := tb.st.VisibleLines()
	if first < 0 {
		first = 0
	}
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
