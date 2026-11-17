package diag

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"time"
)

var runID string

// RunID 本次启动的短随机 id,写入日志与崩溃记录。
func RunID() string { return runID }

func initRunID() {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		runID = fmt.Sprintf("%d", time.Now().UnixNano()%0xffffff)
		return
	}
	runID = hex.EncodeToString(b)
}

// LogUISwitch 记录界面切换耗时(仅 VerbosePerf)。
func LogUISwitch(from, to string, d time.Duration) {
	PerfLog("ui", "界面切换", "从", from, "到", to, "耗时", d)
}

// WriteStartupEnv 启动时写一条环境摘要(不含密钥)。
func WriteStartupEnv(version, mode string) {
	initRunID()
	SetBootStarted(time.Now())
	write(LevelInfo, "diag", "环境摘要", []any{
		"run", runID,
		"版本", version,
		"模式", mode,
		"系统", runtime.GOOS,
		"架构", runtime.GOARCH,
		"CPU", runtime.NumCPU(),
		"GOMAXPROCS", runtime.GOMAXPROCS(0),
		"屏", os.Getenv("MINI_SCREEN"),
		"MINI_DEMO", os.Getenv("MINI_DEMO"),
	})
}

// ResetAllMetrics 重置性能统计(设置页「重置统计」)。
func ResetAllMetrics() {
	ResetStats()
	ResetFrameMetrics()
	ResetTrends()
	ResetHangEvents()
	ResetProcs()
}
