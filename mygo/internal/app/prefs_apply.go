package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// 设置页里真正驱动行为的项(其余控件仍只存偏好,见 AGENTS.md「已知缺口」)。
// 键名与 settings_tabs.go 里的声明一一对应;默认值也必须和那里一致(setPref 在值等于默认时会删键)。
const (
	pMotion      = "appearance.motion"
	pMorph       = "appearance.morph"
	pShimmer     = "appearance.shimmer"
	pFontScale   = "appearance.fontScale"
	pThinkOpen   = "chat.thinkOpen"
	pStats       = "chat.stats"
	pGroupFold   = "chat.groupFold"
	pSend        = "general.send"
	pBusy        = "general.busy"
	pEdFont      = "editor.fontSize"
	pEdIndent    = "editor.indent"
	pEdAutosave  = "editor.autosave"
	pMinimap     = "editor.minimap"
	pEdNumbers   = "editor.lineNumbers"
	pEdFolding   = "editor.folding"
	pEdWrap      = "editor.wrap"
	pBlameInline = "editor.blameInline"
	pTermFont    = "term.fontSize"
)

// edScale 是编辑器字号相对 12px 的倍数(浮层定位用的字宽 / 行高按它缩放)。
var edScale float32 = 1

// applyBehavior 把会立刻影响行为的偏好套到运行时(applyPrefs 末尾调用)。
func (a *App) applyBehavior() {
	on := a.prefs.Bool(pMotion, true)
	kit.SetMotion(on)
	kit.SetPulse(a.prefs.Int(pShimmer, 0) != 2)
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
	// 终端字号:改了之后各终端标签重新量字宽
	fam := a.prefs.Str("term.font", "")
	if fam == "" {
		fam = "monospace"
	}
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
}

// applyEditorPrefs 把编辑器偏好套到一个编辑器状态(每帧调用,值没变时什么都不做)。
func (a *App) applyEditorPrefs(tb *ideTab) {
	st := tb.st
	if st == nil {
		return
	}
	size := a.prefs.Int(pEdFont, 12)
	if a.prefs.Has(pEdFont) {
		st.Font = ui.Font{Family: "monospace", Size: float32(size)}
	} else {
		st.Font = ui.Font{}
	}
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
}

// autosave 自动保存:「延时」= 最后一次修改 1.5 秒后写盘;「失焦」= 切到别的标签时写盘。
// 每帧调用一次,只读标志与时间戳。
func (a *App) autosave() {
	d := a.ide
	mode := a.prefs.Int(pEdAutosave, 1)
	cur := d.cur()
	if mode == 2 && d.lastActive != "" && (cur == nil || cur.id != d.lastActive) {
		if prev := d.tab(d.lastActive); prev != nil && prev.dirty() && !prev.diff && prev.virt == "" {
			a.saveTab(prev)
		}
	}
	if cur != nil {
		d.lastActive = cur.id
	} else {
		d.lastActive = ""
	}
	if mode != 1 || cur == nil || cur.diff || cur.virt != "" || cur.buf == nil {
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
