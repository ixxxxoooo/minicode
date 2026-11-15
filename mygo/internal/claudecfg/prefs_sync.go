package claudecfg

import (
	"os"
	"reflect"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// SessionPrefs 是要写入 Claude 用户设置 JSON 的会话相关项(不写密钥)。
type SessionPrefs struct {
	Layer                 Layer
	Project               string
	AlwaysThinking        *bool
	DefaultMode           string // permissions.defaultMode;空 = 不写入
	DisableBypass         *bool  // true → disableBypassPermissionsMode=disable
	AdditionalDirectories []string
}

// SyncSessionPrefs 把 MiniCode 偏好同步到对应层的 ~/.claude/settings.json 等。
func SyncSessionPrefs(p SessionPrefs) error {
	path := settingsPath(p.Layer, p.Project)
	root, err := readJSON(path)
	if err != nil {
		return err
	}
	before, _ := readJSON(path) // 深拷贝一份原样内容,用来判断这次同步有没有真的改动
	if p.AlwaysThinking != nil {
		root["alwaysThinkingEnabled"] = *p.AlwaysThinking
	}
	perm, _ := root["permissions"].(map[string]any)
	if perm == nil {
		perm = map[string]any{}
		root["permissions"] = perm
	}
	if p.DefaultMode != "" {
		perm["defaultMode"] = p.DefaultMode
	}
	if p.DisableBypass != nil {
		if *p.DisableBypass {
			perm["disableBypassPermissionsMode"] = "disable"
		} else {
			delete(perm, "disableBypassPermissionsMode")
		}
	}
	if p.AdditionalDirectories != nil {
		var a []any
		for _, d := range p.AdditionalDirectories {
			d = strings.TrimSpace(d)
			if d != "" {
				a = append(a, d)
			}
		}
		if len(a) > 0 {
			perm["additionalDirectories"] = a
		} else {
			delete(perm, "additionalDirectories")
		}
	}
	if _, err := os.Stat(path); err == nil && reflect.DeepEqual(before, root) {
		return nil // 内容没变:不重写文件、不换备份(否则每次开会话都在用户仓库里留下 .mini-backup)
	}
	log.Info("同步 Claude 设置", "layer", p.Layer, "alwaysThinking", p.AlwaysThinking != nil, "defaultMode", p.DefaultMode)
	return writeJSON(path, root)
}

// SettingsFileExists 该层的设置文件是否已存在(决定「没有显式设置过就不创建」)。
func SettingsFileExists(layer Layer, project string) bool {
	_, err := os.Stat(settingsPath(layer, project))
	return err == nil
}

// PermModeCLI 把界面分段索引映射为 permissions.defaultMode 字符串。
func PermModeCLI(ix int) string {
	switch ix {
	case 1:
		return core.ModeAcceptEdits.CLI()
	case 2:
		return core.ModePlan.CLI()
	default:
		return core.ModeDefault.CLI()
	}
}
