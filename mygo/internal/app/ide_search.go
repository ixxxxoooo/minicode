package app

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/search"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/workspace"
)

// 项目内搜索(对标 VS Code「搜索」,⌘⇧F):内核在 internal/search,走 workspace.FS 所以演示 / 真实通用。
// 输入后 300 ms 防抖自动搜,新搜索取消旧搜索并只采纳最新一次;结果按文件分组、虚拟滚动;
// 替换只改列出的匹配(被忽略的不动);点击匹配在预览标签里打开并选中命中处。

type searchState struct {
	query, repl, inc, exc string
	scope                 string // 多根:只在这个文件夹里搜(空 = 全部文件夹)
	cs, word, re, pc      bool   // 区分大小写 / 全词 / 正则 / 保留大小写
	showRepl, showGlobs   bool
	gen                   int
	running               bool
	res                   search.Result
	err                   string
	done                  bool // 至少搜过一次
	sig                   string
	folded                map[string]bool
	ignored               map[string]bool // 被忽略的匹配:path:line:col
	list                  ui.ListState
	rows                  []srow
	cancel                context.CancelFunc
	focusNext             bool // 打开面板后把焦点给搜索框
}

// srow 是拍平后的一行:文件行或匹配行。
type srow struct {
	file  *search.FileResult
	match *search.Match
	n     int // 文件行:显示的匹配数
}

type searchMatch = search.Match

func mkey(m search.Match) string { return m.Path + ":" + itoa(m.Line) + ":" + itoa(m.Col) }

func (s *searchState) q() search.Query {
	return search.Query{Text: s.query, Case: s.cs, Word: s.word, Regex: s.re, Include: s.inc, Exclude: s.exc, PreserveCase: s.pc}
}

// flatten 把结果拍平成行(折叠的文件只留文件行),忽略的匹配不计入文件计数。
func (s *searchState) flatten() []srow {
	var rows []srow
	for i := range s.res.Files {
		f := &s.res.Files[i]
		n := 0
		for _, m := range f.Matches {
			if !s.ignored[mkey(m)] {
				n++
			}
		}
		rows = append(rows, srow{file: f, n: n})
		if s.folded[f.Path] {
			continue
		}
		for j := range f.Matches {
			rows = append(rows, srow{match: &f.Matches[j]})
		}
	}
	return rows
}

// live 列出没被忽略的全部匹配。
func (s *searchState) live() []search.Match {
	var out []search.Match
	for _, f := range s.res.Files {
		for _, m := range f.Matches {
			if !s.ignored[mkey(m)] {
				out = append(out, m)
			}
		}
	}
	return out
}

// OpenSearch 打开左栏「搜索」(⌘⇧F),有选区就预填。
func (a *App) OpenSearch() {
	if a.mode != ModeIDE {
		a.SetMode(ModeIDE)
	}
	a.nav = "search"
	a.sideOpen = true
	a.sr.focusNext = true
}

// SetSearch 直接设置查询(测试 / 截图用)并立即搜索。
func (a *App) SetSearch(query string, replace string, showRepl bool) {
	a.sr.query, a.sr.repl, a.sr.showRepl = query, replace, showRepl
	a.runSearch(true)
}

// searchFS 搜索用的文件系统:多根且限定了文件夹范围时只含那一个根(路径前缀不变)。
func (a *App) searchFS() workspace.FS {
	if m, ok := a.ide.fs.(*workspace.Multi); ok && a.sr.scope != "" {
		for i, r := range m.Roots() {
			if r.Name == a.sr.scope {
				return m.Only(i)
			}
		}
		a.sr.scope = "" // 文件夹已不在工作区里
	}
	return a.ide.fs
}

// scopeIdx 范围对应的 a.repos 序号(-1 = 全部文件夹)。
func (a *App) scopeIdx() int {
	for i, e := range a.repos {
		if e.name == a.sr.scope {
			return i
		}
	}
	return -1
}

// setSearchScope 限定(或清除)搜索的文件夹范围并重搜。
func (a *App) setSearchScope(name string) {
	a.sr.scope = name
	log.Info("搜索范围", "folder", name)
}

// searchTick 每帧调用:查询或选项变了就安排一次搜索。
func (a *App) searchTick() {
	s := &a.sr
	sig := strings.Join([]string{s.query, s.scope, s.inc, s.exc, b2s(s.cs), b2s(s.word), b2s(s.re)}, "\x00")
	if sig == s.sig {
		return
	}
	s.sig = sig
	a.runSearch(false)
}

