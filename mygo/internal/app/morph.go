package app

import (
	"time"

	"github.com/egoist/mygo/ui"
)

const morphLockDur = 470 * time.Millisecond // 对齐设计稿 .morphing

// beginMorph 形态切换开始:锁定指针直至过渡结束。
func (a *App) beginMorph() {
	d := morphLockDur
	if morph.Duration > 0 && morph.Duration < morphLockDur {
		d = morph.Duration
	}
	if morph.Duration == 0 {
		d = time.Millisecond
	}
	a.morphUntil = time.Now().Add(d)
	if a.win != nil {
		a.win.Invalidate()
	}
}

func (a *App) morphing() bool {
	// 测试 / 截图(没有真实窗口)直达终态,不挡点击。
	if a.win == nil {
		return false
	}
	if a.morphUntil.IsZero() {
		return false
	}
	if time.Now().Before(a.morphUntil) {
		return true
	}
	a.morphUntil = time.Time{}
	return false
}

// morphShield 在根上盖住交互,防止过渡中误点(对标设计稿 pointer-events 锁)。
func (a *App) morphShield(c *ui.Context) {
	if !a.morphing() {
		return
	}
	ui.Box(c).Absolute().Left(0).Right(0).Top(0).Bottom(0).
		Background(ui.Transparent).
		OnClick(func() {}).
		Children(func() {})
	if a.win != nil {
		a.frameSoon()
	}
}
