package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 自定义整页(设计稿 customize.js):左分类栏 + 列表 + 各类专属详情抽屉。
// 数据模型与校验在 internal/customize;真实模式经 customize.OpenDisk 读盘写盘,演示仍用 demo.Customize()。

type custMeta struct {
	icon, title, desc, add string
}

var custCats = map[customize.Cat]custMeta{
	customize.Plugins:  {"plug", "插件", "插件把 Skills、命令、子 Agent 与钩子打包在一起,可在市场里浏览、安装与启停。", "从市场安装"},
	customize.MCP:      {"server", "MCP 服务器", "给 Claude 接入外部工具与数据。连接状态由 Claude 实时报告;开关只对当前项目生效。", "添加服务器"},
	customize.Skills:   {"wand", "Skills", "Claude 按场景自动调用的能力包:一个目录加一份 SKILL.md。", "新建 Skill"},
	customize.Commands: {"slash", "斜杠命令", "在输入框里用 / 调用的提示词模板,支持 $ARGUMENTS 参数。", "新建命令"},
	customize.Agents:   {"bot", "子 Agent", "带独立上下文与工具权限的专用助手,由主 Agent 委派任务。", "新建子 Agent"},
	customize.Rules:    {"book", "规则与记忆", "CLAUDE.md 与 .claude/rules 里的长期约定,新会话自动加载;层级越靠后优先级越高。", "新建规则"},
	customize.Hooks:    {"bolt", "钩子", "在工具调用前后自动运行的命令。陌生项目的钩子需要先在「权限与信任」里允许。", "添加钩子"},
}

type custState struct {
	cat          customize.Cat
	scope        customize.Scope
	q            string
	market       bool // 插件:浏览市场
	detail       bool // <900px:列表 / 详情两级
	sel          *customize.Item
	mk           *customize.Market // 市场里选中的
	mkMore       int               // 市场里「显示更多」点过几次(每次多一页)
	draft        customize.Item    // 选中项的草稿副本(文本框绑它,保存才写回)
	dOf          *customize.Item   // 草稿对应哪一项
	ruleEdit     bool              // 规则:编辑模式(默认 false = 预览;切换条目 / 分类时回到预览)
	hookOut      string
	runOut       string
	err          string
	delArm       *customize.Item // 点过一次「删除」,等待确认
	dlg          *custDlg
	ask          string
	disk         *customize.DiskStore
	plug         *pluginInstall
	mcpEdit      bool
	mcpEnv       string
	mcpHdr       string // 编辑 MCP 时的请求头草稿(每行「名称: 值」)
	skillOf      string // 当前预览文件列表对应的 Skill id
	skillFile    string
	skillPrev    string
	skillPrevErr string
	skillLoad    int
	toolQ        string    // 命令工具选择器:搜索 / 规则输入
	toolErr      string    // 规则写法错误(空 = 无)
	toolErrAt    time.Time // 错误出现时刻(输入框抖动)
}

// pluginInstall 跟踪市场插件安装进度(0 进行中 / 1 收尾 / 2 成功 / 3 失败)。
type pluginInstall struct {
	name, market string
	phase        int
	err          string
}

// custDlg 是「新建」对话框的状态。
type custDlg struct {
	name, extra string
	scope       customize.Scope
	event       int
	err         string
}

func (a *App) cuStore() *customize.Store {
	if a.cuDemo() {
		if a.cust == nil {
			a.cust = demo.CustomizeFor(a.demoPgCustomize)
		}
		return a.cust
	}
	a.cuEnsureDisk()
	if a.cust == nil {
		a.cust = customize.NewStore(nil, nil)
	}
	return a.cust
}

// cuCat 返回当前分类(默认 MCP,与设计稿一致)。
func (a *App) cuCat() customize.Cat {
	if a.cu.cat == "" {
		a.cu.cat = customize.MCP
	}
	return a.cu.cat
}

