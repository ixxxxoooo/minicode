package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 下一步建议(设计稿「会话流 · 排队引导 / 问答卡 / 回到最新」页「下一步建议」一节):
// 每轮正常结束后,在最后一轮回答的末尾(回合脚注之后)竖排 3 行可一键发送的建议,是时间线的一部分、随会话流滚动。建议不另发请求:系统提示词里要求 Claude 在回答末尾顺带给出一个
// <next_steps> 块(core.NextStepsPrompt),翻译器在流式正文里把它截走、发 core.EvNextSteps(见 core/nextsteps.go),
// 这里先存着(pending),回合正常结束时才显示。点一行 = 当作用户消息直接发送;悬停出现的铅笔 = 填进输入框再改。
//
// @author ixxxxoooo

var nsLog = diag.L("app.nextsteps")

// kNextSteps 是设置 → 对话 → 回答后给出下一步建议(默认开;新会话生效,因为说明是随系统提示词在进程启动时传给 claude 的)。
const kNextSteps = "chat.nextSteps"

// nextStepsState 是下一步建议条的状态(只在界面线程读写)。
type nextStepsState struct {
	pending  []string // 本回合回答里带的建议,等回合正常结束才显示
	items    []string // 正在显示的建议
	hidden   bool     // 点了 ✕:本轮不再显示(新回合 / 发送 / 切会话时随 nextStepsClear 复位)
	sess     string   // 这批建议属于哪个会话,切走就不显示
	replayed []string // 历史回放里最后一轮回答带的建议,回放结束才还原显示
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

// nextStepsOnReplayEnd 在历史回放结束时调用:最后一轮回答里留着的 <next_steps> 块(会话文件里本来就有)还原成建议,
// 重启 / 重新打开会话后仍能看到上次给的下一步;之后又有新提问、设置关着、或建议不属于最后一轮就不显示。
func (a *App) nextStepsOnReplayEnd() {
	items := a.ns.replayed
	a.nextStepsClear()
	if !a.nextStepsEnabled() || len(items) == 0 {
		return
	}
	a.ns.items, a.ns.sess = items, a.curSess
	nsLog.Info("还原下一步建议", "条数", len(items))
}

func demoNextSteps() []string {
	return []string{"运行测试确认没问题", "把这次改动提交", "再检查一下边界情况"}
}

// nextStepsVisible 此刻要不要画建议条。
func (a *App) nextStepsVisible() bool {
	if !a.nextStepsEnabled() || a.ns.hidden || a.ns.sess != a.curSess || len(a.ns.items) == 0 {
		return false
	}
	// 在输入框里打字不收起:建议在时间线里不挡输入,收起会让列表跳动。
	return a.st.Phase == core.PhaseIdle && a.q.len()+len(a.q.steers) == 0
}

// nextStepSend 点了一条建议:当作用户消息直接发送(与在输入框里敲完回车同一条路径)。
func (a *App) nextStepSend(text string) {
	nsLog.Info("发送下一步建议", "len", len(text))
	a.nextStepsClear()
	a.submitPrompt(text, false)
}

// nextStepsRow 是会话流末尾(回合脚注之后)的建议列表:标题行「下一步」+ 右侧 ✕(本轮不再显示),下面竖排最多 3 行,
// 行间 1px 细线,整行可点。它是最后一轮回答的延续,随虚拟列表滚动(flow_rows.go 在行表末尾追加),不再贴着输入框。
func (a *App) nextStepsRow(c *ui.Context) {
	t := a.tk
	ui.Column(c).Key("nsteps").Margin(6, 0, 0, 0).Children(func() {
		ui.Row(c).Height(theme.Px(22)).Gap(4).AlignItems(ui.Center).TextColor(t.Tx4).FontSize(theme.FsCap).Children(func() {
			kit.Ic(c, "spark", kit.S12)
			ui.Text(c, "下一步")
			ui.Box(c).Grow(1)
			kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "本轮不再显示"}).OnClick(func() { a.ns.hidden = true })
		})
		for i, s := range a.ns.items {
			a.nextStepLine(c, i, s)
		}
		ui.Box(c).Height(1).Background(t.Bd1)
	})
}

// nextStepLine 画一行建议:行首 ↵ 图标(第一条染强调色 = 最推荐),正文单行省略;悬停整行浅底色,
// 右侧出现铅笔(放进输入框编辑)与「发送」(窄档只留铅笔);点行本身 = 直接发送。
func (a *App) nextStepLine(c *ui.Context, i int, text string) {
	t := a.tk
	ui.Box(c).Height(1).Background(t.Bd1)
	row := ui.Row(c).Key("ns-" + itoa(i) + text).Height(theme.Px(32)).AlignItems(ui.Center)
	hov := row.Hovered()
	bg, fg, lead := ui.Transparent, t.Tx2, t.Tx4
	if hov {
		bg, fg, lead = t.Fill1, t.Tx1, t.AcTx
	} else if i == 0 {
		lead = t.Ac
	}
	row.Background(bg).Radius(t.RSm()).TextColor(fg).FontSize(theme.FsBody).Transition(kit.Fade)
	row.Children(func() {
		b := ui.ButtonBase(c).Key("nsb-" + itoa(i)).Height(theme.Px(32)).PaddingX(8).Gap(8).AlignItems(ui.Center).Cursor(ui.CursorPointer).Grow(1).MinWidth(0).Tooltip(text)
		b.Children(func() {
			ui.Row(c).TextColor(lead).Shrink(0).Children(func() { kit.Ic(c, "ret", kit.S12) })
			ui.Text(c, text).SingleLine().Grow(1).MinWidth(0)
			if hov && a.chatTier() != tierNarrow {
				ui.Text(c, "发送").TextColor(t.AcTx).FontSize(theme.FsCap).Shrink(0)
			}
		})
		if b.Clicked() {
			a.nextStepSend(text)
		}
		if hov {
			kit.Ib(c, "edit", kit.IbOpt{Small: true, Tip: "放进输入框编辑"}).Margin(0, 6, 0, 0).OnClick(func() {
				a.nextStepsClear()
				a.draft = text
				a.ux.focusDraft = true
			})
		}
	})
}
