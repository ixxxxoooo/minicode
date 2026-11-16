package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// subView 子 Agent 虚拟标签:与主会话流同一套步骤行 / 助手正文 / 权限与问答样式。
func (a *App) subView(c *ui.Context, id string) {
	t := a.tk
	s := a.findSub(id)
	if s == nil {
		ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "这个子 Agent 已不在当前会话里").TextColor(t.Tx3)
		})
		return
	}
	run := s.Active()
	a.wsHead(c, "bot", t.ToolAgent, s.Description, func() {
		a.subStatusTag(c, s)
	}, func() {
		el, _ := s.ElapsedMs(a.st.Now())
		meta := core.FormatElapsed(el) + " · " + itoa(s.Tools()) + " 个工具"
		if s.Tokens > 0 {
			meta += " · " + fmtTok(s.Tokens) + " token"
		}
		ui.Text(c, meta).TextColor(t.Tx3).FontSize(theme.FsCap).FontFeatures("tnum").SingleLine()
		kit.Btn(c, "引用到对话", kit.BtnOpt{Small: true, Icon: "quote"}).OnClick(func() {
			a.draft = strings.TrimSpace(a.draft + "\n> 子 Agent「" + s.Description + "」:" + s.Summary)
		})
		kit.Btn(c, "复制全部", kit.BtnOpt{Variant: kit.BtnGhost, Small: true, Icon: "copy"}).OnClick(func() {
			c.WriteClipboard(a.subExportText(s))
		})
		if run {
			kit.Ib(c, "stop", kit.IbOpt{Small: true, Tip: "停止"}).OnClick(func() {
				if s.TaskID != "" {
					a.send(core.CmdStopTask{TaskID: s.TaskID})
					log.Info("停止子 Agent", "task", s.TaskID, "card", s.ID)
				}
			})
		}
	})
	st := &a.subSt
	if run && a.subFollowBottom() {
		st.Y = 1e9
	}
	ui.Box(c).Grow(1).MinHeight(0).Children(func() {
		kit.VScroll(c).Key("subview-" + id).TrackScroll(st).Grow(1).Children(func() {
			// 内容列(flowCol)自带最大宽度与左右内边距;外层居中,与主会话流同样的版心
			ui.Column(c).Padding(10, 0, 16, 0).AlignItems(ui.Center).Children(func() {
				a.subFlowItems(c, s)
			})
		})
		a.subScrollBtn(c, s.Children)
	})
}
