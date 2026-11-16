package kit

import (
	"testing"
	"time"
)

// 回归:30 根柱子、总时长 = DurSlow + 30×24ms 时,动画结束后每一根都必须满高。
// 以前错峰按 DurSlow 换算,序号 ≥ 11 的柱子永远停在 0.001。
func TestGrowScaleEndsFull(t *testing.T) {
	const n = 30
	total := DurSlow + n*24*time.Millisecond
	for i := 0; i < n; i++ {
		delay := time.Duration(i) * 24 * time.Millisecond
		if got := growScale(1, delay, total); got < .999 {
			t.Fatalf("第 %d 根动画结束后缩放 = %v,应为满高", i, got)
		}
	}
}

// 起点全部收起,中途后面的柱子比前面的矮(错峰),单调不减。
func TestGrowScaleStaggerAndMonotonic(t *testing.T) {
	total := time.Second
	if got := growScale(0, 0, total); got > .01 {
		t.Fatalf("起点应收起,得 %v", got)
	}
	early := growScale(.5, 0, total)
	late := growScale(.5, 500*time.Millisecond, total)
	if !(early > late) {
		t.Fatalf("错峰失败:先出的 %v 应高于后出的 %v", early, late)
	}
	prev := float32(0)
	for f := float32(0); f <= 1.0001; f += .05 {
		cur := growScale(f, 300*time.Millisecond, total)
		if cur+1e-4 < prev {
			t.Fatalf("进度 %.2f 时缩放回落 %v < %v", f, cur, prev)
		}
		prev = cur
	}
	if got := growScale(.5, time.Second, 0); got != 1 {
		t.Fatalf("总时长为 0 时应直接满高,得 %v", got)
	}
}
