package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const (
	pPerfVerbose   = "perf.verboseLog"
	pPerfDemoSet   = "perf.demoSet" // 演示数据套(仅演示模式)
	pPerfChartRng  = "perf.chartRng"
	pPerfLogSearch = "perf.logSearch"
	pPerfLogMod    = "perf.logMod"
)

// perfTabDef 在 settingsTabs 里引用 perfGroups。
var perfGroups = []setGroup{
	{title: "实时概览", body: perfOverviewBody},
	{title: "趋势", body: perfTrendBody},
	{title: "慢操作", body: perfSlowBody},
	{title: "卡死与停滞", body: perfHangBody},
	{title: "日志", rows: []setRow{
		rSeg(pPrivacyLvl, "日志级别", "环境变量 MINI_LOG 优先;此处改动立即生效并写入设置", 2, "错误", "警告", "信息", "调试"),
		rTog(pPerfVerbose, "详细性能日志", "额外记录界面切换、超阈值帧与后台任务耗时", false),
		rCustom("日志文件", fmt.Sprintf("单文件上限 %s · 保留 %d 份", diag.FormatBytes(diag.MaxLogBytes), diag.LogFiles), func(a *App, c *ui.Context) {
			ui.Row(c).Gap(8).Children(func() {
				kit.Btn(c, "打开目录", kit.BtnOpt{Small: true, Icon: "folder"}).OnClick(func() { a.openPath(diag.Home() + "/logs") })
				kit.Btn(c, "复制最近 200 行", kit.BtnOpt{Small: true, Icon: "copy"}).OnClick(func() { a.copyLogTail(200) })
			})
		}),
	}, body: perfLogViewerBody},
	{title: "诊断包与采样", rows: []setRow{
		rCustom("诊断包", "导出 zip,含配置摘要、日志尾部与 performance.json", func(a *App, c *ui.Context) {
			kit.Btn(c, "导出", kit.BtnOpt{Small: true, Icon: "download"}).OnClick(func() { a.exportDiagBundle() })
		}),
		rCustom("统计", "清空内存中的慢操作、帧与趋势计数(不写磁盘)", func(a *App, c *ui.Context) {
			kit.Btn(c, "重置统计", kit.BtnOpt{Small: true, Variant: kit.BtnDanger}).OnClick(func() {
				diag.ResetAllMetrics()
				a.flash("已重置性能统计")
			})
		}),
	}},
	{title: "阈值与看门狗", rows: []setRow{
		rTog(pAdvWatchdog, "主线程卡死看门狗", "停滞超过阈值时抓取调用栈并写入记录", true),
		rCustom("慢帧日志", "由环境变量 MYGO_FRAME_STATS 控制(毫秒);默认 8", func(a *App, c *ui.Context) {
			v := os.Getenv("MYGO_FRAME_STATS")
			if v == "" {
				v = "8"
			}
			kit.Mono(c, v+" ms", a.tk.Tx3)
		}),
	}},
}

func (a *App) perfDemoSet() demo.PerfDemoSet {
	if !a.demoWs() {
		return 0
	}
	i := a.prefs.Int(pPerfDemoSet, 0)
	if i < 0 {
		i = 0
	}
	return demo.PerfDemoSet(i % 3)
}

func (a *App) perfChartWindow() time.Duration {
	if a.prefs.Int(pPerfChartRng, 0) == 1 {
		return time.Hour
	}
	return 5 * time.Minute
}

