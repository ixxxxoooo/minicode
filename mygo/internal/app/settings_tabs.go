package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 行构造器:让下面的分类定义读起来像设计稿的 R(...)。
func rSeg(key, title, desc string, def int, opts ...string) setRow {
	return setRow{key: key, title: title, desc: desc, kind: kSeg, def: def, opts: opts}
}
func rTog(key, title, desc string, def bool) setRow {
	return setRow{key: key, title: title, desc: desc, kind: kTog, def: def}
}
func rStp(key, title, desc string, def, lo, hi, step int, unit string) setRow {
	return setRow{key: key, title: title, desc: desc, kind: kStep, def: def, min: lo, max: hi, step: step, unit: unit}
}
func rFld(key, title, desc, def string, w float32, mono bool, ph string) setRow {
	return setRow{key: key, title: title, desc: desc, kind: kField, def: def, w: w, mono: mono, ph: ph}
}
func rCustom(title, desc string, right func(a *App, c *ui.Context)) setRow {
	return setRow{title: title, desc: desc, kind: kNone, right: right}
}
func restart(r setRow) setRow { r.restart = true; return r }

// rowBox 是自定义块里的一行(与设置行同样的内边距与分隔线)。
func rowBox(a *App, c *ui.Context, first bool, minH float32) ui.Element {
	r := ui.Row(c).MinHeight(minH).Padding(6, 12).Gap(12).AlignItems(ui.Center)
	if !first {
		r.BorderWidth(1, 0, 0, 0).BorderColor(a.tk.Bd1)
	}
	return r
}

// iconBox 是列表行左侧的 26px 图标方块(.si-box)。
func iconBox(a *App, c *ui.Context, icon string) {
	ui.Row(c).Size(26, 26).Radius(a.tk.RSm()).Background(a.tk.Fill1).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).TextColor(a.tk.Tx2).
		Children(func() { kit.Ic(c, icon, kit.S12) })
}

// titled 是两行(标题 + 说明)的左侧文字块。
func titled(a *App, c *ui.Context, title, desc string, mono bool) {
	ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
		ui.Text(c, title).FontWeight(500).SingleLine()
		if desc != "" {
			d := ui.Text(c, desc).FontSize(theme.FsCap).TextColor(a.tk.Tx3).SingleLine()
			if mono {
				d.Font("monospace")
			}
		}
	})
}

// keycap 画一个快捷键(用 kit.Kbd 的键帽)。
func keycap(c *ui.Context, combo string) {
	kit.Kbd(c, strings.Fields(combo)...)
}

// openPath 在访达里打开目录(设置里的「在访达中显示」)。
func openPath(c *ui.Context, p string) {
	if strings.HasPrefix(p, "~/") {
		p = filepath.Join(diag.Home(), "..", p[2:])
		p = filepath.Clean(p)
	}
	_ = os.MkdirAll(p, 0o755)
	if err := exec.Command("open", p).Start(); err != nil {
		c.Toast("无法打开:" + err.Error())
		return
	}
	c.Toast("已在访达中显示 " + p)
}

// ───────── 外观的自定义块 ─────────

// layCards 是三张布局方案卡:点击立即切换整个窗口。
func layCards(a *App, c *ui.Context) {
	t := a.tk
	ui.Column(c).Padding(10, 12, 12, 12).Gap(8).Children(func() {
		ui.Text(c, "三种布局共用同一份令牌与组件,只在「面板怎么分隔、导航放在哪、步骤怎么串」上不同。点一下立即切换,Agent 与 IDE 形态同步变化。").
			FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(1.5)
		info := map[string][2]string{
			"a": {"Ink", "同一平面,细线分隔"},
			"b": {"Float", "面板浮在底色上,圆润柔和"},
			"c": {"Rail", "图标轨常驻,步骤串成时间轴"},
		}
		ui.Row(c).Gap(8).Children(func() {
			for _, s := range theme.Schemes {
				on := t.Scheme.ID == s.ID
				card := ui.ButtonBase(c).Key("lay-"+s.ID).Grow(1).Basis(0).MinWidth(0).Padding(5, 5, 7, 5).Gap(4).Radius(t.RMd()).Cursor(ui.CursorPointer).
					Column().AlignItems(ui.Stretch).Transition(kit.Fade)
				switch {
				case on:
					card.Background(t.AcSoft).Border(1.5, t.Ac)
				case card.Hovered():
					card.Background(t.Fill1).Border(1, t.Bd3)
				default:
					card.Border(1, t.Bd2)
				}
				card.OnClick(func() { a.prefs.Set(kScheme, s.ID); a.applyPrefs() })
				card.Children(func() {
					layPreview(a, c, s.ID)
					ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
						ui.Text(c, s.Name).FontWeight(700)
						ui.Text(c, info[s.ID][0]).FontSize(theme.FsMicro).TextColor(t.Tx4)
					})
					ui.Text(c, info[s.ID][1]).FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(1.4)
				})
			}
		})
	})
}

