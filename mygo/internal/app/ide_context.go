package app

import (
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
)

// 偏好:发消息时把 IDE 里的当前文件 / 选区 / 打开的标签告诉 Claude(默认开)。
const pIDECtx = "chat.ideCtx"

const (
	ideCtxMaxSel   = 4000 // 选区文本上限(字符)
	ideCtxMaxTabs  = 8    // 列出的其他打开文件数上限
	ideCtxMaxLines = 120  // 选区文本最多行数
)

// ideContext 拼出随下一条消息发给模型的 IDE 上下文;没有可说的(没开文件、偏好关闭、演示)返回空串。
// 只含相对路径、光标行、选区文本(限长)和分支名,不含任何密钥或环境变量。
func (a *App) ideContext() string {
	if a == nil || a.ide == nil || a.runDemo || (a.prefs != nil && !a.prefs.Bool(pIDECtx, true)) {
		return ""
	}
	tb := a.ide.cur()
	if tb == nil || tb.diff || tb.virt != "" || tb.buf == nil || tb.path == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("用户此刻在 IDE 里的状态(仅供参考,和本次提问无关时请忽略;路径相对项目根):\n")
	b.WriteString("- 当前文件:" + tb.path)
	if tb.st != nil {
		cur := tb.buf.Pos(tb.st.Cursor())
		b.WriteString("(光标在第 " + itoa(cur.Line+1) + " 行)")
	}
	if tb.dirty() {
		b.WriteString(" [有未保存的修改,磁盘上的内容可能与编辑器不同]")
	}
	b.WriteString("\n")
	if tb.st != nil {
		s := tb.st.Selection()
		lo, hi := s.Anchor, s.Head
		if lo > hi {
			lo, hi = hi, lo
		}
		if lo != hi {
			pa, pb := tb.buf.Pos(lo), tb.buf.Pos(hi)
			text := tb.buf.Slice(lo, hi)
			note := ""
			if r := []rune(text); len(r) > ideCtxMaxSel {
				text, note = string(r[:ideCtxMaxSel]), "(已截断)"
			}
			if lines := strings.Split(text, "\n"); len(lines) > ideCtxMaxLines {
				text, note = strings.Join(lines[:ideCtxMaxLines], "\n"), "(已截断)"
			}
			b.WriteString("- 选中的代码(第 " + itoa(pa.Line+1) + "–" + itoa(pb.Line+1) + " 行)" + note + ":\n```\n" + text + "\n```\n")
		}
	}
	var others []string
	seen := map[string]bool{tb.path: true}
	for _, group := range [][]*ideTab{a.ide.tabs, a.ide.tabs2} {
		for _, t := range group {
			if t == nil || t.diff || t.virt != "" || t.path == "" || seen[t.path] {
				continue
			}
			seen[t.path] = true
			if len(others) < ideCtxMaxTabs {
				others = append(others, t.path)
			}
		}
	}
	if len(others) > 0 {
		b.WriteString("- 其他打开的文件:" + strings.Join(others, "、") + "\n")
	}
	if a.branch != "" {
		b.WriteString("- Git 分支:" + a.branch + "\n")
	}
	return b.String()
}

// attachIDEContext 给一条要发出的提示附上 IDE 上下文。同一份上下文在本会话里只发一次(模型已经在历史里有了),
// 内容变化(换文件 / 换选区 / 开关标签)后下一条再发。
func (a *App) attachIDEContext(p core.CmdPrompt) core.CmdPrompt {
	if p.Context != "" {
		return p
	}
	ctx := a.ideContext()
	if ctx == "" || ctx == a.ctxSent {
		return p
	}
	a.ctxSent = ctx
	p.Context = ctx
	diag.L("app").Info("随消息附带 IDE 上下文", "字符数", len(ctx))
	return p
}
