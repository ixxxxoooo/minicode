package claudecfg

import (
	"path/filepath"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// EnvRow 是环境变量一行(值不落日志)。
type EnvRow struct {
	Name  string
	Scope string // 用户 / 项目 / 本地
	Mask  string // 界面显示的遮盖文本
}

// ReadEnvRows 合并用户层 settings 的 env(变量名 + 遮盖值)。
func ReadEnvRows(project string) ([]EnvRow, error) {
	root, err := readJSON(userSettings())
	if err != nil {
		return nil, err
	}
	env, _ := root["env"].(map[string]any)
	var out []EnvRow
	for k := range env {
		out = append(out, EnvRow{Name: k, Scope: "用户", Mask: "••••••••"})
	}
	if project != "" {
		pr, err := readJSON(filepath.Join(project, ".claude", "settings.json"))
		if err == nil {
			if pe, _ := pr["env"].(map[string]any); pe != nil {
				for k := range pe {
					out = append(out, EnvRow{Name: k, Scope: "项目", Mask: "••••••••"})
				}
			}
		}
		loc, err := readJSON(filepath.Join(project, ".claude", "settings.local.json"))
		if err == nil {
			if le, _ := loc["env"].(map[string]any); le != nil {
				for k := range le {
					out = append(out, EnvRow{Name: k, Scope: "本地", Mask: "••••••••"})
				}
			}
		}
	}
	return out, nil
}

// SetEnv 在用户层 settings 写入 env(值不写日志)。
func SetEnv(name, value string, layer Layer, project string) error {
	path := settingsPath(layer, project)
	root, err := readJSON(path)
	if err != nil {
		return err
	}
	env, _ := root["env"].(map[string]any)
	if env == nil {
		env = map[string]any{}
		root["env"] = env
	}
	env[name] = value
	log.Info("写入环境变量", "name", name, "layer", layer)
	return writeJSON(path, root)
}

// RemoveEnv 删除 env 项。
func RemoveEnv(name string, layer Layer, project string) error {
	path := settingsPath(layer, project)
	root, err := readJSON(path)
	if err != nil {
		return err
	}
	env, _ := root["env"].(map[string]any)
	if env == nil {
		return nil
	}
	delete(env, name)
	log.Info("删除环境变量", "name", name, "layer", layer)
	return writeJSON(path, root)
}

// DemoEnv 演示列表。
func DemoEnv() []EnvRow {
	return []EnvRow{
		{Name: "ANTHROPIC_API_KEY", Scope: "用户", Mask: "••••••••"},
		{Name: "HTTPS_PROXY", Scope: "用户", Mask: "••••••••"},
		{Name: "ENABLE_TOOL_SEARCH", Scope: "项目", Mask: "••••••••"},
		{Name: "API_TIMEOUT_MS", Scope: "本地", Mask: "••••••••"},
	}
}

// MiniHome 返回 ~/.minicode(密钥文件)。
func MiniHome() string { return diag.Home() }