// layPreview 是方案卡里的微缩示意图:侧栏 / 主区(三条横线)/ 工作区。
func layPreview(a *App, c *ui.Context, id string) {
	t := a.tk
	box := ui.Row(c).Height(70).Padding(4).Gap(1).Radius(t.RSm()).Background(t.Chrome).Border(1, t.Bd1).Clip().AlignItems(ui.Stretch)
	if id == "b" {
		box.Gap(3)
	}
	box.Children(func() {
		panel := func(grow float32, plain bool, edge string) ui.Element {
			p := ui.Column(c).Grow(grow).Basis(0).Radius(1)
			switch {
			case plain && edge == "r":
				p.BorderWidth(0, 1, 0, 0).BorderColor(t.Bd2)
			case plain && edge == "l":
				p.BorderWidth(0, 0, 0, 1).BorderColor(t.Bd2)
			case plain:
			default:
				p.Background(t.BG)
				if id == "b" {
					p.Radius(5)
					for _, s := range t.Shadow1() {
						p.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
					}
				}
			}
			return p
		}
		if id == "c" {
			ui.Box(c).Width(8).Margin(0, 2, 0, 0).BorderWidth(0, 1, 0, 0).BorderColor(t.Bd2)
		}
		sd := 24
		if id == "c" {
			sd = 20
		}
		panel(float32(sd), id == "a" || id == "b", "r")
		panel(54, false, "").Padding(5).Gap(3).Children(func() {
			ui.Box(c).Height(3).Radius(2).Background(t.Ac).WidthPercent(60)
			ui.Box(c).Height(3).Radius(2).Background(t.Fill3)
			ui.Box(c).Height(3).Radius(2).Background(t.Fill3).WidthPercent(75)
		})
		panel(22, id == "a", "l")
	})
}

// themeCards 是六张主题卡(预览:侧栏 + 主区三条线)。
func themeCards(a *App, c *ui.Context) {
	t := a.tk
	ui.Row(c).Gap(6).Wrap().Children(func() {
		for _, p := range theme.Palettes {
			on := a.pal.ID == p.ID
			tk := theme.Resolve(p, t.Scheme, "")
			card := ui.ButtonBase(c).Key("th-" + p.ID).Width(92).Padding(4).Radius(t.RMd()).Cursor(ui.CursorPointer).Column().AlignItems(ui.Stretch).Transition(kit.Fade)
			switch {
			case on:
				card.Border(1.5, t.Ac)
			case card.Hovered():
				card.Border(1, t.Bd3)
			default:
				card.Border(1, ui.Transparent)
			}
			card.OnClick(func() { a.prefs.Set(kTheme, p.ID); a.applyPrefs() })
			card.Children(func() {
				ui.Row(c).Height(50).Radius(t.RMd()-2).Background(tk.Chrome).Border(1, tk.Bd1).Clip().Children(func() {
					ui.Box(c).WidthPercent(28)
					ui.Column(c).Grow(1).Background(tk.BG).Padding(7, 6).Gap(4).Children(func() {
						ui.Box(c).Height(4).Radius(2).Background(tk.Ac).WidthPercent(60)
						ui.Box(c).Height(4).Radius(2).Background(tk.Tx1.Alpha(.22)).WidthPercent(90)
						ui.Box(c).Height(4).Radius(2).Background(tk.Tx1.Alpha(.22)).WidthPercent(70)
					})
				})
				ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Padding(5, 3, 1, 3).Children(func() {
					ui.Text(c, p.Name).FontWeight(700)
					d := "浅"
					if p.Dark {
						d = "深"
					}
					ui.Text(c, d).FontSize(theme.FsMicro).TextColor(t.Tx4)
				})
			})
		}
	})
}

// accentSwatches 是 8 个强调色圆点(第一个是「沿用主题」)。
func accentSwatches(a *App, c *ui.Context) {
	t := a.tk
	cur := a.prefs.Int(kAccent, 0)
	ui.Row(c).Gap(8).Shrink(0).Children(func() {
		for i := 0; i < 8; i++ {
			col := accentOf(a.pal, i)
			if col == "" {
				col = a.pal.Accent
			}
			b := ui.ButtonBase(c).Key("sw-"+itoa(i)).Size(16, 16).Radius(8).Background(ui.Hex(col)).Cursor(ui.CursorPointer).Shrink(0).
				Transition(ui.ElementTransition{Size: true, Duration: kit.DurBase, Ease: kit.EaseSpring})
			if i == cur {
				b.Shadow(0, 0, 0, 2, t.Raised).Shadow(0, 0, 0, 3.5, t.Tx1)
			}
			if b.Clicked() {
				if i == 0 {
					a.prefs.Reset(kAccent)
				} else {
					a.prefs.Set(kAccent, i)
				}
				a.applyPrefs()
			}
		}
	})
}

// ───────── 其它分类的自定义块(演示数据:真实配置接入后读 ~/.claude) ─────────

func ruleRows(a *App, c *ui.Context, kind string, rules []string) {
	t := a.tk
	tagK := map[string]kit.TagKind{"allow": kit.KOk, "ask": kit.KWarn, "deny": kit.KErr}[kind]
	lbl := map[string]string{"allow": "允许", "ask": "询问", "deny": "拒绝"}[kind]
	for i, r := range rules {
		rowBox(a, c, i == 0, 30).Children(func() {
			kit.Tag(c, lbl, tagK)
			kit.Mono(c, r, t.Tx1).Grow(1)
			kit.Ib(c, "trash", kit.IbOpt{Small: true, Tip: "删除"})
		})
	}
}

func provRow(a *App, c *ui.Context, first bool, name, url, status string) {
	rowBox(a, c, first, 44).Children(func() {
		iconBox(a, c, "server")
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, name).FontWeight(500)
				switch status {
				case "已连通":
					kit.Tag(c, status, kit.KOk)
				case "未验证":
					kit.Tag(c, status, kit.KWarn)
				}
			})
			ui.Text(c, url).Font("monospace").FontSize(theme.FsCap).TextColor(a.tk.Tx3).SingleLine()
		})
		kit.Btn(c, "测试", kit.BtnOpt{Small: true, Icon: "refresh"}).OnClick(func() { c.Toast("连通性检测需要真实后端,稍后接入") })
	})
}

