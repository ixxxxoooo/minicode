// Package app 是 MiniCode 的原生界面:一个窗口、两种形态(Agent / IDE),所有部件共用同一份会话状态与后端。
// 视图是纯函数式的(每帧从 App 的状态构建),交互一律转成 core.Command 发给后端。
package app

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui/editor"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/egoist/mygo/ui/richtext"
	"github.com/ixxxxoooo/minicode/internal/automation"
	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/browser"
	"github.com/ixxxxoooo/minicode/internal/canvas"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/customize"
	"github.com/ixxxxoooo/minicode/internal/demo"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/lsp"
	"github.com/ixxxxoooo/minicode/internal/prefs"
	"github.com/ixxxxoooo/minicode/internal/projects"
	"github.com/ixxxxoooo/minicode/internal/sessmeta"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/update"
	"github.com/ixxxxoooo/minicode/internal/vcs"
	"github.com/ixxxxoooo/minicode/internal/workspace"
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
	diffForce  map[string]bool // 已确认「仍然显示」的超大差异标签 id(会话内有效)
	ide        *ide            // IDE 形态独有的状态(编辑器标签、文件树、底部面板)
	ux         uxState         // 拖拽引用 / Markdown 所见即所得等编辑体验的零散状态(edit_ux.go)
	edTheme    *editor.Theme
	edThemeKey string
	brTheme    *editor.Theme // 带括号深度色的编辑器配色(ide_brackets.go)
	brThemeKey string
	bottomPrev bottomSig // 上一帧的底部面板状态(ws_tabs.go revealOnBottomChange)
	bottomSeen bool
	csvTh      *editor.Theme // csv / tsv 彩虹列配色(ide_csv_color.go)
	csvThKey   string
	flowSt     ui.ScrollState // 会话流滚动状态(与虚拟列表同步,供「回到最新」等读取)
	flowList   ui.ListState   // 会话流虚拟列表(只构建视口内的行,见 flow_rows.go)
	flowBox    ui.Rect        // 会话流列表上一帧的矩形(吸顶判断用)
	flowPerf   flowPerf       // 会话流行构建统计(性能日志)
	win        *mygo.Window

	// 会话:当前显示的会话状态与它的后端
	st   *core.State
	be   backend.Backend
	warm warmCtl // 连接预热与空闲释放的状态(见 warm.go)

	// 外观
	pal            theme.Palette
	sch            theme.Scheme
	accent         string
	tk             *theme.Tokens
	rtOpen         bool                   // 状态行是否展开阶段清单(回合结束丢弃)
	stRunning      bool                   // 上一帧回合是否在进行(用来捕捉「刚结束」)
	stEnd          time.Time              // 回合结束的时刻;状态行据此淡出并收折
	frameInPending bool                   // frameIn 的合并标记
	sessUnread     map[string]bool        // 后台跑完、还没打开看过的会话(侧栏行首小圆点)
	pvEnvPending   bool                   // 回合进行中切换了服务商 / 模型,环境还没推给后端
	mcpOn          map[string]mcpOverride // 刚点过的 MCP 开关:先在界面上生效,等后端状态回来再对账(此前要点刷新才变)
	rtPop          string                 // 脚注瀑布弹卡对应的回合(用户消息 id)

	// 窗口形态与布局
	mode         Mode
	screen       Screen
	sideOpen     bool
	rightOn      bool                                     // 右侧栏开合:Agent 形态 = 工作区,IDE 形态 = 对话栏
	chatMax      bool                                     // IDE 形态:对话栏占满主区(工作区让位)
	memBy        [2]layMem                                // 各形态各自记住的右侧栏 / 底栏 / 导航栏 / 对话占满
	secondary    bool                                     // 从属新窗口进程:不写偏好与工作区存储
	runDemo      bool                                     // 演示模式(不切换真实磁盘项目)
	morphUntil   time.Time                                // 形态切换期间锁定指针(设计稿 .morphing)
	morphAt      time.Time                                // 形态切换起点(内容分轨淡入 / 淡出)
	morphFrom    Mode                                     // 切换前的形态
	belt0        beltGeom                                 // 切换那一刻记下的旧几何(传送带平移的起点,见 morph_belt.go)
	belt0Ok      bool                                     // belt0 有效
	belt         beltFrame                                // 本帧传送带摆位(只在 stage 构建期间有值)
	stageW       float32                                  // 舞台宽度(上一帧量到的)
	fWs          float32                                  // Agent 右侧工作区开合因子 0..1(逐帧动画值,stage 每帧先算好)
	fDock        float32                                  // IDE 对话栏停靠开合因子 0..1
	fMax         float32                                  // IDE 对话栏最大化因子 0..1
	sideCur      float32                                  // 导航栏本帧的布局宽度(含开合动画;<0 = 还没量过),右栏宽度按它实时预判
	sideCurOk    bool                                     // sideCur 有效
	navSwapKey   string                                   // 左栏内容上一帧的「形态 + 分段」(变了就播换内容入场)
	navSwapAt    time.Time                                // 左栏内容换内容入场的起点
	navDropKey   string                                   // 左栏上一帧的「分段 + 形态 + 当前项目」(变了就播逐行落下,见 side_drop.go)
	navDropAt    time.Time                                // 逐行落下的起点
	navDropIdx   int                                      // 本帧已画的行序号(决定每行推迟多久)
	navDropRate  float64                                  // 本帧的落下时长倍数(设置「列表落下速度」;0 = 不播)
	slotLife     map[string]*slotLife                     // 各收起 / 展开槽位的活跃状态(收起动画期间继续画内容)
	wsFloat      bool                                     // Agent 窄窗:工作区浮层打开
	dockFloat    bool                                     // IDE 窄窗:对话栏浮层打开
	sideDrawer   bool                                     // <720:导航栏以抽屉展示
	flashAt      time.Time                                // 工作区被「打开标签」唤起的时刻(边框闪一下)
	panelFlashAt time.Time                                // 底部终端面板展开 / 新建时的边框闪一下
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
	splitBox     ui.Rect     // 编辑器拆分区上一帧的矩形(拖分隔线按它换算比例)
	fit          autoFit     // 窗口变窄时自动收起的状态(responsive.go)
	diffPref     int         // 已应用的 Diff 默认视图偏好(0 左右 / 1 统一)
	nav          string      // chat / files / search / scm
	navBy        [2]string   // 各形态最后停留的左栏分段,只用于重启后恢复;切形态不读它(一律回默认:Agent 会话 / IDE 文件)
	popup        string      // 状态栏 / 输入框弹层:cfg / ctx / upd / acct
	paletteOn    bool
	pq           palState // 命令面板状态
	setTab       string   // 设置页当前分类
	set          settingsState
	prefs        *prefs.Store // 用户偏好(~/.minicode/settings.json)
	density      int          // 界面密度:0 紧凑 / 1 标准 / 2 宽松
	radius       int          // 圆角:0 利落 / 1 适中 / 2 圆润

	// 侧栏数据
	projects   []core.ProjectInfo
	curSess    string
	freshSess  string // 本次运行里新开的会话在握手后拿到的 id:还没有条目时仍显示居中的欢迎页(session_load.go showChatWelcome)
	projOpen   map[string]bool
	reg        *projects.Store     // 手动添加的项目与工作区(真实:~/.minicode/workspaces.json;演示:内存)
	ws         *projects.Workspace // 当前激活的工作区(nil = 单项目)
	repos      []*repoEntry        // 多根时每个顶层文件夹一项
	repoAct    int                 // 活动仓库(源代码管理里正在操作的那个)
	solo       *soloState          // 进入工作区前的单项目现场(退出时还原)
	pd         projDlg             // 添加项目 / 工作区对话框状态
	treeClip   fileTreeClip        // 文件树剪贴板(tree_menus.go)
	treeAsk    treeClaudeDlg       // 文件树「让 Claude…」小浮层
	wsPerm     wsPermPending       // 工作区附加目录权限卡
	sessMore   map[string]bool
	meta       *sessmeta.Store       // 会话的本机整理信息(置顶 / 归档 / 自定义标题)
	rawProj    []core.ProjectInfo    // 未套用整理信息的项目与会话(a.projects 是套用后的)
	sessDel    func(id string) error // 真实模式:删除会话的磁盘文件(演示为空)
	ren        sessRename            // 侧栏正在行内重命名的会话
	fo         fileOp                // 文件树正在进行的行内新建 / 重命名
	termRen    termRen               // 终端标签行内重命名
	filesFocus bool                  // 文件树正在接键盘(F2 / ⌘⌫)
	sg         *suggestSt            // 行内补全(ide_suggest.go)
	ns         nextStepsState        // 回答后输入框上方的下一步建议(nextsteps.go)
	inl        *inlinePatch          // 行内编辑预览(ide_inline.go)
	claudeJob  context.CancelFunc    // 取消进行中的 Haiku 行内请求(ide_claude_edit.go)
	codeActs   []lsp.CodeAction      // ⌘. 拉取的 LSP 代码操作

	// 会话流的折叠状态:条目 id → 是否展开(没有记录就用默认)
	folds          map[string]bool
	thinkMq        map[string]thinkMqSt
	thinkMqHide    map[string]time.Time
	thinkWasLive   map[string]bool
	outFold        map[string]bool        // 长输出 / 用户消息展开
	hlKey          string                 // 命令 / 输出高亮缓存对应的「主题|强调色|字号」
	hlCache        map[string][]ui.Span   // 命令 / 输出行高亮结果(flow_hl.go)
	refHlKey       string                 // 引用面板语法色缓存键(主题)
	refHlCache     map[string][]ui.Span   // path+行内容 → 着色 span(ide_refs_hl.go)
	mdCodeKey      string                 // Markdown 围栏代码块着色缓存键(主题|强调色|字号)
	mdCodeHl       map[string][][]ui.Span // 语言+代码 → 逐行着色 span(md_code_hl.go)
	mdCell         *markdown.Options      // 预览表格单元格用的 Markdown 选项(edit_md_cell.go)
	mdCellKey      string
	foldSt         map[string]*ui.ScrollState // 命令卡 / 代码块输出的内部滚动位置(按 foldKey,必须跨帧保留)
	tick           map[string]*tickState      // 「最新一步」行的翻滚动画状态(按行键,flow_group.go)
	runWas         map[string]bool            // 精简摘要行上一帧是否在运行(结束时 120 ms 淡入换句)
	fadeAt         map[string]time.Time       // 精简摘要行结束换句的淡入起点
	termLive       map[string]string          // 运行中命令 output_file 尾读缓存
	planOpened     map[string]bool            // 已因流式自动打开过的计划
	userPinned     []pinnedUserEdit           // 悬浮编辑中的用户消息(可多条)
	userAnchor     map[string]userMsgAnchor   // 用户消息卡上一帧位置(悬浮编辑定位)
	lb             *lightboxState             // 图片灯箱
	flowNoFollow   bool                       // 会话流:用户上滑后停止贴底
	flowAwayLen    int                        // 离开底部时的末尾标记(flowTailMark),变了 = 回到最新按钮出现「有新内容」圆点
	flowScroll     map[string]flowScrollMem   // 各会话离开时的滚动位置(切换回来恢复)
	flowRiseSess   string                     // 正在播切换入场的会话 id
	flowRiseAt     time.Time                  // 切换入场起点(flow_rise.go)
	flowRisePlayed map[string]bool            // 本会话内已播过切换入场(同一会话不重复)
	projRealOf     string                     // projReal 对应的项目根
	projReal       string                     // 项目根解开软链后的路径(与 proj 相同则为空)
	dockH          float32                    // 输入框区域(任务坞 + 排队卡 + 输入框)上一帧高度,「回到最新」按钮据此抬高

	// Markdown 选项(随主题重建)
	md         *markdown.Options
	mdThemeKey string
	buildCtx   *ui.Context       // 本帧的构建上下文(只在 View 调用期间有效,回调里不能用)
	pathRefs   pathRefCache      // 会话里行内代码写的路径是否真实存在(path_ref.go)
	planMd     *markdown.Options // 计划小卡概要用的小号 Markdown 选项
	planMdKey  string
	thinkMd    *markdown.Options // 思考正文用的 Markdown 选项(说明字号、次要灰)
	thinkMdKey string

	// 会话切换:由入口注入,按 id 造后端(空串 = 新会话)
	opener func(id string) backend.Backend

	// 输入框
	draft           string
	sr              searchState                                                // 项目内搜索
	gi              map[string]*gitInfo                                        // 文件的追溯 / 历史缓存
	autoRun         func(it automation.Automation, dir string) (string, error) // 自动化运行器(nil = 演示:只记录)
	autoRunCtx      func(ctx context.Context, it automation.Automation, dir string, onChunk func(string)) (string, error)
	autoSeen        time.Time             // 上次检查定时的时间
	autoBusy        map[int]bool          // 正在运行的自动化
	autoLive        map[int]*autoLive     // 运行中输出的累积与取消
	live            map[string]*liveSess  // 切走时还在跑的会话:后端继续保活,事件归约到自己的状态
	faults          map[string]*viewFault // 界面构建出错被隔离的区块(safe_view.go)
	guardAlways     bool                  // 测试里也拦截界面 panic(默认测试不吞,免得藏缺陷)
	blinkPending    bool                  // 终端光标闪烁:已排好下一次切换的重绘
	pr              animProbe             // 布局动画探针(anim_probe.go)
	perfLastView    time.Time             // 上一帧 View 开始时刻(帧间隔)
	svc             ui.Services           // 本窗口的持久服务(每帧刷新),frameSoon 用它请求下一帧
	driveFrames     bool                  // 无窗口时也让 frameSoon 出帧(cmd/shot 逐帧追踪与动画测试)
	extHold         bool                  // 窗口还没出来:系统要求打开的文件先排队
	extQueue        []string              // 排队中的文件(窗口出来后依次处理)
	foldAt          map[string]time.Time  // 步骤行展开的时刻(入场动画)
	dlgSeen         string                // 已开始计时入场动画的居中浮层
	motionCatOpen   bool                  // 动效总览弹层
	motionReplayKey string
	motionReplayAt  time.Time
	dlgAt           time.Time
	popSeen         string    // 已开始计时入场动画的弹层
	popAt           time.Time // 该弹层打开的时刻
	popClosing      string    // 正在播退场动画的弹层
	popCloseAt      time.Time
	dlgClosing      string
	dlgCloseAt      time.Time
	wsOvlAt         time.Time                      // 工作区浮层滑入
	sideOvlAt       time.Time                      // 侧栏抽屉滑入
	dockOvlAt       time.Time                      // 对话栏浮层滑入
	projScan        bool                           // 侧栏正在重扫会话索引(显示骨架)
	projRefresh     func() []core.ProjectInfo      // 真实模式:立即重扫 ~/.claude/projects
	delUndo         kit.UndoSlot                   // 全局删除撤销(会话 / 自定义等)
	sessHide        map[string]time.Time           // 折叠删除中的会话行
	todoDockOpen    bool                           // 输入框上方的待办坞是否展开(见 todo_dock.go)
	sessDelWait     map[string]func()              // 延迟落盘的取消函数
	redrawPending   bool                           // redrawSoon 已排了定时重画
	autoLay         int                            // 上次自动落盘的各形态开合(变了才写)
	autoRtAt        time.Time                      // 上次自动落盘运行状态的时刻
	autoRtSig       string                         // 上次自动落盘时的运行状态签名
	pickOnReady     bool                           // 新会话握手后应用已记住的模型 / 思考强度
	cf              map[string]*confState          // 合并冲突标签的状态(按文件)
	sc              scmState                       // 源代码管理
	headC           headCache                      // 文件在 HEAD 的内容缓存(head_cache.go)
	auto            *automation.Store              // 自动化清单
	au              autoState                      // 自动化页状态
	clock           func() time.Time               // 当前时间(测试 / 截图注入固定时间)
	canv            *canvas.Store                  // 画布文档(首次打开页面时建立)
	cvLibData       *canvas.Library                // 多画布落盘库
	cv              canvasState                    // 画布页状态
	cvView          cvViewer                       // 画布图表全屏查看
	mmCache         map[string]*mmCached           // Mermaid 排版缓存(按源码哈希)
	mmView          mmViewer                       // Mermaid 全屏查看
	br              *browser.Browser               // 内置浏览器(首次打开页面时建立)
	brs             browserState                   // 浏览器页状态
	brSel           brSelect                       // 阅读模式框选引用
	cust            *customize.Store               // 自定义页数据(首次打开页面时建立)
	cu              custState                      // 自定义页状态
	cuNoMem         bool                           // 自定义页:「加载 CLAUDE.md 记忆文件」被关掉(只是界面状态,尚未接入 env)
	use             usageState                     // 用量页
	usageSrc        func(rng int) core.UsageReport // 用量数据源(nil = 演示数据)
	acct            acctState                      // 账号菜单(真实模式缓存)
	qp              *qpState                       // 快速选择浮层(历史会话 / 分支),nil = 关闭
	repo            vcs.Repo                       // 当前项目的仓库(演示为内存实现)
	branch          string                         // 当前分支名(缓存,避免每帧查 git)
	copyText        string                         // 待写入剪贴板的文字(需要 ui.Context 才能写,下一帧消费)
	termPasteReq    bool                           // 终端右键「粘贴」:下一帧在 shortcuts 里读剪贴板粘进当前终端
	toast           string                         // 待显示的一条轻提示(渲染时消费)
	welcomeFocus    bool                           // 欢迎页的输入框自动聚焦
	cardHov         bool                           // 当前这张工具卡是否被悬停(卡片内容里的悬停按钮用)
	hist            histState                      // 历史会话弹层的搜索词
	rv              reviewState                    // 审查标签状态
	rvStageUndo     kit.UndoSlot                   // 审查「暂存已接受」撤销
	reviewDemoEmpty bool                           // 演示:审查标签空态(-ed review:empty)
	wsDiagDemo      string                         // 演示 / -ed:工作区诊断态 scan|done|trunc|empty|err
	diffSelTab      *ideTab                        // 差异标签的行选区
	diffSelRange    editor.DiffLineRange
	cw              *commitWizard       // 「逐个提交」向导(nil = 没有进行中的)
	taskSt          ui.ScrollState      // 任务输出标签的滚动
	taskNoFollow    bool                // 任务输出:关闭「跟随输出」
	subSt           ui.ScrollState      // 子 Agent 标签「过程」区滚动
	subNoFollow     bool                // 子 Agent 流:用户手动滚离底部
	subAwayLen      int                 // 离开底部时的流尾标记(回到最新圆点)
	q               promptQueue         // 排队的消息
	qa              map[string]*qaState // 提问向导的本地进度
	sessLoad        sessLoadState       // 切换会话载入中(抑制欢迎页闪现)
	dockDraft       string              // 审批卡顶替输入框时保存的草稿
	dockStash       bool
	dockCardKey     string    // 当前交互卡 id(切换权限 / 提问时重播入场)
	dockCardAt      time.Time // 交互卡 a-pop-up 起点
	composerInAt    time.Time // 输入框淡入动画起点
	slashSel        int       // 斜杠补全选中行
	slashProj       string    // 已缓存斜杠命令的项目路径
	slashDisk       *customize.DiskStore

	// 测试 / 截图用:没有系统红绿灯时自己画
	fakeLights        bool
	devBadge          string
	proj              string // 真实项目目录(演示模式为空)
	demoExtra         bool   // 演示:工作区里还有一个本会话没碰过的未提交文件
	welcomeLine       string // 演示欢迎页状态行变体(clean / mcp-fail / review)
	demoPgCanvas      int    // 演示画布数据套(见 demo.CanvasSetNames)
	demoPgAuto        int
	demoPgBrowser     int
	demoPgCustomize   int
	demoPgUsage       int
	update            string // 状态栏更新提示文案(空 = 不显示)
	updSt             updState
	upd               *update.Runner
	updSilent         bool   // 当前这次检查是后台静默检查(失败 / 已是最新都不打扰界面)
	updNoticed        string // 已经为哪个版本发过「发现新版本」通知,避免每次检查都弹
	nt                *noticeStore
	mcpExp            map[string]bool // MCP 弹层里各服务器是否展开工具列表
	ctxLoading        bool
	ctxCompacting     bool
	compDoc           *richtext.Document  // 输入框的富文本文档(正文里的文件 / 文件夹是行内胶囊原子),见 composer_rich.go
	compVer           uint64              // compDoc 上次同步回 a.draft 时的版本
	draftBody         string              // 输入框正文(不含图片附件语法;胶囊以原子标记串出现),由 compDoc 导出;与 a.draft 的同步见 composer_draft.go
	mcpToolsNet       map[string][]string // 直连 http MCP 服务器读到的工具名(没有会话时的来源):名称 → 工具
	mcpToolsNetErr    map[string]string   // 直连失败的原因(服务器原话);键存在但值空 = 读取中
	cuDenyFor         string              // MCP 详情当前已为哪台服务器读过工具禁用规则
	mcpProbe          map[string]string   // 「诊断」的结果(服务器原话):名称 → 文字;空串 = 诊断中
	mcpEnvSrc         map[string]string   // MCP 配置里引用的 ${变量} 能从哪取到:app / shell / none(没有该键 = 还在检测)
	pillImgPaths      []string            // 非 nil = 建「图片 N」胶囊时按这个列表取序号(编辑卡用),否则按输入框附件
	pasteTo           string              // 非空 = 粘贴 / 添加的图片送去这张编辑卡(用户消息 id)而不是输入框
	imgPilled         map[string]bool     // 输入框正文里现有的图片位置胶囊(路径集合):胶囊被删就把对应图片附件一并移除
	draftImgs         []string            // 草稿里的图片附件(`![](路径)` 串,按顺序),缩略图卡的数据源
	draftSeen         string              // 上一次同步时的 a.draft;a.draft 变了说明别处改过草稿,需要重新拆分
	ctxOpenKey        string              // 上下文弹层当前已触发过查询的那一次打开(popup 键);关掉后清空
	pageKey           string              // 整页功能(自动化 / 画布 / 浏览器 / 自定义 / 用量)打开时的工作台上下文,变了就自动退出整页(page_auto_leave.go)
	notifyDeniedShown bool                // 「系统通知已关闭」提示这次运行里已经给过
	ctxRefreshAt      time.Time
	mcpRefreshAt      time.Time
	mcpRefreshing     bool
	mdChunks          map[string]mdChunkEntry // 长助手正文的分块缓存(见 chunksOf)
	ctxSent           string                  // 本会话已发给模型的 IDE 上下文(相同内容不重复发)
	mcpListBusy       bool                    // claude mcp list 正在后台跑(它会逐个探活,可能要几十秒,不能叠着起)
	mcpListAgain      bool                    // 跑的时候又有人要求刷新(如连续保存多台 MCP):跑完后再测一遍
	lspJobs           map[string]*lspJob      // 语言包下载安装任务(键 = lspJobKey);只在界面线程读写
	lspJobOrder       []string                // 任务的先后顺序(排队用)
	mcpListAt         time.Time               // 上次拿到 mcp list 结果的时间
	mcpDeny           map[string]bool         // 本地层被禁用的 MCP 工具规则(缓存,弹层打开 / 切换后后台重读)
	mcpRuntimeAt      int64                   // 最近一次 mcp_status 成功的 Unix 秒
	mcpTurnRefDone    bool                    // 本会话首回合结束后是否已自动刷新 MCP
	sessProcLive      bool                    // claude 子进程是否至少启动过一次
	cuMcpManageLoaded bool                    // 本次进入自定义 MCP 页是否已触发加载
	sideBF            string                  // 方案 B 侧栏当前浏览的项目 id
	sessCtxWs         string                  // 侧栏正在画哪个工作区的会话行(点会话时先进入该工作区)
	sessMenuOpen      map[string]*bool        // 会话行「⋯」菜单是否打开(按会话 id;打开期间按钮不随悬停消失)
	sessFork          func(fromID string) (newID string, err error)
	brand             brand // Dock / 托盘(brand.go)
}

