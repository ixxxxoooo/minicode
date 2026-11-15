package app

import (
	"net/url"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/lsp"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func (a *App) permLayer() claudecfg.Layer {
	switch a.prefs.Int("perm.layer", 1) {
	case 0:
		return claudecfg.LayerUser
	case 2:
		return claudecfg.LayerLocal
	default:
		return claudecfg.LayerProject
	}
}

func (a *App) rulesBody(a2 *App, c *ui.Context) {
	layer := a2.permLayer()
	rules, err := claudecfg.ReadRules(layer, a2.proj)
	if a2.demoWs() {
		rules = demo.SettingDemoRules()
	} else if err != nil || len(rules.Allow)+len(rules.Ask)+len(rules.Deny) == 0 {
		rules = claudecfg.DemoRules()
	}
	drawRuleGroup(a2, c, "allow", "允许", kit.KOk, rules.Allow, 0)
	drawRuleGroup(a2, c, "ask", "询问", kit.KWarn, rules.Ask, 1)
	drawRuleGroup(a2, c, "deny", "拒绝", kit.KErr, rules.Deny, 2)
	a2.rulesAddRow(a2, c)
}

func drawRuleGroup(a *App, c *ui.Context, kind, lbl string, tag kit.TagKind, rules []string, gi int) {
	t := a.tk
	ui.Column(c).Gap(0).Children(func() {
		if gi > 0 {
			ui.Box(c).Height(1).Background(t.Bd1).Margin(0, 12)
		}
		ui.Text(c, map[string]string{"allow": "始终允许", "ask": "每次询问", "deny": "始终拒绝"}[kind]).
			FontSize(theme.FsCap).FontWeight(700).TextColor(t.Tx3).Padding(8, 12, 4, 12)
		for i, r := range rules {
			rule := r
			rowBox(a, c, i == 0, 30).Children(func() {
				kit.Tag(c, lbl, tag)
				kit.Mono(c, rule, t.Tx1).Grow(1)
				del := kit.Ib(c, "trash", kit.IbOpt{Small: true, Tip: "删除"})
				if del.Clicked() {
					a.deleteRuleWithUndo(kind, rule)
				}
			})
		}
	})
}

func (a *App) deleteRuleWithUndo(kind, rule string) {
	layer := a.permLayer()
	snap := rule
	a.set.undo.Begin("已删除规则", func() {
		_ = claudecfg.AddRule(layer, a.proj, kind, snap)
	})
	if err := claudecfg.RemoveRule(layer, a.proj, kind, rule); err != nil {
		setLog.Warn("删除规则失败", "err", err)
	}
}

func (a *App) trustBody(a2 *App, c *ui.Context) {
	var rows []claudecfg.TrustEntry
	if a2.demoWs() {
		rows = demo.DemoTrust()
	} else {
		paths := make([]string, 0, len(a2.projects))
		for _, pi := range a2.projects {
			if len(paths) >= 8 {
				break
			}
			paths = append(paths, pi.Path)
		}
		if len(paths) == 0 && a2.proj != "" {
			paths = []string{a2.proj}
		}
		rows = claudecfg.ListTrust(paths)
	}
	for i, p := range rows {
		p := p
		rowBox(a2, c, i == 0, 44).Children(func() {
			iconBox(a2, c, "folder")
			titled(a2, c, filepathBase(p.Path), tildePath(p.Path), true)
			ui.Text(c, "钩子").FontSize(theme.FsCap).TextColor(a2.tk.Tx3)
			if kit.Tog(c, p.HookOK, "trust-h-"+itoa(i)) {
				_ = claudecfg.SetTrust(p.Path, p.Trusted, !p.HookOK, p.McpOK)
			}
			ui.Text(c, "MCP").FontSize(theme.FsCap).TextColor(a2.tk.Tx3)
			if kit.Tog(c, p.McpOK, "trust-m-"+itoa(i)) {
				_ = claudecfg.SetTrust(p.Path, p.Trusted, p.HookOK, !p.McpOK)
			}
			ui.Text(c, "信任").FontSize(theme.FsCap).TextColor(a2.tk.Tx3)
			if kit.Tog(c, p.Trusted, "trust-t-"+itoa(i)) {
				_ = claudecfg.SetTrust(p.Path, !p.Trusted, p.HookOK, p.McpOK)
			}
		})
	}
}