func spark(a *App, c *ui.Context, vals []float64) {
	t := a.tk
	box := ui.Box(c).Size(120, 26).Shrink(0)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if len(vals) < 2 {
			return
		}
		lo, hi := vals[0], vals[0]
		for _, v := range vals {
			lo, hi = min(lo, v), max(hi, v)
		}
		if hi == lo {
			hi = lo + 1
		}
		pts := make([][2]float32, len(vals))
		for i, v := range vals {
			pts[i] = [2]float32{r.X + float32(i)/float32(len(vals)-1)*r.W, r.Y + r.H - 2 - float32((v-lo)/(hi-lo))*(r.H-4)}
		}
		p.StrokePolyline(pts, 1.5, t.Ac)
	})
}

// settingsTabs 是全部 14 个分类(顺序无关,导航顺序见 setNavGroups)。
var settingsTabs = []setTabDef{
	{id: "general", title: "通用", icon: "sliders", desc: "启动、窗口、输入、通知与语言。", groups: []setGroup{
		{title: "启动与窗口", rows: []setRow{
			rSeg("general.start", "启动时的形态", "Agent 与 IDE 是同一个窗口的两种形态;「上次」沿用关闭时的形态", 2, "Agent", "IDE", "上次"),
			rTog("general.restore", "恢复上次的会话与布局", "项目、会话、右栏标签、面板开合、编辑器标签", true),
			rSeg("general.close", "关闭窗口时", "红色按钮的行为;「隐藏」时运行中的会话与后台任务继续跑", 0, "隐藏", "退出"),
			rTog("general.insp", "开标签时自动展开右侧工作区", "Agent 形态下在会话里点计划 / 文件 / 审查时展开工作区;各形态的面板开合各自记忆", true),
		}},
		{title: "输入", rows: []setRow{
			rSeg("general.send", "发送键", "另一个键用于换行(Shift+Enter 始终换行)", 0, "Enter", "⌘ Enter"),
			rSeg("general.busy", "回合进行中再发消息", "Enter = 排队;⌘Enter = 立即引导", 0, "排队", "立即引导"),
			rTog("general.paste", "粘贴长文本时折叠为附件", "超过 2000 字时作为文本附件加入,不撑开输入框", true),
		}},
		{title: "通知与菜单栏", rows: []setRow{
			rTog("general.notify", "任务完成时通知", "回合结束且 MiniCode 不在前台时发送系统通知", true),
			rTog("general.notifyAsk", "需要确认时提醒", "权限请求、提问在后台出现时弹通知", true),
			rSeg("general.sound", "通知声音", "", 1, "静音", "轻提示"),
			rTog("general.tray", "菜单栏图标", "在系统菜单栏显示 MiniCode 图标", true),
		}},
		{title: "语言", rows: []setRow{
			rCustom("界面语言", "目前仅提供简体中文", func(a *App, c *ui.Context) { kit.Chip(c, "简体中文", kit.ChipOpt{Line: true}) }),
		}},
	}, note: ""},

	{id: "appearance", title: "外观", icon: "palette", desc: "布局、主题、密度与动效。所有选项都落在设计令牌上,即改即生效。", groups: []setGroup{
		{title: "界面布局", body: func(a *App, c *ui.Context) { layCards(a, c) }},
		{title: "主题", body: func(a *App, c *ui.Context) {
			ui.Column(c).Padding(10, 12).Children(func() { themeCards(a, c) })
			a.settingsRow(c, rTog("appearance.system", "跟随系统深浅色", "浅色用「宣纸」,深色用「墨夜」", false), false)
		}},
		{title: "强调色", rows: []setRow{
			{key: kAccent, title: "强调色", desc: "按钮、选中态、焦点环、选区、链接等使用的主题色;每个主题单独记忆,并按明暗自动调整明度", kind: kNone, def: 0,
				right: func(a *App, c *ui.Context) { accentSwatches(a, c) }},
		}},
		{title: "密度与形状", rows: []setRow{
			{key: kDensity, title: "界面密度", desc: "行高令牌 --h-row:紧凑 −2 / 标准 / 宽松 +2(在当前布局的基础上)", kind: kSeg, def: 1, opts: []string{"紧凑", "标准", "宽松"}},
			{key: kRadius, title: "圆角", desc: "圆角倍率 --r-k:利落 ×0.55 / 适中 / 圆润 ×1.35", kind: kSeg, def: 1, opts: []string{"利落", "适中", "圆润"}},
		}},
		{title: "Agent 会话", rows: []setRow{
			rSeg("appearance.flowStyle", "会话样式", "对话:你的消息靠右成气泡;通栏:与正文同宽的整行卡片", 0, "跟随布局", "对话", "通栏"),
			rSeg("appearance.width", "内容宽度", "会话时间线与输入框的最大宽度;窄屏或分屏时会自动收窄", 1, "窄", "标准", "宽"),
			rSeg("appearance.density2", "工具调用详细程度", "精简只显示动词与对象;详细默认展开输出", 1, "精简", "标准", "详细"),
			rStp("appearance.fontScale", "时间线字号", "会话里 Claude 回复正文的字号(85%–150%)", 100, 85, 150, 5, "%"),
		}},
		{title: "文件树", rows: []setRow{
			rTog("appearance.fileIcons", "彩色文件图标", "按文件类型给图标上色(Rust 橙、Go 蓝、Python 绿…);关闭后全部用灰色。作用于文件树、标签、搜索与源代码管理", true),
		}},
		brandAppGroup(),
		brandTrayGroup(),
		{title: "动效", rows: []setRow{
			rTog("appearance.motion", "动效", "流光、滑入与布局变形;关闭后所有时长令牌归零(也跟随系统「减弱动态效果」)", true),
			rSeg("appearance.morph", "形态切换动画", "Agent ⇄ IDE 的布局变形,约 0.44 秒;「简化」只淡入淡出", 0, "流畅", "简化", "无"),
			rSeg("appearance.shimmer", "流光样式", "「正在进行」的文字扫光", 0, "流光", "呼吸", "无"),
			rSeg(pWaitAnim, "等待动画", "转圈的样式", 0, "圆环", "点阵", "脉冲"),
			rSeg(pScrollbar, "滚动条", "「跟随系统」沿用 macOS「显示滚动条」设置;MyGo 全局滚动条 API 有限,仅部分区域生效", 1, "跟随系统", "自动隐藏", "始终显示"),
		}},
	}},

	{id: "shortcuts", title: "快捷键", icon: "kbd", desc: "点键帽录制;冲突会当场提示;恢复默认会清掉自定义组合。", groups: shortcutGroupsLive()},

	{id: "models", title: "模型与服务商", icon: "cpu", desc: "默认模型与思考强度、上下文压缩,以及第三方模型服务商。密钥单独存放在 0600 权限的文件里。", groups: []setGroup{
		{title: "默认", rows: []setRow{
			{key: prefModel, title: "默认模型", desc: "新会话的起始模型;运行中可在输入框的运行配置里随时切换", kind: kNone, def: "sonnet",
				right: func(a *App, c *ui.Context) { a.defaultModelRow(c) }},
			rSeg(prefEffort, "默认思考强度", "", 2, "关", "低", "中", "高"),
			rTog("models.thinking", "默认开启扩展思考", "alwaysThinkingEnabled", true),
		}},
		{title: "上下文", rows: []setRow{
			rTog("models.compact", "自动压缩", "接近上限时自动 /compact", true),
			rStp("models.compactAt", "压缩触发阈值", "占用达到该比例时压缩", 85, 50, 95, 5, "%"),
			rCustom("输出风格", "", func(a *App, c *ui.Context) { kit.Chip(c, "默认", kit.ChipOpt{Line: true, Car: true}) }),
		}},
		{title: "服务商", body: func(a *App, c *ui.Context) { a.providersBody(a, c) }},
		{title: "模型映射", body: func(a *App, c *ui.Context) { a.mappingBody(a, c) }},
	}},

	{id: "chat", title: "对话与提示词", icon: "chat", desc: "会话流的显示与回合行为,以及追加到系统提示词的内容。", groups: []setGroup{
		{title: "显示", rows: []setRow{
			rTog("chat.thinkOpen", "思考内容默认展开", "关闭时只显示「思考 · 4 秒」一行", false),
			rTog("chat.fold", "回合结束后折叠过程", "把本回合的工具步骤折成「已处理 1 分 13 秒」", true),
			rTog("chat.stickyUser", "用户消息吸顶", "长会话滚动时当前回合的提问固定在顶部;下一轮顶走后只留一张", true),
			rTog("chat.stats", "显示回合用量", "回复下方的模型 · 用时 · token · 费用", true),
			rTog("chat.groupFold", "工具组默认折叠", "连续的读取 / 搜索收进一行;失败时自动展开", true),
		}},
		{title: "新会话", rows: []setRow{
			rSeg(pPermMode, "默认权限模式", "新会话的起始模式;运行中可随时在输入框切换", 0, "逐项确认", "自动接受编辑", "计划"),
			rTog("chat.ding", "回合结束提示音", "", false),
		}},
		{title: "排队与引导", rows: []setRow{
			rTog("chat.queueAtt", "排队消息带图片与文件引用", "", true),
			rTog("chat.queuePause", "被中断后暂停队列", "中断或出错后不自动发送排队消息", true),
		}},
		{title: "追加系统提示词", body: func(a *App, c *ui.Context) { sysPromptArea(a, c) }},
		{title: "记忆文件(CLAUDE.md)", rows: []setRow{
			rCustom("记忆文件已移到「自定义 → 规则与记忆」", "在那里编辑文件、查看加载顺序,并开关「加载 CLAUDE.md」", func(a *App, c *ui.Context) {
				kit.Btn(c, "前往", kit.BtnOpt{Small: true, Icon: "arrow"}).OnClick(func() { a.screen = ScreenCustomize })
			}),
		}},
	}},

	{id: "permissions", title: "权限与信任", icon: "shield", desc: "允许、询问与拒绝的规则,以及哪些项目可以运行钩子与 MCP。", groups: []setGroup{
		{title: "策略", rows: []setRow{
			rSeg("perm.layer", "配置层", "规则写入哪一层", 1, "用户", "项目", "本地"),
			rSeg("perm.mode", "默认权限模式", "permissions.defaultMode", 0, "逐项确认", "自动接受编辑", "计划"),
			rTog("perm.noBypass", "禁止绕过权限模式", "permissions.disableBypassPermissionsMode", false),
			{key: "perm.sandbox", title: "Bash 沙盒", desc: "claude 暂不支持从外部开关,请在 claude 的 /sandbox 里设置", kind: kNone,
				right: func(a *App, c *ui.Context) {
					kit.Tog(c, false, "tog-perm.sandbox-disabled")
				}},
			rFld("perm.dirs", "额外可访问目录", "permissions.additionalDirectories", "~/Projects/shared", 200, true, ""),
		}},
		{title: "权限规则", body: func(a *App, c *ui.Context) { a.rulesBody(a, c) }},
		{title: "项目信任", body: func(a *App, c *ui.Context) { a.trustBody(a, c) }},
	}, note: "陌生项目带来的钩子与 MCP 默认不运行,信任后才生效。"},

	{id: "runtime", title: "运行环境", icon: "cube", desc: "MiniCode 如何找到并启动 claude:可执行文件、代理与网络、环境变量。", groups: []setGroup{
		{title: "claude 命令行", body: func(a *App, c *ui.Context) { a.claudeCmdRow(c) }},
		{title: "代理与网络", rows: []setRow{
			rSeg("runtime.proxy", "代理模式", "跟随系统:沿用继承的 HTTP(S)_PROXY;直连:清空代理变量", 0, "跟随系统", "直连", "自定义"),
			rFld("runtime.proxyUrl", "代理地址", "写入 HTTP_PROXY、HTTPS_PROXY、ALL_PROXY", "http://127.0.0.1:7890", 200, true, ""),
			rFld("runtime.noProxy", "绕过代理", "NO_PROXY:这些主机直连", "localhost,127.0.0.1", 200, true, ""),
			rFld("runtime.timeout", "请求超时(秒)", "API_TIMEOUT_MS;留空为 claude 默认", "", 90, false, "默认"),
		}},
		{title: "环境变量(只显示变量名,值已遮盖)", body: func(a *App, c *ui.Context) { a.envBody(a, c) }},
		{title: "连通性", body: func(a *App, c *ui.Context) { a.netCheckBody(a, c) }},
	}},

	{id: "editor", title: "编辑器与语言服务", icon: "code", desc: "代码编辑器的显示、括号与 Git 追溯,以及语言服务器(LSP)与搜索忽略规则。", groups: []setGroup{
		{title: "字体与缩进", rows: []setRow{
			rStp("editor.fontSize", "字号", "", 12, 9, 24, 1, "px"),
			rSeg("editor.indent", "缩进", "", 1, "2 空格", "4 空格", "Tab"),
			rSeg("editor.autosave", "自动保存", "", 1, "关", "延时", "失焦"),
		}},
		{title: "迷你地图", rows: []setRow{
			rTog("editor.minimap", "显示迷你地图", "", true),
			rSeg("editor.minimapSlider", "滑块", "视口滑块何时显示", 0, "悬停", "常驻"),
			rSeg("editor.minimapWidth", "宽度", "宽一点能看清更多列", 1, "窄", "中", "宽"),
			rTog("editor.minimapGit", "Git 改动刻度", "", true),
		}},
		{title: "显示", rows: []setRow{
			rTog("editor.lineNumbers", "行号", "", true),
			rTog("editor.folding", "代码折叠", "", true),
			rTog("editor.wrap", "自动换行", "", false),
			rTog("editor.bracketMatch", "括号配对高亮", "", true),
			rTog("editor.bracketColor", "括号配对着色", "", true),
			rTog("editor.bracketGuide", "括号配对竖线", "", true),
			rTog("editor.syntax", "语法高亮", "", true),
		}},
		{title: "Markdown 与格式化", rows: []setRow{
			rSeg("editor.mdView", "Markdown 默认视图", "打开 .md / .mdc 时先显示哪一种;标签上随时可切换", 0, "预览", "源码"),
			rTog("editor.formatOnSave", "保存时格式化", "语言服务可用时用它,否则用内置格式化(Go / JSON / Markdown / 文本)", false),
			rTog("editor.trimOnSave", "保存时去掉行尾空白", "并保证文件末尾只有一个换行", false),
		}},
		{title: "Git 追溯", rows: []setRow{
			rTog("editor.blameInline", "行内追溯", "光标行末尾显示作者与时间", true),
			rTog("editor.blameGutter", "追溯栏", "行号左侧逐行显示提交", false),
		}},
		{title: "语言服务", rows: []setRow{
			rTog("editor.lsp", "启用语言服务", "悬停、补全、转到定义、诊断", true),
			rTog("editor.inlay", "内联提示", "类型与参数名(inlayHint);无能力时自动跳过", true),
			rTog("lsp.workspaceDiag", "工作区级诊断", "工作区分段下可拉取或扫描项目源文件(有上限)", true),
			rFld("lsp.workspaceDiagLimit", "工作区扫描文件上限", "单次检查最多包含的文件数", "2000", 60, false, ""),
			rTog("lsp.workspaceDiagGitignore", "扫描时包含被 gitignore 的文件", "关闭时跳过 target、node_modules 等常见目录", false),
		}},
		{title: "语言服务器", body: func(a *App, c *ui.Context) { a.lspBody(a, c) }},
		{title: "忽略规则", body: func(a *App, c *ui.Context) { ignoreArea(a, c) }},
	}},

	{id: "terminal", title: "终端", icon: "terminal", desc: "内置终端的外观与 Shell。", groups: []setGroup{
		{title: "外观", rows: []setRow{
			rStp("term.fontSize", "字号", "终端文字大小,行高自动按比例换算", 12, 9, 24, 1, "px"),
			rFld("term.font", "字体", "终端使用的等宽字体;留空 = 系统等宽字体", "", 150, true, "SF Mono"),
			rSeg("term.lineHeight", "行高", "", 1, "紧凑", "标准", "宽松"),
			rSeg("term.cursor", "光标形状", "", 0, "块", "竖线", "下划线"),
			rTog("term.blink", "光标闪烁", "", true),
			rFld("term.scrollback", "回看行数", "滚轮可回看的历史输出行数;对之后新开的终端生效", "10000", 90, false, ""),
			rTog("term.nerdFallback", "无 Nerd Font 时回退系统等宽", "图标显示成方框时可开启", true),
		}},
		{title: "Shell", rows: []setRow{
			rFld("term.shell", "Shell 程序", "留空使用环境变量 $SHELL;失焦或回车应用", "/bin/zsh", 160, true, ""),
			rTog("term.login", "登录 shell", "", true),
			rTog("term.cwd", "在项目目录启动", "", true),
			rTog("term.shellIntegration", "Shell 集成 (OSC 133)", "命令成败装饰与点击复制输出;关闭后不再自动 source 集成脚本", true),
		}},
	}},

	{id: "scm", title: "源代码管理", icon: "branch", desc: "改动面板、差异与 Worktree。", groups: []setGroup{
		{title: "改动面板", rows: []setRow{
			rSeg("scm.diff", "Diff 视图", "改动、提交详情里差异的默认呈现;差异块右上角也能随时切换", 0, "左右", "统一"),
			rFld("scm.diffMax", "Diff 大小上限", "超过这个大小的文件不计算 diff,避免卡顿", "512 KB", 100, false, ""),
			rTog("scm.aiMsg", "AI 提交说明", "用 Haiku 根据改动生成", true),
			rTog("scm.fetch", "自动获取远端", "每 5 分钟 git fetch", false),
			rTog(pScmExpandAll, "多仓库:全部展开", "多根工作区里每个仓库各显示一整块更改与提交框(默认只显示活动仓库 + 总览)", false),
		}},
		{title: "Worktree", rows: []setRow{
			rFld("scm.worktrees", "存放位置", "用于新建 worktree 时的默认位置(新建入口尚未做)", "~/.minicode/worktrees", 200, true, ""),
		}, body: func(a *App, c *ui.Context) { a.scmWorktreeBody(c) }},
	}},

	{id: "browser", title: "浏览器", icon: "globe", desc: "内置阅读模式浏览器的抓取参数。浏览器页右上角的齿轮会直接跳到这里。", groups: []setGroup{
		{title: "抓取", rows: []setRow{
			rTog("browser.fetch", "允许抓取网页", "关闭后只保留演示源与已缓存页面;真实网址会提示去设置开启", true),
			rStp("browser.timeout", "请求超时", "单个页面的最长等待时间", 20, 5, 120, 5, "秒"),
			rFld("browser.ua", "User-Agent", "留空使用内置标识;个别站点会按它返回不同内容", "", 220, false, "默认"),
			rTog("browser.reader", "默认进入阅读模式", "", true),
		}},
	}},

	{id: "privacy", title: "隐私与诊断", icon: "lock", desc: "默认不上报任何数据;性能、崩溃与卡顿记录都在本机。", groups: []setGroup{
		{title: "数据上报", rows: []setRow{
			rTog("privacy.crash", "崩溃报告", "本应用不上报任何数据;此开关仅为占位", false),
			rTog("privacy.usage", "使用统计", "本应用不上报任何数据;此开关仅为占位", false),
			rCustom("凭据", "MiniCode 不读取钥匙串;服务商 API Key 单独存放在 0600 权限的文件里", func(a *App, c *ui.Context) { kit.Tag(c, "本地", kit.KOk) }),
		}},
		{title: "性能(最近 5 分钟,来自本进程的实时采样)", body: perfBody},
		{title: "日志与诊断包", rows: []setRow{
			rSeg(pPrivacyLvl, "日志级别", "环境变量 MINI_LOG 优先", 2, "错误", "警告", "信息", "调试"),
			rCustom("记录目录", "panic、崩溃、卡死、异常退出的记录与日志文件", func(a *App, c *ui.Context) {
				kit.Btn(c, "在访达中显示", kit.BtnOpt{Small: true, Icon: "folder"}).OnClick(func() { openPath(c, diag.Home()) })
			}),
			rCustom("诊断包", "家目录显示为 ~,只含配置概况与变量名,不含密钥和环境变量值", func(a *App, c *ui.Context) {
				kit.Btn(c, "导出", kit.BtnOpt{Small: true, Icon: "download"}).OnClick(func() { a.exportDiagBundle() })
			}),
		}, body: func(a *App, c *ui.Context) {
			ui.Column(c).Children(func() {
				ui.Text(c, "崩溃与卡顿").FontSize(theme.FsCap).FontWeight(700).TextColor(a.tk.Tx3).Padding(8, 12, 4, 12)
				a.privacyCrashesBody(c)
				rowBox(a, c, false, 36).Children(func() {
					ui.Text(c, "claude doctor").FontWeight(500).Grow(1)
					kit.Btn(c, "运行", kit.BtnOpt{Small: true, Icon: "play"}).OnClick(func() { a.runClaudeDoctor() })
				})
				a.doctorFoldBody(c)
			})
		}},
	}},

	{id: "advanced", title: "高级", icon: "wrench", desc: "数据、运行模式、实验功能与重置。", groups: []setGroup{
		{title: "数据", rows: []setRow{
			rCustom("数据目录", "设置、服务商、日志与诊断包所在目录", func(a *App, c *ui.Context) {
				kit.Mono(c, "~/.minicode", a.tk.Tx2)
				kit.Btn(c, "打开", kit.BtnOpt{Small: true, Icon: "folder"}).OnClick(func() { openPath(c, diag.Home()) })
			}),
			{key: "settings.io", title: "设置文件", desc: "导出 / 导入 settings.json(不含密钥)", kind: kNone, right: func(a *App, c *ui.Context) {
				ui.Row(c).Gap(8).Children(func() {
					kit.Btn(c, "导出", kit.BtnOpt{Small: true, Icon: "download"}).OnClick(func() { a.exportSettingsJSON() })
					kit.Btn(c, "导入…", kit.BtnOpt{Small: true, Icon: "upload"}).OnClick(func() { a.askImportSettings() })
				})
			}},
		}},
		{title: "运行模式", rows: []setRow{
			restart(rSeg("adv.runmode", "真实 / 演示", "演示用脚本化后端与沙盒项目,不连接 claude;切换后会重启", 0, "真实", "演示")),
		}},
		{title: "实验功能", rows: []setRow{
			rTog("adv.liveOut", "前台命令实时输出", "命令运行期间每 0.7 秒读取任务输出文件,在工具卡里实时显示", true),
			rTog(pAdvWatchdog, "主线程卡死看门狗", "停滞超过阈值时抓取调用栈,写入诊断记录", true),
			rTog(pAdvFlowMeas, "会话流一次测完行高", "载入长会话时先测量全部行,滚动更稳(首帧稍慢)(Go 版暂不生效:列表由库自行测量)", true),
		}},
		{title: "Claude 配置摘要(演示)", body: func(a *App, c *ui.Context) { a.claudeDiagBody(a, c) }},
		{title: "重置", rows: []setRow{
			rCustom("恢复默认客户端偏好", "主题、发送键、通知、代理等恢复默认;不影响 Claude 配置、栏宽与最近项目", func(a *App, c *ui.Context) {
				kit.Btn(c, "恢复默认…", kit.BtnOpt{Small: true, Variant: kit.BtnDanger}).OnClick(func() {
					a.prefs.ResetAll()
					a.set = settingsState{}
					a.applyPrefs()
					c.Toast("已恢复默认设置")
				})
			}),
		}},
	}},

	{id: "about", title: "关于", icon: "info", groups: []setGroup{
		{body: aboutCard},
		{title: "更新", rows: []setRow{
			rTog("about.autoUpdate", "自动下载更新", "发现新版本时自动下载,重启后生效", true),
			rSeg("about.channel", "更新通道", "", 0, "稳定", "测试"),
		}},
		{rows: []setRow{
			rCustom("许可", "", func(a *App, c *ui.Context) { ui.Text(c, "MIT").TextColor(a.tk.Tx3) }),
			rCustom("开源致谢", "使用到的第三方库与许可", func(a *App, c *ui.Context) {
				ui.ButtonBase(c).Cursor(ui.CursorPointer).OnClick(func() { openURL("https://github.com/ixxxxoooo/minicode") }).Children(func() {
					kit.Ic(c, "chevR", kit.S12).TextColor(a.tk.Tx4)
				})
			}),
			rCustom("在 GitHub 上查看", "发行说明与问题反馈", func(a *App, c *ui.Context) {
				ui.ButtonBase(c).Cursor(ui.CursorPointer).OnClick(func() { openURL("https://github.com/ixxxxoooo/minicode") }).Children(func() {
					kit.Ic(c, "ext", kit.S12).TextColor(a.tk.Tx4)
				})
			}),
		}},
	}},
}

