package kit

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// GrowY 让柱状条自下而上生长(设计稿 grow-y:逐根延迟 delay)。
// fraction 0..1 是整段动画(总时长 total)的进度;motion 关闭时直接满高。
func GrowY(bar ui.Element, fraction float32, delay, total time.Duration) {
	if !MotionOn() {
		return
	}
	// 缩放原点放在底边,才是「自下而上」生长;默认原点在中心,柱子会悬在各自的中线上
	bar.TransformOrigin(.5, 1).Scale(1, growScale(fraction, delay, total))
}

// growScale 算某一根柱子此刻的纵向缩放。错峰要按「整段动画时长」换算:
// 以前除以 DurSlow(260ms),30 根柱子的延迟最大 696ms,序号 ≥ 11 的柱子
// shift ≥ 1,动画结束后仍被卡在 0.001,图永远画不全。
func growScale(fraction float32, delay, total time.Duration) float32 {
	f := max(0, min(1, fraction))
	if total <= 0 {
		return 1
	}
	// 延迟占比最多到 0.9,保证每根柱子至少有一小段时间生长,且 f=1 时一定满高
	shift := min(.9, float32(delay)/float32(total))
	if f <= shift {
		return .001
	}
	p := (f - shift) / (1 - shift)
	ease := 1 - (1-p)*(1-p)
	return max(.001, ease)
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
