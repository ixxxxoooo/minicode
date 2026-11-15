package kit

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// GrowY 让柱状条自下而上生长(设计稿 grow-y:逐根延迟 delay)。
// fraction 0..1 表示当前动画进度;motion 关闭时直接满高。
func GrowY(bar ui.Element, fraction float32, delay time.Duration) {
	if !MotionOn() {
		return
	}
	f := max(0, min(1, fraction))
	// 错峰:delay 内保持 0
	shift := float32(delay) / float32(DurSlow)
	if f <= shift {
		bar.Scale(1, 0.001)
		return
	}
	p := (f - shift) / max(.001, 1-shift)
	ease := 1 - (1-p)*(1-p)
	bar.Scale(1, max(.001, ease))
}

// Crossfade 骨架与真实内容交叉淡入(设计稿 demo-skel / demo-real)。
// skel/real 建议绝对叠放;at 为开始时刻,total 为一轮时长。
func Crossfade(skel, real ui.Element, at time.Time, total time.Duration, frame func()) {
	if !MotionOn() || at.IsZero() {
		skel.Opacity(0)
		real.Opacity(1)
		return
	}
	p := float32(time.Since(at)%total) / float32(total)
	if frame != nil && p < 1 {
		frame()
	}
	// 前半骨架 1→0,后半内容 0→1(中段重叠)
	const overlap = .35
	skelOp := float32(1)
	realOp := float32(0)
	switch {
	case p < .5-overlap/2:
		skelOp = 1
	case p < .5+overlap/2:
		t := (p - (.5 - overlap/2)) / overlap
		skelOp = 1 - t
		realOp = t
	default:
		skelOp = 0
		realOp = 1
	}
	skel.Opacity(skelOp)
	real.Opacity(realOp)
}
