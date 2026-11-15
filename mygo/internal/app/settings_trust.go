package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func (a *App) reloadTrustList() {
	if a.demoWs() {
		rows, err := demo.SettingTrustList()
		if err != nil {
			a.set.trustErr = err.Error()
			a.set.trustRows = nil
		} else {
			a.set.trustErr = ""
			a.set.trustRows = rows
		}
		a.set.trustLoading = false
		return
	}
	a.set.trustLoading = true
	a.set.trustErr = ""
	a.bg("settings.trust", func() func() {
		defer diag.SpanSlow("settings", "读取项目信任", 50*time.Millisecond)()
		claudePaths, err := claudecfg.PathsInClaudeJSON()
		if err != nil {
			return func() {
				a.set.trustLoading = false
				a.set.trustErr = "无法读取 ~/.claude.json"
				a.set.trustRows = nil
				setLog.Warn("读取项目信任列表失败", "err", err)
			}
		}
		paths := claudecfg.MergeTrustPaths(claudePaths, a.registry().Projects(), trustSidebarPaths(a), a.proj)
		rows, err2 := claudecfg.ListTrust(paths)
		return func() {
			a.set.trustLoading = false
			if err2 != nil {
				a.set.trustErr = "无法读取 ~/.claude.json"
				a.set.trustRows = nil
				setLog.Warn("解析项目信任失败", "err", err2)
				return
			}
			a.set.trustErr = ""
			a.set.trustRows = rows
		}
	})
}

func trustSidebarPaths(a *App) []string {
	var out []string
	for _, pi := range a.projects {
		if pi.Path != "" {
			out = append(out, pi.Path)
		}
	}
	return out
}

func (a *App) trustApply(path string, trusted, hook, mcp bool) {
	if a.demoWs() {
		if demo.TrustWriteFail() {
			if a.set.trustRowErr == nil {
				a.set.trustRowErr = map[string]string{}
			}
			a.set.trustRowErr[path] = "写入失败,请重试"
			return
		}
		for i := range a.set.trustRows {
			if a.set.trustRows[i].Path == path {
				a.set.trustRows[i].Trusted = trusted
				a.set.trustRows[i].HookOK = hook
				a.set.trustRows[i].McpOK = mcp
				break
			}
		}
		return
	}
	if err := claudecfg.SetTrust(path, trusted, hook, mcp); err != nil {
		if a.set.trustRowErr == nil {
			a.set.trustRowErr = map[string]string{}
		}
		a.set.trustRowErr[path] = "写入失败,请重试"
		setLog.Warn("更新项目信任失败", "path", path, "err", err)
		return
	}
	delete(a.set.trustRowErr, path)
	for i := range a.set.trustRows {
		if a.set.trustRows[i].Path == path {
			a.set.trustRows[i].Trusted = trusted
			a.set.trustRows[i].HookOK = hook
			a.set.trustRows[i].McpOK = mcp
			break
		}
	}
	setLog.Info("更新项目信任", "path", path, "trusted", trusted)
}

func (a *App) trustBody(a2 *App, c *ui.Context) {
	t := a2.tk
	if len(a2.set.trustRows) == 0 && a2.set.trustErr == "" && !a2.set.trustLoading {
		a2.reloadTrustList()
	}
	if a2.set.trustLoading && len(a2.set.trustRows) == 0 {
		rowBox(a2, c, true, 40).Children(func() {
			ui.Text(c, "正在读取项目信任…").FontSize(theme.FsCap).TextColor(t.Tx3)
		})
		return
	}
	if a2.set.trustErr != "" {
		rowBox(a2, c, true, 44).Children(func() {
			ui.Text(c, a2.set.trustErr).FontSize(theme.FsCap).TextColor(t.Err).Grow(1)
			if kit.Btn(c, "重试", kit.BtnOpt{Small: true, Icon: "refresh"}).Clicked() {
				a2.reloadTrustList()
			}
		})
		return
	}
	rows := a2.set.trustRows
	if len(rows) == 0 {
		rowBox(a2, c, true, 48).Children(func() {
			ui.Text(c, "还没有任何项目").FontWeight(500).TextColor(t.Tx2)
			ui.Text(c, "打开会话或添加项目后,会出现在这里。").FontSize(theme.FsCap).TextColor(t.Tx3)
		})
		return
	}
	for i, p := range rows {
		p := p
		rowBox(a2, c, i == 0, 52).Children(func() {
			iconBox(a2, c, "folder")
			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				titled(a2, c, filepathBase(p.Path), tildePath(p.Path), true)
				if msg := a2.set.trustRowErr[p.Path]; msg != "" {
					ui.Text(c, msg).FontSize(theme.FsCap).TextColor(t.Err)
				}
			})
			ui.Text(c, "钩子").FontSize(theme.FsCap).TextColor(a2.tk.Tx3)
			hookKey := "trust-h-" + itoa(i)
			if kit.Tog(c, p.HookOK, hookKey) {
				a2.trustApply(p.Path, p.Trusted, !p.HookOK, p.McpOK)
			}
			ui.Text(c, "MCP").FontSize(theme.FsCap).TextColor(a2.tk.Tx3)
			if kit.Tog(c, p.McpOK, "trust-m-"+itoa(i)) {
				a2.trustApply(p.Path, p.Trusted, p.HookOK, !p.McpOK)
			}
			ui.Text(c, "信任").FontSize(theme.FsCap).TextColor(a2.tk.Tx3)
			if kit.Tog(c, p.Trusted, "trust-t-"+itoa(i)) {
				if p.Trusted {
					name := filepathBase(p.Path)
					if name == "" {
						name = "该项目"
					}
					a2.confirmDlg("撤销信任?", "撤销后「"+name+"」上的钩子与 MCP 将默认不运行,直到你再次信任。", "撤销信任", true, func() {
						a2.trustApply(p.Path, false, p.HookOK, p.McpOK)
					})
				} else {
					a2.trustApply(p.Path, true, p.HookOK, p.McpOK)
				}
			}
		})
	}
	rowBox(a2, c, false, 36).Children(func() {
		if kit.Btn(c, "刷新列表", kit.BtnOpt{Small: true, Icon: "refresh"}).Clicked() {
			a2.reloadTrustList()
		}
		ui.Text(c, "合并 ~/.claude.json 与项目注册表").FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1)
	})
}