func b2s(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// runSearch 取消旧搜索并开新的;immediate 为真时不防抖(测试、手动刷新),且无窗口时同步执行。
func (a *App) runSearch(immediate bool) {
	s := &a.sr
	if s.cancel != nil {
		s.cancel()
	}
	s.gen++
	gen := s.gen
	if strings.TrimSpace(s.query) == "" {
		s.res, s.err, s.running, s.done, s.rows = search.Result{}, "", false, false, nil
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true
	q := s.q()
	fs := a.searchFS()
	work := func() {
		if !immediate {
			select {
			case <-time.After(300 * time.Millisecond):
			case <-ctx.Done():
				return
			}
		}
		end := diag.SpanSlow("search", "项目搜索", 400*time.Millisecond, "q", q.Text)
		res, err := search.Run(ctx, fs, q)
		end()
		apply := func() {
			if gen != a.sr.gen { // 只采纳最新一次
				return
			}
			a.sr.running, a.sr.done = false, true
			if err != nil && ctx.Err() == nil {
				a.sr.err, a.sr.res = err.Error(), search.Result{}
			} else {
				a.sr.err, a.sr.res = "", res
			}
			a.sr.ignored = map[string]bool{}
			a.sr.rows = a.sr.flatten()
			log.Debug("搜索完成", "q", q.Text, "files", len(res.Files), "total", res.Total)
		}
		if a.win != nil {
			a.win.Update(apply)
		} else {
			apply()
		}
	}
	if a.win == nil {
		work()
		return
	}
	go func() { defer diag.Recover("app.search"); work() }()
}

// replaceOne 替换搜索列表里当前选中的一处匹配(⏎)。
func (a *App) replaceOne() {
	s := &a.sr
	if s.list.Selected == nil {
		return
	}
	sel := *s.list.Selected
	if sel < 0 || sel >= len(s.rows) {
		return
	}
	row := s.rows[sel]
	if row.match == nil {
		return
	}
	m := *row.match
	picks := []search.Match{m}
	n, err := search.Replace(a.ide.fs, s.q(), s.repl, picks)
	if err != nil {
		a.toast = "替换失败:" + err.Error()
		return
	}
	if n > 0 {
		a.toast = "已替换 1 处"
	}
	a.ide.reloadOpen()
	a.runSearch(true)
}

// replaceAll 替换列出的匹配并重搜。
func (a *App) replaceAll() {
	s := &a.sr
	picks := s.live()
	if len(picks) == 0 {
		return
	}
	n, err := search.Replace(a.ide.fs, s.q(), s.repl, picks)
	if err != nil {
		log.Warn("替换失败", "err", err)
		a.toast = "替换失败:" + err.Error()
	} else {
		a.toast = "已替换 " + itoa(n) + " 处"
		log.Info("全部替换", "n", n)
	}
	a.ide.reloadOpen() // 已打开的标签读回磁盘新内容
	a.runSearch(true)
}

// ───────── 视图 ─────────

func (a *App) searchView(c *ui.Context) {
	t := a.tk
	s := &a.sr
	if s.folded == nil {
		s.folded, s.ignored = map[string]bool{}, map[string]bool{}
	}
	a.searchTick()
	ui.Column(c).Grow(1).MinHeight(0).Children(func() {
		// 输入区
		ui.Column(c).Gap(4).Padding(0, 2, 6, 2).Shrink(0).Children(func() {
			ui.Row(c).Gap(2).AlignItems(ui.Start).Children(func() {
				chev := ui.ButtonBase(c).Key("sr-chev").Size(14, 26).Justify(ui.Center).AlignItems(ui.Center).TextColor(t.Tx3).Cursor(ui.CursorPointer)
				chev.OnClick(func() { s.showRepl = !s.showRepl })
				chev.Children(func() { kit.Ic(c, map[bool]string{true: "chevD", false: "chevR"}[s.showRepl], kit.S12) })
				_ = a.searchBox(c, "sr-q", "search", &s.query, "搜索", func() {
					a.toggleBtn(c, "Aa", &s.cs, "区分大小写 ⌥⌘C")
					a.toggleBtn(c, "ab", &s.word, "全词匹配 ⌥⌘W")
					a.toggleBtn(c, ".*", &s.re, "正则 ⌥⌘R")
				}, s.focusNext)
				s.focusNext = false
			})
			if s.showRepl {
				ui.Row(c).Gap(2).AlignItems(ui.Start).Children(func() {
					ui.Box(c).Width(14).Shrink(0)
					box := a.searchBox(c, "sr-r", "swap", &s.repl, "替换", func() {
						a.toggleBtn(c, "AB", &s.pc, "保留大小写 ⌥⌘P")
						kit.Ib(c, "check", kit.IbOpt{Small: true, Tip: "全部替换 ⌥⌘↩"}).OnClick(a.replaceAll)
					}, false)
					box.OnSubmit(a.replaceOne)
				})
			}
			ui.Row(c).Gap(6).PaddingStart(16).AlignItems(ui.Center).FontSize(theme.FsCap).TextColor(t.Tx3).Children(func() {
				a.toggleBtn(c, "···", &s.showGlobs, "要包含 / 排除的文件")
				if a.multiRoot() { // 多根:范围 chip(只在多根工作区里出现)
					a.folderPicker(c, "search-scope", a.scopeIdx(), "范围:全部文件夹", false, func(i int) {
						if i < 0 {
							a.setSearchScope("")
						} else {
							a.setSearchScope(a.repos[i].name)
						}
					})
					if s.scope != "" {
						kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "清除范围"}).OnClick(func() { a.setSearchScope("") })
					}
				}
				ui.Spacer(c).Grow(1)
				if s.showRepl {
					ui.Text(c, "仅替换列出的匹配")
				}
			})
			if s.showGlobs {
				for _, g := range []struct {
					key, ph string
					v       *string
				}{{"sr-inc", "要包含的文件,如 src/**,*.rs", &s.inc}, {"sr-exc", "要排除的文件", &s.exc}} {
					ui.Row(c).PaddingStart(16).Children(func() {
						ui.TextInputBase(c.Key(g.key), g.v).Grow(1).Height(24).Padding(0, 8).FontSize(theme.FsCap).Placeholder(g.ph).PlaceholderColor(t.Tx4).
							Background(t.Raised).Radius(t.RSm()).Border(1, t.Bd2)
					})
				}
			}
		})
		// 摘要行
		if s.done || s.running || s.err != "" {
			ui.Row(c).Padding(0, 8, 4, 8).Gap(2).AlignItems(ui.Center).FontSize(theme.FsCap).TextColor(t.Tx3).Shrink(0).Children(func() {
				ui.Text(c, a.searchSummary()).Grow(1).SingleLine().MinWidth(0)
				kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "重新搜索"}).OnClick(func() { a.runSearch(true) })
				kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "清除结果"}).OnClick(func() { s.query, s.sig = "", ""; a.runSearch(true) })
			})
		}
		if s.err != "" {
			ui.Text(c, s.err).FontSize(theme.FsCap).TextColor(t.Err).Padding(0, 8).Shrink(0)
		}
		if len(s.rows) > 0 {
			ui.List(c, &s.list, len(s.rows), func(i int) {
				if i < len(s.rows) { // 菜单里「折叠 / 忽略」会在本帧中途重排 rows
					a.searchRow(c, s.rows[i], i)
				}
			}).Grow(1).MinHeight(0)
		} else if s.running {
			ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(6).Children(func() {
				ui.Text(c, "搜索中…").TextColor(t.Tx3)
			})
		} else if strings.TrimSpace(s.query) == "" {
			ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(6).Children(func() {
				ui.Text(c, "输入关键字开始搜索").TextColor(t.Tx4)
				ui.Text(c, "支持 Aa / ab / .* 与包含·排除 glob").FontSize(theme.FsCap).TextColor(t.Tx4)
			})
		} else if s.done && len(s.rows) == 0 {
			ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(6).Children(func() {
				ui.Text(c, "没有结果").TextColor(t.Tx3)
				if s.err != "" {
					ui.Text(c, s.err).TextColor(t.Err).FontSize(theme.FsCap)
				}
			})
		} else {
			ui.Spacer(c).Grow(1)
		}
	})
}

