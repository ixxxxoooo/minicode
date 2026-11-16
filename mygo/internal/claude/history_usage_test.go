package claude

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func TestEstimateContextFromLines(t *testing.T) {
	lines := []string{
		`{"type":"assistant","message":{"id":"m1","usage":{"input_tokens":1000,"cache_read_input_tokens":5000,"output_tokens":200}}}`,
		`{"type":"result","modelUsage":[{"contextWindow":180000}],"usage":{"input_tokens":1000,"cache_read_input_tokens":5000,"output_tokens":200}}`,
	}
	u, ok := estimateContextFromLines(lines)
	if !ok {
		t.Fatal("应有估算")
	}
	if !u.ContextEstimated {
		t.Fatal("应标记为估算")
	}
	want := uint64(6200)
	if u.ContextUsed != want {
		t.Fatalf("used=%d want %d", u.ContextUsed, want)
	}
	if u.ContextSize < 180_000 {
		t.Fatalf("window=%d", u.ContextSize)
	}
}

func TestReplayLinesEmitsUsageEstimate(t *testing.T) {
	lines := []string{
		`{"type":"user","uuid":"u1","timestamp":"2026-10-09T10:00:00Z","message":{"content":"hi"}}`,
		`{"type":"assistant","timestamp":"2026-10-09T10:00:05Z","message":{"id":"m1","usage":{"input_tokens":4000,"output_tokens":100},"content":[{"type":"text","text":"ok"}]}}`,
		`{"type":"result","modelUsage":[{"contextWindow":200000}]}`,
	}
	var got bool
	for _, e := range ReplayLines(lines, "/w") {
		if u, ok := e.(core.EvUsage); ok && u.Usage.ContextEstimated {
			got = true
			if u.Usage.ContextUsed != 4100 {
				t.Fatalf("used=%d", u.Usage.ContextUsed)
			}
		}
	}
	if !got {
		t.Fatal("回放应附带 EvUsage 估算")
	}
}
