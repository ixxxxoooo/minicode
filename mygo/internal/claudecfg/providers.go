package claudecfg

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// 鉴权头:多数 Anthropic 兼容服务用 `Authorization: Bearer`(注入 ANTHROPIC_AUTH_TOKEN),
// Anthropic 官方用 `x-api-key`(注入 ANTHROPIC_API_KEY)。
const (
	AuthBearer  = "bearer"
	AuthXAPIKey = "x-api-key"
)

// Provider 是第三方模型服务商。
type Provider struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	// Auth 鉴权头,空 = bearer。
	Auth string `json:"auth,omitempty"`
	// ModelsURL 模型列表接口的完整地址;空 = 自动尝试 Base URL 下的 /v1/models 等常见路径。
	ModelsURL string `json:"modelsUrl,omitempty"`
	// Models 是用户勾选的模型:只有它们会出现在模型映射的下拉与会话的模型选择器里。
	Models []string `json:"models,omitempty"`
	// Available 是上次「获取模型列表」拿到的完整列表(缓存,供勾选)。
	Available []string `json:"available,omitempty"`
	// Manual 是手动添加的模型 ID(界面上标「手动」,可单独移除)。
	Manual []string `json:"manual,omitempty"`
	// FetchedAt 上次获取模型列表的时间(Unix 秒),0 = 从未获取。
	FetchedAt int64  `json:"fetchedAt,omitempty"`
	Status    string `json:"status"` // 未测 / 测试中 / 已连通 / 失败
}

// AuthOrDefault 返回有效的鉴权头名。
func (p Provider) AuthOrDefault() string {
	if p.Auth == AuthXAPIKey {
		return AuthXAPIKey
	}
	return AuthBearer
}

// AllModels 返回「可勾选」的全部模型:获取到的 + 手动添加的,去重保序。
func (p Provider) AllModels() []string {
	return dedup(append(append([]string{}, p.Available...), p.Manual...))
}

// HasModel 报告模型是否已勾选。
func (p Provider) HasModel(id string) bool {
	for _, m := range p.Models {
		if m == id {
			return true
		}
	}
	return false
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// ProvTestNone / Testing / OK / Fail 连通性四态。
const (
	ProvTestNone   = ""
	ProvTestIng    = "测试中"
	ProvOK         = "已连通"
	ProvFail       = "失败"
	ProvUnverified = "未验证"
)

// Preset 是「添加服务商」菜单里的预设。
type Preset struct {
	Name, BaseURL, Auth string
	Models              []string
}

// Presets 预设服务商(模型名会随服务商更新,可用「获取模型列表」刷新)。
var Presets = []Preset{
	{"DeepSeek", "https://api.deepseek.com/anthropic", AuthBearer, []string{"deepseek-chat", "deepseek-reasoner"}},
	{"智谱 GLM", "https://open.bigmodel.cn/api/anthropic", AuthBearer, []string{"glm-4.6", "glm-4.5-air"}},
	{"Kimi", "https://api.moonshot.cn/anthropic", AuthBearer, []string{"kimi-k2-turbo-preview"}},
	{"MiniMax", "https://api.minimaxi.com/anthropic", AuthBearer, []string{"MiniMax-M2"}},
	{"阿里百炼 Qwen", "https://dashscope.aliyuncs.com/apps/anthropic", AuthBearer, []string{"qwen3-coder-plus"}},
	{"Anthropic API", "https://api.anthropic.com", AuthXAPIKey, []string{"claude-opus-5-5", "claude-sonnet-5-5", "claude-haiku-5-5"}},
}

// PickOfficial 是会话弹层里 Claude 官方分组在 picks 里用的键(与 ActiveID 空串区分)。
const PickOfficial = "official"

// Store 是 providers.json 的内容:服务商列表 + 模型映射。
type Store struct {
	Providers []Provider        `json:"providers"`
	Mapping   Mapping           `json:"mapping"`
	Picks     map[string]string `json:"picks,omitempty"` // 各服务商在运行配置弹层里上次选的模型
}

func providersPath() string {
	return filepath.Join(MiniHome(), "providers.json")
}

func keysPath() string {
	return filepath.Join(MiniHome(), "provider-keys.json")
}

// LoadStore 读服务商与映射;文件不存在返回空配置,旧格式(纯数组)自动迁移。
func LoadStore() (Store, error) {
	b, err := os.ReadFile(providersPath())
	if os.IsNotExist(err) {
		return Store{}, nil
	}
	if err != nil {
		return Store{}, err
	}
	var s Store
	if err := json.Unmarshal(b, &s); err != nil {
		// 旧版是纯数组
		var list []Provider
		if err2 := json.Unmarshal(b, &list); err2 != nil {
			log.Warn("providers.json 无法解析,按空配置处理", "err", err)
			return Store{}, nil
		}
		s.Providers = list
	}
	fillLegacyBaseURL(b, &s)
	return s, nil
}

// fillLegacyBaseURL 兼容 Rust 旧版写的 snake_case 字段 base_url:新字段 baseUrl 为空时从旧字段补上。
func fillLegacyBaseURL(raw []byte, s *Store) {
	var old struct {
		Providers []struct {
			ID      string `json:"id"`
			BaseURL string `json:"base_url"`
		} `json:"providers"`
	}
	if json.Unmarshal(raw, &old) != nil {
		return
	}
	for _, o := range old.Providers {
		if o.BaseURL == "" {
			continue
		}
		for i := range s.Providers {
			if s.Providers[i].ID == o.ID && s.Providers[i].BaseURL == "" {
				s.Providers[i].BaseURL = o.BaseURL
			}
		}
	}
}

// SaveStore 原子落盘 providers.json(0600)。
func SaveStore(s Store) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	path := providersPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadProviders 读服务商列表(不含映射)。
func LoadProviders() ([]Provider, error) {
	s, err := LoadStore()
	return s.Providers, err
}

// SaveProviders 只更新服务商列表,保留已有的模型映射。
func SaveProviders(list []Provider) error {
	s, _ := LoadStore()
	s.Providers = list
	return SaveStore(s)
}

// SetProviderKey 把密钥写入 0600 文件(不落 settings.json、不写日志值)。
func SetProviderKey(id, key string) error {
	keys, _ := LoadKeys()
	if keys == nil {
		keys = map[string]string{}
	}
	if key == "" {
		delete(keys, id)
	} else {
		keys[id] = key
	}
	b, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}
	path := keysPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	log.Info("更新服务商密钥", "id", id)
	return os.Rename(tmp, path)
}

