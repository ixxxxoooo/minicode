package prefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "settings.json")
	s := Open(p)
	if s.Int("a", 7) != 7 || s.Has("a") {
		t.Fatal("默认值")
	}
	s.Set("a", 3)
	s.Set("b", true)
	s.Set("c", "x")
	s2 := Open(p)
	if s2.Int("a", 7) != 3 || !s2.Bool("b", false) || s2.Str("c", "") != "x" {
		t.Fatal("重新打开后应读回写入的值")
	}
	s2.Reset("a")
	if Open(p).Has("a") {
		t.Fatal("Reset 应落盘")
	}
	if _, err := os.Stat(p + ".tmp"); err == nil {
		t.Fatal("不应留下临时文件")
	}
}

func TestBrokenFileKeptAside(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	_ = os.WriteFile(p, []byte("{坏的"), 0o600)
	s := Open(p)
	if s.Len() != 0 {
		t.Fatal("损坏文件应从空开始")
	}
	m, _ := filepath.Glob(p + ".broken-*")
	if len(m) != 1 {
		t.Fatalf("损坏文件应改名留档,得到 %v", m)
	}
}

func TestMemoryNoDisk(t *testing.T) {
	s := Memory()
	s.Set("k", 1)
	if s.Int("k", 0) != 1 {
		t.Fatal("内存存取")
	}
	s.ResetAll()
	if s.Len() != 0 {
		t.Fatal("ResetAll")
	}
}
