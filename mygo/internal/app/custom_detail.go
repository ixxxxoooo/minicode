package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 自定义页右侧抽屉:各类的专属详情。命令 / 子 Agent / 规则 / 钩子是表单,改的是草稿 `cu.draft`,「保存」才写回;
// 插件 / MCP / Skills 的动作(启停、工具开关、重连、授权)立即生效。

func (a *App) cuDrawer(c *ui.Context, st *customize.Store) {
	t := a.tk
	s := &a.cu
	cat := a.cuCat()
	w := float32(drawerW)
	if cat == customize.Commands || cat == customize.Agents || cat == customize.Rules || cat == customize.Hooks {
		w = 360
	}
	col := ui.Column(c).Key("cu-drawer").Shrink(0).MinHeight(0).Clip()
	if a.pageNarrowMid() {
		col.Grow(1).MinWidth(0)
	} else {
		col.Width(w).BorderWidth(0, 0, 0, 1).BorderColor(t.Bd1)
	}
	col.Children(func() {
		if cat == customize.Plugins && s.market {
			a.cuMarketDetail(c, st)
			return
		}
		it := s.sel
		if it == nil {
			ui.Text(c, "没有可显示的"+custCats[cat].title).FontSize(theme.FsCap).TextColor(t.Tx4).TextAlign(ui.Center).Padding(20, 0)
			return
		}
		if s.dOf != it { // 换了一项:重新载入草稿
			s.draft, s.dOf = it.Clone(), it
		}
		var foot func()
		ui.Scroll(c).Key("cu-dscroll-" + string(cat) + "-" + it.Name).Grow(1).MinHeight(0).Clip().Children(func() {
			ui.Column(c).Padding(14, 16, 14, 16).Gap(12).Children(func() {
				switch cat {
				case customize.Plugins:
					foot = a.cuPlugin(c, st, it)
				case customize.MCP:
					foot = a.cuMCP(c, st, it)
				case customize.Skills:
					foot = a.cuSkill(c, it)
				case customize.Commands:
					foot = a.cuCommand(c, st, it)
				case customize.Agents:
					foot = a.cuAgent(c, st, it)
				case customize.Rules:
					foot = a.cuRule(c, st, it)
				case customize.Hooks:
					foot = a.cuHook(c, st, it)
				}
				if foot != nil {
					ui.Row(c).Key("cu-foot").Margin(8, 0, 0, 0).Padding(10, 0, 0, 0).Gap(8).AlignItems(ui.Center).
						BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1).Children(foot)
				}
			})
		})
	})
}

// ───────── 公共小部件 ─────────

func (a *App) cuHead(c *ui.Context, st *customize.Store, it *customize.Item, icon, name string, extra func()) {
	t := a.tk
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		kit.Ic(c, icon, kit.S12).TextColor(t.Ac)
		ui.Text(c, name).FontWeight(600).Grow(1).MinWidth(0).SingleLine()
		if extra != nil {
			extra()
		}
		mb := kit.Ib(c, "more", kit.IbOpt{Small: true, Tip: "更多"})
		if it != nil {
			mb.Menu(func(m *ui.Menu) { a.cuMoreMenu(m, st, it) })
		}
	})
}

func (a *App) cuBanner(c *ui.Context, kind kit.TagKind, icon, text string, acts func()) {
	t := a.tk
	// 设计稿的提示条底色几乎是中性的,只有图标带级别色;错误与警告多一点淡底
	bg, fg := t.Fill1, t.Ok
	switch kind {
	case kit.KErr:
		bg, fg = t.ErrSoft, t.Err
	case kit.KWarn:
		bg, fg = t.WarnSoft, t.Warn
	case kit.KInfo:
		bg, fg = t.Fill1, t.Info
	}
	ui.Row(c).Padding(8, 10).Gap(8).AlignItems(ui.Center).Background(bg).Radius(t.RMd()).Children(func() {
		kit.Ic(c, icon, kit.S12).TextColor(fg)
		ui.Text(c, text).Grow(1).MinWidth(0).LineHeight(1.5)
		if acts != nil {
			acts()
		}
	})
}

