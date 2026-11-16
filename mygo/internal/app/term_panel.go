package app

import (
	"cmp"
	"math"
	"os"
	"path/filepath"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/egoist/mygo/ui"

	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/term"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 真实终端面板:界面只做三件事——量出能放下多少列 / 行并通知内核改大小、把内核的画面快照画成
// 带色文字、把按键翻译成仿真器的按键事件。shell 的启动、输出解析都在 internal/term。

// termDebugDone:MINI_TERM_CMD 只对进程里第一个启动的终端生效。
var termDebugDone bool

const termHistory = 200 // 面板里最多带的历史行数

// 终端字号与行高(设置「终端 → 字号」改它;行高 = 字号 × 1.6)。
var (
	termFont   float32 = 11.5
	termLineH  float32 = 18.4
	termFamily         = "monospace" // 终端字体(设置 → 终端 → 字体;空 = 系统等宽)
)

// termPanel 是 IDE 底部终端的状态。
type termPanel struct {
	t         *term.Term
	label     string // 用户重命名的标签名(空 = 走标题优先级)
	demoTitle string // 演示 / 截图用 OSC 标题替身(非演示应为空)
	err       string // 启动失败的原因
	cellW     float32
	st        ui.ScrollState
	waits     int    // 等待量出大小的帧数(防止面板根本没有尺寸时空转)
	pend      ui.Key // 按下了会打出字的键,还在等 InputText
	pendSeq   int    // pend 的代号(超时补发时认它,免得补发后来的键)
	pendSh    bool
	compose   bool   // 输入法正在组字
	preedit   string // 组字中的文字(显示在光标处,提交时由 InputText 送入)
	focusIt   bool   // 下一帧把键盘焦点给终端(刚启动 / 刚重启)
	initCmd   string // shell 起来后自动输入的一条命令(安装语言服务器用)
	cwd       string // 起始目录(空 = 项目根;文件树「在此处打开终端」用)

	scr       *term.Screen // 最近一帧的画面快照(选择 / 搜索 / 链接按它换算)
	viewH     float32      // 可视高度(搜索命中滚到中间用)
	sel       termSel
	find      termFind
	hover     cellPos // 鼠标悬停的格
	hoverOn   bool
	hoverMods ui.Modifiers
	link      *termLink // 当前悬停的链接(⌘ 按住时才有)
	nerdWarn  string    // Nerd Font 缺字形提示(见 term_nerd.go)
	over      []ui.Rect // 画在正文上的控件(提示条、搜索条、回到底部)的区域,相对正文左上角;正文的选择 / 鼠标处理要放过它们
	bodyR     ui.Rect   // 正文上一帧的窗口坐标区域
	nerdKey   string    // 上次判断时的「画面版本|字体|扫描完成」,没变就不重判
}

// redraw 让窗口重画一帧(终端输出到达时由内核的通知调用,任意 goroutine)。
func (a *App) redraw() {
	if a.win != nil {
		a.win.Update(func() {})
	}
}

// closeTerm 结束当前终端进程(切项目、重启、退出时)。
func (a *App) closeTerm() {
	for _, tp := range a.ide.tps {
		if tp.t != nil {
			tp.t.Close()
		}
	}
	a.ide.tps, a.ide.tpSel = nil, 0
}

// newTerm 新建一个终端标签并切过去(新 shell 在面板下一次渲染量出大小后启动)。
func (a *App) newTerm() {
	cw := a.ide.tpc().cellW
	a.ide.tps = append(a.ide.tps, &termPanel{cellW: cw, focusIt: true})
	a.ide.tpSel = len(a.ide.tps) - 1
	a.ide.termSel = ""
	a.ide.bottomOpen, a.ide.bottomTab = true, "term"
	a.panelFlashAt = time.Now()
	log.Info("新建终端标签", "index", a.ide.tpSel)
	a.redraw()
}

// closeTermTab 关掉第 i 个终端标签(结束它的 shell);最后一个关掉后留一个空白标签,下次显示时重开。
func (a *App) closeTermTab(i int) {
	if i < 0 || i >= len(a.ide.tps) {
		return
	}
	if tp := a.ide.tps[i]; tp.t != nil {
		tp.t.Close()
	}
	cw := a.ide.tpc().cellW
	a.ide.tps = append(a.ide.tps[:i:i], a.ide.tps[i+1:]...)
	if len(a.ide.tps) == 0 {
		a.ide.tps = []*termPanel{{cellW: cw}}
	}
	if a.ide.tpSel >= len(a.ide.tps) {
		a.ide.tpSel = len(a.ide.tps) - 1
	}
	if a.ide.tpSel < 0 {
		a.ide.tpSel = 0
	}
	a.ide.tpc().focusIt = true
	log.Info("关闭终端标签", "remaining", len(a.ide.tps))
	a.redraw()
}

// CloseTerminals 应用退出前调用:结束所有终端子进程。
func (a *App) CloseTerminals() { a.closeTerm(); a.lspReset(); a.closeBackends() }

// shellName 是标签上显示的 shell 名。
func shellName() string {
	if s := os.Getenv("SHELL"); s != "" {
		return filepath.Base(s)
	}
	return "zsh"
}

// realTerminal 画真实终端(项目目录下的登录 shell)。
func (a *App) realTerminal(c *ui.Context) {
	tp := a.ide.tpc()
	t := a.tk
	if tp.cellW == 0 {
		w, _ := c.MeasureText(10000, ui.Span{Text: "MMMMMMMMMM", Font: termFamily, Size: termFont})
		tp.cellW = w / 10
		if tp.cellW < 4 {
			tp.cellW = 6.9
		}
	}
	const padX, padY = padXTerm, 6
	body := ui.Column(c).Key("termbody").Grow(1).MinHeight(0).Focusable()
	b := body.Bounds()
	cols := int((b.W - 2*padX) / tp.cellW)
	rows := int((b.H - 2*padY) / termLineH)

	if tp.t == nil && tp.err == "" && cols >= 4 && rows >= 1 {
		sb := a.termScrollback()
		tm, err := term.Start(a.TermStartOpts(cols, rows, sb, cmp.Or(tp.cwd, a.proj)))
		if err != nil {
			tp.err = err.Error()
			log.Error("启动终端失败", "err", err)
		} else {
			tp.t, tp.focusIt = tm, true
			// zsh 已在启动时静默注入;其余 shell 才退回往终端里敲 source 命令
			if a.prefs.Bool(pTermIntegrate, true) && !tm.Integrated() {
				tm.SendText(term.IntegrationCmd(shellName()))
			}
			if tp.initCmd == "" && !termDebugDone {
				// 调试开关 MINI_TERM_CMD:第一个终端起来后自动输入一条命令,便于复现 / 实测(只此一次)
				tp.initCmd, termDebugDone = os.Getenv("MINI_TERM_CMD"), true
			}
			if tp.initCmd != "" {
				tm.SendText(tp.initCmd + "\n")
				log.Info("终端自动输入命令", "len", len(tp.initCmd))
				tp.initCmd = ""
			}
		}
	}
	if tp.t != nil && cols >= 4 && rows >= 1 {
		tp.t.Resize(cols, rows)
	}
	if tp.focusIt {
		body.Focus()
		tp.focusIt = false
	}
	focused := body.Focused()

	var scr *term.Screen
	tp.viewH = b.H
	if tp.t != nil {
		scr = tp.t.Snapshot(termHistory)
		tp.scr = scr
		// 输入法的候选框跟着光标走
		body.TextCaret(ui.Rect{
			X: b.X + padX + float32(scr.CursorX)*tp.cellW,
			Y: b.Y + padY + float32(scr.CursorY)*termLineH - tp.st.Y,
			W: tp.cellW, H: termLineH,
		})
	}
	body.HandleInput(func(ev ui.InputEvent) bool { return a.termInput(c, ev) })
	a.termHasFocus = focused
	if tp.link != nil && tp.hoverMods&ui.Super != 0 {
		body.Cursor(ui.CursorPointer)
	} else {
		body.Cursor(ui.CursorText)
	}
	a.termMenu(c, body, tp)
	tp.bodyR = b
	body.Children(func() {
		tp.over = tp.over[:0]
		a.termNerdCheck(c, tp)
		switch {
		case tp.err != "":
			ui.Column(c).Grow(1).Padding(padY, padX).Children(func() {
				ui.Text(c, "终端启动失败:"+tp.err).TextColor(t.Err)
			})
			return
		case tp.t == nil:
			if tp.waits++; tp.waits < 8 {
				a.redraw() // 第一帧还没量出大小,下一帧再启动
			}
			return
		}
		exited, why := tp.t.Exited()
		if tp.st.Y >= tp.st.MaxY-1 { // 停在底部就一直跟随输出
			tp.st.Y = math.MaxFloat32
		}
		kit.VScroll(c).Key("term").TrackScroll(&tp.st).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Padding(padY, padX).Children(func() {
				for y, ln := range scr.Lines {
					cx := -1
					if y == scr.CursorY && !scr.CursorHidden && !exited {
						cx = scr.CursorX
					}
					a.termRow(c, ln, cx, focused, tp.cellW, tp.preedit, a.rowDeco(tp, y, scr.Cols))
				}
				if exited {
					ui.Row(c).Height(termLineH).Margin(4, 0, 0, 0).Gap(8).AlignItems(ui.Center).TextColor(t.Tx3).Children(func() {
						ui.Text(c, why+" · 按回车重新开始").NoWrap()
					})
				}
			})
		})
		if tp.find.open || !tp.find.closingAt.IsZero() {
			a.termFindBar(c, tp)
			if tp.find.open {
				a.termFindMarks(c, tp, body)
			}
		}
		atBottom := tp.st.MaxY <= 0 || tp.st.Y >= tp.st.MaxY-8
		if !atBottom && tp.t != nil {
			btn := kit.Btn(c, "↓ 回到底部", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).Absolute().Right(12).Bottom(10).
				OnClick(func() { tp.st.Y = math.MaxFloat32; a.redraw() })
			tp.addOverlay(btn.Bounds())
		}
	})
}

