package app

import (
	"math"
	"path"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

const (
	graphRowH  = 24 // 图谱行高
	graphLaneW = 10 // 泳道间距
)

// scmView 是左栏「源代码管理」:分支行、提交框、暂存的更改、更改、图形。
func (a *App) scmView(c *ui.Context) {
	t := a.tk
	s := &a.sc
	if s.folded == nil {
		s.folded = map[string]bool{}
	}
	if a.repo == nil {
		ui.Column(c).Grow(1).Gap(8).Padding(24, 12).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "branch", 20).TextColor(t.Tx4)
			ui.Text(c, "这个文件夹还不是 Git 仓库").TextColor(t.Tx3)
		})
		return
	}
	a.scmTick()
	staged, work := s.split()
	ui.Column(c).Grow(1).MinHeight(0).Children(func() {
		a.scmHead(c)
		a.scmCommitBox(c, len(staged))
		ui.Scroll(c).Key("scm").Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Padding(0, 0, 8, 0).Children(func() {
				if len(staged) > 0 {
					a.scmGroup(c, "staged", "暂存的更改", staged, "minus", "全部取消暂存")
				}
				a.scmGroup(c, "changes", "更改", work, "plus", "全部暂存")
				if len(staged)+len(work) == 0 && s.loaded {
					ui.Text(c, "没有未提交的更改").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(6, 8)
				}
				a.scmGraph(c)
			})
		})
	})
}

// scmHead 是分支行:分支按钮 + 刷新 / 拉取 / 推送。
func (a *App) scmHead(c *ui.Context) {
	t := a.tk
	s := &a.sc
	ui.Row(c).Height(30).Gap(2).AlignItems(ui.Center).Shrink(0).PaddingX(2).Children(func() {
		b := ui.ButtonBase(c).Key("scm-br").Height(24).PaddingX(7).Gap(5).Radius(t.RSm()).AlignItems(ui.Center).Cursor(ui.CursorPointer).
			Background(t.Raised).Border(1, t.Bd2).FontSize(theme.FsCap).Shrink(1).MinWidth(0).Tooltip("切换分支")
		if b.Hovered() {
			b.Background(t.Raised.Mix(t.Tx1, .04))
		}
		b.OnClick(func() { a.OpenQuick("branch", "") })
		b.Children(func() {
			kit.Ic(c, "branch", kit.S12).TextColor(t.Tx3)
			ui.Text(c, a.branch).SingleLine().MinWidth(0)
		})
		ui.Spacer(c).Grow(1)
		if s.busy != "" {
			kit.Spin(c, t.Ac)
			ui.Text(c, s.busy+"中").FontSize(theme.FsMicro).TextColor(t.Tx3).Shrink(0)
		}
		kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "刷新"}).OnClick(func() { s.last = s.last.Add(-scmRefreshEvery * 2) })
		pull := kit.Ib(c, "download", kit.IbOpt{Small: true, Tip: "拉取" + countTip("落后", s.behind)})
		pull.OnClick(func() { a.scmSync(true, false) })
		push := kit.Ib(c, "upload", kit.IbOpt{Small: true, Tip: "推送" + countTip("领先", s.ahead)})
		push.OnClick(func() { a.scmSync(false, true) })
		if s.ahead > 0 {
			ui.Text(c, itoa(s.ahead)).FontSize(theme.FsMicro).TextColor(t.AcTx).FontFeatures("tnum").Shrink(0)
		}
	})
}

func countTip(what string, n int) string {
	if n == 0 {
		return ""
	}
	return "(" + what + " " + itoa(n) + " 个提交)"
}