func (a *App) searchSummary() string {
	s := &a.sr
	switch {
	case s.err != "":
		return "查询有误"
	case s.running:
		return "搜索中…"
	case s.res.Total == 0 && s.scope != "":
		return s.scope + " 内没有结果"
	case s.res.Total == 0:
		return "没有结果"
	}
	extra := ""
	if s.res.Truncated {
		extra = "(已达上限)"
	}
	in := ""
	if s.scope != "" {
		in = "在 " + s.scope + " 内 · "
	}
	return in + itoa(len(s.res.Files)) + " 个文件中 " + itoa(s.res.Total) + " 处结果" + extra
}

// searchBox 是带前缀图标与尾部控件的输入框(设计稿 .inp)。
func (a *App) searchBox(c *ui.Context, key, icon string, v *string, ph string, trail func(), focus bool) ui.Element {
	t := a.tk
	box := ui.Row(c).Key(key+"-box").Grow(1).MinWidth(0).MinHeight(26).PaddingX(6).Gap(4).AlignItems(ui.Center).Background(t.Raised).Radius(t.RSm()).Border(1, t.Bd2)
	if box.FocusWithin() {
		box.BorderColor(t.Ac)
	}
	var in ui.Element
	box.Children(func() {
		kit.Ic(c, icon, kit.S12).TextColor(t.Tx3)
		in = ui.TextInputBase(c.Key(key), v).Grow(1).MinWidth(0).FontSize(theme.FsBody).Placeholder(ph).PlaceholderColor(t.Tx4).Background(ui.Transparent).BorderWidth(0).Padding(0).FocusRing(false)
		if focus {
			in.AutoFocus()
		}
		trail()
	})
	return in
}

