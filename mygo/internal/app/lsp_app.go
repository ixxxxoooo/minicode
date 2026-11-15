package app

import (
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/lsp"
)

// 语言服务(LSP)接入:打开文件时按扩展名选服务器(rust-analyzer / gopls / Pyright / lua-language-server …),
// 后台启动并同步文本;诊断画成波浪线并汇入「问题」面板与状态栏;悬停 / 转到定义 / 引用 / 补全
// 优先问服务器,服务器没装、还在启动或没答案时退回本地符号索引。
// 服务器没装时在「问题」面板给出安装命令,一键在终端里运行(用户看得见进度),装完点「重新检测」。

type lspState int

const (
	lspStarting lspState = iota
	lspReady
	lspMissing // 没装
	lspFailed  // 装了但启动 / 握手失败
)

type lspDoc struct {
	ver       int
	bufVer    uint64
	changedAt time.Time
}

// lspSess 是一个(项目根 + 服务器)会话。
type lspSess struct {
	spec     *lsp.Spec
	root     string
	state    lspState
	err      string
	c        *lsp.Client
	cmd      *exec.Cmd
	docs     map[string]*lspDoc // 项目相对路径 → 同步状态
	work     string             // 服务器正在做的事(索引进度)
	restarts int                // 自动重启次数
	uiAt     time.Time          // 安装卡入场动画起点
	wsPrevID string             // 工作区拉取增量 id

	qmu sync.Mutex
	q   chan func() // 串行发送通道:保证 didOpen / didChange / 请求按序
}

// do 把一件事排进该会话的串行队列。
func (s *lspSess) do(f func()) {
	s.qmu.Lock()
	if s.q == nil {
		s.q = make(chan func(), 512)
		go func(q chan func()) {
			defer diag.Recover("lsp.queue")
			for f := range q {
				f()
			}
		}(s.q)
	}
	q := s.q
	s.qmu.Unlock()
	select {
	case q <- f:
	default: // 队列满了(服务器卡死):丢弃,免得拖住界面
	}
}

// barrier 等队列里此前排的事都做完。
func (s *lspSess) barrier() {
	done := make(chan struct{})
	s.do(func() { close(done) })
	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}
}

type lspHub struct {
	sess         map[string]*lspSess
	diags        map[string][]lsp.Diagnostic // 项目相对路径 → 诊断(已打开 + publish)
	wsDiags      map[string][]lsp.Diagnostic // 工作区扫描单独存储
	wsScan       map[string]*lspWsScan       // 各会话扫描状态
	wsPrevID     map[string]string           // 增量拉取 previousResultId
	gen          int                         // 诊断变化计数
	workspaceGen int                         // 工作区扫盘代次(取消旧任务)
	noted        map[string]bool             // 已提示过「没装」的服务器
	outline      map[string]outlineEntry     // 文件 → LSP 大纲缓存
}

// onUI 回到界面线程执行(没有窗口的测试里直接执行)。
func (a *App) onUI(f func()) {
	if a.win == nil { // 测试:攒着,由 drainUI 在测试线程里执行(后台线程不能直接改界面状态)
		a.pmu.Lock()
		a.pend = append(a.pend, f)
		a.pmu.Unlock()
		return
	}
	a.win.Update(f)
}

// drainUI 执行攒下的界面回调(只在没有窗口的测试里有东西)。
func (a *App) drainUI() {
	a.pmu.Lock()
	fs := a.pend
	a.pend = nil
	a.pmu.Unlock()
	for _, f := range fs {
		f()
	}
}

// lspOn 当前能不能用语言服务:真实项目 + 偏好开着。
func (a *App) lspOn() bool { return a.proj != "" && a.prefs.Bool("editor.lsp", true) }

// lspReset 项目切换 / 退出时结束所有服务器。
func (a *App) lspReset() {
	for _, s := range a.lspH.sess {
		s := s
		if s.c != nil {
			go func() {
				defer diag.Recover("lsp.shutdown")
				s.c.Shutdown()
				if s.cmd != nil && s.cmd.Process != nil {
					_ = s.cmd.Process.Kill()
				}
			}()
		}
	}
	a.lspH = lspHub{}
}

