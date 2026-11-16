package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/customize"
)

// 确认框在测试里只调 pd.ok 回调而不走 UI 关框时,dialogOpen 退场遮罩会残留。
func TestConfirmBackdropAfterPdOkOnly(t *testing.T) {
	a, tt := newTestApp(t)
	a.SeedDemoExtras()
	a.screen = ScreenCustomize
	tt.Frame()
	st := a.cuStore()
	a.confirmDlg("删除?", "不可撤销", "删除", true, func() {
		st.Remove(customize.MCP, a.cu.sel)
	})
	tt.Frame()
	if a.pd.kind != "confirm" || a.dlgSeen != "confirm" {
		t.Fatalf("应有确认框遮罩:kind=%q seen=%q", a.pd.kind, a.dlgSeen)
	}
	a.pd.ok() // 与多处旧测试相同:只跑回调,不 closeProjDlg
	tt.Frame()
	if a.pd.kind != "confirm" {
		t.Fatalf("pd.ok 未清 kind,仍是 %q", a.pd.kind)
	}
	if a.dlgSeen != "confirm" && a.dlgClosing != "confirm" {
		t.Fatalf("遮罩应仍在:seen=%q closing=%q", a.dlgSeen, a.dlgClosing)
	}
	a.closeProjDlg()
	tt.Frame()
	tt.Frame() // 等退场
	if a.dlgSeen == "confirm" && a.dlgClosing == "" {
		// dialogOpen 需要再画一帧才开始退场
		tt.Frame()
	}
}
