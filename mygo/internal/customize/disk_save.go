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
	p := strings.TrimSpace(it.Path)
	if p == "" {
		return ""
	}
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
		name := strings.TrimSpace(draft.Name)
		switch {
		case strings.HasPrefix(name, "~"):
			path = filepath.Join(ds.home, strings.TrimPrefix(name, "~"))
		case filepath.IsAbs(name):
			path = name
		case ds.proj != "" && (strings.Contains(name, "/") || strings.HasSuffix(name, ".md")):
			path = filepath.Join(ds.proj, name)
		case ds.proj != "":
			path = filepath.Join(ds.proj, ".claude", "rules", name+".md")
		default:
			path = filepath.Join(userClaude(ds.home), "rules", name+".md")
		}
		old.Path = ds.relDisplay(path)
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

func (ds *DiskStore) skillDirAbs(it *Item, draft Item) string {
	name := strings.TrimSpace(draft.Name)
	if name == "" {
		name = it.Name
	}
	if it.Path != "" {
		p := it.Path
		if strings.HasPrefix(p, "~") {
			return filepath.Join(ds.home, strings.TrimPrefix(p, "~/"))
		}
		if filepath.IsAbs(p) {
			return p
		}
		if strings.HasPrefix(p, ".claude/") {
			return filepath.Join(ds.home, p)
		}
		if draft.Scope == Project && ds.proj != "" {
			return filepath.Join(ds.proj, p)
		}
		if ds.proj != "" {
			return filepath.Join(ds.proj, p)
		}
		return filepath.Join(ds.home, p)
	}
	if draft.Scope == Project && ds.proj != "" {
		return filepath.Join(ds.proj, ".claude", "skills", name)
	}
	return filepath.Join(userClaude(ds.home), "skills", name)
}

func (ds *DiskStore) saveSkillMeta(old *Item, draft Item) error {
	newName := strings.TrimSpace(draft.Name)
	if newName == "" {
		return fmt.Errorf("名称不能为空")
	}
	abs := ds.skillDirAbs(old, draft)
	if old.Path != "" && newName != old.Name {
		parent := filepath.Dir(abs)
		dst := filepath.Join(parent, newName)
		if err := os.Rename(abs, dst); err != nil {
			return fmt.Errorf("重命名目录失败:%w", err)
		}
		abs = dst
		old.Name = newName
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	old.Path = ds.relDisplay(abs)
	skill := filepath.Join(abs, "SKILL.md")
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
	// 项目条目里没有该键很常见,必须用带检查的断言,否则 nil 接口会 panic
	oldDisabled, _ := ent["disabledMcpServers"].([]any)
	for _, x := range oldDisabled {
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

// MCPDenySet 一次读出本地层 permissions.deny 里全部 mcp__ 规则(键即规则原文)。
// 弹层里每个工具行都要问「是否被禁用」,逐个读文件每帧会读几十次,所以批量读一次再缓存。
func (ds *DiskStore) MCPDenySet() map[string]bool {
	out := map[string]bool{}
	path := filepath.Join(ds.proj, ".claude", "settings.local.json")
	var root map[string]any
	if !readJSON(path, &root) {
		return out
	}
	perms, _ := root["permissions"].(map[string]any)
	deny, _ := perms["deny"].([]any)
	for _, x := range deny {
		if s, ok := x.(string); ok && strings.HasPrefix(s, "mcp__") {
			out[s] = true
		}
	}
	return out
}

// MCPToolKey 是工具对应的 permissions.deny 规则文本。
func MCPToolKey(server, tool string) string {
	return "mcp__" + strings.ReplaceAll(server, " ", "_") + "__" + tool
}