// padXTerm 是终端正文左右留白;命令成败点就画在左边这条留白里。
const padXTerm = 10

// restartTerm 在同一目录重开一个 shell。
func (a *App) restartTerm() {
	tp := a.ide.tpc()
	if tp.t != nil {
		tp.t.Close()
	}
	*tp = termPanel{cellW: tp.cellW}
	a.redraw()
}

// termInput 把窗口输入交给终端:按键走仿真器编码(应用光标键、修饰键),文字与粘贴直接送入。
func (a *App) termInput(c *ui.Context, ev ui.InputEvent) bool {
	tp := a.ide.tpc()
	if tp.t == nil {
		return false
	}
	if exited, _ := tp.t.Exited(); exited {
		if ev.Kind == ui.InputKeyDown && ev.Key == ui.KeyEnter {
			a.restartTerm()
			return true
		}
		return false
	}
	switch ev.Kind {
	case ui.InputPointerDown, ui.InputPointerUp, ui.InputPointerMove:
		// 点在叠在正文上的控件(提示条按钮、搜索条、回到底部)上:不当成选择 / 鼠标上报,交给控件自己
		if tp.overlayHit(ev.X, ev.Y) {
			return false
		}
	}
	switch ev.Kind {
	case ui.InputScroll:
		if tp.t.MouseTracking() && ev.Mods&ui.Alt == 0 {
			if a.termPointerWheel(tp, ev) {
				return true
			}
		}
		return false
	case ui.InputPointerDown:
		if tp.t.MouseTracking() && ev.Mods&ui.Alt == 0 {
			if a.termPointerToMouse(tp, ev) {
				tp.focusIt = true
				return true
			}
		}
		if ev.Button != 0 {
			return false // 右键交给上下文菜单
		}
		tp.focusIt = true
		p := a.termCell(tp, ev.X, ev.Y)
		// 点在左侧留白里的命令成败点 = 复制这条命令的输出(正文的指针处理在子控件之前,
		// 所以这里直接判断,不依赖圆点按钮自己的点击)
		if ev.X < padXTerm && tp.scr != nil {
			if d := a.rowDeco(tp, p.row, tp.scr.Cols); d.cmdCopy != nil {
				d.cmdCopy()
				return true
			}
		}
		if ev.Mods&ui.Super != 0 {
			if l := a.linkAt(tp, p); l != nil {
				a.openLink(l)
				return true
			}
		}
		switch {
		case ev.Clicks >= 3:
			tp.selectLine(p)
		case ev.Clicks == 2:
			tp.wordAt(p)
		default:
			tp.sel = termSel{a: p, b: p, drag: true}
		}
		return true
	case ui.InputPointerMove:
		if tp.t.MouseTracking() && ev.Mods&ui.Alt == 0 {
			if a.termPointerToMouse(tp, ev) {
				return true
			}
		}
		p := a.termCell(tp, ev.X, ev.Y)
		tp.hover, tp.hoverOn, tp.hoverMods = p, true, ev.Mods
		tp.link = nil
		if ev.Mods&ui.Super != 0 {
			tp.link = a.linkAt(tp, p)
		}
		if tp.sel.drag && ev.Button == 0 {
			tp.sel.b = p
			tp.sel.on = tp.sel.a != p
			return true
		}
		return false
	case ui.InputPointerUp:
		if tp.t.MouseTracking() && ev.Mods&ui.Alt == 0 {
			if a.termPointerToMouse(tp, ev) {
				return true
			}
		}
		if tp.sel.drag {
			tp.sel.drag = false
			return true
		}
		return false
	case ui.InputText:
		tp.pend, tp.compose, tp.preedit = ui.KeyUnknown, false, ""
		if s := printable(ev.Text); s != "" {
			tp.sel.on = false
			tp.t.SendText(s)
			return true
		}
	case ui.InputCompose:
		tp.compose, tp.preedit = ev.Text != "", ev.Text
		tp.pend = ui.KeyUnknown
		return true
	case ui.InputCommand:
		switch ev.Text {
		case "copy":
			if s := tp.selectedText(); s != "" {
				a.copyText = s
				a.toast = "已复制选中的终端文字"
				return true
			}
			return false
		case "selectAll":
			tp.selectAll()
			return true
		}
		if ev.Text == "paste" {
			if s := c.ReadClipboard(); s != "" {
				tp.t.Paste(s)
			}
			return true
		}
	case ui.InputKeyDown:
		if ev.Mods&ui.Super != 0 && ev.Mods&^ui.Super == 0 && ev.Key == ui.KeyC && !tp.compose {
			// ⌘C 兜底:正常由系统「拷贝」菜单变成 copy 命令;按键直接到达(合成按键、菜单没接管)时,
			// 有选区也要复制。重复复制无副作用,所以不怕和菜单各触发一次;没选区不拦,留给窗口
			if s := tp.selectedText(); s != "" {
				a.copyText = s
				a.toast = "已复制选中的终端文字"
				return true
			}
		}
		if ev.Mods&ui.Super != 0 || tp.compose {
			return false // ⌘ 组合留给窗口快捷键;组字期间的按键(回车提交、退格、选词)留给输入法
		}
		if code, text, ok := termKey(ev.Key, ev.Mods); ok {
			tp.sel.on = false
			tp.t.SendKey(code, text, termMods(ev.Mods))
			return true
		}
		if ev.Mods&(ui.Ctrl|ui.Alt) == 0 {
			if _, ok := typedChar(ev.Key, false); ok {
				tp.pend, tp.pendSh = ev.Key, ev.Mods&ui.Shift != 0
				a.flushPendLater()
			}
		}
	case ui.InputKeyUp:
		// 系统没给文字事件的打字键(合成按键、个别输入源):按美式键盘补发。组字期间不补。
		if k := tp.pend; k != ui.KeyUnknown && k == ev.Key {
			tp.pend = ui.KeyUnknown
			if !tp.compose {
				if ch, ok := typedChar(k, tp.pendSh); ok {
					tp.t.SendText(string(ch))
					return true
				}
			}
		}
	}
	return false
}

