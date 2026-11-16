package app

import (
	"os"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/demo"
)

// 演示模式启动路径(SetProjects + SeedDemoRegistry + SeedDemoData)必须带上整页功能的演示数据,
// 并且不能读写用户真实的 ~/.minicode 数据:自动化有演示项、画布库来自演示套(默认 Q3 路由层)且落在临时目录。
func TestSeedDemoData(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	a := New(Options{Palette: "paper", Scheme: "a", FakeLights: true})
	a.SetProjects(demo.Projects(), "fixbug")
	a.SetClock(func() time.Time { return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) })
	a.SeedDemoRegistry()
	a.SeedDemoData()

	if n := len(a.auto.List()); n < 2 {
		t.Fatalf("演示自动化应有多项,实际 %d", n)
	}
	if !a.demoExtra {
		t.Fatal("演示数据源应启用")
	}
	if ids := a.cvLib().IDs(); len(ids) != 1 || ids[0] != "q3-router" {
		t.Fatalf("演示画布库默认应是「Q3 路由层」一张,实际 %v", ids)
	}
	if _, err := os.Stat(home + "/.minicode/canvases"); err == nil {
		t.Fatal("演示模式不应在真实目录下创建画布库")
	}
}
