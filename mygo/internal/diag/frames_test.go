package diag

import (
	"testing"
	"time"
)

func TestFrameMetricsIdleAndActive(t *testing.T) {
	ResetFrameMetrics()
	m := FrameMetricsSnapshot()
	if !m.Idle {
		t.Fatal("无帧应空闲")
	}
	for i := 0; i < 30; i++ {
		RecordUIFrame(8*time.Millisecond, 16*time.Millisecond)
	}
	m = FrameMetricsSnapshot()
	if m.Idle || m.Samples < 10 || m.FPS < 30 {
		t.Fatalf("应有活跃帧率: %+v", m)
	}
	time.Sleep(frameIdleGap + 50*time.Millisecond)
	m = FrameMetricsSnapshot()
	if !m.Idle {
		t.Fatal("长时间无帧应回到空闲")
	}
}