// SetCustomize 直接选中某个分类(截图 / 调试用)。
func (a *App) SetCustomize(cat string) {
	a.cu.cat = customize.Cat(cat)
	a.cuResetNav()
}

// cuSelect 选中一项并载入草稿。
func (a *App) cuSelect(it *customize.Item) {
	a.cu.sel = it
	a.cu.dOf = nil
	a.cu.delArm, a.cu.err, a.cu.hookOut, a.cu.runOut = nil, "", "", ""
	a.cu.ruleEdit = false
	a.cu.skillOf, a.cu.skillFile, a.cu.skillPrev, a.cu.skillPrevErr = "", "", "", ""
}

// cuResetNav 切换分类或整页入口时清空详情与草稿,避免旧选中残留。
func (a *App) cuResetNav() {
	s := &a.cu
	s.sel, s.mk, s.dOf, s.delArm = nil, nil, nil, nil
	s.draft = customize.Item{}
	s.err, s.hookOut, s.runOut, s.detail = "", "", "", false
	s.ruleEdit = false
	s.skillOf, s.skillFile, s.skillPrev, s.skillPrevErr = "", "", "", ""
}

// cuCurrent 返回当前选中项:选中项不在筛选结果里(被筛掉 / 被删)就退回第一项。
func (a *App) cuCurrent(vis []*customize.Item) *customize.Item {
	s := &a.cu
	for _, it := range vis {
		if it == s.sel {
			return it
		}
	}
	if len(vis) == 0 {
		s.sel = nil
		return nil
	}
	a.cuSelect(vis[0])
	return vis[0]
}

func (a *App) customizePage(c *ui.Context) {
	t := a.tk
	s := &a.cu
	st := a.cuStore()
	cat := a.cuCat()
	meta := custCats[cat]
	if cat == customize.MCP && !a.cuMcpManageLoaded {
		a.cuMcpManageLoaded = true
		a.mcpOnOpenManage()
	}
	narrow := a.pageNarrowMid()
	if !narrow {
		s.detail = false
	}
	head := func() {
		ui.Row(c).Padding(16, 20, 10, 20).Gap(12).AlignItems(ui.Start).Shrink(0).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
				ui.Text(c, meta.title).FontSize(theme.FsHead).FontWeight(600)
				ui.Text(c, meta.desc).TextColor(t.Tx3).LineHeight(1.5)
			})
			ui.Row(c).Gap(8).Shrink(0).Children(func() {
				if cat == customize.MCP {
					kit.Btn(c, "刷新状态", kit.BtnOpt{Icon: "refresh", Loading: a.mcpRefreshing}).OnClick(a.cuRefreshMCP)
				}
				if cat == customize.Plugins {
					kit.Btn(c, "浏览市场", kit.BtnOpt{Icon: "search"}).OnClick(func() { s.market = true; s.mk = nil })
					kit.Btn(c, "添加市场…", kit.BtnOpt{Variant: kit.BtnPri, Icon: "plus"}).Tooltip("选一个本地目录作为插件市场(里面要有 .claude-plugin/marketplace.json)").OnClick(a.cuAddMarketplace)
					return
				}
				kit.Btn(c, meta.add, kit.BtnOpt{Variant: kit.BtnPri, Icon: "plus"}).OnClick(func() { s.dlg = &custDlg{scope: customize.User} })
			})
		})
	}
	a.pane(c).Grow(1).MinHeight(0).Padding(0, 0, float32(theme.HStatus), 0).Children(func() {
		switch {
		case narrow && s.detail && s.sel != nil:
			ui.Column(c).Grow(1).MinHeight(0).Children(func() {
				ui.Row(c).Padding(12, 16, 6, 16).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
					kit.Btn(c, "返回列表", kit.BtnOpt{Small: true, Variant: kit.BtnGhost, Icon: "chevL"}).OnClick(func() { s.detail = false })
					ui.Text(c, s.sel.Name).FontWeight(500).SingleLine().Grow(1).MinWidth(0)
				})
				a.cuCatsBar(c, st)
				ui.Row(c).Grow(1).MinHeight(0).Children(func() {
					a.cuDrawer(c, st)
				})
			})
		case narrow:
			ui.Column(c).Grow(1).MinHeight(0).Children(func() {
				head()
				a.cuCatsBar(c, st)
				a.cuMain(c, st)
			})
		default:
			// 两栏:顶部分类标签 + 列表 + 右侧详情(原来左边还有一条 180 宽的分类栏,加上应用侧栏就是四栏,太挤)
			ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Children(func() {
				head()
				a.cuTabs(c, st)
				ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
					a.cuMain(c, st)
					a.cuDrawer(c, st)
				})
			})
		}
	})
	a.cuDialog(c, st)
	a.cuMCPEditDlg(c, st)
}

