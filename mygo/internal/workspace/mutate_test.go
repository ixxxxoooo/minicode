package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidName(t *testing.T) {
	for _, bad := range []string{"", "  ", ".", "..", "a/b", "a\\b", "x\ny"} {
		if ValidName(bad) == nil {
			t.Fatalf("%q 应不合法", bad)
		}
	}
	if ValidName("main.rs") != nil || ValidName("我的 文件.md") != nil {
		t.Fatal("正常名字应合法")
	}
	if ValidPath("/abs") == nil || ValidPath("a//b") == nil || ValidPath("a/../b") == nil {
		t.Fatal("路径里的非法段应被拒绝")
	}
	if ValidPath("a/b/c.rs") != nil || ValidPath("newdir/") != nil {
		t.Fatal("正常路径应合法")
	}
}

func TestMemMutate(t *testing.T) {
	m := NewMem("p", map[string]string{"src/a.rs": "A", "src/b/c.rs": "C", "README.md": "R"})
	if err := m.Create("src/new/deep.rs", false); err != nil {
		t.Fatal(err)
	}
	if err := m.Create("src/a.rs", false); !errors.Is(err, ErrExist) {
		t.Fatalf("已存在应报 ErrExist:%v", err)
	}
	if err := m.Create("README.md/x", false); !errors.Is(err, ErrExist) {
		t.Fatalf("文件挡住目录应报 ErrExist:%v", err)
	}
	if err := m.Create("empty", true); err != nil {
		t.Fatal(err)
	}
	if m.Count("src") != 3 || m.Count("README.md") != 1 {
		t.Fatalf("计数:%d", m.Count("src"))
	}
	if err := m.Rename("src", "lib"); err != nil {
		t.Fatal(err)
	}
	if s, err := m.Read("lib/b/c.rs"); err != nil || s != "C" {
		t.Fatalf("目录改名应连带子文件:%q %v", s, err)
	}
	if _, err := m.Read("src/a.rs"); err == nil {
		t.Fatal("旧路径应消失")
	}
	if err := m.Rename("lib/a.rs", "README.md"); !errors.Is(err, ErrExist) {
		t.Fatalf("目标已存在:%v", err)
	}
	if err := m.Rename("lib", "lib/inner"); err == nil {
		t.Fatal("不能移到自己里面")
	}
	if err := m.Rename("nope", "x"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("源不存在:%v", err)
	}
	if err := m.Remove("lib"); err != nil {
		t.Fatal(err)
	}
	if es, _ := m.List(""); len(es) != 2 { // README.md、empty
		t.Fatalf("删除后应只剩 2 项:%v", es)
	}
	if err := m.Remove("lib"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("重复删除:%v", err)
	}
}

func TestOSMutate(t *testing.T) {
	root := t.TempDir()
	o := NewOS(root)
	var trashed []string
	old := TrashFunc
	TrashFunc = func(abs string) error { trashed = append(trashed, abs); return os.RemoveAll(abs) }
	defer func() { TrashFunc = old }()

	if err := o.Create("a/b/c.txt", false); err != nil {
		t.Fatal(err)
	}
	if err := o.Create("a/b/c.txt", false); !errors.Is(err, ErrExist) {
		t.Fatalf("已存在:%v", err)
	}
	if err := o.Create("../escape", false); err == nil {
		t.Fatal("不能逃出项目根")
	}
	if err := o.Rename("a/b", "a/z"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "a/z/c.txt")); err != nil {
		t.Fatal("改名后文件应在新位置")
	}
	_ = o.Create("a/y.txt", false)
	if err := o.Rename("a/y.txt", "a/z"); !errors.Is(err, ErrExist) {
		t.Fatalf("目标已存在:%v", err)
	}
	if o.Count("a") != 2 {
		t.Fatalf("计数:%d", o.Count("a"))
	}
	if err := o.Remove(""); err == nil {
		t.Fatal("不能删项目根")
	}
	if err := o.Remove("a/z"); err != nil || len(trashed) != 1 || trashed[0] != filepath.Join(root, "a/z") {
		t.Fatalf("应走废纸篓函数:%v %v", trashed, err)
	}
	if appleQuote(`a"b\c`) != `"a\"b\\c"` {
		t.Fatal("AppleScript 转义不对")
	}
}

func TestMultiMutate(t *testing.T) {
	a := NewMem("a", map[string]string{"x.txt": "1"})
	b := NewMem("b", map[string]string{"y.txt": "2"})
	m := NewMulti("ws", []Root{{Name: "a", FS: a}, {Name: "b", FS: b}})
	if err := m.Create("a/new.txt", false); err != nil {
		t.Fatal(err)
	}
	if err := m.Create("a", true); err == nil {
		t.Fatal("顶层文件夹不可改动")
	}
	if err := m.Rename("a/x.txt", "b/x.txt"); err == nil {
		t.Fatal("不能跨文件夹移动")
	}
	if err := m.Rename("a/x.txt", "a/z.txt"); err != nil {
		t.Fatal(err)
	}
	if err := m.Remove("b/y.txt"); err != nil {
		t.Fatal(err)
	}
	if m.Count("a/z.txt") != 1 || m.Count("b") != 0 {
		t.Fatal("计数不对")
	}
}
