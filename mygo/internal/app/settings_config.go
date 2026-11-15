package app

import (
	"net/url"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claudecfg"
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
	if err != nil || (len(rules.Allow)+len(rules.Ask)+len(rules.Deny) == 0 && a2.demoWs()) {
		rules = claudecfg.DemoRules()
	}
	drawRuleGroup(a2, c, "allow", "允许", kit.KOk, rules.Allow, 0)
	drawRuleGroup(a2, c, "ask", "询问", kit.KWarn, rules.Ask, 1)
	drawRuleGroup(a2, c, "deny", "拒绝", kit.KErr, rules.Deny, 2)
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
	if a2.demoWs() && len(a2.projects) == 0 {
		rows = claudecfg.DemoTrust()
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
	rows, err := claudecfg.ReadEnvRows(a2.proj)
	if err != nil {
		rows = claudecfg.DemoEnv()
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
}

func (a *App) deleteEnvWithUndo(name string) {
	layer := claudecfg.LayerUser
	a.set.undo.Begin("已删除 "+name, func() {
		_ = claudecfg.SetEnv(name, "••••••••", layer, a.proj)
	})
	_ = claudecfg.RemoveEnv(name, layer, a.proj)
}

func (a *App) lspBody(a2 *App, c *ui.Context) {
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

func (a *App) providersBody(a2 *App, c *ui.Context) {
	list, err := claudecfg.LoadProviders()
	if err != nil {
		list = nil
	}
	if a2.set.providers == nil {
		a2.set.providers = list
	}
	for i, p := range a2.set.providers {
		p := p
		first := i == 0
		rowBox(a2, c, first, 44).Children(func() {
			iconBox(a2, c, "server")
			ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, p.Name).FontWeight(500)
					if p.Status != "" && p.Status != claudecfg.ProvTestNone {
						kind := kit.KWarn
						if p.Status == claudecfg.ProvOK || strings.HasPrefix(p.Status, claudecfg.ProvOK) {
							kind = kit.KOk
						}
						if p.Status == claudecfg.ProvFail {
							kind = kit.KErr
						}
						if p.Status == claudecfg.ProvTestIng {
							kind = kit.KWarn
						}
						kit.Tag(c, p.Status, kind)
					}
				})
				ui.Text(c, p.BaseURL).Font("monospace").FontSize(theme.FsCap).TextColor(a2.tk.Tx3).SingleLine()
			})
			st := a2.set.provBusy[p.ID]
			btnLabel := "测试"
			if st == 1 {
				btnLabel = "测试中"
			}
			kit.Btn(c, btnLabel, kit.BtnOpt{Small: true, Icon: "refresh"}).OnClick(func() {
				a2.testProvider(p.ID)
			})
		})
	}
}

func (a *App) testProvider(id string) {
	if a.set.provBusy == nil {
		a.set.provBusy = map[string]int{}
	}
	a.set.provBusy[id] = 1
	for i, p := range a.set.providers {
		if p.ID != id {
			continue
		}
		a.set.providers[i].Status = claudecfg.ProvTestIng
		base := p.BaseURL
		a.bg("连通性测试", func() func() {
			ok, ms := claudecfg.TestProvider(base)
			st := claudecfg.ProvFail
			if ok {
				st = claudecfg.FormatTestResult(true, ms)
			}
			return func() {
				a.set.provBusy[id] = 0
				if i < len(a.set.providers) {
					a.set.providers[i].Status = st
				}
				_ = claudecfg.SaveProviders(a.set.providers)
			}
		})
		break
	}
}

func (a *App) netCheckBody(a2 *App, c *ui.Context) {
	hosts := []string{"api.anthropic.com", "github.com"}
	if a2.set.net == nil {
		a2.set.net = map[string]int{}
	}
	for i, h := range hosts {
		h := h
		st := a2.set.net[h]
		rowBox(a2, c, i == 0, 32).Children(func() {
			switch st {
			case 2:
				kit.Ic(c, "okc", kit.S12).TextColor(a2.tk.Ok)
			case 3:
				kit.Ic(c, "errc", kit.S12).TextColor(a2.tk.Err)
			case 1:
				kit.Ic(c, "refresh", kit.S12).TextColor(a2.tk.Warn)
			default:
				kit.Ic(c, "activity", kit.S12).TextColor(a2.tk.Tx4)
			}
			ui.Text(c, h).Font("monospace").Grow(1)
			lbl := "未测"
			switch st {
			case 1:
				lbl = claudecfg.ProvTestIng
			case 2:
				lbl = claudecfg.ProvOK
			case 3:
				lbl = claudecfg.ProvFail
			}
			if ms, ok := a2.set.netMs[h]; ok && st == 2 {
				lbl = claudecfg.FormatTestResult(true, ms)
			}
			ui.Text(c, lbl).FontSize(theme.FsCap).TextColor(a2.tk.Tx3)
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

func (a *App) settingsUndoToast(c *ui.Context) {
	if !a.set.undo.Active() {
		return
	}
	t := a.tk
	ui.Row(c).Gap(10).Padding(8, 10).Radius(t.RMd()).Background(t.Fill1).Border(1, t.Bd2).Margin(0, 0, 8, 0).Children(func() {
		ui.Text(c, a.set.undo.Label).Grow(1).FontSize(theme.FsCap)
		kit.Btn(c, "撤销", kit.BtnOpt{Small: true}).OnClick(func() {
			fn := a.set.undo.Undo
			a.set.undo.Clear()
			if fn != nil {
				fn()
			}
		})
		rem := time.Until(a.set.undo.Until).Seconds()
		ui.Text(c, itoa(int(rem+0.5))+"s").FontSize(theme.FsMicro).TextColor(t.Tx4).FontFeatures("tnum")
	})
}

func (a *App) initSettingsConfig() {
	if a.set.providers == nil {
		list, _ := claudecfg.LoadProviders()
		a.set.providers = list
	}
	if a.set.provBusy == nil {
		a.set.provBusy = map[string]int{}
	}
	if a.set.net == nil {
		a.set.net = map[string]int{}
	}
	if a.set.netMs == nil {
		a.set.netMs = map[string]int64{}
	}
}
