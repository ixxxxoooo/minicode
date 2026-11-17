// Package customize 是「自定义」页背后的数据模型:插件、MCP 服务器、Skills、斜杠命令、子 Agent、
// 规则与记忆、钩子七类条目,以及筛选(范围 / 关键字)、增删、启停与校验。与界面无关。
// 真实读写 ~/.claude 与项目 .claude 目录以后实现持久化层即可;演示数据在 internal/demo。
package customize

import (
	"errors"
	"sort"
	"strings"
)

// Cat 是条目类别。
type Cat string

const (
	Plugins  Cat = "plugins"
	MCP      Cat = "mcp"
	Skills   Cat = "skills"
	Commands Cat = "commands"
	Agents   Cat = "agents"
	Rules    Cat = "rules"
	Hooks    Cat = "hooks"
)

// Cats 是左侧分类栏的顺序。
var Cats = []Cat{Plugins, MCP, Skills, Commands, Agents, Rules, Hooks}

// Scope 是配置所在的层。
type Scope byte

const (
	All     Scope = 0
	User    Scope = 'u'
	Project Scope = 'p'
	Local   Scope = 'l'
)

// Label 是范围的中文名。
func (s Scope) Label() string {
	switch s {
	case User:
		return "用户"
	case Project:
		return "项目"
	case Local:
		return "本地"
	}
	return "全部"
}

// Status 是 MCP 服务器的连接状态。
type Status string

const (
	StNone Status = ""
	StOK   Status = "ok"
	StWarn Status = "warn" // 需要授权
	StErr  Status = "err"
	StOff  Status = "off" // 在本项目里被停用(不是连接失败)
)

// Tool 是 MCP 的一个工具及其开关。
type Tool struct {
	Name string
	On   bool
}

// Inc 是插件包含的一类内容及数量。
type Inc struct {
	Kind string
	N    int
	On   bool
}

// File 是 Skill 目录里的一个文件。
type File struct {
	Name   string
	Indent int
	Active bool
	Dir    bool
	Bytes  int64 // 磁盘字节数,演示数据可手写
}

// Item 是任一类别的一个条目;各类只用到自己那部分字段。
type Item struct {
	Name, Desc string
	Scope      Scope
	On         bool

	// 插件
	Ver, Upd, Src, About string
	Inc                  []Inc
	// MCP
	Status    Status
	Transport string // stdio / http
	Cmd       string // 命令或地址
	Env       []string
	EnvVals   map[string]string // 键值(读盘时填充;保存 MCP 时用)
	Tok       string
	Tools     []Tool
	OnDemand  bool
	// Skills
	Path  string
	Trig  []string
	Files []File
	// 斜杠命令 / 子 Agent
	Note   string // 说明(命令的说明、子 Agent 何时委派)
	Hint   string
	Allow  []string // 允许的工具
	Model  int      // 下标见 Models
	Body   string   // 命令提示词 / 子 Agent 系统提示词 / 规则正文
	Color  int      // 子 Agent 颜色下标
	Size   string   // 规则文件大小
	Tokens int      // 规则 token 估算
	// 钩子
	Event   int
	Match   string
	Timeout int
	Trusted bool
}

// Clone 深拷贝一项(草稿编辑用:改的是副本,保存时才写回)。
func (it *Item) Clone() Item {
	d := *it
	d.Inc = append([]Inc(nil), it.Inc...)
	d.Env = append([]string(nil), it.Env...)
	d.Tools = append([]Tool(nil), it.Tools...)
	d.Trig = append([]string(nil), it.Trig...)
	d.Files = append([]File(nil), it.Files...)
	d.Allow = append([]string(nil), it.Allow...)
	return d
}

// Models 是命令 / 子 Agent 可选的模型。
var Models = []string{"继承会话", "Sonnet 5.5", "Opus 5.5", "Haiku 5.5"}

// AllTools 是子 Agent / 命令可授予的工具。
var AllTools = []string{"Read", "Grep", "Glob", "Edit", "Write", "Bash", "WebFetch", "WebSearch"}

