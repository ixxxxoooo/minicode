// Package app 是 MiniCode 的原生界面:一个窗口、两种形态(Agent / IDE),所有部件共用同一份会话状态与后端。
// 视图是纯函数式的(每帧从 App 的状态构建),交互一律转成 core.Command 发给后端。
package app

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui/editor"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/automation"
	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/browser"
	"github.com/ixxxxoooo/minicode/internal/canvas"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/prefs"
	"github.com/ixxxxoooo/minicode/internal/projects"
	"github.com/ixxxxoooo/minicode/internal/sessmeta"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

var log = diag.L("app")

// Mode 是窗口形态。
type Mode int

const (
	ModeAgent Mode = iota
	ModeIDE
)

func (m Mode) String() string {
	if m == ModeIDE {
		return "ide"
	}
	return "agent"
}

// Screen 是整窗显示的内容:工作台,或某个整页。
type Screen string

const (
	ScreenWork       Screen = "work"
	ScreenWelcome    Screen = "welcome"
	ScreenSettings   Screen = "settings"
	ScreenUsage      Screen = "usage"
	ScreenAutomation Screen = "automation"
	ScreenCanvas     Screen = "canvas"
	ScreenBrowser    Screen = "browser"
	ScreenCustomize  Screen = "customize"
)

