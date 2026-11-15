package app

import (
	"os/exec"
	"strings"
)

// macNotifyApp 由 maybeSystemNotify 在发送前设置,供平台实现读取。
var macNotifyApp *App

func (a *App) notifyOSAScript(title, body string) {
	script := `display notification "` + escOSA(body) + `" with title "` + escOSA(title) + `" subtitle "MiniCode"`
	a.bg("osascript-notify", func() func() {
		err := exec.Command("osascript", "-e", script).Run()
		return func() {
			if err != nil {
				log.Warn("系统通知失败", "err", err)
			}
		}
	})
}

func (a *App) notifyMacNative(title, body, sessID string) {
	macNotifyApp = a
	postMacNotification(title, body, sessID)
}

// OpenSessionFromNotify 供未来原生通知点击回调跳到会话。
func (a *App) OpenSessionFromNotify(sessID string) {
	sessID = strings.TrimSpace(sessID)
	if sessID == "" {
		return
	}
	log.Info("通知点击打开会话", "sess", sessID)
	a.OpenSession(sessID)
}
