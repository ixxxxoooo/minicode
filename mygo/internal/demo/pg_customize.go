// 演示数据索引 · 自定义
//
// 套 0:默认与设计稿一致(见 customize.go)。
// 套 1:七类各仅空列表占位说明。
// 套 2:七类各 40 项(名称带序号)。
// 套 3:插件加载失败、MCP 多种错误、Skill 缺文件、长短正文混排、来源混合。
//
// 切换:命令面板「演示:自定义」。
package demo

import (
	"fmt"
	"strings"

	c "github.com/ixxxxoooo/minicode/internal/customize"
)

func customizeDefault() *c.Store {
	return Customize()
}

func customizeEmpty() *c.Store {
	items := map[c.Cat][]*c.Item{}
	for _, cat := range c.Cats {
		items[cat] = nil
	}
	return c.NewStore(items, nil)
}

func customizeCrowded40() *c.Store {
	items := map[c.Cat][]*c.Item{}
	scopes := []c.Scope{c.User, c.Project, c.Local}
	for ci, cat := range c.Cats {
		for i := 1; i <= 40; i++ {
			it := &c.Item{
				Name:  fmt.Sprintf("%s-%03d", cat, i),
				Desc:  fmt.Sprintf("演示拥挤列表 · %s #%d", catLabel(cat), i),
				Scope: scopes[i%3],
				On:    i%3 != 0,
			}
			fillCustomizeFields(cat, it, i)
			items[cat] = append(items[cat], it)
		}
		_ = ci
	}
	return c.NewStore(items, customizeMarketStub())
}

func catLabel(cat c.Cat) string {
	switch cat {
	case c.Plugins:
		return "插件"
	case c.MCP:
		return "MCP"
	case c.Skills:
		return "Skill"
	case c.Commands:
		return "命令"
	case c.Agents:
		return "子 Agent"
	case c.Rules:
		return "规则"
	case c.Hooks:
		return "钩子"
	}
	return string(cat)
}

func fillCustomizeFields(cat c.Cat, it *c.Item, i int) {
	switch cat {
	case c.Plugins:
		it.Ver, it.Src = "1.0.0", "演示数据"
		it.About = "插件说明 #" + fmt.Sprint(i)
	case c.MCP:
		it.Transport, it.Cmd = "stdio", "npx demo-mcp-"+fmt.Sprint(i)
		it.Status = c.StOK
		if i%7 == 0 {
			it.Status, it.Desc = c.StErr, "连接失败"
		} else if i%5 == 0 {
			it.Status, it.Desc = c.StWarn, "需要授权"
		} else if i%4 == 0 {
			it.On = false
			it.Desc = "已禁用"
		}
	case c.Skills:
		it.Path = "~/.claude/skills/demo-" + fmt.Sprint(i)
		it.Files = []c.File{{Name: "SKILL.md", Active: true}}
		it.Body = shortOrLong(i, "# Skill\n\n内容。")
	case c.Commands:
		it.Note = "斜杠命令"
		it.Body = shortOrLong(i, fmt.Sprintf("执行演示命令 #%d", i))
		it.Allow = []string{"Read", "Bash"}
	case c.Agents:
		it.Note = "子 Agent"
		it.Model = i % 4
		it.Body = shortOrLong(i, "你是子 Agent。")
	case c.Rules:
		it.Size = "2 KB"
		it.Tokens = 400 + i
		it.Body = "---\nglobs: \"**/*.go\"\n---\n" + shortOrLong(i, "规则正文")
	case c.Hooks:
		it.Event = i % len(c.Events)
		it.Match = "Bash"
		it.Cmd = "echo hook-" + fmt.Sprint(i)
	}
}

func shortOrLong(i int, base string) string {
	if i%5 == 0 {
		return strings.Repeat(base+"\n", 12)
	}
	return base
}

