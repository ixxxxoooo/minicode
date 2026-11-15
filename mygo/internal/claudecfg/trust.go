package claudecfg

import (
	"os"
	"path/filepath"
	"sort"
)

// TrustEntry 是某个项目的信任状态。
type TrustEntry struct {
	Path    string
	Trusted bool
	HookOK  bool // 演示 / 缺字段时与 Trusted 对齐
	McpOK   bool
}

// PathsInClaudeJSON 返回 ~/.claude.json 的 projects 下所有路径键(不读其它字段)。
func PathsInClaudeJSON() ([]string, error) {
	root, err := readJSON(homeClaudeJSON())
	if err != nil {
		return nil, err
	}
	proj, _ := root["projects"].(map[string]any)
	if len(proj) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(proj))
	for k := range proj {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out, nil
}

// MergeTrustPaths 合并 claude.json、项目注册表、侧栏项目与会话 cwd,去重后排序。
func MergeTrustPaths(claudePaths, registry, sidebar []string, cwd string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		if p == "" {
			return
		}
		k := projectKey(p)
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	for _, p := range claudePaths {
		add(p)
	}
	for _, p := range registry {
		add(p)
	}
	for _, p := range sidebar {
		add(p)
	}
	add(cwd)
	sort.Strings(out)
	return out
}

// ListTrust 列出 ~/.claude.json 里各项目的 hasTrustDialogAccepted。
func ListTrust(paths []string) ([]TrustEntry, error) {
	root, err := readJSON(homeClaudeJSON())
	if err != nil {
		return nil, err
	}
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
	return out, nil
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
