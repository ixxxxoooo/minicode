package core

import (
	"fmt"
	"strings"
)

// ExportMarkdown 把一个会话导出成 Markdown(「复制为 Markdown」用),返回文本与条目数。
// 用户消息与助手正文原样输出;思考省略;工具调用折成一行引用;计划原文保留;提示 / 画布 / 待办略。
func ExportMarkdown(title string, items []Item) (string, int) {
	var b strings.Builder
	if title != "" {
		fmt.Fprintf(&b, "## %s\n\n", title)
	}
	n := 0
	for _, it := range items {
		switch x := it.(type) {
		case *User:
			if strings.TrimSpace(x.Text) == "" && len(x.Files) == 0 {
				continue
			}
			b.WriteString("**你**\n")
			if x.Text != "" {
				b.WriteString(x.Text + "\n")
			}
			for _, f := range x.Files {
				fmt.Fprintf(&b, "@%s\n", f)
			}
			b.WriteString("\n")
		case *Assistant:
			if strings.TrimSpace(x.Text) == "" {
				continue
			}
			b.WriteString("**Claude**\n" + strings.TrimSpace(x.Text) + "\n\n")
		case *ToolCall:
			b.WriteString("> " + toolLine(x) + "\n\n")
		case *PlanApproval:
			if strings.TrimSpace(x.Markdown) != "" {
				b.WriteString("**计划**\n" + strings.TrimSpace(x.Markdown) + "\n\n")
			}
		case *Changes:
			if len(x.Files) == 0 {
				continue
			}
			var fs []string
			for _, f := range x.Files {
				fs = append(fs, fmt.Sprintf("%s(+%d −%d)", f.Path, f.Added, f.Removed))
			}
			b.WriteString("> 本轮改动:" + strings.Join(fs, "、") + "\n\n")
		default:
			continue
		}
		n++
	}
	return strings.TrimRight(b.String(), "\n") + "\n", n
}

func toolLine(t *ToolCall) string {
	s := t.Title
	if s == "" {
		s = t.Name
	}
	switch t.Status {
	case ToolFailed:
		s += " · 失败"
	}
	return s
}
