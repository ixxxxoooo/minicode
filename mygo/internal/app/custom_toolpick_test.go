package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/projects"
)

// 「按需加载工具」要真的写进 Claude 设置,并且写在它当前生效的那一层;重读配置后状态一致。
func TestMCPOnDemandPersists(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	a, _ := newTestApp(t)
	a.reg = projects.Open(filepath.Join(t.TempDir(), "ws.json")) // 持久存储 = 非演示
	a.demoExtra = false
	ds, err := customize.OpenDisk(proj)
	if err != nil {
		t.Skip("无法打开磁盘存储:", err)
	}
	a.proj = proj
	a.cu.disk = ds
	a.cust = ds.Store
	if a.cuDemo() {
		t.Skip("测试应用处于演示模式")
	}
	a.cuMCPOnDemand(false)
	b, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil || !strings.Contains(string(b), `"ENABLE_TOOL_SEARCH": "false"`) {
		t.Fatalf("应写入用户层设置:%v %s", err, b)
	}
	if on, _ := ds.ToolSearchValue(); on {
		t.Fatal("重读后应是关闭")
	}
	// 项目层有覆盖时,改动写进项目层
	_ = os.MkdirAll(filepath.Join(proj, ".claude"), 0o755)
	_ = os.WriteFile(filepath.Join(proj, ".claude", "settings.json"), []byte(`{"env":{"ENABLE_TOOL_SEARCH":"false"}}`), 0o644)
	a.cuMCPOnDemand(true)
	pb, _ := os.ReadFile(filepath.Join(proj, ".claude", "settings.json"))
	if !strings.Contains(string(pb), `"auto"`) {
		t.Fatalf("应写进项目层:%s", pb)
	}
	if on, _ := ds.ToolSearchValue(); !on {
		t.Fatal("项目层改回后应是开启")
	}
}
