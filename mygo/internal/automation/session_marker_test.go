package automation

import "testing"

func TestSplitSession(t *testing.T) {
	const sid = "abc-123"
	raw := EmbedSession("hello", sid)
	got, id := SplitSession(raw)
	if got != "hello" || id != sid {
		t.Fatalf("want hello,%q got %q,%q", sid, got, id)
	}
	if _, id2 := SplitSession("plain"); id2 != "" {
		t.Fatalf("无标记时不应有 session")
	}
}