// cuKV 是「名称 / 值」两列;值像路径或命令时用等宽字。
func (a *App) cuKV(c *ui.Context, rows [][2]string) {
	t := a.tk
	ui.Column(c).Gap(6).Children(func() {
		for _, r := range rows {
			ui.Row(c).Gap(6).AlignItems(ui.Start).Children(func() {
				ui.Text(c, r[0]).TextColor(t.Tx3).Width(72).Shrink(0)
				v := ui.Text(c, r[1]).Grow(1).MinWidth(0).LineHeight(1.4)
				if strings.ContainsAny(r[1], "/") || strings.Contains(r[1], "npx") || strings.Contains(r[1], ".md") {
					v.Font("monospace").FontSize(11.5)
				}
			})
		}
	})
}

func (a *App) cuField(c *ui.Context, key string, v *string, mono bool, ph string) {
	in := ui.TextInput(c.Key(key), v).Height(26).FontSize(theme.FsBody).Placeholder(ph).PlaceholderColor(a.tk.Tx4)
	if mono {
		in.Font("monospace").FontSize(11.5)
	}
}

func (a *App) cuArea(c *ui.Context, key string, v *string, h float32, mono bool) {
	in := ui.TextArea(c.Key(key), v).Height(h).FontSize(theme.FsBody)
	if mono {
		in.Font("monospace").FontSize(11.5)
	}
}

// cuCard 是带描边的小卡片(行之间不画线)。
func (a *App) cuCard(c *ui.Context, body func()) {
	a.box(c).Padding(2, 4).Children(body)
}

func (a *App) cuToggleRow(c *ui.Context, key, title, sub string, on bool, flip func()) {
	t := a.tk
	ui.Row(c).Padding(6, 8).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			ui.Text(c, title).FontWeight(500)
			if sub != "" {
				ui.Text(c, sub).FontSize(theme.FsMicro).TextColor(t.Tx3).LineHeight(1.4)
			}
		})
		if kit.Tog(c, on, key) {
			flip()
		}
	})
}

func (a *App) cuDeleteAsk(st *customize.Store, it *customize.Item, verb string) {
	a.confirmDlg(verb+"「"+it.Name+"」?", "这会从当前范围移除它的配置。", verb, true, func() {
		a.cuDeleteDo(st, it, verb)
	})
}

// cuHeadDel 在详情标题行放删除(避免底部操作条被滚动层挡住点击)。
func (a *App) cuHeadDel(c *ui.Context, st *customize.Store, it *customize.Item, verb string) func() {
	return func() {
		kit.Ib(c, "trash", kit.IbOpt{Small: true, Tip: verb}).Key("cu-del").OnClick(func() {
			a.cuDeleteAsk(st, it, verb)
		})
	}
}

func (a *App) cuDeleteDo(st *customize.Store, it *customize.Item, verb string) {
	cat := a.cuCat()
	snap := *it
	if a.cuDisk() != nil && cat != customize.Plugins {
		a.cuRemoveDisk(st, it)
	} else if cat == customize.Plugins {
		if !a.cuDemo() {
			res := customize.RunPluginUninstall(it.Name)
			if res.Code != 0 && res.Err != "" {
				a.toast = res.Err
				return
			}
			if ds := a.cuDisk(); ds != nil {
				ds.Reload()
			}
		}
		st.Uninstall(it)
	} else {
		st.Remove(cat, it)
	}
	log.Info("删除自定义项", "cat", cat, "name", it.Name)
	a.cu.sel, a.cu.dOf, a.cu.delArm = nil, nil, nil
	a.delUndo.Begin("已"+verb+"「"+it.Name+"」", func() {
		if cat == customize.Plugins {
			return
		}
		n := &snap
		if ni, err := st.Add(cat, n.Name, n.Scope, n.Note); err == nil {
			*ni = snap
		}
		a.cu.sel = n
	})
}

func (a *App) cuSave(st *customize.Store, it *customize.Item) {
	a.cuSaveDisk(st, it)
}

func (a *App) cuErr(c *ui.Context) {
	if a.cu.err != "" {
		ui.Text(c, a.cu.err).FontSize(theme.FsCap).TextColor(a.tk.Err)
	}
}

// ───────── 插件 ─────────

