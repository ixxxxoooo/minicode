package term

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CmdBlock 是一个命令输出块(由 OSC 133 标出),用于终端里的装饰与复制。
type CmdBlock struct {
	StartRow int  // 命令起始行(Screen.Lines 下标)
	EndRow   int  // 结束行(不含)
	ExitOK   bool // 退出码是否为 0
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
	// VS Code / iTerm 约定:133;X 或 133;X;code
	parts := strings.Split(payload, ";")
	if len(parts) == 0 {
		return
	}
	kind := parts[0]
	switch kind {
	case "A", "D": // Prompt / prompt end
		t.cmdRunning.Store(false)
		t.pruneCmdBlocks()
	case "B": // Command start
		t.cmdRunning.Store(true)
		row := t.currentRow()
		t.mu.Lock()
		t.cmdBlocks = append(t.cmdBlocks, CmdBlock{StartRow: row, ExitOK: true})
		t.mu.Unlock()
	case "C": // Command finished
		t.cmdRunning.Store(false)
		ok := true
		if len(parts) > 1 {
			if c, err := strconv.Atoi(parts[1]); err == nil {
				ok = c == 0
			}
		}
		row := t.currentRow()
		t.mu.Lock()
		if n := len(t.cmdBlocks); n > 0 {
			t.cmdBlocks[n-1].EndRow = row
			t.cmdBlocks[n-1].ExitOK = ok
		}
		t.mu.Unlock()
	}
}

func (t *Term) currentRow() int {
	pos := t.em.CursorPosition()
	return pos.Y + t.em.ScrollbackLen()
}

// pruneCmdBlocks 丢掉已滚出可视回看区的命令块(清屏 / 大段输出后装饰不错位)。
func (t *Term) pruneCmdBlocks() {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := t.em.ScrollbackLen()
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

// quietTail 接在注入命令后面:执行完立刻清屏(含回看),
// 用户看不到那行 source 的回显,只见到干净的提示符。
const quietTail = "; printf '\\033[H\\033[2J\\033[3J'\n"

// IntegrationCmd 返回应在 shell 启动后执行的一条 source 命令(按 shell 类型)。
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