// shortcutGroups 是快捷键分类的分组(显示当前生效的键位)。
func shortcutGroups() []setGroup {
	mk := func(title string, items ...[2]string) setGroup {
		rows := make([]setRow, len(items))
		for i, it := range items {
			combo := it[1]
			rows[i] = rCustom(it[0], "", func(a *App, c *ui.Context) { keycap(c, combo) })
		}
		return setGroup{title: title, rows: rows}
	}
	return []setGroup{
		mk("全局", [2]string{"命令面板", "⌘ K"}, [2]string{"新对话", "⌘ N"}, [2]string{"新窗口", "⇧ ⌘ N"}, [2]string{"设置", "⌘ ,"}, [2]string{"循环权限模式", "⇧ ⇥"}),
		mk("形态与面板", [2]string{"切换 Agent / IDE", "⌘ E"}, [2]string{"显示 / 隐藏导航栏", "⌘ B"}, [2]string{"显示 / 隐藏右侧栏", "⌥ ⌘ B"}, [2]string{"底部面板", "⌘ J"}, [2]string{"源代码管理", "⌃ ⇧ G"}),
		mk("编辑器", [2]string{"当前文件查找", "⌘ F"}, [2]string{"项目内搜索", "⇧ ⌘ F"}, [2]string{"转到符号", "⇧ ⌘ O"}, [2]string{"保存", "⌘ S"}, [2]string{"行注释", "⌘ /"}),
		mk("对话", [2]string{"立即引导", "⌘ ⏎"}, [2]string{"中断回合", "esc"}, [2]string{"编辑上一条消息", "↑"}),
	}
}

