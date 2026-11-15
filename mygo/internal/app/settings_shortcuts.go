package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// shortcutSpec 是一条可录制的快捷键。
type shortcutSpec struct {
	id, title, group, def string
}

var shortcutTable = []shortcutSpec{
	{id: "palette", title: "命令面板", group: "全局", def: "⌘ K"},
	{id: "newChat", title: "新对话", group: "全局", def: "⌘ N"},
	{id: "newWin", title: "新窗口", group: "全局", def: "⇧ ⌘ N"},
	{id: "settings", title: "设置", group: "全局", def: "⌘ ,"},
	{id: "modeCycle", title: "循环权限模式", group: "全局", def: "⇧ ⇥"},
	{id: "toggleMode", title: "切换 Agent / IDE", group: "形态与面板", def: "⌘ E"},
	{id: "nav", title: "显示 / 隐藏导航栏", group: "形态与面板", def: "⌘ B"},
	{id: "right", title: "显示 / 隐藏检查器", group: "形态与面板", def: "⌥ ⌘ B"},
	{id: "panel", title: "底部面板", group: "形态与面板", def: "⌘ J"},
	{id: "scm", title: "源代码管理", group: "形态与面板", def: "⌃ ⇧ G"},
	{id: "find", title: "当前文件查找", group: "编辑器", def: "⌘ F"},
	{id: "search", title: "项目内搜索", group: "形态与面板", def: "⇧ ⌘ F"},
	{id: "symbols", title: "转到符号", group: "编辑器", def: "⇧ ⌘ O"},
	{id: "save", title: "保存", group: "编辑器", def: "⌘ S"},
	{id: "comment", title: "行注释", group: "编辑器", def: "⌘ /"},
	{id: "format", title: "格式化文档", group: "编辑器", def: "⇧ ⌥ F"},
	{id: "codeAction", title: "源代码操作", group: "编辑器", def: "⌘ ."},
	{id: "goDef", title: "转到定义", group: "编辑器", def: "F12"},
	{id: "findRefs", title: "查找所有引用", group: "编辑器", def: "⇧ F12"},
	{id: "splitRight", title: "向右拆分", group: "编辑器", def: "⌘ \\"},
	{id: "wrap", title: "切换自动换行", group: "编辑器", def: "⌥ Z"},
	{id: "reopenTab", title: "重新打开已关闭标签", group: "编辑器", def: "⇧ ⌘ T"},
	{id: "termFind", title: "终端查找", group: "终端", def: "⌘ F"},
	{id: "termClear", title: "终端清屏", group: "终端", def: "⌘ K"},
	{id: "guide", title: "立即引导", group: "对话", def: "⌘ ⏎"},
}

func prefKeyShortcut(id string) string { return "shortcuts." + id }

func (a *App) shortcutLabel(id string) string {
	if s := a.prefs.Str(prefKeyShortcut(id), ""); s != "" {
		return s
	}
	for _, sp := range shortcutTable {
		if sp.id == id {
			return sp.def
		}
	}
	return ""
}

// comboMatches 判断窗口快捷键是否匹配录制结果(如 "⌘ K")。
func comboMatches(label string, mods ui.Modifiers, key ui.Key) bool {
	parts := strings.Fields(label)
	wantMod := ui.Modifiers(0)
	var wantKey ui.Key
	for _, p := range parts {
		switch p {
		case "⌘", "Cmd", "cmd":
			wantMod |= ui.Cmd
		case "⇧", "Shift", "shift":
			wantMod |= ui.Shift
		case "⌥", "Alt", "alt", "Option":
			wantMod |= ui.Alt
		case "⌃", "Ctrl", "ctrl", "Control":
			wantMod |= ui.Ctrl
		default:
			wantKey = keyName(p)
		}
	}
	if wantKey == 0 {
		wantKey = key
	}
	return mods == wantMod && key == wantKey
}

