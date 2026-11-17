//go:build !darwin || !cgo

package app

func initMacNotifyBundle() {}

// postMacNotification 非 macOS 或禁用 cgo 时回退 osascript(无点击回调、没有动作按钮)。
func postMacNotification(n sysNote, sound bool) {
	if macNotifyApp != nil {
		if sound {
			macNotifyApp.notifyOSAScript(n.title, n.body)
		} else {
			macNotifyApp.notifyOSAScriptSilent(n.title, n.body)
		}
	}
}

func clearMacNotification(permID string) {}
