package claudecfg

import "strings"

// Layer 是规则写入的配置层。
type Layer int

const (
	LayerUser Layer = iota
	LayerProject
	LayerLocal
)

func settingsPath(layer Layer, project string) string {
	switch layer {
	case LayerUser:
		return userSettings()
	default:
		if project == "" {
			return userSettings()
		}
		if layer == LayerLocal {
			return project + "/.claude/settings.local.json"
		}
		return project + "/.claude/settings.json"
	}
}

// PermRules 是三类权限规则。
type PermRules struct {
	Allow, Ask, Deny []string
}

// ReadRules 按层读取 permissions.*。
func ReadRules(layer Layer, project string) (PermRules, error) {
	path := settingsPath(layer, project)
	root, err := readJSON(path)
	if err != nil {
		return PermRules{}, err
	}
	return rulesOf(root), nil
}

func rulesOf(root map[string]any) PermRules {
	var out PermRules
	perm, _ := root["permissions"].(map[string]any)
	if perm == nil {
		return out
	}
	out.Allow = strList(perm["allow"])
	out.Ask = strList(perm["ask"])
	out.Deny = strList(perm["deny"])
	return out
}

func strList(v any) []string {
	a, _ := v.([]any)
	var out []string
	for _, x := range a {
		if s, ok := x.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

// AddRule 追加一条不重复的规则。
func AddRule(layer Layer, project, kind, rule string) error {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return nil
	}
	path := settingsPath(layer, project)
	root, err := readJSON(path)
	if err != nil {
		return err
	}
	perm, _ := root["permissions"].(map[string]any)
	if perm == nil {
		perm = map[string]any{}
		root["permissions"] = perm
	}
	key := map[string]string{"allow": "allow", "ask": "ask", "deny": "deny"}[kind]
	if key == "" {
		key = "allow"
	}
	list := strList(perm[key])
	for _, s := range list {
		if s == rule {
			return writeJSON(path, root)
		}
	}
	perm[key] = append(list, rule)
	log.Info("添加权限规则", "layer", layer, "kind", kind, "rule", rule)
	return writeJSON(path, root)
}

// RemoveRule 删除一条规则。
func RemoveRule(layer Layer, project, kind, rule string) error {
	path := settingsPath(layer, project)
	root, err := readJSON(path)
	if err != nil {
		return err
	}
	perm, _ := root["permissions"].(map[string]any)
	if perm == nil {
		return nil
	}
	key := map[string]string{"allow": "allow", "ask": "ask", "deny": "deny"}[kind]
	list := strList(perm[key])
	var next []string
	for _, s := range list {
		if s != rule {
			next = append(next, s)
		}
	}
	perm[key] = next
	log.Info("删除权限规则", "layer", layer, "kind", kind, "rule", rule)
	return writeJSON(path, root)
}