// cuCatsBar 窄窗顶部横向分类(设计稿 resp · 自定义 <900)。
func (a *App) cuCatsBar(c *ui.Context, st *customize.Store) {
	s := &a.cu
	items := make([]kit.SegItem, len(customize.Cats))
	cur := 0
	for i, cat := range customize.Cats {
		items[i] = kit.SegItem{Text: custCats[cat].title}
		if cat == a.cuCat() {
			cur = i
		}
	}
	if i := kit.Seg(c, items, cur, kit.SegOpt{Key: "cu-cats", Fill: true}); i >= 0 && customize.Cats[i] != s.cat {
		s.cat = customize.Cats[i]
		s.q, s.scope, s.market = "", customize.All, false
		a.cuResetNav()
	}
	ui.Box(c).Height(6)
}

func (a *App) cuRail(c *ui.Context, st *customize.Store) {
	t := a.tk
	s := &a.cu
	ui.Column(c).Key("cu-rail").Width(180).Shrink(0).Padding(12, 8).Gap(1).BorderWidth(0, 0, 0, 1).BorderColor(t.Bd1).Children(func() {
		kit.Grp(c, "自定义")
		for _, cat := range customize.Cats {
			m := custCats[cat]
			r := kit.RowEl(c, cat == a.cuCat()).Key("cr-" + string(cat)).Justify(ui.Start)
			r.OnClick(func() {
				if s.cat != cat {
					s.cat, s.q, s.scope, s.market = cat, "", customize.All, false
					a.cuResetNav()
				}
			})
			r.Children(func() {
				kit.Ic(c, m.icon, kit.S12)
				ui.Text(c, m.title).Grow(1).SingleLine()
				ui.Text(c, itoa(st.Count(cat))).FontSize(theme.FsCap).TextColor(t.Tx4)
			})
		}
	})
}

var cuScopes = []customize.Scope{customize.All, customize.User, customize.Project, customize.Local}

