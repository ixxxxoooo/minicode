package browser

import "testing"

type fake map[string]Page

func (f fake) Fetch(u string) (Page, error) {
	if p, ok := f[u]; ok {
		return p, nil
	}
	return Page{}, ErrNoFetch
}

func TestHistory(t *testing.T) {
	src := fake{"a.com/x": {URL: "a.com/x", Sections: []Section{{Heading: "A", Body: "x"}}}, "b.com": {URL: "b.com", Sections: []Section{{Heading: "B"}}}}
	b := New(src, "https://a.com/x/")
	if b.URL() != "a.com/x" || b.Page().Title() != "A" || b.CanBack() {
		t.Fatalf("起始页不对:%q", b.URL())
	}
	if err := b.Go("http://b.com"); err != nil || b.Page().Title() != "B" || !b.CanBack() {
		t.Fatalf("导航失败:%v", err)
	}
	if err := b.Go("zzz.com"); err == nil || b.Page().Title() != "B" || b.URL() != "b.com" || b.Err == nil {
		t.Fatal("失败应保留旧页并记录错误")
	}
	b.Back()
	if b.Page().Title() != "A" || b.Err != nil || !b.CanForward() {
		t.Fatal("后退应回到 A 并清错误")
	}
	b.Forward()
	if b.URL() != "b.com" {
		t.Fatal("前进应回到 B")
	}
	b.Back()
	_ = b.Go("b.com") // 在中间导航会截断前进历史
	if b.CanForward() {
		t.Fatal("前进历史应被截断")
	}
	if !b.ToggleFav() || b.ToggleFav() {
		t.Fatal("收藏应能开关")
	}
}

func TestOutline(t *testing.T) {
	p := Page{Sections: []Section{{Heading: "T"}, {Body: "无标题"}, {Heading: "S", Level: 1}}}
	o := p.Outline()
	if len(o) != 2 || o[1].Index != 2 || o[1].Level != 1 {
		t.Fatalf("%+v", o)
	}
}
