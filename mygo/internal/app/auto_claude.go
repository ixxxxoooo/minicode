package app

import (
	"context"

	"github.com/ixxxxoooo/minicode/internal/automation"
)

// SetAutomationRunnerCtx 注入带 ctx 与流式输出的运行器(真实模式由 main 调用)。
func (a *App) SetAutomationRunnerCtx(f func(ctx context.Context, it automation.Automation, dir string, onChunk func(string)) (string, error)) {
	a.autoRunCtx = f
	// 兼容仍读取 autoRun 的路径(演示测试)
	if f != nil {
		a.autoRun = func(it automation.Automation, dir string) (string, error) {
			return f(context.Background(), it, dir, nil)
		}
	}
}
