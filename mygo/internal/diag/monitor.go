package diag

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Options 是 Init 的参数。
type Options struct {
	Version string // 应用版本,写进崩溃记录
	Mode    string // 运行模式(demo / real / dev)
	Level   string // 日志级别 off/error/warn/info/debug,空 = info;环境变量 MINI_LOG 优先
	Mirror  bool   // 同时输出到标准输出
}

var meta Options

// Init 初始化诊断底座。必须在应用最开头调用,且只调用一次。
// 它依次:打开日志文件 → 补记上次的异常退出 → 写运行标记 → 把标准错误接到 stderr.log
// (Go 运行时的 fatal 报错、MyGo 的帧统计都在那里)→ 启动资源采样与帧统计解析。
func Init(o Options) {
	meta = o
	lv := ParseLevel(o.Level, LevelInfo)
	lv = ParseLevel(os.Getenv("MINI_LOG"), lv)
	SetLevel(lv)
	mirror.Store(o.Mirror)
	openLog(LogPath())

	// MyGo 的帧统计:超过 8 ms 的帧写一行到标准错误,我们解析后进入统计表
	if os.Getenv("MYGO_FRAME_STATS") == "" {
		_ = os.Setenv("MYGO_FRAME_STATS", "8")
	}
	started := time.Now()
	WriteStartupEnv(o.Version, o.Mode)
	write(LevelInfo, "diag", "启动", []any{"版本", o.Version, "模式", o.Mode, "run", RunID(), "pid", os.Getpid(), "go", runtime.Version(), "日志级别", lv.String(), "数据目录", Home()})
	if n := ScanStale(); n > 0 {
		write(LevelWarn, "diag", "补记了异常退出记录", []any{"条数", n, "目录", CrashDir()})
	}
	writeMarker(started)
	if os.Getenv("MINI_KEEP_STDERR") == "" { // 命令行工具(截图)设它,崩溃信息直接打在终端上
		captureStderr()
	}
	go heartbeatLoop(started)
	StartSampler()
	StartTrendSampler()
	StartProctreeSampler()
}

// heartbeatLoop 每 5 秒刷新运行标记的心跳时间。
func heartbeatLoop(started time.Time) {
	defer Recover("diag.heartbeat")
	for {
		time.Sleep(5 * time.Second)
		writeMarker(started)
	}
}

// ───────── 标准错误接管与帧统计解析 ─────────

var stderrPath = func() string { return filepath.Join(Home(), "logs", "stderr.log") }

// captureStderr 把进程的标准错误(fd 2)重定向到 logs/stderr.log:直接 dup2 到文件,
// 所以即便进程被 Go 运行时的 fatal error 杀掉,最后的栈也已经落盘。上一次的内容留作 stderr.log.prev。
func captureStderr() {
	path := stderrPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.Rename(path, path+".prev")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		write(LevelWarn, "diag", "无法接管标准错误", []any{"error", err})
		return
	}
	if err := dup2(int(f.Fd()), 2); err != nil {
		write(LevelWarn, "diag", "dup2 失败", []any{"error", err})
		return
	}
	go tailFrames(path)
}

// frameRe 匹配 MyGo 的帧日志:`mygo: frame 412 took 17.3 ms: build 3.1 (2 passes), layout 1.2, paint 0.8, present 11.5 …`
var frameRe = regexp.MustCompile(`mygo: frame (\d+) took ([\d.]+) ms: build ([\d.]+)(?: \((\d+) passes?\))?, layout ([\d.]+), paint ([\d.]+), present ([\d.]+)`)

// FrameSlow 单帧超过它就记 WARN(60fps 下约两帧)。
const FrameSlow = 33 * time.Millisecond

// ParseFrameLine 解析一行帧日志并记入统计;不是帧日志返回 false。
func ParseFrameLine(line string) bool {
	m := frameRe.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	ms := func(s string) time.Duration {
		f, _ := strconv.ParseFloat(s, 64)
		return time.Duration(f * float64(time.Millisecond))
	}
	total := ms(m[2])
	RecordFrame(total)
	record("ui.frame", total, FrameSlow)
	record("ui.frame.build", ms(m[3]), FrameSlow)
	record("ui.frame.layout", ms(m[5]), FrameSlow)
	record("ui.frame.paint", ms(m[6]), FrameSlow)
	record("ui.frame.present", ms(m[7]), FrameSlow)
	NewCounter("ui.frame.slow8ms").Inc()
	if total >= FrameSlow {
		write(LevelWarn, "ui", "慢帧", []any{"帧", m[1], "总计", total, "构建", ms(m[3]), "轮数", m[4], "布局", ms(m[5]), "绘制", ms(m[6]), "呈现", ms(m[7])})
	}
	return true
}

