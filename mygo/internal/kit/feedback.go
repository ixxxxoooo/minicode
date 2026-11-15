package kit

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

const undoToastDur = 6 * time.Second

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
	p := float32(time.Since(*at)) / 320
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
