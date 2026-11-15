package app

import (
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 代码区上的浮层:粘性滚动、选中工具条、预览定义、重命名、代码操作、行内编辑,
// 以及编辑器自己看不到的叠加(同词高亮、Claude 改过的行)。位置都相对代码区。

const (
	edGutter = 46.0 // 行号栏宽(编辑器的行号 + 折叠 + 改动条)
	edChar   = 7.2  // 12px 等宽字的字宽
	edLineH  = 20.0
)

// ideOv 是代码区当前打开的浮层(同时只有一个)。
type ideOv struct {
	kind   string // peek / rename / acts / inline
	word   string
	input  string
	peekP  string
	peekL  int
	peekSt *editor.State
	frames int
	hits   int // 重命名将更新的处数
	files  int
}

// claudeLines 是 Claude 本轮改过、还没处理的行范围(0 起,含)。
func (a *App) claudeLines(tb *ideTab) [][2]int {
	var out [][2]int
	for _, f := range a.changedFiles() {
		if a.wsRel(f.path) != tb.path {
			continue
		}
		for i, b := range f.blocks {
			if a.rv.hunk[hunkKey(f.path, i)] != "" {
				continue
			}
			if lo, hi := hunkRange(b); lo > 0 {
				out = append(out, [2]int{lo - 1, hi - 1})
			}
		}
	}
	return out
}

// findQuery 是编辑器查找条里当前的查询词与是否区分大小写。
func (t *ideTab) findQuery() (string, bool) {
	if t.fq == nil {
		return "", false
	}
	return *t.fq, t.fcs != nil && *t.fcs
}

// cursorXY 是光标行左下在代码区里的位置(用于定位浮层)。
func cursorXY(tb *ideTab, line, col int) (x, y float32) {
	y = float32(tb.st.LineTop(line)-tb.st.ScrollY()) + edLineH*edScale
	x = edGutter + float32(col)*edChar*edScale
	return
}

// codeView 是代码区:编辑器 + 右侧迷你地图 + 浮层。
func (a *App) codeView(c *ui.Context, tb *ideTab) {
	t := a.tk
	st := tb.st
	st.Theme = a.editorTheme()
	st.Font = ui.Font{Family: "monospace", Size: 12}
	st.LineHeight = 20.0 / 12.0
	st.NoScrollbars = true
	a.attachEditor(tb)
	if st.RenderFindBar == nil { // 记下查找词,迷你地图 / 标尺据此画命中刻度
		st.RenderFindBar = func(c *ui.Context, bar ui.Element, f *editor.FindBar) {
			tb.fq, tb.fcs = f.Query(), f.CaseSensitive()
			editor.DefaultFindBar(c, bar, f)
		}
	}
	if r := tb.reveal; r != nil { // 搜索结果 / 跳转:选中命中处并滚到可见
		tb.reveal = nil
		p0 := editor.Pos{Line: r[0], Col: r[1]}
		st.GoTo(p0)
		st.Select(tb.buf.Offset(p0), tb.buf.Offset(editor.Pos{Line: r[0], Col: r[2]}))
	}
	a.decorate(tb)

	ui.Column(c).Grow(1).MinHeight(0).Children(func() {
		ui.Row(c).Grow(1).MinHeight(0).Children(func() {
			editor.View(c.Key("ed-"+tb.id), st).Grow(1).MinWidth(0).FillHeight()
			if !a.wsNarrowMM() && a.prefs.Bool(pMinimap, true) {
				a.minimap(c, tb)
			}
		})
		a.stickyScroll(c, tb)
		a.askBar(c, tb)
		a.ovPanel(c, tb)
	})
	_ = t
}

// wsNarrowMM 工作区中档(520–599)起隐藏迷你地图。
func (a *App) wsNarrowMM() bool { return a.wsW > 0 && a.wsW < 600 }

// decorate 把编辑器自己画不出的叠加交给它:同词浅蓝底、Claude 改过的行(紫底)、演示诊断。
func (a *App) decorate(tb *ideTab) {
	t := a.tk
	st := tb.st
	var tbs []editor.TextBackground
	if w := a.tabWord(tb); w != "" {
		n := tb.buf.LineCount()
		for i := 0; i < n && len(tbs) < 300; i++ {
			ln := tb.buf.Line(i)
			base := tb.buf.Offset(editor.Pos{Line: i, Col: 0})
			for _, c := range wordHits(ln, w) {
				tbs = append(tbs, editor.TextBackground{Start: base + c, End: base + c + len(w), Color: t.Info.Alpha(.18)})
			}
		}
	}
	st.TextBackgrounds = tbs
	var lbs []editor.LineBackground
	for _, g := range a.claudeLines(tb) {
		lbs = append(lbs, editor.LineBackground{First: g[0], Last: g[1], Color: t.Ac.Alpha(.10)})
	}
	st.LineBackgrounds = lbs
	// 演示:问题面板里的三条诊断同时画成编辑器里的波浪线
	if a.proj == "" && !tb.diagSet {
		tb.diagSet = true
		var ds []editor.Diagnostic
		for _, p := range demoProblems() {
			if p.file != path.Base(tb.path) {
				continue
			}
			ln := tb.buf.Line(p.line - 1)
			col := -1
			if h := wordHits(ln, p.word); len(h) > 0 {
				col = h[0]
			}
			if col < 0 {
				col = len(ln) - len(strings.TrimLeft(ln, " \t"))
			}
			base := tb.buf.Offset(editor.Pos{Line: p.line - 1, Col: col})
			sev := editor.SeverityWarning
			if p.level == "err" {
				sev = editor.SeverityError
			}
			ds = append(ds, editor.Diagnostic{Start: base, End: base + len(p.word), Severity: sev, Message: p.msg, Source: p.src})
		}
		st.SetDiagnostics(tb.buf.Version(), ds)
	}
	a.applyEditorPrefs(tb)
	a.blameInlay(tb)
}

// ───────── 粘性滚动 ─────────

// stickyScroll 滚进长函数后,把所属的 impl / fn 头固定在顶部,点击回到该行。
func (a *App) stickyScroll(c *ui.Context, tb *ideTab) {
	t := a.tk
	first, _ := tb.st.VisibleLines()
	if first <= 0 {
		return
	}
	syms := a.ide.fileSyms(tb.path)
	var rows []symbol
	for _, k := range enclosing(syms, first) {
		if syms[k].Line < first && syms[k].Kind != "h" {
			rows = append(rows, syms[k])
		}
	}
	if len(rows) == 0 {
		return
	}
	if len(rows) > 2 {
		rows = rows[len(rows)-2:]
	}
	ui.Column(c).Key("sticky").Absolute().Left(0).Right(mmW+mmRulW+6).Top(0).Background(t.Raised).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd2).Children(func() {
		for _, s := range rows {
			r := ui.ButtonBase(c).Key("st-" + itoa(s.Line)).Height(22).PaddingStart(edGutter).PaddingEnd(8).Justify(ui.Start).AlignItems(ui.Center).Gap(6).
				Cursor(ui.CursorPointer).TextColor(t.Tx2)
			if r.Hovered() {
				r.Background(t.Fill1)
			}
			if r.Clicked() {
				a.jumpTo(tb.path, s.Line, s.Col, s.Col)
			}
			r.Children(func() {
				ui.Text(c, s.Detail).Font("monospace").FontSize(12).SingleLine()
			})
		}
	})
}

