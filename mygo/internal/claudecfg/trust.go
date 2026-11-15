package claudecfg

import (
	"os"
	"path/filepath"
)

// TrustEntry 是某个项目的信任状态。
type TrustEntry struct {
	Path    string
	Trusted bool
	HookOK  bool // 演示 / 缺字段时与 Trusted 对齐
	McpOK   bool
}

// ListTrust 列出 ~/.claude.json 里各项目的 hasTrustDialogAccepted。
func ListTrust(paths []string) []TrustEntry {
	root, _ := readJSON(homeClaudeJSON())
	proj, _ := root["projects"].(map[string]any)
	out := make([]TrustEntry, 0, len(paths))
	for _, p := range paths {
		key := projectKey(p)
		e := TrustEntry{Path: p, HookOK: false, McpOK: false}
		if proj != nil {
			if pe, ok := proj[key].(map[string]any); ok {
				if v, ok := pe["hasTrustDialogAccepted"].(bool); ok {
					e.Trusted, e.HookOK, e.McpOK = v, v, v
				}
				if v, ok := pe["hooksTrusted"].(bool); ok {
					e.HookOK = v
				}
				if v, ok := pe["mcpTrusted"].(bool); ok {
					e.McpOK = v
				}
			}
		}
		out = append(out, e)
	}
	return out
}

// SetTrust 写 hasTrustDialogAccepted(及其它信任开关)到 ~/.claude.json。
func SetTrust(project string, trusted, hook, mcp bool) error {
	path := homeClaudeJSON()
	root, err := readJSON(path)
	if err != nil {
		return err
	}
	proj, _ := root["projects"].(map[string]any)
	if proj == nil {
		proj = map[string]any{}
		root["projects"] = proj
	}
	key := projectKey(project)
	pe, _ := proj[key].(map[string]any)
	if pe == nil {
		pe = map[string]any{}
	}
	pe["hasTrustDialogAccepted"] = trusted
	pe["hooksTrusted"] = hook
	pe["mcpTrusted"] = mcp
	proj[key] = pe
	log.Info("更新项目信任", "path", key, "trusted", trusted, "hook", hook, "mcp", mcp)
	return writeJSON(path, root)
}

// DemoTrust 演示模式用的固定数据。
func DemoTrust() []TrustEntry {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, "Projects")
	return []TrustEntry{
		{Path: filepath.Join(base, "router-demo"), Trusted: true, HookOK: true, McpOK: true},
		{Path: filepath.Join(base, "web-dashboard"), Trusted: true, HookOK: true, McpOK: false},
		{Path: filepath.Join(base, "docs-site"), Trusted: false, HookOK: false, McpOK: false},
	}
}
