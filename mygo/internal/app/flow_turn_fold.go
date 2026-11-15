package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
)

const pTurnFold = "chat.fold"

func (a *App) turnFoldEnabled() bool { return a.prefs.Bool(pTurnFold, true) }

func turnStepItems(rest []core.Item) []core.Item {
	var out []core.Item
	for _, it := range rest {
		if isStepItem(it) {
			out = append(out, it)
		}
	}
	return out
}

func turnMustStayOpen(rest []core.Item) bool {
	for _, it := range rest {
		switch v := it.(type) {
		case *core.ToolCall:
			if v.Status == core.ToolFailed || v.Status == core.ToolRunning {
				return true
			}
		case *core.SubagentCard:
			if v.Status == core.SubFailed || v.Active() {
				return true
			}
		case *core.PermissionRequest:
			if v.Resolved == "" || stringsContainsDeny(v.Resolved) {
				return true
			}
		case *core.PlanApproval:
			if v.Approved == nil {
				return true
			}
		}
		if core.AwaitsUser(it) {
			return true
		}
	}
	return false
}

func stringsContainsDeny(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "deny") || strings.Contains(l, "reject") || strings.Contains(s, "拒绝")
}

func (a *App) roundComplete(uid string) bool {
	if a.st.Phase != core.PhaseRunning {
		return true
	}
	for i := len(a.st.Items) - 1; i >= 0; i-- {
		if u, ok := a.st.Items[i].(*core.User); ok {
			return u.ID != uid
		}
	}
	return true
}

func (a *App) shouldFoldTurn(uid string, rest []core.Item, complete bool) bool {
	if !a.turnFoldEnabled() || !complete {
		return false
	}
	steps := turnStepItems(rest)
	if len(steps) == 0 {
		return false
	}
	return !turnMustStayOpen(rest)
}

func turnFoldSummary(steps []core.Item) (nSteps int, nFiles int) {
	seen := map[string]bool{}
	for _, it := range steps {
		nSteps++
		switch v := it.(type) {
		case *core.ToolCall:
			for _, ct := range v.Content {
				if d, ok := ct.(*core.DiffBlock); ok && d.Path != "" && !seen[d.Path] {
					seen[d.Path] = true
					nFiles++
				}
			}
			if p := inputStr(v, "file_path"); p != "" && !seen[p] {
				seen[p] = true
				nFiles++
			}
		}
	}
	return nSteps, nFiles
}

func (a *App) turnFoldGroup(c *ui.Context, uid string, steps []core.Item) {
	elapsed := ""
	if d, ok := a.st.Durations[uid]; ok && d > 0 {
		elapsed = core.FormatElapsed(int64(d) * 1000)
	}
	nSteps, nFiles := turnFoldSummary(steps)
	meta := fmt.Sprintf("%d 步", nSteps)
	if nFiles > 0 {
		meta += fmt.Sprintf(" · %d 个文件", nFiles)
	}
	id := "turnfold:" + uid
	open := false
	if set, ok := a.folds[id]; ok {
		open = set
	}
	sp := step{id: id, icon: "layers", verb: "已处理", obj: elapsed, meta: meta, fam: famExplore, hasBody: true, open: open}
	a.stepRow(c, sp, func() {
		a.nest(c, func() {
			for _, it := range steps {
				a.stepItem(c, it)
			}
		})
	})
	if !open {
		ui.Row(c).Margin(0, 0, 0, 22).Padding(6, 0, 0, 0).Children(func() {
			ui.Text(c, "点开查看本回合的全部步骤").FontSize(11).TextColor(a.tk.Tx4)
		})
	}
}
