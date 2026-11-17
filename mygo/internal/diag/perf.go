package diag

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ───────── 面包屑 ─────────

var crumbs = newRing(64)

// Crumb 记一条面包屑:只进内存环形缓冲,不写文件;崩溃 / 卡死记录里会带上最近 64 条,
// 用来还原「出事前用户在做什么」。放在用户可感知的动作入口(切换会话、发送、打开文件……)。
func Crumb(format string, args ...any) {
	crumbs.add(timestamp(time.Now()) + " " + fmt.Sprintf(format, args...))
}

// Crumbs 返回最近的面包屑(最旧的在前)。
func Crumbs() []string { return crumbs.snapshot() }

// ───────── 慢操作计时与统计 ─────────

// DefaultSlow 是 Span 的默认慢阈值;超过即记 WARN。
const DefaultSlow = 50 * time.Millisecond

// buckets 是耗时直方图的上界(毫秒),用来估算 P95 而不保存全部样本。
var buckets = [...]float64{0.1, 0.25, 0.5, 1, 2, 4, 8, 16, 33, 66, 125, 250, 500, 1000, 2000, 5000}

// Stat 是一个命名操作的累计统计。
type Stat struct {
	Name  string
	Count int64
	Total time.Duration
	Max   time.Duration
	Slow  int64 // 超过阈值的次数
	hist  [len(buckets) + 1]int64
}

// Mean 平均耗时。
func (s Stat) Mean() time.Duration {
	if s.Count == 0 {
		return 0
	}
	return s.Total / time.Duration(s.Count)
}

// P50 由直方图估算的 50 分位耗时。
func (s Stat) P50() time.Duration { return s.pQuantile(0.50) }

// P95 由直方图估算的 95 分位耗时(取桶上界,偏保守)。
func (s Stat) P95() time.Duration { return s.pQuantile(0.95) }

// P99 由直方图估算的 99 分位耗时。
func (s Stat) P99() time.Duration { return s.pQuantile(0.99) }

func (s Stat) pQuantile(q float64) time.Duration {
	if s.Count == 0 {
		return 0
	}
	want := int64(math.Ceil(float64(s.Count) * q))
	var acc int64
	for i, n := range s.hist {
		acc += n
		if acc >= want {
			if i < len(buckets) {
				return time.Duration(buckets[i] * float64(time.Millisecond))
			}
			return s.Max
		}
	}
	return s.Max
}

var (
	statsMu sync.Mutex
	stats   = map[string]*Stat{}
)

func record(name string, d time.Duration, slow time.Duration) {
	ms := float64(d) / float64(time.Millisecond)
	idx := len(buckets)
	for i, b := range buckets {
		if ms <= b {
			idx = i
			break
		}
	}
	statsMu.Lock()
	s := stats[name]
	if s == nil {
		s = &Stat{Name: name}
		stats[name] = s
	}
	s.Count++
	s.Total += d
	if d > s.Max {
		s.Max = d
	}
	if d > slow {
		s.Slow++
	}
	s.hist[idx]++
	statsMu.Unlock()
}

// Span 开始计时一个操作,返回结束函数:`defer diag.Span("session", "apply")()`。
// 超过 DefaultSlow 记 WARN(带键值),所有耗时进入统计表供诊断页与退出汇总使用。
func Span(module, name string, kv ...any) func() {
	return SpanSlow(module, name, DefaultSlow, kv...)
}

// SpanSlow 同 Span,但自定义慢阈值(渲染类操作用更小的值)。
func SpanSlow(module, name string, slow time.Duration, kv ...any) func() {
	start := time.Now()
	return func() {
		d := time.Since(start)
		record(module+"."+name, d, slow)
		if d > slow {
			write(LevelWarn, module, "慢操作 "+name, append([]any{"耗时", d}, kv...))
		}
	}
}

// Time 对一个函数计时并返回它的结果耗时,不做慢日志(只入统计)。
func Time(module, name string, fn func()) time.Duration {
	start := time.Now()
	fn()
	d := time.Since(start)
	record(module+"."+name, d, time.Hour)
	return d
}

// Stats 返回统计快照,按总耗时从大到小排序。
func Stats() []Stat {
	statsMu.Lock()
	out := make([]Stat, 0, len(stats))
	for _, s := range stats {
		out = append(out, *s)
	}
	statsMu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Total > out[j].Total })
	return out
}

// ResetStats 清空统计(测试用)。
func ResetStats() {
	statsMu.Lock()
	stats = map[string]*Stat{}
	statsMu.Unlock()
}

// ───────── 计数指标 ─────────

// Counter 是一个原子计数器,命名后进入全局表。
type Counter struct {
	name string
	n    atomic.Int64
}

var (
	counterMu sync.Mutex
	counters  = map[string]*Counter{}
)

// NewCounter 取得(必要时创建)同名计数器。
func NewCounter(name string) *Counter {
	counterMu.Lock()
	defer counterMu.Unlock()
	c := counters[name]
	if c == nil {
		c = &Counter{name: name}
		counters[name] = c
	}
	return c
}

func (c *Counter) Add(n int64) { c.n.Add(n) }
func (c *Counter) Inc()        { c.n.Add(1) }
func (c *Counter) Load() int64 { return c.n.Load() }

// Counters 返回全部计数器当前值。
func Counters() map[string]int64 {
	counterMu.Lock()
	defer counterMu.Unlock()
	out := make(map[string]int64, len(counters))
	for k, c := range counters {
		out[k] = c.Load()
	}
	return out
}

// SummaryLines 把统计与计数器格式化成多行文本(退出汇总、诊断包)。
func SummaryLines() []string {
	var out []string
	for _, s := range Stats() {
		out = append(out, fmt.Sprintf("%-32s 次数=%d 平均=%s P95=%s 最大=%s 慢=%d", s.Name, s.Count, s.Mean().Round(time.Microsecond), s.P95(), s.Max.Round(time.Microsecond), s.Slow))
	}
	cs := Counters()
	keys := make([]string, 0, len(cs))
	for k := range cs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%-32s %d", k, cs[k]))
	}
	return out
}
