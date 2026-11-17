package customize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiskStore 在内存 Store 上挂上项目根,支持从本机 ~/.claude 与项目 .claude 读写。
type DiskStore struct {
	*Store
	proj, home string
	claudeJSON string
}

// OpenDisk 加载指定项目的自定义配置;proj 空则只读用户层。
func OpenDisk(proj string) (*DiskStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	proj = strings.TrimSpace(proj)
	if proj != "" {
		proj = filepath.Clean(proj)
	}
	ds := &DiskStore{
		Store:      NewStore(map[Cat][]*Item{}, nil),
		proj:       proj,
		home:       home,
		claudeJSON: filepath.Join(home, ".claude.json"),
	}
	// 市场列表要起 claude 子进程(最长 60 秒),而 OpenDisk 会在界面线程里被调用(输入框 MCP 芯片、斜杠命令等),
	// 所以这里只放兜底数据;真实列表由调用方放到后台取(RunMarketplaceList),回主线程后用 SetMarket 替换。
	if mk := ReadMarketplaces(home); len(mk) > 0 {
		ds.Market = mk // 真实的市场内容,直接读磁盘,不起子进程
	} else {
		ds.Market = defaultMarket()
	}
	ds.loadAll()
	return ds, nil
}

// SetMarket 用真实的市场列表替换兜底数据,并按已安装的插件重新标注「已安装」。空列表忽略。
// 只能在持有该 DiskStore 的线程(界面线程)调用。
func (ds *DiskStore) SetMarket(mk []*Market) {
	if len(mk) == 0 {
		return
	}
	ds.Market = mk
	for _, it := range ds.items[Plugins] {
		for _, m := range mk {
			if m.Name == it.Name {
				m.Installed = true
			}
		}
	}
}

func defaultMarket() []*Market {
	// 市场列表仍为演示数据,安装走真实 CLI
	return []*Market{
		{Name: "frontend-design", By: "官方", Desc: "让界面设计有风格,而不是千篇一律", Stars: "4.2k"},
		{Name: "security-guidance", By: "官方", Desc: "写代码时提示常见的安全隐患", Stars: "2.8k"},
		{Name: "pr-review-toolkit", By: "官方", Desc: "一组专门审查 PR 的子 Agent", Stars: "3.1k"},
		{Name: "code-review", By: "官方", Desc: "按严重程度审查当前改动", Stars: "5.6k"},
		{Name: "superpowers", By: "官方", Desc: "工程方法论 Skills 合集", Stars: "9.4k"},
	}
}

// ProjRoot 返回绑定的项目目录。
func (ds *DiskStore) ProjRoot() string { return ds.proj }

func (ds *DiskStore) loadAll() {
	ds.loadPlugins()
	ds.loadMCP()
	ds.loadSkills()
	ds.loadCommands()
	ds.loadAgents()
	ds.loadRules()
	ds.loadHooks()
}

func userClaude(home string) string { return filepath.Join(home, ".claude") }

