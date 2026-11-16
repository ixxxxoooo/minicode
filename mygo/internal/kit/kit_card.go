package kit

import (
	"github.com/egoist/mygo/ui"
)

// Card 是可点击或纯展示的卡片外壳(设计稿:悬停上浮 1px、阴影 2→3、描边加深)。
// 作为 ButtonBase 使用 Children 填内容;纯展示时可将 OnClick 留空并设 CursorDefault。
//
//	kit.Card(c, kit.CardOpt{On: sel}).OnClick(...).Children(func() { … })
func Card(c *ui.Context, o ...CardOpt) ui.Element {
	var op CardOpt
	if len(o) > 0 {
		op = o[0]
	}
	el := ui.ButtonBase(c).Padding(12, 14).Radius(T.RMd()).Background(T.Raised).Border(1, T.Bd1).
		Cursor(ui.CursorPointer).Justify(ui.Start).AlignItems(ui.Stretch).Transition(Fade)
	switch {
	case op.On:
		el.BorderColor(T.AcLine).Border(1, T.AcLine)
	case el.Hovered():
		el.BorderColor(T.Bd3)
	}
	for _, sh := range T.Shadow1() {
		el.Shadow(sh.X, sh.Y, sh.Blur, sh.Spread, sh.Color)
	}
	if el.Hovered() && MotionOn() {
		el.Translate(0, -1)
		for _, sh := range T.Shadow2() {
			el.Shadow(sh.X, sh.Y+1, sh.Blur+1, sh.Spread, sh.Color)
		}
	}
	if el.Pressed() {
		clickPressScale(el)
	}
	clickFocusRing(el)
	return el
}

// CardOpt 是 Card 的可选项。
type CardOpt struct {
	On bool // 选中:强调色描边
}
