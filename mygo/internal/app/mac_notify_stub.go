//go:build !darwin || nocgo

package app

func initMacNotifyBundle() {}

// postMacNotification 非 macOS 或禁用 cgo 时回退 osascript(无点击回调)。
func postMacNotification(title, body, sessID string, sound bool) {
	if macNotifyApp != nil {
		if sound {
			macNotifyApp.notifyOSAScript(title, body)
		} else {
			macNotifyApp.notifyOSAScriptSilent(title, body)
		}
	}
	_ = sessID
}
