package app

import (
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// 快速选择浮层(设计稿「选择器与面板」):顶部居中,一个搜索框 + 分组列表。
// 目前两种:历史会话选择器、分支选择器。↑↓ 移动、回车选中、Esc / 点遮罩关闭。

type qpState struct {
	kind     string // history / branch
	query    string
	sel      int
	lastQ    string
	listSt   ui.ScrollState
	lastSel  int
	branches []vcs.Branch // 分支选择器打开时读一次
	loadErr  string
	symFile  string       // 符号选择器:要列符号的文件
	cmpC     vcs.Commit   // 「与工作区比较」:被比较的提交
	cmpFiles []vcs.Change // 及它涉及的文件
}

type qpRow struct {
	group, icon, title, right string
	current                   bool
	run                       func()
}

const (
	qpW    = 440
	qpRowH = 28
	qpCapH = 22
	qpMaxH = 320
)

// OpenQuick 打开快速选择(history / branch),query 预填。
func (a *App) OpenQuick(kind, query string) {
	s := &qpState{kind: kind, query: query, lastSel: -1}
	if kind == "branch" && a.repo != nil {
		bs, err := a.repo.Branches()
		if err != nil {
			s.loadErr = err.Error()
			log.Warn("读取分支失败", "err", err)
		}
		s.branches = bs
	}
	if kind == "symbols" {
		if tb := a.ide.cur(); tb != nil && !tb.isVirt() && !tb.diff {
			s.symFile = tb.path
		}
	}
	a.qp = s
	log.Debug("打开快速选择", "kind", kind)
}

func (a *App) closeQuick() { a.qp = nil; a.sc.createAt = "" }

// qpRows 按查询构建当前列表。
func (a *App) qpRows() []qpRow {
	s := a.qp
	q := strings.TrimSpace(s.query)
	var rows []qpRow
	switch s.kind {
	case "history":
		// 跨项目合并:按「今天 / 昨天 / 更早」分组,组内保持原有顺序
		for _, g := range []string{"今天", "昨天", "更早"} {
			for _, p := range a.projects {
				for _, se := range p.Sessions {
					if se.Archived || se.Group != g || (q != "" && fuzzy(q, se.Title) < 0) {
						continue
					}
					id := se.ID
					rows = append(rows, qpRow{group: g, icon: "chat", title: se.Title, right: se.Clock, current: id == a.curSess,
						run: func() { a.OpenSession(id) }})
				}
			}
		}
	case "symbols":
		syms := a.symbolsForFile(s.symFile)
		for _, sy := range syms {
			if q != "" && fuzzy(q, sy.Name) < 0 {
				continue
			}
			depth := 0
			for k := sy.Parent; k >= 0; k = syms[k].Parent {
				depth++
			}
			line, col, name := sy.Line, sy.Col, sy.Name
			icon := "cube"
			if sy.Kind == "h" {
				icon = "fileText"
			}
			rows = append(rows, qpRow{group: path.Base(s.symFile) + " 中的符号", icon: icon, title: strings.Repeat("  ", depth) + name, right: itoa(line + 1),
				run: func() { a.jumpTo(s.symFile, line, col, col+len(name)) }})
		}
	case "cmpfiles":
		for _, f := range s.cmpFiles {
			if q != "" && fuzzy(q, f.Path) < 0 {
				continue
			}
			p := f.Path
			rows = append(rows, qpRow{group: " ", icon: fileIcon(path.Base(p)), title: p, right: statusLetter(f.Status), run: func() { a.openCompareWork(s.cmpC, p) }})
		}
	case "branch":
		exact := false
		add := func(group string, b vcs.Branch) {
			if q != "" && fuzzy(q, b.Name) < 0 {
				return
			}
			if b.Name == q && !b.Remote {
				exact = true
			}
			name := b.Name
			r := qpRow{group: group, icon: "branch", title: name, right: b.When, current: b.Current, run: func() { a.switchBranchNamed(name) }}
			if b.Current {
				r.icon, r.right = "check", "当前"
			}
			if b.Remote {
				r.icon = "globe"
			}
			rows = append(rows, r)
		}
		for _, b := range s.branches {
			if !b.Remote {
				add("本地", b)
			}
		}
		for _, b := range s.branches {
			if b.Remote {
				add("远程", b)
			}
		}
		if q != "" && !exact && vcs.ValidName(q) {
			rows = append(rows, qpRow{group: " ", icon: "plus", title: "创建并切换到 “" + q + "”", right: "⏎", run: func() { a.createBranchNamed(q) }})
		}
	}
	return rows
}

func (a *App) gitOp(what, okMsg string, f func() error) {
	if a.repo == nil {
		a.toast = "当前项目不是 Git 仓库"
		return
	}
	run := func() {
		err := f()
		done := func() {
			a.branch = a.repo.Current()
			if err != nil {
				log.Warn(what+"失败", "err", err)
				a.toast = what + "失败:" + err.Error()
			} else {
				log.Info(what, "branch", a.branch)
				a.toast = okMsg
			}
		}
		if a.win != nil {
			a.win.Update(done)
		} else {
			done()
		}
	}
	if a.win != nil {
		go func() { defer diag.Recover("app.git"); run() }()
	} else {
		run()
	}
}

func (a *App) quickView(c *ui.Context) {
	s := a.qp
	if s == nil {
		return
	}
	t := a.tk
	open := true
	rows := a.qpRows()
	if s.query != s.lastQ {
		s.sel, s.lastQ = 0, s.query
	}
	if s.sel >= len(rows) {
		s.sel = len(rows) - 1
	}
	if s.sel < 0 {
		s.sel = 0
	}
	move := func(d int) {
		if len(rows) > 0 {
			s.sel = (s.sel + d + len(rows)) % len(rows)
		}
	}
	run := func(i int) {
		if i < 0 || i >= len(rows) {
			return
		}
		f := rows[i].run
		a.closeQuick()
		f()
	}
	ph, ic := "搜索历史会话", "history"
	switch s.kind {
	case "branch":
		ph, ic = "切换或创建分支", "branch"
		if e := a.activeRepo(); e != nil {
			ph = "切换或创建 " + e.name + " 的分支"
		}
	case "symbols":
		ph, ic = "转到符号", "cube"
	case "cmpfiles":
		ph, ic = "选择要与工作区比较的文件("+s.cmpC.Short()+")", "diff"
	}
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(t.Scrim).Justify(ui.Start).AlignItems(ui.Center).Padding(96, 0, 0, 0)
		panel.Width(qpW).Background(t.Raised).Radius(t.RXl()).Clip().Column().Border(1, t.Bd1)
		for _, sh := range t.Shadow3() {
			panel.Shadow(sh.X, sh.Y, sh.Blur, sh.Spread, sh.Color)
		}
		a.dialogIn("quick", backdrop, panel)
		panel.Children(func() {
			ui.Row(c).Height(36).PaddingX(14).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
				kit.Ic(c, ic, kit.S12).TextColor(t.Tx3)
				in := ui.TextInputBase(c.Key("qp-in"), &s.query).Grow(1).FontSize(theme.FsBody).AutoFocus().Placeholder(ph).PlaceholderColor(t.Tx4)
				in.OnSubmit(func() { run(s.sel) })
				in.OnShortcut(0, ui.KeyDown, func() { move(1) })
				in.OnShortcut(0, ui.KeyUp, func() { move(-1) })
				in.OnShortcut(ui.Ctrl, ui.KeyN, func() { move(1) })
				in.OnShortcut(ui.Ctrl, ui.KeyP, func() { move(-1) })
			})
			if s.kind == "branch" && a.multiRoot() && a.repoCount() > 1 { // 多仓库:顶部 chip 选仓库,下面是该仓库的分支
				ui.Row(c).Gap(6).Padding(6, 10).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
					for i, e := range a.repos {
						if e.repo == nil {
							continue
						}
						i, e := i, e
						on := i == a.repoAct
						b := ui.ButtonBase(c).Key("qp-repo-" + itoa(i)).Height(22).PaddingX(8).Gap(5).Radius(11).AlignItems(ui.Center).Cursor(ui.CursorPointer).
							FontSize(theme.FsCap).Transition(kit.Fade)
						switch {
						case on:
							b.Background(t.AcSoft).TextColor(t.AcTx)
						case b.Hovered():
							b.Background(t.Fill2).TextColor(t.Tx2)
						default:
							b.Background(ui.Transparent).TextColor(t.Tx3).Border(1, t.Bd2)
						}
						b.OnClick(func() {
							q := s.query
							a.setActiveRepo(i)
							a.OpenQuick("branch", q)
						})
						b.Children(func() {
							kit.Ic(c, "folder", kit.S12)
							ui.Text(c, e.name).SingleLine()
						})
					}
				})
			}
			a.qpList(c, rows, run)
		})
	})
	if !open {
		a.closeQuick()
	}
}