// lspFor 取文件所属的服务器会话;首次用到时启动(后台)。没有对应服务器返回 nil。
func (a *App) lspFor(rel string) *lspSess {
	if !a.lspOn() {
		return nil
	}
	spec := lsp.ForFile(rel)
	if spec == nil {
		return nil
	}
	abs := filepath.Join(a.proj, rel)
	root := spec.Root(abs, a.proj)
	key := root + "|" + spec.ID
	if s := a.lspH.sess[key]; s != nil {
		return s
	}
	if a.lspH.sess == nil {
		a.lspH.sess = map[string]*lspSess{}
	}
	s := &lspSess{spec: spec, root: root, docs: map[string]*lspDoc{}}
	a.lspH.sess[key] = s
	a.lspStart(key, s)
	return s
}

func (a *App) lspStart(key string, s *lspSess) {
	s.state, s.err = lspStarting, ""
	hub := &a.lspH
	a.bg("lsp.start", func() func() {
		defer diag.Recover("lsp.start")
		bin, ok := s.spec.Find()
		if !ok {
			return func() {
				if hub.sess[key] != s {
					return
				}
				s.state = lspMissing
				a.outAppend(outLSP, "未找到 "+s.spec.Name+"("+s.spec.Lang+");安装:"+s.spec.Install)
				if hub.noted == nil {
					hub.noted = map[string]bool{}
				}
				if !hub.noted[s.spec.ID] {
					hub.noted[s.spec.ID] = true
					a.toast = "未检测到 " + s.spec.Name + ":在底部「问题」面板可一键安装"
				}
			}
		}
		end := diag.SpanSlow("lsp", "启动语言服务器", 5*time.Second, "server", s.spec.Name)
		c, cmd, err := s.spec.Start(bin, s.root, lsp.Handlers{
			Diagnostics: func(p lsp.PublishDiagnostics) { a.onUI(func() { a.lspDiags(s, p) }) },
			Log: func(level int, msg string) {
				if level <= 2 { // 只收错误与警告,Info / Log 太吵
					a.onUI(func() { a.outAppend(outLSP, s.spec.Name+": "+msg) })
				}
			},
			Progress: func(title, msg string, done bool) {
				a.onUI(func() {
					if done {
						s.work = ""
					} else {
						s.work = strings.TrimSpace(title + " " + msg)
					}
				})
			},
		}, func(line string) { a.onUI(func() { a.outAppend(outLSP, s.spec.Name+": "+line) }) })
		end()
		return func() {
			if hub.sess[key] != s { // 期间项目换了:这个服务器作废
				if c != nil {
					go func() { c.Shutdown(); _ = cmd.Process.Kill() }()
				}
				return
			}
			if err != nil {
				s.state, s.err = lspFailed, err.Error()
				a.outAppend(outLSP, s.spec.Name+" 启动失败:"+err.Error())
				return
			}
			s.c, s.cmd, s.state = c, cmd, lspReady
			a.outAppend(outLSP, s.spec.Name+" 已启动("+s.root+")")
			go func() { // 服务器退出:自动重启(带退避)
				<-c.Done()
				a.onUI(func() {
					if hub.sess[key] == s {
						a.lspOnCrash(key, s)
					}
				})
			}()
		}
	})
}

// lspRetry 丢掉会话(含「没装」的)让它重新检测并启动。
func (a *App) lspRetry(s *lspSess) {
	for k, v := range a.lspH.sess {
		if v == s {
			delete(a.lspH.sess, k)
		}
	}
	delete(a.lspH.noted, s.spec.ID)
	if s.c != nil {
		go func() { s.c.Shutdown(); _ = s.cmd.Process.Kill() }()
	}
	a.redraw()
}

// lspInstall 在终端面板新开一个标签并运行安装命令(用户能看到进度与错误)。
func (a *App) lspInstall(s *lspSess) {
	if a.proj == "" {
		return
	}
	a.newTerm()
	a.ide.tpc().initCmd = s.spec.Install
	a.ide.bottomOpen, a.ide.bottomTab = true, "term"
	a.ide.termSel = ""
	a.outAppend(outLSP, "在终端安装 "+s.spec.Name+":"+s.spec.Install)
}

// ───────── 文本同步 ─────────

