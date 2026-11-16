package vcs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, rel, s string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

// 真实 git:根目录与嵌套 .gitignore、取反规则、已跟踪文件、含空格 / 中文路径都要判对。
func TestCLIIgnored(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("没有 git")
	}
	dir := t.TempDir()
	gitIn(t, dir, false, "init", "-q", "-b", "main")
	writeFile(t, dir, ".gitignore", "out/\n*.log\n!keep.log\n")
	writeFile(t, dir, "sub/.gitignore", "gen.go\n")
	for _, f := range []string{"a.go", "out/x.go", "b.log", "keep.log", "sub/gen.go", "sub/keep.go", "gen.go", "我 的.log", "tracked.log"} {
		writeFile(t, dir, f, "x\n")
	}
	gitIn(t, dir, false, "add", "-f", "tracked.log") // 已跟踪:即使匹配 *.log 也不算被忽略
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	all := []string{"a.go", "out/x.go", "b.log", "keep.log", "sub/gen.go", "sub/keep.go", "gen.go", "我 的.log", "tracked.log"}
	got, err := r.Ignored(all)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"out/x.go": true, "b.log": true, "sub/gen.go": true, "我 的.log": true}
	for _, p := range all {
		if got[p] != want[p] {
			t.Errorf("%q 被忽略=%v,期望 %v", p, got[p], want[p])
		}
	}
}

// 一个都没被忽略(git 退出码 1)不是错误;空输入直接返回。
func TestCLIIgnoredNoneAndEmpty(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("没有 git")
	}
	dir := t.TempDir()
	gitIn(t, dir, false, "init", "-q", "-b", "main")
	writeFile(t, dir, "a.go", "x\n")
	r, _ := Open(dir)
	got, err := r.Ignored([]string{"a.go"})
	if err != nil || len(got) != 0 {
		t.Fatalf("没有被忽略的文件应返回空且无错误:%v %v", got, err)
	}
	if got, err := r.Ignored(nil); err != nil || len(got) != 0 {
		t.Fatalf("空输入:%v %v", got, err)
	}
}

// 超过一批(400)的路径要分批处理且结果完整。
func TestCLIIgnoredBatches(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("没有 git")
	}
	dir := t.TempDir()
	gitIn(t, dir, false, "init", "-q", "-b", "main")
	writeFile(t, dir, ".gitignore", "*.tmp\n")
	var paths []string
	want := 0
	for i := 0; i < ignoreBatch*2+37; i++ {
		ext := "go"
		if i%3 == 0 {
			ext, want = "tmp", want+1
		}
		paths = append(paths, fmt.Sprintf("f%d.%s", i, ext))
	}
	r, _ := Open(dir)
	got, err := r.Ignored(paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != want {
		t.Fatalf("应有 %d 个被忽略,得到 %d", want, len(got))
	}
}

// 不是仓库:返回错误(调用方据此退回硬编码兜底),不 panic。
func TestCLIIgnoredNotARepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("没有 git")
	}
	r := &CLI{Dir: t.TempDir()}
	if _, err := r.Ignored([]string{"a"}); err == nil {
		t.Skip("该环境下 git 把临时目录当成了某个仓库的一部分")
	}
}

var _ IgnoreChecker = (*CLI)(nil)
