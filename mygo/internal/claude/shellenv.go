package claude

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// MCP 配置里常用 ${NAME}(或 ${NAME:-默认值})引用环境变量,例如 headers 里的 Cookie / Authorization。
// Claude Code 在自己的进程环境里展开它们;但从访达 / Dock 启动的本应用拿不到你在 ~/.zshrc 里 export 的变量
// (图形应用只继承 launchd 的环境),结果是 ${NAME} 展开成空,服务器回登录页,报「JSON Parse error: Unrecognized token '<'」。
// 这里在启动 claude 之前,把配置里引用到、但本进程环境里没有的变量,从登录 shell 里补上。

var shellLog = diag.L("shellenv")

var envRefRE = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::?-[^}]*)?\}`)

// EnvRefNames 取一段文本里所有 ${NAME} 引用的变量名(去重、排序)。
func EnvRefNames(s string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range envRefRE.FindAllStringSubmatch(s, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	sort.Strings(out)
	return out
}

// MCPEnvRefs 扫描用户层(~/.claude.json)、该项目在 ~/.claude.json 里的条目与项目层 .mcp.json 的 mcpServers,
// 返回其中引用的环境变量名。
func MCPEnvRefs(cwd string) []string {
	var sections []any
	home, _ := os.UserHomeDir()
	var root map[string]any
	if b, err := os.ReadFile(filepath.Join(home, ".claude.json")); err == nil && json.Unmarshal(b, &root) == nil {
		sections = append(sections, root["mcpServers"])
		if pr, ok := root["projects"].(map[string]any); ok && cwd != "" {
			if ent, ok := pr[cwd].(map[string]any); ok {
				sections = append(sections, ent["mcpServers"])
			}
		}
	}
	if cwd != "" {
		var pj map[string]any
		if b, err := os.ReadFile(filepath.Join(cwd, ".mcp.json")); err == nil && json.Unmarshal(b, &pj) == nil {
			sections = append(sections, pj["mcpServers"])
		}
	}
	b, _ := json.Marshal(sections)
	return EnvRefNames(string(b))
}

type shellEnvEntry struct {
	val string
	ok  bool
	at  time.Time
}

var (
	shellEnvMu    sync.Mutex
	shellEnvCache = map[string]shellEnvEntry{}
)

const shellEnvTTL = 5 * time.Minute

var envNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ShellEnv 从登录 shell(zsh -lic)里取一个环境变量的值;没有或为空返回 ok=false。结果缓存 5 分钟。
// 只在变量名合法时才拼进命令;值不写日志。
func ShellEnv(name string) (string, bool) {
	if !envNameRE.MatchString(name) {
		return "", false
	}
	shellEnvMu.Lock()
	if e, hit := shellEnvCache[name]; hit && time.Since(e.at) < shellEnvTTL {
		shellEnvMu.Unlock()
		return e.val, e.ok
	}
	shellEnvMu.Unlock()
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/zsh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	// 用 \001 / \002 夹住值,zshrc 里的 echo 之类的杂音就不会混进来
	cmd := exec.CommandContext(ctx, sh, "-lic", `printf '\001%s\002' "${`+name+`}"`)
	out, err := cmd.Output()
	val, ok := "", false
	if err == nil {
		s := string(out)
		if i := strings.LastIndexByte(s, '\001'); i >= 0 {
			if j := strings.IndexByte(s[i:], '\002'); j > 0 {
				val = s[i+1 : i+j]
				ok = val != ""
			}
		}
	}
	shellLog.Debug("登录 shell 取环境变量", "name", name, "found", ok, "err", err)
	shellEnvMu.Lock()
	shellEnvCache[name] = shellEnvEntry{val: val, ok: ok, at: time.Now()}
	shellEnvMu.Unlock()
	return val, ok
}

// ForgetShellEnv 清掉缓存(用户说「重新检测」时)。
func ForgetShellEnv() {
	shellEnvMu.Lock()
	shellEnvCache = map[string]shellEnvEntry{}
	shellEnvMu.Unlock()
}

// EnvRefSource 说明一个变量在哪里能取到:"app" = 本进程环境、"shell" = 只在登录 shell 里(启动 claude 时会补上)、"" = 都没有。
func EnvRefSource(name string) string {
	if v, ok := os.LookupEnv(name); ok && v != "" {
		return "app"
	}
	if _, ok := ShellEnv(name); ok {
		return "shell"
	}
	return ""
}

// WithMCPEnv 返回 env 加上 MCP 配置引用到、但 env 里没有的变量(从登录 shell 补)。cwd 用来找项目层配置。
// 没有缺失的变量就不起子进程。
func WithMCPEnv(env []string, cwd string) []string {
	have := map[string]bool{}
	for _, e := range env {
		if k, v, ok := strings.Cut(e, "="); ok && v != "" {
			have[k] = true
		}
	}
	var added []string
	for _, name := range MCPEnvRefs(cwd) {
		if have[name] {
			continue
		}
		if v, ok := ShellEnv(name); ok {
			env = append(env, name+"="+v)
			added = append(added, name)
		}
	}
	if len(added) > 0 {
		shellLog.Info("从登录 shell 补上 MCP 配置引用的环境变量", "names", strings.Join(added, ","))
	}
	return env
}
