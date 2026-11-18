package customize

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ProbeHTTPMCP 对 http 类 MCP 服务器发一次 initialize 请求,把服务器的原话告诉人。
// claude 只会说「Unexpected content type」,看不出是登录失效(服务器回「您尚未登录」)、地址错还是网关报错;
// 这里用同样的地址与请求头(请求头里的 ${变量} 按 lookup 展开)请求一次,返回「HTTP 状态 · 内容类型 · 正文开头」。
// 请求头的值不会出现在返回里。
func ProbeHTTPMCP(url string, headers map[string]string, lookup func(string) string) string {
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"minicode-probe","version":"1"}}}`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return "地址无效:" + err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	missing := map[string]bool{}
	for k, v := range headers {
		req.Header.Set(k, os.Expand(v, func(name string) string {
			val := lookup(name)
			if val == "" {
				missing[name] = true
			}
			return val
		}))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "请求失败:" + err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
	snippet := strings.Join(strings.Fields(string(bytes.TrimSpace(b))), " ")
	if r := []rune(snippet); len(r) > 120 {
		snippet = string(r[:120]) + "…"
	}
	out := fmt.Sprintf("HTTP %d · %s · %s", resp.StatusCode, resp.Header.Get("Content-Type"), snippet)
	if len(missing) > 0 {
		var ns []string
		for n := range missing {
			ns = append(ns, n)
		}
		out += "(请求头里引用的 " + strings.Join(ns, "、") + " 没有值)"
	}
	return out
}