// flushPendLater 安排一次超时补发:系统正常会在按键之后立刻给文字事件(清掉 pend);
// 40 ms 后还没来(合成按键、没有文字事件的输入源),就按美式键盘补发这个字符。
func (a *App) flushPendLater() {
	tp := a.ide.tpc()
	tp.pendSeq++
	if a.win == nil {
		return // 测试里没有窗口:靠 KeyUp 补发
	}
	seq := tp.pendSeq
	time.AfterFunc(40*time.Millisecond, func() {
		a.win.Update(func() {
			tp := a.ide.tpc()
			if tp.t == nil || tp.pend == ui.KeyUnknown || tp.pendSeq != seq || tp.compose {
				return
			}
			k := tp.pend
			tp.pend = ui.KeyUnknown
			if ch, ok := typedChar(k, tp.pendSh); ok {
				tp.t.SendText(string(ch))
			}
		})
	})
}

// typedChar 是美式键盘上某个键(带或不带 Shift)打出的字符。
func typedChar(k ui.Key, shift bool) (rune, bool) {
	switch {
	case k >= ui.KeyA && k <= ui.KeyZ:
		if shift {
			return 'A' + rune(k-ui.KeyA), true
		}
		return 'a' + rune(k-ui.KeyA), true
	case k >= ui.Key0 && k <= ui.Key9:
		if shift {
			return rune(")!@#$%^&*("[k-ui.Key0]), true
		}
		return '0' + rune(k-ui.Key0), true
	}
	pairs := map[ui.Key][2]rune{
		ui.KeySpace: {' ', ' '}, ui.KeyMinus: {'-', '_'}, ui.KeyEqual: {'=', '+'}, ui.KeyComma: {',', '<'},
		ui.KeyPeriod: {'.', '>'}, ui.KeySlash: {'/', '?'}, ui.KeySemicolon: {';', ':'}, ui.KeyQuote: {'\'', '"'},
		ui.KeyBracketLeft: {'[', '{'}, ui.KeyBracketRight: {']', '}'}, ui.KeyBackslash: {'\\', '|'}, ui.KeyBackquote: {'`', '~'},
	}
	if p, ok := pairs[k]; ok {
		if shift {
			return p[1], true
		}
		return p[0], true
	}
	return 0, false
}

