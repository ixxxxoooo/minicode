package automation

import (
	"testing"
	"time"
)

func TestCronMatch(t *testing.T) {
	c, err := ParseCron("0 9 * * 1")
	if err != nil {
		t.Fatal(err)
	}
	mon := time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC) // 周一
	if !c.Match(mon) {
		t.Fatal("周一 9:00 应命中")
	}
	if c.Match(mon.AddDate(0, 0, 1)) {
		t.Fatal("周二不应命中")
	}
	c2, _ := ParseCron("*/15 8-17 * * 1-5")
	wd := time.Date(2026, 11, 17, 10, 30, 0, 0, time.UTC) // 周二
	if !c2.Match(wd) {
		t.Fatal("工作日 10:30 每 15 分应命中")
	}
	if c2.Match(time.Date(2026, 11, 15, 10, 30, 0, 0, time.UTC)) {
		t.Fatal("周日不应命中")
	}
}

func TestCronFiresOn(t *testing.T) {
	a := Automation{On: true, Days: "0 14 * * 2", Time: "14:00"}
	tue := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	if !a.FiresOn(tue) {
		t.Fatal("周二 cron 应在周二触发")
	}
	wed := tue.AddDate(0, 0, 1)
	if a.FiresOn(wed) {
		t.Fatal("周三不应")
	}
}

func TestCronDue(t *testing.T) {
	s := NewStore([]Automation{{ID: 1, Name: "c", Days: "0 9 * * *", On: true, Time: "09:00"}})
	day := time.Date(2026, 11, 16, 0, 0, 0, 0, time.Local)
	if due := s.Due(day.Add(8*time.Hour+59*time.Minute), day.Add(9*time.Hour+2*time.Minute)); len(due) != 1 {
		t.Fatalf("cron 9点应到点:%d", len(due))
	}
}
