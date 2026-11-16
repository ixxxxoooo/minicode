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
		if turnFoldedItem(it) {
			out = append(out, it)
		}
	}
	return out
}

// turnFoldedItem 回合折叠时收进「已处理」的条目(步骤行 + 已回答的提问卡)。
func turnFoldedItem(it core.Item) bool {
	if isStepItem(it) {
		return true
	}
	if q, ok := it.(*core.QuestionRequest); ok && q.Answered {
		return true
	}
	return false
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
		if v := doneToolOf(it); v != nil {
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
			// 里面每一串按当前档位再排一次(精简档 = 摘要行,标准档 = 探索组 + 逐行);已答的提问卡把串断开
			var run []core.Item
			flush := func() {
				if len(run) > 0 {
					a.stepSeq(c, run)
					run = nil
				}
			}
			for _, it := range steps {
				if q, ok := it.(*core.QuestionRequest); ok && q.Answered {
					flush()
					a.askEcho(c, q)
					continue
				}
				run = append(run, it)
			}
			flush()
		})
	})
}
