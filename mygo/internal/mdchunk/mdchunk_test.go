package mdchunk

import (
	"fmt"
	"strings"
	"testing"
)

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

func TestSplitRoundTripAndSafe(t *testing.T) {
	doc := longDoc(120)
	chunks := Split(doc)
	if len(chunks) < 5 {
		t.Fatalf("长文应被分块:%d", len(chunks))
	}
	if strings.Join(chunks, "") != doc {
		t.Fatal("块拼起来必须等于原文")
	}
	for i, ch := range chunks {
		if strings.Count(ch, "```")%2 != 0 {
			t.Fatalf("第 %d 块把围栏代码块切开了", i)
		}
		if i > 0 && !isBlockStart(strings.SplitN(ch, "\n", 2)[0]) {
			t.Fatalf("第 %d 块以不安全的行开头:%q", i, strings.SplitN(ch, "\n", 2)[0])
		}
	}
	if got := Split("短文"); len(got) != 1 {
		t.Fatal("短文不分块")
	}
}

// 文本在末尾增长(流式)时,已经切好的块不能变,否则行键 / 缓存全部失效。
func TestSplitStableWhileGrowing(t *testing.T) {
	doc := longDoc(100)
	prev := Split(doc[:len(doc)/2])
	for cut := len(doc) / 2; cut < len(doc); cut += 997 {
		cur := Split(doc[:cut])
		n := len(prev) - 1 // 最后一块还在长,前面的必须不变
		if len(cur) < n {
			t.Fatalf("块数不应减少:%d → %d", len(prev), len(cur))
		}
		for i := 0; i < n; i++ {
			if cur[i] != prev[i] {
				t.Fatalf("增长到 %d 字节时第 %d 块变了", cut, i)
			}
		}
		prev = cur
	}
}
