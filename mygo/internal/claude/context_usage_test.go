package claude

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/core"
)

func TestContextUsageEmptySuccessDegrades(t *testing.T) {
	tr := NewTranslator("")
	ev, ok := tr.ContextBreakdown(obj{"categories": []any{}}, true)
	if ok {
		t.Fatal("空 categories 不应视为成功")
	}
	if ev != nil {
		t.Fatal("应无事件")
	}
	u := tr.UsageSnapshot()
	u.ContextDetailErr = "当前后端未返回上下文分类明细"
	ev2 := core.EvUsage{Usage: u}
	if ev2.Usage.ContextDetailErr == "" {
		t.Fatal("应有降级文案")
	}
}
