package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// subagentTimeline 子 Agent 内嵌时间线:已答的提问画回答卡片,其余步骤行照常。
func (a *App) subagentTimeline(c *ui.Context, items []core.Item) {
	if len(items) == 0 {
		return
	}
	ui.Column(c).Gap(6).Padding(4, 0, 0, 22).Children(func() {
		for _, it := range items {
			switch v := it.(type) {
			case *core.QuestionRequest:
				if v.Answered {
					a.askEcho(c, v)
				}
			default:
				if isStepItem(it) {
					a.stepItem(c, it)
				}
			}
		}
	})
}

// 折叠工具分组的「最新一步」:分组折叠且运行中时,标题下方 22px 显示最后一个未结束的工具。

func lastOpenTool(g []*core.ToolCall) *core.ToolCall {
	for i := len(g) - 1; i >= 0; i-- {
		if g[i].Status == core.ToolRunning {
			return g[i]
		}
	}
	return nil
}

// tickRow 是「最新一步」行显示的内容:图标(可染色 / 转圈)+ 一句话。
type tickRow struct {
	icon string
	tone ui.Color // tint 为真时给图标染色;spin 时是沙漏颜色
	tint bool
	spin bool
	text string
}

// tickState 记着某一行上一次显示的内容,换步时拿它当「被顶出去」的旧行。
type tickState struct {
	id        string
	at        time.Time // 本次换步的动画起点;零值 = 没在动
	last, out tickRow
}

const tickDur = 200 * time.Millisecond

// tickerRow 画一行「最新一步」(设计稿 .ticker:22px 高、裁切):id 变了就是换步——
// 新行自下滑入并淡入,旧行同时向上顶出并淡出,200 ms 缓出;减弱动态时直接换。
// 探索组、子 Agent 调用卡、并行组共用,key 是这一行的唯一键。
func (a *App) tickerRow(c *ui.Context, key, id string, cur tickRow) {
	t := a.tk
	ts := a.tick[key]
	if ts == nil {
		ts = &tickState{id: id}
		a.tick[key] = ts
	}
	if ts.id != id {
		ts.out, ts.id, ts.at = ts.last, id, time.Now()
	}
	ts.last = cur
	p, moving := float32(1), false
	if !ts.at.IsZero() {
		if q := float32(time.Since(ts.at)) / float32(tickDur); kit.MotionOn() && q < 1 {
			inv := 1 - q
			p, moving = 1-inv*inv*inv, true // ease-out
			a.frameSoon()
		} else {
			ts.at, ts.out = time.Time{}, tickRow{}
		}
	}
	fs := theme.FsCtl * a.ChatFontScale()
	draw := func(k string, r tickRow, dy, op float32) {
		ui.Row(c).Key(k).Absolute().Left(0).Right(0).Top(0).Bottom(0).Gap(6).AlignItems(ui.Center).TextColor(t.Tx3).FontSize(fs).
			Translate(0, dy).Opacity(op).Children(func() {
			switch {
			case r.spin:
				kit.Spin(c, r.tone)
			case r.icon != "":
				ic := kit.Ic(c, r.icon, kit.S12)
				if r.tint {
					ic.TextColor(r.tone)
				}
			}
			kit.RunText(ui.Text(c, r.text).SingleLine().Grow(1).MinWidth(0))
		})
	}
	ui.Box(c).Key("tick-"+key).Height(theme.Px(22)).Margin(0, 0, 2, theme.FlowInset).Clip().Children(func() {
		if moving && ts.out.text != "" {
			draw("out", ts.out, -p*22, 1-p)
		}
		draw("in", cur, (1-p)*22, p)
	})
}

func (a *App) groupLatest(c *ui.Context, grpID string, tc *core.ToolCall) {
	if tc == nil {
		return
	}
	sp := a.presentTool(tc)
	a.tickerRow(c, "grp-"+grpID, tc.ID, tickRow{icon: sp.icon, text: sp.verb + " " + sp.obj})
}