// qpRowY 第 i 行的顶边与高(含前面的分组标题)。
func qpRowY(rows []qpRow, i int) (top, h float32) {
	var y float32 = 4
	last := "\x00"
	for k, r := range rows {
		if r.group != last {
			if strings.TrimSpace(r.group) != "" {
				y += qpCapH
			} else {
				y += 9 // 分隔线
			}
			last = r.group
		}
		if k == i {
			return y, qpRowH
		}
		y += qpRowH
	}
	return y, qpRowH
}

func (a *App) qpList(c *ui.Context, rows []qpRow, run func(int)) {
	t := a.tk
	s := a.qp
	if s.sel != s.lastSel {
		top, h := qpRowY(rows, s.sel)
		switch {
		case s.sel == 0:
			s.listSt.Y = 0
		case top < s.listSt.Y:
			s.listSt.Y = top - qpCapH
		case top+h > s.listSt.Y+qpMaxH:
			s.listSt.Y = top + h - qpMaxH + 4
		}
		if s.listSt.Y < 0 {
			s.listSt.Y = 0
		}
		s.lastSel = s.sel
	}
	// 高度随行数增长,封顶 qpMaxH
	n := float32(8)
	last := "\x00"
	for _, r := range rows {
		if r.group != last {
			if strings.TrimSpace(r.group) != "" {
				n += qpCapH
			} else {
				n += 9
			}
			last = r.group
		}
		n += qpRowH
	}
	if len(rows) == 0 {
		n = 64
	}
	if n > qpMaxH {
		n = qpMaxH
	}
	ui.Scroll(c).Key("qp-list").TrackScroll(&s.listSt).Height(n).Children(func() {
		ui.Column(c).Padding(4).Children(func() {
			last := "\x00"
			for i, r := range rows {
				if r.group != last {
					last = r.group
					if strings.TrimSpace(r.group) != "" {
						ui.Row(c).Height(qpCapH).PaddingX(8).AlignItems(ui.Center).FontSize(theme.FsMicro).FontWeight(600).TextColor(t.Tx3).
							Children(func() { ui.Text(c, r.group) })
					} else {
						ui.Box(c).Height(1).Background(t.Bd1).Margin(4, 4)
					}
				}
				i, r := i, r
				row := ui.ButtonBase(c).Key("qp-" + itoa(i)).Height(qpRowH).PaddingX(8).Gap(8).Radius(t.RSm()).AlignItems(ui.Center).Cursor(ui.CursorDefault)
				switch {
				case i == s.sel:
					row.Background(t.AcSoft).TextColor(t.AcTx)
				case row.Hovered():
					row.Background(t.Fill1)
				case r.current:
					row.Background(t.Fill2)
				}
				row.OnClick(func() { run(i) })
				row.Children(func() {
					kit.Ic(c, r.icon, kit.S12)
					ui.Text(c, r.title).Grow(1).SingleLine().MinWidth(0)
					if r.right != "" {
						col := t.Tx4
						if i == s.sel {
							col = t.AcTx
						}
						ui.Text(c, r.right).FontSize(theme.FsCap).TextColor(col).SingleLine().Shrink(0)
					}
				})
			}
			if len(rows) == 0 {
				msg := "没有匹配的项"
				if s.loadErr != "" {
					msg = "无法读取分支:" + s.loadErr
				}
				ui.Row(c).Height(56).Justify(ui.Center).AlignItems(ui.Center).TextColor(t.Tx3).Children(func() { ui.Text(c, msg) })
			}
		})
	})
}