// cuMain 是列表区:筛选条 + 列表(或市场网格)。
func (a *App) cuMain(c *ui.Context, st *customize.Store) {
	t := a.tk
	s := &a.cu
	cat := a.cuCat()
	kit.VScroll(c).Key("cu-main").Grow(1).MinWidth(0).Children(func() { // 宽度 = 主区减去右侧详情(cuDrawerWidth 保证至少 cuListMinW)
		ui.Column(c).Padding(12, 20, 24, 20).Gap(10).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Row(c.Key("cu-qbox")).Grow(1).MinWidth(0).Height(theme.Px(26)).PaddingX(8).Gap(6).AlignItems(ui.Center).Background(t.Fill1).Radius(t.RSm()).Children(func() {
					kit.Ic(c, "search", kit.S12).TextColor(t.Tx4)
					ui.TextInput(c.Key("cu-q"), &s.q).Grow(1).MinWidth(0).Background(ui.Transparent).BorderWidth(0).Padding(0).FontSize(theme.FsCap + .5).
						Placeholder("筛选" + custCats[cat].title).PlaceholderColor(t.Tx4).FocusRing(false)
				})
				if cat == customize.Plugins {
					on := 0
					if s.market {
						on = 1
					}
					if i := kit.Seg(c, []kit.SegItem{{Text: "已安装"}, {Text: "市场"}}, on, kit.SegOpt{Key: "cu-tab"}); i >= 0 {
						s.market, s.mk, s.sel = i == 1, nil, nil
					}
				}
				if !s.market {
					on := 0
					for i, sc := range cuScopes {
						if sc == s.scope {
							on = i
						}
					}
					items := make([]kit.SegItem, len(cuScopes))
					for i, sc := range cuScopes {
						items[i].Text = sc.Label()
					}
					if i := kit.Seg(c, items, on, kit.SegOpt{Key: "cu-scope"}); i >= 0 {
						s.scope = cuScopes[i]
					}
				}
			})
			if cat == customize.Plugins && s.market {
				a.cuMarket(c, st)
				return
			}
			vis := st.List(cat, s.scope, s.q)
			cur := a.cuCurrent(vis)
			if len(vis) == 0 {
				a.cuEmpty(c, st)
				return
			}
			a.box(c).Clip().Children(func() { // 裁剪到卡片圆角:选中 / 悬停行的底色不再盖住四个圆角
				for i, it := range vis {
					a.cuRow(c, st, it, it == cur, i < len(vis)-1)
				}
			})
			hint := "点一行在右侧查看与编辑"
			if a.pageNarrowMid() {
				hint = "点一行进入详情"
			}
			ui.Text(c, itoa(len(vis))+" / "+itoa(st.Count(cat))+" 个 · "+hint).FontSize(theme.FsCap).TextColor(t.Tx4).Padding(4, 2)
		})
	})
}

func (a *App) cuEmpty(c *ui.Context, st *customize.Store) {
	t := a.tk
	cat := a.cuCat()
	m := custCats[cat]
	ui.Column(c).AlignItems(ui.Center).Gap(6).Padding(36, 12).Children(func() {
		kit.Ic(c, m.icon, kit.S20).TextColor(t.Tx4)
		if st.Count(cat) == 0 {
			sp := ""
			if m.title != "" && m.title[0] < 0x80 { // 英文开头的名称与「还没有」之间补空格
				sp = " "
			}
			ui.Text(c, "还没有"+sp+m.title).FontWeight(500)
			// 页头已经写过这类内容的说明,空态只告诉用户下一步
			ui.Text(c, "点右上角的按钮添加第一个。").FontSize(theme.FsCap).TextColor(t.Tx3).TextAlign(ui.Center)
		} else {
			ui.Text(c, "没有符合条件的"+m.title).FontWeight(500)
			ui.Text(c, "换个关键字,或把范围切回「全部」。").FontSize(theme.FsCap).TextColor(t.Tx3)
		}
	})
}

