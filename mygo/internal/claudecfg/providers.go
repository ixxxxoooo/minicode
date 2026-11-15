package claudecfg

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// Provider 是第三方模型服务商。
type Provider struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	Active  bool   `json:"active"`
	Status  string `json:"status"` // 未测 / 测试中 / 已连通 / 失败
}

// ProvTestNone / Testing / OK / Fail 连通性四态。
const (
	ProvTestNone   = ""
	ProvTestIng    = "测试中"
	ProvOK         = "已连通"
	ProvFail       = "失败"
	ProvUnverified = "未验证"
)

func providersPath() string {
	return filepath.Join(MiniHome(), "providers.json")
}

func keysPath() string {
	return filepath.Join(MiniHome(), "provider-keys.json")
}

// LoadProviders 读服务商列表;不存在返回演示默认。
func LoadProviders() ([]Provider, error) {
	path := providersPath()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaultProviders(), nil
	}
	if err != nil {
		return nil, err
	}
	var list []Provider
	if err := json.Unmarshal(b, &list); err != nil {
		return defaultProviders(), nil
	}
	if len(list) == 0 {
		return defaultProviders(), nil
	}
	return list, nil
}

func defaultProviders() []Provider {
	return []Provider{
		{ID: "anthropic", Name: "Anthropic 官方", BaseURL: "https://api.anthropic.com", Active: true, Status: ProvOK},
		{ID: "deepseek", Name: "DeepSeek", BaseURL: "https://api.deepseek.com/anthropic", Status: ProvOK},
		{ID: "custom", Name: "自建网关", BaseURL: "https://llm.example.com", Status: ProvUnverified},
	}
}

// SaveProviders 落盘 providers.json。
func SaveProviders(list []Provider) error {
	b, err := json.MarshalIndent(list, "", "  ")
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

// SetProviderKey 把密钥写入 0600 文件(不落 settings.json、不写日志值)。
func SetProviderKey(id, key string) error {
	keys, _ := loadKeys()
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

func loadKeys() (map[string]string, error) {
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

// HasProviderKey 报告是否已存密钥(不读值)。
func HasProviderKey(id string) bool {
	m, err := loadKeys()
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
	if url[:4] != "http" {
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