func (a *App) cuMarketDetail(c *ui.Context, st *customize.Store) {
	t := a.tk
	m := a.cu.mk
	if m == nil {
		return
	}
	ui.Scroll(c).Key("cu-mkscroll").Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(14, 16).Gap(12).Children(func() {
			a.cuHead(c, st, nil, "plug", m.Name, func() { kit.Tag(c, m.By, kit.KAc) })
			ui.Text(c, m.Desc+"。安装后会把它包含的 Skills、命令、子 Agent 与钩子一并加入。").TextColor(t.Tx2).LineHeight(1.6)
			a.cuKV(c, [][2]string{{"来源", "claude-plugins-official"}, {"热度", "★ " + m.Stars}, {"安装到", "用户(所有项目)"}})
			a.cuLbl(c, "将包含", func() {
				a.cuCard(c, func() {
					for _, r := range []struct {
						i, n string
						v    int
					}{{"spark", "Skills", 6}, {"slash", "斜杠命令", 2}, {"bot", "子 Agent", 1}} {
						ui.Row(c).Height(t.Scheme.RowH).PaddingX(8).Gap(6).AlignItems(ui.Center).Children(func() {
							kit.Ic(c, r.i, kit.S12)
							ui.Text(c, r.n).Grow(1)
							ui.Text(c, itoa(r.v)).TextColor(t.Tx3)
						})
					}
				})
			})
			ui.Row(c).Key("cu-mkfoot").Margin(8, 0, 0, 0).Padding(10, 0, 0, 0).Gap(8).AlignItems(ui.Center).
				BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1).Children(func() {
				if m.Installed {
					kit.Tag(c, "已安装", kit.KOk)
				} else if a.cu.plug != nil && a.cu.plug.name == m.Name {
					phases := []string{"正在下载…", "正在解包…", "安装完成", "安装失败"}
					p := a.cu.plug.phase
					if p < 0 || p >= len(phases) {
						p = 0
					}
					kit.Tag(c, phases[p], kit.KInfo)
					if a.cu.plug.phase < 3 {
						kit.Btn(c, "安装中", kit.BtnOpt{Variant: kit.BtnPri, Loading: true, Disabled: true})
					} else if a.cu.plug.phase == 3 {
						kit.Btn(c, "重试", kit.BtnOpt{Small: true, Variant: kit.BtnSoft, Icon: "refresh"}).OnClick(func() { a.cuInstallPlugin(st, m) })
					}
				} else {
					kit.Btn(c, "安装", kit.BtnOpt{Variant: kit.BtnPri, Icon: "download"}).OnClick(func() { a.cuInstallPlugin(st, m) })
				}
				ui.Spacer(c).Grow(1)
				kit.Btn(c, "查看源码", kit.BtnOpt{Variant: kit.BtnGhost, Icon: "ext"}).OnClick(func() { a.toast = "查看源码:尚未接入" })
			})
		})
	})
}

func (a *App) cuPlugin(c *ui.Context, st *customize.Store, it *customize.Item) func() {
	t := a.tk
	a.cuHead(c, st, it, "plug", it.Name, func() {
		kit.Tag(c, "v"+it.Ver)
		a.cuHeadDel(c, st, it, "卸载")()
	})
	if it.Upd != "" {
		a.cuBanner(c, kit.KInfo, "download", "有新版本 "+it.Upd, func() {
			kit.Btn(c, "更新", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).OnClick(func() {
				it.Update()
				a.toast = "已更新「" + it.Name + "」到 v" + it.Ver
			})
		})
	}
	ui.Text(c, it.About).TextColor(t.Tx2).LineHeight(1.6)
	a.cuKV(c, [][2]string{{"来源", it.Src}, {"版本", it.Ver}})
	a.cuLbl(c, "安装到", func() {
		on := 0
		if it.Scope == customize.Project {
			on = 1
		}
		if i := kit.Seg(c, []kit.SegItem{{Text: "用户"}, {Text: "项目"}}, on, kit.SegOpt{Key: "cp-scope"}); i >= 0 {
			it.Scope = []customize.Scope{customize.User, customize.Project}[i]
		}
	})
	a.cuLbl(c, "包含内容", func() {
		a.cuCard(c, func() {
			for i := range it.Inc {
				inc := &it.Inc[i]
				ui.Row(c).Height(t.Scheme.RowH).PaddingX(8).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, inc.Kind).Grow(1)
					ui.Text(c, itoa(inc.N)).TextColor(t.Tx3)
					if kit.Tog(c, inc.On, "cpi-"+it.Name+inc.Kind) {
						inc.On = !inc.On
					}
				})
			}
		})
	})
	return func() {
		ui.Spacer(c).Grow(1)
		label := "启用"
		if it.On {
			label = "停用"
		}
		kit.Btn(c, label).OnClick(func() { it.On = !it.On })
	}
}

