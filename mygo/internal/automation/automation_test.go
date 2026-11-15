package automation

import (
	"testing"
	"time"
)

var sun = time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) // 周日

func TestKindAndSchedule(t *testing.T) {
	cases := map[string]Kind{"每天": KDaily, "工作日": KDaily, "周二": KWeekly, "每月 5 日": KMonthly, "0 9 * * 1": KCron}
	for d, k := range cases {
		if KindOf(d) != k {
			t.Errorf("%q 种类应为 %d", d, k)
		}
	}
	if s := (Automation{Days: "周二", Time: "14:00"}).Schedule(); s != "每周二 · 14:00" {
		t.Fatal(s)
	}
	if s := (Automation{Days: "工作日", Time: "09:00"}).Schedule(); s != "工作日 · 09:00" {
		t.Fatal(s)
	}
}

func TestFiresOn(t *testing.T) {
	mon := MondayOf(sun) // 11 月 9 日周一
	if mon.Day() != 9 {
		t.Fatalf("周一应是 9 日:%v", mon)
	}
	wk := Automation{On: true, Days: "工作日"}
	for i, want := range []bool{true, true, true, true, true, false, false} {
		if wk.FiresOn(mon.AddDate(0, 0, i)) != want {
			t.Errorf("工作日第 %d 天应 %v", i, want)
		}
	}
	if !(Automation{On: true, Days: "周二"}).FiresOn(mon.AddDate(0, 0, 1)) || (Automation{On: true, Days: "周二"}).FiresOn(mon) {
		t.Fatal("周二只在周二运行")
	}
	if !(Automation{On: true, Days: "每月 15 日"}).FiresOn(sun) || (Automation{On: false, Days: "每天"}).FiresOn(sun) {
		t.Fatal("每月 15 日应在 15 号运行;暂停的不运行")
	}
}

func TestWhen(t *testing.T) {
	at := func(d, h int) time.Time { return time.Date(2026, 11, d, h, 0, 0, 0, time.UTC) }
	cases := []struct {
		t    time.Time
		want string
	}{{at(15, 9), "今天 09:00"}, {at(14, 9), "昨天 09:00"}, {at(10, 14), "周二 14:00"}, {at(3, 14), "上周二 14:00"}, {at(2, 14), "上周一 14:00"}, {at(1, 14), "11 月 1 日"}}
	for _, c := range cases {
		if got := When(sun, c.t); got != c.want {
			t.Errorf("%v → %q,想要 %q", c.t, got, c.want)
		}
	}
}

func TestWeekSlots(t *testing.T) {
	mon := MondayOf(sun)
	items := []Automation{
		{ID: 1, Name: "日报", Days: "工作日", Time: "09:00", On: true},
		{ID: 2, Name: "依赖", Days: "周二", Time: "14:00", On: true, Runs: []Run{{At: mon.AddDate(0, 0, 1).Add(14 * time.Hour), OK: false, Err: "网络超时"}}},
		{ID: 3, Name: "备份", Days: "每天", Time: "20:00", On: false},
	}
	w := Week(items, mon, sun)
	if len(w[1]) != 2 || w[1][0].Time != "09:00" || w[1][1].Time != "14:00" || !w[1][1].Err {
		t.Fatalf("周二应有 2 个块且 14:00 标失败:%+v", w[1])
	}
	if len(w[5]) != 0 { // 周六在 from(周日)之前,暂停的备份不显示
		t.Fatalf("周六不应有暂停的备份:%+v", w[5])
	}
	if len(w[6]) != 1 || !w[6][0].Off || w[6][0].Name != "备份" {
		t.Fatalf("周日只应有划线的备份:%+v", w[6])
	}
}

func TestStore(t *testing.T) {
	s := NewStore([]Automation{{ID: 4, Name: "a", Days: "每天", Time: "09:00", On: true}})
	n := s.Add("router-demo")
	if n.ID != 5 || len(s.List()) != 2 {
		t.Fatal("新建应取下一个 ID")
	}
	n.Name, n.Time = "  ", "09:00"
	if s.Update(n) == nil {
		t.Fatal("空名称应拒绝")
	}
	n.Name, n.Time = "x", "9:00"
	if s.Update(n) == nil {
		t.Fatal("坏时间应拒绝")
	}
	n.Time = "10:30"
	if err := s.Update(n); err != nil {
		t.Fatal(err)
	}
	s.RunNow(5, sun)
	s.Toggle(5)
	got, _ := s.Get(5)
	if got.Name != "x" || got.On || len(got.Runs) != 1 {
		t.Fatalf("%+v", got)
	}
	if txt, st := got.Last(sun); txt != "已暂停" || st != 0 {
		t.Fatal(txt)
	}
	s.Toggle(5)
	if txt, st := mustGet(s, 5).Last(sun); txt != "今天 12:00 · 成功" || st != 1 {
		t.Fatal(txt)
	}
	s.Delete(5)
	if _, ok := s.Get(5); ok {
		t.Fatal("应已删除")
	}
}

func mustGet(s *Store, id int) Automation { a, _ := s.Get(id); return a }
