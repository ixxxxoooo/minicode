package app

import (
	"os"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 自定义页右侧抽屉:各类的专属详情。命令 / 子 Agent / 规则 / 钩子是表单,改的是草稿 `cu.draft`,「保存」才写回;
// 插件 / MCP / Skills 的动作(启停、工具开关、重连、授权)立即生效。

// 自定义页详情宽度(设计稿 .cu .drawer):原来 300 / 360 太紧,表单与列表都挤。
const (
	cuDrawerW     = 380 // 详情宽度下限(普通类)
	cuDrawerWideW = 460 // 详情宽度下限(表单类:命令 / 子 Agent / 规则 / 钩子)
	cuDrawerMaxW  = 560 // 详情宽度上限(普通类)
	cuDrawerWideM = 680 // 详情宽度上限(表单类)
	cuListMinW    = 340 // 左侧列表最小宽度:再窄行内的名称 / 标签就挤不下了
)

// cuDrawerWidth 返回右侧详情的宽度:按页面主区宽度的 52%(表单类 58%)取,夹在上下限之间,
// 并给左侧列表至少留 cuListMinW。原来是固定 380 / 460,窗口一宽左边列表被撑得过宽、右边详情显得窄。
// 参数 wide:是否表单类;返回值:像素宽度。
func (a *App) cuDrawerWidth(wide bool) float32 {
	lo, hi, frac := float32(cuDrawerW), float32(cuDrawerMaxW), float32(0.52)
	if wide {
		lo, hi, frac = cuDrawerWideW, cuDrawerWideM, 0.58
	}
	main := a.winW - a.sideOcc() // 主区宽度:窗口减去导航栏
	if main <= 0 {
		return lo
	}
	return max(min(main*frac, hi, main-cuListMinW), min(lo, main-cuListMinW))
}

func (a *App) cuDrawer(c *ui.Context, st *customize.Store) {
	t := a.tk
	s := &a.cu
	cat := a.cuCat()
	w := a.cuDrawerWidth(cat == customize.Commands || cat == customize.Agents || cat == customize.Rules || cat == customize.Hooks)
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
		kit.VScroll(c).Key("cu-dscroll-" + string(cat) + "-" + it.Name).Grow(1).MinHeight(0).Clip().Children(func() {
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
		ui.Text(c, name).FontWeight(500).Grow(1).MinWidth(0).SingleLine()
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
	// 文字占满一行,按钮另起一行靠右:右侧抽屉很窄,按钮和文字挤在同一行会把文字挤成一列窄条
	ui.Column(c).Padding(8, 10).Gap(8).Background(bg).Radius(t.RMd()).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Start).Children(func() {
			kit.Ic(c, icon, kit.S12).TextColor(fg).Margin(3, 0, 0, 0).Shrink(0)
			ui.Text(c, text).Grow(1).MinWidth(0).LineHeight(1.5)
		})
		if acts != nil {
			ui.Row(c).Gap(8).Justify(ui.End).AlignItems(ui.Center).Children(acts)
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
					v.Font("monospace").FontSize(theme.Px(11.5))
				}
			})
		}
	})
}

func (a *App) cuField(c *ui.Context, key string, v *string, mono bool, ph string) {
	in := ui.TextInput(c.Key(key), v).Height(theme.Px(26)).FontSize(theme.FsBody).Placeholder(ph).PlaceholderColor(a.tk.Tx4)
	if mono {
		in.Font("monospace").FontSize(theme.Px(11.5))
	}
}

func (a *App) cuArea(c *ui.Context, key string, v *string, h float32, mono bool) {
	in := ui.TextArea(c.Key(key), v).Height(h).FontSize(theme.FsBody)
	if mono {
		in.Font("monospace").FontSize(theme.Px(11.5))
	}
}

