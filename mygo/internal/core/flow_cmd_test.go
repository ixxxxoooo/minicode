package core

import "testing"

func TestTruncateFromUser(t *testing.T) {
	s := NewState()
	s.Apply(EvAppend{Item: &User{ID: "u1", Text: "你好"}})
	s.Apply(EvAppend{Item: &Assistant{ID: "a1", Text: "回复"}})
	s.Apply(EvAppend{Item: &User{ID: "u2", Text: "第二问"}})
	txt, ok := s.TruncateBeforeUser("u2")
	if !ok || txt != "第二问" || len(s.Items) != 2 {
		t.Fatalf("截断失败: ok=%v txt=%q n=%d", ok, txt, len(s.Items))
	}
	if s.Phase != PhaseIdle {
		t.Fatalf("截断后应回到空闲")
	}
}
