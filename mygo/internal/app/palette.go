package app

import (
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 命令面板(⌘K,设计稿「页面:命令面板」):输入 + 分组列表 + 右侧预览 + 底部提示。
// 无查询时是「欢迎列表」(常用命令、最近会话、最近文件);有查询时按模糊得分排序;
// 「主题:切换…」展开二级主题列表,上下移动即时预览,回车确认,Esc 还原。

// palState 是面板的临时状态,每次打开重置。
type palState struct {
	query    string
	sel      int
	sub      string // "" = 主列表;"theme" = 主题二级列表
	origPal  string // 进入主题列表时的主题,Esc 时还原
	lastSel  int    // 上一帧的选中项(变了才滚动到可见)
	lastQ    string
	listSt   ui.ScrollState
	preview  string // 文件预览缓存的路径
	previewT string
	palAt    time.Time // 列表 stagger 起点
}

// palItem 是列表里的一项。
type palItem struct {
	group  string // 命令 / 会话 / 文件 / 主题
	icon   string
	title  string
	right  string // 右侧文字(快捷键 / 项目名 / 目录)
	desc   string // 预览里的说明
	path   string // 文件项的相对路径
	theme  string // 主题项的主题 id
	run    func() // 执行(返回后面板关闭,除非 keep)
	keep   bool   // 执行后不关闭(进入二级列表)
	off    bool   // 置灰:不能执行
	offWhy string
}

// 主题的副标题(设计稿 U.THEMES)。
var themeSub = map[string]string{
	"paper": "暖白 · 橙", "mist": "冷灰 · 靛蓝", "celadon": "灰绿 · 松石",
	"ink": "暖黑 · 橙", "graphite": "中性黑 · 钴蓝", "aurora": "深蓝 · 堇紫",
}

const (
	palRowH  = 28
	palCapH  = 24
	palListH = 250
)

// openPalette 打开面板并重置状态。
func (a *App) openPalette() {
	a.pq = palState{lastSel: -1}
	a.paletteOn = true
	a.pq.palAt = time.Now()
	log.Debug("打开命令面板")
}

// closePalette 关闭面板;还在主题预览中就还原。
func (a *App) closePalette() {
	if a.pq.sub == "theme" && a.pq.origPal != "" && a.pal.ID != a.pq.origPal {
		a.previewPalette(a.pq.origPal) // 取消:还原(偏好没动过,不必写)
	}
	a.paletteOn = false
	a.pq.sub = ""
}

// previewPalette 临时套用主题(命令面板里上下移动时预览),不写偏好。
func (a *App) previewPalette(id string) {
	a.pal = theme.Find(id)
	a.accent = accentOf(a.pal, a.prefs.Int(kAccent, 0))
	a.retheme()
}

// setPalette / setScheme 换主题与布局方案并记进偏好(命令面板确认、设置页共用)。
func (a *App) setPalette(id string) { a.prefs.Set(kTheme, id); a.applyPrefs() }
func (a *App) setScheme(id string)  { a.prefs.Set(kScheme, id); a.applyPrefs() }

// descHit 用说明文字做子串匹配(不区分大小写)。至少 2 个字符才匹配,免得单字查询把说明里的字都撞上。
func descHit(q, desc string) bool {
	if len([]rune(q)) < 2 || desc == "" {
		return false
	}
	return strings.Contains(strings.ToLower(desc), strings.ToLower(q))
}

// fuzzy 是子序列模糊匹配:返回得分(越大越好),不匹配为 -1。
// 连续命中、词首命中与靠前命中加分,与 VS Code 的快速打开手感接近。
func fuzzy(q, s string) int {
	if q == "" {
		return 0
	}
	qr := []rune(strings.ToLower(q))
	sr := []rune(strings.ToLower(s))
	score, qi, prev := 0, 0, -2
	for i, r := range sr {
		if qi >= len(qr) || r != qr[qi] {
			continue
		}
		score += 10
		if prev == i-1 {
			score += 8 // 连续
		}
		if i == 0 || !unicode.IsLetter(sr[i-1]) && !unicode.IsDigit(sr[i-1]) {
			score += 6 // 词首
		}
		score -= i / 8 // 越靠前越好
		prev = i
		qi++
	}
	if qi < len(qr) {
		return -1
	}
	return score
}

// palCommands 是全部命令;common 标记「欢迎列表」里出现的常用项。
func (a *App) palCommands() (cmds []palItem, common map[int]bool) {
	t := a.tk
	_ = t
	add := func(icon, title, right, desc string, run func()) int {
		cmds = append(cmds, palItem{group: "命令", icon: icon, title: title, right: right, desc: desc, run: run})
		return len(cmds) - 1
	}
	common = map[int]bool{}
	addPal := func(icon, title, right, desc string, ok bool, why string, run func()) {
		i := add(icon, title, right, desc, run)
		if !ok {
			cmds[i].off, cmds[i].offWhy = true, why
		} else {
			common[i] = true
		}
	}
	common[add("palette", "主题:切换…", "⏎ 展开", "展开主题列表。上下移动时即时预览,回车确认,Esc 还原。", func() {
		a.pq.sub, a.pq.origPal, a.pq.query, a.pq.sel = "theme", a.pal.ID, "", 0
		for i, p := range theme.Palettes {
			if p.ID == a.pal.ID {
				a.pq.sel = i
			}
		}
	})] = true
	cmds[0].keep = true
	if a.mode == ModeAgent {
		common[add("swap", "切换到 IDE", "⌘E", "把窗口切到 IDE 形态:编辑区在前,对话停靠在右侧。", func() { a.SetMode(ModeIDE) })] = true
	} else {
		common[add("swap", "切换到 Agent", "⌘E", "把窗口切到 Agent 形态:对话在前,工作区在右侧。", func() { a.SetMode(ModeAgent) })] = true
	}
	if tb := a.ide.cur(); tb != nil && tb.buf != nil && isMarkdownFile(tb.path) {
		add("code", "Markdown:切换预览 / 源码", "", "当前 Markdown 标签在渲染后直接编辑与源码之间切换,两边读写同一份文本。", a.toggleMdView)
	}
	if tb := a.ide.cur(); tb != nil && tb.buf != nil && tb.virt == "" && !tb.diff {
		add("align", "格式化文档", "⇧⌥F", "有语言服务器就交给它,否则用内置格式化(Go / JSON / Markdown / 文本)。", a.formatDoc)
		add("wrap", "切换自动换行", "⌥Z", "编辑器与差异视图共用的偏好 editor.wrap。", a.toggleWrap)
		add("search", "编辑器:放大字号", "⌘+", "编辑器字号 +1(最大 24 px)。", func() { a.setEditorFont(a.prefs.Int(pEdFont, edFontDefault) + 1) })
		add("search", "编辑器:缩小字号", "⌘-", "编辑器字号 −1(最小 9 px)。", func() { a.setEditorFont(a.prefs.Int(pEdFont, edFontDefault) - 1) })
		add("refresh", "编辑器:重置字号", "⌘0", "把编辑器字号恢复为默认的 12 px。", a.resetEditorFont)
		add("list", "转到行…", a.shortcutLabel("gotoLine"), "在当前文件跳到指定行,支持 行:列。", a.openGotoLine)
		add("layers", "块注释", a.shortcutLabel("blockComment"), "用块注释包裹选区;无块注释时改用行注释。", a.toggleBlockComment)
		okMc, whyMc := a.tbCodeEditable(tb)
		addPal("cursor", "添加下一个匹配", a.shortcutLabel("selectNext"), "在当前词上重复添加选区。", okMc, whyMc, func() { a.mcRun(tb, editor.SelectNext) })
		addPal("cursor", "选中所有匹配", a.shortcutLabel("selectAllMatches"), "把当前词的所有出现变成多个光标。", okMc, whyMc, func() { a.mcRun(tb, editor.SelectAllMatches) })
		addPal("cursor", "回到单光标", "Esc", "只保留主插入点。", okMc, whyMc, func() { a.mcSingle(tb) })
	}
	common[add("search", "在项目中搜索", "⇧⌘F", "在整个项目里搜索 / 替换文本,支持正则、全词、包含与排除。", func() { a.OpenSearch() })] = true
	common[add("branch", "源代码管理", "⌃⇧G", "查看改动、暂存与提交,浏览提交图谱。", func() { a.OpenSCM() })] = true
	common[add("branch", "Git:切换分支…", "", "搜索本地与远程分支并切换;输入新名字可创建并切换。", func() { a.OpenQuick("branch", "") })] = true
	common[add("history", "历史会话…", "", "按标题搜索并打开历史会话。", func() { a.OpenQuick("history", "") })] = true
	common[add("sliders", "设置:外观", a.shortcutLabel("settings"), "打开设置的「外观」:主题、布局方案、强调色。", func() { a.openSettings("appearance") })] = true
	common[add("gear", "设置:通用", "", "打开设置的「通用」。", func() { a.openSettings("general") })] = true
	for _, tab := range settingsTabs {
		if tab.id == "appearance" || tab.id == "general" {
			continue
		}
		tab := tab
		desc := "打开设置的「" + tab.title + "」。"
		if tab.desc != "" {
			desc = tab.desc
		}
		common[add(tab.icon, "设置:"+tab.title, "", desc, func() { a.openSettings(tab.id) })] = true
	}
	common[add("shield", "切换权限模式", a.shortcutLabel("modeCycle"), "循环切换逐项确认 / 自动接受编辑 / 计划等权限模式。", a.cyclePermissionMode)] = true
	okSess := a.hasActiveSession()
	addPal("compress", "压缩上下文", "", "发送 /compact,整理当前会话的上下文。", okSess, "还没有活动会话", func() { a.send(core.CmdCompact{}) })
	addPal("plug", "刷新 MCP 状态", "", "重读 MCP 配置;会话已启动时向 Claude 查询连接状态。", okSess, "还没有活动会话", func() {
		a.mcpManualRefresh(false)
		if !a.mcpRuntimeReady() {
			a.toast = "已重读配置;发送第一条消息后可查看连接状态"
		}
	})
	addPal("fork", "分叉当前会话", "", "以当前会话为起点复制出一条新会话。", okSess, "还没有活动会话", func() { a.forkSessionAsk(a.curSess) })
	common[add("chat", "新会话", a.shortcutLabel("newChat"), "开始新会话并回到欢迎页。", a.NewSession)] = true
	addPal("list", "打开计划", "", "在工作区打开最新一份计划。", okSess && a.latestPlan() != nil, map[bool]string{true: "本轮还没有计划", false: "还没有活动会话"}[okSess], a.paletteOpenPlan)
	addPal("eye", "打开审查", "", "在工作区打开「审查 · 本轮改动」标签。", okSess, "还没有活动会话", a.OpenReview)
	addPal("terminal", "打开任务输出", "", "在工作区打开后台任务输出标签。", okSess && a.st != nil && len(a.st.Tasks) > 0, map[bool]string{true: "没有可查看的任务", false: "还没有活动会话"}[okSess], a.paletteOpenTask)
	lbl := map[bool]string{true: "演示", false: "真实"}[!a.runDemo]
	common[add("swap", "切换到"+lbl+"模式(重启)", "", "切换真实 / 演示后端并重启本应用(开发版会记住选择)。", a.askSwitchRunMode)] = true
	common[add("panelL", "显示 / 隐藏导航栏", "⌘B", "收起或展开左侧导航栏。", func() { a.sideOpen = !a.sideOpen })] = true
	if !a.secondary {
		add("win", "新建窗口", "⇧⌘N", "再起一个本应用进程,会话与布局各自独立。", func() { a.SpawnNewWindow(a.proj) })
	}
	add("panelR", "显示 / 隐藏右侧栏", "⌥⌘B", "Agent 形态 = 工作区,IDE 形态 = 对话栏。", a.toggleRight)
	if a.mode == ModeIDE {
		add("terminal", "显示 / 隐藏底部面板", "⌘J", "终端、问题、输出所在的底部面板。", func() { a.ide.bottomOpen = !a.ide.bottomOpen })
		add("download", "保存当前文件", "⌘S", "把当前编辑器标签写回磁盘。", func() {
			if !a.extmodSaveIntercept() {
				_ = a.ide.save()
			}
		})
	}
	for _, s := range theme.Schemes {
		s := s
		add("cols", "布局:"+s.Name, "", "把整个窗口切到「"+s.Name+"」布局方案。", func() { a.setScheme(s.ID) })
	}
	if a.demoExtra {
		add("fork", "演示:打开合并冲突", "", "写入一个带冲突标记的文件并打开三方合并标签。", a.demoConflict)
		for i, name := range demo.CanvasSetNames() {
			i, name := i, name
			add("layers", name, "", "切换画布演示数据套并刷新列表。", func() { a.demoSwitchCanvas(i) })
		}
		for i, name := range demo.AutomationsSetNames() {
			i, name := i, name
			add("clock", name, "", "切换自动化演示数据套。", func() { a.demoSwitchAuto(i) })
		}
		add("clock", "演示:自动化·模拟运行中", "", "把第一条自动化标成运行中(界面态)。", a.demoAutoSimBusy)
		for i, name := range demo.BrowserSetNames() {
			i, name := i, name
			add("globe", name, "", "切换浏览器演示数据并预置历史。", func() { a.demoSwitchBrowser(i) })
		}
		for i, name := range demo.CustomizeSetNames() {
			i, name := i, name
			add("plug", name, "", "切换自定义页演示数据套。", func() { a.demoSwitchCustomize(i) })
		}
		for i, name := range demo.UsageSetNames() {
			i, name := i, name
			add("chart", name, "", "切换用量页演示数据套。", func() { a.demoSwitchUsage(i) })
		}
		add("chart", "演示:用量·限额查询中", "", "右栏限额区显示「更新中…」。", func() {
			a.use.limitsState = core.LimitsChecking
			a.toast = "演示:限额查询中"
		})
		add("chart", "演示:用量·限额查询失败", "", "清空限额并模拟不可用。", func() {
			a.use.limits = nil
			a.use.limitsState = core.LimitsUnavailable
			a.toast = "演示:限额查询失败"
		})
		for _, ph := range demo.UpdatePhases() {
			ph := ph
			add("download", "演示:更新 · "+ph.Label, "", "切换状态栏、关于与 upd 弹层的更新阶段。", func() {
				a.SeedUpdateDemo(ph.ID, ph.Pct)
			})
		}
	}
	add("fork", "Git:打开提交图谱", "", "在工作区打开完整的 Git 图谱(搜索、详情、右键操作)。", a.openGraph)
	add("chart", "用量与成本", "", "打开用量整页。", func() { a.screen = ScreenUsage })
	add("clock", "自动化", "", "打开自动化页面。", func() { a.screen = ScreenAutomation })
	add("layers", "画布", "", "打开画布页面。", func() { a.screen = ScreenCanvas })
	add("globe", "浏览器", "", "打开内置浏览器页面。", func() { a.screen = ScreenBrowser })
	add("plug", "自定义", "", "插件、MCP、技能、命令、子 Agent、规则与钩子。", func() { a.screen = ScreenCustomize })
	return cmds, common
}

// palItems 按当前查询产出要显示的项(已按组排好)。
func (a *App) palItems() []palItem {
	q := strings.TrimSpace(a.pq.query)
	if a.pq.sub == "theme" {
		var out []palItem
		for _, p := range theme.Palettes {
			if q != "" && fuzzy(q, p.Name+" "+p.ID) < 0 {
				continue
			}
			out = append(out, palItem{group: "主题", icon: "palette", title: p.Name, right: themeSub[p.ID], theme: p.ID, desc: themeSub[p.ID],
				run: func() {}})
		}
		return out
	}
	cmds, common := a.palCommands()
	var cs, ss, fs []scored
	for i, cm := range cmds {
		if q == "" {
			if common[i] {
				cs = append(cs, scored{cm, 0, i})
			}
			continue
		}
		if sc := fuzzy(q, cm.title); sc >= 0 {
			cs = append(cs, scored{cm, sc, i})
		} else if descHit(q, cm.desc) {
			// 标题没命中时看说明文字(如搜「主题」找到「设置:外观」),排在标题命中之后
			cs = append(cs, scored{cm, -1000, i})
		}
	}
	n := 0
	for _, p := range a.projects {
		for _, s := range p.Sessions {
			if s.Archived {
				continue
			}
			s, p := s, p
			it := palItem{group: "会话", icon: "chat", title: s.Title, right: p.Name, desc: p.Name + " · " + s.Label,
				run: func() { a.openSession(p.ID, s.ID) }}
			if q == "" {
				capN := 3
				if a.demoWs() {
					capN = 5
				}
				if n < capN {
					ss = append(ss, scored{it, 0, n})
				}
				n++
			} else if sc := fuzzy(q, s.Title); sc >= 0 {
				ss = append(ss, scored{it, sc, n})
				n++
			}
		}
	}
	if p, line, col, ok := parsePathLine(q); ok && p == "" {
		fs = append(fs, scored{palItem{
			group: "命令", icon: "list", title: "转到第 " + itoa(line+1) + " 行",
			right: "当前文件", desc: "在当前编辑器标签跳到该行并选中。",
			run: func() { a.gotoLineInCurrent(line, col) },
		}, 1000, 0})
	}
	if q == "" {
		seen := map[string]bool{}
		paths := []string(nil)
		for _, tb := range a.ide.tabs {
			if tb.diff || seen[tb.path] {
				continue
			}
			seen[tb.path] = true
			paths = append(paths, tb.path)
		}
		if a.demoWs() {
			for _, p := range demo.PaletteExtraFiles() {
				if !seen[p] {
					seen[p] = true
					paths = append(paths, p)
				}
			}
		}
		for i, p := range paths {
			if i >= 6 {
				break
			}
			fs = append(fs, scored{a.fileItem(p), 0, i})
		}
	} else {
		for i, p := range a.ide.allFiles() {
			if sc := fuzzy(q, p); sc >= 0 {
				fs = append(fs, scored{a.fileItem(p), sc + fileNameBonus(q, p), i})
			}
		}
		if cand := a.paletteAbsFile(q); cand != nil {
			fs = append([]scored{*cand}, fs...)
		}
	}
	for _, g := range [][]scored{cs, ss, fs} {
		sort.SliceStable(g, func(i, j int) bool { return g[i].score > g[j].score })
	}
	if len(fs) > 20 {
		fs = fs[:20]
	}
	if q != "" && len(ss) > 8 { // 模糊匹配的会话容易很多,只留得分最高的几条
		ss = ss[:8]
	}
	order := [][]scored{cs, ss, fs}
	if q != "" && a.mode == ModeIDE { // IDE 里搜东西多半是找文件:文件 → 命令 → 会话
		order = [][]scored{fs, cs, ss}
	}
	var out []palItem
	for _, g := range order {
		for _, s := range g {
			out = append(out, s.palItem)
		}
	}
	return out
}

type scored struct {
	palItem
	score, ord int
}

// fileNameBonus:查询命中文件名(而不是目录)时加分。
func fileNameBonus(q, p string) int {
	name := p[strings.LastIndex(p, "/")+1:]
	if fuzzy(q, name) >= 0 {
		return 20
	}
	return 0
}

func (a *App) fileItem(p string) palItem {
	dir := ""
	if i := strings.LastIndex(p, "/"); i >= 0 {
		dir = p[:i]
	}
	name := p[strings.LastIndex(p, "/")+1:]
	q := a.pq.query
	return palItem{group: "文件", icon: fileIcon(name), title: name, right: dir, path: p, desc: p,
		run: func() {
			a.paletteOpenFile(p, q)
		}}
}

// openSession 切到某个会话(真实后端接入前只记日志并更新选中)。
func (a *App) openSession(project, id string) {
	log.Info("命令面板打开会话", "project", project, "session", id)
	a.OpenSession(id)
}

// palRowY 返回第 i 项的顶边(含前面的分组标题),与渲染保持同一套尺寸。
func palRowY(items []palItem, i int) (top, h float32) {
	var y float32 = 4
	last := ""
	for k, it := range items {
		if it.group != last {
			y += palCapH
			last = it.group
		}
		if k == i {
			return y, palRowH
		}
		y += palRowH
	}
	return y, palRowH
}

// paletteView 渲染面板(浮层)。
func (a *App) paletteView(c *ui.Context) {
	if a.paletteOn && c.Shortcut(0, ui.KeyEscape) {
		a.closePalette()
		return
	}
	if a.dlgClosing == "palette" && time.Since(a.dlgCloseAt) >= kit.DurFast {
		a.dlgClosing, a.dlgSeen = "", ""
		a.pq = palState{}
		return
	}
	if !a.paletteOn && a.dlgClosing != "palette" {
		return
	}
	t := a.tk
	if a.paletteOn && a.pq.palAt.IsZero() {
		a.pq.palAt = time.Now()
	}
	// 上下 / 回车只在面板开着时处理;输入框不吃 Up / Down
	items := a.palItems()
	if a.pq.sel >= len(items) {
		a.pq.sel = len(items) - 1
	}
	if a.pq.sel < 0 {
		a.pq.sel = 0
	}
	move := func(d int) {
		if len(items) == 0 {
			return
		}
		a.pq.sel = (a.pq.sel + d + len(items)) % len(items)
	}
	run := func(i int) {
		if i < 0 || i >= len(items) {
			return
		}
		it := items[i]
		if a.pq.sub == "theme" {
			a.pq.origPal = it.theme // 确认:关闭时不再还原
			a.setPalette(it.theme)
			log.Info("切换主题", "palette", it.theme)
			a.closePalette()
			return
		}
		it.run()
		if !it.keep {
			a.closePalette()
		}
	}
	// 主题列表里上下移动即时预览
	if a.pq.sub == "theme" && len(items) > 0 && items[a.pq.sel].theme != a.pal.ID {
		a.previewPalette(items[a.pq.sel].theme)
		t = a.tk
	}
	a.dialogOpen("palette", a.paletteOn, func(open *bool) {
		ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
			backdrop.Background(t.Scrim).Justify(ui.Start).AlignItems(ui.Center).Padding(96, 0, 0, 0)
			pw := float32(620)
			if a.winW > 0 && a.winW < palHidePrv {
				pw = min(520, a.winW-24)
				if pw < 280 {
					pw = max(280, a.winW-24)
				}
			}
			panel.Width(pw).Background(t.Raised).Radius(t.RXl()).Clip().Column()
			for _, s := range t.Shadow3() {
				panel.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
			}
			panel.Border(1, t.Bd1)
			a.dialogIn("palette", backdrop, panel)
			panel.Children(func() {
				// 输入行
				ui.Row(c).Height(44).PaddingX(14).Gap(10).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
					kit.Ic(c, "search", kit.S16).TextColor(t.Tx3)
					in := ui.TextInputBase(c.Key("pal-in"), &a.pq.query).Grow(1).FontSize(theme.FsTitle).AutoFocus().
						Placeholder(map[bool]string{true: "筛选主题…", false: "搜索命令、会话与文件"}[a.pq.sub == "theme"]).PlaceholderColor(t.Tx4)
					in.OnSubmit(func() {
						if a.paletteGotoLine(strings.TrimSpace(a.pq.query)) {
							a.closePalette()
							return
						}
						run(a.pq.sel)
					})
					in.OnShortcut(0, ui.KeyEscape, func() { a.closePalette() })
					// 单行输入框会自己吃 ↑↓(跳到行首 / 行尾),所以挂在输入框自己的快捷键上
					in.OnShortcut(0, ui.KeyDown, func() { move(1) })
					in.OnShortcut(0, ui.KeyUp, func() { move(-1) })
					in.OnShortcut(ui.Ctrl, ui.KeyN, func() { move(1) })
					in.OnShortcut(ui.Ctrl, ui.KeyP, func() { move(-1) })
				})
				ui.Row(c).Height(palListH).AlignItems(ui.Stretch).Children(func() {
					a.palList(c, items, run)
					if a.winW >= palHidePrv || a.winW <= 0 {
						a.palPreview(c, items)
					}
				})
				ui.Row(c).Height(theme.Px(28)).PaddingX(12).Gap(12).AlignItems(ui.Center).FontSize(theme.FsCap).TextColor(t.Tx3).
					BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1).Children(func() {
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() { kit.Kbd(c, "↑", "↓"); ui.Text(c, "选择") })
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() { kit.Kbd(c, "⏎"); ui.Text(c, "执行") })
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() { kit.Kbd(c, "esc"); ui.Text(c, "关闭") })
					ui.Spacer(c).Grow(1)
					ui.Text(c, "共 "+itoa(len(items))+" 项")
				})
			})
		})
	})
}