// Options 是创建应用的参数。
type Options struct {
	Palette, Scheme, Accent string
	Mode                    Mode
	Screen                  Screen
	FakeLights              bool
	Prefs                   *prefs.Store // 空 = 只在内存
	DevBadge                string
	Secondary               bool // 从属新窗口进程
}

// New 创建应用状态(不开窗口)。
func New(o Options) *App {
	a := &App{
		auto:         automation.NewStore(nil),
		st:           core.NewState(),
		pal:          theme.Find(o.Palette),
		sch:          theme.FindScheme(o.Scheme),
		accent:       o.Accent,
		mode:         o.Mode,
		screen:       o.Screen,
		sideOpen:     true,
		nav:          "",
		projOpen:     map[string]bool{},
		mcpExp:       map[string]bool{},
		sessMore:     map[string]bool{},
		folds:        map[string]bool{},
		thinkMq:      map[string]thinkMqSt{},
		thinkMqHide:  map[string]time.Time{},
		thinkWasLive: map[string]bool{},
		outFold:      map[string]bool{},
		tick:         map[string]*tickState{},
		runWas:       map[string]bool{},
		fadeAt:       map[string]time.Time{},
		termLive:     map[string]string{},
		planOpened:   map[string]bool{},
		foldAt:       map[string]time.Time{},
		ide:          newIDE(),
		prefs:        o.Prefs,
		density:      1,
		radius:       1,
		fakeLights:   o.FakeLights,
		devBadge:     o.DevBadge,
		secondary:    o.Secondary,
	}
	if a.screen == "" {
		a.screen = ScreenWork
	}
	// 默认开合:Agent = 工作区收起、底部面板收起;IDE = 对话栏展开、底部面板展开
	a.memBy = [2]layMem{
		{right: false, panel: false, side: true},
		{right: true, panel: true, side: true},
	}
	a.loadLayMemFromPrefs()
	a.applyLayMem(a.memBy[a.mode])
	if a.prefs == nil {
		a.prefs = prefs.Memory()
	}
	a.MigrateSettingsPrefs()
	a.loadLayout()
	if o.Mode == ModeAgent && a.setTab == "" {
		a.setTab = "appearance"
	}
	a.applyPrefs()
	a.ide.onSaved = a.lspSaved
	a.ide.beforeSave = a.beforeSave
	a.ide.onDiskSnap = func(t *ideTab) { a.recordDiskSnap(t) }
	a.ide.onReveal = a.ensureWorkspace
	a.ide.listAsync = a.listDirAsync // 目录列表缓存失效后在后台读,不在界面线程同步读盘(见 ide_tree_async.go)
	a.ide.hideDot = func() bool { return !a.prefs.Bool(pTreeHidden, true) }
	return a
}