// Events 是钩子事件。
var Events = []string{"PreToolUse", "PostToolUse", "Stop", "UserPromptSubmit", "Notification"}

// Market 是市场里的一个插件。
type Market struct {
	Name, By, Desc, Stars string
	Installed             bool
	MP                    string // 所属市场(安装时拼成 name@market)
	Ver                   string // 市场里标注的版本(没有就空)
}

// Store 保存全部条目。
type Store struct {
	items  map[Cat][]*Item
	Market []*Market
}

// NewStore 用给定条目建仓。
func NewStore(items map[Cat][]*Item, market []*Market) *Store {
	if items == nil {
		items = map[Cat][]*Item{}
	}
	return &Store{items: items, Market: market}
}

// Count 返回某类的条目数。
func (s *Store) Count(c Cat) int { return len(s.items[c]) }

// List 按范围与关键字(不分大小写,匹配名称与说明)筛选。
func (s *Store) List(c Cat, sc Scope, q string) []*Item {
	q = strings.ToLower(strings.TrimSpace(q))
	var out []*Item
	for _, it := range s.items[c] {
		if sc != All && it.Scope != sc {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(it.Name+" "+it.Desc), q) {
			continue
		}
		out = append(out, it)
	}
	return out
}

// Has 判断同类里是否已有同名条目(忽略 except 自己)。
func (s *Store) Has(c Cat, name string, except *Item) bool {
	for _, it := range s.items[c] {
		if it != except && it.Name == name {
			return true
		}
	}
	return false
}

// ValidName 校验名称:不能为空、不含空白(钩子与规则文件除外)、同类不重名;命令必须以 / 开头。
func (s *Store) ValidName(c Cat, name string, except *Item) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("名称不能为空")
	}
	if c != Hooks && c != Rules && strings.ContainsAny(name, " \t\n") {
		return errors.New("名称不能包含空格")
	}
	if c == Commands && !strings.HasPrefix(name, "/") {
		return errors.New("命令名要以 / 开头")
	}
	if s.Has(c, name, except) {
		return errors.New("已存在同名的" + name)
	}
	return nil
}

// Add 新建一项并返回它;名称不合法时返回错误。extra 是各类的第一项必填说明(MCP 的命令 / 地址,命令与子 Agent 的说明)。
func (s *Store) Add(c Cat, name string, sc Scope, extra string) (*Item, error) {
	name = strings.TrimSpace(name)
	if err := s.ValidName(c, name, nil); err != nil {
		return nil, err
	}
	if sc == All {
		sc = User
	}
	it := &Item{Name: name, Scope: sc, On: true, Desc: sc.Label() + " · 刚刚创建"}
	switch c {
	case MCP:
		it.Status, it.Transport, it.Cmd, it.Tok = StErr, "stdio", extra, "—"
		if strings.HasPrefix(extra, "http") {
			it.Transport = "http"
		}
		it.Desc = sc.Label() + " · 正在连接…"
	case Skills:
		it.Path = ".claude/skills/" + name
		it.Files = []File{{Name: "SKILL.md", Active: true}}
	case Commands:
		it.Body, it.Allow = "", nil
		it.Hint = ""
		it.Note = extra
	case Agents:
		it.Allow = []string{"Read", "Grep", "Glob"}
		it.Note = extra
	case Rules:
		it.Size, it.Body = "0 KB", "# "+name
	case Hooks:
		it.Timeout = 10
	case Plugins:
		return nil, errors.New("插件请从市场或本地目录安装")
	}
	s.items[c] = append(s.items[c], it)
	return it, nil
}

// Remove 删除一项。
func (s *Store) Remove(c Cat, it *Item) {
	l := s.items[c]
	for i, x := range l {
		if x == it {
			s.items[c] = append(l[:i:i], l[i+1:]...)
			return
		}
	}
}

