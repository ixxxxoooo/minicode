package core

import "testing"

func TestTruncateFromUserEdit(t *testing.T) {
	s := NewState()
	s.Apply(EvAppend{Item: &User{ID: "u1", Text: "a"}})
	s.Apply(EvAppend{Item: &Assistant{ID: "a1", Text: "b"}})
	s.Apply(EvAppend{Item: &User{ID: "u2", Text: "c"}})
	txt, ok := s.TruncateFromUser("u1")
	if !ok || txt != "a" || len(s.Items) != 1 {
		t.Fatalf("truncate u1: ok=%v txt=%q n=%d", ok, txt, len(s.Items))
	}
}
