package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 切换会话时,会话流各行淡入下落(设计稿 a-rise / stagger,间隔 30ms)。
// 视口上方与下方不排队:只有当前可见行按顺序播放入场,避免滚在中间时出现大片空白。

var flowRiseLog = diag.L("flow.rise")

const (
	flowRiseStep    = 30 * time.Millisecond  // 相邻可见行的间隔(同 StaggerRise)
	flowRiseTail    = 8 * time.Millisecond   // 超过标准窗口后的间隔
	flowRiseMaxIdx  = 12                     // 前 12 行用标准间隔
	flowRiseCap     = 400 * time.Millisecond // 延迟封顶(总动画上限)
	flowRiseAvgRow  = 72                     // 首帧未布局时用滚动偏移估算视口的默认行高(px)
	flowRiseShiftPx = 6
)

// flowScrollMem 记住每个会话离开时的滚动位置。
type flowScrollMem struct {
	y        float32
	noFollow bool
	awayLen  int
}

// flowRiseDelay 第 stagger 位(0 起,仅对「当前视口内、从第一可见行起算」的行)的出场延迟。
func flowRiseDelay(stagger int) time.Duration {
	if stagger < 0 {
		return 0
	}
	if stagger <= flowRiseMaxIdx {
		return time.Duration(stagger) * flowRiseStep
	}
	tail := flowRiseMaxIdx*flowRiseStep + time.Duration(stagger-flowRiseMaxIdx)*flowRiseTail
	if tail > flowRiseCap {
		return flowRiseCap
	}
	return tail
}

// flowListLaidOut 上一帧列表是否已完成布局(Visible 的 last≥0)。
func flowListLaidOut(first, last int) bool {
	return last >= 0
}

// flowRiseViewport 根据行号、视口与滚动估算,决定这一行是否瞬时显示,以及其在可见序列里的 stagger 序号。
func flowRiseViewport(row, first, last, total int, laidOut bool, scrollY, viewH float32) (stagger int, instant bool) {
	if row < 0 || total <= 0 {
		return 0, true
	}
	if !laidOut {
		ef := flowRiseEstimateFirst(scrollY, viewH, float32(flowRiseAvgRow), total)
		el := flowRiseEstimateLast(scrollY, viewH, float32(flowRiseAvgRow), total, ef)
		first, last = ef, el
		if scrollY <= 0 && viewH <= 0 {
			// 首帧既没有布局也没有滚动量:不播,避免整屏透明
			return 0, true
		}
	}
	if row < first {
		return 0, true
	}
	if row > last {
		return 0, true
	}
	return row - first, false
}

// flowRiseEstimateFirst 用滚动偏移与默认行高估算首个可见行(布局未知时)。
func flowRiseEstimateFirst(scrollY, viewH, avgRow float32, total int) int {
	if total <= 0 {
		return 0
	}
	if scrollY <= 0 || avgRow <= 0 {
		return 0
	}
	i := int(scrollY / avgRow)
	if i >= total {
		return max(0, total-1)
	}
	return i
}

// flowRiseEstimateLast 估算末个可见行(布局未知时)。
func flowRiseEstimateLast(scrollY, viewH, avgRow float32, total, first int) int {
	if total <= 0 {
		return -1
	}
	if avgRow <= 0 || viewH <= 0 {
		return min(total-1, first)
	}
	n := int((scrollY + viewH) / avgRow)
	if n >= total {
		n = total - 1
	}
	if n < first {
		n = first
	}
	return n
}

func (a *App) saveFlowScroll(sess string) {
	if sess == "" {
		return
	}
	if a.flowScroll == nil {
		a.flowScroll = map[string]flowScrollMem{}
	}
	a.flowScroll[sess] = flowScrollMem{y: a.flowSt.Y, noFollow: a.flowNoFollow, awayLen: a.flowAwayLen}
}

func (a *App) restoreFlowScroll(sess string) {
	if sess == "" {
		return
	}
	mem, ok := a.flowScroll[sess]
	if !ok {
		return
	}
	a.flowNoFollow = mem.noFollow
	a.flowAwayLen = mem.awayLen
	a.flowSt.Y = mem.y
}

// triggerFlowRise 在切换到已有内容的会话时启动入场(同一会话只播一次,直到进程重启)。
func (a *App) triggerFlowRise() {
	sid := a.curSess
	if sid == "" || strings.HasPrefix(sid, "new:") {
		a.flowRiseAt, a.flowRiseSess = time.Time{}, ""
		return
	}
	if a.st == nil || len(a.st.Items) == 0 {
		a.flowRiseAt, a.flowRiseSess = time.Time{}, ""
		return
	}
	if a.flowRisePlayed != nil && a.flowRisePlayed[sid] {
		a.flowRiseAt, a.flowRiseSess = time.Time{}, ""
		return
	}
	a.flowRiseSess = sid
	a.flowRiseAt = kit.Now()
	flowRiseLog.Info("会话流入场", "会话", sid, "条目", len(a.st.Items))
}

func (a *App) finishFlowRiseIfDone() {
	if a.flowRiseAt.IsZero() || a.flowRiseSess == "" {
		return
	}
	if kit.Since(a.flowRiseAt) < flowRiseCap+kit.DurBase {
		return
	}
	if a.flowRisePlayed == nil {
		a.flowRisePlayed = map[string]bool{}
	}
	a.flowRisePlayed[a.flowRiseSess] = true
	a.flowRiseAt, a.flowRiseSess = time.Time{}, ""
}

// flowRiseRow 给会话流的一行套上切换入场;rowIdx 为行表序号。
func (a *App) flowRiseRow(el ui.Element, rowIdx, total int) {
	if !kit.MotionOn() || a.flowRiseAt.IsZero() || a.flowRiseSess != a.curSess {
		return
	}
	first, last := a.flowList.Visible()
	laidOut := flowListLaidOut(first, last)
	viewH := a.flowBox.H
	stagger, instant := flowRiseViewport(rowIdx, first, last, total, laidOut, a.flowSt.Y, viewH)
	if instant {
		return
	}
	delay := flowRiseDelay(stagger)
	p := float32(kit.Since(a.flowRiseAt)-delay) / float32(kit.DurBase)
	if p >= 1 {
		return
	}
	if p <= 0 {
		el.Opacity(0).Translate(0, flowRiseShiftPx)
		a.frameSoon()
		return
	}
	if p < 1 {
		a.frameSoon()
	}
	ease := 1 - (1-p)*(1-p)
	el.Opacity(ease).Translate(0, (1-ease)*flowRiseShiftPx)
}