// scmCommitBox 是提交说明输入框 + 提交按钮(带下拉)。
func (a *App) scmCommitBox(c *ui.Context, nStaged int) {
	t := a.tk
	s := &a.sc
	ui.Column(c).Gap(6).Padding(2, 2, 8, 2).Shrink(0).Children(func() {
		box := ui.Column(c).Key("scm-box").Background(t.Raised).Radius(t.RSm()).Border(1, t.Bd2)
		if box.FocusWithin() {
			box.BorderColor(t.Ac)
		}
		box.Children(func() {
			in := ui.TextArea(c.Key("scm-msg"), &s.msg).Lines(2, 6).Placeholder("消息(⌘↩ 提交)").PlaceholderColor(t.Tx4).
				Background(ui.Transparent).BorderWidth(0).Padding(6, 8, 2, 8).FontSize(theme.FsBody).LineHeight(1.5).TextColor(t.Tx1).FocusRing(false)
			in.OnTextKey(func(k ui.TextKey) bool {
				if k.Key == ui.KeyEnter && k.Mods == ui.Cmd {
					a.scmCommit(false)
					return true
				}
				return false
			})
			ui.Row(c).Height(22).PaddingX(4).AlignItems(ui.Center).Children(func() {
				if s.amend {
					kit.Tag(c, "修改上次提交", kit.KWarn)
				}
				ui.Spacer(c).Grow(1)
				kit.Ib(c, "star", kit.IbOpt{Small: true, Tip: "根据更改起草提交说明"}).OnClick(a.scmGenMsg)
			})
		})
		ui.Row(c).Gap(1).Children(func() {
			ready := (s.msg != "" || s.amend) && (nStaged > 0 || s.amend) && s.busy == ""
			label := "提交"
			if nStaged > 0 {
				label = "提交 " + itoa(nStaged) + " 个文件"
			}
			b := ui.ButtonBase(c).Key("scm-commit").Grow(1).Height(26).Radius(t.RSm()).Justify(ui.Center).AlignItems(ui.Center).Gap(5).
				Background(t.Ac).TextColor(t.OnAc).FontWeight(500).Cursor(ui.CursorPointer)
			if !ready {
				b.Opacity(.55)
			} else if b.Hovered() {
				b.Background(t.Ac.Mix(t.Tx1, .10))
			}
			b.OnClick(func() { a.scmCommit(false) })
			b.Children(func() {
				kit.Ic(c, "check", kit.S12)
				ui.Text(c, label).SingleLine()
			})
			dd := ui.ButtonBase(c).Key("scm-dd").Size(26, 26).Radius(t.RSm()).Justify(ui.Center).AlignItems(ui.Center).
				Background(t.Ac).TextColor(t.OnAc).Cursor(ui.CursorPointer).Tooltip("更多提交方式")
			if !ready {
				dd.Opacity(.55)
			}
			dd.OnClick(func() { a.togglePop("scm-more") })
			dd.Children(func() { kit.Ic(c, "chevD", kit.S12) })
			a.popOpen("scm-more", func(open *bool) {
				ui.PopoverBase(c, dd, open, func(panel ui.Element) {
					a.popPanel(panel, 200).AttachTo(dd, ui.AnchorTopRight, ui.AnchorBottomRight).Margin(4, 0, 0, 0)
					if a.mi(c, "scm-push", "upload", "提交并推送", "", false).Clicked() {
						a.popup = ""
						a.scmCommit(true)
					}
					if a.mi(c, "scm-amend", "undo", map[bool]string{true: "取消修改上次提交", false: "修改上次提交"}[s.amend], "", s.amend).Clicked() {
						a.popup = ""
						s.amend = !s.amend
					}
				})
			})
		})
	})
}

// scmGroup 是「暂存的更改」/「更改」分组:标题行(折叠、计数、批量操作)+ 文件行。
func (a *App) scmGroup(c *ui.Context, id, title string, cs []vcs.Change, actIcon, actTip string) {
	t := a.tk
	s := &a.sc
	if len(cs) == 0 && id == "changes" {
		return
	}
	h := ui.ButtonBase(c).Key("scm-h-" + id).Height(24).PaddingX(4).Gap(4).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).TextColor(t.Tx3)
	if h.Hovered() {
		h.Background(t.Fill1)
	}
	h.OnClick(func() { s.folded[id] = !s.folded[id] })
	h.Children(func() {
		kit.Ic(c, map[bool]string{true: "chevR", false: "chevD"}[s.folded[id]], kit.S12)
		ui.Text(c, title).FontSize(10.5).FontWeight(700).LetterSpacing(.3)
		ui.Spacer(c).Grow(1)
		if h.Hovered() {
			paths := make([]string, len(cs))
			for i, ch := range cs {
				paths[i] = ch.Path
			}
			kit.Ib(c, actIcon, kit.IbOpt{Small: true, Tip: actTip}).OnClick(func() {
				if id == "staged" {
					a.scmUnstage(paths...)
				} else {
					a.scmStage(paths...)
				}
			})
		}
		kit.Cnt(c, itoa(len(cs)), false)
	})
	if s.folded[id] {
		return
	}
	for _, ch := range cs {
		a.scmFile(c, id, ch)
	}
}

func statusColor(t *theme.Tokens, st byte) ui.Color {
	switch st {
	case 'A', 'U':
		return t.Ok
	case 'D':
		return t.Err
	case 'R':
		return t.Info
	}
	return t.Warn
}

func statusLetter(st byte) string {
	if st == 'U' {
		return "U"
	}
	return string(st)
}

