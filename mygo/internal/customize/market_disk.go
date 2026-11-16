package customize

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ReadMarketplaces 读 ~/.claude/plugins/marketplaces/*/.claude-plugin/marketplace.json,
// 列出所有已添加市场里真实可装的插件(不起子进程,毫秒级)。
// 此前市场列表是 claude plugin marketplace list 的输出——那列的是「市场」本身而不是插件,
// 界面上的名字、说明、热度全是假的,安装时还把市场写死成 claude-plugins-official。
func ReadMarketplaces(home string) []*Market {
	files, _ := filepath.Glob(filepath.Join(userClaude(home), "plugins", "marketplaces", "*", ".claude-plugin", "marketplace.json"))
	var out []*Market
	for _, f := range files {
		var raw struct {
			Name    string `json:"name"`
			Plugins []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Version     string `json:"version"`
				Author      any    `json:"author"` // 有的是字符串,有的是 {name,email}
			} `json:"plugins"`
		}
		if !readJSON(f, &raw) {
			continue
		}
		mp := raw.Name
		if mp == "" {
			mp = filepath.Base(filepath.Dir(filepath.Dir(f)))
		}
		for _, p := range raw.Plugins {
			if strings.TrimSpace(p.Name) == "" {
				continue
			}
			by := mp
			switch a := p.Author.(type) {
			case string:
				if a != "" {
					by = a
				}
			case map[string]any:
				if n, _ := a["name"].(string); n != "" {
					by = n
				}
			}
			out = append(out, &Market{Name: p.Name, By: by, Desc: strings.TrimSpace(p.Description), MP: mp, Ver: p.Version})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { // 先按市场,再按名字,同一市场的挨在一起
		if out[i].MP != out[j].MP {
			return out[i].MP < out[j].MP
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// enabledPlugins 汇总用户层与项目层 settings 里的 enabledPlugins(后读的覆盖先读的:用户 < 项目 < 本地)。
func (ds *DiskStore) enabledPlugins() map[string]bool {
	on := map[string]bool{}
	srcs := []string{filepath.Join(userClaude(ds.home), "settings.json")}
	if ds.proj != "" {
		srcs = append(srcs, filepath.Join(ds.proj, ".claude", "settings.json"), filepath.Join(ds.proj, ".claude", "settings.local.json"))
	}
	for _, p := range srcs {
		var v struct {
			Enabled map[string]bool `json:"enabledPlugins"`
		}
		if !readJSON(p, &v) {
			continue
		}
		for id, b := range v.Enabled {
			on[id] = b
		}
	}
	return on
}

// pluginInc 数一数插件目录里各类内容的数量(技能 / 命令 / 子 Agent / 钩子 / MCP),给详情页「包含内容」用。
func pluginInc(dir string) []Inc {
	if dir == "" {
		return nil
	}
	countDir := func(sub string, wantDir bool) int {
		ents, _ := os.ReadDir(filepath.Join(dir, sub))
		n := 0
		for _, e := range ents {
			if strings.HasPrefix(e.Name(), ".") {
				continue
			}
			if wantDir && e.IsDir() {
				n++
			}
			if !wantDir && !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				n++
			}
		}
		return n
	}
	exists := func(rel string) int {
		if _, err := os.Stat(filepath.Join(dir, rel)); err == nil {
			return 1
		}
		return 0
	}
	var out []Inc
	add := func(kind string, n int) {
		if n > 0 {
			out = append(out, Inc{Kind: kind, N: n, On: true})
		}
	}
	add("Skills", countDir("skills", true))
	add("斜杠命令", countDir("commands", false))
	add("子 Agent", countDir("agents", false))
	add("钩子", exists("hooks/hooks.json"))
	add("MCP 服务器", exists(".mcp.json"))
	return out
}
