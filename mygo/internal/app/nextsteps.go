package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 下一步建议(设计稿「会话流 · 排队引导 / 问答卡 / 回到最新」页「下一步建议」一节):
// 每轮正常结束后,输入框正上方给 3 条可一键发送的建议胶囊。建议不另发请求:系统提示词里要求 Claude 在回答末尾顺带给出一个
// <next_steps> 块(core.NextStepsPrompt),翻译器在流式正文里把它截走、发 core.EvNextSteps(见 core/nextsteps.go),
// 这里先存着(pending),回合正常结束时才显示。点胶囊 = 当作用户消息直接发送;悬停出现的铅笔 = 填进输入框再改。
//
// @author ixxxxoooo

var nsLog = diag.L("app.nextsteps")

// kNextSteps 是设置 → 对话 → 回答后给出下一步建议(默认开;新会话生效,因为说明是随系统提示词在进程启动时传给 claude 的)。
const kNextSteps = "chat.nextSteps"

// nextStepsState 是下一步建议条的状态(只在界面线程读写)。
type nextStepsState struct {
	pending []string // 本回合回答里带的建议,等回合正常结束才显示
	items   []string // 正在显示的建议
	hidden  bool     // 点了 ✕:本轮不再显示
	sess    string   // 这批建议属于哪个会话,切走就不显示
}

func (a *App) nextStepsEnabled() bool {
	return a.prefs != nil && a.prefs.Bool(kNextSteps, true)
}

// nextStepsPrompt 是要追加到系统提示词的说明(设置关着时为空)。
func (a *App) nextStepsPrompt() string {
	if !a.nextStepsEnabled() {
		return ""
	}
	return core.NextStepsPrompt
}

// nextStepsClear 清掉建议(新回合开始 / 发送 / 编辑时调用)。
func (a *App) nextStepsClear() { a.ns = nextStepsState{} }

// onNextSteps 收到回答里带的建议:先存着,回合正常结束时才显示。
func (a *App) onNextSteps(items []string) {
	if a.nextStepsEnabled() {
		a.ns.pending = items
	}
}

// nextStepsOnTurnEnd 在回合正常结束后(afterEvent)调用:把回答里带的建议升级为显示。
func (a *App) nextStepsOnTurnEnd() {
	items := a.ns.pending
	a.nextStepsClear()
	if !a.nextStepsEnabled() {
		return
	}
	if items == nil && a.cuDemo() { // 演示模式:演示后端不会带建议块,用写好的 3 条,界面才能复现
		items = demoNextSteps()
	}
	if len(items) == 0 {
		return // 这一轮 Claude 没给(只跑了工具 / 简单问答 / 设置刚打开还在旧会话里):不显示,也不另发请求
	}
	if a.st.Phase == core.PhaseRunning || a.pendingDock() != nil || a.q.len()+len(a.q.steers) > 0 {
		return // 马上有下一轮 / 正在等用户批准:这时候给建议没意义
	}
	a.ns.items, a.ns.sess = items, a.curSess
	nsLog.Info("显示下一步建议", "条数", len(items))
}

func demoNextSteps() []string {
	return []string{"运行测试确认没问题", "把这次改动提交", "再检查一下边界情况"}
}

// nextStepsVisible 此刻要不要画建议条。
func (a *App) nextStepsVisible() bool {
	if !a.nextStepsEnabled() || a.ns.hidden || a.ns.sess != a.curSess || len(a.ns.items) == 0 {
		return false
	}
	if a.st.Phase != core.PhaseIdle || a.q.len()+len(a.q.steers) > 0 {
		return false
	}
	return trim(a.draftNow()) == "" // 开始输入就收起,清空后同一批建议重新出现
}

// nextStepSend 点了一条建议:当作用户消息直接发送(与在输入框里敲完回车同一条路径)。
func (a *App) nextStepSend(text string) {
	nsLog.Info("发送下一步建议", "len", len(text))
	a.nextStepsClear()
	a.submitPrompt(text, false)
}

// nextStepsBar 是输入框正上方的建议条:「下一步」+ 最多 3 个胶囊 + 右侧 ✕(本轮不再显示)。
func (a *App) nextStepsBar(c *ui.Context) {
	if !a.nextStepsVisible() {
		return
	}
	t := a.tk
	ui.Row(c).Key("nsteps").Margin(0, 14, 6, 14).Gap(8).Children(func() {
		ui.Row(c).Grow(1).MinWidth(0).Wrap().Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Gap(4).AlignItems(ui.Center).Shrink(0).Margin(0, 2, 0, 0).TextColor(t.Tx4).FontSize(theme.FsCap).Children(func() {
				kit.Ic(c, "spark", kit.S12)
				ui.Text(c, "下一步")
			})
			for i, s := range a.ns.items {
				a.nextStepChip(c, i, s)
			}
		})
		kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "本轮不再显示"}).OnClick(func() { a.ns.hidden = true })
	})
}

// nextStepChip 画一个建议胶囊:第一条(最推荐)是强调淡底;点胶囊发送,悬停出现铅笔把它放进输入框编辑。
func (a *App) nextStepChip(c *ui.Context, i int, text string) {
	t := a.tk
	rec := i == 0
	row := ui.Row(c).Key("ns-" + itoa(i) + text).Height(theme.Px(24)).MaxWidth(420).Radius(theme.RFull).AlignItems(ui.Center).Shrink(1)
	hov := row.Hovered()
	bg, bd, fg := t.Raised, t.Bd2, t.Tx2
	switch {
	case rec && hov:
		bg, bd, fg = t.AcSoft, t.Ac, t.AcTx
	case rec:
		bg, bd, fg = t.AcSoft, t.AcLine, t.AcTx
	case hov:
		bg, bd, fg = t.Fill1.Over(t.Raised), t.Bd3, t.Tx1
	}
	row.Background(bg).Border(1, bd).TextColor(fg).FontSize(theme.FsCap).Transition(kit.Fade)
	row.Children(func() {
		b := ui.ButtonBase(c).Key("nsb-" + itoa(i)).Height(theme.Px(22)).PaddingX(10).AlignItems(ui.Center).Cursor(ui.CursorPointer).Grow(1).MinWidth(0).Tooltip(text)
		b.Children(func() { ui.Text(c, text).SingleLine() })
		if b.Clicked() {
			a.nextStepSend(text)
		}
		if hov {
			kit.Ib(c, "edit", kit.IbOpt{Small: true, Tip: "放进输入框编辑"}).Margin(0, 3, 0, 0).OnClick(func() {
				a.nextStepsClear()
				a.draft = text
				a.ux.focusDraft = true
			})
		}
	})
}