func customizeErrors() *c.Store {
	tools := func(kv ...any) []c.Tool {
		var out []c.Tool
		for i := 0; i < len(kv); i += 2 {
			out = append(out, c.Tool{Name: kv[i].(string), On: kv[i+1].(bool)})
		}
		return out
	}
	longName := strings.Repeat("超长插件名·", 5) + "enterprise-audit"
	items := map[c.Cat][]*c.Item{
		c.Plugins: {
			{Name: longName, Desc: "加载失败:入口脚本抛错", Scope: c.User, On: true, Ver: "0.0.0", Src: "本地",
				About: "模拟插件启用后加载失败。"},
			{Name: "builtin-shell", Desc: "内置 · 始终可用", Scope: c.User, On: true, Ver: "内置", Src: "内置"},
		},
		c.MCP: {
			{Name: "github", Desc: "已连接", Scope: c.User, On: true, Status: c.StOK, Transport: "stdio", Cmd: "npx github-mcp", Tools: tools("issues", true)},
			{Name: "figma", Desc: "需要授权", Scope: c.User, On: true, Status: c.StWarn, Transport: "http", Cmd: "https://mcp.figma.com"},
			{Name: "postgres", Desc: "ECONNREFUSED", Scope: c.Local, Status: c.StErr, Transport: "stdio", Cmd: "postgres-mcp", On: false},
			{Name: "disabled-srv", Desc: "已禁用", Scope: c.Project, On: false, Status: c.StOK, Transport: "stdio", Cmd: "demo"},
		},
		c.Skills: {
			{Name: "missing-skill", Desc: "缺 SKILL.md", Scope: c.Project, Path: ".claude/skills/missing", Files: []c.File{{Name: "README.md"}}},
			{Name: "ok-skill", Desc: "正常", Scope: c.User, On: true, Path: "~/.claude/skills/ok",
				Files: []c.File{{Name: "SKILL.md", Active: true, Bytes: 1200}, {Name: "ref.md", Bytes: 340}, {Name: "examples/demo.sh", Bytes: 89}},
				Body:  "---\nname: ok-skill\ntriggers:\n  - 审查\n---\n\n# OK\n\n短正文。"},
		},
		c.Commands: {
			{Name: "/short", Desc: "短命令", Scope: c.User, On: true, Body: "一句话命令。"},
			{Name: "/long", Desc: "长命令", Scope: c.Project, On: true, Body: strings.Repeat("步骤说明。\n", 30)},
		},
		c.Agents: {
			{Name: "haiku-fast", Desc: "Haiku · 内置风格", Scope: c.User, On: true, Model: 3, Body: "快。"},
			{Name: "opus-deep", Desc: "Opus · 长系统提示", Scope: c.Project, On: false, Model: 2, Body: strings.Repeat("深度推理。\n", 20)},
		},
		c.Rules: {
			{Name: "project/.claude/rules/go-style.md", Desc: "项目 · globs", Scope: c.Project, On: true, Body: "---\nglobs: \"**/*.go\"\n---\n\nGo 风格约定。"},
			{Name: "~/.claude/CLAUDE.md", Desc: "用户 · 短", Scope: c.User, On: true, Body: "# 用户层\n\n一条规则。"},
		},
		c.Hooks: {
			{Name: "PreToolUse", Desc: "项目", Scope: c.Project, On: true, Event: 0, Match: "Bash(rm*)", Cmd: "deny.sh"},
			{Name: "Stop", Desc: "用户 · 长脚本", Scope: c.User, On: true, Event: 2, Cmd: strings.Repeat("echo done;", 8)},
		},
	}
	return c.NewStore(items, customizeMarketStub())
}

func customizeMarketStub() []*c.Market {
	return []*c.Market{{Name: "demo-pack", By: "演示", Desc: "占位市场", Stars: "1k"}}
}

// CustomizeFor 返回指定套的自定义存储。
func CustomizeFor(set int) *c.Store {
	switch set {
	case 1:
		return customizeEmpty()
	case 2:
		return customizeCrowded40()
	case 3:
		return customizeErrors()
	default:
		return customizeDefault()
	}
}
