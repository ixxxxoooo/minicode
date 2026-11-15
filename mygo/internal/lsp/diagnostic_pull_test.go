package lsp

import (
	"encoding/json"
	"testing"
)

func TestParseDiagnosticCaps(t *testing.T) {
	cases := []struct {
		raw  string
		ws   bool
		doc  bool
		name string
	}{
		{`false`, false, false, ""},
		{`true`, false, true, ""},
		{`{"workspaceDiagnostics":true,"identifier":"gopls"}`, true, false, "gopls"},
		{`{"interFileDependencies":false}`, false, true, ""},
	}
	for _, tc := range cases {
		got := ParseDiagnosticCaps(json.RawMessage(tc.raw))
		if got.WorkspacePull != tc.ws || got.DocumentPull != tc.doc || got.Identifier != tc.name {
			t.Fatalf("ParseDiagnosticCaps(%s) = %+v,想要 ws=%v doc=%v id=%q", tc.raw, got, tc.ws, tc.doc, tc.name)
		}
	}
}

func TestParseWorkspacePull(t *testing.T) {
	// 规范形状:items 里每项是一个文件的报告
	raw := `{"items":[{"kind":"full","uri":"file:///a.go","version":1,"items":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}},"severity":1,"message":"e"}]},{"kind":"unchanged","uri":"file:///b.go","resultId":"r1"}]}`
	res := parseWorkspacePull(json.RawMessage(raw))
	if len(res.ByURI) != 1 || res.ByURI["file:///a.go"][0].Message != "e" {
		t.Fatalf("解析 workspace pull:%+v", res)
	}
	if res.FilesChecked != 1 {
		t.Fatalf("unchanged 不计入已检查文件数:%d", res.FilesChecked)
	}
	// 空 / null
	if r := parseWorkspacePull(json.RawMessage("null")); len(r.ByURI) != 0 {
		t.Fatalf("null 应为空结果:%+v", r)
	}
}

func TestMergeWorkspacePull(t *testing.T) {
	a := `{"items":[{"kind":"full","items":[]}]}`
	b := `{"items":[{"kind":"full","items":[{"uri":"file:///x"}]}]}`
	merged := mergeWorkspacePull(json.RawMessage(a), json.RawMessage(b))
	var v struct {
		Items []json.RawMessage `json:"items"`
	}
	if json.Unmarshal(merged, &v) != nil || len(v.Items) != 2 {
		t.Fatalf("合并 items 数量不对:%s", merged)
	}
}