func (ds *DiskStore) relDisplay(abs string) string {
	abs = filepath.Clean(abs)
	if ds.proj != "" {
		if rel, err := filepath.Rel(ds.proj, abs); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	if strings.HasPrefix(abs, ds.home) {
		return "~" + strings.TrimPrefix(abs, ds.home)
	}
	return abs
}

func envKeysOnly(m map[string]any) []string {
	var out []string
	for k := range m {
		if !strings.Contains(strings.ToUpper(k), "KEY") && !strings.Contains(strings.ToUpper(k), "SECRET") && !strings.Contains(strings.ToUpper(k), "TOKEN") {
			// 仍只显示名称,不校验是否真密钥
		}
		out = append(out, k)
	}
	return out
}

func envValsOf(m map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func readJSON(path string, v any) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	if err := json.Unmarshal(b, v); err != nil {
		quarantine(path, err)
		return false
	}
	return true
}

func (ds *DiskStore) loadPlugins() {
	p := filepath.Join(userClaude(ds.home), "plugins", "installed_plugins.json")
	var raw struct {
		Plugins map[string][]struct {
			Scope       string `json:"scope"`
			InstallPath string `json:"installPath"`
			Version     string `json:"version"`
			ProjectPath string `json:"projectPath"`
		} `json:"plugins"`
	}
	if !readJSON(p, &raw) {
		return
	}
	enabled := ds.enabledPlugins()
	ids := make([]string, 0, len(raw.Plugins))
	for id := range raw.Plugins {
		ids = append(ids, id)
	}
	sort.Strings(ids) // map 遍历随机,按名字排,列表不乱跳
	for _, id := range ids {
		// 同一个插件可能装在多个层:项目层的只在它所属的项目里生效,别的项目里不列
		pick := -1
		for i, v := range raw.Plugins[id] {
			if strings.EqualFold(v.Scope, "project") && filepath.Clean(v.ProjectPath) != ds.proj {
				continue
			}
			pick = i
		}
		if pick < 0 {
			continue
		}
		v := raw.Plugins[id][pick]
		name, mp := id, ""
		if i := strings.Index(id, "@"); i > 0 {
			name, mp = id[:i], id[i+1:]
		}
		sc := User
		if strings.EqualFold(v.Scope, "project") {
			sc = Project
		}
		on, known := enabled[id]
		if !known {
			on = false // settings 里没写 = 没启用(claude 的默认)
		}
		src := mp
		if src == "" {
			src = "已安装"
		}
		it := &Item{
			Name: name, Scope: sc, On: on, Ver: v.Version, Src: src,
			Desc: scopeLabel(v.Scope) + " · v" + v.Version,
			Path: v.InstallPath, Inc: pluginInc(v.InstallPath),
		}
		ds.items[Plugins] = append(ds.items[Plugins], it)
		for _, m := range ds.Market {
			if m.Name == name && (mp == "" || m.MP == mp) {
				m.Installed = true
				it.About = m.Desc // 详情页的说明取自市场清单
			}
		}
	}
}

// PluginID 返回插件在 claude 里的完整标识(名称@市场),装机时没有市场信息就退回名称。
func PluginID(it *Item) string {
	if it.Src != "" && it.Src != "已安装" {
		return it.Name + "@" + it.Src
	}
	return it.Name
}

func mcpCmdOf(ent map[string]any) (transport, cmd string) {
	if u, _ := ent["url"].(string); strings.TrimSpace(u) != "" {
		return "http", u
	}
	if u, _ := ent["serverUrl"].(string); strings.TrimSpace(u) != "" {
		return "http", u
	}
	parts := []string{}
	if c, _ := ent["command"].(string); c != "" {
		parts = append(parts, c)
	}
	if args, ok := ent["args"].([]any); ok {
		for _, a := range args {
			if s, ok := a.(string); ok {
				parts = append(parts, s)
			}
		}
	}
	return "stdio", strings.Join(parts, " ")
}

func (ds *DiskStore) loadMCP() {
	ds.items[MCP] = ds.readMCP()
}

// ReadMCPConfig 只读磁盘配置并返回一份全新的 MCP 列表,不改 ds 的任何状态,
// 所以可以放在后台线程里调用;结果交回界面线程用 ApplyMCPConfig 合并。
func (ds *DiskStore) ReadMCPConfig() []*Item { return ds.readMCP() }

// ApplyMCPConfig 把磁盘读到的 MCP 配置合并进当前列表(只能在界面线程调用):
// 同名项原地更新(保留指针,界面选中的条目不会丢,也保留已查到的连接状态 / 工具列表),
// 新增的追加,磁盘上已经没有的移除。此前每次刷新都是「再追加一遍」,列表越刷越多。
func (ds *DiskStore) ApplyMCPConfig(fresh []*Item) {
	old := map[string]*Item{}
	for _, it := range ds.items[MCP] {
		if _, dup := old[it.Name]; !dup { // 旧版本留下的重复项只认第一个,其余丢弃
			old[it.Name] = it
		}
	}
	out := make([]*Item, 0, len(fresh))
	for _, n := range fresh {
		if cur, ok := old[n.Name]; ok {
			cur.Scope, cur.On = n.Scope, n.On
			cur.Transport, cur.Cmd = n.Transport, n.Cmd
			cur.Env, cur.EnvVals = n.Env, n.EnvVals
			cur.OnDemand = n.OnDemand
			out = append(out, cur)
			continue
		}
		out = append(out, n)
	}
	ds.items[MCP] = out
}

// readMCP 读用户层与项目层的 MCP 服务器配置(按名字去重,用户层优先)。
func (ds *DiskStore) readMCP() []*Item {
	var result []*Item
	type layer struct {
		scope Scope
		path  string
		key   string // projects[proj].mcpServers
	}
	var layers []layer
	layers = append(layers, layer{User, ds.claudeJSON, "mcpServers"})
	if ds.proj != "" {
		layers = append(layers, layer{Project, filepath.Join(ds.proj, ".mcp.json"), "mcpServers"})
	}
	disabled := map[string]bool{}
	var root map[string]any
	if readJSON(ds.claudeJSON, &root) && ds.proj != "" {
		if pr, ok := root["projects"].(map[string]any); ok {
			if ent, ok := pr[ds.proj].(map[string]any); ok {
				// 项目条目里没有该键(或类型不对)很常见,不能直接断言,否则整个界面构建会 panic
				list, _ := ent["disabledMcpServers"].([]any)
				for _, n := range list {
					if s, ok := n.(string); ok {
						disabled[s] = true
					}
				}
			}
		}
	}
	onDemand, _ := ds.ToolSearchValue()
	seen := map[string]bool{}
	for _, ly := range layers {
		var v map[string]any
		if ly.key == "mcpServers" && ly.scope == User {
			if root == nil {
				readJSON(ly.path, &v)
			} else {
				v = root
			}
		} else {
			readJSON(ly.path, &v)
		}
		if v == nil {
			continue
		}
		mcp, _ := v["mcpServers"].(map[string]any)
		if mcp == nil && ly.scope == User && root != nil {
			mcp, _ = root["mcpServers"].(map[string]any)
		}
		for name, cfg := range mcp {
			if seen[name] {
				continue
			}
			seen[name] = true
			ent, _ := cfg.(map[string]any)
			tr, cmd := mcpCmdOf(ent)
			it := &Item{Name: name, Scope: ly.scope, On: !disabled[name], Status: StNone, Transport: tr, Cmd: cmd, OnDemand: onDemand,
				Desc: ly.scope.Label() + " · 未查询连接状态"} // 还没查过就是「未知」,不能先标成失败(红点)
			if em, ok := ent["env"].(map[string]any); ok {
				it.Env = envKeysOnly(em)
				it.EnvVals = envValsOf(em)
			}
			result = append(result, it)
		}
	}
	// map 遍历顺序随机:按名字排序,列表才不会每次刷新都乱跳
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (ds *DiskStore) loadSkills() {
	loadSkillDir := func(base string, sc Scope) {
		ents, _ := os.ReadDir(base)
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			skillMD := filepath.Join(base, e.Name(), "SKILL.md")
			if st, err := os.Stat(skillMD); err != nil || st.IsDir() {
				continue
			}
			desc := sc.Label() + " · Skill"
			if b, err := os.ReadFile(skillMD); err == nil {
				_, body := splitFrontmatter(string(b))
				if i := strings.Index(body, "\n"); i > 0 {
					line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(body), "#"))
					if line != "" {
						desc = sc.Label() + " · " + line
					}
				}
			}
			var files []File
			filepath.Walk(filepath.Join(base, e.Name()), func(p string, fi os.FileInfo, err error) error {
				if err != nil || fi.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(filepath.Join(base, e.Name()), p)
				files = append(files, File{Name: rel, Active: rel == "SKILL.md", Bytes: fi.Size()})
				return nil
			})
			it := &Item{Name: e.Name(), Scope: sc, On: true, Path: ds.relDisplay(filepath.Join(base, e.Name())), Desc: desc, Files: files}
			ds.items[Skills] = append(ds.items[Skills], it)
		}
	}
	loadSkillDir(filepath.Join(userClaude(ds.home), "skills"), User)
	if ds.proj != "" {
		loadSkillDir(filepath.Join(ds.proj, ".claude", "skills"), Project)
	}
}

