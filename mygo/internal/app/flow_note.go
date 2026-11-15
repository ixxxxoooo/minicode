package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// noteLine 是会话流里的灰字提示行(与步骤行同栅格,无底色;设计稿 F.note)。
func (a *App) noteLine(c *ui.Context, icon, text string) {
	t := a.tk
	ui.Row(c).MinHeight(22).Gap(6).Margin(0, -8).PaddingX(8).TextColor(t.Tx3).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Width(16).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).Children(func() {
			kit.Ic(c, icon, kit.S12)
		})
		ui.Text(c, text).FontSize(a.FlowCapFont()).Grow(1).LineHeight(1.45)
	})
}

func (a *App) taskNoteLine(c *ui.Context, n *core.TaskNote) {
	icon := "check"
	switch n.Status {
	case core.TaskStopped:
		icon = "alert"
	case core.TaskFailed:
		icon = "errc"
	}
	a.noteLine(c, icon, n.Text)
}

func (a *App) permissionDenyNote(c *ui.Context, tool *core.ToolCall) {
	if tool == nil {
		return
	}
	obj := a.presentTool(tool).obj
	if obj == "" {
		obj = tool.Name
	}
	a.noteLine(c, "shield", "你拒绝了"+obj+",Claude 会换一种方式继续。")
}
