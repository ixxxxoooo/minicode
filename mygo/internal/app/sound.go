package app

import (
	"os/exec"
	"runtime"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

var soundLog = diag.L("sound")

const (
	pNotifySound = "general.sound"
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
	if !a.turnDingOn() {
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
