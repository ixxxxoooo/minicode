package kit

import (
	"testing"
	"time"
)

func TestUndoSlot(t *testing.T) {
	var u UndoSlot
	called := false
	u.Begin("删除", func() { called = true })
	if !u.Active() {
		t.Fatal("应处于可撤销")
	}
	if fn := u.Undo; fn != nil {
		fn()
		called = true
	}
	u.Clear()
	if u.Active() {
		t.Fatal("清空后不应激活")
	}
	_ = called
}

func TestUndoExpire(t *testing.T) {
	var u UndoSlot
	u.Begin("x", func() {})
	u.Until = time.Now().Add(-time.Second)
	if u.Active() {
		t.Fatal("过期")
	}
}