func (a *App) cuRow(c *ui.Context, st *customize.Store, it *customize.Item, on, sep bool) {
	t := a.tk
	r := ui.ButtonBase(c).Key("cur-" + it.Name).PaddingX(12).Height(52).Gap(12).AlignItems(ui.Center).Justify(ui.Start).Cursor(ui.CursorPointer)
	if on {
		r.Background(t.Fill2)
	} else if r.Hovered() {
		r.Background(t.Fill1)
	}
	if sep {
		r.BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1)
	}
	r.OnClick(func() {
		a.cuSelect(it)
		if a.pageNarrowMid() {
			a.cu.detail = true
		}
	})
	r.Children(func() {
		ui.Box(c).Size(24, 24).Radius(t.RSm()).Background(t.Fill2).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, custCats[a.cuCat()].icon, kit.S12).TextColor(t.Tx2)
		})
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			sub := it.Desc
			if a.cuCat() == customize.MCP {
				if row, ok := a.mcpLookup(it.Name); ok {
					sub = row.Detail
				}
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, it.Name).FontWeight(500).SingleLine()
				if a.cuCat() == customize.MCP {
					a.cuMcpDot(c, it.Name)
				} else {
					switch it.Status {
					case customize.StOK:
						kit.Dot(c, kit.KOk)
					case customize.StWarn:
						kit.Dot(c, kit.KWarn)
					case customize.StErr:
						kit.Dot(c, kit.KErr)
					}
				}
				if it.Upd != "" {
					kit.Tag(c, "可更新", kit.KInfo)
				}
				if !it.On {
					kit.Tag(c, "已停用")
				}
			})
			ui.Text(c, sub).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
		})
		kit.Tag(c, it.Scope.Label())
		if a.cuCat() == customize.MCP && !a.cuDemo() {
			if row, ok := a.mcpLookup(it.Name); ok && row.Conn == core.McpFailed { // 连接失败的直接在行上给「编辑」,不用再去详情里找
				kit.Btn(c, "编辑", kit.BtnOpt{Small: true, Variant: kit.BtnSoft, Icon: "edit"}).OnClick(func() {
					a.cuSelect(it)
					a.cuMCPStartEdit(it)
				})
			}
		}
		if it.Status == customize.StWarn {
			kit.Btn(c, "授权", kit.BtnOpt{Small: true, Variant: kit.BtnSoft}).OnClick(func() { a.cuAuth(it) })
		}
		if kit.Tog(c, it.On, "cut-"+it.Name) {
			if a.cuCat() == customize.MCP && !a.cuDemo() {
				a.cuMCPOn(it, !it.On)
			} else if a.cuCat() == customize.Plugins {
				a.cuPluginToggle(it)
			} else {
				it.On = !it.On
				log.Info("自定义启停", "cat", a.cuCat(), "name", it.Name, "on", it.On)
			}
		}
	})
}

func (a *App) cuMcpDot(c *ui.Context, name string) {
	if row, ok := a.mcpLookup(name); ok {
		switch row.Conn {
		case core.McpConnected:
			kit.Dot(c, kit.KOk)
		case core.McpNeedsAuth, core.McpPending:
			kit.Dot(c, kit.KWarn)
		case core.McpFailed:
			kit.Dot(c, kit.KErr)
		default:
			kit.Dot(c, kit.KNeutral)
		}
		return
	}
	kit.Dot(c, kit.KNeutral)
}

func (a *App) cuAuth(it *customize.Item) {
	if a.cuDemo() {
		it.Authorize()
		a.toast = "已授权「" + it.Name + "」"
		return
	}
	a.cuAuthReal(it)
}

// cuMarket 是插件市场:两列卡片。
func (a *App) cuMarket(c *ui.Context, st *customize.Store) {
	t := a.tk
	s := &a.cu
	q := strings.ToLower(strings.TrimSpace(s.q))
	var ms []*customize.Market
	for _, m := range st.Market {
		if q == "" || strings.Contains(strings.ToLower(m.Name+" "+m.Desc), q) {
			ms = append(ms, m)
		}
	}
	mkOK := false
	for _, m := range ms {
		if m == s.mk {
			mkOK = true
			break
		}
	}
	if !mkOK {
		if len(ms) > 0 {
			s.mk = ms[0]
		} else {
			s.mk = nil
		}
	}
	if len(ms) == 0 {
		ui.Text(c, "没有符合条件的插件").TextColor(t.Tx3).Padding(20, 0)
		return
	}
	// 官方市场有 300 多个插件,每帧全画太重:先画一页,其余靠搜索或「显示更多」
	total := len(ms)
	if limit := marketPage * (1 + s.mkMore); total > limit {
		ms = ms[:limit]
	}
	for i := 0; i < len(ms); i += 2 {
		ui.Row(c).Gap(8).AlignItems(ui.Stretch).Children(func() {
			for j := i; j < i+2; j++ {
				if j >= len(ms) {
					ui.Box(c).Grow(1).Basis(0)
					continue
				}
				m := ms[j]
				card := ui.ButtonBase(c).Key("mk-"+m.Name).Grow(1).Basis(0).MinWidth(0).Padding(10).Gap(6).Column().Justify(ui.Start).AlignItems(ui.Stretch).
					Radius(t.RMd()).Border(1, t.Bd1).Background(t.Raised).Cursor(ui.CursorPointer)
				if m == s.mk {
					card.BorderColor(t.Ac)
				}
				card.OnClick(func() { s.mk = m })
				card.Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(24, 24).Radius(t.RSm()).Background(t.Fill2).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
							kit.Ic(c, "plug", kit.S12).TextColor(t.Tx2)
						})
						ui.Column(c).Grow(1).MinWidth(0).Children(func() {
							ui.Text(c, m.Name).FontWeight(500).SingleLine()
							sub := m.By
							if m.MP != "" && m.MP != m.By {
								sub += " · " + m.MP
							}
							if m.Stars != "" {
								sub += " · ★ " + m.Stars
							}
							ui.Text(c, sub).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine()
						})
						if m.Installed {
							kit.Tag(c, "已安装", kit.KOk)
						}
					})
					ui.Text(c, m.Desc).FontSize(theme.FsCap).TextColor(t.Tx2).LineHeight(1.45).MaxLines(3)
				})
			}
		})
	}
	if shown := len(ms); shown < total {
		kit.Btn(c, "显示更多(还有 "+itoa(total-shown)+" 个,也可以直接搜索)", kit.BtnOpt{Variant: kit.BtnGhost}).OnClick(func() { s.mkMore++ })
	}
}

