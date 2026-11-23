package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// longDoc 造一篇很长的 Markdown(分块算法本身的测试在 internal/mdchunk,这里只验证会话流接线)。
func longDoc(paras int) string {
	var b strings.Builder
	for i := 0; i < paras; i++ {
		fmt.Fprintf(&b, "## 第 %d 节\n\n这是第 %d 段正文,写得长一点让它占几行:%s\n\n", i, i, strings.Repeat("内容", 40))
		if i%7 == 3 {
			b.WriteString("```go\nfunc a() {\n\n\n\treturn\n}\n```\n\n")
		}
		if i%5 == 1 {
			b.WriteString("- 列表一\n\n- 列表二(松散列表,中间有空行)\n\n- 列表三\n\n")
		}
	}
	return b.String()
}

// 超长助手正文要拆成多行,且同屏只构建视口附近的块(整篇不进一行)。
func TestFlowRowsChunkLongAssistant(t *testing.T) {
	a, tt := sessionApp(t, "flow-gaps-b4")
	doc := longDoc(300)
	a.st.Apply(core.EvAppend{Item: &core.User{ID: "lu-long", Text: "写一篇长文"}})
	a.st.Apply(core.EvAppend{Item: &core.Assistant{ID: "la-long", Text: doc}})
	rows := a.flowRows()
	n := 0
	for _, r := range rows {
		if strings.HasPrefix(r.key, "i:la-long~") {
			n++
		}
	}
	if n < 10 {
		t.Fatalf("长正文应切成多行,实际 %d 行", n)
	}
	start := time.Now()
	tt.Frame()
	tt.Frame()
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("长正文两帧耗时 %v,虚拟化没生效", d)
	}
	if !tt.HasText("第 299 节") {
		t.Fatal("末尾应可见")
	}
	if tt.HasText("第 5 节") {
		t.Fatal("远离视口的块不应被构建")
	}
}
