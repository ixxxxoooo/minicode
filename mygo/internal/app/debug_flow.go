package app

import (
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

var flowDbgLog = diag.L("app.flowdbg")

// flowDebug 处理会话流 / 排队 / 问答 / 滚动的 -ed 动作(cmd/shot 与开发版调试)。
func (a *App) flowDebug(name, arg string) bool {
	switch name {
	case "queue":
		a.QueuePrompts("先检查单测是否全绿", "再更新 README 里的路由示例")
		flowDbgLog.Info("调试排队", "条数", a.q.len())
		return true
	case "steer":
		txt := "改一下匹配逻辑,优先用 normalize"
		if arg != "" {
			txt = arg
		}
		if a.st.Phase != core.PhaseRunning {
			a.st.Apply(core.EvTurnStarted{})
		}
		a.SeedSteers(txt)
		flowDbgLog.Info("调试引导", "正文", txt)
		return true
	case "ask":
		qid := ""
		cur := 0
		if arg != "" {
			if q, n, ok := strings.Cut(arg, ":"); ok && n != "" {
				qid, cur = q, max(atoi(n)-1, 0)
			} else {
				cur = max(atoi(arg)-1, 0)
			}
		}
		if qid == "" {
			qid = a.pendingQuestionID()
		}
		a.SetAskCur(qid, cur, false)
		flowDbgLog.Info("调试问答题号", "id", qid, "cur", cur)
		return true
	case "latest":
		fresh := arg == "fresh"
		a.PrepFlowScrollAway(fresh)
		if a.flowSt.MaxY > 64 {
			a.ApplyFlowScrollAway()
		}
		if fresh {
			a.AppendFlowTailForFresh()
		}
		flowDbgLog.Info("调试滚离底部", "fresh", fresh)
		return true
	default:
		return false
	}
}

// pendingQuestionID 返回当前待处理提问的 id(没有则空)。
func (a *App) pendingQuestionID() string {
	if it := a.pendingDock(); it != nil {
		if q, ok := it.(*core.QuestionRequest); ok {
			return q.ID
		}
	}
	for _, it := range a.st.Items {
		if q, ok := it.(*core.QuestionRequest); ok && !q.Answered {
			return q.ID
		}
	}
	return ""
}

// SeedSteers 登记引导行(截图用):from = 当前条目数,回合需保持进行中且其后无新非用户条目,状态才稳定。
func (a *App) SeedSteers(texts ...string) {
	from := len(a.st.Items)
	// 必须与 syncSteers / 真实的 steerSend 一样用真实时钟:截图注入的是固定的 a.now()(昨天),
	// 用它登记会被 20 秒兜底超时立刻放出,引导行就看不到了。
	now := time.Now()
	for _, t := range texts {
		show := t
		if len(show) > 48 {
			show = show[:45] + "…"
		}
		a.q.addSteer(t, show, from, now)
	}
}

// SeedQueuePaused 排队暂停态:需已有多条排队且回合被中断结束。
func (a *App) SeedQueuePaused(texts ...string) {
	if len(texts) == 0 {
		texts = []string{"顺便把 README 里的路由示例也更新一下", "再跑一遍完整测试"}
	}
	a.QueuePrompts(texts...)
	if a.st.Phase != core.PhaseRunning {
		a.st.Apply(core.EvTurnStarted{})
	}
	a.st.Apply(core.EvTurnEnded{Reason: core.StopCancelled})
	a.afterEvent(core.EvTurnEnded{Reason: core.StopCancelled})
}

// SetAskCur 切换问答向导到第 cur 题(0 起);presetFirst 为真时预填第 1 题答案(多选一项)。
func (a *App) SetAskCur(qid string, cur int, presetFirst bool) {
	for _, it := range a.st.Items {
		q, ok := it.(*core.QuestionRequest)
		if !ok || qid != "" && q.ID != qid {
			continue
		}
		s := a.qaOf(q)
		if cur < 0 {
			cur = 0
		}
		if cur >= len(q.Questions) {
			cur = len(q.Questions) - 1
		}
		s.cur = cur
		if presetFirst && len(q.Questions) > 0 && len(q.Questions[0].Options) > 0 {
			s.sel[0] = []int{0}
		}
		return
	}
}

// PrepFlowScrollAway 为「回到最新」按钮造状态:须先 Frame 量出 MaxY,再调 ApplyFlowScrollAway。
func (a *App) PrepFlowScrollAway(fresh bool) {
	a.flowNoFollow = true
	if fresh {
		a.flowAwayLen = flowTailMark(a.st.Items) - 1
		if a.flowAwayLen < 0 {
			a.flowAwayLen = 0
		}
	} else {
		a.flowAwayLen = flowTailMark(a.st.Items)
	}
}

// ApplyFlowScrollAway 在 ScrollState.MaxY 已量好后滚离底部(距底 ≥ 48px)。
func (a *App) ApplyFlowScrollAway() {
	const gap float32 = 48
	if a.flowSt.MaxY > gap+32 {
		a.flowSt.Y = a.flowSt.MaxY - gap
	} else {
		a.flowSt.Y = 0
	}
}

// AppendFlowTailForFresh 追加一条助手摘要,使 flowTailMark 变大(配合 latest-fresh)。
func (a *App) AppendFlowTailForFresh() {
	id := "dbg-tail"
	a.st.Apply(core.EvAppend{Item: &core.Assistant{ID: id, Text: "（调试）回合又有新输出…"}})
}
