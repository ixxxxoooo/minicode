package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// connChip 状态栏上的连接状态:连接中 / 已连接 / 未连接 / 连接失败。只有真实 claude 后端才显示;
// 点「未连接 / 连接失败」立刻连接。
func (a *App) connChip(c *ui.Context) {
	if !a.warmEligible() || a.st == nil {
		return
	}
	t := a.tk
	st := a.st.Conn
	tip := map[core.ConnState]string{
		core.ConnNone:     "还没有连接 Claude:发送第一条消息时才会启动(点这里立刻连接)。空闲释放后也是这个状态,发送会自动接着聊",
		core.ConnStarting: "正在启动 Claude 并握手,完成后发送即响应;现在也可以先打字,消息会在连接好后立刻发出",
		core.ConnReady:    "Claude 已连接,发送即响应。闲置超过设定时间会自动释放,再发送时自动接着聊(设置 → 通用 → 连接)",
		core.ConnFailed:   "上次连接 Claude 失败,发送时会再试一次;点这里现在重试。详情见 输出 面板 / 日志",
	}[st]
	chip := a.sbItem(c, "conn", func() {
		switch st {
		case core.ConnStarting:
			kit.Spin(c, t.Ac)
		case core.ConnReady:
			kit.Ic(c, "okc", kit.S12).TextColor(t.Ok)
		case core.ConnFailed:
			kit.Ic(c, "errc", kit.S12).TextColor(t.Err)
		default:
			kit.Ic(c, "circle", kit.S12).TextColor(t.Tx4)
		}
		ui.Text(c, st.Label())
	})
	chip.Tooltip(tip)
	if st == core.ConnNone || st == core.ConnFailed {
		chip.OnClick(func() {
			a.warm.sent = false
			a.noteActive()
			a.warmFire("手动点击状态栏")
		})
	} else {
		chip.Cursor(ui.CursorDefault)
	}
}
