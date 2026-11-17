package diag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// TrendPoint 每 5 秒一次的资源趋势点(内存 / CPU / 协程)。
type TrendPoint struct {
	At         time.Time
	CPU        float64 // 进程 CPU%,100=单核满载
	RSS        uint64  // 常驻内存(字节)
	HeapAlloc  uint64
	Goroutines int
	FrameBuild time.Duration // 窗口内 app.构建界面 平均
}

const trendKeep1h = 720 // 1 小时 @ 5s

var (
	trendMu      sync.Mutex
	trendPts     []TrendPoint
	trendStarted atomic.Bool
	runCPU       float64
	runCPUMu     sync.Mutex
	firstFrameAt time.Time
	firstFrameMu sync.Mutex
	bootStarted  time.Time
)

// MarkFirstFrame 在首帧呈现后调用,供启动耗时展示。
func MarkFirstFrame() {
	firstFrameMu.Lock()
	if firstFrameAt.IsZero() {
		firstFrameAt = time.Now()
	}
	firstFrameMu.Unlock()
}

// SetBootStarted 由 Init 写入启动时刻。
func SetBootStarted(t time.Time) { bootStarted = t }

// BootMetrics 返回冷启动到首帧耗时。
func BootMetrics() (coldStart time.Duration, ok bool) {
	if bootStarted.IsZero() {
		return 0, false
	}
	firstFrameMu.Lock()
	ff := firstFrameAt
	firstFrameMu.Unlock()
	if ff.IsZero() {
		return time.Since(bootStarted), true
	}
	return ff.Sub(bootStarted), true
}

// StartTrendSampler 启动 5 秒趋势采样(重复调用无效)。
func StartTrendSampler() {
	if !trendStarted.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer Recover("diag.trendSampler")
		for {
			time.Sleep(5 * time.Second)
			pt := takeTrendPoint()
			trendMu.Lock()
			trendPts = append(trendPts, pt)
			if len(trendPts) > trendKeep1h {
				trendPts = trendPts[len(trendPts)-trendKeep1h:]
			}
			trendMu.Unlock()
		}
	}()
}

func takeTrendPoint() TrendPoint {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	cpu := sampleProcessCPU()
	runCPUMu.Lock()
	runCPU = cpu
	runCPUMu.Unlock()
	var avgBuild time.Duration
	for _, s := range Stats() {
		if s.Name == "app.构建界面" && s.Count > 0 {
			avgBuild = s.Mean()
			break
		}
	}
	return TrendPoint{
		At:         time.Now(),
		CPU:        cpu,
		RSS:        sampleProcessRSS(),
		HeapAlloc:  ms.HeapAlloc,
		Goroutines: runtime.NumGoroutine(),
		FrameBuild: avgBuild,
	}
}

// Trends 返回趋势点副本(最旧在前)。window 为 0 表示全部(最多 1 小时)。
func Trends(window time.Duration) []TrendPoint {
	trendMu.Lock()
	defer trendMu.Unlock()
	if len(trendPts) == 0 {
		return nil
	}
	if window <= 0 {
		return append([]TrendPoint(nil), trendPts...)
	}
	cut := time.Now().Add(-window)
	out := make([]TrendPoint, 0, len(trendPts))
	for _, p := range trendPts {
		if !p.At.Before(cut) {
			out = append(out, p)
		}
	}
	return out
}

// CurrentCPU 返回最近一次趋势采样的 CPU%。
func CurrentCPU() float64 {
	runCPUMu.Lock()
	defer runCPUMu.Unlock()
	return runCPU
}

// ChartData 是趋势图序列。
type ChartData struct {
	Labels []string
	CPU    []float64
	RSS    []float64
	Heap   []float64
}

// ChartSeries 按范围返回 CPU / RSS / 堆(5 分钟用 1s 采样,更长用 5s 趋势)。
func ChartSeries(window time.Duration) ChartData {
	var d ChartData
	if window <= 5*time.Minute+time.Second {
		for _, s := range Samples() {
			d.Labels = append(d.Labels, s.At.Format("15:04:05"))
			d.CPU = append(d.CPU, s.CPU)
			d.RSS = append(d.RSS, float64(s.RSS))
			d.Heap = append(d.Heap, float64(s.HeapAlloc))
		}
		return d
	}
	for _, p := range Trends(window) {
		d.Labels = append(d.Labels, p.At.Format("15:04"))
		d.CPU = append(d.CPU, p.CPU)
		d.RSS = append(d.RSS, float64(p.RSS))
		d.Heap = append(d.Heap, float64(p.HeapAlloc))
	}
	return d
}

// PerformanceSnapshotJSON 供诊断包附带的性能快照。
func PerformanceSnapshotJSON() []byte {
	fm := FrameMetricsSnapshot()
	cold, _ := BootMetrics()
	top := Stats()
	if len(top) > 20 {
		top = top[:20]
	}
	snap := map[string]any{
		"frames":      fm,
		"coldStartMs": cold.Milliseconds(),
		"trendCount":  len(Trends(0)),
		"statsTop":    top,
		"counters":    Counters(),
		"cpuNow":      CurrentCPU(),
	}
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return []byte("{}")
	}
	return b
}

// ResetTrends 清空趋势(测试与重置)。
func ResetTrends() {
	trendMu.Lock()
	trendPts = nil
	trendMu.Unlock()
	runCPUMu.Lock()
	runCPU = 0
	runCPUMu.Unlock()
}

// FormatCPU 格式化为一位小数的百分比。
func FormatCPU(p float64) string {
	if p <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", p)
}

// ReadProcRSS 当前进程 RSS(字节)。
func ReadProcRSS() uint64 { return sampleProcessRSS() }

// LogViewLine 是日志查看器的一行。
type LogViewLine struct {
	Text  string
	Level string
	Mod   string
}

// ReadLogLines 从日志文件末尾读最多 maxLines 行,并按级别 / 模块 / 搜索过滤。
func ReadLogLines(maxLines int, lvl Level, modFilter, search string) ([]LogViewLine, error) {
	path := LogPath()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := bytes.Split(b, []byte{'\n'})
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	searchLow := strings.ToLower(search)
	var out []LogViewLine
	for _, ln := range lines {
		s := string(ln)
		if s == "" {
			continue
		}
		lv, mod := parseLogMeta(s)
		if lv < lvl {
			continue
		}
		if modFilter != "" && !strings.Contains(mod, modFilter) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(s), searchLow) {
			continue
		}
		out = append(out, LogViewLine{Text: s, Level: lv.String(), Mod: mod})
	}
	return out, nil
}

func parseLogMeta(line string) (Level, string) {
	parts := strings.SplitN(line, " ", 4)
	if len(parts) < 3 {
		return LevelDebug, ""
	}
	lv := ParseLevel(strings.TrimSpace(parts[2]), LevelInfo)
	mod := ""
	if i := strings.Index(line, "["); i >= 0 {
		if j := strings.Index(line[i:], "]"); j > 0 {
			mod = line[i+1 : i+j]
		}
	}
	return lv, mod
}