// App 是整个应用的状态。只在主线程读写(后端事件经 win.Update 切回主线程)。
type App struct {
	ide        *ide    // IDE 形态独有的状态(编辑器标签、文件树、底部面板)
	ux         uxState // 拖拽引用 / Markdown 所见即所得等编辑体验的零散状态(edit_ux.go)
	edTheme    *editor.Theme
	edThemeKey string
	flowSt     ui.ScrollState // 会话流滚动状态(跟随底部)
	win        *mygo.Window

	// 会话:当前显示的会话状态与它的后端
	st *core.State
	be backend.Backend

	// 外观
	pal    theme.Palette
	sch    theme.Scheme
	accent string
	tk     *theme.Tokens

	// 窗口形态与布局
	mode         Mode
	screen       Screen
	sideOpen     bool
	rightOn      bool                                     // 右侧栏开合:Agent 形态 = 工作区,IDE 形态 = 对话栏
	chatMax      bool                                     // IDE 形态:对话栏占满主区(工作区让位)
	memBy        [2]layMem                                // 各形态各自记住的右侧栏 / 底部面板开合(切回来还是原样)
	flashAt      time.Time                                // 工作区被「打开标签」唤起的时刻(边框闪一下)
	genText      func(dir, prompt string) (string, error) // 一次性文本生成(提交说明等)
	graphTab     graphTabState                            // Git 图谱标签的搜索与选中
	out          outState                                 // 底部「输出」面板
	lspH         lspHub                                   // 语言服务会话与诊断
	pmu          sync.Mutex
	pend         []func() // 没有窗口时攒下的界面回调(测试)
	termHasFocus bool     // 终端面板此刻有键盘焦点(⌘F 查找走终端)
	sessNav      []string // Agent 形态的「后退 / 前进」:打开过的会话序列
	sessI        int
	sessWalking  bool
	ly           layoutSizes // 拖出来的栏宽 / 面板高度
	winW, winH   float32     // 窗口内容区尺寸(夹栏宽用)
	chatW        float32     // 对话栏上一帧的宽度(响应式三档按它判断)
	wsW          float32     // 工作区上一帧的宽度
	fit          autoFit     // 窗口变窄时自动收起的状态(responsive.go)
	diffPref     int         // 已应用的 Diff 默认视图偏好(0 左右 / 1 统一)
	nav          string      // chat / files / search / scm
	navBy        [2]string   // 各形态各自记住的左栏分段(切回来还停在原处)
	popup        string      // 状态栏 / 输入框弹层:cfg / ctx / upd / acct
	paletteOn    bool
	pq           palState // 命令面板状态
	setTab       string   // 设置页当前分类
	set          settingsState
	prefs        *prefs.Store // 用户偏好(~/.minicode/settings.json)
	density      int          // 界面密度:0 紧凑 / 1 标准 / 2 宽松
	radius       int          // 圆角:0 利落 / 1 适中 / 2 圆润

	// 侧栏数据
	projects []core.ProjectInfo
	curSess  string
	projOpen map[string]bool
	reg      *projects.Store     // 手动添加的项目与工作区(真实:~/.minicode/workspaces.json;演示:内存)
	ws       *projects.Workspace // 当前激活的工作区(nil = 单项目)
	repos    []*repoEntry        // 多根时每个顶层文件夹一项
	repoAct  int                 // 活动仓库(源代码管理里正在操作的那个)
	solo     *soloState          // 进入工作区前的单项目现场(退出时还原)
	pd       projDlg             // 添加项目 / 工作区对话框状态
	sessMore map[string]bool
	meta     *sessmeta.Store       // 会话的本机整理信息(置顶 / 归档 / 自定义标题)
	rawProj  []core.ProjectInfo    // 未套用整理信息的项目与会话(a.projects 是套用后的)
	sessDel  func(id string) error // 真实模式:删除会话的磁盘文件(演示为空)
	ren      sessRename            // 侧栏正在行内重命名的会话
	fo       fileOp                // 文件树正在进行的行内新建 / 重命名

	// 会话流的折叠状态:条目 id → 是否展开(没有记录就用默认)
	folds map[string]bool

	// Markdown 选项(随主题重建)
	md *markdown.Options

	// 会话切换:由入口注入,按 id 造后端(空串 = 新会话)
	opener func(id string) backend.Backend

	// 输入框
	draft         string
	sr            searchState                                                // 项目内搜索
	gi            map[string]*gitInfo                                        // 文件的追溯 / 历史缓存
	autoRun       func(it automation.Automation, dir string) (string, error) // 自动化运行器(nil = 演示:只记录)
	autoSeen      time.Time                                                  // 上次检查定时的时间
	autoBusy      map[int]bool                                               // 正在运行的自动化
	live          map[string]*liveSess                                       // 切走时还在跑的会话:后端继续保活,事件归约到自己的状态
	framePending  bool                                                       // frameSoon 已排队
	foldAt        map[string]time.Time                                       // 步骤行展开的时刻(入场动画)
	dlgSeen       string                                                     // 已开始计时入场动画的居中浮层
	dlgAt         time.Time
	popSeen       string                         // 已开始计时入场动画的弹层
	popAt         time.Time                      // 该弹层打开的时刻
	redrawPending bool                           // redrawSoon 已排了定时重画
	pickOnReady   bool                           // 新会话握手后应用已记住的模型 / 思考强度
	cf            map[string]*confState          // 合并冲突标签的状态(按文件)
	sc            scmState                       // 源代码管理
	auto          *automation.Store              // 自动化清单
	au            autoState                      // 自动化页状态
	clock         func() time.Time               // 当前时间(测试 / 截图注入固定时间)
	canv          *canvas.Store                  // 画布文档(首次打开页面时建立)
	cv            canvasState                    // 画布页状态
	br            *browser.Browser               // 内置浏览器(首次打开页面时建立)
	brs           browserState                   // 浏览器页状态
	cust          *customize.Store               // 自定义页数据(首次打开页面时建立)
	cu            custState                      // 自定义页状态
	cuNoMem       bool                           // 自定义页:「加载 CLAUDE.md 记忆文件」被关掉(只是界面状态,尚未接入 env)
	use           usageState                     // 用量页
	usageSrc      func(rng int) core.UsageReport // 用量数据源(nil = 演示数据)
	qp            *qpState                       // 快速选择浮层(历史会话 / 分支),nil = 关闭
	repo          vcs.Repo                       // 当前项目的仓库(演示为内存实现)
	branch        string                         // 当前分支名(缓存,避免每帧查 git)
	copyText      string                         // 待写入剪贴板的文字(需要 ui.Context 才能写,下一帧消费)
	toast         string                         // 待显示的一条轻提示(渲染时消费)
	welcomeFocus  bool                           // 欢迎页的输入框自动聚焦
	cardHov       bool                           // 当前这张工具卡是否被悬停(卡片内容里的悬停按钮用)
	hist          histState                      // 历史会话弹层的搜索词
	rv            reviewState                    // 审查标签状态
	cw            *commitWizard                  // 「逐个提交」向导(nil = 没有进行中的)
	taskSt        ui.ScrollState                 // 任务输出标签的滚动
	taskNoFollow  bool                           // 任务输出:关闭「跟随输出」
	q             promptQueue                    // 排队的消息
	qa            map[string]*qaState            // 提问向导的本地进度

	// 测试 / 截图用:没有系统红绿灯时自己画
	fakeLights bool
	devBadge   string
	proj       string // 真实项目目录(演示模式为空)
	demoExtra  bool   // 演示:工作区里还有一个本会话没碰过的未提交文件
	update     string // 状态栏更新提示文案(空 = 不显示)
}