// retheme 重新解析令牌(换主题 / 方案 / 强调色后调用)。
func (a *App) retheme() {
	sch := a.sch
	sch.RK *= [3]float32{.55, 1, 1.35}[a.radius]           // 圆角倍率
	sch.RowH = theme.Px(sch.RowH) + float32(a.density-1)*2 // 行高随界面大小缩放,再按密度 ±2
	a.tk = theme.Resolve(a.tintedPal(a.pal), sch, a.accent)
	// 表面色调变了,主题 id 没变:按「主题 id」缓存的编辑器配色与各处语法高亮要一并作废,否则代码块底色还是旧的
	a.edTheme, a.brTheme, a.csvTh = nil, nil, nil
	a.hlKey, a.refHlKey, a.mdCodeKey = "", "", ""
	// 系统「降低透明度」/「增强对比度」:流光的染色 / 深度 / 高光 / 半透明卡片换成不透明色(见 Tokens.Tone);
	// 玻璃插件自己会退成不透明表面。取值在换主题时刷新,运行中改系统设置要等下次换主题或重启。
	a.tk.ReduceTransp = a11yReduceTransparency() || a11yIncreaseContrast()
	kit.Use(a.tk)
	a.syncTermColors()
	a.md = nil
	a.ux.rt = nil
	log.Debug("主题", "palette", a.pal.ID, "scheme", a.sch.ID)
	if !a.secondary {
		a.syncBrand()
	}
}