// sysPromptArea 是「追加系统提示词」的多行输入。
func sysPromptArea(a *App, c *ui.Context) {
	t := a.tk
	if a.set.bufs == nil {
		a.set.bufs = map[string]*string{}
	}
	const key = "chat.sys"
	buf, ok := a.set.bufs[key]
	if !ok {
		s := a.prefs.Str(key, "所有回答用中文,注释也用中文。\n提交信息使用 Conventional Commits。\n不要读取或输出任何密钥。")
		buf = &s
		a.set.bufs[key] = buf
	}
	ui.Column(c).Padding(10, 12).Gap(6).Children(func() {
		in := ui.TextArea(c.Key("sys"), buf).Height(88).FontSize(theme.FsCap + .5)
		in.OnChange(func() { a.prefs.Set(key, *buf) })
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, "以 --append-system-prompt 传给 claude,新会话生效;多根工作区说明会自动合并").FontSize(theme.FsCap).TextColor(t.Tx3).Grow(1)
			ui.Text(c, itoa(len([]rune(*buf)))+" 字").FontSize(theme.FsCap).TextColor(t.Tx4).FontFeatures("tnum")
		})
	})
}

// ignoreArea 是「忽略规则」的多行输入(同时作用于文件树、搜索与文件监视)。
func ignoreArea(a *App, c *ui.Context) {
	t := a.tk
	if a.set.bufs == nil {
		a.set.bufs = map[string]*string{}
	}
	const key = "editor.ignore"
	buf, ok := a.set.bufs[key]
	if !ok {
		s := a.prefs.Str(key, "target\nnode_modules\n.git\ndist")
		buf = &s
		a.set.bufs[key] = buf
	}
	ui.Column(c).Padding(10, 12).Gap(6).Children(func() {
		in := ui.TextArea(c.Key("ign"), buf).Height(80).Font("monospace").FontSize(theme.FsCap + .5)
		in.OnChange(func() { a.prefs.Set(key, *buf) })
		ui.Text(c, "同时作用于文件树、搜索与文件监视").FontSize(theme.FsCap).TextColor(t.Tx3)
	})
}

