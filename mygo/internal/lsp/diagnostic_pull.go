package lsp

import (
	"encoding/json"
	"fmt"
	"time"
)

// DiagnosticCaps 从 initialize 响应里的 diagnosticProvider 解析出的拉取能力。
type DiagnosticCaps struct {
	WorkspacePull bool   // workspace/diagnostic
	DocumentPull  bool   // textDocument/diagnostic(无 workspace 时)
	Identifier    string // 服务器标识(增量 previousResultId 用)
}

// ParseDiagnosticCaps 解析 diagnosticProvider(布尔 / 空对象 / 完整选项都认)。
func ParseDiagnosticCaps(raw json.RawMessage) DiagnosticCaps {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "false" {
		return DiagnosticCaps{}
	}
	var caps DiagnosticCaps
	if string(raw) == "true" {
		caps.DocumentPull = true
		return caps
	}
	var obj struct {
		WorkspaceDiagnostics  bool   `json:"workspaceDiagnostics"`
		InterFileDependencies bool   `json:"interFileDependencies"`
		Identifier            string `json:"identifier"`
	}
	if json.Unmarshal(raw, &obj) != nil {
		return DiagnosticCaps{}
	}
	caps.Identifier = obj.Identifier
	if obj.WorkspaceDiagnostics {
		caps.WorkspacePull = true
	} else {
		caps.DocumentPull = true
	}
	return caps
}

// WorkspacePullResult 是一次 workspace/diagnostic 的合并结果。
type WorkspacePullResult struct {
	ByURI        map[string][]Diagnostic
	ResultID     string
	FilesChecked int
}

// DocumentPullResult 是一次 textDocument/diagnostic 的结果。
type DocumentPullResult struct {
	Items    []Diagnostic
	ResultID string
	Kind     string
}

type wsDiagItem struct {
	URI   string `json:"uri"`
	Items []struct {
		Kind          string       `json:"kind"`
		ResultID      string       `json:"resultId"`
		ResultIDSnake string       `json:"result_id"`
		Items         []Diagnostic `json:"items"`
	} `json:"items"`
}

func diagnosticsFromDocReports(reports []struct {
	Kind          string       `json:"kind"`
	ResultID      string       `json:"resultId"`
	ResultIDSnake string       `json:"result_id"`
	Items         []Diagnostic `json:"items"`
}) []Diagnostic {
	var out []Diagnostic
	for _, r := range reports {
		if r.Kind == "unchanged" {
			continue
		}
		out = append(out, r.Items...)
	}
	return out
}

func mergeWorkspacePull(acc, chunk json.RawMessage) json.RawMessage {
	if len(acc) == 0 {
		return chunk
	}
	var a, b struct {
		Items []json.RawMessage `json:"items"`
	}
	_ = json.Unmarshal(acc, &a)
	_ = json.Unmarshal(chunk, &b)
	merged := struct {
		Items []json.RawMessage `json:"items"`
	}{Items: append(a.Items, b.Items...)}
	out, _ := json.Marshal(merged)
	return out
}

func parseWorkspacePull(raw json.RawMessage) WorkspacePullResult {
	res := WorkspacePullResult{ByURI: map[string][]Diagnostic{}}
	if len(raw) == 0 || string(raw) == "null" {
		return res
	}
	// 可能是 {items:[{kind:full,items:[...]}]} 或直接是数组
	var top struct {
		Items []struct {
			Kind  string          `json:"kind"`
			Items json.RawMessage `json:"items"`
		} `json:"items"`
		ResultID string `json:"resultId"`
	}
	if json.Unmarshal(raw, &top) == nil && len(top.Items) > 0 {
		res.ResultID = top.ResultID
		for _, block := range top.Items {
			if block.Kind == "unchanged" {
				continue
			}
			var docs []wsDiagItem
			if json.Unmarshal(block.Items, &docs) != nil {
				var one wsDiagItem
				if json.Unmarshal(block.Items, &one) == nil && one.URI != "" {
					docs = []wsDiagItem{one}
				}
			}
			for _, d := range docs {
				if d.URI == "" {
					continue
				}
				ds := diagnosticsFromDocReports(d.Items)
				if len(ds) > 0 {
					res.ByURI[d.URI] = append(res.ByURI[d.URI], ds...)
				}
				res.FilesChecked++
			}
		}
		return res
	}
	// 单块 full
	var docs []wsDiagItem
	if json.Unmarshal(raw, &docs) == nil {
		for _, d := range docs {
			if d.URI == "" {
				continue
			}
			ds := diagnosticsFromDocReports(d.Items)
			res.ByURI[d.URI] = ds
			res.FilesChecked++
		}
	}
	return res
}

