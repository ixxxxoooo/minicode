package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 常用的带参数规则(语法同权限规则),点一下加入允许的工具。
var toolSuggestions = []string{
	"Bash(git status:*)", "Bash(git diff:*)", "Bash(git log:*)", "Bash(npm run:*)", "Bash(go test:*)", "Read(./src/**)",
}

const toolPickKey = "cu-toolpick"

// cuToolPicker 是命令「允许的工具」的选择器弹层(同权限规则语法):
// 勾选内置工具、从已配置的 MCP 服务器里选整台或单个工具、点常用规则,或手写 `Bash(git diff:*)` 这样的规则。
// 改的是草稿 d,点「保存」才写进 frontmatter 的 allowed-tools。
func (a *App) cuToolPicker(c *ui.Context, anchor ui.Element, d *customize.Item, st *customize.Store) {
	t := a.tk
	a.popOpen(toolPickKey, func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, theme.Px(320)).AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(4, 0, 0, 0)
			add := func() {
				rule := strings.TrimSpace(a.cu.toolQ)
				if msg := customize.ValidateToolRule(rule); msg != "" {
					a.cu.toolErr, a.cu.toolErrAt = msg, time.Now()
					return
				}
				if !d.AddAllow(rule) {
					a.cu.toolErr, a.cu.toolErrAt = "已经在允许的工具里了", time.Now()
					return
				}
				diag.L("custom").Info("命令添加允许的工具", "规则", rule)
				a.cu.toolQ, a.cu.toolErr = "", ""
			}
			ui.Column(c).Padding(4, 4, 6, 4).Children(func() {
				kit.Field(c, "tp-in", kit.FieldOpt{Err: a.cu.toolErr, ShakeAt: &a.cu.toolErrAt}, func(box ui.Element) {
					in := ui.TextInputBase(c.Key("tp-q"), &a.cu.toolQ).Grow(1).MinWidth(0).FontSize(theme.FsBody).
						Placeholder("搜索或输入规则,如 Bash(git diff:*)").PlaceholderColor(t.Tx4).AutoFocus()
					in.OnSubmit(add)
					kit.Btn(c, "添加", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).OnClick(add)
				})
			})
			q := strings.ToLower(strings.TrimSpace(a.cu.toolQ))
			match := func(s string) bool { return q == "" || strings.Contains(strings.ToLower(s), q) }
			row := func(rule string) {
				on := d.HasAllow(rule)
				if a.mi(c, "tp-"+rule, "", rule, "", on).Clicked() {
					if on {
						d.ToggleAllow(rule)
					} else {
						d.AddAllow(rule)
					}
					a.cu.toolErr = ""
				}
			}
			shown := 0
			kit.Cap(c, "内置工具")
			for _, tl := range customize.AllTools {
				if match(tl) {
					row(tl)
					shown++
				}
			}
			var mcp []string
			for _, r := range a.mcpToolRules(st) {
				if match(r) {
					mcp = append(mcp, r)
				}
			}
			if len(mcp) > 0 {
				kit.Sep(c)
				kit.Cap(c, "MCP 工具(mcp__服务器 = 整台,再加 __工具名 = 单个)")
				for i, r := range mcp {
					if i >= 24 {
						ui.Text(c, "还有 "+itoa(len(mcp)-i)+" 项,输入关键字缩小范围").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(4, 8)
						break
					}
					row(r)
					shown++
				}
			}
			var sug []string
			for _, r := range toolSuggestions {
				if match(r) {
					sug = append(sug, r)
				}
			}
			if len(sug) > 0 {
				kit.Sep(c)
				kit.Cap(c, "常用规则")
				for _, r := range sug {
					row(r)
					shown++
				}
			}
			if shown == 0 {
				ui.Text(c, "没有匹配的工具;按 ⏎ 把「"+strings.TrimSpace(a.cu.toolQ)+"」当作规则添加").FontSize(theme.FsCap).TextColor(t.Tx3).Padding(6, 8)
			}
		})
	})
}

// mcpToolRules 是可以勾选的 MCP 规则:配置层(claude mcp list 读到的)加上运行中会话汇报的(mcp_status)。
// 只看配置层会漏:`claude mcp list` 没读到工具(连接失败 / 没查询)时一个都没有,而会话里其实已经连上、有一堆工具。
func (a *App) mcpToolRules(st *customize.Store) []string {
	seen := map[string]bool{}
	var out []string
	add := func(r string) {
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	for _, r := range st.MCPToolRules() {
		add(r)
	}
	if a.st != nil {
		for _, s := range a.st.Mcp {
			base := "mcp__" + core.McpNormName(s.Name)
			add(base)
			for _, tl := range s.Tools {
				add(base + "__" + tl)
			}
		}
	}
	return out
}