func loadCmdMD(path string, sc Scope, rel string) *Item {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	fm, body := splitFrontmatter(string(b))
	name := filepath.Base(path)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	it := &Item{
		Name: name, Scope: sc, On: true, Path: rel, Body: body,
		Note: fmLine(fm, "description"), Hint: fmLine(fm, "argument-hint"),
		Desc: sc.Label() + " · " + fmLine(fm, "description"),
	}
	it.Allow = append(it.Allow, SplitTools(fmLine(fm, "allowed-tools"))...) // 括号里的逗号不拆
	return it
}

func (ds *DiskStore) loadCommands() {
	scan := func(dir string, sc Scope) {
		ents, _ := os.ReadDir(dir)
		for _, e := range ents {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			p := filepath.Join(dir, e.Name())
			if it := loadCmdMD(p, sc, ds.relDisplay(p)); it != nil {
				ds.items[Commands] = append(ds.items[Commands], it)
			}
		}
	}
	scan(filepath.Join(userClaude(ds.home), "commands"), User)
	if ds.proj != "" {
		scan(filepath.Join(ds.proj, ".claude", "commands"), Project)
	}
}

func (ds *DiskStore) loadAgents() {
	scan := func(dir string, sc Scope) {
		ents, _ := os.ReadDir(dir)
		for _, e := range ents {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			p := filepath.Join(dir, e.Name())
			b, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			fm, body := splitFrontmatter(string(b))
			name := strings.TrimSuffix(e.Name(), ".md")
			it := &Item{
				Name: name, Scope: sc, On: true, Path: ds.relDisplay(p), Body: body,
				Note: fmLine(fm, "description"), Desc: sc.Label() + " · " + fmLine(fm, "description"),
			}
			ds.items[Agents] = append(ds.items[Agents], it)
		}
	}
	scan(filepath.Join(userClaude(ds.home), "agents"), User)
	if ds.proj != "" {
		scan(filepath.Join(ds.proj, ".claude", "agents"), Project)
	}
}

