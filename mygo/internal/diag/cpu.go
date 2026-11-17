package diag

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	cpuMu         sync.Mutex
	lastRusageCPU time.Duration
	lastRusageAt  time.Time
)

// sampleProcessCPU 返回本进程 CPU 占用(100=单核满载);先 ps,失败则用 rusage 增量。
func sampleProcessCPU() float64 {
	if v := psFloat("pcpu"); v >= 0 {
		return v
	}
	return rusageCPUPercent()
}

func psFloat(field string) float64 {
	out, err := exec.Command("ps", "-o", field+"=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		return -1
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return -1
	}
	return f
}

// sampleProcessRSS 当前常驻内存(字节);macOS ps rss 单位为 KB。
func sampleProcessRSS() uint64 {
	out, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err == nil {
		if kb, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64); err == nil && kb > 0 {
			return kb * 1024
		}
	}
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	return uint64(ru.Maxrss)
}

func timevalDur(t syscall.Timeval) time.Duration {
	return time.Duration(t.Sec)*time.Second + time.Duration(t.Usec)*time.Microsecond
}

func rusageCPUPercent() float64 {
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	cpu := timevalDur(ru.Utime) + timevalDur(ru.Stime)
	now := time.Now()
	cpuMu.Lock()
	defer cpuMu.Unlock()
	if lastRusageAt.IsZero() {
		lastRusageCPU, lastRusageAt = cpu, now
		return 0
	}
	dt := now.Sub(lastRusageAt)
	dCPU := cpu - lastRusageCPU
	lastRusageCPU, lastRusageAt = cpu, now
	if dt <= 0 {
		return 0
	}
	return float64(dCPU) / float64(dt) * 100
}