// printable 去掉控制字符(回车、退格等由按键事件处理,免得重复发送)。
func printable(s string) string {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return ""
		}
	}
	return s
}

func termMods(m ui.Modifiers) uv.KeyMod {
	var o uv.KeyMod
	if m&ui.Shift != 0 {
		o |= uv.ModShift
	}
	if m&ui.Ctrl != 0 {
		o |= uv.ModCtrl
	}
	if m&ui.Alt != 0 {
		o |= uv.ModAlt
	}
	return o
}

var termSpecial = map[ui.Key]rune{
	ui.KeyEnter: uv.KeyEnter, ui.KeyEscape: uv.KeyEscape, ui.KeyBackspace: uv.KeyBackspace, ui.KeyTab: uv.KeyTab,
	ui.KeyDelete: uv.KeyDelete, ui.KeyInsert: uv.KeyInsert, ui.KeyHome: uv.KeyHome, ui.KeyEnd: uv.KeyEnd,
	ui.KeyPageUp: uv.KeyPgUp, ui.KeyPageDown: uv.KeyPgDown,
	ui.KeyLeft: uv.KeyLeft, ui.KeyRight: uv.KeyRight, ui.KeyUp: uv.KeyUp, ui.KeyDown: uv.KeyDown,
	ui.KeyF1: uv.KeyF1, ui.KeyF2: uv.KeyF2, ui.KeyF3: uv.KeyF3, ui.KeyF4: uv.KeyF4, ui.KeyF5: uv.KeyF5, ui.KeyF6: uv.KeyF6,
	ui.KeyF7: uv.KeyF7, ui.KeyF8: uv.KeyF8, ui.KeyF9: uv.KeyF9, ui.KeyF10: uv.KeyF10, ui.KeyF11: uv.KeyF11, ui.KeyF12: uv.KeyF12,
}

