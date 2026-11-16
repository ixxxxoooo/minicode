package kit

import (
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

// 滚动条显示策略(与设置 appearance.scrollbar 分段一致)。
const (
	ScrollbarFollowSystem = 0 // 沿用 MyGo 默认:鼠标在内容区上就显示
	ScrollbarAutoHide     = 1 // 滚动 / 拖动 / 靠近滚动条时显示,静止约 3 秒后隐藏
	ScrollbarAlways       = 2
)

// scrollIdle 是静止多久后隐藏滚动条(对标 macOS 触控板 overlay 滚动条、VS Code)。
const scrollIdle = 3 * time.Second

var (
	scrollMu    sync.Mutex
	scrollMode  = ScrollbarAutoHide
	scrollNow   func() time.Time
	scrollSlots = map[string]*scrollSlot{}
)

// scrollSlot 记录一个带标识的滚动区最近一次「有动静」的时刻。
type scrollSlot struct {
	lastMove time.Time // 最近一次滚动 / 靠近滚动条 / 按下的时刻
	edge     bool      // 指针最近一次落在滚动条轨道上
	seen     bool
	x, y     float64 // 上一帧的滚动偏移,用来发现键盘 / 拖动 / 程序引起的滚动
	w, h     float32 // 上一帧的可视尺寸(输入事件的坐标相对元素左上角,要靠它判断是否在右 / 下边缘)
}

// SetScrollbarMode 设置全局滚动条策略(设置页保存后调用)。
func SetScrollbarMode(mode int) {
	scrollMu.Lock()
	scrollMode = mode
	scrollMu.Unlock()
}

// ScrollbarMode 返回当前滚动条策略。
func ScrollbarMode() int {
	scrollMu.Lock()
	defer scrollMu.Unlock()
	return scrollMode
}

// SetScrollClock 注入当前时间(测试用)。
func SetScrollClock(f func() time.Time) {
	scrollMu.Lock()
	scrollNow = f
	scrollMu.Unlock()
}

func scrollClock() time.Time {
	if scrollNow != nil {
		return scrollNow()
	}
	return time.Now()
}

func slotOf(id string) *scrollSlot {
	sl, ok := scrollSlots[id]
	if !ok {
		sl = &scrollSlot{}
		scrollSlots[id] = sl
	}
	return sl
}

// markScrollActivity 记一次滚动活动(滚轮 / 按下滚动条)。
func markScrollActivity(id string) {
	if id == "" {
		return
	}
	scrollMu.Lock()
	defer scrollMu.Unlock()
	slotOf(id).lastMove = scrollClock()
}

// setScrollEdge 记录指针是否在滚动条轨道上;在轨道上就算有动静。
func setScrollEdge(id string, on bool) {
	if id == "" {
		return
	}
	scrollMu.Lock()
	defer scrollMu.Unlock()
	sl := slotOf(id)
	sl.edge = on
	if on {
		sl.lastMove = scrollClock()
	}
}

// scrollShown 报告这个滚动区的滚动条此刻是否该显示。
// hovered = 指针此刻是否还在这个滚动区里(离开后「靠近滚动条」的状态不再保持)。
// st 不为空时,偏移相对上一帧有变化也算活动。
func scrollShown(id string, st *ui.ScrollState, hovered bool) bool {
	scrollMu.Lock()
	defer scrollMu.Unlock()
	now := scrollClock()
	sl := slotOf(id)
	if st != nil {
		x, y := float64(st.X), float64(st.Y)
		if sl.seen && (x != sl.x || y != sl.y) {
			sl.lastMove = now
		}
		sl.x, sl.y, sl.seen = x, y, true
	}
	if sl.edge && hovered {
		return true // 指针停在滚动条上:一直显示
	}
	return !sl.lastMove.IsZero() && now.Sub(sl.lastMove) < scrollIdle
}

// ScrollFrameActive 报告是否还有滚动条处在「等待隐藏」的 3 秒里(应用据此 frameSoon,到点才能收起)。
func ScrollFrameActive(now time.Time) bool {
	if ScrollbarMode() != ScrollbarAutoHide {
		return false
	}
	scrollMu.Lock()
	defer scrollMu.Unlock()
	for _, sl := range scrollSlots {
		if !sl.lastMove.IsZero() && now.Sub(sl.lastMove) < scrollIdle+150*time.Millisecond {
			return true
		}
	}
	return false
}

// scrollSize 记录滚动区上一帧的可视尺寸。
func scrollSize(id string, r ui.Rect) {
	scrollMu.Lock()
	defer scrollMu.Unlock()
	sl := slotOf(id)
	sl.w, sl.h = r.W, r.H
}

// scrollOnBar 判断指针(x, y 相对滚动区左上角)是否落在右 / 下边缘的滚动条轨道上。
func scrollOnBar(id string, x, y float32, horiz, vert bool) bool {
	th := T.UITheme()
	tw := th.ScrollbarStyle.Resolve(th).TrackWidth
	if tw <= 0 {
		tw = th.ScrollbarWidth + 6
	}
	scrollMu.Lock()
	sl := slotOf(id)
	w, h := sl.w, sl.h
	scrollMu.Unlock()
	if w <= 0 || h <= 0 || x < 0 || x > w || y < 0 || y > h {
		return false
	}
	if vert && x >= w-tw {
		return true
	}
	return horiz && y >= h-tw
}

// applyScrollPolicy 按全局策略给滚动容器挂上滚动条显示规则,并记录活动:
// 自动隐藏 = 活动期间常驻(KeepScrollbars),静止满 3 秒后整条隐藏(Scrollbars(false));
// 没有标识的滚动区没法记时,沿用框架默认(悬停显示)。
func applyScrollPolicy(el ui.Element, id string, st *ui.ScrollState, horiz, vert bool) ui.Element {
	if id != "" {
		scrollSize(id, el.Bounds())
		// 不能用 HandleInput:框架只把指针事件交给命中链上最里面的处理器,滚动区挂了处理器,
		// 它外面容器(终端正文、可拖拽区域……)的按下 / 移动就收不到了(终端无法鼠标选择就是这样来的)。
		// 滚轮 / 拖动 / 键盘引起的滚动由 scrollShown 比较偏移发现,「靠近滚动条」每帧按指针位置判断。
		onBar := false
		if el.Hovered() {
			if x, y, ok := el.PointerPosition(); ok {
				onBar = scrollOnBar(id, x, y, horiz, vert)
			}
		}
		setScrollEdge(id, onBar)
	}
	switch ScrollbarMode() {
	case ScrollbarAlways:
		return el.KeepScrollbars(true)
	case ScrollbarAutoHide:
		if id == "" {
			return el
		}
		if scrollShown(id, st, el.Hovered()) {
			return el.KeepScrollbars(true)
		}
		return el.Scrollbars(false)
	}
	return el
}
