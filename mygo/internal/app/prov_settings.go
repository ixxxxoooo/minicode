package app

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/claudecfg"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 设置 → 模型与服务商:服务商列表、添加 / 编辑(含「获取模型列表」与勾选)、模型映射下拉。
// 设计稿:docs/design/js/providers.js。

// providersBody 是「服务商」分组:固定的「Claude 官方登录」行 + 各第三方服务商 + 添加按钮 + 编辑区。
func (a *App) providersBody(a2 *App, c *ui.Context) {
	if !a.cuDemo() {
		a.acctRefresh(false)
	}
	p := a.pvLoad()
	a.provRow(c, nil, true)
	for i := range p.store.Providers {
		pr := p.store.Providers[i]
		a.provRow(c, &pr, false)
	}
	rowBox(a, c, false, 36).Children(func() {
		b := kit.Btn(c, "添加服务商", kit.BtnOpt{Small: true, Icon: "plus"})
		b.OnClick(func() { a.togglePop("prov-add") })
		ui.Text(c, "服务商需兼容 Anthropic 协议;选中后,Opus / Sonnet / Haiku 等槽位映射到它已勾选的模型").
			FontSize(theme.FsCap).TextColor(a.tk.Tx3).Grow(1).MinWidth(0)
		a.provAddMenu(c, b)
	})
	if p.editing {
		a.provEditor(c)
	}
}

// provRow 是列表里的一行;pr == nil 是固定的「Claude 官方登录」。
func (a *App) provRow(c *ui.Context, pr *claudecfg.Provider, first bool) {
	t := a.tk
	p := a.pvLoad()
	id, name, sub := "", "Claude 官方登录", "通过本机 claude 登录"
	if pr != nil {
		id, name, sub = pr.ID, pr.Name, pr.BaseURL
	}
	active := p.store.ActiveID() == id
	rowBox(a, c, first, 52).Children(func() {
		if pr == nil {
			logoBox(a, c, "claude")
		} else {
			logoBox(a, c, provBrand(pr.Name, pr.BaseURL))
		}
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, name).FontWeight(500).SingleLine()
				if pr == nil {
					kit.Tag(c, "已连通", kit.KOk)
				} else if pr.Status != "" && pr.Status != claudecfg.ProvTestNone {
					kit.Tag(c, pr.Status, provStatusKind(pr.Status))
				}
			})
			ui.Text(c, sub).Font("monospace").FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
			if pr == nil {
				ui.Text(c, a.provOfficialMetaLine()).FontSize(theme.FsCap).TextColor(t.Tx4).SingleLine()
			} else {
				meta := "未勾选模型"
				if n := len(pr.Models); n > 0 {
					meta = strconv.Itoa(n) + " 个模型"
				}
				if a.pvHasKey(pr.ID) {
					meta += " · 密钥已保存"
				} else {
					meta += " · 未填写密钥"
				}
				ui.Text(c, meta).FontSize(theme.FsCap).TextColor(t.Tx4).SingleLine()
			}
		})
		if active {
			kit.Tag(c, "当前", kit.KAc)
		} else if kit.Btn(c, "切换", kit.BtnOpt{Small: true}).Key("prov-use-" + id).Clicked() {
			a.provUse(id)
		}
		if pr == nil {
			return
		}
		pid := pr.ID
		kit.Btn(c, "测试", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "activity", Loading: p.busy[pid] == 1}).Key("prov-test-" + pid).
			OnClick(func() { a.testProvider(pid) })
		kit.Ib(c, "edit", kit.IbOpt{Small: true, Tip: "编辑"}).Key("prov-edit-" + pid).OnClick(func() { a.pvOpenEditor(pr, false) })
		kit.Ib(c, "trash", kit.IbOpt{Small: true, Tip: "删除"}).Key("prov-del-" + pid).OnClick(func() { a.provDelete(pid) })
	})
}