func (ds *DiskStore) loadRules() {
	add := func(path string, sc Scope) {
		st, err := os.Stat(path)
		if err != nil || st.IsDir() {
			return
		}
		b, _ := os.ReadFile(path)
		body := string(b)
		tok := len(body) / 4
		if tok < 1 {
			tok = 1
		}
		size := fmt.Sprintf("%.1f KB", float64(st.Size())/1024)
		it := &Item{
			Name: ds.relDisplay(path), Scope: sc, On: true, Path: path, Body: body,
			Size: size, Tokens: tok, Desc: sc.Label() + " · " + size,
		}
		ds.items[Rules] = append(ds.items[Rules], it)
	}
	add(filepath.Join(ds.home, ".claude", "CLAUDE.md"), User)
	if ds.proj != "" {
		add(filepath.Join(ds.proj, "CLAUDE.md"), Project)
		add(filepath.Join(ds.proj, "CLAUDE.local.md"), Local)
		rd := filepath.Join(ds.proj, ".claude", "rules")
		ents, _ := os.ReadDir(rd)
		for _, e := range ents {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				add(filepath.Join(rd, e.Name()), Project)
			}
		}
	}
}

func hookEventIndex(name string) int {
	for i, e := range Events {
		if e == name {
			return i
		}
	}
	return 0
}

