package claudecfg

import (
	"strconv"
	"strings"
)

// Mapping 是模型映射:Claude Code 的模型槽位 → 服务商的模型 ID。
// Enabled=false 表示使用 Claude 官方登录(不注入任何环境变量)。
type Mapping struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider,omitempty"`
	Default  string `json:"default,omitempty"`  // ANTHROPIC_MODEL
	Opus     string `json:"opus,omitempty"`     // ANTHROPIC_DEFAULT_OPUS_MODEL
	Sonnet   string `json:"sonnet,omitempty"`   // ANTHROPIC_DEFAULT_SONNET_MODEL
	Haiku    string `json:"haiku,omitempty"`    // ANTHROPIC_DEFAULT_HAIKU_MODEL
	Subagent string `json:"subagent,omitempty"` // CLAUDE_CODE_SUBAGENT_MODEL
}

// Slot 是一个映射槽位。Required 的槽位不能留空。
type Slot struct {
	Key, Label, Env string
	Required        bool
}

// Slots 全部槽位(界面从上到下的顺序)。
var Slots = []Slot{
	{"default", "默认", "ANTHROPIC_MODEL", true},
	{"opus", "Opus", "ANTHROPIC_DEFAULT_OPUS_MODEL", false},
	{"sonnet", "Sonnet", "ANTHROPIC_DEFAULT_SONNET_MODEL", false},
	{"haiku", "Haiku(后台小任务)", "ANTHROPIC_DEFAULT_HAIKU_MODEL", false},
	{"subagent", "子 Agent", "CLAUDE_CODE_SUBAGENT_MODEL", false},
}

// Get 取槽位值。
func (m Mapping) Get(key string) string {
	switch key {
	case "default":
		return m.Default
	case "opus":
		return m.Opus
	case "sonnet":
		return m.Sonnet
	case "haiku":
		return m.Haiku
	}
	return m.Subagent
}

// Set 设置槽位值。
func (m *Mapping) Set(key, v string) {
	switch key {
	case "default":
		m.Default = v
	case "opus":
		m.Opus = v
	case "sonnet":
		m.Sonnet = v
	case "haiku":
		m.Haiku = v
	default:
		m.Subagent = v
	}
}

// Get 按 id 找服务商。
func (s Store) Get(id string) *Provider {
	for i := range s.Providers {
		if s.Providers[i].ID == id {
			return &s.Providers[i]
		}
	}
	return nil
}

// ActiveID 当前使用的服务商 id;空串 = Claude 官方登录。
func (s Store) ActiveID() string {
	if s.Mapping.Enabled && s.Get(s.Mapping.Provider) != nil {
		return s.Mapping.Provider
	}
	return ""
}

// Blocked 说明一个服务商为什么现在不能切过去;可用返回空串。
// keys 是各服务商的 API Key(只看有没有)。
func (s Store) Blocked(p Provider, keys map[string]string) string {
	switch {
	case strings.TrimSpace(p.BaseURL) == "":
		return "未填写 Base URL"
	case strings.TrimSpace(keys[p.ID]) == "":
		return "未填写 API Key"
	case len(p.Models) == 0:
		return "模型清单为空"
	}
	return ""
}

// MappingStatus 说明映射为什么没有生效;生效返回空串。
func (s Store) MappingStatus(keys map[string]string) string {
	m := s.Mapping
	if !m.Enabled {
		return "使用 Claude 官方登录"
	}
	p := s.Get(m.Provider)
	if p == nil {
		return "没有选择服务商"
	}
	if strings.TrimSpace(p.BaseURL) == "" {
		return p.Name + " 没有填写 Base URL"
	}
	if strings.TrimSpace(keys[p.ID]) == "" {
		return p.Name + " 没有填写 API Key"
	}
	if strings.TrimSpace(m.Default) == "" {
		return "没有设置「默认」槽位的模型"
	}
	return ""
}

// EnvPair 是一个环境变量。
type EnvPair struct{ Key, Value string }

