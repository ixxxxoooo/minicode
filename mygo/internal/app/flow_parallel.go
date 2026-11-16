package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
)

// parallelSubGroup 把连续 ≥2 个子 Agent 卡收成「并行 N 个」汇总步骤行。
func (a *App) parallelSubGroup(c *ui.Context, cards []*core.SubagentCard) {
	n := len(cards)
	if n < 2 {
		return
	}
	run, done := 0, 0
	for _, s := range cards {
		if s.Active() {
			run++
		} else if s.Status == core.SubCompleted || s.Status == core.SubFailed || s.Status == core.SubStopped {
			done++
		}
	}
	right := fmt.Sprintf("%d 个完成", done)
	if run > 0 {
		right = fmt.Sprintf("%d 个运行中", run)
	}
	id := "psub:" + cards[0].ID
	open := !a.prefs.Bool(pGroupFold, true)
	if set, ok := a.folds[id]; ok {
		open = set
	}
	sp := step{id: id, icon: "bot", verb: "子 Agent", obj: fmt.Sprintf("并行 %d 个", n), fam: famAgent, state: map[bool]string{true: "run", false: ""}[run > 0], right: right, hasBody: true, open: open}
	a.stepRow(c, sp, func() {
		a.nest(c, func() {
			for _, s := range cards {
				a.compactSubRow(c, s)
			}
		})
	})
	if !open && run > 0 {
		for i := len(cards) - 1; i >= 0; i-- {
			if cards[i].Active() {
				if step := cards[i].SubLatestStep(); step != "" {
					a.subLatestTicker(c, cards[i], step)
				} else {
					a.groupLatestSub(c, id, cards[i])
				}
				break
			}
		}
	}
}

func (a *App) groupLatestSub(c *ui.Context, grpID string, s *core.SubagentCard) {
	a.tickerRow(c, "psub-"+grpID, s.ID, tickRow{spin: true, tone: a.tk.ToolAgent, text: s.Description})
}
