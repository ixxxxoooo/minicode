package diag

import (
	"math"
	"sync"
	"time"
)

// 最近帧耗时环形缓冲,供性能页估算帧率与卡顿占比(解析 stderr 帧日志写入)。
const frameRingN = 600

var (
	frameMu    sync.Mutex
	frameBuf   = make([]time.Duration, frameRingN)
	frameHead  int
	frameCount int64
	// 卡顿分档计数(>16.7 / >33 / >100 ms)
	frameJank167 int64
	frameJank33  int64
	frameJank100 int64
)

const (
	ms167 = 167 * time.Microsecond * 100 // 16.7ms
	ms33  = 33 * time.Millisecond
	ms100 = 100 * time.Millisecond
)

// RecordFrame 记入一帧总耗时(ui.frame 与 ParseFrameLine 调用)。
func RecordFrame(d time.Duration) {
	frameMu.Lock()
	frameBuf[frameHead] = d
	frameHead = (frameHead + 1) % frameRingN
	frameCount++
	if d > ms167 {
		frameJank167++
	}
	if d > ms33 {
		frameJank33++
	}
	if d > ms100 {
		frameJank100++
	}
	frameMu.Unlock()
}

// FrameMetrics 是帧与卡顿的只读快照。
type FrameMetrics struct {
	Samples    int64
	FPS        float64
	P50        time.Duration
	P95        time.Duration
	P99        time.Duration
	Jank167Pct float64 // >16.7ms
	Jank33Pct  float64
	Jank100Pct float64
}

// FrameMetricsSnapshot 返回当前帧指标;无样本时各值为 0。
func FrameMetricsSnapshot() FrameMetrics {
	frameMu.Lock()
	defer frameMu.Unlock()
	var out FrameMetrics
	if frameCount == 0 {
		return out
	}
	n := int(frameCount)
	if n > frameRingN {
		n = frameRingN
	}
	ds := make([]float64, n)
	for i := 0; i < n; i++ {
		idx := frameHead - 1 - i
		if idx < 0 {
			idx += frameRingN
		}
		ds[i] = float64(frameBuf[idx])
	}
	out.Samples = frameCount
	// 最近窗口内 FPS: 以 60 帧或全部样本时间跨度估算
	span := ds[0]
	for i := 1; i < n; i++ {
		span += ds[i]
	}
	if span > 0 {
		out.FPS = float64(n) / span
	}
	out.P50 = durPercentile(ds, 0.50)
	out.P95 = durPercentile(ds, 0.95)
	out.P99 = durPercentile(ds, 0.99)
	if frameCount > 0 {
		c := float64(frameCount)
		out.Jank167Pct = 100 * float64(frameJank167) / c
		out.Jank33Pct = 100 * float64(frameJank33) / c
		out.Jank100Pct = 100 * float64(frameJank100) / c
	}
	return out
}

func durPercentile(ds []float64, q float64) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	cp := append([]float64(nil), ds...)
	for i := 0; i < len(cp); i++ {
		for j := i + 1; j < len(cp); j++ {
			if cp[j] < cp[i] {
				cp[i], cp[j] = cp[j], cp[i]
			}
		}
	}
	idx := int(math.Ceil(q*float64(len(cp)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return time.Duration(cp[idx])
}

// ResetFrameMetrics 清空帧环形缓冲(测试与「重置统计」)。
func ResetFrameMetrics() {
	frameMu.Lock()
	frameHead = 0
	frameCount = 0
	frameJank167, frameJank33, frameJank100 = 0, 0, 0
	for i := range frameBuf {
		frameBuf[i] = 0
	}
	frameMu.Unlock()
}
