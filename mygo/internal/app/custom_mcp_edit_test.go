package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/kit"
)

// MCP 详情「编辑配置」必须真能弹出编辑框(原先 autoDlg 只认两个 key,mcp-edit 永远不显示)。
func TestMcpEditDialogOpens(t *testing.T) {
	a, tt := newTestApp(t)
	a.screen = ScreenCustomize
	tt.Frame()
	if err := tt.Click("添加服务器"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	a.cu.dlg.name, a.cu.dlg.extra = "edit-mcp", "npx demo"
	tt.Frame()
	if err := tt.Click("创建"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(kit.DurFast + 20*time.Millisecond)
	tt.Frame()
	if err := tt.Click("编辑配置"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(kit.DurFast + 20*time.Millisecond)
	tt.Frame()
	if !a.cu.mcpEdit {
		t.Fatal("点「编辑配置」后应进入编辑态")
	}
	if !tt.HasText("编辑 MCP 配置") {
		t.Fatal("编辑对话框没有显示出来")
	}
	a.cu.draft.Cmd = "npx -y changed"
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(kit.DurFast + 20*time.Millisecond)
	tt.Frame()
	if a.cu.mcpEdit {
		t.Fatalf("保存后应关闭编辑框,err=%q", a.cu.err)
	}
}
