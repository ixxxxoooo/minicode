package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

// 源代码管理的固定分区(设计稿「多仓库 Git」页「固定分区:仓库在顶、图形在底」):
// 「仓库」钉在顶部、「图形」钉在底部,都不随中间的更改列表滚走;各自在块内滚动,点标题收起 / 展开,
// 图形上沿有拖拽把手调高度(双击还原,松手记入偏好 layout.scmGraph)。
//
// @author ixxxxoooo
const (
	scmGraphDef     float32 = 200 // 图形分区默认高度
	scmRepoRowH             = 26  // 仓库行高(与文件行 22 / 图形行 24 同一量级)
	scmRepoMaxRows          = 5   // 仓库块最多显示的行数,超出在块内滚动
	scmGraphMaxFrac         = 0.6 // 图形分区最高占源代码管理区高度的比例
)

// scmGraphHeight 返回图形分区当前高度(已夹在范围内)。
// 返回值:像素高度;窗口很矮时上限随之缩小,但不小于 sideSecMin。
func (a *App) scmGraphHeight() float32 {
	return clampF(orDef(a.ly.scmGraph, scmGraphDef), sideSecMin, a.scmGraphMax())
}

// scmGraphBodyHeight 返回图形列表区(不含标题)的实际高度:
// 取「用户拖出的高度 / 默认高度」与「更改列表下方的剩余空间」中较大的一个——
// 更改少或为空时图形自动上移、占满剩余空间;更改多时保持设定高度,更改列表自己滚动。
// 上一帧尚未量到区域高度时只用设定高度。
func (a *App) scmGraphBodyHeight() float32 {
	s := &a.sc
	h := a.scmGraphHeight()
	if s.splitH > 0 && s.contentH >= 0 {
		const headBorder = 27 // 标题行 26 + 上边线 1
		h = max(h, s.splitH-s.contentH-headBorder)
	}
	return h
}

// scmGraphMax 返回图形分区的高度上限:源代码管理区高度的 60%(上一帧取到的高度,尚未取到时给 400)。
func (a *App) scmGraphMax() float32 {
	if a.sc.bodyH <= 0 {
		return 400
	}
	return max(sideSecMin, a.sc.bodyH*scmGraphMaxFrac)
}

// scmGraph 画固定在底部的「图形」分区:上沿拖拽把手 + 可点击收起的标题行 + 块内滚动的提交列表。
// 没有参数;状态取自 a.sc(folded["graph"]、log、rows)与 a.ly.scmGraph。
func (a *App) scmGraph(c *ui.Context) {
	t := a.tk
	s := &a.sc
	open := !s.folded["graph"]
	has := len(s.log) > 0
	h := a.scmGraphBodyHeight()
	ui.Column(c).Key("scm-graph-pane").Shrink(0).BorderWidth(1, 0, 0, 0).BorderColor(t.Bd1).Children(func() {
		if open && has { // 把手只在有内容、展开时出现
			a.resizeHandle(c, lyScmGraph, "t", h, scmGraphDef, sideSecMin, a.scmGraphMax(), func(v float32) { a.ly.scmGraph = v })
		}
		hd := a.navDrop(ui.ButtonBase(c).Key("scm-h-graph")).Height(theme.Px(26)).PaddingX(4).Gap(4).AlignItems(ui.Center).Cursor(ui.CursorPointer).TextColor(t.Tx3) // 标题行悬停不加底色,直接点
		hd.OnClick(func() { s.folded["graph"] = !s.folded["graph"] })
		hd.Children(func() {
			kit.Ic(c, map[bool]string{true: "chevD", false: "chevR"}[open], kit.S12)
			ui.Text(c, "图形").FontSize(theme.Px(10.5)).FontWeight(500).LetterSpacing(.3)
			ui.Spacer(c).Grow(1)
			if has {
				ui.Text(c, itoa(len(s.log))+" 个提交").FontSize(theme.FsMicro).TextColor(t.Tx4)
				kit.Ib(c, "max", kit.IbOpt{Small: true, Tip: "在工作区打开完整图谱"}).OnClick(a.openGraph)
			}
		})
		if !open {
			return
		}
		if !has {
			s.graphShown = false
			if s.loaded {
				ui.Text(c, "还没有提交").FontSize(theme.FsCap).TextColor(t.Tx4).Padding(6, 8)
			}
			return
		}
		ui.Column(c).Key("scm-graph-body").Height(h).Shrink(0).Children(func() {
			kit.VScroll(c).Key("scm-graph").Grow(1).MinHeight(0).Children(func() {
				ui.Column(c).Padding(0, 0, 4, 0).Children(func() { a.scmGraphRows(c) })
			})
		})
	})
}

// scmGraphRows 画提交行(逐行落下动画见下)。
func (a *App) scmGraphRows(c *ui.Context) {
	s := &a.sc
	// 提交记录常常比「切到 Git」晚几百毫秒才读完:这时导航栏的逐行落下早已播完,
	// 数据一到整片图谱就会一下子蹦出来。数据晚到(距导航落下起点 > 400 ms)时,图谱自己从头逐行落一遍。
	if !s.graphShown {
		s.graphShown, s.graphAt = true, kit.Now()
	}
	saveAt, saveIdx := a.navDropAt, a.navDropIdx
	if s.graphAt.Sub(a.navDropAt) > 400*time.Millisecond && kit.Since(s.graphAt) < 2*time.Second {
		a.navDropAt, a.navDropIdx = s.graphAt, 1
	}
	gw := float32(s.lanes*graphLaneW + 6)
	for i, cm := range s.log {
		a.scmCommitRow(c, i, cm, s.rows[i], gw)
	}
	a.navDropAt, a.navDropIdx = saveAt, max(saveIdx, a.navDropIdx)
}