// ───────── MCP ─────────

func (a *App) cuMCP(c *ui.Context, st *customize.Store, it *customize.Item) func() {
	t := a.tk
	a.cuHead(c, st, it, "server", it.Name, a.cuHeadDel(c, st, it, "删除"))
	switch it.Status {
	case customize.StOK:
		a.cuBanner(c, kit.KOk, "check", "已连接 · "+itoa(len(it.Tools))+" 个工具", func() {
			kit.Btn(c, "重新连接", kit.BtnOpt{Small: true, Icon: "refresh"}).OnClick(func() { a.cuReconnectReal(it) })
		})
	case customize.StWarn:
		a.cuBanner(c, kit.KWarn, "alert", "需要授权后才能使用", func() {
			kit.Btn(c, "授权", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).OnClick(func() { a.cuAuth(it) })
		})
	default:
		reason := ""
		if i := strings.Index(it.Desc, ":"); i >= 0 {
			reason = it.Desc[i+len(":"):]
		}
		a.cuBanner(c, kit.KErr, "alert", "连接失败:"+reason+"。检查命令与环境变量后重试。", func() {
			kit.Btn(c, "重试", kit.BtnOpt{Small: true, Icon: "refresh"}).OnClick(func() { a.cuReconnectReal(it) })
		})
	}
	where := "命令"
	if it.Transport == "http" {
		where = "地址"
	}
	a.cuKV(c, [][2]string{{"传输", it.Transport}, {where, it.Cmd}})
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "配置层").TextColor(t.Tx3).Width(72).Shrink(0)
		on := 0
		for i, sc := range cuScopes[1:] {
			if sc == it.Scope {
				on = i
			}
		}
		if i := kit.Seg(c, []kit.SegItem{{Text: "用户"}, {Text: "项目"}, {Text: "本地"}}, on, kit.SegOpt{Key: "cm-scope"}); i >= 0 {
			it.Scope = cuScopes[1:][i]
		}
	})
	a.cuKV(c, [][2]string{{"工具定义占用", it.Tok}})
	if len(it.Env) > 0 {
		a.cuLbl(c, "环境变量(值已遮盖)", func() {
			a.cuCard(c, func() {
				for _, n := range it.Env {
					ui.Row(c).Height(t.Scheme.RowH).PaddingX(8).Gap(6).AlignItems(ui.Center).Children(func() {
						kit.Mono(c, n, t.Tx1).Grow(1)
						kit.Mono(c, "••••••••", t.Tx3)
					})
				}
			})
		})
	}
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "工具("+itoa(len(it.Tools))+")").FontWeight(600).Grow(1)
		ui.Text(c, "逐项开关 · 写入本地层 deny 规则").FontSize(theme.FsCap).TextColor(t.Tx3)
	})
	if len(it.Tools) == 0 {
		msg := "连接成功后才会列出工具"
		if it.Status == customize.StOK {
			msg = "没有工具"
		}
		ui.Text(c, msg).FontSize(theme.FsCap).TextColor(t.Tx4)
	}
	ui.Column(c).Children(func() {
		for i := range it.Tools {
			ui.Row(c).Height(t.Scheme.RowH).PaddingX(8).Gap(6).AlignItems(ui.Center).Children(func() {
				kit.Mono(c, it.Tools[i].Name, t.Tx1).Grow(1).SingleLine()
				if kit.Tog(c, it.Tools[i].On, "cmt-"+it.Name+"-"+it.Tools[i].Name) {
					a.cuMCPTool(it, i)
				}
			})
		}
	})
	a.cuCard(c, func() {
		a.cuToggleRow(c, "cm-ondemand-"+it.Name, "按需加载工具", "ENABLE_TOOL_SEARCH · 工具很多时只在需要时加载描述,节省上下文", it.OnDemand, func() { it.OnDemand = !it.OnDemand })
	})
	return func() {
		ui.Spacer(c).Grow(1)
		kit.Btn(c, "编辑配置", kit.BtnOpt{Icon: "edit"}).OnClick(func() { a.cuMCPStartEdit(it) })
	}
}

