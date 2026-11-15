package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMemList(t *testing.T) {
	m := NewMem("p", map[string]string{"src/a.rs": "a", "src/h/x.rs": "x", "Cargo.toml": "c", "README.md": "r"})
	root, _ := m.List("")
	names := []string{}
	for _, e := range root {
		names = append(names, e.Name)
	}
	want := []string{"src", "Cargo.toml", "README.md"}
	for i, w := range want {
		if names[i] != w {
			t.Fatalf("根目录顺序 %v, 想要 %v", names, want)
		}
	}
	src, _ := m.List("src")
	if len(src) != 2 || !src[0].Dir || src[0].Name != "h" || src[1].Path != "src/a.rs" {
		t.Fatalf("src = %+v", src)
	}
	if _, err := m.List("nope"); err == nil {
		t.Fatal("不存在的目录应报错")
	}
}

func TestOSReadWrite(t *testing.T) {
	d := t.TempDir()
	fs := NewOS(d)
	if err := fs.Write("a.txt", "hi"); err != nil {
		t.Fatal(err)
	}
	if s, _ := fs.Read("a.txt"); s != "hi" {
		t.Fatalf("读回 %q", s)
	}
	if _, err := os.Stat(filepath.Join(d, "a.txt.minicode-tmp")); err == nil {
		t.Fatal("临时文件应已改名")
	}
	os.MkdirAll(filepath.Join(d, "node_modules"), 0o755)
	os.MkdirAll(filepath.Join(d, "sub"), 0o755)
	es, _ := fs.List("")
	if len(es) != 2 || es[0].Name != "sub" {
		t.Fatalf("应跳过 node_modules 且文件夹在前: %+v", es)
	}
}