func keyName(s string) ui.Key {
	switch strings.ToLower(s) {
	case "k":
		return ui.KeyK
	case "n":
		return ui.KeyN
	case "e":
		return ui.KeyE
	case "j":
		return ui.KeyJ
	case "f":
		return ui.KeyF
	case "s":
		return ui.KeyS
	case "o":
		return ui.KeyO
	case "b":
		return ui.KeyB
	case "tab":
		return ui.KeyTab
	case "esc", "escape":
		return ui.KeyEscape
	case "enter", "⏎", "return":
		return ui.KeyEnter
	case ",":
		return ui.KeyComma
	case "/":
		return ui.KeySlash
	case "`":
		return ui.KeyBackquote
	case "↑":
		return ui.KeyUp
	default:
		return 0
	}
}

func formatCombo(mods ui.Modifiers, key ui.Key) string {
	var parts []string
	if mods&ui.Cmd != 0 {
		parts = append(parts, "⌘")
	}
	if mods&ui.Shift != 0 {
		parts = append(parts, "⇧")
	}
	if mods&ui.Alt != 0 {
		parts = append(parts, "⌥")
	}
	if mods&ui.Ctrl != 0 {
		parts = append(parts, "⌃")
	}
	parts = append(parts, keyLabel(key))
	return strings.Join(parts, " ")
}

func keyLabel(k ui.Key) string {
	switch k {
	case ui.KeyK:
		return "K"
	case ui.KeyN:
		return "N"
	case ui.KeyE:
		return "E"
	case ui.KeyJ:
		return "J"
	case ui.KeyF:
		return "F"
	case ui.KeyS:
		return "S"
	case ui.KeyO:
		return "O"
	case ui.KeyB:
		return "B"
	case ui.KeyTab:
		return "Tab"
	case ui.KeyEscape:
		return "esc"
	case ui.KeyEnter:
		return "⏎"
	case ui.KeyComma:
		return ","
	case ui.KeySlash:
		return "/"
	case ui.KeyBackquote:
		return "`"
	case ui.KeyUp:
		return "↑"
	default:
		return fmt.Sprintf("%d", k)
	}
}

func (a *App) shortcutConflict(id, combo string) string {
	if combo == "" {
		return ""
	}
	for _, sp := range shortcutTable {
		if sp.id == id {
			continue
		}
		cur := a.shortcutLabel(sp.id)
		if cur == "" {
			cur = sp.def
		}
		if cur == combo {
			return sp.title
		}
	}
	return ""
}

// recordShortcut 结束录制并写入偏好。
func (a *App) recordShortcut(mods ui.Modifiers, key ui.Key) {
	id := a.set.recID
	if id == "" || key == 0 {
		return
	}
	combo := formatCombo(mods, key)
	if other := a.shortcutConflict(id, combo); other != "" {
		a.set.kbWarn = "与「" + other + "」冲突"
		a.set.recID = ""
		return
	}
	a.prefs.Set(prefKeyShortcut(id), combo)
	a.set.kbWarn = ""
	a.set.recID = ""
	setLog.Info("快捷键已录制", "id", id, "combo", combo)
}

// shortcutEvent 判断当前按键是否匹配录制的组合。
func shortcutEvent(c *ui.Context, label string) bool {
	parts := strings.Fields(label)
	var mods ui.Modifiers
	var key ui.Key
	for _, p := range parts {
		switch p {
		case "⌘", "Cmd":
			mods |= ui.Cmd
		case "⇧", "Shift":
			mods |= ui.Shift
		case "⌥", "Alt", "Option":
			mods |= ui.Alt
		case "⌃", "Ctrl", "Control":
			mods |= ui.Ctrl
		default:
			key = keyName(p)
		}
	}
	if key == 0 {
		return false
	}
	return c.Shortcut(mods, key)
}

func (a *App) shortcutHit(c *ui.Context, id string, defMods ui.Modifiers, defKey ui.Key) bool {
	if label := a.shortcutLabel(id); label != "" {
		return shortcutEvent(c, label)
	}
	return c.Shortcut(defMods, defKey)
}

