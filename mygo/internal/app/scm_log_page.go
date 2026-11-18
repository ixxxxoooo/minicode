package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
	"github.com/ixxxxoooo/minicode/internal/vcs"
)

// Git 图谱的懒加载(设计稿「多仓库 Git」页「Git 图谱 · 滚动到底继续读」):
// 首批读 scmLogFirst 条,滚到离底部不足 scmLogPrefetch 行时再读 scmLogStep 条,直到读完或到上限。
// 分页 = 把请求上限加大后「整体重读」(git log -n 新上限),不用 --skip:两次读取之间有新提交 / 分支变动时
// 不会错位、重复或漏行;泳道每次对全部已加载提交整体重算(vcs.Layout 是纯函数),连线不会在分页处断开。
// 左栏「图形」分区与工作区「Git 图谱」整页共用同一份数据与同一套状态。
//
// @author ixxxxoooo
const (
	scmLogFirst    = 100  // 首批条数
	scmLogStep     = 200  // 每次翻页增加的条数
	scmLogMax      = 5000 // 加载上限,防止无限增长
	scmLogPrefetch = 12   // 可见区末尾离列表末尾不足这么多行就预取下一页
	scmLogFootH    = 24   // 列表末尾状态行高
)

// scmLogLimit 返回此刻应向 git 请求的提交条数(自动刷新也只重读这个窗口)。
// 返回值:已记下的上限;还没翻过页时是首批条数。
func (a *App) scmLogLimit() int {
	if a.sc.logLimit <= 0 {
		return scmLogFirst
	}
	return a.sc.logLimit
}

// scmApplyLog 在界面线程应用后台读回的提交,并重算泳道与「是否还有更早的提交」。
// 参数 lg / err:读回的提交与错误;limit:这次请求的条数上限。
// 读取失败时:已有提交则原样保留(不清空列表)、记下失败原因并把上限退回已加载条数,等用户重试;
// 还没有提交(首次读取)则显示为空,与改动前一致。
func (a *App) scmApplyLog(lg []vcs.Commit, err error, limit int) {
	s := &a.sc
	paging := s.logPaging
	s.logPaging = false
	if err != nil && len(s.log) > 0 {
		s.logErr = err.Error()
		s.logLimit = max(scmLogFirst, len(s.log))
		if paging {
			log.Warn("读取更早的提交失败", "limit", limit, "err", err)
		} else {
			log.Debug("刷新提交图谱失败", "err", err)
		}
		return
	}
	s.logErr = ""
	s.log = lg
	s.logMore = len(lg) >= limit && limit < scmLogMax
	s.rows = vcs.Layout(lg)
	s.lanes = 1
	for _, r := range s.rows {
		s.lanes = max(s.lanes, r.Width)
	}
}

// scmLoadMore 读取更早的提交:上限加 scmLogStep 后整体重读。没有更多 / 正在读 / 没有仓库时什么都不做。
func (a *App) scmLoadMore() {
	s := &a.sc
	if !s.logMore || s.logPaging || a.repo == nil {
		return
	}
	s.logLimit = min(a.scmLogLimit()+scmLogStep, scmLogMax)
	s.logPaging, s.logErr = true, ""
	log.Info("读取更早的提交", "已加载", len(s.log), "新上限", s.logLimit)
	a.scmRefresh()
}

// scmLogBind 给图谱列表接上懒加载,在每次构建列表前调用。参数 st:图谱列表的状态。
//   - 用户滚动使可见行进入末尾 scmLogPrefetch 行内:mygo 的 ListState.EndReached 回调一次(v0.10 起),
//     在回调里读下一页;不再每帧自己读 Visible() 判断。
//   - 兜底:列表整体装得下(搜索时匹配很少、一屏都填不满)用户滚不动,EndReached 不会触发,
//     此时列表已显示到末尾(AtEnd),就继续往更早的历史里找,直到填满一屏或读完。
//     上一次失败时不兜底(等用户点「重试」,避免失败后每帧重试)。
func (a *App) scmLogBind(st *ui.ListState) {
	st.NearEnd = scmLogPrefetch
	st.EndReached = func(int) { a.scmLoadMore() }
	s := &a.sc
	if s.logMore && !s.logPaging && s.logErr == "" && st.AtEnd() {
		a.scmLoadMore()
	}
}

// scmLogCount 返回头部计数文案:读完写「N 个提交」,没读完写「已加载 N 个」。
func (a *App) scmLogCount() string {
	s := &a.sc
	if s.logMore {
		return "已加载 " + itoa(len(s.log)) + " 个"
	}
	return itoa(len(s.log)) + " 个提交"
}

// scmLogFooter 画列表末尾的状态行:读取中(转圈)/ 失败(重试)/ 还有更多(加载更多)/ 已到最早的提交。
// 参数 endText:读完时显示的文案(图谱搜索用来写匹配结果),空串 = 默认「已到最早的提交 · 共 N 个」。
// 返回值:无;这一行始终是 scmLogFootH 高,不随状态变化,列表因此不会抖动。
func (a *App) scmLogFooter(c *ui.Context, endText string) {
	t := a.tk
	s := &a.sc
	ui.Row(c).Key("scm-log-foot").Height(scmLogFootH).Shrink(0).PaddingX(8).Gap(6).AlignItems(ui.Center).Justify(ui.Center).FontSize(theme.FsCap).TextColor(t.Tx4).Children(func() {
		switch {
		case s.logErr != "":
			kit.Ic(c, "alert", kit.S12).TextColor(t.Err)
			ui.Text(c, "读取更早的提交失败").TextColor(t.Err).Tooltip(s.logErr)
			kit.Btn(c, "重试", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(a.scmLoadMore)
		case s.logPaging:
			kit.Spin(c, t.Ac)
			ui.Text(c, "正在读取更早的提交…")
		case s.logMore:
			kit.Btn(c, "加载更多", kit.BtnOpt{Small: true, Variant: kit.BtnGhost}).OnClick(a.scmLoadMore)
		case len(s.log) >= scmLogMax:
			ui.Text(c, "已加载 "+itoa(len(s.log))+" 个提交(上限)")
		case endText != "":
			ui.Text(c, endText)
		default:
			ui.Text(c, "已到最早的提交 · 共 "+itoa(len(s.log))+" 个")
		}
	})
}
