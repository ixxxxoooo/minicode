package claudecfg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// ErrModelsUnsupported 表示服务商没有模型列表接口(所有候选地址都是 404 / 405)。
var ErrModelsUnsupported = errors.New("该服务商没有模型列表接口")

// ModelsEndpoints 返回「获取模型列表」要依次尝试的地址。
// 填了 ModelsURL 就只用它;否则按常见路径猜:Base URL 下的 /v1/models、/models,
// 以及站点根下的 /v1/models、/models(例如 DeepSeek 的 Anthropic 兼容地址在 /anthropic 下,模型列表却在根上)。
func (p Provider) ModelsEndpoints() []string {
	if u := strings.TrimSpace(p.ModelsURL); u != "" {
		return []string{u}
	}
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		return nil
	}
	var out []string
	add := func(s string) {
		for _, x := range out {
			if x == s {
				return
			}
		}
		out = append(out, s)
	}
	if strings.HasSuffix(base, "/v1") {
		add(base + "/models")
	} else {
		add(base + "/v1/models")
		add(base + "/models")
	}
	if u, err := url.Parse(base); err == nil && u.Host != "" && u.Path != "" && u.Path != "/" {
		root := u.Scheme + "://" + u.Host
		add(root + "/v1/models")
		add(root + "/models")
	}
	return out
}

// FetchModels 请求模型列表,返回模型 ID(去重保序)。阻塞调用,界面须放后台线程。
//
// 错误:全部候选地址都是 404 / 405 返回 ErrModelsUnsupported;401 / 403 / 网络错误等带原因文案(不含密钥)。
func FetchModels(p Provider, key string) ([]string, error) {
	defer diag.SpanSlow("claudecfg", "获取模型列表", 5*time.Second)()
	if strings.TrimSpace(p.BaseURL) == "" && strings.TrimSpace(p.ModelsURL) == "" {
		return nil, errors.New("没有填写 Base URL")
	}
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("没有填写 API Key")
	}
	cli := &http.Client{Timeout: 15 * time.Second}
	var lastUnsupported bool
	for _, u := range p.ModelsEndpoints() {
		ids, code, err := fetchModelsFrom(cli, u, p, key)
		switch {
		case err == nil:
			log.Info("获取模型列表", "provider", p.ID, "host", HostOf(u), "count", len(ids))
			return ids, nil
		case code == http.StatusNotFound || code == http.StatusMethodNotAllowed:
			lastUnsupported = true
			log.Debug("模型列表地址不存在,换下一个", "provider", p.ID, "code", code)
			continue
		default:
			log.Warn("获取模型列表失败", "provider", p.ID, "host", HostOf(u), "code", code, "err", err)
			return nil, err
		}
	}
	if lastUnsupported {
		return nil, ErrModelsUnsupported
	}
	return nil, errors.New("没有可用的模型列表地址")
}

func fetchModelsFrom(cli *http.Client, u string, p Provider, key string) ([]string, int, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("地址无效:%w", err)
	}
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Accept", "application/json")
	if p.AuthOrDefault() == AuthXAPIKey {
		req.Header.Set("x-api-key", key)
	} else {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := cli.Do(req)
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return nil, 0, errors.New("请求超时,检查地址与网络")
		}
		return nil, 0, fmt.Errorf("网络错误:%s", shortErr(err))
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode/100 != 2 {
		snippet := strings.Join(strings.Fields(string(body)), " ")
		if r := []rune(snippet); len(r) > 160 {
			snippet = string(r[:160]) + "…"
		}
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d:%s", resp.StatusCode, snippet)
	}
	ids := parseModelIDs(body)
	if len(ids) == 0 {
		return nil, resp.StatusCode, errors.New("服务商返回了空的模型列表,请手动添加")
	}
	return ids, resp.StatusCode, nil
}

// shortErr 去掉 url.Error 里带出的完整 URL(可能含查询串里的令牌),只留底层原因。
func shortErr(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) && ue.Err != nil {
		return ue.Err.Error()
	}
	return err.Error()
}

// parseModelIDs 兼容常见几种响应:{"data":[{"id"}]}、{"models":[{"id"|"name"}]}、顶层数组(字符串或对象)。
func parseModelIDs(body []byte) []string {
	var raw any
	if json.Unmarshal(body, &raw) != nil {
		return nil
	}
	var arr []any
	switch v := raw.(type) {
	case []any:
		arr = v
	case map[string]any:
		for _, k := range []string{"data", "models", "result"} {
			if a, ok := v[k].([]any); ok {
				arr = a
				break
			}
		}
	}
	var ids []string
	for _, it := range arr {
		switch m := it.(type) {
		case string:
			ids = append(ids, m)
		case map[string]any:
			for _, k := range []string{"id", "name", "model"} {
				if s, ok := m[k].(string); ok && s != "" {
					ids = append(ids, strings.TrimPrefix(s, "models/"))
					break
				}
			}
		}
	}
	return dedup(ids)
}

// MergeFetched 把获取结果并入服务商:更新 Available / FetchedAt;
// 已勾选但这次没返回的模型保留(可能只是列表不全);首次获取且没有勾选时不自动勾选,由用户挑。
func (p *Provider) MergeFetched(ids []string, now time.Time) {
	p.Available = dedup(ids)
	p.FetchedAt = now.Unix()
}

// AddManual 手动添加模型并自动勾选;已存在返回 false。
func (p *Provider) AddManual(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, m := range p.AllModels() {
		if m == id {
			return false
		}
	}
	p.Manual = append(p.Manual, id)
	p.Models = append(p.Models, id)
	return true
}

// RemoveManual 移除手动添加的模型(同时取消勾选)。
func (p *Provider) RemoveManual(id string) {
	p.Manual = without(p.Manual, id)
	p.Models = without(p.Models, id)
}

// ToggleModel 切换勾选状态;返回切换后是否已勾选。
func (p *Provider) ToggleModel(id string) bool {
	if p.HasModel(id) {
		p.Models = without(p.Models, id)
		return false
	}
	p.Models = append(p.Models, id)
	return true
}

// SelectAll 勾选给定的全部模型(保持已有勾选)。
func (p *Provider) SelectAll(ids []string) {
	p.Models = dedup(append(p.Models, ids...))
}

// SelectNone 取消勾选给定的模型。
func (p *Provider) SelectNone(ids []string) {
	for _, id := range ids {
		p.Models = without(p.Models, id)
	}
}

func without(list []string, id string) []string {
	out := list[:0:0]
	for _, s := range list {
		if s != id {
			out = append(out, s)
		}
	}
	return out
}
