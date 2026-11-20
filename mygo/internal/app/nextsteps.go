package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 下一步建议(设计稿「会话流 · 排队引导 / 问答卡 / 回到最新」页「下一步建议」一节):
// 每轮正常结束后,输入框正上方给 3 条可一键发送的建议胶囊。建议由 Haiku 的一次精简调用生成(见 claude.RunPromptLeanCtx),
// 只带最近一轮的「用户要求 + Claude 回复尾部 + 改动文件」,不带整个会话。点胶囊 = 当作用户消息直接发送;悬停出现的铅笔 = 填进输入框再改。
//
// @author ixxxxoooo

var nsLog = diag.L("app.nextsteps")

const (
	kNextSteps = "chat.nextSteps" // 设置 → 对话 → 回答后给出下一步建议(默认开)
	nsCount    = 3                // 每轮给几条
	nsMaxRunes = 40               // 单条上限(界面再按宽度省略)
	nsTimeout  = 15 * time.Second // 生成超时:到点静默不显示
)

// nsSystem 是精简调用的系统提示词(替换默认那份,省 token)。
const nsSystem = "你是编程助手里的「下一步建议」生成器,只按要求的格式输出,不要解释。"

// nextStepsCtx 是生成建议用的最近一轮摘要。
type nextStepsCtx struct {
	ask, reply string
	files      []string
}

// nextStepsState 是下一步建议条的状态(只在界面线程读写)。
type nextStepsState struct {
	items   []string
	loading bool
	hidden  bool   // 点了 ✕:本轮不再显示
	sess    string // 这批建议属于哪个会话,切走就不显示
	gen     int    // 世代号:作废迟到的生成结果
	cancel  context.CancelFunc
	info    nextStepsCtx
	given   []string // 已给过的(「换一批」时让模型避开)
}

func (a *App) nextStepsEnabled() bool {
	return a.prefs != nil && a.prefs.Bool(kNextSteps, true)
}

// nextStepsClear 清掉当前建议并作废正在进行的生成(新回合开始 / 发送 / 关闭设置时调用)。
func (a *App) nextStepsClear() {
	if a.ns.cancel != nil {
		a.ns.cancel()
	}
	a.ns = nextStepsState{gen: a.ns.gen + 1}
}

// nextStepsOnTurnEnd 在回合正常结束后(afterEvent)调用:条件满足就开始生成。
func (a *App) nextStepsOnTurnEnd() {
	a.nextStepsClear()
	if !a.nextStepsEnabled() || testing.Testing() {
		return
	}
	if a.st.Phase == core.PhaseRunning || a.pendingDock() != nil || a.q.len()+len(a.q.steers) > 0 {
		return // 马上有下一轮 / 正在等用户批准:这时候给建议没意义
	}
	info := a.nextStepsContext()
	if info.reply == "" {
		return // 本轮 Claude 没有文字回复(只跑了工具),没有可建议的
	}
	a.ns.sess, a.ns.info = a.curSess, info
	if a.cuDemo() { // 演示模式:写好的建议,不调用 claude
		a.ns.items = demoNextSteps()
		return
	}
	a.nextStepsGenerate(false)
}

func demoNextSteps() []string {
	return []string{"运行测试确认没问题", "把这次改动提交", "再检查一下边界情况"}
}

// nextStepsContext 取最近一轮(最后一条用户消息之后)的摘要:用户要求、Claude 文字回复的尾部、改过的文件。
func (a *App) nextStepsContext() nextStepsCtx {
	items := a.st.Items
	start := 0
	for i := len(items) - 1; i >= 0; i-- {
		if _, ok := items[i].(*core.User); ok {
			start = i
			break
		}
	}
	var info nextStepsCtx
	var reply []string
	seen := map[string]bool{}
	for _, it := range items[start:] {
		switch v := it.(type) {
		case *core.User:
			info.ask = v.Text
		case *core.Assistant:
			if s := strings.TrimSpace(v.Text); s != "" {
				reply = append(reply, s)
			}
		default:
			tc := doneToolOf(it)
			if tc == nil || (tc.Kind != core.KindEdit && tc.Kind != core.KindWrite && tc.Kind != core.KindDelete && tc.Kind != core.KindMove) {
				continue
			}
			for _, p := range tc.Locations {
				if b := filepath.Base(p); b != "" && !seen[b] && len(info.files) < 8 {
					seen[b] = true
					info.files = append(info.files, b)
				}
			}
		}
	}
	info.ask = headRunes(strings.TrimSpace(info.ask), 600)
	info.reply = tailRunes(strings.Join(reply, "\n\n"), 1500)
	return info
}

func headRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func tailRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return "…" + string(r[len(r)-n:])
	}
	return s
}