// lspTick 每帧对当前文件标签调用:首次 didOpen,编辑停顿 250 ms 后 didChange,并把诊断铺到编辑器上。
func (a *App) lspTick(tb *ideTab) {
	a.drainUI()
	if tb.buf == nil || tb.diff || tb.virt != "" {
		return
	}
	s := a.lspFor(tb.path)
	if s == nil {
		return
	}
	if s.state == lspStarting {
		a.frameSoon()
		return
	}
	if s.state != lspReady {
		return
	}
	uri := lsp.PathToURI(filepath.Join(a.proj, tb.path))
	v := tb.buf.Version()
	d := s.docs[tb.path]
	switch {
	case d == nil:
		d = &lspDoc{ver: 1, bufVer: v}
		s.docs[tb.path] = d
		text, lang, ver := tb.buf.String(), s.spec.LangID(tb.path), d.ver
		s.do(func() { _ = s.c.DidOpen(uri, lang, ver, text) })
	case v != d.bufVer:
		if d.changedAt.IsZero() {
			d.changedAt = time.Now()
		}
		if time.Since(d.changedAt) >= 250*time.Millisecond {
			d.ver++
			d.bufVer, d.changedAt = v, time.Time{}
			text, ver := tb.buf.String(), d.ver
			s.do(func() { _ = s.c.DidChange(uri, ver, text) })
		} else {
			a.frameSoon()
		}
	}
	a.lspApplyDiags(tb)
	a.lspInlayTick(tb)
}

// lspSaved 文件保存后通知服务器(先把挂起的修改发出去)。
func (a *App) lspSaved(rel string) {
	s := a.lspFor(rel)
	if s == nil || s.state != lspReady {
		return
	}
	if tb := a.ide.tab(rel); tb != nil && tb.buf != nil {
		a.lspTick(tb)
	}
	uri := lsp.PathToURI(filepath.Join(a.proj, rel))
	if s.docs[rel] != nil {
		s.do(func() { _ = s.c.DidSave(uri) })
	}
}

// ───────── 诊断 ─────────

func (a *App) lspDiags(s *lspSess, p lsp.PublishDiagnostics) {
	abs := lsp.URIToPath(p.URI)
	if abs == "" {
		return
	}
	rel, err := filepath.Rel(a.proj, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return
	}
	if a.lspH.diags == nil {
		a.lspH.diags = map[string][]lsp.Diagnostic{}
	}
	if len(p.Diagnostics) == 0 {
		delete(a.lspH.diags, rel)
	} else {
		a.lspH.diags[rel] = p.Diagnostics
	}
	a.lspH.gen++
	a.redraw()
}

// lspApplyDiags 把某个文件的诊断换成编辑器的波浪线(诊断集变了才重算)。
func (a *App) lspApplyDiags(tb *ideTab) {
	if tb.st == nil || tb.diagGen == a.lspH.gen+1 {
		return
	}
	tb.diagGen = a.lspH.gen + 1
	var ds []editor.Diagnostic
	for _, d := range a.lspH.diags[tb.path] {
		sp, ok1 := lspOffset(tb, d.Range.Start)
		ep, ok2 := lspOffset(tb, d.Range.End)
		if !ok1 || !ok2 {
			continue
		}
		if ep <= sp {
			ep = min(sp+1, tb.buf.Offset(editor.Pos{Line: tb.buf.LineCount() - 1, Col: len(tb.buf.Line(tb.buf.LineCount() - 1))}))
		}
		sev := editor.SeverityWarning
		switch d.Severity {
		case 1:
			sev = editor.SeverityError
		case 3:
			sev = editor.SeverityInfo
		case 4:
			sev = editor.SeverityHint
		}
		ds = append(ds, editor.Diagnostic{Start: sp, End: ep, Severity: sev, Message: d.Message, Source: d.Source})
	}
	tb.st.SetDiagnostics(tb.buf.Version(), ds)
}

// lspOffset 把 LSP 位置(UTF-16 列)换成缓冲区字节偏移。
func lspOffset(tb *ideTab, p lsp.Position) (int, bool) {
	if p.Line < 0 || p.Line >= tb.buf.LineCount() {
		return 0, false
	}
	ln := tb.buf.Line(p.Line)
	return tb.buf.Offset(editor.Pos{Line: p.Line, Col: lsp.ColFromUTF16(ln, p.Character)}), true
}

