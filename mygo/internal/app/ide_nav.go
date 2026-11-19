package app

import (
	"path"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// 编辑器的「跳转」能力:转到定义、引用、悬停、补全、重命名、前进后退。
// 数据来自 symbols.go 的文本符号索引(不是语言服务器);界面上的措辞不暗示精确的语义分析。

// ───────── 导航历史 ─────────

// navPos 是一个光标位置(记录的是位置,不是标签)。
type navPos struct {
	id        string
	line, col int
}

// curPos 是当前文件标签里的光标位置。
func (d *ide) curPos() (navPos, bool) {
	t := d.cur()
	if t == nil || t.isVirt() || t.diff || t.st == nil || t.buf == nil {
		return navPos{}, false
	}
	p := t.st.CursorPos()
	return navPos{t.id, p.Line, p.Col}, true
}

// navMark 把「现在的位置」写进历史的当前项(没有历史就建一个)。
func (d *ide) navMark() {
	p, ok := d.curPos()
	if !ok {
		return
	}
	if len(d.nav) == 0 {
		d.nav, d.ni = []navPos{p}, 0
		return
	}
	d.nav[d.ni] = p
}

// navPush 记下一次跳转的目的地:丢掉「前进」那一半,压入新位置。
func (d *ide) navPush(p navPos) {
	if len(d.nav) == 0 {
		d.nav, d.ni = []navPos{p}, 0
		return
	}
	d.nav = append(d.nav[:d.ni+1], p)
	if len(d.nav) > 50 {
		d.nav = d.nav[len(d.nav)-50:]
	}
	d.ni = len(d.nav) - 1
}

func (d *ide) canBack() bool    { return d.ni > 0 }
func (d *ide) canForward() bool { return d.ni+1 < len(d.nav) }

// navGo 沿历史后退(-1)/ 前进(+1)。
func (a *App) navGo(dir int) {
	if a.mode == ModeAgent {
		a.sessGo(dir)
		return
	}
	d := a.ide
	n := d.ni + dir
	if n < 0 || n >= len(d.nav) {
		return
	}
	d.navMark()
	d.ni = n
	p := d.nav[n]
	d.openAt(p.id, p.line, p.col, p.col, true)
}

// jumpTo 是所有「跳转」的出口:先记下现在的位置,打开目标,再把目标压入历史。
func (a *App) jumpTo(p string, line, col, end int) {
	if workspace.IsExternalPath(p) {
		a.jumpToExternalPos(p, line, col, end)
		return
	}
	d := a.ide
	d.navMark()
	d.openAt(p, line, col, end, true)
	d.sel = p
	d.navPush(navPos{p, line, col})
}

// ───────── 符号与文本缓存 ─────────

type symEntry struct {
	ver  uint64
	syms []symbol
}

// fileText 返回文件内容:已打开的取缓冲区(含未保存修改),其余读磁盘并缓存。
func (d *ide) fileText(p string) (string, bool) {
	if t := d.tab(p); t != nil && t.buf != nil {
		return t.buf.String(), true
	}
	if s, ok := d.textCache[p]; ok {
		return s, true
	}
	s, err := d.fs.Read(p)
	if err != nil || strings.IndexByte(s, 0) >= 0 || len(s) > 400<<10 {
		return "", false
	}
	if d.textCache == nil {
		d.textCache = map[string]string{}
	}
	d.textCache[p] = s
	return s, true
}

// fileSyms 返回文件的符号(随缓冲区版本重算)。
func (d *ide) fileSyms(p string) []symbol {
	if symbolLang(p) == "" {
		return nil
	}
	var ver uint64
	if t := d.tab(p); t != nil && t.buf != nil {
		ver = t.buf.Version() + 1
	}
	if e, ok := d.symCache[p]; ok && e.ver == ver {
		return e.syms
	}
	src, ok := d.fileText(p)
	if !ok {
		return nil
	}
	if d.symCache == nil {
		d.symCache = map[string]symEntry{}
	}
	e := symEntry{ver, parseSymbols(p, src)}
	d.symCache[p] = e
	return e.syms
}

// invalidate 磁盘内容变了(保存 / 替换 / 刷新)后丢掉读盘缓存。
func (d *ide) invalidate() {
	d.textCache, d.symCache = nil, nil
}

// codeFiles 是索引范围内的源码文件(同扩展名支持的语言,封顶 600 个)。
func (d *ide) codeFiles() []string {
	var out []string
	for _, p := range d.allFiles() {
		if symbolLang(p) != "" {
			out = append(out, p)
			if len(out) >= 600 {
				break
			}
		}
	}
	return out
}

// defHit 是一处定义。
type defHit struct {
	path string
	sym  symbol
}

// findDefs 找名字为 name 的声明,当前文件排最前。
func (d *ide) findDefs(name, here string) []defHit {
	var out []defHit
	for _, p := range d.codeFiles() {
		for _, s := range d.fileSyms(p) {
			if s.Name == name && s.Kind != "h" && s.Kind != "mod" || s.Kind == "impl" && s.Name == "impl "+name {
				out = append(out, defHit{p, s})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := rank(out[i], here), rank(out[j], here)
		return ri < rj
	})
	return out
}

func rank(h defHit, here string) int {
	r := 0
	if h.path != here {
		r += 10
	}
	if h.sym.Kind == "impl" {
		r += 5
	}
	return r
}

// refItem 是引用列表里的一行。
type refItem struct {
	path      string
	line, col int
	text      string
}

// findRefs 搜项目里整词出现 word 的位置(封顶 500)。
func (d *ide) findRefs(word string) []refItem {
	var out []refItem
	for _, p := range d.codeFiles() {
		src, ok := d.fileText(p)
		if !ok || !strings.Contains(src, word) {
			continue
		}
		for i, ln := range strings.Split(src, "\n") {
			for _, c := range wordHits(codePart(ln), word) {
				out = append(out, refItem{p, i, c, strings.TrimSpace(ln)})
				if len(out) >= 500 {
					return out
				}
			}
		}
	}
	return out
}

// codePart 去掉一行里的注释部分:整行注释返回空,行尾的 `//` 注释截掉;`package x` 声明行也不算引用。
// 只是启发式(不认字符串里的 `//`),用于本地索引的引用计数,语言服务器在线时以它为准。
func codePart(ln string) string {
	t := strings.TrimSpace(ln)
	switch {
	case t == "", strings.HasPrefix(t, "//"), strings.HasPrefix(t, "/*"), strings.HasPrefix(t, "* "), t == "*", strings.HasPrefix(t, "# "), strings.HasPrefix(t, "package "):
		return ""
	}
	if i := strings.Index(ln, " //"); i >= 0 && !strings.Contains(ln[:i], `"`) && !strings.Contains(ln[:i], "`") {
		return ln[:i]
	}
	return ln
}

// showRefs 在底部面板的「引用」分段列出 word 的全部引用。
func (a *App) showRefs(word string) {
	d := a.ide
	if word == "" || isKeyword(word) {
		a.toast = "把光标放在一个符号上再查找引用"
		return
	}
	refsLog.Info("查找引用", "word", word)
	local := func() {
		st := a.refsBeginSearch(word)
		var items []refItem
		if a.proj == "" {
			items = a.refsDemoItems(word)
		} else {
			items = d.findRefs(word)
		}
		a.refsFinishLocal(st, items)
	}
	if tb := d.cur(); tb != nil && tb.st != nil && tb.buf != nil {
		// 先进入「正在查找」状态再发请求:后台任务无窗口(测试)时是同步执行的,
		// 要是请求返回后才开始新一轮查找,刚填好的结果会被清空。
		st := a.refsBeginSearch(word)
		st.loading = true
		if a.lspReferences(tb, tb.st.Cursor(), word, local) {
			return
		}
	}
	local()
}

// ───────── 编辑器钩子 ─────────

// attachEditor 给文件标签接上悬停 / 转到定义 / 补全 / 右键菜单(每个标签一次)。
func (a *App) attachEditor(tb *ideTab) {
	if tb.hooked || tb.st == nil {
		return
	}
	tb.hooked = true
	st := tb.st
	st.OnDefinition = func(off int) { a.goDefAt(tb, off) }
	st.OnHover = func(req editor.HoverRequest) { a.hoverAt(tb, req) }
	st.OnCompletion = func(req editor.CompletionRequest) { a.completeAt(tb, req) }
	st.ContextMenu = func(m *ui.Menu) { a.editorMenu(tb, m) }
}

// wordUnder 取 off 处的单词及其起止偏移。
func wordUnder(tb *ideTab, off int) (w string, start, end int) {
	p := tb.buf.Pos(off)
	ln := tb.buf.Line(p.Line)
	w, a, b := wordAt(ln, p.Col)
	if w == "" {
		return "", 0, 0
	}
	base := off - p.Col
	return w, base + a, base + b
}

// goDefAt 是 ⌘点击 / F12:跳到名字的定义;没有定义就提示。
func (a *App) goDefAt(tb *ideTab, off int) {
	if a.lspDefinition(tb, off, func() { a.goDefLocal(tb, off) }) {
		return
	}
	a.goDefLocal(tb, off)
}

// goDefLocal 用本地符号索引转到定义(没有语言服务器、服务器没答案时的退路)。
func (a *App) goDefLocal(tb *ideTab, off int) {
	w, _, _ := wordUnder(tb, off)
	if w == "" || isKeyword(w) {
		return
	}
	defs := a.ide.findDefs(w, tb.path)
	if len(defs) == 0 {
		a.toast = "找不到「" + w + "」的定义"
		return
	}
	h := defs[0]
	cur := tb.buf.Pos(off)
	if h.path == tb.path && h.sym.Line == cur.Line { // 已经在定义上:改成列出引用
		a.showRefs(w)
		return
	}
	a.jumpTo(h.path, h.sym.Line, h.sym.Col, h.sym.Col+len(w))
}

// goDef 对当前光标处的单词转到定义。
func (a *App) goDef() {
	if tb := a.ide.cur(); tb != nil && tb.st != nil && tb.buf != nil {
		a.goDefAt(tb, tb.st.Cursor())
	}
}

// cursorWord 是光标处的单词。
func (a *App) cursorWord() string {
	if tb := a.ide.cur(); tb != nil && tb.st != nil && tb.buf != nil {
		w, _, _ := wordUnder(tb, tb.st.Cursor())
		return w
	}
	return ""
}

// docAbove 取声明上方连续的文档注释(/// 或 // 或 #),去掉注释符。
func docAbove(src string, line int) string {
	lines := strings.Split(src, "\n")
	var doc []string
	for i := line - 1; i >= 0 && len(doc) < 6; i-- {
		s := strings.TrimSpace(lines[i])
		switch {
		case strings.HasPrefix(s, "///"):
			s = strings.TrimSpace(strings.TrimPrefix(s, "///"))
		case strings.HasPrefix(s, "//"):
			s = strings.TrimSpace(strings.TrimPrefix(s, "//"))
		case strings.HasPrefix(s, "#") && !strings.HasPrefix(s, "#["):
			s = strings.TrimSpace(strings.TrimPrefix(s, "#"))
		default:
			i = -1
			continue
		}
		doc = append([]string{s}, doc...)
	}
	return strings.Join(doc, " ")
}

// hoverAt 悬停:有定义就给签名 + 文档 + 动作行。
func (a *App) hoverAt(tb *ideTab, req editor.HoverRequest) {
	if tb.link != nil { // ⌘ 按住时指的是「跳转」,悬停卡会盖住下划线
		return
	}
	w, s, e := wordUnder(tb, req.Offset)
	if w == "" || isKeyword(w) {
		return
	}
	if a.lspHover(tb, req, s, e, func() { a.hoverLocal(tb, req, w, s, e) }) {
		return
	}
	a.hoverLocal(tb, req, w, s, e)
}

// hoverLocal 用本地符号索引悬停。
func (a *App) hoverLocal(tb *ideTab, req editor.HoverRequest, w string, s, e int) {
	defs := a.ide.findDefs(w, tb.path)
	if len(defs) == 0 {
		return
	}
	h := defs[0]
	src, _ := a.ide.fileText(h.path)
	doc := docAbove(src, h.sym.Line)
	refs := len(a.ide.findRefs(w))
	t := a.tk
	tb.st.ShowHover(s, e, "", func(c *ui.Context) {
		ui.Column(c).Gap(6).Width(300).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				kit.Tag(c, h.sym.Kind, kit.KAc)
				ui.Text(c, h.sym.Name).FontWeight(500)
				ui.Text(c, path.Base(h.path)+":"+itoa(h.sym.Line+1)).FontSize(theme.FsMicro).TextColor(t.Tx4)
			})
			ui.Text(c, h.sym.Detail).Font("monospace").FontSize(theme.Px(11.5)).TextColor(t.Tx2)
			if doc != "" {
				ui.Text(c, doc).FontSize(theme.FsCap).TextColor(t.Tx2).LineHeight(1.5)
			}
			ui.Row(c).Gap(8).FontSize(theme.FsCap).TextColor(t.AcTx).Children(func() {
				ui.Text(c, "转到定义 F12")
				ui.Text(c, itoa(refs)+" 个引用").TextColor(t.Tx3)
			})
		})
	})
}

