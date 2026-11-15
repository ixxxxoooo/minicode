package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// Git 图谱标签(vGraph):把导航栏里窄窄的「图形」放大成一整页——
// 泳道 | 描述与引用 | 作者 | 日期 | 哈希,顶部搜索(描述 / 作者 / 哈希 / 分支名,不匹配的行变淡而不是消失,泳道保持连续),
// 选中一行后底部是详情(元信息 + 涉及文件,点文件在工作区开该次提交的差异)。右键菜单与导航栏里一致。

type graphTabState struct {
	q   string // 搜索词
	sel string // 选中的提交哈希
}

const graphBigRowH = graphRowH // 与泳道格同高,线条才连得上

// openGraph 在工作区打开 Git 图谱标签。
func (a *App) openGraph() {
	a.scmTickNow()
	a.OpenWorkspaceTab(vGraph, "")
}

// scmTickNow 立刻保证图谱数据已读(标签打开时 SCM 面板可能没显示过)。
func (a *App) scmTickNow() {
	if a.repo != nil && !a.sc.loaded && !a.sc.loading {
		a.scmRefresh()
	}
}

func (cm graphMatch) hit(c vcs.Commit) bool {
	if cm.q == "" {
		return true
	}
	q := strings.ToLower(cm.q)
	if strings.Contains(strings.ToLower(c.Subject), q) || strings.Contains(strings.ToLower(c.Author), q) || strings.HasPrefix(c.Hash, q) {
		return true
	}
	for _, r := range c.Refs {
		if strings.Contains(strings.ToLower(r), q) {
			return true
		}
	}
	return false
}

type graphMatch struct{ q string }

func (a *App) graphView(c *ui.Context) {
	t := a.tk
	s := &a.sc
	g := &a.graphTab
	a.scmTickNow()
	m := graphMatch{g.q}
	a.wsHead(c, "fork", t.Ac, "Git 图谱", func() {
		if e := a.activeRepo(); e != nil { // 多根:仓库下拉(切换 = 切换活动仓库)
			if a.repoCount() > 1 {
				a.folderPicker(c, "graph-repo", a.repoAct, "", true, func(i int) {
					a.setActiveRepo(i)
					g.sel = ""
				})
			} else {
				ui.Text(c, e.name).FontWeight(600).TextColor(t.Tx2)
			}
		}
		if len(s.log) > 0 {
			ui.Text(c, a.branch+" · "+itoa(len(s.log))+" 个提交").FontSize(theme.FsCap).TextColor(t.Tx3)
		}
	}, func() {
		in := ui.TextInputBase(c.Key("graph-q"), &g.q).Width(200).Height(24).PaddingX(8).Radius(t.RSm()).Background(t.Fill1).
			FontSize(theme.FsCap).Placeholder("搜索描述 / 作者 / 哈希 / 分支").PlaceholderColor(t.Tx4)
		_ = in
		if g.q != "" && kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "清除搜索"}).Clicked() {
			g.q = ""
		}
		if kit.Ib(c, "refresh", kit.IbOpt{Small: true, Tip: "刷新"}).Clicked() {
			a.scmRefresh()
		}
	})
	if a.repo == nil {
		ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "当前项目还不是 Git 仓库").TextColor(t.Tx3)
		})
		return
	}
	if len(s.log) == 0 {
		ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, map[bool]string{true: "还没有提交", false: "正在读取提交…"}[s.loaded]).TextColor(t.Tx3)
		})
		return
	}
	gw := float32(s.lanes*graphLaneW + 10)
	// 表头
	ui.Row(c).Height(22).Shrink(0).PaddingX(10).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Bd1).Children(func() {
		hd := func(s string, w float32) {
			e := ui.Text(c, s).FontSize(theme.FsMicro).TextColor(t.Tx4).FontWeight(600).SingleLine()
			if w > 0 {
				e.Width(w).Shrink(0)
			} else {
				e.Grow(1)
			}
		}
		ui.Box(c).Width(gw).Shrink(0)
		hd("描述", 0)
		hd("作者", 110)
		hd("日期", 84)
		hd("哈希", 60)
	})
	ui.Scroll(c).Key("graph-scroll").Grow(1).MinHeight(0).Children(func() {
		for i, cm := range s.log {
			a.graphBigRow(c, i, cm, s.rows[i], gw, m.hit(cm))
		}
	})
	a.graphDetail(c)
}

