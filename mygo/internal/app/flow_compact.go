package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 压缩上下文卡(设计稿 flow-compact 页):会话流里独立的一张卡,
// 压缩中(不确定进度条 + 已用时)→ 已完成(压缩前 / 后对比条)/ 失败(原因 + 重试)。

const compactBarAnim = 260 * time.Millisecond // 对比条从满宽收缩到占比

func (a *App) compactCard(c *ui.Context, cp *core.Compaction) {
	t := a.tk
	box := ui.Row(c).Key("cp-"+cp.ID).Gap(10).AlignItems(ui.Start).Padding(10, 12).Radius(t.RMd()).FontSize(a.FlowCapFont()).TextColor(t.Tx2)
	if cp.State == core.CompactFailed {
		box.Background(t.ErrSoft)
	} else {
		box.Background(t.Raised)
		for _, s := range t.Shadow1() {
			box.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
	}
	trig := "手动"
	if cp.Auto {
		trig = "自动"
	}
	box.Children(func() {
		ui.Row(c).Width(16).Height(16).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).Margin(1, 0, 0, 0).Children(func() {
			switch cp.State {
			case core.CompactRunning:
				a.glyph(c, t.Ac)
			case core.CompactFailed:
				kit.Ic(c, "errc", kit.S14).TextColor(t.Err)
			default:
				kit.Ic(c, "check", kit.S14).TextColor(t.Ok)
			}
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(7).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				switch cp.State {
				case core.CompactRunning:
					kit.RunText(ui.Text(c, "正在压缩上下文…").FontWeight(600).TextColor(t.Tx1).SingleLine().Shrink(0))
				case core.CompactFailed:
					ui.Text(c, "压缩失败").FontWeight(600).TextColor(t.Tx1).SingleLine().Shrink(0)
				default:
					ui.Text(c, "已压缩上下文").FontWeight(600).TextColor(t.Tx1).SingleLine().Shrink(0)
				}
				kit.Tag(c, trig)
				switch {
				case cp.State == core.CompactRunning && !cp.StartedAt.IsZero():
					a.frameIn(100 * time.Millisecond)
					if e := elapsedText(a.now().Sub(cp.StartedAt)); e != "" {
						ui.Text(c, e).FontFeatures("tnum").TextColor(t.Tx4).SingleLine().Shrink(0)
					}
				case cp.State == core.CompactDone && cp.Ms > 0:
					ui.Text(c, "用时 "+core.FormatResp(cp.Ms)).FontFeatures("tnum").TextColor(t.Tx4).SingleLine().Shrink(1)
				}
				ui.Spacer(c).Grow(1)
				if cp.State == core.CompactDone && cp.Pre > 0 && cp.Post > 0 && cp.Post < cp.Pre {
					save := int(float64(cp.Pre-cp.Post)/float64(cp.Pre)*100 + .5)
					ui.Text(c, fmt.Sprintf("节省 %d%%", save)).FontWeight(600).TextColor(t.Ok).FontFeatures("tnum").SingleLine().Shrink(0)
				}
			})
			switch cp.State {
			case core.CompactRunning:
				kit.IndetMeter(c)
				pre := a.st.Usage.ContextUsed
				txt := "压缩会让 Claude 把前面的对话整理成摘要,释放上下文空间"
				if pre > 0 {
					txt = "压缩前约 " + fmtTok(pre) + " tokens · " + txt
				}
				ui.Text(c, txt).TextColor(t.Tx3).LineHeight(1.55)
			case core.CompactFailed:
				ui.Text(c, cp.Err).TextColor(t.Tx2).LineHeight(1.55)
				ui.Row(c).Gap(6).Children(func() {
					if a.st.Phase == core.PhaseRunning {
						kit.Btn(c, "重试", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).Opacity(.4).Cursor(ui.CursorDefault)
					} else if kit.Btn(c, "重试", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).Clicked() {
						a.send(core.CmdCompact{})
					}
					if kit.Btn(c, "复制错误", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).Clicked() {
						a.copyOut(cp.Err, "错误信息")
					}
				})
			default:
				a.compactBars(c, cp)
			}
		})
	})
}

// compactBars 压缩前后的对比条:前恒为满宽灰条,后按占比画强调色;收缩动画 260 ms。
func (a *App) compactBars(c *ui.Context, cp *core.Compaction) {
	t := a.tk
	if cp.Pre == 0 || cp.Post == 0 { // 回放等拿不到前后数字时,只靠标题说话
		return
	}
	frac := float32(cp.Post) / float32(cp.Pre)
	shown := frac
	if kit.MotionOn() && !cp.DoneAt.IsZero() {
		if p := float32(time.Since(cp.DoneAt)) / float32(compactBarAnim); p < 1 {
			e := 1 - (1-p)*(1-p)
			shown = 1 - (1-frac)*e
			a.frameSoon()
		}
	}
	row := func(label, val string, w float32, col ui.Color) {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).Width(44).TextColor(t.Tx3).SingleLine().Shrink(0)
			ui.Box(c).Grow(1).Height(6).Radius(3).Background(t.Fill1).Clip().Children(func() {
				ui.Box(c).WidthPercent(max(w*100, 1.5)).Height(6).Radius(3).Background(col)
			})
			ui.Text(c, val).Width(44).TextAlign(ui.End).FontFeatures("tnum").TextColor(t.Tx1).SingleLine().Shrink(0)
		})
	}
	row("压缩前", fmtTok(cp.Pre), 1, t.Fill3)
	row("压缩后", fmtTok(cp.Post), shown, t.Ac)
}