// provOfficialMetaLine 官方登录行第三行:套餐 / 账号摘要(不含令牌);与账号菜单同源缓存。
func (a *App) provOfficialMetaLine() string {
	if a.cuDemo() {
		sub := demo.AcctMenuDemo().Subtitle
		if sub != "" {
			return sub
		}
		return "官方账号"
	}
	if a.acct.loading {
		return "读取账号信息…"
	}
	if a.acct.profileReady {
		if sub := strings.TrimSpace(a.acct.subtitle); sub != "" {
			return sub
		}
	}
	return "未找到 OAuth 账户"
}

func provStatusKind(s string) kit.TagKind {
	switch {
	case s == claudecfg.ProvFail:
		return kit.KErr
	case s == claudecfg.ProvOK || strings.HasPrefix(s, claudecfg.ProvOK):
		return kit.KOk
	case s == claudecfg.ProvTestIng:
		return kit.KAc
	}
	return kit.KWarn
}

// provUse 点「切换」:缺密钥 / 缺模型的服务商给出可行动的提示。
func (a *App) provUse(id string) {
	if id == "" {
		a.pvActivate("", "")
		a.toast = "已切换到 Claude 官方登录"
		return
	}
	st := a.pvStore()
	pr := st.Get(id)
	if pr == nil {
		return
	}
	if why := st.Blocked(*pr, a.pvLoad().keys); why != "" {
		a.toast = pr.Name + ":" + why + ",先到「编辑」里补上"
		return
	}
	a.pvActivate(id, "")
	a.toast = "已切换到 " + pr.Name + modelSwitchToastNote()
}

// provAddMenu 是「添加服务商」的下拉菜单:各预设 + 自定义。
func (a *App) provAddMenu(c *ui.Context, anchor ui.Element) {
	t := a.tk
	a.popOpen("prov-add", func(open *bool) {
		ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
			a.popPanel(panel, 280).AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(4, 0, 0, 0)
			kit.Cap(c, "预设")
			for i, ps := range claudecfg.Presets {
				ps := ps
				b := ui.ButtonBase(c).Key("prov-ps-" + strconv.Itoa(i)).Height(34).PaddingX(8).Gap(8).Radius(t.RSm()).AlignItems(ui.Center).Justify(ui.Start).Cursor(ui.CursorDefault).Transition(kit.Fade)
				if b.Hovered() {
					b.Background(t.Fill1)
				}
				b.OnClick(func() {
					a.popup = ""
					a.pvOpenEditor(&claudecfg.Provider{Name: ps.Name, BaseURL: ps.BaseURL, Auth: ps.Auth, Models: ps.Models, Available: ps.Models}, true)
				})
				b.Children(func() {
					ui.Column(c).Grow(1).MinWidth(0).Children(func() {
						ui.Text(c, ps.Name).SingleLine()
						ui.Text(c, claudecfg.HostOf(ps.BaseURL)).Font("monospace").FontSize(10.5).TextColor(t.Tx3).SingleLine()
					})
				})
			}
			kit.Sep(c)
			if a.mi(c, "prov-custom", "plus", "自定义服务商…", "", false).Clicked() {
				a.popup = ""
				a.pvOpenEditor(&claudecfg.Provider{Auth: claudecfg.AuthBearer}, true)
			}
		})
	})
}

// pvOpenEditor 打开编辑区:新增(isNew,src 是预设或空白)或编辑已有服务商。
func (a *App) pvOpenEditor(src *claudecfg.Provider, isNew bool) {
	p := a.pvLoad()
	d := *src
	d.Models = append([]string(nil), src.Models...)
	d.Available = append([]string(nil), src.Available...)
	d.Manual = append([]string(nil), src.Manual...)
	p.editing, p.isNew, p.draft = true, isNew, d
	p.keyDraft, p.keyShow = "", false
	p.errName, p.errURL = "", ""
	p.fetching, p.fetchErr, p.unsupported = false, "", false
	p.filter, p.manual, p.manualErr = "", "", ""
	p.editSeq++
}

