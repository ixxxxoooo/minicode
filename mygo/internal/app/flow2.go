package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/core"
)

// assistant 是助手正文:已结束用 markdown.View,流式中用 markdown.Stream(逐词淡入)。
func (a *App) assistant(c *ui.Context, m *core.Assistant) {
	if strings.TrimSpace(m.Text) == "" {
		return
	}
	box := ui.Column(c).Key("asst-" + m.ID)
	box.Children(func() {
		if m.Streaming {
			markdown.Stream(c, padCode(m.Text), a.mdOptions())
		} else {
			markdown.View(c, padCode(m.Text), a.mdOptions())
		}
	})
}

// assistantChunk 画长正文的一块;流式输出时只有最后一块走流式渲染,前面的块内容已定,走带缓存的静态渲染。
func (a *App) assistantChunk(c *ui.Context, m *core.Assistant, text string, last bool) {
	if strings.TrimSpace(text) == "" {
		return
	}
	ui.Column(c).Key("asst-" + m.ID).Children(func() { // 每块在自己的行上下文里,键不会冲突;不能带随内容变化的值
		if m.Streaming && last {
			markdown.Stream(c, padCode(text), a.mdOptions())
		} else {
			markdown.View(c, padCode(text), a.mdOptions())
		}
	})
}

// turnEnd 返回 items[i] 是不是某个回合的最后一项(下一项是用户消息或已到末尾),以及该回合的用户消息 id。
func turnEnd(items []core.Item, i int) (string, bool) {
	if i+1 < len(items) {
		if _, ok := items[i+1].(*core.User); !ok {
			return "", false
		}
	}
	for k := i; k >= 0; k-- {
		if u, ok := items[k].(*core.User); ok {
			return u.ID, true
		}
	}
	return "", false
}

// padCode 给行内代码两侧各补一个窄空格,让底色比文字宽出一点(设计稿的行内代码有 1×5 px 内边距,
// Markdown 组件没有这个选项)。围栏代码块里的反引号不处理;窄空格不会被 CommonMark 当作可裁掉的空白。
func padCode(src string) string {
	const thin = "\u2009"
	var b strings.Builder
	fenced := false
	for _, line := range strings.SplitAfter(src, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " "), "```") {
			fenced = !fenced
			b.WriteString(line)
			continue
		}
		if fenced || !strings.Contains(line, "`") {
			b.WriteString(line)
			continue
		}
		in := false
		for _, r := range line {
			if r == '`' {
				if !in {
					b.WriteString("`" + thin)
				} else {
					b.WriteString(thin + "`")
				}
				in = !in
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}
