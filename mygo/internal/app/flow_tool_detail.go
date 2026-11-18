package app

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 通用工具详情:任何工具调用(MCP / 浏览器 / 读取 / 搜索 / 抓取…)都能展开看到
// 「入参」与「结果」,不再只有编辑与命令才有展开体。

// toolHasDetail 这个工具调用有没有可以展开看的内容(入参或结果)。
func toolHasDetail(tc *core.ToolCall) bool {
	return tc != nil && (len(tc.Input) > 0 || len(tc.Content) > 0)
}

// toolRichBody 工具自己带专用展开体(差异 / 终端输出)。
func toolRichBody(tc *core.ToolCall) bool {
	return hasContent[*core.DiffBlock](tc) || hasContent[*core.TerminalBlock](tc)
}

// formatToolInput 把入参排成「键: 值」多行文本:字符串值原样(多行缩进),其余用紧凑 JSON;键按字母序,稳定可比对。
func formatToolInput(in map[string]any) string {
	if len(in) == 0 {
		return ""
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		switch v := in[k].(type) {
		case string:
			if strings.Contains(v, "\n") {
				b.WriteString(k + ":\n")
				for _, ln := range strings.Split(strings.TrimRight(v, "\n"), "\n") {
					b.WriteString("  " + ln + "\n")
				}
			} else {
				b.WriteString(k + ": " + v + "\n")
			}
		default:
			raw, err := json.MarshalIndent(v, "  ", "  ")
			if err != nil {
				raw = []byte(fmt.Sprint(v))
			}
			b.WriteString(k + ": " + string(raw) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// toolMetaLine 详情底部的一行元信息:完整工具名 · 状态 · 耗时。
func toolMetaLine(tc *core.ToolCall) string {
	parts := []string{tc.Name}
	switch tc.Status {
	case core.ToolRunning, core.ToolPending:
		parts = append(parts, "运行中")
	case core.ToolCompleted:
		parts = append(parts, "已完成")
	case core.ToolFailed:
		parts = append(parts, "失败")
	case core.ToolDenied:
		parts = append(parts, "已拒绝")
	case core.ToolInterrupted:
		parts = append(parts, "已中断")
	}
	if d := durText(tc.DurationMs); d != "" {
		parts = append(parts, d)
	}
	return strings.Join(parts, " · ")
}

// toolDetail 画通用展开体:入参卡 + 结果卡 + 元信息行。
func (a *App) toolDetail(c *ui.Context, tc *core.ToolCall) {
	t := a.tk
	ui.Column(c).Gap(6).Children(func() {
		if in := formatToolInput(tc.Input); in != "" {
			a.detailSection(c, "入参", func() {
				a.card(c, nil, func() {
					a.foldBody(c, foldKey("tin", tc.ID), in, false, true, func() { a.copyOut(in, "入参") })
				})
			})
		}
		a.detailSection(c, "结果", func() { a.toolResult(c, tc) })
		ui.Text(c, toolMetaLine(tc)).Font("monospace").FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine()
	})
}

// detailSection 是详情里的一节:小标题 + 内容。
func (a *App) detailSection(c *ui.Context, title string, body func()) {
	ui.Column(c).Gap(3).Children(func() {
		ui.Text(c, title).FontSize(theme.FsMicro).FontWeight(500).LetterSpacing(.4).TextColor(a.tk.Tx3)
		body()
	})
}

// toolResultIsMarkdown 这个工具的结果文本是 Markdown(MCP 工具、联网抓取 / 搜索、技能、子 Agent 消息),按 Markdown 渲染;
// Bash / Read / Grep 等的输出是原样文本,仍走等宽折叠块。
func toolResultIsMarkdown(tc *core.ToolCall) bool {
	if strings.HasPrefix(tc.Name, "mcp__") {
		return true
	}
	switch tc.Name {
	case "WebFetch", "WebSearch", "Skill", "SendMessage", "Agent", "Task":
		return true
	}
	return false
}

// mdResultCard 把 Markdown 结果画进一张卡:最高 foldExpandMax,超出在卡内滚动。
func (a *App) mdResultCard(c *ui.Context, key, text string, live bool) {
	a.card(c, nil, func() {
		kit.XYScroll(c).TrackScroll(a.foldScroll(key, live)).MaxHeight(float32(foldExpandMax)).Key("mdout-" + key).Children(func() {
			ui.Column(c).Padding(8, 10).Children(func() {
				markdown.View(c, padCode(strings.TrimSpace(text)), a.planMdOptions())
			})
		})
	})
}

// toolResult 结果区:文本块逐段成卡,图片块给一行说明,没有内容时按状态说明原因。
func (a *App) toolResult(c *ui.Context, tc *core.ToolCall) {
	t := a.tk
	shown := 0
	for i, ct := range tc.Content {
		switch v := ct.(type) {
		case *core.TextBlock:
			if strings.TrimSpace(v.Text) == "" {
				continue
			}
			shown++
			text := v.Text
			if toolResultIsMarkdown(tc) {
				a.mdResultCard(c, foldKey("tmd", tc.ID+":"+itoa(i)), text, !tc.Status.Finished())
				continue
			}
			a.card(c, nil, func() {
				a.foldBody(c, foldKey("tout", tc.ID+":"+itoa(i)), text, !tc.Status.Finished(), true, func() { a.copyOut(text, "结果") })
			})
		case *core.ImageBlock:
			shown++
			label := firstNonEmpty(v.Label, v.Path, "图片")
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				kit.Ic(c, "image", kit.S12).TextColor(t.Tx3)
				ui.Text(c, label).FontSize(theme.FsCap).TextColor(t.Tx2).SingleLine()
			})
		}
	}
	if shown > 0 {
		return
	}
	msg := "没有返回内容"
	switch tc.Status {
	case core.ToolRunning, core.ToolPending:
		msg = "运行中,结果还没返回…"
	case core.ToolDenied:
		msg = "已拒绝,没有执行"
	case core.ToolInterrupted:
		msg = "回合被中断,没有拿到结果"
	}
	ui.Text(c, msg).FontSize(theme.FsCap).TextColor(t.Tx3)
}
