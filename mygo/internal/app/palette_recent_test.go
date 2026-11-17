package app

import "testing"

// 「最近使用」:执行过的命令按时间倒序置顶,最多 5 条,同一条不重复,查询时不出现。
// @author ixxxxoooo
func TestPaletteRecentCommands(t *testing.T) {
	a, _ := newTestApp(t)
	a.prefs.Set(pPalRecent, "")
	if it := a.palItems(); len(it) > 0 && it[0].group == palRecentGrp {
		t.Fatal("没执行过命令时不应有「最近使用」")
	}
	names := []string{"新会话", "切换权限模式", "源代码管理", "在项目中搜索", "历史会话…", "显示 / 隐藏导航栏", "设置:通用"}
	byTitle := map[string]palItem{}
	cmds, _ := a.palCommands()
	for _, c := range cmds {
		byTitle[c.title] = c
	}
	for _, n := range names {
		it, ok := byTitle[n]
		if !ok {
			t.Fatalf("缺命令 %q", n)
		}
		a.palRecordRecent(it)
	}
	a.palRecordRecent(byTitle["新会话"]) // 再执行一次:提前到最前,不重复
	var got []string
	for _, it := range a.palItems() {
		if it.group != palRecentGrp {
			break
		}
		got = append(got, it.title)
	}
	want := []string{"新会话", "设置:通用", "显示 / 隐藏导航栏", "历史会话…", "在项目中搜索"}
	if len(got) != len(want) {
		t.Fatalf("最近使用应有 %d 条,实际 %v", len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("第 %d 条应为 %q,实际 %q(全部:%v)", i, want[i], got[i], got)
		}
	}
	// 置顶的命令不再在「命令」组里重复出现
	for _, it := range a.palItems() {
		if it.group == "命令" && it.title == "新会话" {
			t.Fatal("最近使用里的命令不应在命令组重复")
		}
	}
	a.pq.query = "会话"
	for _, it := range a.palItems() {
		if it.group == palRecentGrp {
			t.Fatal("有查询时不显示「最近使用」")
		}
	}
	// 「转到第 N 行」这类动态标题不记录
	before := a.prefs.Str(pPalRecent, "")
	a.palRecordRecent(palItem{group: "命令", title: "转到第 12 行"})
	if a.prefs.Str(pPalRecent, "") != before {
		t.Fatal("动态标题不应被记录")
	}
}
