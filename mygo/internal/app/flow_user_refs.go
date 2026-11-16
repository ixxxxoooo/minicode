package app

import (
	"github.com/egoist/mygo/ui"
)

// userRefChip 用户消息里的 @ 文件引用胶囊(设计稿 flow.css .ref)。
func (a *App) userRefChip(c *ui.Context, path string) {
	t := a.tk
	name := baseName(path)
	ui.Row(c).Height(19).PaddingX(6).Gap(4).Radius(t.RXs()).Background(t.InfoSoft).AlignItems(ui.Center).Shrink(0).Children(func() {
		a.fileIc(c, name)
		ui.Text(c, name).FontSize(a.FlowCapFont()).TextColor(t.Info).SingleLine()
	})
}