// provEditor 是编辑区:基本信息 + 模型清单 + 底栏按钮。
func (a *App) provEditor(c *ui.Context) {
	t := a.tk
	p := a.pvLoad()
	d := &p.draft
	title := "新服务商"
	if !p.isNew {
		title = "编辑 · " + d.Name
	}
	ui.Column(c).Key("prov-editor").Gap(0).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1).Background(t.Fill1.Mix(t.BG, .5)).Children(func() {
		rowBox(a, c, true, 34).Children(func() {
			ui.Text(c, title).FontWeight(600).Grow(1)
			kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭"}).OnClick(func() { p.editing = false })
		})
		a.provFieldRow(c, "名称", "例如 DeepSeek", func() {
			kit.Field(c, "prov-name", kit.FieldOpt{Err: p.errName, ShakeAt: &p.shakeAt}, func(box ui.Element) {
				a.provBoxStyle(box, 260)
				ui.TextInputBase(c.Key("prov-name-in"), &d.Name).Grow(1).FontSize(theme.FsBody).Placeholder("名称").PlaceholderColor(t.Tx4)
			})
		})
		a.provFieldRow(c, "Base URL", "Anthropic 协议兼容的地址", func() {
			kit.Field(c, "prov-url", kit.FieldOpt{Err: p.errURL, ShakeAt: &p.shakeAt}, func(box ui.Element) {
				a.provBoxStyle(box, 260)
				ui.TextInputBase(c.Key("prov-url-in"), &d.BaseURL).Grow(1).FontSize(theme.FsCap + .5).Font("monospace").Placeholder("https://…").PlaceholderColor(t.Tx4)
			})
		})
		ph := "sk-…"
		if !p.isNew && a.pvHasKey(d.ID) {
			ph = "已保存,留空保持原值"
		}
		a.provFieldRow(c, "API Key", "保存后不再显示,留空保持原值", func() {
			kit.Field(c, "prov-key", kit.FieldOpt{}, func(box ui.Element) {
				a.provBoxStyle(box, 220)
				in := ui.TextInputBase(c.Key("prov-key-in"), &p.keyDraft).Grow(1).FontSize(theme.FsCap + .5).Font("monospace").Placeholder(ph).PlaceholderColor(t.Tx4)
				if !p.keyShow {
					in.Password()
				}
			})
			kit.Ib(c, map[bool]string{true: "eyeOff", false: "eye"}[p.keyShow], kit.IbOpt{Small: true, Tip: "显示 / 隐藏"}).Key("prov-key-eye").
				OnClick(func() { p.keyShow = !p.keyShow })
			if a.pvHasKey(d.ID) && !p.isNew && p.keyDraft == "" {
				kit.Tag(c, "已保存", kit.KOk)
			}
		})
		xkey := d.AuthOrDefault() == claudecfg.AuthXAPIKey
		a.provFieldRow(c, "鉴权方式", map[bool]string{true: "请求头 x-api-key(Anthropic 官方)", false: "请求头 Authorization: Bearer(多数兼容服务)"}[xkey], func() {
			on := 0
			if xkey {
				on = 1
			}
			if i := kit.Seg(c, []kit.SegItem{{Text: "Bearer"}, {Text: "x-api-key"}}, on, kit.SegOpt{Key: "prov-auth"}); i >= 0 {
				d.Auth = []string{claudecfg.AuthBearer, claudecfg.AuthXAPIKey}[i]
			}
		})
		a.provFieldRow(c, "模型列表地址", "可选。留空自动尝试 Base URL 下的 /v1/models;有些服务商的列表在别处(如 DeepSeek 是 …/models)", func() {
			kit.Field(c, "prov-murl", kit.FieldOpt{}, func(box ui.Element) {
				a.provBoxStyle(box, 260)
				ui.TextInputBase(c.Key("prov-murl-in"), &d.ModelsURL).Grow(1).FontSize(theme.FsCap + .5).Font("monospace").Placeholder("留空自动尝试 {Base URL}/v1/models").PlaceholderColor(t.Tx4)
			})
		})
		a.provInventory(c)
		rowBox(a, c, false, 40).Children(func() {
			if !p.isNew {
				if kit.Btn(c, "删除", kit.BtnOpt{Small: true, Variant: kit.BtnDangerGhost, Icon: "trash"}).Key("prov-del-btn").Clicked() {
					id := d.ID
					p.editing = false
					a.provDelete(id)
				}
			}
			ui.Box(c).Grow(1)
			kit.Btn(c, "取消", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).Key("prov-cancel").OnClick(func() { p.editing = false })
			kit.Btn(c, "保存", kit.BtnOpt{Small: true}).Key("prov-save").OnClick(func() { a.pvSaveDraft(false) })
			kit.Btn(c, "保存并测试", kit.BtnOpt{Small: true, Variant: kit.BtnPri}).Key("prov-save-test").OnClick(func() { a.pvSaveDraft(true) })
		})
	})
}