// LoadKeys 读全部服务商密钥(id → 密钥)。只在需要请求或注入环境变量时用,不要显示、不要写日志。
func LoadKeys() (map[string]string, error) {
	b, err := os.ReadFile(keysPath())
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]string{}, nil
	}
	return m, nil
}

// ProviderKey 取单个服务商的密钥(不存在返回空)。
func ProviderKey(id string) string {
	m, _ := LoadKeys()
	return m[id]
}

// HasProviderKey 报告是否已存密钥(不读值)。
func HasProviderKey(id string) bool {
	m, err := LoadKeys()
	return err == nil && m[id] != ""
}

// TestProvider 探测 base URL 是否可达(HEAD,5s)。
func TestProvider(base string) (bool, int64) {
	defer diag.SpanSlow("claudecfg", "连通性测试", 3*time.Second)()
	cli := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodHead, base, nil)
	if err != nil {
		return false, 0
	}
	t0 := time.Now()
	resp, err := cli.Do(req)
	ms := time.Since(t0).Milliseconds()
	if err != nil {
		log.Warn("连通性失败", "url", base, "err", err)
		return false, ms
	}
	_ = resp.Body.Close()
	ok := resp.StatusCode < 500
	log.Info("连通性测试", "url", base, "code", resp.StatusCode, "ms", ms)
	return ok, ms
}

// PingHost 对主机名做连通测试(api.xxx.com)。
func PingHost(host string) (bool, int64) {
	if host == "" {
		return false, 0
	}
	url := host
	if len(url) < 4 || url[:4] != "http" {
		url = "https://" + host
	}
	return TestProvider(url)
}

// HostOf 从 base URL 取主机用于展示。
func HostOf(base string) string {
	if base == "" {
		return ""
	}
	req, err := http.NewRequest(http.MethodGet, base, nil)
	if err != nil {
		return base
	}
	return req.URL.Host
}

// FormatTestResult 格式化测试结果文案。
func FormatTestResult(ok bool, ms int64) string {
	if ms == 0 && !ok {
		return ProvFail
	}
	if ok {
		return fmt.Sprintf("%s · %d ms", ProvOK, ms)
	}
	return ProvFail
}
