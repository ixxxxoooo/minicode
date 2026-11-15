// Package automation 是「自动化」的计划、存储与运行记录(与界面无关)。
// 计划用一个人话字段 Days 表示(每天 / 工作日 / 周二 / 每月 5 日),界面的「每天 / 每周 / 每月 / Cron」分段由它推出。
// 真正的运行器接上 Claude 后端后(t8)再挂到 RunNow;现在 RunNow 只落一条运行记录。
package automation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Kind 是计划的种类(界面分段)。
type Kind int

const (
	KDaily Kind = iota
	KWeekly
	KMonthly
	KCron
)

// Run 是一次运行记录。
type Run struct {
	At     time.Time
	OK     bool
	Err    string // 失败原因,如「网络超时」
	Output string // 运行输出的末尾摘要(真实运行才有)
	Key    string // 查看键:auto:<自动化id>:<unix>,空则按 At 生成
}

// Automation 是一条自动化。
type Automation struct {
	ID      int
	Name    string
	Prompt  string
	Days    string // 每天 / 工作日 / 周一…周日 / 每月 N 日 / 一条 cron 表达式
	Time    string // HH:MM
	Project string
	Perm    string // 权限说明,如「只读 · 计划模式」
	On      bool
	Runs    []Run // 新的在前
}

var weekdays = []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}

// Weekdays 返回周一到周日的名字。
func Weekdays() []string { return append([]string(nil), weekdays...) }

// KindOf 由 Days 推出种类。
func KindOf(days string) Kind {
	switch {
	case days == "每天" || days == "工作日":
		return KDaily
	case strings.HasPrefix(days, "周") && len([]rune(days)) == 2:
		return KWeekly
	case strings.HasPrefix(days, "每月"):
		return KMonthly
	}
	return KCron
}

// Schedule 是列表里的计划说明,如「每周二 · 14:00」「工作日 · 09:00」。
func (a Automation) Schedule() string {
	if KindOf(a.Days) == KCron {
		return "Cron " + a.Days
	}
	d := a.Days
	if KindOf(d) == KWeekly {
		d = "每" + d
	}
	return d + " · " + a.Time
}

// FiresOn 这条自动化在 d 这天是否会运行(只看日期;已暂停的不运行;cron 不展开)。
func (a Automation) FiresOn(d time.Time) bool {
	if !a.On {
		return false
	}
	wd := int(d.Weekday()+6) % 7 // 周一 = 0
	switch KindOf(a.Days) {
	case KDaily:
		return a.Days == "每天" || wd < 5
	case KWeekly:
		return weekdays[wd] == a.Days
	case KMonthly:
		n, _ := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(a.Days, "每月")), "日")))
		return n == d.Day()
	case KCron:
		return cronFiresOn(a.Days, d)
	}
	return false
}

// ValidTime 判断是否是 HH:MM。
func ValidTime(s string) bool {
	h, m, ok := strings.Cut(s, ":")
	if !ok || len(h) != 2 || len(m) != 2 {
		return false
	}
	hh, e1 := strconv.Atoi(h)
	mm, e2 := strconv.Atoi(m)
	return e1 == nil && e2 == nil && hh >= 0 && hh < 24 && mm >= 0 && mm < 60
}

// Last 是列表卡片右下角的「最近一次」说明与状态:1 成功 / 2 失败 / 0 无(已暂停或没跑过)。
func (a Automation) Last(now time.Time) (string, int) {
	if !a.On {
		return "已暂停", 0
	}
	if len(a.Runs) == 0 {
		return "还没运行过", 0
	}
	r := a.Runs[0]
	if r.OK {
		return When(now, r.At) + " · 成功", 1
	}
	txt := When(now, r.At) + " · 失败"
	if r.Err != "" {
		txt += ":" + r.Err
	}
	return txt, 2
}

// When 把时间说成相对当前的人话:今天 09:00 / 昨天 09:00 / 周五 09:00 / 上周二 14:00 / 11 月 2 日。
func When(now, t time.Time) string {
	day := func(x time.Time) time.Time { return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, time.UTC) }
	diff := int(day(now).Sub(day(t)).Hours() / 24)
	hm := fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
	wd := weekdays[int(t.Weekday()+6)%7]
	nowOff := int(now.Weekday()+6) % 7 // 本周已过去几天
	switch {
	case diff == 0:
		return "今天 " + hm
	case diff == 1:
		return "昨天 " + hm
	case diff > 1 && diff <= nowOff:
		return wd + " " + hm
	case diff > nowOff && diff <= nowOff+7:
		return "上" + wd + " " + hm
	}
	return fmt.Sprintf("%d 月 %d 日", int(t.Month()), t.Day())
}

// Slot 是周历里某天的一个时间块。
type Slot struct {
	ID   int
	Time string
	Name string
	Err  bool // 当天这条自动化跑失败了
	Off  bool // 已暂停(划线)
}