// marketPage 是插件市场每次显示的数量(官方市场有 300 多个)。
const marketPage = 40

// cuDialog 是「新建」对话框。
func (a *App) cuDialog(c *ui.Context, st *customize.Store) {
	s := &a.cu
	if a.dlgClosing == "cu-new" && time.Since(a.dlgCloseAt) >= kit.DurFast {
		a.dlgClosing, a.dlgSeen = "", ""
	}
	d := s.dlg
	if d == nil {
		if a.dlgSeen == "cu-new" || a.dlgClosing == "cu-new" {
			a.dialogOpen("cu-new", false, func(open *bool) {
				t := a.tk
				ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
					backdrop.Background(t.Scrim)
					a.dialogIn("cu-new", backdrop, panel)
				})
			})
		}
		return
	}
	t := a.tk
	cat := a.cuCat()
	meta := custCats[cat]
	ph := map[customize.Cat]string{customize.MCP: "github", customize.Skills: "my-skill", customize.Commands: "/my-command", customize.Agents: "my-agent", customize.Rules: "docs/CLAUDE.md", customize.Hooks: "PostToolUse · Edit"}[cat]
	extraLbl, extraPh := "", ""
	switch cat {
	case customize.MCP:
		extraLbl, extraPh = "命令或地址", "npx -y @modelcontextprotocol/server-…"
	case customize.Commands:
		extraLbl, extraPh = "说明", "这个命令做什么"
	case customize.Agents:
		extraLbl, extraPh = "什么时候委派给它", "例如:审查安全问题"
	}
	create := func() {
		it, err := st.Add(cat, d.name, d.scope, d.extra)
		if err != nil {
			d.err = err.Error()
			return
		}
		if cat == customize.Hooks {
			it.Event = d.event
		}
		log.Info("新建自定义项", "cat", cat, "name", it.Name)
		if ds := a.cuDisk(); ds != nil {
			if err := ds.Persist(cat, it, it.Clone()); err != nil {
				d.err = "写盘失败: " + err.Error()
				return
			}
		}
		s.dlg, s.scope, s.q = nil, customize.All, ""
		a.dlgClosing, a.dlgSeen = "", "" // 成功创建直达终态,避免退场遮罩挡住详情操作
		a.cuSelect(it)
		a.toast = "已新建「" + it.Name + "」"
	}
	a.dialogOpen("cu-new", s.dlg != nil, func(open *bool) {
		ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
			backdrop.Background(t.Scrim).Justify(ui.Center).AlignItems(ui.Center)
			panel.Width(420).Background(t.Raised).Radius(t.RXl()).Column().Border(1, t.Bd1).Padding(16).Gap(12)
			for _, sh := range t.Shadow3() {
				panel.Shadow(sh.X, sh.Y, sh.Blur, sh.Spread, sh.Color)
			}
			kit.Float(panel)
			a.dialogIn("cu-new", backdrop, panel)
			panel.Children(func() {
				ui.Text(c, meta.add).FontSize(theme.FsTitle).FontWeight(600)
				kit.Field(c, "cd-name", kit.FieldOpt{Label: "名称", Err: d.err}, func(box ui.Element) {
					box.Height(theme.Px(30))
					in := ui.TextInputBase(c.Key("cd-name"), &d.name).Grow(1).FontSize(theme.FsBody).Placeholder(ph).PlaceholderColor(t.Tx4).AutoFocus()
					in.OnSubmit(create)
				})
				if extraLbl != "" {
					kit.Field(c, "cd-extra", kit.FieldOpt{Label: extraLbl}, func(box ui.Element) {
						box.Height(theme.Px(30))
						ui.TextInputBase(c.Key("cd-extra"), &d.extra).Grow(1).FontSize(theme.FsBody).Placeholder(extraPh).PlaceholderColor(t.Tx4)
					})
				}
				if cat == customize.Hooks {
					a.cuLbl(c, "事件", func() {
						ui.Row(c).Gap(6).Wrap().Children(func() {
							for i, e := range customize.Events {
								ch := kit.Chip(c, e, kit.ChipOpt{Line: i != d.event, On: i == d.event}).Key("cd-ev-" + e)
								ch.OnClick(func() { d.event = i })
							}
						})
					})
				}
				a.cuLbl(c, "范围", func() {
					on := 0
					for i, sc := range cuScopes[1:] {
						if sc == d.scope {
							on = i
						}
					}
					items := []kit.SegItem{{Text: "用户"}, {Text: "项目"}, {Text: "本地"}}
					if i := kit.Seg(c, items, on, kit.SegOpt{Key: "cd-scope"}); i >= 0 {
						d.scope = cuScopes[1:][i]
					}
				})
				ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
					kit.Btn(c, "取消").OnClick(func() { *open = false })
					kit.Btn(c, "创建", kit.BtnOpt{Variant: kit.BtnPri}).OnClick(create)
				})
			})
		})
	})
}

