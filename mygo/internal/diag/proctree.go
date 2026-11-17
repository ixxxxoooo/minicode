package diag

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ChildProc 是本进程的子进程快照行。
type ChildProc struct {
	Name string
	PID  int
	CPU  float64
	RSS  uint64 // 字节
	Cmd  string
}

var (
	procMu      sync.Mutex
	procList    []ChildProc
	procStarted atomic.Bool
)

const procRefresh = 8 * time.Second

// StartProctreeSampler 周期性刷新子进程表。
func StartProctreeSampler() {
	if !procStarted.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer Recover("diag.proctree")
		for {
			refreshChildProcs(os.Getpid())
			time.Sleep(procRefresh)
		}
	}()
}

// ChildProcs 返回最近一次采样的子进程(最新在前)。
func ChildProcs() []ChildProc {
	procMu.Lock()
	defer procMu.Unlock()
	return append([]ChildProc(nil), procList...)
}

func refreshChildProcs(root int) {
	raw, err := exec.Command("ps", "-ax", "-o", "pid=,ppid=,pcpu=,rss=,command=").Output()
	if err != nil {
		return
	}
	type row struct {
		pid, ppid int
		cpu       float64
		rssKB     uint64
		cmd       string
	}
	byPID := map[int]row{}
	children := map[int][]int{}
	for _, ln := range bytes.Split(raw, []byte{'\n'}) {
		s := strings.TrimSpace(string(ln))
		if s == "" {
			continue
		}
		parts := strings.Fields(s)
		if len(parts) < 5 {
			continue
		}
		pid, _ := strconv.Atoi(parts[0])
		ppid, _ := strconv.Atoi(parts[1])
		cpu, _ := strconv.ParseFloat(parts[2], 64)
		rss, _ := strconv.ParseUint(parts[3], 10, 64)
		cmd := strings.Join(parts[4:], " ")
		byPID[pid] = row{pid: pid, ppid: ppid, cpu: cpu, rssKB: rss, cmd: cmd}
		children[ppid] = append(children[ppid], pid)
	}
	var desc []int
	var walk func(int)
	walk = func(p int) {
		for _, c := range children[p] {
			desc = append(desc, c)
			walk(c)
		}
	}
	walk(root)
	var out []ChildProc
	for _, pid := range desc {
		r, ok := byPID[pid]
		if !ok {
			continue
		}
		out = append(out, ChildProc{
			Name: classifyCmd(r.cmd),
			PID:  r.pid,
			CPU:  r.cpu,
			RSS:  r.rssKB * 1024,
			Cmd:  trimCmd(r.cmd),
		})
	}
	procMu.Lock()
	procList = out
	procMu.Unlock()
}

func classifyCmd(cmd string) string {
	low := strings.ToLower(cmd)
	switch {
	case strings.Contains(low, "claude"):
		return "Claude"
	case strings.Contains(low, "gopls"), strings.Contains(low, "rust-analyzer"), strings.Contains(low, "pyright"), strings.Contains(low, "typescript-language-server"):
		return "语言服务"
	case strings.Contains(low, "zsh"), strings.Contains(low, "bash"), strings.Contains(low, "/bin/sh"):
		return "终端"
	case strings.Contains(low, "git"):
		return "Git"
	default:
		return "子进程"
	}
}

func trimCmd(cmd string) string {
	if len(cmd) > 80 {
		return cmd[:77] + "..."
	}
	return cmd
}

// ResetProcs 清空子进程缓存(测试)。
func ResetProcs() {
	procMu.Lock()
	procList = nil
	procMu.Unlock()
}

// FormatProcLine 供日志 / 诊断。
func FormatProcLine(p ChildProc) string {
	return fmt.Sprintf("%s pid=%d cpu=%s rss=%s", p.Name, p.PID, FormatCPU(p.CPU), FormatBytes(p.RSS))
}