// Options 是创建应用的参数。
type Options struct {
	Palette, Scheme, Accent string
	Mode                    Mode
	Screen                  Screen
	FakeLights              bool
	Prefs                   *prefs.Store // 空 = 只在内存
	DevBadge                string
}

// New 创建应用状态(不开窗口)。
func New(o Options) *App {
	a := &App{
		auto:       automation.NewStore(nil),
		st:         core.NewState(),
		pal:        theme.Find(o.Palette),
		sch:        theme.FindScheme(o.Scheme),
		accent:     o.Accent,
		mode:       o.Mode,
		screen:     o.Screen,
		sideOpen:   true,
		nav:        "",
		projOpen:   map[string]bool{},
		sessMore:   map[string]bool{},
		folds:      map[string]bool{},
		foldAt:     map[string]time.Time{},
		ide:        newIDE(),
		prefs:      o.Prefs,
		density:    1,
		radius:     1,
		fakeLights: o.FakeLights,
		devBadge:   o.DevBadge,
	}
	if a.screen == "" {
		a.screen = ScreenWork
	}
	// 默认开合:Agent = 工作区收起、底部面板收起;IDE = 对话栏展开、底部面板展开
	a.memBy = [2]layMem{{right: false, panel: false}, {right: true, panel: true}}
	a.rightOn, a.ide.bottomOpen = a.memBy[a.mode].right, a.memBy[a.mode].panel
	if a.prefs == nil {
		a.prefs = prefs.Memory()
	}
	a.loadLayout()
	if o.Mode == ModeAgent && a.setTab == "" {
		a.setTab = "appearance"
	}
	a.applyPrefs()
	a.ide.onSaved = a.lspSaved
	a.ide.beforeSave = a.beforeSave
	return a
}

// retheme 重新解析令牌(换主题 / 方案 / 强调色后调用)。
func (a *App) retheme() {
	sch := a.sch
	sch.RK *= [3]float32{.55, 1, 1.35}[a.radius] // 圆角倍率
	sch.RowH += float32(a.density-1) * 2         // 行高 ±2
	a.tk = theme.Resolve(a.pal, sch, a.accent)
	kit.Use(a.tk)
	a.md = nil
	a.ux.rt = nil
	log.Debug("主题", "palette", a.pal.ID, "scheme", a.sch.ID)
}

// Attach 把后端接到应用上:事件经窗口的 Update 切回主线程再归约。
// win 为 nil 时(测试)直接在调用线程归约。
func (a *App) Attach(be backend.Backend) {
	a.be = be
	be.Start(func(ev core.Event) {
		apply := func() {
			if a.be != be {
				a.applyParked(be, ev) // 后台保活的会话:事件归约到它自己的状态;其余旧后端的迟到事件丢弃
				return
			}
			end := diag.SpanSlow("app", "归约事件", 8*time.Millisecond, "ev", evName(ev))
			a.st.Apply(ev)
			a.afterEvent(ev)
			end()
		}
		if a.win != nil {
			a.win.Update(apply)
		} else {
			apply()
		}
	})
}

