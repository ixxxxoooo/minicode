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
	if a.multiRoot() { // 工作区:顶部仓库总览 + 活动仓库的完整源代码管理(或全部展开)
		ui.Column(c).Grow(1).MinHeight(0).Children(func() {
			if a.scmExpandAll() {
				a.scmExpandAllView(c)
				return
			}
			a.repoOverview(c)
			if a.repo == nil {
				ui.Column(c).Grow(1).Gap(8).Padding(24, 12).AlignItems(ui.Center).Children(func() {
					kit.Ic(c, "branch", 20).TextColor(t.Tx4)
					ui.Text(c, a.activeRepo().name+" 还不是 Git 仓库").TextColor(t.Tx3)
					kit.Btn(c, "初始化仓库", kit.BtnOpt{Variant: kit.BtnSoft, Icon: "branch"}).OnClick(func() { a.initRepo(a.repoAct) })
				})
				return
			}
			a.scmBody(c)
		})
		return
	}
	if a.repo == nil {
		ui.Column(c).Grow(1).Gap(8).Padding(24, 12).AlignItems(ui.Center).Children(func() {
			kit.Ic(c, "branch", 20).TextColor(t.Tx4)
			ui.Text(c, "这个文件夹还不是 Git 仓库").TextColor(t.Tx3)
		})
		return
	}
	a.scmBody(c)
}

// scmBody 是活动仓库的源代码管理主体:分支行、提交框、各分组、储藏、图谱。
func (a *App) scmBody(c *ui.Context) {
	t := a.tk
	s := &a.sc
	a.scmTick()
	staged, work := s.split()
	var conf []vcs.Change // 合并冲突单独成组,排在最前
	for i := 0; i < len(work); {
		if work[i].Status == vcs.StatusConflict {
			conf = append(conf, work[i])
			work = append(work[:i:i], work[i+1:]...)
			continue
		}
		i++
	}
	body := ui.Column(c).Key("scm-body").Grow(1).MinHeight(0)
	defer func() { s.bodyH = body.Bounds().H }() // 上一帧高度:图形分区的高度上限据此算
	body.Children(func() {
		a.scmHead(c)
		a.scmPushGuide(c)
		if e := a.activeRepo(); e != nil { // 多仓库:提交前再确认一眼作用在哪个仓库
			ui.Row(c).Height(18).PaddingX(4).Gap(4).AlignItems(ui.Center).Shrink(0).FontSize(theme.FsMicro).TextColor(t.Tx3).Children(func() {
				ui.Text(c, "提交到").Shrink(0)
				ui.Text(c, e.name).FontWeight(500).TextColor(t.Tx1).SingleLine().Shrink(1).MinWidth(0)
				ui.Text(c, "· "+a.branch).SingleLine().Shrink(0)
			})
		}
		a.wizardBar(c)
		a.scmCommitBox(c, len(staged))
		a.scmReviewRow(c)
		kit.VScroll(c).Key("scm").Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Padding(0, 0, 8, 0).Children(func() {
				a.scmNestedHints(c)
				if len(conf) > 0 {
					a.scmGroup(c, "merge", "合并冲突", conf, "", "")
				}
				if len(staged) > 0 {
					a.scmGroup(c, "staged", "暂存的更改", staged, "minus", "全部取消暂存")
				}
				a.scmGroup(c, "changes", "更改", work, "plus", "全部暂存")
				if len(staged)+len(work)+len(conf) == 0 && s.loaded {
					ui.Text(c, "没有未提交的更改").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(6, 8)
				}
				a.scmStashes(c)
			})
		})
		a.scmGraph(c) // 固定在底部,不随更改列表滚走
	})
}

