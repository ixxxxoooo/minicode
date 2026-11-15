package workspace

import "testing"

func TestMultiTree(t *testing.T) {
	a := NewMem("api", map[string]string{"src/main.go": "A", "README.md": "ra"})
	b := NewMem("web", map[string]string{"src/app.ts": "B"})
	b2 := NewMem("web", map[string]string{"x": "C"}) // 重名
	m := NewMulti("ws", []Root{{FS: a}, {FS: b}, {FS: b2}})

	top, _ := m.List("")
	if len(top) != 3 || top[0].Name != "api" || top[1].Name != "web" || top[2].Name != "web (2)" {
		t.Fatalf("顶层应保持添加顺序并给重名加后缀:%+v", top)
	}
	es, err := m.List("api/src")
	if err != nil || len(es) != 1 || es[0].Path != "api/src/main.go" {
		t.Fatalf("子目录路径应带根名前缀:%+v %v", es, err)
	}
	if s, _ := m.Read("web/src/app.ts"); s != "B" {
		t.Fatalf("读文件:%q", s)
	}
	if err := m.Write("web (2)/x", "D"); err != nil {
		t.Fatal(err)
	}
	if s, _ := b2.Read("x"); s != "D" {
		t.Fatalf("写进了第三个根:%q", s)
	}
	if m.Abs("api/README.md") != a.Abs("README.md") {
		t.Fatal("Abs 应交给对应的根")
	}
	if _, err := m.Read("nope/x"); err == nil {
		t.Fatal("不在工作区里应报错")
	}
	if i, rest, ok := m.Split("web/src/app.ts"); !ok || i != 1 || rest != "src/app.ts" {
		t.Fatalf("Split:%d %q %v", i, rest, ok)
	}
}
