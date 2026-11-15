package app

import "testing"

func TestMorphOpacityIdle(t *testing.T) {
	a := New(Options{})
	if op := a.morphOpacity("ws"); op != 1 {
		t.Fatalf("ws opacity=%v want 1", op)
	}
}
