package app

// persistPrefs 从属新窗口进程不写用户偏好(只读)。
func (a *App) persistPrefs() bool { return !a.secondary && a.prefs != nil }