// Week 算出 monday 起 7 天的周历:每天按时间排序;暂停的自动化只在 from(通常是今天)及以后显示成划线,
// 提示「恢复后会在这天运行」,过去的日子它没有运行过,不显示。
func Week(items []Automation, monday, from time.Time) [7][]Slot {
	fromDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	var out [7][]Slot
	for i := 0; i < 7; i++ {
		d := monday.AddDate(0, 0, i)
		for _, a := range items {
			probe := a
			probe.On = true
			if !probe.FiresOn(d) || (!a.On && d.Before(fromDay)) {
				continue
			}
			tm := a.Time
			if KindOf(a.Days) == KCron {
				tm = cronSlotTime(a.Days, d)
			}
			s := Slot{ID: a.ID, Time: tm, Name: a.Name, Off: !a.On}
			for _, r := range a.Runs {
				if sameDay(r.At, d) && !r.OK {
					s.Err = true
				}
			}
			out[i] = append(out[i], s)
		}
		sortSlots(out[i])
	}
	return out
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

func sortSlots(s []Slot) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j].Time < s[j-1].Time; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// MondayOf 返回 t 所在周的周一 0 点(UTC 日期,时区无关)。
func MondayOf(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return d.AddDate(0, 0, -(int(t.Weekday()+6) % 7))
}

// ───────── 存储 ─────────

// Store 是内存里的自动化清单(并发安全)。落盘与运行器后续接入。
type Store struct {
	mu    sync.Mutex
	items []Automation
	next  int
	path  string // 非空:每次修改都原子写盘
}

// OpenStore 读取 path 里的自动化清单(不存在 = 空),之后的每次修改自动落盘。
func OpenStore(path string) *Store {
	s := &Store{path: path}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &s.items)
	}
	for _, a := range s.items {
		s.next = max(s.next, a.ID)
	}
	return s
}

// save 在持锁状态下写盘(临时文件 + 改名);失败只丢掉这次持久化,不影响内存里的数据。
func (s *Store) save() {
	if s.path == "" {
		return
	}
	b, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.path), 0o755)
	tmp := s.path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, s.path)
	}
}

// Record 给某条自动化追加一条运行记录(新的在前,最多留 20 条)。
func (s *Store) Record(id int, r Run) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			runs := append([]Run{r}, s.items[i].Runs...)
			if len(runs) > 20 {
				runs = runs[:20]
			}
			s.items[i].Runs = runs
		}
	}
	s.save()
}

// Due 返回在 (from, now] 这段时间里到点、且今天还没跑过的自动化。
func (s *Store) Due(from, now time.Time) []Automation {
	var out []Automation
	for _, a := range s.List() {
		if !a.On {
			continue
		}
		if KindOf(a.Days) == KCron {
			if _, err := ParseCron(a.Days); err != nil {
				continue
			}
			for _, at := range cronFireTimesBetween(a.Days, from, now) {
				if firedAlready(a.Runs, at) {
					continue
				}
				out = append(out, a)
				break
			}
			continue
		}
		if !a.FiresOn(now) || !ValidTime(a.Time) {
			continue
		}
		h, _ := strconv.Atoi(a.Time[:2])
		m, _ := strconv.Atoi(a.Time[3:])
		at := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, now.Location())
		if at.After(from) && !at.After(now) && !firedAlready(a.Runs, at) {
			out = append(out, a)
		}
	}
	return out
}

func firedAlready(runs []Run, at time.Time) bool {
	for _, r := range runs {
		if r.At.Truncate(time.Minute).Equal(at.Truncate(time.Minute)) {
			return true
		}
	}
	return false
}

// NewStore 用给定清单建一个存储。
func NewStore(items []Automation) *Store {
	s := &Store{items: append([]Automation(nil), items...)}
	for _, a := range items {
		s.next = max(s.next, a.ID)
	}
	return s
}

// List 返回副本。
func (s *Store) List() []Automation {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Automation, len(s.items))
	for i, a := range s.items {
		a.Runs = append([]Run(nil), a.Runs...)
		out[i] = a
	}
	return out
}

// Get 按 id 取一条。
func (s *Store) Get(id int) (Automation, bool) {
	for _, a := range s.List() {
		if a.ID == id {
			return a, true
		}
	}
	return Automation{}, false
}

// Add 新建一条默认的自动化并返回它。
func (s *Store) Add(project string) Automation {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	a := Automation{ID: s.next, Name: "新的自动化", Days: "每天", Time: "09:00", Project: project, Perm: "只读 · 计划模式", On: true}
	s.items = append(s.items, a)
	s.save()
	return a
}

// Update 整条替换(按 ID);保存前先校验时间格式与名称。
func (s *Store) Update(a Automation) error {
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("名称不能为空")
	}
	if KindOf(a.Days) == KCron {
		if _, err := ParseCron(a.Days); err != nil {
			return fmt.Errorf("cron 表达式不合法:%v", err)
		}
	} else if !ValidTime(a.Time) {
		return fmt.Errorf("时间要写成 HH:MM,如 09:00")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == a.ID {
			a.Runs = s.items[i].Runs // 运行记录不随编辑改动
			s.items[i] = a
			s.save()
			return nil
		}
	}
	return fmt.Errorf("找不到这条自动化")
}

// Delete 删除。
func (s *Store) Delete(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items = append(s.items[:i:i], s.items[i+1:]...)
			s.save()
			return
		}
	}
}

// Toggle 启停。
func (s *Store) Toggle(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].On = !s.items[i].On
		}
	}
	s.save()
}

// RunNow 立即运行一次:现在只记一条成功的运行记录(运行器接上后换成真正的会话)。
func (s *Store) RunNow(id int, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Runs = append([]Run{{At: now, OK: true}}, s.items[i].Runs...)
		}
	}
	s.save()
}