// Attach 把后端接到应用上:事件经窗口的 Update 切回主线程再归约。
// win 为 nil 时(测试)直接在调用线程归约。
func (a *App) Attach(be backend.Backend) {
	a.be = be
	a.ctxSent = ""
	a.warmAttach(be)
	co := &deltaCoalescer{}
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
			co.push(ev, func() { a.win.Update(apply) }, func(m core.Event) func() {
				return func() { a.win.Update(func() { a.applyEv(be, m) }) }
			})
		} else {
			apply()
		}
	})
}

// applyEv 在主线程归约一个(可能是合并过的)事件。
func (a *App) applyEv(be backend.Backend, ev core.Event) {
	if a.be != be {
		a.applyParked(be, ev)
		return
	}
	end := diag.SpanSlow("app", "归约事件", 8*time.Millisecond, "ev", evName(ev))
	a.st.Apply(ev)
	a.afterEvent(ev)
	end()
}

// afterEvent 归约之后的界面侧反应:回合结束时处理排队。
func (a *App) afterEvent(ev core.Event) {
	a.onSessLoadEvent(ev)
	// 历史回放区间:事件早已发生,只还原画面。回合结束 / 待审批 / 后台任务完成都不能再弹通知、响提示音、
	// 亮托盘徽标,也不能把排队的消息当成「刚结束的回合」发出去(此前打开历史会话会突然弹一条「回合结束」)。
	replay := a.st != nil && (a.st.Replaying || isReplayEv(ev))
	a.warmOnEvent(ev)
	// 新会话的 id 在握手后才知道:记下来,侧栏就能高亮它(演示后端的占位 id 不记)
	if _, ok := ev.(core.EvReady); ok {
		isNew := a.curSess == "" || strings.HasPrefix(a.curSess, "new:")
		// 切换会话里开的新会话由 pickOnReady 标记;启动时直接落在新会话(没有恢复历史会话)没经过 OpenSession,
		// 同样要沿用上次选的模型 / 思考强度
		if a.pickOnReady || isNew {
			a.pickOnReady = false
			a.applySavedPicks()
		}
		a.syncGlobalPermMode(isNew) // 权限模式全局生效:新会话 / 恢复的历史会话都对齐最后一次的选择
	}
	if e, ok := ev.(core.EvReady); ok && a.curSess == "" && e.SessionID != "" && !strings.HasPrefix(e.SessionID, "demo-") {
		a.curSess = e.SessionID
		a.freshSess = e.SessionID // 新会话握手拿到 id 不等于「已有会话」:没发消息前还是欢迎页
	}
	if _, ok := ev.(core.EvTurnStarted); ok {
		a.nextStepsClear() // 新回合开始:上一轮的建议作废
		a.sessProcLive = true
		if a.cuDemo() && a.curSess == "b4-queue-img" && a.q.len() == 0 {
			a.QueuePrompts(demo.QueuePromptSamples()...)
		}
	}
	if e, ok := ev.(core.EvTurnEnded); ok && !replay {
		// 回合里 Claude 可能新建了被 .gitignore 忽略的文件(git 状态看不到),文件树缓存一并作废
		a.ide.dropLists()
		if a.q.preempt != "" {
			a.qTurnEnded(e.Reason == core.StopEndTurn)
			a.flushPreempt()
			return
		}
		a.qTurnEnded(e.Reason == core.StopEndTurn)
		a.drainQueue()
		if e.Reason == core.StopEndTurn {
			a.mcpAfterTurnEnd()
			a.nextStepsOnTurnEnd() // 排队里的消息发出去后又是运行中,函数里会判断
		} else {
			a.nextStepsClear()
		}
	}
	if e, ok := ev.(core.EvPlanUpdated); ok && e.Streaming && !replay {
		a.onPlanUpdated(e.ID, true)
	}
	if e, ok := ev.(core.EvTaskOutput); ok {
		a.syncTaskOutput(e.TaskID, e.Text)
	}
	switch ev.(type) {
	case core.EvTaskStarted, core.EvTaskProgress, core.EvTaskUpdated:
		if a.LiveTaskOutput() {
			a.pollTaskOutputFiles()
		}
	}
	if u, ok := ev.(core.EvUsage); ok {
		a.onUsageEvent(u.Usage)
	}
	if _, ok := ev.(core.EvMcpServers); ok {
		a.onMcpServers()
	}
	if n, ok := ev.(core.EvAppend); ok {
		if no, y := n.Item.(*core.Notice); y {
			a.ingestCompactNotice(no.Text)
		}
	}
	if !replay {
		a.afterEventNotify(ev)
	}
	a.afterEventCanvas(ev)
	a.brandAfterEvent(ev)
}