// cuLbl 是「小标题 + 内容」的竖排。
func (a *App) cuLbl(c *ui.Context, text string, body func()) {
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, text).FontSize(theme.FsCap).FontWeight(500).TextColor(a.tk.Tx3)
		body()
	})
}

// cuTabs 是页头下面的分类标签(原左侧分类栏):图标 + 名称 + 数量,当前项底部一条强调色横线。
func (a *App) cuTabs(c *ui.Context, st *customize.Store) {
	t := a.tk
	s := &a.cu
	ui.Row(c).Key("cu-tabs").Padding(0, 20).Gap(2).Shrink(0).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		for _, cat := range customize.Cats {
			cat := cat
			m := custCats[cat]
			on := cat == a.cuCat()
			b := ui.ButtonBase(c).Key("ct-" + string(cat)).Height(36).PaddingX(12).Gap(6).AlignItems(ui.Center).Cursor(ui.CursorPointer).Shrink(0)
			col := t.Tx3
			if on {
				col = t.Tx1
			} else if b.Hovered() {
				col = t.Tx1
			}
			b.OnClick(func() {
				if s.cat != cat {
					s.cat, s.q, s.scope, s.market = cat, "", customize.All, false
					a.cuResetNav()
				}
			})
			b.Children(func() {
				kit.Ic(c, m.icon, kit.S12).TextColor(col)
				ui.Text(c, m.title).TextColor(col).SingleLine()
				ui.Text(c, itoa(st.Count(cat))).FontSize(theme.FsCap).TextColor(t.Tx4)
				if on { // 当前项:底部强调线(压在分隔线上)
					ui.Box(c).Absolute().Left(8).Right(8).Bottom(-1).Height(2).Radius(1).Background(t.Ac)
				}
			})
		}
	})
}