// ───────── 选中工具条 ─────────

// askBar 选中代码时在选区上方浮出的工具条:问 / 编辑 / 解释 / 加到对话。
func (a *App) askBar(c *ui.Context, tb *ideTab) {
	t := a.tk
	s := tb.st.Selection()
	if s.Anchor == s.Head || a.ide.ov.kind != "" {
		return
	}
	if q, _ := tb.findQuery(); q != "" && strings.EqualFold(tb.st.SelectedText(), q) { // 选区是查找命中:不弹工具条
		return
	}
	lo := s.Anchor
	if s.Head < lo {
		lo = s.Head
	}
	p := tb.buf.Pos(lo)
	x, y := cursorXY(tb, p.Line, p.Col)
	y -= edLineH*edScale + 32
	if y < 24 {
		y += edLineH*edScale + 32 + 26
	}
	if x > 260 {
		x = 260
	}
	bar := ui.Row(c).Key("askbar").Absolute().Left(x).Top(y).Height(28).Padding(2).Gap(1).AlignItems(ui.Center).
		Background(t.Raised).Radius(t.RMd()).Border(1, t.Bd1)
	for _, sh := range t.Shadow2() {
		bar.Shadow(sh.X, sh.Y, sh.Blur, sh.Spread, sh.Color)
	}
	bar.Children(func() {
		btn := func(icon, text, key string, run func()) {
			b := ui.ButtonBase(c).Key("ab-" + key).Height(24).PaddingX(7).Gap(4).Radius(t.RSm()).AlignItems(ui.Center).Cursor(ui.CursorPointer).TextColor(t.Tx2).FontSize(theme.FsCap)
			if b.Hovered() {
				b.Background(t.Fill1).TextColor(t.Tx1)
			}
			if b.Clicked() {
				run()
			}
			b.Children(func() {
				kit.Ic(c, icon, kit.S12).TextColor(t.AcTx)
				ui.Text(c, text)
			})
		}
		btn("spark", "问 Claude ⌘L", "ask", func() { a.askClaude("问") })
		btn("edit", "编辑 ⌘I", "edit", func() { a.openInline() })
		btn("bulb", "解释", "exp", func() { a.askClaude("解释") })
	})
}

