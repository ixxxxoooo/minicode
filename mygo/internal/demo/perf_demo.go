package demo

import (
	"math"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// PerfDemoSet 演示性能数据的套号(0=正常,1=卡顿多,2=空数据)。
type PerfDemoSet int

// PerfOverview 是演示用的性能概览数字。
type PerfOverview struct {
	Label      string
	FPS        float64
	FrameP50Ms float64
	FrameP95Ms float64
	FrameP99Ms float64
	Jank167    float64
	Jank33     float64
	Jank100    float64
	HangCount  int
	CPU        float64
	RSSMB      float64
	Goroutines int
}

// PerfOverviewFor 返回指定套的概览(界面标注「演示数据」)。
func PerfOverviewFor(set PerfDemoSet) PerfOverview {
	switch set % 3 {
	case 1:
		return PerfOverview{
			Label:      "演示: 卡顿偏多",
			FPS:        42,
			FrameP50Ms: 14.2,
			FrameP95Ms: 68,
			FrameP99Ms: 120,
			Jank167:    22,
			Jank33:     8.5,
			Jank100:    1.2,
			HangCount:  3,
			CPU:        18.4,
			RSSMB:      520,
			Goroutines: 187,
		}
	case 2:
		return PerfOverview{Label: "演示: 暂无采样", FPS: 0, CPU: 0, RSSMB: 0}
	default:
		return PerfOverview{
			Label:      "演示: 正常",
			FPS:        58,
			FrameP50Ms: 8.1,
			FrameP95Ms: 22,
			FrameP99Ms: 41,
			Jank167:    4.2,
			Jank33:     0.8,
			Jank100:    0,
			HangCount:  0,
			CPU:        6.2,
			RSSMB:      412,
			Goroutines: 92,
		}
	}
}

// PerfTrendDemo 生成假的 CPU / 内存曲线。
func PerfTrendDemo(set PerfDemoSet, n int) (cpu, mem []float64, labels []string) {
	base := time.Now().Add(-time.Duration(n) * 5 * time.Second)
	for i := 0; i < n; i++ {
		t := base.Add(time.Duration(i) * 5 * time.Second)
		labels = append(labels, t.Format("15:04"))
		switch set % 3 {
		case 1:
			cpu = append(cpu, 8+math.Sin(float64(i)/4)*6+float64(i%7))
			mem = append(mem, (450+float64(i%5)*8)*1048576)
		case 2:
			cpu = append(cpu, 0)
			mem = append(mem, 0)
		default:
			cpu = append(cpu, 4+math.Sin(float64(i)/6)*2)
			mem = append(mem, (380+float64(i%3)*5)*1048576)
		}
	}
	return cpu, mem, labels
}

// PerfHangsDemo 演示停滞记录。
func PerfHangsDemo(set PerfDemoSet) []diag.HangEvent {
	if set%3 != 1 {
		return nil
	}
	now := time.Now()
	return []diag.HangEvent{
		{At: now.Add(-2 * time.Minute), Stall: 1200 * time.Millisecond, Kind: "stall", Stack: "app.openSession@settings.go:42 <- app.buildFlow@flow.go:88"},
		{At: now.Add(-11 * time.Minute), Stall: 3200 * time.Millisecond, Kind: "dump", Stack: "app.buildIDE@ide.go:12", LogPath: "~/.minicode/crashes/hang-demo.txt"},
	}
}
