package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// userMsgHoverBar 悬停动作条:浮在卡片右上角(不占位,悬停出现时不会把卡片撑高),
// 用卡片底色盖住下面的文字。
func (a *App) userMsgHoverBar(c *ui.Context, u *core.User, bg ui.Color) {
	t := a.tk
	ui.Row(c).Absolute().Top(4).Right(6).ZIndex(3).Padding(1, 3).Background(bg).Radius(t.RSm()).Gap(2).AlignItems(ui.Center).TextColor(t.Tx4).FontSize(theme.FsCap).Children(func() {
		rwTip := "回退到这条消息之前的文件改动"
		canRw := a.st.Phase != core.PhaseRunning
		if _, ok := a.st.UserUUIDs[u.ID]; !ok {
			canRw = false
			rwTip = "这条消息没有检查点记录"
		}
		rw := kit.Ib(c, "undo", kit.IbOpt{Small: true, Tip: rwTip})
		if !canRw {
			rw.Opacity(0.35).Cursor(ui.CursorDefault)
		} else if rw.Clicked() {
			a.userRewindBegin(u.ID)
		}
		retry := kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "重试:以此消息重新发送"})
		if a.st.Phase != core.PhaseRunning {
			retry.OnClick(func() {
				if strings.TrimSpace(u.Text) != "" {
					a.send(core.CmdPrompt{Text: u.Text})
					log.Info("重试用户消息", "user", u.ID)
				}
			})
		} else {
			retry.Opacity(0.35).Cursor(ui.CursorDefault)
		}
		if kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制消息正文"}).Clicked() {
			a.copyOut(userDisplayText(u.Text), "消息")
		}
	})
}