// provBoxStyle 是编辑区输入框外壳:定宽、24 高。
func (a *App) provBoxStyle(box ui.Element, w float32) {
	box.Height(24).Width(w).PaddingX(8).Shrink(0)
}

// provFieldRow 是编辑区的一行:左侧标题 + 说明,右侧控件(输入校验错误由 kit.Field 画在输入框下方)。
func (a *App) provFieldRow(c *ui.Context, title, desc string, right func()) {
	t := a.tk
	rowBox(a, c, false, 40).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			ui.Text(c, title).FontWeight(500).SingleLine()
			if desc != "" {
				ui.Text(c, desc).FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(theme.LhCap)
			}
		})
		ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).Children(right)
	})
}

// pvSaveDraft 校验并保存编辑区;test 为真时保存后立刻测试连接。
func (a *App) pvSaveDraft(test bool) {
	p := a.pvLoad()
	d := p.draft
	d.Name = strings.TrimSpace(d.Name)
	d.BaseURL = strings.TrimSpace(d.BaseURL)
	d.ModelsURL = strings.TrimSpace(d.ModelsURL)
	p.errName, p.errURL = "", ""
	switch {
	case d.BaseURL == "":
		p.errURL = "请填写 Base URL"
	case !strings.HasPrefix(d.BaseURL, "http://") && !strings.HasPrefix(d.BaseURL, "https://"):
		p.errURL = "地址需以 http:// 或 https:// 开头"
	}
	if d.Name == "" {
		p.errName = "请填写名称"
	}
	if p.errName != "" || p.errURL != "" {
		p.shakeAt = time.Now()
		return
	}
	// 勾选只保留还在清单里的
	all := d.AllModels()
	var picked []string
	for _, m := range d.Models {
		for _, x := range all {
			if m == x {
				picked = append(picked, m)
				break
			}
		}
	}
	d.Models = picked
	if p.isNew {
		d.ID = p.store.NewID(d.Name)
		p.store.Providers = append(p.store.Providers, d)
	} else {
		for i := range p.store.Providers {
			if p.store.Providers[i].ID == d.ID {
				p.store.Providers[i] = d
			}
		}
	}
	if k := strings.TrimSpace(p.keyDraft); k != "" {
		a.pvSetKey(d.ID, k)
	}
	if p.store.Mapping.Provider == d.ID {
		p.store.CleanMapping()
	}
	a.pvPersist()
	if p.store.ActiveID() == d.ID {
		a.pvPushEnv()
	}
	p.editing = false
	a.toast = "已保存服务商 " + d.Name
	provLog.Info("保存服务商", "id", d.ID, "name", d.Name, "host", claudecfg.HostOf(d.BaseURL), "models", len(d.Models), "new", p.isNew)
	if test {
		a.testProvider(d.ID)
	}
}

