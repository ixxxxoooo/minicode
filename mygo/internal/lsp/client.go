// Package lsp 是语言服务器协议(LSP)客户端内核:Content-Length 帧、请求 / 响应配对、
// 通知分发、服务端请求自动应答,以及常用请求(悬停、定义、引用、补全、符号)的类型与解析。
// 与界面无关;界面在 internal/app/lsp_app.go。
package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ───────── 协议类型 ─────────

// Position 是文档里的位置:行与列都从 0 起,列按 UTF-16 码元计(LSP 的规定)。
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Diagnostic 严重程度:1 错误 / 2 警告 / 3 信息 / 4 提示。
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Code     any    `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

type PublishDiagnostics struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// CompletionItem 是补全项(只取用得到的字段)。
type CompletionItem struct {
	Label            string `json:"label"`
	Kind             int    `json:"kind"`
	Detail           string `json:"detail"`
	Documentation    any    `json:"documentation"`
	InsertText       string `json:"insertText"`
	InsertTextFormat int    `json:"insertTextFormat"` // 2 = 代码片段
	FilterText       string `json:"filterText"`
	SortText         string `json:"sortText"`
	TextEdit         *struct {
		NewText string `json:"newText"`
	} `json:"textEdit"`
}

// Doc 取补全项文档的纯文本(字符串或 MarkupContent)。
func (c CompletionItem) Doc() string { return markupText(c.Documentation) }

// DocSymbol 是文档符号(层级的 DocumentSymbol 或平铺的 SymbolInformation 统一成这个)。
type DocSymbol struct {
	Name     string
	Kind     int
	Detail   string
	Range    Range // 选中范围(名字所在)
	Children []DocSymbol
}

// ───────── 帧 ─────────

// WriteFrame 写一条带 Content-Length 头的消息。
func WriteFrame(w io.Writer, body []byte) error {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Content-Length: %d\r\n\r\n", len(body))
	buf.Write(body)
	_, err := w.Write(buf.Bytes())
	return err
}

// ReadFrame 读一条消息;流结束返回 io.EOF。
func ReadFrame(r *bufio.Reader) ([]byte, error) {
	n := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if k, v, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(k), "Content-Length") {
			n, err = strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return nil, fmt.Errorf("Content-Length 不是数字: %q", v)
			}
		}
	}
	if n < 0 {
		return nil, errors.New("缺少 Content-Length")
	}
	if n > 64<<20 {
		return nil, fmt.Errorf("消息过大: %d 字节", n)
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

// ───────── 客户端 ─────────

type rpcMsg struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError 是服务器返回的错误。
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("语言服务器错误 %d: %s", e.Code, e.Message)
}

// Handlers 是服务器主动推送的事件回调(在读取线程里调用,别阻塞)。
type Handlers struct {
	Diagnostics func(PublishDiagnostics)
	Log         func(level int, msg string) // window/logMessage、showMessage
	Progress    func(title, msg string, done bool)
}

// Client 是一个已连接的语言服务器。
type Client struct {
	w   io.Writer
	wmu sync.Mutex
	h   Handlers

	next    atomic.Int64
	mu      sync.Mutex
	pending map[int64]chan rpcMsg
	closed  bool
	done    chan struct{}

	// 初始化后得到的能力(只记录有没有)
	Caps map[string]bool
	// 服务器名(initialize 响应里的 serverInfo.name)
	ServerName string
}

// NewClient 在读写流上建一个客户端并开始读取;r 结束(服务器退出)后所有等待中的请求返回错误。
func NewClient(r io.Reader, w io.Writer, h Handlers) *Client {
	c := &Client{w: w, h: h, pending: map[int64]chan rpcMsg{}, done: make(chan struct{}), Caps: map[string]bool{}}
	go c.readLoop(bufio.NewReaderSize(r, 1<<20))
	return c
}

// Done 在服务器连接结束时关闭。
func (c *Client) Done() <-chan struct{} { return c.done }

func (c *Client) readLoop(r *bufio.Reader) {
	defer func() {
		c.mu.Lock()
		c.closed = true
		for id, ch := range c.pending {
			ch <- rpcMsg{Error: &RPCError{Code: -32000, Message: "语言服务器已退出"}}
			delete(c.pending, id)
		}
		c.mu.Unlock()
		close(c.done)
	}()
	for {
		body, err := ReadFrame(r)
		if err != nil {
			return
		}
		var m rpcMsg
		if json.Unmarshal(body, &m) != nil {
			continue
		}
		switch {
		case m.Method != "" && len(m.ID) > 0: // 服务器发来的请求:一律应答,免得它卡住
			go c.answer(m) // 异步:读取线程不能因为写出而阻塞,否则和同样在写的服务器互相等死
		case m.Method != "":
			c.notify(m)
		case len(m.ID) > 0: // 响应
			id, err := strconv.ParseInt(string(m.ID), 10, 64)
			if err != nil {
				continue
			}
			c.mu.Lock()
			ch := c.pending[id]
			delete(c.pending, id)
			c.mu.Unlock()
			if ch != nil {
				ch <- m
			}
		}
	}
}

// answer 应答服务器的请求:配置返回空、注册 / 进度创建返回成功。
func (c *Client) answer(m rpcMsg) {
	var result any
	switch m.Method {
	case "workspace/configuration":
		var p struct {
			Items []json.RawMessage `json:"items"`
		}
		_ = json.Unmarshal(m.Params, &p)
		out := make([]any, len(p.Items))
		result = out
	case "workspace/workspaceFolders":
		result = nil
	}
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(m.ID), "result": result})
	c.send(b)
}

func (c *Client) notify(m rpcMsg) {
	switch m.Method {
	case "textDocument/publishDiagnostics":
		var p PublishDiagnostics
		if json.Unmarshal(m.Params, &p) == nil && c.h.Diagnostics != nil {
			c.h.Diagnostics(p)
		}
	case "window/logMessage", "window/showMessage":
		var p struct {
			Type    int    `json:"type"`
			Message string `json:"message"`
		}
		if json.Unmarshal(m.Params, &p) == nil && c.h.Log != nil {
			c.h.Log(p.Type, p.Message)
		}
	case "$/progress":
		var p struct {
			Value struct {
				Kind    string `json:"kind"`
				Title   string `json:"title"`
				Message string `json:"message"`
			} `json:"value"`
		}
		if json.Unmarshal(m.Params, &p) == nil && c.h.Progress != nil {
			c.h.Progress(p.Value.Title, p.Value.Message, p.Value.Kind == "end")
		}
	}
}

func (c *Client) send(b []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	return WriteFrame(c.w, b)
}

// Call 发请求并等响应(最多 timeout)。result 非 nil 时解码进去。
func (c *Client) Call(method string, params, result any, timeout time.Duration) error {
	id := c.next.Add(1)
	ch := make(chan rpcMsg, 1)
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errors.New("语言服务器已退出")
	}
	c.pending[id] = ch
	c.mu.Unlock()
	b, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	if err != nil {
		return err
	}
	if err := c.send(b); err != nil {
		return err
	}
	select {
	case m := <-ch:
		if m.Error != nil {
			return m.Error
		}
		if result != nil && len(m.Result) > 0 && string(m.Result) != "null" {
			return json.Unmarshal(m.Result, result)
		}
		return nil
	case <-time.After(timeout):
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		_ = c.Notify("$/cancelRequest", map[string]any{"id": id})
		return fmt.Errorf("%s 超时(%v)", method, timeout)
	}
}

// Notify 发通知(没有响应)。
func (c *Client) Notify(method string, params any) error {
	b, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
	if err != nil {
		return err
	}
	return c.send(b)
}

// Initialize 完成握手:发 initialize(带我们支持的能力),再发 initialized。
func (c *Client) Initialize(rootURI, rootName string, timeout time.Duration) error {
	params := map[string]any{
		"processId":        nil,
		"clientInfo":       map[string]any{"name": "MiniCode"},
		"rootUri":          rootURI,
		"workspaceFolders": []map[string]any{{"uri": rootURI, "name": rootName}},
		"capabilities": map[string]any{
			"general": map[string]any{"positionEncodings": []string{"utf-16"}},
			"window":  map[string]any{"workDoneProgress": true},
			"workspace": map[string]any{
				"configuration": true, "workspaceFolders": true,
			},
			"textDocument": map[string]any{
				"synchronization":    map[string]any{"didSave": true},
				"publishDiagnostics": map[string]any{"relatedInformation": false},
				"hover":              map[string]any{"contentFormat": []string{"markdown", "plaintext"}},
				"definition":         map[string]any{"linkSupport": false},
				"references":         map[string]any{},
				"documentSymbol":     map[string]any{"hierarchicalDocumentSymbolSupport": true},
				"inlayHint":          map[string]any{},
				"completion": map[string]any{"completionItem": map[string]any{
					"snippetSupport": false, "documentationFormat": []string{"markdown", "plaintext"},
				}},
			},
		},
	}
	var res struct {
		Capabilities map[string]json.RawMessage `json:"capabilities"`
		ServerInfo   struct {
			Name string `json:"name"`
		} `json:"serverInfo"`
	}
	if err := c.Call("initialize", params, &res, timeout); err != nil {
		return err
	}
	for k, v := range res.Capabilities {
		c.Caps[k] = string(v) != "false" && string(v) != "null"
	}
	c.ServerName = res.ServerInfo.Name
	return c.Notify("initialized", map[string]any{})
}

// Shutdown 礼貌地关闭:shutdown 请求 + exit 通知。
func (c *Client) Shutdown() {
	_ = c.Call("shutdown", nil, nil, 2*time.Second)
	_ = c.Notify("exit", nil)
}

// ───────── 文档同步 ─────────

func (c *Client) DidOpen(uri, langID string, version int, text string) error {
	return c.Notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "languageId": langID, "version": version, "text": text}})
}

// DidChange 整篇替换式同步(简单可靠;文件很大时也只在停止输入后发一次)。
func (c *Client) DidChange(uri string, version int, text string) error {
	return c.Notify("textDocument/didChange", map[string]any{
		"textDocument":   map[string]any{"uri": uri, "version": version},
		"contentChanges": []map[string]any{{"text": text}}})
}

func (c *Client) DidSave(uri string) error {
	return c.Notify("textDocument/didSave", map[string]any{"textDocument": map[string]any{"uri": uri}})
}

func (c *Client) DidClose(uri string) error {
	return c.Notify("textDocument/didClose", map[string]any{"textDocument": map[string]any{"uri": uri}})
}

// ───────── 请求 ─────────

func docPos(uri string, p Position) map[string]any {
	return map[string]any{"textDocument": map[string]any{"uri": uri}, "position": p}
}

// Hover 返回悬停内容的纯文本 / Markdown;没有内容返回空串。
func (c *Client) Hover(uri string, p Position) (string, error) {
	var raw struct {
		Contents json.RawMessage `json:"contents"`
	}
	if err := c.Call("textDocument/hover", docPos(uri, p), &raw, 8*time.Second); err != nil {
		return "", err
	}
	return hoverText(raw.Contents), nil
}

func hoverText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) == nil {
		var parts []string
		for _, a := range arr {
			if s := hoverText(a); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "\n\n")
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case map[string]any:
		val, _ := x["value"].(string)
		if lang, ok := x["language"].(string); ok && val != "" { // MarkedString {language, value}
			return "```" + lang + "\n" + val + "\n```"
		}
		return val
	}
	return ""
}

func markupText(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case map[string]any:
		s, _ := x["value"].(string)
		return s
	}
	return ""
}

// locations 解析 Location | Location[] | LocationLink[]。
func locations(raw json.RawMessage) []Location {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var one Location
	if json.Unmarshal(raw, &one) == nil && one.URI != "" {
		return []Location{one}
	}
	var many []struct {
		Location
		TargetURI            string `json:"targetUri"`
		TargetSelectionRange Range  `json:"targetSelectionRange"`
	}
	if json.Unmarshal(raw, &many) != nil {
		return nil
	}
	out := make([]Location, 0, len(many))
	for _, m := range many {
		if m.TargetURI != "" {
			out = append(out, Location{URI: m.TargetURI, Range: m.TargetSelectionRange})
		} else {
			out = append(out, m.Location)
		}
	}
	return out
}

func (c *Client) locCall(method string, params any) ([]Location, error) {
	var raw json.RawMessage
	if err := c.Call(method, params, &raw, 10*time.Second); err != nil {
		return nil, err
	}
	return locations(raw), nil
}

func (c *Client) Definition(uri string, p Position) ([]Location, error) {
	return c.locCall("textDocument/definition", docPos(uri, p))
}

func (c *Client) TypeDefinition(uri string, p Position) ([]Location, error) {
	return c.locCall("textDocument/typeDefinition", docPos(uri, p))
}

func (c *Client) Implementation(uri string, p Position) ([]Location, error) {
	return c.locCall("textDocument/implementation", docPos(uri, p))
}

func (c *Client) References(uri string, p Position) ([]Location, error) {
	params := docPos(uri, p)
	params["context"] = map[string]any{"includeDeclaration": true}
	return c.locCall("textDocument/references", params)
}

// Completion 返回补全项(CompletionList 与数组两种形态都认)。
func (c *Client) Completion(uri string, p Position) ([]CompletionItem, error) {
	var raw json.RawMessage
	if err := c.Call("textDocument/completion", docPos(uri, p), &raw, 10*time.Second); err != nil {
		return nil, err
	}
	var list struct {
		Items []CompletionItem `json:"items"`
	}
	if json.Unmarshal(raw, &list) == nil && list.Items != nil {
		return list.Items, nil
	}
	var arr []CompletionItem
	_ = json.Unmarshal(raw, &arr)
	return arr, nil
}

type rawSym struct {
	Name           string          `json:"name"`
	Kind           int             `json:"kind"`
	Detail         string          `json:"detail"`
	SelectionRange *Range          `json:"selectionRange"`
	Range          *Range          `json:"range"`
	Location       *Location       `json:"location"`
	Children       json.RawMessage `json:"children"`
}

func convSyms(items []rawSym) []DocSymbol {
	out := make([]DocSymbol, 0, len(items))
	for _, it := range items {
		s := DocSymbol{Name: it.Name, Kind: it.Kind, Detail: it.Detail}
		switch {
		case it.SelectionRange != nil:
			s.Range = *it.SelectionRange
		case it.Range != nil:
			s.Range = *it.Range
		case it.Location != nil:
			s.Range = it.Location.Range
		}
		if len(it.Children) > 0 {
			var ch []rawSym
			if json.Unmarshal(it.Children, &ch) == nil {
				s.Children = convSyms(ch)
			}
		}
		out = append(out, s)
	}
	return out
}

// DocumentSymbols 返回文档符号(大纲);层级与平铺两种形态都认。
func (c *Client) DocumentSymbols(uri string) ([]DocSymbol, error) {
	var raw json.RawMessage
	if err := c.Call("textDocument/documentSymbol", map[string]any{"textDocument": map[string]any{"uri": uri}}, &raw, 10*time.Second); err != nil {
		return nil, err
	}
	var items []rawSym
	if json.Unmarshal(raw, &items) != nil {
		return nil, nil
	}
	return convSyms(items), nil
}

// InlayHint 表示一条内联提示。
type InlayHint struct {
	Position Position
	Label    string
	Kind     int
}

// InlayHint 查询某范围内的内联提示。
func (c *Client) InlayHint(uri string, rng Range) ([]InlayHint, error) {
	var raw []struct {
		Position Position `json:"position"`
		Label    any      `json:"label"`
		Kind     int      `json:"kind"`
	}
	err := c.Call("textDocument/inlayHint", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"range":        rng,
	}, &raw, 5*time.Second)
	if err != nil {
		return nil, err
	}
	out := make([]InlayHint, 0, len(raw))
	for _, h := range raw {
		out = append(out, InlayHint{Position: h.Position, Label: markupText(h.Label), Kind: h.Kind})
	}
	return out, nil
}

// ───────── 编辑类请求 ─────────

// TextEdit 是一处文本替换。
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// Formatting 请求整篇格式化,返回要应用的编辑。
func (c *Client) Formatting(uri string, tabSize int, spaces bool) ([]TextEdit, error) {
	var edits []TextEdit
	err := c.Call("textDocument/formatting", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"options":      map[string]any{"tabSize": tabSize, "insertSpaces": spaces},
	}, &edits, 15*time.Second)
	return edits, err
}

// Rename 请求重命名,返回各文件(URI)要应用的编辑。
func (c *Client) Rename(uri string, p Position, newName string) (map[string][]TextEdit, error) {
	params := docPos(uri, p)
	params["newName"] = newName
	var raw struct {
		Changes         map[string][]TextEdit `json:"changes"`
		DocumentChanges []struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			Edits []TextEdit `json:"edits"`
		} `json:"documentChanges"`
	}
	if err := c.Call("textDocument/rename", params, &raw, 20*time.Second); err != nil {
		return nil, err
	}
	out := raw.Changes
	if out == nil {
		out = map[string][]TextEdit{}
	}
	for _, dc := range raw.DocumentChanges {
		if dc.TextDocument.URI != "" {
			out[dc.TextDocument.URI] = append(out[dc.TextDocument.URI], dc.Edits...)
		}
	}
	return out, nil
}

// CodeAction 是 LSP 快速修复 / 重构项。
type CodeAction struct {
	Title  string `json:"title"`
	Kind   string `json:"kind"`
	IsPref *bool  `json:"isPreferred"`
	Edit   *struct {
		Changes map[string][]TextEdit `json:"changes"`
	} `json:"edit"`
}

// CodeAction 请求光标附近的代码操作。
func (c *Client) CodeAction(uri string, rng Range) ([]CodeAction, error) {
	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"range":        rng,
		"context": map[string]any{
			"diagnostics": []any{},
			"only":        []string{"quickfix", "refactor", "source"},
		},
	}
	var actions []CodeAction
	if err := c.Call("textDocument/codeAction", params, &actions, 12*time.Second); err != nil {
		return nil, err
	}
	return actions, nil
}

// ApplyEdits 把一组编辑应用到文本上(位置按 UTF-16 列解释),返回新文本。重叠的编辑返回错误。
func ApplyEdits(text string, edits []TextEdit) (string, error) {
	lines := strings.Split(text, "\n")
	starts := make([]int, len(lines)+1)
	for i, l := range lines {
		starts[i+1] = starts[i] + len(l) + 1
	}
	off := func(p Position) int {
		if p.Line >= len(lines) {
			return len(text)
		}
		if p.Line < 0 {
			return 0
		}
		return starts[p.Line] + ColFromUTF16(strings.TrimSuffix(lines[p.Line], "\r"), p.Character)
	}
	type span struct {
		s, e int
		t    string
	}
	sp := make([]span, 0, len(edits))
	for _, e := range edits {
		sp = append(sp, span{off(e.Range.Start), off(e.Range.End), e.NewText})
	}
	// 稳定排序(按起点),再从后往前应用
	for i := 1; i < len(sp); i++ {
		for j := i; j > 0 && sp[j].s < sp[j-1].s; j-- {
			sp[j], sp[j-1] = sp[j-1], sp[j]
		}
	}
	for i := 1; i < len(sp); i++ {
		if sp[i].s < sp[i-1].e {
			return text, fmt.Errorf("编辑范围重叠")
		}
	}
	out := text
	for i := len(sp) - 1; i >= 0; i-- {
		out = out[:sp[i].s] + sp[i].t + out[sp[i].e:]
	}
	return out, nil
}
