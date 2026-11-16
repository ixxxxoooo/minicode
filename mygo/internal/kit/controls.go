package kit

import (
	"github.com/ixxxxoooo/minicode/internal/theme"
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// TextW 粗估一段文字的显示宽度(CJK 算全角)。只用于分段控件这类要「按最宽项等分」的地方。
func TextW(s string, size float32) float32 {
	var w float32
	for _, r := range s {
		switch {
		case r > 0x2E80:
			w += size
		case unicode.IsUpper(r) || r == 'm' || r == 'w' || r == 'M' || r == 'W':
			w += size * .64
		case r == ' ':
			w += size * .3
		case r == 'i' || r == 'l' || r == 'j' || r == '.' || r == ',' || r == ':' || r == '|' || r == '1' || r == 'I' || r == 't' || r == 'f':
			w += size * .3
		default:
			w += size * .54
		}
	}
	return w
}

// SegItem 是分段控件的一段:图标、文字或两者都有。
type SegItem struct {
	Icon, Text, Tip string
}

// SegOpt 是分段控件的可选项。
type SegOpt struct {
	Large bool    // .lg:按钮高 22、横向内边距 12
	Icons bool    // .icons:仅图标,内边距 7
	W     float32 // 指定每段宽度(0 = 按最宽项估算)
	Fill  bool    // 撑满父级宽度,各段等分(navseg)
	Key   string  // 同一父级下有多个分段时区分状态
}

var thumbSpring = ui.Animation{Duration: DurBase, Ease: EaseSpring}

// Seg 是分段控件(设计稿 .seg):底色 fill-2,选中的滑块是 raised + 阴影,换段时滑块带轻微回弹滑过去。
// 返回被点击的下标(没有点击返回 -1)。
func Seg(c *ui.Context, items []SegItem, on int, o ...SegOpt) int {
	var op SegOpt
	if len(o) > 0 {
		op = o[0]
	}
	h, px := float32(20), float32(9)
	if op.Large {
		h, px = 22, 12
	}
	if op.Icons {
		px = 7
	}
	w := op.W
	if w == 0 && !op.Fill {
		for _, it := range items {
			cw := float32(0)
			if it.Icon != "" {
				cw += 12
			}
			if it.Text != "" {
				if it.Icon != "" {
					cw += 4
				}
				cw += TextW(it.Text, 11.5) + 1
			}
			w = max(w, cw+px*2)
		}
	}
	n := float32(len(items))
	clicked := -1
	root := ui.Row(c).Radius(T.RMd()).Background(T.Fill2).Padding(2).Shrink(0)
	if op.Fill {
		root.Grow(1)
	}
	if op.Key != "" {
		root.Key(op.Key)
	}
	root.Children(func() {
		// 内层容器恰好是「去掉 2px 内边距后的区域」,滑块用百分比定位才对得准
		inner := ui.Row(c)
		if op.Fill {
			inner.Grow(1)
		}
		inner.Children(func() {
			th := ui.Box(c).Absolute().ZIndex(0).Top(0).Bottom(0).WidthPercent(100 / n).Radius(T.RMd() - 2).Background(T.Raised)
			for _, s := range T.Shadow1() {
				th.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
			}
			th.LeftPercent(th.AnimateTo("x", float32(on)*100/n, thumbSpring))
			for i, it := range items {
				col := T.Tx2
				if i == on {
					col = T.Tx1
				}
				b := ui.ButtonBase(c).ZIndex(1).Height(h).Justify(ui.Center).AlignItems(ui.Center).Gap(4).Cursor(ui.CursorPointer).
					FontSize(theme.Px(11.5)).FontWeight(500).TextColor(col).Transition(Fade)
				if op.Fill {
					b.Grow(1).Basis(0)
				} else {
					b.Width(w).Shrink(0)
				}
				if it.Tip != "" {
					b.Tooltip(it.Tip)
				}
				if it.Text == "" {
					b.Label(it.Icon)
				}
				clickFocusRing(b)
				b.Children(func() {
					if it.Icon != "" {
						Ic(c, it.Icon, S12)
					}
					if it.Text != "" {
						ui.Text(c, it.Text).SingleLine()
					}
				})
				if b.Clicked() {
					clicked = i
				}
			}
		})
	})
	return clicked
}

// TogOpt 是开关的可选项。
type TogOpt struct {
	Disabled bool // 半透明、保持当前开/关、不响应点击
}

// Tog 是开关(设计稿 .tog:26×15,圆点滑动带回弹)。返回是否被点击。
func Tog(c *ui.Context, on bool, key string, o ...TogOpt) bool {
	var op TogOpt
	if len(o) > 0 {
		op = o[0]
	}
	bg := T.Fill3
	if on {
		bg = T.Ac
	}
	b := ui.ButtonBase(c).Key(key).Size(26*theme.UIScale(), 15*theme.UIScale()).Radius(8 * theme.UIScale()).Background(bg).Shrink(0).Transition(ui.ElementTransition{Colors: true, Duration: DurBase, Ease: EaseIO})
	if op.Disabled {
		b.Disabled(true).Opacity(.42).Cursor(ui.CursorDefault)
	} else {
		b.Cursor(ui.CursorPointer)
	}
	b.Label("开关")
	x := 2 * theme.UIScale()
	if on {
		x = 13 * theme.UIScale()
	}
	b.Children(func() {
		k := ui.Box(c).Absolute().Top(2*theme.UIScale()).Size(11*theme.UIScale(), 11*theme.UIScale()).Radius(6*theme.UIScale()).Background(ui.Hex("#FFFFFF")).Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, .25))
		k.Left(k.AnimateTo("x", x, thumbSpring))
	})
	clickFocusRing(b)
	if op.Disabled {
		return false
	}
	return b.Clicked()
}

