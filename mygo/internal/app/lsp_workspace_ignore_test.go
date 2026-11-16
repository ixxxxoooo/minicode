package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// 在临时目录里建一个真实 git 项目:.gitignore 忽略 gen/ 与 *.gen.go,node_modules 里还有一份源码。
func ignoreProject(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("没有 git")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	write := func(rel, s string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".gitignore", "gen/\n*.gen.go\n")
	write("main.go", "package main\n")
	write("pkg/util.go", "package pkg\n")
	write("gen/api.go", "package gen\n")
	write("pkg/x.gen.go", "package pkg\n")
	write("node_modules/dep/index.js", "x\n")
	return dir
}

func TestWorkspaceCandidatesRespectGitignore(t *testing.T) {
	dir := ignoreProject(t)
	a := New(Options{Palette: "paper", Scheme: "a"})
	a.SetProject(dir)

	cands := a.lspWorkspaceCandidates(false)
	if slices.Contains(cands, "node_modules/dep/index.js") {
		t.Fatalf("常见产物目录应先被跳过:%v", cands)
	}
	groups := a.ignoreGroups(cands, false)
	if len(groups) != 1 {
		t.Fatalf("真实 git 项目应有 1 个仓库分组,得到 %d", len(groups))
	}
	kept := dropIgnored(cands, groups)
	for _, want := range []string{"main.go", "pkg/util.go"} {
		if !slices.Contains(kept, want) {
			t.Errorf("%s 不该被跳过:%v", want, kept)
		}
	}
	for _, bad := range []string{"gen/api.go", "pkg/x.gen.go"} {
		if slices.Contains(kept, bad) {
			t.Errorf("%s 被 .gitignore 忽略,不该进扫描:%v", bad, kept)
		}
	}
}

// 勾选「包含被忽略的文件」:不分组、不过滤,被忽略的 Go 文件也在候选里(仍排除不了的产物目录例外不在此测)。
func TestWorkspaceIncludeIgnoredSkipsGit(t *testing.T) {
	dir := ignoreProject(t)
	a := New(Options{Palette: "paper", Scheme: "a"})
	a.SetProject(dir)
	cands := a.lspWorkspaceCandidates(true)
	if !slices.Contains(cands, "gen/api.go") {
		t.Fatalf("包含被忽略文件时应保留 gen/api.go:%v", cands)
	}
	if g := a.ignoreGroups(cands, true); len(g) != 0 {
		t.Fatalf("包含被忽略文件时不应再查 git:%v", g)
	}
}

// 读 .gitignore 规则失败时该仓库原样保留(兜底),不丢文件也不 panic。
type failingIgnore struct{}

func (failingIgnore) Ignored([]string) (map[string]bool, error) { return nil, os.ErrPermission }

func TestDropIgnoredKeepsFilesOnError(t *testing.T) {
	cands := []string{"a.go", "b.go"}
	got := dropIgnored(cands, []ignoreGroup{{ck: failingIgnore{}, rels: cands, trees: cands}})
	if !slices.Equal(got, cands) {
		t.Fatalf("读规则失败应保留全部文件:%v", got)
	}
}

// 上限:排序后只取前 limit 个。
func TestCapFiles(t *testing.T) {
	in := []string{"a", "b", "c"}
	if got := capFiles(in, 2); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("得到 %v", got)
	}
	if got := capFiles(in, 9); !slices.Equal(got, in) {
		t.Fatalf("不足上限应原样返回:%v", got)
	}
}
