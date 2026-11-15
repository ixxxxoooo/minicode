package kit

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// ConnState 连通性测试四态:空闲 / 测试中 / 成功 / 失败。
type ConnState int

const (
	ConnIdle ConnState = iota
	ConnRun
	ConnOK
	ConnFail
)

// ConnTestRow 统一的「测试连接」行:状态点 + 按钮 + 结果文案。
func ConnTestRow(c *ui.Context, key string, st ConnState, btnLabel string, result string, onTest func()) {
	t := T
	row := ui.Row(c).Key(key).Gap(8).AlignItems(ui.Center).MinHeight(28)
	row.Children(func() {
		var icon, tip string
		var col ui.Color
		switch st {
		case ConnRun:
			icon, tip, col = "refresh", "测试中", t.Warn
		case ConnOK:
			icon, tip, col = "okc", "成功", t.Ok
		case ConnFail:
			icon, tip, col = "errc", "失败", t.Err
		default:
			icon, tip, col = "activity", "空闲", t.Tx4
		}
		_ = tip
		Ic(c, icon, S12).TextColor(col)
		b := Btn(c, btnLabel, BtnOpt{Small: true, Icon: "refresh", Loading: st == ConnRun, Disabled: st == ConnRun})
		b.OnClick(onTest)
		if result != "" && st != ConnIdle && st != ConnRun {
			Txt(c, result, theme.FsCap, map[ConnState]ui.Color{ConnOK: t.Ok, ConnFail: t.Err}[st])
		}
	})
}
