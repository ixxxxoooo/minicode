package app

import (
	"github.com/ixxxxoooo/minicode/internal/diag"
)

// graphTogglePick ⌘ 点击切换多选提交(保留最近两个)。
func (a *App) graphTogglePick(hash string) {
	g := &a.graphTab
	for i, h := range g.picks {
		if h == hash {
			g.picks = append(g.picks[:i], g.picks[i+1:]...)
			return
		}
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
