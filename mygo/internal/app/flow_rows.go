package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 会话流虚拟化(设计稿 resp 页「拖动 / 重排要丝滑」):
//
// 原先整条会话流是一个滚动容器里的一整棵树:窗口 / 分隔线每动 1px,所有条目(包括看不见的)都要重新量尺寸,
// 长会话每帧几十毫秒。现在把时间线拆成「行」(用户消息 / 一串步骤 / 助手正文 / 回合尾部……),
// 交给库的 ui.List:只构建并排版视口内(加上下各 2 行)的行,屏幕外的行用测过的高度占位,
// 滚动时用「锚点行 + 偏移」保位置,新内容在底部时 FollowEnd 贴底。
// 用户消息是「分节标题」(ListState.Header),吸顶由库按分节处理,不再靠 Sticky。

var flowPerfLog = diag.L("flow")

// flowRow 是会话流里的一行。
type flowRow struct {
	key    string
	header bool                         // 用户消息:分节标题,吸顶
	build  func(c *ui.Context, idx int) // 在行的 Context 里构建内容;idx 是这一行在行表里的序号
}

// plainRow 包装不关心行号的构建函数。
func plainRow(fn func(c *ui.Context)) func(*ui.Context, int) {
	return func(c *ui.Context, _ int) { fn(c) }
}

// flowRows 把时间线切成行。每帧都重算(只是几个切片与闭包,几千条也在 1 ms 内),
// 因为权限卡放行、问答作答、工具失败都会改变分组,而它们不改变条目数,没有可靠的缓存键。
func (a *App) flowRows() []flowRow {
	defer diag.SpanSlow("flow", "切分会话流行", 8*time.Millisecond)()
	return a.buildFlowRows(a.visibleItems())
}

// buildFlowRows 复刻原先 flowRounds / flowRound 的分组逻辑,只是每一块变成独立的一行。
func (a *App) buildFlowRows(items []core.Item) []flowRow {
	rows := make([]flowRow, 0, len(items)/2+4)
	seen := make(map[string]int, len(items)/2+4)
	add := func(r flowRow) {
		if n := seen[r.key]; n > 0 { // 行键必须唯一:偶尔有空 id 或重复 id,加后缀
			seen[r.key] = n + 1
			r.key += "#" + itoa(n)
		} else {
			seen[r.key] = 1
		}
		rows = append(rows, r)
	}
	sticky := a.prefs.Bool(pStickyUser, true)

	for i := 0; i < len(items); {
		u, ok := items[i].(*core.User)
		if !ok { // 开头没有用户消息(恢复的片段):按「步骤串 / 单条」各成一行
			j := i + 1
			if isStepItem(items[i]) {
				for j < len(items) && isStepItem(items[j]) {
					j++
				}
			}
			a.addTailRow(add, items[i:j])
			i = j
			continue
		}
		j := i + 1
		for j < len(items) {
			if _, ok := items[j].(*core.User); ok {
				break
			}
			j++
		}
		a.addRoundRows(add, u, items[i+1:j], sticky)
		i = j
	}
	if _, ok := a.statusVisible(); ok { // 状态行:回合进行中始终在场(设计稿 resp-time)
		add(flowRow{key: "rt-st", build: plainRow(a.statusRow)})
	}
	if a.st.Rewind != nil {
		add(flowRow{key: "rewind", build: plainRow(a.flowRewindCard)})
	}
	return rows
}

// addTailRow 为「不属于任何回合」的一段条目加行。
func (a *App) addTailRow(add func(flowRow), seg []core.Item) {
	first := seg[0]
	if isStepItem(first) {
		add(flowRow{key: "s:" + first.ItemID(), build: plainRow(func(c *ui.Context) { a.stepGroup(c, seg) })})
		return
	}
	if flowTailVisible(first) {
		a.addItemRows(add, first)
	}
}