func (a *App) graphBigRow(c *ui.Context, i int, cm vcs.Commit, gr vcs.GraphRow, gw float32, hit bool) {
	t := a.tk
	g := &a.graphTab
	on := g.sel == cm.Hash
	r := ui.ButtonBase(c).Key("gr-" + cm.Hash).Height(graphBigRowH).PaddingX(10).Gap(8).AlignItems(ui.Center).Cursor(ui.CursorPointer).TextColor(t.Tx2)
	switch {
	case on:
		r.Background(t.AcSoft)
	case r.Hovered():
		r.Background(t.Fill1)
	}
	if !hit {
		r.Opacity(.35)
	}
	r.ContextMenu(func(m *ui.Menu) {
		for _, it := range []struct{ label, op string }{
			{"检出此提交", "checkout"}, {"从此提交创建分支…", "branch"}, {"-", ""},
			{"拣选(cherry-pick)", "pick"}, {"还原(revert)", "revert"}, {"-", ""}, {"复制哈希", "copy"},
		} {
			if it.label == "-" {
				m.Separator()
			} else if m.Item(it.label).Chosen() {
				a.scmCommitOp(cm, it.op)
			}
		}
	})
	r.OnClick(func() {
		g.sel = cm.Hash
		a.scmCommitFiles(cm.Hash)
	})
	r.Children(func() {
		// 泳道格子比行矮时仍用同一个绘制函数(行高由 graphRowH 决定,这里居中放)
		ui.Column(c).Height(graphBigRowH).Shrink(0).Justify(ui.Center).Children(func() {
			a.graphCell(c, gr, gw, i == 0, t)
		})
		ui.Row(c).Grow(1).MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, cm.Subject).SingleLine().Shrink(1).MinWidth(0).FontWeight(map[bool]int{true: 600, false: 400}[i == 0])
			for _, ref := range cm.Refs {
				switch {
				case ref == "HEAD":
					kit.Tag(c, ref, kit.KAc)
				case strings.HasPrefix(ref, "origin/"):
					kit.Tag(c, ref, kit.KInfo)
				default:
					kit.Tag(c, ref, kit.KOk)
				}
			}
		})
		ui.Text(c, cm.Author).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Width(110).Shrink(0)
		ui.Text(c, cm.When).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().Width(84).Shrink(0)
		ui.Text(c, cm.Short()).FontSize(theme.FsCap).TextColor(t.Tx4).Font("monospace").SingleLine().Width(60).Shrink(0)
	})
}

// graphDetail 是选中提交的详情:元信息 + 涉及文件(单击预览、双击固定)。
func (a *App) graphDetail(c *ui.Context) {
	t := a.tk
	g := &a.graphTab
	var cm *vcs.Commit
	for i := range a.sc.log {
		if a.sc.log[i].Hash == g.sel {
			cm = &a.sc.log[i]
		}
	}
	if cm == nil {
		return
	}
	ui.Column(c).Key("graph-detail").Height(190).Shrink(0).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1).Background(t.Raised).Children(func() {
		ui.Row(c).Height(30).Shrink(0).PaddingX(10).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, cm.Subject).FontWeight(600).SingleLine().Shrink(1).MinWidth(0)
			ui.Spacer(c).Grow(1)
			ui.Text(c, cm.Short()+" · "+cm.Author+" · "+cm.When).FontSize(theme.FsCap).TextColor(t.Tx3).Font("monospace").SingleLine().Shrink(0)
			if kit.Ib(c, "copy", kit.IbOpt{Small: true, Tip: "复制完整哈希"}).Clicked() {
				a.scmCommitOp(*cm, "copy")
			}
			if kit.Ib(c, "x", kit.IbOpt{Small: true, Tip: "关闭详情"}).Clicked() {
				g.sel = ""
			}
		})
		ui.Scroll(c).Key("graph-files").Grow(1).MinHeight(0).PaddingX(6).Children(func() {
			files, ok := a.sc.files[cm.Hash]
			if !ok {
				ui.Text(c, "正在读取…").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(4, 6)
				return
			}
			if len(files) == 0 {
				ui.Text(c, "没有文件改动(合并提交或空提交)").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(4, 6)
			}
			for _, ch := range files {
				ch := ch
				fr := ui.ButtonBase(c).Key("gf-" + cm.Hash + ch.Path).Height(24).PaddingX(6).Gap(6).AlignItems(ui.Center).Radius(t.RSm()).Cursor(ui.CursorPointer).TextColor(t.Tx2)
				if fr.Hovered() {
					fr.Background(t.Fill1)
				}
				if fr.Clicked() {
					a.scmOpenCommitFile(*cm, ch.Path, false)
				}
				if fr.DoubleClicked() {
					a.scmOpenCommitFile(*cm, ch.Path, true)
				}
				fr.Children(func() {
					ui.Text(c, ch.Path).SingleLine().Shrink(1).MinWidth(0).FontSize(theme.FsCap)
					ui.Spacer(c).Grow(1)
					a.scmStat(c, ch)
				})
			}
		})
	})
}
