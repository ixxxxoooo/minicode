package app

import (
	"os/exec"
	"runtime"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var soundLog = diag.L("sound")

const (
	pNotifySound = "general.sound"
	pNotifyPause = "general.pauseNotify"
	pPauseUntil  = "general.pauseUntil"
	pTurnDing    = "chat.ding"
)

// notifySoundOn 系统通知是否带声音(general.sound:0 静音,1 轻提示)。
func (a *App) notifySoundOn() bool {
	if a == nil || a.prefs == nil {
		return true
	}
	return a.prefs.Int(pNotifySound, 1) != 0
}

// turnDingOn 回合结束是否播放提示音(chat.ding)。
func (a *App) turnDingOn() bool {
	if a == nil || a.prefs == nil {
		return false
	}
	return a.prefs.Bool(pTurnDing, false)
}

// playTurnDing 回合结束时播放短提示音(前台 audible,与系统通知无关)。
func (a *App) playTurnDing() {
	if !a.turnDingOn() || a.notifyPaused() {
		return
	}
	a.playSystemSound("Tink")
}

// playSystemSound 后台 afplay,不阻塞界面。
func (a *App) playSystemSound(name string) {
	if runtime.GOOS != "darwin" {
		return
	}
	path := "/System/Library/Sounds/" + name + ".aiff"
	a.bg("ui-sound", func() func() {
		end := diag.SpanSlow("sound", "播放", 0, "name", name)
		err := exec.Command("afplay", path).Run()
		end()
		return func() {
			if err != nil {
				soundLog.Warn("播放提示音失败", "name", name, "err", err)
			}
		}
	})
}

// notifyPaused 「暂停通知」(托盘菜单与 设置 → 通用):暂停后不发系统通知、不播提示音,应用内通知中心照常记录。
// 暂停 = 开关 general.pauseNotify;限时暂停另存到期时间 general.pauseUntil(Unix 秒),到期自动恢复并清掉两项。
func (a *App) notifyPaused() bool {
	if a == nil || a.prefs == nil {
		return false
	}
	until := a.prefs.Int(pPauseUntil, 0)
	if !a.prefs.Bool(pNotifyPause, false) {
		if until != 0 {
			a.prefs.Reset(pPauseUntil) // 开关被手动关掉:丢掉残留的到期时间,免得之后「一直暂停」被它提前结束
		}
		return false
	}
	if until > 0 && a.now().Unix() >= int64(until) {
		a.prefs.Reset(pPauseUntil)
		a.prefs.Reset(pNotifyPause)
		soundLog.Info("限时暂停通知已到期,自动恢复")
		return false
	}
	return true
}

// pauseNotifyFor 暂停通知:d > 0 限时(到点自动恢复),d == 0 一直暂停,d < 0 恢复。
func (a *App) pauseNotifyFor(d time.Duration) {
	if a.prefs == nil {
		return
	}
	switch {
	case d < 0:
		a.prefs.Reset(pNotifyPause)
		a.prefs.Reset(pPauseUntil)
	case d == 0:
		a.prefs.Set(pNotifyPause, true)
		a.prefs.Reset(pPauseUntil)
	default:
		a.prefs.Set(pNotifyPause, true)
		a.prefs.Set(pPauseUntil, int(a.now().Add(d).Unix()))
	}
	soundLog.Info("暂停通知", "时长", d.String())
	a.schedulePauseExpiry()
	a.syncBrand()
}

// schedulePauseExpiry 限时暂停到点后回主线程刷新托盘(到期状态由 notifyPaused 懒清理)。
func (a *App) schedulePauseExpiry() {
	if a.brand.pauseTimer != nil {
		a.brand.pauseTimer.Stop()
		a.brand.pauseTimer = nil
	}
	if a.prefs == nil || a.win == nil || !a.prefs.Bool(pNotifyPause, false) {
		return
	}
	until := a.prefs.Int(pPauseUntil, 0)
	if until <= 0 {
		return
	}
	wait := time.Until(time.Unix(int64(until), 0)) + time.Second
	if wait < time.Second {
		wait = time.Second
	}
	a.brand.pauseTimer = time.AfterFunc(wait, func() {
		a.win.Update(func() {
			a.notifyPaused()
			a.syncBrand()
		})
	})
}

// pauseLabel 托盘菜单里「暂停通知」一项的标题:未暂停 / 暂停到几点 / 一直暂停。
func (a *App) pauseLabel() string {
	if !a.notifyPaused() {
		return "暂停通知"
	}
	if until := a.prefs.Int(pPauseUntil, 0); until > 0 {
		return "通知已暂停 · 到 " + time.Unix(int64(until), 0).Format("15:04")
	}
	return "通知已暂停"
}
