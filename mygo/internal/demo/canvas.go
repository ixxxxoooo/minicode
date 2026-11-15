package demo

import (
	"time"

	"github.com/ixxxxoooo/minicode/internal/canvas"
)

// Canvas 是演示画布「Q3 路由层重构」:三个版本(v1 只有架构、v2 加里程碑、v3 加待办)。
func Canvas(now time.Time) *canvas.Store {
	node := func(x, w float32, l string, hot bool) canvas.Node {
		return canvas.Node{X: x, Y: 30, W: w, Label: l, Hot: hot}
	}
	dg := &canvas.Diagram{
		Caption: "请求经过路由表,先归一化再匹配",
		Nodes: []canvas.Node{
			node(20, 90, "请求", false), node(160, 90, "路由表", true), node(300, 90, "处理器", false), node(430, 70, "响应", false),
			{X: 300, Y: 85, W: 90, Label: "normalize"},
		},
		Edges: [][][2]float32{
			{{110, 45}, {160, 45}}, {{250, 45}, {300, 45}}, {{390, 45}, {430, 45}}, {{205, 60}, {205, 100}, {300, 100}},
		},
	}
	arch := []canvas.Block{
		{ID: "h-bg", Kind: canvas.Para, Lead: true, Anchor: "背景", Text: "把路径匹配收敛到单一的归一化入口,消除大小写与末尾斜杠带来的重复路由。"},
		{ID: "h-arch", Kind: canvas.Heading, Level: 2, Text: "架构"},
		{ID: "dg", Kind: canvas.DiagramBlock, Diagram: dg},
	}
	ms := []canvas.Block{
		{ID: "h-ms", Kind: canvas.Heading, Level: 2, Text: "里程碑"},
		{ID: "tb", Kind: canvas.TableBlock, Table: &canvas.Table{
			Head: []string{"阶段", "负责人", "时间", "状态"},
			Rows: [][]string{{"归一化抽取", "你", "10 月 18 日", "已完成"}, {"匹配层替换", "Lin", "10 月 25 日", "进行中"}, {"旧路由清理", "Wen", "11 月 3 日", "未开始"}},
		}},
	}
	todo := []canvas.Block{
		{ID: "h-todo", Kind: canvas.Heading, Level: 2, Text: "待办"},
		{ID: "td", Kind: canvas.TodoBlock, Todos: []canvas.Todo{{Text: "抽出 normalize", State: 2}, {Text: "补充大小写用例", State: 1}, {Text: "清理旧的 fallback 逻辑", State: 0}}},
	}
	doc := func(parts ...[]canvas.Block) canvas.Doc {
		d := canvas.Doc{Title: "Q3 路由层重构", Status: "草稿"}
		for _, p := range parts {
			d.Blocks = append(d.Blocks, p...)
		}
		return d.Clone()
	}
	s := canvas.NewStore(doc(arch, ms, todo), "Claude", "补充里程碑表", now)
	s.Versions = []canvas.Version{
		{N: 3, By: "Claude", Note: "补充里程碑表", At: now, Doc: doc(arch, ms, todo)},
		{N: 2, By: "你", Note: "调整架构图", At: now.Add(-2 * time.Hour), Doc: doc(arch, ms)},
		{N: 1, By: "Claude", Note: "初稿", At: now.Add(-24 * time.Hour), Doc: doc(arch)},
	}
	return s
}
