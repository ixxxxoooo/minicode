package app

import "testing"

func TestTermTabLifecycle(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a"})
	before := len(a.ide.tps)
	a.newTerm()
	a.newTerm()
	if len(a.ide.tps) <= before {
		t.Fatalf("newTerm 应增加标签数, %d -> %d", before, len(a.ide.tps))
	}
	last := len(a.ide.tps) - 1
	a.closeTermTab(last)
	if len(a.ide.tps) >= last+1 {
		t.Fatalf("关闭末个标签后数量应变少")
	}
	for len(a.ide.tps) > 1 {
		a.closeTermTab(0)
	}
	if len(a.ide.tps) != 1 {
		t.Fatalf("关到最后应留 1 个壳,实际 %d", len(a.ide.tps))
	}
}
