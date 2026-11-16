package term

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CmdBlock 是一个命令输出块(由 OSC 133 标出),用于终端里的装饰与复制。
type CmdBlock struct {
	StartRow int  // 输出的第一行(Screen.Lines 下标,即命令行的下一行)
	EndRow   int  // 输出的结束行(不含);命令还在跑时为 0
	ExitOK   bool // 退出码是否为 0
	Exit     int  // 退出码
	Done     bool // 命令已结束(收到 C)
}

// installShellIntegration 注册 OSC 133 处理并在合适时写入集成脚本路径到环境。
func (t *Term) installShellIntegration(shell string) {
	base := filepath.Base(shell)
	t.em.RegisterOscHandler(133, func(data []byte) bool {
		t.handleOsc133(string(data))
		return true
	})
	_ = base
	// 集成脚本在首次 Start 时落到 ~/.minicode/shell/,由 sendIntegration 在 shell 起来后 source。
}

func (t *Term) handleOsc133(payload string) {
	// vt 交给处理器的是整段载荷,带命令号前缀:"133;B"、"133;C;1"
	// (早先把 "133" 当成动作,所有动作都匹配不上,命令装饰从来没生效过)
	payload = strings.TrimPrefix(payload, "133;")
	parts := strings.Split(payload, ";")
	switch parts[0] {
	case "A", "D": // 提示符开始 / 结束
		t.cmdRunning.Store(false)
		t.pruneCmdBlocks()
	case "B": // 命令开始(preexec,光标已在输出的第一行)
		t.cmdRunning.Store(true)
		row := t.currentRow()
		t.mu.Lock()
		// 回看被清掉(clear / ⌘K 带 \e[3J)后,旧块的行号已经对不上,整体丢弃
		if n := t.em.Emulator.ScrollbackLen(); n < t.blockScroll {
			t.cmdBlocks = nil
		} else {
			t.blockScroll = n
		}
		t.cmdBlocks = append(t.cmdBlocks, CmdBlock{StartRow: row, ExitOK: true})
		t.mu.Unlock()
	case "C": // 命令结束,可带退出码
		t.cmdRunning.Store(false)
		code := 0
		if len(parts) > 1 {
			code, _ = strconv.Atoi(parts[1])
		}
		row := t.currentRow()
		t.mu.Lock()
		// 只收尾「打开中」的块:空回车也会触发 precmd,不能改写上一个已结束的块
		if n := len(t.cmdBlocks); n > 0 && !t.cmdBlocks[n-1].Done {
			b := &t.cmdBlocks[n-1]
			if row < b.StartRow { // 命令把屏幕清了(clear):这块已无意义
				t.cmdBlocks = t.cmdBlocks[:n-1]
			} else {
				b.EndRow, b.ExitOK, b.Exit, b.Done = row, code == 0, code, true
			}
		}
		t.mu.Unlock()
	}
}

// currentRow 只能在 OSC 处理器里调:此时仿真器的 Write 正持有写锁,
// 所以直接用内嵌的无锁 Emulator,绝不能走 SafeEmulator 的加锁版本(会死锁)。
func (t *Term) currentRow() int {
	pos := t.em.Emulator.CursorPosition()
	return pos.Y + t.em.Emulator.ScrollbackLen()
}

// pruneCmdBlocks 丢掉已滚出可视回看区的命令块(清屏 / 大段输出后装饰不错位)。
func (t *Term) pruneCmdBlocks() {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := t.em.Emulator.ScrollbackLen()
	keep := t.cmdBlocks[:0]
	for _, b := range t.cmdBlocks {
		if b.EndRow == 0 || b.EndRow >= n-200 {
			keep = append(keep, b)
		}
	}
	t.cmdBlocks = keep
}

// ClearCmdBlocks 清屏后重置全部命令块记录。
func (t *Term) ClearCmdBlocks() {
	t.mu.Lock()
	t.cmdBlocks = nil
	t.mu.Unlock()
}