// termKey 决定一个 KeyDown 要不要由这里发给终端:功能键始终发;Ctrl / Alt 与字母、空格组合也发
// (Ctrl-C、Alt-B 等);其余会打出字的按键不处理,等随后的 InputText 事件。
func termKey(k ui.Key, m ui.Modifiers) (code rune, text string, ok bool) {
	if r, found := termSpecial[k]; found {
		return r, "", true
	}
	if m&(ui.Ctrl|ui.Alt) == 0 {
		return 0, "", false
	}
	switch {
	case k >= ui.KeyA && k <= ui.KeyZ:
		return 'a' + rune(k-ui.KeyA), "", true
	case k >= ui.Key0 && k <= ui.Key9:
		return '0' + rune(k-ui.Key0), "", true
	case k == ui.KeySpace:
		return ' ', "", true
	case k == ui.KeyBracketLeft:
		return '[', "", true
	case k == ui.KeyBracketRight:
		return ']', "", true
	case k == ui.KeyBackslash:
		return '\\', "", true
	case k == ui.KeySlash:
		return '/', "", true
	case k == ui.KeyMinus:
		return '-', "", true
	case k == ui.KeyPeriod:
		return '.', "", true
	case k == ui.KeyComma:
		return ',', "", true
	}
	return 0, "", false
}