// palList 是左侧分组列表(可滚动,键盘移动时保证选中项可见)。
func (a *App) palList(c *ui.Context, items []palItem, run func(int)) {
	t := a.tk
	st := &a.pq.listSt
	if a.pq.sel != a.pq.lastSel || a.pq.query != a.pq.lastQ {
		if a.pq.query != a.pq.lastQ {
			a.pq.sel = 0
		}
		top, h := palRowY(items, a.pq.sel)
		switch {
		case top < st.Y+palCapH && a.pq.sel == 0:
			st.Y = 0
		case top < st.Y:
			st.Y = top - palCapH
		case top+h > st.Y+palListH:
			st.Y = top + h - palListH + 4
		}
		if st.Y < 0 {
			st.Y = 0
		}
		a.pq.lastSel, a.pq.lastQ = a.pq.sel, a.pq.query
	}
	kit.VScroll(c).Key("pal-list").TrackScroll(st).Grow(1).MinWidth(0).Children(func() {
		ui.Column(c).Padding(4).Children(func() {
			last := ""
			for i, it := range items {
				if it.group != last {
					last = it.group
					ui.Row(c).Height(palCapH).PaddingX(8).AlignItems(ui.Center).FontSize(theme.FsMicro).FontWeight(600).TextColor(t.Tx3).
						Children(func() { ui.Text(c, it.group) })
				}
				i, it := i, it
				row := ui.ButtonBase(c).Key("pal-" + itoa(i)).Height(palRowH).PaddingX(8).Gap(8).Radius(t.RSm()).AlignItems(ui.Center).Cursor(ui.CursorDefault)
				kit.StaggerRise(row, i, a.pq.palAt, a.frameSoon)
				switch {
				case it.off:
					row.TextColor(t.Tx4)
				case i == a.pq.sel:
					row.Background(t.AcSoft).TextColor(t.AcTx)
				case row.Hovered():
					row.Background(t.Fill1)
				}
				row.OnClick(func() {
					if it.off {
						if it.offWhy != "" {
							a.toast = it.offWhy
						}
						return
					}
					run(i)
				})
				row.Children(func() {
					if it.theme != "" {
						p := theme.Find(it.theme)
						ui.Box(c).Size(8, 8).Radius(4).Background(ui.Hex(p.Accent)).Shrink(0)
					} else if it.path != "" {
						a.fileIc(c, it.title)
					} else {
						kit.Ic(c, it.icon, kit.S12)
					}
					ui.Text(c, it.title).Grow(1).SingleLine()
					if it.theme != "" && it.theme == a.pq.origPal {
						ui.Text(c, "当前").FontSize(theme.FsCap).TextColor(t.Tx3)
					}
					if it.right != "" {
						col := t.Tx3
						if i == a.pq.sel {
							col = t.AcTx
						}
						ui.Text(c, it.right).FontSize(theme.FsCap).TextColor(col).SingleLine().Shrink(1).MinWidth(0)
					}
				})
			}
			if len(items) == 0 {
				ui.Row(c).Height(60).Justify(ui.Center).AlignItems(ui.Center).TextColor(t.Tx3).Children(func() { ui.Text(c, "没有匹配的项") })
			}
		})
	})
}