// tailFrames 轮询 stderr.log 的新增内容,逐行解析帧日志;其余行原样记 WARN(那往往是运行时报错)。
func tailFrames(path string) {
	defer Recover("diag.tailFrames")
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	var pending []byte
	buf := make([]byte, 32<<10)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			pending = append(pending, buf[:n]...)
			for {
				i := indexByte(pending, '\n')
				if i < 0 {
					break
				}
				line := string(pending[:i])
				pending = pending[i+1:]
				if line != "" && !ParseFrameLine(line) {
					lv := LevelWarn
					// macOS 输入法框架在键盘事件(含自动化输入)时打的无害提示,不当警告
					if strings.Contains(line, "TSM AdjustCapsLockLEDForKeyTransitionHandling") {
						lv = LevelDebug
					}
					write(lv, "stderr", line, nil)
				}
			}
		}
		if err == io.EOF || n == 0 {
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func indexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

// stderrTailOf 读文件末尾最多 max 字节。
func stderrTailOf(path string, max int64) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return ""
	}
	off := st.Size() - max
	if off < 0 {
		off = 0
	}
	b := make([]byte, st.Size()-off)
	_, _ = f.ReadAt(b, off)
	return string(b)
}

func dup2(from, to int) error { return syscall.Dup2(from, to) }

// ───────── 资源采样 ─────────

// Sample 是一次资源采样。
type Sample struct {
	At         time.Time
	HeapAlloc  uint64 // 堆上存活对象
	HeapSys    uint64
	Goroutines int
	NumGC      uint32
	GCPause    time.Duration // 累计 GC 暂停
	RSS        uint64        // 当前常驻内存
	CPU        float64       // 进程 CPU%
}

const sampleKeep = 300 // 保留 5 分钟(每秒一次)

var (
	sampleMu sync.Mutex
	samples  []Sample
	started  atomic.Bool
)

// StartSampler 启动每秒一次的资源采样(重复调用无效)。
// 异常自动记 WARN:goroutine 数突增、堆在半分钟内增长超过 128 MB,每类一分钟至多一条。
func StartSampler() {
	if !started.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer Recover("diag.sampler")
		var lastWarnG, lastWarnH time.Time
		for {
			time.Sleep(time.Second)
			s := takeSample()
			sampleMu.Lock()
			samples = append(samples, s)
			if len(samples) > sampleKeep {
				samples = samples[len(samples)-sampleKeep:]
			}
			old := samples[0]
			if len(samples) > 30 {
				old = samples[len(samples)-30]
			}
			sampleMu.Unlock()
			if s.Goroutines > 500 && time.Since(lastWarnG) > time.Minute {
				lastWarnG = time.Now()
				write(LevelWarn, "monitor", "goroutine 数量偏多,可能泄漏", []any{"数量", s.Goroutines})
			}
			if s.HeapAlloc > old.HeapAlloc+128<<20 && time.Since(lastWarnH) > time.Minute {
				lastWarnH = time.Now()
				// HeapAlloc 含「已不可达、还没被 GC 收走」的垃圾:切换长会话时的瞬时峰值多半如此。
				// 先记 Info,过 15 秒再看:回落了就是瞬时峰值(Info),仍高居不下才是疑似泄漏(Warn)。
				write(LevelInfo, "monitor", "堆内存半分钟内增长超过 128 MB,观察是否回落", []any{"现在", fmtBytes(s.HeapAlloc), "之前", fmtBytes(old.HeapAlloc), "GC次数", s.NumGC})
				before, peak := old.HeapAlloc, s.HeapAlloc
				go func() {
					defer Recover("diag.heapwatch")
					time.Sleep(15 * time.Second)
					now := takeSample()
					if now.HeapAlloc > before+96<<20 {
						write(LevelWarn, "monitor", "堆内存涨上去后 15 秒仍未回落,疑似泄漏", []any{"峰值", fmtBytes(peak), "现在", fmtBytes(now.HeapAlloc), "之前", fmtBytes(before), "GC次数", now.NumGC})
						return
					}
					write(LevelInfo, "monitor", "堆内存已回落(瞬时峰值,非泄漏)", []any{"峰值", fmtBytes(peak), "现在", fmtBytes(now.HeapAlloc), "之前", fmtBytes(before), "GC次数", now.NumGC})
				}()
			}
		}
	}()
}

func takeSample() Sample {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	return Sample{
		At:         time.Now(),
		HeapAlloc:  ms.HeapAlloc,
		HeapSys:    ms.HeapSys,
		Goroutines: runtime.NumGoroutine(),
		NumGC:      ms.NumGC,
		GCPause:    time.Duration(ms.PauseTotalNs),
		RSS:        sampleProcessRSS(),
		CPU:        sampleProcessCPU(),
	}
}

// Samples 返回最近的采样(最旧的在前),诊断页画曲线用。
func Samples() []Sample {
	sampleMu.Lock()
	defer sampleMu.Unlock()
	return append([]Sample(nil), samples...)
}

