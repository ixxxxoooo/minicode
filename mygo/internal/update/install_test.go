package update

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifySHAAndReplaceScript(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "pkg.bin")
	if err := os.WriteFile(pkg, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if err := verifySHA(pkg, sum); err != nil {
		t.Fatalf("校验失败: %v", err)
	}
	stage := filepath.Join(dir, "staging")
	if err := os.MkdirAll(filepath.Join(stage, "MiniCode.app"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "replace.sh")
	if err := WriteReplaceScript(stage, "/Applications/MiniCode.app", script); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "ditto") || !strings.Contains(string(b), "MiniCode.app") {
		t.Fatalf("脚本内容不对:%s", b)
	}
}

func TestUnzipTo(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "u.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	fw, _ := w.Create("MiniCode.app/Contents/Info.plist")
	_, _ = fw.Write([]byte("plist"))
	w.Close()
	f.Close()
	dest := filepath.Join(dir, "out")
	if err := unzipTo(zipPath, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "MiniCode.app", "Contents", "Info.plist")); err != nil {
		t.Fatal(err)
	}
}
