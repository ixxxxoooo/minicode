package core

import "testing"

func TestTitleFromFirstPrompt(t *testing.T) {
	s := NewState()
	s.Apply(EvAppend{Item: &User{ID: "u1", Text: "\n  修一下 add 函数\n第二行"}})
	if s.Title != "修一下 add 函数" {
		t.Fatalf("标题=%q", s.Title)
	}
	s.Apply(EvAppend{Item: &User{ID: "u2", Text: "另一个问题"}})
	if s.Title != "修一下 add 函数" {
		t.Fatalf("后续提问不应改标题:%q", s.Title)
	}
	s2 := NewState()
	s2.Apply(EvTitleChanged{Title: "后端给的标题"})
	s2.Apply(EvAppend{Item: &User{ID: "u1", Text: "hello"}})
	if s2.Title != "后端给的标题" {
		t.Fatalf("已有标题不应被覆盖:%q", s2.Title)
	}
	if got := TitleFromPrompt("一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十一二三四五"); len([]rune(got)) != 41 {
		t.Fatalf("超长应截成 40 字 + 省略号:%q", got)
	}
}