// toggleBtn 是小方块开关(设计稿 tgl)。
func (a *App) toggleBtn(c *ui.Context, label string, on *bool, tip string) {
	t := a.tk
	b := ui.ButtonBase(c).Key("tg-"+label).Size(20, 18).Justify(ui.Center).AlignItems(ui.Center).Radius(t.RXs()).FontSize(10.5).FontWeight(600).Cursor(ui.CursorPointer).Shrink(0).Tooltip(tip)
	switch {
	case *on:
		b.Background(t.AcSoft).TextColor(t.AcTx).Border(1, t.AcLine)
	case b.Hovered():
		b.Background(t.Fill1).TextColor(t.Tx1)
	default:
		b.TextColor(t.Tx3)
	}
	b.OnClick(func() { *on = !*on })
	b.Children(func() { ui.Text(c, label).Font("monospace") })
}

func (a *App) searchRow(c *ui.Context, r srow, i int) {
	t := a.tk
	s := &a.sr
	if f := r.file; f != nil {
		row := ui.ButtonBase(c).Key("sf-" + f.Path).Height(22).PaddingX(4).Gap(4).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer)
		if row.Hovered() {
			row.Background(t.Fill1)
		}
		row.OnClick(func() { s.folded[f.Path] = !s.folded[f.Path] })
		row.ContextMenu(func(m *ui.Menu) { a.searchFileMenu(m, f.Path) })
		row.Children(func() {
			kit.Ic(c, map[bool]string{true: "chevR", false: "chevD"}[s.folded[f.Path]], kit.S12).TextColor(t.Tx3)
			a.fileIc(c, path.Base(f.Path))
			ui.Text(c, path.Base(f.Path)).SingleLine().Shrink(0)
			if d := path.Dir(f.Path); d != "." {
				ui.Text(c, d).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().Shrink(1).MinWidth(0)
			}
			ui.Spacer(c).Grow(1)
			kit.Cnt(c, itoa(r.n), false)
		})
		return
	}
	m := *r.match
	ign := s.ignored[mkey(m)]
	row := ui.ButtonBase(c).Key("sm-" + mkey(m)).Height(20).PaddingX(4).PaddingStart(22).Gap(4).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer)
	if row.Hovered() {
		row.Background(t.Fill1)
	}
	row.ContextMenu(func(menu *ui.Menu) { a.searchMatchMenu(menu, m) })
	row.OnClick(func() {
		a.ide.navMark()
		a.ide.openAt(m.Path, m.Line, m.Col, m.End, false)
		a.ide.sel = m.Path
		a.ide.navPush(navPos{m.Path, m.Line, m.Col})
	})
	pre, hit, post := search.Preview(m, 60)
	row.Children(func() {
		col := t.Tx2
		if ign {
			col = t.Tx4
		}
		mk := func(text string, col ui.Color, bold, strike bool) ui.Span {
			w := 400
			if bold {
				w = 700
			}
			return ui.Span{Text: text, Color: col, Weight: w, Strikethrough: strike}
		}
		sp := []ui.Span{mk(pre, col, false, false)}
		if s.showRepl && s.repl != "" && !ign {
			sp = append(sp, mk(hit, t.Err, false, true), mk(s.repl, t.Ok, true, false))
		} else {
			hc := t.AcTx
			if ign {
				hc = t.Tx4
			}
			sp = append(sp, mk(hit, hc, true, ign))
		}
		sp = append(sp, mk(post, col, false, false))
		// 外层容器负责限宽 + 裁剪:RichText 自身的 Shrink 对超长单行不生效,
		// 会把行号挤出面板。
		ui.Box(c).Grow(1).Shrink(1).MinWidth(0).Clip().Children(func() {
			ui.RichText(c, sp...).Font("monospace").FontSize(11.5).NoWrap()
		})
		if !row.Hovered() {
			ui.Text(c, itoa(m.Line+1)).FontSize(theme.FsMicro).TextColor(t.Tx4).FontFeatures("tnum").Shrink(0)
		} else {
			tip, icon := "忽略此处", "x"
			if ign {
				tip, icon = "恢复此处", "refresh"
			}
			kit.Ib(c, icon, kit.IbOpt{Small: true, Tip: tip}).OnClick(func() {
				k := mkey(m)
				s.ignored[k] = !s.ignored[k]
				s.rows = s.flatten()
			})
		}
	})
}
