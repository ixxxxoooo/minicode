package core

import "testing"

// 实时会话里思考结束要记下用时(后端不发 EvDuration),否则结束后的思考行右侧没有耗时。
func TestThinkingDurationRecorded(t *testing.T) {
	s := NewState()
	now := int64(1_000_000)
	s.Now = func() int64 { return now }

	s.Apply(EvThinkingDelta{ID: "th1", Delta: "先看路由表"})
	now += 4200
	s.Apply(EvThinkingDelta{ID: "th1", Delta: "。再看测试"})
	now += 300
	s.Apply(EvItemDone{ID: "th1"})
	if got := s.Durations["th1"]; got != 5 { // 4.5 秒四舍五入
		t.Fatalf("思考用时应记 5 秒,实际 %d", got)
	}

	// 很短的思考至少记 1 秒,不显示成 0
	s.Apply(EvThinkingDelta{ID: "th2", Delta: "嗯"})
	now += 100
	s.Apply(EvItemDone{ID: "th2"})
	if got := s.Durations["th2"]; got != 1 {
		t.Fatalf("很短的思考应记 1 秒,实际 %d", got)
	}

	// 回合收尾时还在流式的思考也要记
	s.Apply(EvThinkingDelta{ID: "th3", Delta: "想"})
	now += 7000
	s.finishStreaming(false)
	if got := s.Durations["th3"]; got != 7 {
		t.Fatalf("收尾时的思考应记 7 秒,实际 %d", got)
	}
}

// 历史回放给了用时(EvDuration)就不覆盖;回放期间没有起点,也不自己算。
func TestThinkingDurationReplayKeeps(t *testing.T) {
	s := NewState()
	now := int64(5_000)
	s.Now = func() int64 { return now }

	s.Apply(EvReplay{On: true})
	s.Apply(EvThinkingDelta{ID: "h1", Delta: "历史思考"})
	now += 60_000
	s.Apply(EvItemDone{ID: "h1"})
	s.Apply(EvReplay{On: false})
	if _, ok := s.Durations["h1"]; ok {
		t.Fatal("回放期间不该自己算用时")
	}
	s.Apply(EvDuration{ID: "h1", Secs: 9})
	if s.Durations["h1"] != 9 {
		t.Fatal("EvDuration 给的用时应生效")
	}

	// 实时思考里先到了 EvDuration:不被覆盖
	s.Apply(EvThinkingDelta{ID: "l1", Delta: "x"})
	s.Apply(EvDuration{ID: "l1", Secs: 3})
	now += 20_000
	s.Apply(EvItemDone{ID: "l1"})
	if s.Durations["l1"] != 3 {
		t.Fatalf("已有记录不该被覆盖,实际 %d", s.Durations["l1"])
	}
}
