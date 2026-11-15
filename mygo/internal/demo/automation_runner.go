// 演示模式的自动化运行器:模拟 claude -p 流式输出,可取消,成功 / 失败两态。
package demo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/automation"
)

// RunAutomationDemo 在演示模式下跑一条自动化:分多段写出输出,ctx 取消则返回已取消。
func RunAutomationDemo(ctx context.Context, it automation.Automation, _ string, onChunk func(string)) (string, error) {
	lines := []string{
		fmt.Sprintf("[演示] 开始运行「%s」", it.Name),
		fmt.Sprintf("项目:%s · 权限:%s", it.Project, it.Perm),
		"---",
		"正在检查仓库状态…",
		"没有未提交的阻塞项。",
		"---",
		"摘要:按提示词完成了只读检查(演示数据,非真实 claude)。",
	}
	fail := strings.Contains(it.Name, "失败") || strings.Contains(strings.ToLower(it.Prompt), "故意失败")
	if fail {
		lines = append(lines, "---", "错误:网络超时(演示)")
	}
	var buf strings.Builder
	tick := time.NewTicker(450 * time.Millisecond)
	defer tick.Stop()
	i := 0
	for {
		select {
		case <-ctx.Done():
			return buf.String(), ctx.Err()
		case <-tick.C:
			if i >= len(lines) {
				if fail {
					return buf.String(), fmt.Errorf("网络超时")
				}
				return buf.String(), nil
			}
			line := lines[i] + "\n"
			i++
			buf.WriteString(line)
			if onChunk != nil {
				onChunk(line)
			}
		}
	}
}