// completeAt 补全:项目里的符号 + 当前文件里出现过的单词。
func (a *App) completeAt(tb *ideTab, req editor.CompletionRequest) {
	if a.lspCompletion(tb, req, func() { a.completeLocal(tb, req) }) {
		return
	}
	a.completeLocal(tb, req)
}

// completeLocal 本地补全:项目符号 + 当前文件单词。
func (a *App) completeLocal(tb *ideTab, req editor.CompletionRequest) {
	seen := map[string]bool{}
	var items []editor.Completion
	add := func(label, detail string, k editor.CompletionKind, doc string) {
		if seen[label] || label == req.Prefix {
			return
		}
		seen[label] = true
		items = append(items, editor.Completion{Label: label, Detail: detail, Kind: k, Documentation: doc})
	}
	for _, p := range a.ide.codeFiles() {
		for _, s := range a.ide.fileSyms(p) {
			if s.Kind == "h" || s.Kind == "impl" || s.Kind == "mod" {
				continue
			}
			add(s.Name, path.Base(p), symKind(s.Kind), s.Detail)
		}
	}
	src := tb.buf.String()
	for _, ln := range strings.Split(src, "\n") {
		for from := 0; from < len(ln); {
			if !isWordByte(ln[from]) {
				from++
				continue
			}
			to := from
			for to < len(ln) && isWordByte(ln[to]) {
				to++
			}
			if w := ln[from:to]; len(w) > 3 && !isKeyword(w) {
				add(w, "当前文件", editor.CompletionText, "")
			}
			from = to
		}
	}
	tb.st.ShowCompletions(req.Offset-len(req.Prefix), items)
}

