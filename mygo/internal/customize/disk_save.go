package customize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (ds *DiskStore) cmdPath(it *Item, draft Item) string {
	if it.Path != "" {
		p := it.Path
		if strings.HasPrefix(p, "~") {
			return filepath.Join(ds.home, strings.TrimPrefix(p, "~"))
		}
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(ds.proj, p)
	}
	name := strings.TrimPrefix(strings.TrimSpace(draft.Name), "/")
	dir := filepath.Join(userClaude(ds.home), "commands")
	if draft.Scope == Project && ds.proj != "" {
		dir = filepath.Join(ds.proj, ".claude", "commands")
	}
	return filepath.Join(dir, name+".md")
}

func (ds *DiskStore) saveCommand(old *Item, draft Item) error {
	fm := setFmLine("", "description", strings.TrimSpace(draft.Note))
	if draft.Hint != "" {
		fm = setFmLine(fm, "argument-hint", draft.Hint)
	}
	if len(draft.Allow) > 0 {
		fm = setFmLine(fm, "allowed-tools", strings.Join(draft.Allow, ", "))
	}
	body := joinFrontmatter(fm, draft.Body)
	path := ds.cmdPath(old, draft)
	if err := writeAtomic(path, []byte(body), 0o644); err != nil {
		return err
	}
	old.Path = ds.relDisplay(path)
	return nil
}

func (ds *DiskStore) agentPath(old *Item, draft Item) string {
	if old.Path != "" {
		p := old.Path
		if strings.HasPrefix(p, "~") {
			return filepath.Join(ds.home, strings.TrimPrefix(p, "~"))
		}
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(ds.proj, p)
	}
	dir := filepath.Join(userClaude(ds.home), "agents")
	if draft.Scope == Project && ds.proj != "" {
		dir = filepath.Join(ds.proj, ".claude", "agents")
	}
	return filepath.Join(dir, strings.TrimSpace(draft.Name)+".md")
}

func (ds *DiskStore) saveAgent(old *Item, draft Item) error {
	fm := setFmLine("", "description", strings.TrimSpace(draft.Note))
	body := joinFrontmatter(fm, draft.Body)
	path := ds.agentPath(old, draft)
	if err := writeAtomic(path, []byte(body), 0o644); err != nil {
		return err
	}
	old.Path = ds.relDisplay(path)
	return nil
}

func (ds *DiskStore) rulePath(it *Item) string {
	p := it.Path
	if strings.HasPrefix(p, "~") {
		return filepath.Join(ds.home, strings.TrimPrefix(p, "~"))
	}
	if filepath.IsAbs(p) {
		return p
	}
	if ds.proj != "" {
		return filepath.Join(ds.proj, p)
	}
	return p
}

func (ds *DiskStore) saveRule(old *Item, draft Item) error {
	path := ds.rulePath(old)
	if path == "" {
		return fmt.Errorf("不知道规则文件写到哪里")
	}
	if err := writeAtomic(path, []byte(draft.Body), 0o644); err != nil {
		return err
	}
	if st, err := os.Stat(path); err == nil {
		old.Size = fmt.Sprintf("%.1f KB", float64(st.Size())/1024)
		old.Tokens = len(draft.Body) / 4
	}
	return nil
}

func (ds *DiskStore) settingsPath(sc Scope) string {
	switch sc {
	case User:
		return filepath.Join(userClaude(ds.home), "settings.json")
	case Local:
		return filepath.Join(ds.proj, ".claude", "settings.local.json")
	default:
		return filepath.Join(ds.proj, ".claude", "settings.json")
	}
}

func (ds *DiskStore) saveHook(old *Item, draft Item) error {
	path := ds.settingsPath(draft.Scope)
	var root map[string]any
	if !readJSON(path, &root) {
		root = map[string]any{}
	}
	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		root["hooks"] = hooks
	}
	ev := Events[draft.Event%len(Events)]
	// 简化:整段替换同名条目
	arr, _ := hooks[ev].([]any)
	var kept []any
	for _, h := range arr {
		hm, _ := h.(map[string]any)
		match, _ := hm["matcher"].(string)
		if match == draft.Match && draft.Name != "" && strings.Contains(old.Name, match) {
			continue
		}
		kept = append(kept, h)
	}
	entry := map[string]any{
		"matcher": draft.Match,
		"hooks": []any{map[string]any{
			"type": "command", "command": strings.TrimSpace(draft.Cmd), "timeout": draft.Timeout,
		}},
	}
	kept = append(kept, entry)
	hooks[ev] = kept
	b, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, b, 0o600)
}

func (ds *DiskStore) saveSkillMeta(old *Item, draft Item) error {
	dir := old.Path
	if dir == "" {
		base := filepath.Join(userClaude(ds.home), "skills", strings.TrimSpace(draft.Name))
		if draft.Scope == Project && ds.proj != "" {
			base = filepath.Join(ds.proj, ".claude", "skills", strings.TrimSpace(draft.Name))
		}
		dir = base
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		old.Path = ds.relDisplay(dir)
	}
	skill := filepath.Join(dir, "SKILL.md")
	if strings.HasPrefix(dir, "~") {
		skill = filepath.Join(ds.home, strings.TrimPrefix(dir, "~"), "SKILL.md")
	} else if !filepath.IsAbs(dir) && ds.proj != "" {
		skill = filepath.Join(ds.proj, dir, "SKILL.md")
	}
	body := joinFrontmatter(setFmLine("", "name", draft.Name)+"\n"+setFmLine("", "description", draft.Desc), "# "+draft.Name+"\n")
	return writeAtomic(skill, []byte(body), 0o644)
}

// SetMCPEnabled 写 disabledMcpServers(用户层 claude.json)。
func (ds *DiskStore) SetMCPEnabled(name string, on bool) error {
	var root map[string]any
	if !readJSON(ds.claudeJSON, &root) {
		root = map[string]any{}
	}
	if ds.proj == "" {
		return fmt.Errorf("需要项目路径才能切换 MCP")
	}
	projects, _ := root["projects"].(map[string]any)
	if projects == nil {
		projects = map[string]any{}
		root["projects"] = projects
	}
	ent, _ := projects[ds.proj].(map[string]any)
	if ent == nil {
		ent = map[string]any{}
		projects[ds.proj] = ent
	}
	var disabled []string
	for _, x := range ent["disabledMcpServers"].([]any) {
		if s, ok := x.(string); ok && s != name {
			disabled = append(disabled, s)
		}
	}
	if !on {
		disabled = append(disabled, name)
	}
	ent["disabledMcpServers"] = disabled
	b, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(ds.claudeJSON, b, 0o600)
}

// SetMCPTool 在本地层写 permissions.deny 的 mcp__name__tool。
func (ds *DiskStore) SetMCPTool(server, tool string, on bool) error {
	path := filepath.Join(ds.proj, ".claude", "settings.local.json")
	var root map[string]any
	if !readJSON(path, &root) {
		root = map[string]any{}
	}
	perms, _ := root["permissions"].(map[string]any)
	if perms == nil {
		perms = map[string]any{}
		root["permissions"] = perms
	}
	deny, _ := perms["deny"].([]any)
	key := "mcp__" + strings.ReplaceAll(server, " ", "_") + "__" + tool
	var out []any
	for _, x := range deny {
		if s, ok := x.(string); ok && s == key {
			continue
		}
		out = append(out, x)
	}
	if !on {
		out = append(out, key)
	}
	perms["deny"] = out
	b, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, b, 0o600)
}
