package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	_ = WriteFrame(&buf, []byte(`{"a":"汉字"}`))
	_ = WriteFrame(&buf, []byte(`{"b":1}`))
	r := bufio.NewReader(&buf)
	b1, err := ReadFrame(r)
	if err != nil || string(b1) != `{"a":"汉字"}` {
		t.Fatalf("第一帧 %q %v", b1, err)
	}
	b2, _ := ReadFrame(r)
	if string(b2) != `{"b":1}` {
		t.Fatalf("第二帧 %q", b2)
	}
	if _, err := ReadFrame(r); err != io.EOF {
		t.Fatalf("流尾应是 EOF:%v", err)
	}
	if _, err := ReadFrame(bufio.NewReader(strings.NewReader("X: 1\r\n\r\n"))); err == nil {
		t.Fatal("缺 Content-Length 应报错")
	}
}

func TestUTF16Columns(t *testing.T) {
	line := "a汉😀b" // 字节:1 + 3 + 4 + 1
	if got := ColToUTF16(line, 4); got != 2 {
		t.Fatalf("汉后的列应是 2,得到 %d", got)
	}
	if got := ColToUTF16(line, 8); got != 4 { // 😀 占两个码元
		t.Fatalf("😀后的列应是 4,得到 %d", got)
	}
	if got := ColFromUTF16(line, 4); got != 8 {
		t.Fatalf("反向换算 %d", got)
	}
	if got := ColFromUTF16(line, 99); got != len(line) {
		t.Fatalf("越界应夹在行尾:%d", got)
	}
}

func TestURI(t *testing.T) {
	p := "/tmp/my proj/a.go"
	u := PathToURI(p)
	if !strings.HasPrefix(u, "file:///tmp/my%20proj") || URIToPath(u) != p {
		t.Fatalf("URI 往返 %q → %q", u, URIToPath(u))
	}
	if URIToPath("https://x/y") != "" {
		t.Fatal("非 file 协议应返回空")
	}
}

func TestSpecForFile(t *testing.T) {
	for file, id := range map[string]string{"a.rs": "rust", "x/b.go": "go", "c.PY": "python", "d.lua": "lua", "e.tsx": "ts", "f.cpp": "c", "g.swift": "swift"} {
		if s := ForFile(file); s == nil || s.ID != id {
			t.Errorf("%s 应选 %s,得到 %v", file, id, s)
		}
	}
	if ForFile("README.md") != nil {
		t.Error("md 没有服务器")
	}
	if got := ForFile("a.tsx").LangID("a.tsx"); got != "typescriptreact" {
		t.Errorf("languageId %q", got)
	}
}

func TestSpecRoot(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "cmd", "x")
	_ = os.MkdirAll(sub, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "cmd", "go.mod"), []byte("module x"), 0o644)
	s := ForFile("a.go")
	if got := s.Root(filepath.Join(sub, "main.go"), dir); got != filepath.Join(dir, "cmd") {
		t.Fatalf("应找到 cmd/go.mod 所在目录:%s", got)
	}
	if got := s.Root(filepath.Join(dir, "main.go"), dir); got != dir {
		t.Fatalf("找不到标记时用项目目录:%s", got)
	}
}

func TestParseLocationsAndHover(t *testing.T) {
	one := locations(json.RawMessage(`{"uri":"file:///a.go","range":{"start":{"line":1,"character":2},"end":{"line":1,"character":5}}}`))
	if len(one) != 1 || one[0].Range.Start.Character != 2 {
		t.Fatalf("单个 Location %v", one)
	}
	link := locations(json.RawMessage(`[{"targetUri":"file:///b.go","targetSelectionRange":{"start":{"line":3,"character":0},"end":{"line":3,"character":4}}}]`))
	if len(link) != 1 || link[0].URI != "file:///b.go" || link[0].Range.Start.Line != 3 {
		t.Fatalf("LocationLink %v", link)
	}
	if locations(json.RawMessage(`null`)) != nil {
		t.Fatal("null 应是空")
	}
	for in, want := range map[string]string{
		`"plain"`:                                "plain",
		`{"kind":"markdown","value":"**x**"}`:    "**x**",
		`{"language":"go","value":"func f()"}`:   "```go\nfunc f()\n```",
		`["a",{"kind":"plaintext","value":"b"}]`: "a\n\nb",
		`null`:                                   "",
	} {
		if got := hoverText(json.RawMessage(in)); got != want {
			t.Errorf("hoverText(%s) = %q,想要 %q", in, got, want)
		}
	}
}