// perfBody 是隐私页的「性能」块:读 diag 的真实采样与统计(内存、协程数、界面构建耗时)。
func perfBody(a *App, c *ui.Context) {
	t := a.tk
	ss := diag.Samples()
	var mem, gr []float64
	for _, s := range ss {
		mem = append(mem, float64(s.HeapAlloc))
		gr = append(gr, float64(s.Goroutines))
	}
	memTxt, grTxt := "—", "—"
	if n := len(ss); n > 0 {
		memTxt = diag.FormatBytes(ss[n-1].HeapAlloc)
		grTxt = itoa(ss[n-1].Goroutines)
	}
	frame := "暂无"
	for _, st := range diag.Stats() {
		if st.Name == "app.构建界面" && st.Count > 0 {
			frame = "平均 " + (time.Duration(int64(st.Total) / st.Count)).Round(10*time.Microsecond).String() + " · 最慢 " + st.Max.Round(10*time.Microsecond).String()
		}
	}
	line := func(first bool, title, desc, val string, vals []float64) {
		rowBox(a, c, first, 38).Children(func() {
			titled(a, c, title, desc, false)
			if vals != nil {
				spark(a, c, vals)
			}
			ui.Text(c, val).FontWeight(700).FontFeatures("tnum").MinWidth(70).TextAlign(ui.End)
		})
	}
	line(true, "内存", "本进程堆上存活对象", memTxt, mem)
	line(false, "协程", "goroutine 数,持续增长说明有泄漏", grTxt, gr)
	line(false, "界面构建耗时", "每帧构建界面的耗时;超过 33 ms 会记「慢帧」日志", frame, nil)
	rowBox(a, c, false, 38).Children(func() {
		titled(a, c, "最近一次卡顿", "界面线程停滞超过 3 秒才会记录", false)
		ui.Text(c, "无").TextColor(t.Tx3)
	})
}