// provDelete 删除服务商(6 秒内可撤销);正在使用的服务商被删除后回到 Claude 官方。
func (a *App) provDelete(id string) {
	p := a.pvLoad()
	idx := -1
	for i, x := range p.store.Providers {
		if x.ID == id {
			idx = i
		}
	}
	if idx < 0 {
		return
	}
	gone := p.store.Providers[idx]
	oldMap := p.store.Mapping
	oldKey := p.keys[id]
	wasActive := p.store.ActiveID() == id
	p.store.Providers = append(append([]claudecfg.Provider{}, p.store.Providers[:idx]...), p.store.Providers[idx+1:]...)
	if p.store.Mapping.Provider == id {
		p.store.Mapping = claudecfg.Mapping{}
	}
	a.pvSetKey(id, "")
	a.pvPersist()
	if wasActive {
		a.pvPushEnv()
	}
	provLog.Info("删除服务商", "id", id)
	a.set.undo.Begin("已删除服务商 "+gone.Name, func() {
		at := min(idx, len(p.store.Providers))
		rest := append([]claudecfg.Provider{}, p.store.Providers[:at]...)
		rest = append(rest, gone)
		rest = append(rest, p.store.Providers[at:]...)
		p.store.Providers = rest
		p.store.Mapping = oldMap
		if oldKey != "" {
			p.keys[id] = oldKey
			if !a.demoWs() {
				_ = claudecfg.SetProviderKey(id, oldKey)
			}
		}
		a.pvPersist()
		if wasActive {
			a.pvPushEnv()
		}
	})
}

// testProvider 测试连接:真实模式探测 Base URL 是否可达;演示模式模拟(地址含 example.com 失败)。
func (a *App) testProvider(id string) {
	p := a.pvLoad()
	idx := -1
	for i, x := range p.store.Providers {
		if x.ID == id {
			idx = i
		}
	}
	if idx < 0 {
		return
	}
	p.busy[id] = 1
	p.store.Providers[idx].Status = claudecfg.ProvTestIng
	prName := p.store.Providers[idx].Name
	base := p.store.Providers[idx].BaseURL
	demoMode := a.demoWs()
	a.bg("连通性测试", func() func() {
		var res claudecfg.ProbeResult
		if demoMode {
			time.Sleep(900 * time.Millisecond)
			if strings.Contains(base, "example.com") {
				res = claudecfg.ProbeResult{Kind: claudecfg.ProbeAuth, Detail: "密钥无效或已过期(401),到 设置 → 服务商 更新 API Key"}
			} else {
				res = claudecfg.ProbeResult{OK: true, Kind: claudecfg.ProbeOK, Ms: int64(80 + len(base)%90), Detail: claudecfg.FormatTestResult(true, int64(80+len(base)%90))}
			}
		} else {
			key := strings.TrimSpace(p.keys[id])
			res = claudecfg.ProbeProvider(p.store.Providers[idx], key)
		}
		st := claudecfg.FormatProbe(res)
		return func() {
			p.busy[id] = 0
			for i := range p.store.Providers {
				if p.store.Providers[i].ID == id {
					p.store.Providers[i].Status = st
				}
			}
			a.pvPersist()
			if res.OK {
				a.toast = prName + " 连接正常 · 用时 " + strconv.FormatInt(res.Ms, 10) + " ms"
			} else {
				a.toast = res.Detail
			}
		}
	})
}

// ───────── 模型清单(编辑区里的重点块) ─────────