// lspPos 把缓冲区字节偏移换成 LSP 位置。
func lspPos(tb *ideTab, off int) lsp.Position {
	p := tb.buf.Pos(off)
	return lsp.Position{Line: p.Line, Character: lsp.ColToUTF16(tb.buf.Line(p.Line), p.Col)}
}

// lspCounts 全项目的错误 / 警告数(已打开 + 工作区,同文件不重复计数)。
func (a *App) lspCounts() (errs, warns int) {
	seen := map[string]bool{}
	count := func(ds []lsp.Diagnostic) {
		for _, d := range ds {
			switch d.Severity {
			case 1:
				errs++
			case 2:
				warns++
			}
		}
	}
	for path, ds := range a.lspH.diags {
		seen[path] = true
		count(ds)
	}
	for path, ds := range a.lspH.wsDiags {
		if seen[path] {
			continue
		}
		count(ds)
	}
	return
}

// lspReady 返回文件对应、已就绪的会话(没有则 nil)。
func (a *App) lspReadyFor(rel string) *lspSess {
	if !a.lspOn() {
		return nil
	}
	if s := a.lspFor(rel); s != nil && s.state == lspReady && s.docs[rel] != nil {
		return s
	}
	return nil
}

// lspJob 在会话里异步做一次请求:先等已排的同步发完,再调 work;结果经 apply 回界面线程。
func (a *App) lspJob(name string, s *lspSess, work func() func()) {
	a.bg("lsp."+name, func() func() {
		s.barrier()
		return work()
	})
}

// ───────── 请求 ─────────

// locRel 把位置换成项目相对路径;项目外返回 ok=false。
func (a *App) locRel(l lsp.Location) (string, bool) {
	abs := lsp.URIToPath(l.URI)
	if abs == "" {
		return "", false
	}
	rel, err := filepath.Rel(a.proj, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return abs, false
	}
	return rel, true
}

// lspDefinition 转到定义:服务器答了就跳,没答案时调 fallback(本地符号索引)。
func (a *App) lspDefinition(tb *ideTab, off int, fallback func()) bool {
	s := a.lspReadyFor(tb.path)
	if s == nil {
		return false
	}
	uri := lsp.PathToURI(filepath.Join(a.proj, tb.path))
	pos := lspPos(tb, off)
	here := tb.path
	a.lspJob("def", s, func() func() {
		locs, err := s.c.Definition(uri, pos)
		return func() {
			if err != nil || len(locs) == 0 {
				fallback()
				return
			}
			l := locs[0]
			rel, in := a.locRel(l)
			switch {
			case !in:
				a.toast = "定义在项目外:" + rel + ":" + itoa(l.Range.Start.Line+1)
			case rel == here && l.Range.Start.Line == pos.Line && l.Range.Start.Character <= pos.Character && pos.Character <= l.Range.End.Character:
				a.showRefs(a.cursorWord()) // 已经在定义上:改成列出引用
			default:
				a.jumpToLSP(rel, l.Range)
			}
		}
	})
	return true
}

// jumpToLSP 跳到 LSP 范围(列换成字节列需要文本;先读文本)。
func (a *App) jumpToLSP(rel string, r lsp.Range) {
	text, _ := a.ide.fileText(rel)
	lines := strings.Split(text, "\n")
	col := func(line, u16 int) int {
		if line >= 0 && line < len(lines) {
			return lsp.ColFromUTF16(lines[line], u16)
		}
		return 0
	}
	c0 := col(r.Start.Line, r.Start.Character)
	c1 := c0
	if r.End.Line == r.Start.Line {
		c1 = col(r.End.Line, r.End.Character)
	}
	a.jumpTo(rel, r.Start.Line, c0, c1)
}

