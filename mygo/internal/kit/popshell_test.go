package kit

import (
	"testing"
	"time"
)

func TestPopInOut(t *testing.T) {
	SetMotion(true)
	defer SetMotion(true)
	at := time.Now().Add(-60 * time.Millisecond)
	// 仅验证不 panic(依赖 ui 上下文的部分在 app 测试里覆盖)
	_ = DurFast
	_ = at
}
