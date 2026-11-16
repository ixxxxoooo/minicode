package app

import (
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 会话里的模型选择器(输入框运行配置与状态栏共用):按服务商分组,选别家的模型 = 同时切换服务商。
// 设计稿:docs/design/js/providers.js 的 cfgPop。

// provAbbr 是服务商的缩写徽标:英文取前两个字母大写,中文取前两个字。
func provAbbr(name string) string {
	var out []rune
	for _, r := range name {
		if unicode.IsSpace(r) {
			continue
		}
		if r < 128 {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				out = append(out, unicode.ToUpper(r))
			}
		} else {
			out = append(out, r)
		}
		if len(out) == 2 {
			break
		}
	}
	if len(out) == 0 {
		return "?"
	}
	return string(out)
}

// cfgModels 画「模型」区:Claude 官方一组 + 每个服务商一组,整体限高滚动。
func (a *App) cfgModels(c *ui.Context) {
	t := a.tk
	p := a.pvLoad()
	activeID := p.store.ActiveID()
	kit.VScroll(c).Key("cfg-models").MaxHeight(280).Children(func() {
		ui.Column(c).Children(func() {
			// 官方组
			a.cfgGroupHead(c, "official", "claude", "Claude 官方", strconv.Itoa(len(a.st.Models))+" 个模型", "", activeID == "")
			if a.cfgGroupOpen("official", activeID == "") {
				offPick := a.pvOfficialPick()
				for _, m := range a.st.Models {
					m := m
					if a.mi(c, "model-"+m.ID, "logo:claude", m.Name, shortDesc(m), activeID == "" && offPick == m.ID).Clicked() {
						if activeID != "" {
							a.pvActivate("", "")
						}
						a.pvSetOfficialPick(m.ID)
						a.pickModel(m.ID)
						a.toast = "已切换到 " + m.Name + modelSwitchToastNote()
					}
				}
			}
			for _, g := range p.store.PickerGroups(p.keys) {
				g := g
				pb := provBrand(g.Name, g.BaseURL)
				a.cfgGroupHead(c, g.ID, pb, g.Name, strconv.Itoa(len(g.Models))+" 个模型", g.Blocked, g.Active)
				if g.Blocked != "" || !a.cfgGroupOpen(g.ID, g.Active) {
					continue
				}
				grpPick := p.store.ResolveProviderPick(g.ID)
				for _, m := range g.Models {
					m := m
					row := a.mi(c, "pm-"+g.ID+"-"+m, "logo:"+modelBrand(m, pb), m, "", g.Active && grpPick == m)
					if row.Clicked() {
						a.pvPickModel(g.ID, g.Name, m)
					}
				}
			}
			_ = t
		})
	})
}

// cfgGroupOpen 分组是否展开:默认只展开当前所在的组,点标题切换(存的是「相对默认翻转了没有」)。
func (a *App) cfgGroupOpen(id string, active bool) bool {
	return active != a.pvLoad().folded[id]
}

// prefCfgFolded 记住模型选择器里各服务商分组的展开 / 折叠(存「相对默认翻转过」的分组 id,逗号分隔)。
const prefCfgFolded = "cfg.folded"

// loadCfgFolded 在读入服务商数据时恢复上次的展开 / 折叠状态。
func (a *App) loadCfgFolded(p *provState) {
	if a.prefs == nil {
		return
	}
	for _, id := range strings.Split(a.prefs.Str(prefCfgFolded, ""), ",") {
		if id = strings.TrimSpace(id); id != "" {
			p.folded[id] = true
		}
	}
}

// saveCfgFolded 把当前的展开 / 折叠状态写进偏好(翻转过的分组按 id 排序,输出稳定)。
func (a *App) saveCfgFolded() {
	if a.prefs == nil {
		return
	}
	var ids []string
	for id, v := range a.pvLoad().folded {
		if v {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	a.prefs.Set(prefCfgFolded, strings.Join(ids, ","))
}

// cfgGroupHead 是分组标题行:箭头 + 名称 + 右侧说明;不可用的组置灰,右侧写原因并带「去设置」。
func (a *App) cfgGroupHead(c *ui.Context, id, brand, name, count, blocked string, active bool) {
	t := a.tk
	p := a.pvLoad()
	open := a.cfgGroupOpen(id, active) && blocked == ""
	b := ui.ButtonBase(c).Key("cfg-grp-"+id).Height(theme.Px(24)).PaddingX(8).Gap(4).Radius(t.RSm()).AlignItems(ui.Center).Justify(ui.Start).
		Cursor(ui.CursorDefault).Transition(kit.Fade).Margin(2, 0, 0, 0)
	if blocked != "" {
		b.Opacity(.6)
	} else if b.Hovered() {
		b.Background(t.Fill1)
	}
	if blocked == "" {
		b.OnClick(func() {
			p.folded[id] = !p.folded[id]
			a.saveCfgFolded()
		})
	}
	b.Children(func() {
		kit.Ic(c, map[bool]string{true: "chevD", false: "chevR"}[open], kit.S12).TextColor(t.Tx3)
		kit.Logo(c, brand, 13, t.Tx2)
		ui.Text(c, name).FontSize(theme.Px(10.5)).FontWeight(600).LetterSpacing(.3).TextColor(t.Tx2).SingleLine()
		if active {
			kit.Tag(c, "当前", kit.KAc)
		}
		ui.Box(c).Grow(1)
		if blocked != "" {
			ui.Text(c, blocked).FontSize(theme.FsCap).TextColor(t.Warn).SingleLine()
			if kit.Btn(c, "去设置", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).Key("cfg-go-" + id).Clicked() {
				a.popup = ""
				a.openSettings("models")
			}
		} else {
			ui.Text(c, count).FontSize(theme.FsCap).TextColor(t.Tx4).SingleLine()
		}
	})
}

// pvPickModel 在选择器里点了某个服务商的模型:切到该服务商并把它设为默认槽位的模型。
func (a *App) pvPickModel(providerID, providerName, model string) {
	p := a.pvLoad()
	p.store.SetPick(providerID, model)
	a.pvActivate(providerID, model)
	// 必须发确切的模型 id:实测 set_model "default" 会被 claude 解析成它自己的默认(opus),而不是 ANTHROPIC_MODEL;
	// 而 --model <别名> 的优先级又高于 ANTHROPIC_MODEL,所以启动参数与运行中切换都要显式给 id(见 FillClaudeConfig)。
	a.send(core.CmdSetModel{Model: model})
	a.st.Model = model
	a.popup = ""
	a.toast = "已切换到 " + providerName + " · " + model + modelSwitchToastNote()
}
