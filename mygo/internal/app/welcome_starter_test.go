package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/vcs"
)

func TestReviewStarterDescUsesRealChanges(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.sc.loaded = false
	if got := a.reviewStarterDesc(); got != "审查未提交的改动" {
		t.Fatalf("未加载时=%q", got)
	}
	a.sc.loaded = true
	a.sc.changes = nil
	if got := a.reviewStarterDesc(); got != "没有未提交的改动" {
		t.Fatalf("无改动时=%q", got)
	}
	a.sc.changes = []vcs.Change{
		{Path: "a.go", Add: 3, Del: 1},
		{Path: "a.go", Staged: true, Add: 2},
		{Path: "b.go", Add: 1, Del: 4},
	}
	if got := a.reviewStarterDesc(); got != "2 个文件 · +6 −5" {
		t.Fatalf("有改动时=%q", got)
	}
}
