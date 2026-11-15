package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 会话流滚离底部时显示「回到底部」;子 Agent 过程流同理(taskSt)。

func (a *App) flowFollowBottom() bool {
	const eps = 8
	return !a.flowNoFollow && a.flowSt.MaxY-a.flowSt.Y < eps
}

func (a *App) flowScrollBtn(c *ui.Context) {
	if a.flowFollowBottom() || len(a.st.Items) == 0 {
		return
	}
	t := a.tk
	kit.Btn(c, "↓ 回到底部", kit.BtnOpt{Small: true, Variant: kit.BtnSoft, Icon: "arrow"}).Absolute().Right(12).Bottom(8).ZIndex(5).
		OnClick(func() {
			a.flowNoFollow = false
			a.flowSt.Y = 1e9
		}).Shadow(0, 2, 8, 0, t.Shadow1()[0].Color)
}

func (a *App) taskScrollBtn(c *ui.Context) {
	if a.taskNoFollow || a.taskSt.MaxY-a.taskSt.Y < 8 {
		return
	}
	kit.Btn(c, "↓ 回到底部", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).Absolute().Right(8).Bottom(8).
		OnClick(func() { a.taskNoFollow = false; a.taskSt.Y = 1e9 })
}
