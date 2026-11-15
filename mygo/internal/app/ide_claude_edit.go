package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

const haikuInline = "haiku"

var claudeEditLog = diag.L("claude.edit")

func (a *App) beginClaudeJob() context.Context {
	if a.claudeJob != nil {
		a.claudeJob()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.claudeJob = cancel
	return ctx
}

func (a *App) endClaudeJob() {
	a.claudeJob = nil
}

// cancelClaudeJob Esc 取消进行中的 Haiku 请求。
func (a *App) cancelClaudeJob() {
	if a.claudeJob != nil {
		a.claudeJob()
		a.claudeJob = nil
		a.toast = "已取消 Claude 请求"
	}
}

func inlineEditPrompt(old, instruction string) string {
	var b strings.Builder
	b.WriteString("你是代码编辑助手。只输出要替换进编辑器的代码片段,不要解释、不要 markdown 围栏、不要引号包裹。\n\n")
	b.WriteString("用户要求:" + instruction + "\n\n当前选中的代码:\n")
	b.WriteString(old)
	return b.String()
}

func suggestPrompt(path, line, prefix string) string {
	return fmt.Sprintf("你是代码补全助手。只输出要接在光标后的补全文本(不含已有前缀),不要解释、不要换行、不要围栏。\n文件:%s\n当前行:%s\n光标前词:%s",
		path, line, prefix)
}
