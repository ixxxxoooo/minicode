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
	navDropStep   = 30 * time.Millisecond // 相邻两行的间隔
	navDropMaxIdx = 12                    // 第 12 行之后不再继续推迟(长列表不至于等太久)
	navDropShift  = 10                    // 起点在落点上方多少 px
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
		a.navDropAt = time.Now()
	}
	a.navDropKey = key
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
	i = min(i, navDropMaxIdx)
	delay := time.Duration(float64(kit.Scale(time.Duration(i)*navDropStep)) * rate)
	p := float32(time.Since(a.navDropAt)-delay) / float32(dur)
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
