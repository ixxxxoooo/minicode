package app

import (
	"net/url"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/kit"
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
	var rules claudecfg.PermRules
	if a2.demoWs() {
		rules = demo.SettingDemoRules()
	} else {
		var err error
		rules, err = claudecfg.ReadRules(layer, a2.proj)
		if err != nil {
			a2.set.rulesErr = "无法读取权限规则"
			setLog.Warn("读取权限规则失败", "err", err)
		} else {
			a2.set.rulesErr = ""
		}
	}
	if a2.set.rulesErr != "" {
		rowBox(a2, c, true, 44).Children(func() {
			ui.Text(c, a2.set.rulesErr).FontSize(theme.FsCap).TextColor(a2.tk.Err).Grow(1)
		})
		a2.rulesAddRow(a2, c)
		return
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
		if len(rules) == 0 {
			ui.Text(c, "暂无规则").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(0, 12, 8, 12)
		}
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
		a2.set.envErr = ""
	} else {
		rows, err = claudecfg.ReadEnvRows(a2.proj)
		if err != nil {
			a2.set.envErr = "无法读取环境变量"
			setLog.Warn("读取环境变量失败", "err", err)
		} else {
			a2.set.envErr = ""
		}
	}
	if a2.set.envErr != "" {
		rowBox(a2, c, true, 44).Children(func() {
			ui.Text(c, a2.set.envErr).FontSize(theme.FsCap).TextColor(a2.tk.Err).Grow(1)
		})
		a2.envAddRow(a2, c)
		return
	}
	if len(rows) == 0 {
		rowBox(a2, c, true, 40).Children(func() {
			ui.Text(c, "没有已配置的环境变量").FontSize(theme.FsCap).TextColor(a2.tk.Tx3)
		})
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