// palPreview 是右侧预览栏(sunken 底):主题列表 / 文件开头 / 命令说明。
func (a *App) palPreview(c *ui.Context, items []palItem) {
	t := a.tk
	var cur *palItem
	if a.pq.sel >= 0 && a.pq.sel < len(items) {
		cur = &items[a.pq.sel]
	}
	ui.Column(c).Width(230).Shrink(0).Padding(10, 12).Background(t.Sunken).BorderWidth(0, 0, 0, 1).BorderColor(t.Bd1).Clip().Children(func() {
		if cur == nil {
			return
		}
		showThemes := a.pq.sub == "theme" || cur.title == "主题:切换…"
		switch {
		case cur.path != "":
			ui.Row(c).Gap(6).AlignItems(ui.Center).Margin(0, 0, 4, 0).Children(func() {
				a.fileIc(c, cur.title)
				ui.Text(c, cur.title).FontWeight(600).SingleLine()
			})
			ui.Text(c, cur.path).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Margin(2, 0, 8, 0)
			for _, ln := range a.filePreview(cur.path, 16) {
				kit.Mono(c, ln, t.Tx2).FontSize(theme.Px(10.5)).SingleLine()
			}
		default:
			title := cur.title
			if a.pq.sub == "theme" {
				title = "主题:切换…"
			}
			ui.Text(c, title).FontWeight(600)
			desc := cur.desc
			if a.pq.sub == "theme" {
				desc = "展开主题列表。上下移动时即时预览,回车确认,Esc 还原。"
			}
			ui.Text(c, desc).FontSize(theme.FsCap).TextColor(t.Tx3).LineHeight(1.5).Margin(4, 0, 10, 0)
			if showThemes {
				for _, p := range theme.Palettes {
					ui.Row(c).Height(theme.Px(20)).Gap(8).AlignItems(ui.Center).FontSize(theme.FsCap).Children(func() {
						ui.Box(c).Size(8, 8).Radius(4).Background(ui.Hex(p.Accent)).Shrink(0)
						ui.Text(c, p.Name)
						ui.Text(c, themeSub[p.ID]).TextColor(t.Tx4)
					})
				}
			}
		}
	})
}

// filePreview 取文件开头 n 行(预览栏用;缓存最近一次,避免每帧读文件)。
func (a *App) filePreview(p string, n int) []string {
	if a.pq.preview != p {
		txt, err := a.ide.fs.Read(p)
		if err != nil {
			txt = "(无法预览:" + err.Error() + ")"
		}
		a.pq.preview, a.pq.previewT = p, txt
	}
	lines := strings.Split(a.pq.previewT, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	for i, l := range lines {
		lines[i] = strings.ReplaceAll(l, "\t", "    ")
	}
	return lines
}

// OpenPalette 打开命令面板并预填查询(截图 / 调试用)。
func (a *App) OpenPalette(q string) {
	a.openPalette()
	a.pq.query = q
}