// afterEvent 归约之后的界面侧反应:回合结束时处理排队。
func (a *App) afterEvent(ev core.Event) {
	// 新会话的 id 在握手后才知道:记下来,侧栏就能高亮它(演示后端的占位 id 不记)
	if _, ok := ev.(core.EvReady); ok && a.pickOnReady {
		a.pickOnReady = false
		a.applySavedPicks()
	}
	if e, ok := ev.(core.EvReady); ok && a.curSess == "" && e.SessionID != "" && !strings.HasPrefix(e.SessionID, "demo-") {
		a.curSess = e.SessionID
	}
	if e, ok := ev.(core.EvTurnEnded); ok {
		a.q.turnEnded(e.Reason == core.StopEndTurn)
		a.drainQueue()
	}
}

// SetRepo 设置当前项目的仓库;nil 表示不是 Git 仓库(分支显示为空)。
func (a *App) SetRepo(r vcs.Repo) {
	a.repo = r
	a.branch = ""
	if r != nil {
		a.branch = r.Current()
	}
}

// BranchName 当前分支名。
func (a *App) BranchName() string { return a.branch }

// SetSessionOpener 注入「按会话 id 造后端」的工厂(演示 / 真实各一个)。
func (a *App) SetSessionOpener(f func(id string) backend.Backend) { a.opener = f }

// OpenSession 切到某个会话:关掉旧后端、清空状态与排队、接上新后端(空 id = 新会话)。
func (a *App) OpenSession(id string) {
	log.Info("切换会话", "from", a.curSess, "to", id)
	a.maybeLeaveWorkspace(id)
	a.recordSess(id)
	a.parkOrClose()
	if lv := a.live[id]; lv != nil && id != "" { // 回到一个在后台跑着的会话:接回它的后端与状态
		delete(a.live, id)
		a.be, a.st = lv.be, lv.st
		a.q.clear()
		a.qa = nil
		a.folds = map[string]bool{}
		a.draft = ""
		a.curSess = id
		a.screen = ScreenWork
		a.flowSt = ui.ScrollState{}
		log.Info("接回后台会话", "id", id)
		return
	}
	a.st = core.NewState()
	a.q.clear()
	a.qa = nil
	a.folds = map[string]bool{}
	a.draft = ""
	a.curSess = id
	a.screen = ScreenWork
	a.flowSt = ui.ScrollState{}
	a.pickOnReady = id == "" || strings.HasPrefix(id, "new:") // 新会话沿用上次选的模型与思考强度
	if a.opener != nil {
		a.Attach(a.opener(id))
	}
	if strings.HasPrefix(id, "new:") { // 项目内新会话:对界面来说仍是「还没有 id 的新会话」
		a.curSess = ""
	}
}

// NewSessionIn 在某个项目里开新会话(侧栏项目行的「+」)。
func (a *App) NewSessionIn(p core.ProjectInfo) {
	a.projOpen[p.ID] = true
	a.OpenSession("new:" + p.Path)
}

// QueuePrompts 直接排入若干条消息(截图 / 测试用)。
func (a *App) QueuePrompts(texts ...string) {
	for _, t := range texts {
		a.q.add(t)
	}
}

// NewSession 开始新会话并回到欢迎页。
func (a *App) NewSession() { a.OpenSession("") }

// SetWindow 记下窗口(用于 Update 与窗口控制)。
func (a *App) SetWindow(w *mygo.Window) { a.win = w }

// State 返回当前会话状态(测试用)。
func (a *App) State() *core.State { return a.st }

// send 把界面命令交给后端。
func (a *App) send(cmd core.Command) {
	if a.be == nil {
		return
	}
	a.be.Send(cmd)
}

