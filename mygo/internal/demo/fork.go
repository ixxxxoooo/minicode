package demo

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// ForkSession 演示模式下的分叉会话:复制父会话就绪事件,新 id。
func ForkSession(id string, speed float64) *Backend {
	parts := strings.Split(id, ":")
	sid := "demo-fork"
	if len(parts) >= 3 {
		sid = "demo-" + parts[2][:min(8, len(parts[2]))]
	}
	steps := []Step{{0, core.EvReady{SessionID: sid, Models: core.DefaultModels(), Model: "sonnet", Mode: core.ModeDefault}}}
	steps = append(steps, Step{0, core.EvTitleChanged{Title: "分叉会话"}})
	b := New(func() []Step { return steps }, speed)
	b.Instant = true
	return b
}