// SetRepo 设置当前项目的仓库;nil 表示不是 Git 仓库(分支显示为空)。
func (a *App) SetRepo(r vcs.Repo) {
	a.headC.reset()
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
	start := time.Now()
	prev := a.curSess
	log.Info("切换会话", "from", prev, "to", id)
	a.revealChat() // IDE 形态下对话栏收着时,新建 / 打开的会话也得看得见
	end := diag.SpanSlow("app", "打开会话", 300*time.Millisecond, "from", a.curSess, "to", id)
	if a.curSess != "" {
		a.saveFlowScroll(a.curSess)
	}
	if id != "" && !strings.HasPrefix(id, "new:") {
		a.sideBF = "" // 方案 B 的项目条跟着打开的会话走
	}
	a.maybeLeaveWorkspace(id)
	a.recordSess(id)
	a.parkOrClose()
	a.closeSessionVirtuals()
	delete(a.sessUnread, id)                     // 打开 = 看过了
	if lv := a.live[id]; lv != nil && id != "" { // 回到一个在后台跑着的会话:接回它的后端与状态
		delete(a.live, id)
		a.be, a.st = lv.be, lv.st
		a.warmAdopt(lv.be)
		a.sessLoad = sessLoadState{}
		a.q.reset()
		a.qa = nil
		a.folds = map[string]bool{}
		a.thinkMq, a.thinkMqHide, a.thinkWasLive = map[string]thinkMqSt{}, map[string]time.Time{}, map[string]bool{}
		a.outFold = map[string]bool{}
		a.termLive = map[string]string{}
		a.userPinned, a.userAnchor, a.lb = nil, nil, nil
		a.flowNoFollow = false
		a.draft, a.dockDraft = "", ""
		a.dockStash = false
		a.curSess = id
		a.screen = ScreenWork
		a.flowSt = ui.ScrollState{}
		a.flowList = ui.ListState{}
		a.restoreFlowScroll(id)
		a.triggerFlowRise()
		a.syncGlobalPermMode(false) // 后台会话离开时的模式可能已过时
		end()
		log.Info("接回后台会话", "id", id, "耗时", time.Since(start))
		diag.LogUISwitch("session:"+prev, "session:"+id, time.Since(start))
		return
	}
	a.beginSessLoad(id)
	a.ctxRefreshAt = time.Time{}
	a.mcpRefreshAt = time.Time{}
	a.mcpRefreshing = false
	a.mcpRuntimeAt = 0
	a.mcpTurnRefDone = false
	a.sessProcLive = false
	a.cuMcpManageLoaded = false
	a.st = core.NewState()
	a.q.reset()
	a.qa = nil
	a.folds = map[string]bool{}
	a.thinkMq, a.thinkMqHide, a.thinkWasLive = map[string]thinkMqSt{}, map[string]time.Time{}, map[string]bool{}
	a.outFold = map[string]bool{}
	a.tick = map[string]*tickState{}
	a.runWas, a.fadeAt = map[string]bool{}, map[string]time.Time{}
	a.termLive = map[string]string{}
	a.planOpened = map[string]bool{}
	a.userPinned, a.userAnchor, a.lb = nil, nil, nil
	a.flowNoFollow = false
	a.draft, a.dockDraft = "", ""
	a.dockStash = false
	a.curSess = id
	a.screen = ScreenWork
	a.flowSt = ui.ScrollState{}
	a.flowList = ui.ListState{}
	a.restoreFlowScroll(id)
	a.pickOnReady = id == "" || strings.HasPrefix(id, "new:") // 新会话沿用上次选的模型与思考强度
	if a.opener != nil {
		a.Attach(a.opener(id))
	}
	if strings.HasPrefix(id, "new:") { // 项目内新会话:对界面来说仍是「还没有 id 的新会话」
		a.curSess = ""
	}
	end()
	log.Info("会话切换已排队", "to", id, "耗时", time.Since(start))
	diag.LogUISwitch("session:"+prev, "session:"+a.curSess, time.Since(start))
}

