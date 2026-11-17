package claudecfg

import (
	"path/filepath"
	"strings"
)

// MergedClaudeEnv 合并 Claude 设置文件里的 env(用户层,以及当前项目层若有 project)。
// 只读、不写日志;供判断官方通道等逻辑使用。
func MergedClaudeEnv(project string) map[string]string {
	out := envFromSettings(userSettings())
	if project != "" {
		for k, v := range envFromSettings(filepath.Join(project, ".claude", "settings.json")) {
			out[k] = v
		}
		for k, v := range envFromSettings(filepath.Join(project, ".claude", "settings.local.json")) {
			out[k] = v
		}
	}
	return out
}

func envFromSettings(path string) map[string]string {
	root, err := readJSON(path)
	if err != nil {
		return nil
	}
	raw, _ := root["env"].(map[string]any)
	if raw == nil {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

// OfficialAnthropicBaseURL 判断 Base URL 是否视为官方 Anthropic 端点(空 = 默认)。
func OfficialAnthropicBaseURL(u string) bool {
	u = strings.TrimSpace(strings.ToLower(u))
	u = strings.TrimRight(u, "/")
	return u == "" || u == "https://api.anthropic.com"
}