// aboutCard 是「关于」顶部的版本卡。
func aboutCard(a *App, c *ui.Context) {
	t := a.tk
	ui.Column(c).Padding(22).Gap(6).AlignItems(ui.Center).Children(func() {
		a.brandLogo(c, 96, false)
		ui.Text(c, "MiniCode").FontSize(theme.FsHead).FontWeight(700).Margin(6, 0, 0, 0)
		ui.Text(c, "版本 "+Version).TextColor(t.Tx3)
		if hint, ok := a.updAboutHint(); ok {
			col := t.Ok
			ic := "okc"
			if a.updSt.Phase == updFail || a.updSt.Phase == updNetErr {
				col, ic = t.Err, "errc"
			} else if a.updSt.Phase == updDownload || a.updSt.Phase == updVerify || a.updSt.Phase == updUnpack || a.updSt.Phase == updChecking {
				col, ic = t.Ac, "download"
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Height(16).Children(func() {
				kit.Ic(c, ic, kit.S12).TextColor(col)
				ui.Text(c, hint).FontSize(theme.FsCap).TextColor(col)
			})
		} else {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Height(16).Children(func() {
				kit.Ic(c, "okc", kit.S12).TextColor(t.Ok)
				ui.Text(c, "已是最新").FontSize(theme.FsCap).TextColor(t.Ok)
			})
		}
		ui.Row(c).Gap(8).Margin(6, 0, 0, 0).Children(func() {
			kit.Btn(c, "检查更新", kit.BtnOpt{Icon: "refresh"}).OnClick(func() {
				a.checkUpdateAsync()
				if h, ok := a.updAboutHint(); ok {
					c.Toast(h)
				} else {
					c.Toast("正在检查更新…")
				}
			})
			kit.Btn(c, "复制版本信息", kit.BtnOpt{Variant: kit.BtnGhost, Icon: "copy"}).OnClick(func() {
				c.WriteClipboard("MiniCode " + Version)
				c.Toast("已复制版本信息")
			})
		})
	})
}

// Version 是显示用的版本号(构建时可用 -ldflags 覆盖)。
var Version = "0.3.0-mygo"