func filepathBase(p string) string {
	if p == "" {
		return "项目"
	}
	i := strings.LastIndexAny(p, `/\`)
	if i < 0 {
		return p
	}
	return p[i+1:]
}

func (a *App) envBody(a2 *App, c *ui.Context) {
	var rows []claudecfg.EnvRow
	var err error
	if a2.demoWs() {
		rows = demo.DemoEnv()
	} else {
		rows, err = claudecfg.ReadEnvRows(a2.proj)
		if err != nil {
			rows = claudecfg.DemoEnv()
		}
	}
	for i, e := range rows {
		e := e
		rowBox(a2, c, i == 0, 32).Children(func() {
			kit.Mono(c, e.Name, a2.tk.Tx1).Width(230)
			kit.Mono(c, e.Mask, a2.tk.Tx3).Grow(1)
			kit.Tag(c, e.Scope)
			if kit.Ib(c, "trash", kit.IbOpt{Small: true, Tip: "删除"}).Clicked() {
				a2.deleteEnvWithUndo(e.Name)
			}
		})
	}
	a2.envAddRow(a2, c)
}

func (a *App) deleteEnvWithUndo(name string) {
	layer := claudecfg.LayerUser
	a.set.undo.Begin("已删除 "+name, func() {
		_ = claudecfg.SetEnv(name, "••••••••", layer, a.proj)
	})
	_ = claudecfg.RemoveEnv(name, layer, a.proj)
}

func (a *App) lspBody(a2 *App, c *ui.Context) {
	if a2.demoWs() {
		for i, row := range demo.LSPDemoRows() {
			r := row
			rowBox(a2, c, i == 0, 32).Children(func() {
				if r.OK {
					kit.Ic(c, "okc", kit.S12).TextColor(a2.tk.Ok)
				} else {
					kit.Ic(c, "alert", kit.S12).TextColor(a2.tk.Warn)
				}
				ui.Text(c, r.Name).FontWeight(500).Grow(1)
				ui.Text(c, r.Status).FontSize(theme.FsCap).TextColor(a2.tk.Tx3).SingleLine()
			})
		}
		return
	}
	show := []string{"rust-analyzer", "Pyright", "gopls", "typescript-language-server"}
	for i, name := range show {
		var sp *lsp.Spec
		for j := range lsp.Specs {
			if lsp.Specs[j].Name == name {
				sp = &lsp.Specs[j]
				break
			}
		}
		ok, status := false, "未安装"
		if sp != nil {
			if _, found := sp.Find(); found {
				ok, status = true, "已就绪"
			} else {
				status = "未安装 · " + sp.Install
			}
		}
		s := struct {
			n, st string
			ok    bool
		}{name, status, ok}
		rowBox(a2, c, i == 0, 32).Children(func() {
			if s.ok {
				kit.Ic(c, "okc", kit.S12).TextColor(a2.tk.Ok)
			} else {
				kit.Ic(c, "alert", kit.S12).TextColor(a2.tk.Warn)
			}
			ui.Text(c, s.n).FontWeight(500).Grow(1)
			ui.Text(c, s.st).FontSize(theme.FsCap).TextColor(a2.tk.Tx3).SingleLine()
			if !s.ok && sp != nil {
				kit.Btn(c, "复制安装命令", kit.BtnOpt{Small: true, Icon: "copy"}).OnClick(func() {
					c.WriteClipboard(sp.Install)
					c.Toast("已复制安装命令")
				})
			}
		})
	}
}

func (a *App) netCheckBody(a2 *App, c *ui.Context) {
	hosts := []string{"api.anthropic.com", "api.deepseek.com", "github.com"}
	if a2.set.net == nil {
		a2.set.net = map[string]int{}
	}
	for i, h := range hosts {
		h := h
		st := a2.set.net[h]
		rowBox(a2, c, i == 0, 32).Children(func() {
			cs := kit.ConnIdle
			res := ""
			switch st {
			case 1:
				cs = kit.ConnRun
			case 2:
				cs = kit.ConnOK
				if ms, ok := a2.set.netMs[h]; ok {
					res = claudecfg.FormatTestResult(true, ms)
				} else {
					res = claudecfg.ProvOK
				}
			case 3:
				cs = kit.ConnFail
				res = claudecfg.ProvFail
			}
			ui.Text(c, h).Font("monospace").Grow(1)
			kit.ConnTestRow(c, "net-"+h, cs, "测试", res, func() { a2.runNetCheck([]string{h}) })
		})
	}
	rowBox(a2, c, false, 32).Children(func() {
		kit.Btn(c, "重新检测", kit.BtnOpt{Small: true, Icon: "refresh"}).OnClick(func() { a2.runNetCheck(hosts) })
		ui.Text(c, "依次访问以上主机,最长等待 5 秒").FontSize(theme.FsCap).TextColor(a2.tk.Tx3).Grow(1)
	})
}

func (a *App) runNetCheck(hosts []string) {
	if a.set.net == nil {
		a.set.net = map[string]int{}
	}
	if a.set.netMs == nil {
		a.set.netMs = map[string]int64{}
	}
	for _, h := range hosts {
		a.set.net[h] = 1
	}
	a.bg("网络连通性", func() func() {
		results := map[string]struct {
			ok bool
			ms int64
		}{}
		for _, h := range hosts {
			if a.demoWs() {
				ok, ms := demo.DemoNetLatency(h)
				results[h] = struct {
					ok bool
					ms int64
				}{ok, ms}
				continue
			}
			u, _ := url.Parse("https://" + h)
			ok, ms := claudecfg.PingHost(u.Host)
			results[h] = struct {
				ok bool
				ms int64
			}{ok, ms}
		}
		return func() {
			for h, r := range results {
				if r.ok {
					a.set.net[h] = 2
					a.set.netMs[h] = r.ms
				} else {
					a.set.net[h] = 3
				}
			}
		}
	})
}

func (a *App) claudeDiagBody(a2 *App, c *ui.Context) {
	if !a2.demoWs() {
		ui.Text(c, "真实模式请在自定义页查看插件与 MCP。").FontSize(theme.FsCap).TextColor(a2.tk.Tx3).Padding(8, 12)
		return
	}
	t := a2.tk
	ui.Column(c).Gap(0).Padding(0, 0, 4, 0).Children(func() {
		for i, line := range demo.ClaudeDiagLines() {
			rowBox(a2, c, i == 0, 28).Children(func() {
				kit.Ic(c, "info", kit.S12).TextColor(t.Tx3)
				ui.Text(c, line).FontSize(theme.FsCap).Grow(1)
			})
		}
	})
}

func (a *App) settingsUndoToast(c *ui.Context) {
	kit.UndoBar(c, &a.set.undo)
}

func (a *App) initSettingsConfig() {
	if a.demoWs() && len(a.set.net) == 0 {
		a.set.net = map[string]int{"api.anthropic.com": 2, "github.com": 2}
		a.set.netMs = map[string]int64{"api.anthropic.com": 142, "github.com": 96}
		for k, v := range demo.ShortcutDemoPrefs() {
			if !a.prefs.Has(k) {
				a.prefs.Set(k, v)
			}
		}
		if a.set.kbWarn == "" {
			a.set.kbWarn = "⌘ E 已被「切换 Agent / IDE」占用"
		}
	}
	a.pvLoad()
	if a.set.net == nil {
		a.set.net = map[string]int{}
	}
	if a.set.netMs == nil {
		a.set.netMs = map[string]int64{}
	}
}
