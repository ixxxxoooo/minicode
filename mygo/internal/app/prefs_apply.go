package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 设置页里真正驱动行为的项(其余控件仍只存偏好,见 AGENTS.md「已知缺口」)。
// 键名与 settings_tabs.go 里的声明一一对应;默认值也必须和那里一致(setPref 在值等于默认时会删键)。
const (
	pMotion        = "appearance.motion"
	pMorph         = "appearance.morph"
	pShimmer       = "appearance.shimmer"
	pNavDrop       = "appearance.navDrop" // 导航栏列表落下速度:0 慢,1 标准(默认),2 快,3 无
	pFontScale     = "appearance.fontScale"
	pUIScale       = "appearance.uiScale" // 界面大小(百分数,默认 110)
	pThinkOpen     = "chat.thinkOpen"
	pStats         = "chat.stats"
	pGroupFold     = "chat.groupFold"
	pSend          = "general.send"
	pBusy          = "general.busy"
	pEdFont        = "editor.fontSize"
	pEdIndent      = "editor.indent"
	pEdAutosave    = "editor.autosave"
	pEdSyntax      = "editor.syntax"
	pMinimap       = "editor.minimap"
	pMmSlider      = "editor.minimapSlider"
	pMmWidth       = "editor.minimapWidth"
	pMmGit         = "editor.minimapGit"
	pBracketGuide  = "editor.bracketGuide"
	pBracketColor  = "editor.bracketColor"
	pBracketMatch  = "editor.bracketMatch"
	pEdScope       = "editor.scopeHighlight"
	pEdScopeStyle  = "editor.scopeStyle"
	pEdNumbers     = "editor.lineNumbers"
	pEdFolding     = "editor.folding"
	pEdWrap        = "editor.wrap"
	pMdView        = "editor.mdView"       // Markdown 默认视图:0 所见即所得,1 源码
	pFmtSave       = "editor.formatOnSave" // 保存时格式化
	pTrimSave      = "editor.trimOnSave"   // 保存时去掉行尾空白
	pBlameInline   = "editor.blameInline"
	pTermFont      = "term.fontSize"
	pTermIntegrate = "term.shellIntegration"
	pTermLogin     = "term.login"
	pTermCwd       = "term.cwd"
)

// edScale 是编辑器字号相对 12px 的倍数(浮层定位用的字宽 / 行高按它缩放)。
var edScale float32 = 1

// ChatFontScale 是会话流正文 Markdown 的字号倍率(appearance.fontScale,默认 1)。
func (a *App) ChatFontScale() float32 {
	if a == nil || a.prefs == nil {
		return 1
	}
	return float32(a.prefs.Int(pFontScale, 100)) / 100
}

// FlowFontSize 是会话流正文与助手 Markdown 的字号(统一走 ChatFontScale)。
func (a *App) FlowFontSize() float32 { return theme.FsRead * a.ChatFontScale() }

// FlowCapFont 是会话流辅助文案(步骤行附注、子 Agent 摘要等)。
func (a *App) FlowCapFont() float32 { return theme.FsCap * a.ChatFontScale() }

// FlowMonoSize 是会话流等宽输出与代码块的基准字号(随会话正文字号缩放,不跟编辑器/终端)。
func (a *App) FlowMonoSize() float32 { return theme.FsMono * a.ChatFontScale() }

// ToolDetailLevel 是工具调用详细程度(0 精简 / 1 标准 / 2 详细);会话流读取点见 chat/flow。
func (a *App) ToolDetailLevel() int {
	if a == nil || a.prefs == nil {
		return 1
	}
	return a.prefs.Int("appearance.density2", 1)
}

// applyBehavior 把会立刻影响行为的偏好套到运行时(applyPrefs 末尾调用)。
func (a *App) applyBehavior() {
	on := a.prefs.Bool(pMotion, true) && !kit.SystemReduceMotion()
	kit.SetMotion(on)
	kit.SetScrollbarMode(a.prefs.Int(pScrollbar, 1))
	kit.SetShimmer(a.prefs.Int(pShimmer, 0))
	// 形态切换动画:流畅 = 440 ms;简化 = 160 ms;无 = 立即(动效整体关闭时也是立即)
	d := kit.DurMorph
	switch a.prefs.Int(pMorph, 0) {
	case 1:
		d = 160 * time.Millisecond
	case 2:
		d = 0
	}
	if !on {
		d = 0
	}
	morph = ui.ElementTransition{Size: true, Position: true, Duration: d, Ease: kit.EaseMorph}
	a.applyCheapPrefs()
	a.applySettingsRuntime()
	// 终端字号:改了之后各终端标签重新量字宽
	fam := a.termFontFamily()
	lh := []float32{1.4, 1.6, 1.8}[min(max(a.prefs.Int("term.lineHeight", 1), 0), 2)]
	f := float32(a.prefs.Int(pTermFont, 12)) - .5
	if f != termFont || fam != termFamily || f*lh != termLineH {
		termFont, termLineH, termFamily = f, f*lh, fam
		for _, tp := range a.ide.tps {
			tp.cellW = 0
		}
	}
	edScale = float32(a.prefs.Int(pEdFont, 12)) / 12
	// 设置里改了 Diff 默认视图:已开的差异标签放弃各自的选择,统一跟随新默认
	if v := a.prefs.Int("scm.diff", 0); v != a.diffPref {
		a.diffPref = v
		for _, tb := range a.ide.tabs {
			tb.lay = 0
		}
	}
	a.syncModelEffortPrefs()
}

