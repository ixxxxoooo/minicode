package kit

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// FieldOpt 是统一表单字段外壳:标签、说明、错误文案与抖动。
type FieldOpt struct {
	Label, Desc, Err string
	ShakeAt          *time.Time
}

// Field 包一行输入:标签 / 说明、错误描边与抖动、错误行文案。
func Field(c *ui.Context, key string, o FieldOpt, build func(row ui.Element)) {
	t := T
	ui.Column(c).Gap(4).Shrink(0).Children(func() {
		if o.Label != "" {
			ui.Text(c, o.Label).FontSize(theme.Px(11.5)).FontWeight(500).TextColor(t.Tx2)
		}
		if o.Desc != "" {
			ui.Text(c, o.Desc).FontSize(theme.Px(10.5)).TextColor(t.Tx3).LineHeight(theme.LhCap)
		}
		box := ui.Row(c).Key(key+"-fld").MinWidth(0).Height(theme.Px(26)).PaddingX(8).Gap(6).AlignItems(ui.Center).Background(t.Raised).Radius(t.RSm()).Border(1, t.Bd2)
		bad := o.Err != ""
		if box.FocusWithin() && !bad {
			box.BorderColor(t.AcLine)
			box.FocusRing(true).FocusRingOffset(0).FocusRingColor(t.Ring())
		}
		if bad {
			box.BorderColor(t.Err)
			ShakeInput(box, o.ShakeAt)
		}
		box.Grow(1).MinWidth(0).Children(func() { build(box) })
		if bad {
			ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
				Ic(c, "alert", S12).TextColor(t.Err)
				ui.Text(c, o.Err).FontSize(theme.Px(10.5)).TextColor(t.Err).Grow(1)
			})
		}
	})
}