// nextStepsPrompt 拼生成提示;avoid 是已给过的建议(换一批时避开)。
func nextStepsPrompt(info nextStepsCtx, avoid []string) string {
	var b strings.Builder
	b.WriteString("下面是编程助手 Claude 刚结束的一轮对话。请给出用户最可能想接着对 Claude 说的 3 句话。\n")
	b.WriteString("要求:\n")
	b.WriteString("- 用和「用户的要求」相同的语言;口吻是用户对 Claude 说话(祈使句),每条不超过 24 个字,能直接执行,不要「继续」「好的」这种空话;\n")
	b.WriteString("- 3 条方向不同(例如:验证 / 收尾 / 深入),按推荐度从高到低;\n")
	b.WriteString("- 如果 Claude 刚向用户提问或在等用户决定,建议就写成对它的几种不同回答;\n")
	b.WriteString("- 只输出一个 JSON 字符串数组,例如 [\"…\",\"…\",\"…\"],不要任何其它文字。\n")
	if len(avoid) > 0 {
		b.WriteString("- 不要和这些已给过的建议重复:" + strings.Join(avoid, " / ") + "\n")
	}
	b.WriteString("\n【用户的要求】\n" + info.ask + "\n\n【Claude 的回复】\n" + info.reply + "\n")
	if len(info.files) > 0 {
		b.WriteString("\n【本轮改动的文件】" + strings.Join(info.files, "、") + "\n")
	}
	return b.String()
}

// parseNextSteps 从模型输出里取出建议:优先 JSON 数组;不是 JSON 就按行取(去掉序号 / 引号)。去重、限长、最多 nsCount 条。
func parseNextSteps(out string) []string {
	var raw []string
	if i, j := strings.Index(out, "["), strings.LastIndex(out, "]"); i >= 0 && j > i {
		_ = json.Unmarshal([]byte(out[i:j+1]), &raw)
	}
	if len(raw) == 0 {
		for _, ln := range strings.Split(out, "\n") {
			ln = strings.TrimSpace(ln)
			ln = strings.TrimLeft(ln, "-*•0123456789.、) ")
			ln = strings.Trim(ln, "\"“”「」,")
			if ln != "" {
				raw = append(raw, ln)
			}
		}
	}
	var res []string
	seen := map[string]bool{}
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		if r := []rune(s); len(r) > nsMaxRunes {
			s = string(r[:nsMaxRunes]) + "…"
		}
		res = append(res, s)
		if len(res) == nsCount {
			break
		}
	}
	return res
}

// nextStepsGenerate 在后台生成一批建议。again = 「换一批」:把已给过的带上让模型避开。
func (a *App) nextStepsGenerate(again bool) {
	if a.ns.cancel != nil {
		a.ns.cancel()
	}
	if again {
		a.ns.given = append(a.ns.given, a.ns.items...)
	}
	a.ns.gen++
	gen, info, avoid := a.ns.gen, a.ns.info, append([]string(nil), a.ns.given...)
	a.ns.loading, a.ns.items, a.ns.hidden = true, nil, false
	ctx, cancel := context.WithCancel(context.Background())
	a.ns.cancel = cancel
	cwd := a.proj
	if cwd == "" {
		cwd = os.TempDir()
	}
	a.bg("nextsteps", func() func() {
		defer cancel()
		defer diag.SpanSlow("nextsteps", "生成下一步建议", 6*time.Second)()
		out, err := claude.RunPromptLeanCtx(ctx, cwd, nsSystem, nextStepsPrompt(info, avoid), haikuInline, nsTimeout)
		var items []string
		if err != nil {
			if ctx.Err() == nil { // 被新一轮作废不算失败
				nsLog.Warn("生成下一步建议失败", "err", err)
			}
		} else if items = parseNextSteps(out); len(items) == 0 {
			nsLog.Warn("下一步建议:模型输出解析不出建议", "len", len(out))
		}
		return func() {
			if a.ns.gen != gen { // 期间新回合开始 / 又点了换一批:丢弃
				return
			}
			a.ns.loading, a.ns.items, a.ns.cancel = false, items, nil
			nsLog.Info("下一步建议已生成", "条数", len(items))
		}
	})
}

// nextStepsVisible 此刻要不要画建议条。
func (a *App) nextStepsVisible() bool {
	if !a.nextStepsEnabled() || a.ns.hidden || a.ns.sess != a.curSess || (!a.ns.loading && len(a.ns.items) == 0) {
		return false
	}
	if a.st.Phase != core.PhaseIdle || a.q.len()+len(a.q.steers) > 0 {
		return false
	}
	return strings.TrimSpace(a.draftNow()) == "" // 开始输入就收起,清空后同一批建议重新出现
}

// nextStepSend 点了一条建议:当作用户消息直接发送(与在输入框里敲完回车同一条路径)。
func (a *App) nextStepSend(text string) {
	nsLog.Info("发送下一步建议", "len", len(text))
	a.nextStepsClear()
	a.submitPrompt(text, false)
}

// nextStepsBar 是输入框正上方的建议条:「下一步」+ 3 个胶囊(生成中是 3 个骨架)+ 右侧 ↻ 换一批 / ✕ 本轮不再显示。
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
			if a.ns.loading {
				for i := 0; i < nsCount; i++ {
					kit.SkelLine(c, 0, theme.Px(24)).Key("nsk" + itoa(i)).Width(104).Radius(theme.RFull)
				}
				return
			}
			for i, s := range a.ns.items {
				a.nextStepChip(c, i, s)
			}
		})
		ui.Row(c).Gap(1).Shrink(0).Children(func() {
			kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "换一批", Loading: a.ns.loading, Disabled: a.ns.loading}).OnClick(func() { a.nextStepsGenerate(true) })
			kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "本轮不再显示"}).OnClick(func() { a.ns.hidden = true })
		})
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