func (ds *DiskStore) loadHooks() {
	type src struct {
		path string
		sc   Scope
	}
	var srcs []src
	srcs = append(srcs, src{filepath.Join(userClaude(ds.home), "settings.json"), User})
	if ds.proj != "" {
		srcs = append(srcs, src{filepath.Join(ds.proj, ".claude", "settings.json"), Project})
		srcs = append(srcs, src{filepath.Join(ds.proj, ".claude", "settings.local.json"), Local})
	}
	for _, s := range srcs {
		var v map[string]any
		if !readJSON(s.path, &v) {
			continue
		}
		hooks, _ := v["hooks"].(map[string]any)
		for evName, arr := range hooks {
			list, _ := arr.([]any)
			for _, h := range list {
				hm, _ := h.(map[string]any)
				match, _ := hm["matcher"].(string)
				inner, _ := hm["hooks"].([]any)
				for _, ih := range inner {
					im, _ := ih.(map[string]any)
					cmd, _ := im["command"].(string)
					if strings.TrimSpace(cmd) == "" {
						continue
					}
					to := 10
					if f, ok := im["timeout"].(float64); ok && f > 0 {
						to = int(f + 0.5)
					}
					name := evName
					if match != "" {
						name += " · " + match
					}
					it := &Item{
						Name: name, Scope: s.sc, On: true, Event: hookEventIndex(evName),
						Match: match, Cmd: cmd, Timeout: to, Trusted: s.sc != Project,
						Desc: s.sc.Label() + " · " + strings.TrimSpace(cmd),
					}
					ds.items[Hooks] = append(ds.items[Hooks], it)
				}
			}
		}
	}
}

// RefreshMCPStatus 用 claude mcp list 的输出更新状态(不启动会话)。
// 同步版本只在测试里用:会跑子进程,界面线程请走 FetchMCPStatus(后台)+ ApplyMCPStatus(界面线程)。
func (ds *DiskStore) RefreshMCPStatus() {
	if st, err := ds.FetchMCPStatus(); err == nil {
		ds.ApplyMCPStatus(st)
	}
}

// MCPStatus 是一次 claude mcp list 的结果快照(按服务器名索引)。
type MCPStatus struct{ byName map[string]mcpLine }

// FetchMCPStatus 跑 claude mcp list 并解析,不改 ds(可在后台线程调用)。
func (ds *DiskStore) FetchMCPStatus() (*MCPStatus, error) {
	out, err := RunMCPList(ds.proj)
	if err != nil {
		alog.Warn("MCP 列表失败", "err", err)
		return nil, err
	}
	st := &MCPStatus{byName: map[string]mcpLine{}}
	for _, ln := range out {
		st.byName[ln.Name] = ln
	}
	return st, nil
}

// ApplyMCPStatus 把状态快照写到各条目上(只能在界面线程调用)。
func (ds *DiskStore) ApplyMCPStatus(st *MCPStatus) {
	if st == nil {
		return
	}
	for _, it := range ds.items[MCP] {
		ln, ok := st.byName[it.Name]
		if !ok {
			it.Status = StErr
			it.Desc = it.Scope.Label() + " · 未在 claude mcp list 中"
			continue
		}
		it.Status = ln.Status
		it.Desc = ln.Desc
		it.Tools = ln.Tools
	}
}

// Persist 把草稿保存到磁盘。
func (ds *DiskStore) Persist(c Cat, it *Item, draft Item) error {
	switch c {
	case Commands:
		return ds.saveCommand(it, draft)
	case Agents:
		return ds.saveAgent(it, draft)
	case Rules:
		return ds.saveRule(it, draft)
	case Hooks:
		return ds.saveHook(it, draft)
	case Skills:
		return ds.saveSkillMeta(it, draft)
	case MCP:
		return ds.SaveMCP(it, draft, nil)
	default:
		return nil // 插件由 CLI 安装
	}
}

// RemoveDisk 从磁盘删除一项。
func (ds *DiskStore) RemoveDisk(c Cat, it *Item) error {
	if it.Path == "" {
		return fmt.Errorf("没有对应的磁盘路径")
	}
	p := it.Path
	if strings.HasPrefix(p, "~") {
		p = filepath.Join(ds.home, strings.TrimPrefix(p, "~"))
	} else if ds.proj != "" && !filepath.IsAbs(p) {
		p = filepath.Join(ds.proj, p)
	}
	if c == Skills {
		return os.RemoveAll(p)
	}
	return os.Remove(p)
}