func (a *App) cuReconnect(it *customize.Item) {
	it.Reconnect()
	if it.Status == customize.StOK {
		a.toast = "已连接「" + it.Name + "」"
	} else {
		a.toast = "连接失败「" + it.Name + "」:命令为空或服务器已停用"
	}
}

// cuReconnect 供演示;真实模式走 cuReconnectReal。

// ───────── Skills ─────────

func (a *App) cuSkill(c *ui.Context, it *customize.Item) func() {
	t := a.tk
	st := a.cuStore()
	d := &a.cu.draft
	a.cuHead(c, st, it, "spark", it.Name, a.cuHeadDel(c, st, it, "删除"))
	a.cuLbl(c, "名称(重命名目录)", func() { a.cuField(c, "sk-n", &d.Name, false, "skill-name") })
	a.cuKV(c, [][2]string{{"范围", it.Scope.Label()}, {"位置", it.Path}})
	a.cuLbl(c, "文件", func() {
		a.cuCard(c, func() {
			for _, f := range it.Files {
				r := ui.Row(c).Height(t.Scheme.RowH).PaddingStart(8 + float32(f.Indent)*14).PaddingEnd(8).Gap(6).AlignItems(ui.Center).Radius(t.RSm())
				if f.Active {
					r.Background(t.Fill2)
				}
				r.Children(func() {
					icon := "fileText"
					if f.Dir {
						icon = "folder"
					}
					kit.Ic(c, icon, kit.S12)
					kit.Mono(c, f.Name, t.Tx1).Grow(1).SingleLine()
				})
			}
		})
	})
	a.cuLbl(c, "SKILL.md 预览", func() {
		desc := it.Desc
		if i := strings.Index(desc, " · "); i >= 0 {
			desc = desc[i+len(" · "):]
		}
		ui.Column(c).Padding(8, 10).Background(t.Fill1).Radius(t.RSm()).Gap(2).Children(func() {
			kit.Mono(c, "---", t.Tx4)
			ui.RichText(c, ui.Span{Text: "name", Color: t.Vio}, ui.Span{Text: ": " + it.Name}).Font("monospace").FontSize(11.5)
			ui.RichText(c, ui.Span{Text: "description", Color: t.Vio}, ui.Span{Text: ": " + desc}).Font("monospace").FontSize(11.5)
			kit.Mono(c, "---", t.Tx4)
			kit.Mono(c, "# 何时使用", t.Tx1).Margin(6, 0, 0, 0)
			kit.Mono(c, "当用户提到下面这些场景时,先读取本文件再动手。", t.Tx2)
		})
	})
	a.cuLbl(c, "会被这些说法触发", func() {
		ui.Row(c).Gap(6).Wrap().Children(func() {
			for _, tr := range it.Trig {
				kit.Chip(c, "“"+tr+"”", kit.ChipOpt{Line: true})
			}
		})
	})
	return func() {
		kit.Btn(c, "复制路径", kit.BtnOpt{Variant: kit.BtnGhost, Icon: "copy"}).OnClick(func() { a.toast = "已复制路径:" + it.Path })
		ui.Spacer(c).Grow(1)
		kit.Btn(c, "在编辑器打开", kit.BtnOpt{Variant: kit.BtnSoft, Icon: "code"}).OnClick(func() { a.cuOpenPath(it.Path) })
		kit.Btn(c, "保存", kit.BtnOpt{Variant: kit.BtnPri}).OnClick(func() { a.cuSaveSkillRename(st, it) })
	}
}

// ───────── 斜杠命令 ─────────

func (a *App) cuModelSeg(c *ui.Context, key string, m *int, short bool) {
	items := make([]kit.SegItem, len(customize.Models))
	for i, n := range customize.Models {
		if short {
			n = strings.TrimSuffix(strings.ReplaceAll(n, "继承会话", "继承"), " 5.5")
		} else {
			n = strings.ReplaceAll(n, "继承会话", "继承")
		}
		items[i].Text = n
	}
	if i := kit.Seg(c, items, *m, kit.SegOpt{Key: key}); i >= 0 {
		*m = i
	}
}