// provInventory 是「模型清单」:获取按钮 + 过滤 / 全选 + 勾选列表 + 手动添加,以及各种状态。
func (a *App) provInventory(c *ui.Context) {
	t := a.tk
	p := a.pvLoad()
	d := &p.draft
	all := d.AllModels()
	q := strings.ToLower(strings.TrimSpace(p.filter))
	var rows []string
	for _, m := range all {
		if q == "" || strings.Contains(strings.ToLower(m), q) {
			rows = append(rows, m)
		}
	}
	picked := 0
	for _, m := range all {
		if d.HasModel(m) {
			picked++
		}
	}
	hasKey := strings.TrimSpace(p.keyDraft) != "" || (!p.isNew && a.pvHasKey(d.ID))
	ui.Column(c).Gap(8).Padding(10, 12).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "模型清单").FontWeight(600)
			ui.Text(c, "已选 "+strconv.Itoa(picked)+" / 共 "+strconv.Itoa(len(all))).FontSize(theme.FsCap).TextColor(t.Tx3)
			ui.Box(c).Grow(1)
			if d.FetchedAt > 0 && !p.fetching {
				ui.Text(c, "上次获取 · "+relTime(time.Now(), time.Unix(d.FetchedAt, 0))).FontSize(theme.FsCap).TextColor(t.Tx4)
			}
			tip := ""
			if !hasKey {
				tip = "先填写 API Key"
			}
			kit.Btn(c, "获取模型列表", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "refresh", Loading: p.fetching, Disabled: !hasKey, Tip: tip}).
				Key("prov-fetch").OnClick(func() { a.pvFetch() })
		})
		ui.Text(c, "只有勾选的模型会出现在「模型映射」下拉与会话里的模型选择器。").FontSize(theme.FsCap).TextColor(t.Tx3)
		if p.fetchErr != "" {
			a.provBanner(c, t.Err, t.ErrSoft, "errc", p.fetchErr, "重试", func() { a.pvFetch() }, "改为手动添加", func() { p.focusManual = true })
		}
		if p.unsupported {
			a.provBanner(c, t.Warn, t.WarnSoft, "alert", "该服务商没有模型列表接口,请手动添加,或在上方填写「模型列表地址」。", "手动添加", func() { p.focusManual = true }, "", nil)
		}
		switch {
		case p.fetching:
			for i := 0; i < 4; i++ {
				ui.Row(c).Height(26).Gap(8).AlignItems(ui.Center).Opacity(.6 + .4*float32(i%2)).Children(func() {
					ui.Box(c).Size(14, 14).Radius(4).Background(t.Fill2).Shrink(0)
					ui.Box(c).Height(10).Radius(4).Background(t.Fill2).Grow(1)
				})
			}
			a.frameSoon()
		case len(all) == 0:
			emptyFetchTip := ""
			if !hasKey {
				emptyFetchTip = "先填写 API Key"
			}
			ui.Column(c).Gap(8).AlignItems(ui.Center).Padding(14, 0).Children(func() {
				kit.Ic(c, "list", kit.S20).TextColor(t.Tx4)
				ui.Text(c, "还没有模型").FontWeight(500)
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					kit.Btn(c, "获取模型列表", kit.BtnOpt{Small: true, Variant: kit.BtnPri, Icon: "refresh", Disabled: !hasKey, Tip: emptyFetchTip}).
						Key("prov-fetch-empty").OnClick(func() { a.pvFetch() })
					kit.Btn(c, "手动添加", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "plus"}).
						Key("prov-manual-focus").OnClick(func() { p.focusManual = true })
				})
			})
		default:
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Grow(1).Height(24).PaddingX(8).Gap(6).AlignItems(ui.Center).Background(t.Fill1).Radius(t.RSm()).Children(func() {
					kit.Ic(c, "search", kit.S12).TextColor(t.Tx3)
					ui.TextInputBase(c.Key("prov-filter"), &p.filter).Grow(1).FontSize(theme.FsCap + .5).Placeholder("过滤模型 ID").PlaceholderColor(t.Tx4)
				})
				kit.Btn(c, "全选", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).Key("prov-all").OnClick(func() { d.SelectAll(rows) })
				kit.Btn(c, "清空", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).Key("prov-none").OnClick(func() { d.SelectNone(rows) })
			})
			kit.VScroll(c).Key("prov-models").MaxHeight(220).Children(func() {
				ui.Column(c).Gap(1).Children(func() {
					for _, m := range rows {
						a.provModelRow(c, d, m)
					}
					if len(rows) == 0 {
						ui.Text(c, "没有匹配的模型").FontSize(theme.FsCap).TextColor(t.Tx3).Padding(8, 6)
					}
				})
			})
		}
		// 手动添加一行:输入框 + 添加(回车也行)
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			kit.Field(c, "prov-manual", kit.FieldOpt{Err: p.manualErr, ShakeAt: &p.shakeAt}, func(box ui.Element) {
				box.Height(24).Grow(1).PaddingX(8)
				in := ui.TextInputBase(c.Key("prov-manual-in"), &p.manual).Grow(1).FontSize(theme.FsCap + .5).Font("monospace").Placeholder("手动添加模型 ID").PlaceholderColor(t.Tx4)
				if p.focusManual {
					in.AutoFocus()
					p.focusManual = false
				}
				in.OnSubmit(func() { a.pvAddManual() })
			})
			kit.Btn(c, "添加", kit.BtnOpt{Small: true, Icon: "plus"}).Key("prov-manual-add").OnClick(func() { a.pvAddManual() })
		})
	})
}

