package kit

import (
	"github.com/ixxxxoooo/minicode/internal/theme"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// UndoDuration 是删除撤销窗口时长。
const UndoDuration = 6 * time.Second

const undoToastDur = UndoDuration

// UndoSlot 保存一条可撤销的删除提示(6 秒内点「撤销」)。
type UndoSlot struct {
	Label string
	Until time.Time
	Undo  func()
}

// UndoActive 报告撤销是否仍有效。
func (u *UndoSlot) Active() bool {
	return u != nil && u.Undo != nil && time.Now().Before(u.Until)
}

// BeginUndo 开始一条撤销窗口。
func (u *UndoSlot) Begin(label string, undo func()) {
	if u == nil {
		return
	}
	u.Label, u.Until, u.Undo = label, time.Now().Add(undoToastDur), undo
}

// ClearUndo 清掉撤销状态。
func (u *UndoSlot) Clear() {
	if u == nil {
		return
	}
	u.Label, u.Undo = "", nil
}

// ShakeDuration 是错误抖动的时长。
const ShakeDuration = 320 * time.Millisecond

// ShakeInput 给输入控件加错误抖动(约 320 ms,依赖动效开关)。
func ShakeInput(el ui.Element, at *time.Time) {
	if !MotionOn() {
		return
	}
	if at == nil {
		return
	}
	if at.IsZero() {
		*at = time.Now()
	}
	p := float32(time.Since(*at)) / float32(ShakeDuration)
	if p >= 1 {
		*at = time.Time{}
		return
	}
	// 衰减正弦,最大 ±4 px
	amp := float32(4) * (1 - p)
	el.Margin(amp*float32(math.Sin(float64(p*8*math.Pi))), 0, 0, 0)
}

// FieldErr 是带错误态的单行输入:失焦校验失败时描边 + 抖动。
func FieldErr(el ui.Element, bad bool, shakeAt *time.Time, errCol ui.Color) {
	if bad {
		el.Border(1.5, errCol)
		ShakeInput(el, shakeAt)
	}
}

// UndoBar 绘制「已删除 · 撤销 · 倒计时」条。
func UndoBar(c *ui.Context, u *UndoSlot) {
	if u == nil || !u.Active() {
		return
	}
	t := T
	ui.Row(c).Gap(10).Padding(8, 10).Radius(t.RMd()).Background(t.Fill1).Border(1, t.Bd2).Children(func() {
		ui.Text(c, u.Label).Grow(1).FontSize(theme.Px(12))
		Btn(c, "撤销", BtnOpt{Small: true}).OnClick(func() {
			fn := u.Undo
			u.Clear()
			if fn != nil {
				fn()
			}
		})
		rem := time.Until(u.Until).Seconds()
		Txt(c, itoa(int(rem+.5))+"s", 10, t.Tx4)
	})
}