// NewSessionIn 在某个项目里开新会话(侧栏项目行的「+」)。
func (a *App) NewSessionIn(p core.ProjectInfo) {
	a.projOpen[p.ID] = true
	a.OpenSessionChecked("new:" + p.Path)
}

// QueuePrompts 直接排入若干条消息(截图 / 测试用)。
func (a *App) QueuePrompts(texts ...string) {
	for _, t := range texts {
		a.q.add(t)
	}
}

// NewSession 开始新会话并回到欢迎页。
// 在工作区里:新会话落在工作区的成员文件夹(沿用当前所在成员,否则第一个),不能用「上一个会话的目录」——
// 那个目录可能根本不在工作区里(例如工作区成员是 kb/a、kb/b,而上一个会话在它们的父目录 kb),
// 落过去后「会话目录不属于工作区」会把人从工作区踢出去,表现为一新建会话就跳到别的项目。
func (a *App) NewSession() {
	if a.ws != nil && !a.demoWs() {
		dir := a.wsStartDir(*a.ws)
		log.Info("工作区新建会话", "ws", a.ws.ID, "dir", dir)
		a.OpenSessionChecked("new:" + dir)
		return
	}
	a.OpenSession("")
}

// SetWindow 记下窗口(用于 Update 与窗口控制)。
func (a *App) SetWindow(w *mygo.Window) {
	a.win = w
	// 红色关闭按钮只隐藏窗口:钩子必须窗口一建好就装上,不能等到某次偏好变更才顺带装(此前关闭按钮直接退出程序)
	a.ensureWindowCloseHook()
}

// SetRunDemo 标记为演示模式(恢复布局时不切真实项目目录)。
func (a *App) SetRunDemo(v bool) { a.runDemo = v }

// State 返回当前会话状态(测试用)。
func (a *App) State() *core.State { return a.st }