// CmdRunning 前台是否有命令在跑(依赖 shell 集成;未注入时恒为 false)。
func (t *Term) CmdRunning() bool { return t.cmdRunning.Load() }

// CommandBlocks 返回已知的命令块快照。
func (t *Term) CommandBlocks() []CmdBlock {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]CmdBlock, len(t.cmdBlocks))
	copy(out, t.cmdBlocks)
	return out
}

// EnsureIntegrationScripts 把 zsh / bash 集成脚本写到 ~/.minicode/shell/。
func EnsureIntegrationScripts() (zsh, bash string) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".minicode", "shell")
	_ = os.MkdirAll(dir, 0o755)
	zsh = filepath.Join(dir, "integration.zsh")
	bash = filepath.Join(dir, "integration.bash")
	_ = os.WriteFile(zsh, []byte(zshIntegration), 0o644)
	_ = os.WriteFile(bash, []byte(bashIntegration), 0o644)
	return zsh, bash
}

// zsh 静默注入:不再往终端里敲 `source …` 命令(那行回显会在每个新终端里露出来,
// 慢 shell 启动时尤其明显),而是让 zsh 自己在启动文件里加载集成脚本。
//
// 做法(同 VS Code):把 ZDOTDIR 指到 ~/.minicode/shell/zdotdir,里面的 .zshenv / .zprofile /
// .zshrc 是代理,先把 ZDOTDIR 还给用户、source 用户自己的同名文件,.zshrc 代理最后加载集成脚本、
// 并把 ZDOTDIR 还原(用户原来没设就 unset),所以 .zlogin 与子 shell 都不受影响。
// 用户原来的 ZDOTDIR 经 MC_USER_ZDOTDIR 传递。
const zshProxyPrefix = `# MiniCode 代理启动文件:先还原 ZDOTDIR,source 用户自己的同名文件
__mc_base=${MC_USER_ZDOTDIR:-$HOME}
__mc_proxy=$ZDOTDIR
ZDOTDIR=$__mc_base
`

const zshProxyEnv = zshProxyPrefix + `[[ -f $ZDOTDIR/.zshenv ]] && source $ZDOTDIR/.zshenv
[[ $ZDOTDIR != $__mc_base ]] && MC_USER_ZDOTDIR=$ZDOTDIR
ZDOTDIR=$__mc_proxy
unset __mc_base __mc_proxy
`

const zshProxyProfile = zshProxyPrefix + `[[ -f $ZDOTDIR/.zprofile ]] && source $ZDOTDIR/.zprofile
[[ $ZDOTDIR != $__mc_base ]] && MC_USER_ZDOTDIR=$ZDOTDIR
ZDOTDIR=$__mc_proxy
unset __mc_base __mc_proxy
`

const zshProxyRc = zshProxyPrefix + `# /etc/zshrc 在 ZDOTDIR 还指着代理目录时已把 HISTFILE 设到了代理目录里,挪回用户目录
[[ $HISTFILE == $__mc_proxy/* ]] && HISTFILE=$__mc_base/${HISTFILE#$__mc_proxy/}
[[ -f $ZDOTDIR/.zshrc ]] && source $ZDOTDIR/.zshrc
[[ $ZDOTDIR != $__mc_base ]] && MC_USER_ZDOTDIR=$ZDOTDIR
[[ -r "$MC_SHELL_DIR/integration.zsh" ]] && source "$MC_SHELL_DIR/integration.zsh"
if [[ -n $MC_USER_ZDOTDIR ]]; then ZDOTDIR=$MC_USER_ZDOTDIR; else unset ZDOTDIR; fi
unset __mc_base __mc_proxy MC_USER_ZDOTDIR MC_SHELL_DIR
`

