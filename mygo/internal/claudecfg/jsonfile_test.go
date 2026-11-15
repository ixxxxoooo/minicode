package claudecfg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONAtomic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.json")
	if err := writeJSON(p, map[string]any{"a": 1}); err != nil {
		t.Fatal(err)
	}
	v, err := readJSON(p)
	if err != nil || v["a"] != float64(1) {
		t.Fatalf("读回失败:%v %v", v, err)
	}
	if _, err := os.Stat(p + ".mini-backup"); err != nil {
		t.Fatal("应有备份")
	}
}

func TestProjectKey(t *testing.T) {
	if projectKey("/tmp/foo/") != "/tmp/foo" {
		t.Fatal("应去掉末尾斜杠")
	}
}