// scmHead 是分支行:分支按钮 + 刷新 / 拉取 / 推送。
func (a *App) scmHead(c *ui.Context) {
	t := a.tk
	s := &a.sc
	ui.Row(c).Height(theme.Px(30)).Gap(2).AlignItems(ui.Center).Shrink(0).PaddingX(2).Children(func() {
		b := ui.ButtonBase(c).Key("scm-br").Height(theme.Px(24)).PaddingX(7).Gap(5).Radius(t.RSm()).AlignItems(ui.Center).Cursor(ui.CursorPointer).
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
		kit.Ib(c, "layers", kit.IbOpt{Small: true, Tip: "储藏全部更改"}).OnClick(a.scmStashPush)
		kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "刷新"}).OnClick(func() { s.last = s.last.Add(-scmRefreshEvery * 2) })
		pull := kit.Ib(c, "download", kit.IbOpt{Small: true, Tip: "拉取" + countTip("落后", s.behind)})
		pull.OnClick(func() { a.scmSync(true, false) })
		push := kit.Ib(c, "upload", kit.IbOpt{Small: true, Tip: "推送" + countTip("领先", s.ahead)})
		push.OnClick(func() { a.scmSync(false, true) })
		if s.ahead > 0 {
			ui.Text(c, itoa(s.ahead)).FontSize(theme.FsMicro).TextColor(t.AcTx).FontFeatures("tnum").Shrink(0)
		}
		kit.Ib(c, "more", kit.IbOpt{Small: true, Tip: "更多操作"}).Menu(a.scmHeadMenu)
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
		if box.FocusWithin() || s.genning {
			box.BorderColor(t.Ac)
		}
		if s.genning {
			kit.Pulse(box)
		}
		box.Children(func() {
			if s.genning {
				ui.Row(c).Height(44).Padding(6, 10).AlignItems(ui.Center).Gap(8).Children(func() {
					sh := ui.Text(c, "正在生成提交说明…").FontSize(theme.FsBody).TextColor(t.Tx2).Grow(1)
					kit.TextShimmer(sh)
					kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "取消生成"}).OnClick(a.scmCancelGenMsg)
				})
			}
			in := ui.TextArea(c.Key("scm-msg"), &s.msg).Lines(2, 6).Placeholder("消息(⌘↩ 提交)").PlaceholderColor(t.Tx4).
				Background(ui.Transparent).BorderWidth(0).Padding(6, 8, 2, 8).FontSize(theme.FsBody).LineHeight(1.5).TextColor(t.Tx1).FocusRing(false)
			in.OnTextKey(func(k ui.TextKey) bool {
				if k.Key == ui.KeyEnter && k.Mods == ui.Cmd {
					a.scmCommit(false)
					return true
				}
				return false
			})
			if s.genning {
				in.Opacity(.35).Placeholder("") // 生成中只留上面那行流光提示,不再叠一行占位文字
			}
			ui.Row(c).Height(theme.Px(22)).PaddingX(4).AlignItems(ui.Center).Children(func() {
				if s.amend {
					kit.Tag(c, "修改上次提交", kit.KWarn)
				}
				ui.Spacer(c).Grow(1)
				tip := "用 AI 根据更改起草提交说明"
				if s.genning {
					tip = "正在生成…"
				}
				kit.Ib(c, "wand", kit.IbOpt{Small: true, Tip: tip, Loading: s.genning}).OnClick(a.scmGenMsg)
			})
		})
		ui.Row(c).Gap(1).Children(func() {
			ready := (s.msg != "" || s.amend) && (nStaged > 0 || s.amend) && s.busy == ""
			label := "提交"
			if nStaged > 0 {
				label = "提交 " + itoa(nStaged) + " 个文件"
			}
			busy := s.busy == "提交" || s.busy == "提交并同步"
			btn := kit.Btn(c, label, kit.BtnOpt{Variant: kit.BtnPri, Icon: "check", Loading: busy, Disabled: !ready || busy})
			btn.Key("scm-commit").Grow(1).OnClick(func() { a.scmCommit(false) })
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
					if a.mi(c, "scm-sync", "refresh", "提交并同步", "", false).Clicked() {
						a.popup = ""
						a.scmCommitSync()
					}
					if a.mi(c, "scm-undo", "rewind", "撤销上次提交(保留改动)", "", false).Clicked() {
						a.popup = ""
						a.scmUndoCommit()
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
	h := a.navDrop(ui.ButtonBase(c).Key("scm-h-" + id)).Height(theme.Px(24)).PaddingX(4).Gap(4).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).TextColor(t.Tx3)
	if h.Hovered() {
		h.Background(t.Fill1)
	}
	h.OnClick(func() { s.folded[id] = !s.folded[id] })
	h.ContextMenu(func(m *ui.Menu) { a.scmGroupMenu(m, id, cs) })
	h.Children(func() {
		kit.Ic(c, map[bool]string{true: "chevR", false: "chevD"}[s.folded[id]], kit.S12)
		ui.Text(c, title).FontSize(theme.Px(10.5)).FontWeight(500).LetterSpacing(.3)
		ui.Spacer(c).Grow(1)
		if h.Hovered() && actIcon != "" {
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
	a.scmGroupFiles(c, id, cs)
}

func statusColor(t *theme.Tokens, st byte) ui.Color {
	switch st {
	case 'A', 'U':
		return t.Ok
	case 'D', vcs.StatusConflict:
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
	r := a.navDrop(ui.ButtonBase(c).Key("scm-f-" + grp + "-" + ch.Path)).Height(theme.Px(22)).PaddingX(4).PaddingStart(14).Gap(5).AlignItems(ui.Center).Radius(a.tk.RSm()).Cursor(ui.CursorPointer)
	dragRef(r, ch.Path, false)
	a.scmFileInner(c, grp, ch, r)
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
	ui.Text(c, statusLetter(ch.Status)).FontSize(theme.FsCap).FontWeight(500).TextColor(statusColor(t, ch.Status)).Width(12).Shrink(0)
}

// ───────── 图形 ─────────

// scmGraph 的实现见 scm_panes.go(固定在底部的「图形」分区)。

func laneColor(t *theme.Tokens, lane int) ui.Color {
	cols := []ui.Color{t.Ac, t.Info, t.Ok, t.Warn, t.Err}
	return cols[lane%len(cols)]
}

func (a *App) scmCommitRow(c *ui.Context, i int, cm vcs.Commit, gr vcs.GraphRow, gw float32) {
	t := a.tk
	s := &a.sc
	open := s.expanded == cm.Hash
	r := a.navDrop(ui.ButtonBase(c).Key("scm-c-" + cm.Hash)).Height(graphRowH).PaddingX(2).Gap(4).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).TextColor(t.Tx2)
	if open && t.Scheme.ID == "b" {
		r.Background(t.Raised)
		for _, sh := range t.Shadow1() {
			r.Shadow(sh.X, sh.Y, sh.Blur, sh.Spread, sh.Color)
		}
	} else if open {
		r.Background(t.Sel)
	} else if r.Hovered() {
		r.Background(t.Fill2)
	}
	r.ContextMenu(func(m *ui.Menu) { a.commitMenu(m, cm) })
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
			fr := ui.ButtonBase(c).Key("scm-cf-" + cm.Hash + ch.Path).Height(theme.Px(22)).PaddingX(4).Gap(5).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).TextColor(t.Tx2)
			if a.ide.active == "diff:rev-"+cm.Short()+":"+ch.Path {
				kit.SelStyle(fr)
			} else if fr.Hovered() {
				fr.Background(t.Fill1)
			}
			if fr.Clicked() {
				a.scmOpenCommitFile(cm, ch.Path, false)
			}
			if fr.DoubleClicked() {
				a.scmOpenCommitFile(cm, ch.Path, true)
			}
			fr.Children(func() {
				a.fileIc(c, path.Base(ch.Path))
				nm := ui.Text(c, path.Base(ch.Path)).SingleLine().Shrink(0)
				if ch.Status == 'D' {
					nm.Strikethrough()
				}
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

// grpDiffKind 按 SCM 分组决定差异标签对比什么:暂存的更改 = HEAD ↔ 暂存区,其余 = 暂存区 ↔ 工作区。
func grpDiffKind(grp string) string {
	if grp == "staged" {
		return dkStaged
	}
	return dkWork
}

// scmOpenFile 点 SCM 里的文件:冲突 = 打开三方合并标签,其余 = 对应来源的差异标签。
func (a *App) scmOpenFile(grp string, ch vcs.Change, pin bool) {
	if grp == "merge" {
		a.OpenWorkspaceTab(vConflict, a.treePath(ch.Path))
		return
	}
	a.openDiffKind(grpDiffKind(grp), a.treePath(ch.Path), pin)
}

// scmStashes 是「储藏」分组:应用(保留)/ 弹出(应用并删除)/ 删除。
func (a *App) scmStashes(c *ui.Context) {
	t := a.tk
	s := &a.sc
	if len(s.stashes) == 0 {
		return
	}
	h := a.navDrop(ui.ButtonBase(c).Key("scm-h-stash")).Height(theme.Px(24)).PaddingX(4).Gap(4).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).TextColor(t.Tx3).Margin(8, 0, 0, 0)
	if h.Hovered() {
		h.Background(t.Fill1)
	}
	h.OnClick(func() { s.folded["stash"] = !s.folded["stash"] })
	h.Children(func() {
		kit.Ic(c, map[bool]string{true: "chevR", false: "chevD"}[s.folded["stash"]], kit.S12)
		ui.Text(c, "储藏").FontSize(theme.Px(10.5)).FontWeight(500).LetterSpacing(.3)
		ui.Spacer(c).Grow(1)
		kit.Cnt(c, itoa(len(s.stashes)), false)
	})
	if s.folded["stash"] {
		return
	}
	for i, st := range s.stashes {
		i, st := i, st
		r := a.navDrop(ui.Row(c).Key("stash-" + itoa(i))).Height(theme.Px(36)).PaddingX(6).Gap(6).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer)
		if r.Hovered() {
			r.Background(t.Fill1)
		}
		r.OnClick(func() { a.scmStashToggle(i) })
		r.Children(func() {
			kit.Ic(c, "layers", kit.S12).TextColor(t.Tx3)
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, st.Msg).SingleLine().FontSize(theme.FsCap)
				ui.Text(c, st.When+" · "+itoa(st.Files)+" 个文件").FontSize(theme.FsMicro).TextColor(t.Tx4).SingleLine()
			})
			if r.Hovered() {
				kit.Ib(c, "play", kit.IbOpt{Small: true, Tip: "应用(保留储藏)"}).OnClick(func() { a.scmStashApply(i, false) })
				kit.Ib(c, "download", kit.IbOpt{Small: true, Tip: "弹出(应用并删除)"}).OnClick(func() { a.scmStashApply(i, true) })
				kit.Ib(c, "trash", kit.IbOpt{Small: true, Tip: "删除"}).OnClick(func() { a.scmStashDrop(i) })
			}
			open := s.stashOpen != nil && s.stashOpen[i]
			kit.Ic(c, map[bool]string{true: "chevD", false: "chevR"}[open], kit.S12).TextColor(t.Tx4).Shrink(0)
		})
		if s.stashOpen != nil && s.stashOpen[i] {
			a.scmStashFileRows(c, i)
		}
	}
}

// scmReviewRow 是「审查本轮」入口:Claude 这一轮改过文件时显示,点击打开审查标签(改动审查的唯一的家)。
func (a *App) scmReviewRow(c *ui.Context) {
	t := a.tk
	n := len(a.changedFiles())
	if n == 0 {
		return
	}
	r := a.navDrop(ui.ButtonBase(c).Key("scm-review")).Height(theme.Px(28)).PaddingX(8).Gap(6).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).
		Background(t.AcSoft).Margin(0, 2, 6, 2).Shrink(0)
	r.OnClick(a.OpenReview)
	r.Children(func() {
		kit.Ic(c, "eye", kit.S12).TextColor(t.Ac)
		ui.Text(c, "审查本轮改动").FontWeight(500).SingleLine()
		ui.Spacer(c).Grow(1)
		ui.Text(c, itoa(n)+" 个文件").FontSize(theme.FsCap).TextColor(t.Tx3)
		kit.Ic(c, "chevR", kit.S12).TextColor(t.Tx3)
	})
}