// ---- 颜色与文字段 ----

var (
	ansiDark = [16]string{"#6e7681", "#ff7b72", "#7ee787", "#e3b341", "#79c0ff", "#d2a8ff", "#76e3ea", "#c9d1d9",
		"#8b949e", "#ffa198", "#56d364", "#f2cc60", "#a5d6ff", "#e2c5ff", "#b3f0ff", "#f0f6fc"}
	ansiLight = [16]string{"#24292f", "#cf222e", "#1a7f37", "#9a6700", "#0969da", "#8250df", "#1b7c83", "#6e7781",
		"#57606a", "#a40e26", "#2da44e", "#bf8700", "#218bff", "#a475f9", "#3192aa", "#8c959f"}
)

// ansiColor 把调色板索引换成颜色:0-15 跟随深浅色主题,16-231 是 6×6×6 色立方,232-255 是灰阶。
func ansiColor(dark bool, i uint8) ui.Color {
	switch {
	case i < 16:
		if dark {
			return ui.Hex(ansiDark[i])
		}
		return ui.Hex(ansiLight[i])
	case i < 232:
		n := int(i) - 16
		lv := func(v int) uint8 {
			if v == 0 {
				return 0
			}
			return uint8(55 + v*40)
		}
		return ui.RGB(lv(n/36), lv(n/6%6), lv(n%6))
	default:
		g := uint8(8 + (int(i)-232)*10)
		return ui.RGB(g, g, g)
	}
}

// termColor 解析一个终端颜色;默认色返回 def。
func termColor(t *theme.Tokens, c term.Color, def ui.Color) ui.Color {
	switch c.Kind {
	case 1:
		return ansiColor(t.Dark(), c.Idx)
	case 2:
		return ui.RGB(c.R, c.G, c.B)
	}
	return def
}

