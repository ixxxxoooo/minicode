package diag

import (
	"sync/atomic"
	"time"
)

var verbosePerf atomic.Bool

// SetVerbosePerf 开关详细性能日志(设置 → 性能与诊断)。
func SetVerbosePerf(on bool) { verbosePerf.Store(on) }

// VerbosePerf 是否记录界面切换、慢帧详情、后台任务等。
func VerbosePerf() bool { return verbosePerf.Load() }

// PerfLog 仅在 VerbosePerf 时写 Info 级日志(带 run id)。
func PerfLog(module, msg string, kv ...any) {
	if !verbosePerf.Load() {
		return
	}
	write(LevelInfo, module, msg, append([]any{"run", RunID()}, kv...))
}

// PerfFrame 记录超阈值或 verbose 下的帧详情。
func PerfFrame(build time.Duration, gap time.Duration, slow bool) {
	if !verbosePerf.Load() && !slow {
		return
	}
	lv := LevelDebug
	if slow {
		lv = LevelInfo
	}
	write(lv, "perf", "帧", []any{"run", RunID(), "构建", build, "间隔", gap})
}