// provModelRow 是清单里的一行:勾选圆 + 模型 ID;手动添加的带「手动」标签与移除按钮。整行可点。
func (a *App) provModelRow(c *ui.Context, d *claudecfg.Provider, m string) {
	t := a.tk
	on := d.HasModel(m)
	manual := false
	for _, x := range d.Manual {
		if x == m {
			manual = true
		}
	}
	b := ui.ButtonBase(c).Key("prov-m-" + m).Height(26).PaddingX(8).Gap(8).Radius(t.RSm()).AlignItems(ui.Center).Justify(ui.Start).Cursor(ui.CursorPointer).Transition(kit.Fade)
	switch {
	case on:
		b.Background(t.AcSoft)
	case b.Hovered():
		b.Background(t.Fill1)
	}
	b.OnClick(func() { d.ToggleModel(m) })
	b.Children(func() {
		kit.Chk(c, map[bool]int{true: 2, false: 0}[on])
		kit.Logo(c, modelBrand(m, provBrand(d.Name, d.BaseURL)), 14, t.Tx2)
		ui.Text(c, m).Font("monospace").FontSize(theme.FsCap + .5).SingleLine().Grow(1).MinWidth(0)
		if manual {
			kit.Tag(c, "手动", kit.KInfo)
			if kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "移除"}).Key("prov-mrm-" + m).Clicked() {
				d.RemoveManual(m)
			}
		}
	})
}

// provBanner 是清单上方的提示条(失败 / 不支持),可带最多两个动作。
func (a *App) provBanner(c *ui.Context, fg, bg ui.Color, icon, text, b1 string, f1 func(), b2 string, f2 func()) {
	ui.Row(c).Gap(8).Padding(6, 10).AlignItems(ui.Center).Background(bg).Radius(a.tk.RSm()).Children(func() {
		kit.Ic(c, icon, kit.S12).TextColor(fg)
		ui.Text(c, text).FontSize(theme.FsCap).TextColor(fg).Grow(1).MinWidth(0)
		if b1 != "" {
			kit.Btn(c, b1, kit.BtnOpt{Small: true}).Key("prov-bn1").OnClick(f1)
		}
		if b2 != "" {
			kit.Btn(c, b2, kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).Key("prov-bn2").OnClick(f2)
		}
	})
}

// pvAddManual 手动添加模型:重复提示,成功后自动勾选。
func (a *App) pvAddManual() {
	p := a.pvLoad()
	id := strings.TrimSpace(p.manual)
	if id == "" {
		return
	}
	if !p.draft.AddManual(id) {
		p.manualErr = "已在清单中"
		p.shakeAt = time.Now()
		return
	}
	p.manual, p.manualErr = "", ""
	p.unsupported = false
}

// pvFetch 在后台获取模型列表,结果并入编辑区的草稿(不直接写盘,点「保存」才生效)。
func (a *App) pvFetch() {
	p := a.pvLoad()
	if p.fetching {
		return
	}
	d := p.draft
	key := strings.TrimSpace(p.keyDraft)
	if key == "" && !p.isNew {
		key = p.keys[d.ID]
	}
	if key == "" {
		a.toast = "先填写 API Key"
		return
	}
	if strings.TrimSpace(d.BaseURL) == "" && strings.TrimSpace(d.ModelsURL) == "" {
		p.errURL = "请填写 Base URL"
		p.shakeAt = time.Now()
		return
	}
	p.fetching, p.fetchErr, p.unsupported = true, "", false
	seq := p.editSeq
	demoMode := a.demoWs()
	provLog.Info("获取模型列表", "provider", d.ID, "host", claudecfg.HostOf(d.BaseURL))
	a.bg("获取模型列表", func() func() {
		var ids []string
		var err error
		if demoMode {
			time.Sleep(900 * time.Millisecond)
			ids, err = demo.DemoFetchModels(d)
		} else {
			ids, err = claudecfg.FetchModels(d, key)
		}
		return func() {
			if p.editSeq != seq || !p.editing { // 编辑区已关闭或换了别的服务商:丢弃
				return
			}
			p.fetching = false
			switch {
			case err == nil:
				p.draft.MergeFetched(ids, time.Now())
				a.toast = "获取到 " + strconv.Itoa(len(ids)) + " 个模型"
			case errors.Is(err, claudecfg.ErrModelsUnsupported):
				p.unsupported = true
			default:
				p.fetchErr = err.Error()
			}
		}
	})
}

