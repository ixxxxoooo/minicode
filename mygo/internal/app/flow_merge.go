package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 「工具调用详细程度 = 精简」时,相邻的工具调用(读 / 搜 / 改 / 跑 / 抓取 / MCP…)合并成一行摘要,
// 例如「编辑了 2 个文件、运行了 2 条命令」,展开后才看到逐条;标准档只合并探索类(见 exploreGroup),
// 详细档不合并且默认展开 diff / 命令输出。设计稿:设置 → 对话 → 工具调用详细程度的预览。

// mergeable 精简档下可并入摘要行的条目:只认普通工具调用(子 Agent、思考、待办、权限卡各自成行,保持可见)。
func mergeable(it core.Item) (*core.ToolCall, bool) {
	tc, ok := it.(*core.ToolCall)
	if !ok {
		return nil, false
	}
	switch tc.Kind {
	case core.KindSubagent, core.KindThink:
		return nil, false
	}
	return tc, true
}

// mergeSummary 把一串工具调用写成「读取了 3 个文件、编辑了 2 个文件、运行了 2 条命令」。
// 返回摘要、是否还有进行中的调用、失败数。
func mergeSummary(g []*core.ToolCall) (string, bool, int) {
	var read, search, edit, run, web, other, failed int
	running := false
	for _, tc := range g {
		switch tc.Kind {
		case core.KindRead:
			read++
		case core.KindSearch:
			search++
		case core.KindEdit, core.KindDelete, core.KindMove:
			edit++
		case core.KindExecute:
			run++
		case core.KindFetch:
			web++
		default:
			other++
		}
		if !tc.Status.Finished() {
			running = true
		}
		if tc.Status.Unsuccessful() {
			failed++
		}
	}
	var parts []string
	add := func(n int, f string) {
		if n > 0 {
			parts = append(parts, fmt.Sprintf(f, n))
		}
	}
	add(read, "读取了 %d 个文件")
	add(search, "搜索了 %d 次")
	add(edit, "编辑了 %d 个文件")
	add(run, "运行了 %d 条命令")
	add(web, "抓取了 %d 个页面")
	add(other, "调用了 %d 次工具")
	return strings.Join(parts, "、"), running, failed
}

// mergeGroup 渲染精简档的合并行:折叠时一行摘要,展开后逐条(与探索组同构)。
func (a *App) mergeGroup(c *ui.Context, g []*core.ToolCall) {
	sum, running, failed := mergeSummary(g)
	id := "mrg:" + g[0].ID
	sp := step{id: id, icon: "layers", verb: "", obj: sum, fam: famExplore, hasBody: true, open: false}
	if running {
		sp.state = "run"
	}
	if failed > 0 && !running {
		sp.right = fmt.Sprintf("%d 项失败", failed)
	}
	if set, ok := a.folds[id]; ok {
		sp.open = set
	}
	a.stepRow(c, sp, func() {
		a.nest(c, func() {
			for _, tc := range g {
				ps := a.presentTool(tc)
				ps.open = false // 精简档逐条也保持折叠
				a.stepRow(c, ps, func() { a.toolBody(c, tc) })
			}
		})
	})
}
