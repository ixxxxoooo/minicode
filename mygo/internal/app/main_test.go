package app

import (
	"os"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/demo"
)

// 编辑器 / 搜索类测试依赖沙盒里的多语言与边界文件;Git 极端数据仍默认关闭,需要时用 stressDemo(t)。
func TestMain(m *testing.M) {
	demo.SetSandboxExtras(true)
	os.Exit(m.Run())
}