// Env 映射生效时注入 claude 进程的环境变量;未生效返回 nil。不写日志(含密钥)。
func (s Store) Env(keys map[string]string) []EnvPair {
	if s.MappingStatus(keys) != "" {
		return nil
	}
	m := s.Mapping
	p := s.Get(m.Provider)
	env := []EnvPair{{"ANTHROPIC_BASE_URL", strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")}}
	if p.AuthOrDefault() == AuthXAPIKey {
		env = append(env, EnvPair{"ANTHROPIC_API_KEY", keys[p.ID]})
	} else {
		env = append(env, EnvPair{"ANTHROPIC_AUTH_TOKEN", keys[p.ID]})
	}
	for _, sl := range Slots {
		if v := strings.TrimSpace(m.Get(sl.Key)); v != "" {
			env = append(env, EnvPair{sl.Env, v})
		}
	}
	// 兼容服务大多不认「工具搜索」产生的 tool_reference 内容块(会回 422),映射到第三方时强制关掉
	env = append(env, EnvPair{"ENABLE_TOOL_SEARCH", "false"})
	return env
}

// ManagedEnvKeys 是模型映射会注入的全部环境变量名;切换服务商时先清掉这些再写入新的,
// 其余(工作区 CLAUDE.md 开关等)不动。
func ManagedEnvKeys() []string {
	keys := []string{"ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY", "ENABLE_TOOL_SEARCH"}
	for _, sl := range Slots {
		keys = append(keys, sl.Env)
	}
	return keys
}

// Activate 切换当前服务商。provider 空 = 回到 Claude 官方;model 写进「默认」槽位。
// 换到另一家时,其余槽位里不属于新服务商清单的模型一并改成该模型,避免把旧服务商的模型名发给新服务商。
func (s *Store) Activate(provider, model string) {
	p := s.Get(provider)
	if provider == "" || p == nil {
		s.Mapping.Enabled = false
		return
	}
	same := s.Mapping.Provider == provider
	pick := strings.TrimSpace(model)
	if pick == "" && same {
		pick = strings.TrimSpace(s.Mapping.Default)
	}
	if pick == "" && len(p.Models) > 0 {
		pick = p.Models[0]
	}
	m := &s.Mapping
	m.Enabled = true
	m.Provider = provider
	m.Default = pick
	if !same {
		for _, sl := range Slots {
			v := strings.TrimSpace(m.Get(sl.Key))
			if sl.Key != "default" && v != "" && !p.HasModel(v) {
				m.Set(sl.Key, pick)
			}
		}
	}
}

// PickerGroup 是会话里模型选择器的一个服务商分组(不含 Claude 官方,官方的模型由 claude 自己报)。
type PickerGroup struct {
	ID, Name string
	BaseURL  string // 用来识别品牌 logo
	Models   []string
	Blocked  string // 不能切过去的原因;空 = 可用
	Active   bool
}

// PickerGroups 按添加顺序返回各服务商分组。
func (s Store) PickerGroups(keys map[string]string) []PickerGroup {
	active := s.ActiveID()
	out := make([]PickerGroup, 0, len(s.Providers))
	for _, p := range s.Providers {
		out = append(out, PickerGroup{ID: p.ID, Name: p.Name, BaseURL: p.BaseURL, Models: p.Models, Blocked: s.Blocked(p, keys), Active: active == p.ID})
	}
	return out
}

// NewID 为新服务商生成 id:名称里的 ASCII 字母数字 + 序号,保证不重复。
func (s Store) NewID(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	base := b.String()
	if base == "" {
		base = "provider"
	}
	id := base
	for n := 2; s.Get(id) != nil; n++ {
		id = base + "-" + strconv.Itoa(n)
	}
	return id
}

// CleanMapping 清理映射里已不在服务商清单中的槽位(取消勾选 / 删除模型之后调用)。
func (s *Store) CleanMapping() {
	p := s.Get(s.Mapping.Provider)
	if p == nil {
		return
	}
	for _, sl := range Slots {
		if v := s.Mapping.Get(sl.Key); v != "" && !p.HasModel(v) {
			if sl.Key == "default" {
				// 默认槽位不能留空:退到清单第一项(没有就留空,状态会提示)
				if len(p.Models) > 0 {
					s.Mapping.Set(sl.Key, p.Models[0])
				} else {
					s.Mapping.Set(sl.Key, "")
				}
			} else {
				s.Mapping.Set(sl.Key, "")
			}
		}
	}
}

// ProviderEnv 读磁盘上的服务商配置与密钥,返回当前映射要注入新 claude 进程的环境变量(未启用返回 nil)。
// 打开真实会话时调用;含密钥,不要写日志。
func ProviderEnv() []EnvPair {
	s, err := LoadStore()
	if err != nil || !s.Mapping.Enabled {
		return nil
	}
	keys, _ := LoadKeys()
	return s.Env(keys)
}
