package app

import (
	"bytes"
	"fmt"
	"image/png"
	stdlog "log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

// 拖动 / 重排性能基准(默认跳过,要指定真实会话文件才跑):
//
//	cd mygo && MINI_PERF_SESSION=~/.claude/projects/<项目>/<id>.jsonl \
//	  go test ./internal/app -run TestPerfResize -v -count=1 -cpuprofile /tmp/resize.prof
//
// 环境变量:MINI_PERF_SESSION(会话文件)、MINI_PERF_CWD(会话目录,默认当前)、MINI_PERF_MODE(agent / ide)、
// MINI_PERF_BUDGET_MS(单帧预算,超过则失败,默认只打印)。
// 方法:先布局一帧暖缓存(用户拖动前窗口已经排过一次版),然后模拟拖动窗口(宽度逐像素变化)、
// 拖动对话栏分隔线(a.ly.dock 逐像素变化),逐帧计时并按分位数汇总。
func TestPerfResize(t *testing.T) {
	path := os.Getenv("MINI_PERF_SESSION")
	if path == "" {
		t.Skip("设置 MINI_PERF_SESSION 指向会话 jsonl 才运行")
	}
	cwd := os.Getenv("MINI_PERF_CWD")
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	evs, err := claude.LoadSession(path, cwd)
	if err != nil {
		t.Fatal(err)
	}
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	a.Attach(&fakeBE{})
	for _, ev := range evs {
		a.st.Apply(ev)
	}
	if os.Getenv("MINI_PERF_MODE") == "ide" {
		a.mode = ModeIDE
	}
	t.Logf("会话 %s:%d 个事件 → %d 条时间线条目", path, len(evs), len(a.st.Items))

	tt := ui.NewTester(a.View, 1280, 800)
	t0 := time.Now()
	tt.Frame()
	t.Logf("首帧 %v", time.Since(t0))
	t0 = time.Now()
	tt.Frame()
	t.Logf("暖缓存后的静止帧 %v", time.Since(t0))

	dump := func(name string) {
		if p := os.Getenv("MINI_PERF_DUMP"); p != "" {
			f, err := os.Create(p + "-" + name + ".png")
			if err == nil {
				_ = png.Encode(f, tt.Image())
				f.Close()
			}
		}
	}
	dump("start")
	var logBuf bytes.Buffer
	stdlog.SetOutput(&logBuf) // MYGO_FRAME_STATS=all 时库把每帧各阶段耗时写到标准日志,这里截下来按场景汇总
	defer stdlog.SetOutput(os.Stderr)
	run := func(name string, n int, step func(i int)) {
		logBuf.Reset()
		ds := make([]time.Duration, 0, n)
		rowsBuilt := 0
		var rowDur time.Duration
		kinds := map[string]*flowKindStat{}
		for i := 0; i < n; i++ {
			s := time.Now()
			step(i) // SetSize 自己会排一帧,也算进这一步的耗时
			tt.Frame()
			ds = append(ds, time.Since(s))
			rowsBuilt += a.flowPerf.rows
			rowDur += a.flowPerf.total
			for k, v := range a.flowPerf.kinds {
				ks := kinds[k]
				if ks == nil {
					ks = &flowKindStat{}
					kinds[k] = ks
				}
				ks.n += v.n
				ks.d += v.d
			}
		}
		report(t, name, ds)
		phaseReport(t, &logBuf)
		line := fmt.Sprintf("    └ 每帧平均构建 %.1f 行,行构建合计 %v/帧;", float64(rowsBuilt)/float64(n), (rowDur / time.Duration(n)).Round(10*time.Microsecond))
		for k, v := range kinds {
			line += fmt.Sprintf(" %s=%d行/%v", k, v.n, (v.d / time.Duration(max(v.n, 1))).Round(10*time.Microsecond))
		}
		t.Log(line)
	}

	// 1. 拖动窗口:1280 → 900 → 1280,每帧 ±4px。
	run("拖动窗口宽度", 200, func(i int) {
		w := 1280 - i*4
		if i >= 100 {
			w = 880 + (i-100)*4
		}
		tt.SetSize(w, 800)
	})
	tt.SetSize(900, 800)
	tt.Frame()
	dump("w900")
	tt.SetSize(1280, 800)
	tt.Frame()

	// 2. 拖动窗口高度(宽度不变,不应该重排文字)。
	run("拖动窗口高度", 100, func(i int) {
		h := 800 - i*3
		if i >= 50 {
			h = 650 + (i-50)*3
		}
		tt.SetSize(1280, h)
	})
	tt.SetSize(1280, 800)
	tt.Frame()

	// 3. IDE 形态下拖动对话栏分隔线(对话栏宽度逐像素变化)。
	a.SetMode(ModeIDE)
	a.rightOn = true
	tt.Frame()
	tt.Frame()
	dump("ide")
	run("拖动对话栏分隔线", 150, func(i int) {
		w := float32(300 + i*2)
		if i >= 75 {
			w = float32(300 + (150-i)*2)
		}
		a.ly.dock = w
	})

	// 4. 静止帧(没有任何变化,应该几乎不花时间)。
	run("静止帧", 60, func(i int) {})

	if b := os.Getenv("MINI_PERF_BUDGET_MS"); b != "" {
		var ms float64
		fmt.Sscanf(b, "%f", &ms)
		_ = ms
	}
}

func report(t *testing.T, name string, ds []time.Duration) {
	t.Helper()
	s := append([]time.Duration(nil), ds...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	var sum time.Duration
	over16, over33 := 0, 0
	for _, d := range ds {
		sum += d
		if d > 16700*time.Microsecond {
			over16++
		}
		if d > 33*time.Millisecond {
			over33++
		}
	}
	q := func(p float64) time.Duration { return s[int(float64(len(s)-1)*p)] }
	t.Logf("%-14s n=%d 平均=%v p50=%v p95=%v p99=%v 最大=%v  >16.7ms:%d 帧  >33ms:%d 帧",
		name, len(ds), (sum / time.Duration(len(ds))).Round(10*time.Microsecond), q(.5).Round(10*time.Microsecond),
		q(.95).Round(10*time.Microsecond), q(.99).Round(10*time.Microsecond), s[len(s)-1].Round(10*time.Microsecond), over16, over33)
}

var frameLine = regexp.MustCompile(`frame \d+ took ([\d.]+) ms: build ([\d.]+), layout ([\d.]+), paint ([\d.]+), present ([\d.]+).*?(\d+) text layouts`)

// phaseReport 汇总库的帧日志(需 MYGO_FRAME_STATS=all):逐引擎帧统计,而不是逐「步」:
// 无窗口测试器每步会连续排最多 20 帧直到界面静止(动画 / 光标闪烁会一直要帧),
// 把它们的总和当成「一帧」会把流畅度高估十几倍。
// 「构建 + 布局」是真机主线程上的 CPU 开销;「绘制」在测试器里是 CPU 软件光栅,比真机 GPU 慢,单列。
func phaseReport(t *testing.T, buf *bytes.Buffer) {
	t.Helper()
	var n int
	var tot, bu, la, pa, pr, tl float64
	var cpu []float64
	over16, over33 := 0, 0
	for _, m := range frameLine.FindAllStringSubmatch(buf.String(), -1) {
		f := func(i int) float64 { v, _ := strconv.ParseFloat(m[i], 64); return v }
		n++
		tot += f(1)
		bu += f(2)
		la += f(3)
		pa += f(4)
		pr += f(5)
		tl += f(6)
		c := f(2) + f(3)
		cpu = append(cpu, c)
		if c > 16.7 {
			over16++
		}
		if c > 33 {
			over33++
		}
	}
	if n == 0 {
		return
	}
	sort.Float64s(cpu)
	k := float64(n)
	q := func(p float64) float64 { return cpu[int(float64(len(cpu)-1)*p)] }
	t.Logf("    └ 引擎帧 %d 个:构建+布局 均值 %.2f / p95 %.2f / p99 %.2f / 最大 %.2f ms(超 16.7ms:%d 帧,超 33ms:%d 帧);均值 构建 %.2f 布局 %.2f 绘制(软件光栅)%.2f 呈现 %.2f;每帧文字排版 %.1f 次",
		n, (bu+la)/k, q(.95), q(.99), cpu[len(cpu)-1], over16, over33, bu/k, la/k, pa/k, pr/k, tl/k)
}

// 可视检查:载入真实会话,滚到指定行(MINI_PERF_ROW,默认中间),落一张 PNG(MINI_PERF_DUMP 前缀)。
//
//	MINI_PERF_SESSION=… MINI_PERF_DUMP=/tmp/v MINI_PERF_ROW=100 go test ./internal/app -run TestFlowListVisual -v
func TestFlowListVisual(t *testing.T) {
	path := os.Getenv("MINI_PERF_SESSION")
	if path == "" {
		t.Skip("设置 MINI_PERF_SESSION 才运行")
	}
	cwd := os.Getenv("MINI_PERF_CWD")
	evs, err := claude.LoadSession(path, cwd)
	if err != nil {
		t.Fatal(err)
	}
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	a.Attach(&fakeBE{})
	for _, ev := range evs {
		a.st.Apply(ev)
	}
	if os.Getenv("MINI_PERF_MODE") == "ide" {
		a.SetMode(ModeIDE)
		a.rightOn = true
	}
	tt := ui.NewTester(a.View, 1280, 800)
	for i := 0; i < 30; i++ {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	row := 0
	fmt.Sscanf(os.Getenv("MINI_PERF_ROW"), "%d", &row)
	if row <= 0 {
		row = len(a.flowRows()) / 2
	}
	a.flowList.ScrollTo(row, ui.Start)
	for i := 0; i < 6; i++ {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	first, last := a.flowList.Visible()
	t.Logf("行数=%d 滚到 %d,可见 %d–%d,AtEnd=%v", len(a.flowRows()), row, first, last, a.flowList.AtEnd())
	if p := os.Getenv("MINI_PERF_DUMP"); p != "" {
		f, _ := os.Create(p + "-visual.png")
		_ = png.Encode(f, tt.Image())
		f.Close()
	}
	if os.Getenv("MINI_PERF_MODE") == "ide" && os.Getenv("MINI_PERF_DUMP") != "" {
		for i, w := range []float32{420, 500, 380} { // 拖动分隔线:每次改完只排一帧就截图,看过渡中途是否正常
			a.ly.dock = w
			tt.Frame()
			f, _ := os.Create(fmt.Sprintf("%s-drag%d.png", os.Getenv("MINI_PERF_DUMP"), i))
			_ = png.Encode(f, tt.Image())
			f.Close()
		}
	}
}
