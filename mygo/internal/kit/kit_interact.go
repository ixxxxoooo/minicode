package kit

import (
	"time"

	"github.com/egoist/mygo/ui"
)

const loadingDelay = 400 * time.Millisecond

// clickPressScale 为可点击控件施加按下缩放到 96%(设计稿 interact.css);分段 / 开关整体不缩放。
func clickPressScale(b ui.Element) {
	if b.Pressed() {
		if MotionOn() {
			sc := b.AnimateTo("press", .96, ui.Animation{Duration: DurInstant, Ease: EaseSpring})
			b.Scale(sc, sc)
			return
		}
		b.Scale(.96, .96)
		return
	}
	if MotionOn() {
		sc := b.AnimateTo("press", 1, ui.Animation{Duration: DurInstant, Ease: EaseSpring})
		b.Scale(sc, sc)
	}
}

// clickFocusRing 强化键盘聚焦环(库已区分 FocusVisible:仅 Tab 聚焦显示)。
func clickFocusRing(b ui.Element) {
	b.FocusRing(true).FocusRingOffset(2).FocusRingColor(T.Ac)
	if b.FocusVisible() {
		b.Border(1.5, T.Ac.Mix(T.Raised, .35))
	}
}

// btnLoadingSpin 加载超过 400ms 才显示转圈,此前保持原样且不可点。
func btnLoadingSpin(b ui.Element, col ui.Color) bool {
	if !MotionOn() {
		return true
	}
	p := b.AnimateTo("loadDly", 1, ui.Animation{Duration: loadingDelay, Ease: ui.Linear})
	return p >= 1
}