// scmFile 是一个文件行:图标 + 名 + 目录;悬停显示 打开 / 放弃 / 暂存 操作,否则显示 +N −N 与状态字母。
func (a *App) scmFile(c *ui.Context, grp string, ch vcs.Change) {
	t := a.tk
	r := ui.ButtonBase(c).Key("scm-f-" + grp + "-" + ch.Path).Height(22).PaddingX(4).PaddingStart(14).Gap(5).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer)
	hov := r.Hovered()
	if hov {
		r.Background(t.Fill1)
	}
	if r.Clicked() {
		a.ide.openDiff(ch.Path, false)
	}
	if r.DoubleClicked() {
		a.ide.openDiff(ch.Path, true)
	}
	col := t.Tx2
	if ch.Status == 'D' {
		col = t.Tx4
	}
	r.TextColor(col).Children(func() {
		kit.Ic(c, fileIcon(path.Base(ch.Path)), kit.S12).TextColor(t.Tx3)
		nm := ui.Text(c, path.Base(ch.Path)).SingleLine().Shrink(0)
		if ch.Status == 'D' {
			nm.Strikethrough()
		}
		if d := path.Dir(ch.Path); d != "." {
			ui.Text(c, d).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().Shrink(1).MinWidth(0)
		}
		ui.Spacer(c).Grow(1)
		if hov {
			kit.Ib(c, "file", kit.IbOpt{Small: true, Tip: "打开文件"}).OnClick(func() { a.ide.openFile(ch.Path, true) })
			if grp == "changes" {
				kit.Ib(c, "undo", kit.IbOpt{Small: true, Tip: "放弃更改"}).OnClick(func() { a.scmDiscard(ch.Path) })
				kit.Ib(c, "plus", kit.IbOpt{Small: true, Tip: "暂存"}).OnClick(func() { a.scmStage(ch.Path) })
			} else {
				kit.Ib(c, "minus", kit.IbOpt{Small: true, Tip: "取消暂存"}).OnClick(func() { a.scmUnstage(ch.Path) })
			}
			return
		}
		a.scmStat(c, ch)
	})
}

// scmStat 是行尾的 +N −N 与状态字母。
func (a *App) scmStat(c *ui.Context, ch vcs.Change) {
	t := a.tk
	if ch.Add > 0 {
		ui.Text(c, "+"+itoa(ch.Add)).FontSize(theme.FsMicro).TextColor(t.Ok).FontFeatures("tnum").Shrink(0)
	}
	if ch.Del > 0 {
		ui.Text(c, "−"+itoa(ch.Del)).FontSize(theme.FsMicro).TextColor(t.Err).FontFeatures("tnum").Shrink(0)
	}
	ui.Text(c, statusLetter(ch.Status)).FontSize(theme.FsCap).FontWeight(600).TextColor(statusColor(t, ch.Status)).Width(12).Shrink(0)
}

// ───────── 图形 ─────────

func (a *App) scmGraph(c *ui.Context) {
	t := a.tk
	s := &a.sc
	h := ui.ButtonBase(c).Key("scm-h-graph").Height(24).PaddingX(4).Gap(4).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).TextColor(t.Tx3).Margin(8, 0, 0, 0)
	if h.Hovered() {
		h.Background(t.Fill1)
	}
	h.OnClick(func() { s.folded["graph"] = !s.folded["graph"] })
	h.Children(func() {
		kit.Ic(c, map[bool]string{true: "chevR", false: "chevD"}[s.folded["graph"]], kit.S12)
		ui.Text(c, "图形").FontSize(10.5).FontWeight(700).LetterSpacing(.3)
		ui.Spacer(c).Grow(1)
		if len(s.log) > 0 {
			ui.Text(c, itoa(len(s.log))+" 个提交").FontSize(theme.FsMicro).TextColor(t.Tx4)
		}
	})
	if s.folded["graph"] {
		return
	}
	if len(s.log) == 0 {
		if s.loaded {
			ui.Text(c, "还没有提交").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(6, 8)
		}
		return
	}
	gw := float32(s.lanes*graphLaneW + 6)
	for i, cm := range s.log {
		a.scmCommitRow(c, i, cm, s.rows[i], gw)
	}
}

func laneColor(t *theme.Tokens, lane int) ui.Color {
	cols := []ui.Color{t.Ac, t.Info, t.Ok, t.Warn, t.Err}
	return cols[lane%len(cols)]
}

