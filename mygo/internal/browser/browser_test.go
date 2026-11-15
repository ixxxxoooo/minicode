package browser

import "testing"

type fake map[string]Page

func (f fake) Fetch(u string) (Page, error) {
	if p, ok := f[Normalize(u)]; ok {
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

// 记录每次抓取参数、可切换成功 / 失败的数据源。
type flaky struct {
	up   bool
	seen []string
}

func (f *flaky) Fetch(u string) (Page, error) {
	f.seen = append(f.seen, u)
	if !f.up {
		return Page{}, ErrNoFetch
	}
	return Page{URL: Normalize(u), Sections: []Section{{Heading: "OK"}}}, nil
}

// 导航失败时(还没进历史)「重新载入」要重试失败的地址,成功后正式入历史。
func TestReloadRetriesFailedNav(t *testing.T) {
	src := &flaky{}
	b := New(src, "")
	if err := b.Go("down.example.com"); err == nil {
		t.Fatal("应失败")
	}
	b.Reload() // 仍然失败,但必须发出请求
	if len(src.seen) != 2 {
		t.Fatalf("重新载入应重试失败的地址:%v", src.seen)
	}
	src.up = true
	b.Reload()
	if b.Err != nil || b.URL() != "down.example.com" || b.Page().Title() != "OK" {
		t.Fatalf("恢复后重新载入应成功并入历史:err=%v url=%q", b.Err, b.URL())
	}
}

// 用户明确写 http:// 时,抓取参数保留协议,重新载入也沿用。
func TestExplicitHTTPKept(t *testing.T) {
	src := &flaky{up: true}
	b := New(src, "")
	_ = b.Go("http://plain.example.com/")
	b.Reload()
	_ = b.Go("secure.example.com")
	want := []string{"http://plain.example.com", "http://plain.example.com", "secure.example.com"}
	if len(src.seen) != len(want) {
		t.Fatalf("抓取参数:%v", src.seen)
	}
	for i := range want {
		if src.seen[i] != want[i] {
			t.Errorf("第 %d 次抓取参数 %q,期望 %q", i, src.seen[i], want[i])
		}
	}
}