func perfOverviewBody(a *App, c *ui.Context) {
	t := a.tk
	demoTag := ""
	if a.demoWs() {
		demoTag = demo.PerfOverviewFor(a.perfDemoSet()).Label
		ui.Row(c).Padding(8, 12, 4, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			kit.Tag(c, "演示数据", kit.KWarn)
			ui.Text(c, demoTag).FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1)
			kit.Btn(c, "切换数据套", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
				n := (int(a.perfDemoSet()) + 1) % 3
				a.prefs.Set(pPerfDemoSet, n)
			})
		})
	}
	fm := diag.FrameMetricsSnapshot()
	cpu := diag.CurrentCPU()
	rss := diag.ReadProcRSS()
	gr := 0
	if ss := diag.Samples(); len(ss) > 0 {
		gr = ss[len(ss)-1].Goroutines
	}
	cold, coldOk := diag.BootMetrics()
	if a.demoWs() && a.perfDemoSet() != 2 {
		d := demo.PerfOverviewFor(a.perfDemoSet())
		fm.FPS = d.FPS
		fm.P50 = time.Duration(d.FrameP50Ms * float64(time.Millisecond))
		fm.P95 = time.Duration(d.FrameP95Ms * float64(time.Millisecond))
		fm.P99 = time.Duration(d.FrameP99Ms * float64(time.Millisecond))
		fm.Jank167Pct, fm.Jank33Pct, fm.Jank100Pct = d.Jank167, d.Jank33, d.Jank100
		cpu = d.CPU
		rss = uint64(d.RSSMB * (1 << 20))
		gr = d.Goroutines
	}
	card := func(title, val, sub string) {
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Padding(10, 12).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(theme.FsCap).TextColor(t.Tx3)
			ui.Text(c, val).FontWeight(600).FontFeatures("tnum")
			if sub != "" {
				ui.Text(c, sub).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine()
			}
		})
	}
	ui.Row(c).Wrap().Children(func() {
		fps := "—"
		if fm.FPS > 0 {
			fps = fmt.Sprintf("%.0f", fm.FPS)
		}
		card("帧率", fps, "最近窗口")
		card("帧耗时 p50", fmtPerfDur(fm.P50), "p95 "+fmtPerfDur(fm.P95)+" · p99 "+fmtPerfDur(fm.P99))
		card("卡顿 >16.7ms", fmt.Sprintf("%.1f%%", fm.Jank167Pct), fmt.Sprintf(">33ms %.1f%% · >100ms %.1f%%", fm.Jank33Pct, fm.Jank100Pct))
		card("CPU", diag.FormatCPU(cpu), "100% = 单核满载")
		card("内存 RSS", diag.FormatBytes(rss), "堆 "+diag.FormatBytes(readHeap()))
		card("协程", itoa(gr), "子进程计数未接入")
	})
	rowBox(a, c, true, 36).Children(func() {
		titled(a, c, "启动到首帧", "冷启动耗时", false)
		val := "采集中…"
		if coldOk {
			val = fmtPerfDur(cold)
		}
		if a.demoWs() {
			val = "1.24 s · 演示"
		}
		ui.Text(c, val).FontWeight(600).FontFeatures("tnum")
	})
}

func readHeap() uint64 {
	ss := diag.Samples()
	if len(ss) == 0 {
		return 0
	}
	return ss[len(ss)-1].HeapAlloc
}

func perfTrendBody(a *App, c *ui.Context) {
	t := a.tk
	ui.Row(c).Padding(8, 12, 6, 12).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "范围").FontSize(theme.FsCap).TextColor(t.Tx3)
		rng := a.prefs.Int(pPerfChartRng, 0)
		if i := kit.Seg(c, []kit.SegItem{{Text: "最近 5 分钟"}, {Text: "最近 1 小时"}}, rng, kit.SegOpt{Key: "perf-rng"}); i >= 0 {
			a.prefs.Set(pPerfChartRng, i)
			a.set.perfHover = -1
		}
		ui.Text(c, "悬停显示数值与十字线").FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1).TextAlign(ui.End)
	})
	win := a.perfChartWindow()
	cpu, mem, labels := diag.ChartSeries(win)
	if a.demoWs() && len(mem) < 8 {
		cpu, mem, labels = demo.PerfTrendDemo(a.perfDemoSet(), 48)
	}
	if len(mem) < 2 {
		ui.Column(c).Padding(24, 12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "数据不足").TextColor(t.Tx3)
			ui.Text(c, "启动后约 10 秒会有曲线").FontSize(theme.FsCap).TextColor(t.Tx4)
		})
		return
	}
	perfLineChart(a, c, "内存", mem, labels, t.Info)
	perfLineChart(a, c, "CPU", cpu, labels, t.Ac)
}