// ───────── 面板:预览定义 / 重命名 / 代码操作 / 行内编辑 ─────────

func (a *App) closeOv() { a.ide.ov = ideOv{} }

// openPeek ⌥F12:在原地内嵌一个只读小编辑器看定义。
func (a *App) openPeek() {
	tb := a.ide.cur()
	if tb == nil || tb.st == nil || tb.buf == nil {
		return
	}
	w, _, _ := wordUnder(tb, tb.st.Cursor())
	if w == "" || isKeyword(w) {
		return
	}
	defs := a.ide.findDefs(w, tb.path)
	if len(defs) == 0 {
		a.toast = "找不到「" + w + "」的定义"
		return
	}
	h := defs[0]
	src, _ := a.ide.fileText(h.path)
	buf := editor.NewBuffer(src)
	buf.SetLanguage(editor.LanguageFor(path.Base(h.path)))
	a.ide.ov = ideOv{kind: "peek", word: w, peekP: h.path, peekL: h.sym.Line,
		peekSt: &editor.State{Buffer: buf, ReadOnly: true, NoScrollbars: true, NoFolding: true, NoActiveLine: true}}
}

// openRename F2:原地输入新名字。
func (a *App) openRename() {
	tb := a.ide.cur()
	if tb == nil || tb.st == nil || tb.buf == nil {
		return
	}
	w, _, _ := wordUnder(tb, tb.st.Cursor())
	if w == "" || isKeyword(w) {
		a.toast = "把光标放在要重命名的符号上"
		return
	}
	refs := a.ide.findRefs(w)
	seen := map[string]bool{}
	for _, r := range refs {
		seen[r.path] = true
	}
	a.ide.ov = ideOv{kind: "rename", word: w, input: w, hits: len(refs), files: len(seen)}
}

// applyRename 把整词 word 在项目里全部换成 to:已打开的文件改缓冲区(可撤销),其余直接写盘。
func (a *App) applyRename(word, to string) {
	d := a.ide
	if to == "" || to == word || strings.ContainsAny(to, " \t\n") {
		a.closeOv()
		return
	}
	n := 0
	for _, p := range d.codeFiles() {
		src, ok := d.fileText(p)
		if !ok || !strings.Contains(src, word) {
			continue
		}
		var b strings.Builder
		changed := 0
		for i, ln := range strings.Split(src, "\n") {
			if i > 0 {
				b.WriteByte('\n')
			}
			prev := 0
			for _, c := range wordHits(ln, word) {
				b.WriteString(ln[prev:c])
				b.WriteString(to)
				prev = c + len(word)
				changed++
			}
			b.WriteString(ln[prev:])
		}
		if changed == 0 {
			continue
		}
		n += changed
		if tb := d.tab(p); tb != nil && tb.buf != nil {
			tb.buf.Set(b.String())
		} else if err := d.fs.Write(p, b.String()); err != nil {
			a.toast = "写入 " + p + " 失败"
			continue
		}
	}
	d.invalidate()
	a.closeOv()
	a.toast = "已重命名 " + itoa(n) + " 处:" + word + " → " + to
}

// openActs ⌘.:代码操作菜单。
func (a *App) openActs() {
	if tb := a.ide.cur(); tb != nil && tb.st != nil {
		a.ide.ov = ideOv{kind: "acts", word: a.cursorWord()}
	}
}