// NavID 当前左栏分段:Agent 默认「会话」,IDE 默认「文件」。
func (a *App) NavID() string {
	if a.nav != "" {
		return a.nav
	}
	if a.mode == ModeIDE {
		return "files"
	}
	return "chat"
}

// SetMode 切换形态。
func (a *App) SetMode(m Mode) {
	if a.mode == m {
		return
	}
	log.Info("切换形态", "from", a.mode, "to", m)
	a.navBy[a.mode] = a.nav
	a.memBy[a.mode] = layMem{right: a.rightOn, panel: a.ide.bottomOpen}
	a.mode = m
	a.nav = a.navBy[m]
	a.rightOn, a.ide.bottomOpen = a.memBy[m].right, a.memBy[m].panel
	a.chatMax = false
}

// layMem 是一个形态记住的开合。
type layMem struct{ right, panel bool }

// View 是窗口的根视图。
func (a *App) View(c *ui.Context) {
	defer diag.SpanSlow("app", "构建界面", 33*time.Millisecond)()
	c.SetTheme(a.tk.UITheme())
	a.winW, a.winH = c.Size()
	a.fitPanels()
	a.reposTick()
	a.shortcuts(c)
	if a.popup == "" {
		a.popSeen = ""
	}
	if !a.paletteOn && a.qp == nil && a.pd.kind == "" {
		a.dlgSeen = ""
	}
	a.autosave()
	a.root(c)
}

func evName(ev core.Event) string {
	switch ev.(type) {
	case core.EvTextDelta, *core.EvTextDelta:
		return "TextDelta"
	case core.EvAppend, *core.EvAppend:
		return "Append"
	case core.EvToolUpdate, *core.EvToolUpdate:
		return "ToolUpdate"
	}
	return "other"
}

// SetClock 注入「当前时间」(测试与截图用,让相对时间可复现)。
func (a *App) SetClock(f func() time.Time) { a.clock = f }

func (a *App) now() time.Time {
	if a.clock != nil {
		return a.clock()
	}
	return time.Now()
}

// SeedDemoExtras 演示 / 截图用:补上设计稿里工作区的附加状态(额外未提交文件、更新提示)。
func (a *App) SeedDemoExtras() {
	a.demoExtra = true
	a.auto = automation.NewStore(demo.Automations(a.now()))
	a.au.sel = 2 // 设计稿里选中的是第二张卡
	a.ide.seedDemo()
	a.seedDemoRegistry()
	a.seedWorkspace()
	a.update = "更新 42%"
}

// ScrollFlowTop 把会话流滚到最上面(截图与设计稿对照用:设计稿的流停在顶部)。
func (a *App) ScrollFlowTop() {
	a.flowSt.Y = 0
	a.flowSt.MaxY = 1e9 // 让「停在底部才跟随」的判断先不成立,下一帧会按真实高度重算
}

// SetPopup 直接打开某个弹层(截图 / 调试用:cfg / sbcfg / ctx / upd / acct)。
func (a *App) SetPopup(k string) { a.popup = k }

