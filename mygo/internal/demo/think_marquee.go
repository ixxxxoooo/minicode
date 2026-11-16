package demo

import "github.com/ixxxxoooo/minicode/internal/core"

func init() {
	Register(Scenario{
		ID: "think-marquee", Title: "思考 · 折叠跑马灯", Project: "",
		Build: buildThinkMarquee,
		Live:  true,
	})
}

// ThinkMarqueeReplay 留一段流式思考不收尾(截图 / -ed thinking)。
func ThinkMarqueeReplay() []Step { return buildThinkMarquee() }

func buildThinkMarquee() []Step {
	s := NewScript("think-marquee", "思考折叠跑马灯")
	u := s.User("演示思考折叠时的右侧跑马灯预览。")
	tid := s.ID("t")
	long := repeatPad("路由表用原始路径做键,注册与匹配都没有归一化。最小改动是抽一个 normalize,两处共用,顺带处理末尾斜杠。", 420)
	s.Add(ms(100), core.EvAppend{Item: &core.Thinking{ID: tid, Streaming: true}})
	for _, chunk := range chunks(long, 14) {
		s.Add(ms(220), core.EvThinkingDelta{ID: tid, Delta: chunk})
	}
	_ = u
	return s.Steps()
}
