package claudecfg

import (
	"path/filepath"
	"testing"
)

func TestWriteJSONAtomic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.json")
	if err := writeJSON(p, map[string]any{"a": 1}); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(p, map[string]any{"a": 2}); err != nil {
		t.Fatal(err)
	}
	v, err := readJSON(p)
	if err != nil || v["a"] != float64(2) {
		t.Fatalf("读回失败:%v %v", v, err)
	}
	bak, err := readJSON(p + ".mini-backup")
	if err != nil || bak["a"] != float64(1) {
		t.Fatalf("备份应保留上一版:%v %v", bak, err)
	}
}

func TestProjectKey(t *testing.T) {
	if projectKey("/tmp/foo/") != "/tmp/foo" {
		t.Fatal("应去掉末尾斜杠")
	}
}
