package demo

func init() {
	Register(Scenario{
		ID:      "mermaid-samples",
		Title:   "Mermaid · 图表示例合集",
		Label:   "刚刚",
		Project: "scenario-lab",
		Group:   "scenario-lab",
		Build:   buildMermaidSamples,
	})
}

func buildMermaidSamples() []Step {
	s := NewScript("mermaid-samples", "Mermaid · 图表示例合集")
	s.User("请用几张图说明各类 Mermaid 围栏的图表卡、全屏与失败回退。")
	s.Say(`下面依次是流程图、时序图、饼图、状态图、不支持的 gantt,以及语法错误示例。

` + "```mermaid\ngraph LR\n  A[请求] --> B{路由}\n  B -->|命中| C[处理]\n  B -->|未命中| D[404]\n```\n\n" +
		"```mermaid\nsequenceDiagram\n  participant U as 用户\n  participant S as 服务\n  U->>S: 查询\n  S-->>U: 结果\n```\n\n" +
		"```mermaid\npie showData title 耗时\n \"渲染\" : 30\n \"网络\" : 50\n \"其他\" : 20\n```\n\n" +
		"```mermaid\nstateDiagram-v2\n  [*] --> Idle\n  Idle --> Running : 启动\n  Running --> [*]\n```\n\n" +
		"```mermaid\ngantt\n  title 里程碑\n  section A\n  任务 : 1d\n```\n\n" +
		"```mermaid\ngraph TD\n  A -->\n```\n\n应看到前四张矢量图,gantt 与语法错误回退源码卡并写明原因。")
	return s.Steps()
}