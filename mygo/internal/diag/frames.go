package diag

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// 帧指标只看最近 frameWindow;超过 frameIdleGap 没有新帧视为空闲。
const (
	frameRingN   = 512
	frameWindow  = 5 * time.Second
	frameIdleGap = 1200 * time.Millisecond
	ms167        = 167 * time.Microsecond * 100 // 16.7ms
	ms33         = 33 * time.Millisecond
	ms100        = 100 * time.Millisecond
)

type frameTick struct {
	At    time.Time
	Build time.Duration
	Gap   time.Duration // 与上一帧的间隔;首帧为 0
}

var (
	frameMu     sync.Mutex
	frameBuf    = make([]frameTick, frameRingN)
	frameHead   int
	frameTotal  int64
	lastFrameAt atomic.Int64 // unix nano
)

// RecordFrame 兼容 stderr 解析:记入总帧耗时(含 present)。
func RecordFrame(d time.Duration) { recordFrameTick(frameTick{At: time.Now(), Build: d, Gap: 0}) }

// RecordUIFrame 在每次 View 构建结束调用:build=构建耗时,gap=距上次 View 的间隔。
func RecordUIFrame(build, gap time.Duration) {
	now := time.Now()
	recordFrameTick(frameTick{At: now, Build: build, Gap: gap})
	lastFrameAt.Store(now.UnixNano())
	slow := build >= FrameSlow || gap >= 34*time.Millisecond
	PerfFrame(build, gap, slow)
}

func recordFrameTick(t frameTick) {
	frameMu.Lock()
	frameBuf[frameHead] = t
	frameHead = (frameHead + 1) % frameRingN
	frameTotal++
	frameMu.Unlock()
}

// FrameMetrics 是帧与卡顿的只读快照。
type FrameMetrics struct {
	Idle       bool
	Samples    int
	FPS        float64
	P50        time.Duration
	P95        time.Duration
	P99        time.Duration
	Jank167Pct float64
	Jank33Pct  float64
	Jank100Pct float64
}

// FrameMetricsSnapshot 返回最近窗口内的帧指标。
func FrameMetricsSnapshot() FrameMetrics {
	now := time.Now()
	ticks := recentTicks(now)
	var out FrameMetrics
	if len(ticks) == 0 {
		out.Idle = true
		return out
	}
	last := ticks[len(ticks)-1].At
	if now.Sub(last) > frameIdleGap {
		out.Idle = true
		return out
	}
	out.Samples = len(ticks)
	var builds, gaps []float64
	var j167, j33, j100 int
	for _, t := range ticks {
		builds = append(builds, float64(t.Build))
		if t.Gap > 0 {
			gaps = append(gaps, float64(t.Gap))
		}
		if t.Build > ms167 {
			j167++
		}
		if t.Build > ms33 {
			j33++
		}
		if t.Build > ms100 {
			j100++
		}
	}
	if n := len(ticks); n > 0 {
		out.Jank167Pct = 100 * float64(j167) / float64(n)
		out.Jank33Pct = 100 * float64(j33) / float64(n)
		out.Jank100Pct = 100 * float64(j100) / float64(n)
	}
	if sum := sumDur(gaps); sum > 0 && len(gaps) > 0 {
		out.FPS = float64(len(gaps)) / (sum / float64(time.Second))
	}
	out.P50 = durPercentile(builds, 0.50)
	out.P95 = durPercentile(builds, 0.95)
	out.P99 = durPercentile(builds, 0.99)
	return out
}

func recentTicks(now time.Time) []frameTick {
	frameMu.Lock()
	defer frameMu.Unlock()
	cut := now.Add(-frameWindow)
	n := int(frameTotal)
	if n > frameRingN {
		n = frameRingN
	}
	var rev []frameTick
	for i := 0; i < n; i++ {
		idx := (frameHead - 1 - i + frameRingN) % frameRingN
		t := frameBuf[idx]
		if t.At.IsZero() || t.At.Before(cut) {
			break
		}
		rev = append(rev, t)
	}
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return rev
}

func sumDur(fs []float64) float64 {
	var s float64
	for _, f := range fs {
		s += f
	}
	return s
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
	frameTotal = 0
	for i := range frameBuf {
		frameBuf[i] = frameTick{}
	}
	frameMu.Unlock()
	lastFrameAt.Store(0)
}
