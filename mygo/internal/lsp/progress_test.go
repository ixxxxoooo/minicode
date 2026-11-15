package lsp

import (
	"encoding/json"
	"testing"
)

func TestParseProgressParams(t *testing.T) {
	raw := json.RawMessage(`{"token":"x","value":{"kind":"report","message":"indexing","percentage":42}}`)
	title, msg, pct, done := ParseProgressParams(raw)
	if done || title != "" || msg != "indexing" || pct != 42 {
		t.Fatalf("report: (%q,%q,%d,%v)", title, msg, pct, done)
	}
	if got := FormatProgressLine(title, msg, pct); got != "indexing 42%" {
		t.Fatalf("format: %q", got)
	}
	end := json.RawMessage(`{"value":{"kind":"end"}}`)
	if _, _, _, done = ParseProgressParams(end); !done {
		t.Fatal("end应标记 done")
	}
	begin := json.RawMessage(`{"value":{"kind":"begin","title":"Loading packages"}}`)
	title, msg, pct, done = ParseProgressParams(begin)
	if done || title != "Loading packages" || msg != "" || pct != -1 {
		t.Fatalf("begin: (%q,%q,%d,%v)", title, msg, pct, done)
	}
}
