package kit

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// ScrollBuilder 封装 ui.Scroll / ScrollHorizontal / ScrollBoth,统一滚动条策略与活动追踪。
type ScrollBuilder struct {
	c                                                         *ui.Context
	id                                                        string
	st                                                        *ui.ScrollState
	horiz                                                     bool
	vert                                                      bool
	both                                                      bool
	grow                                                      float32
	minH                                                      float32
	maxH                                                      float32
	minW, width, height                                       float32
	padX                                                      float32
	growOn, minHOn, maxHOn, minWOn, widthOn, heightOn, padXOn bool
	clip, selectable                                          bool
	align                                                     ui.Align
	alignOn                                                   bool
	shortcuts                                                 []scrollShortcut
}

type scrollShortcut struct {
	mods ui.Modifiers
	key  ui.Key
	fn   func()
}

// VScroll 创建纵向滚动容器(替代 ui.Scroll)。
func VScroll(c *ui.Context) *ScrollBuilder {
	return &ScrollBuilder{c: c, vert: true}
}

// HScroll 创建横向滚动容器(替代 ui.ScrollHorizontal)。
func HScroll(c *ui.Context) *ScrollBuilder {
	return &ScrollBuilder{c: c, horiz: true}
}

// XYScroll 创建双向滚动容器(长行代码块等);Shift+滚轮横向滚由 MyGo 框架转换。
func XYScroll(c *ui.Context) *ScrollBuilder {
	return &ScrollBuilder{c: c, horiz: true, vert: true, both: true}
}

func (b *ScrollBuilder) axes() (horiz, vert bool) {
	if b.both {
		return true, true
	}
	if b.horiz {
		return true, false
	}
	return false, true
}

func (b *ScrollBuilder) Key(id string) *ScrollBuilder {
	b.id = id
	return b
}

func (b *ScrollBuilder) TrackScroll(st *ui.ScrollState) *ScrollBuilder {
	b.st = st
	return b
}

func (b *ScrollBuilder) Grow(v float32) *ScrollBuilder {
	b.grow, b.growOn = v, true
	return b
}

func (b *ScrollBuilder) Shrink(v float32) *ScrollBuilder { b.grow, b.growOn = v, true; return b }
func (b *ScrollBuilder) MinHeight(v float32) *ScrollBuilder {
	b.minH, b.minHOn = v, true
	return b
}
func (b *ScrollBuilder) MaxHeight(v float32) *ScrollBuilder {
	b.maxH, b.maxHOn = v, true
	return b
}
func (b *ScrollBuilder) MinWidth(v float32) *ScrollBuilder {
	b.minW, b.minWOn = v, true
	return b
}
func (b *ScrollBuilder) Width(v float32) *ScrollBuilder {
	b.width, b.widthOn = v, true
	return b
}
func (b *ScrollBuilder) PaddingX(v float32) *ScrollBuilder {
	b.padX, b.padXOn = v, true
	return b
}
func (b *ScrollBuilder) Height(v float32) *ScrollBuilder {
	b.height, b.heightOn = v, true
	return b
}
func (b *ScrollBuilder) Clip() *ScrollBuilder  { b.clip = true; return b }
func (b *ScrollBuilder) ClipX() *ScrollBuilder { b.clip = true; return b }
func (b *ScrollBuilder) Selectable() *ScrollBuilder {
	b.selectable = true
	return b
}
func (b *ScrollBuilder) AlignItems(a ui.Align) *ScrollBuilder {
	b.align, b.alignOn = a, true
	return b
}

func (b *ScrollBuilder) OnShortcut(mods ui.Modifiers, key ui.Key, fn func()) *ScrollBuilder {
	b.shortcuts = append(b.shortcuts, scrollShortcut{mods: mods, key: key, fn: fn})
	return b
}

