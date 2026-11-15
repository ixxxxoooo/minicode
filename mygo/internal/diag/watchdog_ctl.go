package diag

import "sync/atomic"

var watchdogEnabled atomic.Bool

func init() {
	watchdogEnabled.Store(true)
}

// SetWatchdogEnabled 开关主线程卡死看门狗(设置 → 高级 → 实验功能)。
func SetWatchdogEnabled(on bool) { watchdogEnabled.Store(on) }

// WatchdogEnabled 当前是否启用看门狗。
func WatchdogEnabled() bool { return watchdogEnabled.Load() }
