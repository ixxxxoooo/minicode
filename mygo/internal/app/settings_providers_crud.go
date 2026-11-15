package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

func (a *App) providersToolbar(a2 *App, c *ui.Context) {
	ui.Row(c).Gap(8).Padding(8, 12).Children(func() {
		kit.Btn(c, "新建服务商", kit.BtnOpt{Small: true, Icon: "plus"}).OnClick(func() {
			a2.set.provEdit = true
			a2.set.provDraft = claudecfg.Provider{ID: "custom-" + itoa(len(a2.set.providers)+1), Name: "新建服务商"}
			a2.set.provKey = ""
		})
	})
	a2.providerEditDlg(c)
}

func (a *App) providerEditDlg(c *ui.Context) {
	s := &a.set
	if !s.provEdit {
		return
	}
	a.autoDlg(c, "prov-edit", 440, "服务商", func() { s.provEdit = false }, func() {
		t := a.tk
		ui.TextInput(c.Key("prov-name"), &s.provDraft.Name).Height(28)
		ui.TextInput(c.Key("prov-url"), &s.provDraft.BaseURL).Height(28).Placeholder("https://…")
		ui.TextInput(c.Key("prov-key"), &s.provKey).Height(28).Placeholder("API 密钥(不落设置.json)")
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if kit.Btn(c, "删除", kit.BtnOpt{Small: true, Variant: kit.BtnDangerGhost}).Clicked() {
				a.deleteProvider(s.provDraft.ID)
				s.provEdit = false
				return
			}
			kit.Btn(c, "取消", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { s.provEdit = false })
			if kit.Btn(c, "保存", kit.BtnOpt{Small: true, Variant: kit.BtnPri}).Clicked() {
				a.saveProviderDraft()
				s.provEdit = false
			}
		})
		_ = t
	})
}

func (a *App) saveProviderDraft() {
	p := a.set.provDraft
	p.Name = strings.TrimSpace(p.Name)
	p.BaseURL = strings.TrimSpace(p.BaseURL)
	if p.Name == "" || p.BaseURL == "" {
		a.toast = "请填写名称与地址"
		return
	}
	found := false
	for i, x := range a.set.providers {
		if x.ID == p.ID {
			a.set.providers[i] = p
			found = true
			break
		}
	}
	if !found {
		a.set.providers = append(a.set.providers, p)
	}
	_ = claudecfg.SaveProviders(a.set.providers)
	if k := strings.TrimSpace(a.set.provKey); k != "" {
		_ = claudecfg.SetProviderKey(p.ID, k)
	}
	a.toast = "已保存服务商"
}

func (a *App) deleteProvider(id string) {
	var next []claudecfg.Provider
	for _, p := range a.set.providers {
		if p.ID != id {
			next = append(next, p)
		}
	}
	a.set.providers = next
	_ = claudecfg.SaveProviders(next)
	_ = claudecfg.SetProviderKey(id, "")
}
