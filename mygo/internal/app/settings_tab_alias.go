package app

// setTabAliases 把旧分类 id / 调试简写映射到新分类(偏好或书签里可能仍是旧 id)。
var setTabAliases = map[string]string{
	"customize":   "general",
	"model":       "models",
	"providers":   "models",
	"prompts":     "chat",
	"trust":       "permissions",
	"perm":        "permissions",
	"env":         "runtime",
	"network":     "runtime",
	"intel":       "lsp",
	"usage":       "general",
	"diagnostics": "privacy",
}

// setTabIDs 与 settings_tabs 里的分类 id 同步(单独列出以免包初始化循环)。
var setTabIDs = map[string]struct{}{
	"general": {}, "notify": {}, "shortcuts": {},
	"appearance": {}, "typography": {}, "motion": {}, "brand": {},
	"models": {}, "chat": {}, "permissions": {}, "runtime": {},
	"editor": {}, "lsp": {}, "terminal": {}, "scm": {}, "browser": {},
	"privacy": {}, "advanced": {}, "about": {},
}

// resolveSetTabID 解析并校验设置分类 id;未知 id 落到「主题与布局」。
func resolveSetTabID(id string) string {
	if id == "" {
		return "appearance"
	}
	seen := map[string]bool{}
	for {
		if seen[id] {
			return "appearance"
		}
		seen[id] = true
		if next, ok := setTabAliases[id]; ok {
			id = next
			continue
		}
		break
	}
	if _, ok := setTabIDs[id]; ok {
		return id
	}
	return "appearance"
}
