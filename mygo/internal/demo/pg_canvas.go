// 演示数据索引 · 画布
//
// 套 0(默认):单张「Q3 路由层重构」,见 canvas.go。
// 套 1:左侧列表多张——空画布、样式总览、草稿 / 评审中 / 已定稿、进行中的「Claude 修改」稿。
// 套 2:单张 300 块长文,测滚动与大纲。
// 套 3:单张 30 版,v1–v30 由 Claude/你 交替,含「已回退」说明。
// 套 4:当前文档状态「Claude 修改中」。
//
// 切换:命令面板搜「演示:画布」。
// 设计稿缺口:引用 / 图片 / 分割线 / 独立 Mermaid 块类型未在模型里,用段落与图表块近似。
package demo

import (
	"fmt"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/canvas"
)

func canvasQ3(now time.Time) *canvas.Store {
	return Canvas(now)
}

func canvasEmpty(now time.Time) *canvas.Store {
	doc := canvas.Doc{Title: "未命名画布", Status: "草稿"}
	return canvas.NewStore(doc, "你", "新建", now)
}

func canvasStyleGuide(now time.Time) *canvas.Store {
	node := func(x, w float32, l string, hot bool) canvas.Node {
		return canvas.Node{X: x, Y: 30, W: w, Label: l, Hot: hot}
	}
	big := &canvas.Diagram{
		Caption: "多层服务:入口 → 鉴权 → 业务 → 存储(大字级演示)",
		Nodes: []canvas.Node{
			node(10, 70, "边缘", false), node(100, 80, "API", true), node(210, 80, "Worker", true),
			node(320, 70, "队列", false), node(410, 80, "DB", false), node(210, 100, "缓存", false),
		},
		Edges: [][][2]float32{
			{{80, 45}, {100, 45}}, {{180, 45}, {210, 45}}, {{290, 45}, {320, 45}}, {{390, 45}, {410, 45}},
			{{250, 60}, {250, 95}, {210, 95}},
		},
	}
	wide := &canvas.Table{
		Head: []string{"模块", "Q1", "Q2", "Q3", "Q4", "负责人", "备注"},
		Rows: [][]string{
			{"路由", "完成", "完成", "进行中", "计划中", "Lin", "大小写归一化"},
			{"鉴权", "完成", "完成", "完成", "维护", "Wen", "OAuth 2.1"},
			{"观测", "计划中", "进行中", "—", "—", "Mia", "Tracing 统一"},
		},
	}
	emptyTbl := &canvas.Table{Head: []string{"指标", "值"}, Rows: nil}
	blocks := []canvas.Block{
		{ID: "h0", Kind: canvas.Heading, Level: 1, Text: "样式总览"},
		{ID: "lead", Kind: canvas.Para, Lead: true, Anchor: "导语", Text: "本文汇总画布支持的块类型与排版(演示数据)。"},
		{ID: "h2a", Kind: canvas.Heading, Level: 2, Text: "二级标题"},
		{ID: "h2b", Kind: canvas.Heading, Level: 2, Text: "另一节"},
		{ID: "h3", Kind: canvas.Heading, Level: 2, Text: "三级样式的段落"},
		{ID: "p1", Kind: canvas.Para, Text: "普通段落:对齐设计稿 1.65 行高,可含 **Markdown** 风格记号(预览走 Markdown)。"},
		{ID: "quote", Kind: canvas.Para, Anchor: "引用", Text: "> 设计的本质是取舍。\n> —— 演示引用块(用段落模拟)"},
		{ID: "code", Kind: canvas.Para, Anchor: "代码", Text: "```go\nfunc main() {\n    fmt.Println(\"hello canvas\")\n}\n```"},
		{ID: "img", Kind: canvas.Para, Text: "[图片占位:架构截图 1200×600,预览模式显示占位文案]"},
		{ID: "hr", Kind: canvas.Para, Text: "---"},
		{ID: "dg", Kind: canvas.DiagramBlock, Diagram: big},
		{ID: "tbl", Kind: canvas.TableBlock, Table: wide},
		{ID: "tble", Kind: canvas.TableBlock, Table: emptyTbl},
		{ID: "todo", Kind: canvas.TodoBlock, Todos: []canvas.Todo{
			{Text: "已完成的条目", State: 2}, {Text: "进行中的条目", State: 1}, {Text: "未开始", State: 0},
		}},
	}
	doc := canvas.Doc{Title: "画布样式总览", Status: "评审中", Blocks: blocks}
	return canvas.NewStore(doc, "Claude", "补齐全块类型", now)
}

