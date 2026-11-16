package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

func (a *App) rulesAddRow(a2 *App, c *ui.Context) {
	t := a2.tk
	ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(8, 12).Children(func() {
		ui.Row(c).Grow(1).MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "plus", kit.S12).TextColor(t.Ac)
			ui.TextInput(c.Key("rule-in"), &a2.set.ruleDraft).Grow(1).Placeholder("示例:Bash(git status)、Read(src/**)")
		})
		if i := kit.Seg(c, []kit.SegItem{{Text: "允许"}, {Text: "询问"}, {Text: "拒绝"}}, a2.set.ruleKind, kit.SegOpt{Key: "rule-kind"}); i >= 0 {
			a2.set.ruleKind = i
		}
		if kit.Btn(c, "添加", kit.BtnOpt{Small: true, Variant: kit.BtnPri}).Clicked() {
			a2.submitRuleAdd()
		}
	})
}

// shakeField 让设置页某个输入框抖一下(错误反馈);map 懒初始化,不依赖是否已渲染过该行。
func (a *App) shakeField(key string) {
	if a.set.fieldShake == nil {
		a.set.fieldShake = map[string]time.Time{}
	}
	a.set.fieldShake[key] = a.now().Add(kit.ShakeDuration)
}

func (a *App) submitRuleAdd() {
	rule := strings.TrimSpace(a.set.ruleDraft)
	if rule == "" {
		a.shakeField("rule")
		a.toast = "先写一条规则,例如 Bash(git status) 或 Read(src/**)"
		return
	}
	kind := []string{"allow", "ask", "deny"}[a.set.ruleKind]
	layer := a.permLayer()
	if err := claudecfg.AddRule(layer, a.proj, kind, rule); err != nil {
		setLog.Warn("添加规则失败", "err", err)
		a.toast = err.Error()
		return
	}
	a.set.ruleDraft = ""
	setLog.Info("添加权限规则", "kind", kind, "layer", layer)
}

func (a *App) envAddRow(a2 *App, c *ui.Context) {
	t := a2.tk
	ui.Column(c).Gap(6).Padding(8, 12).Children(func() {
		ui.Text(c, "添加环境变量").FontSize(theme.FsCap).FontWeight(500).TextColor(t.Tx3)
		ui.Row(c).Gap(8).Children(func() {
			ui.TextInput(c.Key("env-name"), &a2.set.envName).Width(180).Placeholder("变量名")
			ui.TextInput(c.Key("env-val"), &a2.set.envVal).Grow(1).Placeholder("值(保存后遮盖)")
		})
		if i := kit.Seg(c, []kit.SegItem{{Text: "用户"}, {Text: "项目"}, {Text: "本地"}}, a2.set.envLayer, kit.SegOpt{Key: "env-layer"}); i >= 0 {
			a2.set.envLayer = i
		}
		if kit.Btn(c, "添加", kit.BtnOpt{Small: true, Variant: kit.BtnPri}).Clicked() {
			a2.submitEnvAdd()
		}
	})
}

func (a *App) submitEnvAdd() {
	name := strings.TrimSpace(a.set.envName)
	if name == "" {
		a.shakeField("env-name")
		return
	}
	layer := claudecfg.LayerUser
	switch a.set.envLayer {
	case 1:
		layer = claudecfg.LayerProject
	case 2:
		layer = claudecfg.LayerLocal
	}
	if err := claudecfg.SetEnv(name, a.set.envVal, layer, a.proj); err != nil {
		a.toast = err.Error()
		return
	}
	a.set.envName, a.set.envVal = "", ""
	setLog.Info("添加环境变量", "name", name, "layer", layer)
}