func (a *App) scmCommitRow(c *ui.Context, i int, cm vcs.Commit, gr vcs.GraphRow, gw float32) {
	t := a.tk
	s := &a.sc
	open := s.expanded == cm.Hash
	r := ui.ButtonBase(c).Key("scm-c-" + cm.Hash).Height(graphRowH).PaddingX(2).Gap(4).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).TextColor(t.Tx2)
	if open {
		r.Background(t.Fill2)
	} else if r.Hovered() {
		r.Background(t.Fill1)
	}
	r.OnClick(func() {
		if open {
			s.expanded = ""
		} else {
			s.expanded = cm.Hash
			a.scmCommitFiles(cm.Hash)
		}
	})
	r.Children(func() {
		a.graphCell(c, gr, gw, i == 0, t)
		ui.Text(c, cm.Subject).SingleLine().Shrink(1).MinWidth(0).FontSize(theme.FsCap)
		for _, ref := range cm.Refs {
			switch {
			case ref == "HEAD":
				kit.Tag(c, ref, kit.KAc)
			case len(ref) > 7 && ref[:7] == "origin/":
				kit.Tag(c, ref, kit.KInfo)
			default:
				kit.Tag(c, ref, kit.KOk)
			}
		}
		ui.Spacer(c).Grow(1)
		if len(cm.Refs) < 2 { // 引用多时让位给标题
			ui.Text(c, cm.When).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().Shrink(0)
		}
	})
	if !open {
		return
	}
	ui.Column(c).PaddingStart(gw + 6).Gap(0).Children(func() {
		ui.Text(c, cm.Short()+" · "+cm.Author+" · "+cm.When).FontSize(theme.FsMicro).TextColor(t.Tx4).Font("monospace").Padding(2, 4)
		files, ok := s.files[cm.Hash]
		if !ok {
			return
		}
		for _, ch := range files {
			fr := ui.ButtonBase(c).Key("scm-cf-" + cm.Hash + ch.Path).Height(22).PaddingX(4).Gap(5).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).TextColor(t.Tx2)
			if fr.Hovered() {
				fr.Background(t.Fill1)
			}
			if fr.Clicked() {
				a.scmOpenCommitFile(cm, ch.Path, false)
			}
			if fr.DoubleClicked() {
				a.scmOpenCommitFile(cm, ch.Path, true)
			}
			fr.Children(func() {
				ui.Text(c, path.Base(ch.Path)).SingleLine().Shrink(0)
				if d := path.Dir(ch.Path); d != "." {
					ui.Text(c, d).FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine().Shrink(1).MinWidth(0)
				}
				ui.Spacer(c).Grow(1)
				a.scmStat(c, ch)
			})
		}
	})
}

// graphCell 画一行的泳道:竖线、汇入 / 分出的曲线、提交节点(第一行 = HEAD 画空心圈)。
func (a *App) graphCell(c *ui.Context, gr vcs.GraphRow, w float32, head bool, t *theme.Tokens) {
	bg := t.BG
	e := ui.Box(c).Size(w, graphRowH).Shrink(0)
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		x := func(l int) float32 { return r.X + 5 + float32(l*graphLaneW) }
		top, mid, bot := r.Y, r.Y+r.H/2, r.Y+r.H
		in := func(set []int, l int) bool {
			for _, v := range set {
				if v == l {
					return true
				}
			}
			return false
		}
		curve := func(x0, y0, x1, y1 float32, col ui.Color) {
			// 竖直起、竖直止的三次贝塞尔,取样成折线
			pts := make([][2]float32, 0, 13)
			for i := 0; i <= 12; i++ {
				u := float64(i) / 12
				b := 3*u*u - 2*u*u*u // smoothstep:x 随 u 平滑过渡
				pts = append(pts, [2]float32{x0 + (x1-x0)*float32(b), y0 + (y1-y0)*float32(u)})
			}
			p.StrokePolyline(pts, 1.5, col)
		}
		for _, l := range gr.Top {
			col := laneColor(t, l)
			switch {
			case in(gr.Join, l):
				curve(x(l), top, x(gr.Node), mid, col)
			case l == gr.Node:
				p.Line(x(l), top, x(l), mid, 1.5, col)
			case in(gr.Bottom, l):
				p.Line(x(l), top, x(l), bot, 1.5, col)
			}
		}
		for _, l := range gr.Bottom {
			col := laneColor(t, l)
			switch {
			case l == gr.Node:
				p.Line(x(l), mid, x(l), bot, 1.5, col)
			case in(gr.Fork, l) || in(gr.Merge, l):
				curve(x(gr.Node), mid, x(l), bot, col)
			}
		}
		// 节点
		nx, col := x(gr.Node), laneColor(t, gr.Node)
		rad := float32(4)
		p.Fill(ui.Rect{X: nx - rad, Y: mid - rad, W: rad * 2, H: rad * 2}, col, rad)
		if head {
			ri := float32(math.Max(float64(rad-1.6), 1))
			p.Fill(ui.Rect{X: nx - ri, Y: mid - ri, W: ri * 2, H: ri * 2}, bg, ri)
		}
	})
}
