package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// 「工具调用详细程度 = 精简」时,相邻的工具调用(读 / 搜 / 改 / 跑 / 抓取 / MCP…)合并成一行摘要,
// 例如「编辑了 2 个文件、运行了 2 条命令」,展开后才看到逐条;标准档只合并探索类(见 exploreGroup),
// 详细档不合并且默认展开 diff / 命令输出。设计稿:设置 → 对话 → 工具调用详细程度的预览。

// mergeable 精简档下可并入摘要行的条目:普通工具调用(子 Agent、思考、待办、待处理的权限卡各自成行,保持可见)。
// 已处理的权限请求(点了「允许 / 拒绝」)里装的也是那次工具调用,同样并入——否则一个需要确认的编辑 / 命令
// 就会把相邻的一串从中间截断(设计稿 flow-detail:被拒绝的调用仍留在原来的组里,计入「N 项失败」)。
func mergeable(it core.Item) (*core.ToolCall, bool) {
	tc := doneToolOf(it)
	if tc == nil {
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

// mergeSummaryNarrow 精简档窄档短句:「读 2 · 搜 1 · 改 2 · 跑 1」。
func mergeSummaryNarrow(g []*core.ToolCall) string {
	var read, search, edit, run, web, other int
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
	}
	var parts []string
	add := func(n int, label string) {
		if n > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", label, n))
		}
	}
	add(read, "读")
	add(search, "搜")
	add(edit, "改")
	add(run, "跑")
	add(web, "抓")
	add(other, "工具")
	return strings.Join(parts, " · ")
}

func mergeLiveTool(g []*core.ToolCall) *core.ToolCall {
	for i := len(g) - 1; i >= 0; i-- {
		if !g[i].Status.Finished() {
			return g[i]
		}
	}
	if len(g) > 0 {
		return g[len(g)-1]
	}
	return nil
}

// mergeGroup 渲染精简档的合并行:折叠时一行摘要,展开后逐条(与探索组同构)。
func (a *App) mergeGroup(c *ui.Context, g []*core.ToolCall) {
	sum, running, failed := mergeSummary(g)
	id := "mrg:" + g[0].ID
	narrow := a.chatTier() == tierNarrow
	sp := step{id: id, icon: "layers", verb: "", obj: sum, fam: famExplore, hasBody: true, open: false, thumbs: a.imagesOf(g)}
	if narrow && !running {
		sp.obj = mergeSummaryNarrow(g)
	}
	if running {
		sp.state = "run"
		if tc := mergeLiveTool(g); tc != nil {
			live := a.presentTool(tc)
			sp.verb, sp.obj = live.verb, live.obj
			if !narrow {
				sp.right = fmt.Sprintf("已 %d 步", len(g))
			}
		}
	} else if failed > 0 {
		sp.icon, sp.state = "errc", "err"
		sp.right = fmt.Sprintf("%d 项失败", failed)
	}
	if set, ok := a.folds[id]; ok {
		sp.open = set
	}
	// 进行中 → 结束:「最新一步」换成摘要句时淡入 120 ms(stepRow 按 fadeAt 画)
	if a.runWas[id] && !running {
		a.fadeAt[id] = time.Now()
	}
	if running {
		a.runWas[id] = true
	} else {
		delete(a.runWas, id)
	}
	a.stepRow(c, sp, func() {
		a.nest(c, func() {
			live := mergeLiveTool(g)
			for _, tc := range g {
				a.groupChild(c, tc, running && tc == live)
			}
		})
	})
}

// groupChild 画分组展开后的一条:普通步骤行(各自再点才展开详情,展开不拆组),默认收起;
// 失败 / 被拒绝 / 被中断的自动展开原因,进行中的那一条(autoOpen)展开看实时输出。
func (a *App) groupChild(c *ui.Context, tc *core.ToolCall, autoOpen bool) {
	ps := a.presentTool(tc)
	ps.open = ps.hasBody && (autoOpen || ps.imgRead || (tc.Status.Unsuccessful() && tc.Status != core.ToolFailed && tc.Status != core.ToolInterrupted))
	a.stepRow(c, ps, func() { a.toolBody(c, tc) })
}