// applyEditorPrefs 把编辑器偏好套到一个编辑器状态(每帧调用,值没变时什么都不做)。
func (a *App) applyEditorPrefs(tb *ideTab) {
	st := tb.st
	if st == nil {
		return
	}
	size := a.prefs.Int(pEdFont, 12)
	// 没改过字号时也要显式写成默认 12:库自带的默认是 13,和行高 / 浮层定位按 12 算的常量对不上
	st.Font = ui.Font{Family: "monospace", Size: float32(size)}
	switch a.prefs.Int(pEdIndent, 1) {
	case 0:
		st.TabSize, st.Indent = 2, editor.IndentSpaces
	case 1:
		st.TabSize, st.Indent = 4, editor.IndentSpaces
	default:
		st.TabSize, st.Indent = 4, editor.IndentTabs
	}
	// 默认「4 空格」只在用户改过时才覆盖语言自己的习惯(Go 用 Tab)
	if !a.prefs.Has(pEdIndent) {
		st.TabSize, st.Indent = 0, editor.IndentAuto
	}
	st.SoftWrap = a.prefs.Bool(pEdWrap, false)
	if a.prefs.Bool(pEdNumbers, true) {
		st.LineNumbers = editor.LineNumbersOn
	} else {
		st.LineNumbers = editor.LineNumbersOff
	}
	st.NoFolding = !a.prefs.Bool(pEdFolding, true)
	st.NoIndentGuides = !a.prefs.Bool(pBracketGuide, true)
	if !a.prefs.Bool(pBracketMatch, true) {
		if st.Theme == nil {
			st.Theme = a.editorTheme()
		}
		if st.Theme.MatchingBracket != ui.Transparent {
			th := *st.Theme // 值拷贝:不改动共享的缓存配色
			th.MatchingBracket = ui.Transparent
			st.Theme = &th
		}
	}
	// 语法高亮关闭:词法着色全部取正文色(语言仍在,注释 / 括号 / 缩进行为不受影响)
	if !a.prefs.Bool(pEdSyntax, true) {
		if st.Theme == nil {
			st.Theme = a.editorTheme()
		}
		// 不能置零:库会把零值颜色补成它自带的默认主题色,所以逐项显式写成正文色
		var plain [len(st.Theme.Tokens)]editor.TokenStyle
		for k := range plain {
			plain[k] = editor.TokenStyle{Color: st.Theme.Text}
		}
		if st.Theme.Tokens != plain {
			th := *st.Theme
			th.Tokens = plain
			st.Theme = &th
		}
	}
	switch a.prefs.Int(pScrollbar, 1) {
	case kit.ScrollbarAlways:
		st.KeepScrollbars = true
	default:
		st.KeepScrollbars = false
	}
}

// autosave 自动保存:「延时」= 最后一次修改 1.5 秒后写盘;「失焦」= 切到别的标签时写盘。
// 每帧调用一次,只读标志与时间戳。
func (a *App) autosave() {
	d := a.ide
	mode := a.prefs.Int(pEdAutosave, 1)
	cur := d.cur()
	if mode == 2 && d.lastActive != "" && (cur == nil || cur.id != d.lastActive) {
		if prev := d.tab(d.lastActive); prev != nil && prev.dirty() && !prev.diff && prev.virt == "" && !prev.extKeep && !prev.extOrphan && !prev.diskRO {
			a.saveTab(prev)
		}
	}
	if cur != nil {
		d.lastActive = cur.id
	} else {
		d.lastActive = ""
	}
	if mode != 1 || cur == nil || cur.diff || cur.virt != "" || cur.buf == nil || cur.extKeep || cur.extOrphan || cur.diskRO {
		return
	}
	v := cur.buf.Version()
	if cur.asVer != v {
		cur.asVer, cur.asAt = v, time.Now()
		return
	}
	if cur.dirty() && !cur.asAt.IsZero() && time.Since(cur.asAt) >= 1500*time.Millisecond {
		a.saveTab(cur)
		cur.asAt = time.Time{}
	} else if cur.dirty() && a.win != nil {
		a.redrawSoon() // 还没到点:稍后再看一眼
	}
}

// saveTab 保存指定标签(autosave 用;失败只记日志)。
func (a *App) saveTab(t *ideTab) {
	prev := d2active(a.ide, t)
	defer prev()
	if err := a.ide.save(); err != nil {
		ideLog.Warn("自动保存失败", "path", t.path, "err", err)
	}
}

// d2active 临时把活动标签切到 t,返回还原函数(save 只保存活动标签)。
func d2active(d *ide, t *ideTab) func() {
	old := d.active
	d.active = t.id
	return func() { d.active = old }
}

// edFontDefault 是编辑器字号的默认值(与设置页「字号」一致)。
const edFontDefault = 12

// setEditorFont 改编辑器字号(9–24):⌘+ / ⌘- / 状态栏芯片 / 命令面板共用;等于默认值时删键,设置页的「已修改」点随之消失。
func (a *App) setEditorFont(n int) {
	n = min(max(n, 9), 24)
	if n == edFontDefault {
		a.prefs.Reset(pEdFont)
	} else {
		a.prefs.Set(pEdFont, n)
	}
	a.applyBehavior()
	a.toast = "编辑器字号 " + itoa(n) + " px"
	log.Info("调整编辑器字号", "px", n)
}

// resetEditorFont 把编辑器字号恢复成默认(⌘0)。
func (a *App) resetEditorFont() { a.setEditorFont(edFontDefault) }
