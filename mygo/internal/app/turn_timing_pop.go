package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 回复脚注里的「用时」:迷你分段条 + 瀑布弹卡(设计稿 resp-time 页,对标 Chrome DevTools Timing)。

const maxPopRows = 10

// timingChip 脚注里可点的「用时 x」,后面跟一条 56×4 的迷你分段条;点开瀑布弹卡。
func (a *App) timingChip(c *ui.Context, uid string, timing core.TurnTiming, stats core.TurnStats, hasStats bool) {
	t := a.tk
	key := "rt:" + uid
	chip := ui.ButtonBase(c).Key("rtchip-"+uid).Height(theme.Px(20)).PaddingX(6).Margin(0, 0, 0, -6).Gap(6).Radius(t.RSm()).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).FontSize(theme.FsCap).Transition(kit.Fade)
	switch {
	case a.popup == key:
		chip.Background(t.Fill1).TextColor(t.Tx2)
	case chip.Hovered():
		chip.Background(t.Fill1).TextColor(t.Tx2)
	default:
		chip.TextColor(t.Tx4)
	}
	chip.OnClick(func() { a.togglePop(key) })
	chip.Children(func() {
		ui.Text(c, timing.ShortText()).FontFeatures("tnum").SingleLine()
		if a.chatTier() != tierNarrow { // 窄档条隐藏,弹卡照常可点
			a.miniBar(c, timing)
		}
	})
	a.popOpen(key, func(open *bool) {
		ui.PopoverBase(c, chip, open, func(panel ui.Element) {
			panel = a.popPanel(panel, 340).Padding(12, 14).Gap(4)
			panel.AttachTo(chip, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(0, 0, 6, 0)
			panel.Children(func() { a.timingPopBody(c, timing, stats, hasStats) })
		})
	})
}

// miniBar 56×4 的分段条:各阶段按占比上色,相邻同色合并,小于 2% 的段保底 2px。
func (a *App) miniBar(c *ui.Context, tm core.TurnTiming) {
	t := a.tk
	type seg struct {
		col ui.Color
		ms  int64
	}
	var segs []seg
	for _, p := range tm.Phases {
		col := stageColor(t, p.Kind)
		if n := len(segs); n > 0 && segs[n-1].col == col {
			segs[n-1].ms += p.Ms
			continue
		}
		segs = append(segs, seg{col, p.Ms})
	}
	ui.Row(c).Width(56).Height(4).Radius(2).Background(t.Fill2).Clip().Gap(1).Shrink(0).Children(func() {
		for _, s := range segs {
			ui.Box(c).Grow(float32(s.ms)).Height(4).MinWidth(2).Background(s.col)
		}
	})
}

// timingPopBody 弹卡内容:标题 + 瀑布行 + 缓存与上下文 + 一句结论。
func (a *App) timingPopBody(c *ui.Context, tm core.TurnTiming, stats core.TurnStats, hasStats bool) {
	t := a.tk
	total := float32(max(1, tm.TotalMs))
	slowWait, _ := tm.SlowestWait()
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "响应耗时").FontWeight(700).FontSize(theme.Px(13))
		if slowWait >= int64(core.SlowWait/1e6) {
			kit.Tag(c, "首字偏慢", kit.KWarn)
		}
		ui.Box(c).Grow(1)
		head := core.FormatResp(tm.TotalMs)
		if m := tm.ModelMs(); m < tm.TotalMs-1000 { // 含「等你确认」:括号里写模型自己的耗时
			head += "(模型 " + core.FormatResp(m) + ")"
		}
		ui.Text(c, head).FontWeight(600).FontFeatures("tnum")
	})
	ui.Box(c).Height(4)
	phases, rest, restMs := tm.Phases, 0, int64(0)
	if len(phases) > maxPopRows {
		for _, p := range phases[maxPopRows:] {
			restMs += p.Ms
		}
		rest, phases = len(phases)-maxPopRows, phases[:maxPopRows]
	}
	var at int64
	for _, p := range phases {
		start := float32(at) / total
		at += p.Ms
		a.waterfallRow(c, tm, p, start, float32(p.Ms)/total)
	}
	if rest > 0 {
		ui.Text(c, fmt.Sprintf("另 %d 段 · 共 %s", rest, core.FormatResp(restMs))).TextColor(t.Tx4).FontSize(theme.FsCap).Margin(2, 0, 0, 82)
	}
	if hasStats && stats.InputTotal() > 0 {
		ui.Box(c).Height(1).Background(t.Bd1).Margin(7, -14)
		a.popKV(c, "缓存", cacheLine(stats))
		a.popKV(c, "上下文", a.contextLine(stats))
	}
	if tip := a.timingTip(tm); tip != "" {
		ui.Text(c, tip).LineHeight(1.55).FontSize(theme.FsCap).TextColor(t.Tx2).Padding(6, 8).Radius(t.RMd()).Background(t.WarnSoft).Margin(6, 0, 0, 0)
	}
}

