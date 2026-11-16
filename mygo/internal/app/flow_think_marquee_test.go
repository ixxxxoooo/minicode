package app

import (
	"strings"
	"testing"
)

func TestThinkPreviewPlain(t *testing.T) {
	s := "a\nb\n" + strings.Repeat("长", 300)
	got := thinkPreviewPlain(s)
	if got == "" {
		t.Fatal("empty")
	}
	if strings.Contains(got, "\n") {
		t.Fatal("newlines remain")
	}
	if len([]rune(got)) > thinkPreviewRunes {
		t.Fatalf("too long %d", len([]rune(got)))
	}
}

// 估算宽度要区分全角与西文:同样 20 个字,中文必须明显宽于英文(此前按固定 6px/字,中文滚不到尾部)。
func TestThinkMqEstimateWidth(t *testing.T) {
	if thinkMqEstimateWidth("") != 0 {
		t.Fatal("空串宽度应为 0")
	}
	cn := thinkMqEstimateWidth("路由表用原始路径做键注册与匹配")
	en := thinkMqEstimateWidth("route table key")
	if cn <= en*1.4 {
		t.Fatalf("中文应明显比西文宽: cn=%v en=%v", cn, en)
	}
}
