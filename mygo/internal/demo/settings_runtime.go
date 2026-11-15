package demo

// DemoClaudeVersion 演示模式下的假 claude 版本。
func DemoClaudeVersion() string { return "2.1.318 (demo)" }

// DemoClaudeDoctor 演示 claude doctor 输出。
func DemoClaudeDoctor() []string {
	return []string{
		"安装方式: 演示模式(不连接本机 claude)",
		"版本: 2.1.318",
		"自动更新: 已关闭(演示)",
		"配置: 使用沙盒 ~/.minicode/demo",
	}
}

// DemoNetLatency 演示连通性检测延迟(毫秒)。
func DemoNetLatency(host string) (ok bool, ms int64) {
	switch host {
	case "api.anthropic.com":
		return true, 128
	case "api.deepseek.com":
		return true, 212
	case "github.com":
		return false, 0
	default:
		return true, 99
	}
}
