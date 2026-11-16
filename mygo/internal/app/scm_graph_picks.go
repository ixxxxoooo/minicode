package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// graphTogglePick ⌘ 点击切换多选提交(保留最近两个)。
func (a *App) graphTogglePick(hash string) {
	g := &a.graphTab
	for i, h := range g.picks {
		if h == hash {
			g.picks = append(g.picks[:i], g.picks[i+1:]...)
			diag.L("scm").Info("图谱取消多选", "hash", shortenHash(hash))
			return
		}
	}
	if len(g.picks) == 1 && g.picks[0] == hash {
		a.toast = "已选中这条提交,请再选另一条"
		return
	}
	g.picks = append(g.picks, hash)
	if len(g.picks) > 2 {
		g.picks = g.picks[len(g.picks)-2:]
	}
	g.sel = hash
	a.scmCommitFiles(hash)
	diag.L("scm").Info("图谱多选", "picks", len(g.picks))
}

func (a *App) graphPicks() []string {
	if p := a.graphTab.picks; len(p) >= 2 {
		return append([]string(nil), p...)
	}
	return nil
}

func (a *App) graphClearPicks() {
	a.graphTab.picks = nil
	diag.L("scm").Info("图谱清除多选")
}

// graphPickBar 多选提交时的提示条(设计稿:选第二个提交的提示与取消)。
func (a *App) graphPickBar(c *ui.Context) {
	g := &a.graphTab
	if len(g.picks) == 0 {
		return
	}
	t := a.tk
	ui.Row(c).Key("graph-picks").Height(theme.Px(32)).Shrink(0).PaddingX(10).Gap(8).AlignItems(ui.Center).
		Background(t.InfoSoft).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		switch len(g.picks) {
		case 1:
			ui.Text(c, "已选 "+shortenHash(g.picks[0])+" · 按住 ⌘ 再点另一条提交以比较").FontSize(theme.FsCap).TextColor(t.Tx2).Grow(1).SingleLine()
		default:
			ui.Text(c, "已选 "+shortenHash(g.picks[0])+" 与 "+shortenHash(g.picks[1])).FontSize(theme.FsCap).TextColor(t.Tx2).Grow(1).SingleLine()
			kit.Btn(c, "比较", kit.BtnOpt{Small: true, Variant: kit.BtnPri, Icon: "diff"}).OnClick(func() {
				a.scmCompareTwo(g.picks[0], g.picks[1])
			})
		}
		kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "取消多选"}).OnClick(a.graphClearPicks)
	})
}
