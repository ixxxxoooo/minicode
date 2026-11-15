package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/demo"
)

// stressDemo 打开演示的「极端数据」(300 文件 bulk、冲突、5 条储藏等),测试结束自动关闭。
// 必须在造 App 之前调用:演示仓库是在造 App 时按当时的开关构建的。
func stressDemo(t *testing.T) {
	t.Helper()
	demo.SetStress(true)
	t.Cleanup(func() { demo.SetStress(false) })
}
