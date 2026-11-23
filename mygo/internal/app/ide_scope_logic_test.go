package app

import (
	"testing"

	"github.com/ixxxxoooo/minicode/internal/symbols"
)

func TestInnermostSymbolNested(t *testing.T) {
	src := "impl Router {\n    pub fn match_path() {\n        let x = 1;\n    }\n}\n"
	syms := symbols.Parse("router.rs", src)
	inner := innermostSymbol(syms, 2)
	if inner < 0 || syms[inner].Name != "match_path" {
		n := ""
		if inner >= 0 {
			n = syms[inner].Name
		}
		t.Fatalf("行 3 应在 match_path,得 %d %q", inner, n)
	}
	outer := innermostSymbol(syms, 0)
	if outer < 0 || syms[outer].Kind != "impl" {
		t.Fatalf("行 1 应在 impl,得 %d", outer)
	}
	if innermostSymbol(syms, 10) >= 0 {
		t.Fatal("文件外应无符号")
	}
}

func TestParentChain(t *testing.T) {
	src := "impl Router {\n    fn inner() {\n    }\n}\n"
	syms := symbols.Parse("router.rs", src)
	ch := parentChain(syms, 1)
	if len(ch) != 2 {
		t.Fatalf("嵌套应有 2 层,得 %d", len(ch))
	}
	if syms[ch[0]].Kind != "impl" || syms[ch[1]].Kind != "fn" {
		t.Fatalf("链顺序错误: %v %v", syms[ch[0]].Kind, syms[ch[1]].Kind)
	}
	if d := scopeDepth(ch, ch[1]); d != 0 {
		t.Fatalf("内层深度应为 0,得 %d", d)
	}
	if d := scopeDepth(ch, ch[0]); d != 1 {
		t.Fatalf("外层深度应为 1,得 %d", d)
	}
}

func TestScopeIndentLevel(t *testing.T) {
	src := "    fn foo() {\n"
	syms := symbols.Parse("x.rs", src)
	lines := []string{src}
	if n := scopeIndentLevel(syms, 0, func(i int) string { return lines[i] }); n != 4 {
		t.Fatalf("缩进应为 4,得 %d", n)
	}
}