// cuCard 是带描边的小卡片(行之间不画线);复用 kit.Card 的悬停上浮与阴影。
func (a *App) cuCard(c *ui.Context, body func()) {
	kit.Card(c).Cursor(ui.CursorDefault).Padding(2, 4).Column().Children(body)
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
	if cat == customize.Plugins && !a.cuDemo() {
		a.cuUninstallPlugin(it) // 真实卸载要跑 claude 子进程,放后台,完成后再刷新列表
		return
	}
	if a.cuDisk() != nil && cat != customize.Plugins {
		a.cuRemoveDisk(st, it)
	} else if cat == customize.Plugins {
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
	kit.VScroll(c).Key("cu-mkscroll").Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(14, 16).Gap(12).Children(func() {
			a.cuHead(c, st, nil, "plug", m.Name, func() { kit.Tag(c, m.By, kit.KAc) })
			desc := m.Desc
			if desc == "" {
				desc = "这个插件没有提供说明。"
			}
			ui.Text(c, desc).TextColor(t.Tx2).LineHeight(1.6)
			src, ver := m.MP, m.Ver
			if src == "" {
				src = "演示数据"
			}
			if ver == "" {
				ver = "以市场最新为准"
			}
			a.cuKV(c, [][2]string{{"作者", m.By}, {"来源市场", src}, {"版本", ver}, {"安装到", "用户(所有项目)"}})
			// 包含的技能 / 命令 / 子 Agent 要装完才知道,不再编数字糊弄
			ui.Text(c, "安装后会把它包含的 Skills、命令、子 Agent、钩子与 MCP 一并加入,可在「已安装」里查看具体数量并逐项启停。").FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(1.5)
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
	// 安装层是装的时候定的(claude plugin install --scope),装完不能在这里改,所以只读展示
	a.cuKV(c, [][2]string{{"来源市场", it.Src}, {"版本", it.Ver}, {"安装到", it.Scope.Label() + "层"}})
	a.cuLbl(c, "包含内容", func() {
		a.cuCard(c, func() {
			if len(it.Inc) == 0 {
				ui.Text(c, "没有读到包含的内容(插件目录里没有 skills / commands / agents / hooks / .mcp.json)").FontSize(theme.FsCap).TextColor(t.Tx3).Padding(8)
			}
			for i := range it.Inc {
				inc := &it.Inc[i]
				// 插件内部各项不能单独启停(claude 只认整个插件),所以这里只展示数量,不放假开关
				ui.Row(c).Height(t.Scheme.RowH).PaddingX(8).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, inc.Kind).Grow(1)
					ui.Text(c, itoa(inc.N)).TextColor(t.Tx3)
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
		kit.Btn(c, label).OnClick(func() { a.cuPluginToggle(it) })
	}
}

// ───────── MCP ─────────

func (a *App) cuMCP(c *ui.Context, st *customize.Store, it *customize.Item) func() {
	t := a.tk
	a.cuHead(c, st, it, "server", it.Name, a.cuHeadDel(c, st, it, "删除"))
	status := it.Status
	// 状态以运行时会话报告的为准(列表就是读它):`claude mcp list` 是另起的子进程,环境 / 登录态与会话里的不一样,
	// 可能报失败而会话里其实已连上(列表绿点「已连接 · 83 个工具」、详情红条「连接失败」就是这样对不上)
	if rs, ok := a.runtimeMcp(it.Name); ok {
		switch rs.Conn {
		case core.McpConnected:
			status = customize.StOK
		case core.McpNeedsAuth:
			status = customize.StWarn
		case core.McpFailed:
			status = customize.StErr
			if rs.Error != "" {
				it.Desc = "连接失败 · " + customize.ShortMCPErr(rs.Error)
			}
		case core.McpPending:
			status = customize.StNone
		}
		if len(it.Tools) == 0 && len(rs.Tools) > 0 { // 运行时才有工具名:补上,详情里才能列出与开关
			for _, n := range rs.Tools {
				it.Tools = append(it.Tools, customize.Tool{Name: n, On: true})
			}
		}
	}
	if !it.On { // 开关关着就是停用,不管上次探活是什么结果
		status = customize.StOff
	}
	switch status {
	case customize.StOK:
		a.cuBanner(c, kit.KOk, "check", "已连接 · "+itoa(len(it.Tools))+" 个工具", func() {
			kit.Btn(c, "重新连接", kit.BtnOpt{Small: true, Icon: "refresh"}).OnClick(func() { a.cuReconnectReal(it) })
		})
	case customize.StWarn:
		a.cuBanner(c, kit.KWarn, "alert", "需要授权后才能使用", func() {
			kit.Btn(c, "授权", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).OnClick(func() { a.cuAuth(it) })
		})
	case customize.StNone:
		a.cuBanner(c, kit.KInfo, "alert", "还没有查询连接状态", func() {
			kit.Btn(c, "查询", kit.BtnOpt{Small: true, Icon: "refresh", Loading: a.mcpRefreshing}).OnClick(a.cuRefreshMCP)
		})
	case customize.StOff:
		a.cuBanner(c, kit.KInfo, "alert", "已在本项目停用,打开列表右侧的开关即可恢复", func() {
			kit.Btn(c, "启用", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).OnClick(func() {
				if !a.cuDemo() {
					a.cuMCPOn(it, true)
				} else {
					it.On, it.Status = true, customize.StNone
				}
			})
		})
	default:
		reason := strings.TrimPrefix(it.Desc, "连接失败 · ")
		a.cuBanner(c, kit.KErr, "alert", "连接失败:"+reason+"。"+mcpFailHint(it, reason), func() {
			if it.Transport == "http" || it.Transport == "sse" {
				kit.Btn(c, "诊断", kit.BtnOpt{Small: true, Icon: "search"}).OnClick(func() { a.cuMCPProbe(it) })
			}
			kit.Btn(c, "编辑配置", kit.BtnOpt{Small: true, Icon: "edit"}).OnClick(func() { a.cuMCPStartEdit(it) })
			kit.Btn(c, "重试", kit.BtnOpt{Small: true, Icon: "refresh"}).OnClick(func() { a.cuReconnectReal(it) })
		})
		if msg, ok := a.mcpProbe[it.Name]; ok {
			if msg == "" {
				msg = "诊断中…"
			}
			ui.Text(c, "服务器回复:"+msg).FontSize(theme.FsCap).TextColor(t.Tx2).LineHeight(1.5)
		}
	}
	where := "命令"
	if it.Transport == "http" {
		where = "地址"
	}
	rows := [][2]string{{"传输", it.Transport}, {where, it.Cmd}}
	if it.Path != "" {
		rows = append(rows, [2]string{"配置文件", it.Path}) // 这台服务器写在哪(「打开 JSON」打开的就是它)
	}
	a.cuKV(c, rows)
	a.cuMCPEnvRefs(c, it)
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
	a.cuMCPTools(c, it)
	a.cuCard(c, func() {
		a.cuToggleRow(c, "cm-ondemand-"+it.Name, "按需加载工具(全部 MCP)", "ENABLE_TOOL_SEARCH · 工具很多时只在需要时加载描述,节省上下文;对所有 MCP 服务器生效,写入 Claude 设置,新会话起效;使用第三方服务商时会被强制关闭", it.OnDemand, func() { a.cuMCPOnDemand(!it.OnDemand) })
	})
	return func() {
		ui.Spacer(c).Grow(1)
		kit.Btn(c, "打开 JSON", kit.BtnOpt{Icon: "code", Variant: kit.BtnGhost}).OnClick(func() { a.cuOpenPath(it.Path) })
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
	a.cuHead(c, st, it, "wand", it.Name, a.cuHeadDel(c, st, it, "删除"))
	a.cuLbl(c, "名称(重命名目录)", func() { a.cuField(c, "sk-n", &d.Name, false, "skill-name") })
	a.cuKV(c, [][2]string{{"范围", it.Scope.Label()}, {"位置", it.Path}})
	a.cuLbl(c, "文件", func() {
		a.cuCard(c, func() { a.cuSkillFileList(c, it) })
	})
	a.cuLbl(c, "文件预览", func() {
		ui.Column(c).Padding(0, 0, 4, 0).Background(t.Fill1).Radius(t.RSm()).Children(func() {
			a.cuSkillPreview(c)
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
		kit.Btn(c, "复制路径", kit.BtnOpt{Variant: kit.BtnGhost, Icon: "copy"}).OnClick(func() { a.copyOut(a.cuSkillFilePath(it), "路径"); a.toast = "已复制路径" })
		ui.Spacer(c).Grow(1)
		kit.Btn(c, "在编辑器打开", kit.BtnOpt{Variant: kit.BtnSoft, Icon: "code"}).OnClick(func() { a.cuOpenPath(a.cuSkillFilePath(it)) })
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
				kit.Chip(c, tl, kit.ChipOpt{On: true, Tip: "点一下移除"}).Key("cc-t-" + tl).OnClick(func() { d.ToggleAllow(tl) })
			}
			if len(d.Allow) == 0 {
				ui.Text(c, "未限定:沿用会话的权限设置").FontSize(theme.FsCap).TextColor(a.tk.Tx4)
			}
			pick := kit.Chip(c, "+ 添加", kit.ChipOpt{Line: true, On: a.popup == toolPickKey}).OnClick(func() {
				a.cu.toolQ, a.cu.toolErr = "", ""
				a.togglePop(toolPickKey)
			})
			a.cuToolPicker(c, pick, d, st)
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
			ui.Row(c).Gap(6).Height(theme.Px(20)).AlignItems(ui.Center).Children(func() {
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
		// 内置工具之外已授予的(MCP 工具 / 带参数的规则):单独列出,点一下移除;「+ MCP 工具」打开与命令里同一个选择器
		ui.Row(c).Gap(6).Wrap().Margin(4, 0, 0, 0).Children(func() {
			for _, r := range d.Allow {
				if isBuiltinTool(r) {
					continue
				}
				kit.Chip(c, r, kit.ChipOpt{On: true, Tip: "点一下移除"}).Key("ca-x-" + r).OnClick(func() { d.ToggleAllow(r) })
			}
			pick := kit.Chip(c, "+ MCP 工具", kit.ChipOpt{Line: true, On: a.popup == toolPickKey}).OnClick(func() {
				a.cu.toolQ, a.cu.toolErr = "", ""
				a.togglePop(toolPickKey)
			})
			a.cuToolPicker(c, pick, d, st)
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
		on := 1 // 默认预览;点「编辑」才进入编辑
		if a.cu.ruleEdit {
			on = 0
		}
		if i := kit.Seg(c, []kit.SegItem{{Text: "编辑"}, {Text: "预览"}}, on, kit.SegOpt{Key: "cr-mode"}); i >= 0 {
			a.cu.ruleEdit = i == 0
		}
		ui.Spacer(c).Grow(1)
		ui.Text(c, it.Size+" · 约 "+itoa(it.Tokens)+" token · 占上下文 "+pct1(it.RuleTokenPct())+"%").FontSize(theme.FsCap).TextColor(t.Tx3)
	})
	if !a.cu.ruleEdit {
		// 预览按正文同样的 Markdown 渲染(标题 / 列表 / 代码块 / 表格 / 链接),不再是手写的逐行解析
		ui.Column(c).Padding(10, 12).Background(t.Fill1).Radius(t.RSm()).Children(func() {
			markdown.View(c, padCode(d.Body), a.mdOptions())
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

// mcpFailHint 按失败原因给下一步建议。
func mcpFailHint(it *customize.Item, reason string) string {
	if it.Transport == "http" || it.Transport == "sse" {
		if strings.Contains(reason, "UNEXPECTED_CONTENT_TYPE") || strings.Contains(reason, "Unexpected content type") {
			return "服务器回的不是 MCP 数据(常见是登录失效时回了一段网页 / 错误 JSON)。点「诊断」看服务器原话,多半要更新请求头里的 Cookie / Authorization。"
		}
		if strings.Contains(reason, "Unrecognized token '<'") || strings.Contains(reason, "<!DOCTYPE") {
			return "服务器返回的是网页(多半是登录页)而不是 MCP 数据:检查地址是否正确、请求头里的 Cookie / Authorization 是否过期、${环境变量} 是否有值、是否需要连内网。点「编辑配置」修改。"
		}
		return "检查地址与请求头后重试,点「编辑配置」修改。"
	}
	return "检查命令与环境变量后重试,点「编辑配置」修改。"
}

// cuMCPEnvRefs 列出这个 MCP 配置里引用的 ${环境变量},并说明本机能不能取到(只显示来源,不显示值)。
// 访达启动的应用拿不到 ~/.zshrc 里 export 的变量;启动 claude 时会自动从登录 shell 补上(claude.WithMCPEnv),这里让人看得见。
func (a *App) cuMCPEnvRefs(c *ui.Context, it *customize.Item) {
	var parts []string
	parts = append(parts, it.Cmd)
	for _, v := range it.Headers {
		parts = append(parts, v)
	}
	for _, v := range it.EnvVals {
		parts = append(parts, v)
	}
	refs := claude.EnvRefNames(strings.Join(parts, "\n"))
	if len(refs) == 0 || a.cuDemo() {
		return
	}
	if a.mcpEnvSrc == nil {
		a.mcpEnvSrc = map[string]string{}
	}
	for _, n := range refs {
		if _, ok := a.mcpEnvSrc[n]; ok {
			continue
		}
		a.mcpEnvSrc[n] = "" // 占位:检测中,别重复起任务
		n := n
		a.bg("检测环境变量", func() func() {
			src := claude.EnvRefSource(n)
			if src == "" {
				src = "none"
			}
			return func() { a.mcpEnvSrc[n] = src }
		})
	}
	t := a.tk
	a.cuLbl(c, "引用的环境变量(只显示能否取到,不显示值)", func() {
		a.cuCard(c, func() {
			for _, n := range refs {
				ui.Row(c).MinHeight(t.Scheme.RowH).PaddingX(8).Gap(8).AlignItems(ui.Center).Children(func() {
					kit.Mono(c, n, t.Tx1).Shrink(0)
					msg, col := "检测中…", t.Tx4
					switch a.mcpEnvSrc[n] {
					case "app":
						msg, col = "✓ 应用环境里有", t.Ok
					case "shell":
						msg, col = "✓ 登录 shell 里有,启动 Claude 时自动带上", t.Ok
					case "none":
						msg, col = "✗ 没找到:在 ~/.zshrc 里 export 后点「重新检测」,或在编辑配置里直接填值", t.Err
					}
					ui.Text(c, msg).FontSize(theme.FsCap).TextColor(col).Grow(1).MinWidth(0)
				})
			}
		})
		kit.Btn(c, "重新检测", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "refresh"}).OnClick(func() {
			claude.ForgetShellEnv()
			a.mcpEnvSrc = nil
		})
	})
}

// runtimeMcp 取运行时会话(mcp_status)里这个服务器的状态;会话没起或没报告返回 false。
func (a *App) runtimeMcp(name string) (core.McpServerStatus, bool) {
	if a.st == nil {
		return core.McpServerStatus{}, false
	}
	for _, s := range a.st.Mcp {
		if s.Name == name {
			return s, true
		}
	}
	return core.McpServerStatus{}, false
}

// cuMCPProbe 对这个 http MCP 服务器发一次 initialize,把服务器的原话显示在失败提示下面(后台跑,网络请求不能卡界面)。
func (a *App) cuMCPProbe(it *customize.Item) {
	if a.mcpProbe == nil {
		a.mcpProbe = map[string]string{}
	}
	name, url, hdr := it.Name, it.Cmd, it.Headers
	a.mcpProbe[name] = ""
	a.bg("MCP诊断", func() func() {
		res := customize.ProbeHTTPMCP(url, hdr, func(n string) string {
			if v := os.Getenv(n); v != "" {
				return v
			}
			v, _ := claude.ShellEnv(n)
			return v
		})
		return func() { a.mcpProbe[name] = res }
	})
}

// isBuiltinTool 是不是内置工具(AllTools 里的那几个)。
func isBuiltinTool(name string) bool {
	for _, x := range customize.AllTools {
		if x == name {
			return true
		}
	}
	return false
}

// cuMCPTools 是 MCP 详情里的工具区(对标 Cursor 的 Tools):列出检测到的全部工具,逐个开关,右上角「批量」一键全部启用 / 禁用。
// 工具名取运行中会话汇报的(连上了就有),没有会话时取 claude mcp list 读到的;开关写本地层 permissions.deny,
// 与输入框 MCP 弹层同一套数据(a.mcpDeny)。
func (a *App) cuMCPTools(c *ui.Context, it *customize.Item) {
	t := a.tk
	var names []string
	if rs, ok := a.runtimeMcp(it.Name); ok && len(rs.Tools) > 0 {
		names = append(names, rs.Tools...)
	} else {
		for _, tl := range it.Tools {
			names = append(names, tl.Name)
		}
	}
	netErr, netBusy := "", false
	if len(names) == 0 && (it.Transport == "http" || it.Transport == "sse") && it.On && !a.cuDemo() {
		// 没有会话时 claude mcp list 不给工具名:http 服务器直接问它(initialize + tools/list),后台跑,结果缓存
		if got, ok := a.mcpToolsNet[it.Name]; ok {
			names = got
		}
		if e, ok := a.mcpToolsNetErr[it.Name]; ok {
			netErr, netBusy = e, e == ""
		}
		if _, tried := a.mcpToolsNet[it.Name]; !tried {
			if _, running := a.mcpToolsNetErr[it.Name]; !running {
				a.cuFetchMCPToolsNet(it)
				netBusy = true
			}
		}
	}
	if a.cuDenyFor != it.Name { // 换了一台:后台重读一次禁用规则
		a.cuDenyFor = it.Name
		a.mcpDenyReload()
	}
	off := 0
	for _, n := range names {
		if !a.mcpToolEnabled(it.Name, n) {
			off++
		}
	}
	ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
		title := "工具 (" + itoa(len(names)) + ")"
		ui.Text(c, title).FontWeight(600)
		if off > 0 {
			ui.Text(c, "已禁用 "+itoa(off)).FontSize(theme.FsCap).TextColor(t.Warn)
		}
		ui.Spacer(c).Grow(1)
		if len(names) > 0 {
			kit.Btn(c, "批量", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "chevD"}).Menu(func(m *ui.Menu) {
				if menuItem(m, "全部启用").Chosen() {
					a.setMcpToolsBulk(it.Name, names, true)
				}
				if menuItem(m, "全部禁用").Chosen() {
					a.setMcpToolsBulk(it.Name, names, false)
				}
			})
		}
	})
	ui.Text(c, "关掉的工具 Claude 不会调用 · 写入本地层 permissions.deny").FontSize(theme.FsCap).TextColor(t.Tx3)
	if len(names) == 0 {
		msg := "连接成功后才会列出工具"
		switch {
		case netBusy:
			msg = "正在读取工具…"
		case netErr != "":
			msg = "无法读取工具:" + netErr
		case it.Status == customize.StOK:
			msg = "没有工具"
		}
		ui.Text(c, msg).FontSize(theme.FsCap).TextColor(t.Tx4).LineHeight(1.5)
		if !a.mcpRuntimeReady() { // 没有运行中的会话(stdio 服务器只能这样查,http 直连失败时也可以):启动会话后由会话汇报
			kit.Btn(c, "启动会话检测工具", kit.BtnOpt{Small: true, Variant: kit.BtnSoft, Icon: "refresh", Loading: a.mcpRefreshing}).
				Tooltip("启动 Claude 连接这些 MCP 服务器并读取它们的工具").OnClick(func() { a.mcpManualRefresh(true) })
		}
		return
	}
	a.cuCard(c, func() {
		for _, n := range names {
			n := n
			on := a.mcpToolEnabled(it.Name, n)
			ui.Row(c).Key("cmt-row-" + it.Name + "-" + n).MinHeight(t.Scheme.RowH).PaddingX(8).Gap(6).AlignItems(ui.Center).Children(func() {
				col := t.Tx1
				if !on {
					col = t.Tx4
				}
				kit.Mono(c, n, col).Grow(1).MinWidth(0).SingleLine()
				if kit.Tog(c, on, "cmt-"+it.Name+"-"+n) {
					a.toggleMcpTool(it.Name, n, !on)
				}
			})
		}
	})
}

// setMcpToolsBulk 一次把一台服务器的这些工具全部启用 / 禁用:先本地生效,后台只写一次文件。
func (a *App) setMcpToolsBulk(server string, tools []string, on bool) {
	if a.proj == "" {
		a.toast = "需要打开项目才能写 MCP 工具开关"
		return
	}
	if a.mcpDeny == nil {
		a.mcpDeny = map[string]bool{}
	}
	prev := map[string]bool{}
	for _, tl := range tools {
		k := customize.MCPToolKey(server, tl)
		prev[k] = a.mcpDeny[k]
		a.mcpDeny[k] = !on
	}
	proj := a.proj
	a.bg("mcp-tools-bulk", func() func() {
		var werr error
		if ds, err := customize.OpenDisk(proj); err != nil {
			werr = err
		} else {
			werr = ds.SetMCPToolsBulk(server, tools, on)
		}
		return func() {
			if werr != nil {
				for k, v := range prev {
					a.mcpDeny[k] = v
				}
				a.toast = "写入失败:" + werr.Error()
				return
			}
			verb := "禁用"
			if on {
				verb = "启用"
			}
			a.toast = "已" + verb + "「" + server + "」的全部 " + itoa(len(tools)) + " 个工具"
			log.Info("MCP 工具批量开关", "server", server, "on", on, "n", len(tools))
			a.mcpManualRefresh(a.mcpRuntimeReady())
		}
	})
}

// cuFetchMCPToolsNet 后台直连一台 http MCP 服务器读工具名;结果存 mcpToolsNet / mcpToolsNetErr。
func (a *App) cuFetchMCPToolsNet(it *customize.Item) {
	if a.mcpToolsNet == nil {
		a.mcpToolsNet, a.mcpToolsNetErr = map[string][]string{}, map[string]string{}
	}
	name, url, hdr := it.Name, it.Cmd, it.Headers
	a.mcpToolsNetErr[name] = "" // 读取中
	a.bg("直连MCP读工具", func() func() {
		names, err := customize.FetchHTTPMCPTools(url, hdr, func(n string) string {
			if v := os.Getenv(n); v != "" {
				return v
			}
			v, _ := claude.ShellEnv(n)
			return v
		})
		return func() {
			if err != nil && len(names) == 0 {
				a.mcpToolsNetErr[name] = err.Error()
				a.mcpToolsNet[name] = nil
				return
			}
			delete(a.mcpToolsNetErr, name)
			a.mcpToolsNet[name] = names
			log.Info("直连读到 MCP 工具", "server", name, "n", len(names))
		}
	})
}