func perfLineChart(a *App, c *ui.Context, title string, vals []float64, labels []string, col ui.Color) {
	t := a.tk
	a.box(c).Padding(10, 12).Gap(6).Children(func() {
		ui.Text(c, title).FontWeight(500)
		hover := a.set.perfHover
		ui.Box(c).Grow(1).Height(120).Clip().Children(func() {
			ui.Box(c).Grow(1).Draw(func(p *ui.Painter, r ui.Rect) {
				drawPerfSeries(p, r, vals, col, hover)
			})
			ui.Row(c).Grow(1).Absolute().Left(0).Right(0).Top(0).Bottom(0).Children(func() {
				n := len(vals)
				for i := 0; i < n; i++ {
					i := i
					str := fmtVal(title, vals[i], labels[i])
					strip := ui.Box(c.Key(fmt.Sprintf("perf-%s-%d", title, i))).Grow(1).Basis(0).Tooltip(str)
					if strip.Hovered() {
						a.set.perfHover = i
					}
				}
			})
		})
		if hover := a.set.perfHover; hover >= 0 && hover < len(labels) {
			ui.Text(c, labels[hover]+" · "+fmtVal(title, vals[hover], labels[hover])).FontSize(theme.FsCap).TextColor(t.Tx3).FontFeatures("tnum")
		}
	})
}

func fmtVal(title string, v float64, _ string) string {
	if title == "CPU" {
		return fmt.Sprintf("%.1f%%", v)
	}
	return diag.FormatBytes(uint64(v))
}

func drawPerfSeries(p *ui.Painter, r ui.Rect, vals []float64, col ui.Color, hover int) {
	if len(vals) < 2 {
		return
	}
	lo, hi := vals[0], vals[0]
	for _, v := range vals {
		lo, hi = min(lo, v), max(hi, v)
	}
	if hi == lo {
		hi = lo + 1
	}
	pts := make([][2]float32, len(vals))
	for i, v := range vals {
		x := r.X + float32(i)/float32(len(vals)-1)*r.W
		y := r.Y + r.H - 2 - float32((v-lo)/(hi-lo))*(r.H-4)
		pts[i] = [2]float32{x, y}
	}
	p.StrokePolyline(pts, 1.5, col)
	if hover >= 0 && hover < len(pts) {
		x := pts[hover][0]
		p.StrokePolyline([][2]float32{{x, r.Y}, {x, r.Y + r.H}}, 1, col.Alpha(.5))
	}
}

func perfSlowBody(a *App, c *ui.Context) {
	stats := diag.Stats()
	if len(stats) == 0 {
		rowBox(a, c, true, 40).Children(func() {
			ui.Text(c, "尚无慢操作样本").TextColor(a.tk.Tx3).Grow(1)
		})
		return
	}
	if a.set.perfSlowOpen == nil {
		a.set.perfSlowOpen = map[string]bool{}
	}
	limit := 12
	if len(stats) < limit {
		limit = len(stats)
	}
	for i, st := range stats[:limit] {
		st := st
		i := i
		rowBox(a, c, i == 0, 44).Children(func() {
			ui.Column(c).Grow(1).Gap(1).Children(func() {
				ui.Text(c, st.Name).FontWeight(500).SingleLine()
				ui.Text(c, fmt.Sprintf("次数 %d · 总 %s · p50 %s · p95 %s · 最大 %s", st.Count, fmtPerfDur(st.Total), fmtPerfDur(st.P50()), fmtPerfDur(st.P95()), fmtPerfDur(st.Max))).
					FontSize(theme.FsCap).TextColor(a.tk.Tx3).FontFeatures("tnum")
			})
			kit.Btn(c, "展开", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() {
				a.set.perfSlowOpen[st.Name] = !a.set.perfSlowOpen[st.Name]
			})
		})
		if a.set.perfSlowOpen[st.Name] {
			ui.Text(c, fmt.Sprintf("慢于阈值 %d 次 · 平均 %s", st.Slow, fmtPerfDur(st.Mean()))).FontSize(theme.FsCap).TextColor(a.tk.Tx4).Padding(0, 24, 8, 12)
		}
	}
}