func (a *App) waterfallRow(c *ui.Context, tm core.TurnTiming, p core.TimingPhase, start, w float32) {
	t := a.tk
	slow := (p.Kind == core.StageWait && p.Ms >= int64(core.SlowWait/1e6)) || (p.Kind == core.StageFin && p.Ms >= int64(core.SlowFin/1e6))
	ui.Row(c).Gap(8).AlignItems(ui.Center).MinHeight(20).FontSize(theme.FsCap).Children(func() {
		ui.Text(c, stageName(core.TimingPhase{Kind: p.Kind, Label: p.Label})).Width(74).TextColor(t.Tx3).SingleLine().Shrink(0)
		ui.Box(c).Grow(1).Height(6).Radius(3).Background(t.Fill1).Children(func() {
			ui.Box(c).Absolute().Top(0).LeftPercent(start * 100).WidthPercent(max(w*100, 1.5)).Height(6).Radius(3).Background(stageColor(t, p.Kind))
		})
		ms := ui.Text(c, core.FormatResp(p.Ms)).FontFeatures("tnum").Width(52).TextAlign(ui.End).SingleLine().Shrink(0)
		if slow {
			ms.TextColor(t.Warn).FontWeight(600)
		} else {
			ms.TextColor(t.Tx1)
		}
	})
	if p.Kind == core.StageWait && p.Call > 0 && p.Call <= len(tm.Calls) {
		sub := fmt.Sprintf("第 %d 次调用", p.Call)
		if ct := tm.Calls[p.Call-1].Text(); ct != "" {
			sub += " · " + ct
		}
		subc := t.Tx4
		if tm.Calls[p.Call-1].Slow() {
			subc = t.Warn
		}
		ui.Text(c, sub).FontSize(theme.Px(10.5)).TextColor(subc).Margin(0, 0, 2, 82).LineHeight(1.4)
	}
}

func (a *App) popKV(c *ui.Context, k, v string) {
	t := a.tk
	ui.Row(c).Gap(6).AlignItems(ui.Start).MinHeight(20).FontSize(theme.FsCap).Children(func() {
		ui.Text(c, k).Width(74).TextColor(t.Tx3).Shrink(0)
		ui.Text(c, v).Grow(1).MinWidth(0).LineHeight(1.5)
	})
}

func cacheLine(s core.TurnStats) string {
	ct := core.CallInfo{Cache: core.CacheTokens{Read: s.CacheRead, Written: s.CacheWrite, Uncached: s.Input}}
	if !ct.HasCache() || (s.CacheRead == 0 && s.CacheWrite == 0) {
		return "没有命中(服务商未返回缓存用量,或本次未使用缓存)"
	}
	k := core.FormatTokens
	return fmt.Sprintf("命中 %d%% · 读 %s · 写 %s · 未缓存 %s", ct.Cache.HitPercent(), k(s.CacheRead), k(s.CacheWrite), k(s.Input))
}

func (a *App) contextLine(s core.TurnStats) string {
	line := core.FormatTokens(s.InputTotal()) + " tokens"
	if n := a.rtLoadedFolders(); n > 0 {
		line += fmt.Sprintf(" · 含 %d 个附加文件夹的 CLAUDE.md", n)
	} else if a.ws != nil {
		line += fmt.Sprintf(" · 工作区 %d 个文件夹", len(a.ws.Folders))
	}
	return line
}

// timingTip 一句结论:最慢的一段是什么、有没有命中缓存、慢在哪;没什么可说的返回空串。
func (a *App) timingTip(tm core.TurnTiming) string {
	var sb []string
	if ms, call := tm.SlowestWait(); ms >= int64(core.SlowWait/1e6) {
		s := fmt.Sprintf("第 %d 次等待模型用了 %s", call, core.FormatResp(ms))
		if call >= 1 && call <= len(tm.Calls) && tm.Calls[call-1].Slow() {
			s += ",且没有命中缓存"
		}
		sb = append(sb, s+"。")
	}
	for _, p := range tm.Phases {
		if p.Kind == core.StageFin && p.Ms >= int64(core.SlowFin/1e6) {
			sb = append(sb, fmt.Sprintf("最后一段回复后 %s 才结束,常见于上游迟迟不关流。", core.FormatResp(p.Ms)))
			break
		}
	}
	if len(sb) > 0 {
		if n := a.rtLoadedFolders(); n > 0 {
			sb = append(sb, fmt.Sprintf("工作区读取了 %d 个附加文件夹的 CLAUDE.md。", n))
		}
	}
	return strings.Join(sb, "")
}