// writeIfChanged 内容没变就不写(多个终端同时起时避免互相截断),写入走临时文件 + 改名。
func writeIfChanged(path, content string) error {
	if old, err := os.ReadFile(path); err == nil && string(old) == content {
		return nil
	}
	tmp := path + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// zshInjectEnv 为 zsh 准备静默注入:写好代理启动文件,返回改过 ZDOTDIR 等变量的环境。
// 不是 zsh 或写文件失败时返回 ok=false,调用方退回 IntegrationCmd。
func zshInjectEnv(shell string, env []string) ([]string, bool) {
	if filepath.Base(shell) != "zsh" {
		return env, false
	}
	EnsureIntegrationScripts()
	home, _ := os.UserHomeDir()
	shellDir := filepath.Join(home, ".minicode", "shell")
	proxy := filepath.Join(shellDir, "zdotdir")
	if err := os.MkdirAll(proxy, 0o755); err != nil {
		log.Warn("集成代理目录创建失败,退回命令注入", "err", err)
		return env, false
	}
	for name, content := range map[string]string{".zshenv": zshProxyEnv, ".zprofile": zshProxyProfile, ".zshrc": zshProxyRc} {
		if err := writeIfChanged(filepath.Join(proxy, name), content); err != nil {
			log.Warn("集成代理文件写入失败,退回命令注入", "file", name, "err", err)
			return env, false
		}
	}
	orig := ""
	out := make([]string, 0, len(env)+3)
	for _, e := range env {
		switch {
		case strings.HasPrefix(e, "ZDOTDIR="):
			orig = strings.TrimPrefix(e, "ZDOTDIR=")
		case strings.HasPrefix(e, "MC_USER_ZDOTDIR="), strings.HasPrefix(e, "MC_SHELL_DIR="):
		default:
			out = append(out, e)
		}
	}
	out = append(out, "ZDOTDIR="+proxy, "MC_USER_ZDOTDIR="+orig, "MC_SHELL_DIR="+shellDir)
	return out, true
}

// quietTail 接在注入命令后面:执行完立刻清屏(含回看),
// 用户看不到那行 source 的回显,只见到干净的提示符。
const quietTail = "; printf '\\033[H\\033[2J\\033[3J'\n"

// IntegrationCmd 返回应在 shell 启动后执行的一条 source 命令(按 shell 类型)。
// 仅用于没法静默注入的 shell(bash 等);zsh 走 zshInjectEnv。
func IntegrationCmd(shell string) string {
	zsh, bash := EnsureIntegrationScripts()
	if strings.Contains(filepath.Base(shell), "bash") {
		return "source " + bash + " 2>/dev/null" + quietTail
	}
	return "source " + zsh + " 2>/dev/null" + quietTail
}

const zshIntegration = `# MiniCode shell integration (OSC 133)
__mc_osc() { printf '\033]133;%s\007' "$1"; }
__mc_precmd() {
  local ec=$?
  __mc_osc A
  __mc_osc D
  __mc_osc "C;$ec"
}
__mc_preexec() { __mc_osc B; }
autoload -Uz add-zsh-hook 2>/dev/null
if typeset -f add-zsh-hook >/dev/null 2>&1; then
  add-zsh-hook precmd __mc_precmd
  add-zsh-hook preexec __mc_preexec
fi
`

const bashIntegration = `# MiniCode shell integration (OSC 133)
__mc_osc() { printf '\033]133;%s\007' "$1"; }
__mc_skip_dbg() {
  case "$BASH_COMMAND" in
  __mc_*|trap*|PROMPT_COMMAND*) return 0;;
  esac
  __mc_osc B
}
__mc_prompt_end() {
  local ec=$?
  __mc_osc A
  __mc_osc D
  __mc_osc "C;$ec"
}
trap '__mc_skip_dbg' DEBUG
if [[ -n "$PROMPT_COMMAND" && "$PROMPT_COMMAND" != *'__mc_prompt_end'* ]]; then
  PROMPT_COMMAND="__mc_prompt_end; $PROMPT_COMMAND"
else
  PROMPT_COMMAND="__mc_prompt_end"
fi
`