// ───────── 列表行 .row ─────────

// RowEl 是 .row 的外壳:行高取方案(随界面大小缩放),悬停 fill-1;
// 选中态所有布局方案统一:强调色淡染底(Sel)+ 正文色 + 半粗体,再配合 SelBar 的左侧竖线,
// 此前方案 A 只有 8% 的灰底、方案 B 只有白底,在宣纸 / 墨夜主题里几乎看不出选中了谁。
// 调用方负责 Children;需要悬停状态时用返回值的 Hovered()。
func RowEl(c *ui.Context, on bool) ui.Element {
	r := ui.ButtonBase(c).Height(T.Scheme.RowH).PaddingX(8).Gap(6).Radius(T.RSm()).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	col := T.Tx2
	switch {
	case on:
		r.Background(T.Sel).FontWeight(500)
		col = T.Tx1
	case r.Hovered():
		r.Background(T.Fill1)
		col = T.Tx1
	}
	return r.TextColor(col).Transition(Fade)
}

// SelBar 画选中行左侧的强调色竖线(放在行的 Children 里,绝对定位,不占位置)。
func SelBar(c *ui.Context) {
	ui.Box(c).Absolute().Left(0).Top(4).Bottom(4).Width(3).Radius(2).Background(T.Ac)
}

// Grp 是分组小标题(.grp-h:24 高,10.5px 粗体,字距 .04em,tx-3)。
func Grp(c *ui.Context, s string) ui.Element {
	return ui.Row(c).Height(theme.Px(24)).PaddingX(8).AlignItems(ui.Center).Gap(5).Shrink(0).Children(func() {
		ui.Text(c, s).FontSize(theme.Px(10.5)).FontWeight(500).LetterSpacing(.4).TextColor(T.Tx3).SingleLine()
	})
}

// Cap 是弹层里的小节标题(.cap)。
func Cap(c *ui.Context, s string) ui.Element {
	return ui.Row(c).PaddingX(8).Padding(6, 8, 3, 8).Children(func() {
		ui.Text(c, s).FontSize(theme.Px(10.5)).FontWeight(500).LetterSpacing(.3).TextColor(T.Tx3)
	})
}

// ───────── 标签条 .tabs ─────────

// TabItem 是标签条的一项。
type TabItem struct {
	ID, Text string
	Icon     string
	Count    int
}

// Tabs 画标签条(设计稿 .tabs:30 高,选中项下方 2px 强调线从中间展开)。返回被点击的 ID(空串 = 无)。
// tail 在条的右侧画额外内容。
func Tabs(c *ui.Context, items []TabItem, on string, h float32, tail func()) string {
	clicked := ""
	ui.Row(c).Height(h).PaddingX(6).Gap(2).AlignItems(ui.Stretch).Shrink(0).
		BorderWidth(0, 0, 1, 0).BorderColor(T.Bd1).Children(func() {
		for _, it := range items {
			sel := it.ID == on
			col := T.Tx3
			if sel {
				col = T.Tx1
			}
			b := ui.ButtonBase(c).Key(it.ID).PaddingX(7).Gap(5).AlignItems(ui.Center).Cursor(ui.CursorPointer).TextColor(col).Transition(Fade)
			if !sel && b.Hovered() {
				b.TextColor(T.Tx1)
			}
			b.Children(func() {
				if it.Icon != "" {
					Ic(c, it.Icon, S12)
				}
				t := ui.Text(c, it.Text).SingleLine()
				if sel {
					t.FontWeight(500)
				}
				if it.Count > 0 {
					Cnt(c, itoa(it.Count), false)
				}
				// 下划线:缩放从中间展开
				ln := ui.Box(c).Absolute().Left(7).Right(7).Bottom(0).Height(2).Radius(2).Background(T.Ac)
				s := float32(0)
				if sel {
					s = 1
				}
				ln.Scale(ln.AnimateWith("s", s, DurBase, EaseOut), 1)
			})
			if b.Clicked() {
				clicked = it.ID
			}
		}
		ui.Spacer(c).Grow(1)
		if tail != nil {
			tail()
		}
	})
	return clicked
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

var _ = time.Second