func (a *App) cuCommand(c *ui.Context, st *customize.Store, it *customize.Item) func() {
	t := a.tk
	d := &a.cu.draft
	a.cuHead(c, st, it, "slash", it.Name, a.cuHeadDel(c, st, it, "删除"))
	ui.Column(c).Gap(8).Children(func() {
		a.cuLbl(c, "名称", func() { a.cuField(c, "cc-n", &d.Name, true, "/command") })
		a.cuLbl(c, "说明", func() { a.cuField(c, "cc-d", &d.Note, false, "这个命令做什么") })
		a.cuLbl(c, "参数提示", func() { a.cuField(c, "cc-h", &d.Hint, true, "无参数") })
	})
	a.cuLbl(c, "允许的工具", func() {
		ui.Row(c).Gap(6).Wrap().Children(func() {
			for _, tl := range d.Allow {
				kit.Chip(c, tl, kit.ChipOpt{On: true}).Key("cc-t-" + tl).OnClick(func() { d.ToggleAllow(tl) })
			}
			kit.Chip(c, "+ 添加", kit.ChipOpt{Line: true}).OnClick(func() { a.toast = "工具选择器(同权限规则语法):尚未接入" })
		})
	})
	a.cuLbl(c, "模型", func() { a.cuModelSeg(c, "cc-model", &d.Model, false) })
	a.cuLbl(c, "提示词($ARGUMENTS 会被命令后面的文字替换)", func() { a.cuArea(c, "cc-b", &d.Body, 110, true) })
	if a.cu.runOut != "" {
		ui.Column(c).Padding(8, 10).Background(t.Fill1).Radius(t.RSm()).Children(func() { kit.Mono(c, a.cu.runOut, t.Tx2) })
	}
	arg, argv := "(空)", ""
	if d.Hint != "" {
		arg, argv = "123", " 123"
	}
	ui.RichText(c, ui.Span{Text: "输入 "}, ui.Span{Text: d.Name + argv, Color: t.Ac}, ui.Span{Text: " 时,$ARGUMENTS = " + arg}).FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(1.5)
	a.cuErr(c)
	return func() {
		ui.Spacer(c).Grow(1)
		kit.Btn(c, "试运行", kit.BtnOpt{Icon: "play"}).OnClick(func() { a.cuTryCommand(it) })
		kit.Btn(c, "填进输入框", kit.BtnOpt{Icon: "send"}).OnClick(func() {
			d := a.cuDraftItem(it)
			a.draft = d.Name + argvOfDraft(&d)
			a.screen = ScreenWork
			a.toast = "已把命令填进输入框"
		})
		kit.Btn(c, "保存", kit.BtnOpt{Variant: kit.BtnPri}).OnClick(func() { a.cuSave(st, it) })
	}
}

func argvOf(it *customize.Item) string {
	return argvOfDraft(it)
}

func argvOfDraft(it *customize.Item) string {
	if it.Hint != "" {
		return " "
	}
	return ""
}

// ───────── 子 Agent ─────────

