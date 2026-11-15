package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// pendingQueue 列出按时间线顺序仍未处理的权限 / 提问(含子 Agent 内嵌)。
func (a *App) pendingQueue() []core.Item {
	var out []core.Item
	var walk func(items []core.Item)
	walk = func(items []core.Item) {
		for _, it := range items {
			switch v := it.(type) {
			case *core.PermissionRequest:
				if v.Resolved == "" {
					out = append(out, v)
				}
			case *core.QuestionRequest:
				if !v.Answered {
					out = append(out, v)
				}
			case *core.SubagentCard:
				walk(v.Children)
			}
		}
	}
	walk(a.st.Items)
	return out
}

// pendingDock 返回队列首项(兼容旧调用点)。
func (a *App) pendingDock() core.Item {
	q := a.pendingQueue()
	if len(q) == 0 {
		return nil
	}
	return q[0]
}

// syncDockComposer 审批卡顶替输入框时保存草稿;全部处理完后淡入输入框并恢复草稿。
func (a *App) syncDockComposer() {
	pending := len(a.pendingQueue()) > 0
	if pending {
		if !a.dockStash {
			a.dockDraft = a.draft
			a.dockStash = true
		}
		return
	}
	if a.dockStash {
		a.draft = a.dockDraft
		a.dockDraft = ""
		a.dockStash = false
		a.composerInAt = time.Now()
		a.frameSoon()
	}
}

// dockBadge 待处理队列序号(设计稿 task-spec「1 / 3」)。
func (a *App) dockBadge(c *ui.Context) {
	n := len(a.pendingQueue())
	if n <= 1 {
		return
	}
	t := a.tk
	ui.Text(c, "1 / "+itoa(n)).FontSize(theme.FsCap).TextColor(t.Tx3).FontFeatures("tnum").Shrink(0)
}
