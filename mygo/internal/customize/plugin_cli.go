package customize

import (
	"bytes"
	"context"
	"encoding/json"
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

// RunPluginUninstall 调用 claude plugin uninstall。
func RunPluginUninstall(name string) PluginInstallResult {
	path, ok := claude.FindClaude("")
	if !ok {
		return PluginInstallResult{Err: "没有找到 claude 命令"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "plugin", "uninstall", name)
	cmd.Env = os.Environ()
	var errb bytes.Buffer
	cmd.Stderr = &errb
	err := cmd.Run()
	code := 0
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	msg := strings.TrimSpace(errb.String())
	if err != nil && msg == "" {
		msg = err.Error()
	}
	alog.Info("plugin uninstall", "name", name, "code", code)
	return PluginInstallResult{Code: code, Err: msg}
}

// RunMarketplaceList 读取插件市场列表;失败返回 nil 由调用方兜底演示数据。
func RunMarketplaceList() []*Market {
	path, ok := claude.FindClaude("")
	if !ok {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "plugin", "marketplace", "list")
	cmd.Env = os.Environ()
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil || out.Len() == 0 {
		alog.Warn("marketplace list 失败", "err", err)
		return nil
	}
	raw := strings.TrimSpace(out.String())
	var list []*Market
	if strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{") {
		var arr []struct {
			Name, ID, Description string
		}
		if json.Unmarshal([]byte(raw), &arr) == nil {
			for _, e := range arr {
				n := e.Name
				if n == "" {
					n = e.ID
				}
				if n == "" {
					continue
				}
				list = append(list, &Market{Name: n, By: "市场", Desc: e.Description})
			}
			if len(list) > 0 {
				return list
			}
		}
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Available") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		list = append(list, &Market{Name: parts[0], By: "市场", Desc: strings.Join(parts[1:], " ")})
	}
	return list
}

// DiskStore.Reload 重新从磁盘读取(安装插件后)。
func (ds *DiskStore) Reload() {
	ds.items = map[Cat][]*Item{}
	if mk := ReadMarketplaces(ds.home); len(mk) > 0 {
		ds.Market = mk // 重新读,「已安装」标记才会跟着装 / 卸载刷新
	}
	ds.loadAll()
}

// runClaudeCLI 在 dir 下执行一条 claude 子命令,返回退出码与 stderr 摘要(只留末尾 240 字)。
func runClaudeCLI(timeout time.Duration, dir string, args ...string) PluginInstallResult {
	path, ok := claude.FindClaude("")
	if !ok {
		return PluginInstallResult{Err: "没有找到 claude 命令"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	start := time.Now()
	err := cmd.Run()
	code := 0
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	alog.Info("claude 子命令", "args", strings.Join(args, " "), "code", code, "ms", time.Since(start).Milliseconds())
	if ctx.Err() == context.DeadlineExceeded {
		return PluginInstallResult{Code: -1, Err: "执行超时"}
	}
	msg := strings.TrimSpace(out.String())
	if r := []rune(msg); len(r) > 240 {
		msg = string(r[len(r)-240:])
	}
	if err != nil && msg == "" {
		msg = err.Error()
	}
	if err == nil {
		msg = ""
	}
	return PluginInstallResult{Code: code, Err: msg}
}

// RunPluginToggle 启用 / 停用已安装的插件(claude plugin enable|disable,写入 enabledPlugins)。
func RunPluginToggle(id string, on bool, proj string) PluginInstallResult {
	verb := "disable"
	if on {
		verb = "enable"
	}
	return runClaudeCLI(2*time.Minute, proj, "plugin", verb, id)
}

// RunMarketplaceAdd 添加一个插件市场:本地目录、owner/repo(GitHub)或 git 地址。
func RunMarketplaceAdd(src string) PluginInstallResult {
	return runClaudeCLI(5*time.Minute, "", "plugin", "marketplace", "add", strings.TrimSpace(src))
}