// shortcuts 是窗口级快捷键(与设计稿一致):⌘B 导航栏、⌥⌘B 右侧栏、⌘E 切换形态、⌘K 命令面板、
// ⌘J 底部面板、⌘S 保存、⌘, 设置。输入框里的 Enter 等由各自控件处理,这里只放带 ⌘ 的组合键。
func (a *App) shortcuts(c *ui.Context) {
	if a.copyText != "" {
		c.WriteClipboard(a.copyText)
		a.copyText = ""
	}
	if a.termShortcuts(c) {
		return
	}
	if a.editorShortcuts(c) {
		return
	}
	switch {
	case c.Shortcut(ui.Cmd, ui.KeyB):
		a.sideOpen = !a.sideOpen
	case c.Shortcut(ui.Cmd|ui.Alt, ui.KeyB):
		a.toggleRight()
	case c.Shortcut(ui.Cmd, ui.KeyE):
		if a.mode == ModeAgent {
			a.SetMode(ModeIDE)
		} else {
			a.SetMode(ModeAgent)
		}
	case c.Shortcut(ui.Cmd, ui.KeyK):
		if a.paletteOn {
			a.closePalette()
		} else {
			a.openPalette()
		}
	case c.Shortcut(ui.Cmd, ui.KeyN):
		a.NewSession()
	case c.Shortcut(ui.Cmd|ui.Shift, ui.KeyF):
		a.OpenSearch()
	case c.Shortcut(ui.Ctrl|ui.Shift, ui.KeyG):
		a.OpenSCM()
	case c.Shortcut(ui.Cmd|ui.Shift, ui.KeyO):
		a.OpenQuick("symbols", "")
	case c.Shortcut(ui.Cmd, ui.KeyJ):
		a.ide.bottomOpen = !a.ide.bottomOpen
	case c.Shortcut(ui.Cmd|ui.Shift, ui.KeyJ):
		a.togglePanelMax()
	case c.Shortcut(ui.Cmd|ui.Shift, ui.KeyM):
		a.toggleChatMax()
	case a.chatMax && !a.paletteOn && a.qp == nil && a.popup == "" && c.Shortcut(0, ui.KeyEscape):
		a.chatMax = false // Esc:退出「对话占满」
	case c.Shortcut(ui.Ctrl, ui.KeyMinus):
		a.navGo(-1)
	case c.Shortcut(ui.Ctrl|ui.Shift, ui.KeyMinus):
		a.navGo(1)
	case c.Shortcut(ui.Cmd, ui.KeyS):
		if a.mode == ModeIDE {
			if err := a.ide.save(); err != nil {
				log.Error("保存失败", "err", err)
			}
		}
	case c.Shortcut(ui.Cmd, ui.KeyComma):
		a.openSettings(a.setTab)
	case a.screen == ScreenSettings && !a.paletteOn && a.set.query == "" && c.Shortcut(0, ui.KeyEscape):
		a.closeSettings()
	}
}

// 模型与思考强度的选择写进偏好,新会话沿用(历史会话保持它自己的)。
const (
	prefModel  = "claude.model"
	prefEffort = "claude.effort"
)

func (a *App) pickModel(id string) {
	a.send(core.CmdSetModel{Model: id})
	a.st.Model = id
	a.prefs.Set(prefModel, id)
}

func (a *App) pickEffort(e core.Effort) {
	a.send(core.CmdSetEffort{Effort: e})
	a.st.Effort = e
	a.prefs.Set(prefEffort, int(e))
}

// applySavedPicks 把记住的模型 / 强度发给刚握手的新会话(模型要在列表里才用)。
func (a *App) applySavedPicks() {
	if m := a.prefs.Str(prefModel, ""); m != "" && m != a.st.Model {
		for _, mi := range a.st.Models {
			if mi.ID == m {
				a.send(core.CmdSetModel{Model: m})
				a.st.Model = m
				break
			}
		}
	}
	if a.prefs.Has(prefEffort) {
		if e := core.Effort(a.prefs.Int(prefEffort, int(core.EffortMedium))); e != a.st.Effort && e >= core.EffortLow && e <= core.EffortMax {
			a.send(core.CmdSetEffort{Effort: e})
			a.st.Effort = e
		}
	}
}

// liveSess 是被切到后台、回合仍在进行的会话。
type liveSess struct {
	be backend.Backend
	st *core.State
	id string
}

// parkOrClose 离开当前会话时:回合正在进行的后端留在后台继续跑(完成后自动结束),其余直接结束。
func (a *App) parkOrClose() {
	be := a.be
	if be == nil {
		return
	}
	a.be = nil
	if a.curSess != "" && a.st != nil && a.st.Phase == core.PhaseRunning && be.Name() != "demo" {
		if a.live == nil {
			a.live = map[string]*liveSess{}
		}
		if old := a.live[a.curSess]; old != nil && old.be != be {
			old.be.Close()
		}
		a.live[a.curSess] = &liveSess{be: be, st: a.st, id: a.curSess}
		log.Info("会话转入后台继续运行", "id", a.curSess)
		return
	}
	be.Close()
}

