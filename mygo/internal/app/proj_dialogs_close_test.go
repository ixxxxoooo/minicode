package app

import (
	"testing"
	"time"
)

// 右上角 ✕ / 取消 应走 dialogOpen 退场,而不是仅把 *open 置 false 后被下一帧复位(表现为点了没反应)。
func TestProjDlgCloseX(t *testing.T) {
	a, tt := newTestApp(t)
	a.SeedDemoExtras()
	a.OpenProjDialog("wsedit")
	tt.Frame()
	time.Sleep(200 * time.Millisecond)
	tt.Frame()
	if a.pd.kind != "ws" || !a.pd.open {
		t.Fatalf("应有编辑工作区:kind=%q open=%v", a.pd.kind, a.pd.open)
	}
	if err := tt.Click("关闭 Esc"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.pd.open {
		t.Fatal("点后 open 应为 false")
	}
	for i := 0; i < 8; i++ {
		time.Sleep(80 * time.Millisecond)
		tt.Frame()
		if a.pd.kind == "" {
			break
		}
	}
	if a.pd.kind != "" {
		t.Fatalf("退场后应关闭:kind=%q", a.pd.kind)
	}
}

func TestProjDlgCloseCancel(t *testing.T) {
	a, tt := newTestApp(t)
	a.SeedDemoExtras()
	a.openAddProject()
	tt.Frame()
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		time.Sleep(80 * time.Millisecond)
		tt.Frame()
		if a.pd.kind == "" {
			break
		}
	}
	if a.pd.kind != "" {
		t.Fatalf("取消后应关闭:kind=%q", a.pd.kind)
	}
}