func (a *App) cuAgent(c *ui.Context, st *customize.Store, it *customize.Item) func() {
	t := a.tk
	d := &a.cu.draft
	colors := []ui.Color{t.Info, t.Ok, t.Vio, t.Cyan, t.Pink, t.Warn}
	a.cuHead(c, st, it, "bot", it.Name, func() {
		ui.Box(c).Size(7, 7).Radius(4).Background(colors[d.Color%len(colors)]).Shrink(0)
		a.cuHeadDel(c, st, it, "删除")()
	})
	ui.Column(c).Gap(8).Children(func() {
		a.cuLbl(c, "名称", func() { a.cuField(c, "ca-n", &d.Name, true, "my-agent") })
		a.cuLbl(c, "什么时候委派给它", func() { a.cuField(c, "ca-d", &d.Note, false, "例如:审查安全问题") })
	})
	ui.Row(c).Gap(14).AlignItems(ui.Start).Wrap().Children(func() {
		a.cuLbl(c, "模型", func() { a.cuModelSeg(c, "ca-model", &d.Model, true) })
		a.cuLbl(c, "颜色", func() {
			ui.Row(c).Gap(6).Height(20).AlignItems(ui.Center).Children(func() {
				for i, col := range colors {
					sw := ui.ButtonBase(c).Key("ca-c-"+itoa(i)).Size(14, 14).Radius(7).Background(col).Cursor(ui.CursorPointer)
					if i == d.Color {
						sw.Border(2, t.Tx1)
					}
					sw.OnClick(func() { d.Color = i })
				}
			})
		})
	})
	a.cuLbl(c, "工具权限", func() {
		if i := kit.Seg(c, []kit.SegItem{{Text: "只读"}, {Text: "可编辑"}, {Text: "全部"}}, d.Preset(), kit.SegOpt{Key: "ca-preset"}); i >= 0 {
			d.SetPreset(i)
		}
		ui.Row(c).Gap(6).Wrap().Margin(4, 0, 0, 0).Children(func() {
			for _, tl := range customize.AllTools {
				on := false
				for _, x := range d.Allow {
					on = on || x == tl
				}
				kit.Chip(c, tl, kit.ChipOpt{Line: !on, On: on}).Key("ca-t-" + tl).OnClick(func() { d.ToggleAllow(tl) })
			}
		})
	})
	a.cuLbl(c, "系统提示词", func() { a.cuArea(c, "ca-p", &d.Body, 110, false) })
	if a.cu.runOut != "" {
		ui.Column(c).Padding(8, 10).Background(t.Fill1).Radius(t.RSm()).Children(func() { kit.Mono(c, a.cu.runOut, t.Tx2) })
	}
	a.cuErr(c)
	return func() {
		ui.Spacer(c).Grow(1)
		kit.Btn(c, "试运行", kit.BtnOpt{Icon: "play"}).OnClick(func() { a.cuTryAgent(it) })
		kit.Btn(c, "保存", kit.BtnOpt{Variant: kit.BtnPri}).OnClick(func() { a.cuSave(st, it) })
	}
}

// ───────── 规则与记忆 ─────────

func (a *App) cuRule(c *ui.Context, st *customize.Store, it *customize.Item) func() {
	t := a.tk
	d := &a.cu.draft
	a.cuHead(c, st, it, "fileText", it.Name, a.cuHeadDel(c, st, it, "删除"))
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		on := 0
		if a.cu.rulePV {
			on = 1
		}
		if i := kit.Seg(c, []kit.SegItem{{Text: "编辑"}, {Text: "预览"}}, on, kit.SegOpt{Key: "cr-mode"}); i >= 0 {
			a.cu.rulePV = i == 1
		}
		ui.Spacer(c).Grow(1)
		ui.Text(c, it.Size+" · 约 "+itoa(it.Tokens)+" token · 占上下文 "+pct1(it.RuleTokenPct())+"%").FontSize(theme.FsCap).TextColor(t.Tx3)
	})
	if a.cu.rulePV {
		ui.Column(c).Padding(8, 10).Background(t.Fill1).Radius(t.RSm()).Gap(3).Children(func() {
			for _, l := range strings.Split(d.Body, "\n") {
				switch {
				case strings.HasPrefix(l, "# "):
					ui.Text(c, l[2:]).FontSize(theme.FsTitle).FontWeight(700).Margin(2, 0, 4, 0)
				case strings.HasPrefix(l, "- "):
					ui.Row(c).Gap(7).AlignItems(ui.Start).Children(func() {
						ui.Box(c).Size(5, 5).Radius(3).Background(t.Ac).Margin(7, 0, 0, 0).Shrink(0)
						ui.Text(c, l[2:]).Grow(1).MinWidth(0).LineHeight(1.5)
					})
				case strings.TrimSpace(l) == "":
					ui.Box(c).Height(6)
				default:
					ui.Text(c, l).LineHeight(1.5)
				}
			}
		})
	} else {
		a.cuArea(c, "cr-b", &d.Body, 150, true)
	}
	a.cuLbl(c, "加载顺序(后加载的优先)", func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			for i, sc := range cuScopes[1:] {
				kit.Chip(c, sc.Label(), kit.ChipOpt{Line: sc != it.Scope, On: sc == it.Scope}).Key("cr-o-" + itoa(i))
				if i < 2 {
					kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx4)
				}
			}
		})
	})
	a.cuCard(c, func() {
		a.cuToggleRow(c, "cr-load", "加载 CLAUDE.md 记忆文件", "关闭后新会话不读任何记忆文件(CLAUDE_CODE_DISABLE_CLAUDE_MDS=1)", !a.cuNoMem, func() { a.cuNoMem = !a.cuNoMem })
	})
	a.cuErr(c)
	return func() {
		ui.Spacer(c).Grow(1)
		kit.Btn(c, "在编辑器打开", kit.BtnOpt{Icon: "code"}).OnClick(func() { a.cuOpenPath(it.Path) })
		kit.Btn(c, "保存", kit.BtnOpt{Variant: kit.BtnPri}).OnClick(func() { a.cuSave(st, it) })
	}
}