func (b *ScrollBuilder) buildScroll() ui.Element {
	var el ui.Element
	switch {
	case b.both:
		el = ui.ScrollBoth(b.c)
	case b.horiz:
		el = ui.ScrollHorizontal(b.c)
	default:
		el = ui.Scroll(b.c)
	}
	if b.id != "" {
		el = el.Key(b.id)
	}
	if b.st != nil {
		el = el.TrackScroll(b.st)
	}
	if b.growOn {
		el = el.Grow(b.grow)
	}
	if b.minHOn {
		el = el.MinHeight(b.minH)
	}
	if b.maxHOn {
		el = el.MaxHeight(b.maxH)
	}
	if b.minWOn {
		el = el.MinWidth(b.minW)
	}
	if b.widthOn {
		el = el.Width(b.width)
	}
	if b.heightOn {
		el = el.Height(b.height)
	}
	if b.padXOn {
		el = el.PaddingX(b.padX)
	}
	if b.clip {
		el = el.Clip()
	}
	if b.selectable {
		el = el.Selectable()
	}
	if b.alignOn {
		el = el.AlignItems(b.align)
	}
	for _, sc := range b.shortcuts {
		el = el.OnShortcut(sc.mods, sc.key, sc.fn)
	}
	el = el.HandleInput(func(ev ui.InputEvent) bool {
		noteScrollWheel(b.id, b.st, ev)
		return false
	})
	h, v := b.axes()
	regScroll(b.id, b.st, h, v)
	switch ScrollbarMode() {
	case ScrollbarAlways:
		el = el.KeepScrollbars(true)
	case ScrollbarAutoHide:
		el = el.Scrollbars(false)
	}
	return el
}

func noteScrollWheel(id string, st *ui.ScrollState, ev ui.InputEvent) {
	if ev.Kind == ui.InputScroll {
		markScrollActivity(id)
	}
}

func (b *ScrollBuilder) overlay(el ui.Element, h, v bool) {
	ui.Box(b.c).Key(b.id + "-sov").Absolute().Left(0).Right(0).Top(0).Bottom(0).PassThrough().
		HandleInput(func(ev ui.InputEvent) bool {
			r := el.Bounds()
			switch ev.Kind {
			case ui.InputPointerMove:
				setScrollBarHover(b.id, scrollBarEdgeHit(r, b.st, ev.X, ev.Y, h, v))
			case ui.InputScroll:
				markScrollActivity(b.id)
			case ui.InputPointerDown:
				if scrollBarEdgeHit(r, b.st, ev.X, ev.Y, h, v) {
					markScrollActivity(b.id)
				}
			}
			return false
		}).
		Draw(func(p *ui.Painter, r ui.Rect) {
			br := el.Bounds()
			if br.W <= 0 || br.H <= 0 {
				return
			}
			scrollOverlayPaint(p, br, b.id, b.st, h, v)
		})
}

// Children 构建滚动内容;自动隐藏模式下叠加自绘 overlay 滚动条(停止滚动约 3 秒后淡出)。
func (b *ScrollBuilder) Children(fn func()) {
	el := b.buildScroll()
	h, v := b.axes()
	if ScrollbarMode() != ScrollbarAutoHide {
		el.Children(fn)
		return
	}
	col := ui.Column(b.c).Key(b.id + "-swrap")
	if b.growOn {
		col = col.Grow(b.grow)
	}
	if b.minHOn {
		col = col.MinHeight(b.minH)
	}
	if b.minWOn {
		col = col.MinWidth(b.minW)
	}
	col.Children(func() {
		el.Children(fn)
		b.overlay(el, h, v)
	})
}

// ApplyNativeScroll 给已有的 ui.Scroll / ScrollHorizontal / ScrollBoth 挂上策略与活动追踪(需保留 Bounds 等 API 时用)。
func ApplyNativeScroll(el ui.Element, id string, st *ui.ScrollState, horiz, vert bool) ui.Element {
	regScroll(id, st, horiz, vert)
	el = el.HandleInput(func(ev ui.InputEvent) bool {
		noteScrollWheel(id, st, ev)
		return false
	})
	switch ScrollbarMode() {
	case ScrollbarAlways:
		return el.KeepScrollbars(true)
	case ScrollbarAutoHide:
		return el.Scrollbars(false)
	default:
		return el
	}
}

// ScrollFrameActive 报告是否仍有滚动条淡出动效在进行(应用据此 frameSoon)。
func ScrollFrameActive(now time.Time) bool {
	if ScrollbarMode() != ScrollbarAutoHide || !MotionOn() {
		return false
	}
	scrollMu.Lock()
	defer scrollMu.Unlock()
	for _, sl := range scrollSlots {
		if sl == nil || sl.lastMove.IsZero() {
			continue
		}
		if sl.dragging || sl.barHover {
			return true
		}
		if now.Sub(sl.lastMove) < scrollFadeOut+DurFast {
			return true
		}
	}
	return false
}
