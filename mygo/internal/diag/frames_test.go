package diag

import (
	"testing"
	"time"
)

func TestFrameMetricsPercentile(t *testing.T) {
	ResetFrameMetrics()
	for _, d := range []time.Duration{8 * time.Millisecond, 12 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond} {
		RecordFrame(d)
	}
	m := FrameMetricsSnapshot()
	if m.Samples != 4 {
		t.Fatalf("samples=%d", m.Samples)
	}
	if m.P50 < 10*time.Millisecond || m.P50 > 15*time.Millisecond {
		t.Fatalf("p50=%v", m.P50)
	}
	if m.Jank167Pct <= 0 {
		t.Fatal("应有 >16.7ms 占比")
	}
}
