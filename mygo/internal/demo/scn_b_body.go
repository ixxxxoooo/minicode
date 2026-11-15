package demo

import "github.com/ixxxxoooo/minicode/internal/core"

func init() {
	Register(Scenario{
		ID:    "body-edge",
		Title: "正文 · 极短与空回复",
		Build: buildBodyEdge,
		Label: "刚刚",
	})
	Register(Scenario{
		ID:    "body-stream",
		Title: "正文 · 流式生成中",
		Live:  true,
		Build: buildBodyStream,
		Label: "刚刚",
	})
	Register(Scenario{
		ID:    "body-stream-code",
		Title: "正文 · 流式未闭合代码块",
		Live:  true,
		Build: buildBodyStreamCode,
		Label: "刚刚",
	})
}

func buildBodyEdge() []Step {
	s := NewScript("body-edge", "正文 · 极短与空回复")
	uid := s.User("回复一个字就行")
	s.Say("好")
	s.End(uid, 1, core.StopEndTurn, statsTiny())

	uid = s.User("这轮不要输出可见正文")
	s.Say(" \n\t ")
	s.End(uid, 2, core.StopEndTurn, statsTiny())
	return s.Steps()
}

func buildBodyStream() []Step {
	s := NewScript("body-stream", "正文 · 流式生成中")
	s.User("继续写完这段发布说明")
	aid := s.ID("a")
	s.Add(ms(300), core.EvAppend{Item: &core.Assistant{ID: aid, Streaming: true}})
	s.Add(ms(120), core.EvTextDelta{ID: aid, Delta: "这是仍在流式输出的正文:"})
	s.Add(ms(200), core.EvTextDelta{ID: aid, Delta: "\n\n- 亮点一\n- 亮点二\n- 亮点"})
	// 故意不 EvItemDone / EvTurnEnded,停在 PhaseRunning
	return s.Steps()
}

func buildBodyStreamCode() []Step {
	s := NewScript("body-stream-code", "正文 · 流式未闭合代码块")
	s.User("贴一段正在输入的 Go 代码")
	aid := s.ID("a")
	s.Add(ms(250), core.EvAppend{Item: &core.Assistant{ID: aid, Streaming: true}})
	delta := "正在写入示例:\n\n```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"streaming\")\n"
	s.Add(ms(100), core.EvTextDelta{ID: aid, Delta: delta})
	// 围栏未闭合,保持流式
	return s.Steps()
}
