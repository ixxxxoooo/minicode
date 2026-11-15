//go:build !darwin || nocgo

package app

// postMacNotification 非 macOS 或禁用 cgo 时回退 osascript(无点击回调)。
func postMacNotification(title, body, sessID string) {
	if macNotifyApp != nil {
		macNotifyApp.notifyOSAScript(title, body)
	}
	_ = sessID
}