// applyParked 把后台会话的事件归约到它自己的状态;回合结束后结束它的后端(历史已落盘,再打开时回放)。
func (a *App) applyParked(be backend.Backend, ev core.Event) {
	for id, lv := range a.live {
		if lv.be != be {
			continue
		}
		lv.st.Apply(ev)
		if lv.st.Phase != core.PhaseRunning {
			if _, ok := ev.(core.EvTurnEnded); ok {
				log.Info("后台会话已结束,释放后端", "id", id)
				delete(a.live, id)
				be.Close()
			}
		}
		return
	}
}

// BackgroundSessions 返回正在后台运行的会话 id(测试 / 侧栏提示用)。
func (a *App) BackgroundSessions() []string {
	var ids []string
	for id := range a.live {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// closeBackends 退出前结束当前与后台会话的 claude 进程。
func (a *App) closeBackends() {
	if a.be != nil {
		a.be.Close()
	}
	for _, lv := range a.live {
		lv.be.Close()
	}
	a.live = nil
}

// SetAutomationRunner 注入自动化运行器:在项目目录里跑提示,返回输出摘要或错误。
func (a *App) SetAutomationRunner(f func(it automation.Automation, dir string) (string, error)) {
	a.autoRun = f
}

// SetAutomations 换上持久化的自动化存储。
func (a *App) SetAutomations(s *automation.Store) { a.auto = s }

// runAutomation 立即运行一条自动化(「立即运行」按钮与定时到点都走这里);没有运行器时只记一条演示记录。
func (a *App) runAutomation(it automation.Automation) {
	if a.autoRun == nil {
		a.auto.RunNow(it.ID, a.now())
		a.toast = "已开始运行「" + it.Name + "」"
		return
	}
	if a.autoBusy == nil {
		a.autoBusy = map[int]bool{}
	}
	if a.autoBusy[it.ID] {
		a.toast = "「" + it.Name + "」正在运行"
		return
	}
	dir := a.projectDir(it.Project)
	if dir == "" {
		a.auto.Record(it.ID, automation.Run{At: a.now(), Err: "找不到项目「" + it.Project + "」的目录"})
		a.toast = "运行失败:找不到项目目录"
		return
	}
	a.autoBusy[it.ID] = true
	a.toast = "已开始运行「" + it.Name + "」"
	run := a.autoRun
	a.bg("automation", func() func() {
		out, err := run(it, dir)
		r := automation.Run{At: time.Now(), OK: err == nil}
		if err != nil {
			r.Err = err.Error()
		}
		if rs := []rune(out); len(rs) > 600 {
			out = string(rs[len(rs)-600:])
		}
		r.Output = out
		return func() {
			delete(a.autoBusy, it.ID)
			a.auto.Record(it.ID, r)
			if r.OK {
				a.toast = "「" + it.Name + "」运行完成"
			} else {
				a.toast = "「" + it.Name + "」运行失败:" + r.Err
			}
		}
	})
}

// projectDir 按项目名找目录(侧栏项目列表里的路径;找不到返回空)。
func (a *App) projectDir(name string) string {
	for _, p := range a.projects {
		if p.Name == name {
			return p.Path
		}
	}
	return ""
}

// AutomationTick 检查有没有到点的自动化并运行(主线程调用;main 里每半分钟一次)。
func (a *App) AutomationTick(now time.Time) {
	from := a.autoSeen
	a.autoSeen = now
	if from.IsZero() || a.autoRun == nil {
		return
	}
	for _, it := range a.auto.Due(from, now) {
		log.Info("自动化到点", "id", it.ID, "name", it.Name)
		a.runAutomation(it)
	}
}
