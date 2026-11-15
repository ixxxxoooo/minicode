// Package app 是 MiniCode 的原生界面:一个窗口、两种形态(Agent / IDE),所有部件共用同一份会话状态与后端。
// 视图是纯函数式的(每帧从 App 的状态构建),交互一律转成 core.Command 发给后端。
package app

import (
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/backend"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
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
	flowSt ui.ScrollState // 会话流滚动状态(跟随底部)
	win    *mygo.Window

	// 会话:当前显示的会话状态与它的后端
	st *core.State
	be backend.Backend

	// 外观
	pal    theme.Palette
	sch    theme.Scheme
	accent string
	tk     *theme.Tokens

	// 窗口形态与布局
	mode      Mode
	screen    Screen
	sideOpen  bool
	inspOpen  bool
	nav       string // chat / files / search / scm
	inspTab   string // changes / files / term / plan / tasks
	popup     string // 状态栏 / 输入框弹层:cfg / ctx / upd / acct
	paletteOn bool

	// 侧栏数据
	projects []core.ProjectInfo
	curSess  string
	projOpen map[string]bool
	sessMore map[string]bool

	// 会话流的折叠状态:条目 id → 是否展开(没有记录就用默认)
	folds map[string]bool

	// Markdown 选项(随主题重建)
	md *markdown.Options

	// 输入框
	draft string

	// 测试 / 截图用:没有系统红绿灯时自己画
	fakeLights bool
	devBadge   string
	demoExtra  bool   // 演示:工作区里还有一个本会话没碰过的未提交文件
	update     string // 状态栏更新提示文案(空 = 不显示)
}

// Options 是创建应用的参数。
type Options struct {
	Palette, Scheme, Accent string
	Mode                    Mode
	Screen                  Screen
	FakeLights              bool
	DevBadge                string
}

// New 创建应用状态(不开窗口)。
func New(o Options) *App {
	a := &App{
		st:         core.NewState(),
		pal:        theme.Find(o.Palette),
		sch:        theme.FindScheme(o.Scheme),
		accent:     o.Accent,
		mode:       o.Mode,
		screen:     o.Screen,
		sideOpen:   true,
		inspOpen:   true,
		nav:        "",
		inspTab:    "changes",
		projOpen:   map[string]bool{},
		sessMore:   map[string]bool{},
		folds:      map[string]bool{},
		fakeLights: o.FakeLights,
		devBadge:   o.DevBadge,
	}
	if a.screen == "" {
		a.screen = ScreenWork
	}
	a.retheme()
	return a
}

// retheme 重新解析令牌(换主题 / 方案 / 强调色后调用)。
func (a *App) retheme() {
	a.tk = theme.Resolve(a.pal, a.sch, a.accent)
	kit.Use(a.tk)
	a.md = nil
	log.Debug("主题", "palette", a.pal.ID, "scheme", a.sch.ID)
}

// Attach 把后端接到应用上:事件经窗口的 Update 切回主线程再归约。
// win 为 nil 时(测试)直接在调用线程归约。
func (a *App) Attach(be backend.Backend) {
	a.be = be
	be.Start(func(ev core.Event) {
		apply := func() {
			end := diag.SpanSlow("app", "归约事件", 8*time.Millisecond, "ev", evName(ev))
			a.st.Apply(ev)
			end()
		}
		if a.win != nil {
			a.win.Update(apply)
		} else {
			apply()
		}
	})
}

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
	a.mode = m
	a.nav = ""
}

// View 是窗口的根视图。
func (a *App) View(c *ui.Context) {
	defer diag.SpanSlow("app", "构建界面", 33*time.Millisecond)()
	c.SetTheme(a.tk.UITheme())
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

// SeedDemoExtras 演示 / 截图用:补上设计稿里工作区的附加状态(额外未提交文件、更新提示)。
func (a *App) SeedDemoExtras() {
	a.demoExtra = true
	a.update = "更新 42%"
}