// openInline ⌘I:在选区上方弹小输入框,一句话描述修改。
func (a *App) openInline() {
	if tb := a.ide.cur(); tb != nil && tb.st != nil {
		a.ide.ov = ideOv{kind: "inline"}
	}
}

// ovPanel 画当前浮层(没有则不画)。
func (a *App) ovPanel(c *ui.Context, tb *ideTab) {
	ov := &a.ide.ov
	if ov.kind == "" {
		return
	}
	t := a.tk
	p := tb.st.Selection()
	pos := tb.buf.Pos(p.Head)
	x, y := cursorXY(tb, pos.Line, pos.Col)
	if x > 240 {
		x = 240
	}
	panel := func(key string, w float32) ui.Element {
		el := ui.Column(c).Key(key).Absolute().Left(x).Top(y+4).Width(w).Background(t.Raised).Radius(t.RLg()).Border(1, t.Bd1).Padding(4).Gap(2)
		for _, s := range t.Shadow2() {
			el.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
		}
		return el
	}
	esc := func(in ui.Element) { in.OnShortcut(0, ui.KeyEscape, a.closeOv) }
	switch ov.kind {
	case "rename":
		panel("ov-rename", 280).Children(func() {
			ui.Text(c, "重命名符号").FontSize(theme.FsMicro).TextColor(t.Tx4).FontWeight(700).Margin(2, 6)
			in := ui.TextInputBase(c.Key("ov-ren-in"), &ov.input).Height(28).PaddingX(8).Radius(t.RSm()).Background(t.Fill1).Border(1, t.Ac).
				Font("monospace").FontSize(12).AutoFocus()
			in.OnSubmit(func() { a.applyRename(ov.word, strings.TrimSpace(ov.input)) })
			esc(in)
			ui.Text(c, "将更新 "+itoa(ov.hits)+" 处引用("+itoa(ov.files)+" 个文件) · ⏎ 应用 · Esc 取消").
				FontSize(theme.FsMicro).TextColor(t.Tx3).Margin(2, 6, 3, 6)
		})
	case "inline":
		panel("ov-inline", 340).Children(func() {
			ui.Text(c, "行内编辑 · 用一句话描述修改").FontSize(theme.FsMicro).TextColor(t.Tx4).FontWeight(700).Margin(2, 6)
			in := ui.TextInputBase(c.Key("ov-inl-in"), &ov.input).Height(28).PaddingX(8).Radius(t.RSm()).Background(t.Fill1).Border(1, t.Ac).AutoFocus().
				Placeholder("例如:把它改成忽略大小写").PlaceholderColor(t.Tx4)
			in.OnSubmit(func() {
				if s := strings.TrimSpace(ov.input); s != "" {
					ref, _ := a.selectionRef()
					a.draft = strings.TrimSpace(a.draft + "\n请修改 " + ref + ":" + s)
					if a.mode == ModeIDE {
						a.rightOn = true
					}
					a.toast = "已把修改要求放进输入框"
				}
				a.closeOv()
			})
			esc(in)
			ui.Text(c, "⏎ 发送给 Claude(进入对话) · Esc 取消").FontSize(theme.FsMicro).TextColor(t.Tx3).Margin(2, 6, 3, 6)
		})
	case "acts":
		panel("ov-acts", 260).Children(func() {
			ui.Text(c, "代码操作").FontSize(theme.FsMicro).TextColor(t.Tx4).FontWeight(700).Margin(2, 6)
			word := ov.word
			item := func(key, icon, text, keys string, run func()) {
				b := a.mi(c, key, icon, text, keys, false)
				if b.Clicked() {
					a.closeOv()
					run()
				}
			}
			item("fix", "spark", "让 Claude 修复", "", func() { a.askClaude("编辑") })
			item("exp", "bulb", "让 Claude 解释", "", func() { a.askClaude("解释") })
			if word != "" && !isKeyword(word) {
				item("def", "arrow", "转到定义", "F12", a.goDef)
				item("ref", "search", "查找所有引用", "⇧F12", func() { a.showRefs(word) })
				item("ren", "edit", "重命名「"+word+"」", "F2", a.openRename)
			}
		})
	case "peek":
		a.peekPanel(c, tb, y)
	}
}

