package app

import (
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/prefs"
)

const pWaitAnimRev = "appearance.waitAnimRev"

var waitAnimLog = diag.L("app.prefs")

// migrateWaitAnim 旧 waitAnim:0 圆环 / 1 点阵 / 2 脉冲 → 新:0 沙漏 / 1 圆环 / 2 点阵 / 3 脉冲。
func migrateWaitAnim(p *prefs.Store) {
	if p == nil || p.Int(pWaitAnimRev, 0) >= 1 {
		return
	}
	if !p.Has(pWaitAnim) {
		p.Set(pWaitAnimRev, 1)
		return
	}
	old := p.Int(pWaitAnim, 0)
	n := old + 1
	if old < 0 || old > 2 {
		n = 0
	}
	p.Set(pWaitAnim, n)
	waitAnimLog.Info("迁移等待动画偏好", "old", old, "new", n)
	p.Set(pWaitAnimRev, 1)
}
