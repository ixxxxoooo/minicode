package app

import "testing"

// IDE 形态下有查询时,文件排在会话之前;Agent 形态保持命令 → 会话 → 文件。
func TestPaletteFilesFirstInIDE(t *testing.T) {
	a, tt := demoIDE(t)
	tt.Frame()
	a.pq.query = "router"
	first := func() string {
		items := a.palItems()
		if len(items) == 0 {
			t.Fatal("应有搜索结果")
		}
		for _, it := range items {
			if it.group == "文件" || it.group == "会话" {
				return it.group
			}
		}
		return ""
	}
	a.mode = ModeIDE
	if g := first(); g != "文件" {
		t.Fatalf("IDE 形态应先列文件,实际先出现 %q", g)
	}
}
