package app

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const (
	thinkPreviewRunes = 200
	thinkMqMinW       = 96
)

type thinkMqSt struct {
	X, Show float32
	TW      float32 // 上一帧量到的预览文本宽度
}

// thinkPreviewPlain 取思考正文尾部,换行压成「 · 」。
func thinkPreviewPlain(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n", " · ")
	text = strings.ReplaceAll(text, " ·  · ", " · ")
	if utf8.RuneCountInString(text) <= thinkPreviewRunes {
		return text
	}
	rs := []rune(text)
	return string(rs[len(rs)-thinkPreviewRunes:])
}

func (a *App) thinkMarqueeVisible(sp step, open bool) bool {
	if !sp.thinkLive || open || !sp.hasBody || strings.TrimSpace(sp.thinkText) == "" {
		return false
	}
	if a.thinkMq == nil {
		a.thinkMq = map[string]thinkMqSt{}
	}
	if a.chatTier() == tierNarrow {
		return false
	}
	if at, hide := a.thinkMqHide[sp.id]; hide && a.now().Sub(at) > 320*time.Millisecond {
		return false
	}
	return true
}

func (a *App) thinkMarqueeRow(c *ui.Context, sp step, tone ui.Color, open bool, row ui.Element) ui.Element {
	if !a.thinkMarqueeVisible(sp, open) {
		return ui.Box(c).Width(0).Shrink(0)
	}
	t := a.tk
	preview := thinkPreviewPlain(sp.thinkText)
	if preview == "" {
		return ui.Box(c).Width(0).Shrink(0)
	}
	rowW := float32(0)
	if b := row.Bounds(); b.W > 0 {
		rowW = b.W
	}
	maxW := rowW * 0.52
	if maxW > 0 && maxW < thinkMqMinW {
		return ui.Box(c).Width(0).Shrink(0)
	}
	box := ui.Box(c).Height(20).Grow(1).MinWidth(0).Shrink(1).Clip().MarginStart(4)
	if maxW > 0 {
		box.MaxWidth(maxW)
	} else {
		box.MaxWidth(rowW * 0.52)
	}
	id := sp.id
	st := a.thinkMq[id]
	boxW := float32(0)
	if b := box.Bounds(); b.W > 8 {
		boxW = b.W
	} else if rowW > 0 {
		boxW = maxW
	}
	if boxW > 0 && boxW < thinkMqMinW {
		return ui.Box(c).Width(0).Shrink(0)
	}
	textW := st.TW
	if textW <= 0 {
		textW = thinkMqEstimateWidth(preview)
	}
	target := textW - boxW + 4
	if target < 0 {
		target = 0
	}
	still := !kit.MotionOn() || kit.SystemReduceMotion() || (a.win != nil && !a.win.IsFocused())
	if still {
		st.Show = target
	} else if diff := target - st.Show; diff > 0.5 || diff < -0.5 {
		st.Show += diff * .22
		a.frameSoon()
	} else {
		st.Show = target
	}
	box.Children(func() {
		inner := ui.Row(c).Height(20).Grow(1).MinWidth(0).Justify(ui.End).AlignItems(ui.Center)
		if st.Show > 0 {
			inner.Translate(-st.Show, 0)
		}
		inner.Children(func() {
			tx := ui.Text(c, preview).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().NoWrap().Shrink(0).PaddingStart(8)
			if w := tx.Bounds().W; w > 0 {
				st.TW = w
			}
		})
		if target > 0 {
			bg := t.BG.Over(t.Fill1)
			if hov := row.Hovered(); hov {
				bg = t.Fill1
			}
			ui.Box(c).Width(32).Height(20).Absolute().HitTest(func(_, _ float32, _ ui.Rect) bool { return false }).
				MaskLinearGradient(ui.LinearGradient{
					Angle: 90,
					Stops: []ui.GradientStop{
						{Color: bg, At: 0},
						{Color: ui.Transparent, At: 1},
					},
				})
		}
	})
	a.thinkMq[id] = st
	if !still {
		_ = box.Loop("think-mq", 33*time.Millisecond, ui.Linear)
	}
	return box
}

// thinkMqEstimateWidth 估算一段文字单行宽度:中日韩全角约 1 em,其余约 0.55 em。
func thinkMqEstimateWidth(s string) float32 {
	var w float32
	em := float32(theme.FsCap)
	for _, r := range s {
		if r >= 0x2E80 {
			w += em
		} else {
			w += em * 0.55
		}
	}
	return w
}

func (a *App) noteThinkStreamEnd(id string) {
	if a.thinkWasLive[id] {
		a.thinkMqHide[id] = a.now()
	}
	delete(a.thinkWasLive, id)
}
