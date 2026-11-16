package app

import (
	"testing"
)

// cuCard 走 kit.Card 后自定义详情区应能正常渲染(不崩)。
func TestCuCardRenders(t *testing.T) {
	a, tt := newTestApp(t)
	a.SeedDemoExtras()
	a.screen = ScreenCustomize
	tt.Frame()
	tt.Frame()
	if !tt.HasText("github") {
		t.Fatal("自定义 MCP 详情应渲染")
	}
}