func perfHangBody(a *App, c *ui.Context) {
	t := a.tk
	events := diag.HangEvents()
	if a.demoWs() {
		events = demo.PerfHangsDemo(a.perfDemoSet())
	}
	if len(events) == 0 {
		rowBox(a, c, true, 40).Children(func() {
			kit.Ic(c, "okc", kit.S12).TextColor(t.Ok)
			ui.Text(c, "最近没有停滞记录").Grow(1)
		})
		return
	}
	for i, ev := range events {
		ev := ev
		rowBox(a, c, i == 0, 48).Children(func() {
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Text(c, ev.Kind+" · "+ev.Stall.Round(time.Millisecond).String()).FontWeight(500)
				ui.Text(c, ev.At.Format("2006-01-02 15:04:05")).FontSize(theme.FsCap).TextColor(t.Tx3)
				if ev.Stack != "" {
					ui.Text(c, ev.Stack).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine()
				}
			})
			kit.Btn(c, "复制栈", kit.BtnOpt{Small: true, Icon: "copy"}).OnClick(func() {
				a.copyText = ev.Stack
				a.flash("已复制")
			})
			if ev.LogPath != "" {
				kit.Btn(c, "记录目录", kit.BtnOpt{Small: true, Icon: "folder"}).OnClick(func() { a.openPath(diag.CrashDir()) })
			}
		})
	}
	rowBox(a, c, false, 36).Children(func() {
		titled(a, c, "崩溃记录", "与隐私页相同列表", false)
		kit.Btn(c, "在访达中打开", kit.BtnOpt{Small: true, Icon: "folder"}).OnClick(func() { a.openPath(diag.CrashDir()) })
	})
}

func perfLogViewerBody(a *App, c *ui.Context) {
	t := a.tk
	if a.set.bufs == nil {
		a.set.bufs = map[string]*string{}
	}
	qKey, mKey := pPerfLogSearch, pPerfLogMod
	q, okQ := a.set.bufs[qKey]
	if !okQ {
		s := a.prefs.Str(qKey, "")
		q = &s
		a.set.bufs[qKey] = q
	}
	m, okM := a.set.bufs[mKey]
	if !okM {
		s := a.prefs.Str(mKey, "")
		m = &s
		a.set.bufs[mKey] = m
	}
	ui.Row(c).Padding(0, 12, 8, 12).Gap(8).Children(func() {
		inQ := ui.TextInputBase(c.Key("perf-log-q"), q).Grow(1).Placeholder("搜索日志…")
		inQ.OnChange(func() { a.prefs.Set(qKey, *q) })
		inM := ui.TextInputBase(c.Key("perf-log-m"), m).Width(100).Placeholder("模块").Font("monospace")
		inM.OnChange(func() { a.prefs.Set(mKey, *m) })
	})
	lv := diag.CurrentLevel()
	lines, _ := diag.ReadLogLines(80, lv, strings.TrimSpace(*m), strings.TrimSpace(*q))
	if len(lines) == 0 {
		ui.Text(c, "没有匹配的日志行(或文件尚空)").FontSize(theme.FsCap).TextColor(t.Tx3).Padding(8, 12)
		return
	}
	mono := ui.Column(c).Padding(8, 12).Gap(2)
	mono.Children(func() {
		for _, ln := range lines {
			col := t.Tx2
			if strings.Contains(ln.Level, "WARN") {
				col = t.Warn
			}
			if strings.Contains(ln.Level, "ERROR") {
				col = t.Err
			}
			ui.Text(c, ln.Text).Font("monospace").FontSize(theme.FsMicro).TextColor(col).SingleLine()
		}
	})
}

func (a *App) copyLogTail(n int) {
	lines, err := diag.ReadLogLines(n, diag.LevelDebug, "", "")
	if err != nil {
		a.flash("读取日志失败")
		return
	}
	var b strings.Builder
	for _, ln := range lines {
		b.WriteString(ln.Text)
		b.WriteByte('\n')
	}
	a.copyText = b.String()
	a.flash("已复制最近日志")
}

// perfTabNote 在设置分类底部展示。
const perfTabNote = "性能数据仅在本机采集中,不会上报。演示模式部分曲线为模拟数据并标有「演示数据」。"
