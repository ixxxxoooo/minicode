package customize

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/claude"
)

// PluginInstallResult 是一次插件安装的结果。
type PluginInstallResult struct {
	Code int
	Err  string
}

// RunPluginInstall 调用 claude plugin install。
func RunPluginInstall(name, marketplace string) PluginInstallResult {
	spec := name
	if marketplace != "" && !strings.Contains(name, "@") {
		spec = name + "@" + marketplace
	}
	path, ok := claude.FindClaude("")
	if !ok {
		return PluginInstallResult{Err: "没有找到 claude 命令"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "plugin", "install", spec)
	cmd.Env = os.Environ()
	var errb bytes.Buffer
	cmd.Stderr = &errb
	start := time.Now()
	err := cmd.Run()
	code := 0
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	alog.Info("plugin install", "spec", spec, "code", code, "ms", time.Since(start).Milliseconds())
	if ctx.Err() == context.DeadlineExceeded {
		return PluginInstallResult{Code: -1, Err: "安装超时"}
	}
	msg := strings.TrimSpace(errb.String())
	if r := []rune(msg); len(r) > 240 {
		msg = string(r[len(r)-240:])
	}
	if err != nil && msg == "" {
		msg = err.Error()
	}
	return PluginInstallResult{Code: code, Err: msg}
}

// DiskStore.Reload 重新从磁盘读取(安装插件后)。
func (ds *DiskStore) Reload() {
	ds.items = map[Cat][]*Item{}
	ds.loadAll()
}