var recordBinds = []struct {
	mods ui.Modifiers
	key  ui.Key
}{
	{ui.Cmd, ui.KeyK}, {ui.Cmd, ui.KeyN}, {ui.Cmd, ui.KeyE}, {ui.Cmd, ui.KeyJ},
	{ui.Cmd, ui.KeyF}, {ui.Cmd, ui.KeyS}, {ui.Cmd, ui.KeyO}, {ui.Cmd, ui.KeyB},
	{ui.Cmd, ui.KeyComma}, {ui.Cmd, ui.KeySlash}, {ui.Cmd, ui.KeyTab},
	{ui.Cmd | ui.Shift, ui.KeyN}, {ui.Cmd | ui.Shift, ui.KeyF}, {ui.Cmd | ui.Shift, ui.KeyO},
	{ui.Cmd | ui.Shift, ui.KeyTab}, {ui.Cmd | ui.Shift, ui.KeyM},
	{ui.Cmd | ui.Alt, ui.KeyB}, {ui.Ctrl | ui.Shift, ui.KeyG},
	{ui.Cmd, ui.KeyEnter}, {0, ui.KeyEscape}, {0, ui.KeyUp},
}

// shortcutRecordCapture 录制层:挂在设置页根上捕获下一步按键。
func (a *App) shortcutRecordCapture(c *ui.Context) {
	if a.set.recID == "" {
		return
	}
	ui.Row(c).Gap(8).Padding(8, 12).Margin(0, 0, 8, 0).Background(a.tk.WarnSoft).Radius(a.tk.RSm()).Children(func() {
		ui.Text(c, "正在录制快捷键:按下一组键,或 Esc / 点别处取消").FontSize(theme.FsCap).TextColor(a.tk.Warn).Grow(1)
		kit.Btn(c, "取消", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(func() { a.set.recID = ""; a.set.kbWarn = "" })
	})
	for _, b := range recordBinds {
		b := b
		c.OnShortcut(b.mods, b.key, func() { a.recordShortcut(b.mods, b.key) })
	}
	c.OnShortcut(0, ui.KeyEscape, func() { a.set.recID = ""; a.set.kbWarn = "" })
}

func (a *App) resetShortcuts() {
	for _, sp := range shortcutTable {
		a.prefs.Reset(prefKeyShortcut(sp.id))
	}
	a.set.kbWarn = ""
	setLog.Info("已恢复默认快捷键")
}

// shortcutKeycap 画可点击录制的键帽行。
func (a *App) shortcutKeycap(c *ui.Context, sp shortcutSpec) {
	t := a.tk
	combo := a.shortcutLabel(sp.id)
	if combo == "" {
		combo = sp.def
	}
	rec := a.set.recID == sp.id
	cap := ui.ButtonBase(c).Key("kb-" + sp.id).Height(24).PaddingX(6).Radius(t.RSm()).Cursor(ui.CursorPointer).Shrink(0)
	switch {
	case rec:
		cap.Background(t.WarnSoft).Border(1, t.Warn)
	case cap.Hovered():
		cap.Background(t.Fill2).Border(1, t.Bd3)
	default:
		cap.Border(1, t.Bd2)
	}
	cap.OnClick(func() { a.set.recID = sp.id; a.set.kbWarn = "" })
	cap.Children(func() {
		if rec {
			ui.Text(c, "按下新快捷键…").FontSize(theme.FsCap).TextColor(t.Warn)
		} else {
			keycap(c, combo)
		}
	})
}

func shortcutGroupsLive() []setGroup {
	byGroup := map[string][]shortcutSpec{}
	order := []string{}
	for _, sp := range shortcutTable {
		if _, ok := byGroup[sp.group]; !ok {
			order = append(order, sp.group)
		}
		byGroup[sp.group] = append(byGroup[sp.group], sp)
	}
	var groups []setGroup
	for _, gname := range order {
		items := byGroup[gname]
		rows := make([]setRow, len(items))
		for i, sp := range items {
			sp := sp
			rows[i] = rCustom(sp.title, "", func(a *App, c *ui.Context) { a.shortcutKeycap(c, sp) })
		}
		groups = append(groups, setGroup{title: gname, rows: rows})
	}
	return groups
}

// shortcutWarnBar 冲突提示条。
func (a *App) shortcutWarnBar(c *ui.Context) {
	if a.set.kbWarn == "" {
		return
	}
	t := a.tk
	ui.Row(c).Gap(8).Padding(6, 4).Radius(t.RSm()).Background(t.WarnSoft).Border(1, t.Warn).Children(func() {
		kit.Ic(c, "alert", kit.S12).TextColor(t.Warn)
		ui.Text(c, a.set.kbWarn).FontSize(theme.FsCap).TextColor(t.Warn)
	})
}