// lspReferences 列出引用到底部「引用」分段。
func (a *App) lspReferences(tb *ideTab, off int, word string, fallback func()) bool {
	s := a.lspReadyFor(tb.path)
	if s == nil {
		return false
	}
	uri := lsp.PathToURI(filepath.Join(a.proj, tb.path))
	pos := lspPos(tb, off)
	a.lspJob("refs", s, func() func() {
		locs, err := s.c.References(uri, pos)
		var items []refItem
		for _, l := range locs {
			rel, in := a.locRelNoUI(l)
			if !in {
				continue
			}
			text, _ := a.ide.fileTextBG(rel)
			lines := strings.Split(text, "\n")
			ln := ""
			if l.Range.Start.Line < len(lines) {
				ln = lines[l.Range.Start.Line]
			}
			items = append(items, refItem{path: rel, line: l.Range.Start.Line, col: lsp.ColFromUTF16(ln, l.Range.Start.Character), text: strings.TrimSpace(ln)})
		}
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].path != items[j].path {
				return items[i].path < items[j].path
			}
			return items[i].line < items[j].line
		})
		return func() {
			if err != nil || len(items) == 0 {
				fallback()
				return
			}
			a.ide.refs = &refsState{word: word, items: items}
			a.ide.bottomOpen, a.ide.bottomTab = true, "refs"
		}
	})
	return true
}

func (a *App) locRelNoUI(l lsp.Location) (string, bool) { return a.locRel(l) }

// lspHover 悬停:服务器有内容就显示(Markdown),没有则 fallback。
func (a *App) lspHover(tb *ideTab, req editor.HoverRequest, start, end int, fallback func()) bool {
	s := a.lspReadyFor(tb.path)
	if s == nil {
		return false
	}
	uri := lsp.PathToURI(filepath.Join(a.proj, tb.path))
	pos := lspPos(tb, req.Offset)
	a.lspJob("hover", s, func() func() {
		text, err := s.c.Hover(uri, pos)
		return func() {
			text = strings.TrimSpace(text)
			if err != nil || text == "" {
				fallback()
				return
			}
			tb.st.ShowHover(start, end, "", func(c *ui.Context) {
				ui.Column(c).Width(380).MaxHeight(280).Children(func() {
					ui.Scroll(c).Key("lsp-hover").Children(func() {
						markdown.View(c, text, a.mdOptions())
					})
				})
			})
		}
	})
	return true
}

// lspCompletion 补全:服务器的结果优先,空 / 失败时 fallback(本地符号 + 文件单词)。
func (a *App) lspCompletion(tb *ideTab, req editor.CompletionRequest, fallback func()) bool {
	s := a.lspReadyFor(tb.path)
	if s == nil {
		return false
	}
	uri := lsp.PathToURI(filepath.Join(a.proj, tb.path))
	pos := lspPos(tb, req.Offset)
	a.lspJob("complete", s, func() func() {
		list, err := s.c.Completion(uri, pos)
		sort.SliceStable(list, func(i, j int) bool {
			ki, kj := list[i].SortText, list[j].SortText
			if ki == "" {
				ki = list[i].Label
			}
			if kj == "" {
				kj = list[j].Label
			}
			return ki < kj
		})
		if len(list) > 300 {
			list = list[:300]
		}
		return func() {
			if err != nil || len(list) == 0 {
				fallback()
				return
			}
			items := make([]editor.Completion, 0, len(list))
			for _, it := range list {
				ins := it.InsertText
				if it.TextEdit != nil && it.TextEdit.NewText != "" {
					ins = it.TextEdit.NewText
				}
				if it.InsertTextFormat == 2 { // 代码片段:我们不支持,退回标签
					ins = ""
				}
				if ins == it.Label {
					ins = ""
				}
				items = append(items, editor.Completion{Label: strings.TrimSpace(it.Label), Detail: it.Detail, Kind: lspKind(it.Kind), Insert: ins, Documentation: it.Doc(), FilterText: it.FilterText})
			}
			tb.st.ShowCompletions(req.Offset-len(req.Prefix), items)
		}
	})
	return true
}

func lspKind(k int) editor.CompletionKind {
	switch k {
	case 2:
		return editor.CompletionMethod
	case 3, 4:
		return editor.CompletionFunction
	case 5, 10:
		return editor.CompletionField
	case 6:
		return editor.CompletionVariable
	case 7, 8, 13, 22, 25:
		return editor.CompletionType
	case 9:
		return editor.CompletionModule
	case 14:
		return editor.CompletionKeyword
	case 15:
		return editor.CompletionSnippet
	case 21, 20:
		return editor.CompletionConstant
	}
	return editor.CompletionText
}
