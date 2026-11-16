package app

import (
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
)

func TestMcpRowsUnifiedComposerAndManage(t *testing.T) {
	a := New(Options{})
	a.cust = demo.CustomizeFor(0)
	a.st.Mcp = []core.McpServerStatus{{Name: "github", Conn: core.McpConnected, Tools: []string{"a", "b"}}}
	rows := a.mcpRowsUnified()
	if len(rows) == 0 {
		t.Fatal("演示应有 MCP 配置")
	}
	var gh *McpRow
	for i := range rows {
		if rows[i].Name == "github" {
			gh = &rows[i]
			break
		}
	}
	if gh == nil {
		t.Fatal("应有 github")
	}
	if !gh.Runtime || gh.Conn != core.McpConnected {
		t.Fatalf("运行时状态未合并: runtime=%v conn=%v", gh.Runtime, gh.Conn)
	}
	a.st.Mcp = nil
	rows2 := a.mcpRowsUnified()
	for _, r := range rows2 {
		if r.Name == "github" && r.Runtime {
			t.Fatal("无运行时应为配置层")
		}
	}
}

func TestMcpRefreshDebounce(t *testing.T) {
	a := New(Options{})
	fb := &fakeBE{}
	a.Attach(fb)
	a.mcpRefreshAt = time.Now()
	a.mcpRefresh(mcpRefreshOpts{Force: false, ReloadDisk: true})
	if len(fb.cmds) != 0 {
		t.Fatal("防抖内不应发后端命令")
	}
}
