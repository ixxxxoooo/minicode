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
	raw := `{"items":[{"kind":"full","items":[{"uri":"file:///a.go","items":[{"kind":"full","items":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}},"severity":1,"message":"e"}]}]}]}]}`
	res := parseWorkspacePull(json.RawMessage(raw))
	if len(res.ByURI) != 1 || res.ByURI["file:///a.go"][0].Message != "e" {
		t.Fatalf("解析 workspace pull:%+v", res)
	}
	raw2 := `{"kind":"full","items":[{"uri":"file:///b.rs","items":[{"kind":"full","items":[{"severity":2,"message":"w","range":{"start":{"line":1,"character":0},"end":{"line":1,"character":1}}}]}]}`
	res2 := parseWorkspacePull(json.RawMessage(raw2))
	if res2.ByURI["file:///b.rs"][0].Message != "w" {
		t.Fatalf("单块 full:%+v", res2)
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