func (a *App) addRoundRows(add func(flowRow), u *core.User, rest []core.Item, sticky bool) {
	uid := u.ID
	foldTurn := a.shouldFoldTurn(uid, rest, a.roundComplete(uid))
	var allSteps []core.Item
	if foldTurn {
		allSteps = turnStepItems(rest)
	}
	add(flowRow{key: "u:" + uid, header: sticky, build: func(c *ui.Context, idx int) { a.userMsg(c, u, idx) }})
	foldedSteps := false
	for i := 0; i < len(rest); {
		last := i
		if isStepItem(rest[i]) {
			j := i
			for j < len(rest) && isStepItem(rest[j]) {
				j++
			}
			if foldTurn {
				if !foldedSteps {
					add(flowRow{key: "fold:" + uid, build: plainRow(func(c *ui.Context) { a.turnFoldGroup(c, uid, allSteps) })})
					foldedSteps = true
				}
			} else {
				seg := rest[i:j]
				add(flowRow{key: "s:" + seg[0].ItemID(), build: plainRow(func(c *ui.Context) { a.stepGroup(c, seg) })})
			}
			last = j - 1
		} else {
			if !(foldTurn && foldedSteps && turnFoldedItem(rest[i])) && flowTailVisible(rest[i]) {
				a.addItemRows(add, rest[i])
			}
			last = i
		}
		if last == len(rest)-1 {
			// 改动卡汇总的是整个会话的改动,只放在最后一轮,免得每一轮都显示同一份累计数字。
			if a.st.Phase != core.PhaseRunning && rest[last] == a.st.Items[len(a.st.Items)-1] && len(a.changedFiles()) > 0 {
				add(flowRow{key: "chg:" + uid, build: plainRow(a.changesCard)})
			}
			_, hasStats := a.st.TurnStats[uid]
			_, hasTiming := a.st.Timings[uid]
			if dur := a.st.Durations[uid]; hasStats || hasTiming || dur > 0 { // 有用时记录也要画脚注(没有 token 统计的回合)
				add(flowRow{key: "foot:" + uid, build: plainRow(func(c *ui.Context) { a.turnFooterActions(c, uid) })})
			}
		}
		i = last + 1
	}
}

// addItemRows 为一条非步骤条目加行。助手正文很长时按 splitMarkdownChunks 切成多行,
// 虚拟列表只构建视口附近的块(整条放一行时,露出一角就要整段解析与排版)。
func (a *App) addItemRows(add func(flowRow), it core.Item) {
	if m, ok := it.(*core.Assistant); ok && len(m.Text) > mdChunkMin {
		if chunks := a.chunksOf(m); len(chunks) > 1 {
			for k, ch := range chunks {
				ch, last := ch, k == len(chunks)-1
				add(flowRow{key: "i:" + m.ID + "~" + itoa(k), build: plainRow(func(c *ui.Context) { a.assistantChunk(c, m, ch, last) })})
			}
			return
		}
	}
	add(flowRow{key: "i:" + it.ItemID(), build: plainRow(func(c *ui.Context) { a.tailItem(c, it) })})
}

// flowTailVisible 这类条目在会话流里会画出东西(其余类型 tailItem 什么都不画,不要留空行)。
func flowTailVisible(it core.Item) bool {
	switch v := it.(type) {
	case *core.Assistant:
		// 模型在工具调用之间常吐出只有空白的文本块(「\n\n」):不画,否则每个都留一段空行,
		// 折叠成「已处理 N 步」后它们堆在一起就是一大块空白。
		return strings.TrimSpace(v.Text) != ""
	case *core.Notice, *core.TaskNote, *core.PlanApproval, *core.Compaction:
		return true
	case *core.QuestionRequest:
		return v.Answered
	}
	return false
}

// tailItem 画一条非步骤条目(原 flowTail 的 switch)。
func (a *App) tailItem(c *ui.Context, it core.Item) {
	switch v := it.(type) {
	case *core.Assistant:
		a.assistant(c, v)
	case *core.Notice:
		a.notice(c, v)
	case *core.TaskNote:
		a.taskNoteLine(c, v)
	case *core.PlanApproval:
		a.planCard(c, v)
	case *core.Compaction:
		a.compactCard(c, v)
	case *core.QuestionRequest:
		if v.Answered {
			a.askEcho(c, v)
		}
	}
}

