package diag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"syscall"
	"time"
)

// CrashRecord 是写进 ~/.minicode/crashes/*.json 的一条崩溃 / 卡死 / 异常退出记录。
type CrashRecord struct {
	Time     string   `json:"time"`
	PID      int      `json:"pid"`
	Kind     string   `json:"kind"` // panic / hang / unclean-exit / fatal
	Where    string   `json:"where,omitempty"`
	Message  string   `json:"message"`
	Version  string   `json:"version,omitempty"`
	Mode     string   `json:"mode,omitempty"`
	Stack    string   `json:"stack,omitempty"`
	Crumbs   []string `json:"crumbs,omitempty"`
	LogTail  []string `json:"log_tail,omitempty"`
	Extra    string   `json:"extra,omitempty"`
	Goroutes int      `json:"goroutines,omitempty"`
}

var crashSeq atomic.Int64

// CrashDir 崩溃记录目录。
func CrashDir() string { return filepath.Join(Home(), "crashes") }

// RunDir 运行标记目录。
func RunDir() string { return filepath.Join(Home(), "run") }

// WriteCrash 写一条崩溃记录文件并返回路径;写失败只记日志,不再抛出。
func WriteCrash(rec CrashRecord) string {
	rec.Time = time.Now().Format(time.RFC3339)
	rec.PID = os.Getpid()
	rec.Version, rec.Mode = meta.Version, meta.Mode
	if rec.Crumbs == nil {
		rec.Crumbs = Crumbs()
	}
	if rec.LogTail == nil {
		rec.LogTail = Tail()
	}
	rec.Goroutes = runtime.NumGoroutine()
	dir := CrashDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		write(LevelError, "diag", "无法创建崩溃目录", []any{"error", err})
		return ""
	}
	name := fmt.Sprintf("%s-%02d-%s-%d.json", time.Now().Format("20060102-150405"), crashSeq.Add(1), rec.Kind, rec.PID)
	path := filepath.Join(dir, name)
	data, _ := json.MarshalIndent(rec, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		write(LevelError, "diag", "写崩溃记录失败", []any{"error", err})
		return ""
	}
	pruneCrashes(dir, 50)
	return path
}

// pruneCrashes 只保留最新的 keep 份记录,避免崩溃循环把磁盘写满。
func pruneCrashes(dir string, keep int) {
	ents, err := os.ReadDir(dir)
	if err != nil || len(ents) <= keep {
		return
	}
	// 文件名以时间开头,字典序即时间序
	for _, e := range ents[:len(ents)-keep] {
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
}

// Recover 放在 goroutine 入口的 defer 里:捕获 panic,记 ERROR 并落崩溃记录,goroutine 安静结束而不拖垮整个应用。
//
//	go func() { defer diag.Recover("backend.read"); ... }()
func Recover(where string) {
	if v := recover(); v != nil {
		handlePanic(where, v)
	}
}

// RecoverWith 同 Recover,并把捕获到的值交给 fn(例如把错误显示在界面上)。
func RecoverWith(where string, fn func(v any)) {
	if v := recover(); v != nil {
		handlePanic(where, v)
		if fn != nil {
			fn(v)
		}
	}
}

// Guard 在保护下执行 fn,panic 被记录后返回 false。界面构建等「出错也不能崩」的位置使用。
func Guard(where string, fn func()) (ok bool) {
	defer func() {
		if v := recover(); v != nil {
			handlePanic(where, v)
			ok = false
		}
	}()
	fn()
	return true
}

// Report 记录一次 panic(日志 + 崩溃记录,含调用栈)但不拦截:调用方随后应原样 panic。
// 用在「吞掉会让状态错乱」的位置(如界面构建:控件树做到一半),保证崩溃前留下线索。
// 必须在 defer 的函数里调用,这样调用栈里才有 panic 现场。
func Report(where string, v any) { handlePanic(where, v) }

func handlePanic(where string, v any) {
	stack := string(debug.Stack())
	msg := fmt.Sprint(v)
	write(LevelError, "crash", "panic 已捕获", []any{"位置", where, "信息", msg})
	path := WriteCrash(CrashRecord{Kind: "panic", Where: where, Message: msg, Stack: stack})
	if path != "" {
		write(LevelError, "crash", "崩溃记录已写入", []any{"文件", path})
	}
	NewCounter("panic.recovered").Inc()
}

// ───────── 运行标记:检测异常退出 ─────────

type runMarker struct {
	PID       int    `json:"pid"`
	Started   string `json:"started"`
	Heartbeat string `json:"heartbeat"`
	Version   string `json:"version"`
	Mode      string `json:"mode"`
}

func markerPath(pid int) string { return filepath.Join(RunDir(), fmt.Sprintf("%d.json", pid)) }

func writeMarker(started time.Time) {
	m := runMarker{PID: os.Getpid(), Started: started.Format(time.RFC3339), Heartbeat: time.Now().Format(time.RFC3339), Version: meta.Version, Mode: meta.Mode}
	data, _ := json.Marshal(m)
	_ = os.MkdirAll(RunDir(), 0o755)
	_ = os.WriteFile(markerPath(os.Getpid()), data, 0o644)
}

// processAlive 用 kill(pid, 0) 判断进程是否还活着。
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// ScanStale 在启动时扫描上次遗留的运行标记:对应进程已不在,又没有走正常退出,就补一条「异常退出」记录。
// 返回补记的条数。
func ScanStale() int {
	ents, err := os.ReadDir(RunDir())
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range ents {
		path := filepath.Join(RunDir(), e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m runMarker
		if json.Unmarshal(data, &m) != nil || m.PID == os.Getpid() || processAlive(m.PID) {
			continue
		}
		extra := stderrTailOf(filepath.Join(Home(), "logs", "stderr.log.prev"), 8<<10)
		write(LevelError, "crash", "上次运行异常退出(没有走正常退出)", []any{"pid", m.PID, "启动", m.Started, "最后心跳", m.Heartbeat, "版本", m.Version})
		WriteCrash(CrashRecord{Kind: "unclean-exit", Message: fmt.Sprintf("pid %d 没有正常退出,最后心跳 %s", m.PID, m.Heartbeat), Extra: extra, Crumbs: []string{}, LogTail: []string{}})
		_ = os.Remove(path)
		n++
	}
	return n
}

// CleanExit 标记正常退出:删除运行标记。所有退出路径(⌘Q、托盘退出、更新重启、切换模式)必须在退出前调用,
// 否则下次启动会被误判为崩溃。
func CleanExit(reason string) {
	write(LevelInfo, "diag", "正常退出", []any{"原因", reason})
	for _, l := range SummaryLines() {
		write(LevelDebug, "perf", l, nil)
	}
	_ = os.Remove(markerPath(os.Getpid()))
}
