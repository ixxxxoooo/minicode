// Package diag 是整个应用的诊断底座:文件日志、面包屑、性能计时、崩溃与卡死记录、资源采样。
//
// 它位于依赖树最底层,不依赖界面与 mygo,所有其它包都只通过它记日志(不直接用 fmt.Print / log)。
// 设计目标:出问题时先看 ~/.minicode/logs/minicode.log 就能定位——
//   - 每行 `时间 级别 [模块] 消息 键=值…`,可 grep;
//   - 慢操作、慢帧、主线程卡死、goroutine 泄漏、内存陡增都会自动落一条 WARN;
//   - panic / 致命错误 / 异常退出 都有独立的崩溃记录文件,带面包屑与日志尾部。
package diag

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Level 日志级别,数值越大越重要。
type Level int32

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelOff
)

// String 返回固定宽度的级别名,便于对齐与 grep。
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO "
	case LevelWarn:
		return "WARN "
	case LevelError:
		return "ERROR"
	}
	return "OFF  "
}

// ParseLevel 解析 off / error / warn / info / debug(不区分大小写),无法识别时返回 def。
func ParseLevel(s string, def Level) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "off":
		return LevelOff
	}
	return def
}

const (
	// MaxLogBytes 单个日志文件的上限,超过即轮转。
	MaxLogBytes = 5 << 20
	// LogFiles 保留的日志文件份数(含当前)。
	LogFiles = 5
)

// Home 返回应用数据目录:环境变量 MINI_HOME 优先(测试与多实例隔离用),否则 ~/.minicode。
func Home() string {
	if h := os.Getenv("MINI_HOME"); h != "" {
		return h
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".minicode")
	}
	return filepath.Join(os.TempDir(), "minicode")
}

// rotator 是带大小轮转的日志文件,句柄预先打开,写入走互斥锁。
type rotator struct {
	mu   sync.Mutex
	path string
	f    *os.File
	size int64
}

func openRotator(path string) (*rotator, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	r := &rotator{path: path}
	if err := r.open(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *rotator) open() error {
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	st, _ := f.Stat()
	if st != nil {
		r.size = st.Size()
	}
	r.f = f
	return nil
}

// rotate 把 log → log.1 → log.2 … 依次后移,最老的丢弃。
func (r *rotator) rotate() {
	_ = r.f.Close()
	for i := LogFiles - 1; i >= 1; i-- {
		from := r.path
		if i > 1 {
			from = fmt.Sprintf("%s.%d", r.path, i-1)
		}
		_ = os.Rename(from, fmt.Sprintf("%s.%d", r.path, i))
	}
	_ = r.open()
}

func (r *rotator) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return len(p), nil
	}
	if r.size+int64(len(p)) > MaxLogBytes {
		r.rotate()
		if r.f == nil {
			return len(p), nil
		}
	}
	n, err := r.f.Write(p)
	r.size += int64(n)
	return n, err
}

// ───────── 全局状态 ─────────

var (
	level   atomic.Int32
	sinkMu  sync.RWMutex
	sink    io.Writer = io.Discard
	logFile *rotator
	// mirror 为真时同时写一份到标准输出(开发时直接在终端看)。
	mirror atomic.Bool
	// 日志尾部环形缓冲,崩溃记录里带上最近的日志。
	tail = newRing(200)
)

func init() { level.Store(int32(LevelInfo)) }

// SetLevel 运行时调整日志级别(设置 → 隐私与诊断)。
func SetLevel(l Level) { level.Store(int32(l)) }

// CurrentLevel 返回当前级别。
func CurrentLevel() Level { return Level(level.Load()) }

// LogPath 返回当前日志文件路径。
func LogPath() string { return filepath.Join(Home(), "logs", "minicode.log") }

// openLog 打开日志文件;失败时退回到只写尾部缓冲(不影响应用运行)。
func openLog(path string) {
	r, err := openRotator(path)
	sinkMu.Lock()
	defer sinkMu.Unlock()
	if err != nil {
		sink = io.Discard
		return
	}
	logFile = r
	sink = r
}

// timestamp 写成 `2006-01-02 15:04:05.000`(本地时区),不走 time.Format 的反射路径以减少分配。
func timestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

// kvString 把键值对写成 `k=v k2=v2`;含空格 / 引号 / 换行的值加引号;奇数个参数最后一个当作 `!BADKEY`。
func kvString(kv []any) string {
	if len(kv) == 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < len(kv); i += 2 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		key := fmt.Sprint(kv[i])
		if i+1 >= len(kv) {
			b.WriteString("!BADKEY=" + quoteIfNeeded(key))
			break
		}
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(quoteIfNeeded(valueString(kv[i+1])))
	}
	return b.String()
}

func valueString(v any) string {
	switch x := v.(type) {
	case time.Duration:
		return x.Round(10 * time.Microsecond).String()
	case error:
		if x == nil {
			return "<nil>"
		}
		return x.Error()
	case string:
		return x
	}
	return fmt.Sprint(v)
}

func quoteIfNeeded(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, " \t\n\"=") {
		return fmt.Sprintf("%q", s)
	}
	return s
}

// write 是所有日志的出口。
func write(l Level, module, msg string, kv []any) {
	if l < Level(level.Load()) {
		return
	}
	line := timestamp(time.Now()) + " " + l.String() + " [" + module + "] " + msg
	if s := kvString(kv); s != "" {
		line += " " + s
	}
	tail.add(line)
	sinkMu.RLock()
	w := sink
	sinkMu.RUnlock()
	_, _ = io.WriteString(w, line+"\n")
	if mirror.Load() {
		_, _ = io.WriteString(os.Stdout, line+"\n")
	}
}

// Logger 是绑定了模块名的日志器,每个包在 var 里建一个:`var log = diag.L("backend")`。
type Logger struct{ module string }

// L 返回模块日志器。
func L(module string) Logger { return Logger{module} }

func (l Logger) Debug(msg string, kv ...any) { write(LevelDebug, l.module, msg, kv) }
func (l Logger) Info(msg string, kv ...any)  { write(LevelInfo, l.module, msg, kv) }
func (l Logger) Warn(msg string, kv ...any)  { write(LevelWarn, l.module, msg, kv) }
func (l Logger) Error(msg string, kv ...any) { write(LevelError, l.module, msg, kv) }

// Enabled 报告某级别是否会被记录,用于避免为被丢弃的日志构造昂贵的参数。
func (l Logger) Enabled(lv Level) bool { return lv >= Level(level.Load()) }

// Tail 返回最近的日志行(最旧的在前),崩溃记录与诊断包使用。
func Tail() []string { return tail.snapshot() }

// ring 固定容量的字符串环形缓冲。
type ring struct {
	mu   sync.Mutex
	buf  []string
	next int
	full bool
}

func newRing(n int) *ring { return &ring{buf: make([]string, n)} }

func (r *ring) add(s string) {
	r.mu.Lock()
	r.buf[r.next] = s
	r.next = (r.next + 1) % len(r.buf)
	if r.next == 0 {
		r.full = true
	}
	r.mu.Unlock()
}

func (r *ring) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.full {
		return append([]string(nil), r.buf[:r.next]...)
	}
	out := make([]string, 0, len(r.buf))
	out = append(out, r.buf[r.next:]...)
	out = append(out, r.buf[:r.next]...)
	return out
}
