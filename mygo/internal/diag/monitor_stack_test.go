package diag

import (
	"strings"
	"testing"
)

// 栈摘要:1 号 goroutine 有栈,每帧都是「函数@文件:行」,数量不超过上限,且没有实参 / 包路径前缀。
func TestMainStackSummary(t *testing.T) {
	s := mainStackSummary(5)
	if s == "" {
		t.Fatal("栈摘要为空")
	}
	frames := strings.Split(s, " <- ")
	if len(frames) > 5 {
		t.Fatalf("超过上限 5 帧:%d", len(frames))
	}
	for _, f := range frames {
		if !strings.Contains(f, "@") || !strings.Contains(f, ".go:") {
			t.Fatalf("帧格式不对:%q", f)
		}
		if strings.Contains(f, "(0x") || strings.Contains(f, "/") {
			t.Fatalf("帧里不该有实参或路径:%q", f)
		}
	}
}
