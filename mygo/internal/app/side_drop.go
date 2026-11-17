// 导航栏内容的「依次落下」入场:换分段 / 换形态 / 换项目时,列表行自上而下逐行淡入并从上方落到位。
//
// @author ixxxxoooo
package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

const (
	navDropStep   = 30 * time.Millisecond  // 相邻两行的间隔
	navDropMaxIdx = 12                     // 前 12 行用标准间隔
	navDropTail   = 8 * time.Millisecond   // 第 12 行之后的间隔(长列表仍然逐行往下,但不至于等太久)
	navDropCap    = 800 * time.Millisecond // 总延迟封顶
	navDropShift  = 10                     // 起点在落点上方多少 px
)

// navDropRates 是设置「列表落下速度」(appearance.navDrop:慢 / 标准 / 快 / 无)对应的时长倍数。
// 标准 = 1,与设计稿 .cascade 完全一致(每行 260ms、间隔 30ms);0 = 不播。
var navDropRates = []float64{1.8, 1, 0.55, 0}

// syncNavDrop 每帧在画导航内容之前调用:行序号归零;「分段 + 形态 + 当前项目」变了就记下落下动效的起点。
// 首帧(还没有上一帧的键)不播。无参数、无返回值。
func (a *App) syncNavDrop() {
	a.navDropIdx = 0
	a.navDropRate = navDropRates[1]
	if a.prefs != nil {
		a.navDropRate = navDropRates[min(max(a.prefs.Int(pNavDrop, 1), 0), len(navDropRates)-1)]
	}
	key := a.navSwapKey
	if a.tk.Scheme.ID == "b" && a.NavID() == "chat" {
		// 方案 B 的会话列表只看当前项目 / 工作区:换项目也是「换了一屏内容」
		_, _, p := a.sideBCtx()
		key += "|" + p.ID
	}
	if a.navDropKey != "" && a.navDropKey != key {
		a.navDropAt = kit.Now()
	}
	a.navDropKey = key
}

// navDropDelay 第 i 行(0 起)的出场延迟。前 navDropMaxIdx 行按标准间隔(设计稿 .cascade);
// 再往后用更小的间隔继续往下排,总延迟封顶 navDropCap。
// 曾经第 12 行之后全部同一时刻出场:改动列表一长,后面的「储藏」「图谱」整块一起蹦出来,
// 看上去比上面的内容早了一大截(用户反馈:储藏区提前出现、图谱一整片突然出现)。
func navDropDelay(i int) time.Duration {
	if i <= navDropMaxIdx {
		return time.Duration(i) * navDropStep
	}
	return min(navDropMaxIdx*navDropStep+time.Duration(i-navDropMaxIdx)*navDropTail, navDropCap)
}

// navDrop 给导航栏里的一行套上落下入场,并占用一个行序号(调用顺序 = 自上而下的顺序)。
// 参数 el:这一行的元素。返回同一个元素,便于链式调用。动效关闭、从未触发或已播完时什么都不做。
func (a *App) navDrop(el ui.Element) ui.Element {
	i := a.navDropIdx
	a.navDropIdx++
	rate := a.navDropRate
	dur := time.Duration(float64(kit.DurSlow) * rate)
	if !kit.MotionOn() || a.navDropAt.IsZero() || dur <= 0 {
		return el
	}
	delay := time.Duration(float64(kit.Scale(navDropDelay(i))) * rate)
	p := float32(kit.Since(a.navDropAt)-delay) / float32(dur)
	if p >= 1 {
		return el
	}
	a.frameSoon()
	if p <= 0 {
		return el.Opacity(0).Translate(0, -navDropShift)
	}
	e := kit.EaseOut(p)
	return el.Opacity(e).Translate(0, -navDropShift*(1-e))
}
