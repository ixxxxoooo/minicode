package automation

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CronExpr 解析 5 段标准 cron(分 时 日 月 周):支持 *、*/n、范围 a-b、列表 a,b。
type CronExpr struct {
	min, hour, dom, mon, dow field
	raw                      string
}

type field struct {
	all    bool
	step   int // */n 的 n,0 表示无步进
	values map[int]bool
}

// ParseCron 解析表达式;非法时返回错误。
func ParseCron(expr string) (CronExpr, error) {
	parts := strings.Fields(strings.TrimSpace(expr))
	if len(parts) != 5 {
		return CronExpr{}, fmt.Errorf("cron 需要 5 段(分 时 日 月 周)")
	}
	var c CronExpr
	c.raw = strings.Join(parts, " ")
	var err error
	if c.min, err = parseField(parts[0], 0, 59); err != nil {
		return CronExpr{}, fmt.Errorf("分: %w", err)
	}
	if c.hour, err = parseField(parts[1], 0, 23); err != nil {
		return CronExpr{}, fmt.Errorf("时: %w", err)
	}
	if c.dom, err = parseField(parts[2], 1, 31); err != nil {
		return CronExpr{}, fmt.Errorf("日: %w", err)
	}
	if c.mon, err = parseField(parts[3], 1, 12); err != nil {
		return CronExpr{}, fmt.Errorf("月: %w", err)
	}
	if c.dow, err = parseField(parts[4], 0, 7); err != nil {
		return CronExpr{}, fmt.Errorf("周: %w", err)
	}
	// 7 与 0 都表示周日
	if c.dow.values[7] {
		c.dow.values[0] = true
		delete(c.dow.values, 7)
	}
	return c, nil
}

func parseField(s string, lo, hi int) (field, error) {
	s = strings.TrimSpace(s)
	f := field{values: map[int]bool{}}
	if s == "*" {
		f.all = true
		return f, nil
	}
	var step int
	if i := strings.Index(s, "/"); i >= 0 {
		base, st := s[:i], s[i+1:]
		n, err := strconv.Atoi(st)
		if err != nil || n < 1 {
			return field{}, fmt.Errorf("步进 %q", st)
		}
		step = n
		if base == "*" {
			f.all, f.step = true, step
			return f, nil
		}
		s = base
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if j := strings.Index(part, "-"); j >= 0 {
			a, err1 := strconv.Atoi(strings.TrimSpace(part[:j]))
			b, err2 := strconv.Atoi(strings.TrimSpace(part[j+1:]))
			if err1 != nil || err2 != nil || a < lo || b > hi || a > b {
				return field{}, fmt.Errorf("范围 %q", part)
			}
			for v := a; v <= b; v++ {
				f.values[v] = true
			}
			continue
		}
		v, err := strconv.Atoi(part)
		if err != nil || v < lo || v > hi {
			return field{}, fmt.Errorf("值 %q", part)
		}
		f.values[v] = true
	}
	if len(f.values) == 0 {
		return field{}, fmt.Errorf("空字段")
	}
	f.step = step
	return f, nil
}

func (f field) ok(v int) bool {
	if f.all {
		if f.step > 1 {
			return v%f.step == 0
		}
		return true
	}
	if !f.values[v] {
		return false
	}
	if f.step > 1 {
		return v%f.step == 0
	}
	return true
}

// Match 时刻 t 是否命中(按 t 的本地年月日时分)。
func (c CronExpr) Match(t time.Time) bool {
	return c.min.ok(t.Minute()) && c.hour.ok(t.Hour()) && c.dom.ok(t.Day()) && c.mon.ok(int(t.Month())) && c.dow.ok(int(t.Weekday()))
}

// NextAfter 返回严格晚于 from 的下一次触发(最多向前搜 366 天)。
func (c CronExpr) NextAfter(from time.Time) (time.Time, bool) {
	t := from.Truncate(time.Minute).Add(time.Minute)
	deadline := from.Add(366 * 24 * time.Hour)
	for t.Before(deadline) {
		if c.Match(t) {
			return t, true
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, false
}

// cronFiresOn 某天是否会按 cron 运行(忽略 Time 字段)。
func cronFiresOn(expr string, d time.Time) bool {
	c, err := ParseCron(expr)
	if err != nil {
		return false
	}
	day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
	for h := 0; h < 24; h++ {
		for m := 0; m < 60; m++ {
			t := day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
			if c.Match(t) {
				return true
			}
		}
	}
	return false
}

// cronFireTimesBetween 返回 (from, now] 内每次触发(升序)。
func cronFireTimesBetween(expr string, from, now time.Time) []time.Time {
	c, err := ParseCron(expr)
	if err != nil {
		return nil
	}
	start := from.Truncate(time.Minute).Add(time.Minute)
	var out []time.Time
	t := start
	for !t.After(now) {
		if c.Match(t) {
			out = append(out, t)
		}
		t = t.Add(time.Minute)
		if len(out) > 500 {
			break
		}
	}
	return out
}

// NextRun 估算下次运行时间(用于界面展示);非 cron 用 Days+Time;cron 用表达式。
func (a Automation) NextRun(from time.Time) (time.Time, bool) {
	if !a.On {
		return time.Time{}, false
	}
	loc := from.Location()
	switch KindOf(a.Days) {
	case KCron:
		c, err := ParseCron(a.Days)
		if err != nil {
			return time.Time{}, false
		}
		return c.NextAfter(from.In(loc))
	default:
		if !ValidTime(a.Time) {
			return time.Time{}, false
		}
		h, _ := strconv.Atoi(a.Time[:2])
		m, _ := strconv.Atoi(a.Time[3:])
		day := time.Date(from.Year(), from.Month(), from.Day(), h, m, 0, 0, loc)
		if !day.After(from) {
			day = day.Add(24 * time.Hour)
		}
		for i := 0; i < 370; i++ {
			if a.FiresOn(day) {
				return day, true
			}
			day = day.Add(24 * time.Hour)
		}
	}
	return time.Time{}, false
}

// cronSlotTime 周历里 cron 条目的显示时间:取当天第一次触发,没有则返回表达式里第一个小时:分。
func cronSlotTime(expr string, d time.Time) string {
	c, err := ParseCron(expr)
	if err != nil {
		return "?"
	}
	day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
	for h := 0; h < 24; h++ {
		for m := 0; m < 60; m++ {
			t := day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
			if c.Match(t) {
				return fmt.Sprintf("%02d:%02d", h, m)
			}
		}
	}
	return strings.Fields(expr)[1] + ":" + strings.Fields(expr)[0]
}