// 用一个内存里的假服务器走完整条链:握手 → 服务器请求自动应答 → 诊断推送 → 悬停 → 超时 → 退出。
func TestClientWithFakeServer(t *testing.T) {
	cr, sw := io.Pipe() // 服务器 → 客户端
	sr, cw := io.Pipe() // 客户端 → 服务器
	diags := make(chan PublishDiagnostics, 1)
	c := NewClient(cr, cw, Handlers{Diagnostics: func(p PublishDiagnostics) { diags <- p }})

	srv := bufio.NewReader(sr)
	answered := make(chan string, 1)
	go func() {
		for {
			body, err := ReadFrame(srv)
			if err != nil {
				return
			}
			var m rpcMsg
			_ = json.Unmarshal(body, &m)
			reply := func(result any) {
				b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(m.ID), "result": result})
				_ = WriteFrame(sw, b)
			}
			switch {
			case m.Method == "initialize":
				// 先问客户端一个配置(服务器请求),再答握手
				req, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 900, "method": "workspace/configuration", "params": map[string]any{"items": []any{1, 2}}})
				_ = WriteFrame(sw, req)
				reply(map[string]any{"capabilities": map[string]any{"hoverProvider": true, "renameProvider": false}, "serverInfo": map[string]any{"name": "fake"}})
			case m.Method == "initialized":
				note, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "textDocument/publishDiagnostics",
					"params": PublishDiagnostics{URI: "file:///a.go", Diagnostics: []Diagnostic{{Severity: 1, Message: "boom"}}}})
				_ = WriteFrame(sw, note)
			case m.Method == "textDocument/hover":
				reply(map[string]any{"contents": map[string]any{"kind": "markdown", "value": "hi"}})
			case m.Method == "textDocument/definition": // 故意不回,测超时
			case len(m.ID) > 0 && m.Method == "" && string(m.ID) == "900":
				answered <- string(m.Result)
			}
		}
	}()

	if err := c.Initialize("file:///root", "root", 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if c.ServerName != "fake" || !c.Caps["hoverProvider"] || c.Caps["renameProvider"] {
		t.Fatalf("能力解析不对:%v %q", c.Caps, c.ServerName)
	}
	select {
	case got := <-answered:
		if got != "[null,null]" {
			t.Fatalf("workspace/configuration 应按条数回 null:%s", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("服务器的请求没有被应答")
	}
	select {
	case d := <-diags:
		if d.URI != "file:///a.go" || d.Diagnostics[0].Message != "boom" {
			t.Fatalf("诊断 %+v", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没收到诊断")
	}
	if s, err := c.Hover("file:///a.go", Position{}); err != nil || s != "hi" {
		t.Fatalf("悬停 %q %v", s, err)
	}
	c.h = Handlers{}
	start := time.Now()
	if err := c.Call("textDocument/definition", nil, nil, 150*time.Millisecond); err == nil || !strings.Contains(err.Error(), "超时") || time.Since(start) > time.Second {
		t.Fatalf("应超时:%v", err)
	}
	_ = sw.Close() // 服务器退出
	select {
	case <-c.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("连接结束后 Done 应关闭")
	}
	if err := c.Call("x", nil, nil, time.Second); err == nil {
		t.Fatal("退出后的请求应立刻失败")
	}
}

// 真实 gopls:有就测(没装自动跳过)。覆盖握手、类型错误诊断、悬停、定义、引用、补全、文档符号。
func TestRealGopls(t *testing.T) {
	spec := ForFile("a.go")
	bin, ok := spec.Find()
	if !ok {
		t.Skip("没有安装 gopls")
	}
	dir := t.TempDir()
	src := "package main\n\nimport \"fmt\"\n\n// Greet 打招呼。\nfunc Greet(name string) string { return \"hi \" + name }\n\nfunc main() {\n\tfmt.Println(Greet(\"x\"))\n\tvar n int = \"bad\"\n\t_ = n\n}\n"
	file := filepath.Join(dir, "main.go")
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0o644)
	_ = os.WriteFile(file, []byte(src), 0o644)

	diags := make(chan PublishDiagnostics, 16)
	c, cmd, err := spec.Start(bin, dir, Handlers{Diagnostics: func(p PublishDiagnostics) { diags <- p }}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { c.Shutdown(); _ = cmd.Process.Kill() }()
	uri := PathToURI(file)
	if err := c.DidOpen(uri, "go", 1, src); err != nil {
		t.Fatal(err)
	}
	// 诊断:`var n int = "bad"` 应有类型错误(第 9 行,0 起)
	deadline := time.After(60 * time.Second)
	found := false
	for !found {
		select {
		case d := <-diags:
			for _, x := range d.Diagnostics {
				if x.Severity == 1 && x.Range.Start.Line == 9 {
					found = true
				}
			}
		case <-deadline:
			t.Fatal("等不到类型错误诊断")
		}
	}
	// 悬停 Greet(第 8 行 `fmt.Println(Greet(` 里的 Greet,列 14)
	h, err := c.Hover(uri, Position{Line: 8, Character: 14})
	if err != nil || !strings.Contains(h, "Greet") {
		t.Fatalf("悬停应含 Greet:%q %v", h, err)
	}
	defs, err := c.Definition(uri, Position{Line: 8, Character: 14})
	if err != nil || len(defs) != 1 || defs[0].Range.Start.Line != 5 {
		t.Fatalf("定义应在第 6 行:%v %v", defs, err)
	}
	refs, err := c.References(uri, Position{Line: 5, Character: 6})
	if err != nil || len(refs) != 2 {
		t.Fatalf("Greet 应有 2 处引用(声明 + 调用):%v %v", refs, err)
	}
	items, err := c.Completion(uri, Position{Line: 8, Character: 6}) // `fmt.` 之后附近
	if err != nil || len(items) == 0 {
		t.Fatalf("补全应有结果:%d %v", len(items), err)
	}
	// 重命名 Greet → Hello:两处
	changes, err := c.Rename(uri, Position{Line: 5, Character: 6}, "Hello")
	if err != nil || len(changes[uri]) != 3 {
		t.Fatalf("重命名应在 3 处改(含文档注释):%v %v", changes, err)
	}
	renamed, err := ApplyEdits(src, changes[uri])
	if err != nil || strings.Count(renamed, "Hello") != 3 || strings.Contains(renamed, "Greet(") {
		t.Fatalf("应用重命名结果不对:%q %v", renamed, err)
	}
	// 格式化:把缩进弄乱后应回到 gofmt 风格
	messy := "package main\n\nfunc  main( ) {\nx:=1\n_=x\n}\n"
	_ = c.DidChange(uri, 2, messy)
	fm, err := c.Formatting(uri, 4, false)
	if err != nil || len(fm) == 0 {
		t.Fatalf("格式化应有编辑:%v %v", fm, err)
	}
	if out, _ := ApplyEdits(messy, fm); !strings.Contains(out, "func main() {") || !strings.Contains(out, "\tx := 1") {
		t.Fatalf("格式化结果 %q", out)
	}
	syms, err := c.DocumentSymbols(uri)
	if err != nil || len(syms) < 1 || syms[0].Name != "main" {
		t.Fatalf("文档符号应含 main(此时文档已被改成只剩 main):%v %v", syms, err)
	}
}

func TestApplyEdits(t *testing.T) {
	text := "a汉😀b\nline2\nend"
	// 把 😀(UTF-16 列 2–4)换成 X,再在第二行行首插入 ">",并把 end 换成 END
	got, err := ApplyEdits(text, []TextEdit{
		{Range: Range{Start: Position{0, 2}, End: Position{0, 4}}, NewText: "X"},
		{Range: Range{Start: Position{1, 0}, End: Position{1, 0}}, NewText: ">"},
		{Range: Range{Start: Position{2, 0}, End: Position{2, 3}}, NewText: "END"},
	})
	if err != nil || got != "a汉Xb\n>line2\nEND" {
		t.Fatalf("应用编辑得到 %q %v", got, err)
	}
	if _, err := ApplyEdits("abcdef", []TextEdit{
		{Range: Range{Start: Position{0, 0}, End: Position{0, 4}}, NewText: "x"},
		{Range: Range{Start: Position{0, 2}, End: Position{0, 5}}, NewText: "y"},
	}); err == nil {
		t.Fatal("重叠的编辑应报错")
	}
}
