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
	old, _ := mcp[name].(map[string]any)
	mcp[name] = mcpEntryOf(old, draft, env)
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
	old, _ := mcp[name].(map[string]any)
	mcp[name] = mcpEntryOf(old, draft, env)
	b, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, b, 0o600)
}

// mcpEntryOf 由草稿生成 mcpServers 里这一项。以磁盘上原有的条目为底(old,可为 nil),
// 只改界面能编辑的字段(type / url / command / args / env / headers),其余字段原样保留——
// 此前是整项重建,保存一次 http 服务器就会丢掉 headers(Cookie / Authorization)和 type,连接从此更不通。
func mcpEntryOf(old map[string]any, d Item, env map[string]string) map[string]any {
	ent := map[string]any{}
	for k, v := range old {
		ent[k] = v
	}
	switch d.Transport {
	case "http", "sse":
		ent["type"] = d.Transport
		ent["url"] = strings.TrimSpace(d.Cmd)
		delete(ent, "command")
		delete(ent, "args")
	default:
		delete(ent, "url")
		delete(ent, "headers")
		parts := strings.Fields(strings.TrimSpace(d.Cmd))
		delete(ent, "command")
		delete(ent, "args")
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
	if d.Transport == "http" || d.Transport == "sse" {
		if len(d.Headers) > 0 {
			hm := map[string]any{}
			for k, v := range d.Headers {
				hm[k] = v
			}
			ent["headers"] = hm
		} else {
			delete(ent, "headers")
		}
	}
	if len(env) > 0 {
		em := map[string]any{}
		for k, v := range env {
			em[k] = v
		}
		ent["env"] = em
	} else {
		delete(ent, "env")
	}
	return ent
}

// ParseHeaderLines 解析「名称: 值」每行一条的请求头文本(值里可以有冒号和等号,如 Cookie);空行与 # 注释跳过。
func ParseHeaderLines(text string) (map[string]string, error) {
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.IndexByte(line, ':')
		if i <= 0 {
			return nil, fmt.Errorf("请求头格式错误(需要 名称: 值):%s", line)
		}
		out[strings.TrimSpace(line[:i])] = strings.TrimSpace(line[i+1:])
	}
	return out, nil
}
