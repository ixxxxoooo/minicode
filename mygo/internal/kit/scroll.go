package kit

import (
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
	h, v := b.axes()
	return applyScrollPolicy(el, b.id, b.st, h, v)
}

// Children 构建滚动内容。
func (b *ScrollBuilder) Children(fn func()) {
	b.buildScroll().Children(fn)
}

// ApplyNativeScroll 给已有的 ui.Scroll / ScrollHorizontal / ScrollBoth 挂上滚动条策略与活动追踪(需保留 Bounds 等 API 时用)。
func ApplyNativeScroll(el ui.Element, id string, st *ui.ScrollState, horiz, vert bool) ui.Element {
	return applyScrollPolicy(el, id, st, horiz, vert)
}
