package kit

import (
	"math"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

// 滚动条显示策略(与设置 appearance.scrollbar 分段一致)。
const (
	ScrollbarFollowSystem = 0
	ScrollbarAutoHide     = 1
	ScrollbarAlways       = 2
)

const scrollFadeOut = 3 * time.Second

var (
	scrollMu     sync.Mutex
	scrollMode   = ScrollbarAutoHide
	scrollNow    func() time.Time
	scrollSlots  = map[string]*scrollSlot{}
	scrollFadeIn = 180 * time.Millisecond
)

type scrollSlot struct {
	st       *ui.ScrollState
	lastMove time.Time
	dragging bool
	barHover bool
	horiz    bool
	vert     bool
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
func SetScrollClock(f func() time.Time) { scrollNow = f }

func scrollClock() time.Time {
	if scrollNow != nil {
		return scrollNow()
	}
	return time.Now()
}

func regScroll(id string, st *ui.ScrollState, horiz, vert bool) {
	if id == "" {
		return
	}
	scrollMu.Lock()
	defer scrollMu.Unlock()
	sl, ok := scrollSlots[id]
	if !ok {
		sl = &scrollSlot{horiz: horiz, vert: vert}
		scrollSlots[id] = sl
	}
	sl.st = st
	sl.horiz = horiz
	sl.vert = vert
}

func markScrollActivity(id string) {
	if id == "" {
		return
	}
	scrollMu.Lock()
	defer scrollMu.Unlock()
	if sl := scrollSlots[id]; sl != nil {
		sl.lastMove = scrollClock()
	}
}

func setScrollBarHover(id string, on bool) {
	scrollMu.Lock()
	defer scrollMu.Unlock()
	if sl := scrollSlots[id]; sl != nil {
		sl.barHover = on
		if on {
			sl.lastMove = scrollClock()
		}
	}
}

// scrollAlpha 返回滚动条不透明度 0–1;拖动 / 悬停滑块时保持 1。
func scrollAlpha(id string) float32 {
	if ScrollbarMode() != ScrollbarAutoHide {
		return 1
	}
	scrollMu.Lock()
	sl := scrollSlots[id]
	scrollMu.Unlock()
	if sl == nil {
		return 0
	}
	now := scrollClock()
	if sl.dragging || sl.barHover {
		return 1
	}
	if sl.lastMove.IsZero() {
		return 0
	}
	elapsed := now.Sub(sl.lastMove)
	if elapsed >= scrollFadeOut {
		return 0
	}
	if !MotionOn() || scrollFadeIn <= 0 {
		return 1
	}
	if elapsed < scrollFadeIn {
		return float32(elapsed) / float32(scrollFadeIn)
	}
	fadeStart := scrollFadeOut - DurFast
	if elapsed > fadeStart {
		left := float32(scrollFadeOut - elapsed)
		return max(0, left/float32(DurFast))
	}
	return 1
}

func scrollBarEdgeHit(r ui.Rect, st *ui.ScrollState, x, y float32, horiz, vert bool) (onBar bool) {
	if st == nil {
		return false
	}
	th := T.UITheme()
	tw := th.ScrollbarStyle.Resolve(th).TrackWidth
	if tw <= 0 {
		tw = th.ScrollbarWidth + 6
	}
	viewW, viewH := r.W, r.H
	if vert && st.MaxY+viewH > viewH+0.5 {
		if x >= r.X+r.W-tw && y >= r.Y && y <= r.Y+r.H {
			return true
		}
	}
	if horiz && st.MaxX+viewW > viewW+0.5 {
		if y >= r.Y+r.H-tw && x >= r.X && x <= r.X+r.W {
			return true
		}
	}
	return false
}

func scrollOverlayPaint(p *ui.Painter, r ui.Rect, id string, st *ui.ScrollState, horiz, vert bool) {
	a := scrollAlpha(id)
	if a <= 0.001 || st == nil {
		return
	}
	th := T.UITheme()
	style := th.ScrollbarStyle.Resolve(th)
	viewW, viewH := r.W, r.H
	contentW := st.MaxX + viewW
	contentH := st.MaxY + viewH
	if vert && contentH > viewH+0.5 {
		bar := ui.Scrollbar{Box: r, View: viewH, Content: contentH, Offset: st.Y, Style: style}
		thumb := style.Thumb
		thumb.A = uint8(math.Round(float64(thumb.A) * float64(a)))
		p.Scrollbar(bar, ui.ScrollbarState{Color: thumb})
	}
	if horiz && contentW > viewW+0.5 {
		bar := ui.Scrollbar{Box: r, Horizontal: true, View: viewW, Content: contentW, Offset: st.X, Style: style}
		thumb := style.Thumb
		thumb.A = uint8(math.Round(float64(thumb.A) * float64(a)))
		p.Scrollbar(bar, ui.ScrollbarState{Color: thumb})
	}
}
