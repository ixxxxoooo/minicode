package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 等待行(设计稿 resp-time 页):发送后立刻出现,拿到第一个字才让位。
// 只在「还没有任何可读内容」的阶段显示:连接 / 本地准备 / 等待模型,以及超过 3 秒的收尾。

const pSlowHint = "chat.slowHint"

// stageColor 阶段的颜色(等待行的转圈、脚注分段条、瀑布弹卡共用)。
func stageColor(t *theme.Tokens, k core.StageKind) ui.Color {
	switch k {
	case core.StageWait:
		return t.Warn
	case core.StageThink:
		return t.ToolThink
	case core.StageWrite:
		return t.Ac
	case core.StageTool:
		return t.ToolExec
	case core.StageAsk:
		return t.Info
	case core.StageFin:
		return t.Fill3
	}
	return t.Tx4
}

// 状态行(设计稿 resp-time 页「状态行:整个回合里始终在场,永不留白」):
// 点发送的同一帧出现(界面先乐观地把回合置为运行中),直到回合结束才淡出并收折;
// 不管模型在等、在想、在写还是在跑工具,流尾始终有一行会动的状态,不再有「看不到任何状态」的空档。

// statusVisible 状态行该不该在流尾:回合进行中,或刚结束、还在淡出。
func (a *App) statusVisible() (fading bool, ok bool) {
	if a.st.Phase == core.PhaseRunning {
		a.stRunning, a.stEnd = true, time.Time{}
		return false, true
	}
	if a.stRunning { // 刚结束:记下时刻,淡出 + 收折
		a.stRunning, a.stEnd = false, a.now()
	}
	if !kit.MotionOn() || a.stEnd.IsZero() || a.now().Sub(a.stEnd) >= kit.DurSlow {
		return false, false
	}
	return true, true
}

// turnStartTime 本回合的开始时刻:后端阶段事件给的优先,否则界面乐观置运行中时记的。
func (a *App) turnStartTime() time.Time {
	if ls := a.st.Live; ls != nil && !ls.TurnStart.IsZero() {
		return ls.TurnStart
	}
	return a.st.TurnAt
}

// stageWord 括号里的阶段词。
func (a *App) stageWord() string {
	if ls := a.st.Live; ls != nil {
		switch ls.Kind {
		case core.StageTool:
			return "运行工具"
		case core.StageNone:
			return ""
		}
		return ls.Kind.Label()
	}
	return ""
}

// statusSlow 当前阶段是否已久到该转琥珀。
func (a *App) statusSlow() bool {
	ls := a.st.Live
	if ls == nil {
		return false
	}
	el := a.now().Sub(ls.Since)
	switch ls.Kind {
	case core.StageWait:
		return el >= core.SlowWait
	case core.StageFin:
		return el >= core.SlowFin
	case core.StageConn:
		return el >= core.SlowConn
	}
	return false
}