func canvasStatusDoc(title, status string, now time.Time) *canvas.Store {
	doc := canvas.Doc{
		Title: title, Status: status,
		Blocks: []canvas.Block{
			{ID: "p", Kind: canvas.Para, Lead: true, Anchor: "说明", Text: "状态标签:「" + status + "」。演示多画布列表时并排对比。"},
		},
	}
	return canvas.NewStore(doc, "你", "更新状态", now)
}

func canvasClaudeBusy(now time.Time) *canvas.Store {
	doc := canvas.Doc{
		Title: "实时协作草稿", Status: "Claude 修改中",
		Blocks: []canvas.Block{
			{ID: "p", Kind: canvas.Para, Lead: true, Anchor: "进行中", Text: "Claude 正在根据你的指令改写本节(演示:仅状态标签,无真实流式)。"},
			{ID: "todo", Kind: canvas.TodoBlock, Todos: []canvas.Todo{{Text: "补风险一节", State: 1}}},
		},
	}
	s := canvas.NewStore(doc, "Claude", "开始修改", now)
	return s
}

func canvasLong300(now time.Time) *canvas.Store {
	var blocks []canvas.Block
	blocks = append(blocks, canvas.Block{ID: "lead", Kind: canvas.Para, Lead: true, Anchor: "导读", Text: "共 300 个块,用于测滚动、大纲跳转与预览性能。"})
	for i := 0; i < 300; i++ {
		id := fmt.Sprintf("b%d", i)
		if i%25 == 0 {
			blocks = append(blocks, canvas.Block{ID: id, Kind: canvas.Heading, Level: 2, Text: fmt.Sprintf("第 %d 章 · 小节簇", i/25+1)})
		} else if i%7 == 0 {
			blocks = append(blocks, canvas.Block{ID: id, Kind: canvas.TodoBlock, Todos: []canvas.Todo{{Text: fmt.Sprintf("待办 #%d", i), State: i % 3}}})
		} else {
			blocks = append(blocks, canvas.Block{ID: id, Kind: canvas.Para, Text: fmt.Sprintf("段落 %d:%s", i, strings.Repeat("内容", 8+i%5))})
		}
	}
	doc := canvas.Doc{Title: "超长滚动测试(300 块)", Status: "草稿", Blocks: blocks}
	return canvas.NewStore(doc, "你", "生成压测文档", now)
}

func canvasVersions30(now time.Time) *canvas.Store {
	base := []canvas.Block{{ID: "h", Kind: canvas.Heading, Level: 2, Text: "版本线"}, {ID: "p", Kind: canvas.Para, Text: "v30 正文"}}
	doc := canvas.Doc{Title: "版本密集型画布", Status: "已定稿", Blocks: cloneBlocks(base)}
	s := canvas.NewStore(doc, "Claude", "v30", now)
	var vers []canvas.Version
	for n := 30; n >= 1; n-- {
		b := cloneBlocks(base)
		b[1].Text = fmt.Sprintf("v%d 正文(演示)", n)
		by := "Claude"
		if n%2 == 0 {
			by = "你"
		}
		note := fmt.Sprintf("迭代 %d", n)
		if n == 12 {
			note = "已回退:从 v15 回到 v12"
		}
		vers = append(vers, canvas.Version{
			N: n, By: by, Note: note, At: now.Add(-time.Duration(30-n) * time.Hour),
			Doc: canvas.Doc{Title: doc.Title, Status: doc.Status, Blocks: b},
		})
	}
	s.Versions = vers
	return s
}

func cloneBlocks(in []canvas.Block) []canvas.Block {
	out := make([]canvas.Block, len(in))
	copy(out, in)
	return out
}

// CanvasDemoLibrary 返回某套的列表 id→存储与默认选中 id。
func CanvasDemoLibrary(set int, now time.Time) (active string, stores map[string]*canvas.Store) {
	stores = map[string]*canvas.Store{}
	switch set {
	case 1:
		stores["empty"] = canvasEmpty(now)
		stores["style-guide"] = canvasStyleGuide(now)
		stores["st-draft"] = canvasStatusDoc("产品路线图", "草稿", now)
		stores["st-review"] = canvasStatusDoc("API 设计评审", "评审中", now.Add(-time.Hour))
		stores["st-final"] = canvasStatusDoc("发布清单", "已定稿", now.Add(-2*time.Hour))
		stores["st-busy"] = canvasClaudeBusy(now)
		return "style-guide", stores
	case 2:
		stores["long-300"] = canvasLong300(now)
		return "long-300", stores
	case 3:
		stores["versions-30"] = canvasVersions30(now)
		return "versions-30", stores
	case 4:
		stores["claude-busy"] = canvasClaudeBusy(now)
		return "claude-busy", stores
	default:
		stores["q3-router"] = canvasQ3(now)
		return "q3-router", stores
	}
}
