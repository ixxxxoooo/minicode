package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 新建成功把 dlg 置 nil 后,仍须走 dialogOpen 退场,否则遮罩会挡住详情操作。
func TestCuDialogCloseAfterCreate(t *testing.T) {
	a, tt := newTestApp(t)
	a.screen = ScreenCustomize
	tt.Frame()
	if err := tt.Click("添加服务器"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	a.cu.dlg.name, a.cu.dlg.extra = "tmp-mcp", "npx demo"
	tt.Frame()
	if err := tt.Click("创建"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(kit.DurFast + 20*time.Millisecond)
	tt.Frame()
	if a.cu.sel == nil || a.cu.sel.Name != "tmp-mcp" {
		t.Fatalf("应选中新建项:%v", a.cu.sel)
	}
	clickMore(t, tt, "删除", func(r ui.Rect) bool { return r.X > 900 })
	tt.Frame()
	if a.pd.kind != "confirm" {
		t.Fatalf("应弹确认框:%q", a.pd.kind)
	}
	a.pd.ok()
	_ = customize.MCP
}