func fmtBytes(n uint64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

// FormatBytes 把字节数写成人读的单位。
func FormatBytes(n uint64) string { return fmtBytes(n) }

// ───────── 主线程卡死看门狗 ─────────

// Watchdog 周期性地让界面主线程执行一个回执:超过 warn 没回执记 WARN,超过 dump 就把全部 goroutine 栈
// 落到 crashes/hang-*.txt 并写一条 hang 记录。ping 由调用方实现(通常是 mygo.RunOnMain),
// 它必须在主线程上执行完后调用 done。窗口空闲时也能判断,因为用的是主动探测而不是帧心跳。
func Watchdog(ping func(done func()), warn, dump time.Duration) {
	go func() {
		defer Recover("diag.watchdog")
		for {
			time.Sleep(500 * time.Millisecond)
			if !watchdogEnabled.Load() {
				continue
			}
			ch := make(chan struct{})
			sent := time.Now()
			go ping(func() { close(ch) })
			warned, dumped := false, false
			snaps := 0
			for {
				select {
				case <-ch:
					if warned {
						d := time.Since(sent)
						write(LevelWarn, "hang", "主线程恢复响应", []any{"停滞", d.Round(time.Millisecond)})
						RecordHang("recovered", d, "", "")
					}
					goto next
				case <-time.After(100 * time.Millisecond):
				}
				stall := time.Since(sent)
				if stall >= warn && !warned {
					warned = true
					stk := mainStackAppFrames(10)
					write(LevelWarn, "hang", "主线程停滞", []any{"已停滞", stall.Round(time.Millisecond)})
					RecordHang("stall", stall, stk, "")
					NewCounter("hang.warn").Inc()
				}
				// 停滞期间每 100 ms 抓一次主 goroutine 的调用栈摘要(最多 3 次):
				// 3 秒才落盘的完整栈抓不到 400~1000 ms 的短卡顿,而这类卡顿(切形态、开会话)正是最常见的。
				if stall >= warn && snaps < 3 {
					snaps++
					write(LevelWarn, "hang", "主线程停滞时的调用栈", []any{"已停滞", stall.Round(time.Millisecond), "栈", mainStackAppFrames(10)})
				}
				if stall >= dump && !dumped {
					dumped = true
					dumpHang(stall)
				}
			}
		next:
		}
	}()
}

// mainStackSummary 取主 goroutine(1 号,界面线程)当前调用栈的摘要:
// 由内到外最多 limit 帧,每帧「函数@文件:行」,用 " <- " 连接;去掉参数与包路径前缀,方便放进一行日志。
func mainStackSummary(limit int) string {
	buf := make([]byte, 256<<10)
	n := runtime.Stack(buf, true)
	text := string(buf[:n])
	// all=true 时第一块是「当前 goroutine」(看门狗自己),要按标题找 1 号块;块之间以空行分隔
	var block string
	for _, b := range strings.Split(text, "\n\n") {
		if strings.HasPrefix(b, "goroutine 1 [") {
			block = b
			break
		}
	}
	if block == "" {
		return ""
	}
	lines := strings.Split(block, "\n")
	var out []string
	for i := 1; i+1 < len(lines) && len(out) < limit; i++ {
		fn, loc := lines[i], lines[i+1]
		if strings.HasPrefix(fn, "\t") || !strings.HasPrefix(loc, "\t") {
			continue
		}
		if k := strings.LastIndex(fn, "("); k > 0 { // 去掉实参 (0x...)
			fn = fn[:k]
		}
		if k := strings.LastIndex(fn, "/"); k >= 0 {
			fn = fn[k+1:]
		}
		loc = strings.TrimSpace(loc)
		if k := strings.LastIndex(loc, "/"); k >= 0 {
			loc = loc[k+1:]
		}
		if k := strings.Index(loc, " +0x"); k > 0 {
			loc = loc[:k]
		}
		out = append(out, fn+"@"+loc)
		i++
	}
	return strings.Join(out, " <- ")
}

// mainStackAppFrames 与 mainStackSummary 同源,但给出「内层 inner 帧 + 其后所有本应用(app. / kit. / main.)帧」,
// 用于看清卡顿是从哪块界面代码进去的(纯库内层帧在深度上会把应用帧挤出摘要)。
func mainStackAppFrames(inner int) string {
	all := strings.Split(mainStackSummary(400), " <- ")
	var out []string
	for i, f := range all {
		if i < inner || strings.HasPrefix(f, "app.") || strings.HasPrefix(f, "kit.") || strings.HasPrefix(f, "main.") {
			out = append(out, f)
		}
	}
	return strings.Join(out, " <- ")
}

func dumpHang(stall time.Duration) {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	dir := CrashDir()
	_ = os.MkdirAll(dir, 0o755)
	name := fmt.Sprintf("hang-%s-%d.txt", time.Now().Format("20060102-150405"), os.Getpid())
	path := filepath.Join(dir, name)
	_ = os.WriteFile(path, buf[:n], 0o644)
	write(LevelError, "hang", "主线程长时间停滞,已保存全部 goroutine 栈", []any{"停滞", stall.Round(time.Millisecond), "文件", path})
	RecordHang("dump", stall, mainStackAppFrames(20), path)
	WriteCrash(CrashRecord{Kind: "hang", Message: fmt.Sprintf("主线程停滞 %s", stall.Round(time.Millisecond)), Extra: "goroutine 栈见 " + path})
}
