package claudecfg

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ixxxxoooo/minicode/internal/diag"
)

// ProbeKind 连通性结果类别。
type ProbeKind string

const (
	ProbeOK        ProbeKind = "ok"
	ProbeAuth      ProbeKind = "auth"
	ProbeNotFound  ProbeKind = "not_found"
	ProbeNetwork   ProbeKind = "network"
	ProbeTimeout   ProbeKind = "timeout"
	ProbeNoKey     ProbeKind = "no_key"
	ProbeNoBaseURL ProbeKind = "no_url"
)

// ProbeResult 是一次服务商连通性测试的结果(不含密钥)。
type ProbeResult struct {
	OK     bool
	Kind   ProbeKind
	Ms     int64
	Detail string // 界面 / toast 用
}

// ProbeProvider 用 API Key 请求模型列表接口(与「获取模型列表」同源),区分鉴权 / 404 / 网络 / 超时。
func ProbeProvider(p Provider, key string) ProbeResult {
	defer diag.SpanSlow("claudecfg", "连通性探测", 8*time.Second)()
	if strings.TrimSpace(p.BaseURL) == "" && strings.TrimSpace(p.ModelsURL) == "" {
		return ProbeResult{Kind: ProbeNoBaseURL, Detail: "请填写 Base URL"}
	}
	if strings.TrimSpace(key) == "" {
		return ProbeResult{Kind: ProbeNoKey, Detail: "请填写 API Key 后再测试"}
	}
	cli := &http.Client{Timeout: 8 * time.Second}
	t0 := time.Now()
	endpoints := p.ModelsEndpoints()
	if len(endpoints) == 0 {
		return ProbeResult{Kind: ProbeNoBaseURL, Detail: "没有可用的探测地址"}
	}
	var saw404, sawAuth bool
	var lastNet string
	for _, u := range endpoints {
		code, err := probeHeadGET(cli, u, p, key)
		ms := time.Since(t0).Milliseconds()
		switch {
		case err == nil:
			log.Info("连通性探测成功", "provider", p.ID, "host", HostOf(u), "ms", ms)
			return ProbeResult{OK: true, Kind: ProbeOK, Ms: ms, Detail: fmt.Sprintf("%s · %d ms", ProvOK, ms)}
		case code == http.StatusUnauthorized || code == http.StatusForbidden:
			sawAuth = true
			continue
		case code == http.StatusNotFound || code == http.StatusMethodNotAllowed:
			saw404 = true
			continue
		default:
			if pe := classifyNetErr(err); pe != "" {
				lastNet = pe
				continue
			}
			if code > 0 {
				return ProbeResult{Kind: ProbeNetwork, Ms: ms, Detail: fmt.Sprintf("HTTP %d,检查地址或服务状态", code)}
			}
			lastNet = shortErr(err)
		}
	}
	ms := time.Since(t0).Milliseconds()
	switch {
	case sawAuth:
		return ProbeResult{Kind: ProbeAuth, Ms: ms, Detail: "密钥无效或已过期(401),到 设置 → 服务商 更新 API Key"}
	case saw404:
		return ProbeResult{Kind: ProbeNotFound, Ms: ms, Detail: "没有模型列表接口(404),仍可在下方手动添加模型 ID"}
	case lastNet != "":
		if strings.Contains(lastNet, "timeout") || strings.Contains(lastNet, "Timeout") {
			return ProbeResult{Kind: ProbeTimeout, Ms: ms, Detail: "请求超时,检查网络、代理与 Base URL"}
		}
		return ProbeResult{Kind: ProbeNetwork, Ms: ms, Detail: "网络不可达:" + lastNet}
	default:
		return ProbeResult{Kind: ProbeNetwork, Ms: ms, Detail: "连接失败,检查地址与网络"}
	}
}

func probeHeadGET(cli *http.Client, u string, p Provider, key string) (int, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	if p.AuthOrDefault() == AuthXAPIKey {
		req.Header.Set("x-api-key", key)
	} else {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := cli.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode/100 == 2 {
		return resp.StatusCode, nil
	}
	return resp.StatusCode, fmt.Errorf("http %d", resp.StatusCode)
}

func classifyNetErr(err error) string {
	if err == nil {
		return ""
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return "timeout"
	}
	return shortErr(err)
}

// FormatProbe 把探测结果写成服务商列表上的状态文案。
func FormatProbe(r ProbeResult) string {
	if r.Detail != "" {
		return r.Detail
	}
	if r.OK {
		return FormatTestResult(true, r.Ms)
	}
	return ProvFail
}
