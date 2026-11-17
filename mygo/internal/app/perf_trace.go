package app

func modePerfName(m Mode) string {
	if m == ModeIDE {
		return "ide"
	}
	return "agent"
}

// screenPerfName 把 Screen 转成性能日志里的短名。
func screenPerfName(s Screen) string {
	switch s {
	case ScreenWork:
		return "work"
	case ScreenSettings:
		return "settings"
	case ScreenUsage:
		return "usage"
	case ScreenCustomize:
		return "customize"
	case ScreenAutomation:
		return "automation"
	case ScreenCanvas:
		return "canvas"
	case ScreenBrowser:
		return "browser"
	default:
		return string(s)
	}
}