func symKind(k string) editor.CompletionKind {
	switch k {
	case "fn":
		return editor.CompletionFunction
	case "struct", "enum", "trait", "class", "type", "interface":
		return editor.CompletionType
	case "const":
		return editor.CompletionConstant
	case "mod":
		return editor.CompletionModule
	}
	return editor.CompletionText
}

// selectionRef 是选区(没有选区取光标所在行)的「文件:行」引用文字。
func (a *App) selectionRef() (ref, text string) {
	tb := a.ide.cur()
	if tb == nil || tb.st == nil || tb.buf == nil {
		return "", ""
	}
	s := tb.st.Selection()
	lo, hi := s.Anchor, s.Head
	if lo > hi {
		lo, hi = hi, lo
	}
	pa, pb := tb.buf.Pos(lo), tb.buf.Pos(hi)
	text = tb.buf.Slice(lo, hi)
	if lo == hi {
		text = tb.buf.Line(pa.Line)
	}
	if pb.Line == pa.Line {
		return "@" + tb.path + ":" + itoa(pa.Line+1), text
	}
	return "@" + tb.path + ":" + itoa(pa.Line+1) + "-" + itoa(pb.Line+1), text
}

// askClaude 把选区作为引用放进输入框:问 / 解释 / 编辑 三种口吻。
func (a *App) askClaude(kind string) {
	ref, text := a.selectionRef()
	if ref == "" {
		return
	}
	lead := map[string]string{"问": "", "解释": "请解释一下这段代码:", "编辑": "请修改这段代码:"}[kind]
	a.draft = strings.TrimSpace(a.draft + "\n" + lead + ref)
	_ = text
	a.revealChat() // IDE 里对话栏在右侧:确保它可见
	a.toast = "已把选区引用到对话"
}