// send 把界面命令交给后端。
func (a *App) send(cmd core.Command) {
	if a.be == nil {
		if _, ok := cmd.(core.CmdPrompt); ok { // 没有后端时发不出去,别让提示词无声消失
			a.toast = "还没有连接到会话,消息没有发出"
			log.Warn("没有后端,丢弃提示词")
		}
		return
	}
	if p, ok := cmd.(core.CmdPrompt); ok {
		p = a.attachIDEContext(p)
		cmd = p
		a.noteActive()
		if a.pvEnvPending && !p.Steer && a.st.Phase != core.PhaseRunning { // 回合进行中切换的服务商 / 模型,在这条消息前生效
			a.pvPushEnv()
		}
		// 发送的同一帧就进入运行态:状态行立刻出现,不等后端回「回合开始」(冷启动要几百毫秒;对标主分支 mark_running)。
		// 后端随后的 EvTurnStarted 是幂等的;启动失败会回 EvTurnEnded,自然恢复。
		if !p.Steer && a.st.Phase != core.PhaseRunning {
			a.st.Apply(core.EvTurnStarted{})
		}
	}
	if c, ok := cmd.(core.CmdSetMode); ok && c.Mode == core.ModeBypassPermissions && a.NoBypassPermissions() {
		a.toast = "已禁止绕过权限模式"
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
	from := a.mode
	t0 := time.Now()
	log.Info("切换形态", "from", from, "to", m)
	a.beltCapture() // 必须在模式与开合还没换之前:记下旧几何,作为传送带平移的起点
	a.rememberLayout()
	a.morphFrom, a.morphAt = from, kit.Now()
	a.mode = m
	a.persistLastMode() // rememberLayout 里写的是切换前的旧形态,这里补写新形态,否则下次启动总是回到上一个形态
	a.restoreLayoutFor(m)
	a.beginMorph()
	diag.LogUISwitch(modePerfName(from), modePerfName(m), time.Since(t0))
}

// layMem 是一个形态记住的开合(见 layout_mem.go)。
type layMem struct {
	right, panel, side bool
	chatMax            bool
}

var viewFirstFrame sync.Once

// View 是窗口的根视图。
func (a *App) View(c *ui.Context) {
	a.buildCtx = c // 构建期间有效:Markdown 的链接图标回调(path_ref.go linkIcon)要量字
	viewFirstFrame.Do(diag.MarkFirstFrame)
	viewStart := time.Now()
	var gap time.Duration
	if !a.perfLastView.IsZero() {
		gap = viewStart.Sub(a.perfLastView)
	}
	a.perfLastView = viewStart
	defer func() {
		diag.RecordUIFrame(time.Since(viewStart), gap)
	}()
	defer diag.SpanSlow("app", "构建界面", 33*time.Millisecond)()
	kit.SetViewActive(a.win != nil && a.win.IsFocused())
	a.autoPersist()
	// 界面构建中 panic:不能吞(控件树做到一半会错乱),但崩溃前必须把调用栈写进日志与 crashes/,
	// 否则只剩一条「上次运行异常退出」,无从排查(审查页空指针闪退就是这样静默消失的)。
	defer func() {
		if v := recover(); v != nil {
			diag.Report("界面构建", v)
			panic(v)
		}
	}()
	c.SetTheme(a.tk.UITheme())
	// 右键菜单与输入框编辑菜单一律画在窗口里、跟随当前主题(而不是系统原生菜单),
	// 圆角 / 描边 / 阴影 / 悬停色与应用内其它弹层一致。
	c.SetMenuPresentation(ui.PopoverMenus)
	a.winW, a.winH = c.Size()
	a.svc = c.Services()
	a.sideCurOk = false // 每帧由 side() 重新写入;没画导航栏的页面不沿用上一帧的宽度
	a.trackLayout()
	a.warmTick() // 到点预热 / 回前台补预热 / 空闲释放(每帧只比几个时间,很便宜)
	a.fitPanels()
	a.reposTick()
	if (a.brand.trayErr || a.brand.trayUnread) && a.appInForeground() {
		a.syncBrand() // 回到前台:托盘的「出错 / 已完成」徽标算已读
	}
	a.shortcuts(c)
	a.pageAutoLeave()
	if a.popup == "" {
		a.popSeen = ""
	}
	// 只有快速选择没走 dialogOpen,关了要手动清;其它浮层(命令面板、自动化 / 自定义新建、
	// 文件树「让 Claude」、项目对话框)由 dialogOpen 自己管。这里不能无条件清空 dlgSeen,
	// 否则入场计时每帧被重置,浮层永远停在透明(却照常拦截点击)。
	if a.qp == nil && a.dlgSeen == "quick" {
		a.dlgSeen = ""
	}
	a.autosave()
	a.extmodTick()
	if kit.ScrollFrameActive(a.now()) {
		a.frameSoon()
	}
	a.root(c)
	a.probeEnd(c.Now())
}

// isReplayEv 是不是历史回放的开始 / 结束标记。
func isReplayEv(ev core.Event) bool {
	_, ok := ev.(core.EvReplay)
	return ok
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
	if a.welcomeLine == "" {
		a.welcomeLine = "review"
	}
	a.auto = automation.NewStore(demo.AutomationsFor(a.demoPgAuto, a.now()))
	a.au.sel = 2 // 设计稿里选中的是第二张卡
	a.ide.seedDemo()
	a.seedDemoRegistry()
	a.seedWorkspace()
	demo.SeedOutput(a.outAppend)
	a.SeedUpdateDemo("download", 42)
}

// SeedDemoData 演示模式启动时调用:只补各整页功能的演示数据源(自动化列表、画布 / 浏览器 / 自定义的演示源、
// 输出面板预置),让演示版与设计稿一致、且不再读写用户真实的 ~/.minicode/canvases / automations.json。
// 不打开编辑器标签、不选中审查标签(那是设计稿截图状态,见 SeedDemoExtras),以免和「恢复上次现场」冲突。
func (a *App) SeedDemoData() {
	a.demoExtra = true
	a.auto = automation.NewStore(demo.AutomationsFor(a.demoPgAuto, a.now()))
	a.au.sel = 2 // 设计稿里选中的是第二张卡
	demo.SeedOutput(a.outAppend)
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
	if a.termPasteReq { // 终端右键菜单的「粘贴」:菜单回调里没有可靠的上下文,置标记在这里读剪贴板
		a.termPasteReq = false
		if tp := a.ide.tpc(); tp != nil && tp.t != nil {
			if s := c.ReadClipboard(); s != "" {
				a.termPaste(tp, s)
			}
			tp.focusIt = true
		}
	}
	if a.termShortcuts(c) {
		return
	}
	if c.Shortcut(ui.Alt, ui.KeyZ) || a.shortcutHit(c, "wrap", ui.Alt, ui.KeyZ) {
		a.toggleWrap()
		return
	}
	if a.editorShortcuts(c) {
		return
	}
	if c.Shortcut(ui.Cmd, ui.KeyBackspace) && a.filesFocus && a.ide.sel != "" {
		a.askDelete(a.ide.sel, workspace.IsDir(a.ide.fs, a.ide.sel))
		return
	}
	// ⌘⏎:文件树里选中的文件在侧边打开(与右键菜单「在侧边打开」同一动作)
	if c.Shortcut(ui.Cmd, ui.KeyEnter) && a.filesFocus && a.ide.sel != "" && !workspace.IsDir(a.ide.fs, a.ide.sel) {
		a.openSide(a.ide.sel, true)
		return
	}
	if a.filesFocus && a.nav == "files" {
		if c.Shortcut(ui.Cmd, ui.KeyC) && a.ide.sel != "" && !workspace.IsDir(a.ide.fs, a.ide.sel) {
			a.treeCopyFile(a.ide.sel, false)
			return
		}
		if c.Shortcut(ui.Cmd, ui.KeyX) && a.ide.sel != "" && !workspace.IsDir(a.ide.fs, a.ide.sel) {
			a.treeCopyFile(a.ide.sel, true)
			return
		}
		if c.Shortcut(ui.Cmd, ui.KeyV) {
			into := a.ide.sel
			if into != "" && !workspace.IsDir(a.ide.fs, into) {
				into = parentDir(into)
			}
			a.treePasteFile(into)
			return
		}
	}
	switch {
	case a.shortcutHit(c, "nav", ui.Cmd, ui.KeyB):
		a.sideOpen = !a.sideOpen
	case a.shortcutHit(c, "goChat", ui.Cmd, ui.Key1):
		a.switchNav("chat")
	case a.shortcutHit(c, "goFiles", ui.Cmd, ui.Key2):
		a.switchNav("files")
	case a.shortcutHit(c, "goSearch", ui.Cmd, ui.Key3):
		a.switchNav("search")
	case a.shortcutHit(c, "goScm", ui.Cmd, ui.Key4):
		a.switchNav("scm")
	case a.shortcutHit(c, "right", ui.Cmd|ui.Alt, ui.KeyB):
		a.toggleRight()
	case a.shortcutHit(c, "toggleMode", ui.Cmd, ui.KeyE):
		if a.mode == ModeAgent {
			a.SetMode(ModeIDE)
		} else {
			a.SetMode(ModeAgent)
		}
	case a.motionCatOpen && c.Shortcut(0, ui.KeyEscape):
		a.closeMotionCatalog()
	case a.paletteOn && c.Shortcut(0, ui.KeyEscape):
		a.closePalette()
	case a.shortcutHit(c, "palette", ui.Cmd, ui.KeyK):
		if a.paletteOn {
			a.closePalette()
		} else {
			a.openPalette()
		}
	case a.shortcutHit(c, "newChat", ui.Cmd, ui.KeyN):
		a.NewSession()
	case a.shortcutHit(c, "newWin", ui.Cmd|ui.Shift, ui.KeyN):
		if !a.secondary {
			a.SpawnNewWindow(a.proj)
		}
	case a.shortcutHit(c, "search", ui.Cmd|ui.Shift, ui.KeyF):
		a.OpenSearch()
	case a.shortcutHit(c, "scm", ui.Ctrl|ui.Shift, ui.KeyG):
		a.OpenSCM()
	case a.shortcutHit(c, "symbols", ui.Cmd|ui.Shift, ui.KeyO):
		a.OpenQuick("symbols", "")
	case a.shortcutHit(c, "panel", ui.Cmd, ui.KeyJ):
		a.ide.bottomOpen = !a.ide.bottomOpen
	case c.Shortcut(ui.Cmd|ui.Shift, ui.KeyJ):
		a.togglePanelMax()
	case c.Shortcut(ui.Cmd|ui.Shift, ui.KeyM):
		a.toggleChatMax()
	case a.editingUser() && c.Shortcut(0, ui.KeyEscape):
		a.cancelUserEdit()
	case a.chatMax && !a.paletteOn && a.qp == nil && a.popup == "" && c.Shortcut(0, ui.KeyEscape):
		a.chatMax = false // Esc:退出「对话占满」
	case c.Shortcut(ui.Ctrl, ui.KeyMinus):
		a.navGo(-1)
	case c.Shortcut(ui.Ctrl|ui.Shift, ui.KeyMinus):
		a.navGo(1)
	case a.shortcutHit(c, "save", ui.Cmd, ui.KeyS):
		if a.mode == ModeIDE {
			if a.extmodSaveIntercept() {
				break
			}
			if err := a.ide.save(); err != nil {
				log.Error("保存失败", "err", err)
			}
		}
	case a.shortcutHit(c, "settings", ui.Cmd, ui.KeyComma):
		a.openSettings(a.setTab)
	case a.shortcutHit(c, "modeCycle", ui.Shift, ui.KeyTab):
		a.cyclePermissionMode()
	case a.screen == ScreenSettings && !a.paletteOn && a.set.query == "" && c.Shortcut(0, ui.KeyEscape):
		a.closeSettings()
	case a.screen == ScreenUsage && !a.paletteOn && a.qp == nil && a.popup == "" && c.Shortcut(0, ui.KeyEscape):
		a.closeUsage() // Esc:退出用量整页
	case a.escInterrupts() && c.Shortcut(0, ui.KeyEscape):
		a.send(core.CmdCancel{}) // Esc 中断回合(状态行写着「Esc 中断」;对标 Claude Code)
		log.Info("Esc 中断回合")
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
	if am := a.pvActiveModel(); am != "" { // 第三方服务商:模型已由 --model 给定,不能再套官方偏好
		a.st.Model = am
	} else if m := a.prefs.Str(prefModel, ""); m != "" && m != a.st.Model {
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
	be        backend.Backend
	st        *core.State
	id        string
	idleSince time.Time // 回合结束的时刻(进行中为零):后台会话跑完后不立刻结束进程,闲置超过时限才回收
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

// applyParked 把后台会话的事件归约到它自己的状态。
// 回合结束后不立刻结束它的进程:留在后台打「已结束未查看」标记,闲置超过时限才由 reapParked 回收
// (用户刚切走不等于不需要了,马上切回来应该零等待接回)。进程自己退出了就直接清掉。
func (a *App) applyParked(be backend.Backend, ev core.Event) {
	for id, lv := range a.live {
		if lv.be != be {
			continue
		}
		lv.st.Apply(ev)
		if lv.st.Replaying {
			return
		}
		switch e := ev.(type) {
		case core.EvTurnStarted:
			lv.idleSince = time.Time{}
		case core.EvTurnEnded:
			if lv.st.Phase != core.PhaseRunning {
				lv.idleSince = a.now()
				if a.sessUnread == nil {
					a.sessUnread = map[string]bool{}
				}
				a.sessUnread[id] = true
				log.Info("后台会话已结束,留在后台待回收", "id", id)
			}
		case core.EvConn:
			if (e.State == core.ConnNone || e.State == core.ConnFailed) && lv.st.Phase != core.PhaseRunning {
				log.Info("后台会话的进程已退出,清理", "id", id)
				delete(a.live, id)
				be.Close()
			}
		}
		return
	}
}

// reapParked 回收闲置够久的后台会话进程(同「设置 → 连接 → 闲置释放」;窗口在后台时按 ⅓ 计)。
// 不回收:回合进行中 / 有存活的后台任务 / 有待处理的权限、提问、计划。
func (a *App) reapParked() {
	limit := a.idleLimit()
	if limit == 0 || len(a.live) == 0 {
		return
	}
	if !a.warmFg() {
		limit /= warmBgDivisor
	}
	now := a.now()
	for id, lv := range a.live {
		if lv.st == nil || lv.st.Phase == core.PhaseRunning || lv.idleSince.IsZero() || now.Sub(lv.idleSince) < limit {
			continue
		}
		if lv.st.AwaitingUser() || len(lv.st.RunningTasks()) > 0 {
			continue
		}
		log.Info("回收闲置的后台会话", "id", id, "闲置", now.Sub(lv.idleSince).Round(time.Second))
		delete(a.live, id)
		lv.be.Close()
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

// runAutomation 定时到点与测试入口(与「立即运行」同路径,会回写 session_id)。
func (a *App) runAutomation(it automation.Automation) { a.runAutomationCore(it) }

type autoRunner func(it automation.Automation, dir string) (string, error)

func runCtxLegacy(ctx context.Context, it automation.Automation, dir string, run autoRunner, lv *autoLive) (string, error) {
	type res struct {
		out string
		err error
	}
	ch := make(chan res, 1)
	go func() {
		o, e := run(it, dir)
		ch <- res{o, e}
	}()
	select {
	case <-ctx.Done():
		return lv.snapshot(), ctx.Err()
	case r := <-ch:
		if r.out != "" {
			lv.appendChunk(r.out)
		}
		return r.out, r.err
	}
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
	if from.IsZero() || (a.autoRun == nil && a.autoRunCtx == nil && !a.demoWs()) {
		return
	}
	for _, it := range a.auto.Due(from, now) {
		log.Info("自动化到点", "id", it.ID, "name", it.Name)
		a.runAutomation(it)
	}
}

// escInterrupts Esc 是否该中断当前回合:只在 Agent 形态的会话主界面、回合进行中、没有任何浮层 / 编辑态 / 斜杠菜单时。
// 其它地方的 Esc(关弹层、退出编辑、关查找条)优先,绝不误伤正在跑的回合。
func (a *App) escInterrupts() bool {
	return a.st.Phase == core.PhaseRunning && a.mode == ModeAgent && a.screen == ScreenWork &&
		!a.paletteOn && a.qp == nil && a.popup == "" && !a.chatMax && !a.editingUser() && a.pd.kind == "" &&
		a.pendingDock() == nil && // 有权限 / 提问卡时 Esc 归卡片(拒绝 / 跳过),不当作中断
		!strings.HasPrefix(a.draft, "/")
}