// flowListView 画会话流:虚拟列表 + 自绘滚动条。
func (a *App) flowListView(c *ui.Context) {
	a.flowPerf.endFrame(len(a.st.Items))
	rows := a.flowRows()
	st := &a.flowList
	st.FollowEnd = true
	st.Key = func(i int) any { return rows[i].key }
	st.Header = func(i int) bool { return rows[i].header }
	pad := a.flowPad()
	c.Key("flow") // 列表的 Key 必须在构造之前给
	list := ui.List(c, st, len(rows)).Grow(1).MinHeight(0).Gap(8).Padding(12, 0, 18, 0).TrackScroll(&a.flowSt)
	list = kit.ApplyNativeScroll(list, "flow", &a.flowSt, false, true) // 滚动条显示策略(自动隐藏 / 常驻)与活动追踪
	list.Rows(func(r ui.ListRow) {
		if r.Index < 0 || r.Index >= len(rows) {
			return
		}
		rc := r.Context
		t0 := time.Now()
		rowWrap := ui.Column(rc).AlignItems(ui.Center)
		rowWrap.Children(func() {
			a.withFlowDisplay(func() {
				a.flowColPad(rc, pad, func() { rows[r.Index].build(rc, r.Index) })
			})
		})
		a.flowPerf.note(rows[r.Index].key, time.Since(t0))
	})
	a.finishFlowRiseIfDone()
	if b := list.Bounds(); b.H > 0 {
		a.flowBox = b
	}
}

// flowColPad 是会话流的居中列(同 flowCol,内边距由调用方给)。
func (a *App) flowColPad(c *ui.Context, pad float32, inner func()) {
	ui.Column(c).MaxWidth(a.flowMaxWidth()).PaddingX(pad).AlignItems(ui.Stretch).WidthPercent(100).Children(inner)
}

// userStuck 报告第 idx 行(用户消息)此刻是否被钉在列表顶部(用上一帧的数据)。
func (a *App) userStuck(card ui.Rect, idx int) bool {
	if idx < 0 || card.H <= 0 || a.flowBox.H <= 0 {
		return false
	}
	first, _ := a.flowList.Visible()
	if first > idx {
		return true
	}
	return first == idx && a.flowSt.Y > 1 && card.Y <= a.flowBox.Y+0.5
}

// flowPerf 统计会话流每帧构建了哪些行、各花多久:拖动 / 重排时一眼看出是哪种行最重。
// 行是在列表布局里按需构建的,所以「一帧」的统计在下一帧开头结算(超过 12 ms 才写日志)。
type flowPerf struct {
	rows   int
	total  time.Duration
	kinds  map[string]*flowKindStat
	worst  string
	worstD time.Duration
}

type flowKindStat struct {
	n int
	d time.Duration
}

func (p *flowPerf) note(key string, d time.Duration) {
	p.rows++
	p.total += d
	kind := key
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			kind = key[:i]
			break
		}
	}
	if p.kinds == nil {
		p.kinds = map[string]*flowKindStat{}
	}
	ks := p.kinds[kind]
	if ks == nil {
		ks = &flowKindStat{}
		p.kinds[kind] = ks
	}
	ks.n++
	ks.d += d
	if d > p.worstD {
		p.worst, p.worstD = key, d
	}
}

func (p *flowPerf) endFrame(items int) {
	if p.rows > 0 && p.total > 12*time.Millisecond {
		kv := []any{"条目", items, "本帧构建行数", p.rows, "构建合计", p.total.Round(10 * time.Microsecond), "最慢行", p.worst, "最慢耗时", p.worstD.Round(10 * time.Microsecond)}
		for k, v := range p.kinds {
			kv = append(kv, "行型_"+k, itoa(v.n)+"行/"+v.d.Round(10*time.Microsecond).String())
		}
		flowPerfLog.Warn("会话流行构建偏慢", kv...)
	}
	*p = flowPerf{kinds: p.kinds}
	for k := range p.kinds {
		delete(p.kinds, k)
	}
}

// chunksOf 取助手正文的分块,按「条目 id + 文本长度」缓存(正文只会在末尾增长),免得每帧都把几十万字扫一遍。
func (a *App) chunksOf(m *core.Assistant) []string {
	if e, ok := a.mdChunks[m.ID]; ok && e.n == len(m.Text) {
		return e.chunks
	}
	if a.mdChunks == nil || len(a.mdChunks) > 64 {
		a.mdChunks = map[string]mdChunkEntry{}
	}
	ch := splitMarkdownChunks(m.Text)
	a.mdChunks[m.ID] = mdChunkEntry{n: len(m.Text), chunks: ch}
	return ch
}

type mdChunkEntry struct {
	n      int
	chunks []string
}
