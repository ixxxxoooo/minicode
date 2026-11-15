package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/ixxxxoooo/minicode/internal/kit"
)

// panelFlash 底部面板展开 / 新建终端时边框闪一下。
func (a *App) panelFlash(c *ui.Context, box ui.Element) {
	el := time.Since(a.panelFlashAt)
	on := !a.panelFlashAt.IsZero() && el < 280*time.Millisecond
	live := !a.panelFlashAt.IsZero() && el < 900*time.Millisecond
	if live {
		c.Invalidate()
	}
	if !live {
		return
	}
	lay := ui.Box(c).Key("pflash").Absolute().Left(0).Right(0).Top(0).Bottom(0).PassThrough().
		Border(2, a.tk.Ac)
	target := float32(0)
	if on {
		target = 1
	}
	dur := kit.DurSlow
	if !on {
		dur = 500 * time.Millisecond
	}
	lay.Opacity(lay.AnimateWith("pf", target, dur, kit.EaseOut) * 0.85)
	_ = box
}
