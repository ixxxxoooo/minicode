package lsp

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseNodeVerAndCmp(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v24.11.1", "22.22.2", 1},
		{"v22.13.0", "22.13.0", 0},
		{"v20.19.5", "22.13.0", -1},
		{"v9.11.2", "v24.0.0", -1}, // 字典序会把 v9 排在 v24 后面,这里必须按数值比
		{"18", "v18.0.0", 0},       // 缺的部分当 0
	}
	for _, c := range cases {
		a, ok1 := parseNodeVer(c.a)
		b, ok2 := parseNodeVer(c.b)
		if !ok1 || !ok2 {
			t.Fatalf("解析失败 %q %q", c.a, c.b)
		}
		if got := a.cmp(b); got != c.want {
			t.Errorf("cmp(%s, %s) = %d,期望 %d", c.a, c.b, got, c.want)
		}
	}
	for _, bad := range []string{"", "v", "abc", "v.1"} {
		if _, ok := parseNodeVer(bad); ok {
			t.Errorf("%q 不应解析成功", bad)
		}
	}
}

func TestNvmBinsNewestFirst(t *testing.T) {
	home := t.TempDir()
	for _, v := range []string{"v9.11.2", "v24.11.1", "v12.22.12"} {
		if err := os.MkdirAll(filepath.Join(home, ".nvm", "versions", "node", v, "bin"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got := nvmBins(home)
	if len(got) != 3 || !strings.Contains(got[0], "v24.11.1") || !strings.Contains(got[1], "v12.22.12") || !strings.Contains(got[2], "v9.11.2") {
		t.Fatalf("顺序不对(应为 v24 > v12 > v9):%v", got)
	}
}

func TestNodeVersionFromPath(t *testing.T) {
	for dir, want := range map[string]string{
		"/Users/x/.nvm/versions/node/v24.11.1/bin":     "v24.11.1",
		"/Users/x/.minicode/runtime/node-v24.11.1/bin": "v24.11.1",
	} {
		got, ok := nodeVersionAt(dir) // 路径里带版本号:不起子进程,目录不存在也能得到
		if !ok || got != want {
			t.Errorf("nodeVersionAt(%s) = %q,%v,期望 %q", dir, got, ok, want)
		}
	}
}

func TestSumFor(t *testing.T) {
	sums := "aaa111  node-v24.11.1-darwin-x64.tar.gz\nB05AA3  node-v24.11.1-darwin-arm64.tar.gz\n"
	if got, ok := sumFor(sums, "node-v24.11.1-darwin-arm64.tar.gz"); !ok || got != "b05aa3" {
		t.Fatalf("got %q,%v", got, ok)
	}
	if _, ok := sumFor(sums, "node-v24.11.1-linux-x64.tar.gz"); ok {
		t.Fatal("不存在的文件不应找到")
	}
}

func TestMirrorURLs(t *testing.T) {
	if got := ParseMirror("").NodeDist(); got != "https://nodejs.org/dist" {
		t.Errorf("默认应是官方源:%s", got)
	}
	cn := ParseMirror("cn")
	if cn.NodeDist() != "https://npmmirror.com/mirrors/node" || cn.Registry() != "https://registry.npmmirror.com" {
		t.Errorf("国内镜像地址不对:%s %s", cn.NodeDist(), cn.Registry())
	}
	if ParseMirror("乱写") != MirrorOfficial {
		t.Error("未知值应退回官方源")
	}
}

// tarOf 造一个 tar:每项 {名字, 类型, 内容 / 链接目标}。
func tarOf(t *testing.T, ents [][3]string) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range ents {
		h := &tar.Header{Name: e[0], Mode: 0o755}
		switch e[1] {
		case "dir":
			h.Typeflag = tar.TypeDir
		case "link":
			h.Typeflag, h.Linkname = tar.TypeSymlink, e[2]
		default:
			h.Typeflag, h.Size = tar.TypeReg, int64(len(e[2]))
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e[2])); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func TestExtractTarGz(t *testing.T) {
	dest := t.TempDir()
	skip := func(rel string) bool { return strings.HasPrefix(rel, "include/") }
	err := extractTarGz(tarOf(t, [][3]string{
		{"node-v1/", "dir", ""},
		{"node-v1/bin/", "dir", ""},
		{"node-v1/bin/node", "file", "#!bin"},
		{"node-v1/bin/npm", "link", "../lib/node_modules/npm/bin/npm-cli.js"},
		{"node-v1/include/node.h", "file", "x"}, // 被 skip
		{"node-v1/lib/node_modules/npm/bin/npm-cli.js", "file", "cli"},
	}), dest, skip)
	if err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dest, "bin", "node")); err != nil || string(b) != "#!bin" {
		t.Fatalf("bin/node 内容不对:%q %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "include", "node.h")); err == nil {
		t.Error("include/ 应被跳过")
	}
	if b, err := os.ReadFile(filepath.Join(dest, "bin", "npm")); err != nil || string(b) != "cli" {
		t.Fatalf("符号链接 bin/npm 应指向 lib 下的文件:%q %v", b, err)
	}
}

func TestExtractTarGzRejectsEscapes(t *testing.T) {
	for name, ents := range map[string][][3]string{
		"路径越界":   {{"../../evil", "file", "x"}}, // 去掉最外层目录后是 ../evil
		"符号链接越界": {{"top/bin/x", "link", "../../../etc/passwd"}},
		"绝对链接":   {{"top/bin/x", "link", "/etc/passwd"}},
	} {
		dest := t.TempDir()
		if err := extractTarGz(tarOf(t, ents), dest, nil); err == nil {
			t.Errorf("%s:应该报错", name)
		}
	}
}
