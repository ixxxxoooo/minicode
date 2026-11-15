package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui/editor"
)

// 括号按深度换种类:嵌套递增、同级回落、字符串里的括号不动,且 Token 仍首尾相接盖满整行。
func TestBracketHLColorsByDepth(t *testing.T) {
	src := "f(a[1], \"(\")\n{x}"
	buf := editor.NewBuffer(src)
	buf.SetLanguage(wrapBracketLang(editor.LanguageFor("a.go")))
	if !isBracketLang(buf.Language()) {
		t.Fatal("Go 应当被包装成括号着色版")
	}
	h := editor.NewTokenHighlighter(buf.Language().Tokenizer, buf.Snapshot())
	bh := &bracketHL{base: h, text: buf.Snapshot()}
	bh.Highlight(10, time.Second)

	kindAt := func(toks []editor.Token, col int) editor.TokenKind {
		start := 0
		for _, tk := range toks {
			if col >= start && col < tk.End {
				return tk.Kind
			}
			start = tk.End
		}
		t.Fatalf("第 %d 列没有 Token", col)
		return 0
	}
	l0, l1 := buf.Line(0), buf.Line(1)
	t0, _ := bh.Tokens(0, l0, nil)
	if got := t0[len(t0)-1].End; got != len(l0) {
		t.Fatalf("Token 没盖满整行: %d != %d", got, len(l0))
	}
	// f( → 深度 0;a[ → 深度 1;] 回到深度 1;最后的 ) 回到深度 0
	if got := kindAt(t0, 1); got != bracketKinds[0] {
		t.Fatalf("外层 ( 种类 = %v", got)
	}
	if got := kindAt(t0, 3); got != bracketKinds[1] {
		t.Fatalf("内层 [ 种类 = %v", got)
	}
	if got := kindAt(t0, 5); got != bracketKinds[1] {
		t.Fatalf("] 应与 [ 同色, 得 %v", got)
	}
	if got := kindAt(t0, len(l0)-1); got != bracketKinds[0] {
		t.Fatalf("外层 ) 应与 ( 同色, 得 %v", got)
	}
	if got := kindAt(t0, 9); got != editor.TokenString {
		t.Fatalf("字符串里的 ( 不该着色, 得 %v", got)
	}
	// 第二行的 { 深度回到 0
	t1, _ := bh.Tokens(1, l1, nil)
	if got := kindAt(t1, 0); got != bracketKinds[0] {
		t.Fatalf("第二行 { 种类 = %v", got)
	}
}

// Markdown / Diff 不包装;关掉后能还原成基础语言。
func TestWrapBracketLangSkips(t *testing.T) {
	md := editor.LanguageFor("a.md")
	if wrapBracketLang(md) != md {
		t.Fatal("Markdown 不应被包装")
	}
	goLang := editor.LanguageFor("a.go")
	if unwrapBracketLang(wrapBracketLang(goLang)) != goLang {
		t.Fatal("unwrap 应还原基础语言")
	}
}
