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
	if got := FormatProgressLine("索引中", "", 42); got != "索引中 42%" {
		t.Fatalf("中文标题+%d: %q", 42, got)
	}
	reportNoPct := json.RawMessage(`{"value":{"kind":"report","message":"loading packages"}}`)
	_, msg2, pct2, _ := ParseProgressParams(reportNoPct)
	if pct2 != -1 || msg2 != "loading packages" {
		t.Fatalf("无 percentage 只留文字:(%q,%d)", msg2, pct2)
	}
	if got := FormatProgressLine("", msg2, pct2); got != "loading packages" {
		t.Fatalf("无百分比时不应伪造:%q", got)
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