// pullMu / pullParts 在客户端上挂 partialResultToken 的流式片段(不阻塞读线程)。
func (c *Client) registerPartial(token int64) {
	c.partMu.Lock()
	if c.pullParts == nil {
		c.pullParts = map[int64][]json.RawMessage{}
	}
	c.pullParts[token] = nil
	c.partMu.Unlock()
}

func (c *Client) takePartials(token int64) []json.RawMessage {
	c.partMu.Lock()
	chunks := c.pullParts[token]
	delete(c.pullParts, token)
	c.partMu.Unlock()
	return chunks
}

func (c *Client) appendPartial(token int64, chunk json.RawMessage) {
	c.partMu.Lock()
	if c.pullParts != nil {
		c.pullParts[token] = append(c.pullParts[token], chunk)
	}
	c.partMu.Unlock()
}

// WorkspaceDiagnostic 调用 workspace/diagnostic;支持 partialResultToken 与 previousResultId。
func (c *Client) WorkspaceDiagnostic(caps DiagnosticCaps, previousResultID string, timeout time.Duration) (WorkspacePullResult, error) {
	token := c.next.Add(1)
	c.registerPartial(token)
	params := map[string]any{
		"identifier":         caps.Identifier,
		"partialResultToken": token,
	}
	if previousResultID != "" {
		params["previousResultId"] = previousResultID
	}
	var raw json.RawMessage
	err := c.Call("workspace/diagnostic", params, &raw, timeout)
	chunks := c.takePartials(token)
	for _, ch := range chunks {
		raw = mergeWorkspacePull(raw, ch)
	}
	if err != nil {
		return WorkspacePullResult{}, err
	}
	return parseWorkspacePull(raw), nil
}

// DocumentDiagnostic 对单个 URI 调用 textDocument/diagnostic。
func (c *Client) DocumentDiagnostic(uri, previousResultID string, timeout time.Duration) (DocumentPullResult, error) {
	token := c.next.Add(1)
	c.registerPartial(token)
	params := map[string]any{
		"textDocument":       map[string]any{"uri": uri},
		"partialResultToken": token,
	}
	if previousResultID != "" {
		params["previousResultId"] = previousResultID
	}
	var raw json.RawMessage
	err := c.Call("textDocument/diagnostic", params, &raw, timeout)
	chunks := c.takePartials(token)
	for _, ch := range chunks {
		raw = ch // 文档级一般是整块替换
	}
	if err != nil {
		return DocumentPullResult{}, err
	}
	var res struct {
		Kind             string       `json:"kind"`
		ResultID         string       `json:"resultId"`
		Items            []Diagnostic `json:"items"`
		RelatedDocuments []any        `json:"relatedDocuments"`
	}
	if len(raw) == 0 || string(raw) == "null" {
		return DocumentPullResult{}, nil
	}
	if json.Unmarshal(raw, &res) != nil {
		return DocumentPullResult{}, fmt.Errorf("无法解析 textDocument/diagnostic 响应")
	}
	if res.Kind == "unchanged" {
		return DocumentPullResult{Kind: res.Kind, ResultID: res.ResultID}, nil
	}
	return DocumentPullResult{Kind: res.Kind, ResultID: res.ResultID, Items: res.Items}, nil
}