// ToggleTool 切换 MCP 工具开关。
func (it *Item) ToggleTool(i int) {
	if i >= 0 && i < len(it.Tools) {
		it.Tools[i].On = !it.Tools[i].On
	}
}

// ToggleAllow 切换允许的工具(用于子 Agent / 命令的工具选择)。
func (it *Item) ToggleAllow(tool string) {
	for i, t := range it.Allow {
		if t == tool {
			it.Allow = append(it.Allow[:i:i], it.Allow[i+1:]...)
			return
		}
	}
	it.Allow = append(it.Allow, tool)
	sort.SliceStable(it.Allow, func(a, b int) bool { return toolIx(it.Allow[a]) < toolIx(it.Allow[b]) })
}

func toolIx(t string) int {
	for i, x := range AllTools {
		if x == t {
			return i
		}
	}
	return len(AllTools) // 带参数的规则 / MCP 工具:排在内置工具之后
}

// Preset 返回子 Agent 工具权限的预设:0 只读 / 1 可编辑 / 2 全部。
func (it *Item) Preset() int {
	if len(it.Allow) == len(AllTools) {
		return 2
	}
	for _, t := range it.Allow {
		if t == "Edit" || t == "Write" || t == "Bash" {
			return 1
		}
	}
	return 0
}

// SetPreset 应用预设。
func (it *Item) SetPreset(p int) {
	switch p {
	case 0:
		it.Allow = []string{"Read", "Grep", "Glob"}
	case 1:
		it.Allow = []string{"Read", "Grep", "Glob", "Edit", "Write", "Bash"}
	default:
		it.Allow = append([]string(nil), AllTools...)
	}
}

// Reconnect 模拟重新连接 MCP:没有失败原因可查的演示里,命令非空就算连上。
func (it *Item) Reconnect() {
	if !it.On || strings.TrimSpace(it.Cmd) == "" {
		it.Status = StErr
		return
	}
	it.Status = StOK
	it.Desc = it.Transport + " · " + itoa(len(it.Tools)) + " 个工具 · 已连接"
}

// Authorize 完成 MCP 授权。
func (it *Item) Authorize() {
	if it.Status == StWarn {
		it.Status = StOK
		it.Desc = it.Transport + " · 已授权"
	}
}

// Install 安装市场里的插件到用户层;已安装返回错误。
func (s *Store) Install(m *Market) (*Item, error) {
	if m.Installed {
		return nil, errors.New("已安装")
	}
	m.Installed = true
	it := &Item{Name: m.Name, Scope: User, On: true, Ver: "1.0.0", Src: "claude-plugins-official", About: m.Desc,
		Desc: "官方市场 · v1.0.0",
		Inc:  []Inc{{"Skills", 6, true}, {"斜杠命令", 2, true}, {"子 Agent", 1, true}}}
	s.items[Plugins] = append(s.items[Plugins], it)
	return it, nil
}

// Update 把插件升到可更新的版本。
func (it *Item) Update() {
	if it.Upd != "" {
		it.Ver, it.Upd = it.Upd, ""
	}
}

// Uninstall 卸载插件,同时让市场里对应项变回未安装。
func (s *Store) Uninstall(it *Item) {
	s.Remove(Plugins, it)
	for _, m := range s.Market {
		if m.Name == it.Name {
			m.Installed = false
		}
	}
}

// RuleTokenPct 规则占上下文的百分比(按 2000 token 的基准,与设计稿一致)。
func (it *Item) RuleTokenPct() float64 { return float64(it.Tokens) / 2000 * 100 }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// Save 校验并把草稿写回条目:名称要合法且同类不重名。
func (s *Store) Save(c Cat, it *Item, draft Item) error {
	draft.Name = strings.TrimSpace(draft.Name)
	if err := s.ValidName(c, draft.Name, it); err != nil {
		return err
	}
	if c == Hooks && draft.Timeout < 1 {
		return errors.New("超时至少 1 秒")
	}
	*it = draft
	return nil
}
