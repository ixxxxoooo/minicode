package customize

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// SaveMCP 把 MCP 草稿写入对应层的 claude.json / .mcp.json(原子写 + 备份)。
// env 非 nil 时写入 mcpServers 条目的 env 字段;否则沿用 draft.EnvVals。
func (ds *DiskStore) SaveMCP(it *Item, draft Item, env map[string]string) error {
	name := strings.TrimSpace(draft.Name)
	if name == "" {
		return fmt.Errorf("名称不能为空")
	}
	if env == nil {
		env = draft.EnvVals
	}
	switch draft.Scope {
	case User:
		return ds.saveMCPUser(name, draft, env)
	case Project, Local:
		if ds.proj == "" {
			return fmt.Errorf("需要打开项目才能写入项目/本地层 MCP")
		}
		file := ".mcp.json"
		if draft.Scope == Local {
			file = ".claude/settings.local.json"
		}
		return ds.saveMCPProject(name, draft, file, env)
	default:
		return ds.saveMCPUser(name, draft, env)
	}
}

func (ds *DiskStore) saveMCPUser(name string, draft Item, env map[string]string) error {
	var root map[string]any
	if !readJSON(ds.claudeJSON, &root) {
		root = map[string]any{}
	}
	mcp, _ := root["mcpServers"].(map[string]any)
	if mcp == nil {
		mcp = map[string]any{}
		root["mcpServers"] = mcp
	}
	mcp[name] = mcpEntryOf(draft, env)
	b, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(ds.claudeJSON, b, 0o600)
}

func (ds *DiskStore) saveMCPProject(name string, draft Item, rel string, env map[string]string) error {
	if draft.Scope == Local {
		rel = filepath.Join(".claude", "settings.local.json")
	}
	path := filepath.Join(ds.proj, rel)
	var root map[string]any
	if !readJSON(path, &root) {
		root = map[string]any{}
	}
	mcp, _ := root["mcpServers"].(map[string]any)
	if mcp == nil {
		mcp = map[string]any{}
		root["mcpServers"] = mcp
	}
	mcp[name] = mcpEntryOf(draft, env)
	b, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, b, 0o600)
}

func mcpEntryOf(d Item, env map[string]string) map[string]any {
	ent := map[string]any{}
	switch d.Transport {
	case "http", "sse":
		ent["url"] = strings.TrimSpace(d.Cmd)
	default:
		parts := strings.Fields(strings.TrimSpace(d.Cmd))
		if len(parts) > 0 {
			ent["command"] = parts[0]
			if len(parts) > 1 {
				var args []any
				for _, x := range parts[1:] {
					args = append(args, x)
				}
				ent["args"] = args
			}
		}
	}
	if len(env) > 0 {
		em := map[string]any{}
		for k, v := range env {
			em[k] = v
		}
		ent["env"] = em
	}
	return ent
}
