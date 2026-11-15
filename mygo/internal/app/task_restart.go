package app

// taskRestartDisabled 真实 claude 后端尚不支持重启后台任务。
func (a *App) taskRestartDisabled() (disabled bool, reason string) {
	if a.be != nil && a.be.Name() == "claude" {
		return true, "真实模式暂不支持重启任务;请用停止后重新发起"
	}
	return false, ""
}
