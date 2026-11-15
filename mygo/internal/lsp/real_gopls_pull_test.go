package lsp

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// 真实 gopls 的诊断拉取:没装 gopls 自动跳过。验证 initialize 的 diagnosticProvider 解析与
// workspace/diagnostic、textDocument/diagnostic 能走通(gopls 是否支持 pull 取决于版本)。
func TestRealGoplsDiagnosticPull(t *testing.T) {
	path, err := exec.LookPath("gopls")
	if err != nil {
		t.Skip("没有安装 gopls")
	}
	dir := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n\ngo 1.22\n"), 0o644))
	src := "package main\n\nfunc main() {\n\tvar x int = \"不是整数\"\n}\n"
	must(os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644))

	cmd := exec.Command(path, "serve")
	cmd.Dir = dir
	in, _ := cmd.StdinPipe()
	out, _ := cmd.StdoutPipe()
	must(cmd.Start())
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()

	c := NewClient(out, in, Handlers{})
	root := "file://" + dir
	must(c.Initialize(root, "demo", 20*time.Second))
	t.Logf("服务器 %s;能力 %+v", c.ServerName, c.Diag)

	uri := "file://" + filepath.Join(dir, "main.go")
	must(c.DidOpen(uri, "go", 1, src))

	if c.Diag.WorkspacePull {
		res, err := c.WorkspaceDiagnostic(c.Diag, "", 30*time.Second)
		if err != nil {
			t.Fatalf("workspace/diagnostic:%v", err)
		}
		t.Logf("workspace pull:%d 个文件有诊断,已检查 %d", len(res.ByURI), res.FilesChecked)
		if len(res.ByURI[uri]) == 0 {
			t.Errorf("workspace pull 应返回 main.go 的类型错误:%+v", res.ByURI)
		}
	} else if c.Diag.DocumentPull {
		res, err := c.DocumentDiagnostic(uri, "", 30*time.Second)
		if err != nil {
			t.Fatalf("textDocument/diagnostic:%v", err)
		}
		t.Logf("document pull:%d 条", len(res.Items))
		if len(res.Items) == 0 {
			t.Errorf("document pull 应返回类型错误")
		}
	} else {
		t.Log("此 gopls 版本不声明 diagnosticProvider,界面会退回 didOpen 扫盘")
	}
	c.Shutdown()
}

// 探测本机各语言服务器是否声明诊断拉取能力(只记录,不断言:取决于版本与初始化选项)。
func TestRealServersDiagnosticCaps(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{{"gopls", []string{"serve"}}, {"rust-analyzer", nil}, {"pyright-langserver", []string{"--stdio"}}}
	for _, tc := range cases {
		path, err := exec.LookPath(tc.name)
		if err != nil {
			t.Logf("%s:未安装,跳过", tc.name)
			continue
		}
		dir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n\ngo 1.22\n"), 0o644)
		_ = os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname=\"demo\"\nversion=\"0.1.0\"\nedition=\"2021\"\n"), 0o644)
		_ = os.MkdirAll(filepath.Join(dir, "src"), 0o755)
		_ = os.WriteFile(filepath.Join(dir, "src", "main.rs"), []byte("fn main() {}\n"), 0o644)
		cmd := exec.Command(path, tc.args...)
		cmd.Dir = dir
		in, _ := cmd.StdinPipe()
		out, _ := cmd.StdoutPipe()
		if err := cmd.Start(); err != nil {
			t.Logf("%s:启动失败 %v", tc.name, err)
			continue
		}
		c := NewClient(out, in, Handlers{})
		if err := c.Initialize("file://"+dir, "demo", 30*time.Second); err != nil {
			t.Logf("%s:initialize 失败 %v", tc.name, err)
		} else {
			t.Logf("%s(%s):workspacePull=%v documentPull=%v", tc.name, c.ServerName, c.Diag.WorkspacePull, c.Diag.DocumentPull)
			c.Shutdown()
		}
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}
