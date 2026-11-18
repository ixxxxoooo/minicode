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
	scmGraphDef    float32 = 200 // 图形分区默认高度
	scmRepoRowH            = 24  // 仓库行高(与图形行 24、文件行 22 一致)
	scmRepoMaxRows         = 5   // 仓库块最多显示的行数,超出在块内滚动
	scmGraphHead           = 27  // 图形标题行 26 + 上边线 1
)

// scmGraphBodyHeight 返回图形列表区(不含标题)的实际高度:
//   - 用户拖过(a.ly.scmGraph > 0):严格按拖出的高度,夹在 [sideSecMin, 上限];
//   - 没拖过:自动占满更改列表下方的剩余空间(更改少或为空时图形自动上移),但不小于默认 200;
//     更改多放不下时保持 200,更改列表自己滚动。
//
// 上一帧尚未量到区域高度时只用默认值。
func (a *App) scmGraphBodyHeight() float32 {
	s := &a.sc
	hi := a.scmGraphMax()
	if a.ly.scmGraph > 0 {
		return clampF(a.ly.scmGraph, sideSecMin, hi)
	}
	h := scmGraphDef
	if s.splitH > 0 {
		h = max(h, s.splitH-s.contentH-scmGraphHead)
	}
	return clampF(h, sideSecMin, hi)
}

// scmGraphMax 返回图形列表区的高度上限:「更改列表 + 图形」区域高度减去标题行,再给更改列表留 48px。
// 上一帧尚未量到时给 400。
func (a *App) scmGraphMax() float32 {
	if a.sc.splitH <= 0 {
		return 400
	}
	return max(sideSecMin, a.sc.splitH-scmGraphHead-48)
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
			a.resizeHandle(c, lyScmGraph, "t", h, 0, sideSecMin, a.scmGraphMax(), func(v float32) { a.ly.scmGraph = v })
		}
		hd := a.navDrop(ui.ButtonBase(c).Key("scm-h-graph")).Height(theme.Px(26)).PaddingX(4).Gap(4).AlignItems(ui.Center).Cursor(ui.CursorPointer).TextColor(t.Tx3) // 标题行悬停不加底色,直接点
		hd.OnClick(func() { s.folded["graph"] = !s.folded["graph"] })
		hd.Children(func() {
			kit.Ic(c, map[bool]string{true: "chevD", false: "chevR"}[open], kit.S12)
			ui.Text(c, "图形").FontSize(theme.Px(10.5)).FontWeight(500).LetterSpacing(.3)
			ui.Spacer(c).Grow(1)
			if has {
				ui.Text(c, a.scmLogCount()).FontSize(theme.FsMicro).TextColor(t.Tx4)
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
			a.scmGraphList(c, h)
		})
	})
}

// scmGraphList 画图形分区里的提交列表:虚拟列表(只构建可见行)+ 末尾状态行,滚到离底部不足一屏就预取下一页。
// 参数 viewH:列表可视高度(落下动画估算可见行用)。
func (a *App) scmGraphList(c *ui.Context, viewH float32) {
	s := &a.sc
	n := len(s.log)
	// 提交记录常常比「切到 Git」晚几百毫秒才读完:这时导航栏的逐行落下早已播完,
	// 数据一到整片图谱就会一下子蹦出来。数据晚到(距导航落下起点 > 400 ms)时,图谱自己从头逐行落一遍。
	if !s.graphShown {
		s.graphShown, s.graphAt = true, kit.Now()
	}
	late := s.graphAt.Sub(a.navDropAt) > 400*time.Millisecond && kit.Since(s.graphAt) < 2*time.Second
	gw := float32(s.lanes*graphLaneW + 6)
	s.sideList.Key = func(i int) any { // 行跟着提交走:追加新行、刷新时滚动位置与展开状态都不乱
		if i < len(s.log) {
			return s.log[i].Hash
		}
		return "scm-log-foot"
	}
	a.scmLogBind(&s.sideList) // 滚到末尾附近预取下一页
	list := ui.List(c, &s.sideList, n+1, func(i int) {
		if i >= len(s.log) || i >= len(s.rows) {
			a.scmLogFooter(c, "")
			return
		}
		cm := s.log[i]
		saveAt := a.navDropAt
		if late {
			a.navDropAt = s.graphAt
		}
		drop := func(el ui.Element) ui.Element { return a.listNavDrop(el, i, n, &s.sideList, s.sideSt.Y, viewH) }
		ui.Column(c).Key("scm-cr-" + cm.Hash).Children(func() { a.scmCommitRow(c, i, cm, s.rows[i], gw, drop) })
		a.navDropAt = saveAt
	}).Grow(1).MinHeight(0).TrackScroll(&s.sideSt)
	kit.ApplyNativeScroll(list, "scm-graph", &s.sideSt, false, true)
}
