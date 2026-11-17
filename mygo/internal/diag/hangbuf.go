package diag

import (
	"sync"
	"time"
)

// HangEvent 是看门狗记录的停滞摘要(内存里保留最近 N 条)。
type HangEvent struct {
	At      time.Time
	Stall   time.Duration
	Stack   string
	Kind    string // stall / dump / recovered
	LogPath string // 完整栈文件(若有)
}

const hangKeep = 32

var (
	hangMu sync.Mutex
	hangs  []HangEvent
)

// RecordHang 写入一条停滞记录(看门狗调用)。
func RecordHang(kind string, stall time.Duration, stack, logPath string) {
	hangMu.Lock()
	hangs = append(hangs, HangEvent{
		At:      time.Now(),
		Stall:   stall,
		Stack:   stack,
		Kind:    kind,
		LogPath: logPath,
	})
	if len(hangs) > hangKeep {
		hangs = hangs[len(hangs)-hangKeep:]
	}
	hangMu.Unlock()
}

// HangEvents 返回停滞记录(最新在前)。
func HangEvents() []HangEvent {
	hangMu.Lock()
	defer hangMu.Unlock()
	out := make([]HangEvent, len(hangs))
	for i := range hangs {
		out[len(hangs)-1-i] = hangs[i]
	}
	return out
}

// ResetHangEvents 清空(测试与重置)。
func ResetHangEvents() {
	hangMu.Lock()
	hangs = nil
	hangMu.Unlock()
}