func pct1(f float64) string {
	n := int(f*10 + .5)
	return itoa(n/10) + "." + itoa(n%10)
}

// ───────── 钩子 ─────────

func (a *App) cuHook(c *ui.Context, st *customize.Store, it *customize.Item) func() {
	t := a.tk
	d := &a.cu.draft
	a.cuHead(c, st, it, "bolt", it.Name, a.cuHeadDel(c, st, it, "删除"))
	if it.Scope == customize.Project && !it.Trusted {
		a.cuBanner(c, kit.KWarn, "shield", "这个项目的钩子还没有被信任,不会运行。", func() {
			kit.Btn(c, "信任", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).OnClick(func() {
				it.Trusted, d.Trusted = true, true
				a.toast = "已信任「" + it.Name + "」"
			})
		})
	}
	a.cuLbl(c, "事件", func() {
		ui.Row(c).Gap(6).Wrap().Children(func() {
			for i, e := range customize.Events {
				kit.Chip(c, e, kit.ChipOpt{Line: i != d.Event, On: i == d.Event}).Key("ch-e-" + itoa(i)).OnClick(func() { d.Event = i })
			}
		})
	})
	ui.Column(c).Gap(8).Children(func() {
		a.cuLbl(c, "名称", func() { a.cuField(c, "ch-n", &d.Name, false, "PostToolUse · Edit") })
		a.cuLbl(c, "匹配(工具名,可用 | 分隔)", func() { a.cuField(c, "ch-m", &d.Match, true, "留空 = 全部") })
		a.cuLbl(c, "命令", func() { a.cuArea(c, "ch-c", &d.Cmd, 52, true) })
	})
	ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
		a.cuLbl(c, "超时", func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				kit.Ib(c, "minus", kit.IbOpt{Small: true, Tip: "减 1 秒"}).OnClick(func() { d.Timeout = max(1, d.Timeout-1) })
				ui.Text(c, itoa(d.Timeout)).FontWeight(500).MinWidth(22).TextAlign(ui.Center)
				ui.Text(c, "秒").TextColor(t.Tx3)
				kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "加 1 秒"}).OnClick(func() { d.Timeout = min(120, d.Timeout+1) })
			})
		})
		a.cuLbl(c, "范围", func() {
			on := 0
			for i, sc := range cuScopes[1:] {
				if sc == d.Scope {
					on = i
				}
			}
			if i := kit.Seg(c, []kit.SegItem{{Text: "用户"}, {Text: "项目"}, {Text: "本地"}}, on, kit.SegOpt{Key: "ch-scope"}); i >= 0 {
				d.Scope = cuScopes[1:][i]
			}
		})
	})
	if a.cu.hookOut != "" {
		ui.Column(c).Padding(8, 10).Background(t.Fill1).Radius(t.RSm()).Children(func() { kit.Mono(c, a.cu.hookOut, t.Tx2) })
	}
	a.cuErr(c)
	return func() {
		ui.Spacer(c).Grow(1)
		kit.Btn(c, "试运行", kit.BtnOpt{Icon: "play"}).OnClick(func() { a.cuTryHook(d) })
		kit.Btn(c, "保存", kit.BtnOpt{Variant: kit.BtnPri}).OnClick(func() { a.cuSave(st, it) })
	}
}