// ───────── 模型映射 ─────────

// mappingBody 是「模型映射」分组:选服务商 + 五个槽位的下拉(只列该服务商已勾选的模型)。
func (a *App) mappingBody(a2 *App, c *ui.Context) {
	t := a.tk
	p := a.pvLoad()
	st := p.store
	official := !st.Mapping.Enabled
	// 使用的服务商
	rowBox(a, c, true, 40).Children(func() {
		titled(a, c, "使用的服务商", "切换后,新建或重新连接的会话使用该服务商", false)
		reason := st.MappingStatus(p.keys)
		switch {
		case official:
			kit.Tag(c, "使用 Claude 官方登录", kit.KNeutral)
		case reason == "":
			kit.Tag(c, "映射生效中", kit.KOk)
		default:
			kit.Tag(c, "未生效 · "+reason, kit.KWarn)
		}
		items := []selItem{{id: "official", label: "Claude 官方登录", logo: true, brand: "claude"}}
		for _, pr := range st.Providers {
			why := st.Blocked(pr, p.keys)
			items = append(items, selItem{id: pr.ID, label: pr.Name, disabled: why != "", hint: why, logo: true, brand: provBrand(pr.Name, pr.BaseURL)})
		}
		cur := "official"
		if !official {
			cur = st.Mapping.Provider
		}
		a.selectBox(c, "map-prov", 190, items, cur, "选择服务商", false, func(id string) {
			if id == "official" {
				a.provUse("")
			} else {
				a.provUse(id)
			}
		})
	})
	pr := st.Get(st.Mapping.Provider)
	if official || pr == nil {
		rowBox(a, c, false, 36).Children(func() {
			ui.Text(c, "使用 Claude 官方登录,不注入模型映射。选一个第三方服务商后,在这里把 Opus / Sonnet / Haiku 等槽位指到它的模型。").
				FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1).MinWidth(0)
		})
		return
	}
	for _, sl := range claudecfg.Slots {
		sl := sl
		rowBox(a, c, false, 40).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
				ui.Text(c, sl.Label).FontWeight(500).SingleLine().Tooltip(sl.Env)
				ui.Text(c, sl.Env).Font("monospace").FontSize(10.5).TextColor(t.Tx4).SingleLine()
			})
			var items []selItem
			if !sl.Required {
				items = append(items, selItem{id: "", label: "不设置(沿用默认)"})
			}
			pb := provBrand(pr.Name, pr.BaseURL)
			for _, m := range pr.Models {
				items = append(items, selItem{id: m, label: m, mono: true, logo: true, brand: modelBrand(m, pb)})
			}
			cur := st.Mapping.Get(sl.Key)
			empty := "不设置(沿用默认)"
			if sl.Required {
				empty = "请选择"
			}
			if len(pr.Models) == 0 {
				empty = "先在服务商里勾选模型"
			}
			a.selectBox(c, "map-"+sl.Key, 220, items, cur, empty, true, func(id string) {
				p.store.Mapping.Set(sl.Key, id)
				a.pvPersist()
				a.pvPushEnv()
				provLog.Info("模型映射", "slot", sl.Key, "model", id, "provider", pr.ID)
			})
		})
	}
	rowBox(a, c, false, 32).Children(func() {
		ui.Text(c, "映射以环境变量在启动 claude 时注入,不改 ~/.claude;对之后新建或重新连接的会话生效。悬停槽位名可看对应的环境变量。").
			FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1).MinWidth(0)
	})
}
