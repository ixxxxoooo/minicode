package app

import (
	"os"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/demo"
)

// 编辑器 / 搜索类测试依赖沙盒里的多语言与边界文件;Git 极端数据仍默认关闭,需要时用 stressDemo(t)。
func TestMain(m *testing.M) {
	demo.SetSandboxExtras(true)
	// 隔离真实 HOME:测试不能读到开发机上的服务商 / Claude 配置(否则断言随机器而变),也不会写到真实目录
	home, err := os.MkdirTemp("", "mc-app-test-home-")
	if err == nil {
		os.Setenv("HOME", home)
	}
	code := m.Run()
	if err == nil {
		os.RemoveAll(home)
	}
	os.Exit(code)
}