// ───────── Agent 形态的前进 / 后退:在打开过的会话之间走 ─────────

// recordSess 记下一次「打开会话」(后退 / 前进触发的不记)。
func (a *App) recordSess(id string) {
	if a.sessWalking || id == a.curSess && len(a.sessNav) > 0 {
		return
	}
	if len(a.sessNav) == 0 {
		a.sessNav = []string{a.curSess}
		a.sessI = 0
	}
	a.sessNav = append(a.sessNav[:a.sessI+1], id)
	if len(a.sessNav) > 50 {
		a.sessNav = a.sessNav[len(a.sessNav)-50:]
	}
	a.sessI = len(a.sessNav) - 1
}

func (a *App) sessGo(dir int) {
	n := a.sessI + dir
	if n < 0 || n >= len(a.sessNav) {
		return
	}
	a.sessI = n
	a.sessWalking = true
	a.OpenSession(a.sessNav[n])
	a.sessWalking = false
}

// canBack / canForward 当前形态下能不能后退 / 前进(IDE:跳转历史;Agent:会话历史)。
func (a *App) canBack() bool {
	if a.mode == ModeAgent {
		return a.sessI > 0
	}
	return a.ide.canBack()
}

func (a *App) canForward() bool {
	if a.mode == ModeAgent {
		return a.sessI+1 < len(a.sessNav)
	}
	return a.ide.canForward()
}
