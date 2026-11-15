package workspace

import "testing"

func TestIsDirMem(t *testing.T) {
	m := NewMem("x", map[string]string{
		"src/a.go": "package a",
	}, "empty")
	if !IsDir(m, "") || !IsDir(m, "empty") || !IsDir(m, "src") {
		t.Fatal("应为目录")
	}
	if IsDir(m, "src/a.go") {
		t.Fatal("文件不应为目录")
	}
}