// elapsedText 已用时间:不到 1 秒不显示,之后「5.8s」,过 60 秒写成「1m 05s」。
func elapsedText(d time.Duration) string {
	switch {
	case d < time.Second:
		return ""
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	s := int(d.Seconds())
	return fmt.Sprintf("%dm %02ds", s/60, s%60)
}

// statusRow 是会话流末尾的状态行。
func (a *App) statusRow(c *ui.Context) {
	fading, ok := a.statusVisible()
	if !ok {
		a.rtOpen = false
		return
	}
	t := a.tk
	a.frameIn(100 * time.Millisecond) // 已用时 100 ms 刷新一次
	row := ui.Row(c).Key("rt-st").Height(26).Gap(6).Margin(0, -8).PaddingX(8).Radius(t.RSm()).AlignItems(ui.Center).
		FontSize(a.FlowCapFont()).Transition(kit.Fade)
	switch {
	case fading: // 回合结束:淡出 + 高度收到 0,同时脚注就位
		kit.CollapseHide(row, a.stEnd, 26, a.frameSoon)
	case !a.turnStartTime().IsZero():
		kit.StaggerRise(row, 0, a.turnStartTime(), a.frameSoon) // 发送瞬间:淡入下落
	}
	if a.st.AwaitingUser() { // 等你确认:静止的「等待你的批准」,不假装在转
		row.TextColor(t.Warn).Children(func() {
			ui.Row(c).Width(16).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).Children(func() { kit.Ic(c, "pause", kit.S12) })
			ui.Text(c, "等待你的批准").SingleLine()
		})
		return
	}
	slow := a.statusSlow()
	col, meta := t.Tx3, t.Tx4
	if slow {
		col, meta = t.Warn, t.Warn
	}
	if row.Hovered() {
		row.Background(t.Fill1)
	}
	row.Cursor(ui.CursorPointer).OnClick(func() { a.rtOpen = !a.rtOpen })
	verb := a.activeTodoVerb()
	if verb == "" {
		verb = spinnerVerb(a.spinner(), a.curSess+a.turnStartTime().Format(time.RFC3339Nano))
	}
	row.TextColor(col).Children(func() {
		ui.Row(c).Width(16).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).Children(func() {
			a.glyph(c, map[bool]ui.Color{true: t.Warn, false: t.Ac}[slow])
		})
		kit.RunText(ui.Text(c, verb).SingleLine().Shrink(0))
		var parts []string
		if e := elapsedText(a.now().Sub(a.turnStartTime())); e != "" && !a.turnStartTime().IsZero() {
			parts = append(parts, e)
		}
		if w := a.stageWord(); w != "" {
			parts = append(parts, w)
		}
		if a.escInterrupts() {
			parts = append(parts, "Esc 中断")
		}
		m := ui.Text(c, "("+strings.Join(parts, " · ")+")").FontFeatures("tnum").TextColor(meta).SingleLine().Shrink(1).MinWidth(0)
		if ls := a.st.Live; ls != nil && kit.MotionOn() { // 阶段词换了:括号 120 ms 淡入,不整行闪
			if p := float32(a.now().Sub(ls.Since)) / float32(120*time.Millisecond); p < 1 {
				m.Opacity(.35 + .65*max(0, p))
				a.frameSoon()
			}
		}
	})
	if ls := a.st.Live; ls != nil && slow && a.prefs.Bool(pSlowHint, true) {
		a.slowHint(c, *ls)
	}
	if a.rtOpen && a.st.Live != nil {
		a.waitSteps(c, *a.st.Live)
	}
}

// rtLoadedFolders 工作区里读取了 CLAUDE.md 的附加文件夹个数(没有 / 没开 = 0)。
func (a *App) rtLoadedFolders() int {
	if a.ws == nil || !a.ws.LoadMD {
		return 0
	}
	return len(a.ws.Extra())
}

// waitSteps 展开后的阶段清单:已结束的阶段打勾带耗时,当前阶段旋转。
func (a *App) waitSteps(c *ui.Context, ls core.LiveStage) {
	t := a.tk
	ui.Column(c).Gap(3).Margin(2, 0, 4, 16).FontSize(a.FlowCapFont()).TextColor(t.Tx3).Children(func() {
		for _, p := range a.st.LiveDone {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Width(16).Justify(ui.Center).Shrink(0).Children(func() { kit.Ic(c, "check", kit.S12).TextColor(t.Ok) })
				ui.Text(c, stageName(p)).Grow(1).SingleLine()
				ui.Text(c, core.FormatResp(p.Ms)).FontFeatures("tnum").TextColor(t.Tx4)
			})
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).TextColor(t.Tx1).Children(func() {
			ui.Row(c).Width(16).Justify(ui.Center).Shrink(0).Children(func() { a.glyph(c, t.Ac) })
			ui.Text(c, stageName(core.TimingPhase{Kind: ls.Kind, Call: ls.Call})).Grow(1).SingleLine()
			ui.Text(c, core.FormatResp(a.now().Sub(ls.Since).Milliseconds())).FontFeatures("tnum").TextColor(t.Tx3)
		})
	})
}

// stageName 阶段行的名字:等待 / 思考 / 生成带「第 N 次调用」,工具带工具名。
func stageName(p core.TimingPhase) string {
	n := p.Kind.Label()
	switch {
	case p.Label != "":
		n += " " + p.Label
	case p.Call > 0 && (p.Kind == core.StageWait):
		n += fmt.Sprintf(" · 第 %d 次调用", p.Call)
	}
	return n
}
