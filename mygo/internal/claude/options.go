// Package claude 是真实后端:直连本机 `claude` 的 stream-json 控制协议(不走 ACP、不依赖 Node)。
//
// 分层:options(启动参数与环境)→ protocol(发往 claude 的消息与应答,纯函数)→
// translate(把 claude 的每一行输出翻译成 core.Event,有状态)→ backend(进程与 goroutine 编排)→
// history(历史 jsonl 回放与会话索引)。
package claude

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// Options 是启动 `claude` 子进程所需的参数。
type Options struct {
	Cwd       string
	SessionID string // 新会话使用的 id(与 Resume 互斥;分叉时是新会话的 id)
	Resume    string // 恢复已有会话
	Fork      bool   // 恢复时分叉成新会话
	Model     string
	Mode      *core.PermissionMode
	Effort    *core.Effort
	AddDirs   []string
	// AppendSystemPrompt 追加到系统提示末尾。
	AppendSystemPrompt string
	AllowedTools       []string
	McpConfig          string
	ExtraEnv           []core.EnvVar
}

// EffectiveSessionID 是当前会话的 id:新建时是预分配的,恢复时是被恢复的那个,分叉时是新 id。
func (o Options) EffectiveSessionID() string {
	if o.Fork {
		if o.SessionID != "" {
			return o.SessionID
		}
		return o.Resume
	}
	if o.Resume != "" {
		return o.Resume
	}
	return o.SessionID
}

// Args 是命令行参数。
func (o Options) Args() []string {
	args := []string{
		"--output-format", "stream-json", "--verbose",
		"--input-format", "stream-json",
		"--include-partial-messages",
		"--permission-prompt-tool=stdio",
		// 只是放行「绕过权限」这个选项,不默认开启;没有它,运行中切到 bypassPermissions 会被 claude 拒绝
		"--allow-dangerously-skip-permissions",
		// 新模型默认不回传思考内容,要显式要求返回摘要版
		"--thinking-display", "summarized",
	}
	if o.Resume != "" && o.Fork {
		args = append(args, "--fork-session")
	}
	push := func(flag, v string) { args = append(args, flag, v) }
	if o.Resume != "" {
		push("--resume", o.Resume)
		if o.Fork && o.SessionID != "" {
			push("--session-id", o.SessionID)
		}
	} else if o.SessionID != "" {
		push("--session-id", o.SessionID)
	}
	if o.Model != "" && o.Model != "default" {
		push("--model", o.Model)
	}
	if o.Mode != nil {
		push("--permission-mode", o.Mode.CLI())
	}
	if o.Effort != nil {
		push("--effort", o.Effort.CLI())
	}
	for _, d := range o.AddDirs {
		push("--add-dir", d)
	}
	if strings.TrimSpace(o.AppendSystemPrompt) != "" {
		push("--append-system-prompt", o.AppendSystemPrompt)
	}
	launch := map[string]string{}
	for _, e := range o.ExtraEnv {
		if strings.HasPrefix(e.Key, "MINICODE_LAUNCH_") {
			launch[strings.TrimPrefix(e.Key, "MINICODE_LAUNCH_")] = e.Value
		}
	}
	if v := strings.TrimSpace(launch["AUTOCOMPACT"]); v != "" {
		push("--autocompact", v)
	}
	if len(o.AllowedTools) > 0 {
		push("--allowedTools", strings.Join(o.AllowedTools, ","))
	}
	if o.McpConfig != "" {
		push("--mcp-config", o.McpConfig)
	}
	return args
}

// Env 是需要额外注入的环境变量(KEY=VALUE)。
func (o Options) Env() []string {
	env := []string{
		"CLAUDE_CODE_ENTRYPOINT=sdk-ts",
		"CLAUDE_AGENT_SDK_VERSION=0.2.0",
		"CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS=1",
		"CLAUDE_CODE_SDK_READS_SESSION_STATE=1",
		// 开启文件检查点,「回退文件」靠它
		"CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING=1",
		// 用户 shell 里的强度变量会压过 --effort 与运行中切换,这里置空;ExtraEnv 显式设置的仍然生效
		"CLAUDE_CODE_EFFORT_LEVEL=",
	}
	for _, e := range o.ExtraEnv {
		env = append(env, e.Key+"="+e.Value)
	}
	return env
}

func executable(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir() && st.Mode()&0o111 != 0
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

// FindClaude 查找 claude 可执行文件:用户配置的路径 > MINI_CLAUDE_PATH > PATH > 常见安装位置。
func FindClaude(custom string) (string, bool) {
	if c := strings.TrimSpace(custom); c != "" {
		if p := expandHome(c); executable(p) {
			return p, true
		}
	}
	if p := os.Getenv("MINI_CLAUDE_PATH"); p != "" && executable(p) {
		return p, true
	}
	if p, err := exec.LookPath("claude"); err == nil {
		return p, true
	}
	home, _ := os.UserHomeDir()
	for _, p := range []string{
		filepath.Join(home, ".local/bin/claude"), filepath.Join(home, ".claude/local/claude"),
		"/opt/homebrew/bin/claude", "/usr/local/bin/claude",
	} {
		if executable(p) {
			return p, true
		}
	}
	return "", false
}
