package customize

import (
	"bytes"
	"context"
	"encoding/json"
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

// FetchHTTPMCPTools 直接对 http 类 MCP 服务器做一次 initialize + tools/list,读出工具名(不依赖 claude 会话)。
// 没有运行中的会话时 `claude mcp list` 只给连接状态不给工具,界面的工具区就是空的;http 服务器可以自己问。
// 请求头里的 ${变量} 按 lookup 展开。服务器回的不是 MCP 数据(如登录页)时返回带原话的错误。
func FetchHTTPMCPTools(url string, headers map[string]string, lookup func(string) string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	sessionID := ""
	call := func(id int, method string, params string) (map[string]any, error) {
		payload := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":%q,"params":%s}`, id, method, params)
		if id == 0 { // 通知:没有 id
			payload = fmt.Sprintf(`{"jsonrpc":"2.0","method":%q}`, method)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if sessionID != "" {
			req.Header.Set("Mcp-Session-Id", sessionID)
		}
		for k, v := range headers {
			req.Header.Set(k, os.Expand(v, lookup))
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
			sessionID = sid
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		if id == 0 {
			return nil, nil
		}
		text := string(raw)
		if strings.Contains(resp.Header.Get("Content-Type"), "event-stream") {
			for _, ln := range strings.Split(text, "\n") {
				if d, ok := strings.CutPrefix(strings.TrimSpace(ln), "data:"); ok && strings.Contains(d, `"jsonrpc"`) {
					text = strings.TrimSpace(d)
					break
				}
			}
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(text), &out); err != nil {
			snip := strings.Join(strings.Fields(text), " ")
			if r := []rune(snip); len(r) > 100 {
				snip = string(r[:100]) + "…"
			}
			return nil, fmt.Errorf("服务器回复(HTTP %d)不是 MCP 数据:%s", resp.StatusCode, snip)
		}
		if e, ok := out["error"].(map[string]any); ok {
			return nil, fmt.Errorf("服务器报错:%v", e["message"])
		}
		return out, nil
	}
	if _, err := call(1, "initialize", `{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"minicode","version":"1"}}`); err != nil {
		return nil, err
	}
	_, _ = call(0, "notifications/initialized", "")
	var names []string
	cursor := ""
	for page := 0; page < 20; page++ {
		params := `{}`
		if cursor != "" {
			params = fmt.Sprintf(`{"cursor":%q}`, cursor)
		}
		out, err := call(2+page, "tools/list", params)
		if err != nil {
			return names, err
		}
		res, _ := out["result"].(map[string]any)
		list, _ := res["tools"].([]any)
		for _, x := range list {
			if m, ok := x.(map[string]any); ok {
				if n, ok := m["name"].(string); ok && n != "" {
					names = append(names, n)
				}
			}
		}
		cursor, _ = res["nextCursor"].(string)
		if cursor == "" {
			break
		}
	}
	return names, nil
}
