package app

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
	"github.com/ixxxxoooo/minicode/internal/theme"
)

const thinkPreviewRunes = 200

type thinkMqSt struct {
	X, Show float32
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

func (a *App) thinkMarqueeRow(c *ui.Context, sp step, tone ui.Color, open bool) ui.Element {
	if !a.thinkMarqueeVisible(sp, open) {
		return ui.Box(c).Width(0).Shrink(0)
	}
	t := a.tk
	preview := thinkPreviewPlain(sp.thinkText)
	if preview == "" {
		return ui.Box(c).Width(0).Shrink(0)
	}
	box := ui.Box(c).Height(20).Grow(1).MinWidth(48).MaxWidth(220).Shrink(1).Clip()
	if !kit.MotionOn() || kit.SystemReduceMotion() || (a.win != nil && !a.win.IsFocused()) {
		box.Children(func() {
			ui.Text(c, preview).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().NoWrap()
		})
		return box
	}
	id := sp.id
	st := a.thinkMq[id]
	target := a.thinkMqTarget(preview, 36)
	if target > st.X {
		st.X = target
	}
	// 缓动追赶,避免逐字跳
	if diff := target - st.Show; diff > 0.5 {
		st.Show += diff * .22
		a.frameSoon()
	} else {
		st.Show = target
	}
	a.thinkMq[id] = st
	if st.Show > 0 {
		box.Translate(-st.Show, 0)
	}
	box.Children(func() {
		ui.Text(c, preview).FontSize(theme.FsCap).TextColor(t.Tx3).SingleLine().NoWrap().Shrink(0)
		bg := t.BG.Over(t.Fill1)
		ui.Box(c).Width(28).Height(20).Absolute().HitTest(func(_, _ float32, _ ui.Rect) bool { return false }).
			MaskLinearGradient(ui.LinearGradient{
				Angle: 90,
				Stops: []ui.GradientStop{
					{Color: bg, At: 0},
					{Color: ui.Transparent, At: 1},
				},
			})
	})
	_ = box.Loop("think-mq", 33*time.Millisecond, ui.Linear)
	return box
}

// thinkMqTarget 按字宽估算尾部可见偏移(约 6px/字)。
func (a *App) thinkMqTarget(preview string, maxChars int) float32 {
	n := utf8.RuneCountInString(preview)
	if n <= maxChars {
		return 0
	}
	return float32(n-maxChars) * 6
}

func (a *App) noteThinkStreamEnd(id string) {
	if a.thinkWasLive[id] {
		a.thinkMqHide[id] = a.now()
	}
	delete(a.thinkWasLive, id)
}