// termRow 画一行:先是每段的背景色块(绝对定位,按列对齐,整行高度),再是文字,最后是光标块。
// 背景不用文字片段自带的 Background(它只贴着字形,上下有缝,反色整行的状态栏会很难看)。
func (a *App) termRow(c *ui.Context, ln term.Line, cx int, focused bool, cellW float32, preedit string, deco rowDeco) {
	t := a.tk
	ui.Row(c).Height(termLineH).Shrink(0).AlignItems(ui.Center).Children(func() {
		col := 0
		for _, r := range ln {
			w := ansi.StringWidth(r.Text)
			if bg, ok := runBackground(t, r); ok {
				ui.Box(c).Absolute().Left(float32(col)*cellW).Top(0).Size(float32(w)*cellW, termLineH).Background(bg)
			}
			col += w
		}
		// 选区 / 搜索命中 / 链接下划线盖在背景之上、文字之下
		if deco.sel[1] > deco.sel[0] {
			ui.Box(c).Absolute().Left(float32(deco.sel[0])*cellW).Top(0).Size(float32(deco.sel[1]-deco.sel[0])*cellW, termLineH).Background(t.Ac.Alpha(.32))
		}
		if deco.cmd != 0 {
			// 命令成败点:画在行首留白处(不压提示符),悬停写退出码,点击复制这条命令的输出
			col, tip := t.Ok, "命令成功 · 点击复制输出"
			if deco.cmd < 0 {
				col, tip = t.Err, "命令失败(退出码 "+itoa(deco.cmdExit)+")· 点击复制输出"
			}
			ui.ButtonBase(c.Key("cmdmark")).Absolute().Left(-padXTerm).Top(0).Size(padXTerm, termLineH).Padding(0).
				Tooltip(tip).OnClick(func() {
				if deco.cmdCopy != nil {
					deco.cmdCopy()
				}
			}).Children(func() {
				ui.Box(c).Size(6, 6).Radius(3).Background(col)
			})
		}
		for _, h := range deco.hits {
			col := t.Warn.Alpha(.38)
			if h.cur {
				col = t.Ac.Alpha(.62)
			}
			ui.Box(c).Absolute().Left(float32(h.c0)*cellW).Top(1).Size(float32(h.c1-h.c0)*cellW, termLineH-2).Radius(2).Background(col)
		}
		if deco.link[1] > deco.link[0] {
			ui.Box(c).Absolute().Left(float32(deco.link[0])*cellW).Top(termLineH-3).Size(float32(deco.link[1]-deco.link[0])*cellW, 1).Background(t.Ac)
		}
		// 文字也放进绝对层:库里绝对定位元素画在同级流内元素之上,不放进来的话
		// 不透明的背景块(反色、带背景色的输出)会把字整个盖住
		if !lineHasWide(ln) {
			ui.Row(c).Absolute().Left(0).Top(0).Height(termLineH).AlignItems(ui.Center).Children(func() {
				ui.RichText(c, termSpans(t, ln)...).NoWrap()
			})
		} else {
			// 含宽字符(中日韩 / 全角):字体里 CJK 的步进是 1em,而格子里占 2 格(1.2em),
			// 整行流式排版会越排越偏,光标 / 选区 / 背景都对不上。所以按格子逐段定位:
			// 窄字符连成一段,每个宽字符单独一段,都落在 列 × 格宽 上。
			for _, sg := range termSegments(t, ln) {
				ui.Row(c).Absolute().Left(float32(sg.col) * cellW).Top(0).Height(termLineH).AlignItems(ui.Center).Children(func() {
					ui.RichText(c, sg.span).NoWrap()
				})
			}
		}
		if cx >= 0 && preedit != "" { // 输入法组字:盖在光标处,带下划线
			ui.RichText(c, ui.Span{Text: preedit, Font: termFamily, Size: termFont, Color: t.Tx1, Underline: true}).NoWrap().
				Absolute().Left(float32(cx) * cellW).Top(0).Height(termLineH).Background(t.Raised)
		} else if cx >= 0 {
			bg := t.Tx1
			if !focused {
				bg = ui.RGBA(t.Tx1.R, t.Tx1.G, t.Tx1.B, 0.35)
			} else {
				bg = ui.RGBA(t.Tx1.R, t.Tx1.G, t.Tx1.B, 0.55)
			}
			blink := a.prefs.Bool("term.blink", true) && focused
			cur := ui.Box(c).Absolute().Left(float32(cx) * cellW)
			switch a.prefs.Int("term.cursor", 0) {
			case 1: // 竖线
				cur.Top(1).Size(2, termLineH-2)
			case 2: // 下划线
				cur.Top(termLineH-3).Size(cellW, 2)
			default: // 块
				cur.Top(0).Size(cellW, termLineH)
			}
			cur.Background(bg)
			if blink && !a.cursorBlinkOn() { // 闪烁:前半周期亮,后半周期暗
				cur.Opacity(.12)
			}
		}
	})
}

// runBackground 是一段的背景色(反色时取前景);默认背景返回 false。
func runBackground(t *theme.Tokens, r term.Run) (ui.Color, bool) {
	if r.Reverse {
		return termColor(t, r.Fg, t.Tx1), true
	}
	if r.Bg.Kind == 0 {
		return ui.Color{}, false
	}
	return termColor(t, r.Bg, ui.Color{}), true
}