// peekPanel 预览定义:光标行下方内嵌一个只读小编辑器。
func (a *App) peekPanel(c *ui.Context, tb *ideTab, y float32) {
	t := a.tk
	ov := &a.ide.ov
	if ov.frames < 3 { // 前几帧等视口量出来再滚到定义行
		ov.peekSt.ScrollTo(max(ov.peekL-1, 0))
		ov.frames++
		c.Invalidate()
	}
	ov.peekSt.Theme = a.editorTheme()
	ov.peekSt.Font = ui.Font{Family: "monospace", Size: 12}
	ov.peekSt.LineHeight = 20.0 / 12.0
	el := ui.Column(c).Key("ov-peek").Absolute().Left(16).Right(mmW+mmRulW+14).Top(y+4).Height(180).Background(t.BG).Radius(t.RLg()).Border(1, t.Ac).Clip()
	for _, s := range t.Shadow2() {
		el.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	el.Children(func() {
		ui.Row(c).Height(26).Shrink(0).PaddingX(10).Gap(6).AlignItems(ui.Center).Background(t.AcSoft).FontSize(theme.FsCap).Children(func() {
			kit.Ic(c, "code", kit.S12).TextColor(t.AcTx)
			ui.Text(c, "定义 · "+ov.peekP+":"+itoa(ov.peekL+1)).FontWeight(600).TextColor(t.AcTx)
			ui.Spacer(c).Grow(1)
			ui.Text(c, "⌥⏎ 在编辑器中打开").TextColor(t.Tx3)
			kit.Ib(c, "x", kit.IbOpt{Small: true}).OnClick(a.closeOv)
		})
		ui.Row(c).Grow(1).MinHeight(0).Children(func() {
			ed := editor.View(c.Key("peek-ed"), ov.peekSt).Grow(1).MinWidth(0).FillHeight()
			ed.OnShortcut(0, ui.KeyEscape, a.closeOv)
		})
	})
	if c.Shortcut(ui.Alt, ui.KeyEnter) {
		p, l := ov.peekP, ov.peekL
		a.closeOv()
		a.jumpTo(p, l, 0, 0)
	}
}

// editorShortcuts 是编辑器级快捷键(只在有当前文件时生效):F12 / ⇧F12 / ⌥F12 / F2 / ⌘. / ⌘L / ⌘I / ⌃- / ⌃⇧-。
func (a *App) editorShortcuts(c *ui.Context) bool {
	if a.screen != ScreenWork || a.ide.cur() == nil || a.ide.cur().st == nil {
		if a.ide.ov.kind != "" && c.Shortcut(0, ui.KeyEscape) {
			a.closeOv()
			return true
		}
		return false
	}
	switch {
	case c.Shortcut(0, ui.KeyF12):
		a.goDef()
	case c.Shortcut(ui.Shift, ui.KeyF12):
		a.showRefs(a.cursorWord())
	case c.Shortcut(ui.Alt, ui.KeyF12):
		a.openPeek()
	case c.Shortcut(0, ui.KeyF2):
		a.openRename()
	case c.Shortcut(ui.Cmd, ui.KeyPeriod):
		a.openActs()
	case c.Shortcut(ui.Cmd, ui.KeyL):
		a.askClaude("问")
	case c.Shortcut(ui.Cmd, ui.KeyI):
		a.openInline()
	case c.Shortcut(ui.Ctrl, ui.KeyMinus):
		a.navGo(-1)
	case c.Shortcut(ui.Ctrl|ui.Shift, ui.KeyMinus):
		a.navGo(1)
	case a.ide.ov.kind != "" && c.Shortcut(0, ui.KeyEscape):
		a.closeOv()
	default:
		return false
	}
	return true
}

// demoProblem 是问题面板与编辑器波浪线共用的演示诊断。
type demoProb struct {
	file, level, msg, src, word string
	line                        int
}

func demoProblems() []demoProb {
	return []demoProb{
		{"router.rs", "warn", "`fallback` 字段从未被读取", "rust-analyzer", "fallback", 6},
		{"router.rs", "warn", "未使用的变量:`h`", "rust-analyzer", "h", 11},
		{"router_test.rs", "err", "找不到函数 `normalize`,需要 `use crate::router::normalize`", "rust-analyzer", "normalize", 8},
	}
}
