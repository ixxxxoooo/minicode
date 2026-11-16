package app

import (
	"fmt"
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

// waitLive 返回等待行该不该出现。
func (a *App) waitLive() (core.LiveStage, bool) {
	ls := a.st.Live
	if ls == nil || a.st.Phase != core.PhaseRunning || len(a.st.Items) == 0 {
		return core.LiveStage{}, false
	}
	if as, ok := a.st.Items[len(a.st.Items)-1].(*core.Assistant); ok && as.Streaming { // 已经在出字:让位给正文
		return core.LiveStage{}, false
	}
	switch ls.Kind {
	case core.StageConn, core.StagePrep, core.StageWait:
		return *ls, true
	case core.StageFin:
		return *ls, a.now().Sub(ls.Since) >= core.SlowFin
	}
	return core.LiveStage{}, false
}

func (a *App) waitSlow(ls core.LiveStage) bool {
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

func waitLabel(ls core.LiveStage) string {
	switch ls.Kind {
	case core.StageConn:
		return "正在连接 Claude…"
	case core.StageFin:
		return "收尾中…"
	}
	return "等待模型响应…"
}

// elapsedText 已等时间:不到 0.5 秒不显示,之后保留一位小数,过 100 秒换成「1 分 40 秒」。
func elapsedText(d time.Duration) string {
	if d < 500*time.Millisecond {
		return ""
	}
	if d >= 100*time.Second {
		return core.FormatElapsed(d.Milliseconds())
	}
	return fmt.Sprintf("%.1f s", d.Seconds())
}

// waitRow 是会话流末尾的等待行。
func (a *App) waitRow(c *ui.Context) {
	ls, ok := a.waitLive()
	if !ok {
		return
	}
	t := a.tk
	slow := a.waitSlow(ls)
	tone := t.Tx3
	if slow {
		tone = t.Warn
	}
	el := a.now().Sub(ls.Since)
	a.frameSoon() // 已用时每帧刷新
	row := ui.Row(c).Key("rt-wait").MinHeight(26).Gap(8).Margin(0, -8).PaddingX(8).Radius(t.RSm()).AlignItems(ui.Center).
		TextColor(tone).Cursor(ui.CursorPointer).FontSize(a.FlowCapFont()).Transition(kit.Fade)
	if row.Hovered() {
		row.Background(t.Fill1)
	}
	row.OnClick(func() { a.rtOpen = !a.rtOpen })
	row.Children(func() {
		ui.Row(c).Width(16).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).Children(func() {
			kit.Spin(c, map[bool]ui.Color{true: t.Warn, false: t.Ac}[slow])
		})
		kit.RunText(ui.Text(c, waitLabel(ls)).SingleLine().Shrink(1))
		if s := elapsedText(el); s != "" {
			ui.Text(c, s).FontFeatures("tnum").TextColor(map[bool]ui.Color{true: t.Warn, false: t.Tx4}[slow]).SingleLine().Shrink(0)
		}
		a.waitChips(c, ls)
	})
	if slow && a.prefs.Bool(pSlowHint, true) {
		a.slowHint(c, ls)
	}
	if a.rtOpen {
		a.waitSteps(c, ls)
	}
}

// waitChips 右侧的体量提示:本次请求带的上下文大小。
func (a *App) waitChips(c *ui.Context, ls core.LiveStage) {
	if ls.Kind != core.StageWait {
		return
	}
	if used := a.st.Usage.ContextUsed; used > 0 {
		kit.Tag(c, "上下文 "+fmtTok(used)).Tooltip(a.contextTip())
	}
}

// contextTip 上下文 chip 的悬停:附加文件夹读取了规则文件时点明。
func (a *App) contextTip() string {
	if n := a.rtLoadedFolders(); n > 0 {
		return fmt.Sprintf("每次请求都带着这部分上下文,其中含 %d 个附加文件夹的 CLAUDE.md", n)
	}
	return "每次请求都带着这部分上下文;越大,模型出第一个字越慢"
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
			ui.Row(c).Width(16).Justify(ui.Center).Shrink(0).Children(func() { kit.Spin(c, t.Ac) })
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