// runSpan 是一段样式对应的富文本片段(只管文字与前景色)。text 可以是这一段的一部分。
func runSpan(t *theme.Tokens, r term.Run, text string) ui.Span {
	fg := termColor(t, r.Fg, t.Tx1)
	if r.Reverse { // 反色:前景取原背景(默认背景用面板底色)
		fg = termColor(t, r.Bg, t.BG)
	}
	if r.Dim {
		fg.A = uint8(float32(fg.A) * 0.6)
	}
	sp := ui.Span{Text: text, Font: termFamily, Size: termFont, Color: fg,
		Italic: r.Italic, Underline: r.Underline, Strikethrough: r.Strike}
	if r.Bold {
		sp.Weight = 700
	}
	return sp
}

// lineHasWide 这一行有没有占 2 格的字符。
func lineHasWide(ln term.Line) bool {
	for _, r := range ln {
		for _, ch := range r.Text {
			if ch >= 0x1100 && ansi.StringWidth(string(ch)) == 2 {
				return true
			}
		}
	}
	return false
}

// rowSeg 是落在固定列上的一小段文字。
type rowSeg struct {
	col  int
	span ui.Span
}

// termSegments 把一行切成「窄字符连续段 + 单个宽字符」,每段带起始列。零宽字符(组合符)并进前一段。
func termSegments(t *theme.Tokens, ln term.Line) []rowSeg {
	var out []rowSeg
	col := 0
	for _, r := range ln {
		var cur []rune
		curCol := col
		flush := func() {
			if len(cur) > 0 {
				out = append(out, rowSeg{col: curCol, span: runSpan(t, r, string(cur))})
				cur = cur[:0]
			}
		}
		for _, ch := range r.Text {
			w := ansi.StringWidth(string(ch))
			switch {
			case w == 0: // 组合符 / 变体选择符:跟着前一个字
				cur = append(cur, ch)
			case w >= 2:
				flush()
				out = append(out, rowSeg{col: col, span: runSpan(t, r, string(ch))})
				col += w
				curCol = col
			default:
				if len(cur) == 0 {
					curCol = col
				}
				cur = append(cur, ch)
				col += w
			}
		}
		flush()
	}
	return out
}

// termSpans 把一行的样式段变成富文本片段(只管文字与前景色)。
func termSpans(t *theme.Tokens, ln term.Line) []ui.Span {
	out := make([]ui.Span, 0, len(ln)+1)
	for _, r := range ln {
		out = append(out, runSpan(t, r, r.Text))
	}
	if len(out) == 0 {
		out = append(out, ui.Span{Text: " ", Font: termFamily, Size: termFont})
	}
	return out
}

// addOverlay 登记一个画在正文上的控件区域(窗口坐标,存成相对正文的)。
func (tp *termPanel) addOverlay(r ui.Rect) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	tp.over = append(tp.over, ui.Rect{X: r.X - tp.bodyR.X, Y: r.Y - tp.bodyR.Y, W: r.W, H: r.H})
}

// overlayHit 判断正文内坐标 (x, y) 是否落在某个覆盖控件上。
func (tp *termPanel) overlayHit(x, y float32) bool {
	for _, r := range tp.over {
		if x >= r.X && x <= r.X+r.W && y >= r.Y && y <= r.Y+r.H {
			return true
		}
	}
	return false
}

// cursorBlinkPeriod 光标亮 / 暗各占多久。
const cursorBlinkPeriod = 550 * time.Millisecond

// cursorBlinkOn 光标此刻是否该亮,并只在下一次切换的那一刻请求一帧。
// 以前用 Loop 动画,整个界面每帧(60 fps)重建一遍,空闲时 CPU 常年 10%;改成离散切换后空闲时每秒只出两帧。
func (a *App) cursorBlinkOn() bool {
	now := time.Now()
	ms := now.UnixMilli()
	per := cursorBlinkPeriod.Milliseconds()
	on := (ms/per)%2 == 0
	if a.win != nil && !a.blinkPending {
		a.blinkPending = true
		wait := time.Duration(per-ms%per) * time.Millisecond
		time.AfterFunc(wait+time.Millisecond, func() {
			a.win.Update(func() { a.blinkPending = false })
		})
	}
	return on
}
